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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

// TestClusterSegregationMultipleUsers verifies that users can only see runs
// from clusters where they have view permission through their groups.
// This is a comprehensive test for the cluster segregation feature.
func TestClusterSegregationMultipleUsers(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tg := auth.NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		TokenDuration,
		"krkn-operator",
	)

	// Create admin
	adminToken, _ := tg.GenerateToken("admin@example.com", "admin", "Admin", "User", "Org")
	adminClaims, _ := tg.ValidateToken(adminToken)

	// Create User A with access to cluster1 only
	userAToken, _ := tg.GenerateToken("usera@example.com", "user", "User", "A", "Org")
	userAClaims, _ := tg.ValidateToken(userAToken)

	// Create User B with access to cluster2 only
	userBToken, _ := tg.GenerateToken("userb@example.com", "user", "User", "B", "Org")
	userBClaims, _ := tg.ValidateToken(userBToken)

	// Create Group A with permissions on cluster1
	groupA := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "group-a",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name:        "group-a",
			Description: "Group A with access to cluster1",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://cluster1.example.com:6443": {
					Actions: []string{"view", "run", "cancel"},
				},
			},
		},
	}

	// Create Group B with permissions on cluster2
	groupB := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "group-b",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name:        "group-b",
			Description: "Group B with access to cluster2",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://cluster2.example.com:6443": {
					Actions: []string{"view", "run", "cancel"},
				},
			},
		},
	}

	// Create User A with membership in Group A
	userA := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-usera-example-com",
			Namespace: "krkn-operator-system",
			Labels: map[string]string{
				"group.krkn.krkn-chaos.dev/group-a": "true",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:            "usera@example.com",
			Name:              "User",
			Surname:           "A",
			Role:              "user",
			PasswordSecretRef: "usera-password",
		},
	}

	// Create User B with membership in Group B
	userB := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-userb-example-com",
			Namespace: "krkn-operator-system",
			Labels: map[string]string{
				"group.krkn.krkn-chaos.dev/group-b": "true",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:            "userb@example.com",
			Name:              "User",
			Surname:           "B",
			Role:              "user",
			PasswordSecretRef: "userb-password",
		},
	}

	// Create scenario runs on different clusters
	runs := []krknv1alpha1.KrknScenarioRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "run-cluster1-only",
				Namespace: "krkn-operator-system",
			},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Running",
				ClusterJobs: []krknv1alpha1.ClusterJobStatus{
					{
						ClusterName:   "cluster1",
						ClusterAPIURL: "https://cluster1.example.com:6443",
						JobID:         "job-1-cluster1",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "run-cluster2-only",
				Namespace: "krkn-operator-system",
			},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Running",
				ClusterJobs: []krknv1alpha1.ClusterJobStatus{
					{
						ClusterName:   "cluster2",
						ClusterAPIURL: "https://cluster2.example.com:6443",
						JobID:         "job-2-cluster2",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "run-both-clusters",
				Namespace: "krkn-operator-system",
			},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Running",
				ClusterJobs: []krknv1alpha1.ClusterJobStatus{
					{
						ClusterName:   "cluster1",
						ClusterAPIURL: "https://cluster1.example.com:6443",
						JobID:         "job-3-cluster1",
					},
					{
						ClusterName:   "cluster2",
						ClusterAPIURL: "https://cluster2.example.com:6443",
						JobID:         "job-4-cluster2",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "run-cluster3-unauthorized",
				Namespace: "krkn-operator-system",
			},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Running",
				ClusterJobs: []krknv1alpha1.ClusterJobStatus{
					{
						ClusterName:   "cluster3",
						ClusterAPIURL: "https://cluster3.example.com:6443",
						JobID:         "job-5-cluster3",
					},
				},
			},
		},
	}

	tests := []struct {
		name          string
		claims        *auth.Claims
		expectedCount int
		expectedRuns  []string // Names of runs the user should see
		forbiddenRuns []string // Names of runs the user should NOT see
	}{
		{
			name:          "admin sees all runs",
			claims:        adminClaims,
			expectedCount: 4,
			expectedRuns:  []string{"run-cluster1-only", "run-cluster2-only", "run-both-clusters", "run-cluster3-unauthorized"},
			forbiddenRuns: []string{},
		},
		{
			name:          "user A sees only cluster1 runs",
			claims:        userAClaims,
			expectedCount: 2,
			expectedRuns:  []string{"run-cluster1-only", "run-both-clusters"},
			forbiddenRuns: []string{"run-cluster2-only", "run-cluster3-unauthorized"},
		},
		{
			name:          "user B sees only cluster2 runs",
			claims:        userBClaims,
			expectedCount: 2,
			expectedRuns:  []string{"run-cluster2-only", "run-both-clusters"},
			forbiddenRuns: []string{"run-cluster1-only", "run-cluster3-unauthorized"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fakeclient.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(groupA, groupB, userA, userB).
				Build()

			handler := &Handler{
				client:    fakeClient,
				clientset: fake.NewSimpleClientset(),
				namespace: "krkn-operator-system",
			}

			ctx := context.WithValue(context.Background(), auth.UserClaimsKey, tt.claims)
			filtered := handler.filterScenarioRunsByGroupPermission(runs, ctx)

			// Check count
			if len(filtered) != tt.expectedCount {
				t.Errorf("Expected %d runs, got %d", tt.expectedCount, len(filtered))
				t.Logf("Filtered runs:")
				for _, run := range filtered {
					t.Logf("  - %s", run.Name)
				}
			}

			// Build map of filtered run names
			filteredNames := make(map[string]bool)
			for _, run := range filtered {
				filteredNames[run.Name] = true
			}

			// Verify expected runs are present
			for _, expectedRun := range tt.expectedRuns {
				if !filteredNames[expectedRun] {
					t.Errorf("Expected to see run '%s' but it was not in filtered results", expectedRun)
				}
			}

			// Verify forbidden runs are NOT present
			for _, forbiddenRun := range tt.forbiddenRuns {
				if filteredNames[forbiddenRun] {
					t.Errorf("Should NOT see run '%s' but it was in filtered results", forbiddenRun)
				}
			}
		})
	}
}

// TestScenarioRunCancelPermissions verifies the cancel/delete permission logic
func TestScenarioRunCancelPermissions(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tg := auth.NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		TokenDuration,
		"krkn-operator",
	)

	// Create admin
	adminToken, _ := tg.GenerateToken("admin@example.com", "admin", "Admin", "User", "Org")
	adminClaims, _ := tg.ValidateToken(adminToken)

	// Create User A with cancel permission on cluster1 only
	userAToken, _ := tg.GenerateToken("usera@example.com", "user", "User", "A", "Org")
	userAClaims, _ := tg.ValidateToken(userAToken)

	// Create User B with cancel permission on both cluster1 and cluster2
	userBToken, _ := tg.GenerateToken("userb@example.com", "user", "User", "B", "Org")
	userBClaims, _ := tg.ValidateToken(userBToken)

	// Group A with cancel permission on cluster1 only
	groupA := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "group-a",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name:        "group-a",
			Description: "Group A with cancel on cluster1",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://cluster1.example.com:6443": {
					Actions: []string{"view", "cancel"},
				},
			},
		},
	}

	// Group B with cancel permission on both clusters
	groupB := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "group-b",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name:        "group-b",
			Description: "Group B with cancel on both clusters",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://cluster1.example.com:6443": {
					Actions: []string{"view", "cancel"},
				},
				"https://cluster2.example.com:6443": {
					Actions: []string{"view", "cancel"},
				},
			},
		},
	}

	userA := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-usera-example-com",
			Namespace: "krkn-operator-system",
			Labels: map[string]string{
				"group.krkn.krkn-chaos.dev/group-a": "true",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:            "usera@example.com",
			Name:              "User",
			Surname:           "A",
			Role:              "user",
			PasswordSecretRef: "usera-password",
		},
	}

	userB := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-userb-example-com",
			Namespace: "krkn-operator-system",
			Labels: map[string]string{
				"group.krkn.krkn-chaos.dev/group-b": "true",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:            "userb@example.com",
			Name:              "User",
			Surname:           "B",
			Role:              "user",
			PasswordSecretRef: "userb-password",
		},
	}

	// Multi-cluster run with jobs on both cluster1 and cluster2
	multiClusterRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "multi-cluster-run",
			Namespace: "krkn-operator-system",
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{
					ClusterName:   "cluster1",
					ClusterAPIURL: "https://cluster1.example.com:6443",
					JobID:         "job-1-cluster1",
				},
				{
					ClusterName:   "cluster2",
					ClusterAPIURL: "https://cluster2.example.com:6443",
					JobID:         "job-2-cluster2",
				},
			},
		},
	}

	// Single cluster run with job only on cluster1
	singleClusterRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "single-cluster-run",
			Namespace: "krkn-operator-system",
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{
					ClusterName:   "cluster1",
					ClusterAPIURL: "https://cluster1.example.com:6443",
					JobID:         "job-3-cluster1",
				},
			},
		},
	}

	tests := []struct {
		name         string
		claims       *auth.Claims
		scenarioRun  *krknv1alpha1.KrknScenarioRun
		expectCancel bool
		testName     string
	}{
		{
			name:         "admin can cancel multi-cluster run",
			claims:       adminClaims,
			scenarioRun:  multiClusterRun,
			expectCancel: true,
			testName:     "admin-multi",
		},
		{
			name:         "admin can cancel single-cluster run",
			claims:       adminClaims,
			scenarioRun:  singleClusterRun,
			expectCancel: true,
			testName:     "admin-single",
		},
		{
			name:         "user A cannot cancel multi-cluster run (missing cancel on cluster2)",
			claims:       userAClaims,
			scenarioRun:  multiClusterRun,
			expectCancel: false,
			testName:     "userA-multi",
		},
		{
			name:         "user A can cancel single-cluster run (has cancel on cluster1)",
			claims:       userAClaims,
			scenarioRun:  singleClusterRun,
			expectCancel: true,
			testName:     "userA-single",
		},
		{
			name:         "user B can cancel multi-cluster run (has cancel on all clusters)",
			claims:       userBClaims,
			scenarioRun:  multiClusterRun,
			expectCancel: true,
			testName:     "userB-multi",
		},
		{
			name:         "user B can cancel single-cluster run",
			claims:       userBClaims,
			scenarioRun:  singleClusterRun,
			expectCancel: true,
			testName:     "userB-single",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fakeclient.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(groupA, groupB, userA, userB).
				Build()

			handler := &Handler{
				client:    fakeClient,
				clientset: fake.NewSimpleClientset(),
				namespace: "krkn-operator-system",
			}

			ctx := context.WithValue(context.Background(), auth.UserClaimsKey, tt.claims)

			hasAccess, err := handler.checkScenarioRunCancelAccess(
				ctx,
				tt.claims.UserID,
				tt.scenarioRun,
			)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if hasAccess != tt.expectCancel {
				t.Errorf("Expected cancel permission=%v, got %v", tt.expectCancel, hasAccess)
			}
		})
	}
}

// TestClusterSegregationWithNoGroups verifies that users with no group memberships
// cannot see any runs
func TestClusterSegregationWithNoGroups(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tg := auth.NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		TokenDuration,
		"krkn-operator",
	)

	userToken, _ := tg.GenerateToken("orphan@example.com", "user", "Orphan", "User", "Org")
	userClaims, _ := tg.ValidateToken(userToken)

	// Create user with NO group memberships
	orphanUser := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-orphan-example-com",
			Namespace: "krkn-operator-system",
			Labels:    map[string]string{}, // No group labels
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:            "orphan@example.com",
			Name:              "Orphan",
			Surname:           "User",
			Role:              "user",
			PasswordSecretRef: "orphan-password",
		},
	}

	runs := []krknv1alpha1.KrknScenarioRun{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "run1"},
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
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(orphanUser).
		Build()

	handler := &Handler{
		client:    fakeClient,
		clientset: fake.NewSimpleClientset(),
		namespace: "krkn-operator-system",
	}

	ctx := context.WithValue(context.Background(), auth.UserClaimsKey, userClaims)
	filtered := handler.filterScenarioRunsByGroupPermission(runs, ctx)

	if len(filtered) != 0 {
		t.Errorf("User with no groups should see 0 runs, got %d", len(filtered))
	}
}

// TestClusterSegregationLegacyRunsWithoutJobs verifies that legacy runs
// without jobs are excluded for regular users but visible to admins
func TestClusterSegregationLegacyRunsWithoutJobs(t *testing.T) {
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
					Actions: []string{"view"},
				},
			},
		},
	}

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
			Name:              "Regular",
			Surname:           "User",
			Role:              "user",
			PasswordSecretRef: "user-password",
		},
	}

	runs := []krknv1alpha1.KrknScenarioRun{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "legacy-run-no-jobs"},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				ClusterJobs: []krknv1alpha1.ClusterJobStatus{}, // No jobs - legacy run
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "new-run-with-jobs"},
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
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(testGroup, testUser).
		Build()

	handler := &Handler{
		client:    fakeClient,
		clientset: fake.NewSimpleClientset(),
		namespace: "krkn-operator-system",
	}

	// Test admin - should see both runs
	adminCtx := context.WithValue(context.Background(), auth.UserClaimsKey, adminClaims)
	adminFiltered := handler.filterScenarioRunsByGroupPermission(runs, adminCtx)

	if len(adminFiltered) != 2 {
		t.Errorf("Admin should see 2 runs (including legacy), got %d", len(adminFiltered))
	}

	// Test regular user - should see only new run with jobs
	userCtx := context.WithValue(context.Background(), auth.UserClaimsKey, userClaims)
	userFiltered := handler.filterScenarioRunsByGroupPermission(runs, userCtx)

	if len(userFiltered) != 1 {
		t.Errorf("User should see 1 run (excluding legacy), got %d", len(userFiltered))
	}

	if len(userFiltered) > 0 && userFiltered[0].Name != "new-run-with-jobs" {
		t.Errorf("User should see 'new-run-with-jobs', got '%s'", userFiltered[0].Name)
	}
}

// TestEndToEndClusterSegregation is an integration test that verifies
// the complete flow: creating runs and listing them with proper filtering
func TestEndToEndClusterSegregation(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tg := auth.NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		TokenDuration,
		"krkn-operator",
	)

	// Create User A with access to cluster1
	userAToken, _ := tg.GenerateToken("usera@example.com", "user", "User", "A", "Org")
	userAClaims, _ := tg.ValidateToken(userAToken)

	// Create User B with access to cluster2
	userBToken, _ := tg.GenerateToken("userb@example.com", "user", "User", "B", "Org")
	userBClaims, _ := tg.ValidateToken(userBToken)

	// Create groups with permissions
	groupA := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "group-a",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name:        "group-a",
			Description: "Group A",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://cluster1.example.com:6443": {
					Actions: []string{"view", "run", "cancel"},
				},
			},
		},
	}

	groupB := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "group-b",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name:        "group-b",
			Description: "Group B",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://cluster2.example.com:6443": {
					Actions: []string{"view", "run", "cancel"},
				},
			},
		},
	}

	// Create users
	userA := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-usera-example-com",
			Namespace: "krkn-operator-system",
			Labels: map[string]string{
				"group.krkn.krkn-chaos.dev/group-a": "true",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:            "usera@example.com",
			Name:              "User",
			Surname:           "A",
			Role:              "user",
			PasswordSecretRef: "usera-password",
		},
	}

	userB := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-userb-example-com",
			Namespace: "krkn-operator-system",
			Labels: map[string]string{
				"group.krkn.krkn-chaos.dev/group-b": "true",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:            "userb@example.com",
			Name:              "User",
			Surname:           "B",
			Role:              "user",
			PasswordSecretRef: "userb-password",
		},
	}

	// Manually create scenario runs (simulating what controller does)
	// In real scenario, these would be created via the API endpoint and controller
	runOnCluster1 := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run-on-cluster1",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			ScenarioImage: "test:latest",
			ScenarioName:  "test-scenario",
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{
					ClusterName:   "cluster1",
					ClusterAPIURL: "https://cluster1.example.com:6443",
					JobID:         "job-1",
				},
			},
		},
	}

	runOnCluster2 := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run-on-cluster2",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			ScenarioImage: "test:latest",
			ScenarioName:  "test-scenario",
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{
					ClusterName:   "cluster2",
					ClusterAPIURL: "https://cluster2.example.com:6443",
					JobID:         "job-2",
				},
			},
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(groupA, groupB, userA, userB, runOnCluster1, runOnCluster2).
		Build()

	handler := &Handler{
		client:    fakeClient,
		clientset: fake.NewSimpleClientset(),
		namespace: "krkn-operator-system",
	}

	// Test 1: User A lists runs - should see only run-on-cluster1
	t.Run("UserA sees only cluster1 runs", func(t *testing.T) {
		var allRuns krknv1alpha1.KrknScenarioRunList
		ctx := context.WithValue(context.Background(), auth.UserClaimsKey, userAClaims)

		if err := fakeClient.List(ctx, &allRuns); err != nil {
			t.Fatalf("Failed to list runs: %v", err)
		}

		filtered := handler.filterScenarioRunsByGroupPermission(allRuns.Items, ctx)

		if len(filtered) != 1 {
			t.Errorf("User A should see 1 run, got %d", len(filtered))
			for _, run := range filtered {
				t.Logf("  - %s", run.Name)
			}
		}

		if len(filtered) > 0 && filtered[0].Name != "run-on-cluster1" {
			t.Errorf("User A should see 'run-on-cluster1', got '%s'", filtered[0].Name)
		}
	})

	// Test 2: User B lists runs - should see only run-on-cluster2
	t.Run("UserB sees only cluster2 runs", func(t *testing.T) {
		var allRuns krknv1alpha1.KrknScenarioRunList
		ctx := context.WithValue(context.Background(), auth.UserClaimsKey, userBClaims)

		if err := fakeClient.List(ctx, &allRuns); err != nil {
			t.Fatalf("Failed to list runs: %v", err)
		}

		filtered := handler.filterScenarioRunsByGroupPermission(allRuns.Items, ctx)

		if len(filtered) != 1 {
			t.Errorf("User B should see 1 run, got %d", len(filtered))
			for _, run := range filtered {
				t.Logf("  - %s", run.Name)
			}
		}

		if len(filtered) > 0 && filtered[0].Name != "run-on-cluster2" {
			t.Errorf("User B should see 'run-on-cluster2', got '%s'", filtered[0].Name)
		}
	})

	// Test 3: Verify job-level ClusterAPIURL is populated correctly
	t.Run("Verify job ClusterAPIURL is populated", func(t *testing.T) {
		var run krknv1alpha1.KrknScenarioRun
		ctx := context.Background()

		if err := fakeClient.Get(ctx, client.ObjectKey{Name: "run-on-cluster1", Namespace: "krkn-operator-system"}, &run); err != nil {
			t.Fatalf("Failed to get run: %v", err)
		}

		if len(run.Status.ClusterJobs) == 0 {
			t.Error("ClusterJobs should be populated, got empty array")
		}

		expectedURL := "https://cluster1.example.com:6443"
		if run.Status.ClusterJobs[0].ClusterAPIURL != expectedURL {
			t.Errorf("Expected ClusterJobs[0].ClusterAPIURL = '%s', got '%s'", expectedURL, run.Status.ClusterJobs[0].ClusterAPIURL)
		}
	})
}

// TestGetScenarioRunStatusWithoutClusterAPIURL verifies that users can view runs
// that were just created and don't have ClusterAPIURL populated yet (race condition)
func TestGetScenarioRunStatusWithoutClusterAPIURL(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tg := auth.NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		TokenDuration,
		"krkn-operator",
	)

	// Create user with permission on cluster1
	userToken, _ := tg.GenerateToken("user@example.com", "user", "User", "Test", "Org")
	userClaims, _ := tg.ValidateToken(userToken)

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
					Actions: []string{"view"},
				},
			},
		},
	}

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
			Name:              "User",
			Surname:           "Test",
			Role:              "user",
			PasswordSecretRef: "user-password",
		},
	}

	tests := []struct {
		name        string
		scenarioRun *krknv1alpha1.KrknScenarioRun
		expectAllow bool
		description string
	}{
		{
			name: "allow access to run with no jobs (just created)",
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "run-no-jobs",
					Namespace: "krkn-operator-system",
				},
				Status: krknv1alpha1.KrknScenarioRunStatus{
					ClusterJobs: []krknv1alpha1.ClusterJobStatus{},
				},
			},
			expectAllow: true,
			description: "Run just created, no jobs yet",
		},
		{
			name: "allow access to run with jobs missing ClusterAPIURL (controller not processed yet)",
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "run-jobs-no-url",
					Namespace: "krkn-operator-system",
				},
				Status: krknv1alpha1.KrknScenarioRunStatus{
					ClusterJobs: []krknv1alpha1.ClusterJobStatus{
						{
							ClusterName:   "cluster1",
							ClusterAPIURL: "", // Not populated yet
							JobID:         "job-1",
						},
					},
				},
			},
			expectAllow: true,
			description: "Jobs exist but ClusterAPIURL not populated yet",
		},
		{
			name: "allow access to run with job having ClusterAPIURL (user has permission)",
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "run-with-permission",
					Namespace: "krkn-operator-system",
				},
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
			description: "Job with ClusterAPIURL, user has view permission",
		},
		{
			name: "deny access to run with job on unauthorized cluster",
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "run-unauthorized",
					Namespace: "krkn-operator-system",
				},
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
			expectAllow: false,
			description: "Job with ClusterAPIURL, user does NOT have permission",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fakeclient.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(testGroup, testUser, tt.scenarioRun).
				Build()

			handler := &Handler{
				client:    fakeClient,
				clientset: fake.NewSimpleClientset(),
				namespace: "krkn-operator-system",
			}

			req := httptest.NewRequest("GET", "/api/v1/scenarios/run/"+tt.scenarioRun.Name, nil)
			ctx := context.WithValue(req.Context(), auth.UserClaimsKey, userClaims)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.GetScenarioRunStatus(w, req)

			if tt.expectAllow {
				if w.Code != http.StatusOK {
					t.Errorf("%s: Expected status 200, got %d. Response: %s",
						tt.description, w.Code, w.Body.String())
				}
			} else {
				if w.Code != http.StatusForbidden {
					t.Errorf("%s: Expected status 403, got %d. Response: %s",
						tt.description, w.Code, w.Body.String())
				}
			}
		})
	}
}
