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

Assisted-by: Claude Opus 4.8 (claude-opus-4-8)
*/

package controller

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// expectedCondition captures the fields we assert for a single condition.
type expectedCondition struct {
	status metav1.ConditionStatus
	reason string
}

func TestBuildStatusConditions(t *testing.T) {
	tests := []struct {
		name        string
		clusterJobs []krknv1alpha1.ClusterJobStatus
		// expected keyed by condition Type
		expected map[string]expectedCondition
	}{
		{
			name:        "pending run with no jobs",
			clusterJobs: nil,
			expected: map[string]expectedCondition{
				"Ready":       {metav1.ConditionFalse, "JobsIncomplete"},
				"Progressing": {metav1.ConditionTrue, "JobsInProgress"},
				"Failed":      {metav1.ConditionFalse, "NoJobFailures"},
			},
		},
		{
			name: "running with jobs in progress",
			clusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ClusterName: "c1", Phase: "Running"},
				{ClusterName: "c2", Phase: "Succeeded"},
			},
			expected: map[string]expectedCondition{
				"Ready":       {metav1.ConditionFalse, "JobsIncomplete"},
				"Progressing": {metav1.ConditionTrue, "JobsInProgress"},
				"Failed":      {metav1.ConditionFalse, "NoJobFailures"},
			},
		},
		{
			name: "all jobs succeeded",
			clusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ClusterName: "c1", Phase: "Succeeded"},
				{ClusterName: "c2", Phase: "Succeeded"},
			},
			expected: map[string]expectedCondition{
				"Ready":       {metav1.ConditionTrue, "AllJobsSucceeded"},
				"Progressing": {metav1.ConditionFalse, "RunComplete"},
				"Failed":      {metav1.ConditionFalse, "NoJobFailures"},
			},
		},
		{
			name: "all jobs failed",
			clusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ClusterName: "c1", Phase: "Failed"},
				{ClusterName: "c2", Phase: "MaxRetriesExceeded"},
			},
			expected: map[string]expectedCondition{
				"Ready":       {metav1.ConditionFalse, "JobsIncomplete"},
				"Progressing": {metav1.ConditionFalse, "RunComplete"},
				"Failed":      {metav1.ConditionTrue, "AllJobsFailed"},
			},
		},
		{
			name: "partially failed run",
			clusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ClusterName: "c1", Phase: "Succeeded"},
				{ClusterName: "c2", Phase: "Failed"},
			},
			expected: map[string]expectedCondition{
				"Ready":       {metav1.ConditionFalse, "JobsIncomplete"},
				"Progressing": {metav1.ConditionFalse, "RunComplete"},
				"Failed":      {metav1.ConditionTrue, "SomeJobsFailed"},
			},
		},
	}

	r := &KrknScenarioRunReconciler{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenarioRun := &krknv1alpha1.KrknScenarioRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-scenario",
					Namespace:  "default",
					Generation: 7,
				},
				Status: krknv1alpha1.KrknScenarioRunStatus{
					ClusterJobs: tt.clusterJobs,
				},
			}

			// calculateOverallStatus computes counters, phase, and conditions.
			r.calculateOverallStatus(scenarioRun)

			conditions := scenarioRun.Status.Conditions
			if len(conditions) != len(tt.expected) {
				t.Fatalf("expected %d conditions, got %d: %+v",
					len(tt.expected), len(conditions), conditions)
			}

			for condType, want := range tt.expected {
				got := findCondition(conditions, condType)
				if got == nil {
					t.Errorf("missing condition %q", condType)
					continue
				}
				if got.Status != want.status {
					t.Errorf("condition %q: expected status %q, got %q",
						condType, want.status, got.Status)
				}
				if got.Reason != want.reason {
					t.Errorf("condition %q: expected reason %q, got %q",
						condType, want.reason, got.Reason)
				}
				// Reason must be non-empty CamelCase (k8s requirement).
				if got.Reason == "" {
					t.Errorf("condition %q: reason must not be empty", condType)
				}
				if got.ObservedGeneration != scenarioRun.Generation {
					t.Errorf("condition %q: expected ObservedGeneration %d, got %d",
						condType, scenarioRun.Generation, got.ObservedGeneration)
				}
				if got.LastTransitionTime.IsZero() {
					t.Errorf("condition %q: LastTransitionTime should be set", condType)
				}
			}
		})
	}
}

func findCondition(conditions []metav1.Condition, condType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == condType {
			return &conditions[i]
		}
	}
	return nil
}
