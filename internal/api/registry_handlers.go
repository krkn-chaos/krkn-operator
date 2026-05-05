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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	"github.com/krkn-chaos/krkn-operator/pkg/registry"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// CreateRegistry handles POST /api/v1/registries
// Creates a new registry Secret (admin only)
func (h *Handler) CreateRegistry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("create-registry")

	// Check admin privileges
	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Parse request body
	var req registry.CreateRegistryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate request
	if err := validateCreateRegistryRequest(ctx, h.client, &req, h.namespace); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	// Check if registry already exists
	if h.registryExists(ctx, req.Name) {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: fmt.Sprintf("Registry '%s' already exists", req.Name),
		})
		return
	}

	// Get current user for audit trail
	claims := auth.GetClaimsFromContext(ctx)
	createdBy := ""
	if claims != nil {
		createdBy = claims.UserID
	}

	// Build dockerconfigjson
	dockerConfigJSON, err := registry.BuildDockerConfigJSON(
		req.RegistryURL,
		req.AuthType,
		req.Token,
		req.Username,
		req.Password,
	)
	if err != nil {
		logger.Error(err, "Failed to build dockerconfigjson", "registryName", req.Name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create registry configuration",
		})
		return
	}

	// Build labels and annotations
	labels := registry.BuildRegistryLabels(req.AuthType, req.Groups, req.AvailableToAll)
	annotations := registry.BuildRegistryAnnotations(
		req.RegistryURL,
		req.ScenarioRepository,
		req.Description,
		req.SkipTLS,
		req.Insecure,
		createdBy,
	)

	// Create Secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   h.namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfigJSON,
		},
	}

	if err := h.client.Create(ctx, secret); err != nil {
		logger.Error(err, "Failed to create registry secret", "registryName", req.Name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create registry",
		})
		return
	}

	logger.Info("Created registry", "registryName", req.Name, "createdBy", createdBy)

	writeJSON(w, http.StatusCreated, registry.CreateRegistryResponse{
		Message: "Registry created successfully",
		Name:    req.Name,
	})
}

// ListRegistries handles GET /api/v1/registries
// Lists all registries (admin only)
func (h *Handler) ListRegistries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("list-registries")

	// Check admin privileges
	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// List all registry Secrets
	var secretList corev1.SecretList
	err := h.client.List(ctx, &secretList,
		client.InNamespace(h.namespace),
		client.MatchingLabels{
			registry.AppNameLabel:      registry.AppName,
			registry.AppComponentLabel: registry.ComponentRegistry,
		},
	)
	if err != nil {
		logger.Error(err, "Failed to list registry secrets")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list registries",
		})
		return
	}

	// Convert to response format
	registries := make([]registry.RegistryResponse, len(secretList.Items))
	for i, secret := range secretList.Items {
		registries[i] = buildRegistryResponse(&secret)
	}

	logger.Info("Listed registries", "total", len(registries))

	writeJSON(w, http.StatusOK, registry.ListRegistriesResponse{
		Registries: registries,
		Total:      len(registries),
	})
}

// GetRegistry handles GET /api/v1/registries/{name}
// Returns a single registry (admin only)
func (h *Handler) GetRegistry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("get-registry")

	// Check admin privileges
	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Extract registry name from path
	registryName, err := extractPathSuffix(r.URL.Path, RegistriesPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid registry name in path",
		})
		return
	}

	// Load secret
	secret, err := h.loadRegistrySecret(ctx, registryName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Registry '%s' not found", registryName),
			})
		} else {
			logger.Error(err, "Failed to get registry", "registryName", registryName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get registry",
			})
		}
		return
	}

	logger.Info("Retrieved registry", "registryName", registryName)
	writeJSON(w, http.StatusOK, buildRegistryResponse(secret))
}

// UpdateRegistry handles PUT /api/v1/registries/{name}
// Updates a registry (admin only)
func (h *Handler) UpdateRegistry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("update-registry")

	// Check admin privileges
	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Extract registry name from path
	registryName, err := extractPathSuffix(r.URL.Path, RegistriesPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid registry name in path",
		})
		return
	}

	// Parse request body
	var req registry.UpdateRegistryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate request
	if err := validateUpdateRegistryRequest(ctx, h.client, &req, h.namespace); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	// Load existing secret
	secret, err := h.loadRegistrySecret(ctx, registryName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Registry '%s' not found", registryName),
			})
		} else {
			logger.Error(err, "Failed to get registry", "registryName", registryName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get registry",
			})
		}
		return
	}

	// Get current user for audit trail
	claims := auth.GetClaimsFromContext(ctx)
	updatedBy := ""
	if claims != nil {
		updatedBy = claims.UserID
	}

	// Build new dockerconfigjson
	dockerConfigJSON, err := registry.BuildDockerConfigJSON(
		req.RegistryURL,
		req.AuthType,
		req.Token,
		req.Username,
		req.Password,
	)
	if err != nil {
		logger.Error(err, "Failed to build dockerconfigjson", "registryName", registryName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update registry configuration",
		})
		return
	}

	// Update labels and annotations
	secret.Labels = registry.BuildRegistryLabels(req.AuthType, req.Groups, req.AvailableToAll)
	secret.Annotations = registry.UpdateRegistryAnnotations(
		secret.Annotations,
		req.RegistryURL,
		req.ScenarioRepository,
		req.Description,
		req.SkipTLS,
		req.Insecure,
		updatedBy,
	)
	secret.Data[corev1.DockerConfigJsonKey] = dockerConfigJSON

	if err := h.client.Update(ctx, secret); err != nil {
		logger.Error(err, "Failed to update registry", "registryName", registryName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update registry",
		})
		return
	}

	logger.Info("Updated registry", "registryName", registryName, "updatedBy", updatedBy)

	writeJSON(w, http.StatusOK, registry.UpdateRegistryResponse{
		Message: "Registry updated successfully",
		Name:    registryName,
	})
}

// DeleteRegistry handles DELETE /api/v1/registries/{name}
// Deletes a registry (admin only)
func (h *Handler) DeleteRegistry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("delete-registry")

	// Check admin privileges
	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Extract registry name from path
	registryName, err := extractPathSuffix(r.URL.Path, RegistriesPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid registry name in path",
		})
		return
	}

	// Load secret (to verify it exists and is a registry)
	secret, err := h.loadRegistrySecret(ctx, registryName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Registry '%s' not found", registryName),
			})
		} else {
			logger.Error(err, "Failed to get registry", "registryName", registryName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get registry",
			})
		}
		return
	}

	// Delete secret
	if err := h.client.Delete(ctx, secret); err != nil {
		logger.Error(err, "Failed to delete registry", "registryName", registryName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete registry",
		})
		return
	}

	logger.Info("Deleted registry", "registryName", registryName)

	writeJSON(w, http.StatusOK, registry.DeleteRegistryResponse{
		Message: "Registry deleted successfully",
	})
}

// ListAvailableRegistries handles GET /api/v1/registries/available
// Lists registries available to the current user
func (h *Handler) ListAvailableRegistries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("list-available-registries")

	// Get user claims
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
		return
	}

	// Admins see all registries
	if auth.IsAdmin(ctx) {
		// Reuse admin list logic but return minimal info
		var secretList corev1.SecretList
		err := h.client.List(ctx, &secretList,
			client.InNamespace(h.namespace),
			client.MatchingLabels{
				registry.AppNameLabel:      registry.AppName,
				registry.AppComponentLabel: registry.ComponentRegistry,
			},
		)
		if err != nil {
			logger.Error(err, "Failed to list registry secrets")
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to list registries",
			})
			return
		}

		registries := make([]registry.RegistryInfo, len(secretList.Items))
		for i, secret := range secretList.Items {
			registries[i] = buildRegistryInfo(&secret)
		}

		writeJSON(w, http.StatusOK, registry.AvailableRegistriesResponse{
			Registries: registries,
		})
		return
	}

	// List all registry secrets
	var secretList corev1.SecretList
	err := h.client.List(ctx, &secretList,
		client.InNamespace(h.namespace),
		client.MatchingLabels{
			registry.AppNameLabel:      registry.AppName,
			registry.AppComponentLabel: registry.ComponentRegistry,
		},
	)
	if err != nil {
		logger.Error(err, "Failed to list registry secrets")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list registries",
		})
		return
	}

	// Filter registries by access
	available := []registry.RegistryInfo{}
	for _, secret := range secretList.Items {
		if h.canAccessRegistry(ctx, &secret) {
			available = append(available, buildRegistryInfo(&secret))
		}
	}

	logger.Info("Listed available registries", "userID", claims.UserID, "total", len(available))

	writeJSON(w, http.StatusOK, registry.AvailableRegistriesResponse{
		Registries: available,
	})
}

// RegistriesRouter routes requests to /api/v1/registries endpoints
func (h *Handler) RegistriesRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Normalize path
	normalizedPath := strings.TrimSuffix(path, "/")

	// Root endpoint: /api/v1/registries
	if normalizedPath == RegistriesPath {
		if r.Method == http.MethodGet {
			h.ListRegistries(w, r)
			return
		}

		if r.Method == http.MethodPost {
			h.CreateRegistry(w, r)
			return
		}

		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only GET and POST are allowed on " + RegistriesPath,
		})
		return
	}

	// Registry-specific endpoints: /api/v1/registries/{name}
	if strings.HasPrefix(path, RegistriesPath+"/") {
		if r.Method == http.MethodGet {
			h.GetRegistry(w, r)
			return
		}

		if r.Method == http.MethodPut {
			h.UpdateRegistry(w, r)
			return
		}

		if r.Method == http.MethodDelete {
			h.DeleteRegistry(w, r)
			return
		}

		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Invalid method for registry endpoint",
		})
		return
	}

	writeJSONError(w, http.StatusNotFound, ErrorResponse{
		Error:   "not_found",
		Message: "Endpoint not found",
	})
}

// Helper functions

// registryExists checks if a registry secret with the given name already exists
func (h *Handler) registryExists(ctx context.Context, name string) bool {
	var secret corev1.Secret
	err := h.client.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: h.namespace,
	}, &secret)

	if err != nil {
		return false
	}

	// Verify it's a registry secret
	return secret.Labels[registry.AppComponentLabel] == registry.ComponentRegistry
}

// loadRegistrySecret loads a registry Secret by name
func (h *Handler) loadRegistrySecret(ctx context.Context, name string) (*corev1.Secret, error) {
	var secret corev1.Secret
	err := h.client.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: h.namespace,
	}, &secret)

	if err != nil {
		return nil, err
	}

	// Validate it's a registry secret
	if secret.Labels[registry.AppComponentLabel] != registry.ComponentRegistry {
		return nil, fmt.Errorf("secret is not a registry secret")
	}

	return &secret, nil
}

// canAccessRegistry checks if the current user can access a registry
func (h *Handler) canAccessRegistry(ctx context.Context, secret *corev1.Secret) bool {
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		return false
	}

	// Admins can access all registries
	if auth.IsAdmin(ctx) {
		return true
	}

	// Check available-to-all flag
	if secret.Labels[registry.AvailableToAllLabel] == "true" {
		return true
	}

	// Check group membership
	userGroups, err := groupauth.GetUserGroups(ctx, h.client, claims.UserID, h.namespace)
	if err != nil {
		return false
	}

	secretGroups := registry.ExtractGroupsFromLabels(secret.Labels)

	// Check if user belongs to any of the registry's groups
	userGroupNames := make(map[string]bool)
	for _, ug := range userGroups {
		userGroupNames[ug.Name] = true
	}

	for _, sg := range secretGroups {
		if userGroupNames[sg] {
			return true
		}
	}

	return false
}

// buildRegistryResponse builds a RegistryResponse from a Secret
func buildRegistryResponse(secret *corev1.Secret) registry.RegistryResponse {
	return registry.RegistryResponse{
		Name:               secret.Name,
		RegistryURL:        secret.Annotations[registry.RegistryURLAnnotation],
		ScenarioRepository: secret.Annotations[registry.ScenarioRepositoryAnnotation],
		AuthType:           secret.Labels[registry.AuthTypeLabel],
		Description:        secret.Annotations[registry.DescriptionAnnotation],
		Groups:             registry.ExtractGroupsFromLabels(secret.Labels),
		AvailableToAll:     secret.Labels[registry.AvailableToAllLabel] == "true",
		SkipTLS:            secret.Annotations[registry.SkipTLSAnnotation] == "true",
		Insecure:           secret.Annotations[registry.InsecureAnnotation] == "true",
		CreatedAt:          secret.Annotations[registry.CreatedAtAnnotation],
		CreatedBy:          secret.Annotations[registry.CreatedByAnnotation],
		UpdatedAt:          secret.Annotations[registry.UpdatedAtAnnotation],
		UpdatedBy:          secret.Annotations[registry.UpdatedByAnnotation],
	}
}

// buildRegistryInfo builds a RegistryInfo from a Secret (minimal user-facing info)
func buildRegistryInfo(secret *corev1.Secret) registry.RegistryInfo {
	return registry.RegistryInfo{
		Name:               secret.Name,
		RegistryURL:        secret.Annotations[registry.RegistryURLAnnotation],
		ScenarioRepository: secret.Annotations[registry.ScenarioRepositoryAnnotation],
		Description:        secret.Annotations[registry.DescriptionAnnotation],
	}
}

// validateCreateRegistryRequest validates a CreateRegistryRequest
func validateCreateRegistryRequest(ctx context.Context, k8sClient client.Client, req *registry.CreateRegistryRequest, namespace string) error {
	// Validate name (RFC 1123 subdomain)
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}

	// Validate registry URL
	if req.RegistryURL == "" {
		return fmt.Errorf("registryUrl is required")
	}

	// Validate scenario repository
	if req.ScenarioRepository == "" {
		return fmt.Errorf("scenarioRepository is required")
	}
	if !strings.Contains(req.ScenarioRepository, "/") {
		return fmt.Errorf("scenarioRepository must be in format 'org/repo'")
	}

	// Validate auth type
	if req.AuthType != registry.AuthTypeToken && req.AuthType != registry.AuthTypePassword {
		return fmt.Errorf("authType must be '%s' or '%s'", registry.AuthTypeToken, registry.AuthTypePassword)
	}

	// Validate credentials match auth type
	if req.AuthType == registry.AuthTypeToken && req.Token == "" {
		return fmt.Errorf("token is required for token auth type")
	}
	if req.AuthType == registry.AuthTypePassword && (req.Username == "" || req.Password == "") {
		return fmt.Errorf("username and password are required for password auth type")
	}

	// Validate groups exist
	for _, groupName := range req.Groups {
		var group krknv1alpha1.KrknUserGroup
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      groupName,
			Namespace: namespace,
		}, &group)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return fmt.Errorf("group '%s' does not exist", groupName)
			}
			return fmt.Errorf("failed to validate group '%s': %w", groupName, err)
		}
	}

	return nil
}

// validateUpdateRegistryRequest validates an UpdateRegistryRequest
func validateUpdateRegistryRequest(ctx context.Context, k8sClient client.Client, req *registry.UpdateRegistryRequest, namespace string) error {
	// Validate registry URL
	if req.RegistryURL == "" {
		return fmt.Errorf("registryUrl is required")
	}

	// Validate scenario repository
	if req.ScenarioRepository == "" {
		return fmt.Errorf("scenarioRepository is required")
	}
	if !strings.Contains(req.ScenarioRepository, "/") {
		return fmt.Errorf("scenarioRepository must be in format 'org/repo'")
	}

	// Validate auth type
	if req.AuthType != registry.AuthTypeToken && req.AuthType != registry.AuthTypePassword {
		return fmt.Errorf("authType must be '%s' or '%s'", registry.AuthTypeToken, registry.AuthTypePassword)
	}

	// Validate credentials match auth type
	if req.AuthType == registry.AuthTypeToken && req.Token == "" {
		return fmt.Errorf("token is required for token auth type")
	}
	if req.AuthType == registry.AuthTypePassword && (req.Username == "" || req.Password == "") {
		return fmt.Errorf("username and password are required for password auth type")
	}

	// Validate groups exist
	for _, groupName := range req.Groups {
		var group krknv1alpha1.KrknUserGroup
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      groupName,
			Namespace: namespace,
		}, &group)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return fmt.Errorf("group '%s' does not exist", groupName)
			}
			return fmt.Errorf("failed to validate group '%s': %w", groupName, err)
		}
	}

	return nil
}
