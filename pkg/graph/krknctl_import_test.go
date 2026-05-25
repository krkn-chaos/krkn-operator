package graph

import (
	"testing"

	"github.com/krkn-chaos/krknctl/pkg/dependencygraph"
	"github.com/krkn-chaos/krknctl/pkg/scenarioorchestrator/models"
)

// TestKrknctlImports verifies that required krknctl packages can be imported
// and that the necessary types and functions are available.
func TestKrknctlImports(t *testing.T) {
	// Verify ScenarioSet type exists and can be initialized
	scenarioSet := make(models.ScenarioSet)
	if scenarioSet == nil {
		t.Error("Failed to create ScenarioSet")
	}

	// Verify ScenarioNode type exists with required fields
	node := models.ScenarioNode{
		Scenario: models.Scenario{
			Name:  "test-scenario",
			Image: "test-image:latest",
			Env: map[string]string{
				"KEY": "value",
			},
		},
		Parent: nil,
	}

	// Verify GetParent() method exists (required by ParentProvider interface)
	parent := node.GetParent()
	if parent != nil {
		t.Error("Expected nil parent")
	}

	// Verify NewGraphFromNodes function exists and can be called
	nodes := map[string]dependencygraph.ParentProvider{
		"test": &node,
	}

	graph, err := dependencygraph.NewGraphFromNodes(nodes)
	if err != nil {
		t.Errorf("NewGraphFromNodes failed: %v", err)
	}
	if graph == nil {
		t.Error("Expected non-nil graph")
	}

	// Verify JSON tags are present on Scenario struct
	// This is implicit - if the struct compiles with json tags, they're valid
	scenario := models.Scenario{
		Name:    "test",
		Image:   "image:tag",
		Comment: "test comment",
		Env: map[string]string{
			"ENV_VAR": "value",
		},
		Volumes: map[string]string{
			"/host/path": "/container/path",
		},
	}

	if scenario.Name != "test" {
		t.Error("Scenario fields not properly initialized")
	}
}
