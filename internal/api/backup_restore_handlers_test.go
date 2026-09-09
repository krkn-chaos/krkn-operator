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

	// Add admin user claims
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
		BackupPath: "/nonexistent/backup.tar.gz",
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

// Helper function to create a test handler
func createTestHandler() *Handler {
	fakeClient := fake.NewClientBuilder().Build()
	return NewHandler(fakeClient, nil, "default", "", &auth.SecretManager{})
}
