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
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	"github.com/krkn-chaos/krkn-operator/pkg/workflows"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CreateWorkflow handles POST /api/v1/workflows
// @Summary Create workflow template
// @Description Create a new workflow template. Validates graph structure (DAG, no cycles). Users can create templates for their own groups or public. Admins can create for any group.
// @Tags workflows
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workflow body workflows.CreateWorkflowRequest true "Workflow data with graph definition"
// @Success 201 {object} workflows.CreateWorkflowResponse "Workflow created"
// @Failure 400 {object} ErrorResponse "Invalid request or graph validation error"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /workflows [post]
func (h *Handler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("create-workflow")

	// Parse workflow request
	var req workflows.CreateWorkflowRequest
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

	// Validate workflow name (required field)
	if req.WorkflowName == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Workflow name is required",
		})
		return
	}

	// Validate graph structure (workflow-specific validation)
	if err := workflows.ValidateWorkflowGraph(req.Graph); err != nil {
		logger.Info("Workflow graph validation failed",
			"workflowName", req.WorkflowName,
			"error", err.Error())
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_graph",
			Message: err.Error(),
		})
		return
	}

	// Convert graph to JSON string
	content, err := workflows.ToFileContent(req.Graph)
	if err != nil {
		logger.Error(err, "Failed to marshal workflow graph")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to serialize workflow graph",
		})
		return
	}

	// Delegate to file creation (reuse ALL file logic)
	fileReq := files.CreateFileRequest{
		FileName:       "workflow.json", // Standard filename for workflows
		Content:        content,         // Graph JSON
		Description:    req.Description,
		FileType:       req.FileType,        // User categorization (optional)
		Groups:         req.Groups,          // RBAC groups
		AvailableToAll: req.AvailableToAll,  // Public flag
		FilePurpose:    "workflow-template", // System marker
	}

	// Call existing CreateFile handler logic
	fileResp, err := h.createFileInternal(ctx, fileReq)
	if err != nil {
		// Distinguish validation errors (4xx) from internal errors (5xx)
		// Only known validation errors should return 400
		statusCode := http.StatusInternalServerError
		errorCode := "internal_error"
		if strings.Contains(err.Error(), "you can only assign files to your own group") {
			statusCode = http.StatusBadRequest
			errorCode = "bad_request"
		}
		writeJSONError(w, statusCode, ErrorResponse{
			Error:   errorCode,
			Message: err.Error(),
		})
		return
	}

	logger.Info("Created workflow",
		"workflowID", fileResp.FileID,
		"workflowName", req.WorkflowName,
		"createdBy", claims.UserID)

	writeJSON(w, http.StatusCreated, workflows.CreateWorkflowResponse{
		Message:    "Workflow created successfully",
		WorkflowID: fileResp.FileID,
	})
}

// WorkflowsRouter routes workflow requests based on method and path
func (h *Handler) WorkflowsRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.CreateWorkflow(w, r)
	case http.MethodGet:
		// Check if path ends with "/available"
		if strings.HasSuffix(r.URL.Path, "/available") {
			h.ListAvailableWorkflows(w, r)
		} else {
			// Check if there's a workflow ID in the path
			workflowID := extractIDFromPath(r.URL.Path, WorkflowsPath)
			if workflowID != "" {
				h.GetWorkflow(w, r)
			} else {
				h.ListWorkflows(w, r)
			}
		}
	case http.MethodPut:
		h.UpdateWorkflow(w, r)
	case http.MethodDelete:
		h.DeleteWorkflow(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Method not allowed",
		})
	}
}

// ListWorkflows handles GET /api/v1/workflows
// @Summary List all workflow templates (admin only)
// @Description Get list of all workflow templates in the system (admin only).
// @Tags workflows
// @Produce json
// @Security BearerAuth
// @Success 200 {object} workflows.ListWorkflowsResponse "List of workflows"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden (admin only)"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /workflows [get]
func (h *Handler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("list-workflows")

	// Check admin privileges
	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Call file list with filePurpose filter
	fileList, err := h.listFilesInternal(ctx, "workflow-template")
	if err != nil {
		logger.Error(err, "Failed to list workflow files")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list workflows",
		})
		return
	}

	// Convert file responses to workflow responses
	workflowList := make([]workflows.WorkflowResponse, 0, len(fileList))
	for _, fileResp := range fileList {
		workflowResp, err := convertFileResponseToWorkflow(fileResp)
		if err != nil {
			logger.Error(err, "Failed to convert file to workflow", "fileID", fileResp.FileID)
			continue // Skip invalid workflows
		}
		workflowList = append(workflowList, workflowResp)
	}

	writeJSON(w, http.StatusOK, workflows.ListWorkflowsResponse{
		Workflows: workflowList,
		Total:     len(workflowList),
	})
}

// ListAvailableWorkflows handles GET /api/v1/workflows/available
// @Summary List available workflow templates
// @Description Get workflows accessible to current user (own workflows, group workflows, public workflows). Includes node count excluding metadata nodes.
// @Tags workflows
// @Produce json
// @Security BearerAuth
// @Success 200 {object} workflows.AvailableWorkflowsResponse "List of accessible workflows"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 405 {object} ErrorResponse "Method not allowed (GET only)"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /workflows/available [get]
func (h *Handler) ListAvailableWorkflows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("list-available-workflows")

	// Get user claims
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
		return
	}

	// Call file list with filePurpose filter (returns ConfigMaps)
	configMaps, err := h.listAvailableFilesInternal(ctx, "workflow-template")
	if err != nil {
		logger.Error(err, "Failed to list available workflow files")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list workflows",
		})
		return
	}

	// Convert ConfigMaps to workflow info
	workflowList := make([]workflows.WorkflowInfo, 0, len(configMaps))
	for _, cm := range configMaps {
		workflowInfo := convertConfigMapToWorkflowInfo(&cm)
		workflowList = append(workflowList, workflowInfo)
	}

	writeJSON(w, http.StatusOK, workflows.AvailableWorkflowsResponse{
		Workflows: workflowList,
	})
}

// GetWorkflow handles GET /api/v1/workflows/{workflowId}
// Gets a workflow template by ID
func (h *Handler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("get-workflow")

	// Extract workflow ID from path
	workflowID := extractIDFromPath(r.URL.Path, WorkflowsPath)
	if workflowID == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Workflow ID is required",
		})
		return
	}

	// Load file by ID
	fileResp, err := h.getFileByIDInternal(ctx, workflowID)
	if err != nil {
		logger.Error(err, "Failed to get workflow file", "workflowID", workflowID)
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Workflow not found",
		})
		return
	}

	// Verify it's a workflow file
	if fileResp.FilePurpose != "workflow-template" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "File is not a workflow template",
		})
		return
	}

	// Convert to workflow response
	workflowResp, err := convertFileResponseToWorkflow(*fileResp)
	if err != nil {
		logger.Error(err, "Failed to parse workflow graph", "workflowID", workflowID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to parse workflow graph",
		})
		return
	}

	logger.Info("Retrieved workflow", "workflowID", workflowID)
	writeJSON(w, http.StatusOK, workflowResp)
}

// UpdateWorkflow handles PUT /api/v1/workflows/{workflowId}
// Updates a workflow template (owner or admin only)
func (h *Handler) UpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("update-workflow")

	// Extract workflow ID from path
	workflowID := extractIDFromPath(r.URL.Path, WorkflowsPath)
	if workflowID == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Workflow ID is required",
		})
		return
	}

	// Parse workflow request
	var req workflows.UpdateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate workflow name (required field)
	if req.WorkflowName == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Workflow name is required",
		})
		return
	}

	// Validate graph structure
	if err := workflows.ValidateWorkflowGraph(req.Graph); err != nil {
		logger.Info("Workflow graph validation failed",
			"workflowID", workflowID,
			"error", err.Error())
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_graph",
			Message: err.Error(),
		})
		return
	}

	// Convert graph to JSON string
	content, err := workflows.ToFileContent(req.Graph)
	if err != nil {
		logger.Error(err, "Failed to marshal workflow graph")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to serialize workflow graph",
		})
		return
	}

	// Delegate to file update
	fileReq := files.UpdateFileRequest{
		FileName:       "workflow.json",
		Content:        content,
		Description:    req.Description,
		FileType:       req.FileType,
		Groups:         req.Groups,
		AvailableToAll: req.AvailableToAll,
		FilePurpose:    "workflow-template",
	}

	err = h.updateFileInternal(ctx, workflowID, fileReq)
	if err != nil {
		logger.Error(err, "Failed to update workflow", "workflowID", workflowID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
		return
	}

	logger.Info("Updated workflow", "workflowID", workflowID)
	writeJSON(w, http.StatusOK, workflows.UpdateWorkflowResponse{
		Message:    "Workflow updated successfully",
		WorkflowID: workflowID,
	})
}

// DeleteWorkflow handles DELETE /api/v1/workflows/{workflowId}
// Deletes a workflow template (owner or admin only)
func (h *Handler) DeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("delete-workflow")

	// Extract workflow ID from path
	workflowID := extractIDFromPath(r.URL.Path, WorkflowsPath)
	if workflowID == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Workflow ID is required",
		})
		return
	}

	// Verify the file is actually a workflow before deleting
	configMap, err := h.loadFileConfigMapByID(ctx, workflowID)
	if err != nil {
		logger.Error(err, "Failed to load workflow", "workflowID", workflowID)
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Workflow not found",
		})
		return
	}

	// Check file purpose
	if files.ExtractFilePurposeFromLabels(configMap.Labels) != "workflow-template" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "File is not a workflow template",
		})
		return
	}

	// Delegate to file deletion
	err = h.deleteFileInternal(ctx, workflowID)
	if err != nil {
		logger.Error(err, "Failed to delete workflow", "workflowID", workflowID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
		return
	}

	logger.Info("Deleted workflow", "workflowID", workflowID)
	writeJSON(w, http.StatusOK, workflows.DeleteWorkflowResponse{
		Message: "Workflow deleted successfully",
	})
}

// Helper functions

// convertFileResponseToWorkflow converts a FileResponse to a WorkflowResponse
func convertFileResponseToWorkflow(fileResp files.FileResponse) (workflows.WorkflowResponse, error) {
	// Parse graph from file content
	graph, err := workflows.FromFileContent(fileResp.Content)
	if err != nil {
		return workflows.WorkflowResponse{}, err
	}

	return workflows.WorkflowResponse{
		WorkflowID:     fileResp.FileID,
		WorkflowName:   fileResp.FileName, // Use filename as workflow name for now
		Description:    fileResp.Description,
		Graph:          graph,
		FileType:       fileResp.FileType,
		Groups:         fileResp.Groups,
		AvailableToAll: fileResp.AvailableToAll,
		CreatedAt:      fileResp.CreatedAt,
		CreatedBy:      fileResp.CreatedBy,
		UpdatedAt:      fileResp.UpdatedAt,
		UpdatedBy:      fileResp.UpdatedBy,
	}, nil
}

// convertConfigMapToWorkflowInfo converts a ConfigMap to WorkflowInfo with accurate NodeCount
func convertConfigMapToWorkflowInfo(cm *corev1.ConfigMap) workflows.WorkflowInfo {
	fileInfo := buildFileInfo(cm)

	// Parse graph to count nodes (exclude metadata nodes starting with _)
	nodeCount := 0
	if content, exists := cm.Data["workflow.json"]; exists {
		graph, err := workflows.FromFileContent(content)
		if err == nil {
			for nodeID := range graph {
				if !strings.HasPrefix(nodeID, "_") {
					nodeCount++
				}
			}
		}
	}

	return workflows.WorkflowInfo{
		WorkflowID:   fileInfo.FileID,
		WorkflowName: fileInfo.FileName,
		Description:  fileInfo.Description,
		FileType:     fileInfo.FileType,
		NodeCount:    nodeCount,
	}
}

// createFileInternal extracts file creation logic for reuse by workflow handlers
func (h *Handler) createFileInternal(ctx context.Context, req files.CreateFileRequest) (*files.CreateFileResponse, error) {
	logger := log.FromContext(ctx).WithName("create-file-internal")

	// Get current user for audit trail
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		return nil, fmt.Errorf("authentication required")
	}

	isAdmin := auth.IsAdmin(ctx)

	// Validate request
	if err := validateCreateFileRequest(ctx, h.client, &req, h.namespace, isAdmin, claims.UserID); err != nil {
		return nil, err
	}

	// Generate unique file ID (UUID)
	fileID := uuid.New().String()
	configMapName := fmt.Sprintf("file-%s", fileID)
	createdBy := claims.UserID

	// Auto-create file type if specified and doesn't exist
	if req.FileType != "" {
		if err := h.ensureFileTypeExists(ctx, req.FileType, createdBy); err != nil {
			logger.Error(err, "Failed to ensure file type exists", "fileType", req.FileType)
			// Continue anyway - file type is optional metadata
		}
	}

	// Build labels and annotations
	labels := files.BuildFileLabels(fileID, req.FileType, req.Groups, req.AvailableToAll, req.FilePurpose)
	annotations := files.BuildFileAnnotations(req.Description, createdBy)

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
		return nil, fmt.Errorf("failed to create file ConfigMap: %w", err)
	}

	logger.Info("Created file", "fileID", fileID, "fileName", req.FileName, "createdBy", createdBy)

	return &files.CreateFileResponse{
		Message: "File created successfully",
		FileID:  fileID,
	}, nil
}

// listFilesInternal extracts file listing logic with filePurpose filter
func (h *Handler) listFilesInternal(ctx context.Context, filePurpose string) ([]files.FileResponse, error) {
	logger := log.FromContext(ctx).WithName("list-files-internal")

	// Build label filters
	matchingLabels := map[string]string{
		files.AppNameLabel:      files.AppName,
		files.AppComponentLabel: files.ComponentFile,
	}

	// Add filePurpose filter if specified
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
		return nil, fmt.Errorf("failed to list file ConfigMaps: %w", err)
	}

	// Convert to response format
	fileList := make([]files.FileResponse, len(configMapList.Items))
	for i, cm := range configMapList.Items {
		fileList[i] = buildFileResponse(&cm)
	}

	logger.Info("Listed files", "total", len(fileList), "filePurpose", filePurpose)
	return fileList, nil
}

// listAvailableFilesInternal extracts available file listing logic with filePurpose filter
func (h *Handler) listAvailableFilesInternal(ctx context.Context, filePurpose string) ([]corev1.ConfigMap, error) {
	logger := log.FromContext(ctx).WithName("list-available-files-internal")

	// Get user claims
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		return nil, fmt.Errorf("authentication required")
	}

	// Build label filters
	matchingLabels := map[string]string{
		files.AppNameLabel:      files.AppName,
		files.AppComponentLabel: files.ComponentFile,
	}

	// Add filePurpose filter if specified
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
			return nil, fmt.Errorf("failed to list file ConfigMaps: %w", err)
		}

		return configMapList.Items, nil
	}

	// List all file ConfigMaps (reuse label filters from above)
	var configMapList corev1.ConfigMapList
	err := h.client.List(ctx, &configMapList,
		client.InNamespace(h.namespace),
		client.MatchingLabels(matchingLabels),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list file ConfigMaps: %w", err)
	}

	// Filter files by access
	available := []corev1.ConfigMap{}
	for _, cm := range configMapList.Items {
		hasAccess, err := h.canAccessFile(ctx, &cm)
		if err != nil {
			logger.Error(err, "Failed to check file access", "fileID", files.ExtractFileIDFromLabels(cm.Labels))
			// Skip files we can't validate access for
			continue
		}
		if hasAccess {
			available = append(available, cm)
		}
	}

	logger.Info("Listed available files", "userID", claims.UserID, "total", len(available), "filePurpose", filePurpose)
	return available, nil
}

// getFileByIDInternal extracts file retrieval logic for reuse by workflow handlers
func (h *Handler) getFileByIDInternal(ctx context.Context, fileID string) (*files.FileResponse, error) {
	logger := log.FromContext(ctx).WithName("get-file-internal")

	// Load ConfigMap by file ID
	configMap, err := h.loadFileConfigMapByID(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	// Check access permissions
	if !auth.IsAdmin(ctx) {
		hasAccess, err := h.canAccessFile(ctx, configMap)
		if err != nil {
			return nil, fmt.Errorf("failed to check file access: %w", err)
		}
		if !hasAccess {
			return nil, fmt.Errorf("access denied")
		}
	}

	logger.Info("Retrieved file", "fileID", fileID)
	resp := buildFileResponse(configMap)
	return &resp, nil
}

// updateFileInternal extracts file update logic for reuse by workflow handlers
func (h *Handler) updateFileInternal(ctx context.Context, fileID string, req files.UpdateFileRequest) error {
	logger := log.FromContext(ctx).WithName("update-file-internal")

	// Get current user info
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		return fmt.Errorf("authentication required")
	}
	isAdmin := auth.IsAdmin(ctx)

	// Validate request
	if err := validateUpdateFileRequest(ctx, h.client, &req, h.namespace, isAdmin, claims.UserID); err != nil {
		return err
	}

	// Load existing ConfigMap by file ID
	configMap, err := h.loadFileConfigMapByID(ctx, fileID)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	// Verify file purpose matches expected type (only for workflow operations)
	// This prevents workflow endpoints from mutating non-workflow files
	if req.FilePurpose == "workflow-template" {
		existingPurpose := files.ExtractFilePurposeFromLabels(configMap.Labels)
		if existingPurpose != "workflow-template" {
			return fmt.Errorf("cannot update non-workflow file via workflow API")
		}
	}

	// Check ownership - only owner or admin can update workflows
	isOwner, err := h.isFileOwnerOrAdmin(ctx, configMap)
	if err != nil {
		return fmt.Errorf("failed to check file ownership: %w", err)
	}
	if !isOwner {
		return fmt.Errorf("only the workflow owner or an admin can update this workflow")
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
	configMap.Labels = files.BuildFileLabels(fileID, req.FileType, req.Groups, req.AvailableToAll, req.FilePurpose)
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
		return fmt.Errorf("failed to update file: %w", err)
	}

	logger.Info("Updated file", "fileID", fileID, "updatedBy", updatedBy)
	return nil
}

// deleteFileInternal extracts file deletion logic for reuse by workflow handlers
func (h *Handler) deleteFileInternal(ctx context.Context, fileID string) error {
	logger := log.FromContext(ctx).WithName("delete-file-internal")

	// Load ConfigMap by file ID (to verify it exists)
	configMap, err := h.loadFileConfigMapByID(ctx, fileID)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	// Check ownership - only owner or admin can delete workflows
	isOwner, err := h.isFileOwnerOrAdmin(ctx, configMap)
	if err != nil {
		return fmt.Errorf("failed to check file ownership: %w", err)
	}
	if !isOwner {
		return fmt.Errorf("only the workflow owner or an admin can delete this workflow")
	}

	// Delete ConfigMap
	if err := h.client.Delete(ctx, configMap); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	logger.Info("Deleted file", "fileID", fileID)
	return nil
}

// extractIDFromPath extracts resource ID from URL path
// Example: /api/v1/workflows/abc-123 -> "abc-123"
func extractIDFromPath(path, basePath string) string {
	// Remove base path
	path = strings.TrimPrefix(path, basePath)
	// Remove leading slash
	path = strings.TrimPrefix(path, "/")
	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")
	// Return first segment
	segments := strings.Split(path, "/")
	if len(segments) > 0 && segments[0] != "" && segments[0] != "available" {
		return segments[0]
	}
	return ""
}
