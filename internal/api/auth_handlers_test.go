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

Assisted-by: Claude Sonnet 4.5 (claude-sonnet-4-5@20250929)
*/

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

func setupAuthTestHandler(users ...*krknv1alpha1.KrknUser) *Handler {
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	objects := []runtime.Object{}
	for _, user := range users {
		objects = append(objects, user)
	}

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()
	fakeClientset := fake.NewSimpleClientset()
	return NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")
}

func TestIsRegistered_NoAdmins(t *testing.T) {
	handler := setupAuthTestHandler()

	req := httptest.NewRequest("GET", "/api/v1/auth/is-registered", nil)
	w := httptest.NewRecorder()
	handler.IsRegistered(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response IsRegisteredResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Registered {
		t.Error("Expected Registered=false when no admins exist")
	}
}

func TestIsRegistered_WithAdmin(t *testing.T) {
	adminUser := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-user",
			Namespace: "default",
			Labels: map[string]string{
				AdminRoleLabel:   "admin",
				UserAccountLabel: "true",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:            "[email protected]",
			Name:              "Admin",
			Surname:           "User",
			Role:              "admin",
			PasswordSecretRef: "admin-password",
		},
	}

	handler := setupAuthTestHandler(adminUser)

	req := httptest.NewRequest("GET", "/api/v1/auth/is-registered", nil)
	w := httptest.NewRecorder()
	handler.IsRegistered(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response IsRegisteredResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Registered {
		t.Error("Expected Registered=true when admin exists")
	}
}

func TestIsRegistered_MethodNotAllowed(t *testing.T) {
	handler := setupAuthTestHandler()

	methods := []string{"POST", "PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		req := httptest.NewRequest(method, "/api/v1/auth/is-registered", nil)
		w := httptest.NewRecorder()
		handler.IsRegistered(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Method %s: Expected status code %d, got %d", method, http.StatusMethodNotAllowed, w.Code)
		}
	}
}

func TestRegister_FirstAdmin_Success(t *testing.T) {
	handler := setupAuthTestHandler()

	reqBody := `{
		"userId": "[email protected]",
		"password": "SecurePassword123",
		"name": "First",
		"surname": "Admin",
		"organization": "Example Corp",
		"role": "admin"
	}`

	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response RegisterResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.UserID != "[email protected]" {
		t.Errorf("Expected userId '[email protected]', got '%s'", response.UserID)
	}

	if response.Role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", response.Role)
	}
}

func TestRegister_FirstUser_MustBeAdmin(t *testing.T) {
	handler := setupAuthTestHandler()

	reqBody := `{
		"userId": "[email protected]",
		"password": "SecurePassword123",
		"name": "First",
		"surname": "User",
		"role": "user"
	}`

	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !strings.Contains(response.Message, "First user must have admin role") {
		t.Errorf("Expected error about first user, got: %s", response.Message)
	}
}

func TestRegister_Validation(t *testing.T) {
	handler := setupAuthTestHandler()

	tests := []struct {
		name        string
		reqBody     string
		wantStatus  int
		wantMessage string
	}{
		{
			name: "missing userId",
			reqBody: `{
				"password": "SecurePassword123",
				"name": "Test",
				"surname": "User",
				"role": "admin"
			}`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "UserID, password, name, surname, and role are required",
		},
		{
			name: "missing password",
			reqBody: `{
				"userId": "[email protected]",
				"name": "Test",
				"surname": "User",
				"role": "admin"
			}`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "UserID, password, name, surname, and role are required",
		},
		{
			name: "invalid role",
			reqBody: `{
				"userId": "[email protected]",
				"password": "SecurePassword123",
				"name": "Test",
				"surname": "User",
				"role": "superadmin"
			}`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "Role must be either 'user' or 'admin'",
		},
		{
			name: "password too short",
			reqBody: `{
				"userId": "[email protected]",
				"password": "short",
				"name": "Test",
				"surname": "User",
				"role": "admin"
			}`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "Password validation failed",
		},
		{
			name:        "invalid JSON",
			reqBody:     `{"invalid json`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "Invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler.Register(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status code %d, got %d", tt.wantStatus, w.Code)
			}

			var response ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if !strings.Contains(response.Message, tt.wantMessage) {
				t.Errorf("Expected message to contain '%s', got '%s'", tt.wantMessage, response.Message)
			}
		})
	}
}

func TestLogin_Success(t *testing.T) {
	// Create password hash
	passwordHash, err := auth.HashPassword("TestPassword123")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Setup handler with user and secret
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	user := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-user",
			Namespace: "default",
			Labels: map[string]string{
				UserAccountLabel: "true",
				AdminRoleLabel:   "user",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:            "[email protected]",
			Name:              "Test",
			Surname:           "User",
			Organization:      "Test Org",
			Role:              "user",
			PasswordSecretRef: "test-password-secret",
		},
		Status: krknv1alpha1.KrknUserStatus{
			Active:  true,
			Created: metav1.Now(),
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-password-secret",
			Namespace: "default",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"passwordHash": []byte(passwordHash),
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(user, secret).
		Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	reqBody := `{
		"userId": "[email protected]",
		"password": "TestPassword123"
	}`

	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Token == "" {
		t.Error("Expected token to be set")
	}

	if response.UserID != "[email protected]" {
		t.Errorf("Expected userId '[email protected]', got '%s'", response.UserID)
	}

	if response.Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", response.Role)
	}

	if response.ExpiresAt == "" {
		t.Error("Expected expiresAt to be set")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	passwordHash, _ := auth.HashPassword("CorrectPassword123")

	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	user := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-user",
			Namespace: "default",
			Labels: map[string]string{
				UserAccountLabel: "true",
				AdminRoleLabel:   "user",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:            "[email protected]",
			Name:              "Test",
			Surname:           "User",
			Role:              "user",
			PasswordSecretRef: "test-password-secret",
		},
		Status: krknv1alpha1.KrknUserStatus{
			Active: true,
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-password-secret",
			Namespace: "default",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"passwordHash": []byte(passwordHash),
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(user, secret).
		Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	tests := []struct {
		name       string
		reqBody    string
		wantStatus int
	}{
		{
			name: "wrong password",
			reqBody: `{
				"userId": "[email protected]",
				"password": "WrongPassword"
			}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "user not found",
			reqBody: `{
				"userId": "[email protected]",
				"password": "SomePassword123"
			}`,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler.Login(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status code %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestLogin_InactiveUser(t *testing.T) {
	passwordHash, _ := auth.HashPassword("TestPassword123")

	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	user := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-user",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:            "[email protected]",
			Name:              "Test",
			Surname:           "User",
			Role:              "user",
			PasswordSecretRef: "test-password-secret",
		},
		Status: krknv1alpha1.KrknUserStatus{
			Active: false, // Inactive user
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-password-secret",
			Namespace: "default",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"passwordHash": []byte(passwordHash),
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(user, secret).
		Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	reqBody := `{
		"userId": "[email protected]",
		"password": "TestPassword123"
	}`

	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !strings.Contains(response.Message, "disabled") {
		t.Errorf("Expected error about disabled account, got: %s", response.Message)
	}
}
