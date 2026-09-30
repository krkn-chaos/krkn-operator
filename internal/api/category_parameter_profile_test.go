package api

import (
	"strings"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func TestCategoryParameterProfileFingerprintScenarioEnvironment(t *testing.T) {
	base := scenarioRunConfigurationFixture()
	baseRun := categoryHistoryRun{scenario: base}
	baseFingerprint := categoryParameterProfileFingerprint(baseRun)

	reordered := base.DeepCopy()
	reordered.Spec.Environment = map[string]string{
		"DURATION":  "60",
		"POD_COUNT": "2",
	}
	if got := categoryParameterProfileFingerprint(categoryHistoryRun{scenario: reordered}); got != baseFingerprint {
		t.Fatalf("fingerprint changed with map insertion order: got %q, want %q", got, baseFingerprint)
	}

	changed := base.DeepCopy()
	changed.Spec.Environment["POD_COUNT"] = "3"
	if got := categoryParameterProfileFingerprint(categoryHistoryRun{scenario: changed}); got == baseFingerprint {
		t.Fatal("fingerprint did not change when an environment value changed")
	}

	if len(baseFingerprint) != 64 {
		t.Fatalf("SHA-256 fingerprint length = %d, want 64 hex characters", len(baseFingerprint))
	}
}

func TestCategoryParameterProfileFingerprintGraphEnvironmentMultiset(t *testing.T) {
	first := &krknv1alpha1.KrknGraphRun{}
	first.Spec.Graph = map[string]krknv1alpha1.GraphScenarioNode{
		"node-a": {Env: map[string]string{"COUNT": "1"}},
		"node-b": {Env: map[string]string{"COUNT": "2"}},
	}
	firstFingerprint := categoryParameterProfileFingerprint(categoryHistoryRun{graph: first})

	regenerated := &krknv1alpha1.KrknGraphRun{}
	regenerated.Spec.Graph = map[string]krknv1alpha1.GraphScenarioNode{
		"replay-node-b": {Env: map[string]string{"COUNT": "2"}},
		"replay-node-a": {Env: map[string]string{"COUNT": "1"}},
	}
	if got := categoryParameterProfileFingerprint(categoryHistoryRun{graph: regenerated}); got != firstFingerprint {
		t.Fatalf("fingerprint changed with node IDs or map order: got %q, want %q", got, firstFingerprint)
	}

	swapped := &krknv1alpha1.KrknGraphRun{}
	swapped.Spec.Graph = map[string]krknv1alpha1.GraphScenarioNode{
		"node-a": {Env: map[string]string{"COUNT": "2"}},
		"node-b": {Env: map[string]string{"COUNT": "1"}},
	}
	if got := categoryParameterProfileFingerprint(categoryHistoryRun{graph: swapped}); got != firstFingerprint {
		t.Fatalf("fingerprint changed when the same parameter multiset moved between nodes: got %q, want %q", got, firstFingerprint)
	}

	withDuplicate := &krknv1alpha1.KrknGraphRun{}
	withDuplicate.Spec.Graph = map[string]krknv1alpha1.GraphScenarioNode{
		"node-a": {Env: map[string]string{"COUNT": "1"}},
		"node-b": {Env: map[string]string{"COUNT": "1"}},
	}
	once := &krknv1alpha1.KrknGraphRun{}
	once.Spec.Graph = map[string]krknv1alpha1.GraphScenarioNode{
		"node-a": {Env: map[string]string{"COUNT": "1"}},
	}
	if categoryParameterProfileFingerprint(categoryHistoryRun{graph: withDuplicate}) == categoryParameterProfileFingerprint(categoryHistoryRun{graph: once}) {
		t.Fatal("duplicate environment parameters were not retained in the profile")
	}
}

func TestCategoryParameterProfileFingerprintIgnoresRunType(t *testing.T) {
	scenario := scenarioRunConfigurationFixture()
	graph := &krknv1alpha1.KrknGraphRun{}
	graph.Spec.Graph = map[string]krknv1alpha1.GraphScenarioNode{
		"node-a": {Env: map[string]string{"POD_COUNT": "2", "DURATION": "60"}},
	}

	scenarioFingerprint := categoryParameterProfileFingerprint(categoryHistoryRun{scenario: scenario})
	graphFingerprint := categoryParameterProfileFingerprint(categoryHistoryRun{graph: graph})
	if graphFingerprint != scenarioFingerprint {
		t.Fatalf("same environment profile produced different fingerprints by run type: %q != %q", graphFingerprint, scenarioFingerprint)
	}
}

func TestCategoryParameterProfileFingerprintForEmptyEnvironment(t *testing.T) {
	emptyScenario := scenarioRunConfigurationFixture()
	emptyScenario.Spec.Environment = nil
	emptyGraph := &krknv1alpha1.KrknGraphRun{}
	emptyGraph.Spec.Graph = map[string]krknv1alpha1.GraphScenarioNode{
		"node-a": {},
		"node-b": {},
	}

	scenarioFingerprint := categoryParameterProfileFingerprint(categoryHistoryRun{scenario: emptyScenario})
	graphFingerprint := categoryParameterProfileFingerprint(categoryHistoryRun{graph: emptyGraph})
	if scenarioFingerprint != graphFingerprint {
		t.Fatalf("empty environment profiles differ: %q != %q", scenarioFingerprint, graphFingerprint)
	}
	if got := categoryParameterProfileName(scenarioFingerprint); got != categoryParameterProfileName(graphFingerprint) {
		t.Fatalf("empty environment profiles have different names: %q != %q", got, categoryParameterProfileName(graphFingerprint))
	}
}

func TestCategoryParameterProfileNameIsDeterministicAndReadable(t *testing.T) {
	base := scenarioRunConfigurationFixture()
	fingerprint := categoryParameterProfileFingerprint(categoryHistoryRun{scenario: base})
	name := categoryParameterProfileName(fingerprint)
	if got := categoryParameterProfileName(fingerprint); got != name {
		t.Fatalf("profile name changed for the same fingerprint: %q != %q", got, name)
	}
	changed := base.DeepCopy()
	changed.Spec.Environment["POD_COUNT"] = "3"
	if changedName := categoryParameterProfileName(categoryParameterProfileFingerprint(categoryHistoryRun{scenario: changed})); changedName == name {
		t.Fatalf("different parameter profiles received the same readable name %q", name)
	}
	parts := strings.Split(name, "-")
	if len(parts) != 3 || len(parts[2]) != 8 {
		t.Fatalf("profile name = %q, want adjective-animal plus an 8-character suffix", name)
	}
}
