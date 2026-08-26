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
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	"k8s.io/client-go/kubernetes/fake"
)

// TestSanitizeUserIDForLabel tests the email sanitization for Kubernetes labels
// via the consolidated groupauth.SanitizeUserIDForLabel function.
func TestSanitizeUserIDForLabel(t *testing.T) {
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
			result, err := groupauth.SanitizeUserIDForLabel(tt.email)
			if err != nil {
				t.Fatalf("SanitizeUserIDForLabel(%s) unexpected error: %v", tt.email, err)
			}
			if result != tt.expected {
				t.Errorf("SanitizeUserIDForLabel(%s) = %s, want %s", tt.email, result, tt.expected)
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
			UserID:            "user@example.com",
			Name:              "Test",
			Surname:           "User",
			Role:              "user",
			PasswordSecretRef: "user-password",
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
				Status: krknv1alpha1.KrknScenarioRunStatus{
					ClusterJobs: []krknv1alpha1.ClusterJobStatus{
						{
							ClusterName:   "cluster1",
							ClusterAPIURL: "https://cluster1.example.com:6443",
							JobID:         "job-1",
						},
					},
				},
			},
			expectAllow: true,
		},
		{
			name:   "run without jobs is rejected (admin bypasses)",
			claims: adminClaims,
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				Status: krknv1alpha1.KrknScenarioRunStatus{
					ClusterJobs: []krknv1alpha1.ClusterJobStatus{},
				},
			},
			expectAllow: true, // Admin bypasses this check
		},
		{
			name:   "user with group permission can access run",
			claims: userClaims,
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				Status: krknv1alpha1.KrknScenarioRunStatus{
					ClusterJobs: []krknv1alpha1.ClusterJobStatus{
						{
							ClusterName:   "cluster1",
							ClusterAPIURL: "https://cluster1.example.com:6443",
							JobID:         "job-1",
						},
					},
				},
			},
			expectAllow: true,
		},
		{
			name:   "user without permission cannot access run",
			claims: userClaims,
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				Status: krknv1alpha1.KrknScenarioRunStatus{
					ClusterJobs: []krknv1alpha1.ClusterJobStatus{
						{
							ClusterName:   "cluster2",
							ClusterAPIURL: "https://cluster2.example.com:6443",
							JobID:         "job-2",
						},
					},
				},
			},
			expectAllow:    false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "run without jobs is rejected (user)",
			claims: userClaims,
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				Status: krknv1alpha1.KrknScenarioRunStatus{
					ClusterJobs: []krknv1alpha1.ClusterJobStatus{},
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
			UserID:            "user@example.com",
			Name:              "Test",
			Surname:           "User",
			Role:              "user",
			PasswordSecretRef: "user-password",
		},
	}

	runs := []krknv1alpha1.KrknScenarioRun{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run1-cluster1"},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				ClusterJobs: []krknv1alpha1.ClusterJobStatus{
					{
						ClusterName:   "cluster1",
						ClusterAPIURL: "https://cluster1.example.com:6443",
						JobID:         "job-1",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run2-cluster2"},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				ClusterJobs: []krknv1alpha1.ClusterJobStatus{
					{
						ClusterName:   "cluster2",
						ClusterAPIURL: "https://cluster2.example.com:6443",
						JobID:         "job-2",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run3-legacy-no-jobs"},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				ClusterJobs: []krknv1alpha1.ClusterJobStatus{},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run4-both-clusters"},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				ClusterJobs: []krknv1alpha1.ClusterJobStatus{
					{
						ClusterName:   "cluster1",
						ClusterAPIURL: "https://cluster1.example.com:6443",
						JobID:         "job-3",
					},
					{
						ClusterName:   "cluster2",
						ClusterAPIURL: "https://cluster2.example.com:6443",
						JobID:         "job-4",
					},
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
			expectedNames: []string{"run1-cluster1", "run2-cluster2", "run3-legacy-no-jobs", "run4-both-clusters"},
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
			filtered := handler.FilterScenarioRunsByGroupPermission(runs, ctx)

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

	// Note: ClusterAPIURL is now populated by the controller in job status,
	// not by the API handler in spec, so we don't verify it here
}

// TestPostScenarioRun_ClusterNameCollision_Returns500 verifies the end-to-end HTTP
// mapping of a cluster name collision: when two providers register the same cluster
// name with different API URLs in the target request status, PostScenarioRun must
// return 500 (a server-side data-integrity condition), NOT 403, and the client-facing
// body must not leak the internal cluster API URLs.
func TestPostScenarioRun_ClusterNameCollision_Returns500(t *testing.T) {
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	const (
		collidingCluster = "shared-cluster"
		urlA             = "https://cluster-a.internal.example.com:6443"
		urlB             = "https://cluster-b.internal.example.com:6443"
	)

	// Target request whose status registers the SAME cluster name under two
	// different providers with DIFFERENT API URLs -> collision.
	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "collision-target-request",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{
			UUID: "collision-uuid",
		},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status: "Completed",
			TargetData: map[string][]krknv1alpha1.ClusterTarget{
				"krkn-operator": {
					{ClusterName: collidingCluster, ClusterAPIURL: urlA},
				},
				"acm-provider": {
					{ClusterName: collidingCluster, ClusterAPIURL: urlB},
				},
			},
		},
	}

	// A non-admin user that belongs to a group. The group grants 'run' so that the
	// collision (detected before the per-cluster permission check) is unambiguously
	// what produces the failure rather than a missing permission.
	testGroup := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-group",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name:        "test-group",
			Description: "Test group",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				urlA: {Actions: []string{"view", "run", "cancel"}},
				urlB: {Actions: []string{"view", "run", "cancel"}},
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

	// The request references the cluster under a single provider, so it passes the
	// handler's request-level duplicate check; the collision lives in the stored
	// target request status.
	reqBody := `{
		"targetRequestId": "collision-target-request",
		"scenarioImage": "quay.io/krkn/pod-scenarios:latest",
		"scenarioName": "pod-scenario",
		"targetClusters": {
			"krkn-operator": ["shared-cluster"]
		}
	}`

	req := httptest.NewRequest("POST", "/api/v1/scenarios/run", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.PostScenarioRun(w, req)

	// A cluster name collision is a server-side data-integrity condition: expect 500,
	// not 403.
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status 500 for cluster name collision, got %d. Body: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error != "internal_error" {
		t.Errorf("Expected error code 'internal_error', got %q", errResp.Error)
	}

	// The client-facing body must NOT leak the internal cluster API URLs.
	body := w.Body.String()
	for _, leaked := range []string{urlA, urlB, "https://"} {
		if strings.Contains(body, leaked) {
			t.Errorf("Response body must not leak internal cluster details (%q), got: %s", leaked, body)
		}
	}
}

// TestPostScenarioRun_SameClusterSameURL_NoCollision is the counter-proof to the
// collision test: when two providers register the same cluster name with the SAME
// API URL there is no collision, so the request is not rejected with a 500.
func TestPostScenarioRun_SameClusterSameURL_NoCollision(t *testing.T) {
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	const (
		sharedCluster = "shared-cluster"
		sharedURL     = "https://cluster-shared.example.com:6443"
	)

	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shared-target-request",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{UUID: "shared-uuid"},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status: "Completed",
			TargetData: map[string][]krknv1alpha1.ClusterTarget{
				"krkn-operator": {
					{ClusterName: sharedCluster, ClusterAPIURL: sharedURL},
				},
				"acm-provider": {
					{ClusterName: sharedCluster, ClusterAPIURL: sharedURL},
				},
			},
		},
	}

	testGroup := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-group",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: "test-group",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				sharedURL: {Actions: []string{"view", "run", "cancel"}},
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
			UserID: "user@test.com",
			Role:   "user",
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
		"targetRequestId": "shared-target-request",
		"scenarioImage": "quay.io/krkn/pod-scenarios:latest",
		"scenarioName": "pod-scenario",
		"targetClusters": {
			"krkn-operator": ["shared-cluster"]
		}
	}`

	req := httptest.NewRequest("POST", "/api/v1/scenarios/run", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.PostScenarioRun(w, req)

	// No collision: the request must NOT be rejected as a server-side data-integrity
	// error. (It is created successfully since the group grants 'run'.)
	if w.Code == http.StatusInternalServerError {
		t.Fatalf("Did not expect 500 when the same cluster name maps to the same URL. Body: %s", w.Body.String())
	}
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}
}
