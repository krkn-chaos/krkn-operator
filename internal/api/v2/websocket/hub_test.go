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
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()

	if hub == nil {
		t.Fatal("NewHub returned nil")
	}

	if hub.clients == nil {
		t.Error("Hub clients map not initialized")
	}

	if hub.register == nil {
		t.Error("Hub register channel not initialized")
	}

	if hub.unregister == nil {
		t.Error("Hub unregister channel not initialized")
	}

	if hub.broadcast == nil {
		t.Error("Hub broadcast channel not initialized")
	}
}

func TestClientSubscribe(t *testing.T) {
	client := &Client{
		userID:        "test-user",
		isAdmin:       false,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]map[string]bool),
	}

	// Subscribe to runs
	client.Subscribe("run", []string{"run-1", "run-2"})

	if !client.subscriptions["run"]["run-1"] {
		t.Error("Expected client to be subscribed to run-1")
	}

	if !client.subscriptions["run"]["run-2"] {
		t.Error("Expected client to be subscribed to run-2")
	}

	// Subscribe to graphruns
	client.Subscribe("graphrun", []string{"graphrun-1"})

	if !client.subscriptions["graphrun"]["graphrun-1"] {
		t.Error("Expected client to be subscribed to graphrun-1")
	}
}

func TestClientUnsubscribe(t *testing.T) {
	client := &Client{
		userID:  "test-user",
		isAdmin: false,
		send:    make(chan []byte, 256),
		subscriptions: map[string]map[string]bool{
			"run": {
				"run-1": true,
				"run-2": true,
			},
		},
	}

	// Unsubscribe from run-1
	client.Unsubscribe("run", []string{"run-1"})

	if client.subscriptions["run"]["run-1"] {
		t.Error("Expected client to be unsubscribed from run-1")
	}

	if !client.subscriptions["run"]["run-2"] {
		t.Error("Expected client to still be subscribed to run-2")
	}
}

func TestHubRegisterUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register) // Clean shutdown

	client := &Client{
		userID:        "test-user",
		isAdmin:       false,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]map[string]bool),
	}

	// Register client
	hub.register <- client
	time.Sleep(10 * time.Millisecond) // Give hub time to process

	hub.mu.RLock()
	if !hub.clients[client] {
		t.Error("Expected client to be registered")
	}
	hub.mu.RUnlock()

	// Unregister client
	hub.unregister <- client
	time.Sleep(10 * time.Millisecond)

	hub.mu.RLock()
	if hub.clients[client] {
		t.Error("Expected client to be unregistered")
	}
	hub.mu.RUnlock()
}

func TestShouldSendToClient(t *testing.T) {
	hub := NewHub()

	tests := []struct {
		name         string
		client       *Client
		msg          *BroadcastMessage
		shouldSend   bool
		description  string
	}{
		{
			name: "subscribed to specific resource",
			client: &Client{
				subscriptions: map[string]map[string]bool{
					"run": {"run-1": true},
				},
			},
			msg: &BroadcastMessage{
				resourceType: "run",
				resourceID:   "run-1",
			},
			shouldSend:  true,
			description: "Client subscribed to run-1 should receive updates",
		},
		{
			name: "not subscribed to specific resource",
			client: &Client{
				subscriptions: map[string]map[string]bool{
					"run": {"run-1": true},
				},
			},
			msg: &BroadcastMessage{
				resourceType: "run",
				resourceID:   "run-2",
			},
			shouldSend:  false,
			description: "Client not subscribed to run-2 should not receive updates",
		},
		{
			name: "subscribed to resource type, global broadcast",
			client: &Client{
				subscriptions: map[string]map[string]bool{
					"dashboard": {"anything": true},
				},
			},
			msg: &BroadcastMessage{
				resourceType: "dashboard",
				resourceID:   "", // Global broadcast
			},
			shouldSend:  true,
			description: "Client subscribed to dashboard should receive global updates",
		},
		{
			name: "not subscribed to resource type",
			client: &Client{
				subscriptions: map[string]map[string]bool{
					"run": {"run-1": true},
				},
			},
			msg: &BroadcastMessage{
				resourceType: "graphrun",
				resourceID:   "graphrun-1",
			},
			shouldSend:  false,
			description: "Client not subscribed to graphrun should not receive updates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hub.shouldSendToClient(tt.client, tt.msg)
			if result != tt.shouldSend {
				t.Errorf("%s: expected %v, got %v", tt.description, tt.shouldSend, result)
			}
		})
	}
}

func TestHubBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)

	// Create client subscribed to run-1
	client := &Client{
		userID:  "test-user",
		isAdmin: false,
		send:    make(chan []byte, 256),
		subscriptions: map[string]map[string]bool{
			"run": {"run-1": true},
		},
	}

	// Register client
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Broadcast message
	testMessage := []byte(`{"resource":"run","id":"run-1","event":"updated"}`)
	hub.Broadcast("run", "run-1", testMessage)

	// Check if message was received
	select {
	case msg := <-client.send:
		if string(msg) != string(testMessage) {
			t.Errorf("Expected message %s, got %s", testMessage, msg)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for broadcast message")
	}
}

// Mock conn for testing
type mockConn struct {
	*websocket.Conn
}
