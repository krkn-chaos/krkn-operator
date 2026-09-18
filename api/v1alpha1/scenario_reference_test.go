package v1alpha1

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestScenarioReferenceValidate(t *testing.T) {
	public := false
	private := true
	tests := []struct {
		name    string
		ref     ScenarioReference
		wantErr bool
	}{
		{name: "public", ref: ScenarioReference{Name: "dummy-scenario", Private: &public}},
		{name: "private", ref: ScenarioReference{Name: "dummy-scenario", Private: &private, RegistryName: "private-registry"}},
		{name: "missing name", ref: ScenarioReference{Private: &public}, wantErr: true},
		{name: "missing private flag", ref: ScenarioReference{Name: "dummy-scenario"}, wantErr: true},
		{name: "private without registry", ref: ScenarioReference{Name: "dummy-scenario", Private: &private}, wantErr: true},
		{name: "public with registry", ref: ScenarioReference{Name: "dummy-scenario", Private: &public, RegistryName: "private-registry"}, wantErr: true},
		{name: "digest reference", ref: ScenarioReference{Name: "dummy-scenario@sha256:abc", Private: &public}, wantErr: true},
		{name: "registry path", ref: ScenarioReference{Name: "registry.example/scenario", Private: &public}, wantErr: true},
		{name: "tag separator", ref: ScenarioReference{Name: "scenario:latest", Private: &public}, wantErr: true},
		{name: "leading hyphen", ref: ScenarioReference{Name: "-scenario", Private: &public}, wantErr: true},
		{name: "whitespace", ref: ScenarioReference{Name: "scenario name", Private: &public}, wantErr: true},
		{name: "too long", ref: ScenarioReference{Name: strings.Repeat("a", 129), Private: &public}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.ref.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolveScenarioReferenceMigratesOnlyLegacyIdentity(t *testing.T) {
	spec := KrknScenarioRunSpec{
		ScenarioName:       "legacy-scenario",
		RegistryName:       "private-registry",
		ScenarioImage:      "attacker.example/ignored:latest",
		RegistryURL:        "https://attacker.example",
		ScenarioRepository: "ignored",
		Token:              "ignored",
		Username:           "ignored",
		Password:           "ignored",
	}

	reference, legacy, err := spec.ResolveScenarioReference()
	if err != nil {
		t.Fatalf("ResolveScenarioReference() error = %v", err)
	}
	if !legacy {
		t.Fatal("ResolveScenarioReference() did not identify legacy storage")
	}
	if reference.Name != "legacy-scenario" || reference.Private == nil || !*reference.Private || reference.RegistryName != "private-registry" {
		t.Fatalf("ResolveScenarioReference() = %+v", reference)
	}
}

func TestLegacyScenarioIdentityRemainsDecodable(t *testing.T) {
	var spec KrknScenarioRunSpec
	if err := json.Unmarshal([]byte(`{
		"scenarioName":"legacy-scenario",
		"registryName":"private-registry",
		"scenarioImage":"attacker.example/ignored:latest",
		"token":"ignored"
	}`), &spec); err != nil {
		t.Fatal(err)
	}
	reference, legacy, err := spec.ResolveScenarioReference()
	if err != nil || !legacy {
		t.Fatalf("legacy identity was not resolved: reference=%+v legacy=%v error=%v", reference, legacy, err)
	}
	if reference.Name != "legacy-scenario" || reference.RegistryName != "private-registry" ||
		reference.Private == nil || !*reference.Private {
		t.Fatalf("unexpected legacy reference: %+v", reference)
	}

	encoded, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "scenarioImage") || strings.Contains(string(encoded), "token") {
		t.Fatalf("legacy executable or credential fields were serialized: %s", encoded)
	}
}
