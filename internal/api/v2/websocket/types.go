package websocket

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ScenarioRunStatusResponse - copy of internal/api type to avoid import cycle
type ScenarioRunStatusResponse struct {
	ScenarioRunName string      `json:"scenarioRunName"`
	Phase           string      `json:"phase"`
	TotalTargets    int         `json:"totalTargets"`
	SuccessfulJobs  int         `json:"successfulJobs"`
	FailedJobs      int         `json:"failedJobs"`
	RunningJobs     int         `json:"runningJobs"`
	ClusterJobs     interface{} `json:"clusterJobs,omitempty"`
	OwnerUserID     string      `json:"ownerUserId,omitempty"`
	RegistryName    string      `json:"registryName,omitempty"`
	GraphRunName    string      `json:"graphRunName,omitempty"`
	GraphNodeID     string      `json:"graphNodeId,omitempty"`
	CreationTimestamp string     `json:"creationTimestamp,omitempty"`
}

// GraphRunResponse - WebSocket response for GraphRun (same fields as REST API)
type GraphRunResponse struct {
	GraphRunName     string                      `json:"graphRunName"`
	Phase            string                      `json:"phase"`
	Summary          GraphRunSummaryResponse     `json:"summary"`
	NodeStatuses     []NodeStatusResponse        `json:"nodeStatuses,omitempty"`
	ResolvedLevels   [][]string                  `json:"resolvedLevels"`
	StartTime        *metav1.Time                `json:"startTime,omitempty"`
	CompletionTime   *metav1.Time                `json:"completionTime,omitempty"`
	OwnerUserID      string                      `json:"ownerUserId,omitempty"`
	CreationTimestamp string                      `json:"creationTimestamp,omitempty"`
	ResiliencyScores []GraphClusterScoreResponse `json:"resiliencyScores,omitempty"`
}

// NodeStatusResponse represents a node in the graph run
type NodeStatusResponse struct {
	NodeID             string                           `json:"nodeId"`
	NodeName           string                           `json:"nodeName"`
	Phase              string                           `json:"phase"`
	ScenarioRunRef     string                           `json:"scenarioRunRef,omitempty"`
	StartTime          *metav1.Time                     `json:"startTime,omitempty"`
	CompletionTime     *metav1.Time                     `json:"completionTime,omitempty"`
	DependsOn          []string                         `json:"dependsOn,omitempty"`
	Message            string                           `json:"message,omitempty"`
	ResiliencyScores   []ClusterResiliencyScoreResponse `json:"resiliencyScores,omitempty"`
	ResiliencyScoreAvg *float64                         `json:"resiliencyScoreAvg,omitempty"`
}

// ClusterResiliencyScoreResponse represents the resiliency score for a specific cluster
type ClusterResiliencyScoreResponse struct {
	ClusterName string  `json:"clusterName"`
	Score       float64 `json:"score"`
}

// GraphClusterScoreResponse represents the aggregated resiliency score for a cluster in a graph run
type GraphClusterScoreResponse struct {
	ClusterName       string             `json:"clusterName"`
	Calculated        float64            `json:"calculated"`
	Baseline          *float64           `json:"baseline,omitempty"`
	Status            string             `json:"status"`
	Message           string             `json:"message,omitempty"`
	NodeContributions map[string]float64 `json:"nodeContributions,omitempty"`
}

// GraphRunSummaryResponse - copy
type GraphRunSummaryResponse struct {
	TotalNodes     int `json:"totalNodes"`
	CompletedNodes int `json:"completedNodes"`
	RunningNodes   int `json:"runningNodes"`
	FailedNodes    int `json:"failedNodes"`
	PendingNodes   int `json:"pendingNodes"`
}

// ServerMessage is sent from server to client
type ServerMessage struct {
	Resource string      `json:"resource"`     // "run", "graphrun", "dashboard"
	ID       string      `json:"id,omitempty"` // resource ID (empty for dashboard)
	Event    string      `json:"event"`        // "updated", "deleted", "snapshot"
	Data     interface{} `json:"data"`
}

// ClientMessage is sent from client to server
type ClientMessage struct {
	Action   string   `json:"action"`        // "subscribe", "unsubscribe"
	Resource string   `json:"resource"`      // "run", "graphrun", "dashboard"
	IDs      []string `json:"ids,omitempty"` // specific resource IDs (empty = wildcard)
}

// ErrorMessage is sent when an error occurs
type ErrorMessage struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
