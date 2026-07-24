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

// Package websocket provides multiplexed WebSocket support for real-time API v2.
//
// # Overview
//
// API v2 introduces WebSocket endpoints that allow clients to subscribe to multiple
// resources and receive real-time updates without polling.
//
// # Endpoints
//
//   - /api/v2/ws/runs - Subscribe to scenario run updates
//   - /api/v2/ws/graphruns - Subscribe to graph run updates
//   - /api/v2/ws/dashboard/active-runs - Subscribe to dashboard updates
//
// # Authentication
//
// WebSocket endpoints use JWT authentication via the Sec-WebSocket-Protocol header:
//
//	// JavaScript/TypeScript
//	const token = "eyJhbGc..."; // JWT from /api/v1/auth/login
//	const ws = new WebSocket("ws://localhost:8080/api/v2/ws/runs", `access_token.${token}`);
//
// # Client → Server Messages
//
// Subscribe to resources:
//
//	{
//	  "action": "subscribe",
//	  "resource": "run",
//	  "ids": ["run-abc123", "run-def456"]
//	}
//
// Unsubscribe from resources:
//
//	{
//	  "action": "unsubscribe",
//	  "resource": "run",
//	  "ids": ["run-abc123"]
//	}
//
// # Server → Client Messages
//
// Resource updated:
//
//	{
//	  "resource": "run",
//	  "id": "run-abc123",
//	  "event": "updated",
//	  "data": { /* ScenarioRunStatus */ }
//	}
//
// Resource deleted:
//
//	{
//	  "resource": "run",
//	  "id": "run-abc123",
//	  "event": "deleted",
//	  "data": null
//	}
//
// # Migration from v1
//
// Instead of polling:
//
//	// v1 - Polling every 2 seconds
//	setInterval(async () => {
//	  const res = await fetch('/api/v1/scenarios/run/run-abc123');
//	  const data = await res.json();
//	  updateUI(data);
//	}, 2000);
//
// Use WebSocket subscription:
//
//	// v2 - Real-time WebSocket
//	const ws = new WebSocket('ws://localhost:8080/api/v2/ws/runs', `access_token.${token}`);
//	ws.onopen = () => {
//	  ws.send(JSON.stringify({
//	    action: "subscribe",
//	    resource: "run",
//	    ids: ["run-abc123", "run-def456"]
//	  }));
//	};
//	ws.onmessage = (event) => {
//	  const msg = JSON.parse(event.data);
//	  if (msg.event === "updated") {
//	    updateUI(msg.data);
//	  }
//	};
package websocket

// ClientMessage represents a message from client to server
type ClientMessage struct {
	// Action to perform: "subscribe" or "unsubscribe"
	Action string `json:"action"`

	// Resource type: "run", "graphrun", "dashboard"
	Resource string `json:"resource"`

	// IDs to subscribe/unsubscribe (empty for dashboard)
	IDs []string `json:"ids,omitempty"`
}

// ServerMessage represents a message from server to client
type ServerMessage struct {
	// Resource type: "run", "graphrun", "dashboard"
	Resource string `json:"resource"`

	// ID of the resource (empty for dashboard broadcasts)
	ID string `json:"id,omitempty"`

	// Event type: "updated", "deleted"
	Event string `json:"event"`

	// Data payload (resource-specific)
	Data interface{} `json:"data"`
}

// ErrorMessage represents an error message from server to client
type ErrorMessage struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
