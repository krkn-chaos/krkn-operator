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

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	"github.com/krkn-chaos/krkn-operator/pkg/filetypes"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// FileTypesRouter routes file type requests
func (h *Handler) FileTypesRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Check if this is a single file type request or list
		if strings.TrimPrefix(r.URL.Path, FileTypesPath+"/") != r.URL.Path {
			h.GetFileType(w, r)
		} else {
			h.ListFileTypes(w, r)
		}
	case http.MethodPost:
		h.CreateFileType(w, r)
	case http.MethodPut:
		h.UpdateFileType(w, r)
	case http.MethodDelete:
		h.DeleteFileType(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Method not allowed",
		})
	}
}

// CreateFileType handles POST /api/v1/file-types
// Creates a new file type (admin only)
func (h *Handler) CreateFileType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("create-file-type")

	// Check admin privileges
	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Parse request body
	var req filetypes.CreateFileTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate request
	if err := filetypes.ValidateCreateRequest(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	// Check if file type already exists
	var existing krknv1alpha1.KrknFileType
	err := h.client.Get(ctx, client.ObjectKey{
		Name:      req.Name,
		Namespace: h.namespace,
	}, &existing)
	if err == nil {
		writeJSONError(w, http.StatusConflict, ErrorResponse{
			Error:   "conflict",
			Message: fmt.Sprintf("File type '%s' already exists", req.Name),
		})
		return
	}
	if !apierrors.IsNotFound(err) {
		logger.Error(err, "Failed to check file type existence", "typeName", req.Name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to check file type existence",
		})
		return
	}

	// Create KrknFileType
	fileType := &krknv1alpha1.KrknFileType{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: h.namespace,
		},
		Spec: krknv1alpha1.KrknFileTypeSpec{
			Name:  req.Name,
			Color: req.Color,
		},
	}

	if err := h.client.Create(ctx, fileType); err != nil {
		logger.Error(err, "Failed to create file type", "typeName", req.Name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create file type",
		})
		return
	}

	logger.Info("Created file type", "typeName", req.Name)

	resp := buildFileTypeResponse(fileType, 0) // 0 usage since just created
	writeJSON(w, http.StatusCreated, resp)
}

// ListFileTypes handles GET /api/v1/file-types
// Lists all file types (all authenticated users)
func (h *Handler) ListFileTypes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("list-file-types")

	// List all KrknFileType resources
	var fileTypeList krknv1alpha1.KrknFileTypeList
	if err := h.client.List(ctx, &fileTypeList, client.InNamespace(h.namespace)); err != nil {
		logger.Error(err, "Failed to list file types")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list file types",
		})
		return
	}

	// Build response with usage counts
	responses := make([]filetypes.FileTypeResponse, len(fileTypeList.Items))
	for i, ft := range fileTypeList.Items {
		usageCount := h.getFileTypeUsageCount(ctx, ft.Spec.Name)
		responses[i] = buildFileTypeResponse(&ft, usageCount)
	}

	writeJSON(w, http.StatusOK, filetypes.ListFileTypesResponse{
		FileTypes: responses,
		Total:     len(responses),
	})
}

// GetFileType handles GET /api/v1/file-types/{name}
// Gets a single file type (all authenticated users)
func (h *Handler) GetFileType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("get-file-type")

	// Extract file type name from path
	typeName, err := extractPathSuffix(r.URL.Path, FileTypesPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid file type name in path",
		})
		return
	}

	// Get KrknFileType
	var fileType krknv1alpha1.KrknFileType
	err = h.client.Get(ctx, client.ObjectKey{
		Name:      typeName,
		Namespace: h.namespace,
	}, &fileType)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("File type '%s' not found", typeName),
			})
		} else {
			logger.Error(err, "Failed to get file type", "typeName", typeName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get file type",
			})
		}
		return
	}

	usageCount := h.getFileTypeUsageCount(ctx, typeName)
	resp := buildFileTypeResponse(&fileType, usageCount)
	writeJSON(w, http.StatusOK, resp)
}

// UpdateFileType handles PUT /api/v1/file-types/{name}
// Updates a file type (admin only)
func (h *Handler) UpdateFileType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("update-file-type")

	// Check admin privileges
	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Extract file type name from path
	typeName, err := extractPathSuffix(r.URL.Path, FileTypesPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid file type name in path",
		})
		return
	}

	// Parse request body
	var req filetypes.UpdateFileTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate request
	if err := filetypes.ValidateUpdateRequest(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	// Get existing file type
	var fileType krknv1alpha1.KrknFileType
	err = h.client.Get(ctx, client.ObjectKey{
		Name:      typeName,
		Namespace: h.namespace,
	}, &fileType)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("File type '%s' not found", typeName),
			})
		} else {
			logger.Error(err, "Failed to get file type", "typeName", typeName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get file type",
			})
		}
		return
	}

	// Update spec
	fileType.Spec.Color = req.Color

	if err := h.client.Update(ctx, &fileType); err != nil {
		logger.Error(err, "Failed to update file type", "typeName", typeName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update file type",
		})
		return
	}

	logger.Info("Updated file type", "typeName", typeName)

	usageCount := h.getFileTypeUsageCount(ctx, typeName)
	resp := buildFileTypeResponse(&fileType, usageCount)
	writeJSON(w, http.StatusOK, resp)
}

// DeleteFileType handles DELETE /api/v1/file-types/{name}
// Deletes a file type (admin only)
// Returns 409 Conflict if files are using this type
func (h *Handler) DeleteFileType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("delete-file-type")

	// Check admin privileges
	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Extract file type name from path
	typeName, err := extractPathSuffix(r.URL.Path, FileTypesPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid file type name in path",
		})
		return
	}

	// Get existing file type
	var fileType krknv1alpha1.KrknFileType
	err = h.client.Get(ctx, client.ObjectKey{
		Name:      typeName,
		Namespace: h.namespace,
	}, &fileType)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("File type '%s' not found", typeName),
			})
		} else {
			logger.Error(err, "Failed to get file type", "typeName", typeName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get file type",
			})
		}
		return
	}

	// Check if any files are using this type
	filesUsingType, err := h.getFilesUsingFileType(ctx, typeName)
	if err != nil {
		logger.Error(err, "Failed to check file type usage", "typeName", typeName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to check file type usage",
		})
		return
	}

	if len(filesUsingType) > 0 {
		fileNames := make([]string, len(filesUsingType))
		for i, cm := range filesUsingType {
			fileNames[i] = cm.Name
		}
		writeJSONError(w, http.StatusConflict, ErrorResponse{
			Error:   "conflict",
			Message: fmt.Sprintf("Cannot delete file type '%s' - currently used by %d files: %v. Remove the type from all files first.", typeName, len(filesUsingType), fileNames),
		})
		return
	}

	// Delete the file type
	if err := h.client.Delete(ctx, &fileType); err != nil {
		logger.Error(err, "Failed to delete file type", "typeName", typeName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete file type",
		})
		return
	}

	logger.Info("Deleted file type", "typeName", typeName)

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "File type deleted successfully",
	})
}

// getFileTypeUsageCount returns the number of files using a specific file type
func (h *Handler) getFileTypeUsageCount(ctx context.Context, typeName string) int {
	files, err := h.getFilesUsingFileType(ctx, typeName)
	if err != nil {
		return 0
	}
	return len(files)
}

// getFilesUsingFileType returns all ConfigMaps (files) that have the specified file type label
func (h *Handler) getFilesUsingFileType(ctx context.Context, typeName string) ([]corev1.ConfigMap, error) {
	typeLabel := filetypes.BuildFileTypeLabel(typeName)

	var configMapList corev1.ConfigMapList
	err := h.client.List(ctx, &configMapList,
		client.InNamespace(h.namespace),
		client.MatchingLabels{
			files.AppNameLabel:      files.AppName,
			files.AppComponentLabel: files.ComponentFile,
			typeLabel:               "true",
		})

	if err != nil {
		return nil, err
	}

	return configMapList.Items, nil
}

// buildFileTypeResponse builds a FileTypeResponse from a KrknFileType
func buildFileTypeResponse(fileType *krknv1alpha1.KrknFileType, usageCount int) filetypes.FileTypeResponse {
	createdAt := ""
	if !fileType.CreationTimestamp.IsZero() {
		createdAt = fileType.CreationTimestamp.Format("2006-01-02T15:04:05Z07:00")
	}

	return filetypes.FileTypeResponse{
		Name:       fileType.Spec.Name,
		Color:      fileType.Spec.Color,
		CreatedAt:  createdAt,
		UsageCount: usageCount,
	}
}
