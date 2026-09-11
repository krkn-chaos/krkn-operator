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
	"encoding/json"
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
	// CACert is an optional PEM-encoded CA certificate (or bundle) trusted in
	// addition to the host's system root CAs, so a self-signed cluster is reachable
	// with TLS verification still enabled and publicly trusted chains keep working.
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
	// CACert is an optional PEM-encoded CA certificate (or bundle) trusted in
	// addition to the host's system root CAs, so a self-signed cluster is reachable
	// with TLS verification still enabled and publicly trusted chains keep working.
	// It is a pointer to make omission (nil, "leave the stored CA unchanged")
	// distinguishable from an explicit empty string ("clear the stored CA").
	CACert *string `json:"caCert,omitempty"`
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
	// DefaultQuerySize is the document count applied when a query request omits
	// size (or sends 0). Unit: documents per request.
	DefaultQuerySize = 50
	// MaxQuerySize is the upper bound on documents a single query request may
	// return; larger requested sizes are clamped to this value. Unit: documents
	// per request.
	MaxQuerySize = 500
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
	// Host is the Elasticsearch host, optionally scheme-bearing (e.g.
	// "https://es.example.com"). Required.
	Host string `json:"host"`
	// Port is the Elasticsearch port. Optional; must be 0-65535. When 0 the
	// query client applies the default port.
	Port int `json:"port,omitempty"`
	// Username is the basic-auth user. Optional; when set the connection must
	// resolve to https (credentials are refused over plaintext HTTP).
	Username string `json:"username,omitempty"`
	// Password is the basic-auth password. Optional; used only with Username and
	// never persisted server-side.
	Password string `json:"password,omitempty"`
	// TelemetryIndex is the index queried for telemetry documents. Required.
	TelemetryIndex string `json:"telemetryIndex"`
}

// QueryTelemetryRequest represents a request to query telemetry documents from
// the telemetry index of an Elasticsearch cluster. It supports two mutually
// exclusive modes: a saved config referenced by ConfigName (credentials resolved
// server-side from the named Secret), or an ephemeral Inline connection whose
// credentials are supplied on the request and never persisted. Exactly one of
// ConfigName or Inline must be provided.
type QueryTelemetryRequest struct {
	// ConfigName references a saved Elasticsearch config by name; credentials are
	// resolved server-side. Mutually exclusive with Inline; exactly one is required.
	ConfigName string `json:"configName,omitempty"`
	// Inline carries an ephemeral connection when no saved config is used.
	Inline *InlineConnection `json:"inline,omitempty"`
	// Size is the max documents to return. Optional; 0 defaults to DefaultQuerySize
	// and values above MaxQuerySize are clamped. Unit: documents.
	Size int `json:"size,omitempty"`
	// StartDate and EndDate bound the search by the document timestamp. They are
	// "yyyy-MM-dd" date strings (as produced by the UI date pickers). Empty
	// values fall back to a default trailing window in the query client.
	StartDate string `json:"startDate,omitempty"`
	EndDate   string `json:"endDate,omitempty"`
}

// ClusterMetadata holds the run-level cluster and infrastructure details
// surfaced alongside a telemetry run so the UI can render an expanded row. All
// fields come from the top level of the telemetry document _source and are
// optional: a missing field decodes to its zero value and is omitted from the
// response.
type ClusterMetadata struct {
	// KubernetesObjectsCount maps object kind (e.g. "Pod", "ConfigMap") to the
	// number of that kind present in the cluster at run time.
	KubernetesObjectsCount map[string]int `json:"kubernetes_objects_count,omitempty"`
	// NetworkPlugins lists the cluster network plugins (e.g. "OVNKubernetes").
	NetworkPlugins        []string `json:"network_plugins,omitempty"`
	TotalNodeCount        int      `json:"total_node_count,omitempty"`
	CloudInfrastructure   string   `json:"cloud_infrastructure,omitempty"`
	CloudType             string   `json:"cloud_type,omitempty"`
	ClusterVersion        string   `json:"cluster_version,omitempty"`
	MajorVersion          string   `json:"major_version,omitempty"`
	BuildURL              string   `json:"build_url,omitempty"`
	FIPSEnabled           bool     `json:"fips_enabled,omitempty"`
	Tag                   string   `json:"tag,omitempty"`
	EtcdEncryptionEnabled bool     `json:"etcd_encryption_enabled,omitempty"`
	IPSecEnabled          bool     `json:"ipsec_enabled,omitempty"`
	// NodeSummaryInfos lists one summary per distinct node group (role/shape) in
	// the cluster at run time; nil when the source document carried none.
	NodeSummaryInfos []NodeSummaryInfo `json:"node_summary_infos,omitempty"`
}

// NodeSummaryInfo holds the summary krkn records for one node group sharing a
// role and hardware/software shape. Count is how many nodes fall in the group;
// the remaining fields describe that group. All fields come from an element of
// the telemetry document's node_summary_infos array.
type NodeSummaryInfo struct {
	Count          int    `json:"count"`
	NodesType      string `json:"nodes_type"`
	Architecture   string `json:"architecture"`
	InstanceType   string `json:"instance_type"`
	KernelVersion  string `json:"kernel_version"`
	KubeletVersion string `json:"kubelet_version"`
	OSVersion      string `json:"os_version"`
}

// RecoveredPod holds the recovery timings krkn records for a single pod that came
// back after a pod_disruption scenario. Times are fractional seconds (e.g.
// 37.533992528915405), so they are represented as float64.
type RecoveredPod struct {
	PodName             string  `json:"pod_name"`
	Namespace           string  `json:"namespace"`
	TotalRecoveryTime   float64 `json:"total_recovery_time"`
	PodReadinessTime    float64 `json:"pod_readiness_time"`
	PodReschedulingTime float64 `json:"pod_rescheduling_time"`
}

// AffectedPods groups the pods a scenario disrupted. Only the recovered pods,
// which carry recovery timings, are surfaced for the pod-recovery chart.
type AffectedPods struct {
	Recovered []RecoveredPod `json:"recovered,omitempty"`
}

// ScenarioDetail describes a single scenario within a telemetry run. The
// Parameters field is passed through verbatim as raw JSON because its shape
// varies by scenario type (e.g. an application_outage block vs a pod-scenario
// config/id object), so a fixed struct cannot represent it.
type ScenarioDetail struct {
	ScenarioType   string          `json:"scenario_type"`
	StartTimestamp int64           `json:"start_timestamp"`
	EndTimestamp   int64           `json:"end_timestamp"`
	ExitStatus     int             `json:"exit_status"`
	Parameters     json.RawMessage `json:"parameters,omitempty"`
	// AffectedPods carries per-pod recovery timings for pod_disruption scenarios;
	// nil when the source document had none.
	AffectedPods *AffectedPods `json:"affected_pods,omitempty"`
}

// TelemetryDocument is the set of telemetry fields surfaced to the UI. Each
// document corresponds to one telemetry run. The scalar table columns (type,
// start/end, namespace) are taken from the run's first scenario for backward
// compatibility with the table view, while Metadata and Scenarios carry the
// full run detail used to render an expanded row.
type TelemetryDocument struct {
	// RunUUID uniquely identifies the telemetry run this document represents.
	RunUUID string `json:"run_uuid"`
	// ScenarioType is the chaos scenario type from the run's first scenario.
	ScenarioType string `json:"scenario_type"`
	// StartTimestamp is the scenario start time. Unit: Unix seconds (UTC).
	StartTimestamp int64 `json:"start_timestamp"`
	// EndTimestamp is the scenario end time. Unit: Unix seconds (UTC).
	EndTimestamp int64 `json:"end_timestamp"`
	// Namespace is the target namespace from the run's first scenario.
	Namespace string `json:"namespace"`
	// Status is the run outcome: true = pass, false = fail. It applies the
	// per-scenario exit_status downgrade, so it may differ from the run-level
	// job_status used by TelemetryStats.
	Status bool `json:"status"`
	// Metadata holds run-level cluster/infrastructure detail; nil when the
	// source document carried none of the metadata fields.
	Metadata *ClusterMetadata `json:"metadata,omitempty"`
	// Scenarios lists every scenario in the run, each with its raw parameters.
	Scenarios []ScenarioDetail `json:"scenarios,omitempty"`
}

// TelemetryStats summarizes run-level pass/fail counts across the entire matched
// time window (not just the size-capped documents page). Counts come from a terms
// aggregation on the run-level job_status boolean, so they intentionally do not
// apply the per-scenario exit_status downgrade that a document's status field uses.
type TelemetryStats struct {
	// Pass is the count of runs with job_status true across the whole matched
	// window (not just the returned page). Unit: runs.
	Pass int `json:"pass"`
	// Fail is the count of runs with job_status false across the whole matched
	// window (not just the returned page). Unit: runs.
	Fail int `json:"fail"`
	// PassPercent is Pass / (Pass + Fail) * 100 across the matched window.
	// Range: 0-100, rounded to 2 decimals; 0 when no runs match.
	PassPercent float64 `json:"pass_percent"`
}

// QueryTelemetryResponse wraps the telemetry documents returned to the client.
type QueryTelemetryResponse struct {
	// Documents is the size-capped page of matched telemetry runs. Length is
	// bounded by the request Size (see DefaultQuerySize/MaxQuerySize).
	Documents []TelemetryDocument `json:"documents"`
	// Total is the number of documents in this returned page (len(Documents)),
	// not the total matched across the window. Unit: documents.
	Total int `json:"total"`
	// Stats summarizes pass/fail across the whole matched window, so Stats.Pass +
	// Stats.Fail can exceed Total (which counts only the returned documents page).
	Stats TelemetryStats `json:"stats"`
}

// ValidateQueryRequest validates a QueryTelemetryRequest and normalizes the
// requested size into the supported bounds. Exactly one of configName or inline
// must be supplied; an inline connection additionally requires a host and a
// telemetry index and must satisfy the shared TLS rules.
func ValidateQueryRequest(req *QueryTelemetryRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}
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
