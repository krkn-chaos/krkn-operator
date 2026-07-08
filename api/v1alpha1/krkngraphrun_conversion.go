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
	krknctlmodels "github.com/krkn-chaos/krknctl/pkg/scenarioorchestrator/models"
)

// ToKrknctlScenarioSet converts a GraphScenarioNode map to a krknctl ScenarioSet
// This enables seamless integration with krknctl's dependency graph resolution
func ToKrknctlScenarioSet(graph map[string]GraphScenarioNode) krknctlmodels.ScenarioSet {
	result := make(krknctlmodels.ScenarioSet)
	for nodeID, node := range graph {
		result[nodeID] = krknctlmodels.ScenarioNode{
			Scenario: krknctlmodels.Scenario{
				Comment: node.Comment,
				Image:   node.Image,
				Name:    node.Name,
				Env:     node.Env,
				Volumes: node.Volumes,
			},
			Parent: node.DependsOn,
		}
	}
	return result
}

// FromKrknctlScenarioSet converts a krknctl ScenarioSet to a GraphScenarioNode map
// This enables loading krknctl-compatible graph definitions into Kubernetes CRs
func FromKrknctlScenarioSet(scenarioSet krknctlmodels.ScenarioSet) map[string]GraphScenarioNode {
	result := make(map[string]GraphScenarioNode)
	for nodeID, node := range scenarioSet {
		result[nodeID] = GraphScenarioNode{
			Comment:   node.Comment,
			Image:     node.Image,
			Name:      node.Name,
			Env:       node.Env,
			Volumes:   node.Volumes,
			DependsOn: node.Parent,
		}
	}
	return result
}
