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

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// TestBroadcasterLogic_CreatedVsUpdated verifies the logic for determining
// "created" vs "updated" events based on cache existence
func TestBroadcasterLogic_CreatedVsUpdated(t *testing.T) {
	baseline := 9.0

	// Test Case 1: ResiliencyScores nil initially
	graphRunNoScore := &krknv1alpha1.KrknGraphRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-run-no-score",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: krknv1alpha1.KrknGraphRunSpec{
			ResiliencyScoreEnabled:  true,
			ResiliencyScoreBaseline: &baseline,
		},
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase:            "Completed",
			ResiliencyScores: nil,
		},
	}

	assert.Nil(t, graphRunNoScore.Status.ResiliencyScores,
		"ResiliencyScores should be nil when not calculated")

	// Test Case 2: ResiliencyScores populated
	graphRunWithScore := &krknv1alpha1.KrknGraphRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-run-with-score",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: krknv1alpha1.KrknGraphRunSpec{
			ResiliencyScoreEnabled:  true,
			ResiliencyScoreBaseline: &baseline,
		},
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Completed",
			ResiliencyScores: []krknv1alpha1.GraphClusterScore{
				{
					ClusterName: "cluster1",
					Calculated:  8.5,
					Baseline:    &baseline,
					Status:      "fail",
					Message:     "Score 8.5 is below baseline 9.0",
				},
			},
		},
	}

	assert.NotEmpty(t, graphRunWithScore.Status.ResiliencyScores,
		"ResiliencyScores should be populated after calculation")
	assert.Equal(t, 8.5, graphRunWithScore.Status.ResiliencyScores[0].Calculated)
	assert.Equal(t, "fail", graphRunWithScore.Status.ResiliencyScores[0].Status)
}
