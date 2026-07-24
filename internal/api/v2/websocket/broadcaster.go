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

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Broadcaster handles broadcasting resource updates to WebSocket clients
// Controllers call these methods when resources change
type Broadcaster struct {
	hub *Hub
}

// NewBroadcaster creates a new Broadcaster
func NewBroadcaster(hub *Hub) *Broadcaster {
	return &Broadcaster{
		hub: hub,
	}
}

// BroadcastScenarioRunUpdate broadcasts a scenario run update to subscribed clients
func (b *Broadcaster) BroadcastScenarioRunUpdate(scenarioRun *krknv1alpha1.KrknScenarioRun) {
	logger := log.Log.WithName("websocket-broadcast")

	// Convert to response format (reuse existing API types)
	// We'll import from internal/api package to reuse ScenarioRunStatusResponse
	msg := ServerMessage{
		Resource: "run",
		ID:       scenarioRun.Name,
		Event:    "updated",
		Data:     scenarioRun.Status, // Controllers will pass full status
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error(err, "Failed to marshal scenario run update", "scenarioRunName", scenarioRun.Name)
		return
	}

	b.hub.Broadcast("run", scenarioRun.Name, data)
}

// BroadcastGraphRunUpdate broadcasts a graph run update to subscribed clients
func (b *Broadcaster) BroadcastGraphRunUpdate(graphRun *krknv1alpha1.KrknGraphRun) {
	logger := log.Log.WithName("websocket-broadcast")

	msg := ServerMessage{
		Resource: "graphrun",
		ID:       graphRun.Name,
		Event:    "updated",
		Data:     graphRun.Status, // Controllers will pass full status
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error(err, "Failed to marshal graph run update", "graphRunName", graphRun.Name)
		return
	}

	b.hub.Broadcast("graphrun", graphRun.Name, data)
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
