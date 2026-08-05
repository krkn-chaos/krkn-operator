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

// Package websocket provides real-time WebSocket broadcasting for Kubernetes resources.
// It implements an Informer-based architecture with Hub/Client/Broadcaster patterns.
package websocket

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"sync"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Broadcaster handles broadcasting resource updates to WebSocket clients
// Controllers call these methods when resources change
type Broadcaster struct {
	hub   *Hub
	authz AuthorizationChecker
	// k8sClient is needed to fetch full resources for authorization checks
	k8sClient k8sclient.Client
	namespace string

	// Cache of last sent state fingerprints to avoid duplicate broadcasts
	// Key: resourceType:resourceID (e.g., "run:dummy-scenario-123")
	// Value: hash of the status that was last broadcast
	lastSentCache map[string]uint64
	cacheMu       sync.RWMutex
}

// NewBroadcaster creates a new Broadcaster
func NewBroadcaster(hub *Hub, authz AuthorizationChecker, k8sClient k8sclient.Client, namespace string) *Broadcaster {
	return &Broadcaster{
		hub:           hub,
		authz:         authz,
		k8sClient:     k8sClient,
		namespace:     namespace,
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
	response := buildScenarioRunResponse(scenarioRun)

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

	// Create authorization check function for this broadcast
	// Admins see all updates, regular users only see runs they have 'view' permission for
	authzCheck := b.makeScenarioRunAuthzCheck(scenarioRun)
	b.hub.BroadcastWithAuthz("run", scenarioRun.Name, data, authzCheck)

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

	// AUTHORIZATION: Apply same check as lightweight broadcast
	authzCheck := b.makeScenarioRunAuthzCheck(scenarioRun)
	b.hub.BroadcastWithAuthz("run-detail", scenarioRun.Name, data, authzCheck)

	// Update cache
	b.cacheMu.Lock()
	b.lastSentCache[cacheKey] = fingerprint
	b.cacheMu.Unlock()
}

// BroadcastGraphRunUpdate broadcasts a graph run update to subscribed clients
// event should be "created" for OnAdd or "updated" for OnUpdate
func (b *Broadcaster) BroadcastGraphRunUpdate(graphRun *krknv1alpha1.KrknGraphRun, event string) {
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
		GraphRunName: graphRun.Name,
		Phase:        graphRun.Status.Phase,
		Summary: GraphRunSummaryResponse{
			TotalNodes:     graphRun.Status.Summary.TotalNodes,
			CompletedNodes: graphRun.Status.Summary.CompletedNodes,
			RunningNodes:   graphRun.Status.Summary.RunningNodes,
			FailedNodes:    graphRun.Status.Summary.FailedNodes,
			PendingNodes:   graphRun.Status.Summary.PendingNodes,
		},
		NodeStatuses:      convertNodeStatusesWithScores(graphRun.Status.NodeStatuses, graphRun.Status.ResiliencyScores),
		ResolvedLevels:    graphRun.Status.ResolvedLevels,
		StartTime:         graphRun.Status.StartTime,
		CompletionTime:    graphRun.Status.CompletionTime,
		OwnerUserID:       graphRun.Spec.OwnerUserID,
		CreationTimestamp: graphRun.CreationTimestamp.Format(time.RFC3339),
		ResiliencyScores:  b.convertGraphClusterScores(graphRun.Status.ResiliencyScores),
	}

	msg := ServerMessage{
		Resource: "graphrun",
		ID:       graphRun.Name,
		Event:    event, // Passed as parameter from watcher
		Data:     response,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error(err, "Failed to marshal graph run update", "graphRunName", graphRun.Name)
		return
	}

	logger.Info("Broadcasting graph run update", "graphRunName", graphRun.Name, "phase", graphRun.Status.Phase)

	// AUTHORIZATION: Filter by user's group permissions
	authzCheck := b.makeGraphRunAuthzCheck(graphRun)
	b.hub.BroadcastWithAuthz("graphrun", graphRun.Name, data, authzCheck)

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
// scenarioRun is the resource BEFORE deletion (needed for authorization check)
func (b *Broadcaster) BroadcastScenarioRunDeleted(scenarioRun *krknv1alpha1.KrknScenarioRun) {
	logger := log.Log.WithName("websocket-broadcast")

	msg := ServerMessage{
		Resource: "run",
		ID:       scenarioRun.Name,
		Event:    "deleted",
		Data:     nil,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error(err, "Failed to marshal scenario run deletion", "scenarioRunName", scenarioRun.Name)
		return
	}

	// AUTHORIZATION: Only send deletion to users who had permission to view the run
	authzCheck := b.makeScenarioRunAuthzCheck(scenarioRun)
	b.hub.BroadcastWithAuthz("run", scenarioRun.Name, data, authzCheck)

	// Clean up cache entry
	cacheKey := "run:" + scenarioRun.Name
	b.cacheMu.Lock()
	delete(b.lastSentCache, cacheKey)
	b.cacheMu.Unlock()
}

// BroadcastGraphRunDeleted broadcasts a graph run deletion event
// graphRun is the resource BEFORE deletion (needed for authorization check)
func (b *Broadcaster) BroadcastGraphRunDeleted(graphRun *krknv1alpha1.KrknGraphRun) {
	logger := log.Log.WithName("websocket-broadcast")

	msg := ServerMessage{
		Resource: "graphrun",
		ID:       graphRun.Name,
		Event:    "deleted",
		Data:     nil,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error(err, "Failed to marshal graph run deletion", "graphRunName", graphRun.Name)
		return
	}

	// AUTHORIZATION: Only send deletion to users who had permission to view the run
	authzCheck := b.makeGraphRunAuthzCheck(graphRun)
	b.hub.BroadcastWithAuthz("graphrun", graphRun.Name, data, authzCheck)

	// Clean up cache entry
	cacheKey := "graphrun:" + graphRun.Name
	b.cacheMu.Lock()
	delete(b.lastSentCache, cacheKey)
	b.cacheMu.Unlock()
}

// hashBytes computes a fast hash of a byte slice for deduplication
func hashBytes(data []byte) uint64 {
	h := fnv.New64a()
	_, _ = h.Write(data) // fnv.Hash.Write never returns an error
	return h.Sum64()
}

// makeScenarioRunAuthzCheck creates an authorization check function for a scenario run broadcast.
// The returned function checks if a user has 'view' permission on the scenario run.
func (b *Broadcaster) makeScenarioRunAuthzCheck(scenarioRun *krknv1alpha1.KrknScenarioRun) AuthorizationCheckFunc {
	return func(userID string, isAdmin bool, resourceType string, resourceID string) bool {
		// Admins bypass all checks
		if isAdmin {
			return true
		}

		// Create a context with claims for the filter function
		// The filter function needs claims in context to extract userID and role
		claims := &auth.Claims{
			UserID: userID,
			Role:   string(auth.RoleUser), // non-admin
		}
		ctx := context.WithValue(context.Background(), auth.UserClaimsKey, claims)

		// Filter the run using authorization checker
		// This simulates what the REST endpoint does
		filtered := b.authz.FilterScenarioRunsByGroupPermission(
			[]krknv1alpha1.KrknScenarioRun{*scenarioRun},
			ctx,
		)

		// If the run was filtered out, user doesn't have permission
		return len(filtered) > 0
	}
}

// makeGraphRunAuthzCheck creates an authorization check function for a graph run broadcast.
// The returned function checks if a user has 'view' permission on the graph run.
func (b *Broadcaster) makeGraphRunAuthzCheck(graphRun *krknv1alpha1.KrknGraphRun) AuthorizationCheckFunc {
	return func(userID string, isAdmin bool, resourceType string, resourceID string) bool {
		// Admins bypass all checks
		if isAdmin {
			return true
		}

		claims := &auth.Claims{
			UserID: userID,
			Role:   string(auth.RoleUser),
		}
		ctx := context.WithValue(context.Background(), auth.UserClaimsKey, claims)

		// Filter the run using authorization checker
		filtered := b.authz.FilterGraphRunsByGroupPermission(
			[]krknv1alpha1.KrknGraphRun{*graphRun},
			ctx,
		)

		// If the run was filtered out, user doesn't have permission
		return len(filtered) > 0
	}
}

// convertNodeStatusesWithScores converts Kubernetes NodeStatus to WebSocket response format
// and enriches each node with its resiliency scores derived from GraphClusterScore.NodeContributions
func convertNodeStatusesWithScores(nodeStatuses []krknv1alpha1.NodeStatus, graphScores []krknv1alpha1.GraphClusterScore) []NodeStatusResponse {
	if nodeStatuses == nil {
		return []NodeStatusResponse{}
	}

	// Precompute nodeID → per-cluster scores from GraphClusterScore.NodeContributions
	nodeScoreMap := make(map[string][]ClusterResiliencyScoreResponse)
	for _, gs := range graphScores {
		for nodeID, score := range gs.NodeContributions {
			nodeScoreMap[nodeID] = append(nodeScoreMap[nodeID], ClusterResiliencyScoreResponse{
				ClusterName: gs.ClusterName,
				Score:       score,
			})
		}
	}

	result := make([]NodeStatusResponse, 0, len(nodeStatuses))
	for _, ns := range nodeStatuses {
		response := NodeStatusResponse{
			NodeID:         ns.NodeID,
			NodeName:       ns.NodeName,
			Phase:          ns.Phase,
			ScenarioRunRef: ns.ScenarioRunRef,
			StartTime:      ns.StartTime,
			CompletionTime: ns.CompletionTime,
			DependsOn:      ns.DependsOn,
			Message:        ns.Message,
		}

		if scores, ok := nodeScoreMap[ns.NodeID]; ok {
			response.ResiliencyScores = scores
			var sum float64
			for _, s := range scores {
				sum += s.Score
			}
			avg := sum / float64(len(scores))
			response.ResiliencyScoreAvg = &avg
		}

		result = append(result, response)
	}

	return result
}

// convertGraphClusterScores converts GraphClusterScore array to WebSocket response format
func (b *Broadcaster) convertGraphClusterScores(scores []krknv1alpha1.GraphClusterScore) []GraphClusterScoreResponse {
	if scores == nil {
		return nil
	}
	result := make([]GraphClusterScoreResponse, len(scores))
	for i, score := range scores {
		result[i] = GraphClusterScoreResponse{
			ProviderName:      score.ProviderName,
			ClusterName:       score.ClusterName,
			Calculated:        score.Calculated,
			Baseline:          score.Baseline,
			Status:            score.Status,
			Message:           score.Message,
			NodeContributions: score.NodeContributions,
		}
	}
	return result
}
