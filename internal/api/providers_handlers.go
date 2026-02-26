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
	"encoding/json"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknoperatortargetproviders,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknoperatortargetproviders/status,verbs=get

// ListProviders handles GET /api/v1/providers endpoint
// Returns a list of all KrknOperatorTargetProvider resources
func (h *Handler) ListProviders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	// List all KrknOperatorTargetProvider CRs
	var providerList krknv1alpha1.KrknOperatorTargetProviderList
	if err := h.client.List(ctx, &providerList); err != nil {
		logger.Error(err, "Failed to list KrknOperatorTargetProvider CRs")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list providers",
		})
		return
	}

	// Build response
	providers := make([]ProviderResponse, 0, len(providerList.Items))
	for _, provider := range providerList.Items {
		var lastHeartbeat *metav1.Time
		if !provider.Status.Timestamp.IsZero() {
			lastHeartbeat = &provider.Status.Timestamp
		}
		providers = append(providers, ProviderResponse{
			Name:          provider.Spec.OperatorName,
			Active:        provider.Spec.Active,
			LastHeartbeat: lastHeartbeat,
		})
	}

	writeJSON(w, http.StatusOK, ListProvidersResponse{
		Providers: providers,
	})
}

// UpdateProviderStatus handles PATCH /api/v1/providers/{name} endpoint
// Activates or deactivates a provider
func (h *Handler) UpdateProviderStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	// Extract provider name from path
	providerName := strings.TrimPrefix(r.URL.Path, "/api/v1/providers/")
	if providerName == "" || providerName == "/api/v1/providers/" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Provider name is required",
		})
		return
	}

	// Parse request body
	var req UpdateProviderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
		return
	}

	// Find provider by name (iterate through all providers in all namespaces)
	var providerList krknv1alpha1.KrknOperatorTargetProviderList
	if err := h.client.List(ctx, &providerList); err != nil {
		logger.Error(err, "Failed to list providers")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to query providers",
		})
		return
	}

	// Find the matching provider
	var targetProvider *krknv1alpha1.KrknOperatorTargetProvider
	for i := range providerList.Items {
		if providerList.Items[i].Spec.OperatorName == providerName {
			targetProvider = &providerList.Items[i]
			break
		}
	}

	if targetProvider == nil {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Provider not found",
		})
		return
	}

	// Update the active status
	targetProvider.Spec.Active = req.Active
	if err := h.client.Update(ctx, targetProvider); err != nil {
		logger.Error(err, "Failed to update provider",
			"provider", providerName,
			"active", req.Active)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update provider status",
		})
		return
	}

	logger.Info("Provider status updated",
		"provider", providerName,
		"active", req.Active)

	writeJSON(w, http.StatusOK, UpdateProviderStatusResponse{
		Message: "Provider status updated successfully",
		Name:    providerName,
		Active:  req.Active,
	})
}

// ProvidersRouter routes provider-related requests
func (h *Handler) ProvidersRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Root endpoint: GET to list all providers
	if path == "/api/v1/providers" {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
				Error:   "method_not_allowed",
				Message: "Only GET is allowed",
			})
			return
		}
		h.ListProviders(w, r)
		return
	}

	// Provider-specific endpoint: PATCH to update status (admin only)
	if strings.HasPrefix(path, "/api/v1/providers/") {
		if r.Method != http.MethodPatch {
			writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
				Error:   "method_not_allowed",
				Message: "Only PATCH is allowed",
			})
			return
		}

		// PATCH requires admin
		if !h.requireAdminForMethods(w, r, []string{http.MethodPatch}) {
			return
		}

		h.UpdateProviderStatus(w, r)
		return
	}

	writeJSONError(w, http.StatusNotFound, ErrorResponse{
		Error:   "not_found",
		Message: "Endpoint not found",
	})
}
