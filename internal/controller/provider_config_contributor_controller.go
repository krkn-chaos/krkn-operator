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
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/provider"
)

// ProviderConfigContributorReconciler contributes krkn-operator's configuration to config requests
type ProviderConfigContributorReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	OperatorName      string
	OperatorNamespace string
}

// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknoperatortargetproviderconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknoperatortargetproviderconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch

// Reconcile watches for new KrknOperatorTargetProviderConfig CRs and contributes krkn-operator's configuration
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

	logger.Info("Contributing configuration for krkn-operator", "uuid", config.Spec.UUID)

	// Prepare configuration
	configMapName, jsonSchema, err := r.prepareConfiguration(ctx)
	if err != nil {
		logger.Error(err, "Failed to prepare configuration")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Contribute data using common function
	if err := provider.UpdateProviderConfig(
		ctx,
		r.Client,
		&config, // Pass the CR object directly
		r.OperatorName,
		configMapName,
		r.OperatorNamespace,
		jsonSchema,
	); err != nil {
		logger.Error(err, "Failed to update provider config")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	logger.Info("✅ Successfully contributed krkn-operator configuration",
		"uuid", config.Spec.UUID,
		"configMapName", configMapName)

	return ctrl.Result{}, nil
}

// prepareConfiguration prepares krkn-operator's configuration data
// Returns: configMapName, jsonSchema (as JSON string), error
func (r *ProviderConfigContributorReconciler) prepareConfiguration(ctx context.Context) (string, string, error) {
	logger := log.FromContext(ctx)
	configMapName := "krkn-operator-config"

	// Create or update the ConfigMap with krkn-operator configuration
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: r.OperatorNamespace,
		},
		Data: map[string]string{
			"config.yaml": `api:
  port: 8080
  enabled: true
scenarios:
  default-timeout: 600s
provider:
  heartbeat-interval: 30s
`,
		},
	}

	// Try to get existing ConfigMap
	var existingConfigMap corev1.ConfigMap
	err := r.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: r.OperatorNamespace,
	}, &existingConfigMap)

	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create new ConfigMap
			if err := r.Create(ctx, configMap); err != nil {
				logger.Error(err, "Failed to create ConfigMap")
				return "", "", err
			}
			logger.Info("✅ Created krkn-operator ConfigMap", "name", configMapName)
		} else {
			return "", "", err
		}
	} else {
		// ConfigMap already exists, update if needed
		existingConfigMap.Data = configMap.Data
		if err := r.Update(ctx, &existingConfigMap); err != nil {
			logger.Error(err, "Failed to update ConfigMap")
			return "", "", err
		}
		logger.Info("✅ Updated krkn-operator ConfigMap", "name", configMapName)
	}

	// Define JSON schema for krkn-operator configuration
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"api": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"port": map[string]interface{}{
						"type":        "number",
						"description": "Port for the REST API server",
						"default":     8080,
					},
					"enabled": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether the REST API is enabled",
						"default":     true,
					},
				},
			},
			"scenarios": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"default-timeout": map[string]interface{}{
						"type":        "string",
						"description": "Default timeout for scenario execution",
						"default":     "600s",
						"pattern":     "^[0-9]+(s|m|h)$",
					},
				},
			},
			"provider": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"heartbeat-interval": map[string]interface{}{
						"type":        "string",
						"description": "Interval for provider heartbeat updates",
						"default":     "30s",
						"pattern":     "^[0-9]+(s|m|h)$",
					},
				},
			},
		},
	}

	// Marshal schema to JSON string
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		logger.Error(err, "Failed to marshal JSON schema")
		return "", "", err
	}

	return configMapName, string(schemaBytes), nil
}

// SetupWithManager sets up the controller with the Manager
func (r *ProviderConfigContributorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	logger := mgr.GetLogger().WithName("provider-config-contributor-setup")
	logger.Info("🚀 Setting up ProviderConfigContributor controller",
		"operatorName", r.OperatorName,
		"operatorNamespace", r.OperatorNamespace)

	return ctrl.NewControllerManagedBy(mgr).
		For(&krknv1alpha1.KrknOperatorTargetProviderConfig{}).
		Named("provider-config-contributor").
		WithEventFilter(NewNamespaceFilter(r.OperatorNamespace)).
		Complete(r)
}
