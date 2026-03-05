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
	"strings"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
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
// the given scenario run. Returns true if access is allowed, false otherwise.
//
// Access rules:
// - Admins can access all scenario runs
// - Regular users can only access runs where OwnerUserID matches their UserID
// - Legacy runs (no OwnerUserID set) are admin-only
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
func checkScenarioRunAccess(w http.ResponseWriter, r *http.Request, scenarioRun *krknv1alpha1.KrknScenarioRun) bool {
	ctx := r.Context()
	claims := auth.GetClaimsFromContext(ctx)

	// Defensive check - should never happen with RequireAuth middleware
	if claims == nil {
		http.Error(w, `{"error":"unauthorized","message":"No authentication claims found"}`, http.StatusUnauthorized)
		return false
	}

	// Admins can access all runs
	if auth.IsAdmin(ctx) {
		return true
	}

	// Legacy runs (no owner) are admin-only
	if scenarioRun.Spec.OwnerUserID == "" {
		http.Error(w, `{"error":"forbidden","message":"Access denied. This scenario run has no owner and can only be accessed by administrators"}`, http.StatusForbidden)
		return false
	}

	// Regular users can only access their own runs
	if scenarioRun.Spec.OwnerUserID != claims.UserID {
		http.Error(w, `{"error":"forbidden","message":"Access denied. You can only access your own scenario runs"}`, http.StatusForbidden)
		return false
	}

	return true
}

// filterScenarioRunsByOwnership filters a list of scenario runs based on user permissions.
//
// Filtering rules:
// - If no claims in context (e.g., tests), return all runs (no filtering)
// - Admins see all runs
// - Regular users see only runs where OwnerUserID matches their UserID
//
// Parameters:
//   - runs: The list of scenario runs to filter
//   - ctx: The request context containing user claims
//
// Returns a filtered list of scenario runs the user is authorized to see
func filterScenarioRunsByOwnership(runs []krknv1alpha1.KrknScenarioRun, ctx context.Context) []krknv1alpha1.KrknScenarioRun {
	claims := auth.GetClaimsFromContext(ctx)

	// Defensive check - if no claims (e.g., in tests), return all runs unfiltered
	if claims == nil {
		return runs
	}

	// Admins see all runs
	if claims.Role == string(auth.RoleAdmin) {
		return runs
	}

	// Regular users see only their own runs
	filtered := make([]krknv1alpha1.KrknScenarioRun, 0)
	for _, run := range runs {
		if run.Spec.OwnerUserID == claims.UserID {
			filtered = append(filtered, run)
		}
	}

	return filtered
}
