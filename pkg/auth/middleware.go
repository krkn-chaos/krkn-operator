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
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
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

// UserStatusChecker checks whether a user account is still active.
// Implementations should look up the user (e.g., KrknUser CR) and return
// whether the account is active. This interface decouples the auth middleware
// from Kubernetes API types, keeping the pkg/auth package reusable.
type UserStatusChecker interface {
	// IsUserActive returns true if the user account identified by userID is active.
	// It may use caching to avoid hitting the backing store on every request.
	IsUserActive(ctx context.Context, userID string) (bool, error)
}

// userStatusCacheEntry stores a cached active status with its expiry time.
type userStatusCacheEntry struct {
	active  bool
	expires time.Time
}

// CachedUserStatusChecker wraps a UserStatusChecker with a TTL cache to avoid
// looking up user status on every authenticated request.
type CachedUserStatusChecker struct {
	checker UserStatusChecker
	mu      sync.RWMutex
	cache   map[string]userStatusCacheEntry
	ttl     time.Duration
}

// NewCachedUserStatusChecker creates a new cached user status checker.
//
// Parameters:
//   - checker: the underlying checker that performs the actual lookup
//   - ttl: how long to cache each result (e.g., 1 minute)
//
// Returns a CachedUserStatusChecker instance.
func NewCachedUserStatusChecker(checker UserStatusChecker, ttl time.Duration) *CachedUserStatusChecker {
	return &CachedUserStatusChecker{
		checker: checker,
		cache:   make(map[string]userStatusCacheEntry),
		ttl:     ttl,
	}
}

// IsUserActive checks the cache first, then falls back to the underlying checker.
func (c *CachedUserStatusChecker) IsUserActive(ctx context.Context, userID string) (bool, error) {
	c.mu.RLock()
	entry, exists := c.cache[userID]
	c.mu.RUnlock()

	if exists && time.Now().Before(entry.expires) {
		return entry.active, nil
	}

	// Cache miss or expired -- look up the user
	active, err := c.checker.IsUserActive(ctx, userID)
	if err != nil {
		return false, err
	}

	c.mu.Lock()
	c.cache[userID] = userStatusCacheEntry{
		active:  active,
		expires: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return active, nil
}

// InvalidateUser removes a user from the cache, forcing a fresh lookup on the next request.
func (c *CachedUserStatusChecker) InvalidateUser(userID string) {
	c.mu.Lock()
	delete(c.cache, userID)
	c.mu.Unlock()
}

// Middleware provides HTTP middleware for JWT authentication and authorization
type Middleware struct {
	tokenGen          *TokenGenerator
	tokenGenLoader    func() *TokenGenerator
	userStatusChecker UserStatusChecker
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

// NewLazyMiddleware creates a new authentication middleware with lazy token generator loading
//
// Parameters:
//   - tokenGenLoader: A function that returns the TokenGenerator (called on first use)
//
// Returns a new Middleware instance that loads the TokenGenerator when first needed
func NewLazyMiddleware(tokenGenLoader func() *TokenGenerator) *Middleware {
	return &Middleware{
		tokenGenLoader: tokenGenLoader,
	}
}

// SetUserStatusChecker sets the user status checker on the middleware.
// When set, the middleware will verify that the user account is still active
// after JWT validation succeeds. If the user is inactive, the request is
// rejected with 401 Unauthorized.
//
// Parameters:
//   - checker: the UserStatusChecker to use (typically a CachedUserStatusChecker)
func (m *Middleware) SetUserStatusChecker(checker UserStatusChecker) {
	m.userStatusChecker = checker
}

// RequireAuth is a middleware that requires a valid JWT token
// It validates the token and adds the claims to the request context
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := log.Log.WithName("auth-middleware")

		// Get token generator (lazy load if needed)
		tokenGen := m.tokenGen
		if tokenGen == nil && m.tokenGenLoader != nil {
			tokenGen = m.tokenGenLoader()
			// Cache it for subsequent requests
			m.tokenGen = tokenGen
		}

		if tokenGen == nil {
			logger.Error(nil, "TokenGenerator not initialized")
			http.Error(w, `{"error":"internal_error","message":"Authentication system not ready"}`, http.StatusInternalServerError)
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get(AuthorizationHeader)
		if authHeader == "" {
			logger.Info("Authentication failed: missing Authorization header",
				"path", r.URL.Path,
				"method", r.Method,
			)
			http.Error(w, `{"error":"unauthorized","message":"Missing authorization token"}`, http.StatusUnauthorized)
			return
		}

		// Check for Bearer prefix
		if !strings.HasPrefix(authHeader, BearerPrefix) {
			logger.Info("Authentication failed: invalid Authorization header format",
				"path", r.URL.Path,
				"method", r.Method,
				"header", authHeader,
			)
			http.Error(w, `{"error":"unauthorized","message":"Invalid authorization header format. Expected: Bearer <token>"}`, http.StatusUnauthorized)
			return
		}

		// Extract token
		token := strings.TrimPrefix(authHeader, BearerPrefix)

		// Validate token
		claims, err := tokenGen.ValidateToken(token)
		if err != nil {
			logger.Info("Authentication failed: token validation failed",
				"path", r.URL.Path,
				"method", r.Method,
				"error", err.Error(),
			)
			http.Error(w, `{"error":"unauthorized","message":"Invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Check if the user account is still active (if a checker is configured)
		if m.userStatusChecker != nil {
			active, err := m.userStatusChecker.IsUserActive(r.Context(), claims.UserID)
			if err != nil {
				logger.Error(err, "Failed to check user active status",
					"path", r.URL.Path,
					"method", r.Method,
					"userId", claims.UserID,
				)
				// Fail open on transient errors would be a security risk.
				// Fail closed: deny access if we cannot verify user status.
				http.Error(w, `{"error":"internal_error","message":"Failed to verify user account status"}`, http.StatusInternalServerError)
				return
			}
			if !active {
				logger.Info("Authentication failed: user account is inactive",
					"path", r.URL.Path,
					"method", r.Method,
					"userId", claims.UserID,
				)
				http.Error(w, `{"error":"unauthorized","message":"User account is inactive"}`, http.StatusUnauthorized)
				return
			}
		}

		logger.V(1).Info("Authentication successful",
			"path", r.URL.Path,
			"method", r.Method,
			"userId", claims.UserID,
			"role", claims.Role,
		)

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
