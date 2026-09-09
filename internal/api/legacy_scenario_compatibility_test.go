package api

import (
	"encoding/json"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func TestNormalizeLegacyScenarioRunRequestIgnoresImage(t *testing.T) {
	var request ScenarioRunRequest
	if err := json.Unmarshal([]byte(`{"scenarioName":"cpu-hog","scenarioImage":"attacker/image:latest"}`), &request); err != nil {
		t.Fatalf("decode legacy request: %v", err)
	}
	normalizeLegacyScenarioRunRequest(&request)

	if request.Scenario.Name != "cpu-hog" || request.Scenario.Private == nil || *request.Scenario.Private {
		t.Fatalf("unexpected normalized public scenario: %+v", request.Scenario)
	}
	if request.ScenarioImage != "attacker/image:latest" {
		t.Fatal("expected legacy image to remain untrusted input")
	}
}

func TestNormalizeLegacyPrivateGraphNode(t *testing.T) {
	registry := "private-registry"
	node := krknv1alpha1.GraphScenarioNode{Name: "cpu-hog", Image: "attacker/image:latest", RegistryName: registry}
	normalizeLegacyGraphScenarioNode(&node)

	if node.Scenario.Name != "cpu-hog" || node.Scenario.Private == nil || !*node.Scenario.Private || node.Scenario.RegistryName != registry {
		t.Fatalf("unexpected normalized private scenario: %+v", node.Scenario)
	}
}
