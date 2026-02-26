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
	"net/http"
	"strings"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// UserClaimsKey is the context key for storing JWT claims
	UserClaimsKey ContextKey = "user-claims"
	// AuthorizationHeader is the HTTP header name for authorization
	AuthorizationHeader = "Authorization"
	// BearerPrefix is the expected prefix for bearer tokens
	BearerPrefix = "Bearer "
)

// Role represents a user role
type Role string

const (
	// RoleAdmin represents an admin user
	RoleAdmin Role = "admin"
	// RoleUser represents a regular user
	RoleUser Role = "user"
)

// Middleware provides HTTP middleware for JWT authentication and authorization
type Middleware struct {
	tokenGen *TokenGenerator
}

// NewMiddleware creates a new authentication middleware
//
// Parameters:
//   - tokenGen: The TokenGenerator used to validate JWT tokens
//
// Returns a new Middleware instance
func NewMiddleware(tokenGen *TokenGenerator) *Middleware {
	return &Middleware{
		tokenGen: tokenGen,
	}
}

// RequireAuth is a middleware that requires a valid JWT token
// It validates the token and adds the claims to the request context
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get(AuthorizationHeader)
		if authHeader == "" {
			http.Error(w, `{"error":"unauthorized","message":"Missing authorization token"}`, http.StatusUnauthorized)
			return
		}

		// Check for Bearer prefix
		if !strings.HasPrefix(authHeader, BearerPrefix) {
			http.Error(w, `{"error":"unauthorized","message":"Invalid authorization header format. Expected: Bearer <token>"}`, http.StatusUnauthorized)
			return
		}

		// Extract token
		token := strings.TrimPrefix(authHeader, BearerPrefix)

		// Validate token
		claims, err := m.tokenGen.ValidateToken(token)
		if err != nil {
			http.Error(w, `{"error":"unauthorized","message":"Invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Add claims to context
		ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole is a middleware that requires a specific role
// Must be used after RequireAuth middleware
func (m *Middleware) RequireRole(role Role, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get claims from context
		claims, ok := r.Context().Value(UserClaimsKey).(*Claims)
		if !ok {
			http.Error(w, `{"error":"unauthorized","message":"No authentication claims found"}`, http.StatusUnauthorized)
			return
		}

		// Check role
		if Role(claims.Role) != role {
			http.Error(w, `{"error":"forbidden","message":"Insufficient permissions"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAnyRole is a middleware that requires any of the specified roles
// Must be used after RequireAuth middleware
func (m *Middleware) RequireAnyRole(roles []Role, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get claims from context
		claims, ok := r.Context().Value(UserClaimsKey).(*Claims)
		if !ok {
			http.Error(w, `{"error":"unauthorized","message":"No authentication claims found"}`, http.StatusUnauthorized)
			return
		}

		// Check if user has any of the required roles
		userRole := Role(claims.Role)
		for _, role := range roles {
			if userRole == role {
				next.ServeHTTP(w, r)
				return
			}
		}

		http.Error(w, `{"error":"forbidden","message":"Insufficient permissions"}`, http.StatusForbidden)
	})
}

// GetClaimsFromContext extracts JWT claims from the request context
//
// Parameters:
//   - ctx: The request context
//
// Returns the claims if found, nil otherwise
func GetClaimsFromContext(ctx context.Context) *Claims {
	claims, ok := ctx.Value(UserClaimsKey).(*Claims)
	if !ok {
		return nil
	}
	return claims
}

// IsAdmin checks if the user in the context is an admin
//
// Parameters:
//   - ctx: The request context
//
// Returns true if the user is an admin, false otherwise
func IsAdmin(ctx context.Context) bool {
	claims := GetClaimsFromContext(ctx)
	if claims == nil {
		return false
	}
	return claims.Role == string(RoleAdmin)
}
