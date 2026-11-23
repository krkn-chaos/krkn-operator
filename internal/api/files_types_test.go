package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
)

func createTypedFile(t *testing.T, handler *Handler, req files.CreateFileRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal create request: %v", err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, FilesPath, bytes.NewReader(body))
	httpReq = addAdminContext(httpReq)
	w := httptest.NewRecorder()
	handler.CreateFile(w, httpReq)
	return w
}

func updateTypedFile(t *testing.T, handler *Handler, fileID string, req files.UpdateFileRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal update request: %v", err)
	}
	httpReq := httptest.NewRequest(http.MethodPut, FilesPath+"/"+fileID, bytes.NewReader(body))
	httpReq = addAdminContext(httpReq)
	w := httptest.NewRecorder()
	handler.UpdateFile(w, httpReq)
	return w
}

func createdFileID(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var response files.CreateFileResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	return response.FileID
}

func typedFileConfigMap(t *testing.T, handler *Handler, fileID string) *corev1.ConfigMap {
	t.Helper()
	configMap := &corev1.ConfigMap{}
	key := client.ObjectKey{Name: "file-" + fileID, Namespace: handler.namespace}
	if err := handler.client.Get(context.Background(), key, configMap); err != nil {
		t.Fatalf("load file ConfigMap: %v", err)
	}
	return configMap
}

func TestCreateFile_StoresFileType(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestAdminUser(t, handler)

	fileID := createdFileID(t, createTypedFile(t, handler, files.CreateFileRequest{
		FileName:       "alerts.yaml",
		Content:        "key: value",
		AvailableToAll: true,
		FileType:       "alerts",
	}))

	configMap := typedFileConfigMap(t, handler, fileID)
	if got := files.ExtractFileTypeFromLabels(configMap.Labels); got != "alerts" {
		t.Fatalf("expected file type %q on the ConfigMap, got %q", "alerts", got)
	}
	if configMap.Labels["file-type.krkn.krkn-chaos.dev/alerts"] != "true" {
		t.Fatalf("expected the file-type label, got labels %v", configMap.Labels)
	}
}

func TestCreateFile_RejectsInvalidFileType(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestAdminUser(t, handler)

	w := createTypedFile(t, handler, files.CreateFileRequest{
		FileName:       "bad.yaml",
		Content:        "key: value",
		AvailableToAll: true,
		FileType:       "not a valid label",
	})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an invalid file type, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListAvailableFiles_ReturnsFileType(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestAdminUser(t, handler)

	fileID := createdFileID(t, createTypedFile(t, handler, files.CreateFileRequest{
		FileName:       "metrics.yaml",
		Content:        "key: value",
		AvailableToAll: true,
		FileType:       "metrics",
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, FilesAvailablePath, nil)
	req = addAdminContext(req)
	handler.ListAvailableFiles(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response files.AvailableFilesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	for _, file := range response.Files {
		if file.FileID == fileID {
			if file.FileType != "metrics" {
				t.Fatalf("expected file type %q, got %q", "metrics", file.FileType)
			}
			return
		}
	}
	t.Fatalf("file %q not found in listing", fileID)
}

func TestUpdateFile_ReplacesAndClearsFileType(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestAdminUser(t, handler)

	fileID := createdFileID(t, createTypedFile(t, handler, files.CreateFileRequest{
		FileName:       "config.yaml",
		Content:        "key: value",
		AvailableToAll: true,
		FileType:       "alerts",
	}))

	replacement := "metrics"
	w := updateTypedFile(t, handler, fileID, files.UpdateFileRequest{
		FileName:       "config.yaml",
		Content:        "key: value",
		AvailableToAll: true,
		FileType:       &replacement,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := files.ExtractFileTypeFromLabels(typedFileConfigMap(t, handler, fileID).Labels); got != "metrics" {
		t.Fatalf("expected file type to be replaced with %q, got %q", "metrics", got)
	}

	empty := ""
	w = updateTypedFile(t, handler, fileID, files.UpdateFileRequest{
		FileName:       "config.yaml",
		Content:        "key: value",
		AvailableToAll: true,
		FileType:       &empty,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := files.ExtractFileTypeFromLabels(typedFileConfigMap(t, handler, fileID).Labels); got != "" {
		t.Fatalf("expected the file type to be cleared, got %q", got)
	}
}

func TestUpdateFile_PreservesFileTypeWhenOmitted(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestAdminUser(t, handler)

	fileID := createdFileID(t, createTypedFile(t, handler, files.CreateFileRequest{
		FileName:       "config.yaml",
		Content:        "key: value",
		AvailableToAll: true,
		FileType:       "alerts",
	}))

	w := updateTypedFile(t, handler, fileID, files.UpdateFileRequest{
		FileName:       "config.yaml",
		Content:        "key: new",
		AvailableToAll: true,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := files.ExtractFileTypeFromLabels(typedFileConfigMap(t, handler, fileID).Labels); got != "alerts" {
		t.Fatalf("expected the file type to be preserved, got %q", got)
	}
}

func TestUpdateFileInternal_PreservesFileTypeOnWorkflowEdit(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestAdminUser(t, handler)

	fileID := "workflow-typed-0001"
	labels := files.BuildFileLabels(fileID, "workflow", nil, true, files.FilePurposeWorkflow, "typed-workflow")
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "file-" + fileID,
			Namespace: handler.namespace,
			Labels:    labels,
			Annotations: map[string]string{
				files.WorkflowNameAnnotation: "typed-workflow",
				files.CreatedByAnnotation:    "admin@test.example",
			},
		},
		Data: map[string]string{files.WorkflowFileName: `{"nodes":[],"edges":[]}`},
	}
	if err := handler.client.Create(context.Background(), configMap); err != nil {
		t.Fatalf("create workflow ConfigMap: %v", err)
	}

	workflowName := "typed-workflow"
	adminCtx := context.WithValue(context.Background(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin@test.example",
		Role:   "admin",
	})
	req := files.UpdateFileRequest{
		FileName:       files.WorkflowFileName,
		Content:        `{"nodes":[],"edges":[]}`,
		WorkflowName:   &workflowName,
		AvailableToAll: true,
		FilePurpose:    files.FilePurposeWorkflow,
	}
	if err := handler.updateFileInternal(adminCtx, fileID, req); err != nil {
		t.Fatalf("update workflow file: %v", err)
	}

	if got := files.ExtractFileTypeFromLabels(typedFileConfigMap(t, handler, fileID).Labels); got != "workflow" {
		t.Fatalf("expected the workflow edit to keep the file type %q, got %q", "workflow", got)
	}
}
