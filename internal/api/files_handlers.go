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
// @Summary Create file
// @Description Create a new file ConfigMap. Users can create files for their own groups or public files. Admins can create files for any group. Cannot create workflow-template files (use POST /api/v1/workflows instead).
// @Tags files
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param file body files.CreateFileRequest true "File data"
// @Success 201 {object} files.CreateFileResponse "File created"
// @Failure 400 {object} ErrorResponse "Invalid request or validation error"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 409 {object} ErrorResponse "File name already exists"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /files [post]
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
		logger.Info("File validation failed",
			"fileName", req.FileName,
			"fileType", req.FileType,
			"groups", req.Groups,
			"availableToAll", req.AvailableToAll,
			"error", err.Error())
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	// Default filePurpose to "file" if not specified
	if req.FilePurpose == "" {
		req.FilePurpose = files.FilePurposeFile
	}

	// Validate filePurpose - only workflow API can create workflow-template files
	if req.FilePurpose == files.FilePurposeWorkflow {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Use POST /api/v1/workflows to create workflow templates",
		})
		return
	}

	// Validate filePurpose is a known value
	if !files.IsValidFilePurpose(req.FilePurpose) {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: fmt.Sprintf("Invalid filePurpose '%s'. Valid values: %s", req.FilePurpose, strings.Join(files.ValidFilePurposes(), ", ")),
		})
		return
	}

	// Only workflow-template files can set workflowName
	if req.FilePurpose != files.FilePurposeWorkflow {
		req.WorkflowName = ""
	}

	// Check for duplicate logical name (global uniqueness)
	logicalName := deriveLogicalName(req.FileName, req.WorkflowName)
	existingID, err := h.checkDuplicateLogicalName(ctx, logicalName, "")
	if err != nil {
		logger.Error(err, "Failed to check for duplicate file name", "logicalName", logicalName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to validate file name uniqueness",
		})
		return
	}
	if existingID != "" {
		logger.Info("duplicate file name detected", "logicalName", logicalName, "existingID", existingID)
		writeJSONError(w, http.StatusConflict, ErrorResponse{
			Error:   "conflict",
			Message: fmt.Sprintf("A file with name '%s' already exists", logicalName),
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
	labels := files.BuildFileLabels(fileID, req.FileType, req.Groups, req.AvailableToAll, req.FilePurpose, logicalName)
	annotations := files.BuildFileAnnotations(
		req.Description,
		createdBy,
		req.WorkflowName, // Empty for regular files, populated for workflows
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
// @Summary List files (admin only)
// @Description Get list of all file ConfigMaps (admin only). Supports filtering by filePurpose query parameter.
// @Tags files
// @Produce json
// @Security BearerAuth
// @Param filePurpose query string false "Filter by file purpose (file, workflow-template, resiliency-score)"
// @Success 200 {object} files.ListFilesResponse "List of files"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden (admin only)"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /files [get]
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

	// Build label filters
	matchingLabels := map[string]string{
		files.AppNameLabel:      files.AppName,
		files.AppComponentLabel: files.ComponentFile,
	}

	// Add optional filePurpose filter from query parameter
	filePurpose := r.URL.Query().Get("filePurpose")
	if filePurpose != "" {
		matchingLabels[files.FilePurposeLabel] = filePurpose
	}

	// List all file ConfigMaps
	var configMapList corev1.ConfigMapList
	err := h.client.List(ctx, &configMapList,
		client.InNamespace(h.namespace),
		client.MatchingLabels(matchingLabels),
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
		hasAccess, err := h.canAccessFile(ctx, configMap)
		if err != nil {
			logger.Error(err, "Failed to check file access", "fileID", fileID)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to validate file access permissions",
			})
			return
		}
		if !hasAccess {
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
		logger.Info("File update validation failed",
			"fileID", fileID,
			"fileName", req.FileName,
			"fileType", req.FileType,
			"groups", req.Groups,
			"availableToAll", req.AvailableToAll,
			"error", err.Error())
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	// Block workflow-template reclassification via /files
	if req.FilePurpose == files.FilePurposeWorkflow {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Use POST /api/v1/workflows to manage workflow templates",
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

	// Preserve existing filePurpose — /files cannot reclassify resources
	existingPurpose := files.ExtractFilePurposeFromLabels(configMap.Labels)
	if existingPurpose != "" {
		req.FilePurpose = existingPurpose
	}

	// Check ownership - only owner or admin can update files
	isOwner, err := h.isFileOwnerOrAdmin(ctx, configMap)
	if err != nil {
		logger.Error(err, "Failed to check file ownership", "fileID", fileID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to validate file ownership",
		})
		return
	}
	if !isOwner {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Only the file owner or an admin can update this file",
		})
		return
	}

	// Only workflow-template files can set workflowName
	workflowName := ""
	if req.FilePurpose == files.FilePurposeWorkflow && req.WorkflowName != nil {
		workflowName = *req.WorkflowName
	}

	// Check for duplicate logical name on rename (exclude current file)
	logicalName := deriveLogicalName(req.FileName, workflowName)
	existingID, err := h.checkDuplicateLogicalName(ctx, logicalName, fileID)
	if err != nil {
		logger.Error(err, "Failed to check for duplicate file name", "logicalName", logicalName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to validate file name uniqueness",
		})
		return
	}
	if existingID != "" {
		logger.Info("duplicate file name detected on update", "logicalName", logicalName, "existingID", existingID)
		writeJSONError(w, http.StatusConflict, ErrorResponse{
			Error:   "conflict",
			Message: fmt.Sprintf("A file with name '%s' already exists", logicalName),
		})
		return
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
	configMap.Labels = files.BuildFileLabels(fileID, req.FileType, req.Groups, req.AvailableToAll, req.FilePurpose, logicalName)
	configMap.Annotations = files.UpdateFileAnnotations(
		configMap.Annotations,
		req.Description,
		updatedBy,
		req.WorkflowName, // Pointer: nil preserves existing, non-nil updates/deletes
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

	// Check ownership - only owner or admin can delete files
	isOwner, err := h.isFileOwnerOrAdmin(ctx, configMap)
	if err != nil {
		logger.Error(err, "Failed to check file ownership", "fileID", fileID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to validate file ownership",
		})
		return
	}
	if !isOwner {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Only the file owner or an admin can delete this file",
		})
		return
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
// @Summary List available files
// @Description Get files accessible to current user (own files, group files, public files). Supports filtering by filePurpose query parameter.
// @Tags files
// @Produce json
// @Security BearerAuth
// @Param filePurpose query string false "Filter by file purpose (file, workflow-template, resiliency-score)"
// @Success 200 {object} files.AvailableFilesResponse "List of accessible files"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /files/available [get]
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

	// Build label filters
	matchingLabels := map[string]string{
		files.AppNameLabel:      files.AppName,
		files.AppComponentLabel: files.ComponentFile,
	}

	// Add optional filePurpose filter from query parameter
	filePurpose := r.URL.Query().Get("filePurpose")
	if filePurpose != "" {
		matchingLabels[files.FilePurposeLabel] = filePurpose
	}

	// Admins see all files
	if auth.IsAdmin(ctx) {
		var configMapList corev1.ConfigMapList
		err := h.client.List(ctx, &configMapList,
			client.InNamespace(h.namespace),
			client.MatchingLabels(matchingLabels),
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

	// List all file ConfigMaps (reuse label filters from above)
	var configMapList corev1.ConfigMapList
	err := h.client.List(ctx, &configMapList,
		client.InNamespace(h.namespace),
		client.MatchingLabels(matchingLabels),
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
		hasAccess, err := h.canAccessFile(ctx, &cm)
		if err != nil {
			logger.Error(err, "Failed to check file access", "fileID", files.ExtractFileIDFromLabels(cm.Labels))
			// Skip files we can't validate access for
			continue
		}
		if hasAccess {
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
func (h *Handler) canAccessFile(ctx context.Context, configMap *corev1.ConfigMap) (bool, error) {
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		return false, nil
	}

	// Admins can access all files
	if auth.IsAdmin(ctx) {
		return true, nil
	}

	// Check available-to-all flag
	if configMap.Labels[files.AvailableToAllLabel] == "true" {
		return true, nil
	}

	// Check group membership
	userGroups, err := groupauth.GetUserGroups(ctx, h.client, claims.UserID, h.namespace)
	if err != nil {
		return false, fmt.Errorf("failed to get user groups: %w", err)
	}

	configMapGroups := files.ExtractGroupsFromLabels(configMap.Labels)

	// Check if user belongs to any of the file's groups
	userGroupNames := make(map[string]bool)
	for _, ug := range userGroups {
		userGroupNames[ug.Name] = true
	}

	for _, sg := range configMapGroups {
		if userGroupNames[sg] {
			return true, nil
		}
	}

	return false, nil
}

// isFileOwnerOrAdmin checks if the current user is the owner of a file or an admin
// This is used for mutation operations (update/delete) where only owner or admin should have access
// For files: ownership is determined by group membership + created-by annotation
func (h *Handler) isFileOwnerOrAdmin(ctx context.Context, configMap *corev1.ConfigMap) (bool, error) {
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		return false, nil
	}

	// Admins can modify all files
	if auth.IsAdmin(ctx) {
		return true, nil
	}

	// Check if user is the creator via created-by annotation
	createdBy := configMap.Annotations[files.CreatedByAnnotation]
	if createdBy != "" && createdBy == claims.UserID {
		return true, nil
	}

	// If file has no groups and is not available-to-all, deny access
	configMapGroups := files.ExtractGroupsFromLabels(configMap.Labels)
	if len(configMapGroups) == 0 && configMap.Labels[files.AvailableToAllLabel] != "true" {
		return false, nil
	}

	// For all files (public and group), only creator or admin can modify
	// Group membership grants READ access only, not write/delete
	return false, nil
}

// buildFileResponse builds a FileResponse from a ConfigMap
func buildFileResponse(configMap *corev1.ConfigMap) files.FileResponse {
	// Extract file name, content, and studioLayout from data
	// studioLayout.json is separate from the main file content
	fileName := ""
	content := ""
	studioLayout := ""
	for k, v := range configMap.Data {
		if k == "studioLayout.json" {
			studioLayout = v
		} else {
			fileName = k
			content = v
		}
	}

	// Get workflow name with backwards-compatible fallback
	workflowName := configMap.Annotations[files.WorkflowNameAnnotation]
	if workflowName == "" && files.ExtractFilePurposeFromLabels(configMap.Labels) == files.FilePurposeWorkflow {
		// Fallback for workflows created before workflowName annotation was added
		workflowName = fileName
	}

	return files.FileResponse{
		FileID:         files.ExtractFileIDFromLabels(configMap.Labels),
		FileName:       fileName,
		Content:        content,
		StudioLayout:   studioLayout,
		WorkflowName:   workflowName,
		Description:    configMap.Annotations[files.DescriptionAnnotation],
		FileType:       files.ExtractFileTypeFromLabels(configMap.Labels),
		FilePurpose:    files.ExtractFilePurposeFromLabels(configMap.Labels),
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
	// Extract primary file name from data (exclude studioLayout.json)
	fileName := ""
	for k := range configMap.Data {
		if k != "studioLayout.json" {
			fileName = k
			break
		}
	}

	return files.FileInfo{
		FileID:      files.ExtractFileIDFromLabels(configMap.Labels),
		FileName:    fileName,
		Description: configMap.Annotations[files.DescriptionAnnotation],
		FileType:    files.ExtractFileTypeFromLabels(configMap.Labels),
		FilePurpose: files.ExtractFilePurposeFromLabels(configMap.Labels),
		CreatedAt:   configMap.Annotations[files.CreatedAtAnnotation],
		UpdatedAt:   configMap.Annotations[files.UpdatedAtAnnotation],
	}
}

// extractLogicalName derives the logical name from an existing file ConfigMap.
// For workflows, the logical name is the workflowName annotation.
// For regular files, it is the primary Data key (excluding studioLayout.json).
func extractLogicalName(cm *corev1.ConfigMap) string {
	if files.ExtractFilePurposeFromLabels(cm.Labels) == files.FilePurposeWorkflow {
		if wn := cm.Annotations[files.WorkflowNameAnnotation]; wn != "" {
			return wn
		}
	}
	for k := range cm.Data {
		if k != "studioLayout.json" {
			return k
		}
	}
	return ""
}

// deriveLogicalName derives the logical name from request fields.
// If workflowName is set, it takes precedence (workflow identity).
// Otherwise, fileName is used (regular file identity).
func deriveLogicalName(fileName, workflowName string) string {
	if workflowName != "" {
		return workflowName
	}
	return fileName
}

// checkDuplicateLogicalName checks if a file with the same logical name already exists.
// Uses a label selector on the logical-name hash for efficient server-side filtering.
// excludeFileID can be set to skip a specific file (for update operations).
// Returns the existing file's ID if a duplicate is found, or empty string if no conflict.
func (h *Handler) checkDuplicateLogicalName(ctx context.Context, logicalName, excludeFileID string) (string, error) {
	var configMapList corev1.ConfigMapList
	err := h.client.List(ctx, &configMapList,
		client.InNamespace(h.namespace),
		client.MatchingLabels{
			files.AppNameLabel:        files.AppName,
			files.AppComponentLabel:   files.ComponentFile,
			files.LogicalNameHashLabel: files.HashLogicalName(logicalName),
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to list file ConfigMaps for duplicate check: %w", err)
	}

	// Verify exact match (hash collisions are theoretically possible)
	for i := range configMapList.Items {
		cm := &configMapList.Items[i]
		existingID := files.ExtractFileIDFromLabels(cm.Labels)
		if excludeFileID != "" && existingID == excludeFileID {
			continue
		}
		if extractLogicalName(cm) == logicalName {
			return existingID, nil
		}
	}
	return "", nil
}

// validateCreateFileRequest validates a CreateFileRequest
// isValidConfigMapKey checks if a string is a valid ConfigMap data key
// ConfigMap keys must consist of alphanumeric characters, '-', '_' or '.'
func isValidConfigMapKey(key string) bool {
	if key == "" {
		return false
	}
	for _, ch := range key {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.') {
			return false
		}
	}
	return true
}

func validateCreateFileRequest(ctx context.Context, k8sClient client.Client, req *files.CreateFileRequest, namespace string, isAdmin bool, userID string) error {
	// Validate file name
	if req.FileName == "" {
		return fmt.Errorf("fileName is required")
	}

	// Validate fileName is a valid ConfigMap key (alphanumeric, -, _, .)
	// ConfigMap keys must match: [a-zA-Z0-9._-]+
	if !isValidConfigMapKey(req.FileName) {
		return fmt.Errorf("fileName contains invalid characters (allowed: alphanumeric, -, _, .)")
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

	// Validate fileName is a valid ConfigMap key (alphanumeric, -, _, .)
	if !isValidConfigMapKey(req.FileName) {
		return fmt.Errorf("fileName contains invalid characters (allowed: alphanumeric, -, _, .)")
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
