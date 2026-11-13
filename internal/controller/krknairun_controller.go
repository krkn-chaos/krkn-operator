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
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	"github.com/krkn-chaos/krkn-operator/pkg/krknaiserver"
	registryutil "github.com/krkn-chaos/krkn-operator/pkg/registry"
	krknctlmodels "github.com/krkn-chaos/krknctl/pkg/provider/models"
	"github.com/krkn-chaos/krknctl/pkg/verify"
)

// KrknAIRunReconciler drives a run from Pending through provisioning and Pod
// observation to a terminal phase, enforcing the global image signature policy.
// The shared artifact service and orchestrator ServiceAccount are installation resources.
type KrknAIRunReconciler struct {
	client.Client
	APIReader                      client.Reader
	Scheme                         *runtime.Scheme
	Clientset                      kubernetes.Interface
	Namespace                      string
	OrchestratorImage              string
	OrchestratorImagePullPolicy    corev1.PullPolicy
	ServiceImage                   string
	ServiceImagePullPolicy         corev1.PullPolicy
	ServiceURL                     string
	ServiceTokenSecretName         string
	OrchestratorServiceAccountName string
	ImagePullSecrets               []corev1.LocalObjectReference
	RunPodConfig                   KrknAIRunPodConfig
	imageVerifier                  func(context.Context, string) (verify.VerifiedImage, error)
	registryInspector              func(context.Context, *corev1.Secret, []string, map[string]string) (*registryutil.AIImageInventory, error)
	// RejectNewRuns blocks unstarted runs without stopping existing runs or finalizer cleanup.
	RejectNewRuns bool
}

// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknairuns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknairuns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknairuns/finalizers,verbs=update
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknscenarioruns,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get

const (
	defaultKrknAIOrchestratorServiceAccount = "krkn-operator-krkn-ai-orchestrator"
	aiRunPhasePending                       = "Pending"
	aiRunPhaseProvisioning                  = "Provisioning"
	aiRunPhaseRunning                       = "Running"
	aiRunPhaseCancelling                    = "Cancelling"
	aiRunPhaseSucceeded                     = "Succeeded"
	aiRunPhaseFailed                        = "Failed"
	aiRunPhaseCancelled                     = "Cancelled"
	krknAIRunFinalizer                      = "krkn.krkn-chaos.dev/ai-run-cleanup"
	krknAIRunChildUIDsAnnotation            = "krkn.krkn-chaos.dev/ai-run-cleanup-child-uids"
	krknAIRunDeletionCondition              = "DeletionCleanup"
	krknAIRunDeleteTimeout                  = 15 * time.Second
)

var errAIRunCancellationRequested = errors.New("KrknAIRun cancellation requested")

// Reconcile drives provisioning and observation to a terminal phase. Invalid
// configuration or image trust fails the run; operational errors remain retryable.
func (r *KrknAIRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var aiRun krknv1alpha1.KrknAIRun
	if err := r.Get(ctx, req.NamespacedName, &aiRun); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !aiRun.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&aiRun, krknAIRunFinalizer) {
			return ctrl.Result{}, nil
		}
		return r.reconcileDeletion(ctx, &aiRun)
	}

	if aiRun.Status.Phase == aiRunPhaseCancelling || aiRun.Status.Phase == aiRunPhaseCancelled ||
		(aiRun.Spec.CancelRequested && aiRun.Status.Phase != aiRunPhaseSucceeded && aiRun.Status.Phase != aiRunPhaseFailed) {
		if !controllerutil.ContainsFinalizer(&aiRun, krknAIRunFinalizer) {
			controllerutil.AddFinalizer(&aiRun, krknAIRunFinalizer)
			if err := r.Update(ctx, &aiRun); err != nil {
				return ctrl.Result{}, fmt.Errorf("add KrknAIRun cleanup finalizer: %w", err)
			}
			return ctrl.Result{}, nil
		}
		return r.reconcileCancellation(ctx, &aiRun)
	}

	if err := r.retainTargetIdentity(ctx, &aiRun); err != nil {
		return ctrl.Result{}, err
	}

	if r.RejectNewRuns && aiRun.Status.OrchestratorPodName == "" &&
		(aiRun.Status.Phase == "" || aiRun.Status.Phase == aiRunPhasePending) {
		return ctrl.Result{}, r.fail(ctx, &aiRun, "Krkn-AI is disabled; new runs are not accepted")
	}
	if !controllerutil.ContainsFinalizer(&aiRun, krknAIRunFinalizer) {
		controllerutil.AddFinalizer(&aiRun, krknAIRunFinalizer)
		if err := r.Update(ctx, &aiRun); err != nil {
			return ctrl.Result{}, fmt.Errorf("add KrknAIRun cleanup finalizer: %w", err)
		}
		return ctrl.Result{}, nil
	}

	if aiRun.Status.Phase == "" {
		aiRun.Status.Phase = aiRunPhasePending
		now := metav1.Now()
		aiRun.Status.StartTime = &now
		if err := r.Status().Update(ctx, &aiRun); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, nil
	}

	switch aiRun.Status.Phase {
	case aiRunPhasePending:
		if err := r.ensureProvisioned(ctx, &aiRun); err != nil {
			if errors.Is(err, errAIRunCancellationRequested) {
				aiRun.Spec.CancelRequested = true
				return r.reconcileCancellation(ctx, &aiRun)
			}
			logger.Error(err, "failed to provision KrknAIRun")
			var configErr *configurationError
			if !errors.As(err, &configErr) {
				return ctrl.Result{}, err
			}
			if statusErr := r.fail(ctx, &aiRun, err.Error()); statusErr != nil {
				return ctrl.Result{}, statusErr
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: 3 * time.Second}, nil
	case aiRunPhaseProvisioning, aiRunPhaseRunning:
		return r.observeRun(ctx, &aiRun)
	case aiRunPhaseCancelling, aiRunPhaseCancelled:
		return r.reconcileCancellation(ctx, &aiRun)
	case aiRunPhaseSucceeded, aiRunPhaseFailed:
		return ctrl.Result{}, nil
	default:
		resultErr := r.fail(ctx, &aiRun, "unsupported phase "+aiRun.Status.Phase)
		return ctrl.Result{}, resultErr
	}
}

func (r *KrknAIRunReconciler) reconcileCancellation(
	ctx context.Context,
	aiRun *krknv1alpha1.KrknAIRun,
) (ctrl.Result, error) {
	var children krknv1alpha1.KrknScenarioRunList
	if err := r.List(ctx, &children, client.InNamespace(aiRun.Namespace)); err != nil {
		return ctrl.Result{}, fmt.Errorf("list KrknScenarioRuns for cancellation: %w", err)
	}
	for i := range children.Items {
		child := &children.Items[i]
		if !hasAIRunOwner(child.OwnerReferences, aiRun.UID) || child.Spec.CancelRequested {
			continue
		}
		child.Spec.CancelRequested = true
		if err := r.Update(ctx, child); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, nil
			}
			return ctrl.Result{}, fmt.Errorf("request cancellation of owned KrknScenarioRun %q: %w", child.Name, err)
		}
	}
	return r.observeRun(ctx, aiRun)
}

func (r *KrknAIRunReconciler) reconcileDeletion(
	ctx context.Context,
	aiRun *krknv1alpha1.KrknAIRun,
) (ctrl.Result, error) {
	pending, err := r.deleteOwnedRunPods(ctx, aiRun)
	if err != nil {
		return r.deletionFailure(ctx, aiRun, err)
	}
	if pending {
		return r.deletionPending(ctx, aiRun, "Waiting for the Krkn-AI orchestrator Pod to stop.")
	}

	var children krknv1alpha1.KrknScenarioRunList
	if err := r.List(ctx, &children, client.InNamespace(aiRun.Namespace)); err != nil {
		return r.deletionFailure(ctx, aiRun, fmt.Errorf("list KrknScenarioRuns for cleanup: %w", err))
	}
	childUIDs := parseUIDAnnotation(aiRun.Annotations[krknAIRunChildUIDsAnnotation])
	ownedChildCount := 0
	for i := range children.Items {
		child := &children.Items[i]
		if hasAIRunOwner(child.OwnerReferences, aiRun.UID) {
			ownedChildCount++
			if child.UID != "" {
				childUIDs = append(childUIDs, child.UID)
			}
		}
	}
	childUIDs = uniqueUIDs(childUIDs)
	if !sameUIDs(parseUIDAnnotation(aiRun.Annotations[krknAIRunChildUIDsAnnotation]), childUIDs) {
		if aiRun.Annotations == nil {
			aiRun.Annotations = make(map[string]string)
		}
		aiRun.Annotations[krknAIRunChildUIDsAnnotation] = formatUIDAnnotation(childUIDs)
		if err := r.Update(ctx, aiRun); err != nil {
			return r.deletionFailure(ctx, aiRun, fmt.Errorf("save KrknScenarioRun owner UIDs for cleanup: %w", err))
		}
		return r.deletionPending(ctx, aiRun, "Recording KrknScenarioRun owners before deleting their workloads.")
	}

	for i := range children.Items {
		child := &children.Items[i]
		if !hasAIRunOwner(child.OwnerReferences, aiRun.UID) {
			continue
		}
		uid := child.UID
		if err := r.Delete(ctx, child, client.Preconditions{UID: &uid}); err != nil && !apierrors.IsNotFound(err) {
			return r.deletionFailure(ctx, aiRun, fmt.Errorf("delete owned KrknScenarioRun %q: %w", child.Name, err))
		}
	}

	pending, err = r.deleteScenarioRunWorkloads(ctx, aiRun.Namespace, childUIDs)
	if err != nil {
		return r.deletionFailure(ctx, aiRun, err)
	}
	if pending {
		return r.deletionPending(ctx, aiRun, "Waiting for owned KrknScenarioRun Pods and ConfigMaps to stop.")
	}

	if ownedChildCount > 0 {
		return r.deletionPending(ctx, aiRun, "Waiting for owned KrknScenarioRuns to be deleted.")
	}

	pending, err = r.deleteOwnedRunConfigMaps(ctx, aiRun)
	if err != nil {
		return r.deletionFailure(ctx, aiRun, err)
	}
	if pending {
		return r.deletionPending(ctx, aiRun, "Waiting for the owned kubeconfig ConfigMap to be deleted.")
	}

	if err := r.deleteArtifactRun(ctx, aiRun); err != nil {
		return r.deletionFailure(ctx, aiRun, err)
	}
	meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
		Type:               krknAIRunDeletionCondition,
		Status:             metav1.ConditionTrue,
		Reason:             "CleanupSucceeded",
		Message:            "Owned workloads and uploaded run artifacts were deleted.",
		ObservedGeneration: aiRun.Generation,
	})
	if err := r.Status().Update(ctx, aiRun); err != nil {
		return ctrl.Result{}, fmt.Errorf("record KrknAIRun cleanup success: %w", err)
	}
	if aiRun.Annotations != nil {
		delete(aiRun.Annotations, krknAIRunChildUIDsAnnotation)
		if len(aiRun.Annotations) == 0 {
			aiRun.Annotations = nil
		}
	}
	controllerutil.RemoveFinalizer(aiRun, krknAIRunFinalizer)
	if err := r.Update(ctx, aiRun); err != nil {
		return ctrl.Result{}, fmt.Errorf("remove KrknAIRun cleanup finalizer: %w", err)
	}
	return ctrl.Result{}, nil
}

func (r *KrknAIRunReconciler) deleteOwnedRunPods(
	ctx context.Context,
	aiRun *krknv1alpha1.KrknAIRun,
) (bool, error) {
	var pods corev1.PodList
	if err := r.List(ctx, &pods, client.InNamespace(aiRun.Namespace)); err != nil {
		return false, fmt.Errorf("list Pods owned by KrknAIRun: %w", err)
	}
	found := false
	for i := range pods.Items {
		pod := &pods.Items[i]
		if !hasAIRunOwner(pod.OwnerReferences, aiRun.UID) {
			continue
		}
		found = true
		uid := pod.UID
		if err := r.Delete(ctx, pod, client.Preconditions{UID: &uid}); err != nil && !apierrors.IsNotFound(err) {
			return false, fmt.Errorf("delete owned orchestrator Pod %q: %w", pod.Name, err)
		}
	}
	return found, nil
}

func (r *KrknAIRunReconciler) deleteScenarioRunWorkloads(
	ctx context.Context,
	namespace string,
	childUIDs []types.UID,
) (bool, error) {
	if len(childUIDs) == 0 {
		return false, nil
	}
	uidSet := make(map[types.UID]struct{}, len(childUIDs))
	for _, uid := range childUIDs {
		uidSet[uid] = struct{}{}
	}
	pending := false
	var pods corev1.PodList
	if err := r.List(ctx, &pods, client.InNamespace(namespace)); err != nil {
		return false, fmt.Errorf("list Pods owned by KrknScenarioRuns: %w", err)
	}
	for i := range pods.Items {
		pod := &pods.Items[i]
		if !hasScenarioRunOwner(pod.OwnerReferences, uidSet) {
			continue
		}
		pending = true
		uid := pod.UID
		if err := r.Delete(ctx, pod, client.Preconditions{UID: &uid}); err != nil && !apierrors.IsNotFound(err) {
			return false, fmt.Errorf("delete owned scenario Pod %q: %w", pod.Name, err)
		}
	}
	var configMaps corev1.ConfigMapList
	if err := r.List(ctx, &configMaps, client.InNamespace(namespace)); err != nil {
		return false, fmt.Errorf("list ConfigMaps owned by KrknScenarioRuns: %w", err)
	}
	for i := range configMaps.Items {
		configMap := &configMaps.Items[i]
		if !hasScenarioRunOwner(configMap.OwnerReferences, uidSet) {
			continue
		}
		pending = true
		uid := configMap.UID
		if err := r.Delete(ctx, configMap, client.Preconditions{UID: &uid}); err != nil && !apierrors.IsNotFound(err) {
			return false, fmt.Errorf("delete owned scenario ConfigMap %q: %w", configMap.Name, err)
		}
	}
	return pending, nil
}

func (r *KrknAIRunReconciler) deleteOwnedRunConfigMaps(
	ctx context.Context,
	aiRun *krknv1alpha1.KrknAIRun,
) (bool, error) {
	var configMaps corev1.ConfigMapList
	if err := r.List(ctx, &configMaps, client.InNamespace(aiRun.Namespace)); err != nil {
		return false, fmt.Errorf("list ConfigMaps owned by KrknAIRun: %w", err)
	}
	found := false
	for i := range configMaps.Items {
		configMap := &configMaps.Items[i]
		if !hasAIRunOwner(configMap.OwnerReferences, aiRun.UID) {
			continue
		}
		found = true
		uid := configMap.UID
		if err := r.Delete(ctx, configMap, client.Preconditions{UID: &uid}); err != nil && !apierrors.IsNotFound(err) {
			return false, fmt.Errorf("delete owned run ConfigMap %q: %w", configMap.Name, err)
		}
	}
	return found, nil
}

func (r *KrknAIRunReconciler) deleteArtifactRun(
	ctx context.Context,
	aiRun *krknv1alpha1.KrknAIRun,
) error {
	var tokenSecret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Name: r.ServiceTokenSecretName, Namespace: aiRun.Namespace,
	}, &tokenSecret); err != nil {
		return fmt.Errorf("read artifact service token Secret: %w", err)
	}
	token := string(tokenSecret.Data["token"])
	if token == "" {
		return fmt.Errorf("artifact service token Secret has no non-empty token key")
	}
	endpoint := krknaiserver.EscapePath("/v1/runs/" + string(aiRun.UID))
	requestCtx, cancel := context.WithTimeout(ctx, krknAIRunDeleteTimeout)
	defer cancel()
	response, err := krknaiserver.New(r.ServiceURL, token).Do(
		requestCtx, http.MethodDelete, endpoint, nil,
	)
	if err != nil {
		return fmt.Errorf("delete uploaded artifacts for run UID %q: %w", aiRun.UID, err)
	}
	closeErr := response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("artifact service DELETE for run UID %q returned HTTP %d", aiRun.UID, response.StatusCode)
	}
	if closeErr != nil {
		return fmt.Errorf("close artifact run deletion response: %w", closeErr)
	}
	return nil
}

func (r *KrknAIRunReconciler) deletionPending(
	ctx context.Context,
	aiRun *krknv1alpha1.KrknAIRun,
	message string,
) (ctrl.Result, error) {
	meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
		Type:               krknAIRunDeletionCondition,
		Status:             metav1.ConditionFalse,
		Reason:             "CleanupPending",
		Message:            message,
		ObservedGeneration: aiRun.Generation,
	})
	if err := r.Status().Update(ctx, aiRun); err != nil {
		return ctrl.Result{}, fmt.Errorf("record KrknAIRun cleanup progress: %w", err)
	}
	return ctrl.Result{RequeueAfter: time.Second}, nil
}

func (r *KrknAIRunReconciler) deletionFailure(
	ctx context.Context,
	aiRun *krknv1alpha1.KrknAIRun,
	cleanupErr error,
) (ctrl.Result, error) {
	meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
		Type:               krknAIRunDeletionCondition,
		Status:             metav1.ConditionFalse,
		Reason:             "CleanupFailed",
		Message:            cleanupErr.Error(),
		ObservedGeneration: aiRun.Generation,
	})
	if err := r.Status().Update(ctx, aiRun); err != nil {
		return ctrl.Result{}, fmt.Errorf("%v; record KrknAIRun cleanup failure: %w", cleanupErr, err)
	}
	return ctrl.Result{}, cleanupErr
}

func hasAIRunOwner(owners []metav1.OwnerReference, uid types.UID) bool {
	for _, owner := range owners {
		if owner.APIVersion == krknv1alpha1.GroupVersion.String() &&
			owner.Kind == "KrknAIRun" && owner.UID == uid {
			return true
		}
	}
	return false
}

func hasScenarioRunOwner(owners []metav1.OwnerReference, uids map[types.UID]struct{}) bool {
	for _, owner := range owners {
		if owner.APIVersion != krknv1alpha1.GroupVersion.String() || owner.Kind != "KrknScenarioRun" {
			continue
		}
		if _, ok := uids[owner.UID]; ok {
			return true
		}
	}
	return false
}

func parseUIDAnnotation(value string) []types.UID {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	uids := make([]types.UID, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			uids = append(uids, types.UID(part))
		}
	}
	return uniqueUIDs(uids)
}

func uniqueUIDs(uids []types.UID) []types.UID {
	sort.Slice(uids, func(i, j int) bool { return uids[i] < uids[j] })
	unique := uids[:0]
	for _, uid := range uids {
		if uid == "" || (len(unique) > 0 && unique[len(unique)-1] == uid) {
			continue
		}
		unique = append(unique, uid)
	}
	return unique
}

func sameUIDs(a, b []types.UID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func formatUIDAnnotation(uids []types.UID) string {
	parts := make([]string, len(uids))
	for i, uid := range uids {
		parts[i] = string(uid)
	}
	return strings.Join(parts, ",")
}

func (r *KrknAIRunReconciler) isAIRunCancellationRequested(
	ctx context.Context,
	aiRun *krknv1alpha1.KrknAIRun,
) (bool, error) {
	var current krknv1alpha1.KrknAIRun
	if err := r.Get(ctx, types.NamespacedName{Name: aiRun.Name, Namespace: aiRun.Namespace}, &current); err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}
	if current.UID != aiRun.UID {
		return true, nil
	}
	return current.Spec.CancelRequested ||
		current.Status.Phase == aiRunPhaseCancelling ||
		current.Status.Phase == aiRunPhaseCancelled, nil
}
func (r *KrknAIRunReconciler) ensureProvisioned(ctx context.Context, aiRun *krknv1alpha1.KrknAIRun) error {
	if cancelled, err := r.isAIRunCancellationRequested(ctx, aiRun); err != nil {
		return err
	} else if cancelled {
		return errAIRunCancellationRequested
	}
	provider, cluster, err := oneTarget(aiRun.Spec.TargetClusters)
	if err != nil {
		return &configurationError{err: err}
	}
	if aiRun.Spec.TargetRequestID == "" {
		return &configurationError{err: fmt.Errorf("targetRequestId is required")}
	}

	configKey := aiRun.Spec.ConfigMapKey
	if configKey == "" {
		configKey = files.KrknAIConfigFileName
	}
	if aiRun.Spec.ConfigMapName == "" {
		return &configurationError{err: fmt.Errorf("configMapName is required")}
	}

	if r.OrchestratorImage == "" && aiRun.Spec.RegistryName == "" {
		return &configurationError{err: fmt.Errorf("orchestrator image not configured")}
	}
	if r.ServiceImage == "" || r.ServiceURL == "" || r.ServiceTokenSecretName == "" {
		return &configurationError{err: fmt.Errorf("artifact service is not configured")}
	}

	serviceAccountName := r.OrchestratorServiceAccountName
	if serviceAccountName == "" {
		serviceAccountName = defaultKrknAIOrchestratorServiceAccount
	}
	var serviceAccount corev1.ServiceAccount
	if err := r.APIReader.Get(ctx, types.NamespacedName{
		Name: serviceAccountName, Namespace: aiRun.Namespace,
	}, &serviceAccount); err != nil {
		return requiredResourceError(fmt.Errorf("orchestrator ServiceAccount %q is unavailable: %w", serviceAccountName, err))
	}

	kubeconfigBase64, err := resolveManagedClusterKubeconfig(
		ctx, r.APIReader, r.Namespace, aiRun.Spec.TargetRequestID, provider, cluster,
	)
	if err != nil {
		return requiredResourceError(fmt.Errorf("failed to get kubeconfig: %w", err))
	}
	kubeconfig, err := base64.StdEncoding.DecodeString(kubeconfigBase64)
	if err != nil {
		return &configurationError{err: fmt.Errorf("failed to decode kubeconfig: %w", err)}
	}

	var sourceConfigMap corev1.ConfigMap
	if err := r.APIReader.Get(ctx, types.NamespacedName{
		Name: aiRun.Spec.ConfigMapName, Namespace: aiRun.Namespace,
	}, &sourceConfigMap); err != nil {
		return requiredResourceError(fmt.Errorf("failed to read config ConfigMap %q: %w", aiRun.Spec.ConfigMapName, err))
	}
	configYAML, ok := sourceConfigMap.Data[configKey]
	if !ok || strings.TrimSpace(configYAML) == "" {
		return &configurationError{err: fmt.Errorf(
			"config ConfigMap %q does not contain non-empty key %q",
			aiRun.Spec.ConfigMapName, configKey,
		)}
	}
	configName := aiRun.Spec.ConfigMapName

	registryImages, err := r.resolveRunRegistry(ctx, aiRun, configYAML)
	if err != nil {
		return err
	}
	image, serviceImage, err := r.verifiedImages(ctx, registryImages.orchestrator, registryImages.uploader, registryImages.registry)
	if err != nil {
		return err
	}

	podName := aiRun.Status.OrchestratorPodName
	if podName == "" {
		podName = aiOrchestratorPodName(aiRun.Name)
		aiRun.Status.OrchestratorPodName = podName
		if err := r.Status().Update(ctx, aiRun); err != nil {
			return fmt.Errorf("failed to reserve orchestrator Pod name: %w", err)
		}
	}

	kubeconfigName := aiResourceName("ai", aiRun.Name, "kubeconfig")
	labels := map[string]string{"krkn.dev/ai-run": aiRunLabelValue(aiRun.Name)}

	kubeconfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: kubeconfigName, Namespace: aiRun.Namespace, Labels: labels},
		Data:       map[string]string{"config": string(kubeconfig)},
	}
	if cancelled, err := r.isAIRunCancellationRequested(ctx, aiRun); err != nil {
		return err
	} else if cancelled {
		return errAIRunCancellationRequested
	}
	if err := r.setOwnerAndCreate(ctx, aiRun, kubeconfigMap); err != nil {
		return fmt.Errorf("failed to create kubeconfig ConfigMap: %w", err)
	}

	pod := buildOrchestratorPod(
		aiRun, podName, image, provider, cluster, configName, configKey, kubeconfigName,
		krknAIPodOptions{
			namespace:                   r.Namespace,
			serviceImage:                serviceImage,
			serviceURL:                  r.ServiceURL,
			serviceTokenSecretName:      r.ServiceTokenSecretName,
			serviceAccountName:          serviceAccountName,
			orchestratorImagePullPolicy: r.OrchestratorImagePullPolicy,
			serviceImagePullPolicy:      r.ServiceImagePullPolicy,
			imagePullSecrets:            registryImages.pullSecrets,
			runPodConfig:                &r.RunPodConfig,
			imageInventory:              registryImages.inventoryJSON,
		},
	)
	if cancelled, err := r.isAIRunCancellationRequested(ctx, aiRun); err != nil {
		_ = r.Delete(ctx, kubeconfigMap)
		return err
	} else if cancelled {
		_ = r.Delete(ctx, kubeconfigMap)
		return errAIRunCancellationRequested
	}
	if err := r.setOwnerAndCreate(ctx, aiRun, pod); err != nil {
		return fmt.Errorf("failed to create orchestrator Pod: %w", err)
	}

	if aiRun.Status.Phase != aiRunPhaseProvisioning {
		aiRun.Status.Phase = aiRunPhaseProvisioning
		return r.Status().Update(ctx, aiRun)
	}
	return nil
}

func (r *KrknAIRunReconciler) verifiedImages(ctx context.Context, orchestrator, uploader string, privateRegistry *krknctlmodels.RegistryV2) (string, string, error) {
	enabled, err := imageSignatureVerificationEnabled(ctx, r.Client, r.Namespace)
	if err != nil {
		return "", "", err
	}
	verifyImage := r.imageVerifier
	if verifyImage == nil {
		verifier, err := verify.New(verify.OptionsForRegistry(privateRegistry))
		if err != nil {
			if !enabled {
				log.FromContext(ctx).Info("WARNING: image signature verification skipped; executing unverified Krkn-AI images",
					"verificationError", err)
				return orchestrator, uploader, nil
			}
			return "", "", &configurationError{err: err}
		}
		verifyImage = verifier.VerifyImage
	}
	images := [2]string{orchestrator, uploader}
	for i, image := range images {
		verified, err := verifyImage(ctx, image)
		if err == nil {
			images[i] = verified.Digest
			continue
		}
		if !enabled {
			log.FromContext(ctx).Info("WARNING: image signature verification result skipped; executing unverified Krkn-AI image",
				"image", image, "verificationError", err)
			continue
		}
		err = fmt.Errorf("verify Krkn-AI image %q: %w", image, err)
		if errors.Is(err, verify.ErrUnsigned) || errors.Is(err, verify.ErrInvalidSignature) ||
			errors.Is(err, verify.ErrInvalidReference) {
			return "", "", &configurationError{err: err}
		}
		return "", "", err
	}
	return images[0], images[1], nil
}

// requiredResourceError classifies missing dependencies after an uncached read.
func requiredResourceError(err error) error {
	if apierrors.IsNotFound(err) {
		return &configurationError{err: err}
	}
	return err
}

func (r *KrknAIRunReconciler) observeRun(ctx context.Context, aiRun *krknv1alpha1.KrknAIRun) (ctrl.Result, error) {
	oldStatus := aiRun.Status.DeepCopy()
	cancelling := aiRun.Spec.CancelRequested ||
		aiRun.Status.Phase == aiRunPhaseCancelling || aiRun.Status.Phase == aiRunPhaseCancelled
	if cancelling {
		if aiRun.Status.CancellationRequestedAt == nil {
			now := metav1.Now()
			aiRun.Status.CancellationRequestedAt = &now
		}
		if aiRun.Status.Phase != aiRunPhaseCancelled {
			aiRun.Status.Phase = aiRunPhaseCancelling
		}
	}

	var children krknv1alpha1.KrknScenarioRunList
	if err := r.List(ctx, &children, client.InNamespace(aiRun.Namespace)); err != nil {
		return ctrl.Result{}, fmt.Errorf("list KrknScenarioRuns while observing run: %w", err)
	}
	refs := make([]string, 0, len(children.Items))
	ownedChildren := make([]*krknv1alpha1.KrknScenarioRun, 0, len(children.Items))
	for i := range children.Items {
		child := &children.Items[i]
		if hasAIRunOwner(child.OwnerReferences, aiRun.UID) {
			refs = append(refs, child.Name)
			ownedChildren = append(ownedChildren, child)
		}
	}
	sort.Strings(refs)
	aiRun.Status.ScenarioRunRefs = refs
	childrenStopped, cleanupSettled, cleanupSucceeded := true, true, true
	for _, child := range ownedChildren {
		switch child.Status.Phase {
		case "Succeeded", "Failed", "PartiallyFailed", "Cancelled":
		default:
			childrenStopped = false
			cleanupSettled, cleanupSucceeded = false, false
		}
		if child.Status.Phase == "Cancelled" {
			condition := meta.FindStatusCondition(child.Status.Conditions, "CleanupComplete")
			if condition == nil || condition.Status == metav1.ConditionUnknown {
				cleanupSettled, cleanupSucceeded = false, false
			} else if condition.Status != metav1.ConditionTrue {
				cleanupSucceeded = false
			}
		}
	}

	if aiRun.Status.OrchestratorPodName == "" {
		if !cancelling {
			return ctrl.Result{}, r.fail(ctx, aiRun, "orchestrator Pod name was not recorded")
		}
		meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
			Type: "ArtifactsFinalized", Status: metav1.ConditionTrue, Reason: "NotRequired",
			Message: "Cancellation occurred before artifact-producing work started.", ObservedGeneration: aiRun.Generation,
		})
		meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
			Type: "ArtifactsCommitted", Status: metav1.ConditionUnknown, Reason: "NotStarted",
			Message: "No run artifacts were produced before cancellation.", ObservedGeneration: aiRun.Generation,
		})
		setAIRunCancellationConditions(aiRun, childrenStopped, cleanupSettled, cleanupSucceeded)
		if childrenStopped {
			meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
				Type: "ExecutionStopped", Status: metav1.ConditionTrue, Reason: "NoOrchestratorPodCreated",
				Message:            "No orchestrator Pod was created, and owned scenario runs have stopped.",
				ObservedGeneration: aiRun.Generation,
			})
		}
		if childrenStopped {
			aiRun.Status.Phase = aiRunPhaseCancelled
			aiRun.Status.CompletionTime = completionTime(aiRun.Status.CompletionTime)
		} else {
			aiRun.Status.Phase = aiRunPhaseCancelling
			aiRun.Status.CompletionTime = nil
		}
		if !reflect.DeepEqual(oldStatus, &aiRun.Status) {
			if err := r.Status().Update(ctx, aiRun); err != nil {
				return ctrl.Result{}, err
			}
		}
		if !childrenStopped || !cleanupSettled {
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		return ctrl.Result{}, nil
	}

	var pod corev1.Pod
	err := r.Get(ctx, types.NamespacedName{Name: aiRun.Status.OrchestratorPodName, Namespace: aiRun.Namespace}, &pod)
	if err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}
	podOwned := err == nil && hasAIRunOwner(pod.OwnerReferences, aiRun.UID)
	if err != nil || !podOwned {
		if !cancelling {
			reason := "orchestrator Pod disappeared before the run completed"
			if err == nil {
				reason = "orchestrator Pod name is owned by a different KrknAIRun UID"
			}
			if statusErr := r.fail(ctx, aiRun, reason); statusErr != nil {
				return ctrl.Result{}, statusErr
			}
			return ctrl.Result{}, nil
		}
		orchestratorStopped := orchestratorStopWasProven(aiRun)
		missingReason := "OrchestratorPodMissing"
		missingMessage := "Cannot determine whether the orchestrator stopped because its Pod is missing."
		if err == nil {
			missingReason = "OrchestratorPodNotOwned"
			missingMessage = "Cannot trust termination status from a same-name Pod owned by a different KrknAIRun UID."
		}
		setAIRunCancellationConditions(aiRun, childrenStopped, cleanupSettled, cleanupSucceeded)
		if orchestratorStopped {
			meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
				Type: "ExecutionStopped", Status: metav1.ConditionTrue, Reason: "PreviouslyObservedStopped",
				Message:            "The orchestrator was previously observed to have stopped.",
				ObservedGeneration: aiRun.Generation,
			})
		} else {
			meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
				Type: "ExecutionStopped", Status: metav1.ConditionUnknown, Reason: missingReason,
				Message: missingMessage, ObservedGeneration: aiRun.Generation,
			})
		}
		markAIRunArtifactFinalizationUnavailable(aiRun, missingReason, missingMessage)
		allStopped := orchestratorStopped && childrenStopped
		if allStopped {
			aiRun.Status.Phase = aiRunPhaseCancelled
			aiRun.Status.CompletionTime = completionTime(aiRun.Status.CompletionTime)
		} else {
			aiRun.Status.Phase = aiRunPhaseCancelling
			aiRun.Status.CompletionTime = nil
		}
		if !reflect.DeepEqual(oldStatus, &aiRun.Status) {
			if err := r.Status().Update(ctx, aiRun); err != nil {
				return ctrl.Result{}, err
			}
		}
		if !allStopped || !cleanupSettled {
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		return ctrl.Result{}, nil
	}

	if cancelling && pod.Status.Phase == corev1.PodPending && !podHasStartedContainer(&pod) {
		if pod.UID == "" {
			return ctrl.Result{}, fmt.Errorf("cannot safely delete pending orchestrator Pod %q without its UID", pod.Name)
		}
		uid := pod.UID
		if err := r.Delete(ctx, &pod, client.Preconditions{UID: &uid}); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("delete never-started orchestrator Pod %q: %w", pod.Name, err)
		}
		meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
			Type: "ArtifactsFinalized", Status: metav1.ConditionTrue, Reason: "NotRequired",
			Message:            "The orchestrator container never started; artifact upload was not required.",
			ObservedGeneration: aiRun.Generation,
		})
		meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
			Type: "ArtifactsCommitted", Status: metav1.ConditionUnknown, Reason: "NotStarted",
			Message:            "No run artifacts were produced before the orchestrator Pod was deleted.",
			ObservedGeneration: aiRun.Generation,
		})
		setAIRunCancellationConditions(aiRun, childrenStopped, cleanupSettled, cleanupSucceeded)
		executionStatus, executionReason, executionMessage := metav1.ConditionFalse, "CancellationPending", "Waiting for owned scenario runs to stop; the orchestrator never started."
		if childrenStopped {
			executionStatus, executionReason, executionMessage = metav1.ConditionTrue, "NoContainerStarted", "The orchestrator never started and all owned scenario runs have stopped."
		}
		meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
			Type: "ExecutionStopped", Status: executionStatus, Reason: executionReason,
			Message: executionMessage, ObservedGeneration: aiRun.Generation,
		})
		if childrenStopped {
			aiRun.Status.Phase = aiRunPhaseCancelled
			aiRun.Status.CompletionTime = completionTime(aiRun.Status.CompletionTime)
		} else {
			aiRun.Status.Phase = aiRunPhaseCancelling
			aiRun.Status.CompletionTime = nil
		}
		if !reflect.DeepEqual(oldStatus, &aiRun.Status) {
			if err := r.Status().Update(ctx, aiRun); err != nil {
				return ctrl.Result{}, err
			}
		}
		if !childrenStopped || !cleanupSettled {
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		return ctrl.Result{}, nil
	}
	orchestrator := containerTermination(&pod, "orchestrator")
	uploader := containerTermination(&pod, "result-uploader")
	podTerminated := pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed
	if cancelling {
		executionStopped := orchestrator != nil || podTerminated
		if !executionStopped {
			cleanupSettled, cleanupSucceeded = false, false
		}
		allStopped := executionStopped && childrenStopped
		executionStatus, executionReason, executionMessage := metav1.ConditionFalse, "CancellationPending", "Waiting for the orchestrator and owned scenario runs to stop."
		if allStopped {
			executionReason = "PodTerminal"
			if orchestrator != nil {
				executionReason = "OrchestratorTerminated"
			}
			executionStatus, executionMessage = metav1.ConditionTrue, "The orchestrator and owned scenario runs have stopped."
		}
		meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
			Type: "ExecutionStopped", Status: executionStatus, Reason: executionReason,
			Message: executionMessage, ObservedGeneration: aiRun.Generation,
		})
		cleanupStatus, cleanupReason, cleanupMessage := metav1.ConditionTrue, "CleanupComplete", "Owned scenario cancellation cleanup is complete."
		if !cleanupSettled {
			cleanupStatus, cleanupReason, cleanupMessage = metav1.ConditionUnknown, "CleanupPending", "Waiting for child cancellation cleanup receipts."
		} else if !cleanupSucceeded {
			cleanupStatus, cleanupReason, cleanupMessage = metav1.ConditionFalse, "CleanupFailed", "At least one child reported unsuccessful cancellation cleanup."
		}
		meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
			Type: "CleanupComplete", Status: cleanupStatus, Reason: cleanupReason,
			Message: cleanupMessage, ObservedGeneration: aiRun.Generation,
		})

		if uploader != nil {
			meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
				Type: "ArtifactsFinalized", Status: metav1.ConditionTrue, Reason: "UploaderExited",
				Message: "The result uploader finished its finalization attempt.", ObservedGeneration: aiRun.Generation,
			})
			commitStatus, commitReason, commitMessage := metav1.ConditionFalse, "ArtifactUploadFailed", terminationReason(uploader)
			if uploader.ExitCode == 0 {
				commitStatus, commitReason, commitMessage = metav1.ConditionTrue, "ArtifactsCommitted", "Run artifacts were committed to the artifact service."
			}
			meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
				Type: "ArtifactsCommitted", Status: commitStatus, Reason: commitReason,
				Message: commitMessage, ObservedGeneration: aiRun.Generation,
			})
		} else if podTerminated {
			markAIRunArtifactFinalizationUnavailable(
				aiRun,
				"UploaderStatusUnavailable",
				"The Pod terminated without a recorded uploader result; no upload attempt can continue.",
			)
		} else {
			meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
				Type: "ArtifactsFinalized", Status: metav1.ConditionFalse, Reason: "UploadPending",
				Message: "Waiting for the result uploader to finish.", ObservedGeneration: aiRun.Generation,
			})
			meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
				Type: "ArtifactsCommitted", Status: metav1.ConditionUnknown, Reason: "UploadPending",
				Message: "Artifact commitment is pending uploader finalization.", ObservedGeneration: aiRun.Generation,
			})
		}
		if allStopped {
			aiRun.Status.Phase = aiRunPhaseCancelled
			aiRun.Status.CompletionTime = completionTime(aiRun.Status.CompletionTime)
		} else {
			aiRun.Status.Phase = aiRunPhaseCancelling
			aiRun.Status.CompletionTime = nil
		}
		if !reflect.DeepEqual(oldStatus, &aiRun.Status) {
			if err := r.Status().Update(ctx, aiRun); err != nil {
				return ctrl.Result{}, err
			}
		}
		if !allStopped || (uploader == nil && !podTerminated) || !cleanupSettled || artifactsFinalizationPending(aiRun) {
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		return ctrl.Result{}, nil
	}

	if pod.Status.Phase == corev1.PodRunning {
		aiRun.Status.Phase = aiRunPhaseRunning
	}
	if orchestrator != nil && uploader == nil && !podTerminated {
		meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
			Type: "ExecutionStopped", Status: metav1.ConditionTrue, Reason: "OrchestratorExited",
			Message: "The orchestrator container has exited.", ObservedGeneration: aiRun.Generation,
		})
		meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
			Type: "ArtifactsFinalized", Status: metav1.ConditionFalse, Reason: "UploadPending",
			Message: "Waiting for the result uploader to finish.", ObservedGeneration: aiRun.Generation,
		})
		if !reflect.DeepEqual(oldStatus, &aiRun.Status) {
			if err := r.Status().Update(ctx, aiRun); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	if podTerminated && (orchestrator == nil || uploader == nil) {
		reason := pod.Status.Reason
		if reason == "" {
			reason = string(pod.Status.Phase)
		}
		if pod.Status.Message != "" {
			reason += ": " + pod.Status.Message
		}
		markAIRunArtifactFinalizationUnavailable(
			aiRun,
			"ContainerStatusUnavailable",
			"The Pod terminated without complete container status; no upload attempt can continue: "+reason,
		)
		meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
			Type: "ExecutionStopped", Status: metav1.ConditionTrue, Reason: "PodTerminal",
			Message: "The orchestrator Pod has terminated.", ObservedGeneration: aiRun.Generation,
		})
		return ctrl.Result{}, r.fail(ctx, aiRun, "orchestrator or uploader container status is unavailable: "+reason)
	}
	if orchestrator != nil && uploader != nil {
		aiRun.Status.CompletionTime = completionTime(aiRun.Status.CompletionTime)
		meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
			Type: "ExecutionStopped", Status: metav1.ConditionTrue, Reason: "OrchestratorExited",
			Message: "The orchestrator container has exited.", ObservedGeneration: aiRun.Generation,
		})
		meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
			Type: "ArtifactsFinalized", Status: metav1.ConditionTrue, Reason: "UploaderExited",
			Message: "The result uploader finished its finalization attempt.", ObservedGeneration: aiRun.Generation,
		})
		artifactsCommitted := uploader.ExitCode == 0
		conditionStatus, conditionReason, conditionMessage := metav1.ConditionFalse, "ArtifactUploadFailed", terminationReason(uploader)
		if artifactsCommitted {
			conditionStatus, conditionReason, conditionMessage = metav1.ConditionTrue, "ArtifactsCommitted", "Run artifacts were committed to the artifact service."
		}
		meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
			Type: "ArtifactsCommitted", Status: conditionStatus, Reason: conditionReason,
			Message: conditionMessage, ObservedGeneration: aiRun.Generation,
		})
		if orchestrator.ExitCode == 0 && artifactsCommitted {
			aiRun.Status.Phase = aiRunPhaseSucceeded
			aiRun.Status.FailureReason = ""
		} else {
			aiRun.Status.Phase = aiRunPhaseFailed
			if orchestrator.ExitCode != 0 {
				aiRun.Status.FailureReason = terminationReason(orchestrator)
			} else {
				aiRun.Status.FailureReason = terminationReason(uploader)
			}
		}
	}
	if !reflect.DeepEqual(oldStatus, &aiRun.Status) {
		if err := r.Status().Update(ctx, aiRun); err != nil {
			return ctrl.Result{}, err
		}
	}
	if aiRun.Status.Phase == aiRunPhaseSucceeded || aiRun.Status.Phase == aiRunPhaseFailed {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func setAIRunCancellationConditions(
	aiRun *krknv1alpha1.KrknAIRun,
	childrenStopped bool,
	cleanupSettled bool,
	cleanupSucceeded bool,
) {
	executionStatus, executionReason, executionMessage := metav1.ConditionFalse, "CancellationPending", "Waiting for owned scenario runs to stop."
	if childrenStopped {
		executionStatus, executionReason, executionMessage = metav1.ConditionTrue, "ExecutionStopped", "Owned scenario runs have stopped."
	}
	meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
		Type: "ExecutionStopped", Status: executionStatus, Reason: executionReason,
		Message: executionMessage, ObservedGeneration: aiRun.Generation,
	})
	cleanupStatus, cleanupReason, cleanupMessage := metav1.ConditionTrue, "CleanupComplete", "Owned scenario cancellation cleanup is complete."
	if !cleanupSettled {
		cleanupStatus, cleanupReason, cleanupMessage = metav1.ConditionUnknown, "CleanupPending", "Waiting for child cancellation cleanup receipts."
	} else if !cleanupSucceeded {
		cleanupStatus, cleanupReason, cleanupMessage = metav1.ConditionFalse, "CleanupFailed", "At least one child reported unsuccessful cancellation cleanup."
	}
	meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
		Type: "CleanupComplete", Status: cleanupStatus, Reason: cleanupReason,
		Message: cleanupMessage, ObservedGeneration: aiRun.Generation,
	})
}

func markAIRunArtifactFinalizationUnavailable(aiRun *krknv1alpha1.KrknAIRun, reason, message string) {
	finalized := meta.FindStatusCondition(aiRun.Status.Conditions, "ArtifactsFinalized")
	if finalized == nil || finalized.Status != metav1.ConditionTrue {
		meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
			Type: "ArtifactsFinalized", Status: metav1.ConditionTrue, Reason: reason,
			Message: message, ObservedGeneration: aiRun.Generation,
		})
	}
	committed := meta.FindStatusCondition(aiRun.Status.Conditions, "ArtifactsCommitted")
	if committed == nil || committed.Status == metav1.ConditionUnknown || committed.Reason == "UploadPending" {
		meta.SetStatusCondition(&aiRun.Status.Conditions, metav1.Condition{
			Type: "ArtifactsCommitted", Status: metav1.ConditionUnknown, Reason: reason,
			Message: message, ObservedGeneration: aiRun.Generation,
		})
	}
}
func artifactsFinalizationPending(aiRun *krknv1alpha1.KrknAIRun) bool {
	condition := meta.FindStatusCondition(aiRun.Status.Conditions, "ArtifactsFinalized")
	return condition == nil || condition.Status != metav1.ConditionTrue
}

func (r *KrknAIRunReconciler) fail(ctx context.Context, aiRun *krknv1alpha1.KrknAIRun, reason string) error {
	aiRun.Status.Phase = aiRunPhaseFailed
	aiRun.Status.FailureReason = reason
	aiRun.Status.CompletionTime = completionTime(aiRun.Status.CompletionTime)
	return r.Status().Update(ctx, aiRun)
}

func (r *KrknAIRunReconciler) setOwnerAndCreate(ctx context.Context, owner client.Object, object client.Object) error {
	if err := controllerutil.SetControllerReference(owner, object, r.Scheme); err != nil {
		return &configurationError{err: err}
	}
	if err := r.Create(ctx, object); err != nil && !apierrors.IsAlreadyExists(err) {
		if apierrors.IsInvalid(err) || apierrors.IsBadRequest(err) {
			return &configurationError{err: err}
		}
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

func (r *KrknAIRunReconciler) retainTargetIdentity(
	ctx context.Context,
	aiRun *krknv1alpha1.KrknAIRun,
) error {
	if _, exists := aiRun.Annotations[krknv1alpha1.KrknAIRunTargetAPIURLAnnotation]; exists {
		return nil
	}
	if _, exists := aiRun.Annotations[krknv1alpha1.KrknAIRunTargetBindingAnnotation]; exists {
		return nil
	}
	requestID := aiRun.Spec.TargetRequestID
	provider, cluster, err := oneTarget(aiRun.Spec.TargetClusters)
	if requestID == "" || err != nil || provider == "" {
		return nil
	}

	var targetRequest krknv1alpha1.KrknTargetRequest
	if err := r.Get(ctx, types.NamespacedName{Name: requestID, Namespace: r.Namespace}, &targetRequest); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("read target request for KrknAIRun identity: %w", err)
	}
	if !strings.EqualFold(targetRequest.Status.Status, "Completed") {
		return nil
	}
	targets := targetRequest.Status.TargetData[provider]
	var apiURL string
	for _, target := range targets {
		if target.ClusterName == cluster {
			apiURL = target.ClusterAPIURL
			break
		}
	}
	if apiURL == "" {
		return nil
	}

	if aiRun.Annotations == nil {
		aiRun.Annotations = make(map[string]string)
	}
	aiRun.Annotations[krknv1alpha1.KrknAIRunTargetAPIURLAnnotation] = apiURL
	aiRun.Annotations[krknv1alpha1.KrknAIRunTargetBindingAnnotation] = files.HashLogicalName(
		requestID + "\x00" + provider + "\x00" + cluster,
	)
	if err := r.Update(ctx, aiRun); err != nil {
		return fmt.Errorf("persist KrknAIRun target identity: %w", err)
	}
	return nil
}

func aiResourceName(prefix, runName, suffix string) string {
	name := prefix + "-" + runName
	if suffix != "" {
		name += "-" + suffix
	}
	if len(name) <= 63 {
		return name
	}
	hash := files.HashLogicalName(name)[:8]
	prefixLength := 63 - len(hash) - 1
	truncated := strings.TrimRight(name[:prefixLength], "-.")
	return truncated + "-" + hash
}

func aiOrchestratorPodName(runName string) string {
	suffix := "-" + uuid.NewString()[:8]
	maxRunNameLength := 63 - len(suffix)
	if len(runName) > maxRunNameLength {
		runName = strings.TrimSuffix(runName[:maxRunNameLength], "-")
	}
	return runName + suffix
}

func aiRunLabelValue(runName string) string {
	if len(runName) <= 63 {
		return runName
	}
	return files.HashLogicalName(runName)
}

func completionTime(existing *metav1.Time) *metav1.Time {
	if existing != nil {
		return existing
	}
	now := metav1.Now()
	return &now
}

func containerTermination(pod *corev1.Pod, name string) *corev1.ContainerStateTerminated {
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == name && status.State.Terminated != nil {
			return status.State.Terminated
		}
	}
	return nil
}

func podHasStartedContainer(pod *corev1.Pod) bool {
	for _, status := range pod.Status.InitContainerStatuses {
		if containerStatusHasStarted(status) {
			return true
		}
	}
	for _, status := range pod.Status.ContainerStatuses {
		if containerStatusHasStarted(status) {
			return true
		}
	}
	for _, status := range pod.Status.EphemeralContainerStatuses {
		if containerStatusHasStarted(status) {
			return true
		}
	}
	return false
}

func containerStatusHasStarted(status corev1.ContainerStatus) bool {
	return status.State.Running != nil ||
		status.State.Terminated != nil ||
		status.LastTerminationState.Running != nil ||
		status.LastTerminationState.Terminated != nil ||
		status.ContainerID != "" ||
		status.RestartCount > 0 ||
		(status.Started != nil && *status.Started)
}
func orchestratorStopWasProven(aiRun *krknv1alpha1.KrknAIRun) bool {
	execution := meta.FindStatusCondition(aiRun.Status.Conditions, "ExecutionStopped")
	if execution != nil && execution.Status == metav1.ConditionTrue {
		switch execution.Reason {
		case "OrchestratorExited", "OrchestratorTerminated", "PodTerminal", "NoContainerStarted", "NoOrchestratorPodCreated", "PreviouslyObservedStopped":
			return true
		}
	}
	finalized := meta.FindStatusCondition(aiRun.Status.Conditions, "ArtifactsFinalized")
	return finalized != nil && finalized.Status == metav1.ConditionTrue && finalized.Reason == "NotRequired"
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
	return fmt.Sprintf("container exited with code %d", terminated.ExitCode)
}

type krknAIPodOptions struct {
	namespace                   string
	serviceImage                string
	serviceURL                  string
	serviceTokenSecretName      string
	serviceAccountName          string
	orchestratorImagePullPolicy corev1.PullPolicy
	serviceImagePullPolicy      corev1.PullPolicy
	imagePullSecrets            []corev1.LocalObjectReference
	runPodConfig                *KrknAIRunPodConfig
	imageInventory              string
}

func buildOrchestratorPod(
	aiRun *krknv1alpha1.KrknAIRun,
	podName, image, provider, cluster, configName, configKey, kubeconfigName string,
	options krknAIPodOptions,
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
		{Name: "MPLCONFIGDIR", Value: "/tmp/matplotlib-orchestrator"},
		{Name: "FORMAT", Value: "json"},
		{Name: "RUNNER_TYPE", Value: "operator"},
		{Name: "VERBOSE", Value: "2"},
		{Name: "KRKNAI_NAMESPACE", Value: options.namespace},
		{Name: "KRKNAI_RUN_NAME", Value: aiRun.Name},
		{Name: "KRKNAI_RUN_UID", Value: string(aiRun.UID)},
		{Name: "KRKNAI_ORCHESTRATOR_POD_NAME", Value: podName},
		{Name: "KRKNAI_TARGET_REQUEST_ID", Value: aiRun.Spec.TargetRequestID},
		{Name: "KRKNAI_PROVIDER", Value: provider},
		{Name: "KRKNAI_CLUSTER", Value: cluster},
	}
	if aiRun.Spec.RegistryName != "" {
		envs = append(envs,
			corev1.EnvVar{Name: "KRKNAI_OPERATOR_REGISTRY_NAME", Value: aiRun.Spec.RegistryName},
			corev1.EnvVar{Name: "KRKNAI_IMAGE_INVENTORY", Value: options.imageInventory},
		)
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
		ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: aiRun.Namespace, Labels: map[string]string{
			"krkn.dev/ai-run":    aiRunLabelValue(aiRun.Name),
			"krkn.dev/component": "orchestrator",
		}},
		Spec: corev1.PodSpec{
			ServiceAccountName:    options.serviceAccountName,
			RestartPolicy:         corev1.RestartPolicyNever,
			ActiveDeadlineSeconds: &activeDeadline,
			ImagePullSecrets:      options.imagePullSecrets,
			SecurityContext:       &corev1.PodSecurityContext{RunAsUser: &runAs, RunAsGroup: &runAs, FSGroup: &runAs},
			Containers: []corev1.Container{
				{
					Name:            "orchestrator",
					Image:           image,
					ImagePullPolicy: defaultPullPolicy(options.orchestratorImagePullPolicy),
					Env:             envs,
					Resources:       options.runPodConfig.OrchestratorResources,
					VolumeMounts: []corev1.VolumeMount{
						{Name: "config", MountPath: "/input/krkn-ai.yaml", SubPath: configKey},
						{Name: "kubeconfig", MountPath: "/input/kubeconfig", SubPath: "config"},
						{Name: "output", MountPath: "/output"},
						{Name: "tmp", MountPath: "/tmp"},
					},
				},
				{
					Name:            "result-uploader",
					Image:           options.serviceImage,
					ImagePullPolicy: defaultPullPolicy(options.serviceImagePullPolicy),
					Command:         []string{"python", "-m", "krkn_ai.server", "uploader"},
					Resources:       options.runPodConfig.ResultUploaderResources,
					Env: []corev1.EnvVar{
						{Name: "KRKNAI_OUTPUT_DIR", Value: "/output"},
						{Name: "KRKNAI_UPLOAD_STATE_DIR", Value: "/upload-state"},
						{Name: "MPLCONFIGDIR", Value: "/tmp/matplotlib-uploader"},
						{Name: "KRKNAI_UPLOAD_INTERVAL_SECONDS", Value: "10"},
						{Name: "KRKNAI_RUN_UID", Value: string(aiRun.UID)},
						{Name: "KRKNAI_SERVICE_URL", Value: options.serviceURL},
						{Name: "KRKNAI_SERVICE_TOKEN", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: options.serviceTokenSecretName},
							Key:                  "token",
						}}},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "output", MountPath: "/output"},
						{Name: "upload-state", MountPath: "/upload-state"},
						{Name: "tmp", MountPath: "/tmp"},
					},
				},
			},
			Volumes: []corev1.Volume{
				{Name: "config", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: configName}}}},
				{Name: "kubeconfig", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: kubeconfigName}}}},
				{Name: "output", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{SizeLimit: &options.runPodConfig.OutputSizeLimit}}},
				{Name: "upload-state", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{SizeLimit: &options.runPodConfig.UploadStateSizeLimit}}},
				{Name: "tmp", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{SizeLimit: &options.runPodConfig.TmpSizeLimit}}},
			},
		},
	}
}

func defaultPullPolicy(policy corev1.PullPolicy) corev1.PullPolicy {
	if policy == "" {
		return corev1.PullAlways
	}
	return policy
}

// SetupWithManager registers KrknAIRun reconciliation and watches its owned
// orchestrator Pods and scenario runs.
func (r *KrknAIRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&krknv1alpha1.KrknAIRun{}).
		Owns(&corev1.Pod{}).
		Owns(&krknv1alpha1.KrknScenarioRun{}).
		Complete(r)
}
