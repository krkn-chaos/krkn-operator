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
*/

package api

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// compareCategoryRunConfigurations reports behavior-affecting configuration
// differences between two run resources. It deliberately compares typed fields
// instead of serialized JSON, so map and graph-node iteration order is irrelevant.
func compareCategoryRunConfigurations(left, right client.Object) []string {
	switch leftRun := left.(type) {
	case *krknv1alpha1.KrknScenarioRun:
		rightRun, ok := right.(*krknv1alpha1.KrknScenarioRun)
		if !ok {
			return []string{"type"}
		}
		return compareScenarioRunConfigurations(leftRun.Spec, rightRun.Spec)
	case *krknv1alpha1.KrknGraphRun:
		rightRun, ok := right.(*krknv1alpha1.KrknGraphRun)
		if !ok {
			return []string{"type"}
		}
		return compareGraphRunConfigurations(leftRun.Spec, rightRun.Spec)
	default:
		return []string{"type"}
	}
}

func compareScenarioRunConfigurations(left, right krknv1alpha1.KrknScenarioRunSpec) []string {
	differences := make([]string, 0)
	if scenarioIdentity(left.Scenario.Name, left.ScenarioName) != scenarioIdentity(right.Scenario.Name, right.ScenarioName) {
		differences = append(differences, "scenario.name")
	}
	if left.KubeconfigPath != right.KubeconfigPath {
		differences = append(differences, "kubeconfigPath")
	}
	if left.MaxRetries != right.MaxRetries {
		differences = append(differences, "maxRetries")
	}
	if left.RetryBackoff != right.RetryBackoff {
		differences = append(differences, "retryBackoff")
	}
	if left.RetryDelay != right.RetryDelay {
		differences = append(differences, "retryDelay")
	}
	differences = append(differences, compareStringMaps("environment", left.Environment, right.Environment)...)
	if !sameScenarioFiles(left.Files, right.Files) {
		differences = append(differences, "files")
	}
	sort.Strings(differences)
	return differences
}

func compareGraphRunConfigurations(left, right krknv1alpha1.KrknGraphRunSpec) []string {
	differences := make([]string, 0)
	if left.MaxRetries != right.MaxRetries {
		differences = append(differences, "maxRetries")
	}
	if left.ResiliencyMountPath != right.ResiliencyMountPath {
		differences = append(differences, "resiliencyMountPath")
	}
	differences = append(differences, compareGraphConfigurations(left.Graph, right.Graph)...)
	sort.Strings(differences)
	return differences
}

func compareGraphConfigurations(left, right map[string]krknv1alpha1.GraphScenarioNode) []string {
	if sameGraphNodeIDs(left, right) {
		differences := compareGraphConfigurationsWithStableNodeIDs(left, right)
		if len(differences) == 0 || sameGraphConfigurationIgnoringNodeIDs(left, right) {
			return nil
		}
		return differences
	}
	// A replay may regenerate node IDs. Preserve the graph behavior and
	// dependency shape without treating those identifiers as configuration.
	if sameGraphConfigurationIgnoringNodeIDs(left, right) {
		return nil
	}
	return []string{"graph"}
}

func compareGraphConfigurationsWithStableNodeIDs(
	left, right map[string]krknv1alpha1.GraphScenarioNode,
) []string {
	differences := make([]string, 0)
	nodeIDs := make([]string, 0, len(left))
	for nodeID := range left {
		nodeIDs = append(nodeIDs, nodeID)
	}
	sort.Strings(nodeIDs)
	for _, nodeID := range nodeIDs {
		leftNode, leftExists := left[nodeID]
		rightNode, rightExists := right[nodeID]
		if !leftExists || !rightExists {
			return []string{fmt.Sprintf("graph.%s", nodeID)}
		}
		differences = append(differences, compareGraphNodeConfigurations(nodeID, leftNode, rightNode)...)
	}
	sort.Strings(differences)
	return differences
}

func sameGraphNodeIDs(left, right map[string]krknv1alpha1.GraphScenarioNode) bool {
	if len(left) != len(right) {
		return false
	}
	for nodeID := range left {
		if _, exists := right[nodeID]; !exists {
			return false
		}
	}
	return true
}

// sameGraphConfigurationIgnoringNodeIDs compares a graph as a forest of
// behavior-labeled nodes. DependsOn references are translated into parent/
// child edges, so equivalent graphs can use different node IDs while changed
// dependencies still produce a different structure.
func sameGraphConfigurationIgnoringNodeIDs(
	left, right map[string]krknv1alpha1.GraphScenarioNode,
) bool {
	if len(left) != len(right) {
		return false
	}
	leftSignature, leftValid := graphConfigurationSignature(left)
	rightSignature, rightValid := graphConfigurationSignature(right)
	return leftValid && rightValid && leftSignature == rightSignature
}

func graphConfigurationSignature(graph map[string]krknv1alpha1.GraphScenarioNode) (string, bool) {
	children := make(map[string][]string, len(graph))
	roots := make([]string, 0, len(graph))
	for nodeID, node := range graph {
		parentID := optionalStringValue(node.DependsOn)
		if parentID == "" {
			roots = append(roots, nodeID)
			continue
		}
		if _, exists := graph[parentID]; !exists {
			return "", false
		}
		children[parentID] = append(children[parentID], nodeID)
	}

	states := make(map[string]uint8, len(graph))
	memo := make(map[string]string, len(graph))
	rootSignatures := make([]string, 0, len(roots))
	for _, rootID := range roots {
		signature, valid := graphNodeConfigurationSignature(rootID, graph, children, states, memo)
		if !valid {
			return "", false
		}
		rootSignatures = append(rootSignatures, signature)
	}
	// Nodes left out of all roots belong to a dependency cycle.
	if len(memo) != len(graph) {
		return "", false
	}
	sort.Strings(rootSignatures)

	var signature strings.Builder
	fmt.Fprintf(&signature, "%d:", len(rootSignatures))
	for _, root := range rootSignatures {
		appendGraphSignaturePart(&signature, root)
	}
	return signature.String(), true
}

func graphNodeConfigurationSignature(
	nodeID string,
	graph map[string]krknv1alpha1.GraphScenarioNode,
	children map[string][]string,
	states map[string]uint8,
	memo map[string]string,
) (string, bool) {
	if signature, exists := memo[nodeID]; exists {
		return signature, true
	}
	if states[nodeID] == 1 {
		return "", false
	}
	node, exists := graph[nodeID]
	if !exists {
		return "", false
	}
	states[nodeID] = 1

	childSignatures := make([]string, 0, len(children[nodeID]))
	for _, childID := range children[nodeID] {
		signature, valid := graphNodeConfigurationSignature(childID, graph, children, states, memo)
		if !valid {
			return "", false
		}
		childSignatures = append(childSignatures, signature)
	}
	sort.Strings(childSignatures)

	var signature strings.Builder
	appendGraphSignaturePart(&signature, scenarioIdentity(node.Scenario.Name, node.Name))
	environmentKeys := make([]string, 0, len(node.Env))
	for key := range node.Env {
		environmentKeys = append(environmentKeys, key)
	}
	sort.Strings(environmentKeys)
	fmt.Fprintf(&signature, "%d:", len(environmentKeys))
	for _, key := range environmentKeys {
		appendGraphSignaturePart(&signature, key)
		appendGraphSignaturePart(&signature, node.Env[key])
	}
	volumePaths := make([]string, 0, len(node.Volumes))
	for _, mountPath := range node.Volumes {
		volumePaths = append(volumePaths, mountPath)
	}
	sort.Strings(volumePaths)
	fmt.Fprintf(&signature, "%d:", len(volumePaths))
	for _, mountPath := range volumePaths {
		appendGraphSignaturePart(&signature, mountPath)
	}
	fmt.Fprintf(&signature, "%d:", len(childSignatures))
	for _, child := range childSignatures {
		appendGraphSignaturePart(&signature, child)
	}

	states[nodeID] = 2
	memo[nodeID] = signature.String()
	return memo[nodeID], true
}

func appendGraphSignaturePart(signature *strings.Builder, part string) {
	fmt.Fprintf(signature, "%d:%s", len(part), part)
}

func compareGraphNodeConfigurations(nodeID string, left, right krknv1alpha1.GraphScenarioNode) []string {
	path := "graph." + nodeID
	differences := make([]string, 0)
	if scenarioIdentity(left.Scenario.Name, left.Name) != scenarioIdentity(right.Scenario.Name, right.Name) {
		differences = append(differences, path+".scenario.name")
	}
	differences = append(differences, compareStringMaps(path+".env", left.Env, right.Env)...)
	if !sameGraphVolumeMounts(left.Volumes, right.Volumes) {
		differences = append(differences, path+".volumes")
	}
	if optionalStringValue(left.DependsOn) != optionalStringValue(right.DependsOn) {
		differences = append(differences, path+".depends_on")
	}
	return differences
}

// sameGraphVolumeMounts compares mount paths as a multiset. Graph volume map
// keys are file IDs, which identify stored files but do not describe scenario
// behavior and therefore must not split otherwise equivalent configurations.
func sameGraphVolumeMounts(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	leftPaths := make([]string, 0, len(left))
	for _, mountPath := range left {
		leftPaths = append(leftPaths, mountPath)
	}
	rightPaths := make([]string, 0, len(right))
	for _, mountPath := range right {
		rightPaths = append(rightPaths, mountPath)
	}
	sort.Strings(leftPaths)
	sort.Strings(rightPaths)
	return slices.Equal(leftPaths, rightPaths)
}

func compareStringMaps(path string, left, right map[string]string) []string {
	keys := make(map[string]struct{}, len(left)+len(right))
	for key := range left {
		keys[key] = struct{}{}
	}
	for key := range right {
		keys[key] = struct{}{}
	}
	sortedKeys := make([]string, 0, len(keys))
	for key := range keys {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	differences := make([]string, 0)
	for _, key := range sortedKeys {
		leftValue, leftExists := left[key]
		rightValue, rightExists := right[key]
		if leftExists != rightExists || leftValue != rightValue {
			differences = append(differences, path+"."+key)
		}
	}
	return differences
}

type scenarioFileBehavior struct {
	name      string
	content   string
	mountPath string
}

func sameScenarioFiles(left, right []krknv1alpha1.FileMount) bool {
	if len(left) != len(right) {
		return false
	}
	counts := make(map[scenarioFileBehavior]int, len(left))
	for _, file := range left {
		counts[scenarioFileBehavior{name: file.Name, content: file.Content, mountPath: file.MountPath}]++
	}
	for _, file := range right {
		key := scenarioFileBehavior{name: file.Name, content: file.Content, mountPath: file.MountPath}
		if counts[key] == 0 {
			return false
		}
		counts[key]--
	}
	return true
}

func scenarioIdentity(name, legacyName string) string {
	if name != "" {
		return name
	}
	return legacyName
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
