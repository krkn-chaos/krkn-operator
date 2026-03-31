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
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

// fetchUserByEmail retrieves a KrknUser by email address (UserID).
// Returns the user and any error encountered.
func (h *Handler) fetchUserByEmail(ctx context.Context, email string) (*krknv1alpha1.KrknUser, error) {
	var users krknv1alpha1.KrknUserList
	if err := h.client.List(ctx, &users, client.InNamespace(h.namespace)); err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	for i := range users.Items {
		if users.Items[i].Spec.UserID == email {
			return &users.Items[i], nil
		}
	}

	return nil, fmt.Errorf("user with email '%s' not found", email)
}

// buildUserResponse constructs a UserResponse from a KrknUser CR.
// Does not include password hash.
func buildUserResponse(user *krknv1alpha1.KrknUser) UserResponse {
	var created, lastLogin *time.Time
	if !user.Status.Created.IsZero() {
		t := user.Status.Created.Time
		created = &t
	}
	if !user.Status.LastLogin.IsZero() {
		t := user.Status.LastLogin.Time
		lastLogin = &t
	}

	return UserResponse{
		UserID:       user.Spec.UserID,
		Name:         user.Spec.Name,
		Surname:      user.Spec.Surname,
		Organization: user.Spec.Organization,
		Role:         user.Spec.Role,
		Active:       user.Status.Active,
		Created:      created,
		LastLogin:    lastLogin,
	}
}

// filterUsers filters users by role, active status, and search term
func filterUsers(users []krknv1alpha1.KrknUser, role, activeParam, search string) []krknv1alpha1.KrknUser {
	filtered := make([]krknv1alpha1.KrknUser, 0)

	for _, user := range users {
		// Filter by role
		if role != "" && user.Spec.Role != role {
			continue
		}

		// Filter by active status
		if activeParam != "" {
			active, err := strconv.ParseBool(activeParam)
			if err == nil && user.Status.Active != active {
				continue
			}
		}

		// Filter by search term (email, name, or surname)
		if search != "" {
			searchLower := strings.ToLower(search)
			if !strings.Contains(strings.ToLower(user.Spec.UserID), searchLower) &&
				!strings.Contains(strings.ToLower(user.Spec.Name), searchLower) &&
				!strings.Contains(strings.ToLower(user.Spec.Surname), searchLower) {
				continue
			}
		}

		filtered = append(filtered, user)
	}

	return filtered
}

// paginateUsers paginates a list of users
func paginateUsers(users []krknv1alpha1.KrknUser, page, limit int) ([]krknv1alpha1.KrknUser, int) {
	total := len(users)

	// Calculate offset
	offset := (page - 1) * limit
	if offset >= total {
		return []krknv1alpha1.KrknUser{}, total
	}

	// Calculate end index
	end := offset + limit
	if end > total {
		end = total
	}

	return users[offset:end], total
}

// parsePaginationParams parses page and limit query parameters
func parsePaginationParams(r *http.Request) (int, int) {
	page := 1
	limit := 50

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	return page, limit
}

// sanitizeUsername converts an email to a valid Kubernetes resource name
func sanitizeUsername(email string) string {
	// Replace @ and . with -
	name := strings.ReplaceAll(email, "@", "-")
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.ToLower(name)
	return fmt.Sprintf("krknuser-%s", name)
}

// ListUsers handles GET /api/v1/users
// Lists all users with filtering and pagination (admin only)
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("list-users")

	// Check admin privileges
	if !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Parse query parameters
	role := r.URL.Query().Get("role")
	activeParam := r.URL.Query().Get("active")
	search := r.URL.Query().Get("search")

	// List all KrknUser CRDs
	var users krknv1alpha1.KrknUserList
	listOpts := []client.ListOption{
		client.InNamespace(h.namespace),
		client.MatchingLabels{UserAccountLabel: "true"},
	}

	if err := h.client.List(ctx, &users, listOpts...); err != nil {
		logger.Error(err, "Failed to list users")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list users: " + err.Error(),
		})
		return
	}

	// Filter users
	filtered := filterUsers(users.Items, role, activeParam, search)

	// Paginate
	page, limit := parsePaginationParams(r)
	paginated, total := paginateUsers(filtered, page, limit)

	// Convert to response format
	userResponses := make([]UserResponse, len(paginated))
	for i, user := range paginated {
		userResponses[i] = buildUserResponse(&user)
	}

	logger.Info("Listed users", "total", total, "page", page, "limit", limit)

	writeJSON(w, http.StatusOK, ListUsersResponse{
		Users: userResponses,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// GetUser handles GET /api/v1/users/:userID
// Returns a single user by email (admin or self)
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("get-user")

	// Extract userID from path
	userID, err := extractPathSuffix(r.URL.Path, "/api/v1/users/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid userID in path",
		})
		return
	}

	// Remove /password suffix if present
	userID = strings.TrimSuffix(userID, "/password")

	// Fetch user by email
	user, err := h.fetchUserByEmail(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
		} else {
			logger.Error(err, "Failed to fetch user", "userID", userID)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: err.Error(),
			})
		}
		return
	}

	// Check permissions (admin or self)
	claims := auth.GetClaimsFromContext(r.Context())
	if !auth.IsAdmin(r.Context()) && (claims == nil || claims.UserID != userID) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "You can only view your own profile",
		})
		return
	}

	logger.Info("Retrieved user", "userID", userID)

	// Return user (without password)
	response := buildUserResponse(user)
	writeJSON(w, http.StatusOK, response)
}

// CreateUser handles POST /api/v1/users
// Creates a new user (admin only)
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("create-user")

	// Check admin privileges
	if !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Parse and validate request
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	// Validate required fields
	if req.UserID == "" || req.Password == "" || req.Name == "" || req.Surname == "" || req.Role == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "UserID, password, name, surname, and role are required",
		})
		return
	}

	// Validate role
	if req.Role != "user" && req.Role != "admin" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Role must be either 'user' or 'admin'",
		})
		return
	}

	// Validate password
	if err := auth.ValidatePassword(req.Password); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: fmt.Sprintf("Password validation failed: %s", err.Error()),
		})
		return
	}

	// Check for duplicate user
	existingUsers := &krknv1alpha1.KrknUserList{}
	err := h.client.List(ctx, existingUsers, client.InNamespace(h.namespace))
	if err != nil {
		logger.Error(err, "Failed to check existing users")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to check existing users",
		})
		return
	}

	for _, user := range existingUsers.Items {
		if user.Spec.UserID == req.UserID {
			writeJSONError(w, http.StatusConflict, ErrorResponse{
				Error:   "user_exists",
				Message: fmt.Sprintf("User with email %s already exists", req.UserID),
			})
			return
		}
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		logger.Error(err, "Failed to hash password")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to hash password",
		})
		return
	}

	// Create password secret
	userName := sanitizeUsername(req.UserID)
	secretName := fmt.Sprintf("%s-password", userName)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: h.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "krkn-operator",
				"app.kubernetes.io/component": "user-auth",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"passwordHash": []byte(passwordHash),
		},
	}

	if err := h.client.Create(ctx, secret); err != nil {
		if apierrors.IsAlreadyExists(err) {
			// Delete and recreate
			if delErr := h.client.Delete(ctx, secret); delErr != nil {
				logger.Error(delErr, "Failed to delete existing password secret", "secret", secretName)
				writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
					Error:   "internal_error",
					Message: "Failed to recreate password secret",
				})
				return
			}
			if createErr := h.client.Create(ctx, secret); createErr != nil {
				logger.Error(createErr, "Failed to recreate password secret", "secret", secretName)
				writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
					Error:   "internal_error",
					Message: "Failed to create password secret",
				})
				return
			}
		} else {
			logger.Error(err, "Failed to create password secret", "secret", secretName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to create password secret: " + err.Error(),
			})
			return
		}
	}

	// Create KrknUser CRD
	user := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userName,
			Namespace: h.namespace,
			Labels: map[string]string{
				UserAccountLabel: "true",
				AdminRoleLabel:   req.Role,
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID:            req.UserID,
			Name:              req.Name,
			Surname:           req.Surname,
			Organization:      req.Organization,
			Role:              req.Role,
			PasswordSecretRef: secretName,
		},
	}

	if err := h.client.Create(ctx, user); err != nil {
		// Clean up secret
		_ = h.client.Delete(ctx, secret)
		logger.Error(err, "Failed to create user", "userID", req.UserID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create user",
		})
		return
	}

	// Update status separately
	user.Status = krknv1alpha1.KrknUserStatus{
		Active:  true,
		Created: metav1.Now(),
	}
	if err := h.client.Status().Update(ctx, user); err != nil {
		logger.Error(err, "Failed to update KrknUser status", "user", userName)
		// Don't fail - status update is non-critical
	}

	logger.Info("User created successfully", "userID", req.UserID, "role", req.Role)

	writeJSON(w, http.StatusCreated, CreateUserResponse{
		Message: "User created successfully",
		UserID:  req.UserID,
		Role:    req.Role,
	})
}

// UpdateUser handles PATCH /api/v1/users/:userID
// Updates user profile (admin can update all fields, users can only update own profile)
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("update-user")

	// Extract userID from path
	userID, err := extractPathSuffix(r.URL.Path, "/api/v1/users/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid userID in path",
		})
		return
	}

	// Remove /password suffix if present
	userID = strings.TrimSuffix(userID, "/password")

	// Parse request
	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	// Validate at least one field provided
	if req.Name == nil && req.Surname == nil && req.Organization == nil && req.Role == nil && req.Active == nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "At least one field must be provided",
		})
		return
	}

	// Fetch existing user
	user, err := h.fetchUserByEmail(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
		} else {
			logger.Error(err, "Failed to fetch user", "userID", userID)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: err.Error(),
			})
		}
		return
	}

	// Check permissions
	claims := auth.GetClaimsFromContext(r.Context())
	isAdmin := auth.IsAdmin(r.Context())
	isSelf := claims != nil && claims.UserID == userID

	if !isAdmin && !isSelf {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "You can only update your own profile",
		})
		return
	}

	// Check field permissions (role and active are admin-only)
	if !isAdmin && (req.Role != nil || req.Active != nil) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Only admins can change role or active status",
		})
		return
	}

	// Prevent user from disabling themselves
	if isSelf && req.Active != nil && !*req.Active {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "You cannot disable your own account",
		})
		return
	}

	// Validate role if provided
	if req.Role != nil && *req.Role != "user" && *req.Role != "admin" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Role must be either 'user' or 'admin'",
		})
		return
	}

	// Update spec fields
	if req.Name != nil {
		user.Spec.Name = *req.Name
	}
	if req.Surname != nil {
		user.Spec.Surname = *req.Surname
	}
	if req.Organization != nil {
		user.Spec.Organization = *req.Organization
	}
	if req.Role != nil {
		user.Spec.Role = *req.Role
		// Update label too
		if user.Labels == nil {
			user.Labels = make(map[string]string)
		}
		user.Labels[AdminRoleLabel] = *req.Role
	}

	if err := h.client.Update(ctx, user); err != nil {
		logger.Error(err, "Failed to update user", "userID", userID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update user: " + err.Error(),
		})
		return
	}

	// Update status separately if active field provided
	if req.Active != nil {
		user.Status.Active = *req.Active
		if err := h.client.Status().Update(ctx, user); err != nil {
			logger.Error(err, "Failed to update user status", "userID", userID)
			// Non-critical, continue
		}
	}

	logger.Info("User updated successfully", "userID", userID)

	// Return updated user
	response := buildUserResponse(user)
	writeJSON(w, http.StatusOK, UpdateUserResponse{
		Message: "User updated successfully",
		User:    response,
	})
}

// DeleteUser handles DELETE /api/v1/users/:userID
// Deletes a user (admin only, cannot delete self, cannot delete last admin)
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("delete-user")

	// Check admin privileges
	if !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Extract userID from path
	userID, err := extractPathSuffix(r.URL.Path, "/api/v1/users/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid userID in path",
		})
		return
	}

	// Remove /password suffix if present
	userID = strings.TrimSuffix(userID, "/password")

	// Check not deleting self
	claims := auth.GetClaimsFromContext(r.Context())
	if claims != nil && claims.UserID == userID {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "You cannot delete your own account",
		})
		return
	}

	// Fetch user to delete
	user, err := h.fetchUserByEmail(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
		} else {
			logger.Error(err, "Failed to fetch user", "userID", userID)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: err.Error(),
			})
		}
		return
	}

	// If deleting admin, check if last admin
	if user.Spec.Role == "admin" {
		adminList := &krknv1alpha1.KrknUserList{}
		err := h.client.List(ctx, adminList, client.InNamespace(h.namespace), client.MatchingLabels{
			AdminRoleLabel:   "admin",
			UserAccountLabel: "true",
		})

		if err != nil {
			logger.Error(err, "Failed to check admin count")
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to check admin count",
			})
			return
		}

		// Count active admins
		activeAdmins := 0
		for _, admin := range adminList.Items {
			if admin.Status.Active {
				activeAdmins++
			}
		}

		if activeAdmins <= 1 {
			writeJSONError(w, http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "Cannot delete the last active admin user",
			})
			return
		}
	}

	// Delete password secret first
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      user.Spec.PasswordSecretRef,
			Namespace: h.namespace,
		},
	}
	if err := h.client.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "Failed to delete password secret", "secret", user.Spec.PasswordSecretRef)
		// Continue anyway - delete user even if secret cleanup fails
	}

	// Delete user
	if err := h.client.Delete(ctx, user); err != nil {
		logger.Error(err, "Failed to delete user", "userID", userID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete user: " + err.Error(),
		})
		return
	}

	logger.Info("User deleted successfully", "userID", userID)

	writeJSON(w, http.StatusOK, DeleteUserResponse{
		Message: "User deleted successfully",
	})
}

// ChangePassword handles PATCH /api/v1/users/:userID/password
// Changes user password (admin can change any password, users can change own password)
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("change-password")

	// Extract userID from path
	userID, err := extractPathSuffix(r.URL.Path, "/api/v1/users/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid userID in path",
		})
		return
	}

	// Remove "/password" suffix
	userID = strings.TrimSuffix(userID, "/password")

	// Parse request
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	// Validate newPassword
	if err := auth.ValidatePassword(req.NewPassword); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: fmt.Sprintf("Password validation failed: %s", err.Error()),
		})
		return
	}

	// Check permissions
	claims := auth.GetClaimsFromContext(r.Context())
	isAdmin := auth.IsAdmin(r.Context())
	isSelf := claims != nil && claims.UserID == userID

	if !isAdmin && !isSelf {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "You can only change your own password",
		})
		return
	}

	// Fetch user
	user, err := h.fetchUserByEmail(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
		} else {
			logger.Error(err, "Failed to fetch user", "userID", userID)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: err.Error(),
			})
		}
		return
	}

	// Get password secret
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: h.namespace,
		Name:      user.Spec.PasswordSecretRef,
	}

	if err := h.client.Get(ctx, secretKey, secret); err != nil {
		logger.Error(err, "Failed to get password secret", "secret", user.Spec.PasswordSecretRef)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get password secret",
		})
		return
	}

	passwordHash, ok := secret.Data["passwordHash"]
	if !ok {
		logger.Error(fmt.Errorf("passwordHash not found in secret"), "Missing password hash")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Password hash not found",
		})
		return
	}

	// If changing own password, verify current password
	if isSelf && !isAdmin {
		if req.CurrentPassword == "" {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "validation_error",
				Message: "Current password is required when changing your own password",
			})
			return
		}

		// Verify current password
		if !auth.VerifyPassword(req.CurrentPassword, string(passwordHash)) {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "validation_error",
				Message: "Current password is incorrect",
			})
			return
		}

		// Check new password is different
		if auth.VerifyPassword(req.NewPassword, string(passwordHash)) {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "validation_error",
				Message: "New password must be different from current password",
			})
			return
		}
	}

	// Hash new password
	newPasswordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		logger.Error(err, "Failed to hash new password")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to hash new password",
		})
		return
	}

	// Update password secret
	secret.Data["passwordHash"] = []byte(newPasswordHash)

	if err := h.client.Update(ctx, secret); err != nil {
		logger.Error(err, "Failed to update password", "userID", userID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update password",
		})
		return
	}

	logger.Info("Password updated successfully", "userID", userID)

	writeJSON(w, http.StatusOK, ChangePasswordResponse{
		Message: "Password updated successfully",
	})
}

// UsersRouter routes requests to /api/v1/users endpoints
func (h *Handler) UsersRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Root endpoint: /api/v1/users
	if path == "/api/v1/users" {
		if r.Method == http.MethodGet {
			h.ListUsers(w, r)
			return
		}

		if r.Method == http.MethodPost {
			h.CreateUser(w, r)
			return
		}

		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only GET and POST are allowed on /api/v1/users",
		})
		return
	}

	// User-specific endpoint: /api/v1/users/:userID
	if strings.HasPrefix(path, "/api/v1/users/") {
		// Check for password change endpoint
		if strings.HasSuffix(path, "/password") {
			if r.Method == http.MethodPatch {
				h.ChangePassword(w, r)
				return
			}

			writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
				Error:   "method_not_allowed",
				Message: "Only PATCH is allowed for password changes",
			})
			return
		}

		// Regular user operations
		if r.Method == http.MethodGet {
			h.GetUser(w, r)
			return
		}

		if r.Method == http.MethodPatch {
			h.UpdateUser(w, r)
			return
		}

		if r.Method == http.MethodDelete {
			h.DeleteUser(w, r)
			return
		}

		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only GET, PATCH, and DELETE are allowed on user endpoints",
		})
		return
	}

	writeJSONError(w, http.StatusNotFound, ErrorResponse{
		Error:   "not_found",
		Message: "Endpoint not found",
	})
}
