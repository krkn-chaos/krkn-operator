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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
)

// setupFilesTestHandler creates a test Handler with fake client
func setupFilesTestHandler() *Handler {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		Build()

	return &Handler{
		client:    fakeClient,
		namespace: "test-namespace",
	}
}

// addAdminContext adds admin claims to request context
func addAdminContext(req *http.Request) *http.Request {
	claims := &auth.Claims{
		UserID:  "admin@test.example",
		Role:    "admin",
		Name:    "Admin",
		Surname: "User",
	}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	return req.WithContext(ctx)
}

// addUserContext adds user claims to request context
func addUserContext(req *http.Request, userID string, groups ...string) *http.Request {
	claims := &auth.Claims{
		UserID:  userID,
		Role:    "user",
		Name:    "Test",
		Surname: "User",
	}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	return req.WithContext(ctx)
}

func TestCreateFile(t *testing.T) {
	handler := setupFilesTestHandler()

	tests := []struct {
		name         string
		request      files.CreateFileRequest
		setupGroups  []string
		userGroups   []string // Groups the test user belongs to
		userID       string
		expectStatus int
		expectInDB   bool
		isAdmin      bool
	}{
		{
			name: "create file successfully",
			request: files.CreateFileRequest{
				Name:           "test-config",
				FileName:       "app.conf",
				Content:        "{\"server\": \"localhost\", \"port\": 8080}",
				Description:    "Application configuration",
				Groups:         []string{},
				AvailableToAll: true,
			},
			setupGroups:  []string{},
			userGroups:   []string{},
			userID:       "admin@test.example",
			expectStatus: http.StatusCreated,
			expectInDB:   true,
			isAdmin:      true,
		},
		{
			name: "create file with groups",
			request: files.CreateFileRequest{
				Name:           "team-config",
				FileName:       "settings.yaml",
				Content:        "key: value",
				Description:    "Team settings",
				Groups:         []string{"dev-team"},
				AvailableToAll: false,
			},
			setupGroups:  []string{"dev-team"},
			expectStatus: http.StatusCreated,
			expectInDB:   true,
			isAdmin:      true,
		},
		{
			name: "user can create public file",
			request: files.CreateFileRequest{
				Name:           "user-config",
				FileName:       "config.yaml",
				Content:        "{\"key\": \"value\"}",
				AvailableToAll: true,
			},
			expectStatus: http.StatusCreated,
			expectInDB:   true,
			isAdmin:      false,
		},
		{
			name: "fail for missing name",
			request: files.CreateFileRequest{
				Name:           "",
				FileName:       "file.txt",
				Content:        "content",
				AvailableToAll: true,
			},
			expectStatus: http.StatusBadRequest,
			expectInDB:   false,
			isAdmin:      true,
		},
		{
			name: "fail for missing fileName",
			request: files.CreateFileRequest{
				Name:           "test-file",
				FileName:       "",
				Content:        "content",
				AvailableToAll: true,
			},
			expectStatus: http.StatusBadRequest,
			expectInDB:   false,
			isAdmin:      true,
		},
		{
			name: "fail for missing content",
			request: files.CreateFileRequest{
				Name:           "test-file",
				FileName:       "file.txt",
				Content:        "",
				AvailableToAll: true,
			},
			expectStatus: http.StatusBadRequest,
			expectInDB:   false,
			isAdmin:      true,
		},
		{
			name: "fail for non-existent group",
			request: files.CreateFileRequest{
				Name:           "test-file",
				FileName:       "file.txt",
				Content:        "content",
				Groups:         []string{"non-existent-group"},
				AvailableToAll: false,
			},
			setupGroups:  []string{},
			userGroups:   []string{},
			userID:       "admin@test.example",
			expectStatus: http.StatusBadRequest,
			expectInDB:   false,
			isAdmin:      true,
		},
		{
			name: "user can create file for own group",
			request: files.CreateFileRequest{
				Name:           "team-file",
				FileName:       "team.yaml",
				Content:        "{\"team\": \"data\"}",
				Description:    "Team file",
				Groups:         []string{"dev-team"},
				AvailableToAll: false,
			},
			setupGroups:  []string{"dev-team"},
			userGroups:   []string{"dev-team"},
			userID:       "user@test.example",
			expectStatus: http.StatusCreated,
			expectInDB:   true,
			isAdmin:      false,
		},
		{
			name: "user cannot create file for other group",
			request: files.CreateFileRequest{
				Name:           "other-file",
				FileName:       "other.yaml",
				Content:        "{\"data\": \"value\"}",
				Groups:         []string{"ops-team"},
				AvailableToAll: false,
			},
			setupGroups:  []string{"dev-team", "ops-team"},
			userGroups:   []string{"dev-team"},
			userID:       "user@test.example",
			expectStatus: http.StatusBadRequest, // Validation fails
			expectInDB:   false,
			isAdmin:      false,
		},
		{
			name: "user can create public file",
			request: files.CreateFileRequest{
				Name:           "public-file",
				FileName:       "public.json",
				Content:        "[1, 2, 3]",
				Description:    "Public file",
				AvailableToAll: true,
			},
			setupGroups:  []string{},
			userGroups:   []string{"dev-team"},
			userID:       "user@test.example",
			expectStatus: http.StatusCreated,
			expectInDB:   true,
			isAdmin:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup groups if needed
			for _, groupName := range tt.setupGroups {
				group := &krknv1alpha1.KrknUserGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      groupName,
						Namespace: handler.namespace,
					},
					Spec: krknv1alpha1.KrknUserGroupSpec{
						Name:        groupName,
						Description: "Test group",
					},
				}
				_ = handler.client.Create(context.Background(), group)
			}

			// Setup user (always create user for validation)
			if tt.userID != "" {
				// Create KrknUser with group labels
				userLabels := map[string]string{}
				for _, groupName := range tt.userGroups {
					labelKey := "group.krkn.krkn-chaos.dev/" + groupName
					userLabels[labelKey] = "true"
				}

				userName := "krknuser-" + sanitizeUserID(tt.userID)
				role := "user"
				if tt.isAdmin {
					role = "admin"
				}
				user := &krknv1alpha1.KrknUser{
					ObjectMeta: metav1.ObjectMeta{
						Name:      userName,
						Namespace: handler.namespace,
						Labels:    userLabels,
					},
					Spec: krknv1alpha1.KrknUserSpec{
						UserID:  tt.userID,
						Name:    "Test",
						Surname: "User",
						Role:    role,
					},
				}
				_ = handler.client.Create(context.Background(), user)
			}

			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, FilesPath, bytes.NewReader(body))
			userID := tt.userID
			if userID == "" {
				userID = "admin@test.example"
			}
			if tt.isAdmin {
				req = addAdminContext(req)
			} else {
				req = addUserContext(req, userID)
			}
			w := httptest.NewRecorder()

			handler.CreateFile(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectStatus, w.Code, w.Body.String())
			}

			if tt.expectInDB && w.Code == http.StatusCreated {
				var response files.CreateFileResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				// Verify ConfigMap was created
				var configMap corev1.ConfigMap
				err := handler.client.Get(context.Background(), client.ObjectKey{
					Name:      response.Name,
					Namespace: handler.namespace,
				}, &configMap)

				if err != nil {
					t.Errorf("Failed to get created ConfigMap: %v", err)
				}

				// Verify labels
				if configMap.Labels[files.AppComponentLabel] != files.ComponentFile {
					t.Errorf("Expected component label 'file', got '%s'", configMap.Labels[files.AppComponentLabel])
				}

				// Verify data
				if content, ok := configMap.Data[tt.request.FileName]; !ok {
					t.Errorf("Expected file '%s' in ConfigMap data", tt.request.FileName)
				} else if content != tt.request.Content {
					t.Errorf("Expected content '%s', got '%s'", tt.request.Content, content)
				}

			}
		})
	}
}

func TestListFiles(t *testing.T) {
	handler := setupFilesTestHandler()

	// Create test files
	files1 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "file1",
			Namespace: handler.namespace,
			Labels: map[string]string{
				files.AppNameLabel:      files.AppName,
				files.AppComponentLabel: files.ComponentFile,
			},
			Annotations: map[string]string{},
		},
		Data: map[string]string{
			"config.txt": "content1",
		},
	}
	files2 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "file2",
			Namespace: handler.namespace,
			Labels: map[string]string{
				files.AppNameLabel:      files.AppName,
				files.AppComponentLabel: files.ComponentFile,
			},
			Annotations: map[string]string{},
		},
		Data: map[string]string{
			"data.yaml": "content2",
		},
	}

	_ = handler.client.Create(context.Background(), files1)
	_ = handler.client.Create(context.Background(), files2)

	tests := []struct {
		name         string
		isAdmin      bool
		expectStatus int
		expectCount  int
	}{
		{
			name:         "admin lists all files",
			isAdmin:      true,
			expectStatus: http.StatusOK,
			expectCount:  2,
		},
		{
			name:         "non-admin forbidden",
			isAdmin:      false,
			expectStatus: http.StatusForbidden,
			expectCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, FilesPath, nil)
			if tt.isAdmin {
				req = addAdminContext(req)
			} else {
				req = addUserContext(req, "admin@test.example")
			}
			w := httptest.NewRecorder()

			handler.ListFiles(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, w.Code)
			}

			if tt.expectStatus == http.StatusOK {
				var response files.ListFilesResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if response.Total != tt.expectCount {
					t.Errorf("Expected %d files, got %d", tt.expectCount, response.Total)
				}
			}
		})
	}
}

func TestGetFile(t *testing.T) {
	handler := setupFilesTestHandler()

	// Create test file
	testFile := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-file",
			Namespace: handler.namespace,
			Labels: map[string]string{
				files.AppNameLabel:      files.AppName,
				files.AppComponentLabel: files.ComponentFile,
			},
			Annotations: map[string]string{
				files.DescriptionAnnotation: "Test file",
			},
		},
		Data: map[string]string{
			"config.yaml": "key: value",
		},
	}
	_ = handler.client.Create(context.Background(), testFile)

	tests := []struct {
		name         string
		fileName     string
		isAdmin      bool
		expectStatus int
	}{
		{
			name:         "get existing file",
			fileName:     "test-file",
			isAdmin:      true,
			expectStatus: http.StatusOK,
		},
		{
			name:         "get non-existent file",
			fileName:     "non-existent",
			isAdmin:      true,
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "non-admin forbidden",
			fileName:     "test-file",
			isAdmin:      false,
			expectStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, FilesPath+"/"+tt.fileName, nil)
			if tt.isAdmin {
				req = addAdminContext(req)
			} else {
				req = addUserContext(req, "admin@test.example")
			}
			w := httptest.NewRecorder()

			handler.GetFile(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, w.Code)
			}

			if tt.expectStatus == http.StatusOK {
				var response files.FileResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if response.Name != tt.fileName {
					t.Errorf("Expected file name '%s', got '%s'", tt.fileName, response.Name)
				}
			}
		})
	}
}

func TestUpdateFile(t *testing.T) {
	handler := setupFilesTestHandler()

	// Create test file
	testFile := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-file",
			Namespace: handler.namespace,
			Labels: map[string]string{
				files.AppNameLabel:      files.AppName,
				files.AppComponentLabel: files.ComponentFile,
			},
			Annotations: map[string]string{},
		},
		Data: map[string]string{
			"old.txt": "old content",
		},
	}
	_ = handler.client.Create(context.Background(), testFile)

	updateReq := files.UpdateFileRequest{
		FileName:       "new.yaml",
		Content:        "{\"updated\": true}",
		Description:    "Updated file",
		AvailableToAll: true,
	}

	tests := []struct {
		name         string
		fileName     string
		request      files.UpdateFileRequest
		setupFile    *corev1.ConfigMap
		userGroups   []string
		userID       string
		isAdmin      bool
		expectStatus int
	}{
		{
			name:         "update file successfully as admin",
			fileName:     "test-file",
			request:      updateReq,
			userGroups:   []string{},
			userID:       "admin@test.example",
			isAdmin:      true,
			expectStatus: http.StatusOK,
		},
		{
			name:         "update non-existent file",
			fileName:     "non-existent",
			request:      updateReq,
			userGroups:   []string{},
			userID:       "admin@test.example",
			isAdmin:      true,
			expectStatus: http.StatusNotFound,
		},
		{
			name:     "user can update file from own group",
			fileName: "team-file",
			request:  updateReq,
			setupFile: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "team-file",
					Namespace: handler.namespace,
					Labels: map[string]string{
						files.AppNameLabel:                   files.AppName,
						files.AppComponentLabel:              files.ComponentFile,
						"group.krkn.krkn-chaos.dev/dev-team": "true",
					},
				},
				Data: map[string]string{
					"old.txt": "old content",
				},
			},
			userGroups:   []string{"dev-team"},
			userID:       "user@test.example",
			isAdmin:      false,
			expectStatus: http.StatusOK,
		},
		{
			name:     "user cannot update file from other group",
			fileName: "ops-file",
			request:  updateReq,
			setupFile: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ops-file",
					Namespace: handler.namespace,
					Labels: map[string]string{
						files.AppNameLabel:                   files.AppName,
						files.AppComponentLabel:              files.ComponentFile,
						"group.krkn.krkn-chaos.dev/ops-team": "true",
					},
				},
				Data: map[string]string{
					"old.txt": "old content",
				},
			},
			userGroups:   []string{"dev-team"},
			userID:       "user@test.example",
			isAdmin:      false,
			expectStatus: http.StatusForbidden,
		},
		{
			name:     "user can update public file",
			fileName: "public-file",
			request:  updateReq,
			setupFile: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "public-file",
					Namespace: handler.namespace,
					Labels: map[string]string{
						files.AppNameLabel:        files.AppName,
						files.AppComponentLabel:   files.ComponentFile,
						files.AvailableToAllLabel: "true",
					},
				},
				Data: map[string]string{
					"old.txt": "old content",
				},
			},
			userGroups:   []string{"dev-team"},
			userID:       "user@test.example",
			isAdmin:      false,
			expectStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup custom file if specified
			if tt.setupFile != nil {
				_ = handler.client.Create(context.Background(), tt.setupFile)
			}

			// Setup user groups and user (always create user for validation)
			if tt.userID != "" {
				// Create groups
				for _, groupName := range tt.userGroups {
					group := &krknv1alpha1.KrknUserGroup{
						ObjectMeta: metav1.ObjectMeta{
							Name:      groupName,
							Namespace: handler.namespace,
						},
						Spec: krknv1alpha1.KrknUserGroupSpec{
							Name: groupName,
						},
					}
					_ = handler.client.Create(context.Background(), group)
				}

				// Create user with group labels
				userLabels := map[string]string{}
				for _, groupName := range tt.userGroups {
					labelKey := "group.krkn.krkn-chaos.dev/" + groupName
					userLabels[labelKey] = "true"
				}

				userName := "krknuser-" + sanitizeUserID(tt.userID)
				role := "user"
				if tt.isAdmin {
					role = "admin"
				}
				user := &krknv1alpha1.KrknUser{
					ObjectMeta: metav1.ObjectMeta{
						Name:      userName,
						Namespace: handler.namespace,
						Labels:    userLabels,
					},
					Spec: krknv1alpha1.KrknUserSpec{
						UserID:  tt.userID,
						Name:    "Test",
						Surname: "User",
						Role:    role,
					},
				}
				_ = handler.client.Create(context.Background(), user)
			}

			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPut, FilesPath+"/"+tt.fileName, bytes.NewReader(body))
			userID := tt.userID
			if userID == "" {
				userID = "admin@test.example"
			}
			if tt.isAdmin {
				req = addAdminContext(req)
			} else {
				req = addUserContext(req, userID)
			}
			w := httptest.NewRecorder()

			handler.UpdateFile(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectStatus, w.Code, w.Body.String())
			}

			if tt.expectStatus == http.StatusOK {
				// Verify ConfigMap was updated
				var configMap corev1.ConfigMap
				err := handler.client.Get(context.Background(), client.ObjectKey{
					Name:      tt.fileName,
					Namespace: handler.namespace,
				}, &configMap)

				if err != nil {
					t.Errorf("Failed to get updated ConfigMap: %v", err)
				}

				// Verify updated data
				if content, ok := configMap.Data[tt.request.FileName]; !ok {
					t.Errorf("Expected file '%s' in ConfigMap data", tt.request.FileName)
				} else if content != tt.request.Content {
					t.Errorf("Expected content '%s', got '%s'", tt.request.Content, content)
				}

			}
		})
	}
}

func TestDeleteFile(t *testing.T) {
	handler := setupFilesTestHandler()

	// Create test file for deletion
	testFile := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "delete-me",
			Namespace: handler.namespace,
			Labels: map[string]string{
				files.AppNameLabel:      files.AppName,
				files.AppComponentLabel: files.ComponentFile,
			},
		},
		Data: map[string]string{
			"file.txt": "content",
		},
	}
	_ = handler.client.Create(context.Background(), testFile)

	tests := []struct {
		name         string
		fileName     string
		setupFile    *corev1.ConfigMap
		userGroups   []string
		userID       string
		isAdmin      bool
		expectStatus int
	}{
		{
			name:         "delete file successfully as admin",
			fileName:     "delete-me",
			userGroups:   []string{},
			userID:       "admin@test.example",
			isAdmin:      true,
			expectStatus: http.StatusOK,
		},
		{
			name:         "delete non-existent file",
			fileName:     "non-existent",
			userGroups:   []string{},
			userID:       "admin@test.example",
			isAdmin:      true,
			expectStatus: http.StatusNotFound,
		},
		{
			name:     "user can delete file from own group",
			fileName: "team-file",
			setupFile: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "team-file",
					Namespace: handler.namespace,
					Labels: map[string]string{
						files.AppNameLabel:                   files.AppName,
						files.AppComponentLabel:              files.ComponentFile,
						"group.krkn.krkn-chaos.dev/dev-team": "true",
					},
				},
				Data: map[string]string{
					"file.txt": "content",
				},
			},
			userGroups:   []string{"dev-team"},
			userID:       "user@test.example",
			isAdmin:      false,
			expectStatus: http.StatusOK,
		},
		{
			name:     "user cannot delete file from other group",
			fileName: "ops-file",
			setupFile: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ops-file",
					Namespace: handler.namespace,
					Labels: map[string]string{
						files.AppNameLabel:                   files.AppName,
						files.AppComponentLabel:              files.ComponentFile,
						"group.krkn.krkn-chaos.dev/ops-team": "true",
					},
				},
				Data: map[string]string{
					"file.txt": "content",
				},
			},
			userGroups:   []string{"dev-team"},
			userID:       "user@test.example",
			isAdmin:      false,
			expectStatus: http.StatusForbidden,
		},
		{
			name:     "user can delete public file",
			fileName: "public-file",
			setupFile: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "public-file",
					Namespace: handler.namespace,
					Labels: map[string]string{
						files.AppNameLabel:        files.AppName,
						files.AppComponentLabel:   files.ComponentFile,
						files.AvailableToAllLabel: "true",
					},
				},
				Data: map[string]string{
					"file.txt": "content",
				},
			},
			userGroups:   []string{"dev-team"},
			userID:       "user@test.example",
			isAdmin:      false,
			expectStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup custom file if specified
			if tt.setupFile != nil {
				_ = handler.client.Create(context.Background(), tt.setupFile)
			}

			// Setup user groups and user (always create user for validation)
			if tt.userID != "" {
				// Create groups
				for _, groupName := range tt.userGroups {
					group := &krknv1alpha1.KrknUserGroup{
						ObjectMeta: metav1.ObjectMeta{
							Name:      groupName,
							Namespace: handler.namespace,
						},
						Spec: krknv1alpha1.KrknUserGroupSpec{
							Name: groupName,
						},
					}
					_ = handler.client.Create(context.Background(), group)
				}

				// Create user with group labels
				userLabels := map[string]string{}
				for _, groupName := range tt.userGroups {
					labelKey := "group.krkn.krkn-chaos.dev/" + groupName
					userLabels[labelKey] = "true"
				}

				userName := "krknuser-" + sanitizeUserID(tt.userID)
				role := "user"
				if tt.isAdmin {
					role = "admin"
				}
				user := &krknv1alpha1.KrknUser{
					ObjectMeta: metav1.ObjectMeta{
						Name:      userName,
						Namespace: handler.namespace,
						Labels:    userLabels,
					},
					Spec: krknv1alpha1.KrknUserSpec{
						UserID:  tt.userID,
						Name:    "Test",
						Surname: "User",
						Role:    role,
					},
				}
				_ = handler.client.Create(context.Background(), user)
			}

			req := httptest.NewRequest(http.MethodDelete, FilesPath+"/"+tt.fileName, nil)
			userID := tt.userID
			if userID == "" {
				userID = "admin@test.example"
			}
			if tt.isAdmin {
				req = addAdminContext(req)
			} else {
				req = addUserContext(req, userID)
			}
			w := httptest.NewRecorder()

			handler.DeleteFile(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, w.Code)
			}

			if tt.expectStatus == http.StatusOK && tt.fileName == "delete-me" {
				// Verify ConfigMap was deleted
				var configMap corev1.ConfigMap
				err := handler.client.Get(context.Background(), client.ObjectKey{
					Name:      tt.fileName,
					Namespace: handler.namespace,
				}, &configMap)

				if err == nil {
					t.Error("Expected ConfigMap to be deleted, but it still exists")
				}
			}
		})
	}
}

func TestListAvailableFiles(t *testing.T) {
	handler := setupFilesTestHandler()

	// Create test group
	group := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dev-team",
			Namespace: handler.namespace,
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name:        "dev-team",
			Description: "Development team",
		},
	}
	_ = handler.client.Create(context.Background(), group)

	// Create user in group
	// Name must match sanitized version: sanitizeResourceName("admin@test.example") -> "krknuser-admin-test-example"
	user := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-admin-test-example",
			Namespace: handler.namespace,
			Labels: map[string]string{
				"krkn.krkn-chaos.dev/user-account":   "true",
				"krkn.krkn-chaos.dev/role":           "user",
				"group.krkn.krkn-chaos.dev/dev-team": "true",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:  "admin@test.example",
			Name:    "Test",
			Surname: "User",
			Role:    "user",
		},
	}
	_ = handler.client.Create(context.Background(), user)

	// Create public file
	publicFile := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "public-file",
			Namespace: handler.namespace,
			Labels: map[string]string{
				files.AppNameLabel:        files.AppName,
				files.AppComponentLabel:   files.ComponentFile,
				files.AvailableToAllLabel: "true",
			},
		},
		Data: map[string]string{
			"public.txt": "public content",
		},
	}
	_ = handler.client.Create(context.Background(), publicFile)

	// Create group file
	groupFile := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "group-file",
			Namespace: handler.namespace,
			Labels: map[string]string{
				files.AppNameLabel:                   files.AppName,
				files.AppComponentLabel:              files.ComponentFile,
				"group.krkn.krkn-chaos.dev/dev-team": "true",
			},
		},
		Data: map[string]string{
			"group.yaml": "group content",
		},
	}
	_ = handler.client.Create(context.Background(), groupFile)

	// Create private file (no access)
	privateFile := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "private-file",
			Namespace: handler.namespace,
			Labels: map[string]string{
				files.AppNameLabel:                   files.AppName,
				files.AppComponentLabel:              files.ComponentFile,
				"group.krkn.krkn-chaos.dev/ops-team": "true",
			},
		},
		Data: map[string]string{
			"private.conf": "private content",
		},
	}
	_ = handler.client.Create(context.Background(), privateFile)

	tests := []struct {
		name         string
		isAdmin      bool
		userID       string
		expectCount  int
		expectStatus int
	}{
		{
			name:         "admin sees all files",
			isAdmin:      true,
			expectCount:  3,
			expectStatus: http.StatusOK,
		},
		{
			name:         "user sees public and group files",
			isAdmin:      false,
			userID:       "admin@test.example",
			expectCount:  2, // public + group
			expectStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, FilesAvailablePath, nil)
			if tt.isAdmin {
				req = addAdminContext(req)
			} else {
				req = addUserContext(req, tt.userID)
			}
			w := httptest.NewRecorder()

			handler.ListAvailableFiles(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, w.Code)
			}

			if tt.expectStatus == http.StatusOK {
				var response files.AvailableFilesResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if len(response.Files) != tt.expectCount {
					t.Errorf("Expected %d files, got %d", tt.expectCount, len(response.Files))
				}
			}
		})
	}
}

func TestCanAccessFile(t *testing.T) {
	handler := setupFilesTestHandler()

	// Create test group and user
	group := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dev-team",
			Namespace: handler.namespace,
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name:        "dev-team",
			Description: "Development team",
		},
	}
	_ = handler.client.Create(context.Background(), group)

	// Name must match sanitized version: sanitizeResourceName("admin@test.example") -> "krknuser-admin-test-example"
	user := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-admin-test-example",
			Namespace: handler.namespace,
			Labels: map[string]string{
				"krkn.krkn-chaos.dev/user-account":   "true",
				"krkn.krkn-chaos.dev/role":           "user",
				"group.krkn.krkn-chaos.dev/dev-team": "true",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:  "admin@test.example",
			Name:    "Test",
			Surname: "User",
			Role:    "user",
		},
	}
	_ = handler.client.Create(context.Background(), user)

	tests := []struct {
		name         string
		configMap    *corev1.ConfigMap
		userID       string
		isAdmin      bool
		expectAccess bool
	}{
		{
			name: "admin can access any file",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"group.krkn.krkn-chaos.dev/ops-team": "true",
					},
				},
			},
			isAdmin:      true,
			expectAccess: true,
		},
		{
			name: "user can access public file",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						files.AvailableToAllLabel: "true",
					},
				},
			},
			userID:       "admin@test.example",
			isAdmin:      false,
			expectAccess: true,
		},
		{
			name: "user can access group file",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"group.krkn.krkn-chaos.dev/dev-team": "true",
					},
				},
			},
			userID:       "admin@test.example",
			isAdmin:      false,
			expectAccess: true,
		},
		{
			name: "user cannot access other group file",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"group.krkn.krkn-chaos.dev/ops-team": "true",
					},
				},
			},
			userID:       "admin@test.example",
			isAdmin:      false,
			expectAccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.isAdmin {
				claims := &auth.Claims{
					UserID: "admin@test.example",
					Role:   "admin",
				}
				ctx = context.WithValue(ctx, auth.UserClaimsKey, claims)
			} else {
				claims := &auth.Claims{
					UserID: tt.userID,
					Role:   "user",
				}
				ctx = context.WithValue(ctx, auth.UserClaimsKey, claims)
			}

			canAccess := handler.canAccessFile(ctx, tt.configMap)
			if canAccess != tt.expectAccess {
				t.Errorf("Expected access %v, got %v", tt.expectAccess, canAccess)
			}
		})
	}
}
