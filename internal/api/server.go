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

// Package api provides HTTP API handlers and server implementation for the krkn-operator.
// It includes endpoints for authentication, target management, scenario execution, and user management.
//
// @title Krkn Operator API
// @version 2.0
// @description REST and WebSocket API for Krkn chaos engineering operator.
// @description
// @description **API Versions:**
// @description - **v1** - REST API with polling (deprecated but maintained)
// @description - **v2** - REST API (same as v1) + WebSocket real-time updates
// @description
// @description **WebSocket Authentication (v2):**
// @description WebSocket endpoints use JWT via subprotocol header:
// @description - JavaScript: `new WebSocket(url, 'access_token.' + jwtToken)`
// @description - Header: `Sec-WebSocket-Protocol: access_token.<jwt_token>`
// @description
// @description **Migration Path:**
// @description 1. v1 REST → v2 REST (no changes, just update base path)
// @description 2. v2 REST → v2 WebSocket (replace polling with multiplexed WebSocket)
// @termsOfService https://krkn-chaos.dev/terms
//
// @contact.name Krkn Team
// @contact.url https://github.com/krkn-chaos/krkn-operator
// @contact.email krkn-chaos@googlegroups.com
//
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
//
// @host localhost:8080
// @BasePath /api/v1
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT token obtained from /api/v1/auth/login or /api/v2/auth/login. Format: "Bearer {token}"
package api

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	httpSwagger "github.com/swaggo/http-swagger"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	_ "github.com/krkn-chaos/krkn-operator/internal/api/docs" // Import generated docs
	v2 "github.com/krkn-chaos/krkn-operator/internal/api/v2"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

// Server represents the REST API server
type Server struct {
	server         *http.Server
	handler        *Handler
	v2Handler      *v2.Handler
	authMiddleware *auth.Middleware
	secretManager  *auth.SecretManager
}

// NewServer creates a new API server
//
// Parameters:
//   - port: HTTP port to listen on
//   - client: Kubernetes client
//   - clientset: Kubernetes clientset
//   - namespace: Operator namespace
//   - grpcServerAddr: gRPC server address
//   - secretManager: JWT secret manager (must be started before API server receives traffic)
//
// Returns a new Server instance
func NewServer(port int, client client.Client, clientset kubernetes.Interface, namespace string, grpcServerAddr string, secretManager *auth.SecretManager) *Server {
	handler := NewHandler(client, clientset, namespace, grpcServerAddr, secretManager)

	// Create auth middleware using SecretManager
	// The SecretManager is started as a Runnable before the API server starts
	// so the JWT secret is guaranteed to be loaded when the first request arrives
	getTokenGen := func() *auth.TokenGenerator {
		tokenGen, err := secretManager.GetTokenGenerator()
		if err != nil {
			// This should never happen if SecretManager started successfully
			// Return nil and let middleware return 503 Service Unavailable
			log.Log.Error(err, "CRITICAL: Failed to get TokenGenerator from SecretManager")
			return nil
		}
		return tokenGen
	}
	authMw := auth.NewLazyMiddleware(getTokenGen)

	// Create v2 handler (WebSocket support only, REST reuses v1 handlers)
	getTokenGenCtx := func(ctx context.Context) (*auth.TokenGenerator, error) {
		return secretManager.GetTokenGenerator()
	}
	v2Handler := v2.NewHandler(client, namespace, handler, getTokenGenCtx) // handler implements AuthorizationChecker

	mux := http.NewServeMux()

	// Public authentication endpoints (no auth required)
	mux.HandleFunc(AuthIsRegistered, handler.IsRegistered)
	mux.HandleFunc(AuthRegister, handler.Register)
	mux.HandleFunc(AuthLogin, handler.Login)

	// Authenticated endpoints - user and admin access
	mux.Handle(HealthPath, authMw.RequireAuth(http.HandlerFunc(handler.HealthCheck)))
	mux.Handle(ClustersPath, authMw.RequireAuth(http.HandlerFunc(handler.GetClusters)))
	mux.Handle(NodesPath, authMw.RequireAuth(http.HandlerFunc(handler.GetNodes)))
	mux.Handle(TerminalPath, authMw.RequireAuth(http.HandlerFunc(handler.ExecuteTerminal)))
	mux.Handle(TerminalAvailableCommandsPath, authMw.RequireAuth(http.HandlerFunc(handler.GetAvailableCommands)))
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

	// Registry management endpoints - CRUD: admin only, available: all users
	mux.Handle(RegistriesPath, authMw.RequireAuth(http.HandlerFunc(handler.RegistriesRouter)))
	mux.Handle(RegistriesPath+"/", authMw.RequireAuth(http.HandlerFunc(handler.RegistriesRouter)))
	mux.Handle(RegistriesAvailablePath, authMw.RequireAuth(http.HandlerFunc(handler.ListAvailableRegistries)))

	// File management endpoints - CRUD: admin only, available: all users
	mux.Handle(FilesPath, authMw.RequireAuth(http.HandlerFunc(handler.FilesRouter)))
	mux.Handle(FilesPath+"/", authMw.RequireAuth(http.HandlerFunc(handler.FilesRouter)))
	mux.Handle(FilesAvailablePath, authMw.RequireAuth(http.HandlerFunc(handler.ListAvailableFiles)))

	// File type management endpoints - all users can list/get, admin can CRUD
	mux.Handle(FileTypesPath, authMw.RequireAuth(http.HandlerFunc(handler.FileTypesRouter)))
	mux.Handle(FileTypesPath+"/", authMw.RequireAuth(http.HandlerFunc(handler.FileTypesRouter)))

	// Workflow management endpoints - CRUD: authenticated users, list all: admin only, available: all users
	// Register /available before generic /workflows to avoid path collision
	mux.Handle(WorkflowsAvailablePath, authMw.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handler.ListAvailableWorkflows(w, r)
	})))
	mux.Handle(WorkflowsPath, authMw.RequireAuth(http.HandlerFunc(handler.WorkflowsRouter)))
	mux.Handle(WorkflowsPath+"/", authMw.RequireAuth(http.HandlerFunc(handler.WorkflowsRouter)))

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

	// Graph Run endpoints - user and admin access (ownership-based authorization)
	mux.Handle(GraphRunsPath, authMw.RequireAuth(http.HandlerFunc(handler.GraphRunsRouter)))
	mux.Handle(GraphRunsPath+"/", authMw.RequireAuth(http.HandlerFunc(handler.GraphRunsRouter)))

	// Swagger UI - public endpoint for API documentation
	mux.Handle("/api/swagger/", httpSwagger.WrapHandler)
	// Elasticsearch config endpoints - admin only
	mux.Handle(ElasticsearchConfigsPath, authMw.RequireAuth(http.HandlerFunc(handler.ElasticsearchConfigsRouter)))
	mux.Handle(ElasticsearchConfigsPath+"/", authMw.RequireAuth(http.HandlerFunc(handler.ElasticsearchConfigsRouter)))

	// ==================== API v2 Endpoints ====================
	// v2 REST endpoints reuse v1 handlers (backward compatible)
	// v2 WebSocket endpoints provide real-time multiplexed updates

	// v2 REST endpoints (same as v1, for gradual migration)
	mux.HandleFunc(v2.ScenariosRunPath+"/", func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a WebSocket logs request (same as v1)
		if strings.Contains(r.URL.Path, "/jobs/") && strings.HasSuffix(r.URL.Path, "/logs") {
			// WebSocket endpoint - auth handled internally via subprotocol
			handler.GetScenarioRunLogs(w, r)
			return
		}
		// All other endpoints require HTTP JWT auth (reuse v1 router)
		authMw.RequireAuth(http.HandlerFunc(handler.ScenariosRunRouter)).ServeHTTP(w, r)
	})
	mux.Handle(v2.ScenariosRunPath, authMw.RequireAuth(http.HandlerFunc(handler.ScenariosRunRouter)))
	mux.Handle(v2.GraphRunsPath, authMw.RequireAuth(http.HandlerFunc(handler.GraphRunsRouter)))
	mux.Handle(v2.GraphRunsPath+"/", authMw.RequireAuth(http.HandlerFunc(handler.GraphRunsRouter)))
	mux.Handle(v2.DashboardActiveRunsPath, authMw.RequireAuth(http.HandlerFunc(handler.GetActiveRunsOverview)))

	// v2 WebSocket endpoints (NEW - real-time multiplexed updates)
	// WebSocket authentication is handled internally via Sec-WebSocket-Protocol header
	mux.HandleFunc(v2.WebSocketRunsPath, v2Handler.WsHandler.HandleWebSocket)
	mux.HandleFunc(v2.WebSocketGraphRunsPath, v2Handler.WsHandler.HandleWebSocket)
	mux.HandleFunc(v2.WebSocketDashboardActiveRunsPath, v2Handler.WsHandler.HandleWebSocket)

	// v2 WebSocket job logs streaming (reuse v1 handler, just different path)
	// Path pattern: /api/v2/ws/scenarios/run/{scenarioRunName}/jobs/{jobID}/logs
	mux.HandleFunc(v2.WebSocketJobLogsPath, func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a logs request (ends with /logs)
		if strings.Contains(r.URL.Path, "/jobs/") && strings.HasSuffix(r.URL.Path, "/logs") {
			// Reuse v1 handler (it handles path parsing for both v1 and v2)
			handler.GetScenarioRunLogs(w, r)
			return
		}
		http.NotFound(w, r)
	})

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
		v2Handler:      v2Handler,
		authMiddleware: authMw,
		secretManager:  secretManager,
	}
}

// Start starts the API server
// It waits for the JWT SecretManager to be ready before accepting traffic
func (s *Server) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("🌐 Starting REST API server (waiting for JWT secret to be ready)", "addr", s.server.Addr)

	// Wait for JWT SecretManager to be ready before starting HTTP server
	// This prevents the server from accepting requests before authentication is configured
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(2 * time.Minute) // Max wait time for JWT secret
	for {
		select {
		case <-ctx.Done():
			logger.Info("Context cancelled while waiting for JWT secret")
			return ctx.Err()

		case <-timeout:
			logger.Error(nil, "❌ Timeout waiting for JWT secret to be ready")
			return fmt.Errorf("timeout waiting for JWT secret to be ready after 2 minutes")

		case <-ticker.C:
			if s.secretManager.IsReady() {
				logger.Info("✅ JWT secret ready, starting HTTP server", "addr", s.server.Addr)
				goto startServer
			}
			logger.V(1).Info("Waiting for JWT secret to be ready...")
		}
	}

startServer:
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	logger.Info("🚀 REST API server started and accepting connections", "addr", s.server.Addr)

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

// NeedLeaderElection implements manager.LeaderElectionRunnable
// Returns false because the API server is stateless and should run on all replicas
// to provide high availability. Kubernetes' optimistic concurrency control (resourceVersion)
// handles any potential race conditions when multiple replicas modify the same resources.
func (s *Server) NeedLeaderElection() bool {
	return false
}

// GetV2Handler returns the v2 handler for access to WebSocket broadcaster
// Controllers use this to send real-time updates to WebSocket clients
func (s *Server) GetV2Handler() *v2.Handler {
	return s.v2Handler
}
