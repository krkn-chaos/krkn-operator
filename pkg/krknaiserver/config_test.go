package krknaiserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krkn-chaos/krkn-operator/pkg/registry"
)

const runnableConfigYAML = "scenario:\n  pod-scenarios:\n    enable: true\n"

func TestValidateScenarioSelection(t *testing.T) {
	for _, tc := range []struct {
		name  string
		yaml  string
		valid bool
	}{
		{"missing scenarios", "genetic:\n  generations: 1\n", false},
		{"empty scenarios", "scenario: {}\n", false},
		{"disabled scenarios", "scenario:\n  pod-scenarios:\n    enable: false\n", false},
		{"unset flag", "scenario:\n  pod-scenarios: {}\n", false},
		{"baseline only", "baseline:\n  enable: true\n", false},
		{"dangerous gate", "scenario:\n  service-disruption:\n    enable: true\n", false},
		{"invalid YAML", "scenario: [\n", false},
		{"enabled scenario", runnableConfigYAML, true},
		{"quoted flag", "scenario:\n  pod_scenarios:\n    enable: 'yes'\n", true},
		{"numeric flag", "scenario:\n  pod-scenarios:\n    enable: 1\n", true},
		{"confirmed dangerous scenario", "allow_dangerous_scenarios: true\nscenario:\n  service-disruption:\n    enable: true\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateScenarioSelection(tc.yaml)
			if (err == nil) != tc.valid {
				t.Fatalf("scenario selection error = %v, valid=%v", err, tc.valid)
			}
			if err != nil {
				var validation *ConfigValidationError
				if !errors.Is(err, ErrInvalidConfig) || !errors.As(err, &validation) || len(validation.Errors) != 1 || validation.Errors[0].Path != "scenario" {
					t.Fatalf("invalid selection did not return a structured scenario error: %v", err)
				}
			}
		})
	}
}

func TestValidateConfigImagesRejectsEmptySelectionBeforeService(t *testing.T) {
	calls := 0
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer service.Close()
	err := New(service.URL, "token").ValidateConfigImages(context.Background(), "scenario: {}", &registry.AIImageInventory{})
	if !errors.Is(err, ErrInvalidConfig) || calls != 0 {
		t.Fatalf("empty selection error=%v, service calls=%d", err, calls)
	}
}
