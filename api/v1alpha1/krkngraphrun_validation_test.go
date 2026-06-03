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

package v1alpha1

import (
	"fmt"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestKrknGraphRun_ValidateGraph(t *testing.T) {
	tests := []struct {
		name    string
		graph   map[string]GraphScenarioNode
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid graph with simple node IDs",
			graph: map[string]GraphScenarioNode{
				"node1": {Name: "scenario1", Image: "image1"},
				"node2": {Name: "scenario2", Image: "image2"},
			},
			wantErr: false,
		},
		{
			name: "valid graph with complex node IDs",
			graph: map[string]GraphScenarioNode{
				"pod-disruption":       {Name: "scenario1", Image: "image1"},
				"network_latency":      {Name: "scenario2", Image: "image2"},
				"container.cpu.stress": {Name: "scenario3", Image: "image3"},
			},
			wantErr: false,
		},
		{
			name:    "empty graph",
			graph:   map[string]GraphScenarioNode{},
			wantErr: true,
			errMsg:  "graph cannot be empty",
		},
		{
			name: "node ID collision - special characters",
			graph: map[string]GraphScenarioNode{
				"node@one": {Name: "scenario1", Image: "image1"},
				"node#one": {Name: "scenario2", Image: "image2"},
			},
			wantErr: true,
			errMsg:  "node ID collision detected",
		},
		{
			name: "node ID collision - case sensitivity",
			graph: map[string]GraphScenarioNode{
				"NodeOne": {Name: "scenario1", Image: "image1"},
				"nodeone": {Name: "scenario2", Image: "image2"},
			},
			wantErr: true,
			errMsg:  "node ID collision detected",
		},
		{
			name: "node ID collision - spaces and hyphens",
			graph: map[string]GraphScenarioNode{
				"node one": {Name: "scenario1", Image: "image1"},
				"node-one": {Name: "scenario2", Image: "image2"},
			},
			wantErr: true,
			errMsg:  "node ID collision detected",
		},
		{
			name: "node ID too long",
			graph: map[string]GraphScenarioNode{
				strings.Repeat("a", 254): {Name: "scenario1", Image: "image1"},
			},
			wantErr: true,
			errMsg:  "exceeds maximum length of 253 characters",
		},
		{
			name: "node ID sanitizes to invalid default",
			graph: map[string]GraphScenarioNode{
				"---": {Name: "scenario1", Image: "image1"},
			},
			wantErr: true,
			errMsg:  "sanitizes to invalid value",
		},
		{
			name: "metadata nodes ignored in validation",
			graph: map[string]GraphScenarioNode{
				"_metadata":  {Name: "metadata", Image: "image1"},
				"_config":    {Name: "config", Image: "image2"},
				"valid-node": {Name: "scenario", Image: "image3"},
			},
			wantErr: false,
		},
		{
			name: "multiple node ID collisions",
			graph: map[string]GraphScenarioNode{
				"node@one":  {Name: "scenario1", Image: "image1"},
				"node.one":  {Name: "scenario2", Image: "image2"},
				"node_one":  {Name: "scenario3", Image: "image3"},
				"node one":  {Name: "scenario4", Image: "image4"},
				"node-two":  {Name: "scenario5", Image: "image5"},
				"node@two":  {Name: "scenario6", Image: "image6"},
				"valid-one": {Name: "scenario7", Image: "image7"},
			},
			wantErr: true,
			errMsg:  "node ID collision detected",
		},
		{
			name: "truncation causing collision",
			graph: map[string]GraphScenarioNode{
				"very-long-node-id-that-will-be-truncated-at-exactly-this-point-a": {Name: "scenario1", Image: "image1"},
				"very-long-node-id-that-will-be-truncated-at-exactly-this-point-b": {Name: "scenario2", Image: "image2"},
			},
			wantErr: true,
			errMsg:  "node ID collision detected",
		},
		{
			name: "valid long node IDs that don't collide after truncation",
			graph: map[string]GraphScenarioNode{
				"short-node-id-a": {Name: "scenario1", Image: "image1"},
				"short-node-id-b": {Name: "scenario2", Image: "image2"},
				"medium-length-node-id-that-is-different-here-a":  {Name: "scenario3", Image: "image3"},
				"medium-length-node-id-that-is-different-there-b": {Name: "scenario4", Image: "image4"},
			},
			wantErr: false,
		},
		{
			name: "mixed valid and metadata nodes",
			graph: map[string]GraphScenarioNode{
				"_config":   {Name: "config", Image: "image1"},
				"node-one":  {Name: "scenario1", Image: "image2"},
				"_metadata": {Name: "metadata", Image: "image3"},
				"node-two":  {Name: "scenario2", Image: "image4"},
			},
			wantErr: false,
		},
		{
			name: "phantom dependency - missing node",
			graph: map[string]GraphScenarioNode{
				"node1": {Name: "scenario1", Image: "image1", DependsOn: stringPtr("phantom-node")},
				"node2": {Name: "scenario2", Image: "image2"},
			},
			wantErr: true,
			errMsg:  "invalid DependsOn reference 'phantom-node': referenced node does not exist",
		},
		{
			name: "dependency on metadata node",
			graph: map[string]GraphScenarioNode{
				"_config": {Name: "config", Image: "image1"},
				"node1":   {Name: "scenario1", Image: "image2", DependsOn: stringPtr("_config")},
			},
			wantErr: true,
			errMsg:  "cannot depend on metadata nodes",
		},
		{
			name: "self-referencing dependency",
			graph: map[string]GraphScenarioNode{
				"node1": {Name: "scenario1", Image: "image1", DependsOn: stringPtr("node1")},
			},
			wantErr: true,
			errMsg:  "cannot depend on itself",
		},
		{
			name: "valid dependency chain",
			graph: map[string]GraphScenarioNode{
				"node1": {Name: "scenario1", Image: "image1"},
				"node2": {Name: "scenario2", Image: "image2", DependsOn: stringPtr("node1")},
				"node3": {Name: "scenario3", Image: "image3", DependsOn: stringPtr("node2")},
			},
			wantErr: false,
		},
		{
			name: "valid parallel nodes with common dependency",
			graph: map[string]GraphScenarioNode{
				"root":   {Name: "root", Image: "image1"},
				"child1": {Name: "child1", Image: "image2", DependsOn: stringPtr("root")},
				"child2": {Name: "child2", Image: "image3", DependsOn: stringPtr("root")},
			},
			wantErr: false,
		},
		{
			name: "phantom dependency in chain",
			graph: map[string]GraphScenarioNode{
				"node1": {Name: "scenario1", Image: "image1"},
				"node2": {Name: "scenario2", Image: "image2", DependsOn: stringPtr("node1")},
				"node3": {Name: "scenario3", Image: "image3", DependsOn: stringPtr("missing-node")},
			},
			wantErr: true,
			errMsg:  "invalid DependsOn reference 'missing-node': referenced node does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graphRun := &KrknGraphRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-graphrun",
					Namespace: "default",
				},
				Spec: KrknGraphRunSpec{
					Graph:           tt.graph,
					TargetRequestID: "test-target",
					TargetClusters: map[string][]string{
						"provider1": {"cluster1"},
					},
					OwnerUserID: "testuser",
				},
			}

			err := graphRun.ValidateGraph()

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateGraph() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateGraph() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateGraph() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestSanitizeNodeIDForKubernetes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid lowercase",
			input:    "node1",
			expected: "node1",
		},
		{
			name:     "uppercase to lowercase",
			input:    "Node1",
			expected: "node1",
		},
		{
			name:     "special characters replaced",
			input:    "node@one",
			expected: "node-one",
		},
		{
			name:     "spaces replaced",
			input:    "node one",
			expected: "node-one",
		},
		{
			name:     "multiple special chars",
			input:    "node@one#two",
			expected: "node-one-two",
		},
		{
			name:     "truncated to 63 chars",
			input:    "very-long-node-id-that-exceeds-the-maximum-kubernetes-label-value-length-of-sixty-three-characters",
			expected: "very-long-node-id-that-exceeds-the-maximum-kubernetes-label-val",
		},
		{
			name:     "leading hyphens trimmed",
			input:    "-node",
			expected: "node",
		},
		{
			name:     "trailing hyphens trimmed",
			input:    "node-",
			expected: "node",
		},
		{
			name:     "only invalid chars",
			input:    "---",
			expected: "node",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeNodeIDForKubernetes(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeNodeIDForKubernetes(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCollisionDetection(t *testing.T) {
	// Test that the collision detection correctly identifies conflicts
	collisionSets := [][]string{
		{"node@one", "node#one", "node one"},       // All become "node-one"
		{"NodeOne", "nodeone"},                     // Case insensitivity
		{"my@node", "my#node"},                     // @ and # both become -
		{"pod@@@kill", "pod###kill", "pod   kill"}, // Multiple separators become -
	}

	for i, set := range collisionSets {
		t.Run(fmt.Sprintf("collision_set_%d", i), func(t *testing.T) {
			graph := make(map[string]GraphScenarioNode)
			for j, nodeID := range set {
				graph[nodeID] = GraphScenarioNode{
					Name:  fmt.Sprintf("scenario%d", j),
					Image: fmt.Sprintf("image%d", j),
				}
			}

			graphRun := &KrknGraphRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-graphrun",
					Namespace: "default",
				},
				Spec: KrknGraphRunSpec{
					Graph:           graph,
					TargetRequestID: "test-target",
					TargetClusters: map[string][]string{
						"provider1": {"cluster1"},
					},
					OwnerUserID: "testuser",
				},
			}

			err := graphRun.ValidateGraph()
			if err == nil {
				t.Errorf("Expected collision error for set %v, got nil", set)
				return
			}
			if !strings.Contains(err.Error(), "collision") {
				t.Errorf("Expected collision error, got: %v", err)
			}
		})
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
