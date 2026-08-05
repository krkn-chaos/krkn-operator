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
