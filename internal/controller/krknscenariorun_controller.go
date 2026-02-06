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
	"reflect"
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
// +kubebuilder:rbac:groups="",resources=pods/log,verbs=get
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krkntargetrequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknoperatortargets,verbs=get;list;watch

// Reconcile handles the reconciliation loop for KrknScenarioRun
func (r *KrknScenarioRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop started",
		"scenarioRun", req.Name,
		"namespace", req.Namespace)

	// Fetch the KrknScenarioRun instance
	var scenarioRun krknv1alpha1.KrknScenarioRun
	if err := r.Get(ctx, req.NamespacedName, &scenarioRun); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("scenarioRun not found, probably deleted", "scenarioRun", req.Name)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch KrknScenarioRun")
		return ctrl.Result{}, err
	}

	// Initialize status if first reconcile
	if scenarioRun.Status.Phase == "" {
		// Calculate total targets
		totalTargets := 0
		for _, clusters := range scenarioRun.Spec.TargetClusters {
			totalTargets += len(clusters)
		}

		logger.Info("initializing scenarioRun status",
			"scenarioRun", scenarioRun.Name,
			"totalTargets", totalTargets,
			"targetClusters", scenarioRun.Spec.TargetClusters)

		scenarioRun.Status.Phase = "Pending"
		scenarioRun.Status.TotalTargets = totalTargets
		scenarioRun.Status.ClusterJobs = make([]krknv1alpha1.ClusterJobStatus, 0)
		if err := r.Status().Update(ctx, &scenarioRun); err != nil {
			logger.Error(err, "failed to initialize status")
			return ctrl.Result{}, err
		}
	}

	// Process each provider and their clusters
	jobsCreated := 0
	for providerName, clusterNames := range scenarioRun.Spec.TargetClusters {
		for _, clusterName := range clusterNames {
			// Check if job already exists for this cluster
			if r.jobExistsForCluster(&scenarioRun, clusterName) {
				logger.V(1).Info("job already exists for cluster, skipping",
					"provider", providerName,
					"cluster", clusterName,
					"scenarioRun", scenarioRun.Name)
				continue
			}

			logger.Info("creating job for cluster",
				"provider", providerName,
				"cluster", clusterName,
				"scenarioRun", scenarioRun.Name)

			// Create new job for this cluster
			if err := r.createClusterJob(ctx, &scenarioRun, providerName, clusterName); err != nil {
				logger.Error(err, "failed to create cluster job",
					"provider", providerName,
					"cluster", clusterName,
					"scenarioRun", scenarioRun.Name)
				// Continue with best-effort approach for other clusters
			} else {
				jobsCreated++
			}
		}
	}

	if jobsCreated > 0 {
		logger.Info("jobs created in this reconcile loop",
			"count", jobsCreated,
			"scenarioRun", scenarioRun.Name)
	}

	logger.V(1).Info("updating cluster job statuses",
		"scenarioRun", scenarioRun.Name,
		"totalJobs", len(scenarioRun.Status.ClusterJobs))

	// Save original status to detect changes
	originalStatus := scenarioRun.Status.DeepCopy()

	// Update status for all jobs
	if err := r.updateClusterJobStatuses(ctx, &scenarioRun); err != nil {
		logger.Error(err, "failed to update cluster job statuses")
		return ctrl.Result{}, err
	}

	// Calculate overall status
	r.calculateOverallStatus(&scenarioRun)

	logger.Info("reconcile loop completed",
		"scenarioRun", scenarioRun.Name,
		"phase", scenarioRun.Status.Phase,
		"totalTargets", scenarioRun.Status.TotalTargets,
		"successfulJobs", scenarioRun.Status.SuccessfulJobs,
		"failedJobs", scenarioRun.Status.FailedJobs,
		"runningJobs", scenarioRun.Status.RunningJobs)

	// Update status only if it has changed
	statusChanged := !r.statusEqual(originalStatus, &scenarioRun.Status)

	logger.V(1).Info("status comparison result",
		"scenarioRun", scenarioRun.Name,
		"statusChanged", statusChanged,
		"oldPhase", originalStatus.Phase,
		"newPhase", scenarioRun.Status.Phase,
		"oldRunning", originalStatus.RunningJobs,
		"newRunning", scenarioRun.Status.RunningJobs)

	if statusChanged {
		// Log what changed
		changes := r.detectStatusChanges(originalStatus, &scenarioRun.Status)
		logger.Info("status changed, updating CR",
			"scenarioRun", scenarioRun.Name,
			"changes", changes)

		if err := r.Status().Update(ctx, &scenarioRun); err != nil {
			logger.Error(err, "failed to update status")
			return ctrl.Result{}, err
		}
	} else {
		logger.V(1).Info("status unchanged, skipping update",
			"scenarioRun", scenarioRun.Name,
			"phase", scenarioRun.Status.Phase,
			"runningJobs", scenarioRun.Status.RunningJobs)
	}

	// Requeue if jobs still running
	if scenarioRun.Status.RunningJobs > 0 {
		logger.V(1).Info("requeuing because jobs still running",
			"scenarioRun", scenarioRun.Name,
			"runningJobs", scenarioRun.Status.RunningJobs)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

// createClusterJob creates all resources needed for a single cluster scenario job
func (r *KrknScenarioRunReconciler) createClusterJob(
	ctx context.Context,
	scenarioRun *krknv1alpha1.KrknScenarioRun,
	providerName string,
	clusterName string,
) error {
	logger := log.FromContext(ctx)

	// Check if this is a retry case
	existingJobIndex := -1
	for i, job := range scenarioRun.Status.ClusterJobs {
		if job.ClusterName == clusterName && job.Phase == "Retrying" {
			existingJobIndex = i
			break
		}
	}

	// Generate unique job ID
	jobId := uuid.New().String()

	// Set default kubeconfig path if not provided
	kubeconfigPath := scenarioRun.Spec.KubeconfigPath
	if kubeconfigPath == "" {
		kubeconfigPath = "/home/krkn/.kube/config"
	}

	logger.Info("getting kubeconfig for cluster",
		"provider", providerName,
		"cluster", clusterName,
		"targetRequestId", scenarioRun.Spec.TargetRequestId)

	// Get kubeconfig from managed-clusters Secret (works for ALL providers)
	kubeconfigBase64, err := r.getKubeconfigFromProvider(ctx, scenarioRun.Spec.TargetRequestId, providerName, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig from provider %s: %w", providerName, err)
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

	// Update status - either update existing entry (retry) or add new entry
	now := metav1.Now()
	if existingJobIndex >= 0 {
		// Update existing entry (retry case)
		scenarioRun.Status.ClusterJobs[existingJobIndex].JobId = jobId
		scenarioRun.Status.ClusterJobs[existingJobIndex].PodName = podName
		scenarioRun.Status.ClusterJobs[existingJobIndex].Phase = "Pending"
		scenarioRun.Status.ClusterJobs[existingJobIndex].StartTime = &now
		scenarioRun.Status.ClusterJobs[existingJobIndex].CompletionTime = nil
		scenarioRun.Status.ClusterJobs[existingJobIndex].Message = ""

		logger.Info("updated retry job in status",
			"cluster", clusterName,
			"newJobId", jobId,
			"retryAttempt", scenarioRun.Status.ClusterJobs[existingJobIndex].RetryCount)
	} else {
		// New job (first attempt)
		jobStatus := krknv1alpha1.ClusterJobStatus{
			ProviderName: providerName,
			ClusterName:  clusterName,
			JobId:        jobId,
			PodName:      podName,
			Phase:        "Pending",
			StartTime:    &now,
			RetryCount:   0,
			MaxRetries:   0, // Will be set from spec on first failure
		}
		scenarioRun.Status.ClusterJobs = append(scenarioRun.Status.ClusterJobs, jobStatus)

		logger.Info("created new cluster job",
			"cluster", clusterName,
			"jobId", jobId,
			"pod", podName)
	}

	return nil
}

// updateClusterJobStatuses updates the status of all cluster jobs by querying their pods
func (r *KrknScenarioRunReconciler) updateClusterJobStatuses(
	ctx context.Context,
	scenarioRun *krknv1alpha1.KrknScenarioRun,
) error {
	logger := log.FromContext(ctx)

	for i := range scenarioRun.Status.ClusterJobs {
		job := &scenarioRun.Status.ClusterJobs[i]

		logger.V(1).Info("checking job status",
			"cluster", job.ClusterName,
			"jobId", job.JobId,
			"currentPhase", job.Phase,
			"podName", job.PodName)

		// Skip terminal jobs
		if job.Phase == "Succeeded" || job.Phase == "Cancelled" || job.Phase == "MaxRetriesExceeded" {
			logger.V(1).Info("skipping terminal job",
				"cluster", job.ClusterName,
				"jobId", job.JobId,
				"phase", job.Phase)
			continue
		}

		// Skip Failed jobs unless they need retry processing
		if job.Phase == "Failed" && job.RetryCount >= job.MaxRetries && !job.CancelRequested {
			logger.V(1).Info("skipping failed job that exceeded retries",
				"cluster", job.ClusterName,
				"jobId", job.JobId,
				"retryCount", job.RetryCount,
				"maxRetries", job.MaxRetries)
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
				// IMPORTANT: Don't mark as Failed if pod was just created
				// Kubernetes might not have created the pod yet
				if job.Phase == "Pending" {
					// Calculate time since job start
					if job.StartTime != nil {
						timeSinceStart := time.Since(job.StartTime.Time)
						if timeSinceStart < 30*time.Second {
							// Pod not found but job is recent - this is normal, keep waiting
							logger.V(1).Info("pod not found but job is recent, keeping Pending status",
								"cluster", job.ClusterName,
								"jobId", job.JobId,
								"podName", job.PodName,
								"timeSinceStart", timeSinceStart.String())
							continue
						}
					}
				}

				// Pod genuinely not found - this is an error
				logger.Info("pod not found for job",
					"cluster", job.ClusterName,
					"jobId", job.JobId,
					"podName", job.PodName,
					"currentPhase", job.Phase)

				job.Phase = "Failed"
				job.Message = "Pod not found"
				job.FailureReason = "PodNotFound"
				now := metav1.Now()
				job.CompletionTime = &now
			} else {
				logger.Error(err, "error fetching pod",
					"cluster", job.ClusterName,
					"jobId", job.JobId,
					"podName", job.PodName)
			}
			continue
		}

		logger.V(1).Info("pod found",
			"cluster", job.ClusterName,
			"jobId", job.JobId,
			"podName", job.PodName,
			"podPhase", pod.Status.Phase)

		// Update job status based on pod phase
		previousPhase := job.Phase
		switch pod.Status.Phase {
		case corev1.PodPending:
			job.Phase = "Pending"
			if previousPhase != "Pending" {
				logger.Info("job phase transition",
					"cluster", job.ClusterName,
					"jobId", job.JobId,
					"from", previousPhase,
					"to", "Pending")
			}
		case corev1.PodRunning:
			job.Phase = "Running"
			if previousPhase != "Running" {
				logger.Info("job phase transition",
					"cluster", job.ClusterName,
					"jobId", job.JobId,
					"from", previousPhase,
					"to", "Running")
			}
		case corev1.PodSucceeded:
			job.Phase = "Succeeded"
			r.setCompletionTime(job)
			logger.Info("job succeeded",
				"cluster", job.ClusterName,
				"jobId", job.JobId,
				"duration", job.CompletionTime.Sub(job.StartTime.Time).String())
		case corev1.PodFailed:
			job.Phase = "Failed"
			job.Message = r.extractPodErrorMessage(&pod)
			job.FailureReason = r.extractFailureReason(&pod)
			r.setCompletionTime(job)

			// Retry logic
			logger.Info("pod failed, checking retry eligibility",
				"cluster", job.ClusterName,
				"jobId", job.JobId,
				"retryCount", job.RetryCount,
				"maxRetries", job.MaxRetries,
				"cancelRequested", job.CancelRequested,
				"failureReason", job.FailureReason)

			maxRetries := job.MaxRetries
			if maxRetries == 0 {
				maxRetries = scenarioRun.Spec.MaxRetries
				if maxRetries == 0 {
					maxRetries = 3 // Default
				}
				job.MaxRetries = maxRetries
			}

			if r.shouldRetryJob(job, maxRetries) {
				// Calculate backoff delay
				delay := r.calculateRetryDelay(job.RetryCount,
					scenarioRun.Spec.RetryBackoff,
					scenarioRun.Spec.RetryDelay)

				// Check if enough time has passed since last retry
				now := metav1.Now()
				if job.LastRetryTime != nil {
					elapsed := now.Sub(job.LastRetryTime.Time)
					if elapsed < delay {
						logger.Info("waiting for retry backoff",
							"cluster", job.ClusterName,
							"jobId", job.JobId,
							"elapsed", elapsed.String(),
							"requiredDelay", delay.String())
						// Don't retry yet, will check again on next reconcile
						continue
					}
				}

				// Retry!
				job.Phase = "Retrying"
				job.RetryCount++
				job.LastRetryTime = &now

				logger.Info("retrying failed job",
					"cluster", job.ClusterName,
					"previousJobId", job.JobId,
					"retryAttempt", job.RetryCount,
					"maxRetries", maxRetries)

				// Create new pod (will get new jobId)
				if err := r.createClusterJob(ctx, scenarioRun, job.ProviderName, job.ClusterName); err != nil {
					logger.Error(err, "failed to create retry job",
						"cluster", job.ClusterName,
						"retryAttempt", job.RetryCount)
					job.Phase = "Failed"
					job.Message = "Retry failed: " + err.Error()
				}
			} else if job.CancelRequested {
				job.Phase = "Cancelled"
				logger.Info("job marked as cancelled, no retry",
					"cluster", job.ClusterName,
					"jobId", job.JobId)
			} else {
				job.Phase = "MaxRetriesExceeded"
				logger.Info("job exceeded max retries",
					"cluster", job.ClusterName,
					"jobId", job.JobId,
					"retryCount", job.RetryCount,
					"maxRetries", maxRetries)
			}
		case corev1.PodUnknown:
			job.Phase = "Failed"
			job.Message = "Pod in unknown state"
			job.FailureReason = "PodUnknown"
			r.setCompletionTime(job)
			logger.Info("pod in unknown state",
				"cluster", job.ClusterName,
				"jobId", job.JobId,
				"podName", job.PodName)
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

// extractFailureReason extracts a categorized failure reason from pod
func (r *KrknScenarioRunReconciler) extractFailureReason(pod *corev1.Pod) string {
	if len(pod.Status.ContainerStatuses) == 0 {
		return "PodNotScheduled"
	}

	cs := pod.Status.ContainerStatuses[0]
	if cs.State.Terminated != nil {
		reason := cs.State.Terminated.Reason
		exitCode := cs.State.Terminated.ExitCode

		// Categorize common failures
		if exitCode == 137 {
			return "OOMKilled"
		}
		if exitCode == 143 {
			return "SIGTERM"
		}
		if reason == "Error" {
			return "ContainerError"
		}
		return reason
	}

	if cs.State.Waiting != nil {
		return cs.State.Waiting.Reason
	}

	return "Unknown"
}

// shouldRetryJob determines if a failed job should be retried
func (r *KrknScenarioRunReconciler) shouldRetryJob(job *krknv1alpha1.ClusterJobStatus, maxRetries int) bool {
	// Don't retry if user cancelled
	if job.CancelRequested {
		return false
	}

	// Don't retry if phase is already terminal
	if job.Phase == "Succeeded" || job.Phase == "Cancelled" || job.Phase == "MaxRetriesExceeded" {
		return false
	}

	// Check retry count against max
	if maxRetries == 0 {
		maxRetries = 3 // Default
	}

	return job.RetryCount < maxRetries
}

// calculateRetryDelay calculates backoff delay based on retry count
func (r *KrknScenarioRunReconciler) calculateRetryDelay(retryCount int, backoffType, delayStr string) time.Duration {
	baseDelay := 10 * time.Second
	if delayStr != "" {
		if d, err := time.ParseDuration(delayStr); err == nil {
			baseDelay = d
		}
	}

	if backoffType == "exponential" {
		// Exponential: 10s, 20s, 40s, ...
		return baseDelay * time.Duration(1<<retryCount)
	}

	// Fixed: always same delay
	return baseDelay
}

// jobExistsForCluster checks if a job already exists for the given cluster
func (r *KrknScenarioRunReconciler) jobExistsForCluster(scenarioRun *krknv1alpha1.KrknScenarioRun, clusterName string) bool {
	for _, job := range scenarioRun.Status.ClusterJobs {
		if job.ClusterName == clusterName {
			// Don't count jobs in "Retrying" phase as existing,
			// since we need to create a new pod for them
			if job.Phase == "Retrying" {
				return false
			}
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
		case "Failed", "Cancelled", "MaxRetriesExceeded":
			failedJobs++
		case "Running", "Retrying":
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


// getKubeconfigFromProvider retrieves kubeconfig from a provider-specific Secret
func (r *KrknScenarioRunReconciler) getKubeconfigFromProvider(ctx context.Context, targetId string, providerName string, clusterName string) (string, error) {
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

	// Get the provider's clusters
	providerClusters, exists := managedClusters[providerName]
	if !exists {
		return "", fmt.Errorf("provider '%s' not found in managed-clusters", providerName)
	}

	// Check if the requested cluster exists
	clusterConfig, exists := providerClusters[clusterName]
	if !exists {
		return "", fmt.Errorf("cluster '%s' not found in %s", clusterName, providerName)
	}

	// Return the base64-encoded kubeconfig
	return clusterConfig.Kubeconfig, nil
}

// statusEqual compares two KrknScenarioRunStatus to determine if they are equal
// This is a semantic comparison that handles pointer fields correctly
func (r *KrknScenarioRunReconciler) statusEqual(old, new *krknv1alpha1.KrknScenarioRunStatus) bool {
	// Compare scalar fields
	if old.Phase != new.Phase {
		return false
	}
	if old.TotalTargets != new.TotalTargets {
		return false
	}
	if old.SuccessfulJobs != new.SuccessfulJobs {
		return false
	}
	if old.FailedJobs != new.FailedJobs {
		return false
	}
	if old.RunningJobs != new.RunningJobs {
		return false
	}

	// Compare ClusterJobs array length
	if len(old.ClusterJobs) != len(new.ClusterJobs) {
		return false
	}

	// Compare each job
	for i := range old.ClusterJobs {
		if !r.jobStatusEqual(&old.ClusterJobs[i], &new.ClusterJobs[i]) {
			return false
		}
	}

	// Compare Conditions array
	if !reflect.DeepEqual(old.Conditions, new.Conditions) {
		return false
	}

	return true
}

// jobStatusEqual compares two ClusterJobStatus semantically
func (r *KrknScenarioRunReconciler) jobStatusEqual(old, new *krknv1alpha1.ClusterJobStatus) bool {
	// Compare scalar fields
	if old.ClusterName != new.ClusterName ||
		old.JobId != new.JobId ||
		old.PodName != new.PodName ||
		old.Phase != new.Phase ||
		old.Message != new.Message ||
		old.RetryCount != new.RetryCount ||
		old.MaxRetries != new.MaxRetries ||
		old.CancelRequested != new.CancelRequested ||
		old.FailureReason != new.FailureReason {
		return false
	}

	// Compare time pointers - check if both nil or both have same value
	if !timeEqual(old.StartTime, new.StartTime) ||
		!timeEqual(old.CompletionTime, new.CompletionTime) ||
		!timeEqual(old.LastRetryTime, new.LastRetryTime) {
		return false
	}

	return true
}

// timeEqual compares two metav1.Time pointers semantically
func timeEqual(t1, t2 *metav1.Time) bool {
	if t1 == nil && t2 == nil {
		return true
	}
	if t1 == nil || t2 == nil {
		return false
	}
	// Compare the actual time values, not the pointers
	return t1.Time.Equal(t2.Time)
}

// detectStatusChanges returns a human-readable description of what changed between two statuses
func (r *KrknScenarioRunReconciler) detectStatusChanges(old, new *krknv1alpha1.KrknScenarioRunStatus) string {
	var changes []string

	// Phase change
	if old.Phase != new.Phase {
		changes = append(changes, fmt.Sprintf("phase:%s→%s", old.Phase, new.Phase))
	}

	// Job count changes
	addCountChange := func(name string, oldVal, newVal int) {
		if oldVal != newVal {
			changes = append(changes, fmt.Sprintf("%s:%d→%d", name, oldVal, newVal))
		}
	}
	addCountChange("successful", old.SuccessfulJobs, new.SuccessfulJobs)
	addCountChange("failed", old.FailedJobs, new.FailedJobs)
	addCountChange("running", old.RunningJobs, new.RunningJobs)

	// Job phase changes
	phaseChanges := countJobPhaseChanges(old.ClusterJobs, new.ClusterJobs)
	if phaseChanges > 0 {
		changes = append(changes, fmt.Sprintf("%d job phase changes", phaseChanges))
	}

	// New jobs
	if newJobs := len(new.ClusterJobs) - len(old.ClusterJobs); newJobs > 0 {
		changes = append(changes, fmt.Sprintf("+%d new jobs", newJobs))
	}

	if len(changes) == 0 {
		return "unknown changes"
	}
	return strings.Join(changes, ", ")
}

// countJobPhaseChanges counts how many jobs changed phase between old and new
func countJobPhaseChanges(oldJobs, newJobs []krknv1alpha1.ClusterJobStatus) int {
	count := 0
	minLen := len(oldJobs)
	if len(newJobs) < minLen {
		minLen = len(newJobs)
	}
	for i := 0; i < minLen; i++ {
		if oldJobs[i].Phase != newJobs[i].Phase {
			count++
		}
	}
	return count
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
