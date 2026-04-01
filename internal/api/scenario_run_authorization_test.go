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
	"sigs.k8s.io/controller-runtime/pkg/client"
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

// TestCheckScenarioRunAccess tests group-based access control for scenario runs
func TestCheckScenarioRunAccess(t *testing.T) {
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

	userToken, _ := tg.GenerateToken("user@example.com", "user", "Regular", "User", "Org")
	userClaims, _ := tg.ValidateToken(userToken)

	// Create test group with permission on cluster1
	testGroup := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-group",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name:        "test-group",
			Description: "Test group",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://cluster1.example.com:6443": {
					Actions: []string{"view", "run", "cancel"},
				},
			},
		},
	}

	// Create test user with group membership
	testUser := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-user-example-com",
			Namespace: "krkn-operator-system",
			Labels: map[string]string{
				"group.krkn.krkn-chaos.dev/test-group": "true",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:             "user@example.com",
			Name:               "Test",
			Surname:            "User",
			Role:               "user",
			PasswordSecretRef:  "user-password",
		},
	}

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
					ClusterAPIURLs: map[string]string{
						"cluster1": "https://cluster1.example.com:6443",
					},
				},
			},
			expectAllow: true,
		},
		{
			name:   "run without cluster API URLs is rejected (admin bypasses)",
			claims: adminClaims,
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				Spec: krknv1alpha1.KrknScenarioRunSpec{
					ClusterAPIURLs: map[string]string{},
				},
			},
			expectAllow: true, // Admin bypasses this check
		},
		{
			name:   "user with group permission can access run",
			claims: userClaims,
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				Spec: krknv1alpha1.KrknScenarioRunSpec{
					ClusterAPIURLs: map[string]string{
						"cluster1": "https://cluster1.example.com:6443",
					},
				},
			},
			expectAllow: true,
		},
		{
			name:   "user without permission cannot access run",
			claims: userClaims,
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				Spec: krknv1alpha1.KrknScenarioRunSpec{
					ClusterAPIURLs: map[string]string{
						"cluster2": "https://cluster2.example.com:6443",
					},
				},
			},
			expectAllow:    false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "run without cluster API URLs is rejected (user)",
			claims: userClaims,
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				Spec: krknv1alpha1.KrknScenarioRunSpec{
					ClusterAPIURLs: map[string]string{},
				},
			},
			expectAllow:    false,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fakeclient.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(testGroup, testUser).
				Build()

			handler := &Handler{
				client:    fakeClient,
				clientset: fake.NewSimpleClientset(),
				namespace: "krkn-operator-system",
			}

			req := httptest.NewRequest("GET", "/test", nil)
			ctx := context.WithValue(req.Context(), auth.UserClaimsKey, tt.claims)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			result := handler.checkScenarioRunAccess(w, req, tt.scenarioRun)

			if result != tt.expectAllow {
				t.Errorf("Expected allow=%v, got %v. Response: %s", tt.expectAllow, result, w.Body.String())
			}

			if !tt.expectAllow && tt.expectedStatus != 0 && w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestFilterScenarioRunsByGroupPermission tests filtering of scenario runs by group permissions
func TestFilterScenarioRunsByGroupPermission(t *testing.T) {
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

	userToken, _ := tg.GenerateToken("user@example.com", "user", "Regular", "User", "Org")
	userClaims, _ := tg.ValidateToken(userToken)

	// Create test group with permission on cluster1
	testGroup := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-group",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name:        "test-group",
			Description: "Test group",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://cluster1.example.com:6443": {
					Actions: []string{"view", "run", "cancel"},
				},
			},
		},
	}

	// Create test user with group membership
	testUser := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-user-example-com",
			Namespace: "krkn-operator-system",
			Labels: map[string]string{
				"group.krkn.krkn-chaos.dev/test-group": "true",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:             "user@example.com",
			Name:               "Test",
			Surname:            "User",
			Role:               "user",
			PasswordSecretRef:  "user-password",
		},
	}

	runs := []krknv1alpha1.KrknScenarioRun{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run1-cluster1"},
			Spec: krknv1alpha1.KrknScenarioRunSpec{
				ClusterAPIURLs: map[string]string{
					"cluster1": "https://cluster1.example.com:6443",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run2-cluster2"},
			Spec: krknv1alpha1.KrknScenarioRunSpec{
				ClusterAPIURLs: map[string]string{
					"cluster2": "https://cluster2.example.com:6443",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run3-legacy-no-urls"},
			Spec: krknv1alpha1.KrknScenarioRunSpec{
				ClusterAPIURLs: map[string]string{},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run4-both-clusters"},
			Spec: krknv1alpha1.KrknScenarioRunSpec{
				ClusterAPIURLs: map[string]string{
					"cluster1": "https://cluster1.example.com:6443",
					"cluster2": "https://cluster2.example.com:6443",
				},
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
			name:          "admin sees all runs",
			claims:        adminClaims,
			expectedCount: 4,
			expectedNames: []string{"run1-cluster1", "run2-cluster2", "run3-legacy-no-urls", "run4-both-clusters"},
		},
		{
			name:          "user sees only runs with group permission on at least one cluster",
			claims:        userClaims,
			expectedCount: 2,
			expectedNames: []string{"run1-cluster1", "run4-both-clusters"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fakeclient.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(testGroup, testUser).
				Build()

			handler := &Handler{
				client:    fakeClient,
				clientset: fake.NewSimpleClientset(),
				namespace: "krkn-operator-system",
			}

			ctx := context.WithValue(context.Background(), auth.UserClaimsKey, tt.claims)
			filtered := handler.filterScenarioRunsByGroupPermission(runs, ctx)

			if len(filtered) != tt.expectedCount {
				t.Errorf("Expected %d runs, got %d", tt.expectedCount, len(filtered))
				for _, run := range filtered {
					t.Logf("  - %s", run.Name)
				}
			}

			// Verify expected names are present
			nameMap := make(map[string]bool)
			for _, run := range filtered {
				nameMap[run.Name] = true
			}

			for _, expectedName := range tt.expectedNames {
				if !nameMap[expectedName] {
					t.Errorf("Expected run %s not found in filtered results", expectedName)
				}
			}
		})
	}
}

// TestPostScenarioRunSetsOwner verifies that PostScenarioRun sets the owner user ID
// and populates ClusterAPIURLs correctly
func TestPostScenarioRunSetsOwner(t *testing.T) {
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	// Create a fake target request
	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-target-request",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{
			UUID: "test-uuid",
		},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status: "Completed",
			TargetData: map[string][]krknv1alpha1.ClusterTarget{
				"krkn-operator": {
					{
						ClusterName:   "cluster-1",
						ClusterAPIURL: "https://cluster1.example.com:6443",
					},
				},
			},
		},
	}

	// Create test group and user
	testGroup := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-group",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name:        "test-group",
			Description: "Test group",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://cluster1.example.com:6443": {
					Actions: []string{"view", "run", "cancel"},
				},
			},
		},
	}

	testUser := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-user-test-com",
			Namespace: "krkn-operator-system",
			Labels: map[string]string{
				"group.krkn.krkn-chaos.dev/test-group": "true",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:            "user@test.com",
			Name:              "Test",
			Surname:           "User",
			Role:              "user",
			PasswordSecretRef: "user-password",
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(targetRequest, testGroup, testUser).
		Build()

	handler := &Handler{
		client:    fakeClient,
		clientset: fake.NewSimpleClientset(),
		namespace: "krkn-operator-system",
	}

	tg := auth.NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		TokenDuration,
		"krkn-operator",
	)
	token, _ := tg.GenerateToken("user@test.com", "user", "Test", "User", "Org")
	claims, _ := tg.ValidateToken(token)

	reqBody := `{
		"targetRequestId": "test-target-request",
		"scenarioImage": "quay.io/krkn/pod-scenarios:latest",
		"scenarioName": "pod-scenario",
		"targetClusters": {
			"krkn-operator": ["cluster-1"]
		}
	}`

	req := httptest.NewRequest("POST", "/api/v1/scenarios/run", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.PostScenarioRun(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response ScenarioRunCreateResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.OwnerUserID != "user@test.com" {
		t.Errorf("Expected OwnerUserID to be 'user@test.com', got '%s'", response.OwnerUserID)
	}

	// Verify the created ScenarioRun has ClusterAPIURLs populated
	scenarioRun := &krknv1alpha1.KrknScenarioRun{}
	scenarioRunKey := client.ObjectKey{
		Name:      response.ScenarioRunName,
		Namespace: "krkn-operator-system",
	}
	if err := fakeClient.Get(ctx, scenarioRunKey, scenarioRun); err != nil {
		t.Fatalf("Failed to get created ScenarioRun: %v", err)
	}

	if len(scenarioRun.Spec.ClusterAPIURLs) == 0 {
		t.Error("Expected ClusterAPIURLs to be populated, got empty map")
	}

	expectedURL := "https://cluster1.example.com:6443"
	if scenarioRun.Spec.ClusterAPIURLs["cluster-1"] != expectedURL {
		t.Errorf("Expected ClusterAPIURLs['cluster-1'] to be '%s', got '%s'",
			expectedURL, scenarioRun.Spec.ClusterAPIURLs["cluster-1"])
	}
}
