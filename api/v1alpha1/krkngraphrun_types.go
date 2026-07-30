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

Assisted-by: Claude Sonnet 4.5 (claude-sonnet-4-5@20250929)
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GraphScenario represents a chaos scenario in the dependency graph
// This mirrors krknctl's Scenario struct but with explicit JSON tags for Kubernetes compatibility
type GraphScenario struct {
	// Comment is an optional comment describing the scenario
	// +optional
	Comment string `json:"_comment,omitempty"`

	// Image is the container image for the scenario
	// +optional
	Image string `json:"image,omitempty"`

	// Name is the name of the scenario
	// +optional
	Name string `json:"name,omitempty"`

	// Env is a map of environment variables for the scenario
	// +optional
	Env map[string]string `json:"env,omitempty"`

	// Volumes is a map of volume mounts for the scenario
	// +optional
	Volumes map[string]string `json:"volumes,omitempty"`
}

// GraphScenarioNode represents a node in the scenario dependency graph
// This mirrors krknctl's ScenarioNode struct but with explicit JSON tags for Kubernetes compatibility
type GraphScenarioNode struct {
	// Comment is an optional comment describing the scenario
	// +optional
	Comment string `json:"_comment,omitempty"`

	// Image is the container image for the scenario
	// +optional
	Image string `json:"image,omitempty"`

	// Name is the name of the scenario
	// +optional
	Name string `json:"name,omitempty"`

	// Env is a map of environment variables for the scenario
	// +optional
	Env map[string]string `json:"env,omitempty"`

	// Volumes is a map of volume mounts for the scenario
	// +optional
	Volumes map[string]string `json:"volumes,omitempty"`

	// DependsOn is the node ID that this scenario depends on (parent in the graph)
	// +optional
	DependsOn *string `json:"depends_on,omitempty"`
}

// NodeStatus represents the status of a single node in the dependency graph
type NodeStatus struct {
	// NodeID is the unique identifier for this node in the graph
	NodeID string `json:"nodeId"`

	// NodeName is the human-readable name of the scenario
	NodeName string `json:"nodeName"`

	// Phase is the current phase of this node
	// +kubebuilder:validation:Enum=Pending;Running;Completed;Failed;Blocked
	Phase string `json:"phase"`

	// ScenarioRunRef is the reference to the KrknScenarioRun CR created for this node
	// +optional
	ScenarioRunRef string `json:"scenarioRunRef,omitempty"`

	// StartTime is when this node started execution
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is when this node completed execution
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// DependsOn is the list of node IDs that this node depends on
	// +optional
	DependsOn []string `json:"dependsOn,omitempty"`

	// Message contains additional information about the node status
	// +optional
	Message string `json:"message,omitempty"`
}

// GraphRunSummary contains aggregate statistics about the graph run
type GraphRunSummary struct {
	// TotalNodes is the total number of nodes in the graph
	TotalNodes int `json:"totalNodes"`

	// CompletedNodes is the number of successfully completed nodes
	CompletedNodes int `json:"completedNodes"`

	// RunningNodes is the number of currently running nodes
	RunningNodes int `json:"runningNodes"`

	// FailedNodes is the number of failed nodes
	FailedNodes int `json:"failedNodes"`

	// PendingNodes is the number of pending nodes (including blocked nodes)
	PendingNodes int `json:"pendingNodes"`
}

// GraphClusterScore represents the resiliency score for a specific cluster in a graph run.
// When a graph run executes on multiple clusters, this structure tracks the score contribution
// from each cluster, along with per-node breakdown.
type GraphClusterScore struct {
	// ClusterName is the name of the cluster this score applies to
	ClusterName string `json:"clusterName"`

	// Calculated is the aggregated resiliency score for this cluster (0-100)
	// This is calculated by averaging the scores from all nodes that ran on this cluster
	Calculated float64 `json:"calculated"`

	// Baseline is the user-defined minimum acceptable score (from Spec.ResiliencyScoreBaseline)
	// Used for pass/fail determination
	// +optional
	Baseline *float64 `json:"baseline,omitempty"`

	// Status indicates whether this cluster's score met the baseline requirements
	// Possible values:
	// - "pass": calculated score >= baseline
	// - "fail": calculated score < baseline
	// - "no-baseline": no baseline was specified, score calculated but not compared
	// +kubebuilder:validation:Enum=pass;fail;no-baseline
	Status string `json:"status"`

	// Message provides human-readable context about the score (e.g., "Score 85.0 meets baseline 80.0")
	// +optional
	Message string `json:"message,omitempty"`

	// NodeContributions maps each node ID to its individual resiliency score on this cluster
	// This allows the frontend to show which nodes contributed what score
	// Example: {"node-1": 85.5, "node-2": 92.3}
	// +optional
	NodeContributions map[string]float64 `json:"nodeContributions,omitempty"`
}

// KrknGraphRunSpec defines the desired state of KrknGraphRun
type KrknGraphRunSpec struct {
	// Graph is the dependency graph of scenarios to execute
	// Maps node ID to scenario node definition with dependencies
	// This is compatible with krknctl ScenarioSet for 1:1 JSON compatibility
	// +kubebuilder:validation:Required
	Graph map[string]GraphScenarioNode `json:"graph"`

	// TargetRequestID is the reference to the KrknTargetRequest CR
	// +kubebuilder:validation:Required
	TargetRequestID string `json:"targetRequestId"`

	// TargetClusters is a map of provider-name to list of cluster names
	// Example: {"krkn-operator": ["cluster1", "cluster2"], "krkn-operator-acm": ["cluster3"]}
	// +kubebuilder:validation:MinProperties=1
	TargetClusters map[string][]string `json:"targetClusters"`

	// OwnerUserID is the email address of the user who created this graph run
	// +optional
	OwnerUserID string `json:"ownerUserId,omitempty"`

	// ResiliencyScoreEnabled enables resiliency score calculation for this graph run.
	// When enabled:
	// 1. RESILIENCY_SCORE=true environment variable is set in all scenario pods
	// 2. If ResiliencyMountPath is specified, the controller identifies the resiliency metrics file
	//    by matching the mount path in node.Volumes and sets RESILIENCY_FILE=<path> env var
	// 3. Upon completion, the controller calculates the final score and stores it in Status.ResiliencyScores
	//
	// Populated from HTTP header: X-Resiliency-Score
	// When enabled, X-Resiliency-Baseline header is REQUIRED
	// +optional
	ResiliencyScoreEnabled bool `json:"resiliencyScoreEnabled,omitempty"`

	// ResiliencyMountPath is the absolute path where resiliency metrics files are mounted.
	// This path is used to identify which file in each node's Volumes map contains resiliency metrics.
	//
	// File Identification Convention:
	// The resiliency metrics file is identified by its MOUNT PATH, not by file UUID.
	// Each node can use a different file UUID in its Volumes map, but they must all mount
	// to the same path specified here for proper detection.
	//
	// Example:
	//   ResiliencyMountPath: "/etc/kraken/metrics.yaml"
	//   node-1.Volumes: {"uuid-A": "/etc/kraken/metrics.yaml", "uuid-B": "/config.yaml"}
	//   node-2.Volumes: {"uuid-C": "/etc/kraken/metrics.yaml"}  // Different UUID, same path - OK!
	//
	// When a matching mount path is found, the controller sets RESILIENCY_FILE environment variable
	// to this path in the scenario pod.
	//
	// If not specified, pods will only have RESILIENCY_SCORE=true and krkn will use its internal
	// default metrics collection.
	//
	// Populated from HTTP header: X-Resiliency-Mount-Path
	// Must be an absolute path if specified
	// +optional
	ResiliencyMountPath string `json:"resiliencyMountPath,omitempty"`

	// ResiliencyScoreBaseline is the user-defined minimum acceptable resiliency score.
	// After the graph run completes, the calculated score is compared against this baseline:
	// - calculated >= baseline: Status = "pass"
	// - calculated < baseline: Status = "fail"
	// - no baseline: Status = "no-baseline"
	//
	// This value is REQUIRED when ResiliencyScoreEnabled is true.
	//
	// Populated from HTTP header: X-Resiliency-Baseline
	// Must be a non-negative number
	// +optional
	ResiliencyScoreBaseline *float64 `json:"resiliencyScoreBaseline,omitempty"`
}

// KrknGraphRunStatus defines the observed state of KrknGraphRun
type KrknGraphRunStatus struct {
	// Phase is the overall phase of the graph run
	// +kubebuilder:validation:Enum=Pending;Running;Completed;Failed;PartiallyFailed
	Phase string `json:"phase,omitempty"`

	// StartTime is when the graph run started
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is when the graph run completed
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Summary contains aggregate statistics about the graph run
	// +optional
	Summary GraphRunSummary `json:"summary,omitempty"`

	// NodeStatuses contains the status of each node in the graph
	// +optional
	NodeStatuses []NodeStatus `json:"nodeStatuses,omitempty"`

	// ResolvedLevels contains the pre-computed topological levels for frontend rendering
	// Each inner array represents a level of nodes that can run in parallel
	// Example: [["node1", "node2"], ["node3"], ["node4", "node5"]]
	// +optional
	ResolvedLevels [][]string `json:"resolvedLevels,omitempty"`

	// ResiliencyScore contains the calculated resiliency score and baseline comparison.
	// This field is populated by the controller when:
	// 1. Spec.ResiliencyScoreEnabled is true
	// 2. The GraphRun has reached a terminal phase (Completed, Failed, or PartiallyFailed)
	//
	// The score is calculated ONCE and is IMMUTABLE to preserve historical accuracy.
	// Each run represents an invariant result that can be compared over time.
	//
	// When a graph run executes on multiple clusters, this array will contain one entry per cluster.
	// For single-cluster runs, the array will have exactly one element.
	// +optional
	ResiliencyScores []GraphClusterScore `json:"resiliencyScores,omitempty"`

	// Conditions represent the latest available observations of the graph run's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Total",type=integer,JSONPath=`.status.summary.totalNodes`
// +kubebuilder:printcolumn:name="Completed",type=integer,JSONPath=`.status.summary.completedNodes`
// +kubebuilder:printcolumn:name="Failed",type=integer,JSONPath=`.status.summary.failedNodes`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=kgr

// KrknGraphRun is the Schema for the krkngraphruns API
type KrknGraphRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KrknGraphRunSpec   `json:"spec,omitempty"`
	Status KrknGraphRunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KrknGraphRunList contains a list of KrknGraphRun
type KrknGraphRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrknGraphRun `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KrknGraphRun{}, &KrknGraphRunList{})
}
