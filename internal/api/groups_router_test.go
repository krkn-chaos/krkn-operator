package api

import (
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
