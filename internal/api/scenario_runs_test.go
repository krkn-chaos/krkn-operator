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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func adminContext(req *http.Request) *http.Request {
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin@example.com",
		Role:   "admin",
	})
	return req.WithContext(ctx)
}

func TestListScenarioRuns(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		runs           []*krknv1alpha1.KrknScenarioRun
		query          string
		expectedCount  int
		expectedTotal  int
		expectedPages  int
		expectedPage   int
		checkSortOrder bool
		expectedFirst  string
		expectedLast   string
	}{
		{
			name: "no pagination returns all items sorted newest first",
			runs: []*krknv1alpha1.KrknScenarioRun{
				makeScenarioRun("sr-old", now.Add(-2*time.Hour)),
				makeScenarioRun("sr-mid", now.Add(-1*time.Hour)),
				makeScenarioRun("sr-new", now),
			},
			query:          "",
			expectedCount:  3,
			expectedTotal:  3,
			expectedPages:  0,
			expectedPage:   0,
			checkSortOrder: true,
			expectedFirst:  "sr-new",
			expectedLast:   "sr-old",
		},
		{
			name: "page 1 with limit 2",
			runs: []*krknv1alpha1.KrknScenarioRun{
				makeScenarioRun("sr-1", now.Add(-4*time.Hour)),
				makeScenarioRun("sr-2", now.Add(-3*time.Hour)),
				makeScenarioRun("sr-3", now.Add(-2*time.Hour)),
				makeScenarioRun("sr-4", now.Add(-1*time.Hour)),
				makeScenarioRun("sr-5", now),
			},
			query:         "?page=1&limit=2",
			expectedCount: 2,
			expectedTotal: 5,
			expectedPages: 3,
			expectedPage:  1,
		},
		{
			name: "last page returns remaining items",
			runs: []*krknv1alpha1.KrknScenarioRun{
				makeScenarioRun("sr-1", now.Add(-4*time.Hour)),
				makeScenarioRun("sr-2", now.Add(-3*time.Hour)),
				makeScenarioRun("sr-3", now.Add(-2*time.Hour)),
				makeScenarioRun("sr-4", now.Add(-1*time.Hour)),
				makeScenarioRun("sr-5", now),
			},
			query:         "?page=3&limit=2",
			expectedCount: 1,
			expectedTotal: 5,
			expectedPages: 3,
			expectedPage:  3,
		},
		{
			name:          "empty result set",
			runs:          []*krknv1alpha1.KrknScenarioRun{},
			query:         "?page=1&limit=20",
			expectedCount: 0,
			expectedTotal: 0,
			expectedPages: 0,
			expectedPage:  1,
		},
		{
			name: "limit only returns all (backward compat)",
			runs: []*krknv1alpha1.KrknScenarioRun{
				makeScenarioRun("sr-1", now.Add(-2*time.Hour)),
				makeScenarioRun("sr-2", now.Add(-1*time.Hour)),
				makeScenarioRun("sr-3", now),
			},
			query:         "?limit=1",
			expectedCount: 3,
			expectedTotal: 3,
			expectedPages: 0,
			expectedPage:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			if err := krknv1alpha1.AddToScheme(scheme); err != nil {
				t.Fatalf("failed to add scheme: %v", err)
			}

			builder := fakeclient.NewClientBuilder().WithScheme(scheme)
			for _, r := range tt.runs {
				builder = builder.WithObjects(r)
			}
			fakeClient := builder.Build()
			fakeClientset := fake.NewSimpleClientset()
			handler := NewTestHandler(fakeClient, fakeClientset, "default", "localhost:50051")

			req := adminContext(httptest.NewRequest("GET", "/api/v1/scenarios/run"+tt.query, nil))
			w := httptest.NewRecorder()

			handler.ListScenarioRuns(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", w.Code)
			}

			var response ScenarioRunListResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if len(response.ScenarioRuns) != tt.expectedCount {
				t.Errorf("expected %d items, got %d", tt.expectedCount, len(response.ScenarioRuns))
			}
			if response.Pagination.Total != tt.expectedTotal {
				t.Errorf("expected total %d, got %d", tt.expectedTotal, response.Pagination.Total)
			}
			if response.Pagination.TotalPages != tt.expectedPages {
				t.Errorf("expected totalPages %d, got %d", tt.expectedPages, response.Pagination.TotalPages)
			}
			if response.Pagination.Page != tt.expectedPage {
				t.Errorf("expected page %d, got %d", tt.expectedPage, response.Pagination.Page)
			}

			if tt.checkSortOrder && len(response.ScenarioRuns) > 0 {
				if response.ScenarioRuns[0].ScenarioRunName != tt.expectedFirst {
					t.Errorf("expected first item %q, got %q", tt.expectedFirst, response.ScenarioRuns[0].ScenarioRunName)
				}
				last := response.ScenarioRuns[len(response.ScenarioRuns)-1]
				if last.ScenarioRunName != tt.expectedLast {
					t.Errorf("expected last item %q, got %q", tt.expectedLast, last.ScenarioRunName)
				}
			}
		})
	}
}

func TestListScenarioRuns_PhaseFilter(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add scheme: %v", err)
	}

	now := time.Now()
	running := makeScenarioRun("sr-running", now)
	running.Status.Phase = "Running"
	succeeded := makeScenarioRun("sr-succeeded", now.Add(-1*time.Hour))
	succeeded.Status.Phase = "Succeeded"

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(running, succeeded).
		Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewTestHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := adminContext(httptest.NewRequest("GET", "/api/v1/scenarios/run?phase=Running&page=1&limit=10", nil))
	w := httptest.NewRecorder()

	handler.ListScenarioRuns(w, req)

	var response ScenarioRunListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response.ScenarioRuns) != 1 {
		t.Fatalf("expected 1 running scenario, got %d", len(response.ScenarioRuns))
	}
	if response.ScenarioRuns[0].ScenarioRunName != "sr-running" {
		t.Errorf("expected sr-running, got %s", response.ScenarioRuns[0].ScenarioRunName)
	}
	if response.Pagination.Total != 1 {
		t.Errorf("expected total 1 (filtered), got %d", response.Pagination.Total)
	}
}
