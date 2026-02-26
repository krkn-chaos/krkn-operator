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
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

const (
	// AdminRoleLabel is the label used to identify admin users
	AdminRoleLabel = "krkn.krkn-chaos.dev/role"
	// UserAccountLabel is the label used to identify user account CRDs
	UserAccountLabel = "krkn.krkn-chaos.dev/user-account"
	// JWTSecretKey is the key in the JWT secret
	JWTSecretKey = "jwt-secret"
	// JWTSecretName is the name of the secret containing the JWT signing key
	JWTSecretName = "krkn-operator-jwt-secret"
	// TokenDuration is how long JWT tokens remain valid
	TokenDuration = 24 * time.Hour
)

// IsRegistered handles GET /auth/is-registered
// Returns whether at least one admin user is registered in the system
func (h *Handler) IsRegistered(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only GET method is allowed",
		})
		return
	}

	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("is-registered")

	// List all KrknUser CRDs with admin role label
	userList := &krknv1alpha1.KrknUserList{}
	err := h.client.List(ctx, userList, client.MatchingLabels{
		AdminRoleLabel:   "admin",
		UserAccountLabel: "true",
	})

	if err != nil {
		logger.Error(err, "Failed to list admin users")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to check admin registration status",
		})
		return
	}

	registered := len(userList.Items) > 0

	writeJSON(w, http.StatusOK, IsRegisteredResponse{
		Registered: registered,
	})
}

// Register handles POST /auth/register
// Registers a new user (admin or regular user)
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only POST method is allowed",
		})
		return
	}

	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("register")

	// Parse request body
	var req RegisterRequest
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

	// Check if any admin users exist
	adminList := &krknv1alpha1.KrknUserList{}
	err := h.client.List(ctx, adminList, client.MatchingLabels{
		AdminRoleLabel:   "admin",
		UserAccountLabel: "true",
	})

	if err != nil {
		logger.Error(err, "Failed to list admin users")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to verify admin status",
		})
		return
	}

	hasAdmins := len(adminList.Items) > 0

	// If no admins exist, allow first admin registration without authentication
	// Otherwise, require authentication (will be handled by middleware in the future)
	if hasAdmins {
		// TODO: Add authentication middleware to verify JWT token
		// For now, reject registration if admins exist (authentication not yet implemented)
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "authentication_required",
			Message: "Authentication required to register new users (authentication middleware not yet implemented)",
		})
		return
	}

	// First admin registration - must be admin role
	if !hasAdmins && req.Role != "admin" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "First user must have admin role",
		})
		return
	}

	// Check if user already exists
	existingUsers := &krknv1alpha1.KrknUserList{}
	err = h.client.List(ctx, existingUsers, client.InNamespace(h.namespace))
	if err != nil {
		logger.Error(err, "Failed to list existing users")
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

	// Hash the password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		logger.Error(err, "Failed to hash password")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to process password",
		})
		return
	}

	// Create password secret
	secretName := fmt.Sprintf("krknuser-password-%s", strings.ReplaceAll(strings.ToLower(req.UserID), "@", "-"))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: h.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "krkn-operator",
				"app.kubernetes.io/component":  "authentication",
				"krkn.krkn-chaos.dev/password": "true",
			},
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"passwordHash": passwordHash,
		},
	}

	if err := h.client.Create(ctx, secret); err != nil {
		logger.Error(err, "Failed to create password secret", "secret", secretName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create user credentials",
		})
		return
	}

	// Create KrknUser CRD
	userName := fmt.Sprintf("krknuser-%s", strings.ReplaceAll(strings.ToLower(req.UserID), "@", "-"))
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
		Status: krknv1alpha1.KrknUserStatus{
			Active:  true,
			Created: metav1.Now(),
		},
	}

	if err := h.client.Create(ctx, user); err != nil {
		logger.Error(err, "Failed to create KrknUser", "user", userName)
		// Clean up the secret we just created
		_ = h.client.Delete(ctx, secret)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create user",
		})
		return
	}

	logger.Info("User registered successfully", "userId", req.UserID, "role", req.Role)

	writeJSON(w, http.StatusCreated, RegisterResponse{
		Message: "User registered successfully",
		UserID:  req.UserID,
		Role:    req.Role,
	})
}

// Login handles POST /auth/login
// Authenticates a user and returns a JWT token
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only POST method is allowed",
		})
		return
	}

	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("login")

	// Parse request body
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	// Validate required fields
	if req.UserID == "" || req.Password == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "UserID and password are required",
		})
		return
	}

	// Find user by email
	userList := &krknv1alpha1.KrknUserList{}
	err := h.client.List(ctx, userList, client.InNamespace(h.namespace))
	if err != nil {
		logger.Error(err, "Failed to list users")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to authenticate",
		})
		return
	}

	var user *krknv1alpha1.KrknUser
	for i := range userList.Items {
		if userList.Items[i].Spec.UserID == req.UserID {
			user = &userList.Items[i]
			break
		}
	}

	if user == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "invalid_credentials",
			Message: "Invalid email or password",
		})
		return
	}

	// Check if user is active
	if !user.Status.Active {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "account_disabled",
			Message: "User account is disabled",
		})
		return
	}

	// Get password hash from secret
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: h.namespace,
		Name:      user.Spec.PasswordSecretRef,
	}

	if err := h.client.Get(ctx, secretKey, secret); err != nil {
		logger.Error(err, "Failed to get password secret", "secret", user.Spec.PasswordSecretRef)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to authenticate",
		})
		return
	}

	passwordHash, ok := secret.Data["passwordHash"]
	if !ok {
		logger.Error(fmt.Errorf("passwordHash not found in secret"), "Missing password hash")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to authenticate",
		})
		return
	}

	// Verify password
	if !auth.VerifyPassword(req.Password, string(passwordHash)) {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "invalid_credentials",
			Message: "Invalid email or password",
		})
		return
	}

	// Get or create JWT secret
	jwtSecret, err := h.getOrCreateJWTSecret(ctx)
	if err != nil {
		logger.Error(err, "Failed to get JWT secret")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to generate token",
		})
		return
	}

	// Generate JWT token
	tokenGen := auth.NewTokenGenerator(jwtSecret, TokenDuration, "krkn-operator")
	token, err := tokenGen.GenerateToken(user.Spec.UserID, user.Spec.Role, user.Spec.Name, user.Spec.Surname, user.Spec.Organization)
	if err != nil {
		logger.Error(err, "Failed to generate token")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to generate token",
		})
		return
	}

	// Update last login timestamp
	user.Status.LastLogin = metav1.Now()
	if err := h.client.Status().Update(ctx, user); err != nil {
		logger.Error(err, "Failed to update last login timestamp")
		// Non-critical error, continue
	}

	logger.Info("User logged in successfully", "userId", user.Spec.UserID)

	expiresAt := time.Now().Add(TokenDuration).Format(time.RFC3339)

	writeJSON(w, http.StatusOK, LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		UserID:    user.Spec.UserID,
		Role:      user.Spec.Role,
		Name:      user.Spec.Name,
		Surname:   user.Spec.Surname,
	})
}

// getOrCreateJWTSecret retrieves the JWT secret or creates it if it doesn't exist
func (h *Handler) getOrCreateJWTSecret(ctx context.Context) ([]byte, error) {
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: h.namespace,
		Name:      JWTSecretName,
	}

	err := h.client.Get(ctx, secretKey, secret)
	if err == nil {
		// Secret exists
		jwtSecret, ok := secret.Data[JWTSecretKey]
		if !ok {
			return nil, fmt.Errorf("jwt-secret key not found in secret")
		}
		return jwtSecret, nil
	}

	// Secret doesn't exist, create it
	// Generate a random 32-byte secret
	randomSecret := make([]byte, 32)
	for i := range randomSecret {
		randomSecret[i] = byte(time.Now().UnixNano() % 256)
	}

	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      JWTSecretName,
			Namespace: h.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "krkn-operator",
				"app.kubernetes.io/component": "authentication",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			JWTSecretKey: randomSecret,
		},
	}

	if err := h.client.Create(ctx, newSecret); err != nil {
		return nil, fmt.Errorf("failed to create JWT secret: %w", err)
	}

	return randomSecret, nil
}
