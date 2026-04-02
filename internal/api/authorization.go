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
	"fmt"
	"net/http"
	"strings"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// requireAdminForMethods checks if the user is admin for specific HTTP methods
// If the method requires admin and user is not admin, returns false and writes error response
func (h *Handler) requireAdminForMethods(w http.ResponseWriter, r *http.Request, methods []string) bool {
	// Check if current method requires admin
	requiresAdmin := false
	for _, method := range methods {
		if r.Method == method {
			requiresAdmin = true
			break
		}
	}

	if !requiresAdmin {
		return true // Method doesn't require admin, allow
	}

	// Check if user is admin
	if !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return false
	}

	return true
}

// sanitizeUserID converts an email address to a valid Kubernetes label value.
// Replaces @ and . with -, then converts to lowercase to comply with
// Kubernetes label value requirements (RFC 1123).
//
// Example: "user@example.com" -> "user-example-com"
func sanitizeUserID(email string) string {
	sanitized := strings.ReplaceAll(email, "@", "-")
	sanitized = strings.ReplaceAll(sanitized, ".", "-")
	return strings.ToLower(sanitized)
}

// checkScenarioRunAccess verifies if the authenticated user has permission to access
// the given scenario run using group-based permissions.
// This is a convenience wrapper that checks for 'view' permission.
//
// For other permissions (e.g., 'cancel'), use checkScenarioRunAccessWithAction.
//
// Access rules:
// - Admins can access all scenario runs
// - Regular users must have 'view' permission on ANY cluster in the run via their groups
// - Scenario runs without ClusterAPIURLs are rejected (defensive check)
//
// If access is denied, this function writes a 403 Forbidden response to the writer
// and returns false. The caller should return immediately without further processing.
//
// Parameters:
//   - w: The HTTP response writer
//   - r: The HTTP request containing user claims in context
//   - scenarioRun: The scenario run to check access for
//
// Returns true if access is allowed, false if denied (with error written to response)
func (h *Handler) checkScenarioRunAccess(w http.ResponseWriter, r *http.Request, scenarioRun *krknv1alpha1.KrknScenarioRun) bool {
	return h.checkScenarioRunAccessWithAction(w, r, scenarioRun, groupauth.ActionView, "view")
}

// checkScenarioRunAccessWithAction verifies if the authenticated user has a specific permission
// on the given scenario run using group-based permissions.
//
// Access rules:
// - Admins can perform all actions on all scenario runs
// - Regular users must have the required permission on ANY cluster in the run via their groups
// - Scenario runs without ClusterAPIURLs are rejected (defensive check)
//
// If access is denied, this function writes a 403 Forbidden response to the writer
// and returns false. The caller should return immediately without further processing.
//
// Parameters:
//   - w: The HTTP response writer
//   - r: The HTTP request containing user claims in context
//   - scenarioRun: The scenario run to check access for
//   - requiredAction: The action to check (e.g., ActionView, ActionCancel)
//   - actionName: Human-readable action name for error messages
//
// Returns true if access is allowed, false if denied (with error written to response)
func (h *Handler) checkScenarioRunAccessWithAction(
	w http.ResponseWriter,
	r *http.Request,
	scenarioRun *krknv1alpha1.KrknScenarioRun,
	requiredAction groupauth.Action,
	actionName string,
) bool {
	ctx := r.Context()
	claims := auth.GetClaimsFromContext(ctx)

	// Defensive check - should never happen with RequireAuth middleware
	if claims == nil {
		http.Error(w, `{"error":"unauthorized","message":"No authentication claims found"}`, http.StatusUnauthorized)
		return false
	}

	// Admins bypass all checks
	if auth.IsAdmin(ctx) {
		return true
	}

	// Reject runs without jobs (defensive - should not happen for new runs)
	if len(scenarioRun.Status.ClusterJobs) == 0 {
		http.Error(w, `{"error":"forbidden","message":"Access denied. This scenario run has no jobs"}`, http.StatusForbidden)
		return false
	}

	// Check if user has required permission on ANY job in this run
	hasAccess, err := h.checkScenarioRunGroupAccess(
		ctx,
		claims.UserID,
		scenarioRun,
		requiredAction,
	)

	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to check scenario run access",
			"userID", claims.UserID,
			"action", requiredAction)
		http.Error(w, `{"error":"internal_error","message":"Failed to validate access"}`, http.StatusInternalServerError)
		return false
	}

	if !hasAccess {
		errorMsg := fmt.Sprintf(`{"error":"forbidden","message":"Access denied. You do not have permission to %s this scenario run"}`, actionName)
		http.Error(w, errorMsg, http.StatusForbidden)
		return false
	}

	return true
}

// checkScenarioRunGroupAccess checks if user has the specified permission on ANY job in the scenario run.
// Returns true if user has permission on at least one job, false otherwise.
func (h *Handler) checkScenarioRunGroupAccess(
	ctx context.Context,
	userID string,
	scenarioRun *krknv1alpha1.KrknScenarioRun,
	requiredAction groupauth.Action,
) (bool, error) {
	// Fetch user groups
	userGroups, err := groupauth.GetUserGroups(ctx, h.client, userID, h.namespace)
	if err != nil {
		return false, err
	}

	if len(userGroups) == 0 {
		return false, nil // No groups = no access
	}

	// Check if user has permission on ANY job in the run
	for _, job := range scenarioRun.Status.ClusterJobs {
		if job.ClusterAPIURL == "" {
			continue // Skip jobs without ClusterAPIURL
		}

		if groupauth.CanPerformAction(userGroups, job.ClusterAPIURL, requiredAction) {
			return true, nil // User has access to at least one job
		}
	}

	return false, nil // No permission on any job
}

// filterScenarioRunsByGroupPermission filters scenario runs based on group permissions.
//
// Filtering rules:
// - If no claims in context (e.g., tests), return all runs
// - Admins see all runs
// - Regular users see only runs where they have 'view' permission on AT LEAST ONE job
//
// Parameters:
//   - runs: The list of scenario runs to filter
//   - ctx: The request context containing user claims
//
// Returns a filtered list of scenario runs the user is authorized to see
func (h *Handler) filterScenarioRunsByGroupPermission(
	runs []krknv1alpha1.KrknScenarioRun,
	ctx context.Context,
) []krknv1alpha1.KrknScenarioRun {
	claims := auth.GetClaimsFromContext(ctx)

	// Defensive check - if no claims (e.g., in tests), return all runs unfiltered
	if claims == nil {
		return runs
	}

	// Admins see all runs
	if claims.Role == string(auth.RoleAdmin) {
		return runs
	}

	// Regular users: filter by group permissions on jobs
	filtered := make([]krknv1alpha1.KrknScenarioRun, 0)

	for _, run := range runs {
		// Check if user has 'view' permission on ANY job in this run
		hasAccess, err := h.checkScenarioRunGroupAccess(
			ctx,
			claims.UserID,
			&run,
			groupauth.ActionView,
		)

		if err != nil {
			// Log error but continue processing other runs
			log.FromContext(ctx).V(1).Info("Failed to check access for run, excluding",
				"runName", run.Name,
				"userID", claims.UserID,
				"error", err.Error(),
			)
			continue
		}

		if hasAccess {
			filtered = append(filtered, run)
		}
	}

	return filtered
}

// filterJobsByPermission filters cluster jobs based on user's group permissions.
// Returns only jobs where user has the specified permission on the job's cluster.
//
// Parameters:
//   - jobs: The list of cluster jobs to filter
//   - ctx: The request context containing user claims
//   - userGroups: User's group memberships
//   - requiredAction: The action to check (e.g., ActionView, ActionCancel)
//
// Returns a filtered list of jobs the user is authorized to see
func (h *Handler) filterJobsByPermission(
	jobs []krknv1alpha1.ClusterJobStatus,
	ctx context.Context,
	userGroups []krknv1alpha1.KrknUserGroup,
	requiredAction groupauth.Action,
) []krknv1alpha1.ClusterJobStatus {
	filtered := make([]krknv1alpha1.ClusterJobStatus, 0)

	for _, job := range jobs {
		// Skip jobs without ClusterAPIURL (defensive - shouldn't happen for new jobs)
		if job.ClusterAPIURL == "" {
			log.FromContext(ctx).V(1).Info("Job missing ClusterAPIURL, skipping",
				"jobID", job.JobID,
				"clusterName", job.ClusterName)
			continue
		}

		// Check if user has required permission on this job's cluster
		if groupauth.CanPerformAction(userGroups, job.ClusterAPIURL, requiredAction) {
			filtered = append(filtered, job)
		}
	}

	return filtered
}

// checkJobAccess verifies if the authenticated user has permission to access
// a specific job using group-based permissions.
//
// Parameters:
//   - w: The HTTP response writer
//   - r: The HTTP request containing user claims in context
//   - job: The job to check access for
//   - requiredAction: The action to check (e.g., ActionView, ActionCancel)
//   - actionName: Human-readable action name for error messages
//
// Returns true if access is allowed, false if denied (with error written to response)
func (h *Handler) checkJobAccess(
	w http.ResponseWriter,
	r *http.Request,
	job *krknv1alpha1.ClusterJobStatus,
	requiredAction groupauth.Action,
	actionName string,
) bool {
	ctx := r.Context()
	claims := auth.GetClaimsFromContext(ctx)

	// Defensive check
	if claims == nil {
		http.Error(w, `{"error":"unauthorized","message":"No authentication claims found"}`, http.StatusUnauthorized)
		return false
	}

	// Admins bypass all checks
	if auth.IsAdmin(ctx) {
		return true
	}

	// Check if job has ClusterAPIURL
	if job.ClusterAPIURL == "" {
		http.Error(w, `{"error":"forbidden","message":"Access denied. Job has no cluster API URL"}`, http.StatusForbidden)
		return false
	}

	// Check group-based permissions
	hasAccess, err := groupauth.HasClusterPermission(
		ctx,
		h.client,
		claims.UserID,
		h.namespace,
		job.ClusterAPIURL,
		requiredAction,
	)

	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to check job access",
			"userID", claims.UserID,
			"jobID", job.JobID,
			"action", requiredAction)
		http.Error(w, `{"error":"internal_error","message":"Failed to validate access"}`, http.StatusInternalServerError)
		return false
	}

	if !hasAccess {
		errorMsg := fmt.Sprintf(`{"error":"forbidden","message":"Access denied. You do not have permission to %s this job"}`, actionName)
		http.Error(w, errorMsg, http.StatusForbidden)
		return false
	}

	return true
}
