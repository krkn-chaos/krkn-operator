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

package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
)

func TestCreateScenarioRun_ResiliencyScore(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	baselineValue := 9.0

	tests := []struct {
		name              string
		graphRun          *krknv1alpha1.KrknGraphRun
		nodeID            string
		setupFiles        []*corev1.ConfigMap
		expectEnvVars     map[string]string
		expectNoEnvVars   []string
	}{
		{
			name: "resiliency score enabled without mount path - only RESILIENCY_SCORE",
			graphRun: &krknv1alpha1.KrknGraphRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-graphrun-1",
					Namespace: "default",
				},
				Spec: krknv1alpha1.KrknGraphRunSpec{
					Graph: map[string]krknv1alpha1.GraphScenarioNode{
						"node-1": {
							Name:  "test-scenario",
							Image: "quay.io/krkn-chaos/krkn-hub:dummy-scenario",
						},
					},
					TargetRequestID:         "test-target",
					TargetClusters:          map[string][]string{"local": {"cluster1"}},
					ResiliencyScoreEnabled:  true,
					ResiliencyScoreBaseline: &baselineValue,
				},
			},
			nodeID: "node-1",
			expectEnvVars: map[string]string{
				"RESILIENCY_SCORE": "true",
			},
			expectNoEnvVars: []string{"RESILIENCY_FILE"},
		},
		{
			name: "resiliency score enabled with mount path but no matching file - only RESILIENCY_SCORE",
			graphRun: &krknv1alpha1.KrknGraphRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-graphrun-2",
					Namespace: "default",
				},
				Spec: krknv1alpha1.KrknGraphRunSpec{
					Graph: map[string]krknv1alpha1.GraphScenarioNode{
						"node-1": {
							Name:  "test-scenario",
							Image: "quay.io/krkn-chaos/krkn-hub:dummy-scenario",
							Volumes: map[string]string{
								"uuid-1": "/config/scenario.yaml",
							},
						},
					},
					TargetRequestID:         "test-target",
					TargetClusters:          map[string][]string{"local": {"cluster1"}},
					ResiliencyScoreEnabled:  true,
					ResiliencyMountPath:     "/etc/kraken/metrics.yaml",
					ResiliencyScoreBaseline: &baselineValue,
				},
			},
			nodeID: "node-1",
			setupFiles: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "file-uuid-1",
						Namespace: "default",
						Labels: map[string]string{
							files.AppNameLabel:      "krkn-operator",
							files.AppComponentLabel: "file",
							files.FileIDLabel:       "uuid-1",
						},
					},
					Data: map[string]string{
						"scenario.yaml": "key: value",
					},
				},
			},
			expectEnvVars: map[string]string{
				"RESILIENCY_SCORE": "true",
			},
			expectNoEnvVars: []string{"RESILIENCY_FILE"},
		},
		{
			name: "resiliency score enabled with matching mount path - both env vars",
			graphRun: &krknv1alpha1.KrknGraphRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-graphrun-3",
					Namespace: "default",
				},
				Spec: krknv1alpha1.KrknGraphRunSpec{
					Graph: map[string]krknv1alpha1.GraphScenarioNode{
						"node-1": {
							Name:  "test-scenario",
							Image: "quay.io/krkn-chaos/krkn-hub:dummy-scenario",
							Volumes: map[string]string{
								"uuid-metrics": "/etc/kraken/metrics.yaml",
								"uuid-config":  "/config/scenario.yaml",
							},
						},
					},
					TargetRequestID:         "test-target",
					TargetClusters:          map[string][]string{"local": {"cluster1"}},
					ResiliencyScoreEnabled:  true,
					ResiliencyMountPath:     "/etc/kraken/metrics.yaml",
					ResiliencyScoreBaseline: &baselineValue,
				},
			},
			nodeID: "node-1",
			setupFiles: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "file-uuid-metrics",
						Namespace: "default",
						Labels: map[string]string{
							files.AppNameLabel:      "krkn-operator",
							files.AppComponentLabel: "file",
							files.FileIDLabel:       "uuid-metrics",
						},
					},
					Data: map[string]string{
						"metrics.yaml": "{}",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "file-uuid-config",
						Namespace: "default",
						Labels: map[string]string{
							files.AppNameLabel:      "krkn-operator",
							files.AppComponentLabel: "file",
							files.FileIDLabel:       "uuid-config",
						},
					},
					Data: map[string]string{
						"scenario.yaml": "key: value",
					},
				},
			},
			expectEnvVars: map[string]string{
				"RESILIENCY_SCORE": "true",
				"RESILIENCY_FILE":  "/etc/kraken/metrics.yaml",
			},
		},
		{
			name: "resiliency score disabled - no env vars",
			graphRun: &krknv1alpha1.KrknGraphRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-graphrun-4",
					Namespace: "default",
				},
				Spec: krknv1alpha1.KrknGraphRunSpec{
					Graph: map[string]krknv1alpha1.GraphScenarioNode{
						"node-1": {
							Name:  "test-scenario",
							Image: "quay.io/krkn-chaos/krkn-hub:dummy-scenario",
						},
					},
					TargetRequestID:        "test-target",
					TargetClusters:         map[string][]string{"local": {"cluster1"}},
					ResiliencyScoreEnabled: false,
				},
			},
			nodeID: "node-1",
			expectNoEnvVars: []string{"RESILIENCY_SCORE", "RESILIENCY_FILE"},
		},
		{
			name: "multiple nodes with different file UUIDs but same mount path",
			graphRun: &krknv1alpha1.KrknGraphRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-graphrun-5",
					Namespace: "default",
				},
				Spec: krknv1alpha1.KrknGraphRunSpec{
					Graph: map[string]krknv1alpha1.GraphScenarioNode{
						"node-1": {
							Name:  "test-scenario",
							Image: "quay.io/krkn-chaos/krkn-hub:dummy-scenario",
							Volumes: map[string]string{
								"uuid-A": "/etc/kraken/metrics.yaml",
							},
						},
						"node-2": {
							Name:  "test-scenario",
							Image: "quay.io/krkn-chaos/krkn-hub:dummy-scenario",
							Volumes: map[string]string{
								"uuid-B": "/etc/kraken/metrics.yaml",
							},
						},
					},
					TargetRequestID:         "test-target",
					TargetClusters:          map[string][]string{"local": {"cluster1"}},
					ResiliencyScoreEnabled:  true,
					ResiliencyMountPath:     "/etc/kraken/metrics.yaml",
					ResiliencyScoreBaseline: &baselineValue,
				},
			},
			nodeID: "node-1",
			setupFiles: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "file-uuid-A",
						Namespace: "default",
						Labels: map[string]string{
							files.AppNameLabel:      "krkn-operator",
							files.AppComponentLabel: "file",
							files.FileIDLabel:       "uuid-A",
						},
					},
					Data: map[string]string{
						"metrics.yaml": "{}",
					},
				},
			},
			expectEnvVars: map[string]string{
				"RESILIENCY_SCORE": "true",
				"RESILIENCY_FILE":  "/etc/kraken/metrics.yaml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a deep copy of the graphRun to avoid test pollution
			graphRun := tt.graphRun.DeepCopy()

			// Initialize node status to prevent "node status not found" error
			graphRun.Status.NodeStatuses = []krknv1alpha1.NodeStatus{
				{
					NodeID:   tt.nodeID,
					NodeName: graphRun.Spec.Graph[tt.nodeID].Name,
					Phase:    "Pending",
				},
			}

			// Setup fake client
			objects := []runtime.Object{graphRun}
			for _, file := range tt.setupFiles {
				objects = append(objects, file)
			}
			fakeClient := fakeclient.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objects...).
				Build()

			// Create reconciler
			reconciler := &KrknGraphRunReconciler{
				Client:    fakeClient,
				Scheme:    scheme,
				Namespace: "default",
			}

			// Call createScenarioRun
			_, err := reconciler.createScenarioRun(context.Background(), graphRun, tt.nodeID)
			if err != nil {
				t.Fatalf("createScenarioRun failed: %v", err)
			}

			// Fetch created ScenarioRun
			var scenarioRunList krknv1alpha1.KrknScenarioRunList
			if err := fakeClient.List(context.Background(), &scenarioRunList); err != nil {
				t.Fatalf("failed to list ScenarioRuns: %v", err)
			}

			if len(scenarioRunList.Items) != 1 {
				t.Fatalf("expected 1 ScenarioRun, got %d", len(scenarioRunList.Items))
			}

			scenarioRun := &scenarioRunList.Items[0]

			// Validate expected env vars are present
			for key, expectedValue := range tt.expectEnvVars {
				actualValue, found := scenarioRun.Spec.Environment[key]
				if !found {
					t.Errorf("expected env var %s to be set", key)
				} else if actualValue != expectedValue {
					t.Errorf("env var %s: expected '%s', got '%s'", key, expectedValue, actualValue)
				}
			}

			// Validate env vars that should NOT be present
			for _, key := range tt.expectNoEnvVars {
				if value, found := scenarioRun.Spec.Environment[key]; found {
					t.Errorf("env var %s should not be set, but got value: '%s'", key, value)
				}
			}
		})
	}
}
