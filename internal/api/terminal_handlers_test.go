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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func TestExecuteTerminal_ValidationErrors(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	k8sClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	clientset := fake.NewSimpleClientset()

	handler := NewTestHandler(k8sClient, clientset, "test-namespace", "localhost:50051")

	tests := []struct {
		name           string
		body           interface{}
		wantStatusCode int
		wantError      string
	}{
		{
			name:           "invalid JSON",
			body:           "invalid json",
			wantStatusCode: http.StatusBadRequest,
			wantError:      "invalid_request",
		},
		{
			name: "missing cluster_id",
			body: TerminalRequest{
				UUID:    "test-uuid",
				Command: "kubectl get pods",
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "invalid_request",
		},
		{
			name: "missing uuid",
			body: TerminalRequest{
				ClusterID: "test-cluster",
				Command:   "kubectl get pods",
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "invalid_request",
		},
		{
			name: "missing command",
			body: TerminalRequest{
				ClusterID: "test-cluster",
				UUID:      "test-uuid",
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      "invalid_request",
		},
		{
			name: "command not kubectl or oc",
			body: TerminalRequest{
				ClusterID: "test-cluster",
				UUID:      "test-uuid",
				Command:   "ls -al",
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      "not_found",
		},
		{
			name: "command not kubectl or oc - bash",
			body: TerminalRequest{
				ClusterID: "test-cluster",
				UUID:      "test-uuid",
				Command:   "bash -c 'echo test'",
			},
			wantStatusCode: http.StatusNotFound,
			wantError:      "not_found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error
			if strBody, ok := tt.body.(string); ok {
				body = []byte(strBody)
			} else {
				body, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/terminal", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ExecuteTerminal(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("ExecuteTerminal() status = %v, want %v", w.Code, tt.wantStatusCode)
			}

			var errResp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}

			if errResp.Error != tt.wantError {
				t.Errorf("ExecuteTerminal() error = %v, want %v", errResp.Error, tt.wantError)
			}
		})
	}
}

func TestExecuteTerminal_CommandValidation(t *testing.T) {
	// Test command validation logic using the terminal package directly
	// These tests verify that validation catches forbidden commands
	// Integration tests with full handler require mocking getKubeconfig

	t.Skip("Requires mocking getKubeconfig - validation logic tested in pkg/terminal/whitelist_test.go")

	// Note: The validation logic is thoroughly tested in pkg/terminal/whitelist_test.go
	// These test cases verify:
	// - Blocked subcommands (delete, apply, create, etc.)
	// - Blocked flags (--watch, --follow, -w, -f)
	// - Allowed subcommands (get, describe, logs, etc.)
	//
	// To test the full ExecuteTerminal handler flow with validation:
	// 1. Mock getKubeconfig to return valid kubeconfig
	// 2. Mock gRPC ExecuteKubectl call
	// 3. Verify HTTP status codes match expected errors
}

func TestExecuteTerminal_ExitCodeHandling(t *testing.T) {
	// Test that when kubectl returns exit code > 0, we get 400 with stdout/stderr in body
	// This requires mocking the gRPC call
	t.Skip("Requires gRPC mock - testing behavior is documented in TERMINAL_API.md")

	// Expected behavior:
	// - Request: kubectl get pod nonexistent-pod
	// - gRPC returns: exit_code=1, stderr="Error: pods 'nonexistent-pod' not found"
	// - HTTP response: 400 Bad Request with TerminalResponse body containing stderr
}

func TestExecuteTerminal_HelpCommand(t *testing.T) {
	// Test that bare "kubectl" and "oc" commands bypass validation and return help
	t.Skip("Requires gRPC mock - testing behavior is documented")

	// Expected behavior:
	// - Request: "kubectl" (no subcommand)
	// - Should bypass validation
	// - Should send to gRPC with empty subcommand
	// - Should return help output in stdout
}

// TestExecuteTerminal_PermissionCheck documents permission enforcement behavior
func TestExecuteTerminal_PermissionCheck(t *testing.T) {
	t.Skip("Requires mocking getClusterAPIURL and HasClusterPermission")

	// Expected behavior:
	// 1. Admin users bypass permission checks
	// 2. Regular users must have 'run' permission on cluster
	// 3. Users without 'run' permission get 403 Forbidden
	// 4. Permission check happens BEFORE command validation
	// 5. Permission check happens BEFORE kubeconfig retrieval
	//
	// This ensures:
	// - Users can only execute terminal commands on clusters they can run scenarios on
	// - Consistent with group-based access control model
	// - Early rejection prevents wasted processing
}

// TestExecuteTerminal_IntegrationCases documents integration test cases
// These require a real gRPC server and are typically run in e2e tests
func TestExecuteTerminal_IntegrationCases(t *testing.T) {
	t.Skip("Integration tests require real gRPC server")

	// Integration test cases to cover:
	// 1. Valid kubectl get pods command returns 200 with stdout
	// 2. kubectl get nonexistent-pod returns 400 with error in stderr
	// 3. Invalid kubeconfig returns 404
	// 4. Command timeout returns 408
	// 5. Short flags (-n namespace) work correctly
	// 6. Long flags (--namespace=default) work correctly
	// 7. Mixed flags work correctly
	// 8. User with run permission can execute commands (200 OK)
	// 9. User without run permission gets 403 Forbidden
	// 10. Admin can execute commands on any cluster
}
