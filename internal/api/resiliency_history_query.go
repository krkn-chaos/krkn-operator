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
	"context"
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
	// ClusterProviders optionally narrows selected cluster names to providers.
	// When omitted, all providers for each selected cluster name are included.
	ClusterProviders map[string][]string `json:"clusterProviders,omitempty"`
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
// @Description Return category-associated scenario and graph scores selected by non-empty category and cluster arrays. Optional clusterProviders selects providers for duplicate cluster names. Datapoints are nested as clusters[clusterName][categoryName]; providerName identifies each point's provider, and configurationGroups[categoryName] resolves each point's configurationGroupId.
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

	selections, err := parseResiliencyHistoryQuery(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: err.Error()})
		return
	}

	logger := log.FromContext(ctx).WithName("resiliency-history-query")
	isAdmin := auth.IsAdmin(ctx)
	selectedClusters := make(map[string]struct{}, len(selections.clusters))
	for _, clusterName := range selections.clusters {
		selectedClusters[clusterName] = struct{}{}
	}

	selectedCategories, accessError, accessStatus := h.loadVisibleHistoryCategories(ctx, selections.categories, claims.UserID, isAdmin)
	if accessStatus != 0 {
		writeJSONError(w, accessStatus, *accessError)
		return
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

	categoryNames := make([]string, 0, len(selectedCategories))
	for _, category := range selectedCategories {
		categoryNames = append(categoryNames, category.Name)
	}
	histories, err := h.loadCategoryResiliencyHistories(ctx, categoryNames, userGroups, selectedClusters, selections.providers)
	if err != nil {
		logger.Error(err, "Failed to load category resiliency histories", "categories", categoryNames)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to load resiliency history",
		})
		return
	}
	writeJSON(w, http.StatusOK, assembleResiliencyHistoryQueryResponse(selectedCategories, histories))
}

type resiliencyHistoryQuerySelections struct {
	categories []string
	clusters   []string
	providers  map[string]map[string]struct{}
}

func parseResiliencyHistoryQuery(body io.Reader) (resiliencyHistoryQuerySelections, error) {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	var request ResiliencyHistoryQueryRequest
	if err := decoder.Decode(&request); err != nil {
		return resiliencyHistoryQuerySelections{}, fmt.Errorf("Request body must be valid JSON with categories and clusters arrays")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return resiliencyHistoryQuerySelections{}, fmt.Errorf("Request body must contain a single JSON value")
	}
	categories, err := normalizeResiliencyHistorySelections(request.Categories)
	if err != nil {
		return resiliencyHistoryQuerySelections{}, fmt.Errorf("categories: %w", err)
	}
	clusters, err := normalizeResiliencyHistorySelections(request.Clusters)
	if err != nil {
		return resiliencyHistoryQuerySelections{}, fmt.Errorf("clusters: %w", err)
	}
	providers, err := normalizeResiliencyHistoryProviders(request.ClusterProviders, clusters)
	if err != nil {
		return resiliencyHistoryQuerySelections{}, fmt.Errorf("clusterProviders: %w", err)
	}
	for _, categoryName := range categories {
		if err := validateCategoryName(categoryName); err != nil {
			return resiliencyHistoryQuerySelections{}, fmt.Errorf("categories: %w", err)
		}
	}
	return resiliencyHistoryQuerySelections{categories: categories, clusters: clusters, providers: providers}, nil
}

func (h *Handler) loadVisibleHistoryCategories(
	ctx context.Context,
	categoryNames []string,
	userID string,
	isAdmin bool,
) ([]*krknv1alpha1.KrknCategory, *ErrorResponse, int) {
	logger := log.FromContext(ctx).WithName("resiliency-history-query")
	categories := make([]*krknv1alpha1.KrknCategory, 0, len(categoryNames))
	for _, categoryName := range categoryNames {
		category := &krknv1alpha1.KrknCategory{}
		if err := h.client.Get(ctx, client.ObjectKey{Name: categoryName, Namespace: h.namespace}, category); err != nil {
			if apierrors.IsNotFound(err) {
				return nil, &ErrorResponse{Error: "not_found", Message: "Category " + categoryName + " not found"}, http.StatusNotFound
			}
			logger.Error(err, "Failed to load category", "category", categoryName)
			return nil, &ErrorResponse{Error: "internal_error", Message: "Failed to load resiliency history query categories"}, http.StatusInternalServerError
		}
		if !isAdmin {
			visible, err := h.canViewCategory(ctx, category, userID)
			if err != nil {
				logger.Error(err, "Failed to check category visibility", "category", categoryName)
				return nil, &ErrorResponse{Error: "internal_error", Message: "Failed to validate category access"}, http.StatusInternalServerError
			}
			if !visible {
				return nil, &ErrorResponse{Error: "forbidden", Message: "You do not have permission to view category " + categoryName}, http.StatusForbidden
			}
		}
		categories = append(categories, category)
	}
	return categories, nil, 0
}

func assembleResiliencyHistoryQueryResponse(
	categories []*krknv1alpha1.KrknCategory,
	histories map[string]CategoryResiliencyHistoryResponse,
) ResiliencyHistoryQueryResponse {
	response := ResiliencyHistoryQueryResponse{
		Clusters:            make(map[string]map[string][]CategoryResiliencyDataPoint),
		ConfigurationGroups: make(map[string]map[string]CategoryConfigurationGroup, len(categories)),
	}
	for _, category := range categories {
		history := histories[category.Name]
		response.ConfigurationGroups[category.Name] = history.ConfigurationGroups
		for clusterName, points := range history.Clusters {
			if response.Clusters[clusterName] == nil {
				response.Clusters[clusterName] = make(map[string][]CategoryResiliencyDataPoint)
			}
			response.Clusters[clusterName][category.Name] = points
		}
	}
	return response
}

func normalizeResiliencyHistoryProviders(
	selections map[string][]string,
	selectedClusters []string,
) (map[string]map[string]struct{}, error) {
	if selections == nil {
		return nil, nil
	}
	clusterSet := make(map[string]struct{}, len(selectedClusters))
	for _, clusterName := range selectedClusters {
		clusterSet[clusterName] = struct{}{}
	}
	normalized := make(map[string]map[string]struct{}, len(selections))
	for clusterName, providers := range selections {
		if _, selected := clusterSet[clusterName]; !selected {
			return nil, fmt.Errorf("cluster %q must also appear in clusters", clusterName)
		}
		providerNames, err := normalizeResiliencyHistorySelections(providers)
		if err != nil {
			return nil, fmt.Errorf("cluster %q: %w", clusterName, err)
		}
		providerSet := make(map[string]struct{}, len(providerNames))
		for _, providerName := range providerNames {
			providerSet[providerName] = struct{}{}
		}
		normalized[clusterName] = providerSet
	}
	return normalized, nil
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
