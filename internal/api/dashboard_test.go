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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

// TestGetActiveRunsOverview_AdminSeesAll tests that admin sees all active runs
func TestGetActiveRunsOverview_AdminSeesAll(t *testing.T) {
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	// Create scenario runs with different owners
	runs := []runtime.Object{
		&krknv1alpha1.KrknScenarioRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "run1",
				Namespace: "default",
			},
			Spec: krknv1alpha1.KrknScenarioRunSpec{
				OwnerUserID: "user1@example.com",
			},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Running",
				ClusterJobs: []krknv1alpha1.ClusterJobStatus{
					{
						ClusterName: "cluster1",
						Phase:       "Running",
					},
					{
						ClusterName: "cluster2",
						Phase:       "Running",
					},
				},
			},
		},
		&krknv1alpha1.KrknScenarioRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "run2",
				Namespace: "default",
			},
			Spec: krknv1alpha1.KrknScenarioRunSpec{
				OwnerUserID: "user2@example.com",
			},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Running",
				ClusterJobs: []krknv1alpha1.ClusterJobStatus{
					{
						ClusterName: "cluster1",
						Phase:       "Running",
					},
				},
			},
		},
		&krknv1alpha1.KrknScenarioRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "run3",
				Namespace: "default",
			},
			Spec: krknv1alpha1.KrknScenarioRunSpec{
				OwnerUserID: "user1@example.com",
			},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Succeeded", // Not running
				ClusterJobs: []krknv1alpha1.ClusterJobStatus{
					{
						ClusterName: "cluster3",
						Phase:       "Succeeded",
					},
				},
			},
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(runs...).
		Build()

	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	// Admin context
	ctx := context.WithValue(context.Background(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin@example.com",
		Role:   "admin",
	})

	req := httptest.NewRequest("GET", "/api/v1/dashboard/active-runs", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.GetActiveRunsOverview(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response ActiveRunsOverviewResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Admin should see 2 active runs (run1 and run2)
	if response.TotalActiveRuns != 2 {
		t.Errorf("Expected 2 active runs, got %d", response.TotalActiveRuns)
	}

	// Should have 2 unique clusters (cluster1 and cluster2)
	if response.TotalClusters != 2 {
		t.Errorf("Expected 2 clusters, got %d", response.TotalClusters)
	}

	// cluster1 should have both run1 and run2
	if len(response.ClusterRuns["cluster1"]) != 2 {
		t.Errorf("Expected cluster1 to have 2 runs, got %d", len(response.ClusterRuns["cluster1"]))
	}

	// cluster2 should have only run1
	if len(response.ClusterRuns["cluster2"]) != 1 {
		t.Errorf("Expected cluster2 to have 1 run, got %d", len(response.ClusterRuns["cluster2"]))
	}

	// cluster3 should not appear (run is not active)
	if _, exists := response.ClusterRuns["cluster3"]; exists {
		t.Error("cluster3 should not appear in active runs")
	}
}

// TestGetActiveRunsOverview_UserSeesOwnOnly tests that regular users see only their own runs
func TestGetActiveRunsOverview_UserSeesOwnOnly(t *testing.T) {
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
				OwnerUserID: "user1@example.com",
			},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Running",
				ClusterJobs: []krknv1alpha1.ClusterJobStatus{
					{
						ClusterName: "cluster1",
						Phase:       "Running",
					},
				},
			},
		},
		&krknv1alpha1.KrknScenarioRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "run2",
				Namespace: "default",
			},
			Spec: krknv1alpha1.KrknScenarioRunSpec{
				OwnerUserID: "user2@example.com",
			},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Running",
				ClusterJobs: []krknv1alpha1.ClusterJobStatus{
					{
						ClusterName: "cluster2",
						Phase:       "Running",
					},
				},
			},
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(runs...).
		Build()

	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	// User context for user1
	ctx := context.WithValue(context.Background(), auth.UserClaimsKey, &auth.Claims{
		UserID: "user1@example.com",
		Role:   "user",
	})

	req := httptest.NewRequest("GET", "/api/v1/dashboard/active-runs", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.GetActiveRunsOverview(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response ActiveRunsOverviewResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// User should see only their own 1 active run
	if response.TotalActiveRuns != 1 {
		t.Errorf("Expected 1 active run, got %d", response.TotalActiveRuns)
	}

	// Should have only 1 cluster
	if response.TotalClusters != 1 {
		t.Errorf("Expected 1 cluster, got %d", response.TotalClusters)
	}

	// cluster1 should have run1
	if len(response.ClusterRuns["cluster1"]) != 1 {
		t.Errorf("Expected cluster1 to have 1 run, got %d", len(response.ClusterRuns["cluster1"]))
	}

	// cluster2 should not appear (belongs to other user)
	if _, exists := response.ClusterRuns["cluster2"]; exists {
		t.Error("cluster2 should not appear for user1")
	}
}

// TestGetActiveRunsOverview_NoActiveRuns tests response when no runs are active
func TestGetActiveRunsOverview_NoActiveRuns(t *testing.T) {
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	// All runs are completed
	runs := []runtime.Object{
		&krknv1alpha1.KrknScenarioRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "run1",
				Namespace: "default",
			},
			Spec: krknv1alpha1.KrknScenarioRunSpec{
				OwnerUserID: "user1@example.com",
			},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Succeeded",
				ClusterJobs: []krknv1alpha1.ClusterJobStatus{
					{
						ClusterName: "cluster1",
						Phase:       "Succeeded",
					},
				},
			},
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(runs...).
		Build()

	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	ctx := context.WithValue(context.Background(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin@example.com",
		Role:   "admin",
	})

	req := httptest.NewRequest("GET", "/api/v1/dashboard/active-runs", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.GetActiveRunsOverview(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response ActiveRunsOverviewResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.TotalActiveRuns != 0 {
		t.Errorf("Expected 0 active runs, got %d", response.TotalActiveRuns)
	}

	if response.TotalClusters != 0 {
		t.Errorf("Expected 0 clusters, got %d", response.TotalClusters)
	}

	if len(response.ClusterRuns) != 0 {
		t.Errorf("Expected empty cluster runs map, got %d entries", len(response.ClusterRuns))
	}
}

// TestGetActiveRunsOverview_MixedJobStates tests runs with mixed job states
func TestGetActiveRunsOverview_MixedJobStates(t *testing.T) {
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
				OwnerUserID: "user1@example.com",
			},
			Status: krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Running",
				ClusterJobs: []krknv1alpha1.ClusterJobStatus{
					{
						ClusterName: "cluster1",
						Phase:       "Running",
					},
					{
						ClusterName: "cluster2",
						Phase:       "Succeeded", // Not running
					},
					{
						ClusterName: "cluster3",
						Phase:       "Running",
					},
				},
			},
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(runs...).
		Build()

	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	ctx := context.WithValue(context.Background(), auth.UserClaimsKey, &auth.Claims{
		UserID: "user1@example.com",
		Role:   "user",
	})

	req := httptest.NewRequest("GET", "/api/v1/dashboard/active-runs", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.GetActiveRunsOverview(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response ActiveRunsOverviewResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Should count as 1 active run (has at least one running job)
	if response.TotalActiveRuns != 1 {
		t.Errorf("Expected 1 active run, got %d", response.TotalActiveRuns)
	}

	// Should have 2 clusters with running jobs (cluster1 and cluster3)
	if response.TotalClusters != 2 {
		t.Errorf("Expected 2 clusters, got %d", response.TotalClusters)
	}

	// cluster2 should not appear (job is succeeded)
	if _, exists := response.ClusterRuns["cluster2"]; exists {
		t.Error("cluster2 should not appear (job is not running)")
	}
}
