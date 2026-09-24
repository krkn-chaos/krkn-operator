/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package v1alpha1

import (
	"testing"

	krknctlmodels "github.com/krkn-chaos/krknctl/pkg/scenarioorchestrator/models"
)

func publicScenario(name string) ScenarioReference {
	private := false
	return ScenarioReference{Name: name, Private: &private}
}

func TestToKrknctlScenarioSetUsesScenarioReference(t *testing.T) {
	parent := "scenario1"
	graph := map[string]GraphScenarioNode{
		"scenario1": {
			Scenario: publicScenario("scenario-1"),
			Comment:  "First scenario",
			Env:      map[string]string{"KEY1": "value1"},
			Volumes:  map[string]string{"/host": "/container"},
		},
		"scenario2": {Scenario: publicScenario("scenario-2"), DependsOn: &parent},
	}

	set := ToKrknctlScenarioSet(graph)
	if set["scenario1"].Name != "scenario-1" || set["scenario1"].Image != "" {
		t.Fatalf("expected name-only krknctl scenario, got %+v", set["scenario1"].Scenario)
	}
	if set["scenario2"].Parent == nil || *set["scenario2"].Parent != parent {
		t.Fatalf("expected parent %q", parent)
	}
}

func TestFromKrknctlScenarioSetCreatesPublicReferences(t *testing.T) {
	parent := "scenario1"
	set := krknctlmodels.ScenarioSet{
		"scenario1": {Scenario: krknctlmodels.Scenario{Name: "scenario-1", Image: "legacy-image"}},
		"scenario2": {Scenario: krknctlmodels.Scenario{Name: "scenario-2"}, Parent: &parent},
	}

	graph := FromKrknctlScenarioSet(set)
	for id, want := range map[string]string{"scenario1": "scenario-1", "scenario2": "scenario-2"} {
		node := graph[id]
		if node.Scenario.Name != want || node.Scenario.Private == nil || *node.Scenario.Private {
			t.Fatalf("node %s has invalid public reference: %+v", id, node.Scenario)
		}
		if node.Image != "" {
			t.Fatalf("node %s retained an executable image", id)
		}
	}
	if graph["scenario2"].DependsOn == nil || *graph["scenario2"].DependsOn != parent {
		t.Fatalf("expected dependency %q", parent)
	}
}
