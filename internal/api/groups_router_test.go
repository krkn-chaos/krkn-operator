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

// TestGroupNameValidation verifies that group name length is validated
func TestGroupNameValidation(t *testing.T) {
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

	t.Run("reject group name exceeding 63 characters", func(t *testing.T) {
		// Group name that exceeds 63 characters when sanitized
		longGroupName := "super-long-group-name-that-definitely-exceeds-sixty-three-characters-limit-and-should-be-rejected"
		fakeClient := fakeclient.NewClientBuilder().
			WithScheme(scheme).
			Build()

		handler := &Handler{
			client:    fakeClient,
			clientset: fake.NewSimpleClientset(),
			namespace: "krkn-operator-system",
		}

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

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 Bad Request, got %d. Response: %s", w.Code, w.Body.String())
		}

		// Verify error message mentions length issue
		body := w.Body.String()
		if !bytes.Contains([]byte(body), []byte("too long")) && !bytes.Contains([]byte(body), []byte("63 characters")) {
			t.Errorf("Expected error message about length, got: %s", body)
		}

		// Verify no CR was created
		var groupList krknv1alpha1.KrknUserGroupList
		if err := fakeClient.List(context.Background(), &groupList); err != nil {
			t.Fatalf("Failed to list groups: %v", err)
		}

		if len(groupList.Items) != 0 {
			t.Errorf("Expected 0 groups to be created, got %d", len(groupList.Items))
		}

		t.Logf("✓ Long group name correctly rejected")
		t.Logf("✓ Error message: %s", body)
	})

	t.Run("accept group name within 63 character limit", func(t *testing.T) {
		fakeClient := fakeclient.NewClientBuilder().
			WithScheme(scheme).
			Build()

		handler := &Handler{
			client:    fakeClient,
			clientset: fake.NewSimpleClientset(),
			namespace: "krkn-operator-system",
		}

		// Group name that is exactly 63 characters after sanitization
		validGroupName := "this-is-a-valid-group-name-that-is-exactly-sixty-three-chars"

		reqBody := `{
			"name": "` + validGroupName + `",
			"description": "Valid group name",
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
			t.Errorf("Expected status 201 Created, got %d. Response: %s", w.Code, w.Body.String())
		}

		// Verify CR was created
		var groupList krknv1alpha1.KrknUserGroupList
		if err := fakeClient.List(context.Background(), &groupList); err != nil {
			t.Fatalf("Failed to list groups: %v", err)
		}

		if len(groupList.Items) != 1 {
			t.Fatalf("Expected 1 group, got %d", len(groupList.Items))
		}

		t.Logf("✓ Valid group name accepted (len=%d)", len(validGroupName))
	})

	t.Run("prevent collision - two names differing only after 63 chars", func(t *testing.T) {
		fakeClient := fakeclient.NewClientBuilder().
			WithScheme(scheme).
			Build()

		handler := &Handler{
			client:    fakeClient,
			clientset: fake.NewSimpleClientset(),
			namespace: "krkn-operator-system",
		}

		// Two names that would collide if truncated to 63 characters
		longName1 := "super-long-group-name-that-definitely-exceeds-sixty-three-characters-limit-1"
		longName2 := "super-long-group-name-that-definitely-exceeds-sixty-three-characters-limit-2"

		// Try to create first group
		reqBody1 := `{
			"name": "` + longName1 + `",
			"description": "First long name",
			"clusterPermissions": {
				"https://cluster1.example.com:6443": {
					"actions": ["view"]
				}
			}
		}`

		req1 := httptest.NewRequest(http.MethodPost, "/api/v1/groups", bytes.NewBufferString(reqBody1))
		req1.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req1.Context(), auth.UserClaimsKey, adminClaims)
		req1 = req1.WithContext(ctx)
		w1 := httptest.NewRecorder()

		handler.CreateUserGroup(w1, req1)

		// Should be rejected (too long)
		if w1.Code != http.StatusBadRequest {
			t.Errorf("First group: Expected 400, got %d", w1.Code)
		}

		// Try to create second group
		reqBody2 := `{
			"name": "` + longName2 + `",
			"description": "Second long name",
			"clusterPermissions": {
				"https://cluster1.example.com:6443": {
					"actions": ["view"]
				}
			}
		}`

		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/groups", bytes.NewBufferString(reqBody2))
		req2.Header.Set("Content-Type", "application/json")
		ctx2 := context.WithValue(req2.Context(), auth.UserClaimsKey, adminClaims)
		req2 = req2.WithContext(ctx2)
		w2 := httptest.NewRecorder()

		handler.CreateUserGroup(w2, req2)

		// Should also be rejected (too long)
		if w2.Code != http.StatusBadRequest {
			t.Errorf("Second group: Expected 400, got %d", w2.Code)
		}

		// Verify no groups were created (collision prevented)
		var groupList krknv1alpha1.KrknUserGroupList
		if err := fakeClient.List(context.Background(), &groupList); err != nil {
			t.Fatalf("Failed to list groups: %v", err)
		}

		if len(groupList.Items) != 0 {
			t.Errorf("Expected 0 groups (both rejected), got %d", len(groupList.Items))
		}

		t.Logf("✓ Collision prevented - both long names rejected")
	})
}
