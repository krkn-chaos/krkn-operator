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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func makeScenarioRun(name string, createdAt time.Time) *krknv1alpha1.KrknScenarioRun {
	return &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(createdAt),
			Labels:            map[string]string{},
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			ScenarioName: "test-scenario",
			OwnerUserID:  "user@example.com",
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
		},
	}
}

func makeGraphRun(name string, createdAt time.Time) *krknv1alpha1.KrknGraphRun {
	return &krknv1alpha1.KrknGraphRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(createdAt),
		},
		Spec: krknv1alpha1.KrknGraphRunSpec{
			OwnerUserID: "user@example.com",
		},
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
		},
	}
}

func TestListJobs_NoPagination(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	now := time.Now()
	sr1 := makeScenarioRun("sr-1", now.Add(-2*time.Hour))
	sr2 := makeScenarioRun("sr-2", now.Add(-1*time.Hour))
	gr1 := makeGraphRun("gr-1", now)

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(sr1, sr2, gr1).
		Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewTestHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("GET", "/api/v2/jobs", nil)
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin@example.com",
		Role:   "admin",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ListJobs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response UnifiedJobsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response.Jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(response.Jobs))
	}
	if response.Pagination.Total != 3 {
		t.Errorf("expected total 3, got %d", response.Pagination.Total)
	}

	// Verify sorted by creation time DESC (newest first)
	if response.Jobs[0].Name != "gr-1" {
		t.Errorf("expected first item to be gr-1 (newest), got %s", response.Jobs[0].Name)
	}
	if response.Jobs[2].Name != "sr-1" {
		t.Errorf("expected last item to be sr-1 (oldest), got %s", response.Jobs[2].Name)
	}
}

func TestListJobs_Paginated(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	now := time.Now()
	sr1 := makeScenarioRun("sr-1", now.Add(-5*time.Hour))
	sr2 := makeScenarioRun("sr-2", now.Add(-4*time.Hour))
	sr3 := makeScenarioRun("sr-3", now.Add(-3*time.Hour))
	gr1 := makeGraphRun("gr-1", now.Add(-2*time.Hour))
	gr2 := makeGraphRun("gr-2", now.Add(-1*time.Hour))

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(sr1, sr2, sr3, gr1, gr2).
		Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewTestHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	// Page 1 with limit 2
	req := httptest.NewRequest("GET", "/api/v2/jobs?page=1&limit=2", nil)
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin@example.com",
		Role:   "admin",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ListJobs(w, req)

	var page1 UnifiedJobsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &page1); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(page1.Jobs) != 2 {
		t.Errorf("expected 2 items on page 1, got %d", len(page1.Jobs))
	}
	if page1.Pagination.Total != 5 {
		t.Errorf("expected total 5, got %d", page1.Pagination.Total)
	}
	if page1.Pagination.TotalPages != 3 {
		t.Errorf("expected 3 pages, got %d", page1.Pagination.TotalPages)
	}
	if page1.Pagination.Page != 1 {
		t.Errorf("expected page 1, got %d", page1.Pagination.Page)
	}

	// Page 3 (last page with 1 item)
	req2 := httptest.NewRequest("GET", "/api/v2/jobs?page=3&limit=2", nil)
	ctx2 := context.WithValue(req2.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin@example.com",
		Role:   "admin",
	})
	req2 = req2.WithContext(ctx2)
	w2 := httptest.NewRecorder()

	handler.ListJobs(w2, req2)

	var page3 UnifiedJobsResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &page3); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(page3.Jobs) != 1 {
		t.Errorf("expected 1 item on last page, got %d", len(page3.Jobs))
	}
}

func TestListJobs_Empty(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewTestHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("GET", "/api/v2/jobs?page=1&limit=20", nil)
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin@example.com",
		Role:   "admin",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ListJobs(w, req)

	var response UnifiedJobsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response.Jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(response.Jobs))
	}
	if response.Pagination.Total != 0 {
		t.Errorf("expected total 0, got %d", response.Pagination.Total)
	}
}

func TestListJobs_SkipsGraphRunChildren(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	now := time.Now()
	standalone := makeScenarioRun("standalone-run", now)

	child := makeScenarioRun("graph-child-run", now.Add(-time.Hour))
	child.Labels["krkn.dev/graph-run"] = "parent-graph"

	gr := makeGraphRun("parent-graph", now.Add(-30*time.Minute))

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(standalone, child, gr).
		Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewTestHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("GET", "/api/v2/jobs", nil)
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin@example.com",
		Role:   "admin",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ListJobs(w, req)

	var response UnifiedJobsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Should only have standalone run + graph run, not the graph child
	if len(response.Jobs) != 2 {
		t.Errorf("expected 2 jobs (standalone + graph), got %d", len(response.Jobs))
	}

	for _, job := range response.Jobs {
		if job.Name == "graph-child-run" {
			t.Error("graph-run child should not appear in unified list")
		}
	}
}

func TestListJobs_TypeDiscriminator(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	now := time.Now()
	sr := makeScenarioRun("sr-1", now)
	gr := makeGraphRun("gr-1", now.Add(-time.Hour))

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(sr, gr).
		Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewTestHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("GET", "/api/v2/jobs", nil)
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin@example.com",
		Role:   "admin",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ListJobs(w, req)

	var response UnifiedJobsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	for _, job := range response.Jobs {
		switch job.Type {
		case "scenarioRun":
			if job.ScenarioRun == nil {
				t.Error("scenarioRun type should have ScenarioRun field set")
			}
			if job.GraphRun != nil {
				t.Error("scenarioRun type should not have GraphRun field set")
			}
		case "graphRun":
			if job.GraphRun == nil {
				t.Error("graphRun type should have GraphRun field set")
			}
			if job.ScenarioRun != nil {
				t.Error("graphRun type should not have ScenarioRun field set")
			}
		default:
			t.Errorf("unexpected type: %s", job.Type)
		}
	}
}

func TestBuildUnifiedJobList_SortOrder(t *testing.T) {
	now := time.Now()

	scenarioRuns := []krknv1alpha1.KrknScenarioRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "sr-oldest",
				CreationTimestamp: metav1.NewTime(now.Add(-3 * time.Hour)),
				Labels:            map[string]string{},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "sr-newest",
				CreationTimestamp: metav1.NewTime(now),
				Labels:            map[string]string{},
			},
		},
	}

	graphRuns := []krknv1alpha1.KrknGraphRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "gr-middle",
				CreationTimestamp: metav1.NewTime(now.Add(-1 * time.Hour)),
			},
		},
	}

	jobs := BuildUnifiedJobList(scenarioRuns, graphRuns)

	if len(jobs) != 3 {
		t.Fatalf("expected 3 items, got %d", len(jobs))
	}

	// Newest first
	expectedOrder := []string{"sr-newest", "gr-middle", "sr-oldest"}
	for i, expected := range expectedOrder {
		if jobs[i].Name != expected {
			t.Errorf("position %d: expected %s, got %s", i, expected, jobs[i].Name)
		}
	}
}
