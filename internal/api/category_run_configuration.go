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

	nodeIDs := make(map[string]struct{}, len(left.Graph)+len(right.Graph))
	for nodeID := range left.Graph {
		nodeIDs[nodeID] = struct{}{}
	}
	for nodeID := range right.Graph {
		nodeIDs[nodeID] = struct{}{}
	}
	sortedNodeIDs := make([]string, 0, len(nodeIDs))
	for nodeID := range nodeIDs {
		sortedNodeIDs = append(sortedNodeIDs, nodeID)
	}
	sort.Strings(sortedNodeIDs)
	for _, nodeID := range sortedNodeIDs {
		leftNode, leftExists := left.Graph[nodeID]
		rightNode, rightExists := right.Graph[nodeID]
		if !leftExists || !rightExists {
			differences = append(differences, fmt.Sprintf("graph.%s", nodeID))
			continue
		}
		differences = append(differences, compareGraphNodeConfigurations(nodeID, leftNode, rightNode)...)
	}
	sort.Strings(differences)
	return differences
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
