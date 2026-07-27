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

package websocket

import (
	"encoding/json"
	"hash/fnv"
	"sync"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Broadcaster handles broadcasting resource updates to WebSocket clients
// Controllers call these methods when resources change
type Broadcaster struct {
	hub *Hub

	// Cache of last sent state fingerprints to avoid duplicate broadcasts
	// Key: resourceType:resourceID (e.g., "run:dummy-scenario-123")
	// Value: hash of the status that was last broadcast
	lastSentCache map[string]uint64
	cacheMu       sync.RWMutex
}

// NewBroadcaster creates a new Broadcaster
func NewBroadcaster(hub *Hub) *Broadcaster {
	return &Broadcaster{
		hub:           hub,
		lastSentCache: make(map[string]uint64),
	}
}

// BroadcastScenarioRunUpdate broadcasts a scenario run update to subscribed clients
func (b *Broadcaster) BroadcastScenarioRunUpdate(scenarioRun *krknv1alpha1.KrknScenarioRun) {
	logger := log.Log.WithName("websocket-broadcast")

	// Calculate fingerprint of current status
	statusData, err := json.Marshal(scenarioRun.Status)
	if err != nil {
		logger.Error(err, "Failed to marshal status for fingerprint", "scenarioRunName", scenarioRun.Name)
		return
	}
	fingerprint := hashBytes(statusData)

	// Check cache - skip if we already sent this exact state
	cacheKey := "run:" + scenarioRun.Name
	b.cacheMu.RLock()
	lastSent, exists := b.lastSentCache[cacheKey]
	b.cacheMu.RUnlock()

	if exists && lastSent == fingerprint {
		logger.V(1).Info("Status unchanged since last broadcast, skipping",
			"runName", scenarioRun.Name,
			"phase", scenarioRun.Status.Phase)
		return
	}

	// Status changed - broadcast and update cache
	// Build the SAME response as REST API
	response := ScenarioRunStatusResponse{
		ScenarioRunName: scenarioRun.Name,
		Phase:           scenarioRun.Status.Phase,
		TotalTargets:    scenarioRun.Status.TotalTargets,
		SuccessfulJobs:  scenarioRun.Status.SuccessfulJobs,
		FailedJobs:      scenarioRun.Status.FailedJobs,
		RunningJobs:     scenarioRun.Status.RunningJobs,
		ClusterJobs:     nil, // WebSocket doesn't send cluster jobs details
		OwnerUserID:     scenarioRun.Spec.OwnerUserID,
		RegistryName:    scenarioRun.Spec.RegistryName,
		GraphRunName:    scenarioRun.Labels["krkn.dev/graph-run"],
		GraphNodeID:     scenarioRun.Labels["krkn.dev/graph-node"],
		CreatedAt:       scenarioRun.CreationTimestamp.Format(time.RFC3339),
	}

	msg := ServerMessage{
		Resource: "run",
		ID:       scenarioRun.Name,
		Event:    "updated",
		Data:     response,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error(err, "Failed to marshal scenario run update", "scenarioRunName", scenarioRun.Name)
		return
	}

	logger.Info("Broadcasting scenario run update", "runName", scenarioRun.Name, "phase", scenarioRun.Status.Phase)
	b.hub.Broadcast("run", scenarioRun.Name, data)

	// Update cache
	b.cacheMu.Lock()
	b.lastSentCache[cacheKey] = fingerprint
	b.cacheMu.Unlock()
}

// BroadcastScenarioRunDetailUpdate broadcasts FULL scenario run update (with clusterJobs) to detail subscribers
func (b *Broadcaster) BroadcastScenarioRunDetailUpdate(scenarioRun *krknv1alpha1.KrknScenarioRun) {
	logger := log.Log.WithName("websocket-broadcast")

	// Calculate fingerprint of current status (same dedup logic as lightweight)
	statusData, err := json.Marshal(scenarioRun.Status)
	if err != nil {
		logger.Error(err, "Failed to marshal status for detail fingerprint", "scenarioRunName", scenarioRun.Name)
		return
	}
	fingerprint := hashBytes(statusData)

	// Check cache - use separate cache key to avoid collision with lightweight broadcast
	cacheKey := "run-detail:" + scenarioRun.Name
	b.cacheMu.RLock()
	lastSent, exists := b.lastSentCache[cacheKey]
	b.cacheMu.RUnlock()

	if exists && lastSent == fingerprint {
		logger.V(1).Info("Detail status unchanged since last broadcast, skipping",
			"runName", scenarioRun.Name,
			"phase", scenarioRun.Status.Phase)
		return
	}

	// Status changed - broadcast FULL response with clusterJobs
	response := buildScenarioRunDetailResponse(scenarioRun)

	msg := ServerMessage{
		Resource: "run-detail",
		ID:       scenarioRun.Name,
		Event:    "updated",
		Data:     response,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error(err, "Failed to marshal scenario run detail update", "scenarioRunName", scenarioRun.Name)
		return
	}

	logger.Info("Broadcasting scenario run detail update",
		"runName", scenarioRun.Name,
		"phase", scenarioRun.Status.Phase,
		"jobs", len(scenarioRun.Status.ClusterJobs))
	b.hub.Broadcast("run-detail", scenarioRun.Name, data)

	// Update cache
	b.cacheMu.Lock()
	b.lastSentCache[cacheKey] = fingerprint
	b.cacheMu.Unlock()
}

// BroadcastGraphRunUpdate broadcasts a graph run update to subscribed clients
func (b *Broadcaster) BroadcastGraphRunUpdate(graphRun *krknv1alpha1.KrknGraphRun) {
	logger := log.Log.WithName("websocket-broadcast")

	// Calculate fingerprint of current status
	statusData, err := json.Marshal(graphRun.Status)
	if err != nil {
		logger.Error(err, "Failed to marshal status for fingerprint", "graphRunName", graphRun.Name)
		return
	}
	fingerprint := hashBytes(statusData)

	// Check cache - skip if we already sent this exact state
	cacheKey := "graphrun:" + graphRun.Name
	b.cacheMu.RLock()
	lastSent, exists := b.lastSentCache[cacheKey]
	b.cacheMu.RUnlock()

	if exists && lastSent == fingerprint {
		logger.V(1).Info("Status unchanged since last broadcast, skipping",
			"graphRunName", graphRun.Name,
			"phase", graphRun.Status.Phase)
		return
	}

	// Status changed - broadcast and update cache
	// Build the SAME response as REST API
	response := GraphRunResponse{
		GraphRunName:   graphRun.Name,
		Phase:          graphRun.Status.Phase,
		Summary: GraphRunSummaryResponse{
			TotalNodes:     graphRun.Status.Summary.TotalNodes,
			CompletedNodes: graphRun.Status.Summary.CompletedNodes,
			RunningNodes:   graphRun.Status.Summary.RunningNodes,
			FailedNodes:    graphRun.Status.Summary.FailedNodes,
			PendingNodes:   graphRun.Status.Summary.PendingNodes,
		},
		NodeStatuses:   nil, // WebSocket doesn't send full node details
		ResolvedLevels: graphRun.Status.ResolvedLevels,
		StartTime:      graphRun.Status.StartTime,
		CompletionTime: graphRun.Status.CompletionTime,
		OwnerUserID:    graphRun.Spec.OwnerUserID,
		CreatedAt:      graphRun.CreationTimestamp.Format(time.RFC3339),
	}

	msg := ServerMessage{
		Resource: "graphrun",
		ID:       graphRun.Name,
		Event:    "updated",
		Data:     response,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error(err, "Failed to marshal graph run update", "graphRunName", graphRun.Name)
		return
	}

	logger.Info("Broadcasting graph run update", "graphRunName", graphRun.Name, "phase", graphRun.Status.Phase)
	b.hub.Broadcast("graphrun", graphRun.Name, data)

	// Update cache
	b.cacheMu.Lock()
	b.lastSentCache[cacheKey] = fingerprint
	b.cacheMu.Unlock()
}

// BroadcastDashboardUpdate broadcasts a dashboard update to all subscribed clients
// dashboardData should be the same format as GET /api/v1/dashboard/active-runs
func (b *Broadcaster) BroadcastDashboardUpdate(dashboardData interface{}) {
	logger := log.Log.WithName("websocket-broadcast")

	msg := ServerMessage{
		Resource: "dashboard",
		Event:    "updated",
		Data:     dashboardData,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error(err, "Failed to marshal dashboard update")
		return
	}

	// Dashboard broadcasts have empty resourceID (global)
	b.hub.Broadcast("dashboard", "", data)
}

// BroadcastScenarioRunDeleted broadcasts a scenario run deletion event
func (b *Broadcaster) BroadcastScenarioRunDeleted(scenarioRunName string) {
	logger := log.Log.WithName("websocket-broadcast")

	msg := ServerMessage{
		Resource: "run",
		ID:       scenarioRunName,
		Event:    "deleted",
		Data:     nil,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error(err, "Failed to marshal scenario run deletion", "scenarioRunName", scenarioRunName)
		return
	}

	b.hub.Broadcast("run", scenarioRunName, data)
}

// BroadcastGraphRunDeleted broadcasts a graph run deletion event
func (b *Broadcaster) BroadcastGraphRunDeleted(graphRunName string) {
	logger := log.Log.WithName("websocket-broadcast")

	msg := ServerMessage{
		Resource: "graphrun",
		ID:       graphRunName,
		Event:    "deleted",
		Data:     nil,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error(err, "Failed to marshal graph run deletion", "graphRunName", graphRunName)
		return
	}

	b.hub.Broadcast("graphrun", graphRunName, data)
}

// hashBytes computes a fast hash of a byte slice for deduplication
func hashBytes(data []byte) uint64 {
	h := fnv.New64a()
	h.Write(data)
	return h.Sum64()
}
