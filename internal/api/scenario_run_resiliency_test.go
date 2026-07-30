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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// TestScenarioRunStatusResponse_ResiliencyScoresField verifies that the
// ScenarioRunStatusResponse struct correctly includes the ResiliencyScores field
func TestScenarioRunStatusResponse_ResiliencyScoresField(t *testing.T) {
	// Create a response with per-cluster resiliency scores
	response := ScenarioRunStatusResponse{
		ScenarioRunName: "test-scenario-run",
		Phase:           "Succeeded",
		TotalTargets:    1,
		SuccessfulJobs:  1,
		ResiliencyScores: []ClusterResiliencyScoreResponse{
			{ClusterName: "cluster1", Score: 9.5},
		},
	}

	// Verify the field is set correctly
	assert.Len(t, response.ResiliencyScores, 1)
	assert.Equal(t, 9.5, response.ResiliencyScores[0].Score)

	// Verify nil scores is also valid
	responseNoScore := ScenarioRunStatusResponse{
		ScenarioRunName: "test-scenario-run-no-score",
		Phase:           "Running",
		RunningJobs:     1,
	}

	assert.Empty(t, responseNoScore.ResiliencyScores)

	// Verify JSON serialization includes the field
	jsonData, err := json.Marshal(response)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "resiliencyScores")
	assert.Contains(t, string(jsonData), "9.5")

	// Verify JSON serialization with no scores (should be omitted)
	jsonDataNoScore, err := json.Marshal(responseNoScore)
	assert.NoError(t, err)
	assert.NotContains(t, string(jsonDataNoScore), "9.5")
}

// TestScenarioRunListItem_ResiliencyScoresField verifies that the
// ScenarioRunListItem struct correctly includes the ResiliencyScores field
func TestScenarioRunListItem_ResiliencyScoresField(t *testing.T) {
	// Create a list item with per-cluster resiliency scores
	listItem := ScenarioRunListItem{
		ScenarioRunName: "test-run",
		ScenarioName:    "test-scenario",
		Phase:           "Succeeded",
		ResiliencyScores: []ClusterResiliencyScoreResponse{
			{ClusterName: "cluster1", Score: 8.7},
		},
	}

	// Verify the field is set correctly
	assert.Len(t, listItem.ResiliencyScores, 1)
	assert.Equal(t, 8.7, listItem.ResiliencyScores[0].Score)

	// Verify empty scores is also valid
	listItemNoScore := ScenarioRunListItem{
		ScenarioRunName: "test-run-no-score",
		ScenarioName:    "test-scenario",
		Phase:           "Running",
	}

	assert.Empty(t, listItemNoScore.ResiliencyScores)

	// Verify JSON serialization
	jsonData, err := json.Marshal(listItem)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "resiliencyScores")
	assert.Contains(t, string(jsonData), "8.7")
}

// TestListScenarioRuns_WithResiliencyScores verifies that the list endpoint
// correctly includes per-cluster resiliency scores for scenario runs
func TestListScenarioRuns_WithResiliencyScores(t *testing.T) {
	// Setup scheme
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	// Create scenario runs with different per-cluster scores
	scenarioRun1 := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scenario-run-1",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "target-1",
			TargetClusters:  map[string][]string{"local": {"cluster1"}},
			ScenarioName:    "scenario-1",
			ScenarioImage:   "quay.io/krkn-chaos/krkn-hub:test",
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Succeeded",
			ResiliencyScores: []krknv1alpha1.ClusterResiliencyScore{
				{ClusterName: "cluster1", Score: 8.5},
			},
		},
	}

	scenarioRun2 := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scenario-run-2",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "target-2",
			TargetClusters:  map[string][]string{"local": {"cluster2"}},
			ScenarioName:    "scenario-2",
			ScenarioImage:   "quay.io/krkn-chaos/krkn-hub:test",
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Succeeded",
			ResiliencyScores: []krknv1alpha1.ClusterResiliencyScore{
				{ClusterName: "cluster2", Score: 9.2},
			},
		},
	}

	scenarioRun3 := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scenario-run-3",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "target-3",
			TargetClusters:  map[string][]string{"local": {"cluster3"}},
			ScenarioName:    "scenario-3",
			ScenarioImage:   "quay.io/krkn-chaos/krkn-hub:test",
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
		},
	}

	// Create fake client
	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(scenarioRun1, scenarioRun2, scenarioRun3).
		Build()

	fakeClientset := fake.NewSimpleClientset()

	// Create handler
	handler := NewTestHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/scenarios/run", nil)

	// Record response
	w := httptest.NewRecorder()

	// Call handler
	handler.ListScenarioRuns(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response ScenarioRunListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Should have 3 scenario runs
	assert.Len(t, response.ScenarioRuns, 3)

	// Find and verify each run
	scoreMap := make(map[string][]ClusterResiliencyScoreResponse)
	for _, run := range response.ScenarioRuns {
		scoreMap[run.ScenarioRunName] = run.ResiliencyScores
	}

	assert.Len(t, scoreMap["scenario-run-1"], 1)
	assert.Equal(t, 8.5, scoreMap["scenario-run-1"][0].Score)

	assert.Len(t, scoreMap["scenario-run-2"], 1)
	assert.Equal(t, 9.2, scoreMap["scenario-run-2"][0].Score)

	assert.Empty(t, scoreMap["scenario-run-3"])
}
