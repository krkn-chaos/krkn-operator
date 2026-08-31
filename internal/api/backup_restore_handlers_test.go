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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPostBackupAdminOnly(t *testing.T) {
	handler := createTestHandler()

	// Test without admin role
	req := httptest.NewRequest("POST", BackupPath, bytes.NewReader([]byte("{}")))
	w := httptest.NewRecorder()

	// Add non-admin user claims
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "user1",
		Role:   "user",
	}))

	handler.PostBackup(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden, got %d", w.Code)
	}

	var errResp ErrorResponse
	json.NewDecoder(w.Body).Decode(&errResp)
	if errResp.Error != "forbidden" {
		t.Errorf("Expected 'forbidden' error, got %s", errResp.Error)
	}
}

func TestPostBackupSuccess(t *testing.T) {
	handler := createTestHandler()

	backupReq := BackupRequest{
		BackupName: strPtr("test-backup"),
	}
	body, _ := json.Marshal(backupReq)
	req := httptest.NewRequest("POST", BackupPath, bytes.NewReader(body))
	w := httptest.NewRecorder()

	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.PostBackup(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected 202 Accepted, got %d", w.Code)
	}

	var resp BackupResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got %s", resp.Status)
	}
	if resp.JobID == "" {
		t.Error("Expected non-empty jobID")
	}
	if resp.BackupName != "test-backup" {
		t.Errorf("Expected backup name 'test-backup', got %s", resp.BackupName)
	}
}

func TestPostBackupInvalidName(t *testing.T) {
	handler := createTestHandler()

	backupReq := BackupRequest{
		BackupName: strPtr("../escape-path"),
	}
	body, _ := json.Marshal(backupReq)
	req := httptest.NewRequest("POST", BackupPath, bytes.NewReader(body))
	w := httptest.NewRecorder()

	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.PostBackup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for path traversal name, got %d", w.Code)
	}
}

func TestGetBackupStatusUnknownJob(t *testing.T) {
	handler := createTestHandler()

	req := httptest.NewRequest("GET", BackupPath+"/nonexistent-id", nil)
	w := httptest.NewRecorder()

	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.GetBackupStatus(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 Not Found for unknown job, got %d", w.Code)
	}
}

func TestPostRestoreAdminOnly(t *testing.T) {
	handler := createTestHandler()

	restoreReq := RestoreRequest{
		BackupPath: "/tmp/test-backup.tar.gz",
	}
	body, _ := json.Marshal(restoreReq)
	req := httptest.NewRequest("POST", RestorePath, bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Add non-admin user claims
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "user1",
		Role:   "user",
	}))

	handler.PostRestore(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden, got %d", w.Code)
	}

	var errResp ErrorResponse
	json.NewDecoder(w.Body).Decode(&errResp)
	if errResp.Error != "forbidden" {
		t.Errorf("Expected 'forbidden' error, got %s", errResp.Error)
	}
}

func TestPostRestoreMissingBackupPath(t *testing.T) {
	handler := createTestHandler()

	restoreReq := RestoreRequest{
		BackupPath: "",
	}
	body, _ := json.Marshal(restoreReq)
	req := httptest.NewRequest("POST", RestorePath, bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Add admin user claims
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.PostRestore(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", w.Code)
	}

	var errResp ErrorResponse
	json.NewDecoder(w.Body).Decode(&errResp)
	if errResp.Error != "bad_request" {
		t.Errorf("Expected 'bad_request' error, got %s", errResp.Error)
	}
}

func TestPostRestoreBackupNotFound(t *testing.T) {
	handler := createTestHandler()

	restoreReq := RestoreRequest{
		BackupPath: "backups/nonexistent-backup.tar.gz",
	}
	body, _ := json.Marshal(restoreReq)
	req := httptest.NewRequest("POST", RestorePath, bytes.NewReader(body))
	w := httptest.NewRecorder()

	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.PostRestore(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", w.Code)
	}

	var errResp ErrorResponse
	json.NewDecoder(w.Body).Decode(&errResp)
	if errResp.Error != "bad_request" {
		t.Errorf("Expected 'bad_request' error, got %s", errResp.Error)
	}
}

func TestPostRestoreRejectsPathOutsideBackups(t *testing.T) {
	handler := createTestHandler()

	restoreReq := RestoreRequest{
		BackupPath: "/etc/passwd",
	}
	body, _ := json.Marshal(restoreReq)
	req := httptest.NewRequest("POST", RestorePath, bytes.NewReader(body))
	w := httptest.NewRecorder()

	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.PostRestore(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for path outside backups dir, got %d", w.Code)
	}

	var errResp ErrorResponse
	json.NewDecoder(w.Body).Decode(&errResp)
	if errResp.Message != "Backup path must be within the backups directory" {
		t.Errorf("Expected path validation error, got: %s", errResp.Message)
	}
}

func TestPostRestoreRejectsTraversalPath(t *testing.T) {
	handler := createTestHandler()

	restoreReq := RestoreRequest{
		BackupPath: "backups/../../../etc/passwd",
	}
	body, _ := json.Marshal(restoreReq)
	req := httptest.NewRequest("POST", RestorePath, bytes.NewReader(body))
	w := httptest.NewRecorder()

	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.PostRestore(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for traversal path, got %d", w.Code)
	}
}

func TestJobTrackerLifecycle(t *testing.T) {
	jt := NewJobTracker()

	// Unknown job returns false
	_, ok := jt.Get("nonexistent")
	if ok {
		t.Error("Expected Get to return false for unknown job")
	}

	// Start a job
	jt.Start("job-1", "backup")
	job, ok := jt.Get("job-1")
	if !ok {
		t.Fatal("Expected Get to find job-1")
	}
	if job.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got %s", job.Status)
	}
	if job.Type != "backup" {
		t.Errorf("Expected type 'backup', got %s", job.Type)
	}
	if job.CompletedAt != nil {
		t.Error("Expected nil CompletedAt for in-progress job")
	}

	// Complete the job
	jt.Complete("job-1")
	job, _ = jt.Get("job-1")
	if job.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", job.Status)
	}
	if job.CompletedAt == nil {
		t.Error("Expected non-nil CompletedAt for completed job")
	}
}

func TestJobTrackerFail(t *testing.T) {
	jt := NewJobTracker()

	jt.Start("job-2", "restore")
	jt.Fail("job-2", "disk full")

	job, ok := jt.Get("job-2")
	if !ok {
		t.Fatal("Expected Get to find job-2")
	}
	if job.Status != "failed" {
		t.Errorf("Expected status 'failed', got %s", job.Status)
	}
	if job.Error != "disk full" {
		t.Errorf("Expected error 'disk full', got %s", job.Error)
	}
	if job.CompletedAt == nil {
		t.Error("Expected non-nil CompletedAt for failed job")
	}
}

func TestJobTrackerGetReturnsCopy(t *testing.T) {
	jt := NewJobTracker()
	jt.Start("job-3", "backup")

	job1, _ := jt.Get("job-3")
	job1.Status = "tampered"

	job2, _ := jt.Get("job-3")
	if job2.Status != "in_progress" {
		t.Errorf("Mutating returned value should not affect tracker, got status %s", job2.Status)
	}
}

func TestBackupNameValidation(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"my-backup", true},
		{"backup_2024", true},
		{"Backup.v1", true},
		{"a", true},
		{"../escape", false},
		{"/absolute", false},
		{"-starts-with-dash", false},
		{".starts-with-dot", false},
		{"has spaces", false},
		{"has/slash", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := validBackupName.MatchString(tc.name)
			if got != tc.valid {
				t.Errorf("validBackupName.MatchString(%q) = %v, want %v", tc.name, got, tc.valid)
			}
		})
	}
}

func TestGetBackupStatusTrackedJob(t *testing.T) {
	handler := createTestHandler()

	// Start a backup to get a tracked job
	backupReq := BackupRequest{}
	body, _ := json.Marshal(backupReq)
	req := httptest.NewRequest("POST", BackupPath, bytes.NewReader(body))
	w := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))
	handler.PostBackup(w, req)

	var backupResp BackupResponse
	json.NewDecoder(w.Body).Decode(&backupResp)

	// Query status of the tracked job
	statusReq := httptest.NewRequest("GET", BackupPath+"/"+backupResp.JobID, nil)
	statusW := httptest.NewRecorder()
	statusReq = statusReq.WithContext(context.WithValue(statusReq.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))
	handler.GetBackupStatus(statusW, statusReq)

	if statusW.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for tracked job, got %d", statusW.Code)
	}

	var statusResp BackupResponse
	json.NewDecoder(statusW.Body).Decode(&statusResp)
	if statusResp.JobID != backupResp.JobID {
		t.Errorf("Expected jobID %s, got %s", backupResp.JobID, statusResp.JobID)
	}
	if statusResp.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got %s", statusResp.Status)
	}
}

func TestHandlerShutdownCancelsContext(t *testing.T) {
	handler := createTestHandler()

	ctx := handler.baseCtx
	select {
	case <-ctx.Done():
		t.Fatal("Context should not be cancelled before Shutdown")
	default:
	}

	handler.Shutdown()

	select {
	case <-ctx.Done():
		// expected
	default:
		t.Fatal("Context should be cancelled after Shutdown")
	}
}

func TestPostBackupDefaultNameUsesJobID(t *testing.T) {
	handler := createTestHandler()

	req := httptest.NewRequest("POST", BackupPath, bytes.NewReader([]byte("{}")))
	w := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.PostBackup(w, req)

	var resp BackupResponse
	json.NewDecoder(w.Body).Decode(&resp)

	expectedPrefix := "krkn-backup-" + resp.JobID[:8]
	if resp.BackupName != expectedPrefix {
		t.Errorf("Expected default backup name %q, got %q", expectedPrefix, resp.BackupName)
	}
}

func TestListBackupsAdminOnly(t *testing.T) {
	handler := createTestHandler()

	req := httptest.NewRequest("GET", BackupsPath, nil)
	w := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "user1",
		Role:   "user",
	}))

	handler.ListBackups(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden, got %d", w.Code)
	}
}

func TestListBackupsEmptyDir(t *testing.T) {
	handler := createTestHandler()

	req := httptest.NewRequest("GET", BackupsPath, nil)
	w := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.ListBackups(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}

	var resp struct {
		Backups []map[string]interface{} `json:"backups"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Backups) != 0 {
		t.Errorf("Expected empty list, got %d items", len(resp.Backups))
	}
}

func TestGetRestoreStatusUnknownJob(t *testing.T) {
	handler := createTestHandler()

	req := httptest.NewRequest("GET", RestorePath+"/nonexistent-id", nil)
	w := httptest.NewRecorder()

	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.GetRestoreStatus(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 Not Found for unknown restore job, got %d", w.Code)
	}
}

func TestGetRestoreStatusTrackedJob(t *testing.T) {
	handler := createTestHandler()

	handler.jobTracker.Start("restore-job-1", "restore")

	req := httptest.NewRequest("GET", RestorePath+"/restore-job-1", nil)
	w := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.GetRestoreStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for tracked restore job, got %d", w.Code)
	}

	var resp RestoreResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.JobID != "restore-job-1" {
		t.Errorf("Expected jobID 'restore-job-1', got %s", resp.JobID)
	}
	if resp.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got %s", resp.Status)
	}
}

func TestGetRestoreStatusAdminOnly(t *testing.T) {
	handler := createTestHandler()

	req := httptest.NewRequest("GET", RestorePath+"/some-id", nil)
	w := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "user1",
		Role:   "user",
	}))

	handler.GetRestoreStatus(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden for non-admin, got %d", w.Code)
	}
}

func TestBackupStatusRejectsRestoreJob(t *testing.T) {
	handler := createTestHandler()

	handler.jobTracker.Start("job-cross", "restore")

	req := httptest.NewRequest("GET", BackupPath+"/job-cross", nil)
	w := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.GetBackupStatus(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 when querying backup status for a restore job, got %d", w.Code)
	}
}

func TestRestoreStatusRejectsBackupJob(t *testing.T) {
	handler := createTestHandler()

	handler.jobTracker.Start("job-cross-2", "backup")

	req := httptest.NewRequest("GET", RestorePath+"/job-cross-2", nil)
	w := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.GetRestoreStatus(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 when querying restore status for a backup job, got %d", w.Code)
	}
}

func TestPostBackupTrailingJSON(t *testing.T) {
	handler := createTestHandler()

	body := []byte(`{"backupName": "test"}{"extra": true}`)
	req := httptest.NewRequest("POST", BackupPath, bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	w := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.PostBackup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for trailing JSON, got %d", w.Code)
	}
}

func TestPostRestoreTrailingJSON(t *testing.T) {
	handler := createTestHandler()

	body := []byte(`{"backupPath": "backups/test.tar.gz"}{"extra": true}`)
	req := httptest.NewRequest("POST", RestorePath, bytes.NewReader(body))
	w := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.PostRestore(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for trailing JSON, got %d", w.Code)
	}
}

func TestJobTrackerEviction(t *testing.T) {
	jt := NewJobTracker()

	for i := 0; i < maxTrackedJobs+5; i++ {
		id := fmt.Sprintf("job-%d", i)
		jt.Start(id, "backup")
		jt.Complete(id)
	}

	jt.Start("new-job", "backup")

	if len(jt.jobs) > maxTrackedJobs {
		t.Errorf("Expected at most %d jobs after eviction, got %d", maxTrackedJobs, len(jt.jobs))
	}

	_, ok := jt.Get("new-job")
	if !ok {
		t.Error("Newly added job should not be evicted")
	}
}

// Helper function to create a test handler
func createTestHandler() *Handler {
	fakeClient := fake.NewClientBuilder().Build()
	return NewHandler(fakeClient, nil, "default", "", &auth.SecretManager{})
}
