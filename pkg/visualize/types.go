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

// Package visualize provides functionality for managing krkn-visualize
// (Grafana-based visualization) deployments on target clusters.
// Instances are stored as Kubernetes ConfigMaps with status tracking.
package visualize

// CreateVisualizeRequest represents the request to install krkn-visualize
type CreateVisualizeRequest struct {
	Name                    string   `json:"name"`
	TargetClusters          []string `json:"targetClusters"`
	Namespace               string   `json:"namespace,omitempty"`
	ElasticsearchConfigName string   `json:"elasticsearchConfigName,omitempty"`
	GrafanaPassword         string   `json:"grafanaPassword"`
	AutoDetectPrometheus    bool     `json:"autoDetectPrometheus,omitempty"`
	PrometheusURL           string   `json:"prometheusUrl,omitempty"`
	PrometheusBearerToken   string   `json:"prometheusBearerToken,omitempty"`
}

// VisualizeInstanceResponse represents a krkn-visualize instance in API responses
type VisualizeInstanceResponse struct {
	Name                    string `json:"name"`
	TargetCluster           string `json:"targetCluster"`
	Namespace               string `json:"namespace"`
	ElasticsearchConfigName string `json:"elasticsearchConfigName,omitempty"`
	Status                  string `json:"status"` // Pending, Installing, Ready, Failed
	GrafanaURL              string `json:"grafanaUrl,omitempty"`
	ErrorMessage            string `json:"errorMessage,omitempty"`
	CreatedAt               string `json:"createdAt,omitempty"`
	CreatedBy               string `json:"createdBy,omitempty"`
	UpdatedAt               string `json:"updatedAt,omitempty"`
}

// ListVisualizeInstancesResponse represents the response for listing instances
type ListVisualizeInstancesResponse struct {
	Instances []VisualizeInstanceResponse `json:"instances"`
	Total     int                         `json:"total"`
}

// CreateVisualizeResponse represents the response after creating an instance
type CreateVisualizeResponse struct {
	Message    string `json:"message"`
	Name       string `json:"name,omitempty"`
	GrafanaURL string `json:"grafanaUrl,omitempty"`
}

// DeleteVisualizeResponse represents the response after deleting an instance
type DeleteVisualizeResponse struct {
	Message string `json:"message"`
}

// StatusResponse represents the status of a visualize instance
type StatusResponse struct {
	Status       string `json:"status"`
	GrafanaURL   string `json:"grafanaUrl,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// LogsResponse represents the logs for a visualize instance
type LogsResponse struct {
	Logs    string `json:"logs"`
	Status  string `json:"status"`
	PodName string `json:"podName,omitempty"`
	JobName string `json:"jobName,omitempty"`
}

// Default namespace for krkn-visualize deployments
const DefaultNamespace = "krkn-visualize"

// Status constants
const (
	StatusPending    = "Pending"
	StatusInstalling = "Installing"
	StatusReady      = "Ready"
	StatusFailed     = "Failed"
	StatusDeleting   = "Deleting"
)

// Label keys for ConfigMap identification
const (
	LabelManagedBy = "app.kubernetes.io/managed-by"
	LabelComponent = "app.kubernetes.io/component"
	LabelName      = "app.kubernetes.io/name"
)

// Label values
const (
	ManagedByValue = "krkn-operator"
	ComponentValue = "krkn-visualize"
)

// Annotation keys for storing instance metadata
const (
	AnnotationTargetCluster        = "krkn.io/target-cluster"
	AnnotationElasticsearchConfig  = "krkn.io/elasticsearch-config"
	AnnotationNamespace            = "krkn.io/namespace"
	AnnotationStatus               = "krkn.io/status"
	AnnotationGrafanaURL           = "krkn.io/grafana-url"
	AnnotationErrorMessage         = "krkn.io/error-message"
	AnnotationCreatedBy            = "krkn.io/created-by"
	AnnotationCreatedAt            = "krkn.io/created-at"
	AnnotationUpdatedAt            = "krkn.io/updated-at"
	AnnotationAutoDetectPrometheus = "krkn.io/auto-detect-prometheus"
	AnnotationPrometheusURL        = "krkn.io/prometheus-url"
)

// Data keys for ConfigMap storing instance configuration
const (
	DataKeyGrafanaPassword       = "grafana-password"
	DataKeyPrometheusBearerToken = "prometheus-bearer-token"
)
