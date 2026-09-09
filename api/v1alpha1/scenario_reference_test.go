package v1alpha1

import "testing"

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.ref.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
