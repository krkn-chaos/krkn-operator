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

import (
	"context"
	"net/http"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
			w.WriteHeader(http.StatusNotFound)
		} else {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to fetch KrknOperatorTargetProviderConfig: " + err.Error(),
			})
		}
		return
	}

	if config.Status.Status != "Completed" {
		// Return 100 Continue when pending
		w.WriteHeader(http.StatusContinue)
	} else {
		// Return 200 OK with config_data when Completed
		response := map[string]interface{}{
			"uuid":        config.Spec.UUID,
			"status":      config.Status.Status,
			"config_data": config.Status.ConfigData,
		}
		writeJSON(w, http.StatusOK, response)
	}
}

// ProviderConfigHandler handles both GET /api/v1/provider-config/{UUID} and POST /api/v1/provider-config endpoints
// It routes to the appropriate handler based on the HTTP method
func (h *Handler) ProviderConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.GetProviderConfigByUUID(w, r)
	} else if r.Method == http.MethodPost {
		h.PostProviderConfig(w, r)
	} else {
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only GET and POST methods are allowed",
		})
	}
}
