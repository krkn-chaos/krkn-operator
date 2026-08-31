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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/backup"
)

const maxTrackedJobs = 1000

var validBackupName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,238}$`)

// JobStatus tracks the status of an async backup or restore operation
type JobStatus struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	Error       string     `json:"error,omitempty"`
	StartedAt   time.Time  `json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	BackupName  string     `json:"backupName,omitempty"`
	BackupPath  string     `json:"backupPath,omitempty"`
}

// JobTracker provides thread-safe tracking of async job status
type JobTracker struct {
	mu   sync.RWMutex
	jobs map[string]*JobStatus
}

// NewJobTracker creates a new JobTracker.
func NewJobTracker() *JobTracker {
	return &JobTracker{jobs: make(map[string]*JobStatus)}
}

// Start registers a new in-progress job with the given ID, type, and optional metadata.
func (jt *JobTracker) Start(id, jobType string, opts ...func(*JobStatus)) {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	jt.evictOldestCompleted()
	job := &JobStatus{
		ID:        id,
		Type:      jobType,
		Status:    "in_progress",
		StartedAt: time.Now(),
	}
	for _, opt := range opts {
		opt(job)
	}
	jt.jobs[id] = job
}

// Complete marks a tracked job as completed.
func (jt *JobTracker) Complete(id string) {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	if job, ok := jt.jobs[id]; ok {
		now := time.Now()
		job.Status = "completed"
		job.CompletedAt = &now
	}
}

// Fail marks a tracked job as failed with the given error message.
func (jt *JobTracker) Fail(id, errMsg string) {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	if job, ok := jt.jobs[id]; ok {
		now := time.Now()
		job.Status = "failed"
		job.Error = errMsg
		job.CompletedAt = &now
	}
}

// Get returns a copy of the job status for the given ID.
func (jt *JobTracker) Get(id string) (JobStatus, bool) {
	jt.mu.RLock()
	defer jt.mu.RUnlock()
	job, ok := jt.jobs[id]
	if !ok {
		return JobStatus{}, false
	}
	return *job, true
}

// evictOldestCompleted removes the oldest completed/failed jobs when capacity is exceeded.
// Must be called with jt.mu held.
func (jt *JobTracker) evictOldestCompleted() {
	if len(jt.jobs) < maxTrackedJobs {
		return
	}
	var completed []*JobStatus
	for _, job := range jt.jobs {
		if job.CompletedAt != nil {
			completed = append(completed, job)
		}
	}
	sort.Slice(completed, func(i, j int) bool {
		return completed[i].CompletedAt.Before(*completed[j].CompletedAt)
	})
	toRemove := len(jt.jobs) - maxTrackedJobs + 1
	for i := 0; i < toRemove && i < len(completed); i++ {
		delete(jt.jobs, completed[i].ID)
	}
}

// BackupRequest represents a backup request
type BackupRequest struct {
	// Optional: custom backup name. If not provided, will use job ID.
	BackupName *string `json:"backupName,omitempty"`
}

// BackupResponse represents a successful backup response
type BackupResponse struct {
	JobID      string    `json:"jobId"`
	Status     string    `json:"status"` // "in_progress", "completed", or "failed"
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
	Status     string    `json:"status"` // "in_progress", "completed", or "failed"
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
// @Failure 409 {object} ErrorResponse "Backup name already exists"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /backup [post]
func (h *Handler) PostBackup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Backup is admin-only. Please contact your administrator.",
		})
		return
	}

	var req BackupRequest
	if r.Body != nil && r.ContentLength != 0 {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				// Empty body is fine for optional request
			} else {
				writeJSONError(w, http.StatusBadRequest, ErrorResponse{
					Error:   "bad_request",
					Message: "Invalid request body: " + err.Error(),
				})
				return
			}
		} else if decoder.More() {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Request body contains trailing data after JSON object",
			})
			return
		}
	}

	jobID := uuid.New().String()

	backupName := fmt.Sprintf("krkn-backup-%s", jobID[:8])
	if req.BackupName != nil && *req.BackupName != "" {
		if !validBackupName.MatchString(*req.BackupName) {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Invalid backup name: must start with alphanumeric and contain only alphanumeric, hyphens, underscores, or dots (max 239 chars)",
			})
			return
		}
		backupName = *req.BackupName
	}

	backupPath := filepath.Join("backups", backupName+".tar.gz")

	if _, err := os.Stat(backupPath); err == nil {
		writeJSONError(w, http.StatusConflict, ErrorResponse{
			Error:   "conflict",
			Message: fmt.Sprintf("A backup named %q already exists", backupName),
		})
		return
	}

	logger.Info("Backup requested", "jobID", jobID, "backupName", backupName)

	h.jobTracker.Start(jobID, "backup", func(j *JobStatus) {
		j.BackupName = backupName
		j.BackupPath = backupPath
	})

	go func() {
		jobCtx := h.baseCtx
		jobLogger := log.FromContext(jobCtx)
		jobLogger.Info("Starting backup", "jobID", jobID, "backupName", backupName)

		backupDir := filepath.Dir(backupPath)
		config := backup.BackupConfig{
			Namespace:  h.namespace,
			OutputDir:  backupDir,
			BackupName: backupName,
		}

		if _, err := backup.CreateBackup(jobCtx, h.client, config); err != nil {
			jobLogger.Error(err, "Backup failed", "jobID", jobID)
			h.jobTracker.Fail(jobID, err.Error())
			return
		}

		jobLogger.Info("Backup completed successfully", "jobID", jobID, "backupName", backupName)
		h.jobTracker.Complete(jobID)
	}()

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

	var req RestoreRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}
	if decoder.More() {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Request body contains trailing data after JSON object",
		})
		return
	}

	if req.BackupPath == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "backupPath is required",
		})
		return
	}

	absPath, err := filepath.Abs(req.BackupPath)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid backup path",
		})
		return
	}
	absBackupDir, err := filepath.Abs("backups")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to resolve backups directory",
		})
		return
	}
	if !strings.HasPrefix(absPath, absBackupDir+string(filepath.Separator)) {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Backup path must be within the backups directory",
		})
		return
	}

	fi, err := os.Stat(absPath)
	if err != nil {
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
	if !fi.Mode().IsRegular() {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Backup path must be a regular file",
		})
		return
	}

	jobID := uuid.New().String()

	logger.Info("Restore requested", "jobID", jobID, "backupPath", req.BackupPath)

	h.jobTracker.Start(jobID, "restore", func(j *JobStatus) {
		j.BackupPath = req.BackupPath
	})

	go func() {
		jobCtx := h.baseCtx
		jobLogger := log.FromContext(jobCtx)
		jobLogger.Info("Starting restore", "jobID", jobID, "backupPath", req.BackupPath)

		config := backup.RestoreConfig{
			Namespace:  h.namespace,
			BackupPath: req.BackupPath,
		}

		if err := backup.RestoreBackup(jobCtx, h.client, config); err != nil {
			jobLogger.Error(err, "Restore failed", "jobID", jobID)
			h.jobTracker.Fail(jobID, err.Error())
			return
		}

		jobLogger.Info("Restore completed successfully", "jobID", jobID, "backupPath", req.BackupPath)
		h.jobTracker.Complete(jobID)
	}()

	response := RestoreResponse{
		JobID:      jobID,
		Status:     "in_progress",
		BackupPath: req.BackupPath,
		StartedAt:  time.Now(),
		Message:    "Restore started. Check status using the job ID. A pod restart is required after restore completes to apply credential changes.",
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

	job, ok := h.jobTracker.Get(jobID)
	if !ok || job.Type != "backup" {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: fmt.Sprintf("No backup job found with ID: %s", jobID),
		})
		return
	}

	response := BackupResponse{
		JobID:      job.ID,
		Status:     job.Status,
		BackupName: job.BackupName,
		BackupPath: job.BackupPath,
		CreatedAt:  job.StartedAt,
		Message:    job.Error,
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

	job, ok := h.jobTracker.Get(jobID)
	if !ok || job.Type != "restore" {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: fmt.Sprintf("No restore job found with ID: %s", jobID),
		})
		return
	}

	response := RestoreResponse{
		JobID:      job.ID,
		Status:     job.Status,
		BackupPath: job.BackupPath,
		StartedAt:  job.StartedAt,
		Message:    job.Error,
	}

	writeJSON(w, http.StatusOK, response)
}

// BackupListItem represents a single backup in the list response
type BackupListItem struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	SizeBytes int64     `json:"sizeBytes"`
	CreatedAt time.Time `json:"createdAt"`
}

// ListBackups handles GET /api/v1/backups endpoint
// Returns available backup archives on disk (admin-only)
//
// @Summary List available backups
// @Description List all backup archives stored on the operator (admin-only)
// @Tags backup
// @Produce json
// @Success 200 {array} BackupListItem "List of backups"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden - admin access required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /backups [get]
func (h *Handler) ListBackups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Listing backups is admin-only.",
		})
		return
	}

	backupDir := "backups"
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"backups": []BackupListItem{},
			})
			return
		}
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to read backups directory",
		})
		return
	}

	var backups []BackupListItem
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) < 7 || name[len(name)-7:] != ".tar.gz" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			logger.Error(err, "Failed to stat backup file, skipping", "file", name)
			continue
		}

		backups = append(backups, BackupListItem{
			Name:      name[:len(name)-7],
			Path:      filepath.Join(backupDir, name),
			SizeBytes: info.Size(),
			CreatedAt: info.ModTime(),
		})
	}

	if backups == nil {
		backups = []BackupListItem{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"backups": backups,
	})
}
