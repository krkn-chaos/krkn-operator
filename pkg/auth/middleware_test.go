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

package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockUserStatusChecker is a test implementation of UserStatusChecker.
type mockUserStatusChecker struct {
	activeUsers map[string]bool // userID -> active
	errUsers    map[string]error
}

func (m *mockUserStatusChecker) IsUserActive(_ context.Context, userID string) (bool, error) {
	if err, ok := m.errUsers[userID]; ok {
		return false, err
	}
	active, ok := m.activeUsers[userID]
	if !ok {
		return false, nil // user not found = inactive
	}
	return active, nil
}

func TestRequireAuth_ValidToken(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)
	middleware := NewMiddleware(tg)

	// Generate a valid token
	token, err := tg.GenerateToken("[email protected]", "user", "Test", "User", "Org")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Create a test handler
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		// Verify claims are in context
		claims := GetClaimsFromContext(r.Context())
		if claims == nil {
			t.Error("Expected claims in context, got nil")
		} else if claims.UserID != "[email protected]" {
			t.Errorf("Expected userID '[email protected]', got '%s'", claims.UserID)
		}

		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	handler := middleware.RequireAuth(testHandler)

	// Create request with valid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if !handlerCalled {
		t.Error("Expected handler to be called")
	}
}

func TestRequireAuth_MissingToken(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)
	middleware := NewMiddleware(tg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	handler := middleware.RequireAuth(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestRequireAuth_InvalidTokenFormat(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)
	middleware := NewMiddleware(tg)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	handler := middleware.RequireAuth(testHandler)

	tests := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "invalid-token"},
		{"wrong prefix", "Basic invalid-token"},
		{"empty bearer", "Bearer "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tt.header)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
			}
		})
	}
}

func TestRequireAuth_ExpiredToken(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		1*time.Millisecond, // Very short duration
		"krkn-operator",
	)
	middleware := NewMiddleware(tg)

	token, err := tg.GenerateToken("[email protected]", "user", "Test", "User", "Org")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for expired token")
	})

	handler := middleware.RequireAuth(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestRequireRole_AdminOnly(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)
	middleware := NewMiddleware(tg)

	tests := []struct {
		name           string
		userRole       string
		expectedStatus int
		expectCalled   bool
	}{
		{
			name:           "admin user - allowed",
			userRole:       "admin",
			expectedStatus: http.StatusOK,
			expectCalled:   true,
		},
		{
			name:           "regular user - forbidden",
			userRole:       "user",
			expectedStatus: http.StatusForbidden,
			expectCalled:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := tg.GenerateToken("[email protected]", tt.userRole, "Test", "User", "Org")
			if err != nil {
				t.Fatalf("Failed to generate token: %v", err)
			}

			handlerCalled := false
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			// Chain middleware: auth -> role check -> handler
			handler := middleware.RequireAuth(middleware.RequireRole(RoleAdmin, testHandler))

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if handlerCalled != tt.expectCalled {
				t.Errorf("Expected handler called=%v, got %v", tt.expectCalled, handlerCalled)
			}
		})
	}
}

func TestRequireAnyRole(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)
	middleware := NewMiddleware(tg)

	allowedRoles := []Role{RoleUser, RoleAdmin}

	tests := []struct {
		name           string
		userRole       string
		expectedStatus int
		expectCalled   bool
	}{
		{
			name:           "admin user - allowed",
			userRole:       "admin",
			expectedStatus: http.StatusOK,
			expectCalled:   true,
		},
		{
			name:           "regular user - allowed",
			userRole:       "user",
			expectedStatus: http.StatusOK,
			expectCalled:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := tg.GenerateToken("[email protected]", tt.userRole, "Test", "User", "Org")
			if err != nil {
				t.Fatalf("Failed to generate token: %v", err)
			}

			handlerCalled := false
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware.RequireAuth(middleware.RequireAnyRole(allowedRoles, testHandler))

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if handlerCalled != tt.expectCalled {
				t.Errorf("Expected handler called=%v, got %v", tt.expectCalled, handlerCalled)
			}
		})
	}
}

func TestGetClaimsFromContext(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)
	middleware := NewMiddleware(tg)

	token, _ := tg.GenerateToken("[email protected]", "admin", "Test", "Admin", "Org")

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaimsFromContext(r.Context())
		if claims == nil {
			t.Error("Expected claims, got nil")
			return
		}

		if claims.UserID != "[email protected]" {
			t.Errorf("Expected userID '[email protected]', got '%s'", claims.UserID)
		}

		if claims.Role != "admin" {
			t.Errorf("Expected role 'admin', got '%s'", claims.Role)
		}
	})

	handler := middleware.RequireAuth(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
}

func TestGetClaimsFromContext_NoClaims(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)

	claims := GetClaimsFromContext(req.Context())
	if claims != nil {
		t.Error("Expected nil claims when not authenticated, got claims")
	}
}

func TestIsAdminFromContext(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)
	middleware := NewMiddleware(tg)

	tests := []struct {
		name     string
		role     string
		expected bool
	}{
		{
			name:     "admin user",
			role:     "admin",
			expected: true,
		},
		{
			name:     "regular user",
			role:     "user",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, _ := tg.GenerateToken("[email protected]", tt.role, "Test", "User", "Org")

			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				result := IsAdmin(r.Context())
				if result != tt.expected {
					t.Errorf("IsAdmin() = %v, want %v", result, tt.expected)
				}
			})

			handler := middleware.RequireAuth(testHandler)

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
		})
	}
}

func TestIsAdminFromContext_NoAuth(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)

	if IsAdmin(req.Context()) {
		t.Error("Expected IsAdmin to return false when not authenticated")
	}
}

func TestRequireAuth_InactiveUser(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)
	middleware := NewMiddleware(tg)

	// Set up a status checker that marks our user as inactive
	checker := &mockUserStatusChecker{
		activeUsers: map[string]bool{
			"[email protected]": false,
		},
		errUsers: make(map[string]error),
	}
	middleware.SetUserStatusChecker(checker)

	token, err := tg.GenerateToken("[email protected]", "user", "Inactive", "User", "Org")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	handler := middleware.RequireAuth(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	if handlerCalled {
		t.Error("Handler should not be called for inactive user")
	}
}

func TestRequireAuth_ActiveUser(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)
	middleware := NewMiddleware(tg)

	// Set up a status checker that marks our user as active
	checker := &mockUserStatusChecker{
		activeUsers: map[string]bool{
			"[email protected]": true,
		},
		errUsers: make(map[string]error),
	}
	middleware.SetUserStatusChecker(checker)

	token, err := tg.GenerateToken("[email protected]", "user", "Active", "User", "Org")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequireAuth(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	if !handlerCalled {
		t.Error("Handler should be called for active user")
	}
}

func TestRequireAuth_UserStatusCheckError(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)
	middleware := NewMiddleware(tg)

	// Set up a status checker that returns an error
	checker := &mockUserStatusChecker{
		activeUsers: make(map[string]bool),
		errUsers: map[string]error{
			"[email protected]": fmt.Errorf("k8s API unavailable"),
		},
	}
	middleware.SetUserStatusChecker(checker)

	token, err := tg.GenerateToken("[email protected]", "user", "Error", "User", "Org")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	handler := middleware.RequireAuth(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should fail closed (500 Internal Server Error)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
	if handlerCalled {
		t.Error("Handler should not be called when status check fails")
	}
}

func TestRequireAuth_NoStatusChecker(t *testing.T) {
	// When no status checker is set, the middleware should allow requests
	// (backward compatible behavior)
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)
	middleware := NewMiddleware(tg)
	// Deliberately NOT setting a user status checker

	token, err := tg.GenerateToken("[email protected]", "user", "Test", "User", "Org")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequireAuth(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	if !handlerCalled {
		t.Error("Handler should be called when no status checker is configured")
	}
}

func TestCachedUserStatusChecker(t *testing.T) {
	callCount := 0
	underlying := &mockUserStatusChecker{
		activeUsers: map[string]bool{
			"[email protected]": true,
		},
		errUsers: make(map[string]error),
	}

	// Wrap with a counting wrapper
	countingChecker := &countingUserStatusChecker{
		delegate:  underlying,
		callCount: &callCount,
	}

	cached := NewCachedUserStatusChecker(countingChecker, 1*time.Minute)

	ctx := context.Background()

	// First call should hit the underlying checker
	active, err := cached.IsUserActive(ctx, "[email protected]")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !active {
		t.Error("Expected user to be active")
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call to underlying checker, got %d", callCount)
	}

	// Second call should use the cache
	active, err = cached.IsUserActive(ctx, "[email protected]")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !active {
		t.Error("Expected user to be active (from cache)")
	}
	if callCount != 1 {
		t.Errorf("Expected still 1 call (cached), got %d", callCount)
	}
}

func TestCachedUserStatusChecker_Invalidation(t *testing.T) {
	underlying := &mockUserStatusChecker{
		activeUsers: map[string]bool{
			"[email protected]": true,
		},
		errUsers: make(map[string]error),
	}

	cached := NewCachedUserStatusChecker(underlying, 1*time.Minute)
	ctx := context.Background()

	// Warm the cache
	active, err := cached.IsUserActive(ctx, "[email protected]")
	if err != nil || !active {
		t.Fatal("Expected active user")
	}

	// Deactivate the user in the underlying checker
	underlying.activeUsers["[email protected]"] = false

	// Cache still returns true
	active, _ = cached.IsUserActive(ctx, "[email protected]")
	if !active {
		t.Error("Expected cached result to still be active")
	}

	// Invalidate the cache entry
	cached.InvalidateUser("[email protected]")

	// Now should see the updated value
	active, _ = cached.IsUserActive(ctx, "[email protected]")
	if active {
		t.Error("Expected user to be inactive after cache invalidation")
	}
}

// countingUserStatusChecker wraps a UserStatusChecker and counts calls.
type countingUserStatusChecker struct {
	delegate  UserStatusChecker
	callCount *int
}

func (c *countingUserStatusChecker) IsUserActive(ctx context.Context, userID string) (bool, error) {
	*c.callCount++
	return c.delegate.IsUserActive(ctx, userID)
}
