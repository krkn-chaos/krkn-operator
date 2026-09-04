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
*/

package controller

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

type KrknAIRunReconciler struct {
	client.Client
	APIReader                client.Reader
	Scheme                   *runtime.Scheme
	Clientset                kubernetes.Interface
	Namespace                string
	OrchestratorImage        string
	ResultsPVCName           string
	ResultsStorageMode       string
	ResultsStorageClassName  string
	ResultsStorageAccessMode string
	ResultsStorageSize       string
}

// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknairuns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknairuns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknairuns/finalizers,verbs=update
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknscenarioruns,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;create;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

func (r *KrknAIRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var aiRun krknv1alpha1.KrknAIRun
	if err := r.Get(ctx, req.NamespacedName, &aiRun); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if aiRun.Status.Phase == "" {
		aiRun.Status.Phase = "Pending"
		now := metav1.Now()
		aiRun.Status.StartTime = &now
		if err := r.Status().Update(ctx, &aiRun); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	switch aiRun.Status.Phase {
	case "Pending":
		if err := r.ensureProvisioned(ctx, &aiRun); err != nil {
			logger.Error(err, "failed to provision KrknAIRun")
			if statusErr := r.fail(ctx, &aiRun, err.Error()); statusErr != nil {
				return ctrl.Result{}, statusErr
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: 3 * time.Second}, nil
	case "Provisioning", "Running":
		return r.observeRun(ctx, &aiRun)
	case "Succeeded", "Failed", "Cancelled":
		return ctrl.Result{}, nil
	default:
		resultErr := r.fail(ctx, &aiRun, "unsupported phase "+aiRun.Status.Phase)
		return ctrl.Result{}, resultErr
	}
}

func (r *KrknAIRunReconciler) ensureProvisioned(ctx context.Context, aiRun *krknv1alpha1.KrknAIRun) error {
	provider, cluster, err := oneTarget(aiRun.Spec.TargetClusters)
	if err != nil {
		return err
	}

	kubeconfigBase64, err := resolveManagedClusterKubeconfig(
		ctx, r.Client, r.Namespace, aiRun.Spec.TargetRequestID, provider, cluster,
	)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}
	kubeconfig, err := base64.StdEncoding.DecodeString(kubeconfigBase64)
	if err != nil {
		return fmt.Errorf("failed to decode kubeconfig: %w", err)
	}

	configName := aiResourceName("ai", aiRun.Name, "config")
	configKey := aiRun.Spec.ConfigMapKey
	if configKey == "" {
		configKey = "krkn-ai.yaml"
	}
	if aiRun.Spec.ConfigMapName == "" {
		return fmt.Errorf("configMapName is required")
	}
	var sourceConfigMap corev1.ConfigMap
	if err := r.Get(ctx, types.NamespacedName{
		Name: aiRun.Spec.ConfigMapName, Namespace: aiRun.Namespace,
	}, &sourceConfigMap); err != nil {
		return fmt.Errorf("failed to read config ConfigMap %q: %w", aiRun.Spec.ConfigMapName, err)
	}
	configYAML, ok := sourceConfigMap.Data[configKey]
	if !ok || strings.TrimSpace(configYAML) == "" {
		return fmt.Errorf(
			"config ConfigMap %q does not contain non-empty key %q",
			aiRun.Spec.ConfigMapName, configKey,
		)
	}

	image := aiRun.Spec.OrchestratorImage
	if image == "" {
		image = r.OrchestratorImage
	}
	if image == "" {
		return fmt.Errorf("orchestrator image not configured")
	}

	kubeconfigName := aiResourceName("ai", aiRun.Name, "kubeconfig")
	podName := aiResourceName("ai-run", aiRun.Name, "")
	labels := map[string]string{"krkn.dev/ai-run": aiRun.Name}

	resultsPVCName, err := r.resolveResultsPVC(ctx, aiRun)
	if err != nil {
		return err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: configName, Namespace: aiRun.Namespace, Labels: labels},
		Data:       map[string]string{"krkn-ai.yaml": string(configYAML)},
	}
	if err := r.setOwnerAndCreate(ctx, aiRun, configMap); err != nil {
		return fmt.Errorf("failed to create config ConfigMap: %w", err)
	}

	kubeconfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: kubeconfigName, Namespace: aiRun.Namespace, Labels: labels},
		Data:       map[string]string{"config": string(kubeconfig)},
	}
	if err := r.setOwnerAndCreate(ctx, aiRun, kubeconfigMap); err != nil {
		return fmt.Errorf("failed to create kubeconfig ConfigMap: %w", err)
	}

	pod := buildOrchestratorPod(aiRun, podName, image, provider, cluster, configName, kubeconfigName, resultsPVCName, r.Namespace)
	if err := r.setOwnerAndCreate(ctx, aiRun, pod); err != nil {
		return fmt.Errorf("failed to create orchestrator Pod: %w", err)
	}

	changed := aiRun.Status.Phase != "Provisioning" ||
		aiRun.Status.PVCName != resultsPVCName || aiRun.Status.OrchestratorPodName != podName
	aiRun.Status.Phase = "Provisioning"
	aiRun.Status.PVCName = resultsPVCName
	aiRun.Status.OrchestratorPodName = podName
	if changed {
		return r.Status().Update(ctx, aiRun)
	}
	return nil
}

func (r *KrknAIRunReconciler) validateExistingResultsPVC(
	ctx context.Context, namespace, name, expectedAccessMode string,
) error {
	var pvc corev1.PersistentVolumeClaim
	reader := r.APIReader
	if reader == nil {
		reader = r.Client
	}
	if err := reader.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &pvc); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf(`results PVC "%s" not found in namespace "%s"`, name, namespace)
		}
		return fmt.Errorf(`failed to get results PVC "%s": %w`, name, err)
	}
	if pvc.DeletionTimestamp != nil {
		return fmt.Errorf(`results PVC "%s" is being deleted`, name)
	}
	if expectedAccessMode != "" && !hasAccessMode(pvc.Spec.AccessModes, expectedAccessMode) {
		return fmt.Errorf(`results PVC "%s" must declare %s access mode`, name, expectedAccessMode)
	}
	return nil
}

func hasAccessMode(modes []corev1.PersistentVolumeAccessMode, expected string) bool {
	for _, mode := range modes {
		if string(mode) == expected {
			return true
		}
	}
	return false
}

const (
	storageModeShared    = "shared"
	storageModeDedicated = "dedicated"
)

func (r *KrknAIRunReconciler) resolveResultsPVC(ctx context.Context, aiRun *krknv1alpha1.KrknAIRun) (string, error) {
	spec := aiRun.Spec.Storage
	if spec != nil && spec.PVCName != "" {
		if spec.StorageClassName != "" || spec.Size != "" {
			return "", fmt.Errorf("storage pvcName cannot be combined with storageClassName or size")
		}
		return spec.PVCName, r.validateExistingResultsPVC(ctx, aiRun.Namespace, spec.PVCName, spec.AccessMode)
	}

	mode := r.ResultsStorageMode
	if mode == "" {
		mode = storageModeShared
	}
	if spec != nil && (spec.StorageClassName != "" || spec.AccessMode != "" || spec.Size != "") {
		mode = storageModeDedicated
	}

	switch mode {
	case storageModeShared:
		name := r.ResultsPVCName
		if name == "" {
			name = "krkn-ai-results"
		}
		expectedAccessMode := r.installationAccessMode()
		if !validAccessMode(expectedAccessMode) {
			return "", fmt.Errorf(`invalid results storage access mode "%s": must be %q or %q`, expectedAccessMode, corev1.ReadWriteOnce, corev1.ReadWriteMany)
		}
		return name, r.validateExistingResultsPVC(ctx, aiRun.Namespace, name, expectedAccessMode)
	case storageModeDedicated:
		return r.ensureDedicatedResultsPVC(ctx, aiRun)
	default:
		return "", fmt.Errorf(`unsupported results storage mode "%s": must be %q or %q`, mode, storageModeShared, storageModeDedicated)
	}
}

func (r *KrknAIRunReconciler) ensureDedicatedResultsPVC(
	ctx context.Context, aiRun *krknv1alpha1.KrknAIRun,
) (string, error) {
	spec := aiRun.Spec.Storage
	storageClassName := r.ResultsStorageClassName
	accessMode := r.installationAccessMode()
	size := r.ResultsStorageSize
	if spec != nil {
		if spec.StorageClassName != "" {
			storageClassName = spec.StorageClassName
		}
		if spec.AccessMode != "" {
			accessMode = spec.AccessMode
		}
		if spec.Size != "" {
			size = spec.Size
		}
	}
	if size == "" {
		size = "1Gi"
	}
	if !validAccessMode(accessMode) {
		return "", fmt.Errorf(`invalid results storage access mode "%s": must be %q or %q`, accessMode, corev1.ReadWriteOnce, corev1.ReadWriteMany)
	}
	storage, err := resource.ParseQuantity(size)
	if err != nil {
		return "", fmt.Errorf("invalid results storage size %q: %w", size, err)
	}

	name := aiResourceName("ai", aiRun.Name, "results")
	var pvc corev1.PersistentVolumeClaim
	reader := r.APIReader
	if reader == nil {
		reader = r.Client
	}
	err = reader.Get(ctx, types.NamespacedName{Name: name, Namespace: aiRun.Namespace}, &pvc)
	if apierrors.IsNotFound(err) {
		pvc = corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: aiRun.Namespace, Labels: map[string]string{
				"krkn.dev/ai-run": aiRun.Name,
			}},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.PersistentVolumeAccessMode(accessMode)},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceStorage: storage},
				},
			},
		}
		if storageClassName != "" {
			pvc.Spec.StorageClassName = &storageClassName
		}
		if err := controllerutil.SetControllerReference(aiRun, &pvc, r.Scheme); err != nil {
			return "", err
		}
		if err := r.Create(ctx, &pvc); err != nil && !apierrors.IsAlreadyExists(err) {
			return "", fmt.Errorf("failed to create results PVC %q: %w", name, err)
		}
	} else if err != nil {
		return "", fmt.Errorf(`failed to get results PVC "%s": %w`, name, err)
	} else if !metav1.IsControlledBy(&pvc, aiRun) {
		return "", fmt.Errorf(`dedicated results PVC "%s" already exists and is not owned by KrknAIRun "%s"`, name, aiRun.Name)
	}

	if err := r.validateExistingResultsPVC(ctx, aiRun.Namespace, name, accessMode); err != nil {
		return "", err
	}
	return name, nil
}

func (r *KrknAIRunReconciler) installationAccessMode() string {
	mode := r.ResultsStorageMode
	if mode == "" {
		mode = storageModeShared
	}
	return r.effectiveAccessMode(mode, r.ResultsStorageAccessMode)
}

func (r *KrknAIRunReconciler) effectiveAccessMode(mode, configured string) string {
	if configured != "" {
		return configured
	}
	if mode == storageModeDedicated {
		return string(corev1.ReadWriteOnce)
	}
	return string(corev1.ReadWriteMany)
}

func validAccessMode(mode string) bool {
	return mode == string(corev1.ReadWriteOnce) || mode == string(corev1.ReadWriteMany)
}

func (r *KrknAIRunReconciler) observeRun(ctx context.Context, aiRun *krknv1alpha1.KrknAIRun) (ctrl.Result, error) {
	var pod corev1.Pod
	err := r.Get(ctx, types.NamespacedName{Name: aiRun.Status.OrchestratorPodName, Namespace: aiRun.Namespace}, &pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	var children krknv1alpha1.KrknScenarioRunList
	if err := r.List(ctx, &children, client.MatchingLabels{"krkn.dev/ai-run": aiRun.Name}); err != nil {
		return ctrl.Result{}, err
	}
	refs := make([]string, 0, len(children.Items))
	for i := range children.Items {
		refs = append(refs, children.Items[i].Name)
	}

	oldStatus := aiRun.Status.DeepCopy()
	aiRun.Status.ScenarioRunRefs = refs
	if pod.Status.Phase == corev1.PodRunning {
		aiRun.Status.Phase = "Running"
	}
	if terminated := orchestratorTermination(&pod); terminated != nil {
		aiRun.Status.CompletionTime = completionTime(aiRun.Status.CompletionTime)
		if terminated.ExitCode == 0 {
			aiRun.Status.Phase = "Succeeded"
		} else {
			aiRun.Status.Phase = "Failed"
			aiRun.Status.FailureReason = terminationReason(terminated)
		}
	} else if pod.Status.Phase == corev1.PodFailed {
		aiRun.Status.Phase = "Failed"
		aiRun.Status.CompletionTime = completionTime(aiRun.Status.CompletionTime)
		aiRun.Status.FailureReason = "orchestrator pod failed"
	}

	if !reflect.DeepEqual(oldStatus, &aiRun.Status) {
		if err := r.Status().Update(ctx, aiRun); err != nil {
			return ctrl.Result{}, err
		}
	}
	if aiRun.Status.Phase == "Succeeded" || aiRun.Status.Phase == "Failed" {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *KrknAIRunReconciler) fail(ctx context.Context, aiRun *krknv1alpha1.KrknAIRun, reason string) error {
	aiRun.Status.Phase = "Failed"
	aiRun.Status.FailureReason = reason
	aiRun.Status.CompletionTime = completionTime(aiRun.Status.CompletionTime)
	return r.Status().Update(ctx, aiRun)
}

func (r *KrknAIRunReconciler) setOwnerAndCreate(ctx context.Context, owner client.Object, object client.Object) error {
	if err := controllerutil.SetControllerReference(owner, object, r.Scheme); err != nil {
		return err
	}
	if err := r.Create(ctx, object); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func oneTarget(targets map[string][]string) (string, string, error) {
	if len(targets) != 1 {
		return "", "", fmt.Errorf("exactly one target provider and cluster is required")
	}
	for provider, clusters := range targets {
		if len(clusters) != 1 || clusters[0] == "" {
			return "", "", fmt.Errorf("exactly one target provider and cluster is required")
		}
		return provider, clusters[0], nil
	}
	return "", "", fmt.Errorf("exactly one target provider and cluster is required")
}

func aiResourceName(prefix, runName, suffix string) string {
	name := prefix + "-" + runName
	if suffix != "" {
		name += "-" + suffix
	}
	if len(name) > 63 {
		name = strings.TrimSuffix(name[:63], "-")
	}
	return name
}

func completionTime(existing *metav1.Time) *metav1.Time {
	if existing != nil {
		return existing
	}
	now := metav1.Now()
	return &now
}

func orchestratorTermination(pod *corev1.Pod) *corev1.ContainerStateTerminated {
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == "orchestrator" && status.State.Terminated != nil {
			return status.State.Terminated
		}
	}
	return nil
}

func terminationReason(terminated *corev1.ContainerStateTerminated) string {
	if terminated.Reason != "" {
		if terminated.Message != "" {
			return terminated.Reason + ": " + terminated.Message
		}
		return terminated.Reason
	}
	if terminated.Message != "" {
		return terminated.Message
	}
	return fmt.Sprintf("orchestrator exited with code %d", terminated.ExitCode)
}

func buildOrchestratorPod(
	aiRun *krknv1alpha1.KrknAIRun,
	podName, image, provider, cluster, configName, kubeconfigName, pvcName, namespace string,
) *corev1.Pod {
	runAs := int64(1001)
	activeDeadline := aiRun.Spec.ActiveDeadlineSeconds
	if activeDeadline == 0 {
		activeDeadline = 21600
	}
	envs := []corev1.EnvVar{
		{Name: "MODE", Value: "run"},
		{Name: "CONFIG_FILE", Value: "/input/krkn-ai.yaml"},
		{Name: "KUBECONFIG", Value: "/input/kubeconfig"},
		{Name: "OUTPUT_DIR", Value: "/output"},
		{Name: "FORMAT", Value: "yaml"},
		{Name: "RUNNER_TYPE", Value: "operator"},
		{Name: "VERBOSE", Value: "2"},
		{Name: "KRKNAI_NAMESPACE", Value: namespace},
		{Name: "KRKNAI_RUN_NAME", Value: aiRun.Name},
		{Name: "KRKNAI_RUN_UID", Value: string(aiRun.UID)},
		{Name: "KRKNAI_TARGET_REQUEST_ID", Value: aiRun.Spec.TargetRequestID},
		{Name: "KRKNAI_PROVIDER", Value: provider},
		{Name: "KRKNAI_CLUSTER", Value: cluster},
	}
	if aiRun.Spec.PrometheusURL != "" {
		envs = append(envs, corev1.EnvVar{Name: "PROMETHEUS_URL", Value: aiRun.Spec.PrometheusURL})
	}
	if aiRun.Spec.PrometheusTokenSecretRef != "" {
		envs = append(envs, corev1.EnvVar{
			Name: "PROMETHEUS_TOKEN",
			ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: aiRun.Spec.PrometheusTokenSecretRef},
				Key:                  "token",
			}},
		})
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: aiRun.Namespace, Labels: map[string]string{"krkn.dev/ai-run": aiRun.Name}},
		Spec: corev1.PodSpec{
			ServiceAccountName:    "krkn-operator-krkn-ai-orchestrator",
			RestartPolicy:         corev1.RestartPolicyNever,
			ActiveDeadlineSeconds: &activeDeadline,
			SecurityContext:       &corev1.PodSecurityContext{RunAsUser: &runAs, RunAsGroup: &runAs, FSGroup: &runAs},
			Containers: []corev1.Container{{
				Name:            "orchestrator",
				Image:           image,
				ImagePullPolicy: corev1.PullAlways,
				Env:             envs,
				VolumeMounts: []corev1.VolumeMount{
					{Name: "config", MountPath: "/input/krkn-ai.yaml", SubPath: "krkn-ai.yaml"},
					{Name: "kubeconfig", MountPath: "/input/kubeconfig", SubPath: "config"},
					{Name: "results", MountPath: "/output"},
					{Name: "tmp", MountPath: "/tmp"},
				},
			}},
			Volumes: []corev1.Volume{
				{Name: "config", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: configName}}}},
				{Name: "kubeconfig", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: kubeconfigName}}}},
				{Name: "results", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName}}},
				{Name: "tmp", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
			},
		},
	}
}

func (r *KrknAIRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&krknv1alpha1.KrknAIRun{}).
		Owns(&corev1.Pod{}).
		Owns(&krknv1alpha1.KrknScenarioRun{}).
		Complete(r)
}
