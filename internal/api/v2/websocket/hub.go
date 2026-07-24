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

	"github.com/gorilla/websocket"
)

// Client represents a connected WebSocket client
type Client struct {
	// WebSocket connection
	conn *websocket.Conn

	// User ID (from JWT claims)
	userID string

	// Is admin flag
	isAdmin bool

	// Send channel for outbound messages
	send chan []byte

	// Subscriptions: map[resourceType]map[resourceID]bool
	// Example: subscriptions["run"]["run-abc123"] = true
	subscriptions map[string]map[string]bool

	// Mutex for subscription updates
	mu sync.RWMutex
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

// BroadcastMessage contains a message and targeting information
type BroadcastMessage struct {
	// Resource type: "run", "graphrun", "dashboard"
	resourceType string

	// Resource ID (empty for global broadcasts like dashboard)
	resourceID string

	// Message payload (JSON-encoded)
	message []byte
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
	for {
		select {
		case client := <-h.register:
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

		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				// Check if client is subscribed to this resource
				if h.shouldSendToClient(client, msg) {
					select {
					case client.send <- msg.message:
					default:
						// Client's send buffer is full, disconnect
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// shouldSendToClient checks if a message should be sent to a client
func (h *Hub) shouldSendToClient(client *Client, msg *BroadcastMessage) bool {
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

	// Check if client is subscribed to this specific resource ID
	return resourceSubs[msg.resourceID]
}

// Subscribe adds a resource subscription for a client
func (c *Client) Subscribe(resourceType string, resourceIDs []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.subscriptions[resourceType] == nil {
		c.subscriptions[resourceType] = make(map[string]bool)
	}

	for _, id := range resourceIDs {
		c.subscriptions[resourceType][id] = true
	}
}

// Unsubscribe removes resource subscriptions for a client
func (c *Client) Unsubscribe(resourceType string, resourceIDs []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.subscriptions[resourceType] == nil {
		return
	}

	for _, id := range resourceIDs {
		delete(c.subscriptions[resourceType], id)
	}
}

// Broadcast sends a message to all subscribed clients
func (h *Hub) Broadcast(resourceType, resourceID string, message []byte) {
	h.broadcast <- &BroadcastMessage{
		resourceType: resourceType,
		resourceID:   resourceID,
		message:      message,
	}
}
