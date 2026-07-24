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
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Handler contains dependencies for WebSocket handlers
type Handler struct {
	hub           *Hub
	tokenGen      *auth.TokenGenerator
	getTokenGen   func(context.Context) (*auth.TokenGenerator, error)
	upgrader      websocket.Upgrader
	pingInterval  time.Duration
	pongWait      time.Duration
	writeWait     time.Duration
	maxMessageSize int64
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, getTokenGen func(context.Context) (*auth.TokenGenerator, error)) *Handler {
	return &Handler{
		hub:         hub,
		getTokenGen: getTokenGen,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins - in production validate origin
				return true
			},
			Subprotocols: []string{"access_token"},
		},
		pingInterval:   30 * time.Second,
		pongWait:       60 * time.Second,
		writeWait:      10 * time.Second,
		maxMessageSize: 512, // Client messages are small (subscribe/unsubscribe)
	}
}

// HandleWebSocket handles WebSocket connections with JWT authentication
//
// @Summary WebSocket real-time updates
// @Description Multiplexed WebSocket endpoint for real-time resource updates. Supports scenario runs, graph runs, and dashboard.
// @Description
// @Description **Authentication:** JWT token via WebSocket subprotocol header:
// @Description - JavaScript: `new WebSocket(url, 'access_token.' + jwtToken)`
// @Description - Header: `Sec-WebSocket-Protocol: access_token.<jwt_token>`
// @Description
// @Description **Client → Server Messages:**
// @Description ```json
// @Description {
// @Description   "action": "subscribe",
// @Description   "resource": "run|graphrun|dashboard",
// @Description   "ids": ["run-abc123", "run-def456"]
// @Description }
// @Description ```
// @Description
// @Description **Server → Client Messages:**
// @Description ```json
// @Description {
// @Description   "resource": "run|graphrun|dashboard",
// @Description   "id": "run-abc123",
// @Description   "event": "updated|deleted",
// @Description   "data": { ... }
// @Description }
// @Description ```
// @Description
// @Description **Endpoints:**
// @Description - `/api/v2/ws/runs` - Subscribe to scenario run updates
// @Description - `/api/v2/ws/graphruns` - Subscribe to graph run updates
// @Description - `/api/v2/ws/dashboard/active-runs` - Subscribe to dashboard updates
// @Tags websocket
// @Accept json
// @Produce json
// @Success 101 {string} string "Switching Protocols"
// @Failure 401 {object} websocket.ErrorMessage "Unauthorized - missing or invalid JWT token"
// @Failure 500 {object} websocket.ErrorMessage "Internal server error"
// @Security BearerAuth
// @Router /v2/ws/runs [get]
// @Router /v2/ws/graphruns [get]
// @Router /v2/ws/dashboard/active-runs [get]
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	logger := log.Log.WithName("websocket-v2")

	// Extract and validate JWT token from subprotocol
	protocols := r.Header.Get("Sec-WebSocket-Protocol")
	if protocols == "" {
		logger.Info("WebSocket authentication failed: missing Sec-WebSocket-Protocol header",
			"path", r.URL.Path,
			"client_ip", r.RemoteAddr)
		http.Error(w, "Unauthorized: Missing Sec-WebSocket-Protocol header", http.StatusUnauthorized)
		return
	}

	// Parse protocol: "access_token.<jwt_token>"
	protocolParts := strings.SplitN(protocols, ".", 2)
	if len(protocolParts) != 2 || protocolParts[0] != "access_token" {
		logger.Info("WebSocket authentication failed: invalid protocol format",
			"path", r.URL.Path,
			"protocol", protocols,
			"client_ip", r.RemoteAddr)
		http.Error(w, "Unauthorized: Invalid Sec-WebSocket-Protocol format. Expected: access_token.<jwt>", http.StatusUnauthorized)
		return
	}

	token := protocolParts[1]
	if token == "" {
		logger.Info("WebSocket authentication failed: empty token",
			"path", r.URL.Path,
			"client_ip", r.RemoteAddr)
		http.Error(w, "Unauthorized: Missing authentication token", http.StatusUnauthorized)
		return
	}

	// Validate JWT token
	tokenGen, err := h.getTokenGen(r.Context())
	if err != nil {
		logger.Error(err, "Failed to get TokenGenerator for WebSocket auth")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	claims, err := tokenGen.ValidateToken(token)
	if err != nil {
		logger.Info("WebSocket authentication failed: invalid token",
			"path", r.URL.Path,
			"error", err.Error(),
			"client_ip", r.RemoteAddr)
		http.Error(w, "Unauthorized: Invalid or expired token", http.StatusUnauthorized)
		return
	}

	logger.Info("WebSocket authentication successful",
		"userId", claims.UserID,
		"role", claims.Role,
		"path", r.URL.Path,
		"client_ip", r.RemoteAddr)

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, http.Header{
		"Sec-WebSocket-Protocol": []string{protocols},
	})
	if err != nil {
		logger.Error(err, "WebSocket upgrade failed",
			"url", r.URL.String(),
			"client_ip", r.RemoteAddr)
		return
	}

	// Create client
	client := &Client{
		conn:          conn,
		userID:        claims.UserID,
		isAdmin:       claims.Role == "admin",
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]map[string]bool),
	}

	// Register client with hub
	h.hub.register <- client

	logger.Info("WebSocket client registered",
		"userId", claims.UserID,
		"isAdmin", client.isAdmin,
		"path", r.URL.Path,
		"client_ip", r.RemoteAddr)

	// Start client read and write pumps
	go h.writePump(client)
	go h.readPump(client)
}

// readPump reads messages from the WebSocket connection
func (h *Handler) readPump(client *Client) {
	logger := log.Log.WithName("websocket-read")

	defer func() {
		h.hub.unregister <- client
		client.conn.Close()
		logger.Info("WebSocket client disconnected", "userId", client.userID)
	}()

	client.conn.SetReadDeadline(time.Now().Add(h.pongWait))
	client.conn.SetReadLimit(h.maxMessageSize)
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(h.pongWait))
		return nil
	})

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error(err, "WebSocket read error", "userId", client.userID)
			}
			break
		}

		// Parse client message
		var msg ClientMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			logger.Info("Invalid client message", "userId", client.userID, "error", err.Error())
			h.sendError(client, "invalid_message", "Invalid JSON message format")
			continue
		}

		// Handle client action
		h.handleClientMessage(client, &msg)
	}
}

// writePump writes messages to the WebSocket connection
func (h *Handler) writePump(client *Client) {
	ticker := time.NewTicker(h.pingInterval)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(h.writeWait))
			if !ok {
				// Hub closed the channel
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(h.writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleClientMessage processes subscribe/unsubscribe requests
func (h *Handler) handleClientMessage(client *Client, msg *ClientMessage) {
	logger := log.Log.WithName("websocket-handler")

	switch msg.Action {
	case "subscribe":
		if msg.Resource == "" {
			h.sendError(client, "invalid_request", "resource field is required")
			return
		}

		// Validate resource type
		validResources := map[string]bool{
			"run":       true,
			"graphrun":  true,
			"dashboard": true,
		}
		if !validResources[msg.Resource] {
			h.sendError(client, "invalid_resource", "Invalid resource type. Valid: run, graphrun, dashboard")
			return
		}

		// Subscribe client
		client.Subscribe(msg.Resource, msg.IDs)

		logger.Info("Client subscribed",
			"userId", client.userID,
			"resource", msg.Resource,
			"ids", msg.IDs)

	case "unsubscribe":
		if msg.Resource == "" {
			h.sendError(client, "invalid_request", "resource field is required")
			return
		}

		client.Unsubscribe(msg.Resource, msg.IDs)

		logger.Info("Client unsubscribed",
			"userId", client.userID,
			"resource", msg.Resource,
			"ids", msg.IDs)

	default:
		h.sendError(client, "invalid_action", "Invalid action. Valid: subscribe, unsubscribe")
	}
}

// sendError sends an error message to a client
func (h *Handler) sendError(client *Client, errCode, errMsg string) {
	errResponse := ErrorMessage{
		Error:   errCode,
		Message: errMsg,
	}

	data, err := json.Marshal(errResponse)
	if err != nil {
		return
	}

	select {
	case client.send <- data:
	default:
		// Client buffer full, ignore
	}
}
