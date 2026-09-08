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
	"fmt"
	"net/http"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

// GetScenarioRunConfig handles GET /api/v1/scenarios/run/{scenarioRunName}/config
// Returns the configuration payload for a scenario run, ready for replay via POST /scenarios/run
//
// @Summary Get scenario run configuration
// @Description Retrieve scenario configuration directly by KrknScenarioRun CR name
// @Tags scenarios
// @Produce json
// @Param scenarioRunName path string true "KrknScenarioRun CR name"
// @Success 200 {object} ScenarioRunRequest "Scenario configuration"
// @Failure 400 {object} ErrorResponse "Missing scenario run name"
// @Failure 401 {object} ErrorResponse "User authentication required"
// @Failure 403 {object} ErrorResponse "Insufficient permissions"
// @Failure 404 {object} ErrorResponse "ScenarioRun not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /scenarios/run/{scenarioRunName}/config [get]
func (h *Handler) GetScenarioRunConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	// Extract scenarioRunName from URL path
	// Path format: /api/v1/scenarios/run/{scenarioRunName}/config
	trimmed := strings.TrimPrefix(r.URL.Path, ScenariosRunPath+"/")
	scenarioRunName := strings.TrimSuffix(trimmed, "/config")
	if scenarioRunName == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Scenario run name is required",
		})
		return
	}

	logger.V(1).Info("Scenario run config request received", "scenarioRunName", scenarioRunName)

	// Fetch KrknScenarioRun CR
	scenarioRun, err := h.getScenarioRun(ctx, scenarioRunName)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("ScenarioRun '%s' not found", scenarioRunName),
			})
		} else {
			logger.Error(err, "Failed to get ScenarioRun", "scenarioRunName", scenarioRunName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to retrieve ScenarioRun",
			})
		}
		return
	}

	// Validate RBAC permissions
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User authentication required",
		})
		return
	}

	if !auth.IsAdmin(ctx) {
		targetRequest := &krknv1alpha1.KrknTargetRequest{}
		if err := h.client.Get(ctx, types.NamespacedName{
			Name:      scenarioRun.Spec.TargetRequestID,
			Namespace: h.namespace,
		}, targetRequest); err != nil {
			logger.Error(err, "Failed to fetch target request", "targetRequestId", scenarioRun.Spec.TargetRequestID)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to validate permissions",
			})
			return
		}

		if err := groupauth.ValidateScenarioRunAccess(
			ctx,
			h.client,
			claims.UserID,
			h.namespace,
			scenarioRun.Spec.TargetClusters,
			targetRequest,
		); err != nil {
			logger.Info("User lacks permission to view scenario run config",
				"userID", claims.UserID,
				"error", err.Error(),
			)
			writeJSONError(w, http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: err.Error(),
			})
			return
		}
	}

	// Reconstruct payload (same as replay)
	payload, err := h.reconstructScenarioRunPayload(ctx, scenarioRun)
	if err != nil {
		logger.Error(err, "Failed to reconstruct scenario payload", "scenarioRunName", scenarioRunName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to reconstruct scenario configuration",
		})
		return
	}

	logger.Info("Scenario run config retrieved successfully",
		"scenarioRunName", scenarioRunName,
		"userID", claims.UserID)

	writeJSON(w, http.StatusOK, payload)
}

// GetGraphRunConfig handles GET /api/v1/graphruns/{graphRunName}/config
// Returns the configuration payload for a graph run, ready for replay via POST /graphruns
//
// @Summary Get graph run configuration
// @Description Retrieve graph run configuration directly by KrknGraphRun CR name
// @Tags graphruns
// @Produce json
// @Param graphRunName path string true "KrknGraphRun CR name"
// @Success 200 {object} GraphRunCreateRequest "Graph run configuration"
// @Failure 400 {object} ErrorResponse "Missing graph run name"
// @Failure 401 {object} ErrorResponse "User authentication required"
// @Failure 403 {object} ErrorResponse "Insufficient permissions"
// @Failure 404 {object} ErrorResponse "GraphRun not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /graphruns/{graphRunName}/config [get]
func (h *Handler) GetGraphRunConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	// Extract graphRunName from URL path
	// Path format: /api/v1/graphruns/{graphRunName}/config
	trimmed := strings.TrimPrefix(r.URL.Path, GraphRunsPath+"/")
	graphRunName := strings.TrimSuffix(trimmed, "/config")
	if graphRunName == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Graph run name is required",
		})
		return
	}

	logger.V(1).Info("Graph run config request received", "graphRunName", graphRunName)

	// Fetch KrknGraphRun CR
	var graphRun krknv1alpha1.KrknGraphRun
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      graphRunName,
		Namespace: h.namespace,
	}, &graphRun); err != nil {
		if client.IgnoreNotFound(err) == nil {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("GraphRun '%s' not found", graphRunName),
			})
		} else {
			logger.Error(err, "Failed to get GraphRun", "graphRunName", graphRunName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to retrieve GraphRun",
			})
		}
		return
	}

	// Validate RBAC permissions
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User authentication required",
		})
		return
	}

	if !auth.IsAdmin(ctx) {
		hasAccess, err := h.checkGraphRunGroupAccess(ctx, claims.UserID, &graphRun, groupauth.ActionView)
		if err != nil {
			logger.Error(err, "Failed to check graph run access", "userID", claims.UserID, "graphRunName", graphRunName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to validate access permissions",
			})
			return
		}
		if !hasAccess {
			logger.Info("User lacks permission to view graph run config",
				"userID", claims.UserID,
				"graphRunName", graphRunName)
			writeJSONError(w, http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have permission to view this graph run",
			})
			return
		}
	}

	// Reconstruct the creation payload from the spec
	payload := GraphRunCreateRequest{
		Graph:           graphRun.Spec.Graph,
		TargetRequestID: graphRun.Spec.TargetRequestID,
		TargetClusters:  graphRun.Spec.TargetClusters,
	}

	logger.Info("Graph run config retrieved successfully",
		"graphRunName", graphRunName,
		"userID", claims.UserID)

	writeJSON(w, http.StatusOK, payload)
}