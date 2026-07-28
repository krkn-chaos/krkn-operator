package websocket

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ScenarioRunStatusResponse - copy of internal/api type to avoid import cycle
type ScenarioRunStatusResponse struct {
	ScenarioRunName string `json:"scenarioRunName"`
	Phase           string `json:"phase"`
	TotalTargets    int    `json:"totalTargets"`
	SuccessfulJobs  int    `json:"successfulJobs"`
	FailedJobs      int    `json:"failedJobs"`
	RunningJobs     int    `json:"runningJobs"`
	ClusterJobs     interface{} `json:"clusterJobs,omitempty"`
	OwnerUserID     string `json:"ownerUserId,omitempty"`
	RegistryName    string `json:"registryName,omitempty"`
	GraphRunName    string `json:"graphRunName,omitempty"`
	GraphNodeID     string `json:"graphNodeId,omitempty"`
	CreatedAt       string `json:"createdAt,omitempty"`
}

// GraphRunResponse - WebSocket response for GraphRun (same fields as REST API)
type GraphRunResponse struct {
	GraphRunName    string                   `json:"graphRunName"`
	Phase           string                   `json:"phase"`
	Summary         GraphRunSummaryResponse  `json:"summary"`
	NodeStatuses    interface{}              `json:"nodeStatuses,omitempty"`
	ResolvedLevels  [][]string               `json:"resolvedLevels"`
	StartTime       *metav1.Time             `json:"startTime,omitempty"`
	CompletionTime  *metav1.Time             `json:"completionTime,omitempty"`
	OwnerUserID     string                   `json:"ownerUserId,omitempty"`
	CreatedAt       string                   `json:"createdAt,omitempty"`
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
	Resource string      `json:"resource"` // "run", "graphrun", "dashboard"
	ID       string      `json:"id,omitempty"` // resource ID (empty for dashboard)
	Event    string      `json:"event"` // "updated", "deleted", "snapshot"
	Data     interface{} `json:"data"`
}

// ClientMessage is sent from client to server
type ClientMessage struct {
	Action   string   `json:"action"` // "subscribe", "unsubscribe"
	Resource string   `json:"resource"` // "run", "graphrun", "dashboard"
	IDs      []string `json:"ids,omitempty"` // specific resource IDs (empty = wildcard)
}

// ErrorMessage is sent when an error occurs
type ErrorMessage struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
