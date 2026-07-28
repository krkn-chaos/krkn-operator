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
	"testing"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBroadcastScenarioRunUpdate(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)

	broadcaster := NewBroadcaster(hub, &mockAuthzChecker{}, nil, "default")

	// Create a subscribed client
	client := &Client{
		userID:  "test-user",
		isAdmin: false,
		send:    make(chan []byte, 256),
		subscriptions: map[string]map[string]bool{
			"run": {"scenario-run-1": true},
		},
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Create test scenario run
	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scenario-run-1",
			Namespace: "default",
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase:        "Running",
			TotalTargets: 3,
			RunningJobs:  2,
		},
	}

	// Broadcast update
	broadcaster.BroadcastScenarioRunUpdate(scenarioRun)

	// Verify client received message
	select {
	case msg := <-client.send:
		var serverMsg ServerMessage
		if err := json.Unmarshal(msg, &serverMsg); err != nil {
			t.Fatalf("Failed to unmarshal server message: %v", err)
		}

		if serverMsg.Resource != "run" {
			t.Errorf("Expected resource 'run', got '%s'", serverMsg.Resource)
		}

		if serverMsg.ID != "scenario-run-1" {
			t.Errorf("Expected ID 'scenario-run-1', got '%s'", serverMsg.ID)
		}

		if serverMsg.Event != "updated" {
			t.Errorf("Expected event 'updated', got '%s'", serverMsg.Event)
		}

	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for broadcast message")
	}
}

func TestBroadcastGraphRunUpdate(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)

	broadcaster := NewBroadcaster(hub, &mockAuthzChecker{}, nil, "default")

	// Create a subscribed client
	client := &Client{
		userID:  "test-user",
		isAdmin: false,
		send:    make(chan []byte, 256),
		subscriptions: map[string]map[string]bool{
			"graphrun": {"graphrun-1": true},
		},
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Create test graph run
	graphRun := &krknv1alpha1.KrknGraphRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "graphrun-1",
			Namespace: "default",
		},
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
			Summary: krknv1alpha1.GraphRunSummary{
				TotalNodes:     5,
				CompletedNodes: 2,
				RunningNodes:   3,
			},
		},
	}

	// Broadcast update
	broadcaster.BroadcastGraphRunUpdate(graphRun)

	// Verify client received message
	select {
	case msg := <-client.send:
		var serverMsg ServerMessage
		if err := json.Unmarshal(msg, &serverMsg); err != nil {
			t.Fatalf("Failed to unmarshal server message: %v", err)
		}

		if serverMsg.Resource != "graphrun" {
			t.Errorf("Expected resource 'graphrun', got '%s'", serverMsg.Resource)
		}

		if serverMsg.ID != "graphrun-1" {
			t.Errorf("Expected ID 'graphrun-1', got '%s'", serverMsg.ID)
		}

		if serverMsg.Event != "updated" {
			t.Errorf("Expected event 'updated', got '%s'", serverMsg.Event)
		}

	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for broadcast message")
	}
}

func TestBroadcastDashboardUpdate(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)

	broadcaster := NewBroadcaster(hub, &mockAuthzChecker{}, nil, "default")

	// Create a subscribed client (subscribed to any dashboard updates)
	client := &Client{
		userID:  "test-user",
		isAdmin: false,
		send:    make(chan []byte, 256),
		subscriptions: map[string]map[string]bool{
			"dashboard": {"anything": true}, // Dashboard uses global broadcasts
		},
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Create test dashboard data
	dashboardData := map[string]interface{}{
		"totalActiveRuns": 5,
		"totalClusters":   3,
	}

	// Broadcast update
	broadcaster.BroadcastDashboardUpdate(dashboardData)

	// Verify client received message
	select {
	case msg := <-client.send:
		var serverMsg ServerMessage
		if err := json.Unmarshal(msg, &serverMsg); err != nil {
			t.Fatalf("Failed to unmarshal server message: %v", err)
		}

		if serverMsg.Resource != "dashboard" {
			t.Errorf("Expected resource 'dashboard', got '%s'", serverMsg.Resource)
		}

		if serverMsg.ID != "" {
			t.Errorf("Expected empty ID for dashboard broadcast, got '%s'", serverMsg.ID)
		}

		if serverMsg.Event != "updated" {
			t.Errorf("Expected event 'updated', got '%s'", serverMsg.Event)
		}

	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for broadcast message")
	}
}

func TestBroadcastScenarioRunDeleted(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)

	broadcaster := NewBroadcaster(hub, &mockAuthzChecker{}, nil, "default")

	// Create a subscribed client
	client := &Client{
		userID:  "test-user",
		isAdmin: false,
		send:    make(chan []byte, 256),
		subscriptions: map[string]map[string]bool{
			"run": {"scenario-run-1": true},
		},
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Create test scenario run for deletion
	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "scenario-run-1",
		},
	}

	// Broadcast deletion
	broadcaster.BroadcastScenarioRunDeleted(scenarioRun)

	// Verify client received message
	select {
	case msg := <-client.send:
		var serverMsg ServerMessage
		if err := json.Unmarshal(msg, &serverMsg); err != nil {
			t.Fatalf("Failed to unmarshal server message: %v", err)
		}

		if serverMsg.Resource != "run" {
			t.Errorf("Expected resource 'run', got '%s'", serverMsg.Resource)
		}

		if serverMsg.ID != "scenario-run-1" {
			t.Errorf("Expected ID 'scenario-run-1', got '%s'", serverMsg.ID)
		}

		if serverMsg.Event != "deleted" {
			t.Errorf("Expected event 'deleted', got '%s'", serverMsg.Event)
		}

		if serverMsg.Data != nil {
			t.Error("Expected nil data for deletion event")
		}

	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for broadcast message")
	}
}

func TestBroadcastGraphRunDeleted(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)

	broadcaster := NewBroadcaster(hub, &mockAuthzChecker{}, nil, "default")

	// Create a subscribed client
	client := &Client{
		userID:  "test-user",
		isAdmin: false,
		send:    make(chan []byte, 256),
		subscriptions: map[string]map[string]bool{
			"graphrun": {"graphrun-1": true},
		},
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Create test graph run for deletion
	graphRun := &krknv1alpha1.KrknGraphRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "graphrun-1",
		},
	}

	// Broadcast deletion
	broadcaster.BroadcastGraphRunDeleted(graphRun)

	// Verify client received message
	select {
	case msg := <-client.send:
		var serverMsg ServerMessage
		if err := json.Unmarshal(msg, &serverMsg); err != nil {
			t.Fatalf("Failed to unmarshal server message: %v", err)
		}

		if serverMsg.Resource != "graphrun" {
			t.Errorf("Expected resource 'graphrun', got '%s'", serverMsg.Resource)
		}

		if serverMsg.ID != "graphrun-1" {
			t.Errorf("Expected ID 'graphrun-1', got '%s'", serverMsg.ID)
		}

		if serverMsg.Event != "deleted" {
			t.Errorf("Expected event 'deleted', got '%s'", serverMsg.Event)
		}

		if serverMsg.Data != nil {
			t.Error("Expected nil data for deletion event")
		}

	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for broadcast message")
	}
}

func TestBroadcastOnlyToSubscribedClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)

	broadcaster := NewBroadcaster(hub, &mockAuthzChecker{}, nil, "default")

	// Create two clients: one subscribed, one not
	subscribedClient := &Client{
		userID:  "subscribed-user",
		isAdmin: false,
		send:    make(chan []byte, 256),
		subscriptions: map[string]map[string]bool{
			"run": {"scenario-run-1": true},
		},
	}

	unsubscribedClient := &Client{
		userID:  "unsubscribed-user",
		isAdmin: false,
		send:    make(chan []byte, 256),
		subscriptions: map[string]map[string]bool{
			"run": {"scenario-run-2": true}, // Different run
		},
	}

	hub.register <- subscribedClient
	hub.register <- unsubscribedClient
	time.Sleep(10 * time.Millisecond)

	// Broadcast update for scenario-run-1
	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "scenario-run-1",
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
		},
	}

	broadcaster.BroadcastScenarioRunUpdate(scenarioRun)

	// Subscribed client should receive message
	select {
	case <-subscribedClient.send:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Subscribed client did not receive message")
	}

	// Unsubscribed client should NOT receive message
	select {
	case <-unsubscribedClient.send:
		t.Error("Unsubscribed client should not receive message")
	case <-time.After(50 * time.Millisecond):
		// Expected - no message
	}
}
