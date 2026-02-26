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
	"net/http"

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
