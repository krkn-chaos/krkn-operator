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

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

func createTestFileCategory(t *testing.T, handler *Handler, name string, availableToAll bool) {
	t.Helper()
	objectMeta := metav1.ObjectMeta{Name: name, Namespace: handler.namespace}
	if availableToAll {
		objectMeta.Labels = map[string]string{categoryAvailableToAllLabel: "true"}
	}
	category := &krknv1alpha1.KrknCategory{ObjectMeta: objectMeta}
	if err := handler.client.Create(context.Background(), category); err != nil {
		t.Fatalf("create category %q: %v", name, err)
	}
}

func createTestUserAccount(t *testing.T, handler *Handler, userID, role string) {
	t.Helper()
	user := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mustSanitizeUserIDForResourceName(t, userID),
			Namespace: handler.namespace,
		},
		Spec: krknv1alpha1.KrknUserSpec{UserID: userID, Role: role},
	}
	if err := handler.client.Create(context.Background(), user); err != nil {
		t.Fatalf("create user %q: %v", userID, err)
	}
}

func createFileThroughAPI(t *testing.T, handler *Handler, req files.CreateFileRequest, admin bool) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal create request: %v", err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, FilesPath, bytes.NewReader(body))
	if admin {
		httpReq = addAdminContext(httpReq)
	} else {
		httpReq = addUserContext(httpReq, "user@test.example")
	}
	w := httptest.NewRecorder()
	handler.CreateFile(w, httpReq)
	return w
}

func updateFileThroughAPI(t *testing.T, handler *Handler, fileID string, req files.UpdateFileRequest, admin bool) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal update request: %v", err)
	}
	httpReq := httptest.NewRequest(http.MethodPut, FilesPath+"/"+fileID, bytes.NewReader(body))
	if admin {
		httpReq = addAdminContext(httpReq)
	} else {
		httpReq = addUserContext(httpReq, "user@test.example")
	}
	w := httptest.NewRecorder()
	handler.UpdateFile(w, httpReq)
	return w
}

func loadFileConfigMap(t *testing.T, handler *Handler, fileID string) *corev1.ConfigMap {
	t.Helper()
	configMap := &corev1.ConfigMap{}
	key := client.ObjectKey{Name: "file-" + fileID, Namespace: handler.namespace}
	if err := handler.client.Get(context.Background(), key, configMap); err != nil {
		t.Fatalf("load file ConfigMap %q: %v", key.Name, err)
	}
	return configMap
}

func fileCategoryLabels(t *testing.T, configMap *corev1.ConfigMap) []string {
	t.Helper()
	return files.ExtractCategoriesFromLabels(configMap.Labels)
}

func decodeCreatedFileID(t *testing.T, w *httptest.ResponseRecorder) string {
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

func TestCreateFile_StoresCategoriesAsLabels(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestAdminUser(t, handler)
	createTestFileCategory(t, handler, "network-chaos", true)
	createTestFileCategory(t, handler, "cluster-reliability", true)

	w := createFileThroughAPI(t, handler, files.CreateFileRequest{
		FileName:       "network-rules.yaml",
		Content:        "key: value",
		AvailableToAll: true,
		Categories:     []string{"network-chaos", "cluster-reliability"},
	}, true)
	fileID := decodeCreatedFileID(t, w)

	configMap := loadFileConfigMap(t, handler, fileID)
	got := fileCategoryLabels(t, configMap)
	want := []string{"cluster-reliability", "network-chaos"}
	if len(got) != len(want) {
		t.Fatalf("expected categories %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected categories %v, got %v", want, got)
		}
	}
	if configMap.Labels[files.CategoryLabelPrefix+"network-chaos"] != "true" {
		t.Errorf("expected network-chaos label on file ConfigMap, got labels %v", configMap.Labels)
	}
}

func TestListAvailableFiles_ReturnsCategories(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestAdminUser(t, handler)
	createTestFileCategory(t, handler, "alerts", true)

	fileID := decodeCreatedFileID(t, createFileThroughAPI(t, handler, files.CreateFileRequest{
		FileName:       "alert-rules.yaml",
		Content:        "groups: []",
		AvailableToAll: true,
		Categories:     []string{"alerts"},
	}, true))

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
		if file.FileID != fileID {
			continue
		}
		if len(file.Categories) != 1 || file.Categories[0] != "alerts" {
			t.Fatalf("expected categories [alerts], got %v", file.Categories)
		}
		return
	}
	t.Fatalf("file %q not found in listing", fileID)
}

func TestCreateFile_RejectsUnknownCategory(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestAdminUser(t, handler)

	w := createFileThroughAPI(t, handler, files.CreateFileRequest{
		FileName:       "orphan.yaml",
		Content:        "key: value",
		AvailableToAll: true,
		Categories:     []string{"does-not-exist"},
	}, true)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown category, got %d: %s", w.Code, w.Body.String())
	}

	configMaps := &corev1.ConfigMapList{}
	if err := handler.client.List(context.Background(), configMaps,
		client.MatchingLabels{files.AppComponentLabel: files.ComponentFile}); err != nil {
		t.Fatalf("list file ConfigMaps: %v", err)
	}
	if len(configMaps.Items) != 0 {
		t.Errorf("expected no file ConfigMap to be created, got %d", len(configMaps.Items))
	}

	reservations := &corev1.ConfigMapList{}
	if err := handler.client.List(context.Background(), reservations,
		client.MatchingLabels{files.AppComponentLabel: files.ComponentFileReservation}); err != nil {
		t.Fatalf("list reservation ConfigMaps: %v", err)
	}
	if len(reservations.Items) != 0 {
		t.Errorf("expected no dangling name reservation, got %d", len(reservations.Items))
	}
}

func TestCreateFile_RejectsCategoryInvisibleToUser(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestUserAccount(t, handler, "user@test.example", "user")

	// A category scoped to a group the caller does not belong to.
	privateCategory := &krknv1alpha1.KrknCategory{ObjectMeta: metav1.ObjectMeta{
		Name:      "private-category",
		Namespace: handler.namespace,
		Labels:    map[string]string{groupauth.GroupLabelKey("other-team"): "true"},
	}}
	if err := handler.client.Create(context.Background(), privateCategory); err != nil {
		t.Fatalf("create private category: %v", err)
	}

	w := createFileThroughAPI(t, handler, files.CreateFileRequest{
		FileName:       "private.yaml",
		Content:        "key: value",
		AvailableToAll: true,
		Categories:     []string{"private-category"},
	}, false)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for invisible category, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateFile_PreservesCategoriesWhenOmitted(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestAdminUser(t, handler)
	createTestFileCategory(t, handler, "network-chaos", true)

	fileID := decodeCreatedFileID(t, createFileThroughAPI(t, handler, files.CreateFileRequest{
		FileName:       "rules.yaml",
		Content:        "key: old",
		AvailableToAll: true,
		Categories:     []string{"network-chaos"},
	}, true))

	w := updateFileThroughAPI(t, handler, fileID, files.UpdateFileRequest{
		FileName:       "rules.yaml",
		Content:        "key: new",
		AvailableToAll: true,
	}, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	got := fileCategoryLabels(t, loadFileConfigMap(t, handler, fileID))
	if len(got) != 1 || got[0] != "network-chaos" {
		t.Fatalf("expected categories to be preserved, got %v", got)
	}
}

func TestUpdateFile_ReplacesCategories(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestAdminUser(t, handler)
	createTestFileCategory(t, handler, "network-chaos", true)
	createTestFileCategory(t, handler, "alerts", true)

	fileID := decodeCreatedFileID(t, createFileThroughAPI(t, handler, files.CreateFileRequest{
		FileName:       "rules.yaml",
		Content:        "key: old",
		AvailableToAll: true,
		Categories:     []string{"network-chaos"},
	}, true))

	replacement := []string{"alerts"}
	w := updateFileThroughAPI(t, handler, fileID, files.UpdateFileRequest{
		FileName:       "rules.yaml",
		Content:        "key: new",
		AvailableToAll: true,
		Categories:     &replacement,
	}, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	got := fileCategoryLabels(t, loadFileConfigMap(t, handler, fileID))
	if len(got) != 1 || got[0] != "alerts" {
		t.Fatalf("expected categories to be replaced with [alerts], got %v", got)
	}
}

func TestUpdateFile_ClearsCategoriesWithEmptyList(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestAdminUser(t, handler)
	createTestFileCategory(t, handler, "network-chaos", true)

	fileID := decodeCreatedFileID(t, createFileThroughAPI(t, handler, files.CreateFileRequest{
		FileName:       "rules.yaml",
		Content:        "key: old",
		AvailableToAll: true,
		Categories:     []string{"network-chaos"},
	}, true))

	empty := []string{}
	w := updateFileThroughAPI(t, handler, fileID, files.UpdateFileRequest{
		FileName:       "rules.yaml",
		Content:        "key: new",
		AvailableToAll: true,
		Categories:     &empty,
	}, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if got := fileCategoryLabels(t, loadFileConfigMap(t, handler, fileID)); len(got) != 0 {
		t.Fatalf("expected categories to be cleared, got %v", got)
	}
}

func TestRemoveCategoryAssociations_ClearsFileLabels(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestAdminUser(t, handler)
	createTestFileCategory(t, handler, "network-chaos", true)

	fileID := decodeCreatedFileID(t, createFileThroughAPI(t, handler, files.CreateFileRequest{
		FileName:       "rules.yaml",
		Content:        "key: value",
		AvailableToAll: true,
		Categories:     []string{"network-chaos"},
	}, true))

	if err := handler.removeCategoryAssociations(context.Background(), "network-chaos"); err != nil {
		t.Fatalf("remove category associations: %v", err)
	}

	if got := fileCategoryLabels(t, loadFileConfigMap(t, handler, fileID)); len(got) != 0 {
		t.Fatalf("expected file category labels to be removed, got %v", got)
	}
}

func createPrivateFileCategory(t *testing.T, handler *Handler, name string) {
	t.Helper()
	category := &krknv1alpha1.KrknCategory{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: handler.namespace,
		Labels:    map[string]string{groupauth.GroupLabelKey("other-team"): "true"},
	}}
	if err := handler.client.Create(context.Background(), category); err != nil {
		t.Fatalf("create private category %q: %v", name, err)
	}
}

func TestUpdateFile_PreservesCategoryInvisibleToEditor(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestUserAccount(t, handler, "user@test.example", "user")
	createPrivateFileCategory(t, handler, "private-category")

	fileID := decodeCreatedFileID(t, createFileThroughAPI(t, handler, files.CreateFileRequest{
		FileName:       "rules.yaml",
		Content:        "key: value",
		AvailableToAll: true,
	}, false))

	// An admin assigned a category the owner cannot see.
	configMap := loadFileConfigMap(t, handler, fileID)
	configMap.Labels[files.CategoryLabelPrefix+"private-category"] = "true"
	if err := handler.client.Update(context.Background(), configMap); err != nil {
		t.Fatalf("assign private category: %v", err)
	}

	// File responses only expose categories the caller can see, so the editor
	// never learns about the private association. Saving without the field (or
	// with only visible names) must not drop it.
	w := updateFileThroughAPI(t, handler, fileID, files.UpdateFileRequest{
		FileName:       "rules.yaml",
		Content:        "key: new",
		AvailableToAll: true,
	}, false)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	got := fileCategoryLabels(t, loadFileConfigMap(t, handler, fileID))
	if len(got) != 1 || got[0] != "private-category" {
		t.Fatalf("expected the invisible association to be preserved, got %v", got)
	}
}

func TestUpdateFile_RejectsNewCategoryInvisibleToEditor(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestUserAccount(t, handler, "user@test.example", "user")
	createPrivateFileCategory(t, handler, "private-category")

	fileID := decodeCreatedFileID(t, createFileThroughAPI(t, handler, files.CreateFileRequest{
		FileName:       "rules.yaml",
		Content:        "key: value",
		AvailableToAll: true,
	}, false))

	added := []string{"private-category"}
	w := updateFileThroughAPI(t, handler, fileID, files.UpdateFileRequest{
		FileName:       "rules.yaml",
		Content:        "key: new",
		AvailableToAll: true,
		Categories:     &added,
	}, false)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when adding an invisible category, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListAvailableFiles_HidesCategoriesInvisibleToCaller(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestUserAccount(t, handler, "user@test.example", "user")
	createPrivateFileCategory(t, handler, "private-category")
	createTestFileCategory(t, handler, "alerts", true)

	fileID := decodeCreatedFileID(t, createFileThroughAPI(t, handler, files.CreateFileRequest{
		FileName:       "shared.yaml",
		Content:        "key: value",
		AvailableToAll: true,
		Categories:     []string{"alerts"},
	}, false))

	// An admin also assigned a category the owner cannot see.
	configMap := loadFileConfigMap(t, handler, fileID)
	configMap.Labels[files.CategoryLabelPrefix+"private-category"] = "true"
	if err := handler.client.Update(context.Background(), configMap); err != nil {
		t.Fatalf("assign private category: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, FilesAvailablePath, nil)
	req = addUserContext(req, "user@test.example")
	handler.ListAvailableFiles(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response files.AvailableFilesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	for _, file := range response.Files {
		if file.FileID != fileID {
			continue
		}
		if len(file.Categories) != 1 || file.Categories[0] != "alerts" {
			t.Fatalf("expected only the visible category [alerts], got %v", file.Categories)
		}
		return
	}
	t.Fatalf("file %q not found in listing", fileID)
}

func TestGetFile_HidesCategoriesInvisibleToCaller(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestUserAccount(t, handler, "user@test.example", "user")
	createPrivateFileCategory(t, handler, "private-category")

	fileID := decodeCreatedFileID(t, createFileThroughAPI(t, handler, files.CreateFileRequest{
		FileName:       "shared.yaml",
		Content:        "key: value",
		AvailableToAll: true,
	}, false))

	configMap := loadFileConfigMap(t, handler, fileID)
	configMap.Labels[files.CategoryLabelPrefix+"private-category"] = "true"
	if err := handler.client.Update(context.Background(), configMap); err != nil {
		t.Fatalf("assign private category: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, FilesPath+"/"+fileID, nil)
	req = addUserContext(req, "user@test.example")
	handler.GetFile(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	categoriesRaw, present := payload["categories"]
	if !present {
		t.Fatalf("expected a categories field in the response, got %s", w.Body.String())
	}
	var categories []string
	if err := json.Unmarshal(categoriesRaw, &categories); err != nil {
		t.Fatalf("decode categories: %v", err)
	}
	if len(categories) != 0 {
		t.Fatalf("expected no visible categories, got %v", categories)
	}
}

func TestRemoveCategoryAssociations_LeavesUnrelatedConfigMapsAlone(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestAdminUser(t, handler)
	createTestFileCategory(t, handler, "network-chaos", true)

	fileID := decodeCreatedFileID(t, createFileThroughAPI(t, handler, files.CreateFileRequest{
		FileName:       "rules.yaml",
		Content:        "key: value",
		AvailableToAll: true,
		Categories:     []string{"network-chaos"},
	}, true))

	// A ConfigMap that does not belong to the files API but happens to carry the
	// category label must not be modified by category deletion.
	unrelated := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "operator-settings",
			Namespace: handler.namespace,
			Labels:    map[string]string{files.CategoryLabelPrefix + "network-chaos": "true"},
		},
	}
	if err := handler.client.Create(context.Background(), unrelated); err != nil {
		t.Fatalf("create unrelated ConfigMap: %v", err)
	}

	if err := handler.removeCategoryAssociations(context.Background(), "network-chaos"); err != nil {
		t.Fatalf("remove category associations: %v", err)
	}

	if got := fileCategoryLabels(t, loadFileConfigMap(t, handler, fileID)); len(got) != 0 {
		t.Fatalf("expected the file association to be removed, got %v", got)
	}

	loaded := &corev1.ConfigMap{}
	key := client.ObjectKey{Name: "operator-settings", Namespace: handler.namespace}
	if err := handler.client.Get(context.Background(), key, loaded); err != nil {
		t.Fatalf("reload unrelated ConfigMap: %v", err)
	}
	if loaded.Labels[files.CategoryLabelPrefix+"network-chaos"] != "true" {
		t.Fatalf("expected the unrelated ConfigMap label to be preserved, got %v", loaded.Labels)
	}
}

func TestUpdateFileInternal_PreservesCategoriesOnWorkflowEdit(t *testing.T) {
	handler := setupFilesTestHandler()
	createTestAdminUser(t, handler)
	createTestFileCategory(t, handler, "alerts", true)

	fileID := "workflow-file-0001"
	labels := files.BuildFileLabels(fileID, nil, true, files.FilePurposeWorkflow, "my-workflow")
	files.ApplyCategories(labels, []string{"alerts"})
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "file-" + fileID,
			Namespace: handler.namespace,
			Labels:    labels,
			Annotations: map[string]string{
				files.WorkflowNameAnnotation: "my-workflow",
				files.CreatedByAnnotation:    "admin@test.example",
			},
		},
		Data: map[string]string{files.WorkflowFileName: `{"nodes":[],"edges":[]}`},
	}
	if err := handler.client.Create(context.Background(), configMap); err != nil {
		t.Fatalf("create workflow ConfigMap: %v", err)
	}

	workflowName := "my-workflow"
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

	got := fileCategoryLabels(t, loadFileConfigMap(t, handler, fileID))
	if len(got) != 1 || got[0] != "alerts" {
		t.Fatalf("expected the workflow edit to keep [alerts], got %v", got)
	}
}
