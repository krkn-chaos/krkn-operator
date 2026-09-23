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

// Package api provides HTTP handlers for the krkn-operator REST API.
// This file implements handlers for Elasticsearch configuration management,
// including storing credentials as Kubernetes Secrets and validating
// connectivity to Elasticsearch clusters used for chaos scenario result storage.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/elasticsearch"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CreateElasticsearchConfig handles POST /api/v1/elasticsearch-configs
// Creates a new Elasticsearch config Secret (admin only)
func (h *Handler) CreateElasticsearchConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("create-elasticsearch-config")

	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	var req elasticsearch.CreateElasticsearchConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := elasticsearch.ValidateCreateRequest(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	exists, err := h.elasticsearchConfigExists(ctx, req.Name)
	if err != nil {
		logger.Error(err, "Failed to check for existing elasticsearch config", "name", req.Name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to check for existing Elasticsearch config",
		})
		return
	}
	if exists {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: fmt.Sprintf("Elasticsearch config '%s' already exists", req.Name),
		})
		return
	}

	claims := auth.GetClaimsFromContext(ctx)
	createdBy := ""
	if claims != nil {
		createdBy = claims.UserID
	}

	port := req.Port
	if port == 0 {
		port = elasticsearch.DefaultPort
	}

	labels := elasticsearch.BuildLabels()
	annotations := elasticsearch.BuildAnnotations(
		req.Host,
		port,
		req.TelemetryIndex,
		req.MetricsIndex,
		req.AlertsIndex,
		req.GrafanaURL,
		createdBy,
	)
	if req.InsecureSkipTLSVerify {
		annotations[elasticsearch.InsecureSkipTLSVerifyAnnotation] = "true"
	}

	data := map[string][]byte{
		elasticsearch.SecretKeyUsername: []byte(req.Username),
		elasticsearch.SecretKeyPassword: []byte(req.Password),
	}
	if req.CACert != "" {
		data[elasticsearch.SecretKeyCACert] = []byte(req.CACert)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   h.namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}

	if err := h.client.Create(ctx, secret); err != nil {
		logger.Error(err, "Failed to create elasticsearch config secret", "name", req.Name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create Elasticsearch config",
		})
		return
	}

	logger.Info("Created Elasticsearch config", "name", req.Name, "createdBy", createdBy)

	writeJSON(w, http.StatusCreated, elasticsearch.CreateElasticsearchConfigResponse{
		Message: "Elasticsearch config created successfully",
		Name:    req.Name,
	})
}

// ListElasticsearchConfigs handles GET /api/v1/elasticsearch-configs
// Lists all Elasticsearch configs (any authenticated user)
func (h *Handler) ListElasticsearchConfigs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("list-elasticsearch-configs")

	var secretList corev1.SecretList
	if err := h.client.List(ctx, &secretList,
		client.InNamespace(h.namespace),
		client.MatchingLabels{
			elasticsearch.AppNameLabel:      elasticsearch.AppName,
			elasticsearch.AppComponentLabel: elasticsearch.ComponentElasticsearchConfig,
		},
	); err != nil {
		logger.Error(err, "Failed to list elasticsearch config secrets")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list Elasticsearch configs",
		})
		return
	}

	configs := make([]elasticsearch.ElasticsearchConfigResponse, len(secretList.Items))
	for i, secret := range secretList.Items {
		configs[i] = buildElasticsearchConfigResponse(&secret)
	}

	logger.Info("Listed Elasticsearch configs", "total", len(configs))

	writeJSON(w, http.StatusOK, elasticsearch.ListElasticsearchConfigsResponse{
		Configs: configs,
		Total:   len(configs),
	})
}

// GetElasticsearchConfig handles GET /api/v1/elasticsearch-configs/{name}
// Returns a single Elasticsearch config (admin only)
func (h *Handler) GetElasticsearchConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("get-elasticsearch-config")

	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	configName, err := extractPathSuffix(r.URL.Path, ElasticsearchConfigsPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid config name in path",
		})
		return
	}

	secret, err := h.loadElasticsearchConfigSecret(ctx, configName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Elasticsearch config '%s' not found", configName),
			})
		} else {
			logger.Error(err, "Failed to get elasticsearch config", "name", configName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get Elasticsearch config",
			})
		}
		return
	}

	logger.Info("Retrieved Elasticsearch config", "name", configName)
	writeJSON(w, http.StatusOK, buildElasticsearchConfigResponse(secret))
}

// UpdateElasticsearchConfig handles PUT /api/v1/elasticsearch-configs/{name}
// Updates an Elasticsearch config (admin only)
func (h *Handler) UpdateElasticsearchConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("update-elasticsearch-config")

	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	configName, err := extractPathSuffix(r.URL.Path, ElasticsearchConfigsPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid config name in path",
		})
		return
	}

	var req elasticsearch.UpdateElasticsearchConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := elasticsearch.ValidateUpdateRequest(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	secret, err := h.loadElasticsearchConfigSecret(ctx, configName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Elasticsearch config '%s' not found", configName),
			})
		} else {
			logger.Error(err, "Failed to get elasticsearch config", "name", configName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get Elasticsearch config",
			})
		}
		return
	}

	claims := auth.GetClaimsFromContext(ctx)
	updatedBy := ""
	if claims != nil {
		updatedBy = claims.UserID
	}

	port := req.Port
	if port == 0 {
		port = elasticsearch.DefaultPort
	}

	secret.Annotations = elasticsearch.UpdateAnnotations(
		secret.Annotations,
		req.Host,
		port,
		req.TelemetryIndex,
		req.MetricsIndex,
		req.AlertsIndex,
		req.GrafanaURL,
		updatedBy,
	)

	// Guard against a Secret that was created without a Data map.
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}

	// Validate the fully merged connection before persisting. Empty update
	// credentials fall back to the stored Secret values, so the request-only
	// validation above cannot see a retained username paired with a newly set
	// http:// host. Compute the effective username/CA and reject a configuration
	// the query client would refuse, rather than saving it and returning success.
	mergedUsername := req.Username
	if mergedUsername == "" {
		mergedUsername = string(secret.Data[elasticsearch.SecretKeyUsername])
	}
	// A nil CACert keeps the stored value; a non-nil one (including an explicit
	// empty string that clears it) is the effective value.
	mergedCACert := string(secret.Data[elasticsearch.SecretKeyCACert])
	if req.CACert != nil {
		mergedCACert = *req.CACert
	}
	if err := elasticsearch.ValidateMergedConnection(req.Host, mergedUsername, mergedCACert); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	// Only overwrite credentials when the caller explicitly supplies them;
	// an empty string (omitempty) means "keep the existing value".
	if req.Username != "" {
		secret.Data[elasticsearch.SecretKeyUsername] = []byte(req.Username)
	}
	if req.Password != "" {
		secret.Data[elasticsearch.SecretKeyPassword] = []byte(req.Password)
	}
	// A nil CACert leaves the stored certificate untouched. A non-nil value with
	// content replaces it; a non-nil empty string explicitly clears it.
	if req.CACert != nil {
		if *req.CACert != "" {
			secret.Data[elasticsearch.SecretKeyCACert] = []byte(*req.CACert)
		} else {
			delete(secret.Data, elasticsearch.SecretKeyCACert)
		}
	}
	// A nil pointer means the caller omitted the field: leave the stored TLS
	// setting untouched. A non-nil value explicitly sets or clears it.
	if req.InsecureSkipTLSVerify != nil {
		if *req.InsecureSkipTLSVerify {
			secret.Annotations[elasticsearch.InsecureSkipTLSVerifyAnnotation] = "true"
		} else {
			delete(secret.Annotations, elasticsearch.InsecureSkipTLSVerifyAnnotation)
		}
	}

	if err := h.client.Update(ctx, secret); err != nil {
		logger.Error(err, "Failed to update elasticsearch config", "name", configName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update Elasticsearch config",
		})
		return
	}

	logger.Info("Updated Elasticsearch config", "name", configName, "updatedBy", updatedBy)

	writeJSON(w, http.StatusOK, elasticsearch.UpdateElasticsearchConfigResponse{
		Message: "Elasticsearch config updated successfully",
		Name:    configName,
	})
}

// DeleteElasticsearchConfig handles DELETE /api/v1/elasticsearch-configs/{name}
// Deletes an Elasticsearch config (admin only)
func (h *Handler) DeleteElasticsearchConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("delete-elasticsearch-config")

	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	configName, err := extractPathSuffix(r.URL.Path, ElasticsearchConfigsPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid config name in path",
		})
		return
	}

	secret, err := h.loadElasticsearchConfigSecret(ctx, configName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Elasticsearch config '%s' not found", configName),
			})
		} else {
			logger.Error(err, "Failed to get elasticsearch config", "name", configName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get Elasticsearch config",
			})
		}
		return
	}

	if err := h.client.Delete(ctx, secret); err != nil {
		logger.Error(err, "Failed to delete elasticsearch config", "name", configName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete Elasticsearch config",
		})
		return
	}

	logger.Info("Deleted Elasticsearch config", "name", configName)

	writeJSON(w, http.StatusOK, elasticsearch.DeleteElasticsearchConfigResponse{
		Message: "Elasticsearch config deleted successfully",
	})
}

// ElasticsearchConfigsRouter routes requests to /api/v1/elasticsearch-configs endpoints
func (h *Handler) ElasticsearchConfigsRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	normalizedPath := strings.TrimSuffix(path, "/")

	// Root endpoint: /api/v1/elasticsearch-configs
	if normalizedPath == ElasticsearchConfigsPath {
		switch r.Method {
		case http.MethodGet:
			h.ListElasticsearchConfigs(w, r)
		case http.MethodPost:
			h.CreateElasticsearchConfig(w, r)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
				Error:   "method_not_allowed",
				Message: "Only GET and POST are allowed on " + ElasticsearchConfigsPath,
			})
		}
		return
	}

	// Config-specific endpoints: /api/v1/elasticsearch-configs/{name}
	if strings.HasPrefix(path, ElasticsearchConfigsPath+"/") {
		switch r.Method {
		case http.MethodGet:
			h.GetElasticsearchConfig(w, r)
		case http.MethodPut:
			h.UpdateElasticsearchConfig(w, r)
		case http.MethodDelete:
			h.DeleteElasticsearchConfig(w, r)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
				Error:   "method_not_allowed",
				Message: "Invalid method for elasticsearch config endpoint",
			})
		}
		return
	}

	writeJSONError(w, http.StatusNotFound, ErrorResponse{
		Error:   "not_found",
		Message: "Endpoint not found",
	})
}

// QueryElasticsearchTelemetry handles POST /api/v1/elasticsearch-query
// It resolves the named Elasticsearch config Secret server-side (credentials
// never leave the backend), connects to the cluster, and returns the most recent
// telemetry documents. Available to any authenticated user, mirroring
// ListElasticsearchConfigs.
//
// @Summary Query Elasticsearch telemetry
// @Description Run a telemetry search against a saved Elasticsearch config or an inline connection. Credentials are resolved server-side and never leave the backend. Available to any authenticated user.
// @Tags elasticsearch
// @Accept json
// @Produce json
// @Param request body elasticsearch.QueryTelemetryRequest true "Query parameters (exactly one of configName or inline)"
// @Success 200 {object} elasticsearch.QueryTelemetryResponse "Most recent telemetry documents"
// @Failure 400 {object} ErrorResponse "Invalid request body or parameters"
// @Failure 401 {object} ErrorResponse "Authentication required"
// @Failure 404 {object} ErrorResponse "Elasticsearch config not found"
// @Failure 405 {object} ErrorResponse "Method not allowed"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /elasticsearch-query [post]
func (h *Handler) QueryElasticsearchTelemetry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("query-elasticsearch-telemetry")

	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only POST is allowed on " + ElasticsearchQueryPath,
		})
		return
	}

	var req elasticsearch.QueryTelemetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := elasticsearch.ValidateQueryRequest(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	// Resolve the connection either from an ephemeral inline config (credentials
	// supplied on the request, never persisted) or from a saved config Secret.
	// ValidateQueryRequest guarantees exactly one of the two is set.
	var conn elasticsearch.ConnectionParams
	source := "inline"
	if req.Inline != nil {
		conn = buildInlineConnectionParams(req.Inline)
	} else {
		source = req.ConfigName
		secret, err := h.loadElasticsearchConfigSecret(ctx, req.ConfigName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				writeJSONError(w, http.StatusNotFound, ErrorResponse{
					Error:   "not_found",
					Message: fmt.Sprintf("Elasticsearch config '%s' not found", req.ConfigName),
				})
			} else {
				logger.Error(err, "Failed to load elasticsearch config", "name", req.ConfigName)
				writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
					Error:   "internal_error",
					Message: "Failed to load Elasticsearch config",
				})
			}
			return
		}
		conn = buildConnectionParams(secret)
	}

	docs, stats, err := h.esClient.QueryTelemetry(ctx, conn, req.Size, req.StartDate, req.EndDate)
	if err != nil {
		// A rejected inline destination is a client error (the caller asked the
		// operator to reach an address the destination policy forbids), not an
		// upstream failure. Return 400 without echoing the specific address.
		if errors.Is(err, elasticsearch.ErrDestinationNotAllowed) {
			logger.Info("Rejected telemetry query to a disallowed destination", "source", source)
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "The requested Elasticsearch destination is not permitted",
			})
			return
		}
		// Log bounded upstream diagnostics server-side for troubleshooting, but
		// never return raw upstream bodies or internal client errors to the
		// caller: respond with a stable, sanitized 502 message instead.
		var statusErr *elasticsearch.StatusError
		if errors.As(err, &statusErr) {
			logger.Error(err, "Elasticsearch returned a non-success status",
				"source", source, "status", statusErr.StatusCode)
		} else {
			logger.Error(err, "Failed to query elasticsearch telemetry", "source", source)
		}
		writeJSONError(w, http.StatusBadGateway, ErrorResponse{
			Error:   "upstream_error",
			Message: "Failed to query Elasticsearch",
		})
		return
	}

	logger.Info("Queried Elasticsearch telemetry", "source", source, "results", len(docs))

	writeJSON(w, http.StatusOK, elasticsearch.QueryTelemetryResponse{
		Documents: docs,
		Total:     len(docs),
		Stats:     stats,
	})
}

// buildConnectionParams assembles the connection parameters for a query from a
// config Secret, reading the host/port/index from annotations and the
// credentials from the Secret data.
func buildConnectionParams(secret *corev1.Secret) elasticsearch.ConnectionParams {
	port := elasticsearch.DefaultPort
	if portStr := secret.Annotations[elasticsearch.PortAnnotation]; portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	username := ""
	if u, ok := secret.Data[elasticsearch.SecretKeyUsername]; ok {
		username = string(u)
	}
	password := ""
	if p, ok := secret.Data[elasticsearch.SecretKeyPassword]; ok {
		password = string(p)
	}
	caCert := ""
	if c, ok := secret.Data[elasticsearch.SecretKeyCACert]; ok {
		caCert = string(c)
	}

	return elasticsearch.ConnectionParams{
		Host:               secret.Annotations[elasticsearch.HostAnnotation],
		Port:               port,
		Username:           username,
		Password:           password,
		Index:              secret.Annotations[elasticsearch.TelemetryIndexAnnotation],
		CACert:             caCert,
		InsecureSkipVerify: secret.Annotations[elasticsearch.InsecureSkipTLSVerifyAnnotation] == "true",
	}
}

// buildInlineConnectionParams assembles connection parameters from an ephemeral
// inline connection supplied on a query request. The credentials are used only
// for this request and are never persisted. A zero port falls back to the
// default Elasticsearch port.
func buildInlineConnectionParams(inline *elasticsearch.InlineConnection) elasticsearch.ConnectionParams {
	port := inline.Port
	if port == 0 {
		port = elasticsearch.DefaultPort
	}

	// Inline connections always use default TLS verification with the system
	// trust store: no custom CA, no insecure skip. Those remain admin-only,
	// per-config settings.
	return elasticsearch.ConnectionParams{
		Host:     inline.Host,
		Port:     port,
		Username: inline.Username,
		Password: inline.Password,
		Index:    inline.TelemetryIndex,
		// Inline connections are user-supplied and reachable by any authenticated
		// user, so they are subject to the destination policy (loopback/private/
		// metadata/etc. are rejected) to prevent server-side request forgery.
		RestrictDestination: true,
	}
}

// elasticsearchConfigExists reports whether an Elasticsearch config Secret with the given name
// exists. It returns (false, nil) when the secret is absent, (true, nil) when it is present, and
// (false, err) for any API or permission error so callers can surface a 500 rather than silently
// treating a transient failure as "not found".
func (h *Handler) elasticsearchConfigExists(ctx context.Context, name string) (bool, error) {
	var secret corev1.Secret
	err := h.client.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: h.namespace,
	}, &secret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return secret.Labels[elasticsearch.AppComponentLabel] == elasticsearch.ComponentElasticsearchConfig, nil
}

// loadElasticsearchConfigSecret loads an Elasticsearch config Secret by name
func (h *Handler) loadElasticsearchConfigSecret(ctx context.Context, name string) (*corev1.Secret, error) {
	var secret corev1.Secret
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: h.namespace,
	}, &secret); err != nil {
		return nil, err
	}

	if secret.Labels[elasticsearch.AppComponentLabel] != elasticsearch.ComponentElasticsearchConfig {
		return nil, fmt.Errorf("secret is not an elasticsearch config secret")
	}

	return &secret, nil
}

// buildElasticsearchConfigResponse constructs an ElasticsearchConfigResponse from a Secret.
// The password is intentionally never included; callers must re-supply it on update.
// Username is included as a non-secret identifier for UI display.
func buildElasticsearchConfigResponse(secret *corev1.Secret) elasticsearch.ElasticsearchConfigResponse {
	port := elasticsearch.DefaultPort
	if portStr := secret.Annotations[elasticsearch.PortAnnotation]; portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	username := ""
	if u, ok := secret.Data[elasticsearch.SecretKeyUsername]; ok {
		username = string(u)
	}

	return elasticsearch.ElasticsearchConfigResponse{
		Name:                  secret.Name,
		Host:                  secret.Annotations[elasticsearch.HostAnnotation],
		Port:                  port,
		Username:              username,
		TelemetryIndex:        secret.Annotations[elasticsearch.TelemetryIndexAnnotation],
		MetricsIndex:          secret.Annotations[elasticsearch.MetricsIndexAnnotation],
		AlertsIndex:           secret.Annotations[elasticsearch.AlertsIndexAnnotation],
		GrafanaURL:            secret.Annotations[elasticsearch.GrafanaURLAnnotation],
		InsecureSkipTLSVerify: secret.Annotations[elasticsearch.InsecureSkipTLSVerifyAnnotation] == "true",
		CreatedAt:             secret.Annotations[elasticsearch.CreatedAtAnnotation],
		CreatedBy:             secret.Annotations[elasticsearch.CreatedByAnnotation],
		UpdatedAt:             secret.Annotations[elasticsearch.UpdatedAtAnnotation],
		UpdatedBy:             secret.Annotations[elasticsearch.UpdatedByAnnotation],
	}
}
