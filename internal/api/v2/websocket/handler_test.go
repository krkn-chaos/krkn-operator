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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

func TestHandleClientMessage_Subscribe(t *testing.T) {
	hub := NewHub()
	handler := NewHandler(hub, nil, "test-namespace", mockTokenGen)

	client := &Client{
		userID:        "test-user",
		isAdmin:       false,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]map[string]bool),
	}

	msg := &ClientMessage{
		Action:   "subscribe",
		Resource: "run",
		IDs:      []string{"run-1", "run-2"},
	}

	handler.handleClientMessage(client, msg)

	if !client.subscriptions["run"]["run-1"] {
		t.Error("Expected client to be subscribed to run-1")
	}

	if !client.subscriptions["run"]["run-2"] {
		t.Error("Expected client to be subscribed to run-2")
	}
}

func TestHandleClientMessage_Unsubscribe(t *testing.T) {
	hub := NewHub()
	handler := NewHandler(hub, nil, "test-namespace", mockTokenGen)

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

	msg := &ClientMessage{
		Action:   "unsubscribe",
		Resource: "run",
		IDs:      []string{"run-1"},
	}

	handler.handleClientMessage(client, msg)

	if client.subscriptions["run"]["run-1"] {
		t.Error("Expected client to be unsubscribed from run-1")
	}

	if !client.subscriptions["run"]["run-2"] {
		t.Error("Expected client to still be subscribed to run-2")
	}
}

func TestHandleClientMessage_InvalidResource(t *testing.T) {
	hub := NewHub()
	handler := NewHandler(hub, nil, "test-namespace", mockTokenGen)

	client := &Client{
		userID:        "test-user",
		isAdmin:       false,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]map[string]bool),
	}

	msg := &ClientMessage{
		Action:   "subscribe",
		Resource: "invalid-resource",
		IDs:      []string{"id-1"},
	}

	handler.handleClientMessage(client, msg)

	// Should receive error message
	select {
	case errMsg := <-client.send:
		var errResponse ErrorMessage
		if err := json.Unmarshal(errMsg, &errResponse); err != nil {
			t.Fatalf("Failed to unmarshal error message: %v", err)
		}

		if errResponse.Error != "invalid_resource" {
			t.Errorf("Expected error code 'invalid_resource', got '%s'", errResponse.Error)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected error message to be sent")
	}
}

func TestHandleClientMessage_InvalidAction(t *testing.T) {
	hub := NewHub()
	handler := NewHandler(hub, nil, "test-namespace", mockTokenGen)

	client := &Client{
		userID:        "test-user",
		isAdmin:       false,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]map[string]bool),
	}

	msg := &ClientMessage{
		Action:   "invalid-action",
		Resource: "run",
		IDs:      []string{"run-1"},
	}

	handler.handleClientMessage(client, msg)

	// Should receive error message
	select {
	case errMsg := <-client.send:
		var errResponse ErrorMessage
		if err := json.Unmarshal(errMsg, &errResponse); err != nil {
			t.Fatalf("Failed to unmarshal error message: %v", err)
		}

		if errResponse.Error != "invalid_action" {
			t.Errorf("Expected error code 'invalid_action', got '%s'", errResponse.Error)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected error message to be sent")
	}
}

func TestHandleWebSocket_MissingAuth(t *testing.T) {
	hub := NewHub()
	handler := NewHandler(hub, nil, "test-namespace", mockTokenGen)

	req := httptest.NewRequest("GET", "/api/v2/ws/runs", nil)
	w := httptest.NewRecorder()

	handler.HandleWebSocket(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Invalid WebSocket protocol") {
		t.Errorf("Expected error message about invalid protocol, got: %s", body)
	}
}

func TestHandleWebSocket_InvalidProtocolFormat(t *testing.T) {
	hub := NewHub()
	handler := NewHandler(hub, nil, "test-namespace", mockTokenGen)

	req := httptest.NewRequest("GET", "/api/v2/ws/runs", nil)
	req.Header.Set("Sec-WebSocket-Protocol", "invalid-format")
	w := httptest.NewRecorder()

	handler.HandleWebSocket(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Invalid WebSocket protocol") {
		t.Errorf("Expected error message about invalid protocol, got: %s", body)
	}
}

func TestHandleWebSocket_InvalidToken(t *testing.T) {
	hub := NewHub()
	handler := NewHandler(hub, nil, "test-namespace", mockTokenGenInvalid)

	req := httptest.NewRequest("GET", "/api/v2/ws/runs", nil)
	req.Header.Set("Sec-WebSocket-Protocol", "access_token.invalid-token")
	w := httptest.NewRecorder()

	handler.HandleWebSocket(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Invalid or expired token") {
		t.Error("Expected error message about invalid token")
	}
}

// Mock token generator for testing
func mockTokenGen(ctx context.Context) (*auth.TokenGenerator, error) {
	secret := []byte("test-secret-key-32-bytes-long!!")
	tokenGen := auth.NewTokenGenerator(secret, 24*time.Hour, "krkn-operator-test")
	return tokenGen, nil
}

func mockTokenGenInvalid(ctx context.Context) (*auth.TokenGenerator, error) {
	// Use different secret to make tokens invalid
	secret := []byte("different-secret-key-32-bytes!")
	tokenGen := auth.NewTokenGenerator(secret, 24*time.Hour, "krkn-operator-test")
	return tokenGen, nil
}

func TestClientMessageJSON(t *testing.T) {
	msg := ClientMessage{
		Action:   "subscribe",
		Resource: "run",
		IDs:      []string{"run-1", "run-2"},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal ClientMessage: %v", err)
	}

	var decoded ClientMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ClientMessage: %v", err)
	}

	if decoded.Action != msg.Action {
		t.Errorf("Expected action %s, got %s", msg.Action, decoded.Action)
	}

	if decoded.Resource != msg.Resource {
		t.Errorf("Expected resource %s, got %s", msg.Resource, decoded.Resource)
	}

	if len(decoded.IDs) != len(msg.IDs) {
		t.Errorf("Expected %d IDs, got %d", len(msg.IDs), len(decoded.IDs))
	}
}

func TestServerMessageJSON(t *testing.T) {
	msg := ServerMessage{
		Resource: "run",
		ID:       "run-1",
		Event:    "updated",
		Data:     map[string]string{"status": "Running"},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal ServerMessage: %v", err)
	}

	var decoded ServerMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ServerMessage: %v", err)
	}

	if decoded.Resource != msg.Resource {
		t.Errorf("Expected resource %s, got %s", msg.Resource, decoded.Resource)
	}

	if decoded.ID != msg.ID {
		t.Errorf("Expected ID %s, got %s", msg.ID, decoded.ID)
	}

	if decoded.Event != msg.Event {
		t.Errorf("Expected event %s, got %s", msg.Event, decoded.Event)
	}
}

func TestErrorMessageJSON(t *testing.T) {
	msg := ErrorMessage{
		Error:   "invalid_action",
		Message: "Action must be subscribe or unsubscribe",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal ErrorMessage: %v", err)
	}

	var decoded ErrorMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ErrorMessage: %v", err)
	}

	if decoded.Error != msg.Error {
		t.Errorf("Expected error %s, got %s", msg.Error, decoded.Error)
	}

	if decoded.Message != msg.Message {
		t.Errorf("Expected message %s, got %s", msg.Message, decoded.Message)
	}
}
