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

package workflows

import (
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func TestValidateWorkflowGraph(t *testing.T) {
	tests := []struct {
		name        string
		graph       map[string]krknv1alpha1.GraphScenarioNode
		expectError bool
	}{
		{
			name:        "empty graph",
			graph:       map[string]krknv1alpha1.GraphScenarioNode{},
			expectError: true,
		},
		{
			name: "valid single node",
			graph: map[string]krknv1alpha1.GraphScenarioNode{
				"node1": {Name: "test", Image: "test:latest"},
			},
			expectError: false,
		},
		{
			name: "valid two nodes with dependency",
			graph: map[string]krknv1alpha1.GraphScenarioNode{
				"node1": {Name: "test1", Image: "test:latest"},
				"node2": {Name: "test2", Image: "test:latest", DependsOn: stringPtr("node1")},
			},
			expectError: false,
		},
		{
			name: "missing depends_on reference",
			graph: map[string]krknv1alpha1.GraphScenarioNode{
				"node1": {Name: "test1", Image: "test:latest"},
				"node2": {Name: "test2", Image: "test:latest", DependsOn: stringPtr("nonexistent")},
			},
			expectError: true,
		},
		{
			name: "circular dependency",
			graph: map[string]krknv1alpha1.GraphScenarioNode{
				"node1": {Name: "test1", Image: "test:latest", DependsOn: stringPtr("node2")},
				"node2": {Name: "test2", Image: "test:latest", DependsOn: stringPtr("node1")},
			},
			expectError: true,
		},
		{
			name: "self-reference",
			graph: map[string]krknv1alpha1.GraphScenarioNode{
				"node1": {Name: "test1", Image: "test:latest", DependsOn: stringPtr("node1")},
			},
			expectError: true,
		},
		{
			name: "valid complex DAG",
			graph: map[string]krknv1alpha1.GraphScenarioNode{
				"node1": {Name: "test1", Image: "test:latest"},
				"node2": {Name: "test2", Image: "test:latest"},
				"node3": {Name: "test3", Image: "test:latest", DependsOn: stringPtr("node1")},
				"node4": {Name: "test4", Image: "test:latest", DependsOn: stringPtr("node2")},
				"node5": {Name: "test5", Image: "test:latest", DependsOn: stringPtr("node3")},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkflowGraph(tt.graph)
			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestToFileContent(t *testing.T) {
	graph := map[string]krknv1alpha1.GraphScenarioNode{
		"node1": {Name: "test1", Image: "test:latest"},
		"node2": {Name: "test2", Image: "test:latest", DependsOn: stringPtr("node1")},
	}

	content, err := ToFileContent(graph)
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	if content == "" {
		t.Error("expected non-empty content")
	}

	// Verify it's valid JSON by parsing it back
	_, err = FromFileContent(content)
	if err != nil {
		t.Errorf("expected to parse back content but got error: %v", err)
	}
}

func TestFromFileContent(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name:        "valid JSON",
			content:     `{"node1": {"name": "test", "image": "test:latest"}}`,
			expectError: false,
		},
		{
			name:        "invalid JSON",
			content:     `{invalid json}`,
			expectError: true,
		},
		{
			name:        "empty string",
			content:     ``,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromFileContent(tt.content)
			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
