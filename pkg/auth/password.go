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
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultCost is the default bcrypt cost (10 is a good balance between security and performance)
	DefaultCost = bcrypt.DefaultCost
	// MinPasswordLength is the minimum password length required
	MinPasswordLength = 8
)

// HashPassword hashes a password using bcrypt.
//
// Parameters:
//   - password: The plaintext password to hash
//
// Returns the bcrypt hash or an error.
func HashPassword(password string) (string, error) {
	if len(password) < MinPasswordLength {
		return "", fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// VerifyPassword compares a plaintext password with a bcrypt hash.
//
// Parameters:
//   - password: The plaintext password to verify
//   - hash: The bcrypt hash to compare against
//
// Returns true if the password matches the hash, false otherwise.
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePassword checks if a password meets the minimum requirements.
//
// Parameters:
//   - password: The password to validate
//
// Returns an error if the password doesn't meet requirements, nil otherwise.
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}

	// Add more validation rules as needed (uppercase, lowercase, numbers, special chars, etc.)
	// For now, we just check minimum length

	return nil
}
