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
	"testing"

	krknctlmodels "github.com/krkn-chaos/krknctl/pkg/scenarioorchestrator/models"
)

func TestToKrknctlScenarioSet(t *testing.T) {
	parent := "scenario1"
	graph := map[string]GraphScenarioNode{
		"scenario1": {
			Name:    "scenario-1",
			Image:   "quay.io/krkn-chaos/scenario1:latest",
			Comment: "First scenario",
			Env: map[string]string{
				"KEY1": "value1",
			},
			Volumes: map[string]string{
				"/host": "/container",
			},
			DependsOn: nil,
		},
		"scenario2": {
			Name:      "scenario-2",
			Image:     "quay.io/krkn-chaos/scenario2:latest",
			DependsOn: &parent,
		},
	}

	scenarioSet := ToKrknctlScenarioSet(graph)

	// Verify scenario1
	if scenario1, ok := scenarioSet["scenario1"]; !ok {
		t.Error("Expected scenario1 in ScenarioSet")
	} else {
		if scenario1.Name != "scenario-1" {
			t.Errorf("Expected name 'scenario-1', got '%s'", scenario1.Name)
		}
		if scenario1.Image != "quay.io/krkn-chaos/scenario1:latest" {
			t.Errorf("Expected image 'quay.io/krkn-chaos/scenario1:latest', got '%s'", scenario1.Image)
		}
		if scenario1.Parent != nil {
			t.Errorf("Expected nil parent, got '%v'", scenario1.Parent)
		}
		if len(scenario1.Env) != 1 || scenario1.Env["KEY1"] != "value1" {
			t.Errorf("Expected Env map with KEY1=value1, got %v", scenario1.Env)
		}
	}

	// Verify scenario2
	if scenario2, ok := scenarioSet["scenario2"]; !ok {
		t.Error("Expected scenario2 in ScenarioSet")
	} else {
		if scenario2.Parent == nil {
			t.Error("Expected non-nil parent")
		} else if *scenario2.Parent != "scenario1" {
			t.Errorf("Expected parent 'scenario1', got '%s'", *scenario2.Parent)
		}
	}
}

func TestFromKrknctlScenarioSet(t *testing.T) {
	parent := "scenario1"
	scenarioSet := krknctlmodels.ScenarioSet{
		"scenario1": {
			Scenario: krknctlmodels.Scenario{
				Name:    "scenario-1",
				Image:   "quay.io/krkn-chaos/scenario1:latest",
				Comment: "First scenario",
				Env: map[string]string{
					"KEY1": "value1",
				},
				Volumes: map[string]string{
					"/host": "/container",
				},
			},
			Parent: nil,
		},
		"scenario2": {
			Scenario: krknctlmodels.Scenario{
				Name:  "scenario-2",
				Image: "quay.io/krkn-chaos/scenario2:latest",
			},
			Parent: &parent,
		},
	}

	graph := FromKrknctlScenarioSet(scenarioSet)

	// Verify scenario1
	if scenario1, ok := graph["scenario1"]; !ok {
		t.Error("Expected scenario1 in graph")
	} else {
		if scenario1.Name != "scenario-1" {
			t.Errorf("Expected name 'scenario-1', got '%s'", scenario1.Name)
		}
		if scenario1.Image != "quay.io/krkn-chaos/scenario1:latest" {
			t.Errorf("Expected image 'quay.io/krkn-chaos/scenario1:latest', got '%s'", scenario1.Image)
		}
		if scenario1.DependsOn != nil {
			t.Errorf("Expected nil DependsOn, got '%v'", scenario1.DependsOn)
		}
		if len(scenario1.Env) != 1 || scenario1.Env["KEY1"] != "value1" {
			t.Errorf("Expected Env map with KEY1=value1, got %v", scenario1.Env)
		}
	}

	// Verify scenario2
	if scenario2, ok := graph["scenario2"]; !ok {
		t.Error("Expected scenario2 in graph")
	} else {
		if scenario2.DependsOn == nil {
			t.Error("Expected non-nil DependsOn")
		} else if *scenario2.DependsOn != "scenario1" {
			t.Errorf("Expected DependsOn 'scenario1', got '%s'", *scenario2.DependsOn)
		}
	}
}

func TestRoundTripConversion(t *testing.T) {
	parent := "scenario1"
	originalGraph := map[string]GraphScenarioNode{
		"scenario1": {
			Name:    "scenario-1",
			Image:   "quay.io/krkn-chaos/scenario1:latest",
			Comment: "First scenario",
			Env: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			Volumes: map[string]string{
				"/host1": "/container1",
				"/host2": "/container2",
			},
			DependsOn: nil,
		},
		"scenario2": {
			Name:      "scenario-2",
			Image:     "quay.io/krkn-chaos/scenario2:latest",
			DependsOn: &parent,
			Env: map[string]string{
				"KEY3": "value3",
			},
		},
	}

	// Convert to krknctl and back
	scenarioSet := ToKrknctlScenarioSet(originalGraph)
	roundTripGraph := FromKrknctlScenarioSet(scenarioSet)

	// Verify all fields are preserved
	if len(roundTripGraph) != len(originalGraph) {
		t.Errorf("Expected %d nodes, got %d", len(originalGraph), len(roundTripGraph))
	}

	for nodeID, originalNode := range originalGraph {
		roundTripNode, ok := roundTripGraph[nodeID]
		if !ok {
			t.Errorf("Node %s missing after round trip", nodeID)
			continue
		}

		if roundTripNode.Name != originalNode.Name {
			t.Errorf("Node %s: name mismatch", nodeID)
		}
		if roundTripNode.Image != originalNode.Image {
			t.Errorf("Node %s: image mismatch", nodeID)
		}
		if roundTripNode.Comment != originalNode.Comment {
			t.Errorf("Node %s: comment mismatch", nodeID)
		}

		// Compare DependsOn
		if (originalNode.DependsOn == nil) != (roundTripNode.DependsOn == nil) {
			t.Errorf("Node %s: DependsOn nil mismatch", nodeID)
		} else if originalNode.DependsOn != nil && *originalNode.DependsOn != *roundTripNode.DependsOn {
			t.Errorf("Node %s: DependsOn value mismatch", nodeID)
		}

		// Compare Env maps
		if len(originalNode.Env) != len(roundTripNode.Env) {
			t.Errorf("Node %s: Env length mismatch", nodeID)
		}
		for k, v := range originalNode.Env {
			if roundTripNode.Env[k] != v {
				t.Errorf("Node %s: Env[%s] mismatch", nodeID, k)
			}
		}

		// Compare Volumes maps
		if len(originalNode.Volumes) != len(roundTripNode.Volumes) {
			t.Errorf("Node %s: Volumes length mismatch", nodeID)
		}
		for k, v := range originalNode.Volumes {
			if roundTripNode.Volumes[k] != v {
				t.Errorf("Node %s: Volumes[%s] mismatch", nodeID, k)
			}
		}
	}
}
