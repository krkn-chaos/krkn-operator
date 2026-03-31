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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"k8s.io/client-go/kubernetes/fake"
)

// TestSanitizeUserID tests the email sanitization for Kubernetes labels
func TestSanitizeUserID(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "standard email",
			email:    "user@example.com",
			expected: "user-example-com",
		},
		{
			name:     "email with dots in username",
			email:    "john.doe@company.org",
			expected: "john-doe-company-org",
		},
		{
			name:     "uppercase email",
			email:    "ADMIN@TEST.COM",
			expected: "admin-test-com",
		},
		{
			name:     "complex email",
			email:    "test.user.dev@example.co.uk",
			expected: "test-user-dev-example-co-uk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeUserID(tt.email)
			if result != tt.expected {
				t.Errorf("sanitizeUserID(%s) = %s, want %s", tt.email, result, tt.expected)
			}
		})
	}
}

// TestCheckScenarioRunAccess tests access control for scenario runs
func TestCheckScenarioRunAccess(t *testing.T) {
	tg := auth.NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		TokenDuration,
		"krkn-operator",
	)

	adminToken, _ := tg.GenerateToken("user@example.com", "admin", "Admin", "User", "Org")
	adminClaims, _ := tg.ValidateToken(adminToken)

	userToken, _ := tg.GenerateToken("user@example.com", "user", "Regular", "User", "Org")
	userClaims, _ := tg.ValidateToken(userToken)

	tests := []struct {
		name           string
		claims         *auth.Claims
		scenarioRun    *krknv1alpha1.KrknScenarioRun
		expectAllow    bool
		expectedStatus int
	}{
		{
			name:   "admin can access any scenario run",
			claims: adminClaims,
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				Spec: krknv1alpha1.KrknScenarioRunSpec{
					OwnerUserID: "user@example.com",
				},
			},
			expectAllow: true,
		},
		{
			name:   "run without owner is rejected (admin)",
			claims: adminClaims,
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				Spec: krknv1alpha1.KrknScenarioRunSpec{
					OwnerUserID: "",
				},
			},
			expectAllow:    false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "user can access own scenario run",
			claims: userClaims,
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				Spec: krknv1alpha1.KrknScenarioRunSpec{
					OwnerUserID: "user@example.com",
				},
			},
			expectAllow: true,
		},
		{
			name:   "user cannot access other user's scenario run",
			claims: userClaims,
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				Spec: krknv1alpha1.KrknScenarioRunSpec{
					OwnerUserID: "other@example.com",
				},
			},
			expectAllow:    false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "run without owner is rejected (user)",
			claims: userClaims,
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				Spec: krknv1alpha1.KrknScenarioRunSpec{
					OwnerUserID: "",
				},
			},
			expectAllow:    false,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			ctx := context.WithValue(req.Context(), auth.UserClaimsKey, tt.claims)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			result := checkScenarioRunAccess(w, req, tt.scenarioRun)

			if result != tt.expectAllow {
				t.Errorf("Expected allow=%v, got %v", tt.expectAllow, result)
			}

			if !tt.expectAllow && w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestFilterScenarioRunsByOwnership tests filtering of scenario runs by ownership
func TestFilterScenarioRunsByOwnership(t *testing.T) {
	tg := auth.NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		TokenDuration,
		"krkn-operator",
	)

	adminToken, _ := tg.GenerateToken("user@example.com", "admin", "Admin", "User", "Org")
	adminClaims, _ := tg.ValidateToken(adminToken)

	userToken, _ := tg.GenerateToken("user@example.com", "user", "Regular", "User", "Org")
	userClaims, _ := tg.ValidateToken(userToken)

	runs := []krknv1alpha1.KrknScenarioRun{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run1"},
			Spec: krknv1alpha1.KrknScenarioRunSpec{
				OwnerUserID: "user@example.com",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run2"},
			Spec: krknv1alpha1.KrknScenarioRunSpec{
				OwnerUserID: "other@example.com",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run3-legacy"},
			Spec: krknv1alpha1.KrknScenarioRunSpec{
				OwnerUserID: "",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run4"},
			Spec: krknv1alpha1.KrknScenarioRunSpec{
				OwnerUserID: "another@example.com",
			},
		},
	}

	tests := []struct {
		name          string
		claims        *auth.Claims
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "admin sees all runs (excluding legacy)",
			claims:        adminClaims,
			expectedCount: 3,
			expectedNames: []string{"run1", "run2", "run4"},
		},
		{
			name:          "user sees only own runs",
			claims:        userClaims,
			expectedCount: 1,
			expectedNames: []string{"run1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), auth.UserClaimsKey, tt.claims)
			filtered := filterScenarioRunsByOwnership(runs, ctx)

			if len(filtered) != tt.expectedCount {
				t.Errorf("Expected %d runs, got %d", tt.expectedCount, len(filtered))
			}

			// Verify expected names are present
			nameMap := make(map[string]bool)
			for _, run := range filtered {
				nameMap[run.Name] = true
			}

			for _, expectedName := range tt.expectedNames {
				if !nameMap[expectedName] {
					t.Errorf("Expected run '%s' not found in filtered results", expectedName)
				}
			}
		})
	}
}

// TestPostScenarioRunSetsOwner tests that POST endpoint sets owner correctly
func TestPostScenarioRunSetsOwner(t *testing.T) {
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	// Create a fake target request
	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-target-request",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{
			UUID: "test-uuid-123",
		},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status: "Completed",
			TargetData: map[string][]krknv1alpha1.ClusterTarget{
				"provider1": {
					{
						ClusterName:   "cluster1",
						ClusterAPIURL: "https://api.cluster1.example.com",
					},
				},
			},
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(targetRequest).
		WithStatusSubresource(&krknv1alpha1.KrknScenarioRun{}, &krknv1alpha1.KrknTargetRequest{}).
		Build()

	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	tg := auth.NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		TokenDuration,
		"krkn-operator",
	)

	adminToken, _ := tg.GenerateToken("admin@example.com", "admin", "Test", "Admin", "Org")
	adminClaims, _ := tg.ValidateToken(adminToken)

	requestBody := ScenarioRunRequest{
		TargetRequestID: "test-target-request",
		TargetClusters: map[string][]string{
			"provider1": {"cluster1"},
		},
		ScenarioName:  "test-scenario",
		ScenarioImage: "test-image:latest",
	}

	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/scenarios/run", bytes.NewReader(bodyBytes))
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, adminClaims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.PostScenarioRun(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	// Verify the scenario run was created with correct owner
	var scenarioRunList krknv1alpha1.KrknScenarioRunList
	if err := fakeClient.List(context.Background(), &scenarioRunList); err != nil {
		t.Fatalf("Failed to list scenario runs: %v", err)
	}

	if len(scenarioRunList.Items) != 1 {
		t.Fatalf("Expected 1 scenario run, got %d", len(scenarioRunList.Items))
	}

	scenarioRun := scenarioRunList.Items[0]

	if scenarioRun.Spec.OwnerUserID != "admin@example.com" {
		t.Errorf("Expected OwnerUserID 'admin@example.com', got '%s'", scenarioRun.Spec.OwnerUserID)
	}

	expectedLabel := "admin-example-com"
	if scenarioRun.Labels["krkn.krkn-chaos.dev/owner-user"] != expectedLabel {
		t.Errorf("Expected owner label '%s', got '%s'",
			expectedLabel, scenarioRun.Labels["krkn.krkn-chaos.dev/owner-user"])
	}
}

// TestListScenarioRunsFiltersBy Ownership tests that LIST endpoint filters by ownership
func TestListScenarioRunsFiltersByOwnership(t *testing.T) {
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	runs := []runtime.Object{
		&krknv1alpha1.KrknScenarioRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "run1",
				Namespace: "default",
			},
			Spec: krknv1alpha1.KrknScenarioRunSpec{
				OwnerUserID: "user@example.com",
			},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Running",
			},
		},
		&krknv1alpha1.KrknScenarioRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "run2",
				Namespace: "default",
			},
			Spec: krknv1alpha1.KrknScenarioRunSpec{
				OwnerUserID: "other@example.com",
			},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Succeeded",
			},
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(runs...).
		Build()

	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	tg := auth.NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		TokenDuration,
		"krkn-operator",
	)

	userToken, _ := tg.GenerateToken("user@example.com", "user", "User", "One", "Org")
	userClaims, _ := tg.ValidateToken(userToken)

	adminToken, _ := tg.GenerateToken("user@example.com", "admin", "Admin", "User", "Org")
	adminClaims, _ := tg.ValidateToken(adminToken)

	tests := []struct {
		name          string
		claims        *auth.Claims
		expectedCount int
	}{
		{
			name:          "user sees only own runs",
			claims:        userClaims,
			expectedCount: 1,
		},
		{
			name:          "admin sees all runs",
			claims:        adminClaims,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/scenarios/run", nil)
			ctx := context.WithValue(req.Context(), auth.UserClaimsKey, tt.claims)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.ListScenarioRuns(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
			}

			var response ScenarioRunListResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if len(response.ScenarioRuns) != tt.expectedCount {
				t.Errorf("Expected %d runs, got %d", tt.expectedCount, len(response.ScenarioRuns))
			}
		})
	}
}
