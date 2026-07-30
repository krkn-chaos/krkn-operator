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

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// TestGetGraphRun_WithNodeResiliencyScores verifies that the GetGraphRun API
// returns per-cluster resiliency scores for each node in the graph
func TestGetGraphRun_WithNodeResiliencyScores(t *testing.T) {
	ctx := context.Background()

	// Setup scheme
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	baseline := 9.0

	// Create scenario runs with per-cluster resiliency scores
	scenarioRun1 := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scenario-run-node-1",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "test-target",
			TargetClusters:  map[string][]string{"local": {"cluster1"}},
			ScenarioName:    "pod-delete",
			ScenarioImage:   "quay.io/krkn-chaos/krkn-hub:pod-scenarios",
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
			Name:      "scenario-run-node-2",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "test-target",
			TargetClusters:  map[string][]string{"local": {"cluster1"}},
			ScenarioName:    "network-chaos",
			ScenarioImage:   "quay.io/krkn-chaos/krkn-hub:network-scenarios",
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Succeeded",
			ResiliencyScores: []krknv1alpha1.ClusterResiliencyScore{
				{ClusterName: "cluster1", Score: 9.3},
			},
		},
	}

	// Create a graph run with per-cluster resiliency scores
	graphRun := &krknv1alpha1.KrknGraphRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-graphrun-with-scores",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknGraphRunSpec{
			Graph: map[string]krknv1alpha1.GraphScenarioNode{
				"node-1": {
					Name:  "pod-delete",
					Image: "quay.io/krkn-chaos/krkn-hub:pod-scenarios",
				},
				"node-2": {
					Name:      "network-chaos",
					Image:     "quay.io/krkn-chaos/krkn-hub:network-scenarios",
					DependsOn: stringPtr("node-1"),
				},
			},
			TargetRequestID:         "test-target",
			TargetClusters:          map[string][]string{"local": {"cluster1"}},
			ResiliencyScoreEnabled:  true,
			ResiliencyScoreBaseline: &baseline,
		},
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Completed",
			NodeStatuses: []krknv1alpha1.NodeStatus{
				{
					NodeID:         "node-1",
					NodeName:       "pod-delete",
					Phase:          "Completed",
					ScenarioRunRef: "scenario-run-node-1",
				},
				{
					NodeID:         "node-2",
					NodeName:       "network-chaos",
					Phase:          "Completed",
					ScenarioRunRef: "scenario-run-node-2",
					DependsOn:      []string{"node-1"},
				},
			},
			ResiliencyScores: []krknv1alpha1.GraphClusterScore{
				{
					ClusterName: "cluster1",
					Calculated:  8.9,
					Baseline:    &baseline,
					Status:      "fail",
					Message:     "Score 8.90 is below baseline 9.00",
					NodeContributions: map[string]float64{
						"node-1": 8.5,
						"node-2": 9.3,
					},
				},
			},
		},
	}

	// Create fake client
	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(graphRun, scenarioRun1, scenarioRun2).
		Build()

	fakeClientset := fake.NewSimpleClientset()

	// Create handler
	handler := NewTestHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/graphruns/test-graphrun-with-scores", nil)
	req = req.WithContext(ctx)

	// Record response
	w := httptest.NewRecorder()

	// Call handler
	handler.GetGraphRun(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response GraphRunDetailResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify overall graph run fields
	assert.Equal(t, "test-graphrun-with-scores", response.Name)
	assert.Equal(t, "Completed", response.Status.Phase)
	assert.Len(t, response.Status.ResiliencyScores, 1)
	assert.Equal(t, 8.9, response.Status.ResiliencyScores[0].Calculated)
	assert.Equal(t, "fail", response.Status.ResiliencyScores[0].Status)

	// Verify node statuses include per-cluster resiliency scores
	assert.Len(t, response.Status.NodeStatuses, 2)

	// Find node-1 and verify its scores
	var node1, node2 *NodeStatusResponse
	for i := range response.Status.NodeStatuses {
		if response.Status.NodeStatuses[i].NodeID == "node-1" {
			node1 = &response.Status.NodeStatuses[i]
		}
		if response.Status.NodeStatuses[i].NodeID == "node-2" {
			node2 = &response.Status.NodeStatuses[i]
		}
	}

	assert.NotNil(t, node1, "node-1 should be present in response")
	assert.NotNil(t, node2, "node-2 should be present in response")

	// Verify node-1 has per-cluster scores and average
	assert.Len(t, node1.ResiliencyScores, 1, "node-1 should have one cluster score")
	assert.Equal(t, 8.5, node1.ResiliencyScores[0].Score)
	assert.Equal(t, "cluster1", node1.ResiliencyScores[0].ClusterName)
	assert.NotNil(t, node1.ResiliencyScoreAvg)
	assert.Equal(t, 8.5, *node1.ResiliencyScoreAvg)

	// Verify node-2 has per-cluster scores and average
	assert.Len(t, node2.ResiliencyScores, 1, "node-2 should have one cluster score")
	assert.Equal(t, 9.3, node2.ResiliencyScores[0].Score)
	assert.NotNil(t, node2.ResiliencyScoreAvg)
	assert.Equal(t, 9.3, *node2.ResiliencyScoreAvg)
}

// TestGetGraphRun_NodeWithoutResiliencyScore verifies that nodes without
// resiliency scores correctly return empty arrays
func TestGetGraphRun_NodeWithoutResiliencyScore(t *testing.T) {
	ctx := context.Background()

	// Setup scheme
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	// Create a scenario run WITHOUT resiliency scores
	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scenario-run-no-score",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "test-target",
			TargetClusters:  map[string][]string{"local": {"cluster1"}},
			ScenarioName:    "pod-delete",
			ScenarioImage:   "quay.io/krkn-chaos/krkn-hub:pod-scenarios",
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
		},
	}

	// Create a graph run without resiliency enabled
	graphRun := &krknv1alpha1.KrknGraphRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-graphrun-no-score",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknGraphRunSpec{
			Graph: map[string]krknv1alpha1.GraphScenarioNode{
				"node-1": {
					Name:  "pod-delete",
					Image: "quay.io/krkn-chaos/krkn-hub:pod-scenarios",
				},
			},
			TargetRequestID:        "test-target",
			TargetClusters:         map[string][]string{"local": {"cluster1"}},
			ResiliencyScoreEnabled: false,
		},
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
			NodeStatuses: []krknv1alpha1.NodeStatus{
				{
					NodeID:         "node-1",
					NodeName:       "pod-delete",
					Phase:          "Running",
					ScenarioRunRef: "scenario-run-no-score",
				},
			},
		},
	}

	// Create fake client
	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(graphRun, scenarioRun).
		Build()

	fakeClientset := fake.NewSimpleClientset()

	// Create handler
	handler := NewTestHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/graphruns/test-graphrun-no-score", nil)
	req = req.WithContext(ctx)

	// Record response
	w := httptest.NewRecorder()

	// Call handler
	handler.GetGraphRun(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response GraphRunDetailResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify node status does not have scores
	assert.Len(t, response.Status.NodeStatuses, 1)
	assert.Empty(t, response.Status.NodeStatuses[0].ResiliencyScores,
		"Node without resiliency scores should have empty array")
	assert.Nil(t, response.Status.NodeStatuses[0].ResiliencyScoreAvg,
		"Node without resiliency scores should have nil average")
}
