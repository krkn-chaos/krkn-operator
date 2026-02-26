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

// Package auth provides authentication and authorization utilities for the krkn-operator ecosystem.
//
// This package contains reusable authentication components that can be shared across
// all operators in the krkn-operator-ecosystem, including:
//   - JWT token generation and validation
//   - Password hashing and verification using bcrypt
//   - User authentication helpers
//
// # JWT Authentication
//
// The package provides a TokenGenerator for creating and validating JWT tokens:
//
//	secretKey := []byte("your-secret-key-at-least-32-bytes-long")
//	tokenGen := auth.NewTokenGenerator(secretKey, 24*time.Hour, "krkn-operator")
//
//	// Generate a token
//	token, err := tokenGen.GenerateToken("[email protected]", "admin", "John", "Doe", "Example Corp")
//	if err != nil {
//	    // handle error
//	}
//
//	// Validate a token
//	claims, err := tokenGen.ValidateToken(token)
//	if err != nil {
//	    // handle error
//	}
//
//	// Check if user is admin
//	isAdmin, err := tokenGen.IsAdmin(token)
//
// # Password Management
//
// The package provides secure password hashing using bcrypt:
//
//	// Hash a password
//	hash, err := auth.HashPassword("user-password-123")
//	if err != nil {
//	    // handle error
//	}
//
//	// Verify a password
//	if auth.VerifyPassword("user-password-123", hash) {
//	    // password is correct
//	}
//
//	// Validate password meets requirements
//	if err := auth.ValidatePassword(password); err != nil {
//	    // password doesn't meet requirements
//	}
//
// All utilities in this package are thread-safe and can be used concurrently
// across multiple goroutines.
package auth
