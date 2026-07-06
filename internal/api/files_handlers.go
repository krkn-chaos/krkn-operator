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

	"github.com/google/uuid"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// CreateFile handles POST /api/v1/files
// Creates a new file ConfigMap (authenticated users)
// Users can create files for their own groups or public files
// Admins can create files for any group
func (h *Handler) CreateFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("create-file")

	// Parse request body
	var req files.CreateFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Get current user info
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
		return
	}
	isAdmin := auth.IsAdmin(ctx)

	// Validate request
	if err := validateCreateFileRequest(ctx, h.client, &req, h.namespace, isAdmin, claims.UserID); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	// Generate unique file ID (UUID)
	fileID := uuid.New().String()

	// Generate ConfigMap name from file ID
	configMapName := fmt.Sprintf("file-%s", fileID)

	// Get current user for audit trail
	createdBy := claims.UserID

	// Auto-create file type if specified and doesn't exist
	if req.FileType != "" {
		if err := h.ensureFileTypeExists(ctx, req.FileType, createdBy); err != nil {
			logger.Error(err, "Failed to ensure file type exists", "fileType", req.FileType)
			// Continue anyway - file type is optional metadata
		}
	}

	// Build labels and annotations
	labels := files.BuildFileLabels(fileID, req.FileType, req.Groups, req.AvailableToAll)
	annotations := files.BuildFileAnnotations(
		req.Description,
		createdBy,
	)

	// Create ConfigMap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        configMapName,
			Namespace:   h.namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Data: map[string]string{
			req.FileName: req.Content,
		},
	}

	if err := h.client.Create(ctx, configMap); err != nil {
		logger.Error(err, "Failed to create file ConfigMap", "fileID", fileID, "configMapName", configMapName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create file",
		})
		return
	}

	logger.Info("Created file", "fileID", fileID, "fileName", req.FileName, "createdBy", createdBy)

	writeJSON(w, http.StatusCreated, files.CreateFileResponse{
		Message: "File created successfully",
		FileID:  fileID,
	})
}

// ListFiles handles GET /api/v1/files
// Lists all files (admin only)
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("list-files")

	// Check admin privileges
	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// List all file ConfigMaps
	var configMapList corev1.ConfigMapList
	err := h.client.List(ctx, &configMapList,
		client.InNamespace(h.namespace),
		client.MatchingLabels{
			files.AppNameLabel:      files.AppName,
			files.AppComponentLabel: files.ComponentFile,
		},
	)
	if err != nil {
		logger.Error(err, "Failed to list file ConfigMaps")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list files",
		})
		return
	}

	// Convert to response format
	fileList := make([]files.FileResponse, len(configMapList.Items))
	for i, cm := range configMapList.Items {
		fileList[i] = buildFileResponse(&cm)
	}

	logger.Info("Listed files", "total", len(fileList))

	writeJSON(w, http.StatusOK, files.ListFilesResponse{
		Files: fileList,
		Total: len(fileList),
	})
}

// GetFile handles GET /api/v1/files/{fileId}
// Gets a single file by UUID (authenticated users with access)
func (h *Handler) GetFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("get-file")

	// Extract file ID from path
	fileID, err := extractPathSuffix(r.URL.Path, FilesPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid file ID in path",
		})
		return
	}

	// Load ConfigMap by file ID
	configMap, err := h.loadFileConfigMapByID(ctx, fileID)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("File with ID '%s' not found", fileID),
			})
		} else {
			logger.Error(err, "Failed to get file", "fileID", fileID)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get file",
			})
		}
		return
	}

	// Check access permissions
	// Admin can read any file, users can only read files they have access to
	if !auth.IsAdmin(ctx) {
		if !h.canAccessFile(ctx, configMap) {
			writeJSONError(w, http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have access to this file",
			})
			return
		}
	}

	logger.Info("Retrieved file", "fileID", fileID)
	writeJSON(w, http.StatusOK, buildFileResponse(configMap))
}

// UpdateFile handles PUT /api/v1/files/{fileId}
// Updates a file by UUID (authenticated users with access)
// Users can update files from their own groups
// Admins can update any file
func (h *Handler) UpdateFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("update-file")

	// Extract file ID from path
	fileID, err := extractPathSuffix(r.URL.Path, FilesPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid file ID in path",
		})
		return
	}

	// Parse request body
	var req files.UpdateFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Get current user info
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
		return
	}
	isAdmin := auth.IsAdmin(ctx)

	// Validate request
	if err := validateUpdateFileRequest(ctx, h.client, &req, h.namespace, isAdmin, claims.UserID); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	// Load existing ConfigMap by file ID
	configMap, err := h.loadFileConfigMapByID(ctx, fileID)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("File with ID '%s' not found", fileID),
			})
		} else {
			logger.Error(err, "Failed to get file", "fileID", fileID)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get file",
			})
		}
		return
	}

	// Check access permissions - users can only update files they have access to
	if !isAdmin {
		if !h.canAccessFile(ctx, configMap) {
			writeJSONError(w, http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have access to this file",
			})
			return
		}
	}

	// Get current user for audit trail
	updatedBy := claims.UserID

	// Auto-create file type if specified and doesn't exist
	if req.FileType != "" {
		if err := h.ensureFileTypeExists(ctx, req.FileType, updatedBy); err != nil {
			logger.Error(err, "Failed to ensure file type exists", "fileType", req.FileType)
			// Continue anyway - file type is optional metadata
		}
	}

	// Update labels and annotations (preserve existing file ID)
	configMap.Labels = files.BuildFileLabels(fileID, req.FileType, req.Groups, req.AvailableToAll)
	configMap.Annotations = files.UpdateFileAnnotations(
		configMap.Annotations,
		req.Description,
		updatedBy,
	)

	// Update data
	configMap.Data = map[string]string{
		req.FileName: req.Content,
	}

	if err := h.client.Update(ctx, configMap); err != nil {
		logger.Error(err, "Failed to update file", "fileID", fileID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update file",
		})
		return
	}

	logger.Info("Updated file", "fileID", fileID, "updatedBy", updatedBy)

	writeJSON(w, http.StatusOK, files.UpdateFileResponse{
		Message: "File updated successfully",
		FileID:  fileID,
	})
}

// DeleteFile handles DELETE /api/v1/files/{fileId}
// Deletes a file by UUID (authenticated users with access)
// Users can delete files from their own groups
// Admins can delete any file
func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("delete-file")

	// Extract file ID from path
	fileID, err := extractPathSuffix(r.URL.Path, FilesPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid file ID in path",
		})
		return
	}

	// Load ConfigMap by file ID (to verify it exists and is a file)
	configMap, err := h.loadFileConfigMapByID(ctx, fileID)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("File with ID '%s' not found", fileID),
			})
		} else {
			logger.Error(err, "Failed to get file", "fileID", fileID)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get file",
			})
		}
		return
	}

	// Check access permissions - users can only delete files they have access to
	isAdmin := auth.IsAdmin(ctx)
	if !isAdmin {
		if !h.canAccessFile(ctx, configMap) {
			writeJSONError(w, http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have access to this file",
			})
			return
		}
	}

	// Delete ConfigMap
	if err := h.client.Delete(ctx, configMap); err != nil {
		logger.Error(err, "Failed to delete file", "fileID", fileID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete file",
		})
		return
	}

	logger.Info("Deleted file", "fileID", fileID)

	writeJSON(w, http.StatusOK, files.DeleteFileResponse{
		Message: "File deleted successfully",
	})
}

// ListAvailableFiles handles GET /api/v1/files/available
// Lists files available to the current user
func (h *Handler) ListAvailableFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("list-available-files")

	// Get user claims
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
		return
	}

	// Admins see all files
	if auth.IsAdmin(ctx) {
		var configMapList corev1.ConfigMapList
		err := h.client.List(ctx, &configMapList,
			client.InNamespace(h.namespace),
			client.MatchingLabels{
				files.AppNameLabel:      files.AppName,
				files.AppComponentLabel: files.ComponentFile,
			},
		)
		if err != nil {
			logger.Error(err, "Failed to list file ConfigMaps")
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to list files",
			})
			return
		}

		fileList := make([]files.FileInfo, len(configMapList.Items))
		for i, cm := range configMapList.Items {
			fileList[i] = buildFileInfo(&cm)
		}

		writeJSON(w, http.StatusOK, files.AvailableFilesResponse{
			Files: fileList,
		})
		return
	}

	// List all file ConfigMaps
	var configMapList corev1.ConfigMapList
	err := h.client.List(ctx, &configMapList,
		client.InNamespace(h.namespace),
		client.MatchingLabels{
			files.AppNameLabel:      files.AppName,
			files.AppComponentLabel: files.ComponentFile,
		},
	)
	if err != nil {
		logger.Error(err, "Failed to list file ConfigMaps")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list files",
		})
		return
	}

	// Filter files by access
	available := []files.FileInfo{}
	for _, cm := range configMapList.Items {
		if h.canAccessFile(ctx, &cm) {
			available = append(available, buildFileInfo(&cm))
		}
	}

	logger.Info("Listed available files", "userID", claims.UserID, "total", len(available))

	writeJSON(w, http.StatusOK, files.AvailableFilesResponse{
		Files: available,
	})
}

// FilesRouter routes requests to /api/v1/files endpoints
func (h *Handler) FilesRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Normalize path
	normalizedPath := strings.TrimSuffix(path, "/")

	// Special endpoint: /api/v1/files/available
	if normalizedPath == FilesPath+"/available" {
		if r.Method == http.MethodGet {
			h.ListAvailableFiles(w, r)
			return
		}

		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only GET is allowed on " + FilesPath + "/available",
		})
		return
	}

	// Root endpoint: /api/v1/files
	if normalizedPath == FilesPath {
		if r.Method == http.MethodGet {
			h.ListFiles(w, r)
			return
		}

		if r.Method == http.MethodPost {
			h.CreateFile(w, r)
			return
		}

		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only GET and POST are allowed on " + FilesPath,
		})
		return
	}

	// File-specific endpoints: /api/v1/files/{fileId}
	if strings.HasPrefix(path, FilesPath+"/") {
		if r.Method == http.MethodGet {
			h.GetFile(w, r)
			return
		}

		if r.Method == http.MethodPut {
			h.UpdateFile(w, r)
			return
		}

		if r.Method == http.MethodDelete {
			h.DeleteFile(w, r)
			return
		}

		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Invalid method for file endpoint",
		})
		return
	}

	writeJSONError(w, http.StatusNotFound, ErrorResponse{
		Error:   "not_found",
		Message: "Endpoint not found",
	})
}

// Helper functions

// loadFileConfigMapByID loads a file ConfigMap by file ID (UUID)
func (h *Handler) loadFileConfigMapByID(ctx context.Context, fileID string) (*corev1.ConfigMap, error) {
	// List ConfigMaps with the file ID label
	var configMapList corev1.ConfigMapList
	err := h.client.List(ctx, &configMapList,
		client.InNamespace(h.namespace),
		client.MatchingLabels{
			files.AppNameLabel:      files.AppName,
			files.AppComponentLabel: files.ComponentFile,
			files.FileIDLabel:       fileID,
		},
	)

	if err != nil {
		return nil, err
	}

	if len(configMapList.Items) == 0 {
		return nil, apierrors.NewNotFound(corev1.Resource("configmap"), fileID)
	}

	if len(configMapList.Items) > 1 {
		return nil, fmt.Errorf("multiple ConfigMaps found with file ID '%s'", fileID)
	}

	return &configMapList.Items[0], nil
}

// canAccessFile checks if the current user can access a file
func (h *Handler) canAccessFile(ctx context.Context, configMap *corev1.ConfigMap) bool {
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		return false
	}

	// Admins can access all files
	if auth.IsAdmin(ctx) {
		return true
	}

	// Check available-to-all flag
	if configMap.Labels[files.AvailableToAllLabel] == "true" {
		return true
	}

	// Check group membership
	userGroups, err := groupauth.GetUserGroups(ctx, h.client, claims.UserID, h.namespace)
	if err != nil {
		return false
	}

	configMapGroups := files.ExtractGroupsFromLabels(configMap.Labels)

	// Check if user belongs to any of the file's groups
	userGroupNames := make(map[string]bool)
	for _, ug := range userGroups {
		userGroupNames[ug.Name] = true
	}

	for _, sg := range configMapGroups {
		if userGroupNames[sg] {
			return true
		}
	}

	return false
}

// buildFileResponse builds a FileResponse from a ConfigMap
func buildFileResponse(configMap *corev1.ConfigMap) files.FileResponse {
	// Extract first file name and content from data
	fileName := ""
	content := ""
	for k, v := range configMap.Data {
		fileName = k
		content = v
		break
	}

	return files.FileResponse{
		FileID:         files.ExtractFileIDFromLabels(configMap.Labels),
		FileName:       fileName,
		Content:        content,
		Description:    configMap.Annotations[files.DescriptionAnnotation],
		FileType:       files.ExtractFileTypeFromLabels(configMap.Labels),
		Groups:         files.ExtractGroupsFromLabels(configMap.Labels),
		AvailableToAll: configMap.Labels[files.AvailableToAllLabel] == "true",
		CreatedAt:      configMap.Annotations[files.CreatedAtAnnotation],
		CreatedBy:      configMap.Annotations[files.CreatedByAnnotation],
		UpdatedAt:      configMap.Annotations[files.UpdatedAtAnnotation],
		UpdatedBy:      configMap.Annotations[files.UpdatedByAnnotation],
	}
}

// buildFileInfo builds a FileInfo from a ConfigMap (minimal user-facing info)
func buildFileInfo(configMap *corev1.ConfigMap) files.FileInfo {
	// Extract first file name from data
	fileName := ""
	for k := range configMap.Data {
		fileName = k
		break
	}

	return files.FileInfo{
		FileID:      files.ExtractFileIDFromLabels(configMap.Labels),
		FileName:    fileName,
		Description: configMap.Annotations[files.DescriptionAnnotation],
		FileType:    files.ExtractFileTypeFromLabels(configMap.Labels),
	}
}

// validateCreateFileRequest validates a CreateFileRequest
func validateCreateFileRequest(ctx context.Context, k8sClient client.Client, req *files.CreateFileRequest, namespace string, isAdmin bool, userID string) error {
	// Validate file name
	if req.FileName == "" {
		return fmt.Errorf("fileName is required")
	}

	// Validate content is not empty
	if req.Content == "" {
		return fmt.Errorf("content is required")
	}

	// Validate content is valid JSON or YAML
	if err := files.ValidateFileContent(req.Content); err != nil {
		return err
	}

	// Validate file groups (max 1, mutually exclusive with availableToAll)
	if err := files.ValidateFileGroups(req.Groups, req.AvailableToAll); err != nil {
		return err
	}

	// Get user's groups for permission validation
	userGroupsObjs, err := groupauth.GetUserGroups(ctx, k8sClient, userID, namespace)
	if err != nil {
		return fmt.Errorf("failed to get user groups: %w", err)
	}

	// Extract group names
	userGroups := make([]string, len(userGroupsObjs))
	for i, ug := range userGroupsObjs {
		userGroups[i] = ug.Name
	}

	// Validate user permissions (non-admin can assign to their own group or make public)
	if err := files.ValidateUserFilePermissions(isAdmin, req.Groups, req.AvailableToAll, userGroups); err != nil {
		return err
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

// validateUpdateFileRequest validates an UpdateFileRequest
func validateUpdateFileRequest(ctx context.Context, k8sClient client.Client, req *files.UpdateFileRequest, namespace string, isAdmin bool, userID string) error {
	// Validate file name
	if req.FileName == "" {
		return fmt.Errorf("fileName is required")
	}

	// Validate content is not empty
	if req.Content == "" {
		return fmt.Errorf("content is required")
	}

	// Validate content is valid JSON or YAML
	if err := files.ValidateFileContent(req.Content); err != nil {
		return err
	}

	// Validate file groups (max 1, mutually exclusive with availableToAll)
	if err := files.ValidateFileGroups(req.Groups, req.AvailableToAll); err != nil {
		return err
	}

	// Get user's groups for permission validation
	userGroupsObjs, err := groupauth.GetUserGroups(ctx, k8sClient, userID, namespace)
	if err != nil {
		return fmt.Errorf("failed to get user groups: %w", err)
	}

	// Extract group names
	userGroups := make([]string, len(userGroupsObjs))
	for i, ug := range userGroupsObjs {
		userGroups[i] = ug.Name
	}

	// Validate user permissions (non-admin can assign to their own group or make public)
	if err := files.ValidateUserFilePermissions(isAdmin, req.Groups, req.AvailableToAll, userGroups); err != nil {
		return err
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

// ensureFileTypeExists creates a KrknFileType if it doesn't exist (auto-creation pattern).
// This allows users to use file types without having to create them explicitly first.
// If the type already exists, this is a no-op.
func (h *Handler) ensureFileTypeExists(ctx context.Context, typeName, createdBy string) error {
	logger := log.FromContext(ctx).WithName("ensure-file-type")

	var fileType krknv1alpha1.KrknFileType
	err := h.client.Get(ctx, client.ObjectKey{
		Name:      typeName,
		Namespace: h.namespace,
	}, &fileType)

	if err == nil {
		// Already exists
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check file type existence: %w", err)
	}

	// Create new file type with defaults (empty color = UI will use defaults)
	newType := &krknv1alpha1.KrknFileType{
		ObjectMeta: metav1.ObjectMeta{
			Name:      typeName,
			Namespace: h.namespace,
		},
		Spec: krknv1alpha1.KrknFileTypeSpec{
			Name:  typeName,
			Color: "", // Empty = use UI default
		},
	}

	if err := h.client.Create(ctx, newType); err != nil {
		return fmt.Errorf("failed to auto-create file type: %w", err)
	}

	logger.Info("Auto-created file type", "typeName", typeName, "createdBy", createdBy)
	return nil
}
