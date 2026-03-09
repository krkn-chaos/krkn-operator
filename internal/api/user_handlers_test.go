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
	"context"
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

// setupUserTestHandler creates a test handler with the given users and secrets
func setupUserTestHandler(objects ...runtime.Object) *Handler {
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(objects...).
		WithStatusSubresource(&krknv1alpha1.KrknUser{}).
		Build()
	fakeClientset := fake.NewSimpleClientset()
	return NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")
}

// createTestUser creates a test user with password secret
func createTestUser(userId, name, surname, role string, active bool) (*krknv1alpha1.KrknUser, *corev1.Secret) {
	userName := sanitizeUsername(userId)
	secretName := userName + "-password"

	// Hash password "TestPass123"
	passwordHash, _ := auth.HashPassword("TestPass123")

	user := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userName,
			Namespace: "default",
			Labels: map[string]string{
				UserAccountLabel: "true",
				AdminRoleLabel:   role,
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:            userId,
			Name:              name,
			Surname:           surname,
			Organization:      "Test Org",
			Role:              role,
			PasswordSecretRef: secretName,
		},
		Status: krknv1alpha1.KrknUserStatus{
			Active:  active,
			Created: metav1.Now(),
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "default",
		},
		Data: map[string][]byte{
			"passwordHash": []byte(passwordHash),
		},
	}

	return user, secret
}

// createAdminContext creates a context with admin claims
func createAdminContext() context.Context {
	claims := &auth.Claims{
		UserID:       "user1@test.local",
		Role:         "admin",
		Name:         "Admin",
		Surname:      "User",
		Organization: "Test Org",
	}
	return context.WithValue(context.Background(), auth.UserClaimsKey, claims)
}

// createUserContext creates a context with user claims
func createUserContext(userId string) context.Context {
	claims := &auth.Claims{
		UserID:       userId,
		Role:         "user",
		Name:         "Test",
		Surname:      "User",
		Organization: "Test Org",
	}
	return context.WithValue(context.Background(), auth.UserClaimsKey, claims)
}

// TestListUsers_AdminOnly_Success tests successful list operation
func TestListUsers_AdminOnly_Success(t *testing.T) {
	user1, secret1 := createTestUser("john.doe@test.local", "John", "Doe", "admin", true)
	user2, secret2 := createTestUser("jane.smith@test.local", "Jane", "Smith", "user", true)

	handler := setupUserTestHandler(user1, secret1, user2, secret2)

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.ListUsers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response ListUsersResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(response.Users))
	}

	if response.Total != 2 {
		t.Errorf("Expected total 2, got %d", response.Total)
	}
}

// TestListUsers_NotAdmin_Forbidden tests non-admin cannot list users
func TestListUsers_NotAdmin_Forbidden(t *testing.T) {
	user1, secret1 := createTestUser("user1@test.local", "Test", "User", "user", true)
	handler := setupUserTestHandler(user1, secret1)

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req = req.WithContext(createUserContext("user1@test.local"))
	w := httptest.NewRecorder()

	handler.ListUsers(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

// TestListUsers_WithFilters tests filtering by role
func TestListUsers_WithFilters(t *testing.T) {
	user1, secret1 := createTestUser("admin1@test.local", "John", "Doe", "admin", true)
	user2, secret2 := createTestUser("user1@test.local", "Jane", "Smith", "user", true)

	handler := setupUserTestHandler(user1, secret1, user2, secret2)

	req := httptest.NewRequest("GET", "/api/v1/users?role=admin", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.ListUsers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response ListUsersResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.Users) != 1 {
		t.Errorf("Expected 1 admin user, got %d", len(response.Users))
	}

	if response.Users[0].Role != "admin" {
		t.Errorf("Expected admin role, got %s", response.Users[0].Role)
	}
}

// TestListUsers_WithPagination tests pagination
func TestListUsers_WithPagination(t *testing.T) {
	user1, secret1 := createTestUser("user1@test.local", "User", "One", "user", true)
	user2, secret2 := createTestUser("user2@test.local", "User", "Two", "user", true)
	user3, secret3 := createTestUser("user3@test.local", "User", "Three", "user", true)

	handler := setupUserTestHandler(user1, secret1, user2, secret2, user3, secret3)

	req := httptest.NewRequest("GET", "/api/v1/users?page=1&limit=2", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.ListUsers(w, req)

	var response ListUsersResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if len(response.Users) != 2 {
		t.Errorf("Expected 2 users on page 1, got %d", len(response.Users))
	}

	if response.Total != 3 {
		t.Errorf("Expected total 3, got %d", response.Total)
	}
}

// TestGetUser_AdminCanViewAny tests admin can view any user
func TestGetUser_AdminCanViewAny(t *testing.T) {
	user1, secret1 := createTestUser("user1@test.local", "Test", "User", "user", true)
	handler := setupUserTestHandler(user1, secret1)

	req := httptest.NewRequest("GET", "/api/v1/users/user1@test.local", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.GetUser(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response UserResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.UserID != "user1@test.local" {
		t.Errorf("Expected userId test@test.local, got %s", response.UserID)
	}
}

// TestGetUser_UserCanViewSelf tests user can view their own profile
func TestGetUser_UserCanViewSelf(t *testing.T) {
	user1, secret1 := createTestUser("user1@test.local", "Test", "User", "user", true)
	handler := setupUserTestHandler(user1, secret1)

	req := httptest.NewRequest("GET", "/api/v1/users/user1@test.local", nil)
	req = req.WithContext(createUserContext("user1@test.local"))
	w := httptest.NewRecorder()

	handler.GetUser(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// TestGetUser_UserCannotViewOthers tests user cannot view other users
func TestGetUser_UserCannotViewOthers(t *testing.T) {
	user1, secret1 := createTestUser("user1@test.local", "Other", "User", "user", true)
	handler := setupUserTestHandler(user1, secret1)

	// User2 trying to view user1's profile (should be forbidden)
	req := httptest.NewRequest("GET", "/api/v1/users/user1@test.local", nil)
	req = req.WithContext(createUserContext("user2@test.local"))
	w := httptest.NewRecorder()

	handler.GetUser(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

// TestGetUser_NotFound tests getting non-existent user
func TestGetUser_NotFound(t *testing.T) {
	handler := setupUserTestHandler()

	req := httptest.NewRequest("GET", "/api/v1/users/user1@test.local", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.GetUser(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// TestCreateUser_AdminOnly_Success tests successful user creation
func TestCreateUser_AdminOnly_Success(t *testing.T) {
	handler := setupUserTestHandler()

	reqBody := `{
		"userId": "user1@test.local",
		"password": "NewPass123",
		"name": "New",
		"surname": "User",
		"role": "user"
	}`

	req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(reqBody))
	req = req.WithContext(createAdminContext())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateUser(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response CreateUserResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.UserID != "user1@test.local" {
		t.Errorf("Expected userId user1@test.local, got %s", response.UserID)
	}
}

// TestCreateUser_NotAdmin_Forbidden tests non-admin cannot create users
func TestCreateUser_NotAdmin_Forbidden(t *testing.T) {
	handler := setupUserTestHandler()

	reqBody := `{
		"userId": "user1@test.local",
		"password": "NewPass123",
		"name": "New",
		"surname": "User",
		"role": "user"
	}`

	req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(reqBody))
	req = req.WithContext(createUserContext("user1@test.local"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateUser(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

// TestCreateUser_Validation tests validation errors
func TestCreateUser_Validation(t *testing.T) {
	handler := setupUserTestHandler()

	tests := []struct {
		name     string
		body     string
		contains string
	}{
		{
			name:     "Missing userId",
			body:     `{"password":"Pass123","name":"Test","surname":"User","role":"user"}`,
			contains: "UserID",
		},
		{
			name:     "Missing password",
			body:     `{"userId":"user1@test.local","name":"Test","surname":"User","role":"user"}`,
			contains: "password",
		},
		{
			name:     "Invalid role",
			body:     `{"userId":"user1@test.local","password":"Pass123","name":"Test","surname":"User","role":"invalid"}`,
			contains: "Role must be either",
		},
		{
			name:     "Weak password",
			body:     `{"userId":"user1@test.local","password":"weak","name":"Test","surname":"User","role":"user"}`,
			contains: "Password validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(tt.body))
			req = req.WithContext(createAdminContext())
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.CreateUser(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
			}

			if !strings.Contains(w.Body.String(), tt.contains) {
				t.Errorf("Expected error to contain '%s', got: %s", tt.contains, w.Body.String())
			}
		})
	}
}

// TestCreateUser_DuplicateUser_Conflict tests duplicate user creation
func TestCreateUser_DuplicateUser_Conflict(t *testing.T) {
	user1, secret1 := createTestUser("user1@test.local", "Existing", "User", "user", true)
	handler := setupUserTestHandler(user1, secret1)

	reqBody := `{
		"userId": "user1@test.local",
		"password": "NewPass123",
		"name": "Duplicate",
		"surname": "User",
		"role": "user"
	}`

	req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(reqBody))
	req = req.WithContext(createAdminContext())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateUser(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
	}
}

// TestUpdateUser_AdminCanUpdateAll tests admin can update all fields
func TestUpdateUser_AdminCanUpdateAll(t *testing.T) {
	user1, secret1 := createTestUser("user1@test.local", "Test", "User", "user", true)
	handler := setupUserTestHandler(user1, secret1)

	newRole := "admin"
	reqBody := `{
		"name": "Updated",
		"role": "` + newRole + `"
	}`

	req := httptest.NewRequest("PATCH", "/api/v1/users/user1@test.local", strings.NewReader(reqBody))
	req = req.WithContext(createAdminContext())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateUser(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response UpdateUserResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.User.Name != "Updated" {
		t.Errorf("Expected name 'Updated', got %s", response.User.Name)
	}

	if response.User.Role != newRole {
		t.Errorf("Expected role '%s', got %s", newRole, response.User.Role)
	}
}

// TestUpdateUser_UserCanUpdateSelf tests user can update own profile
func TestUpdateUser_UserCanUpdateSelf(t *testing.T) {
	user1, secret1 := createTestUser("user1@test.local", "Test", "User", "user", true)
	handler := setupUserTestHandler(user1, secret1)

	reqBody := `{"name": "Updated Name"}`

	req := httptest.NewRequest("PATCH", "/api/v1/users/user1@test.local", strings.NewReader(reqBody))
	req = req.WithContext(createUserContext("user1@test.local"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateUser(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// TestUpdateUser_UserCannotChangeRole tests user cannot change their own role
func TestUpdateUser_UserCannotChangeRole(t *testing.T) {
	user1, secret1 := createTestUser("user1@test.local", "Test", "User", "user", true)
	handler := setupUserTestHandler(user1, secret1)

	reqBody := `{"role": "admin"}`

	req := httptest.NewRequest("PATCH", "/api/v1/users/user1@test.local", strings.NewReader(reqBody))
	req = req.WithContext(createUserContext("user1@test.local"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateUser(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Only admins can change role") {
		t.Errorf("Expected error about role change, got: %s", w.Body.String())
	}
}

// TestUpdateUser_Validation tests update validation
func TestUpdateUser_Validation(t *testing.T) {
	user1, secret1 := createTestUser("user1@test.local", "Test", "User", "user", true)
	handler := setupUserTestHandler(user1, secret1)

	// Test invalid role
	reqBody := `{"role": "invalid"}`

	req := httptest.NewRequest("PATCH", "/api/v1/users/user1@test.local", strings.NewReader(reqBody))
	req = req.WithContext(createAdminContext())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Role must be either") {
		t.Errorf("Expected role validation error, got: %s", w.Body.String())
	}
}

// TestUpdateUser_CannotDisableSelf tests user cannot disable own account
func TestUpdateUser_CannotDisableSelf(t *testing.T) {
	admin, adminSecret := createTestUser("selfadmin@domain20.test", "Admin", "User", "admin", true)
	handler := setupUserTestHandler(admin, adminSecret)

	// Try to disable own account
	reqBody := `{"active": false}`

	req := httptest.NewRequest("PATCH", "/api/v1/users/selfadmin@domain20.test", strings.NewReader(reqBody))
	// Create context with same email as user
	ctx := context.WithValue(context.Background(), auth.UserClaimsKey, &auth.Claims{
		UserID: "selfadmin@domain20.test",
		Role:   "admin",
	})
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateUser(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d: %s", http.StatusForbidden, w.Code, w.Body.String())
	}

	if !strings.Contains(w.Body.String(), "cannot disable your own account") {
		t.Errorf("Expected error about disabling self, got: %s", w.Body.String())
	}
}

// TestDeleteUser_Success tests successful user deletion
func TestDeleteUser_Success(t *testing.T) {
	admin, adminSecret := createTestUser("admin@test.local", "Admin", "User", "admin", true)
	user1, secret1 := createTestUser("user1@test.local", "Test", "User", "user", true)
	handler := setupUserTestHandler(admin, adminSecret, user1, secret1)

	req := httptest.NewRequest("DELETE", "/api/v1/users/user1@test.local", nil)
	// Create context with the actual admin email and admin role
	ctx := context.WithValue(context.Background(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin@test.local",
		Role:   "admin",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.DeleteUser(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// TestDeleteUser_CannotDeleteSelf tests admin cannot delete own account
func TestDeleteUser_CannotDeleteSelf(t *testing.T) {
	admin, adminSecret := createTestUser("user1@test.local", "Admin", "User", "admin", true)
	handler := setupUserTestHandler(admin, adminSecret)

	req := httptest.NewRequest("DELETE", "/api/v1/users/user1@test.local", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.DeleteUser(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	if !strings.Contains(w.Body.String(), "cannot delete your own account") {
		t.Errorf("Expected error about deleting self, got: %s", w.Body.String())
	}
}

// TestDeleteUser_CannotDeleteLastAdmin tests cannot delete last admin
func TestDeleteUser_CannotDeleteLastAdmin(t *testing.T) {
	// Create two admins, but admin2 is inactive - so admin1 is the last ACTIVE admin
	admin1, admin1Secret := createTestUser("admin1@test.local", "Admin", "One", "admin", true)
	admin2, admin2Secret := createTestUser("admin2@test.local", "Admin", "Two", "admin", false)
	handler := setupUserTestHandler(admin1, admin1Secret, admin2, admin2Secret)

	// Admin2 (inactive) tries to delete admin1 (the last active admin)
	req := httptest.NewRequest("DELETE", "/api/v1/users/admin1@test.local", nil)
	ctx := context.WithValue(context.Background(), auth.UserClaimsKey, &auth.Claims{
		UserID: "admin2@test.local",
		Role:   "admin",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handler.DeleteUser(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d: %s", http.StatusForbidden, w.Code, w.Body.String())
	}

	if !strings.Contains(w.Body.String(), "last active admin") {
		t.Errorf("Expected error about last admin, got: %s", w.Body.String())
	}
}

// TestDeleteUser_NotAdmin_Forbidden tests non-admin cannot delete users
func TestDeleteUser_NotAdmin_Forbidden(t *testing.T) {
	user1, secret1 := createTestUser("user1@test.local", "Test", "User", "user", true)
	handler := setupUserTestHandler(user1, secret1)

	req := httptest.NewRequest("DELETE", "/api/v1/users/user1@test.local", nil)
	req = req.WithContext(createUserContext("user1@test.local"))
	w := httptest.NewRecorder()

	handler.DeleteUser(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

// TestChangePassword_SelfChange_RequiresCurrentPassword tests user must provide current password
func TestChangePassword_SelfChange_RequiresCurrentPassword(t *testing.T) {
	user1, secret1 := createTestUser("user1@test.local", "Test", "User", "user", true)
	handler := setupUserTestHandler(user1, secret1)

	// Without current password
	reqBody := `{"newPassword": "NewPass456"}`
	req := httptest.NewRequest("PATCH", "/api/v1/users/user1@test.local/password", strings.NewReader(reqBody))
	req = req.WithContext(createUserContext("user1@test.local"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ChangePassword(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}

	if !strings.Contains(w.Body.String(), "Current password is required") {
		t.Errorf("Expected error about current password, got: %s", w.Body.String())
	}
}

// TestChangePassword_AdminChange_NoCurrentPassword tests admin can change password without current
func TestChangePassword_AdminChange_NoCurrentPassword(t *testing.T) {
	user1, secret1 := createTestUser("user1@test.local", "Test", "User", "user", true)
	handler := setupUserTestHandler(user1, secret1)

	reqBody := `{"newPassword": "NewPass456"}`
	req := httptest.NewRequest("PATCH", "/api/v1/users/user1@test.local/password", strings.NewReader(reqBody))
	req = req.WithContext(createAdminContext())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ChangePassword(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// TestChangePassword_Validation tests password validation
func TestChangePassword_Validation(t *testing.T) {
	user1, secret1 := createTestUser("user1@test.local", "Test", "User", "user", true)
	handler := setupUserTestHandler(user1, secret1)

	// Weak password
	reqBody := `{"currentPassword": "TestPass123", "newPassword": "weak"}`
	req := httptest.NewRequest("PATCH", "/api/v1/users/user1@test.local/password", strings.NewReader(reqBody))
	req = req.WithContext(createUserContext("user1@test.local"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ChangePassword(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Password validation failed") {
		t.Errorf("Expected password validation error, got: %s", w.Body.String())
	}
}

// TestChangePassword_SamePassword_Rejected tests same password is rejected
func TestChangePassword_SamePassword_Rejected(t *testing.T) {
	user1, secret1 := createTestUser("user1@test.local", "Test", "User", "user", true)
	handler := setupUserTestHandler(user1, secret1)

	// Use same password as current (TestPass123)
	reqBody := `{"currentPassword": "TestPass123", "newPassword": "TestPass123"}`
	req := httptest.NewRequest("PATCH", "/api/v1/users/user1@test.local/password", strings.NewReader(reqBody))
	req = req.WithContext(createUserContext("user1@test.local"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ChangePassword(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}

	if !strings.Contains(w.Body.String(), "must be different") {
		t.Errorf("Expected error about same password, got: %s", w.Body.String())
	}
}

// TestUsersRouter_MethodRouting tests router method handling
func TestUsersRouter_MethodRouting(t *testing.T) {
	handler := setupUserTestHandler()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{"List users - GET", "GET", "/api/v1/users", http.StatusOK},
		{"Create user - POST", "POST", "/api/v1/users", http.StatusBadRequest}, // Will fail validation but route works
		{"Invalid method on root", "PUT", "/api/v1/users", http.StatusMethodNotAllowed},
		{"Get user - GET", "GET", "/api/v1/users/test@test.local", http.StatusNotFound},          // User doesn't exist
		{"Update user - PATCH", "PATCH", "/api/v1/users/test@test.local", http.StatusBadRequest}, // Validates body before checking existence
		{"Delete user - DELETE", "DELETE", "/api/v1/users/test@test.local", http.StatusNotFound},
		{"Change password - PATCH", "PATCH", "/api/v1/users/test@test.local/password", http.StatusBadRequest}, // Validates body before checking existence
		{"Invalid method on user", "POST", "/api/v1/users/test@test.local", http.StatusMethodNotAllowed},
		{"Invalid method on password", "GET", "/api/v1/users/test@test.local/password", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == "POST" || tt.method == "PATCH" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader("{}"))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			req = req.WithContext(createAdminContext())
			w := httptest.NewRecorder()

			handler.UsersRouter(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d for %s %s: %s", tt.expectedStatus, w.Code, tt.method, tt.path, w.Body.String())
			}
		})
	}
}
