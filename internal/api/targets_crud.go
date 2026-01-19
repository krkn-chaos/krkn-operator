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

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/kubeconfig"
)

// CreateTarget handles POST /api/v1/operator/targets
// Creates a new KrknOperatorTarget CR with a generated UUID and associated Secret
func (h *Handler) CreateTarget(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Parse request body
	var req CreateTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.ClusterName == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "clusterName is required",
		})
		return
	}

	if req.SecretType == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "secretType is required (kubeconfig, token, or credentials)",
		})
		return
	}

	// Validate secretType
	if req.SecretType != "kubeconfig" && req.SecretType != "token" && req.SecretType != "credentials" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "secretType must be one of: kubeconfig, token, credentials",
		})
		return
	}

	// Generate kubeconfig and validate based on secretType
	var kubeconfigBase64 string
	var apiURL string
	var err error

	switch req.SecretType {
	case "kubeconfig":
		// Validate kubeconfig is provided
		if req.Kubeconfig == "" {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "kubeconfig is required when secretType is 'kubeconfig'",
			})
			return
		}

		// Validate kubeconfig format
		if err := kubeconfig.Validate(req.Kubeconfig); err != nil {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Invalid kubeconfig: " + err.Error(),
			})
			return
		}

		// Extract API URL from kubeconfig
		apiURL, err = kubeconfig.ExtractAPIURL(req.Kubeconfig)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Failed to extract API URL from kubeconfig: " + err.Error(),
			})
			return
		}

		kubeconfigBase64 = req.Kubeconfig

	case "token":
		// Validate token and clusterAPIURL
		if req.Token == "" {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "token is required when secretType is 'token'",
			})
			return
		}

		if req.ClusterAPIURL == "" {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "clusterAPIURL is required when secretType is 'token'",
			})
			return
		}

		// Determine if TLS should be skipped
		insecureSkipTLS := req.CABundle == ""

		// Generate kubeconfig from token
		kubeconfigBase64, err = kubeconfig.GenerateFromToken(
			req.ClusterName,
			req.ClusterAPIURL,
			req.CABundle,
			req.Token,
			insecureSkipTLS,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to generate kubeconfig from token: " + err.Error(),
			})
			return
		}

		apiURL = req.ClusterAPIURL

	case "credentials":
		// Validate credentials and clusterAPIURL
		if req.Username == "" || req.Password == "" {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "username and password are required when secretType is 'credentials'",
			})
			return
		}

		if req.ClusterAPIURL == "" {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "clusterAPIURL is required when secretType is 'credentials'",
			})
			return
		}

		// Determine if TLS should be skipped
		insecureSkipTLS := req.CABundle == ""

		// Generate kubeconfig from credentials
		kubeconfigBase64, err = kubeconfig.GenerateFromCredentials(
			req.ClusterName,
			req.ClusterAPIURL,
			req.CABundle,
			req.Username,
			req.Password,
			insecureSkipTLS,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to generate kubeconfig from credentials: " + err.Error(),
			})
			return
		}

		apiURL = req.ClusterAPIURL
	}

	// Check for duplicate clusterName or clusterAPIURL
	var existingTargets krknv1alpha1.KrknOperatorTargetList
	if err := h.client.List(ctx, &existingTargets, client.InNamespace(h.namespace)); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to check existing targets: " + err.Error(),
		})
		return
	}

	for _, target := range existingTargets.Items {
		if target.Spec.ClusterName == req.ClusterName {
			writeJSONError(w, http.StatusConflict, ErrorResponse{
				Error:   "conflict",
				Message: fmt.Sprintf("Target with clusterName '%s' already exists", req.ClusterName),
			})
			return
		}

		if target.Spec.ClusterAPIURL != "" && target.Spec.ClusterAPIURL == apiURL {
			writeJSONError(w, http.StatusConflict, ErrorResponse{
				Error:   "conflict",
				Message: fmt.Sprintf("Target with clusterAPIURL '%s' already exists", apiURL),
			})
			return
		}
	}

	// Generate UUIDs
	targetUUID := uuid.New().String()
	secretUUID := uuid.New().String()

	// Create Secret with kubeconfig
	secretData, err := kubeconfig.MarshalSecretData(kubeconfigBase64)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to marshal secret data: " + err.Error(),
		})
		return
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretUUID,
			Namespace: h.namespace,
			Labels: map[string]string{
				"krkn-target-uuid": targetUUID,
			},
		},
		Data: map[string][]byte{
			"kubeconfig": secretData,
		},
	}

	if err := h.client.Create(ctx, secret); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create secret: " + err.Error(),
		})
		return
	}

	// Create KrknOperatorTarget CR
	target := &krknv1alpha1.KrknOperatorTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      targetUUID,
			Namespace: h.namespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetSpec{
			UUID:                  targetUUID,
			ClusterName:           req.ClusterName,
			ClusterAPIURL:         apiURL,
			SecretType:            req.SecretType,
			SecretUUID:            secretUUID,
			CABundle:              req.CABundle,
			InsecureSkipTLSVerify: req.CABundle == "",
		},
		Status: krknv1alpha1.KrknOperatorTargetStatus{
			Ready:       true,
			LastUpdated: metav1.Now(),
		},
	}

	if err := h.client.Create(ctx, target); err != nil {
		// Cleanup secret on error
		h.client.Delete(ctx, secret)

		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create target: " + err.Error(),
		})
		return
	}

	// Return success response
	response := CreateTargetResponse{
		UUID:    targetUUID,
		Message: "Target created successfully",
	}

	writeJSON(w, http.StatusCreated, response)
}

// ListTargets handles GET /api/v1/operator/targets
// Returns a list of all KrknOperatorTarget CRs
func (h *Handler) ListTargets(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// List all targets
	var targets krknv1alpha1.KrknOperatorTargetList
	if err := h.client.List(ctx, &targets, client.InNamespace(h.namespace)); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list targets: " + err.Error(),
		})
		return
	}

	// Convert to response format
	var targetResponses []TargetResponse
	for _, target := range targets.Items {
		createdAt := target.CreationTimestamp.Time
		targetResponses = append(targetResponses, TargetResponse{
			UUID:          target.Spec.UUID,
			ClusterName:   target.Spec.ClusterName,
			ClusterAPIURL: target.Spec.ClusterAPIURL,
			SecretType:    target.Spec.SecretType,
			Ready:         target.Status.Ready,
			CreatedAt:     &createdAt,
		})
	}

	response := ListTargetsResponse{
		Targets: targetResponses,
	}

	writeJSON(w, http.StatusOK, response)
}

// GetTarget handles GET /api/v1/operator/targets/{uuid}
// Returns a single KrknOperatorTarget by UUID
func (h *Handler) GetTarget(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Extract UUID from path
	path := r.URL.Path
	prefix := "/api/v1/operator/targets/"

	if len(path) <= len(prefix) {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "UUID parameter is required in path",
		})
		return
	}

	targetUUID := path[len(prefix):]
	if targetUUID == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "UUID parameter cannot be empty",
		})
		return
	}

	// Fetch the target
	var target krknv1alpha1.KrknOperatorTarget
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      targetUUID,
		Namespace: h.namespace,
	}, &target); err != nil {
		if client.IgnoreNotFound(err) == nil {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Target with UUID '%s' not found", targetUUID),
			})
			return
		}

		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get target: " + err.Error(),
		})
		return
	}

	// Convert to response format
	createdAt := target.CreationTimestamp.Time
	response := TargetResponse{
		UUID:          target.Spec.UUID,
		ClusterName:   target.Spec.ClusterName,
		ClusterAPIURL: target.Spec.ClusterAPIURL,
		SecretType:    target.Spec.SecretType,
		Ready:         target.Status.Ready,
		CreatedAt:     &createdAt,
	}

	writeJSON(w, http.StatusOK, response)
}

// UpdateTarget handles PUT /api/v1/operator/targets/{uuid}
// Updates an existing KrknOperatorTarget (overwrites the Secret kubeconfig)
func (h *Handler) UpdateTarget(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Extract UUID from path
	path := r.URL.Path
	prefix := "/api/v1/operator/targets/"

	if len(path) <= len(prefix) {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "UUID parameter is required in path",
		})
		return
	}

	targetUUID := path[len(prefix):]
	if targetUUID == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "UUID parameter cannot be empty",
		})
		return
	}

	// Parse request body
	var req UpdateTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Fetch existing target
	var target krknv1alpha1.KrknOperatorTarget
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      targetUUID,
		Namespace: h.namespace,
	}, &target); err != nil {
		if client.IgnoreNotFound(err) == nil {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Target with UUID '%s' not found", targetUUID),
			})
			return
		}

		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get target: " + err.Error(),
		})
		return
	}

	// Generate new kubeconfig based on secretType
	var kubeconfigBase64 string
	var apiURL string
	var err error

	switch req.SecretType {
	case "kubeconfig":
		if req.Kubeconfig == "" {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "kubeconfig is required when secretType is 'kubeconfig'",
			})
			return
		}

		if err := kubeconfig.Validate(req.Kubeconfig); err != nil {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Invalid kubeconfig: " + err.Error(),
			})
			return
		}

		apiURL, err = kubeconfig.ExtractAPIURL(req.Kubeconfig)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Failed to extract API URL from kubeconfig: " + err.Error(),
			})
			return
		}

		kubeconfigBase64 = req.Kubeconfig

	case "token":
		if req.Token == "" || req.ClusterAPIURL == "" {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "token and clusterAPIURL are required when secretType is 'token'",
			})
			return
		}

		insecureSkipTLS := req.CABundle == ""

		kubeconfigBase64, err = kubeconfig.GenerateFromToken(
			req.ClusterName,
			req.ClusterAPIURL,
			req.CABundle,
			req.Token,
			insecureSkipTLS,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to generate kubeconfig from token: " + err.Error(),
			})
			return
		}

		apiURL = req.ClusterAPIURL

	case "credentials":
		if req.Username == "" || req.Password == "" || req.ClusterAPIURL == "" {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "username, password, and clusterAPIURL are required when secretType is 'credentials'",
			})
			return
		}

		insecureSkipTLS := req.CABundle == ""

		kubeconfigBase64, err = kubeconfig.GenerateFromCredentials(
			req.ClusterName,
			req.ClusterAPIURL,
			req.CABundle,
			req.Username,
			req.Password,
			insecureSkipTLS,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to generate kubeconfig from credentials: " + err.Error(),
			})
			return
		}

		apiURL = req.ClusterAPIURL
	}

	// Update Secret with new kubeconfig
	var secret corev1.Secret
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      target.Spec.SecretUUID,
		Namespace: h.namespace,
	}, &secret); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get secret: " + err.Error(),
		})
		return
	}

	secretData, err := kubeconfig.MarshalSecretData(kubeconfigBase64)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to marshal secret data: " + err.Error(),
		})
		return
	}

	secret.Data["kubeconfig"] = secretData

	if err := h.client.Update(ctx, &secret); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update secret: " + err.Error(),
		})
		return
	}

	// Update KrknOperatorTarget CR
	if req.ClusterName != "" {
		target.Spec.ClusterName = req.ClusterName
	}
	target.Spec.ClusterAPIURL = apiURL
	target.Spec.SecretType = req.SecretType
	target.Spec.CABundle = req.CABundle
	target.Spec.InsecureSkipTLSVerify = req.CABundle == ""
	target.Status.LastUpdated = metav1.Now()

	if err := h.client.Update(ctx, &target); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update target: " + err.Error(),
		})
		return
	}

	response := CreateTargetResponse{
		UUID:    targetUUID,
		Message: "Target updated successfully",
	}

	writeJSON(w, http.StatusOK, response)
}

// DeleteTarget handles DELETE /api/v1/operator/targets/{uuid}
// Deletes a KrknOperatorTarget and its associated Secret
func (h *Handler) DeleteTarget(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Extract UUID from path
	path := r.URL.Path
	prefix := "/api/v1/operator/targets/"

	if len(path) <= len(prefix) {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "UUID parameter is required in path",
		})
		return
	}

	targetUUID := path[len(prefix):]
	if targetUUID == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "UUID parameter cannot be empty",
		})
		return
	}

	// Fetch the target
	var target krknv1alpha1.KrknOperatorTarget
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      targetUUID,
		Namespace: h.namespace,
	}, &target); err != nil {
		if client.IgnoreNotFound(err) == nil {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Target with UUID '%s' not found", targetUUID),
			})
			return
		}

		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get target: " + err.Error(),
		})
		return
	}

	// Delete associated Secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      target.Spec.SecretUUID,
			Namespace: h.namespace,
		},
	}

	if err := h.client.Delete(ctx, secret); err != nil && client.IgnoreNotFound(err) != nil {
		// Log error but continue with target deletion
		// The secret might have been manually deleted
	}

	// Delete the target
	if err := h.client.Delete(ctx, &target); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete target: " + err.Error(),
		})
		return
	}

	response := CreateTargetResponse{
		UUID:    targetUUID,
		Message: "Target deleted successfully",
	}

	writeJSON(w, http.StatusOK, response)
}

// TargetsCRUDRouter routes requests to /api/v1/operator/targets endpoints
func (h *Handler) TargetsCRUDRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// POST /api/v1/operator/targets - create new target
	if path == "/api/v1/operator/targets" && r.Method == http.MethodPost {
		h.CreateTarget(w, r)
		return
	}

	// GET /api/v1/operator/targets - list all targets
	if path == "/api/v1/operator/targets" && r.Method == http.MethodGet {
		h.ListTargets(w, r)
		return
	}

	// Path with UUID: /api/v1/operator/targets/{uuid}
	if strings.HasPrefix(path, "/api/v1/operator/targets/") {
		// GET /api/v1/operator/targets/{uuid} - get single target
		if r.Method == http.MethodGet {
			h.GetTarget(w, r)
			return
		}

		// PUT /api/v1/operator/targets/{uuid} - update target
		if r.Method == http.MethodPut {
			h.UpdateTarget(w, r)
			return
		}

		// DELETE /api/v1/operator/targets/{uuid} - delete target
		if r.Method == http.MethodDelete {
			h.DeleteTarget(w, r)
			return
		}
	}

	// Method not allowed
	writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
		Error:   "method_not_allowed",
		Message: "Method " + r.Method + " not allowed for path " + path,
	})
}
