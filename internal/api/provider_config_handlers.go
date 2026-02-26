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

package api

// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknoperatortargetproviderconfigs,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknoperatortargetproviderconfigs/status,verbs=get

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/provider"
)

// PostProviderConfig handles POST /api/v1/provider-config endpoint
// Creates a new KrknOperatorTargetProviderConfig CR and returns the UUID
func (h *Handler) PostProviderConfig(w http.ResponseWriter, r *http.Request) {
	// Use common function to create provider config request
	uuid, err := provider.CreateProviderConfigRequest(
		context.Background(),
		h.client,
		h.namespace,
		"", // Let the function generate the name
	)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create KrknOperatorTargetProviderConfig: " + err.Error(),
		})
		return
	}

	// Return 102 Processing with the UUID
	response := map[string]string{
		"uuid": uuid,
	}
	writeJSON(w, http.StatusProcessing, response)
}

// GetProviderConfigByUUID handles GET /api/v1/provider-config/{uuid} endpoint
// Returns 100 Continue when pending, 200 OK with config_data when Completed
func (h *Handler) GetProviderConfigByUUID(w http.ResponseWriter, r *http.Request) {
	uuid, err := extractPathSuffix(r.URL.Path, "/api/v1/provider-config/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "UUID " + err.Error(),
		})
		return
	}

	var config krknv1alpha1.KrknOperatorTargetProviderConfig
	if err := h.client.Get(context.Background(), types.NamespacedName{
		Name:      uuid,
		Namespace: h.namespace,
	}, &config); err != nil {
		if client.IgnoreNotFound(err) == nil {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "KrknOperatorTargetProviderConfig not found",
			})
		} else {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to fetch KrknOperatorTargetProviderConfig: " + err.Error(),
			})
		}
		return
	}

	// Return 202 Accepted when pending, 200 OK when Completed
	// The controller marks as "Completed" only when all active providers have contributed
	if config.Status.Status != "Completed" {
		// Return 202 Accepted when pending (client should retry)
		// No body needed, just the status code
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// Return 200 OK with config_data when Completed
	response := map[string]interface{}{
		"uuid":        config.Spec.UUID,
		"status":      config.Status.Status,
		"config_data": config.Status.ConfigData,
	}
	writeJSON(w, http.StatusOK, response)
}

// UpdateProviderConfigValues handles POST /api/v1/provider-config/{uuid}
// Updates a provider's ConfigMap with validated configuration values
func (h *Handler) UpdateProviderConfigValues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	// Extract UUID from path
	uuid := strings.TrimPrefix(r.URL.Path, "/api/v1/provider-config/")
	if uuid == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "UUID is required",
		})
		return
	}

	// Parse request body
	var req ProviderConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	// Validate request
	if req.ProviderName == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "provider_name is required",
		})
		return
	}

	if len(req.Values) == 0 {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "values cannot be empty",
		})
		return
	}

	// Find KrknOperatorTargetProviderConfig by UUID using label selector
	var configList krknv1alpha1.KrknOperatorTargetProviderConfigList
	if err := h.client.List(ctx, &configList, client.MatchingLabels{
		"krkn.krkn-chaos.dev/uuid": uuid,
	}, client.InNamespace(h.namespace)); err != nil {
		logger.Error(err, "Failed to list KrknOperatorTargetProviderConfig")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to query config",
		})
		return
	}

	if len(configList.Items) == 0 {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "KrknOperatorTargetProviderConfig not found",
		})
		return
	}

	config := &configList.Items[0]

	// Get provider config data from status
	if config.Status.ConfigData == nil {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: fmt.Sprintf("target provider: %s not found", req.ProviderName),
		})
		return
	}

	providerData, exists := config.Status.ConfigData[req.ProviderName]
	if !exists {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: fmt.Sprintf("target provider: %s not found", req.ProviderName),
		})
		return
	}

	// Validate all values against schema
	var updatedFields []string
	for key, value := range req.Values {
		if err := ValidateValueAgainstSchema(key, value, providerData.ConfigSchema); err != nil {
			// Check if it's a "field not found" error
			if strings.Contains(err.Error(), "not found in schema") {
				writeJSONError(w, http.StatusBadRequest, ErrorResponse{
					Error:   "bad_request",
					Message: fmt.Sprintf("field %s not found in schema", key),
				})
				return
			}
			// Validation error
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: err.Error(),
			})
			return
		}
		updatedFields = append(updatedFields, key)
	}

	// Get or create ConfigMap
	configMapName := providerData.ConfigMap
	configMapNamespace := providerData.Namespace

	var configMap corev1.ConfigMap
	err := h.client.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: configMapNamespace,
	}, &configMap)

	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create new ConfigMap
			configMap = corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMapName,
					Namespace: configMapNamespace,
				},
				Data: make(map[string]string),
			}

			// Convert values to YAML and store
			yamlData, err := convertValuesToYAML(req.Values)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
					Error:   "internal_error",
					Message: fmt.Sprintf("Failed to convert values to YAML: %v", err),
				})
				return
			}
			configMap.Data["config.yaml"] = yamlData

			if err := h.client.Create(ctx, &configMap); err != nil {
				logger.Error(err, "Failed to create ConfigMap")
				writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
					Error:   "internal_error",
					Message: "Failed to create ConfigMap",
				})
				return
			}
		} else {
			logger.Error(err, "Failed to get ConfigMap")
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get ConfigMap",
			})
			return
		}
	} else {
		// Update existing ConfigMap
		yamlData, err := mergeValuesIntoYAML(configMap.Data["config.yaml"], req.Values)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: fmt.Sprintf("Failed to merge values: %v", err),
			})
			return
		}
		configMap.Data["config.yaml"] = yamlData

		if err := h.client.Update(ctx, &configMap); err != nil {
			logger.Error(err, "Failed to update ConfigMap")
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to update ConfigMap",
			})
			return
		}
	}

	// Delete the KrknOperatorTargetProviderConfig CR after successful ConfigMap update
	if err := h.client.Delete(ctx, config); err != nil {
		logger.Error(err, "Failed to delete KrknOperatorTargetProviderConfig after ConfigMap update",
			"uuid", uuid)
		// Don't fail the request, just log the error
		// The ConfigMap was updated successfully
	} else {
		logger.Info("✅ Deleted KrknOperatorTargetProviderConfig after successful ConfigMap update",
			"uuid", uuid)
	}

	writeJSON(w, http.StatusOK, ProviderConfigUpdateResponse{
		Message:       "Configuration updated successfully",
		UpdatedFields: updatedFields,
	})
}

// convertValuesToYAML converts a map to YAML format
func convertValuesToYAML(values map[string]string) (string, error) {
	yamlBytes, err := yaml.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(yamlBytes), nil
}

// mergeValuesIntoYAML merges new values into existing YAML
func mergeValuesIntoYAML(existingYAML string, newValues map[string]string) (string, error) {
	// Parse existing YAML
	var existing map[string]interface{}
	if existingYAML != "" {
		if err := yaml.Unmarshal([]byte(existingYAML), &existing); err != nil {
			return "", err
		}
	} else {
		existing = make(map[string]interface{})
	}

	// Merge new values (supports nested keys like "api.port")
	for k, v := range newValues {
		setNestedValue(existing, k, v)
	}

	// Marshal back to YAML
	yamlBytes, err := yaml.Marshal(existing)
	if err != nil {
		return "", err
	}
	return string(yamlBytes), nil
}

// setNestedValue sets a value in a nested map using dot notation
// For example, "api.port" with value 9090 will set existing["api"]["port"] = 9090
func setNestedValue(data map[string]interface{}, key string, value interface{}) {
	parts := strings.Split(key, ".")

	// Navigate to the parent of the final key
	current := data
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]

		// Create nested map if it doesn't exist
		if _, exists := current[part]; !exists {
			current[part] = make(map[string]interface{})
		}

		// Navigate deeper
		currentMap, ok := current[part].(map[string]interface{})
		if !ok {
			// If it's not a map, replace it with a map
			currentMap = make(map[string]interface{})
			current[part] = currentMap
		}
		current = currentMap
	}

	// Set the final value
	current[parts[len(parts)-1]] = value
}

// ProviderConfigHandler handles both GET /api/v1/provider-config/{UUID} and POST /api/v1/provider-config endpoints
// It routes to the appropriate handler based on the HTTP method and path
func (h *Handler) ProviderConfigHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Root endpoint: POST to create new config request
	if path == "/api/v1/provider-config" {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
				Error:   "method_not_allowed",
				Message: "Only POST is allowed",
			})
			return
		}
		h.PostProviderConfig(w, r)
		return
	}

	// Nested endpoints with UUID
	if strings.HasPrefix(path, "/api/v1/provider-config/") {
		switch r.Method {
		case http.MethodGet:
			h.GetProviderConfigByUUID(w, r)
		case http.MethodPost:
			h.UpdateProviderConfigValues(w, r)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
				Error:   "method_not_allowed",
				Message: "Only GET and POST are allowed",
			})
		}
		return
	}

	writeJSONError(w, http.StatusNotFound, ErrorResponse{
		Error:   "not_found",
		Message: "Endpoint not found",
	})
}
