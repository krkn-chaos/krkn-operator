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
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

// ListGraphRuns handles GET /api/v1/graphruns
// Lists all graph runs with optional filtering by owner
func (h *Handler) ListGraphRuns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	// Get query parameters
	ownerFilter := r.URL.Query().Get("ownerUserId")

	// List all graph runs
	var graphRunList krknv1alpha1.KrknGraphRunList
	if err := h.client.List(ctx, &graphRunList, client.InNamespace(h.namespace)); err != nil {
		logger.Error(err, "Failed to list graph runs")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list graph runs",
		})
		return
	}

	// Filter by group permissions (admins see all, users see runs with group view permission)
	graphRunList.Items = h.filterGraphRunsByGroupPermission(graphRunList.Items, ctx)

	// Filter by owner if specified
	var filteredRuns []krknv1alpha1.KrknGraphRun
	for _, run := range graphRunList.Items {
		// Apply owner filter
		if ownerFilter != "" && run.Spec.OwnerUserID != ownerFilter {
			continue
		}

		filteredRuns = append(filteredRuns, run)
	}

	// Convert to response format
	response := make([]GraphRunListItem, 0, len(filteredRuns))
	for i := range filteredRuns {
		run := &filteredRuns[i]
		response = append(response, GraphRunListItem{
			Name:              run.Name,
			Namespace:         run.Namespace,
			CreationTimestamp: run.CreationTimestamp.Time,
			Phase:             run.Status.Phase,
			OwnerUserID:       run.Spec.OwnerUserID,
			TargetRequestID:   run.Spec.TargetRequestID,
			Summary: GraphRunSummaryResponse{
				TotalNodes:     run.Status.Summary.TotalNodes,
				CompletedNodes: run.Status.Summary.CompletedNodes,
				RunningNodes:   run.Status.Summary.RunningNodes,
				FailedNodes:    run.Status.Summary.FailedNodes,
				PendingNodes:   run.Status.Summary.PendingNodes,
			},
			StartTime:      run.Status.StartTime,
			CompletionTime: run.Status.CompletionTime,
		})
	}

	writeJSON(w, http.StatusOK, GraphRunListResponse{GraphRuns: response})
}

// GetGraphRun handles GET /api/v1/graphruns/:name
// Returns detailed information about a specific graph run
func (h *Handler) GetGraphRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	// Extract name from URL path
	name := strings.TrimPrefix(r.URL.Path, GraphRunsPath+"/")
	if name == "" || name == GraphRunsPath {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Graph run name is required",
		})
		return
	}

	// Fetch the graph run
	var graphRun krknv1alpha1.KrknGraphRun
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: h.namespace,
	}, &graphRun); err != nil {
		if client.IgnoreNotFound(err) == nil {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Graph run '%s' not found", name),
			})
		} else {
			logger.Error(err, "Failed to fetch graph run", "name", name)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to fetch graph run",
			})
		}
		return
	}

	// Check authorization (admins bypass, regular users need group view permission)
	claims := auth.GetClaimsFromContext(ctx)
	if claims != nil && !auth.IsAdmin(ctx) {
		hasAccess, err := h.checkGraphRunGroupAccess(ctx, claims.UserID, &graphRun, groupauth.ActionView)
		if err != nil {
			logger.Error(err, "Failed to check graph run access", "userID", claims.UserID, "graphRunName", name)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to validate access permissions",
			})
			return
		}
		if !hasAccess {
			logger.Info("User attempted to access graph run without permission",
				"userID", claims.UserID,
				"graphRunName", name)
			writeJSONError(w, http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have permission to view this graph run",
			})
			return
		}
	}

	// Build response
	response := GraphRunDetailResponse{
		Name:              graphRun.Name,
		Namespace:         graphRun.Namespace,
		CreationTimestamp: graphRun.CreationTimestamp.Time,
		Spec: GraphRunSpecResponse{
			Graph:           graphRun.Spec.Graph,
			TargetRequestID: graphRun.Spec.TargetRequestID,
			TargetClusters:  graphRun.Spec.TargetClusters,
			OwnerUserID:     graphRun.Spec.OwnerUserID,
		},
		Status: GraphRunStatusResponse{
			Phase: graphRun.Status.Phase,
			Summary: GraphRunSummaryResponse{
				TotalNodes:     graphRun.Status.Summary.TotalNodes,
				CompletedNodes: graphRun.Status.Summary.CompletedNodes,
				RunningNodes:   graphRun.Status.Summary.RunningNodes,
				FailedNodes:    graphRun.Status.Summary.FailedNodes,
				PendingNodes:   graphRun.Status.Summary.PendingNodes,
			},
			NodeStatuses:   convertNodeStatuses(graphRun.Status.NodeStatuses),
			ResolvedLevels: graphRun.Status.ResolvedLevels,
			StartTime:      graphRun.Status.StartTime,
			CompletionTime: graphRun.Status.CompletionTime,
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// CreateGraphRun handles POST /api/v1/graphruns
// Creates a new graph run
func (h *Handler) CreateGraphRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	// Parse request body
	var req GraphRunCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if len(req.Graph) == 0 {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "graph is required and cannot be empty",
		})
		return
	}

	if req.TargetRequestID == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "targetRequestId is required",
		})
		return
	}

	if len(req.TargetClusters) == 0 {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "targetClusters is required and must contain at least one provider with clusters",
		})
		return
	}

	// Get user from JWT claims
	userClaims := auth.GetClaimsFromContext(ctx)
	if userClaims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User authentication required",
		})
		return
	}

	// Fetch KrknTargetRequest to validate permissions
	targetRequest := &krknv1alpha1.KrknTargetRequest{}
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      req.TargetRequestID,
		Namespace: h.namespace,
	}, targetRequest); err != nil {
		logger.Error(err, "Failed to fetch target request", "targetRequestId", req.TargetRequestID)
		if client.IgnoreNotFound(err) == nil {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Target request '%s' not found", req.TargetRequestID),
			})
		} else {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to fetch target request",
			})
		}
		return
	}

	// Check if target request is completed
	if targetRequest.Status.Status != "Completed" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Target request is not completed yet",
		})
		return
	}

	// Validate user permissions (group-based access control)
	// Admins bypass validation, regular users must have 'run' permission on all target clusters
	if !auth.IsAdmin(ctx) {
		// Validate user has 'run' permission on all target clusters
		if err := groupauth.ValidateScenarioRunAccess(
			ctx,
			h.client,
			userClaims.UserID,
			h.namespace,
			req.TargetClusters,
			targetRequest,
		); err != nil {
			logger.Info("User lacks permission to run graph on requested clusters",
				"userID", userClaims.UserID,
				"error", err.Error(),
			)
			writeJSONError(w, http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: err.Error(),
			})
			return
		}
	}

	// Generate unique name for the graph run
	graphRunName := fmt.Sprintf("graphrun-%s", uuid.New().String()[:8])

	// Create KrknGraphRun CR
	graphRun := &krknv1alpha1.KrknGraphRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      graphRunName,
			Namespace: h.namespace,
		},
		Spec: krknv1alpha1.KrknGraphRunSpec{
			Graph:           req.Graph,
			TargetRequestID: req.TargetRequestID,
			TargetClusters:  req.TargetClusters,
			OwnerUserID:     userClaims.UserID,
		},
	}

	if err := h.client.Create(ctx, graphRun); err != nil {
		logger.Error(err, "Failed to create graph run", "name", graphRunName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create graph run",
		})
		return
	}

	logger.Info("Graph run created successfully",
		"name", graphRunName,
		"ownerUserID", userClaims.UserID,
		"totalNodes", len(req.Graph))

	// Build response
	response := GraphRunDetailResponse{
		Name:              graphRun.Name,
		Namespace:         graphRun.Namespace,
		CreationTimestamp: graphRun.CreationTimestamp.Time,
		Spec: GraphRunSpecResponse{
			Graph:           graphRun.Spec.Graph,
			TargetRequestID: graphRun.Spec.TargetRequestID,
			TargetClusters:  graphRun.Spec.TargetClusters,
			OwnerUserID:     graphRun.Spec.OwnerUserID,
		},
		Status: GraphRunStatusResponse{
			Phase: graphRun.Status.Phase,
		},
	}

	writeJSON(w, http.StatusCreated, response)
}

// DeleteGraphRun handles DELETE /api/v1/graphruns/:name
// Deletes a graph run (cascade deletes scenario runs via owner references)
func (h *Handler) DeleteGraphRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	// Extract name from URL path
	name := strings.TrimPrefix(r.URL.Path, GraphRunsPath+"/")
	if name == "" || name == GraphRunsPath {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Graph run name is required",
		})
		return
	}

	// Fetch the graph run first to check ownership
	var graphRun krknv1alpha1.KrknGraphRun
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: h.namespace,
	}, &graphRun); err != nil {
		if client.IgnoreNotFound(err) == nil {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Graph run '%s' not found", name),
			})
		} else {
			logger.Error(err, "Failed to fetch graph run", "name", name)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to fetch graph run",
			})
		}
		return
	}

	// Check ownership authorization
	userClaims := auth.GetClaimsFromContext(ctx)
	if userClaims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User authentication required",
		})
		return
	}

	// Only owner or admin can delete
	if !auth.IsAdmin(ctx) && graphRun.Spec.OwnerUserID != userClaims.UserID {
		logger.Info("User attempted to delete graph run they don't own",
			"userID", userClaims.UserID,
			"ownerUserID", graphRun.Spec.OwnerUserID,
			"graphRunName", name)
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "You can only delete your own graph runs",
		})
		return
	}

	// Delete the graph run
	if err := h.client.Delete(ctx, &graphRun); err != nil {
		logger.Error(err, "Failed to delete graph run", "name", name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete graph run",
		})
		return
	}

	logger.Info("Graph run deleted successfully",
		"name", name,
		"ownerUserID", graphRun.Spec.OwnerUserID)

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

// convertNodeStatuses converts Kubernetes NodeStatus to API response format
func convertNodeStatuses(nodeStatuses []krknv1alpha1.NodeStatus) []NodeStatusResponse {
	if nodeStatuses == nil {
		return []NodeStatusResponse{}
	}

	result := make([]NodeStatusResponse, 0, len(nodeStatuses))
	for _, ns := range nodeStatuses {
		result = append(result, NodeStatusResponse{
			NodeID:         ns.NodeID,
			NodeName:       ns.NodeName,
			Phase:          ns.Phase,
			ScenarioRunRef: ns.ScenarioRunRef,
			StartTime:      ns.StartTime,
			CompletionTime: ns.CompletionTime,
			DependsOn:      ns.DependsOn,
			Message:        ns.Message,
		})
	}

	return result
}

// GraphRunsRouter routes GraphRun HTTP requests to appropriate handlers
func (h *Handler) GraphRunsRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Root endpoint: /api/v1/graphruns
	if path == GraphRunsPath {
		switch r.Method {
		case http.MethodGet:
			h.ListGraphRuns(w, r)
		case http.MethodPost:
			h.CreateGraphRun(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Nested endpoints: /api/v1/graphruns/:name
	if strings.HasPrefix(path, GraphRunsPath+"/") {
		switch r.Method {
		case http.MethodGet:
			h.GetGraphRun(w, r)
		case http.MethodDelete:
			h.DeleteGraphRun(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}
