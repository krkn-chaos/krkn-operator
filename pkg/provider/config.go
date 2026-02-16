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

package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

const (
	// UUIDLabel is the label key for the UUID
	UUIDLabel = "krkn.krkn-chaos.dev/uuid"
)

// CreateProviderConfigRequest creates a new KrknOperatorTargetProviderConfig CR
// and generates a unique UUID for tracking. The UUID is set in both spec.uuid
// and as a label for easy selection.
//
// Parameters:
//   - ctx: Context
//   - c: Kubernetes client
//   - namespace: Namespace where the CR will be created
//   - name: Optional name for the CR (if empty, generates "config-" + UUID prefix)
//
// Returns:
//   - uuid: The generated UUID for this config request
//   - error: Error if creation fails
func CreateProviderConfigRequest(
	ctx context.Context,
	c client.Client,
	namespace string,
	name string,
) (string, error) {
	if namespace == "" {
		return "", fmt.Errorf("namespace cannot be empty")
	}

	// Generate UUID
	newUUID := uuid.New().String()

	// Use UUID as CR name if not provided (required for API GET to work)
	if name == "" {
		name = newUUID
	}

	// Create the CR
	config := &krknv1alpha1.KrknOperatorTargetProviderConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				UUIDLabel: newUUID,
			},
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderConfigSpec{
			UUID: newUUID,
		},
	}

	if err := c.Create(ctx, config); err != nil {
		return "", fmt.Errorf("failed to create KrknOperatorTargetProviderConfig: %w", err)
	}

	return newUUID, nil
}

// UpdateProviderConfig updates a KrknOperatorTargetProviderConfig CR with provider data.
// This function should be called by each operator's reconcile loop to contribute their configuration schema.
//
// The caller (provider controller) has already fetched the CR in the reconcile loop,
// so it simply needs to pass the CR object directly.
//
// Parameters:
//   - ctx: Context
//   - c: Kubernetes client
//   - config: The KrknOperatorTargetProviderConfig CR object (already fetched by the reconcile loop)
//   - operatorName: Name of the provider contributing the data (e.g., "krkn-operator-acm")
//   - configMapName: Name of the ConfigMap containing the provider's configuration
//   - jsonSchema: JSON schema string for the provider's configuration (must be valid JSON, not base64)
//
// Returns:
//   - error: Error if update fails or validation fails
func UpdateProviderConfig(
	ctx context.Context,
	c client.Client,
	config *krknv1alpha1.KrknOperatorTargetProviderConfig,
	operatorName string,
	configMapName string,
	jsonSchema string,
) error {
	// Validate input
	if operatorName == "" {
		return fmt.Errorf("operatorName cannot be empty")
	}
	if configMapName == "" {
		return fmt.Errorf("configMapName cannot be empty")
	}

	// Validate JSON schema if provided
	if jsonSchema != "" {
		var schemaObj interface{}
		if err := json.Unmarshal([]byte(jsonSchema), &schemaObj); err != nil {
			return fmt.Errorf("jsonSchema is not valid JSON: %w", err)
		}
	}

	// Initialize ConfigData map if nil
	if config.Status.ConfigData == nil {
		config.Status.ConfigData = make(map[string]krknv1alpha1.ProviderConfigData)
	}

	// Update provider data
	config.Status.ConfigData[operatorName] = krknv1alpha1.ProviderConfigData{
		ConfigMap:    configMapName,
		ConfigSchema: jsonSchema,
	}

	// Update status
	if err := c.Status().Update(ctx, config); err != nil {
		return fmt.Errorf("failed to update KrknOperatorTargetProviderConfig status: %w", err)
	}

	return nil
}
