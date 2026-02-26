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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

func TestRequireAdminForMethods(t *testing.T) {
	handler := setupAuthTestHandler()

	// Generate tokens
	tg := auth.NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		TokenDuration,
		"krkn-operator",
	)

	adminToken, _ := tg.GenerateToken("[email protected]", "admin", "Admin", "User", "Org")
	userToken, _ := tg.GenerateToken("[email protected]", "user", "Regular", "User", "Org")

	tests := []struct {
		name           string
		token          string
		method         string
		adminMethods   []string
		expectedAllow  bool
		expectedStatus int
	}{
		{
			name:          "admin user - admin method",
			token:         adminToken,
			method:        http.MethodPost,
			adminMethods:  []string{http.MethodPost},
			expectedAllow: true,
		},
		{
			name:           "regular user - admin method",
			token:          userToken,
			method:         http.MethodPost,
			adminMethods:   []string{http.MethodPost},
			expectedAllow:  false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:          "regular user - non-admin method",
			token:         userToken,
			method:        http.MethodGet,
			adminMethods:  []string{http.MethodPost},
			expectedAllow: true,
		},
		{
			name:          "admin user - non-admin method",
			token:         adminToken,
			method:        http.MethodGet,
			adminMethods:  []string{http.MethodPost},
			expectedAllow: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)

			// Validate token and add claims to context (simulating RequireAuth middleware)
			claims, _ := tg.ValidateToken(tt.token)
			ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			result := handler.requireAdminForMethods(w, req, tt.adminMethods)

			if result != tt.expectedAllow {
				t.Errorf("Expected allow=%v, got %v", tt.expectedAllow, result)
			}

			if !tt.expectedAllow && w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
