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
	"net/http/httptest"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

// TestWorkflowsAvailableMethodGuard tests that the /workflows/available route
// rejects non-GET methods at the routing layer (server.go)
func TestWorkflowsAvailableMethodGuard(t *testing.T) {
	// Setup fake Kubernetes client
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		Build()

	// Create JWT secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-jwt-secret",
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			"jwt-secret": []byte("test-secret-key-for-testing-only-min-32-chars"),
		},
	}
	if err := fakeClient.Create(context.Background(), secret); err != nil {
		t.Fatalf("Failed to create JWT secret: %v", err)
	}

	// Create secret manager
	secretManager := auth.NewSecretManager(fakeClient, "test-namespace", 24*time.Hour, "krkn-test")

	// Initialize secret manager (loads JWT secret)
	ctx := context.Background()
	if err := secretManager.Start(ctx); err != nil {
		t.Fatalf("Failed to start secret manager: %v", err)
	}

	// Create real server instance
	server := NewServer(8080, fakeClient, nil, "test-namespace", "localhost:50051", secretManager)

	// Get token generator for creating test tokens
	tokenGen, err := secretManager.GetTokenGenerator()
	if err != nil {
		t.Fatalf("Failed to get token generator: %v", err)
	}

	// Test non-GET methods on /workflows/available
	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run("reject_"+method, func(t *testing.T) {
			req := httptest.NewRequest(method, WorkflowsAvailablePath, nil)

			// Add valid auth token (method guard should trigger before auth check)
			token, err := tokenGen.GenerateToken("test@example.com", "admin", "Test", "User", "TestOrg")
			if err != nil {
				t.Fatalf("Failed to generate token: %v", err)
			}
			req.Header.Set("Authorization", "Bearer "+token)

			rr := httptest.NewRecorder()

			// Call the real server HTTP handler
			server.server.Handler.ServeHTTP(rr, req)

			// Should return 405 Method Not Allowed (not 200, 401, or any other code)
			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d. Body: %s",
					http.StatusMethodNotAllowed, method, rr.Code, rr.Body.String())
			}
		})
	}

	// Verify GET still works
	t.Run("allow_GET", func(t *testing.T) {
		// Create test user
		userName := "krknuser-" + sanitizeUserID("test@example.com")
		user := &krknv1alpha1.KrknUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      userName,
				Namespace: "test-namespace",
			},
			Spec: krknv1alpha1.KrknUserSpec{
				UserID: "test@example.com",
			},
		}
		if err := fakeClient.Create(context.Background(), user); err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, WorkflowsAvailablePath, nil)

		token, err := tokenGen.GenerateToken("test@example.com", "admin", "Test", "User", "TestOrg")
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		server.server.Handler.ServeHTTP(rr, req)

		// Should return 200 OK (or at least not 405)
		if rr.Code == http.StatusMethodNotAllowed {
			t.Errorf("GET should be allowed, got %d", rr.Code)
		}
	})
}
