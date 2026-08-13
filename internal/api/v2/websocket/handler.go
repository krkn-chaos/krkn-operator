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
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/gorilla/websocket"
	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	kvstore "github.com/krkn-chaos/krkn-operator/pkg/configstore"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AuthorizationChecker provides group-based authorization for filtering resources.
// Implemented by internal/api.Handler.
type AuthorizationChecker interface {
	FilterScenarioRunsByGroupPermission(
		runs []krknv1alpha1.KrknScenarioRun,
		ctx context.Context,
	) []krknv1alpha1.KrknScenarioRun

	FilterGraphRunsByGroupPermission(
		runs []krknv1alpha1.KrknGraphRun,
		ctx context.Context,
	) []krknv1alpha1.KrknGraphRun
}

// Handler contains dependencies for WebSocket handlers
type Handler struct {
	hub            *Hub
	k8sClient      k8sclient.Client
	namespace      string
	authz          AuthorizationChecker // Group-based authorization
	getTokenGen    func(context.Context) (*auth.TokenGenerator, error)
	upgrader       websocket.Upgrader
	pingInterval   time.Duration
	pongWait       time.Duration
	writeWait      time.Duration
	maxMessageSize int64
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, k8sClient k8sclient.Client, namespace string, authz AuthorizationChecker, getTokenGen func(context.Context) (*auth.TokenGenerator, error)) *Handler {
	return &Handler{
		hub:         hub,
		k8sClient:   k8sClient,
		namespace:   namespace,
		authz:       authz,
		getTokenGen: getTokenGen,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// SECURITY: Intentionally permissive (allows all origins)
				// Rationale:
				// 1. Authentication/authorization is enforced via JWT in Sec-WebSocket-Protocol
				// 2. Invalid/missing JWT → connection refused at upgrade time
				// 3. Valid JWT but insufficient permissions → filtered broadcasts (authz layer)
				// 4. CORS-style origin checks don't prevent token theft attacks
				// 5. Allows legitimate cross-origin use (e.g., separate frontend domain)
				//
				// If stricter origin validation is needed, configure an allowlist here.
				return true
			},
		},
		pingInterval:   54 * time.Second,
		pongWait:       60 * time.Second,
		writeWait:      10 * time.Second,
		maxMessageSize: 512,
	}
}

// HandleWebSocket handles WebSocket connections
//
// @Summary WebSocket real-time updates
// @Description Multiplexed WebSocket for real-time updates across all resources
// @Description
// @Description **Authentication:** JWT token via Sec-WebSocket-Protocol subprotocol
// @Description - JavaScript: `new WebSocket(url, 'access_token.' + jwtToken)`
// @Description - Header: `Sec-WebSocket-Protocol: access_token.<jwt_token>`
// @Description
// @Description **Client → Server Messages (subscribe/unsubscribe):**
// @Description ```json
// @Description {
// @Description   "action": "subscribe",
// @Description   "resource": "run",
// @Description   "ids": ["run-abc123", "run-xyz789"]
// @Description }
// @Description ```
// @Description
// @Description **Resource types:** `run`, `graphrun`, `dashboard`
// @Description
// @Description **Server → Client Messages (updates):**
// @Description ```json
// @Description {
// @Description   "resource": "run",
// @Description   "id": "run-abc123",
// @Description   "event": "updated",
// @Description   "data": { ... }
// @Description }
// @Description ```
// @Description
// @Description **Available endpoints:**
// @Description - `/api/v2/ws/runs` - Subscribe to scenario run updates
// @Description - `/api/v2/ws/graphruns` - Subscribe to graph run updates
// @Description - `/api/v2/ws/dashboard/active-runs` - Subscribe to dashboard updates
// @Tags websocket
// @Accept json
// @Produce json
// @Security BearerAuth
// @Failure 401 {object} websocket.ErrorMessage "Unauthorized - missing or invalid JWT token"
// @Failure 500 {object} websocket.ErrorMessage "Internal server error"
// @Success 101 {object} websocket.ServerMessage "Switching protocols - WebSocket upgrade successful"
// @Router /v2/ws/runs [get]
// @Router /v2/ws/graphruns [get]
// @Router /v2/ws/dashboard/active-runs [get]
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	logger := log.Log.WithName("websocket-v2")

	// Extract JWT token from Sec-WebSocket-Protocol header
	// Format: "access_token.{jwt_token}"
	protocol := r.Header.Get("Sec-WebSocket-Protocol")
	parts := strings.SplitN(protocol, ".", 2)
	if len(parts) != 2 || parts[0] != "access_token" {
		logger.Error(nil, "Invalid WebSocket protocol format",
			"protocol", "access_token.<redacted>", // Don't log raw token
			"client_ip", r.RemoteAddr)
		http.Error(w, "Invalid WebSocket protocol. Expected: access_token.{jwt_token}", http.StatusBadRequest)
		return
	}

	tokenString := parts[1]

	// Get TokenGenerator from SecretManager
	tokenGen, err := h.getTokenGen(r.Context())
	if err != nil {
		logger.Error(err, "Failed to get TokenGenerator")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Validate JWT token
	claims, err := tokenGen.ValidateToken(tokenString)
	if err != nil {
		logger.Error(err, "Invalid WebSocket JWT token",
			"client_ip", r.RemoteAddr)
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	logger.Info("WebSocket authentication successful",
		"userId", claims.UserID,
		"role", claims.Role,
		"path", r.URL.Path,
		"client_ip", r.RemoteAddr)

	// Upgrade HTTP connection to WebSocket
	// NOTE: We must echo back the exact subprotocol sent by the client
	// WebSocket spec requires server to respond with one of the client's proposed protocols
	// Attempting to respond with a different protocol causes browser to reject the connection
	// The JWT is already in the request headers, so echoing it in the response doesn't
	// increase exposure risk (and connections are HTTPS in production)
	h.upgrader.Subprotocols = []string{protocol}
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(err, "Failed to upgrade WebSocket connection",
			"userId", claims.UserID,
			"client_ip", r.RemoteAddr)
		return
	}

	// Create client and register with hub
	client := &Client{
		conn:            conn,
		userID:          claims.UserID,
		isAdmin:         claims.Role == "admin",
		send:            make(chan []byte, 256),
		subscriptions:   make(map[string]map[string]bool),
		paginationState: make(map[string]*PaginationClientState),
	}

	h.hub.register <- client

	logger.Info("WebSocket client registered",
		"userId", claims.UserID,
		"isAdmin", client.isAdmin,
		"path", r.URL.Path,
		"client_ip", r.RemoteAddr)

	// Start goroutines for reading and writing
	go h.writePump(client)
	go h.readPump(client)
}

// readPump reads messages from the WebSocket connection
func (h *Handler) readPump(client *Client) {
	defer func() {
		h.hub.unregister <- client
		_ = client.conn.Close()
	}()

	logger := log.Log.WithName("websocket-read")

	_ = client.conn.SetReadDeadline(time.Now().Add(h.pongWait))
	client.conn.SetReadLimit(h.maxMessageSize)
	client.conn.SetPongHandler(func(string) error {
		_ = client.conn.SetReadDeadline(time.Now().Add(h.pongWait))
		return nil
	})

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error(err, "WebSocket read error",
					"userId", client.userID)
			}
			logger.Info("WebSocket client disconnected",
				"userId", client.userID)
			break
		}

		// Parse client message (subscribe/unsubscribe)
		var msg ClientMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			h.sendError(client, "invalid_json", "Invalid JSON message")
			continue
		}

		h.handleClientMessage(client, &msg)
	}
}

// writePump writes messages to the WebSocket connection
func (h *Handler) writePump(client *Client) {
	ticker := time.NewTicker(h.pingInterval)
	defer func() {
		ticker.Stop()
		_ = client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			_ = client.conn.SetWriteDeadline(time.Now().Add(h.writeWait))
			if !ok {
				// Hub closed the channel
				_ = client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			_ = client.conn.SetWriteDeadline(time.Now().Add(h.writeWait))
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
			"run":        true,
			"run-detail": true, // Full ScenarioRun with clusterJobs
			"graphrun":   true,
			"dashboard":  true,
			"jobs":       true, // Unified paginated view
		}
		if !validResources[msg.Resource] {
			h.sendError(client, "invalid_resource", "Invalid resource type. Valid: run, run-detail, graphrun, dashboard, jobs")
			return
		}

		// Subscribe client
		client.Subscribe(msg.Resource, msg.IDs)

		// Store pagination state for "jobs" subscriptions
		if msg.Resource == "jobs" {
			page := 1
			limit := h.getDefaultPageSize()
			if msg.Page != nil && *msg.Page > 0 {
				page = *msg.Page
			}
			if msg.Limit != nil && *msg.Limit > 0 {
				limit = *msg.Limit
			}
			client.mu.Lock()
			client.paginationState["jobs"] = &PaginationClientState{
				Page:  page,
				Limit: limit,
			}
			client.mu.Unlock()
		}

		logger.Info("Client subscribed",
			"userId", client.userID,
			"resource", msg.Resource,
			"ids", msg.IDs)

		// Send initial snapshot of current state
		h.sendInitialSnapshot(client, msg.Resource, msg.IDs)

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

// roleFromAdmin converts isAdmin boolean to auth.Role string
func roleFromAdmin(isAdmin bool) string {
	if isAdmin {
		return string(auth.RoleAdmin)
	}
	return string(auth.RoleUser)
}

// getDefaultPageSize reads the default page size from configstore.
func (h *Handler) getDefaultPageSize() int {
	if val, ok := kvstore.Get().GetValue("jobs.defaultPageSize"); ok {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			return n
		}
	}
	return 20
}

// sendInitialSnapshot sends the current state of resources to a newly subscribed client
func (h *Handler) sendInitialSnapshot(client *Client, resourceType string, resourceIDs []string) {
	// Skip snapshot if k8sClient is not available (e.g., in tests)
	if h.k8sClient == nil {
		return
	}

	logger := log.Log.WithName("websocket-snapshot")

	// CRITICAL: Create context with user claims for authorization filtering
	// Without claims, the filter functions cannot enforce group-based permissions
	claims := &auth.Claims{
		UserID: client.userID,
		Role:   roleFromAdmin(client.isAdmin),
	}
	ctx := context.WithValue(context.Background(), auth.UserClaimsKey, claims)

	switch resourceType {
	case "run":
		h.sendScenarioRunsSnapshot(ctx, client, resourceIDs, logger)
	case "run-detail":
		h.sendScenarioRunDetailSnapshot(ctx, client, resourceIDs, logger)
	case "graphrun":
		h.sendGraphRunsSnapshot(ctx, client, resourceIDs, logger)
	case "jobs":
		h.sendJobsSnapshot(ctx, client, logger)
	case "dashboard":
		// Dashboard is a global subscription - no snapshot needed as it will be broadcasted
		logger.V(1).Info("Dashboard subscription - no initial snapshot needed")
	}
}

// sendScenarioRunsSnapshot sends current scenario runs to the client
func (h *Handler) sendScenarioRunsSnapshot(ctx context.Context, client *Client, resourceIDs []string, logger logr.Logger) {
	var runs krknv1alpha1.KrknScenarioRunList
	if err := h.k8sClient.List(ctx, &runs, k8sclient.InNamespace(h.namespace)); err != nil {
		logger.Error(err, "Failed to list scenario runs for snapshot")
		return
	}

	// AUTHORIZATION: Filter runs based on user's group permissions
	// Admins see all runs, regular users only see runs they have 'view' permission for
	filteredRuns := h.authz.FilterScenarioRunsByGroupPermission(runs.Items, ctx)

	// If specific IDs requested, filter to those
	// If empty (wildcard), send all authorized runs
	for i := range filteredRuns {
		run := &filteredRuns[i]

		// Skip ScenarioRuns that are part of a GraphRun (client subscribes to GraphRun instead)
		if graphRunName := run.Labels["krkn.dev/graph-run"]; graphRunName != "" {
			logger.V(2).Info("Skipping GraphRun node in snapshot", "runName", run.Name, "graphRun", graphRunName)
			continue
		}

		// Check if we should send this run
		shouldSend := len(resourceIDs) == 0 // wildcard
		if !shouldSend {
			for _, id := range resourceIDs {
				if run.Name == id {
					shouldSend = true
					break
				}
			}
		}

		if !shouldSend {
			continue
		}

		// Send this run as an initial snapshot (using "updated" event for compatibility)
		// Build the SAME response as REST API
		response := buildScenarioRunResponse(run)

		msg := ServerMessage{
			Resource: "run",
			ID:       run.Name,
			Event:    "updated",
			Data:     response,
		}

		data, err := json.Marshal(msg)
		if err != nil {
			logger.Error(err, "Failed to marshal scenario run snapshot", "runName", run.Name)
			continue
		}

		select {
		case client.send <- data:
			logger.Info("Sent initial snapshot", "resource", "run", "id", run.Name, "phase", run.Status.Phase)
		default:
			logger.Error(nil, "Client buffer full, dropping snapshot", "runName", run.Name)
		}
	}
}

// sendScenarioRunDetailSnapshot sends full scenario run details (with clusterJobs) to the client
func (h *Handler) sendScenarioRunDetailSnapshot(ctx context.Context, client *Client, resourceIDs []string, logger logr.Logger) {
	var runs krknv1alpha1.KrknScenarioRunList
	if err := h.k8sClient.List(ctx, &runs, k8sclient.InNamespace(h.namespace)); err != nil {
		logger.Error(err, "Failed to list scenario runs for detail snapshot")
		return
	}

	// AUTHORIZATION: Filter runs based on user's group permissions
	filteredRuns := h.authz.FilterScenarioRunsByGroupPermission(runs.Items, ctx)

	// For run-detail, specific IDs are expected (not wildcard)
	for i := range filteredRuns {
		run := &filteredRuns[i]

		// Skip ScenarioRuns that are part of a GraphRun
		if graphRunName := run.Labels["krkn.dev/graph-run"]; graphRunName != "" {
			logger.V(2).Info("Skipping GraphRun node in detail snapshot", "runName", run.Name, "graphRun", graphRunName)
			continue
		}

		// Check if we should send this run (match specific IDs)
		shouldSend := false
		for _, id := range resourceIDs {
			if run.Name == id {
				shouldSend = true
				break
			}
		}

		if !shouldSend {
			continue
		}

		// Build FULL response with clusterJobs
		response := buildScenarioRunDetailResponse(run)

		msg := ServerMessage{
			Resource: "run-detail",
			ID:       run.Name,
			Event:    "updated",
			Data:     response,
		}

		data, err := json.Marshal(msg)
		if err != nil {
			logger.Error(err, "Failed to marshal scenario run detail snapshot", "runName", run.Name)
			continue
		}

		select {
		case client.send <- data:
			logger.Info("Sent detail snapshot", "resource", "run-detail", "id", run.Name, "jobs", len(run.Status.ClusterJobs))
		default:
			logger.Error(nil, "Client buffer full, dropping detail snapshot", "runName", run.Name)
		}
	}
}

// sendGraphRunsSnapshot sends current graph runs to the client
func (h *Handler) sendGraphRunsSnapshot(ctx context.Context, client *Client, resourceIDs []string, logger logr.Logger) {
	var runs krknv1alpha1.KrknGraphRunList
	if err := h.k8sClient.List(ctx, &runs, k8sclient.InNamespace(h.namespace)); err != nil {
		logger.Error(err, "Failed to list graph runs for snapshot")
		return
	}

	// AUTHORIZATION: Filter runs based on user's group permissions
	// Admins see all runs, regular users only see runs they have 'view' permission for
	filteredRuns := h.authz.FilterGraphRunsByGroupPermission(runs.Items, ctx)

	// If specific IDs requested, filter to those
	// If empty (wildcard), send all authorized runs
	for i := range filteredRuns {
		run := &filteredRuns[i]

		// Check if we should send this run
		shouldSend := len(resourceIDs) == 0 // wildcard
		if !shouldSend {
			for _, id := range resourceIDs {
				if run.Name == id {
					shouldSend = true
					break
				}
			}
		}

		if !shouldSend {
			continue
		}

		// Send this run as an initial snapshot (using "updated" event for compatibility)
		// Build the SAME response as REST API
		response := buildGraphRunResponse(run)

		msg := ServerMessage{
			Resource: "graphrun",
			ID:       run.Name,
			Event:    "updated",
			Data:     response,
		}

		data, err := json.Marshal(msg)
		if err != nil {
			logger.Error(err, "Failed to marshal graph run snapshot", "runName", run.Name)
			continue
		}

		select {
		case client.send <- data:
			logger.Info("Sent initial snapshot", "resource", "graphrun", "id", run.Name, "phase", run.Status.Phase)
		default:
			logger.Error(nil, "Client buffer full, dropping snapshot", "runName", run.Name)
		}
	}
}

// sendJobsSnapshot sends a paginated unified jobs snapshot to the client.
func (h *Handler) sendJobsSnapshot(ctx context.Context, client *Client, logger logr.Logger) {
	// List ScenarioRuns
	var scenarioRuns krknv1alpha1.KrknScenarioRunList
	if err := h.k8sClient.List(ctx, &scenarioRuns, k8sclient.InNamespace(h.namespace)); err != nil {
		logger.Error(err, "Failed to list scenario runs for jobs snapshot")
		return
	}

	// List GraphRuns
	var graphRuns krknv1alpha1.KrknGraphRunList
	if err := h.k8sClient.List(ctx, &graphRuns, k8sclient.InNamespace(h.namespace)); err != nil {
		logger.Error(err, "Failed to list graph runs for jobs snapshot")
		return
	}

	// Apply authorization filters
	filteredScenarioRuns := h.authz.FilterScenarioRunsByGroupPermission(scenarioRuns.Items, ctx)
	filteredGraphRuns := h.authz.FilterGraphRunsByGroupPermission(graphRuns.Items, ctx)

	// Build unified sorted list
	allJobs := buildUnifiedJobList(filteredScenarioRuns, filteredGraphRuns)

	// Get pagination state
	client.mu.RLock()
	ps := client.paginationState["jobs"]
	client.mu.RUnlock()

	page := 1
	limit := h.getDefaultPageSize()
	if ps != nil {
		page = ps.Page
		limit = ps.Limit
	}

	// Compute stats from the full list before pagination
	stats := computeWSJobStats(allJobs)

	// Paginate
	pageItems, meta := paginateJobItems(allJobs, page, limit)

	snapshot := WSUnifiedJobsSnapshot{Jobs: pageItems, Stats: stats}
	msg := ServerMessage{
		Resource:   "jobs",
		Event:      "snapshot",
		Data:       snapshot,
		Pagination: &meta,
		Stats:      &stats,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error(err, "Failed to marshal jobs snapshot")
		return
	}

	// Update fingerprint
	fp := hashBytes(data)
	client.mu.Lock()
	if client.paginationState["jobs"] != nil {
		client.paginationState["jobs"].LastHash = fp
	}
	client.mu.Unlock()

	select {
	case client.send <- data:
		logger.Info("Sent jobs snapshot", "page", meta.Page, "limit", meta.Limit, "total", meta.Total, "items", len(pageItems))
	default:
		logger.Error(nil, "Client buffer full, dropping jobs snapshot")
	}
}
