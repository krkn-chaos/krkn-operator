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

// TestScenarioRunStatusResponse_ResiliencyScoreField verifies that the
// ScenarioRunStatusResponse struct correctly includes the ResiliencyScore field
func TestScenarioRunStatusResponse_ResiliencyScoreField(t *testing.T) {
	score := 9.5

	// Create a response with a resiliency score
	response := ScenarioRunStatusResponse{
		ScenarioRunName: "test-scenario-run",
		Phase:           "Succeeded",
		TotalTargets:    1,
		SuccessfulJobs:  1,
		ResiliencyScore: &score,
	}

	// Verify the field is set correctly
	assert.NotNil(t, response.ResiliencyScore)
	assert.Equal(t, 9.5, *response.ResiliencyScore)

	// Verify nil score is also valid
	responseNoScore := ScenarioRunStatusResponse{
		ScenarioRunName: "test-scenario-run-no-score",
		Phase:           "Running",
		RunningJobs:     1,
		ResiliencyScore: nil,
	}

	assert.Nil(t, responseNoScore.ResiliencyScore)

	// Verify JSON serialization includes the field
	jsonData, err := json.Marshal(response)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "resiliencyScore")
	assert.Contains(t, string(jsonData), "9.5")

	// Verify JSON serialization with nil score (should be omitted or null)
	jsonDataNoScore, err := json.Marshal(responseNoScore)
	assert.NoError(t, err)
	// The field should either be omitted or be null due to omitempty
	assert.NotContains(t, string(jsonDataNoScore), "9.5")
}

// TestScenarioRunListItem_ResiliencyScoreField verifies that the
// ScenarioRunListItem struct correctly includes the ResiliencyScore field
func TestScenarioRunListItem_ResiliencyScoreField(t *testing.T) {
	score := 8.7

	// Create a list item with a resiliency score
	listItem := ScenarioRunListItem{
		ScenarioRunName: "test-run",
		ScenarioName:    "test-scenario",
		Phase:           "Succeeded",
		ResiliencyScore: &score,
	}

	// Verify the field is set correctly
	assert.NotNil(t, listItem.ResiliencyScore)
	assert.Equal(t, 8.7, *listItem.ResiliencyScore)

	// Verify nil score is also valid
	listItemNoScore := ScenarioRunListItem{
		ScenarioRunName: "test-run-no-score",
		ScenarioName:    "test-scenario",
		Phase:           "Running",
		ResiliencyScore: nil,
	}

	assert.Nil(t, listItemNoScore.ResiliencyScore)

	// Verify JSON serialization
	jsonData, err := json.Marshal(listItem)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "resiliencyScore")
	assert.Contains(t, string(jsonData), "8.7")
}

// TestListScenarioRuns_WithResiliencyScore verifies that the list endpoint
// correctly includes resiliency scores for scenario runs
func TestListScenarioRuns_WithResiliencyScore(t *testing.T) {
	// Setup scheme
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	score1 := 8.5
	score2 := 9.2

	// Create scenario runs with different scores
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
			Phase:           "Succeeded",
			ResiliencyScore: &score1,
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
			Phase:           "Succeeded",
			ResiliencyScore: &score2,
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
			Phase:           "Running",
			ResiliencyScore: nil, // No score yet
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
	scoreMap := make(map[string]*float64)
	for _, run := range response.ScenarioRuns {
		scoreMap[run.ScenarioRunName] = run.ResiliencyScore
	}

	assert.NotNil(t, scoreMap["scenario-run-1"])
	assert.Equal(t, 8.5, *scoreMap["scenario-run-1"])

	assert.NotNil(t, scoreMap["scenario-run-2"])
	assert.Equal(t, 9.2, *scoreMap["scenario-run-2"])

	assert.Nil(t, scoreMap["scenario-run-3"])
}
