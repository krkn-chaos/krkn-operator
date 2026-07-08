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

// Package graph provides graph orchestration functionality for resolving and executing
// scenario dependency graphs in the krkn operator.
package graph

import (
	"fmt"
	"strings"

	v1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krknctl/pkg/dependencygraph"
)

// ResolveGraph takes a scenario graph and resolves it into topological levels
// using krknctl's dependency graph resolution.
// Returns a 2D array where each inner array represents a level of nodes that can run in parallel.
// Example: [["node1", "node2"], ["node3"], ["node4", "node5"]]
func ResolveGraph(scenarioGraph map[string]v1alpha1.GraphScenarioNode) ([][]string, error) {
	if len(scenarioGraph) == 0 {
		return nil, fmt.Errorf("scenario graph is empty")
	}

	// Filter out metadata nodes (starting with '_')
	filteredGraph := make(map[string]v1alpha1.GraphScenarioNode)
	for nodeID, node := range scenarioGraph {
		if !strings.HasPrefix(nodeID, "_") {
			filteredGraph[nodeID] = node
		}
	}

	if len(filteredGraph) == 0 {
		return nil, fmt.Errorf("scenario graph contains no valid nodes (all nodes are metadata)")
	}

	// Validate that all DependsOn references exist in the graph
	// This prevents "phantom nodes" from appearing in resolved levels
	for nodeID, node := range filteredGraph {
		if node.DependsOn != nil && *node.DependsOn != "" {
			dependsOnID := *node.DependsOn

			// Check if referenced node exists in filtered graph
			if _, exists := filteredGraph[dependsOnID]; !exists {
				// Check if it's a metadata node (which we filtered out)
				if strings.HasPrefix(dependsOnID, "_") {
					return nil, fmt.Errorf("node '%s' depends on metadata node '%s': dependencies on metadata nodes (starting with '_') are not allowed", nodeID, dependsOnID)
				}
				return nil, fmt.Errorf("node '%s' depends on non-existent node '%s': referenced node does not exist in graph", nodeID, dependsOnID)
			}

			// Check for self-reference
			if dependsOnID == nodeID {
				return nil, fmt.Errorf("node '%s' has self-referencing dependency: a node cannot depend on itself", nodeID)
			}
		}
	}

	// Convert to krknctl format for graph resolution
	krknctlScenarioSet := v1alpha1.ToKrknctlScenarioSet(filteredGraph)

	// Check if all nodes are root nodes (no dependencies)
	// If so, return them all in a single level
	allNodesAreRoots := true
	for _, node := range krknctlScenarioSet {
		if node.Parent != nil {
			allNodesAreRoots = false
			break
		}
	}

	if allNodesAreRoots {
		// All nodes can run in parallel
		level := make([]string, 0, len(filteredGraph))
		for nodeID := range filteredGraph {
			level = append(level, nodeID)
		}
		return [][]string{level}, nil
	}

	// Convert ScenarioSet to map[string]ParentProvider for NewGraphFromNodes
	nodes := make(map[string]dependencygraph.ParentProvider)
	for nodeID, node := range krknctlScenarioSet {
		// Create a copy of the node to avoid pointer issues
		nodeCopy := node
		nodes[nodeID] = &nodeCopy
	}

	// Build dependency graph using krknctl
	// This will return an error if there are cycles
	graph, err := dependencygraph.NewGraphFromNodes(nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Resolve the graph into topological levels
	// TopoSortedLayers returns only [][]string, no error
	levels := graph.TopoSortedLayers()

	return levels, nil
}

// MapScenarioNodeToScenarioRunSpec maps a GraphScenarioNode to a KrknScenarioRunSpec
// This function creates the spec for a KrknScenarioRun that will execute a single node
// from the dependency graph.
func MapScenarioNodeToScenarioRunSpec(
	node v1alpha1.GraphScenarioNode,
	scenarioName string,
	targetRequestID string,
	targetClusters map[string][]string,
	ownerUserID string,
) (v1alpha1.KrknScenarioRunSpec, error) {
	// Validate required fields
	if node.Image == "" {
		return v1alpha1.KrknScenarioRunSpec{}, fmt.Errorf("scenario node image is required")
	}
	if node.Name == "" {
		return v1alpha1.KrknScenarioRunSpec{}, fmt.Errorf("scenario node name is required")
	}
	if scenarioName == "" {
		return v1alpha1.KrknScenarioRunSpec{}, fmt.Errorf("scenario name is required")
	}
	if targetRequestID == "" {
		return v1alpha1.KrknScenarioRunSpec{}, fmt.Errorf("target request ID is required")
	}
	if len(targetClusters) == 0 {
		return v1alpha1.KrknScenarioRunSpec{}, fmt.Errorf("target clusters are required")
	}

	// Create the scenario run spec
	spec := v1alpha1.KrknScenarioRunSpec{
		TargetRequestID: targetRequestID,
		TargetClusters:  targetClusters,
		ScenarioName:    scenarioName,
		ScenarioImage:   node.Image,
		OwnerUserID:     ownerUserID,
	}

	// Map environment variables from node
	spec.Environment = make(map[string]string)
	for k, v := range node.Env {
		spec.Environment[k] = v
	}

	// Add RESILIENCY_SCORE=true for all graph run scenarios
	// This enables resiliency scoring in the krkn-hub scenarios
	spec.Environment["RESILIENCY_SCORE"] = "true"

	// TODO: Implement volumes mapping when FileMounts support is ready
	// The node.Volumes field contains volume mount specifications,
	// but the current KrknScenarioRunSpec uses Files []FileMount instead.
	// We need to define the mapping strategy:
	// - Option 1: Convert volume mounts to file mounts (limited use case)
	// - Option 2: Extend KrknScenarioRunSpec to support volume mounts directly
	// - Option 3: Use a different mechanism for volume mounting in graph runs
	// For now, volumes are skipped and will be addressed in a future iteration
	// based on the chosen approach for volume mounting.

	return spec, nil
}
