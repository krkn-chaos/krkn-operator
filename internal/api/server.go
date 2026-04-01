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
*/

// Package api provides HTTP API handlers and server implementation for the krkn-operator.
// It includes endpoints for authentication, target management, scenario execution, and user management.
package api

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

// Server represents the REST API server
type Server struct {
	server         *http.Server
	handler        *Handler
	authMiddleware *auth.Middleware
}

// NewServer creates a new API server
func NewServer(port int, client client.Client, clientset kubernetes.Interface, namespace string, grpcServerAddr string) *Server {
	handler := NewHandler(client, clientset, namespace, grpcServerAddr)

	// Create auth middleware with lazy JWT secret loading
	// The secret will be loaded on first request when the cache is ready
	getTokenGen := func() *auth.TokenGenerator {
		jwtSecret, err := handler.getOrCreateJWTSecret(context.Background())
		if err != nil {
			log.Log.Error(err, "Failed to get JWT secret, using fallback")
			jwtSecret = []byte("fallback-secret-key-change-this-immediately")
		}
		return auth.NewTokenGenerator(jwtSecret, TokenDuration, "krkn-operator")
	}
	authMw := auth.NewLazyMiddleware(getTokenGen)

	mux := http.NewServeMux()

	// Public authentication endpoints (no auth required)
	mux.HandleFunc(AuthIsRegistered, handler.IsRegistered)
	mux.HandleFunc(AuthRegister, handler.Register)
	mux.HandleFunc(AuthLogin, handler.Login)

	// Authenticated endpoints - user and admin access
	mux.Handle(HealthPath, authMw.RequireAuth(http.HandlerFunc(handler.HealthCheck)))
	mux.Handle(ClustersPath, authMw.RequireAuth(http.HandlerFunc(handler.GetClusters)))
	mux.Handle(NodesPath, authMw.RequireAuth(http.HandlerFunc(handler.GetNodes)))
	mux.Handle(TargetsPath, authMw.RequireAuth(http.HandlerFunc(handler.TargetsHandler)))
	mux.Handle(TargetsPath+"/", authMw.RequireAuth(http.HandlerFunc(handler.TargetsHandler)))

	// Scenario endpoints - user and admin access
	mux.Handle(ScenariosPath, authMw.RequireAuth(http.HandlerFunc(handler.PostScenarios)))
	mux.Handle(ScenariosDetailPath+"/", authMw.RequireAuth(http.HandlerFunc(handler.PostScenarioDetail)))
	mux.Handle(ScenariosGlobalsPath+"/", authMw.RequireAuth(http.HandlerFunc(handler.PostScenarioGlobals)))

	// WebSocket endpoint for log streaming - handles JWT auth internally via Sec-WebSocket-Protocol
	// MUST be registered BEFORE the catch-all ScenariosRunPath to match first
	mux.HandleFunc(ScenariosRunPath+"/", func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a WebSocket logs request
		if strings.Contains(r.URL.Path, "/jobs/") && strings.HasSuffix(r.URL.Path, "/logs") {
			// WebSocket endpoint - auth handled internally via subprotocol
			handler.GetScenarioRunLogs(w, r)
			return
		}
		// All other ScenariosRunPath endpoints require HTTP JWT auth
		authMw.RequireAuth(http.HandlerFunc(handler.ScenariosRunRouter)).ServeHTTP(w, r)
	})

	// Scenario run endpoints - user and admin access
	mux.Handle(ScenariosRunPath, authMw.RequireAuth(http.HandlerFunc(handler.ScenariosRunRouter)))

	// Dashboard endpoints - user and admin access
	mux.Handle(DashboardActiveRunsPath, authMw.RequireAuth(http.HandlerFunc(handler.GetActiveRunsOverview)))

	// User management endpoints - authenticated users
	mux.Handle(UsersPath, authMw.RequireAuth(http.HandlerFunc(handler.UsersRouter)))
	mux.Handle(UsersPath+"/", authMw.RequireAuth(http.HandlerFunc(handler.UsersRouter)))

	// User group management endpoints - admin only
	mux.Handle(GroupsPath, authMw.RequireAuth(http.HandlerFunc(handler.GroupsRouter)))
	mux.Handle(GroupsPath+"/", authMw.RequireAuth(http.HandlerFunc(handler.GroupsRouter)))

	// Provider config endpoints - admin only (POST), user and admin (GET)
	// Note: handler.ProviderConfigHandler internally handles method-based authorization
	mux.Handle(ProviderConfigPath, authMw.RequireAuth(http.HandlerFunc(handler.ProviderConfigHandler)))
	mux.Handle(ProviderConfigPath+"/", authMw.RequireAuth(http.HandlerFunc(handler.ProviderConfigHandler)))

	// Provider endpoints - GET: user and admin, PATCH: admin only
	// Note: handler.ProvidersRouter internally handles method-based authorization
	mux.Handle(ProvidersPath, authMw.RequireAuth(http.HandlerFunc(handler.ProvidersRouter)))
	mux.Handle(ProvidersPath+"/", authMw.RequireAuth(http.HandlerFunc(handler.ProvidersRouter)))

	// Target CRUD endpoints - GET: user and admin, POST/PUT/DELETE: admin only
	// Note: handler.TargetsCRUDRouter internally handles method-based authorization
	mux.Handle(OperatorTargetsPath, authMw.RequireAuth(http.HandlerFunc(handler.TargetsCRUDRouter)))
	mux.Handle(OperatorTargetsPath+"/", authMw.RequireAuth(http.HandlerFunc(handler.TargetsCRUDRouter)))

	// Wrap mux with logging middleware
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 30 * time.Second,  // Prevent Slowloris attacks
		ReadTimeout:       60 * time.Second,  // Total request read timeout
		WriteTimeout:      60 * time.Second,  // Response write timeout
		IdleTimeout:       120 * time.Second, // Keep-alive timeout
	}

	return &Server{
		server:         server,
		handler:        handler,
		authMiddleware: authMw,
	}
}

// Start starts the API server
func (s *Server) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting REST API server", "addr", s.server.Addr)

	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return s.Shutdown()
	}
}

// Shutdown gracefully shuts down the API server
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// loggingMiddleware is a logging middleware for HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		logger := log.Log.WithName("api")
		logger.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"status", rw.statusCode,
			"duration", time.Since(start),
			"client_ip", r.RemoteAddr,
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Hijack implements http.Hijacker interface for WebSocket support
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("ResponseWriter does not implement http.Hijacker")
	}
	return hijacker.Hijack()
}
