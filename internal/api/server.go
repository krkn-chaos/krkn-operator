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
)

// Server represents the REST API server
type Server struct {
	server  *http.Server
	handler *Handler
}

// NewServer creates a new API server
func NewServer(port int, client client.Client, clientset kubernetes.Interface, namespace string, grpcServerAddr string) *Server {
	handler := NewHandler(client, clientset, namespace, grpcServerAddr)

	mux := http.NewServeMux()

	// API v1 routes
	mux.HandleFunc("/api/v1/health", handler.HealthCheck)
	mux.HandleFunc("/api/v1/clusters", handler.GetClusters)
	mux.HandleFunc("/api/v1/nodes", handler.GetNodes)
	mux.HandleFunc("/api/v1/targets", handler.TargetsHandler)  // POST, GET - legacy endpoints (KrknTargetRequest)
	mux.HandleFunc("/api/v1/targets/", handler.TargetsHandler) // GET /{uuid} - legacy endpoint (check status)

	// Provider config endpoints (KrknOperatorTargetProviderConfig)
	mux.HandleFunc("/api/v1/provider-config", handler.ProviderConfigHandler)  // POST, GET
	mux.HandleFunc("/api/v1/provider-config/", handler.ProviderConfigHandler) // GET /{uuid}

	// CRUD endpoints for KrknOperatorTarget (operator-managed targets)
	mux.HandleFunc("/api/v1/operator/targets", handler.TargetsCRUDRouter)  // POST, GET
	mux.HandleFunc("/api/v1/operator/targets/", handler.TargetsCRUDRouter) // GET, PUT, DELETE /{uuid}

	// Scenario management endpoints
	mux.HandleFunc("/api/v1/scenarios", handler.PostScenarios)                // POST - list scenarios
	mux.HandleFunc("/api/v1/scenarios/detail/", handler.PostScenarioDetail)   // POST /{scenario_name}
	mux.HandleFunc("/api/v1/scenarios/globals/", handler.PostScenarioGlobals) // POST /{scenario_name}
	mux.HandleFunc("/api/v1/scenarios/run", handler.ScenariosRunRouter)       // POST, GET
	mux.HandleFunc("/api/v1/scenarios/run/", handler.ScenariosRunRouter)      // GET, DELETE /{jobId}

	// Wrap mux with logging middleware
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: loggingMiddleware(mux),
	}

	return &Server{
		server:  server,
		handler: handler,
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
