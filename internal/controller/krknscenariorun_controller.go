/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Assisted-by: Claude Sonnet 4.5 (claude-sonnet-4-5@20250929)
*/

package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/kubeconfig"

	"github.com/google/uuid"
)

// KrknScenarioRunReconciler reconciles a KrknScenarioRun object
type KrknScenarioRunReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Clientset kubernetes.Interface
	Namespace string
}

// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknscenarioruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknscenarioruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknscenarioruns/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile handles the reconciliation loop for KrknScenarioRun
func (r *KrknScenarioRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the KrknScenarioRun instance
	var scenarioRun krknv1alpha1.KrknScenarioRun
	if err := r.Get(ctx, req.NamespacedName, &scenarioRun); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch KrknScenarioRun")
		return ctrl.Result{}, err
	}

	// Initialize status if first reconcile
	if scenarioRun.Status.Phase == "" {
		scenarioRun.Status.Phase = "Pending"
		scenarioRun.Status.TotalTargets = len(scenarioRun.Spec.ClusterNames)
		scenarioRun.Status.ClusterJobs = make([]krknv1alpha1.ClusterJobStatus, 0)
		if err := r.Status().Update(ctx, &scenarioRun); err != nil {
			logger.Error(err, "failed to initialize status")
			return ctrl.Result{}, err
		}
	}

	// Process each cluster
	for _, clusterName := range scenarioRun.Spec.ClusterNames {
		// Check if job already exists for this cluster
		if r.jobExistsForCluster(&scenarioRun, clusterName) {
			continue
		}

		// Create new job for this cluster
		if err := r.createClusterJob(ctx, &scenarioRun, clusterName); err != nil {
			logger.Error(err, "failed to create cluster job", "cluster", clusterName)
			// Continue with best-effort approach for other clusters
		}
	}

	// Update status for all jobs
	if err := r.updateClusterJobStatuses(ctx, &scenarioRun); err != nil {
		logger.Error(err, "failed to update cluster job statuses")
		return ctrl.Result{}, err
	}

	// Calculate overall status
	r.calculateOverallStatus(&scenarioRun)

	// Update status
	if err := r.Status().Update(ctx, &scenarioRun); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	// Requeue if jobs still running
	if scenarioRun.Status.RunningJobs > 0 {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

// createClusterJob creates all resources needed for a single cluster scenario job
func (r *KrknScenarioRunReconciler) createClusterJob(
	ctx context.Context,
	scenarioRun *krknv1alpha1.KrknScenarioRun,
	clusterName string,
) error {
	logger := log.FromContext(ctx)

	// Generate unique job ID
	jobId := uuid.New().String()

	// Set default kubeconfig path if not provided
	kubeconfigPath := scenarioRun.Spec.KubeconfigPath
	if kubeconfigPath == "" {
		kubeconfigPath = "/home/krkn/.kube/config"
	}

	// Get kubeconfig using targetRequestId and clusterName
	kubeconfigBase64, err := r.getKubeconfig(ctx, "", scenarioRun.Spec.TargetRequestId, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Decode kubeconfig for ConfigMap
	kubeconfigDecoded, err := base64.StdEncoding.DecodeString(kubeconfigBase64)
	if err != nil {
		return fmt.Errorf("failed to decode kubeconfig: %w", err)
	}

	// Create ConfigMap for kubeconfig
	kubeconfigConfigMapName := fmt.Sprintf("krkn-job-%s-kubeconfig", jobId)
	kubeconfigConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubeconfigConfigMapName,
			Namespace: r.Namespace,
			Labels: map[string]string{
				"krkn-job-id":         jobId,
				"krkn-scenario-run":   scenarioRun.Name,
				"krkn-scenario-name":  scenarioRun.Spec.ScenarioName,
				"krkn-cluster-name":   clusterName,
				"krkn-target-request": scenarioRun.Spec.TargetRequestId,
			},
		},
		Data: map[string]string{
			"config": string(kubeconfigDecoded),
		},
	}

	// Set owner reference for automatic cleanup
	if err := controllerutil.SetControllerReference(scenarioRun, kubeconfigConfigMap, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on kubeconfig ConfigMap: %w", err)
	}

	if err := r.Create(ctx, kubeconfigConfigMap); err != nil {
		return fmt.Errorf("failed to create kubeconfig ConfigMap: %w", err)
	}

	// Track created resources for cleanup on error
	var fileConfigMaps []string
	var imagePullSecretName string

	// Cleanup helper
	cleanup := func() {
		r.Delete(ctx, kubeconfigConfigMap)
		for _, cm := range fileConfigMaps {
			r.Delete(ctx, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cm,
					Namespace: r.Namespace,
				},
			})
		}
		if imagePullSecretName != "" {
			r.Delete(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      imagePullSecretName,
					Namespace: r.Namespace,
				},
			})
		}
	}

	// Create ConfigMaps for user-provided files
	for _, file := range scenarioRun.Spec.Files {
		// Sanitize filename for ConfigMap name
		sanitizedName := strings.ReplaceAll(file.Name, "/", "-")
		sanitizedName = strings.ReplaceAll(sanitizedName, ".", "-")
		configMapName := fmt.Sprintf("krkn-job-%s-file-%s", jobId, sanitizedName)

		// Decode base64 content
		fileContent, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			cleanup()
			return fmt.Errorf("failed to decode file content for '%s': %w", file.Name, err)
		}

		fileConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: r.Namespace,
				Labels: map[string]string{
					"krkn-job-id":         jobId,
					"krkn-scenario-run":   scenarioRun.Name,
					"krkn-scenario-name":  scenarioRun.Spec.ScenarioName,
					"krkn-cluster-name":   clusterName,
					"krkn-target-request": scenarioRun.Spec.TargetRequestId,
				},
			},
			Data: map[string]string{
				file.Name: string(fileContent),
			},
		}

		// Set owner reference
		if err := controllerutil.SetControllerReference(scenarioRun, fileConfigMap, r.Scheme); err != nil {
			cleanup()
			return fmt.Errorf("failed to set owner reference on file ConfigMap: %w", err)
		}

		if err := r.Create(ctx, fileConfigMap); err != nil {
			cleanup()
			return fmt.Errorf("failed to create file ConfigMap: %w", err)
		}

		fileConfigMaps = append(fileConfigMaps, configMapName)
	}

	// Handle private registry authentication
	var imagePullSecrets []corev1.LocalObjectReference
	if scenarioRun.Spec.RegistryURL != "" && scenarioRun.Spec.ScenarioRepository != "" {
		imagePullSecretName = fmt.Sprintf("krkn-job-%s-registry", jobId)

		// Build docker config JSON
		authStr := ""
		if scenarioRun.Spec.Token != "" {
			authStr = base64.StdEncoding.EncodeToString([]byte(scenarioRun.Spec.Token))
		} else if scenarioRun.Spec.Username != "" && scenarioRun.Spec.Password != "" {
			authStr = base64.StdEncoding.EncodeToString([]byte(scenarioRun.Spec.Username + ":" + scenarioRun.Spec.Password))
		}

		dockerConfig := map[string]interface{}{
			"auths": map[string]interface{}{
				scenarioRun.Spec.RegistryURL: map[string]string{
					"auth": authStr,
				},
			},
		}

		dockerConfigJSON, _ := json.Marshal(dockerConfig)

		imagePullSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      imagePullSecretName,
				Namespace: r.Namespace,
				Labels: map[string]string{
					"krkn-job-id":         jobId,
					"krkn-scenario-run":   scenarioRun.Name,
					"krkn-scenario-name":  scenarioRun.Spec.ScenarioName,
					"krkn-cluster-name":   clusterName,
					"krkn-target-request": scenarioRun.Spec.TargetRequestId,
				},
			},
			Type: corev1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				".dockerconfigjson": dockerConfigJSON,
			},
		}

		// Set owner reference
		if err := controllerutil.SetControllerReference(scenarioRun, imagePullSecret, r.Scheme); err != nil {
			cleanup()
			return fmt.Errorf("failed to set owner reference on imagePullSecret: %w", err)
		}

		if err := r.Create(ctx, imagePullSecret); err != nil {
			cleanup()
			return fmt.Errorf("failed to create ImagePullSecret: %w", err)
		}

		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{
			Name: imagePullSecretName,
		})
	}

	// Build volumes and volume mounts
	volumes := []corev1.Volume{
		{
			Name: "kubeconfig",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: kubeconfigConfigMapName,
					},
				},
			},
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "kubeconfig",
			MountPath: kubeconfigPath,
			SubPath:   "config",
		},
	}

	// Add file mounts
	for i, file := range scenarioRun.Spec.Files {
		volumeName := fmt.Sprintf("file-%d", i)

		volumes = append(volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fileConfigMaps[i],
					},
				},
			},
		})

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: file.MountPath,
			SubPath:   file.Name,
		})
	}

	// Add writable tmp volume
	volumes = append(volumes, corev1.Volume{
		Name: "tmp",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "tmp",
		MountPath: "/tmp",
	})

	// Convert environment map to EnvVar slice
	envVars := make([]corev1.EnvVar, 0, len(scenarioRun.Spec.Environment))
	for key, value := range scenarioRun.Spec.Environment {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	// SecurityContext for running as krkn user (UID 1001)
	var runAsUser int64 = 1001
	var runAsGroup int64 = 1001
	var fsGroup int64 = 1001

	// Create the pod
	podName := fmt.Sprintf("krkn-job-%s", jobId)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: r.Namespace,
			Labels: map[string]string{
				"app":                 "krkn-scenario",
				"krkn-job-id":         jobId,
				"krkn-scenario-run":   scenarioRun.Name,
				"krkn-scenario-name":  scenarioRun.Spec.ScenarioName,
				"krkn-cluster-name":   clusterName,
				"krkn-target-request": scenarioRun.Spec.TargetRequestId,
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "krkn-operator-krkn-scenario-runner",
			RestartPolicy:      corev1.RestartPolicyNever,
			ImagePullSecrets:   imagePullSecrets,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser:  &runAsUser,
				RunAsGroup: &runAsGroup,
				FSGroup:    &fsGroup,
			},
			Containers: []corev1.Container{
				{
					Name:            "scenario",
					Image:           scenarioRun.Spec.ScenarioImage,
					Env:             envVars,
					VolumeMounts:    volumeMounts,
					ImagePullPolicy: corev1.PullAlways,
				},
			},
			Volumes: volumes,
		},
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(scenarioRun, pod, r.Scheme); err != nil {
		cleanup()
		return fmt.Errorf("failed to set owner reference on pod: %w", err)
	}

	if err := r.Create(ctx, pod); err != nil {
		cleanup()
		return fmt.Errorf("failed to create pod: %w", err)
	}

	// Add job to status
	now := metav1.Now()
	jobStatus := krknv1alpha1.ClusterJobStatus{
		ClusterName: clusterName,
		JobId:       jobId,
		PodName:     podName,
		Phase:       "Pending",
		StartTime:   &now,
	}

	scenarioRun.Status.ClusterJobs = append(scenarioRun.Status.ClusterJobs, jobStatus)

	logger.Info("created cluster job", "cluster", clusterName, "jobId", jobId, "pod", podName)

	return nil
}

// updateClusterJobStatuses updates the status of all cluster jobs by querying their pods
func (r *KrknScenarioRunReconciler) updateClusterJobStatuses(
	ctx context.Context,
	scenarioRun *krknv1alpha1.KrknScenarioRun,
) error {
	for i := range scenarioRun.Status.ClusterJobs {
		job := &scenarioRun.Status.ClusterJobs[i]

		// Skip completed jobs
		if job.Phase == "Succeeded" || job.Phase == "Failed" {
			continue
		}

		// Fetch pod
		var pod corev1.Pod
		err := r.Get(ctx, types.NamespacedName{
			Name:      job.PodName,
			Namespace: r.Namespace,
		}, &pod)

		if err != nil {
			if apierrors.IsNotFound(err) {
				job.Phase = "Failed"
				job.Message = "Pod not found"
				now := metav1.Now()
				job.CompletionTime = &now
			}
			continue
		}

		// Update job status based on pod phase
		switch pod.Status.Phase {
		case corev1.PodPending:
			job.Phase = "Pending"
		case corev1.PodRunning:
			job.Phase = "Running"
		case corev1.PodSucceeded:
			job.Phase = "Succeeded"
			r.setCompletionTime(job)
		case corev1.PodFailed:
			job.Phase = "Failed"
			job.Message = r.extractPodErrorMessage(&pod)
			r.setCompletionTime(job)
		case corev1.PodUnknown:
			job.Phase = "Failed"
			job.Message = "Pod in unknown state"
			r.setCompletionTime(job)
		}
	}

	return nil
}

// setCompletionTime sets the completion time if not already set
func (r *KrknScenarioRunReconciler) setCompletionTime(job *krknv1alpha1.ClusterJobStatus) {
	if job.CompletionTime == nil {
		now := metav1.Now()
		job.CompletionTime = &now
	}
}

// extractPodErrorMessage extracts error message from pod status
func (r *KrknScenarioRunReconciler) extractPodErrorMessage(pod *corev1.Pod) string {
	if len(pod.Status.ContainerStatuses) == 0 {
		return ""
	}

	containerStatus := pod.Status.ContainerStatuses[0]
	if terminated := containerStatus.State.Terminated; terminated != nil {
		return terminated.Reason + ": " + terminated.Message
	}
	if waiting := containerStatus.State.Waiting; waiting != nil {
		return waiting.Reason + ": " + waiting.Message
	}
	return ""
}

// jobExistsForCluster checks if a job already exists for the given cluster
func (r *KrknScenarioRunReconciler) jobExistsForCluster(scenarioRun *krknv1alpha1.KrknScenarioRun, clusterName string) bool {
	for _, job := range scenarioRun.Status.ClusterJobs {
		if job.ClusterName == clusterName {
			return true
		}
	}
	return false
}

// calculateOverallStatus computes the overall phase and counters
func (r *KrknScenarioRunReconciler) calculateOverallStatus(scenarioRun *krknv1alpha1.KrknScenarioRun) {
	var successfulJobs, failedJobs, runningJobs, pendingJobs int

	for _, job := range scenarioRun.Status.ClusterJobs {
		switch job.Phase {
		case "Succeeded":
			successfulJobs++
		case "Failed":
			failedJobs++
		case "Running":
			runningJobs++
		case "Pending":
			pendingJobs++
		}
	}

	scenarioRun.Status.SuccessfulJobs = successfulJobs
	scenarioRun.Status.FailedJobs = failedJobs
	scenarioRun.Status.RunningJobs = runningJobs

	// Calculate overall phase
	totalJobs := len(scenarioRun.Status.ClusterJobs)
	if totalJobs == 0 {
		scenarioRun.Status.Phase = "Pending"
	} else if runningJobs > 0 || pendingJobs > 0 {
		scenarioRun.Status.Phase = "Running"
	} else if failedJobs == totalJobs {
		scenarioRun.Status.Phase = "Failed"
	} else if successfulJobs == totalJobs {
		scenarioRun.Status.Phase = "Succeeded"
	} else {
		// Some succeeded, some failed
		scenarioRun.Status.Phase = "PartiallyFailed"
	}
}

// getKubeconfig is a helper method adapted from the API handler
// It retrieves kubeconfig from KrknTargetRequest (legacy) or KrknOperatorTarget (new)
func (r *KrknScenarioRunReconciler) getKubeconfig(ctx context.Context, targetUUID string, targetId string, clusterName string) (string, error) {
	// Try new system first (KrknOperatorTarget)
	if targetUUID != "" {
		kubeconfigBase64, err := r.getKubeconfigFromOperatorTarget(ctx, targetUUID)
		if err == nil {
			return kubeconfigBase64, nil
		}
		// If KrknOperatorTarget not found but we have legacy params, try legacy
		if targetId == "" || clusterName == "" {
			return "", err
		}
	}

	// Fall back to legacy system (KrknTargetRequest)
	if targetId != "" && clusterName != "" {
		return r.getKubeconfigFromTargetRequest(ctx, targetId, clusterName)
	}

	return "", fmt.Errorf("insufficient parameters: provide either targetUUID (new) or targetId+clusterName (legacy)")
}

// getKubeconfigFromOperatorTarget retrieves kubeconfig from KrknOperatorTarget
func (r *KrknScenarioRunReconciler) getKubeconfigFromOperatorTarget(ctx context.Context, targetUUID string) (string, error) {
	// Fetch KrknOperatorTarget
	var target krknv1alpha1.KrknOperatorTarget
	if err := r.Get(ctx, types.NamespacedName{
		Name:      targetUUID,
		Namespace: r.Namespace,
	}, &target); err != nil {
		return "", fmt.Errorf("failed to fetch KrknOperatorTarget: %w", err)
	}

	// Fetch Secret
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Name:      target.Spec.SecretUUID,
		Namespace: r.Namespace,
	}, &secret); err != nil {
		return "", fmt.Errorf("failed to fetch secret: %w", err)
	}

	// Extract kubeconfig from secret data
	kubeconfigData, exists := secret.Data["kubeconfig"]
	if !exists {
		return "", fmt.Errorf("kubeconfig not found in secret")
	}

	// Unmarshal JSON to get base64-encoded kubeconfig
	kubeconfigBase64, err := kubeconfig.UnmarshalSecretData(kubeconfigData)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal kubeconfig from secret: %w", err)
	}

	return kubeconfigBase64, nil
}

// getKubeconfigFromTargetRequest retrieves kubeconfig from KrknTargetRequest (legacy)
func (r *KrknScenarioRunReconciler) getKubeconfigFromTargetRequest(ctx context.Context, targetId string, clusterName string) (string, error) {
	// Fetch the secret with the same name as the KrknTargetRequest ID
	var secret corev1.Secret
	err := r.Get(ctx, types.NamespacedName{
		Name:      targetId,
		Namespace: r.Namespace,
	}, &secret)

	if err != nil {
		return "", fmt.Errorf("failed to fetch secret: %w", err)
	}

	// Retrieve the managed-clusters JSON from the secret data
	managedClustersBytes, exists := secret.Data["managed-clusters"]
	if !exists {
		return "", fmt.Errorf("managed-clusters not found in secret")
	}

	// Parse the JSON to extract cluster configurations
	var managedClusters map[string]map[string]struct {
		Kubeconfig string `json:"kubeconfig"`
	}
	if err := json.Unmarshal(managedClustersBytes, &managedClusters); err != nil {
		return "", fmt.Errorf("failed to parse managed-clusters JSON: %w", err)
	}

	// Get the krkn-operator-acm object
	acmClusters, exists := managedClusters["krkn-operator-acm"]
	if !exists {
		return "", fmt.Errorf("krkn-operator-acm not found in managed-clusters")
	}

	// Check if the requested cluster exists
	clusterConfig, exists := acmClusters[clusterName]
	if !exists {
		return "", fmt.Errorf("cluster '%s' not found in krkn-operator-acm", clusterName)
	}

	// Return the base64-encoded kubeconfig
	return clusterConfig.Kubeconfig, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *KrknScenarioRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&krknv1alpha1.KrknScenarioRun{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
