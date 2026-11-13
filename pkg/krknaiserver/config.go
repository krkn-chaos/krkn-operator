package krknaiserver

import (
	"strings"

	"sigs.k8s.io/yaml"
)

// ValidateScenarioSelection rejects configurations with no enabled experiment.
func ValidateScenarioSelection(configYAML string) error {
	var config struct {
		Scenario map[string]struct {
			Enable any `json:"enable"`
		} `json:"scenario"`
		AllowDangerousScenarios any `json:"allow_dangerous_scenarios"`
	}
	if err := yaml.Unmarshal([]byte(configYAML), &config); err == nil {
		for name, scenario := range config.Scenario {
			if (name == "service-disruption" || name == "service_disruption") && !enabled(config.AllowDangerousScenarios) {
				continue
			}
			if enabled(scenario.Enable) {
				return nil
			}
		}
	}
	return &ConfigValidationError{Errors: []ConfigValidationIssue{{
		Path: "scenario", Message: "Enable at least one runnable scenario.",
	}}}
}

func enabled(value any) bool {
	switch value := value.(type) {
	case bool:
		return value
	case float64:
		return value == 1
	case string:
		switch strings.ToLower(value) {
		case "1", "true", "yes", "on", "t", "y":
			return true
		}
	}
	return false
}
