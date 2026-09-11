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
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPostBackupMethodNotAllowed(t *testing.T) {
	handler := createTestHandler()

	req := httptest.NewRequest("GET", BackupPath, nil)
	w := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.PostBackup(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405 Method Not Allowed, got %d", w.Code)
	}
}

func TestPostBackupAdminOnly(t *testing.T) {
	handler := createTestHandler()

	req := httptest.NewRequest("POST", BackupPath, nil)
	w := httptest.NewRecorder()
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

func TestPostBackupReturnsErrorOnEmptyCluster(t *testing.T) {
	handler := createTestHandler()

	req := httptest.NewRequest("POST", BackupPath, nil)
	w := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.PostBackup(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 Internal Server Error on empty cluster, got %d", w.Code)
	}
}

func TestPostRestoreMethodNotAllowed(t *testing.T) {
	handler := createTestHandler()

	req := httptest.NewRequest("GET", RestorePath, nil)
	w := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.PostRestore(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405 Method Not Allowed, got %d", w.Code)
	}
}

func TestPostRestoreAdminOnly(t *testing.T) {
	handler := createTestHandler()

	body, contentType := createMultipartBody(t, "backup", "test.tar.gz", []byte("fake"))
	req := httptest.NewRequest("POST", RestorePath, body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
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

func TestPostRestoreNoFile(t *testing.T) {
	handler := createTestHandler()

	req := httptest.NewRequest("POST", RestorePath, bytes.NewReader([]byte("not multipart")))
	w := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.PostRestore(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for non-multipart body, got %d", w.Code)
	}
}

func TestPostRestoreMissingFileField(t *testing.T) {
	handler := createTestHandler()

	body, contentType := createMultipartBody(t, "wrong_field", "test.tar.gz", []byte("fake"))
	req := httptest.NewRequest("POST", RestorePath, body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.PostRestore(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for wrong field name, got %d", w.Code)
	}

	var errResp ErrorResponse
	json.NewDecoder(w.Body).Decode(&errResp)
	if errResp.Message != "Missing required file field 'backup'" {
		t.Errorf("Expected missing field error, got: %s", errResp.Message)
	}
}

func TestPostRestoreInvalidExtension(t *testing.T) {
	handler := createTestHandler()

	body, contentType := createMultipartBody(t, "backup", "test.zip", []byte("fake"))
	req := httptest.NewRequest("POST", RestorePath, body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.PostRestore(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for non-.tar.gz file, got %d", w.Code)
	}

	var errResp ErrorResponse
	json.NewDecoder(w.Body).Decode(&errResp)
	if errResp.Message != "Backup file must be a .tar.gz archive" {
		t.Errorf("Expected extension validation error, got: %s", errResp.Message)
	}
}

func TestPostRestoreAcceptsUpload(t *testing.T) {
	handler := createTestHandler()

	body, contentType := createMultipartBody(t, "backup", "my-backup.tar.gz", []byte("fake archive content"))
	req := httptest.NewRequest("POST", RestorePath, body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin",
		Role:   "admin",
	}))

	handler.PostRestore(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected 202 Accepted, got %d", w.Code)
	}

	var resp RestoreResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got %s", resp.Status)
	}
	if resp.JobID == "" {
		t.Error("Expected non-empty jobID")
	}
	if resp.Message == "" {
		t.Error("Expected non-empty message")
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

func TestJobTrackerLifecycle(t *testing.T) {
	jt := NewJobTracker()

	_, ok := jt.Get("nonexistent")
	if ok {
		t.Error("Expected Get to return false for unknown job")
	}

	jt.Start("job-1", "restore")
	job, ok := jt.Get("job-1")
	if !ok {
		t.Fatal("Expected Get to find job-1")
	}
	if job.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got %s", job.Status)
	}
	if job.Type != "restore" {
		t.Errorf("Expected type 'restore', got %s", job.Type)
	}
	if job.CompletedAt != nil {
		t.Error("Expected nil CompletedAt for in-progress job")
	}

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
	jt.Start("job-3", "restore")

	job1, _ := jt.Get("job-3")
	job1.Status = "tampered"

	job2, _ := jt.Get("job-3")
	if job2.Status != "in_progress" {
		t.Errorf("Mutating returned value should not affect tracker, got status %s", job2.Status)
	}
}

func TestJobTrackerEviction(t *testing.T) {
	jt := NewJobTracker()

	for i := 0; i < maxTrackedJobs+5; i++ {
		id := fmt.Sprintf("job-%d", i)
		jt.Start(id, "restore")
		jt.Complete(id)
	}

	jt.Start("new-job", "restore")

	if len(jt.jobs) > maxTrackedJobs {
		t.Errorf("Expected at most %d jobs after eviction, got %d", maxTrackedJobs, len(jt.jobs))
	}

	_, ok := jt.Get("new-job")
	if !ok {
		t.Error("Newly added job should not be evicted")
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

func createMultipartBody(t *testing.T, fieldName, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, writer.FormDataContentType()
}

// createTestHandler creates a handler with a fake K8s client for testing.
func createTestHandler() *Handler {
	fakeClient := fake.NewClientBuilder().Build()
	return NewHandler(fakeClient, nil, "default", "", &auth.SecretManager{})
}
