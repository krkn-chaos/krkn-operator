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
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PaginationClientState tracks per-client pagination state for a resource type.
type PaginationClientState struct {
	Page           int
	Limit          int
	Categories     []string
	GroupRollbacks bool
	Filters        JobsSubscriptionFilters
	SortBy         string
	SortOrder      string
	SubscriptionID string
	LastHash       uint64 // fingerprint of last sent page content
}

// Client represents a connected WebSocket client
type Client struct {
	// WebSocket connection
	conn *websocket.Conn

	// User ID (from JWT claims)
	userID string

	// claims retain provider group values for authorization and revocation.
	claims *auth.Claims

	// Is admin flag
	isAdmin bool

	// Send channel for outbound messages
	send chan []byte

	// Subscriptions: map[resourceType]map[resourceID]bool
	// Example: subscriptions["run"]["run-abc123"] = true
	subscriptions map[string]map[string]bool

	// Pagination state per resource type (e.g., "jobs" → page/limit)
	paginationState map[string]*PaginationClientState

	// Mutex for subscription updates
	mu sync.RWMutex

	// Successful identity checks are briefly reused across outbound messages.
	statusMu         sync.Mutex
	statusValidUntil time.Time
}

// authClaims preserves OIDC provider and group claims when filtering broadcasts.
// Older in-package clients without claims retain the local-user fallback.
func (c *Client) authClaims() *auth.Claims {
	if c.claims != nil {
		return c.claims
	}
	role := string(auth.RoleUser)
	if c.isAdmin {
		role = string(auth.RoleAdmin)
	}
	return &auth.Claims{UserID: c.userID, Role: role}
}

// Hub maintains active clients and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to clients
	broadcast chan *BroadcastMessage

	// Mutex for client map
	mu sync.RWMutex
}

// AuthorizationCheckFunc is called to verify if a client can receive a broadcast message.
// Returns true if client is authorized to receive the message.
type AuthorizationCheckFunc func(client *Client, resourceType string, resourceID string) bool

// BroadcastViewFunc builds the authorized payload for one subscribed client.
// Returning false suppresses the broadcast for that client.
type BroadcastViewFunc func(client *Client) ([]byte, bool)

// BroadcastMessage contains a message and targeting information

type BroadcastMessage struct {
	// Resource type: "run", "graphrun", "dashboard"
	resourceType string

	// Resource ID (empty for global broadcasts like dashboard)
	resourceID string

	// Message payload (JSON-encoded)
	message []byte

	// Optional authorization check function
	// If nil, only subscription check is performed (e.g., for dashboard)
	// If non-nil, called to verify user has permission to receive this message
	authzCheck AuthorizationCheckFunc

	// Optional per-client payload for resources with cluster-specific visibility.
	view BroadcastViewFunc
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 256),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	stopBroadcasts := make(chan struct{})
	go h.runBroadcasts(stopBroadcasts)
	defer close(stopBroadcasts)

	for {
		select {
		case client, ok := <-h.register:
			if !ok {
				return
			}
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		}
	}
}

// runBroadcasts keeps delivery ordered without making slow authorization
// checks block client registration and removal in Run.
func (h *Hub) runBroadcasts(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		case msg := <-h.broadcast:
			h.mu.RLock()
			clients := make([]*Client, 0, len(h.clients))
			for client := range h.clients {
				clients = append(clients, client)
			}
			h.mu.RUnlock()

			for _, client := range clients {
				if !h.shouldSendToClient(client, msg) {
					continue
				}
				payload := msg.message
				if msg.view != nil {
					var allowed bool
					payload, allowed = msg.view(client)
					if !allowed {
						continue
					}
				}
				h.mu.Lock()
				if h.clients[client] && h.clientSubscribedToMessage(client, msg) {
					select {
					case client.send <- payload:
					default:
						// Client's send buffer is full, disconnect.
						close(client.send)
						delete(h.clients, client)
					}
				}
				h.mu.Unlock()
			}
		}
	}
}

// shouldSendToClient checks if a message should be sent to a client
func (h *Hub) shouldSendToClient(client *Client, msg *BroadcastMessage) bool {
	if !h.clientSubscribedToMessage(client, msg) {
		return false
	}

	// Authorization can read Kubernetes; never hold either client or hub locks.
	if msg.authzCheck != nil {
		return msg.authzCheck(client, msg.resourceType, msg.resourceID)
	}

	return true
}

func (h *Hub) clientSubscribedToMessage(client *Client, msg *BroadcastMessage) bool {
	client.mu.RLock()
	defer client.mu.RUnlock()

	// Check if client has subscriptions for this resource type
	resourceSubs, ok := client.subscriptions[msg.resourceType]
	if !ok {
		return false
	}

	// Global broadcasts (empty resourceID) go to all subscribers of the resource type
	if msg.resourceID == "" {
		return len(resourceSubs) > 0
	}

	// Check for wildcard subscription (subscribed to ALL resources of this type)
	return resourceSubs["*"] || resourceSubs[msg.resourceID]
}

// Subscribe adds a resource subscription for a client
// If resourceIDs is empty, subscribes to ALL resources of that type (wildcard)
func (c *Client) Subscribe(resourceType string, resourceIDs []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.subscriptions[resourceType] == nil {
		c.subscriptions[resourceType] = make(map[string]bool)
	}

	// Empty array means subscribe to ALL resources of this type (wildcard)
	if len(resourceIDs) == 0 {
		c.subscriptions[resourceType]["*"] = true
		return
	}

	for _, id := range resourceIDs {
		c.subscriptions[resourceType][id] = true
	}
}

// Unsubscribe removes resource subscriptions for a client.
// Cleans up the resource key and pagination state when no subscriptions remain.
func (c *Client) Unsubscribe(resourceType string, resourceIDs []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.subscriptions[resourceType] == nil {
		return
	}

	for _, id := range resourceIDs {
		delete(c.subscriptions[resourceType], id)
	}

	if len(c.subscriptions[resourceType]) == 0 {
		delete(c.subscriptions, resourceType)
		delete(c.paginationState, resourceType)
	}
}

// ForEachSubscriber calls fn for each client subscribed to the given resource type.
// Snapshots the matching client set under a short lock so fn can do expensive work
// without blocking hub register/unregister.
func (h *Hub) ForEachSubscriber(resourceType string, fn func(*Client)) {
	h.mu.RLock()
	subscribers := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		client.mu.RLock()
		subs, ok := client.subscriptions[resourceType]
		hasActive := ok && len(subs) > 0
		client.mu.RUnlock()
		if hasActive {
			subscribers = append(subscribers, client)
		}
	}
	h.mu.RUnlock()

	for _, client := range subscribers {
		fn(client)
	}
}

// Broadcast sends a message to all subscribed clients
// Does NOT enforce authorization - use BroadcastWithAuthz for resources that require authz checks
// Non-blocking: drops message if broadcast channel is full (prevents informer stalls)
func (h *Hub) Broadcast(resourceType, resourceID string, message []byte) {
	msg := &BroadcastMessage{
		resourceType: resourceType,
		resourceID:   resourceID,
		message:      message,
		authzCheck:   nil, // No authorization check
	}

	select {
	case h.broadcast <- msg:
		// Message enqueued successfully
	default:
		// Channel full - drop message to prevent blocking informer
		log.Log.V(1).Info("Broadcast channel full, dropping message",
			"resource", resourceType,
			"id", resourceID)
	}
}

// BroadcastWithAuthz sends a message to all subscribed AND authorized clients
// The authzCheck function is called for each client to verify permission
// Non-blocking: drops message if broadcast channel is full (prevents informer stalls)
func (h *Hub) BroadcastWithAuthz(resourceType, resourceID string, message []byte, authzCheck AuthorizationCheckFunc) {
	msg := &BroadcastMessage{
		resourceType: resourceType,
		resourceID:   resourceID,
		message:      message,
		authzCheck:   authzCheck,
	}

	select {
	case h.broadcast <- msg:
		// Message enqueued successfully
	default:
		// Channel full - drop message to prevent blocking informer
		log.Log.V(1).Info("Broadcast channel full, dropping message",
			"resource", resourceType,
			"id", resourceID)
	}
}

// BroadcastWithView sends a separate authorized payload to each subscriber.
func (h *Hub) BroadcastWithView(resourceType, resourceID string, view BroadcastViewFunc) {
	msg := &BroadcastMessage{resourceType: resourceType, resourceID: resourceID, view: view}
	select {
	case h.broadcast <- msg:
	default:
		log.Log.V(1).Info("Broadcast channel full, dropping personalized message", "resource", resourceType, "id", resourceID)
	}
}
