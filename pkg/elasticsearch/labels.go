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
	"time"
)

// Label and annotation keys for Elasticsearch config Secrets
const (
	// AppNameLabel is the standard app name label
	AppNameLabel = "app.kubernetes.io/name"
	// AppComponentLabel is the standard component label
	AppComponentLabel = "app.kubernetes.io/component"

	// HostAnnotation stores the Elasticsearch host URL
	HostAnnotation = "elasticsearch.krkn.krkn-chaos.dev/host"
	// PortAnnotation stores the Elasticsearch port number
	PortAnnotation = "elasticsearch.krkn.krkn-chaos.dev/port"
	// TelemetryIndexAnnotation stores the telemetry index name
	TelemetryIndexAnnotation = "elasticsearch.krkn.krkn-chaos.dev/telemetry-index"
	// MetricsIndexAnnotation stores the metrics index name
	MetricsIndexAnnotation = "elasticsearch.krkn.krkn-chaos.dev/metrics-index"
	// AlertsIndexAnnotation stores the alerts index name
	AlertsIndexAnnotation = "elasticsearch.krkn.krkn-chaos.dev/alerts-index"
	// GrafanaURLAnnotation stores the optional Grafana dashboard URL
	GrafanaURLAnnotation = "elasticsearch.krkn.krkn-chaos.dev/grafana-url"
	// CreatedByAnnotation stores the user ID who created the config
	CreatedByAnnotation = "elasticsearch.krkn.krkn-chaos.dev/created-by"
	// CreatedAtAnnotation stores the creation timestamp
	CreatedAtAnnotation = "elasticsearch.krkn.krkn-chaos.dev/created-at"
	// UpdatedByAnnotation stores the user ID who last updated the config
	UpdatedByAnnotation = "elasticsearch.krkn.krkn-chaos.dev/updated-by"
	// UpdatedAtAnnotation stores the last update timestamp
	UpdatedAtAnnotation = "elasticsearch.krkn.krkn-chaos.dev/updated-at"

	// AppName is the value for AppNameLabel
	AppName = "krkn-operator"
	// ComponentElasticsearchConfig is the value for AppComponentLabel
	ComponentElasticsearchConfig = "elasticsearch-config"

	// SecretKeyUsername is the key in Secret.Data for the username
	SecretKeyUsername = "username"
	// SecretKeyPassword is the key in Secret.Data for the password
	SecretKeyPassword = "password"

	// DefaultPort is the default Elasticsearch port
	DefaultPort = 9200
)

// BuildLabels creates the labels map for an Elasticsearch config Secret
func BuildLabels() map[string]string {
	return map[string]string{
		AppNameLabel:      AppName,
		AppComponentLabel: ComponentElasticsearchConfig,
	}
}

// BuildAnnotations creates the annotations map for an Elasticsearch config Secret
func BuildAnnotations(host string, port int, telemetryIndex, metricsIndex, alertsIndex, grafanaURL, createdBy string) map[string]string {
	annotations := map[string]string{
		HostAnnotation:      host,
		PortAnnotation:      fmt.Sprintf("%d", port),
		CreatedByAnnotation: createdBy,
		CreatedAtAnnotation: time.Now().UTC().Format(time.RFC3339),
	}

	if telemetryIndex != "" {
		annotations[TelemetryIndexAnnotation] = telemetryIndex
	}
	if metricsIndex != "" {
		annotations[MetricsIndexAnnotation] = metricsIndex
	}
	if alertsIndex != "" {
		annotations[AlertsIndexAnnotation] = alertsIndex
	}
	if grafanaURL != "" {
		annotations[GrafanaURLAnnotation] = grafanaURL
	}

	return annotations
}

// UpdateAnnotations updates the annotations for an Elasticsearch config Secret
func UpdateAnnotations(existing map[string]string, host string, port int, telemetryIndex, metricsIndex, alertsIndex, grafanaURL, updatedBy string) map[string]string {
	updated := make(map[string]string)
	for k, v := range existing {
		updated[k] = v
	}

	updated[HostAnnotation] = host
	updated[PortAnnotation] = fmt.Sprintf("%d", port)
	updated[UpdatedByAnnotation] = updatedBy
	updated[UpdatedAtAnnotation] = time.Now().UTC().Format(time.RFC3339)

	// Update index annotations (clear if empty)
	if telemetryIndex != "" {
		updated[TelemetryIndexAnnotation] = telemetryIndex
	} else {
		delete(updated, TelemetryIndexAnnotation)
	}
	if metricsIndex != "" {
		updated[MetricsIndexAnnotation] = metricsIndex
	} else {
		delete(updated, MetricsIndexAnnotation)
	}
	if alertsIndex != "" {
		updated[AlertsIndexAnnotation] = alertsIndex
	} else {
		delete(updated, AlertsIndexAnnotation)
	}
	if grafanaURL != "" {
		updated[GrafanaURLAnnotation] = grafanaURL
	} else {
		delete(updated, GrafanaURLAnnotation)
	}

	return updated
}
