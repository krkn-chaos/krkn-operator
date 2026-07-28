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

// Package v2 provides WebSocket-based real-time API endpoints.
// REST endpoints reuse v1 handlers for backward compatibility.
package v2

import (
	"context"

	v2ws "github.com/krkn-chaos/krkn-operator/internal/api/v2/websocket"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Handler manages v2 WebSocket endpoints
// v2 REST endpoints are handled directly by v1 handlers in server.go (no wrapper needed)
type Handler struct {
	// WsHandler handles WebSocket connections (public for server.go routing)
	WsHandler *v2ws.Handler

	// broadcaster sends updates to WebSocket clients
	broadcaster *v2ws.Broadcaster
}

// NewHandler creates a new v2 Handler
// authz provides group-based authorization for filtering WebSocket broadcasts
func NewHandler(k8sClient client.Client, namespace string, authz v2ws.AuthorizationChecker, getTokenGen func(context.Context) (*auth.TokenGenerator, error)) *Handler {
	// Create WebSocket hub and start it
	hub := v2ws.NewHub()
	go hub.Run()

	return &Handler{
		WsHandler:   v2ws.NewHandler(hub, k8sClient, namespace, authz, getTokenGen),
		broadcaster: v2ws.NewBroadcaster(hub, authz, k8sClient, namespace),
	}
}

// GetBroadcaster returns the WebSocket broadcaster
// Controllers use this to send real-time updates
func (h *Handler) GetBroadcaster() *v2ws.Broadcaster {
	return h.broadcaster
}
