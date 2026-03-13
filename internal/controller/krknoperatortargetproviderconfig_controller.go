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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/provider"
)

// KrknOperatorTargetProviderConfigReconciler reconciles a KrknOperatorTargetProviderConfig object
type KrknOperatorTargetProviderConfigReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	OperatorName      string
	OperatorNamespace string
}

// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknoperatortargetproviderconfigs,verbs=get;list;watch;update;patch;delete
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknoperatortargetproviderconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknoperatortargetproviders,verbs=get;list;watch

// Reconcile manages the lifecycle of KrknOperatorTargetProviderConfig resources.
// It ensures UUID labels are set, initializes status, and checks for completion
// when all active providers have contributed their configuration data.
func (r *KrknOperatorTargetProviderConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("🔄 Reconciling KrknOperatorTargetProviderConfig",
		"name", req.Name,
		"namespace", req.Namespace)

	// 1. Fetch KrknOperatorTargetProviderConfig
	var config krknv1alpha1.KrknOperatorTargetProviderConfig
	if err := r.Get(ctx, req.NamespacedName, &config); err != nil {
		if client.IgnoreNotFound(err) == nil {
			logger.Info("KrknOperatorTargetProviderConfig not found, probably deleted", "name", req.Name)
		} else {
			logger.Error(err, "Failed to get KrknOperatorTargetProviderConfig")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("✅ Found KrknOperatorTargetProviderConfig",
		"uuid", config.Spec.UUID,
		"status", config.Status.Status,
		"configDataKeys", len(config.Status.ConfigData))

	// 2. Skip if already completed
	if config.Status.Status == "Completed" {
		logger.Info("Config request already completed, skipping", "uuid", config.Spec.UUID)
		return ctrl.Result{}, nil
	}

	// 3. Fetch provider list early (will be reused in completion check)
	providerList := &krknv1alpha1.KrknOperatorTargetProviderList{}
	if err := r.List(ctx, providerList); err != nil {
		logger.Error(err, "Failed to list KrknOperatorTargetProviders")
		return ctrl.Result{}, err
	}

	// 4. Ensure UUID label is set
	if err := r.ensureUUIDLabel(ctx, &config); err != nil {
		if isConflictError(err) {
			logger.Info("Conflict during UUID label update, requeuing", "error", err.Error())
			return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, nil
		}
		logger.Error(err, "Failed to set UUID label")
		return ctrl.Result{}, err
	}

	// Refetch after label update to avoid conflicts
	if err := r.Get(ctx, req.NamespacedName, &config); err != nil {
		logger.Error(err, "Failed to refetch KrknOperatorTargetProviderConfig after label update")
		return ctrl.Result{}, err
	}

	// 5. Initialize status if pending
	if err := r.initializeStatus(ctx, &config); err != nil {
		if isConflictError(err) {
			logger.Info("Conflict during status init, requeuing", "error", err.Error())
			return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, nil
		}
		logger.Error(err, "Failed to initialize status")
		return ctrl.Result{}, err
	}

	// Refetch after status initialization
	if err := r.Get(ctx, req.NamespacedName, &config); err != nil {
		logger.Error(err, "Failed to refetch after status init")
		return ctrl.Result{}, err
	}

	// 5.5. Check if this provider itself is active before contributing
	isActive, _, err := checkProviderActive(ctx, r.Client, r.OperatorName)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !isActive {
		logger.Info("krkn-operator provider is not active, skipping contribution")
		// Still continue to completion check and cleanup
		goto checkCompletion
	}

	// 5.6. Skip if already contributed
	if config.Status.ConfigData != nil {
		if _, exists := config.Status.ConfigData[r.OperatorName]; exists {
			logger.Info("Already contributed, skipping", "uuid", config.Spec.UUID)
			goto checkCompletion
		}
	}

	// 5.7. Contribute krkn-operator's configuration (empty marker for now)
	logger.Info("Contributing krkn-operator configuration", "uuid", config.Spec.UUID)

	// Direct status update with empty ProviderConfigData (no ConfigMap/schema needed)
	// We don't use provider.UpdateProviderConfig() because it requires non-empty configMapName
	if config.Status.ConfigData == nil {
		config.Status.ConfigData = make(map[string]krknv1alpha1.ProviderConfigData)
	}

	// Add empty contribution marker for krkn-operator
	config.Status.ConfigData[r.OperatorName] = krknv1alpha1.ProviderConfigData{
		ConfigMap:    "", // Empty - no config to expose yet
		Namespace:    r.OperatorNamespace,
		ConfigSchema: "", // Empty - no schema yet
	}

	if err := r.Status().Update(ctx, &config); err != nil {
		if isConflictError(err) {
			logger.Info("Conflict updating provider config, will retry")
			return ctrl.Result{Requeue: true}, nil
		}
		logger.Error(err, "Failed to update provider config")
		return ctrl.Result{}, err
	}

	logger.Info("✅ Successfully contributed krkn-operator configuration", "uuid", config.Spec.UUID)

	// Refetch after contribution to get latest version
	if err := r.Get(ctx, req.NamespacedName, &config); err != nil {
		logger.Error(err, "Failed to refetch after contribution")
		return ctrl.Result{}, err
	}

checkCompletion:
	// Refetch before completion check to ensure we have latest version
	if err := r.Get(ctx, req.NamespacedName, &config); err != nil {
		logger.Error(err, "Failed to refetch KrknOperatorTargetProviderConfig before completion check")
		return ctrl.Result{}, err
	}

	// 6. Check if all active providers have contributed (reuse providerList from step 3)
	if err := r.checkCompletion(ctx, &config, providerList); err != nil {
		if isConflictError(err) {
			logger.Info("Conflict detected during completion check, requeuing", "error", err.Error())
			return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, nil
		}
		logger.Error(err, "Failed to check completion")
		return ctrl.Result{}, err
	}

	// 7. Clean up old completed KrknOperatorTargetProviderConfig resources
	// This runs on every reconcile but is idempotent and logs only deletions/conflicts
	_, _ = provider.CleanupOldResources(
		ctx,
		r.Client,
		&krknv1alpha1.KrknOperatorTargetProviderConfigList{},
		r.OperatorNamespace,
		CleanupThresholdSeconds,
		func(obj client.Object) *metav1.Time {
			config := obj.(*krknv1alpha1.KrknOperatorTargetProviderConfig)
			// Only delete if Completed to avoid deleting pending requests
			if config.Status.Status == "Completed" {
				return config.Status.Created
			}
			return nil
		},
	)

	return ctrl.Result{}, nil
}

// ensureUUIDLabel ensures the UUID label is set on the KrknOperatorTargetProviderConfig
func (r *KrknOperatorTargetProviderConfigReconciler) ensureUUIDLabel(ctx context.Context, config *krknv1alpha1.KrknOperatorTargetProviderConfig) error {
	logger := log.FromContext(ctx)
	if _, exists := config.Labels["krkn.krkn-chaos.dev/uuid"]; !exists {
		logger.Info("Setting UUID label", "uuid", config.Spec.UUID)
		if config.Labels == nil {
			config.Labels = make(map[string]string)
		}
		config.Labels["krkn.krkn-chaos.dev/uuid"] = config.Spec.UUID
		if err := r.Update(ctx, config); err != nil {
			return err
		}
		logger.Info("✅ UUID label set successfully")
	}
	return nil
}

// initializeStatus sets the status to pending and sets Created timestamp if not already set
func (r *KrknOperatorTargetProviderConfigReconciler) initializeStatus(ctx context.Context, config *krknv1alpha1.KrknOperatorTargetProviderConfig) error {
	logger := log.FromContext(ctx)
	if config.Status.Status == "" {
		logger.Info("Initializing status to pending")
		config.Status.Status = "pending"
		now := metav1.NewTime(time.Now())
		config.Status.Created = &now
		// ConfigData map will be initialized when contributing
		if err := r.Status().Update(ctx, config); err != nil {
			return err
		}
		logger.Info("✅ Status initialized to pending")
	}
	return nil
}

// checkCompletion checks if all active providers have contributed and marks the request as completed
func (r *KrknOperatorTargetProviderConfigReconciler) checkCompletion(ctx context.Context, config *krknv1alpha1.KrknOperatorTargetProviderConfig, providerList *krknv1alpha1.KrknOperatorTargetProviderList) error {
	logger := log.FromContext(ctx)

	logger.Info("Found providers", "totalProviders", len(providerList.Items))

	// Count active providers (reuse the list from early fetch in Reconcile)
	activeProviders, activeProviderNames := countActiveProviders(providerList)

	// Log active providers
	for _, p := range providerList.Items {
		if p.Spec.Active {
			logger.Info("Active provider found",
				"name", p.Spec.OperatorName,
				"timestamp", p.Status.Timestamp)
		}
	}

	// Count contributors (operators that have added config data)
	contributorCount := len(config.Status.ConfigData)
	contributorNames := []string{}
	for name := range config.Status.ConfigData {
		contributorNames = append(contributorNames, name)
	}

	logger.Info("🔍 Checking completion",
		"activeProviders", activeProviders,
		"activeProviderNames", activeProviderNames,
		"contributors", contributorCount,
		"contributorNames", contributorNames,
		"uuid", config.Spec.UUID)

	// If all active providers have contributed, mark as completed
	if activeProviders > 0 && contributorCount >= activeProviders {
		logger.Info("✅ All active providers have contributed, marking as Completed",
			"uuid", config.Spec.UUID,
			"activeProviders", activeProviders,
			"contributors", contributorCount)
		config.Status.Status = "Completed"
		now := metav1.NewTime(time.Now())
		config.Status.Completed = &now
		if err := r.Status().Update(ctx, config); err != nil {
			return err
		}
		logger.Info("✅ Config request marked as Completed successfully")
	} else {
		logger.Info("⏳ Waiting for more providers to contribute",
			"needed", activeProviders,
			"current", contributorCount)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager
func (r *KrknOperatorTargetProviderConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	logger := mgr.GetLogger().WithName("krknoperatortargetproviderconfig-setup")
	logger.Info("🚀 Setting up KrknOperatorTargetProviderConfig controller",
		"operatorNamespace", r.OperatorNamespace)

	return ctrl.NewControllerManagedBy(mgr).
		For(&krknv1alpha1.KrknOperatorTargetProviderConfig{}).
		Named("krknoperatortargetproviderconfig").
		WithEventFilter(NewNamespaceFilter(r.OperatorNamespace)).
		Complete(r)
}
