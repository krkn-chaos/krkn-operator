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
*/

package elasticsearch

import (
	"fmt"
	"testing"
	"time"
)

func TestBuildLabels(t *testing.T) {
	got := BuildLabels()

	if got[AppNameLabel] != AppName {
		t.Errorf("AppNameLabel = %q, want %q", got[AppNameLabel], AppName)
	}
	if got[AppComponentLabel] != ComponentElasticsearchConfig {
		t.Errorf("AppComponentLabel = %q, want %q", got[AppComponentLabel], ComponentElasticsearchConfig)
	}
	if len(got) != 2 {
		t.Errorf("BuildLabels() returned %d labels, want 2", len(got))
	}
}

func TestBuildAnnotations(t *testing.T) {
	tests := []struct {
		name           string
		host           string
		port           int
		telemetryIndex string
		metricsIndex   string
		alertsIndex    string
		grafanaURL     string
		createdBy      string
		// keys that should be absent when empty
		expectNoTelemetry bool
		expectNoMetrics   bool
		expectNoAlerts    bool
		expectNoGrafana   bool
	}{
		{
			name:           "all fields populated",
			host:           "https://es.example.com",
			port:           9200,
			telemetryIndex: "krkn-telemetry",
			metricsIndex:   "krkn-metrics",
			alertsIndex:    "krkn-alerts",
			grafanaURL:     "https://grafana.example.com",
			createdBy:      "admin@example.com",
		},
		{
			name:              "optional fields empty",
			host:              "https://es.example.com",
			port:              9300,
			telemetryIndex:    "",
			metricsIndex:      "",
			alertsIndex:       "",
			grafanaURL:        "",
			createdBy:         "user@example.com",
			expectNoTelemetry: true,
			expectNoMetrics:   true,
			expectNoAlerts:    true,
			expectNoGrafana:   true,
		},
		{
			name:            "default port",
			host:            "https://es.io",
			port:            DefaultPort,
			telemetryIndex:  "telemetry",
			metricsIndex:    "",
			alertsIndex:     "",
			grafanaURL:      "",
			createdBy:       "admin",
			expectNoMetrics: true,
			expectNoAlerts:  true,
			expectNoGrafana: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAnnotations(tt.host, tt.port, tt.telemetryIndex, tt.metricsIndex, tt.alertsIndex, tt.grafanaURL, tt.createdBy)

			if got[HostAnnotation] != tt.host {
				t.Errorf("HostAnnotation = %q, want %q", got[HostAnnotation], tt.host)
			}
			if got[PortAnnotation] != fmt.Sprintf("%d", tt.port) {
				t.Errorf("PortAnnotation = %q, want %q", got[PortAnnotation], fmt.Sprintf("%d", tt.port))
			}
			if got[CreatedByAnnotation] != tt.createdBy {
				t.Errorf("CreatedByAnnotation = %q, want %q", got[CreatedByAnnotation], tt.createdBy)
			}

			if createdAt, ok := got[CreatedAtAnnotation]; !ok {
				t.Error("CreatedAtAnnotation missing")
			} else if _, err := time.Parse(time.RFC3339, createdAt); err != nil {
				t.Errorf("CreatedAtAnnotation has invalid RFC3339 format: %v", err)
			}

			checkOptional := func(key, value string, expectAbsent bool) {
				t.Helper()
				if expectAbsent {
					if _, exists := got[key]; exists {
						t.Errorf("annotation %q should be absent when value is empty", key)
					}
				} else {
					if got[key] != value {
						t.Errorf("annotation %q = %q, want %q", key, got[key], value)
					}
				}
			}

			checkOptional(TelemetryIndexAnnotation, tt.telemetryIndex, tt.expectNoTelemetry)
			checkOptional(MetricsIndexAnnotation, tt.metricsIndex, tt.expectNoMetrics)
			checkOptional(AlertsIndexAnnotation, tt.alertsIndex, tt.expectNoAlerts)
			checkOptional(GrafanaURLAnnotation, tt.grafanaURL, tt.expectNoGrafana)
		})
	}
}

func TestUpdateAnnotations(t *testing.T) {
	existing := map[string]string{
		HostAnnotation:           "https://old.example.com",
		PortAnnotation:           "9200",
		TelemetryIndexAnnotation: "old-telemetry",
		MetricsIndexAnnotation:   "old-metrics",
		CreatedByAnnotation:      "admin@example.com",
		CreatedAtAnnotation:      "2025-01-01T00:00:00Z",
	}

	tests := []struct {
		name              string
		host              string
		port              int
		telemetryIndex    string
		metricsIndex      string
		alertsIndex       string
		grafanaURL        string
		updatedBy         string
		expectNoTelemetry bool
		expectNoMetrics   bool
		expectNoAlerts    bool
		expectNoGrafana   bool
	}{
		{
			name:           "update all fields",
			host:           "https://new.example.com",
			port:           9300,
			telemetryIndex: "new-telemetry",
			metricsIndex:   "new-metrics",
			alertsIndex:    "new-alerts",
			grafanaURL:     "https://grafana.example.com",
			updatedBy:      "user@example.com",
		},
		{
			name:              "clear optional fields",
			host:              "https://new.example.com",
			port:              9200,
			telemetryIndex:    "",
			metricsIndex:      "",
			alertsIndex:       "",
			grafanaURL:        "",
			updatedBy:         "user@example.com",
			expectNoTelemetry: true,
			expectNoMetrics:   true,
			expectNoAlerts:    true,
			expectNoGrafana:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UpdateAnnotations(existing, tt.host, tt.port, tt.telemetryIndex, tt.metricsIndex, tt.alertsIndex, tt.grafanaURL, tt.updatedBy)

			if got[HostAnnotation] != tt.host {
				t.Errorf("HostAnnotation = %q, want %q", got[HostAnnotation], tt.host)
			}
			if got[UpdatedByAnnotation] != tt.updatedBy {
				t.Errorf("UpdatedByAnnotation = %q, want %q", got[UpdatedByAnnotation], tt.updatedBy)
			}

			// Original creation fields must be preserved
			if got[CreatedByAnnotation] != existing[CreatedByAnnotation] {
				t.Errorf("CreatedByAnnotation changed: got %q, want %q", got[CreatedByAnnotation], existing[CreatedByAnnotation])
			}
			if got[CreatedAtAnnotation] != existing[CreatedAtAnnotation] {
				t.Errorf("CreatedAtAnnotation changed: got %q, want %q", got[CreatedAtAnnotation], existing[CreatedAtAnnotation])
			}

			if updatedAt, ok := got[UpdatedAtAnnotation]; !ok {
				t.Error("UpdatedAtAnnotation missing")
			} else if _, err := time.Parse(time.RFC3339, updatedAt); err != nil {
				t.Errorf("UpdatedAtAnnotation has invalid RFC3339 format: %v", err)
			}

			checkOptional := func(key, value string, expectAbsent bool) {
				t.Helper()
				if expectAbsent {
					if _, exists := got[key]; exists {
						t.Errorf("annotation %q should be absent after being cleared", key)
					}
				} else {
					if got[key] != value {
						t.Errorf("annotation %q = %q, want %q", key, got[key], value)
					}
				}
			}

			checkOptional(TelemetryIndexAnnotation, tt.telemetryIndex, tt.expectNoTelemetry)
			checkOptional(MetricsIndexAnnotation, tt.metricsIndex, tt.expectNoMetrics)
			checkOptional(AlertsIndexAnnotation, tt.alertsIndex, tt.expectNoAlerts)
			checkOptional(GrafanaURLAnnotation, tt.grafanaURL, tt.expectNoGrafana)
		})
	}
}
