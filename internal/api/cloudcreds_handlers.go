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
	"github.com/krkn-chaos/krkn-operator/pkg/cloudcreds"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CreateCloudCredential handles POST /api/v1/cloud-credentials
func (h *Handler) CreateCloudCredential(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("create-cloud-credential")

	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	var req cloudcreds.CreateCloudCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := cloudcreds.ValidateCreateRequest(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	exists, err := h.cloudCredentialExists(ctx, req.Name)
	if err != nil {
		logger.Error(err, "Failed to check for existing cloud credential", "name", req.Name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to check for existing cloud credential",
		})
		return
	}
	if exists {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: fmt.Sprintf("Cloud credential '%s' already exists", req.Name),
		})
		return
	}

	claims := auth.GetClaimsFromContext(ctx)
	createdBy := ""
	if claims != nil {
		createdBy = claims.UserID
	}

	labels := cloudcreds.BuildLabels(req.Provider, req.Groups, req.AvailableToAll)
	annotations := cloudcreds.BuildAnnotations(req.Description, createdBy)
	data := buildCloudCredentialSecretData(&req)

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
		logger.Error(err, "Failed to create cloud credential secret", "name", req.Name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create cloud credential",
		})
		return
	}

	logger.Info("Created cloud credential", "name", req.Name, "provider", req.Provider, "createdBy", createdBy)

	writeJSON(w, http.StatusCreated, cloudcreds.CreateCloudCredentialResponse{
		Message: "Cloud credential created successfully",
		Name:    req.Name,
	})
}

// ListCloudCredentials handles GET /api/v1/cloud-credentials
func (h *Handler) ListCloudCredentials(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("list-cloud-credentials")

	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	var secretList corev1.SecretList
	if err := h.client.List(ctx, &secretList,
		client.InNamespace(h.namespace),
		client.MatchingLabels{
			cloudcreds.AppNameLabel:      cloudcreds.AppName,
			cloudcreds.AppComponentLabel: cloudcreds.ComponentCloudCredential,
		},
	); err != nil {
		logger.Error(err, "Failed to list cloud credential secrets")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list cloud credentials",
		})
		return
	}

	credentials := make([]cloudcreds.CloudCredentialResponse, len(secretList.Items))
	for i, secret := range secretList.Items {
		credentials[i] = buildCloudCredentialResponse(&secret)
	}

	logger.Info("Listed cloud credentials", "total", len(credentials))

	writeJSON(w, http.StatusOK, cloudcreds.ListCloudCredentialsResponse{
		Credentials: credentials,
		Total:       len(credentials),
	})
}

// GetCloudCredential handles GET /api/v1/cloud-credentials/{name}
func (h *Handler) GetCloudCredential(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("get-cloud-credential")

	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	credName, err := extractPathSuffix(r.URL.Path, CloudCredentialsPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid credential name in path",
		})
		return
	}

	secret, err := h.loadCloudCredentialSecret(ctx, credName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Cloud credential '%s' not found", credName),
			})
		} else {
			logger.Error(err, "Failed to get cloud credential", "name", credName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get cloud credential",
			})
		}
		return
	}

	logger.Info("Retrieved cloud credential", "name", credName)
	writeJSON(w, http.StatusOK, buildCloudCredentialResponse(secret))
}

// UpdateCloudCredential handles PUT /api/v1/cloud-credentials/{name}
func (h *Handler) UpdateCloudCredential(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("update-cloud-credential")

	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	credName, err := extractPathSuffix(r.URL.Path, CloudCredentialsPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid credential name in path",
		})
		return
	}

	var req cloudcreds.UpdateCloudCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	secret, err := h.loadCloudCredentialSecret(ctx, credName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Cloud credential '%s' not found", credName),
			})
		} else {
			logger.Error(err, "Failed to get cloud credential", "name", credName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get cloud credential",
			})
		}
		return
	}

	existingProvider := secret.Labels[cloudcreds.ProviderTypeLabel]
	if err := cloudcreds.ValidateUpdateRequest(&req, existingProvider); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	claims := auth.GetClaimsFromContext(ctx)
	updatedBy := ""
	if claims != nil {
		updatedBy = claims.UserID
	}

	secret.Annotations = cloudcreds.UpdateAnnotations(secret.Annotations, req.Description, updatedBy)
	secret.Labels = cloudcreds.BuildLabels(existingProvider, req.Groups, req.AvailableToAll)

	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	updateCloudCredentialSecretData(secret, existingProvider, &req)

	if err := h.client.Update(ctx, secret); err != nil {
		logger.Error(err, "Failed to update cloud credential", "name", credName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update cloud credential",
		})
		return
	}

	logger.Info("Updated cloud credential", "name", credName, "updatedBy", updatedBy)

	writeJSON(w, http.StatusOK, cloudcreds.UpdateCloudCredentialResponse{
		Message: "Cloud credential updated successfully",
		Name:    credName,
	})
}

// DeleteCloudCredential handles DELETE /api/v1/cloud-credentials/{name}
func (h *Handler) DeleteCloudCredential(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("delete-cloud-credential")

	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	credName, err := extractPathSuffix(r.URL.Path, CloudCredentialsPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid credential name in path",
		})
		return
	}

	secret, err := h.loadCloudCredentialSecret(ctx, credName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Cloud credential '%s' not found", credName),
			})
		} else {
			logger.Error(err, "Failed to get cloud credential", "name", credName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get cloud credential",
			})
		}
		return
	}

	if err := h.client.Delete(ctx, secret); err != nil {
		logger.Error(err, "Failed to delete cloud credential", "name", credName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete cloud credential",
		})
		return
	}

	logger.Info("Deleted cloud credential", "name", credName)

	writeJSON(w, http.StatusOK, cloudcreds.DeleteCloudCredentialResponse{
		Message: "Cloud credential deleted successfully",
	})
}

// ListAvailableCloudCredentials handles GET /api/v1/cloud-credentials/available
func (h *Handler) ListAvailableCloudCredentials(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("list-available-cloud-credentials")

	var secretList corev1.SecretList
	if err := h.client.List(ctx, &secretList,
		client.InNamespace(h.namespace),
		client.MatchingLabels{
			cloudcreds.AppNameLabel:      cloudcreds.AppName,
			cloudcreds.AppComponentLabel: cloudcreds.ComponentCloudCredential,
		},
	); err != nil {
		logger.Error(err, "Failed to list cloud credential secrets")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list cloud credentials",
		})
		return
	}

	var credentials []cloudcreds.CloudCredentialResponse
	for i := range secretList.Items {
		if h.canAccessCloudCredential(ctx, &secretList.Items[i]) {
			credentials = append(credentials, buildCloudCredentialResponse(&secretList.Items[i]))
		}
	}

	if credentials == nil {
		credentials = []cloudcreds.CloudCredentialResponse{}
	}

	logger.Info("Listed available cloud credentials", "total", len(credentials))

	writeJSON(w, http.StatusOK, cloudcreds.ListCloudCredentialsResponse{
		Credentials: credentials,
		Total:       len(credentials),
	})
}

// CloudCredentialsRouter routes requests to /api/v1/cloud-credentials endpoints
func (h *Handler) CloudCredentialsRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	normalizedPath := strings.TrimSuffix(path, "/")

	if normalizedPath == CloudCredentialsPath {
		switch r.Method {
		case http.MethodGet:
			h.ListCloudCredentials(w, r)
		case http.MethodPost:
			h.CreateCloudCredential(w, r)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
				Error:   "method_not_allowed",
				Message: "Only GET and POST are allowed on " + CloudCredentialsPath,
			})
		}
		return
	}

	if strings.HasPrefix(path, CloudCredentialsPath+"/") {
		switch r.Method {
		case http.MethodGet:
			h.GetCloudCredential(w, r)
		case http.MethodPut:
			h.UpdateCloudCredential(w, r)
		case http.MethodDelete:
			h.DeleteCloudCredential(w, r)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
				Error:   "method_not_allowed",
				Message: "Invalid method for cloud credential endpoint",
			})
		}
		return
	}

	writeJSONError(w, http.StatusNotFound, ErrorResponse{
		Error:   "not_found",
		Message: "Endpoint not found",
	})
}

// cloudCredentialExists checks whether a cloud credential Secret exists
func (h *Handler) cloudCredentialExists(ctx context.Context, name string) (bool, error) {
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
	return secret.Labels[cloudcreds.AppComponentLabel] == cloudcreds.ComponentCloudCredential, nil
}

// loadCloudCredentialSecret loads a cloud credential Secret by name
func (h *Handler) loadCloudCredentialSecret(ctx context.Context, name string) (*corev1.Secret, error) {
	var secret corev1.Secret
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: h.namespace,
	}, &secret); err != nil {
		return nil, err
	}

	if secret.Labels[cloudcreds.AppComponentLabel] != cloudcreds.ComponentCloudCredential {
		return nil, fmt.Errorf("secret is not a cloud credential secret")
	}

	return &secret, nil
}

// canAccessCloudCredential checks if the current user can access a cloud credential
func (h *Handler) canAccessCloudCredential(ctx context.Context, secret *corev1.Secret) bool {
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		return false
	}

	if auth.IsAdmin(ctx) {
		return true
	}

	if secret.Labels[cloudcreds.AvailableToAllLabel] == "true" {
		return true
	}

	userGroups, err := groupauth.GetUserGroups(ctx, h.client, claims.UserID, h.namespace)
	if err != nil {
		return false
	}

	secretGroups := cloudcreds.ExtractGroupsFromLabels(secret.Labels)

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

// buildCloudCredentialResponse constructs a CloudCredentialResponse from a Secret.
// Secret data is intentionally never included.
func buildCloudCredentialResponse(secret *corev1.Secret) cloudcreds.CloudCredentialResponse {
	return cloudcreds.CloudCredentialResponse{
		Name:           secret.Name,
		Provider:       secret.Labels[cloudcreds.ProviderTypeLabel],
		Description:    secret.Annotations[cloudcreds.DescriptionAnnotation],
		Groups:         cloudcreds.ExtractGroupsFromLabels(secret.Labels),
		AvailableToAll: secret.Labels[cloudcreds.AvailableToAllLabel] == "true",
		CreatedAt:      secret.Annotations[cloudcreds.CreatedAtAnnotation],
		CreatedBy:      secret.Annotations[cloudcreds.CreatedByAnnotation],
		UpdatedAt:      secret.Annotations[cloudcreds.UpdatedAtAnnotation],
		UpdatedBy:      secret.Annotations[cloudcreds.UpdatedByAnnotation],
	}
}

// buildCloudCredentialSecretData creates the Secret.Data map from a create request
func buildCloudCredentialSecretData(req *cloudcreds.CreateCloudCredentialRequest) map[string][]byte {
	data := make(map[string][]byte)

	switch req.Provider {
	case cloudcreds.ProviderAWS:
		data[cloudcreds.SecretKeyAWSAccessKeyID] = []byte(req.AWSAccessKeyID)
		data[cloudcreds.SecretKeyAWSSecretAccessKey] = []byte(req.AWSSecretAccessKey)
		data[cloudcreds.SecretKeyAWSDefaultRegion] = []byte(req.AWSDefaultRegion)
	case cloudcreds.ProviderGCP:
		data[cloudcreds.SecretKeyGCPServiceAccountJSON] = []byte(req.GCPServiceAccountJSON)
	case cloudcreds.ProviderAzure:
		data[cloudcreds.SecretKeyAzureTenantID] = []byte(req.AzureTenantID)
		data[cloudcreds.SecretKeyAzureClientID] = []byte(req.AzureClientID)
		data[cloudcreds.SecretKeyAzureClientSecret] = []byte(req.AzureClientSecret)
		data[cloudcreds.SecretKeyAzureSubscriptionID] = []byte(req.AzureSubscriptionID)
	case cloudcreds.ProviderOpenStack:
		data[cloudcreds.SecretKeyOSAuthURL] = []byte(req.OSAuthURL)
		data[cloudcreds.SecretKeyOSUsername] = []byte(req.OSUsername)
		data[cloudcreds.SecretKeyOSPassword] = []byte(req.OSPassword)
		data[cloudcreds.SecretKeyOSProjectName] = []byte(req.OSProjectName)
		if req.OSDomainName != "" {
			data[cloudcreds.SecretKeyOSDomainName] = []byte(req.OSDomainName)
		}
	}

	return data
}

// updateCloudCredentialSecretData updates Secret.Data fields from an update request.
// Empty strings mean "keep existing value".
func updateCloudCredentialSecretData(secret *corev1.Secret, provider string, req *cloudcreds.UpdateCloudCredentialRequest) {
	switch provider {
	case cloudcreds.ProviderAWS:
		if req.AWSAccessKeyID != "" {
			secret.Data[cloudcreds.SecretKeyAWSAccessKeyID] = []byte(req.AWSAccessKeyID)
		}
		if req.AWSSecretAccessKey != "" {
			secret.Data[cloudcreds.SecretKeyAWSSecretAccessKey] = []byte(req.AWSSecretAccessKey)
		}
		if req.AWSDefaultRegion != "" {
			secret.Data[cloudcreds.SecretKeyAWSDefaultRegion] = []byte(req.AWSDefaultRegion)
		}
	case cloudcreds.ProviderGCP:
		if req.GCPServiceAccountJSON != "" {
			secret.Data[cloudcreds.SecretKeyGCPServiceAccountJSON] = []byte(req.GCPServiceAccountJSON)
		}
	case cloudcreds.ProviderAzure:
		if req.AzureTenantID != "" {
			secret.Data[cloudcreds.SecretKeyAzureTenantID] = []byte(req.AzureTenantID)
		}
		if req.AzureClientID != "" {
			secret.Data[cloudcreds.SecretKeyAzureClientID] = []byte(req.AzureClientID)
		}
		if req.AzureClientSecret != "" {
			secret.Data[cloudcreds.SecretKeyAzureClientSecret] = []byte(req.AzureClientSecret)
		}
		if req.AzureSubscriptionID != "" {
			secret.Data[cloudcreds.SecretKeyAzureSubscriptionID] = []byte(req.AzureSubscriptionID)
		}
	case cloudcreds.ProviderOpenStack:
		if req.OSAuthURL != "" {
			secret.Data[cloudcreds.SecretKeyOSAuthURL] = []byte(req.OSAuthURL)
		}
		if req.OSUsername != "" {
			secret.Data[cloudcreds.SecretKeyOSUsername] = []byte(req.OSUsername)
		}
		if req.OSPassword != "" {
			secret.Data[cloudcreds.SecretKeyOSPassword] = []byte(req.OSPassword)
		}
		if req.OSProjectName != "" {
			secret.Data[cloudcreds.SecretKeyOSProjectName] = []byte(req.OSProjectName)
		}
		if req.OSDomainName != "" {
			secret.Data[cloudcreds.SecretKeyOSDomainName] = []byte(req.OSDomainName)
		}
	}
}
