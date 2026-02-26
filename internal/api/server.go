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

package api

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
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

	// Initialize JWT secret and auth middleware
	jwtSecret, err := handler.getOrCreateJWTSecret(context.Background())
	if err != nil {
		log.Log.Error(err, "Failed to initialize JWT secret, authentication will not work")
		jwtSecret = []byte("fallback-secret-key-change-this-immediately")
	}

	tokenGen := auth.NewTokenGenerator(jwtSecret, TokenDuration, "krkn-operator")
	authMw := auth.NewMiddleware(tokenGen)

	mux := http.NewServeMux()

	// Public authentication endpoints (no auth required)
	mux.HandleFunc("/api/v1/auth/is-registered", handler.IsRegistered)
	mux.HandleFunc("/api/v1/auth/register", handler.Register)
	mux.HandleFunc("/api/v1/auth/login", handler.Login)

	// Authenticated endpoints - user and admin access
	mux.Handle("/api/v1/health", authMw.RequireAuth(http.HandlerFunc(handler.HealthCheck)))
	mux.Handle("/api/v1/clusters", authMw.RequireAuth(http.HandlerFunc(handler.GetClusters)))
	mux.Handle("/api/v1/nodes", authMw.RequireAuth(http.HandlerFunc(handler.GetNodes)))
	mux.Handle("/api/v1/targets", authMw.RequireAuth(http.HandlerFunc(handler.TargetsHandler)))
	mux.Handle("/api/v1/targets/", authMw.RequireAuth(http.HandlerFunc(handler.TargetsHandler)))

	// Scenario endpoints - user and admin access
	mux.Handle("/api/v1/scenarios", authMw.RequireAuth(http.HandlerFunc(handler.PostScenarios)))
	mux.Handle("/api/v1/scenarios/detail/", authMw.RequireAuth(http.HandlerFunc(handler.PostScenarioDetail)))
	mux.Handle("/api/v1/scenarios/globals/", authMw.RequireAuth(http.HandlerFunc(handler.PostScenarioGlobals)))
	mux.Handle("/api/v1/scenarios/run", authMw.RequireAuth(http.HandlerFunc(handler.ScenariosRunRouter)))
	mux.Handle("/api/v1/scenarios/run/", authMw.RequireAuth(http.HandlerFunc(handler.ScenariosRunRouter)))

	// Provider config endpoints - admin only (POST), user and admin (GET)
	// Note: handler.ProviderConfigHandler internally handles method-based authorization
	mux.Handle("/api/v1/provider-config", authMw.RequireAuth(http.HandlerFunc(handler.ProviderConfigHandler)))
	mux.Handle("/api/v1/provider-config/", authMw.RequireAuth(http.HandlerFunc(handler.ProviderConfigHandler)))

	// Provider endpoints - GET: user and admin, PATCH: admin only
	// Note: handler.ProvidersRouter internally handles method-based authorization
	mux.Handle("/api/v1/providers", authMw.RequireAuth(http.HandlerFunc(handler.ProvidersRouter)))
	mux.Handle("/api/v1/providers/", authMw.RequireAuth(http.HandlerFunc(handler.ProvidersRouter)))

	// Target CRUD endpoints - GET: user and admin, POST/PUT/DELETE: admin only
	// Note: handler.TargetsCRUDRouter internally handles method-based authorization
	mux.Handle("/api/v1/operator/targets", authMw.RequireAuth(http.HandlerFunc(handler.TargetsCRUDRouter)))
	mux.Handle("/api/v1/operator/targets/", authMw.RequireAuth(http.HandlerFunc(handler.TargetsCRUDRouter)))

	// Wrap mux with logging middleware
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: loggingMiddleware(mux),
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
