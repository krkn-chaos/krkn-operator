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

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   h.namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			elasticsearch.SecretKeyUsername: []byte(req.Username),
			elasticsearch.SecretKeyPassword: []byte(req.Password),
		},
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

	// Only overwrite credentials when the caller explicitly supplies them;
	// an empty string (omitempty) means "keep the existing value".
	if req.Username != "" {
		secret.Data[elasticsearch.SecretKeyUsername] = []byte(req.Username)
	}
	if req.Password != "" {
		secret.Data[elasticsearch.SecretKeyPassword] = []byte(req.Password)
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
		Name:           secret.Name,
		Host:           secret.Annotations[elasticsearch.HostAnnotation],
		Port:           port,
		Username:       username,
		TelemetryIndex: secret.Annotations[elasticsearch.TelemetryIndexAnnotation],
		MetricsIndex:   secret.Annotations[elasticsearch.MetricsIndexAnnotation],
		AlertsIndex:    secret.Annotations[elasticsearch.AlertsIndexAnnotation],
		GrafanaURL:     secret.Annotations[elasticsearch.GrafanaURLAnnotation],
		CreatedAt:      secret.Annotations[elasticsearch.CreatedAtAnnotation],
		CreatedBy:      secret.Annotations[elasticsearch.CreatedByAnnotation],
		UpdatedAt:      secret.Annotations[elasticsearch.UpdatedAtAnnotation],
		UpdatedBy:      secret.Annotations[elasticsearch.UpdatedByAnnotation],
	}
}
