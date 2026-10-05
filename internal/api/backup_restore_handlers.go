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
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
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
	operatorprovider "github.com/krkn-chaos/krkn-operator/pkg/provider"
	"github.com/krkn-chaos/krknctl/pkg/backup"
	"k8s.io/client-go/dynamic"
)

const (
	maxTrackedJobs    = 1000
	maxActiveRestores = 1
	maxUploadSize     = 100 << 20 // 100MB (nginx default is 1MB, must configure larger if needed)
)

// backupHandler owns the dependencies and lifecycle state for backup and
// restore endpoints. Keeping these fields here prevents backup-specific
// Kubernetes access from expanding the shared API Handler.
type backupHandler struct {
	dynamicClient dynamic.Interface
	namespace     string
	jobTracker    *JobTracker
	restoreSlots  chan struct{}
	baseCtx       context.Context
	baseCancel    context.CancelFunc
}

func newBackupHandler(namespace string, dynamicClient dynamic.Interface) *backupHandler {
	baseCtx, baseCancel := context.WithCancel(context.Background())
	return &backupHandler{
		dynamicClient: dynamicClient,
		namespace:     namespace,
		jobTracker:    NewJobTracker(),
		restoreSlots:  make(chan struct{}, maxActiveRestores),
		baseCtx:       baseCtx,
		baseCancel:    baseCancel,
	}
}

func (h *backupHandler) Shutdown() {
	h.baseCancel()
}

// PostBackup keeps the existing API Handler method while delegating backup
// work to the backup-specific handler.
func (h *Handler) PostBackup(w http.ResponseWriter, r *http.Request) {
	h.backupHandler.PostBackup(w, r)
}

// PostRestore keeps the existing API Handler method while delegating restore
// work to the backup-specific handler.
func (h *Handler) PostRestore(w http.ResponseWriter, r *http.Request) {
	h.backupHandler.PostRestore(w, r)
}

// GetRestoreStatus keeps the existing API Handler method while delegating
// restore status queries to the backup-specific handler.
func (h *Handler) GetRestoreStatus(w http.ResponseWriter, r *http.Request) {
	h.backupHandler.GetRestoreStatus(w, r)
}

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
// @Description Create and download a backup archive of all operator configuration (admin-only). Pass the X-Backup-Passphrase header to encrypt the archive with AES-256-GCM.
// @Tags backup
// @Produce application/gzip
// @Param X-Backup-Passphrase header string false "Passphrase to encrypt the archive"
// @Success 200 {file} binary "Backup archive (tar.gz or tar.gz.enc)"
// @Failure 403 {object} ErrorResponse "Forbidden - admin access required"
// @Failure 405 {object} ErrorResponse "Method not allowed"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /backup [post]
func (h *backupHandler) PostBackup(w http.ResponseWriter, r *http.Request) {
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
	if h.dynamicClient == nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Backup Kubernetes client is not configured",
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
	defer func() {
		if cleanupErr := os.RemoveAll(tempDir); cleanupErr != nil {
			logger.Error(cleanupErr, "Failed to remove temporary backup directory", "path", tempDir)
		}
	}()

	passphrase := r.Header.Get("X-Backup-Passphrase")
	if passphrase != "" {
		decoded, err := base64.StdEncoding.DecodeString(passphrase)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_passphrase",
				Message: "X-Backup-Passphrase header must be Base64-encoded",
			})
			return
		}
		passphrase = string(decoded)
	}
	backupName := fmt.Sprintf("krkn-backup-%s", time.Now().Format("2006-01-02-150405"))

	config := backup.BackupConfig{
		Namespace:  h.namespace,
		OutputDir:  tempDir,
		BackupName: backupName,
		Passphrase: passphrase,
	}

	logger.Info("Creating backup for download", "backupName", backupName)

	archivePath, err := backup.CreateBackup(ctx, h.dynamicClient, config)
	if err != nil {
		logger.Error(err, "Backup failed")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Backup failed: " + err.Error(),
		})
		return
	}

	archiveFile, err := os.Open(archivePath) // #nosec G304 -- archivePath is returned by CreateBackup in the operator-created temporary directory
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

	if passphrase != "" {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.tar.gz.enc"`, backupName))
	} else {
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.tar.gz"`, backupName))
	}
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
// @Description Upload a backup archive and restore all operator configuration (admin-only). Restore runs asynchronously and rechecks restored target connectivity when complete. For encrypted archives (.tar.gz.enc), include the passphrase in the X-Backup-Passphrase header.
// @Tags backup
// @Accept multipart/form-data
// @Produce json
// @Param backup formData file true "Backup archive (tar.gz or tar.gz.enc)"
// @Param X-Backup-Passphrase header string false "Passphrase to decrypt an encrypted archive"
// @Success 202 {object} RestoreResponse "Restore started"
// @Failure 400 {object} ErrorResponse "Invalid request or missing file"
// @Failure 403 {object} ErrorResponse "Forbidden - admin access required"
// @Failure 405 {object} ErrorResponse "Method not allowed"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /restore [post]
func (h *backupHandler) PostRestore(w http.ResponseWriter, r *http.Request) {
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
	if h.dynamicClient == nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Restore Kubernetes client is not configured",
		})
		return
	}

	select {
	case h.restoreSlots <- struct{}{}:
		// Reserve the restore slot before parsing or storing the upload so that
		// concurrent requests cannot consume unbounded temporary disk space.
	default:
		writeJSONError(w, http.StatusTooManyRequests, ErrorResponse{
			Error:   "restore_in_progress",
			Message: "Another restore is already in progress. Try again later.",
		})
		return
	}

	slotTransferred := false
	defer func() {
		if !slotTransferred {
			<-h.restoreSlots
		}
	}()

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	cleanupMultipartForm := func() {
		if r.MultipartForm != nil {
			if cleanupErr := r.MultipartForm.RemoveAll(); cleanupErr != nil {
				logger.Error(cleanupErr, "Failed to remove multipart upload files")
			}
		}
	}
	defer cleanupMultipartForm()

	// The request body is capped by http.MaxBytesReader above.
	if err := r.ParseMultipartForm(32 << 20); err != nil { // #nosec G120 -- request body is limited to maxUploadSize
		status := http.StatusBadRequest
		errorCode := "bad_request"
		message := "Failed to parse upload: " + err.Error()
		if isRequestBodyTooLarge(err) {
			status = http.StatusRequestEntityTooLarge
			errorCode = "request_entity_too_large"
			message = "Request body too large"
		}
		writeJSONError(w, status, ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
		return
	}
	file, header, err := r.FormFile("backup")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Missing required file field 'backup'",
		})
		return
	}
	defer file.Close()

	lowerName := strings.ToLower(header.Filename)
	if !strings.HasSuffix(lowerName, ".tar.gz") && !strings.HasSuffix(lowerName, ".tar.gz.enc") {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Backup file must be a .tar.gz or .tar.gz.enc archive",
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
	tempPath := tempFile.Name()
	removeTempFile := func() {
		if removeErr := os.Remove(tempPath); removeErr != nil && !os.IsNotExist(removeErr) {
			logger.Error(removeErr, "Failed to remove temporary restore file", "path", tempPath)
		}
	}

	if _, err := io.Copy(tempFile, file); err != nil {
		if closeErr := tempFile.Close(); closeErr != nil {
			logger.Error(closeErr, "Failed to close temporary restore file", "path", tempFile.Name())
		}
		removeTempFile()
		status := http.StatusInternalServerError
		errorCode := "internal_error"
		message := "Failed to save uploaded file"
		if isRequestBodyTooLarge(err) {
			status = http.StatusRequestEntityTooLarge
			errorCode = "request_entity_too_large"
			message = "Request body too large"
		}
		writeJSONError(w, status, ErrorResponse{
			Error:   errorCode,
			Message: message,
		})
		return
	}
	if err := tempFile.Close(); err != nil {
		removeTempFile()

		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to finalize uploaded file",
		})
		return
	}

	passphrase := r.Header.Get("X-Backup-Passphrase")
	if passphrase != "" {
		decoded, err := base64.StdEncoding.DecodeString(passphrase)
		if err != nil {
			removeTempFile()
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_passphrase",
				Message: "X-Backup-Passphrase header must be Base64-encoded",
			})
			return
		}
		passphrase = string(decoded)
	}

	jobID := uuid.New().String()

	logger.Info("Restore requested", "jobID", jobID, "filename", header.Filename, "encrypted", passphrase != "")

	if err := h.jobTracker.Start(jobID, "restore"); err != nil {
		removeTempFile()
		writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{
			Error:   "service_unavailable",
			Message: err.Error(),
		})
		return
	}
	slotTransferred = true

	go func() {
		jobCtx := h.baseCtx
		jobLogger := log.FromContext(jobCtx)
		defer func() {
			<-h.restoreSlots
		}()
		defer func() {
			if removeErr := os.Remove(tempPath); removeErr != nil && !os.IsNotExist(removeErr) {
				jobLogger.Error(removeErr, "Failed to remove temporary restore file", "path", tempPath, "jobID", jobID)
			}
		}()
		jobLogger.Info("Starting restore", "jobID", jobID)

		restoreConfig := backup.RestoreConfig{
			Namespace:  h.namespace,
			BackupPath: tempPath,
			Passphrase: passphrase,
			LegacyProviderConfigHandler: func(ctx context.Context, configs []backup.LegacyProviderConfig) error {
				return operatorprovider.RestoreLegacyProviderConfigs(ctx, h.dynamicClient, h.namespace, configs)
			},
			RefreshTargetReadiness: true,
			WarningHandler: func(message string) {
				jobLogger.Info("Restore warning", "jobID", jobID, "message", message)
			},
		}

		if err := backup.RestoreBackup(jobCtx, h.dynamicClient, restoreConfig); err != nil {
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
		Message:   "Restore started. Check the restore status, then refresh the page if the restored settings are not visible.",
	}

	writeJSON(w, http.StatusAccepted, response)
}

func isRequestBodyTooLarge(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return errors.Is(err, multipart.ErrMessageTooLarge) || errors.As(err, &maxBytesErr)
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
func (h *backupHandler) GetRestoreStatus(w http.ResponseWriter, r *http.Request) {
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
