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

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	"github.com/krkn-chaos/krkn-operator/pkg/filetypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreateFileType(t *testing.T) {
	handler := setupFilesTestHandler()
	reqBody := filetypes.CreateFileTypeRequest{Name: "configuration", Color: "#357edd"}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, FileTypesPath, bytes.NewReader(body))
	req = addAdminContext(req)
	response := httptest.NewRecorder()

	handler.CreateFileType(response, req)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}

	var created filetypes.FileTypeResponse
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if created.Name != reqBody.Name || created.Color != reqBody.Color {
		t.Fatalf("unexpected response: %+v", created)
	}

	var stored krknv1alpha1.KrknFileType
	if err := handler.client.Get(context.Background(), clientKey(handler.namespace, reqBody.Name), &stored); err != nil {
		t.Fatalf("expected file type to be stored: %v", err)
	}
	if stored.Spec.Name != reqBody.Name || stored.Spec.Color != reqBody.Color {
		t.Fatalf("unexpected stored file type: %+v", stored.Spec)
	}
}

func TestCreateFileTypeReturnsConflictForExistingType(t *testing.T) {
	handler := setupFilesTestHandler()
	existing := &krknv1alpha1.KrknFileType{
		ObjectMeta: metav1.ObjectMeta{Name: "configuration", Namespace: handler.namespace},
		Spec:       krknv1alpha1.KrknFileTypeSpec{Name: "configuration"},
	}
	if err := handler.client.Create(context.Background(), existing); err != nil {
		t.Fatalf("failed to create existing file type: %v", err)
	}

	body, _ := json.Marshal(filetypes.CreateFileTypeRequest{Name: "configuration"})
	req := addAdminContext(httptest.NewRequest(http.MethodPost, FileTypesPath, bytes.NewReader(body)))
	response := httptest.NewRecorder()

	handler.CreateFileType(response, req)

	if response.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, response.Code, response.Body.String())
	}
}

func TestCreateFileAutoCreatesFileType(t *testing.T) {
	handler := setupFilesTestHandler()
	adminUser := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-admin-test-example",
			Namespace: handler.namespace,
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID: "admin@test.example",
			Role:   "admin",
		},
	}
	if err := handler.client.Create(context.Background(), adminUser); err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}

	reqBody := files.CreateFileRequest{
		FileName:       "configuration.yaml",
		Content:        "key: value",
		FileType:       "configuration",
		AvailableToAll: true,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := addAdminContext(httptest.NewRequest(http.MethodPost, FilesPath, bytes.NewReader(body)))
	response := httptest.NewRecorder()

	handler.CreateFile(response, req)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}

	var stored krknv1alpha1.KrknFileType
	if err := handler.client.Get(context.Background(), clientKey(handler.namespace, reqBody.FileType), &stored); err != nil {
		t.Fatalf("expected file type to be auto-created: %v", err)
	}
}

// clientKey keeps the test focused on handler behavior without repeating the
// namespaced object-key construction at each assertion site.
func clientKey(namespace, name string) client.ObjectKey {
	return client.ObjectKey{Namespace: namespace, Name: name}
}
