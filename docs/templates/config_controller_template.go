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

================================================================================
PROVIDER CONFIG CONTROLLER TEMPLATE
================================================================================

This file provides a template for implementing a provider config contributor
controller in your operator. Copy this template to your operator's controller
package and customize it according to your needs.

OVERVIEW:
---------
Each operator in the krkn-operator ecosystem should implement a controller that:
1. Watches for new KrknOperatorTargetProviderConfig CRs
2. Contributes its own configuration schema when a new config request is created
3. Uses the UpdateProviderConfig() common function to contribute data

STEPS TO IMPLEMENT:
-------------------
1. Copy this file to your operator's internal/controller/ directory
2. Rename to something like: provider_config_contributor.go
3. Update package name to match your controller package
4. Replace "YOUR_OPERATOR_NAME" with your operator's name (e.g., "krkn-operator-acm")
5. Customize prepareConfiguration() to create your ConfigMap and JSON schema
6. Wire the controller in your main.go (see examples below)
7. Add required RBAC markers

DEPENDENCIES:
-------------
You need to import the following packages:
- github.com/krkn-chaos/krkn-operator/api/v1alpha1 (for CRD types)
- github.com/krkn-chaos/krkn-operator/pkg/provider (for UpdateProviderConfig)

RBAC REQUIREMENTS:
------------------
Add these RBAC markers to your controller:
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknoperatortargetproviderconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknoperatortargetproviderconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch

Then run: make manifests

WIRING IN main.go:
------------------
Add this to your main.go (around where other controllers are set up):

	if err = (&controller.ProviderConfigContributorReconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		OperatorName:      "YOUR_OPERATOR_NAME",  // e.g., "krkn-operator-acm"
		OperatorNamespace: operatorNamespace,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ProviderConfigContributor")
		os.Exit(1)
	}

EXAMPLE USAGE:
--------------
See krkn-operator's provider_config_contributor_controller.go for a complete
working example.

================================================================================
*/

package controller

import (
	"context"
	"encoding/json"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/provider"
)

// ProviderConfigContributorReconciler contributes this operator's configuration to config requests
//
// CUSTOMIZE: Update the comment to reflect your operator's name
type ProviderConfigContributorReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	OperatorName      string // IMPORTANT: Set this to your operator's name (e.g., "krkn-operator-acm")
	OperatorNamespace string
}

// Reconcile watches for new KrknOperatorTargetProviderConfig CRs and contributes this operator's configuration
//
// CUSTOMIZE: Update log messages if needed, but the flow should remain the same
func (r *ProviderConfigContributorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("🔄 Reconciling provider config contribution",
		"name", req.Name,
		"namespace", req.Namespace,
		"operatorName", r.OperatorName)

	// Fetch the config request
	var config krknv1alpha1.KrknOperatorTargetProviderConfig
	if err := r.Get(ctx, req.NamespacedName, &config); err != nil {
		if client.IgnoreNotFound(err) == nil {
			logger.Info("KrknOperatorTargetProviderConfig not found, probably deleted", "name", req.Name)
		} else {
			logger.Error(err, "Failed to get KrknOperatorTargetProviderConfig")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Skip if already completed
	if config.Status.Status == "Completed" {
		logger.Info("Config request already completed, skipping", "uuid", config.Spec.UUID)
		return ctrl.Result{}, nil
	}

	// Skip if we've already contributed
	if config.Status.ConfigData != nil {
		if _, exists := config.Status.ConfigData[r.OperatorName]; exists {
			logger.Info("Already contributed configuration, skipping", "uuid", config.Spec.UUID)
			return ctrl.Result{}, nil
		}
	}

	logger.Info("Contributing configuration", "uuid", config.Spec.UUID, "operator", r.OperatorName)

	// Prepare configuration
	// CUSTOMIZE: This method should be customized to your operator's needs
	configMapName, jsonSchema, err := r.prepareConfiguration(ctx)
	if err != nil {
		logger.Error(err, "Failed to prepare configuration")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Contribute data using common function
	// DO NOT MODIFY: This uses the standard UpdateProviderConfig function
	// Pass the CR object directly (no need to refetch it!)
	if err := provider.UpdateProviderConfig(
		ctx,
		r.Client,
		&config, // Pass the CR object we already fetched
		r.OperatorName,
		configMapName,
		jsonSchema,
	); err != nil {
		logger.Error(err, "Failed to update provider config")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	logger.Info("✅ Successfully contributed configuration",
		"uuid", config.Spec.UUID,
		"operator", r.OperatorName,
		"configMapName", configMapName)

	return ctrl.Result{}, nil
}

// prepareConfiguration prepares this operator's configuration data
// Returns: configMapName, jsonSchema (as JSON string), error
//
// CUSTOMIZE: This is the main method you need to customize for your operator.
// Create/update your ConfigMap and define your JSON schema here.
func (r *ProviderConfigContributorReconciler) prepareConfiguration(ctx context.Context) (string, string, error) {
	logger := log.FromContext(ctx)

	// CUSTOMIZE: Set your ConfigMap name (e.g., "acm-operator-config")
	configMapName := "YOUR_OPERATOR_NAME-config"

	// CUSTOMIZE: Define your configuration data
	// This is an example - replace with your actual configuration
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: r.OperatorNamespace,
		},
		Data: map[string]string{
			"config.yaml": `# CUSTOMIZE: Your operator's configuration here
example-setting: value
another-setting: 123
`,
		},
	}

	// Create or update the ConfigMap
	// DO NOT MODIFY: This is standard ConfigMap creation/update logic
	var existingConfigMap corev1.ConfigMap
	err := r.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: r.OperatorNamespace,
	}, &existingConfigMap)

	if err != nil {
		if apierrors.IsNotFound(err) {
			if err := r.Create(ctx, configMap); err != nil {
				logger.Error(err, "Failed to create ConfigMap")
				return "", "", err
			}
			logger.Info("✅ Created ConfigMap", "name", configMapName)
		} else {
			return "", "", err
		}
	} else {
		existingConfigMap.Data = configMap.Data
		if err := r.Update(ctx, &existingConfigMap); err != nil {
			logger.Error(err, "Failed to update ConfigMap")
			return "", "", err
		}
		logger.Info("✅ Updated ConfigMap", "name", configMapName)
	}

	// CUSTOMIZE: Define your JSON schema
	// This schema describes the structure and validation rules for your configuration
	// See: https://json-schema.org/ for more details
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			// EXAMPLE: Replace this with your actual configuration fields
			"example-setting": map[string]interface{}{
				"type":        "string",
				"description": "Description of this setting",
				"default":     "value",
			},
			"another-setting": map[string]interface{}{
				"type":        "number",
				"description": "Description of this numeric setting",
				"minimum":     0,
				"maximum":     1000,
			},
			// Add more properties as needed for your configuration
		},
		// OPTIONAL: Specify which fields are required
		"required": []string{"example-setting"},
	}

	// Marshal schema to JSON string
	// DO NOT MODIFY: This is standard JSON marshaling
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		logger.Error(err, "Failed to marshal JSON schema")
		return "", "", err
	}

	return configMapName, string(schemaBytes), nil
}

// SetupWithManager sets up the controller with the Manager
//
// DO NOT MODIFY: This is standard controller setup
func (r *ProviderConfigContributorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	logger := mgr.GetLogger().WithName("provider-config-contributor-setup")
	logger.Info("🚀 Setting up ProviderConfigContributor controller",
		"operatorName", r.OperatorName,
		"operatorNamespace", r.OperatorNamespace)

	return ctrl.NewControllerManagedBy(mgr).
		For(&krknv1alpha1.KrknOperatorTargetProviderConfig{}).
		Named("provider-config-contributor").
		WithEventFilter(r.namespaceFilter()).
		Complete(r)
}

// namespaceFilter creates a predicate that filters events by namespace
//
// DO NOT MODIFY: This ensures the controller only processes CRs in its namespace
func (r *ProviderConfigContributorReconciler) namespaceFilter() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		return obj.GetNamespace() == r.OperatorNamespace
	})
}
