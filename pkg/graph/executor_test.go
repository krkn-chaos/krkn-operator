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

package graph

import (
	"testing"

	v1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func TestResolveGraph(t *testing.T) {
	tests := []struct {
		name        string
		graph       map[string]v1alpha1.GraphScenarioNode
		wantLevels  int
		wantNodes   int // Expected number of nodes in resolved graph (0 = same as input)
		wantErr     bool
		errContains string
	}{
		{
			name: "simple linear graph",
			graph: map[string]v1alpha1.GraphScenarioNode{
				"node1": {
					Name:      "scenario-1",
					Image:     "image1:latest",
					DependsOn: nil,
				},
				"node2": {
					Name:      "scenario-2",
					Image:     "image2:latest",
					DependsOn: strPtr("node1"),
				},
			},
			wantLevels: 2,
			wantNodes:  0, // Same as input
			wantErr:    false,
		},
		{
			name: "parallel nodes",
			graph: map[string]v1alpha1.GraphScenarioNode{
				"node1": {
					Name:      "scenario-1",
					Image:     "image1:latest",
					DependsOn: nil,
				},
				"node2": {
					Name:      "scenario-2",
					Image:     "image2:latest",
					DependsOn: nil,
				},
				"node3": {
					Name:      "scenario-3",
					Image:     "image3:latest",
					DependsOn: nil,
				},
			},
			wantLevels: 1, // All nodes can run in parallel
			wantNodes:  0, // Same as input
			wantErr:    false,
		},
		{
			name: "diamond graph",
			graph: map[string]v1alpha1.GraphScenarioNode{
				"node1": {
					Name:      "scenario-1",
					Image:     "image1:latest",
					DependsOn: nil,
				},
				"node2": {
					Name:      "scenario-2",
					Image:     "image2:latest",
					DependsOn: strPtr("node1"),
				},
				"node3": {
					Name:      "scenario-3",
					Image:     "image3:latest",
					DependsOn: strPtr("node1"),
				},
				"node4": {
					Name:      "scenario-4",
					Image:     "image4:latest",
					DependsOn: strPtr("node2"), // Could also depend on node3
				},
			},
			wantLevels: 3,
			wantNodes:  0, // Same as input
			wantErr:    false,
		},
		{
			name: "cyclic dependency",
			graph: map[string]v1alpha1.GraphScenarioNode{
				"node1": {
					Name:      "scenario-1",
					Image:     "image1:latest",
					DependsOn: strPtr("node2"),
				},
				"node2": {
					Name:      "scenario-2",
					Image:     "image2:latest",
					DependsOn: strPtr("node1"),
				},
			},
			wantErr:     true,
			errContains: "circular",
		},
		{
			name: "missing parent - krknctl does not validate parent existence",
			graph: map[string]v1alpha1.GraphScenarioNode{
				"node1": {
					Name:      "scenario-1",
					Image:     "image1:latest",
					DependsOn: strPtr("nonexistent"),
				},
			},
			// krknctl's NewGraphFromNodes does not validate that the parent exists
			// It only creates the dependency edge, adding both parent and child to the graph
			// This results in a "phantom" node for the missing parent
			// The controller should validate that all nodes in the spec exist before resolution
			wantErr:    false,
			wantLevels: 2, // Level 0: "nonexistent" (phantom), Level 1: "node1"
			wantNodes:  2, // node1 + phantom "nonexistent"
		},
		{
			name:        "empty graph",
			graph:       map[string]v1alpha1.GraphScenarioNode{},
			wantErr:     true,
			errContains: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			levels, err := ResolveGraph(tt.graph)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolveGraph() expected error containing '%s', got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ResolveGraph() error = %v, want error containing %s", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ResolveGraph() unexpected error = %v", err)
				return
			}

			if len(levels) != tt.wantLevels {
				t.Errorf("ResolveGraph() returned %d levels, want %d", len(levels), tt.wantLevels)
			}

			// Verify node count
			nodesSeen := make(map[string]bool)
			for _, level := range levels {
				for _, nodeID := range level {
					nodesSeen[nodeID] = true
				}
			}

			expectedNodes := tt.wantNodes
			if expectedNodes == 0 {
				expectedNodes = len(tt.graph) // Default: same as input
			}

			if len(nodesSeen) != expectedNodes {
				t.Errorf("ResolveGraph() returned %d nodes, want %d", len(nodesSeen), expectedNodes)
			}
		})
	}
}

func TestMapScenarioNodeToScenarioRunSpec(t *testing.T) {
	tests := []struct {
		name            string
		node            v1alpha1.GraphScenarioNode
		scenarioName    string
		targetRequestID string
		targetClusters  map[string][]string
		ownerUserID     string
		wantErr         bool
		errContains     string
		validateSpec    func(*testing.T, v1alpha1.KrknScenarioRunSpec)
	}{
		{
			name: "complete node with all fields",
			node: v1alpha1.GraphScenarioNode{
				Name:  "test-scenario",
				Image: "quay.io/krkn-chaos/scenario:latest",
				Env: map[string]string{
					"KEY1": "value1",
					"KEY2": "value2",
				},
				Volumes: map[string]string{
					"/host": "/container",
				},
			},
			scenarioName:    "test-scenario",
			targetRequestID: "target-123",
			targetClusters: map[string][]string{
				"provider1": {"cluster1", "cluster2"},
			},
			ownerUserID: "user@example.com",
			wantErr:     false,
			validateSpec: func(t *testing.T, spec v1alpha1.KrknScenarioRunSpec) {
				if spec.ScenarioName != "test-scenario" {
					t.Errorf("ScenarioName = %s, want test-scenario", spec.ScenarioName)
				}
				if spec.ScenarioImage != "quay.io/krkn-chaos/scenario:latest" {
					t.Errorf("ScenarioImage = %s, want quay.io/krkn-chaos/scenario:latest", spec.ScenarioImage)
				}
				if spec.TargetRequestID != "target-123" {
					t.Errorf("TargetRequestID = %s, want target-123", spec.TargetRequestID)
				}
				if spec.OwnerUserID != "user@example.com" {
					t.Errorf("OwnerUserID = %s, want user@example.com", spec.OwnerUserID)
				}
				if len(spec.Environment) != 3 {
					t.Errorf("Environment has %d entries, want 3", len(spec.Environment))
				}
				if spec.Environment["KEY1"] != "value1" {
					t.Errorf("Environment[KEY1] = %s, want value1", spec.Environment["KEY1"])
				}
				if spec.Environment["RESILIENCY_SCORE"] != "true" {
					t.Errorf("Environment[RESILIENCY_SCORE] = %s, want true", spec.Environment["RESILIENCY_SCORE"])
				}
			},
		},
		{
			name: "minimal node without env and volumes",
			node: v1alpha1.GraphScenarioNode{
				Name:  "minimal-scenario",
				Image: "minimal:latest",
			},
			scenarioName:    "minimal-scenario",
			targetRequestID: "target-456",
			targetClusters: map[string][]string{
				"provider1": {"cluster1"},
			},
			ownerUserID: "",
			wantErr:     false,
			validateSpec: func(t *testing.T, spec v1alpha1.KrknScenarioRunSpec) {
				if spec.ScenarioName != "minimal-scenario" {
					t.Errorf("ScenarioName = %s, want minimal-scenario", spec.ScenarioName)
				}
				// Verify RESILIENCY_SCORE is always added for graph runs
				if len(spec.Environment) != 1 {
					t.Errorf("Environment has %d entries, want 1 (RESILIENCY_SCORE)", len(spec.Environment))
				}
				if spec.Environment["RESILIENCY_SCORE"] != "true" {
					t.Errorf("Environment[RESILIENCY_SCORE] = %s, want true", spec.Environment["RESILIENCY_SCORE"])
				}
			},
		},
		{
			name: "node with empty image",
			node: v1alpha1.GraphScenarioNode{
				Name:  "test-scenario",
				Image: "",
			},
			scenarioName:    "test-scenario",
			targetRequestID: "target-123",
			targetClusters: map[string][]string{
				"provider1": {"cluster1"},
			},
			ownerUserID: "user@example.com",
			wantErr:     true,
			errContains: "image is required",
		},
		{
			name: "node with empty name",
			node: v1alpha1.GraphScenarioNode{
				Name:  "",
				Image: "image:latest",
			},
			scenarioName:    "",
			targetRequestID: "target-123",
			targetClusters: map[string][]string{
				"provider1": {"cluster1"},
			},
			ownerUserID: "user@example.com",
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name: "empty target request ID",
			node: v1alpha1.GraphScenarioNode{
				Name:  "test-scenario",
				Image: "image:latest",
			},
			scenarioName:    "test-scenario",
			targetRequestID: "",
			targetClusters: map[string][]string{
				"provider1": {"cluster1"},
			},
			ownerUserID: "user@example.com",
			wantErr:     true,
			errContains: "target request ID is required",
		},
		{
			name: "empty target clusters",
			node: v1alpha1.GraphScenarioNode{
				Name:  "test-scenario",
				Image: "image:latest",
			},
			scenarioName:    "test-scenario",
			targetRequestID: "target-123",
			targetClusters:  map[string][]string{},
			ownerUserID:     "user@example.com",
			wantErr:         true,
			errContains:     "target clusters are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := MapScenarioNodeToScenarioRunSpec(
				tt.node,
				tt.scenarioName,
				tt.targetRequestID,
				tt.targetClusters,
				tt.ownerUserID,
			)

			if tt.wantErr {
				if err == nil {
					t.Errorf("MapScenarioNodeToScenarioRunSpec() expected error containing '%s', got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("MapScenarioNodeToScenarioRunSpec() error = %v, want error containing %s", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("MapScenarioNodeToScenarioRunSpec() unexpected error = %v", err)
				return
			}

			if tt.validateSpec != nil {
				tt.validateSpec(t, spec)
			}
		})
	}
}

// Helper functions

func strPtr(s string) *string {
	return &s
}

func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(substr) == 0 ||
		(len(str) > 0 && len(substr) > 0 && stringContains(str, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
