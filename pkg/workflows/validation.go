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

// Package workflows provides types and utilities for workflow template management.
// Workflows are stored as files with filePurpose="workflow-template" and validated
// against the graph execution rules (no cycles, valid dependencies).
package workflows

import (
	"encoding/json"
	"fmt"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/graph"
)

// ValidateWorkflowGraph validates workflow graph structure using existing graph.ResolveGraph.
// This ensures the workflow graph is a valid DAG (Directed Acyclic Graph) with:
// - All depends_on references point to existing nodes
// - No circular dependencies
// - No self-references
// - At least one root node (no all-dependent graph)
//
// Returns an error if the graph is invalid, nil if valid.
func ValidateWorkflowGraph(scenarioGraph map[string]krknv1alpha1.GraphScenarioNode) error {
	if len(scenarioGraph) == 0 {
		return fmt.Errorf("workflow graph cannot be empty")
	}

	// Reuse existing validation from graph package
	// ResolveGraph returns error if graph is invalid (cycles, bad refs, etc.)
	_, err := graph.ResolveGraph(scenarioGraph)
	if err != nil {
		return fmt.Errorf("invalid workflow graph: %w", err)
	}

	return nil
}

// ToFileContent marshals a workflow graph to JSON string for file storage.
// The returned JSON is indented for readability.
func ToFileContent(scenarioGraph map[string]krknv1alpha1.GraphScenarioNode) (string, error) {
	bytes, err := json.MarshalIndent(scenarioGraph, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal workflow graph: %w", err)
	}
	return string(bytes), nil
}

// FromFileContent parses a workflow graph from JSON file content.
// Returns the parsed graph or an error if the JSON is invalid.
func FromFileContent(content string) (map[string]krknv1alpha1.GraphScenarioNode, error) {
	var scenarioGraph map[string]krknv1alpha1.GraphScenarioNode
	if err := json.Unmarshal([]byte(content), &scenarioGraph); err != nil {
		return nil, fmt.Errorf("failed to parse workflow graph JSON: %w", err)
	}
	return scenarioGraph, nil
}

// StudioLayoutToJSON marshals studioLayout to JSON string for ConfigMap storage.
// Returns empty string if studioLayout is nil or empty.
func StudioLayoutToJSON(studioLayout map[string]interface{}) (string, error) {
	if len(studioLayout) == 0 {
		return "", nil
	}
	bytes, err := json.Marshal(studioLayout)
	if err != nil {
		return "", fmt.Errorf("failed to marshal studioLayout: %w", err)
	}
	return string(bytes), nil
}

// StudioLayoutFromJSON parses studioLayout from JSON string.
// Returns nil if content is empty, error if JSON is invalid.
func StudioLayoutFromJSON(content string) (map[string]interface{}, error) {
	if content == "" {
		return nil, nil
	}
	var studioLayout map[string]interface{}
	if err := json.Unmarshal([]byte(content), &studioLayout); err != nil {
		return nil, fmt.Errorf("failed to parse studioLayout JSON: %w", err)
	}
	return studioLayout, nil
}
