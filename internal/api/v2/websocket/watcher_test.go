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

func TestHasGraphRunStatusChanged_ResiliencyScoreChange(t *testing.T) {
	baseline := 90.0
	old := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase:           "Succeeded",
			ResiliencyScore: nil,
		},
	}

	new := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Succeeded",
			ResiliencyScore: &krknv1alpha1.ResiliencyScoreResult{
				Calculated: 85.5,
				Baseline:   &baseline,
				Status:     "success",
			},
		},
	}

	if !hasGraphRunStatusChanged(old, new) {
		t.Error("Expected status changed when resiliency score calculated")
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
