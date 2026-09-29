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

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ResiliencyHistoryQueryRequest selects category history for the given clusters.
type ResiliencyHistoryQueryRequest struct {
	// Categories selects the categories included in the query.
	Categories []string `json:"categories"`
	// Clusters selects the clusters included in the query.
	Clusters []string `json:"clusters"`
}

// ResiliencyHistoryQueryResponse groups score datapoints by cluster and then
// category. Configuration group IDs are scoped by category and resolved through
// ConfigurationGroups[categoryName].
type ResiliencyHistoryQueryResponse struct {
	// Clusters maps cluster names to category names to sorted score datapoints.
	Clusters map[string]map[string][]CategoryResiliencyDataPoint `json:"clusters"`
	// ConfigurationGroups maps category names to configuration group IDs.
	ConfigurationGroups map[string]map[string]CategoryConfigurationGroup `json:"configurationGroups"`
}

// QueryResiliencyHistory handles POST /api/v2/resiliency-history.
//
// @Summary Query resiliency score history
// @Description Return category-associated scenario and graph scores selected by non-empty category and cluster arrays. Datapoints are nested as clusters[clusterName][categoryName]; configurationGroups[categoryName] resolves each datapoint's configurationGroupId.
// @Tags resiliency-history
// @Accept json
// @Produce json
// @Param request body ResiliencyHistoryQueryRequest true "History filters"
// @Success 200 {object} ResiliencyHistoryQueryResponse
// @Failure 400 {object} ErrorResponse "Invalid query"
// @Failure 401 {object} ErrorResponse "Authentication required"
// @Failure 403 {object} ErrorResponse "The caller cannot access a category"
// @Failure 404 {object} ErrorResponse "Category not found"
// @Failure 405 {object} ErrorResponse "Method not allowed"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v2/resiliency-history [post]
func (h *Handler) QueryResiliencyHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only POST is allowed for resiliency history queries",
		})
		return
	}

	ctx := r.Context()
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request ResiliencyHistoryQueryRequest
	if err := decoder.Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Request body must be valid JSON with categories and clusters arrays",
		})
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Request body must contain a single JSON value",
		})
		return
	}

	categories, err := normalizeResiliencyHistorySelections(request.Categories)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "categories: " + err.Error()})
		return
	}
	clusters, err := normalizeResiliencyHistorySelections(request.Clusters)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "clusters: " + err.Error()})
		return
	}
	for _, categoryName := range categories {
		if err := validateCategoryName(categoryName); err != nil {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "categories: " + err.Error()})
			return
		}
	}

	logger := log.FromContext(ctx).WithName("resiliency-history-query")
	isAdmin := auth.IsAdmin(ctx)
	selectedClusters := make(map[string]struct{}, len(clusters))
	for _, clusterName := range clusters {
		selectedClusters[clusterName] = struct{}{}
	}

	selectedCategories := make([]*krknv1alpha1.KrknCategory, 0, len(categories))
	for _, categoryName := range categories {
		category := &krknv1alpha1.KrknCategory{}
		if err := h.client.Get(ctx, client.ObjectKey{Name: categoryName, Namespace: h.namespace}, category); err != nil {
			if apierrors.IsNotFound(err) {
				writeJSONError(w, http.StatusNotFound, ErrorResponse{
					Error:   "not_found",
					Message: "Category " + categoryName + " not found",
				})
				return
			}
			logger.Error(err, "Failed to load category", "category", categoryName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to load resiliency history query categories",
			})
			return
		}
		if !isAdmin {
			visible, err := h.canViewCategory(ctx, category, claims.UserID)
			if err != nil {
				logger.Error(err, "Failed to check category visibility", "category", categoryName)
				writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
					Error:   "internal_error",
					Message: "Failed to validate category access",
				})
				return
			}
			if !visible {
				writeJSONError(w, http.StatusForbidden, ErrorResponse{
					Error:   "forbidden",
					Message: "You do not have permission to view category " + categoryName,
				})
				return
			}
		}
		selectedCategories = append(selectedCategories, category)
	}

	var userGroups []krknv1alpha1.KrknUserGroup
	if !isAdmin {
		groups, err := groupauth.GetUserGroups(ctx, h.client, claims.UserID, h.namespace)
		if err != nil {
			logger.Error(err, "Failed to load cluster visibility permissions")
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to validate cluster visibility",
			})
			return
		}
		userGroups = groups
	}

	response := ResiliencyHistoryQueryResponse{
		Clusters:            make(map[string]map[string][]CategoryResiliencyDataPoint),
		ConfigurationGroups: make(map[string]map[string]CategoryConfigurationGroup, len(selectedCategories)),
	}
	categoryNames := make([]string, 0, len(selectedCategories))
	for _, category := range selectedCategories {
		categoryNames = append(categoryNames, category.Name)
	}
	histories, err := h.loadCategoryResiliencyHistories(ctx, categoryNames, userGroups, selectedClusters)
	if err != nil {
		logger.Error(err, "Failed to load category resiliency histories", "categories", categoryNames)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to load resiliency history",
		})
		return
	}
	for _, category := range selectedCategories {
		history := histories[category.Name]
		response.ConfigurationGroups[category.Name] = history.ConfigurationGroups
		for clusterName, points := range history.Clusters {
			if response.Clusters[clusterName] == nil {
				response.Clusters[clusterName] = make(map[string][]CategoryResiliencyDataPoint)
			}
			response.Clusters[clusterName][category.Name] = points
		}
	}

	writeJSON(w, http.StatusOK, response)
}

func normalizeResiliencyHistorySelections(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("must contain at least one value")
	}
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" || strings.TrimSpace(value) != value {
			return nil, fmt.Errorf("values must be non-empty and have no surrounding whitespace")
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique, nil
}
