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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	"github.com/krkn-chaos/krknctl/pkg/backupstore"
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
	backupStore   backupstore.Store // nil when S3 is disabled
	jobTracker    *JobTracker
	restoreSlots  chan struct{}
	baseCtx       context.Context
	baseCancel    context.CancelFunc
}

func newBackupHandler(namespace string, dynamicClient dynamic.Interface, store backupstore.Store) *backupHandler {
	baseCtx, baseCancel := context.WithCancel(context.Background())
	return &backupHandler{
		dynamicClient: dynamicClient,
		namespace:     namespace,
		backupStore:   store,
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

// PostRestoreFromURL keeps the existing API Handler method while delegating
// URL-based restore to the backup-specific handler.
func (h *Handler) PostRestoreFromURL(w http.ResponseWriter, r *http.Request) {
	h.backupHandler.PostRestoreFromURL(w, r)
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

// BackupDownloadResponse describes an externally stored backup archive.
// DownloadURL is a short-lived, presigned URL and must be treated as a
// bearer capability by clients.
type BackupDownloadResponse struct {
	DownloadURL string `json:"downloadUrl"`
	Filename    string `json:"filename"`
}

// PostBackup handles POST /api/v1/backup endpoint.
// Creates a backup archive for download (admin-only). When S3 storage is
// configured, the response contains a short-lived presigned download URL;
// otherwise the archive is streamed directly in the response.
//
// @Summary Download operator configuration backup
// @Description Create and download a backup archive of all operator configuration (admin-only). Pass the X-Backup-Passphrase header (Base64-encoded) to encrypt the archive with AES-256-GCM. When S3 storage is configured, the response is a JSON object with a short-lived presigned download URL; otherwise the archive is streamed directly.
// @Tags backup
// @Produce application/gzip
// @Produce application/json
// @Param X-Backup-Passphrase header string false "Base64-encoded passphrase to encrypt the archive"
// @Success 200 {object} BackupDownloadResponse "Presigned download URL (S3 mode) or streamed archive (direct mode)"
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

	passphrase, err := decodePassphraseHeader(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_passphrase",
			Message: err.Error(),
		})
		return
	}
	backupName := fmt.Sprintf("krkn-backup-%s-%s", time.Now().Format("20060102-150405.000000000"), uuid.New().String()[:8])

	config := backup.BackupConfig{
		Namespace:  h.namespace,
		OutputDir:  tempDir,
		BackupName: backupName,
		Passphrase: passphrase,
	}

	logger.Info("Creating backup for download", "backupName", backupName)

	result, err := backup.CreateBackupAndStore(ctx, h.dynamicClient, config, h.backupStore)
	if err != nil {
		logger.Error(err, "Backup failed")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Backup failed: " + err.Error(),
		})
		return
	}

	if result.Location != nil {
		logger.Info("Backup uploaded to S3", "backupName", backupName, "key", result.Location.Key)
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, BackupDownloadResponse{
			DownloadURL: result.Location.DownloadURL,
			Filename:    filepath.Base(result.ArchivePath),
		})
		return
	}

	archiveFile, err := os.Open(result.ArchivePath) // #nosec G304 -- path from CreateBackupAndStore in operator-created temporary directory
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
// @Description Upload a backup archive and restore all operator configuration (admin-only). Restore runs asynchronously and rechecks restored target connectivity when complete. For encrypted archives (.tar.gz.enc), include the Base64-encoded passphrase in the X-Backup-Passphrase header.
// @Tags backup
// @Accept multipart/form-data
// @Produce json
// @Param backup formData file true "Backup archive (tar.gz or tar.gz.enc)"
// @Param X-Backup-Passphrase header string false "Base64-encoded passphrase to decrypt an encrypted archive"
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

	passphrase, err := decodePassphraseHeader(r)
	if err != nil {
		removeTempFile()
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_passphrase",
			Message: err.Error(),
		})
		return
	}

	jobID := uuid.New().String()

	logger.Info("Restore requested", "jobID", jobID, "filename", header.Filename, "encrypted", passphrase != "")

	resp, err := h.startRestore(jobID, tempPath, passphrase)
	if err != nil {
		removeTempFile()
		writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{
			Error:   "service_unavailable",
			Message: err.Error(),
		})
		return
	}
	slotTransferred = true

	writeJSON(w, http.StatusAccepted, resp)
}

// decodePassphraseHeader reads and base64-decodes the X-Backup-Passphrase
// header. Returns the decoded passphrase (empty if header is absent) and
// any decoding error.
func decodePassphraseHeader(r *http.Request) (string, error) {
	raw := r.Header.Get("X-Backup-Passphrase")
	if raw == "" {
		return "", nil
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", fmt.Errorf("X-Backup-Passphrase header must be Base64-encoded")
	}
	return string(decoded), nil
}

func isRequestBodyTooLarge(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return errors.Is(err, multipart.ErrMessageTooLarge) || errors.As(err, &maxBytesErr)
}

// startRestore registers a restore job and launches the async restore
// goroutine. The caller must have already acquired a restore slot; ownership
// of the slot and tempPath cleanup are transferred to the goroutine on
// success. Returns the HTTP response payload or an error if the job tracker
// is at capacity.
func (h *backupHandler) startRestore(jobID, tempPath, passphrase string) (RestoreResponse, error) {
	if err := h.jobTracker.Start(jobID, "restore"); err != nil {
		return RestoreResponse{}, err
	}

	go func() {
		jobCtx := h.baseCtx
		jobLogger := log.FromContext(jobCtx)
		defer func() { <-h.restoreSlots }()
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

	return RestoreResponse{
		JobID:     jobID,
		Status:    "in_progress",
		StartedAt: time.Now(),
		Message:   "Restore started. Check the restore status, then refresh the page if the restored settings are not visible.",
	}, nil
}

// RestoreFromURLRequest is the JSON request body for POST /api/v1/restore/url.
type RestoreFromURLRequest struct {
	URL string `json:"url"`
}

// redactURL strips query parameters from a URL for safe logging.
// Presigned S3 URLs carry credentials in query params (X-Amz-Signature, etc.)
// that must not appear in logs.
func redactURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "<invalid-url>"
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

// PostRestoreFromURL handles POST /api/v1/restore/url endpoint.
// Accepts a remote backup URL or S3 URI and starts async download + restore (admin-only).
//
// @Summary Restore from a remote backup URL or S3 object
// @Description Accept a remote backup URL or S3 URI and start an async download and restore of operator configuration (admin-only). The download runs asynchronously as part of the restore job. S3 objects must use the configured backup bucket. Pass the X-Backup-Passphrase header (Base64-encoded) to decrypt encrypted archives.
// @Tags backup
// @Accept json
// @Produce json
// @Param body body RestoreFromURLRequest true "Remote backup URL or S3 object"
// @Param X-Backup-Passphrase header string false "Base64-encoded passphrase to decrypt an encrypted archive"
// @Success 202 {object} RestoreResponse "Restore started"
// @Failure 400 {object} ErrorResponse "Invalid request or URL"
// @Failure 403 {object} ErrorResponse "Forbidden - admin access required"
// @Failure 405 {object} ErrorResponse "Method not allowed"
// @Failure 429 {object} ErrorResponse "Another restore in progress"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /restore/url [post]
func (h *backupHandler) PostRestoreFromURL(w http.ResponseWriter, r *http.Request) {
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

	var reqBody RestoreFromURLRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid JSON body",
		})
		return
	}

	remoteSource := strings.TrimSpace(reqBody.URL)
	if remoteSource == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "A backup URL or S3 URI is required",
		})
		return
	}

	var (
		sourceDescription string
		downloadArchive   func(ctx context.Context) (string, func(), error)
	)
	if strings.HasPrefix(strings.ToLower(remoteSource), "s3://") {
		object, err := backupstore.ParseS3URI(remoteSource)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Invalid S3 backup URI",
			})
			return
		}
		configuredBucket := strings.TrimSpace(os.Getenv(backupstore.BucketEnv))
		if h.backupStore == nil || configuredBucket == "" {
			writeJSONError(w, http.StatusUnprocessableEntity, ErrorResponse{
				Error:   "s3_restore_unavailable",
				Message: "S3 restore is not configured",
			})
			return
		}
		if object.Bucket != configuredBucket {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "S3 backup URI must use the configured backup bucket",
			})
			return
		}
		reader, ok := h.backupStore.(backupstore.Reader)
		if !ok {
			writeJSONError(w, http.StatusUnprocessableEntity, ErrorResponse{
				Error:   "s3_restore_unavailable",
				Message: "Configured backup store does not support restore downloads",
			})
			return
		}
		sourceDescription = "s3://" + object.Bucket + "/" + object.Key
		downloadArchive = func(dlCtx context.Context) (string, func(), error) {
			return backup.DownloadArchiveFromStore(dlCtx, reader, object.Key, backup.DefaultMaxDownloadSize, backup.DefaultDownloadTimeout)
		}
	} else {
		parsed, err := url.Parse(remoteSource)
		if err != nil || parsed.Host == "" {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Invalid URL",
			})
			return
		}
		if parsed.Scheme != "https" {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Only HTTPS URLs or S3 URIs are allowed",
			})
			return
		}
		hostname := parsed.Hostname()
		if ip := net.ParseIP(hostname); ip != nil {
			if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
				writeJSONError(w, http.StatusBadRequest, ErrorResponse{
					Error:   "bad_request",
					Message: "Restore URLs must not target loopback, private, or link-local addresses",
				})
				return
			}
		}
		if hostname == "localhost" {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Restore URLs must not target localhost",
			})
			return
		}
		sourceDescription = redactURL(remoteSource)
		downloadArchive = func(dlCtx context.Context) (string, func(), error) {
			return backup.DownloadArchive(dlCtx, remoteSource, backup.DefaultMaxDownloadSize, backup.DefaultDownloadTimeout)
		}
	}

	passphrase, err := decodePassphraseHeader(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_passphrase",
			Message: err.Error(),
		})
		return
	}

	select {
	case h.restoreSlots <- struct{}{}:
	default:
		writeJSONError(w, http.StatusTooManyRequests, ErrorResponse{
			Error:   "restore_in_progress",
			Message: "Another restore is already in progress. Try again later.",
		})
		return
	}

	jobID := uuid.New().String()
	if err := h.jobTracker.Start(jobID, "restore"); err != nil {
		<-h.restoreSlots
		writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{
			Error:   "service_unavailable",
			Message: err.Error(),
		})
		return
	}

	logger.Info("Remote restore accepted", "jobID", jobID, "source", sourceDescription)

	go func() {
		jobCtx := h.baseCtx
		jobLogger := log.FromContext(jobCtx)
		defer func() { <-h.restoreSlots }()

		jobLogger.Info("Downloading backup archive", "jobID", jobID, "source", sourceDescription)

		tempPath, cleanup, dlErr := downloadArchive(jobCtx)
		if dlErr != nil {
			jobLogger.Error(dlErr, "Failed to download backup archive", "jobID", jobID, "source", sourceDescription)
			h.jobTracker.Fail(jobID, "Failed to download backup archive")
			return
		}
		defer func() {
			if cleanup != nil {
				cleanup()
			}
		}()

		jobLogger.Info("Download complete, starting restore", "jobID", jobID)

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

	writeJSON(w, http.StatusAccepted, RestoreResponse{
		JobID:     jobID,
		Status:    "in_progress",
		StartedAt: time.Now(),
		Message:   "Restore started. Download and restore run in the background. Check the restore status, then refresh the page if the restored settings are not visible.",
	})
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
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only GET is allowed",
		})
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
