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

package websocket

import (
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func TestHasScenarioRunStatusChanged_PhaseChange(t *testing.T) {
	old := &krknv1alpha1.KrknScenarioRun{
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Pending",
		},
	}

	new := &krknv1alpha1.KrknScenarioRun{
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
		},
	}

	if !hasScenarioRunStatusChanged(old, new) {
		t.Error("Expected status changed when phase changed")
	}
}

func TestHasScenarioRunStatusChanged_JobCountsChange(t *testing.T) {
	old := &krknv1alpha1.KrknScenarioRun{
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase:       "Running",
			RunningJobs: 2,
		},
	}

	new := &krknv1alpha1.KrknScenarioRun{
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase:       "Running",
			RunningJobs: 3,
		},
	}

	if !hasScenarioRunStatusChanged(old, new) {
		t.Error("Expected status changed when job counts changed")
	}
}

func TestHasScenarioRunStatusChanged_JobPhaseChange(t *testing.T) {
	old := &krknv1alpha1.KrknScenarioRun{
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{JobID: "job-1", Phase: "Pending"},
			},
		},
	}

	new := &krknv1alpha1.KrknScenarioRun{
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{JobID: "job-1", Phase: "Running"},
			},
		},
	}

	if !hasScenarioRunStatusChanged(old, new) {
		t.Error("Expected status changed when job phase changed")
	}
}

func TestHasScenarioRunStatusChanged_NoChange(t *testing.T) {
	old := &krknv1alpha1.KrknScenarioRun{
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase:          "Running",
			TotalTargets:   3,
			RunningJobs:    2,
			SuccessfulJobs: 1,
			FailedJobs:     0,
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{JobID: "job-1", Phase: "Running", RetryCount: 0},
			},
		},
	}

	new := &krknv1alpha1.KrknScenarioRun{
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase:          "Running",
			TotalTargets:   3,
			RunningJobs:    2,
			SuccessfulJobs: 1,
			FailedJobs:     0,
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{JobID: "job-1", Phase: "Running", RetryCount: 0},
			},
		},
	}

	if hasScenarioRunStatusChanged(old, new) {
		t.Error("Expected no status change when everything is identical")
	}
}

func TestHasScenarioRunStatusChanged_RetryCountChange(t *testing.T) {
	old := &krknv1alpha1.KrknScenarioRun{
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{JobID: "job-1", Phase: "Running", RetryCount: 0},
			},
		},
	}

	new := &krknv1alpha1.KrknScenarioRun{
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{JobID: "job-1", Phase: "Running", RetryCount: 1},
			},
		},
	}

	if !hasScenarioRunStatusChanged(old, new) {
		t.Error("Expected status changed when retry count changed")
	}
}

func TestHasGraphRunStatusChanged_PhaseChange(t *testing.T) {
	old := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Pending",
		},
	}

	new := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
		},
	}

	if !hasGraphRunStatusChanged(old, new) {
		t.Error("Expected status changed when phase changed")
	}
}

func TestHasGraphRunStatusChanged_SummaryCountersChange(t *testing.T) {
	old := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
			Summary: krknv1alpha1.GraphRunSummary{
				TotalNodes:     5,
				CompletedNodes: 2,
				RunningNodes:   3,
			},
		},
	}

	new := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
			Summary: krknv1alpha1.GraphRunSummary{
				TotalNodes:     5,
				CompletedNodes: 3,
				RunningNodes:   2,
			},
		},
	}

	if !hasGraphRunStatusChanged(old, new) {
		t.Error("Expected status changed when summary counters changed")
	}
}

func TestHasGraphRunStatusChanged_NodeStatusChange(t *testing.T) {
	old := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
			NodeStatuses: []krknv1alpha1.NodeStatus{
				{NodeID: "node-1", Phase: "Pending"},
			},
		},
	}

	new := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
			NodeStatuses: []krknv1alpha1.NodeStatus{
				{NodeID: "node-1", Phase: "Running"},
			},
		},
	}

	if !hasGraphRunStatusChanged(old, new) {
		t.Error("Expected status changed when node phase changed")
	}
}

func TestHasGraphRunStatusChanged_ResiliencyScoresChange(t *testing.T) {
	baseline := 90.0
	old := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase:            "Succeeded",
			ResiliencyScores: nil,
		},
	}

	new := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Succeeded",
			ResiliencyScores: []krknv1alpha1.GraphClusterScore{
				{
					ClusterName: "cluster1",
					Calculated:  85.5,
					Baseline:    &baseline,
					Status:      "pass",
				},
			},
		},
	}

	if !hasGraphRunStatusChanged(old, new) {
		t.Error("Expected status changed when resiliency scores calculated")
	}
}

func TestHasGraphRunStatusChanged_NoChange(t *testing.T) {
	old := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
			Summary: krknv1alpha1.GraphRunSummary{
				TotalNodes:     5,
				CompletedNodes: 2,
				RunningNodes:   3,
				FailedNodes:    0,
				PendingNodes:   0,
			},
			NodeStatuses: []krknv1alpha1.NodeStatus{
				{NodeID: "node-1", Phase: "Running"},
			},
		},
	}

	new := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
			Summary: krknv1alpha1.GraphRunSummary{
				TotalNodes:     5,
				CompletedNodes: 2,
				RunningNodes:   3,
				FailedNodes:    0,
				PendingNodes:   0,
			},
			NodeStatuses: []krknv1alpha1.NodeStatus{
				{NodeID: "node-1", Phase: "Running"},
			},
		},
	}

	if hasGraphRunStatusChanged(old, new) {
		t.Error("Expected no status change when everything is identical")
	}
}

func TestHasGraphRunStatusChanged_ScenarioRunRefChange(t *testing.T) {
	old := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
			NodeStatuses: []krknv1alpha1.NodeStatus{
				{NodeID: "node-1", Phase: "Running", ScenarioRunRef: ""},
			},
		},
	}

	new := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
			NodeStatuses: []krknv1alpha1.NodeStatus{
				{NodeID: "node-1", Phase: "Running", ScenarioRunRef: "run-abc123"},
			},
		},
	}

	if !hasGraphRunStatusChanged(old, new) {
		t.Error("Expected status changed when scenario run ref changed")
	}
}

// TestHasScenarioRunStatusChanged_JobOrderChanged tests that job phase changes
// are detected even when jobs are reordered in the array (comparing by JobID, not index)
func TestHasScenarioRunStatusChanged_JobOrderChanged(t *testing.T) {
	old := &krknv1alpha1.KrknScenarioRun{
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{JobID: "job-1", Phase: "Pending"},
				{JobID: "job-2", Phase: "Running"},
			},
		},
	}

	// Same jobs but reordered, and job-1's phase changed
	new := &krknv1alpha1.KrknScenarioRun{
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{JobID: "job-2", Phase: "Running"},
				{JobID: "job-1", Phase: "Succeeded"}, // Changed from Pending
			},
		},
	}

	if !hasScenarioRunStatusChanged(old, new) {
		t.Error("Expected status changed when job phase changed, even if jobs reordered")
	}
}

// TestHasScenarioRunStatusChanged_MultipleJobsWithPhaseChange tests that
// changes are detected correctly when there are multiple jobs and one changes
func TestHasScenarioRunStatusChanged_MultipleJobsWithPhaseChange(t *testing.T) {
	old := &krknv1alpha1.KrknScenarioRun{
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{JobID: "job-a8499b7f", Phase: "Pending"},
				{JobID: "job-xyz123", Phase: "Running"},
				{JobID: "job-abc456", Phase: "Succeeded"},
			},
		},
	}

	// job-a8499b7f changed from Pending to Succeeded (the bug scenario)
	new := &krknv1alpha1.KrknScenarioRun{
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{JobID: "job-a8499b7f", Phase: "Succeeded"}, // This job succeeded
				{JobID: "job-xyz123", Phase: "Running"},
				{JobID: "job-abc456", Phase: "Succeeded"},
			},
		},
	}

	if !hasScenarioRunStatusChanged(old, new) {
		t.Error("Expected status changed when one of multiple jobs changed phase")
	}
}

// TestHasScenarioRunStatusChanged_NewJobAdded tests that adding a new job
// is correctly detected
func TestHasScenarioRunStatusChanged_NewJobAdded(t *testing.T) {
	old := &krknv1alpha1.KrknScenarioRun{
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{JobID: "job-1", Phase: "Running"},
			},
		},
	}

	new := &krknv1alpha1.KrknScenarioRun{
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{JobID: "job-1", Phase: "Running"},
				{JobID: "job-2", Phase: "Pending"}, // New job
			},
		},
	}

	if !hasScenarioRunStatusChanged(old, new) {
		t.Error("Expected status changed when new job added")
	}
}

// TestHasGraphRunStatusChanged_NodeOrderChanged tests that node phase changes
// are detected even when nodes are reordered in the array (comparing by NodeID, not index)
func TestHasGraphRunStatusChanged_NodeOrderChanged(t *testing.T) {
	old := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
			NodeStatuses: []krknv1alpha1.NodeStatus{
				{NodeID: "node-1", Phase: "Pending"},
				{NodeID: "node-2", Phase: "Running"},
			},
		},
	}

	// Same nodes but reordered, and node-1's phase changed
	new := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
			NodeStatuses: []krknv1alpha1.NodeStatus{
				{NodeID: "node-2", Phase: "Running"},
				{NodeID: "node-1", Phase: "Completed"}, // Changed from Pending
			},
		},
	}

	if !hasGraphRunStatusChanged(old, new) {
		t.Error("Expected status changed when node phase changed, even if nodes reordered")
	}
}

// TestHasGraphRunStatusChanged_NewNodeAdded tests that adding a new node
// is correctly detected
func TestHasGraphRunStatusChanged_NewNodeAdded(t *testing.T) {
	old := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
			NodeStatuses: []krknv1alpha1.NodeStatus{
				{NodeID: "node-1", Phase: "Running"},
			},
		},
	}

	new := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
			NodeStatuses: []krknv1alpha1.NodeStatus{
				{NodeID: "node-1", Phase: "Running"},
				{NodeID: "node-2", Phase: "Pending"}, // New node
			},
		},
	}

	if !hasGraphRunStatusChanged(old, new) {
		t.Error("Expected status changed when new node added")
	}
}
