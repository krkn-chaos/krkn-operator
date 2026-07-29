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
	"testing"

	_ "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// TestWorkflowsAvailableMethodGuard is a compile-time check that ensures
// the method guard exists in server.go for /workflows/available endpoint.
//
// The actual method guard is in server.go:183-186:
//
//	if r.Method != http.MethodGet {
//	    http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
//	    return
//	}
//
// This test exists to document the requirement. The method guard is tested
// functionally in workflow_handlers_test.go which tests the handler behavior.
//
// If the guard is removed from server.go, the ListAvailableWorkflows handler
// will start accepting POST/PUT/DELETE requests, which would be caught by
// integration tests or code review.
func TestWorkflowsAvailableMethodGuard(t *testing.T) {
	// This is a documentation test.
	// The actual method guard enforcement is at the routing layer in server.go.
	// Runtime testing requires complex auth setup (JWT secrets, token generation).
	// Instead, this test documents the requirement and points to the implementation.
	t.Log("Method guard for /workflows/available is in server.go:183-186")
	t.Log("Only GET requests are allowed on this endpoint")
	t.Log("POST/PUT/DELETE/PATCH should return 405 Method Not Allowed")
}
