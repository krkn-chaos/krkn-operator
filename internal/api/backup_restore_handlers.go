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
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/backup"
)

const (
	maxTrackedJobs = 1000
	maxUploadSize  = 100 << 20 // 100MB (nginx default is 1MB, must configure larger if needed)
)

// JobStatus tracks the status of an async restore operation.
type JobStatus struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	Error       string     `json:"error,omitempty"`
	StartedAt   time.Time  `json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// JobTracker provides thread-safe tracking of async job status.
type JobTracker struct {
	mu   sync.RWMutex
	jobs map[string]*JobStatus
}

// NewJobTracker creates a new JobTracker.
func NewJobTracker() *JobTracker {
	return &JobTracker{jobs: make(map[string]*JobStatus)}
}

// Start registers a new in-progress job with the given ID and type.
// Returns error if at capacity and no completed jobs can be evicted.
func (jt *JobTracker) Start(id, jobType string) error {
	jt.mu.Lock()
	defer jt.mu.Unlock()
	jt.evictOldestCompleted()
	// Check if we're still at capacity after eviction
	if len(jt.jobs) >= maxTrackedJobs {
		return fmt.Errorf("restore queue is at capacity (%d jobs); try again later", maxTrackedJobs)
	}
	jt.jobs[id] = &JobStatus{
		ID:        id,
		Type:      jobType,
		Status:    "in_progress",
		StartedAt: time.Now(),
	}
	return nil
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

// RestoreResponse represents a successful restore response.
type RestoreResponse struct {
	JobID     string    `json:"jobId"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"startedAt"`
	Message   string    `json:"message"`
}

// PostBackup handles POST /api/v1/backup endpoint.
// Creates and streams a backup archive for download (admin-only).
//
// @Summary Download operator configuration backup
// @Description Create and download a backup archive of all operator configuration (admin-only)
// @Tags backup
// @Produce application/gzip
// @Success 200 {file} binary "Backup archive (tar.gz)"
// @Failure 403 {object} ErrorResponse "Forbidden - admin access required"
// @Failure 405 {object} ErrorResponse "Method not allowed"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /backup [post]
func (h *Handler) PostBackup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only POST is allowed",
		})
		return
	}

	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Backup is admin-only. Please contact your administrator.",
		})
		return
	}

	tempDir, err := os.MkdirTemp("", "krkn-backup-download-*")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create temporary directory",
		})
		return
	}
	defer os.RemoveAll(tempDir)

	backupName := fmt.Sprintf("krkn-backup-%s", time.Now().Format("2006-01-02-150405"))

	config := backup.BackupConfig{
		Namespace:  h.namespace,
		OutputDir:  tempDir,
		BackupName: backupName,
	}

	logger.Info("Creating backup for download", "backupName", backupName)

	archivePath, err := backup.CreateBackup(ctx, h.client, config)
	if err != nil {
		logger.Error(err, "Backup failed")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Backup failed: " + err.Error(),
		})
		return
	}

	archiveFile, err := os.Open(archivePath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to read backup archive",
		})
		return
	}
	defer archiveFile.Close()

	fi, err := archiveFile.Stat()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to stat backup archive",
		})
		return
	}

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.tar.gz"`, backupName))
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	w.WriteHeader(http.StatusOK)

	written, err := io.Copy(w, archiveFile)
	if err != nil {
		logger.Error(err, "Download interrupted", "backupName", backupName, "bytesWritten", written, "totalSize", fi.Size())
		return
	}

	if written != fi.Size() {
		logger.Error(fmt.Errorf("incomplete transfer"), "Download incomplete", "backupName", backupName, "bytesWritten", written, "totalSize", fi.Size())
		return
	}

	logger.Info("Backup downloaded successfully", "backupName", backupName, "size", fi.Size())
}

// PostRestore handles POST /api/v1/restore endpoint.
// Accepts a backup archive upload and starts async restore (admin-only).
//
// @Summary Restore operator configuration from uploaded backup
// @Description Upload a backup archive and restore all operator configuration (admin-only). A pod restart is required after restore completes.
// @Tags backup
// @Accept multipart/form-data
// @Produce json
// @Param backup formData file true "Backup archive (tar.gz)"
// @Success 202 {object} RestoreResponse "Restore started"
// @Failure 400 {object} ErrorResponse "Invalid request or missing file"
// @Failure 403 {object} ErrorResponse "Forbidden - admin access required"
// @Failure 405 {object} ErrorResponse "Method not allowed"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /restore [post]
func (h *Handler) PostRestore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only POST is allowed",
		})
		return
	}

	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Restore is admin-only. Please contact your administrator.",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Failed to parse upload: " + err.Error(),
		})
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, header, err := r.FormFile("backup")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Missing required file field 'backup'",
		})
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".tar.gz") {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Backup file must be a .tar.gz archive",
		})
		return
	}

	tempFile, err := os.CreateTemp("", "krkn-restore-*.tar.gz")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create temporary file",
		})
		return
	}

	if _, err := io.Copy(tempFile, file); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to save uploaded file",
		})
		return
	}
	tempFile.Close()

	jobID := uuid.New().String()
	tempPath := tempFile.Name()

	logger.Info("Restore requested", "jobID", jobID, "filename", header.Filename)

	if err := h.jobTracker.Start(jobID, "restore"); err != nil {
		os.Remove(tempPath)
		writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{
			Error:   "service_unavailable",
			Message: err.Error(),
		})
		return
	}

	go func() {
		defer os.Remove(tempPath)
		jobCtx := h.baseCtx
		jobLogger := log.FromContext(jobCtx)
		jobLogger.Info("Starting restore", "jobID", jobID)

		restoreConfig := backup.RestoreConfig{
			Namespace:  h.namespace,
			BackupPath: tempPath,
		}

		if err := backup.RestoreBackup(jobCtx, h.client, restoreConfig); err != nil {
			jobLogger.Error(err, "Restore failed", "jobID", jobID)
			h.jobTracker.Fail(jobID, err.Error())
			return
		}

		jobLogger.Info("Restore completed successfully", "jobID", jobID)
		h.jobTracker.Complete(jobID)
	}()

	response := RestoreResponse{
		JobID:     jobID,
		Status:    "in_progress",
		StartedAt: time.Now(),
		Message:   "Restore started. Check status using the job ID. A pod restart is required after restore completes to apply credential changes.",
	}

	writeJSON(w, http.StatusAccepted, response)
}

// GetRestoreStatus handles GET /api/v1/restore/{jobID} endpoint.
// Returns the status of a restore job (admin-only).
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

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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
		JobID:     job.ID,
		Status:    job.Status,
		StartedAt: job.StartedAt,
		Message:   job.Error,
	}

	writeJSON(w, http.StatusOK, response)
}
