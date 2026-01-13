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
	"context"
	"fmt"
	"net/http"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Server represents the REST API server
type Server struct {
	server  *http.Server
	handler *Handler
}

// NewServer creates a new API server
func NewServer(port int, client client.Client, namespace string, grpcServerAddr string) *Server {
	handler := NewHandler(client, namespace, grpcServerAddr)

	mux := http.NewServeMux()

	// Routes
	mux.HandleFunc("/health", handler.HealthCheck)
	mux.HandleFunc("/clusters", handler.GetClusters)
	mux.HandleFunc("/nodes", handler.GetNodes)
	mux.HandleFunc("/targets", handler.PostTarget)                 // POST /targets
	mux.HandleFunc("/targets/", handler.GetTargetByUUID)           // GET /targets/{uuid}
	mux.HandleFunc("/scenarios", handler.PostScenarios)            // POST /scenarios
	mux.HandleFunc("/scenarios/detail/", handler.PostScenarioDetail) // POST /scenarios/detail/{scenario_name}

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
