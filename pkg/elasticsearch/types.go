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

// Package elasticsearch provides functionality for managing Elasticsearch
// connection configurations in the krkn-operator ecosystem. Configs are stored
// as Kubernetes Secrets with labeled metadata and are used to pre-populate
// scenario global parameters for chaos experiment runs.
package elasticsearch

import (
	"fmt"
	"time"
)

// CreateElasticsearchConfigRequest represents the request to create an ES config
type CreateElasticsearchConfigRequest struct {
	Name           string `json:"name"`
	Host           string `json:"host"`
	Port           int    `json:"port,omitempty"`
	Username       string `json:"username,omitempty"`
	Password       string `json:"password,omitempty"`
	TelemetryIndex string `json:"telemetryIndex,omitempty"`
	MetricsIndex   string `json:"metricsIndex,omitempty"`
	AlertsIndex    string `json:"alertsIndex,omitempty"`
	GrafanaURL     string `json:"grafanaUrl,omitempty"`
	// CACert is an optional PEM-encoded CA certificate (or bundle) used to trust
	// a self-signed cluster while keeping TLS verification enabled.
	CACert string `json:"caCert,omitempty"`
	// InsecureSkipTLSVerify disables TLS certificate verification entirely. It is
	// a restricted last resort for self-signed clusters without CA material;
	// prefer CACert.
	InsecureSkipTLSVerify bool `json:"insecureSkipTlsVerify,omitempty"`
}

// UpdateElasticsearchConfigRequest represents the request to update an ES config.
// InsecureSkipTLSVerify is a pointer so an omitted field (nil, "leave as-is") is
// distinguishable from an explicit false ("re-enable verification").
type UpdateElasticsearchConfigRequest struct {
	Host           string `json:"host"`
	Port           int    `json:"port,omitempty"`
	Username       string `json:"username,omitempty"`
	Password       string `json:"password,omitempty"`
	TelemetryIndex string `json:"telemetryIndex,omitempty"`
	MetricsIndex   string `json:"metricsIndex,omitempty"`
	AlertsIndex    string `json:"alertsIndex,omitempty"`
	GrafanaURL     string `json:"grafanaUrl,omitempty"`
	// CACert is an optional PEM-encoded CA certificate (or bundle) used to trust
	// a self-signed cluster while keeping TLS verification enabled.
	CACert string `json:"caCert,omitempty"`
	// InsecureSkipTLSVerify disables TLS certificate verification entirely. It is
	// a restricted last resort for self-signed clusters without CA material;
	// prefer CACert. A nil pointer leaves the stored setting unchanged; a non-nil
	// value explicitly sets or clears it.
	InsecureSkipTLSVerify *bool `json:"insecureSkipTlsVerify,omitempty"`
}

// ElasticsearchConfigResponse represents an ES config in API responses.
// The password is never included; callers must re-supply it on update.
type ElasticsearchConfigResponse struct {
	Name           string `json:"name"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Username       string `json:"username,omitempty"`
	TelemetryIndex string `json:"telemetryIndex,omitempty"`
	MetricsIndex   string `json:"metricsIndex,omitempty"`
	AlertsIndex    string `json:"alertsIndex,omitempty"`
	GrafanaURL     string `json:"grafanaUrl,omitempty"`
	// InsecureSkipTLSVerify reports whether TLS certificate verification is
	// disabled for this config. Surfaced so the admin edit form can show and
	// re-submit the current setting; it is not a secret.
	InsecureSkipTLSVerify bool   `json:"insecureSkipTlsVerify,omitempty"`
	CreatedAt             string `json:"createdAt,omitempty"`
	CreatedBy             string `json:"createdBy,omitempty"`
	UpdatedAt             string `json:"updatedAt,omitempty"`
	UpdatedBy             string `json:"updatedBy,omitempty"`
}

// ListElasticsearchConfigsResponse represents the response for listing ES configs
type ListElasticsearchConfigsResponse struct {
	Configs []ElasticsearchConfigResponse `json:"configs"`
	Total   int                           `json:"total"`
}

// CreateElasticsearchConfigResponse represents the response after creating an ES config
type CreateElasticsearchConfigResponse struct {
	Message string `json:"message"`
	Name    string `json:"name"`
}

// UpdateElasticsearchConfigResponse represents the response after updating an ES config
type UpdateElasticsearchConfigResponse struct {
	Message string `json:"message"`
	Name    string `json:"name"`
}

// DeleteElasticsearchConfigResponse represents the response after deleting an ES config
type DeleteElasticsearchConfigResponse struct {
	Message string `json:"message"`
}

// Query size bounds for telemetry searches. DefaultQuerySize is used when the
// request omits a size; MaxQuerySize caps how many documents a single request
// may return to protect the API server and browser.
const (
	DefaultQuerySize = 50
	MaxQuerySize     = 500
)

// InlineConnection carries an ephemeral Elasticsearch connection supplied
// directly on a query request instead of referencing a saved config. It lets a
// user (including non-admins, who cannot create stored configs) connect to an ES
// cluster and fetch telemetry without persisting any credentials server-side.
// The values are used only for the duration of the request and are never stored.
//
// TLS verification is intentionally not configurable here: disabling it and
// supplying a custom CA are admin-only, per-config concerns. An inline
// connection always uses default TLS verification, and credentials are still
// rejected over plaintext HTTP.
type InlineConnection struct {
	Host           string `json:"host"`
	Port           int    `json:"port,omitempty"`
	Username       string `json:"username,omitempty"`
	Password       string `json:"password,omitempty"`
	TelemetryIndex string `json:"telemetryIndex"`
}

// QueryTelemetryRequest represents a request to query telemetry documents from
// the telemetry index of an Elasticsearch cluster. It supports two mutually
// exclusive modes: a saved config referenced by ConfigName (credentials resolved
// server-side from the named Secret), or an ephemeral Inline connection whose
// credentials are supplied on the request and never persisted. Exactly one of
// ConfigName or Inline must be provided.
type QueryTelemetryRequest struct {
	ConfigName string `json:"configName,omitempty"`
	// Inline carries an ephemeral connection when no saved config is used.
	Inline *InlineConnection `json:"inline,omitempty"`
	Size   int               `json:"size,omitempty"`
	// StartDate and EndDate bound the search by the document timestamp. They are
	// "yyyy-MM-dd" date strings (as produced by the UI date pickers). Empty
	// values fall back to a default trailing window in the query client.
	StartDate string `json:"startDate,omitempty"`
	EndDate   string `json:"endDate,omitempty"`
}

// TelemetryDocument is the flattened set of telemetry fields surfaced to the UI
// table. Each document corresponds to one telemetry run; the scenario-level
// fields (type, start/end, namespace) are taken from the run's first scenario.
// Additional run detail (cluster config, node info, etc.) is intentionally not
// included here — it will be fetched for an expanded row in a later change.
type TelemetryDocument struct {
	RunUUID        string `json:"run_uuid"`
	ScenarioType   string `json:"scenario_type"`
	StartTimestamp int64  `json:"start_timestamp"`
	EndTimestamp   int64  `json:"end_timestamp"`
	Namespace      string `json:"namespace"`
	Status         bool   `json:"status"`
}

// TelemetryStats summarizes run-level pass/fail counts across the entire matched
// time window (not just the size-capped documents page). Counts come from a terms
// aggregation on the run-level job_status boolean, so they intentionally do not
// apply the per-scenario exit_status downgrade that a document's status field uses.
type TelemetryStats struct {
	Pass        int     `json:"pass"`
	Fail        int     `json:"fail"`
	PassPercent float64 `json:"pass_percent"` // 0-100, rounded to 2 decimals; 0 when no runs
}

// QueryTelemetryResponse wraps the telemetry documents returned to the client.
type QueryTelemetryResponse struct {
	Documents []TelemetryDocument `json:"documents"`
	Total     int                 `json:"total"`
	// Stats summarizes pass/fail across the whole matched window, so Stats.Pass +
	// Stats.Fail can exceed Total (which counts only the returned documents page).
	Stats TelemetryStats `json:"stats"`
}

// ValidateQueryRequest validates a QueryTelemetryRequest and normalizes the
// requested size into the supported bounds. Exactly one of configName or inline
// must be supplied; an inline connection additionally requires a host and a
// telemetry index and must satisfy the shared TLS rules.
func ValidateQueryRequest(req *QueryTelemetryRequest) error {
	hasConfigName := req.ConfigName != ""
	hasInline := req.Inline != nil
	if hasConfigName == hasInline {
		return fmt.Errorf("exactly one of configName or inline is required")
	}
	if hasInline {
		if req.Inline.Host == "" {
			return fmt.Errorf("inline.host is required")
		}
		if req.Inline.TelemetryIndex == "" {
			return fmt.Errorf("inline.telemetryIndex is required")
		}
		if req.Inline.Port < 0 || req.Inline.Port > 65535 {
			return fmt.Errorf("inline.port must be between 0 and 65535")
		}
		// Inline connections cannot supply a custom CA; pass an empty CA so only
		// the plaintext-credential rule applies.
		if err := validateTLSSettings(req.Inline.Host, req.Inline.Username, ""); err != nil {
			return err
		}
	}
	if req.Size < 0 {
		return fmt.Errorf("size must not be negative")
	}
	if req.Size == 0 {
		req.Size = DefaultQuerySize
	}
	if req.Size > MaxQuerySize {
		req.Size = MaxQuerySize
	}
	if err := validateDate("startDate", req.StartDate); err != nil {
		return err
	}
	if err := validateDate("endDate", req.EndDate); err != nil {
		return err
	}
	if req.StartDate != "" && req.EndDate != "" && req.StartDate > req.EndDate {
		return fmt.Errorf("startDate must not be after endDate")
	}
	if req.EndDate != "" && req.EndDate > time.Now().UTC().Format(dateLayout) {
		return fmt.Errorf("endDate must not be in the future")
	}
	return nil
}

// dateLayout is the date-only layout accepted for query bounds and understood by
// the Elasticsearch range filter's "yyyy-MM-dd" format.
const dateLayout = "2006-01-02"

// validateDate ensures an optional date string is empty or a valid yyyy-MM-dd
// date. field is used in the error message to identify the offending parameter.
func validateDate(field, value string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse(dateLayout, value); err != nil {
		return fmt.Errorf("%s must be a valid yyyy-MM-dd date", field)
	}
	return nil
}
