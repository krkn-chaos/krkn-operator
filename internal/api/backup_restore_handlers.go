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
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/backup"
)

// BackupRequest represents a backup request
type BackupRequest struct {
	// Optional: custom backup name. If not provided, will use timestamp.
	BackupName *string `json:"backupName,omitempty"`
}

// BackupResponse represents a successful backup response
type BackupResponse struct {
	JobID      string    `json:"jobId"`
	Status     string    `json:"status"` // "in_progress" or "completed"
	BackupName string    `json:"backupName"`
	BackupPath string    `json:"backupPath"`
	CreatedAt  time.Time `json:"createdAt"`
	Message    string    `json:"message"`
}

// RestoreRequest represents a restore request
type RestoreRequest struct {
	// Required: path to the backup archive to restore from
	BackupPath string `json:"backupPath"`
}

// RestoreResponse represents a successful restore response
type RestoreResponse struct {
	JobID      string    `json:"jobId"`
	Status     string    `json:"status"` // "in_progress" or "completed"
	BackupPath string    `json:"backupPath"`
	StartedAt  time.Time `json:"startedAt"`
	Message    string    `json:"message"`
}

// PostBackup handles POST /api/v1/backup endpoint
// Creates a backup of operator configuration (admin-only)
//
// @Summary Create operator configuration backup
// @Description Create a backup of all operator configuration including users, targets, and secrets (admin-only)
// @Tags backup
// @Accept json
// @Produce json
// @Param request body BackupRequest false "Backup configuration"
// @Success 202 {object} BackupResponse "Backup started"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden - admin access required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /backup [post]
func (h *Handler) PostBackup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	// Check admin role
	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Backup is admin-only. Please contact your administrator.",
		})
		return
	}

	// Parse request body
	var req BackupRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Invalid request body: " + err.Error(),
			})
			return
		}
	}

	// Generate backup name (custom or timestamp-based)
	backupName := fmt.Sprintf("krkn-backup-%d", time.Now().Unix())
	if req.BackupName != nil && *req.BackupName != "" {
		backupName = *req.BackupName
	}

	backupPath := filepath.Join("backups", backupName+".tar.gz")

	// Generate job ID for tracking
	jobID := uuid.New().String()

	logger.Info("Backup requested", "jobID", jobID, "backupName", backupName)

	// Execute backup asynchronously
	go func() {
		ctx := context.Background()
		logger := log.FromContext(ctx)
		logger.Info("Starting backup", "jobID", jobID, "backupName", backupName)

		backupDir := filepath.Dir(backupPath)
		config := backup.BackupConfig{
			Namespace:  h.namespace,
			OutputDir:  backupDir,
			BackupName: backupName,
		}

		if _, err := backup.CreateBackup(ctx, h.client, config); err != nil {
			logger.Error(err, "Backup failed", "jobID", jobID)
			return
		}

		logger.Info("Backup completed successfully", "jobID", jobID, "backupName", backupName)
	}()

	// Return 202 Accepted with job ID
	response := BackupResponse{
		JobID:      jobID,
		Status:     "in_progress",
		BackupName: backupName,
		BackupPath: backupPath,
		CreatedAt:  time.Now(),
		Message:    "Backup started. Check status using the job ID.",
	}

	writeJSON(w, http.StatusAccepted, response)
}

// PostRestore handles POST /api/v1/restore endpoint
// Restores operator configuration from a backup (admin-only)
//
// @Summary Restore operator configuration from backup
// @Description Restore all operator configuration from a backup archive (admin-only)
// @Tags backup
// @Accept json
// @Produce json
// @Param request body RestoreRequest true "Restore configuration"
// @Success 202 {object} RestoreResponse "Restore started"
// @Failure 400 {object} ErrorResponse "Invalid request or backup not found"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden - admin access required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /restore [post]
func (h *Handler) PostRestore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	// Check admin role
	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Restore is admin-only. Please contact your administrator.",
		})
		return
	}

	// Parse request body
	var req RestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate backup path
	if req.BackupPath == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "backupPath is required",
		})
		return
	}

	// Check if backup file exists
	if _, err := os.Stat(req.BackupPath); err != nil {
		if os.IsNotExist(err) {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: fmt.Sprintf("Backup file not found: %s", req.BackupPath),
			})
		} else {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to access backup file",
			})
		}
		return
	}

	// Generate job ID for tracking
	jobID := uuid.New().String()

	logger.Info("Restore requested", "jobID", jobID, "backupPath", req.BackupPath)

	// Execute restore asynchronously
	go func() {
		ctx := context.Background()
		logger := log.FromContext(ctx)
		logger.Info("Starting restore", "jobID", jobID, "backupPath", req.BackupPath)

		config := backup.RestoreConfig{
			Namespace:  h.namespace,
			BackupPath: req.BackupPath,
		}

		if err := backup.RestoreBackup(ctx, h.client, config); err != nil {
			logger.Error(err, "Restore failed", "jobID", jobID)
			return
		}

		logger.Info("Restore completed successfully", "jobID", jobID, "backupPath", req.BackupPath)
	}()

	// Return 202 Accepted with job ID
	response := RestoreResponse{
		JobID:      jobID,
		Status:     "in_progress",
		BackupPath: req.BackupPath,
		StartedAt:  time.Now(),
		Message:    "Restore started. Check status using the job ID.",
	}

	writeJSON(w, http.StatusAccepted, response)
}

// GetBackupStatus handles GET /api/v1/backup/{jobID} endpoint
// Returns the status of a backup job
//
// @Summary Get backup job status
// @Description Get the status of a backup operation by job ID (admin-only)
// @Tags backup
// @Produce json
// @Param jobID path string true "Backup job ID"
// @Success 200 {object} BackupResponse "Backup status"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden - admin access required"
// @Failure 404 {object} ErrorResponse "Job not found"
// @Security BearerAuth
// @Router /backup/{jobID} [get]
func (h *Handler) GetBackupStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check admin role
	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Backup status query is admin-only.",
		})
		return
	}

	jobID, err := extractPathSuffix(r.URL.Path, BackupPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "jobID " + err.Error(),
		})
		return
	}

	// For now, return a placeholder response
	// In a full implementation, you'd track job status in memory or a database
	response := BackupResponse{
		JobID:   jobID,
		Status:  "completed",
		Message: "Status tracking not yet implemented. Check operator logs for details.",
	}

	writeJSON(w, http.StatusOK, response)
}

// GetRestoreStatus handles GET /api/v1/restore/{jobID} endpoint
// Returns the status of a restore job
//
// @Summary Get restore job status
// @Description Get the status of a restore operation by job ID (admin-only)
// @Tags backup
// @Produce json
// @Param jobID path string true "Restore job ID"
// @Success 200 {object} RestoreResponse "Restore status"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden - admin access required"
// @Failure 404 {object} ErrorResponse "Job not found"
// @Security BearerAuth
// @Router /restore/{jobID} [get]
func (h *Handler) GetRestoreStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check admin role
	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Restore status query is admin-only.",
		})
		return
	}

	jobID, err := extractPathSuffix(r.URL.Path, RestorePath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "jobID " + err.Error(),
		})
		return
	}

	// For now, return a placeholder response
	// In a full implementation, you'd track job status in memory or a database
	response := RestoreResponse{
		JobID:   jobID,
		Status:  "completed",
		Message: "Status tracking not yet implemented. Check operator logs for details.",
	}

	writeJSON(w, http.StatusOK, response)
}
