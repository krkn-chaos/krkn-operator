package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

// TestGroupsRouterEdgeCases tests edge cases in the groups router
func TestGroupsRouterEdgeCases(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Create admin token for authentication
	tg := auth.NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		TokenDuration,
		"krkn-operator",
	)
	adminToken, _ := tg.GenerateToken("admin@example.com", "admin", "Admin", "User", "Org")
	adminClaims, _ := tg.ValidateToken(adminToken)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		description    string
	}{
		{
			name:           "root path with trailing slash",
			method:         http.MethodGet,
			path:           "/api/v1/groups/",
			expectedStatus: http.StatusOK,
			description:    "Should accept /api/v1/groups/ (with trailing slash)",
		},
		{
			name:           "root path without trailing slash",
			method:         http.MethodGet,
			path:           "/api/v1/groups",
			expectedStatus: http.StatusOK,
			description:    "Should accept /api/v1/groups (without trailing slash)",
		},
		{
			name:           "group name containing 'members'",
			method:         http.MethodGet,
			path:           "/api/v1/groups/members-team",
			expectedStatus: http.StatusNotFound, // Group doesn't exist, but shouldn't be treated as members endpoint
			description:    "Group name 'members-team' should not be treated as members endpoint",
		},
		{
			name:           "group name starting with 'members'",
			method:         http.MethodGet,
			path:           "/api/v1/groups/members",
			expectedStatus: http.StatusNotFound, // Group doesn't exist
			description:    "Group name 'members' should not be confused with members endpoint",
		},
		{
			name:           "group name ending with 'members'",
			method:         http.MethodGet,
			path:           "/api/v1/groups/dev-members",
			expectedStatus: http.StatusNotFound,
			description:    "Group name 'dev-members' should not be treated as members endpoint",
		},
		{
			name:           "actual members endpoint",
			method:         http.MethodGet,
			path:           "/api/v1/groups/test-group/members",
			expectedStatus: http.StatusNotFound, // Group doesn't exist, but route should recognize it
			description:    "Actual members endpoint should be recognized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fakeclient.NewClientBuilder().
				WithScheme(scheme).
				Build()

			handler := &Handler{
				client:    fakeClient,
				clientset: fake.NewSimpleClientset(),
				namespace: "krkn-operator-system",
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			ctx := context.WithValue(req.Context(), auth.UserClaimsKey, adminClaims)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			handler.GroupsRouter(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s: Expected status %d, got %d. Response: %s",
					tt.description, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestGroupNameTruncation verifies that group names >63 characters work correctly
func TestGroupNameTruncation(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tg := auth.NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		TokenDuration,
		"krkn-operator",
	)
	adminToken, _ := tg.GenerateToken("admin@example.com", "admin", "Admin", "User", "Org")
	adminClaims, _ := tg.ValidateToken(adminToken)

	// Test with a group name that exceeds 63 characters when sanitized
	longGroupName := "super-long-group-name-that-definitely-exceeds-sixty-three-characters-limit-and-should-be-truncated"

	t.Run("create group with long name", func(t *testing.T) {
		fakeClient := fakeclient.NewClientBuilder().
			WithScheme(scheme).
			Build()

		handler := &Handler{
			client:    fakeClient,
			clientset: fake.NewSimpleClientset(),
			namespace: "krkn-operator-system",
		}

		// Create group
		reqBody := `{
			"name": "` + longGroupName + `",
			"description": "Test group with long name",
			"clusterPermissions": {
				"https://cluster1.example.com:6443": {
					"actions": ["view"]
				}
			}
		}`

		req := httptest.NewRequest(http.MethodPost, "/api/v1/groups", bytes.NewBufferString(reqBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserClaimsKey, adminClaims)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.CreateUserGroup(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d. Response: %s", w.Code, w.Body.String())
		}

		// Verify CR was created with sanitized name (truncated to 63 chars)
		var groupList krknv1alpha1.KrknUserGroupList
		if err := fakeClient.List(context.Background(), &groupList); err != nil {
			t.Fatalf("Failed to list groups: %v", err)
		}

		if len(groupList.Items) != 1 {
			t.Fatalf("Expected 1 group, got %d", len(groupList.Items))
		}

		createdGroup := groupList.Items[0]

		// CR name should be truncated to 63 characters
		if len(createdGroup.Name) > 63 {
			t.Errorf("CR name exceeds 63 characters: %d (name: %s)", len(createdGroup.Name), createdGroup.Name)
		}

		// CR name should match what SanitizeGroupName produces
		expectedCRName := "super-long-group-name-that-definitely-exceeds-sixty-three-chara"
		if createdGroup.Name != expectedCRName {
			t.Errorf("Expected CR name to be '%s', got '%s'", expectedCRName, createdGroup.Name)
		}

		if len(createdGroup.Name) != 63 {
			t.Errorf("Expected CR name to be exactly 63 chars, got %d", len(createdGroup.Name))
		}

		// Verify spec.name preserves original name
		if createdGroup.Spec.Name != longGroupName {
			t.Errorf("Expected spec.name to preserve original name '%s', got '%s'", longGroupName, createdGroup.Spec.Name)
		}

		t.Logf("✓ CR name (truncated): %s (len=%d)", createdGroup.Name, len(createdGroup.Name))
		t.Logf("✓ Spec name (original): %s", createdGroup.Spec.Name)
	})
}
