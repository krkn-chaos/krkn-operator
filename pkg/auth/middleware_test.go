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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

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
