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
}

// UpdateElasticsearchConfigRequest represents the request to update an ES config
type UpdateElasticsearchConfigRequest struct {
	Host           string `json:"host"`
	Port           int    `json:"port,omitempty"`
	Username       string `json:"username,omitempty"`
	Password       string `json:"password,omitempty"`
	TelemetryIndex string `json:"telemetryIndex,omitempty"`
	MetricsIndex   string `json:"metricsIndex,omitempty"`
	AlertsIndex    string `json:"alertsIndex,omitempty"`
	GrafanaURL     string `json:"grafanaUrl,omitempty"`
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
	CreatedAt      string `json:"createdAt,omitempty"`
	CreatedBy      string `json:"createdBy,omitempty"`
	UpdatedAt      string `json:"updatedAt,omitempty"`
	UpdatedBy      string `json:"updatedBy,omitempty"`
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

// QueryTelemetryRequest represents a request to query telemetry documents from
// the telemetry index of a saved Elasticsearch config. Credentials are resolved
// server-side from the named config; the client only references it by name.
type QueryTelemetryRequest struct {
	ConfigName string `json:"configName"`
	Size       int    `json:"size,omitempty"`
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

// QueryTelemetryResponse wraps the telemetry documents returned to the client.
type QueryTelemetryResponse struct {
	Documents []TelemetryDocument `json:"documents"`
	Total     int                 `json:"total"`
}

// ValidateQueryRequest validates a QueryTelemetryRequest and normalizes the
// requested size into the supported bounds.
func ValidateQueryRequest(req *QueryTelemetryRequest) error {
	if req.ConfigName == "" {
		return fmt.Errorf("configName is required")
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
