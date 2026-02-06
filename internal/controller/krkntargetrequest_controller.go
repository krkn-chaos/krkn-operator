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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/kubeconfig"
)

// KrknTargetRequestReconciler reconciles a KrknTargetRequest object
type KrknTargetRequestReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	OperatorName      string
	OperatorNamespace string
}

// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krkntargetrequests,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krkntargetrequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknoperatortargets,verbs=get;list;watch
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknoperatortargetproviders,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknoperatortargetproviders/status,verbs=get;update;patch

// Reconcile processes KrknTargetRequest resources to populate target data from KrknOperatorTarget CRs
func (r *KrknTargetRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("🔄 Reconciling KrknTargetRequest",
		"name", req.Name,
		"namespace", req.Namespace,
		"operatorName", r.OperatorName,
		"operatorNamespace", r.OperatorNamespace)

	// 1. Fetch KrknTargetRequest
	var krknRequest krknv1alpha1.KrknTargetRequest
	if err := r.Get(ctx, req.NamespacedName, &krknRequest); err != nil {
		if client.IgnoreNotFound(err) == nil {
			logger.Info("KrknTargetRequest not found, probably deleted", "name", req.Name)
		} else {
			logger.Error(err, "Failed to get KrknTargetRequest")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("✅ Found KrknTargetRequest",
		"uuid", krknRequest.Spec.UUID,
		"status", krknRequest.Status.Status,
		"targetDataKeys", len(krknRequest.Status.TargetData))

	// 2. Skip if already completed
	if krknRequest.Status.Status == "Completed" {
		logger.Info("Request already completed, skipping", "uuid", krknRequest.Spec.UUID)
		return ctrl.Result{}, nil
	}

	// 3. Ensure UUID label is set
	if err := r.ensureUUIDLabel(ctx, &krknRequest); err != nil {
		if isConflictError(err) {
			logger.Info("Conflict during UUID label update, requeuing", "error", err.Error())
			return ctrl.Result{Requeue: true}, nil
		}
		logger.Error(err, "Failed to set UUID label")
		return ctrl.Result{}, err
	}

	// Refetch after label update to avoid conflicts
	if err := r.Get(ctx, req.NamespacedName, &krknRequest); err != nil {
		logger.Error(err, "Failed to refetch KrknTargetRequest after label update")
		return ctrl.Result{}, err
	}

	// 4. Initialize status if pending
	if err := r.initializeStatus(ctx, &krknRequest); err != nil {
		if isConflictError(err) {
			logger.Info("Conflict during status init, requeuing", "error", err.Error())
			return ctrl.Result{Requeue: true}, nil
		}
		logger.Error(err, "Failed to initialize status")
		return ctrl.Result{}, err
	}

	// Refetch after status init to avoid conflicts
	if err := r.Get(ctx, req.NamespacedName, &krknRequest); err != nil {
		logger.Error(err, "Failed to refetch KrknTargetRequest after status init")
		return ctrl.Result{}, err
	}

	// 5. Query all KrknOperatorTarget CRs in operator namespace
	var targets krknv1alpha1.KrknOperatorTargetList
	if err := r.List(ctx, &targets, client.InNamespace(r.OperatorNamespace)); err != nil {
		logger.Error(err, "Failed to list KrknOperatorTarget CRs")
		return ctrl.Result{}, err
	}

	// 6. Build ClusterTarget list from ready targets
	clusterTargets := r.buildClusterTargets(targets.Items)
	logger.Info("Built cluster targets", "count", len(clusterTargets), "operator", r.OperatorName)

	// 7. Update Status.TargetData[operatorName]
	if err := r.updateTargetData(ctx, &krknRequest, clusterTargets); err != nil {
		if isConflictError(err) {
			logger.Info("Conflict during target data update, requeuing", "error", err.Error())
			return ctrl.Result{Requeue: true}, nil
		}
		logger.Error(err, "Failed to update target data")
		return ctrl.Result{}, err
	}

	// Refetch after target data update to avoid conflicts
	if err := r.Get(ctx, req.NamespacedName, &krknRequest); err != nil {
		logger.Error(err, "Failed to refetch KrknTargetRequest after target data update")
		return ctrl.Result{}, err
	}

	// 8. Write kubeconfigs to Secret (managed-clusters format)
	if err := r.writeManagedClustersSecret(ctx, &krknRequest, targets.Items); err != nil {
		logger.Error(err, "Failed to write managed-clusters Secret")
		return ctrl.Result{}, err
	}

	// Refetch before completion check to avoid conflicts (another provider might have updated)
	if err := r.Get(ctx, req.NamespacedName, &krknRequest); err != nil {
		logger.Error(err, "Failed to refetch KrknTargetRequest before completion check")
		return ctrl.Result{}, err
	}

	// 9. Check if all active providers have contributed
	if err := r.checkCompletion(ctx, &krknRequest); err != nil {
		// If conflict error, requeue instead of failing
		if isConflictError(err) {
			logger.Info("Conflict detected during completion check, requeuing", "error", err.Error())
			return ctrl.Result{Requeue: true}, nil
		}
		logger.Error(err, "Failed to check completion")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// ensureUUIDLabel ensures the UUID label is set on the KrknTargetRequest
func (r *KrknTargetRequestReconciler) ensureUUIDLabel(ctx context.Context, krknRequest *krknv1alpha1.KrknTargetRequest) error {
	logger := log.FromContext(ctx)
	if _, exists := krknRequest.Labels["krkn.krkn-chaos.dev/uuid"]; !exists {
		logger.Info("Setting UUID label", "uuid", krknRequest.Spec.UUID)
		if krknRequest.Labels == nil {
			krknRequest.Labels = make(map[string]string)
		}
		krknRequest.Labels["krkn.krkn-chaos.dev/uuid"] = krknRequest.Spec.UUID
		if err := r.Update(ctx, krknRequest); err != nil {
			return err
		}
		logger.Info("✅ UUID label set successfully")
	}
	return nil
}

// initializeStatus sets the status to pending and sets Created timestamp if not already set
func (r *KrknTargetRequestReconciler) initializeStatus(ctx context.Context, krknRequest *krknv1alpha1.KrknTargetRequest) error {
	logger := log.FromContext(ctx)
	if krknRequest.Status.Status == "" {
		logger.Info("Initializing status to pending")
		krknRequest.Status.Status = "pending"
		now := metav1.NewTime(time.Now())
		krknRequest.Status.Created = &now
		if err := r.Status().Update(ctx, krknRequest); err != nil {
			return err
		}
		logger.Info("✅ Status initialized to pending")
	}
	return nil
}

// buildClusterTargets builds a list of ClusterTarget from KrknOperatorTarget CRs
func (r *KrknTargetRequestReconciler) buildClusterTargets(targets []krknv1alpha1.KrknOperatorTarget) []krknv1alpha1.ClusterTarget {
	logger := log.Log.WithName("buildClusterTargets")
	clusterTargets := make([]krknv1alpha1.ClusterTarget, 0, len(targets))

	logger.Info("Building cluster targets", "totalTargets", len(targets))

	for _, target := range targets {
		logger.V(1).Info("Processing target",
			"name", target.Name,
			"clusterName", target.Spec.ClusterName,
			"ready", target.Status.Ready)

		// Only include ready targets
		if target.Status.Ready {
			clusterTargets = append(clusterTargets, krknv1alpha1.ClusterTarget{
				ClusterName:   target.Spec.ClusterName,
				ClusterAPIURL: target.Spec.ClusterAPIURL,
			})
			logger.Info("✅ Added ready target",
				"clusterName", target.Spec.ClusterName,
				"apiURL", target.Spec.ClusterAPIURL)
		} else {
			logger.Info("⏭️  Skipping non-ready target", "clusterName", target.Spec.ClusterName)
		}
	}

	logger.Info("Built cluster targets", "readyCount", len(clusterTargets))
	return clusterTargets
}

// updateTargetData updates the TargetData map with cluster targets for this operator
func (r *KrknTargetRequestReconciler) updateTargetData(ctx context.Context, krknRequest *krknv1alpha1.KrknTargetRequest, clusterTargets []krknv1alpha1.ClusterTarget) error {
	logger := log.FromContext(ctx)
	if krknRequest.Status.TargetData == nil {
		krknRequest.Status.TargetData = make(map[string][]krknv1alpha1.ClusterTarget)
	}

	// Update target data for this operator
	logger.Info("Updating TargetData",
		"operatorName", r.OperatorName,
		"targetsCount", len(clusterTargets))

	krknRequest.Status.TargetData[r.OperatorName] = clusterTargets

	if err := r.Status().Update(ctx, krknRequest); err != nil {
		return err
	}

	logger.Info("✅ TargetData updated successfully", "totalProviders", len(krknRequest.Status.TargetData))
	return nil
}

// checkCompletion checks if all active providers have contributed and marks the request as completed
func (r *KrknTargetRequestReconciler) checkCompletion(ctx context.Context, krknRequest *krknv1alpha1.KrknTargetRequest) error {
	logger := log.FromContext(ctx)

	// Query all KrknOperatorTargetProvider CRs
	var providers krknv1alpha1.KrknOperatorTargetProviderList
	if err := r.List(ctx, &providers); err != nil {
		logger.Error(err, "Failed to list KrknOperatorTargetProvider CRs")
		return err
	}

	logger.Info("Found providers", "totalProviders", len(providers.Items))

	// Count active providers
	activeProviders := 0
	activeProviderNames := []string{}
	for _, provider := range providers.Items {
		if provider.Spec.Active {
			activeProviders++
			activeProviderNames = append(activeProviderNames, provider.Spec.OperatorName)
			logger.Info("Active provider found",
				"name", provider.Spec.OperatorName,
				"timestamp", provider.Status.Timestamp)
		}
	}

	// Count contributors (operators that have added target data)
	contributorCount := len(krknRequest.Status.TargetData)
	contributorNames := []string{}
	for name := range krknRequest.Status.TargetData {
		contributorNames = append(contributorNames, name)
	}

	logger.Info("🔍 Checking completion",
		"activeProviders", activeProviders,
		"activeProviderNames", activeProviderNames,
		"contributors", contributorCount,
		"contributorNames", contributorNames,
		"uuid", krknRequest.Spec.UUID)

	// If all active providers have contributed, mark as completed
	if activeProviders > 0 && contributorCount >= activeProviders {
		logger.Info("✅ All active providers have contributed, marking as Completed",
			"uuid", krknRequest.Spec.UUID,
			"activeProviders", activeProviders,
			"contributors", contributorCount)
		krknRequest.Status.Status = "Completed"
		now := metav1.NewTime(time.Now())
		krknRequest.Status.Completed = &now
		if err := r.Status().Update(ctx, krknRequest); err != nil {
			return err
		}
		logger.Info("✅ Request marked as Completed successfully")
	} else {
		logger.Info("⏳ Waiting for more providers to contribute",
			"needed", activeProviders,
			"current", contributorCount)
	}

	return nil
}

// NewNamespaceFilter creates a predicate that filters events by namespace
func NewNamespaceFilter(namespace string) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		match := obj.GetNamespace() == namespace
		if !match {
			log.Log.WithName("namespace-filter").V(1).Info("Filtering out event",
				"expected", namespace,
				"actual", obj.GetNamespace(),
				"name", obj.GetName())
		}
		return match
	})
}

// writeManagedClustersSecret writes kubeconfigs to the managed-clusters Secret
func (r *KrknTargetRequestReconciler) writeManagedClustersSecret(ctx context.Context, krknRequest *krknv1alpha1.KrknTargetRequest, targets []krknv1alpha1.KrknOperatorTarget) error {
	logger := log.FromContext(ctx)

	// Fetch or create Secret
	var secret corev1.Secret
	secretName := krknRequest.Spec.UUID
	err := r.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: r.OperatorNamespace,
	}, &secret)

	secretExists := err == nil
	if err != nil && client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get Secret: %w", err)
	}

	// Decode existing managed-clusters or create new structure
	var managedClusters map[string]map[string]map[string]string
	if secretExists && len(secret.Data["managed-clusters"]) > 0 {
		if err := json.Unmarshal(secret.Data["managed-clusters"], &managedClusters); err != nil {
			logger.Error(err, "Failed to unmarshal managed-clusters, creating new structure")
			managedClusters = make(map[string]map[string]map[string]string)
		}
	} else {
		managedClusters = make(map[string]map[string]map[string]string)
	}

	// Ensure our provider section exists
	if managedClusters[r.OperatorName] == nil {
		managedClusters[r.OperatorName] = make(map[string]map[string]string)
	}

	// Add each ready target
	for _, target := range targets {
		if !target.Status.Ready {
			continue
		}

		// Fetch kubeconfig from target's Secret
		var targetSecret corev1.Secret
		if err := r.Get(ctx, types.NamespacedName{
			Name:      target.Spec.SecretUUID,
			Namespace: r.OperatorNamespace,
		}, &targetSecret); err != nil {
			logger.Error(err, "Failed to get target Secret, skipping",
				"cluster", target.Spec.ClusterName,
				"secretUUID", target.Spec.SecretUUID)
			continue
		}

		kubeconfigData, exists := targetSecret.Data["kubeconfig"]
		if !exists {
			logger.Error(fmt.Errorf("kubeconfig not found"), "Skipping target",
				"cluster", target.Spec.ClusterName)
			continue
		}

		// Unmarshal to get base64 kubeconfig
		kubeconfigBase64, err := kubeconfig.UnmarshalSecretData(kubeconfigData)
		if err != nil {
			logger.Error(err, "Failed to unmarshal kubeconfig, skipping",
				"cluster", target.Spec.ClusterName)
			continue
		}

		// Add to managed-clusters structure
		managedClusters[r.OperatorName][target.Spec.ClusterName] = map[string]string{
			"cluster-name": target.Spec.ClusterName,
			"cluster-api":  target.Spec.ClusterAPIURL,
			"kubeconfig":   kubeconfigBase64,
		}

		logger.Info("Added cluster to managed-clusters",
			"provider", r.OperatorName,
			"cluster", target.Spec.ClusterName)
	}

	// Marshal back to JSON
	managedClustersBytes, err := json.Marshal(managedClusters)
	if err != nil {
		return fmt.Errorf("failed to marshal managed-clusters: %w", err)
	}

	// Create or update Secret
	if !secretExists {
		secret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: r.OperatorNamespace,
				Labels: map[string]string{
					"krkn.krkn-chaos.dev/target-request": krknRequest.Spec.UUID,
				},
			},
			Data: map[string][]byte{
				"managed-clusters": managedClustersBytes,
			},
		}
		if err := r.Create(ctx, &secret); err != nil {
			return fmt.Errorf("failed to create Secret: %w", err)
		}
		logger.Info("✅ Created managed-clusters Secret", "secretName", secretName)
	} else {
		secret.Data["managed-clusters"] = managedClustersBytes
		if err := r.Update(ctx, &secret); err != nil {
			return fmt.Errorf("failed to update Secret: %w", err)
		}
		logger.Info("✅ Updated managed-clusters Secret", "secretName", secretName)
	}

	return nil
}

// isConflictError checks if an error is a Kubernetes conflict error (optimistic locking failure)
func isConflictError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "the object has been modified")
}

// SetupWithManager sets up the controller with the Manager
func (r *KrknTargetRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	logger := mgr.GetLogger().WithName("krkntargetrequest-setup")
	logger.Info("🚀 Setting up KrknTargetRequest controller",
		"operatorName", r.OperatorName,
		"operatorNamespace", r.OperatorNamespace)

	return ctrl.NewControllerManagedBy(mgr).
		For(&krknv1alpha1.KrknTargetRequest{}).
		Named("krkntargetrequest").
		WithEventFilter(NewNamespaceFilter(r.OperatorNamespace)).
		Complete(r)
}
