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
	"testing"
	"time"
)

func TestNewTokenGenerator(t *testing.T) {
	secretKey := []byte("test-secret-key-at-least-32-bytes-long")
	duration := 24 * time.Hour
	issuer := "krkn-operator"

	tg := NewTokenGenerator(secretKey, duration, issuer)

	if tg == nil {
		t.Fatal("Expected TokenGenerator to be created, got nil")
	}

	if string(tg.secretKey) != string(secretKey) {
		t.Error("Secret key not set correctly")
	}

	if tg.tokenDuration != duration {
		t.Errorf("Token duration = %v, want %v", tg.tokenDuration, duration)
	}

	if tg.issuer != issuer {
		t.Errorf("Issuer = %v, want %v", tg.issuer, issuer)
	}
}

func TestGenerateToken(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)

	tests := []struct {
		name         string
		userID       string
		role         string
		userName     string
		surname      string
		organization string
		wantErr      bool
	}{
		{
			name:         "valid admin user",
			userID:       "[email protected]",
			role:         "admin",
			userName:     "John",
			surname:      "Doe",
			organization: "Example Corp",
			wantErr:      false,
		},
		{
			name:         "valid regular user",
			userID:       "[email protected]",
			role:         "user",
			userName:     "Jane",
			surname:      "Smith",
			organization: "Test Org",
			wantErr:      false,
		},
		{
			name:         "user without organization",
			userID:       "[email protected]",
			role:         "user",
			userName:     "Bob",
			surname:      "Johnson",
			organization: "",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := tg.GenerateToken(tt.userID, tt.role, tt.userName, tt.surname, tt.organization)

			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && token == "" {
				t.Error("Expected token to be generated, got empty string")
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)

	// Generate a valid token
	userID := "[email protected]"
	role := "admin"
	name := "John"
	surname := "Doe"
	organization := "Example Corp"

	token, err := tg.GenerateToken(userID, role, name, surname, organization)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Validate the token
	claims, err := tg.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}

	if claims.Role != role {
		t.Errorf("Role = %v, want %v", claims.Role, role)
	}

	if claims.Name != name {
		t.Errorf("Name = %v, want %v", claims.Name, name)
	}

	if claims.Surname != surname {
		t.Errorf("Surname = %v, want %v", claims.Surname, surname)
	}

	if claims.Organization != organization {
		t.Errorf("Organization = %v, want %v", claims.Organization, organization)
	}

	if claims.Issuer != "krkn-operator" {
		t.Errorf("Issuer = %v, want krkn-operator", claims.Issuer)
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "malformed token",
			token:   "not.a.valid.token",
			wantErr: true,
		},
		{
			name:    "random string",
			token:   "definitely-not-a-jwt-token",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tg.ValidateToken(tt.token)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Create generator with very short duration
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		1*time.Millisecond, // Very short duration
		"krkn-operator",
	)

	token, err := tg.GenerateToken("[email protected]", "user", "Test", "User", "Org")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	_, err = tg.ValidateToken(token)
	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}
}

func TestValidateToken_DifferentSecret(t *testing.T) {
	tg1 := NewTokenGenerator(
		[]byte("secret-key-1-at-least-32-bytes-long!!"),
		24*time.Hour,
		"krkn-operator",
	)

	tg2 := NewTokenGenerator(
		[]byte("secret-key-2-at-least-32-bytes-long!!"),
		24*time.Hour,
		"krkn-operator",
	)

	// Generate token with first generator
	token, err := tg1.GenerateToken("[email protected]", "user", "Test", "User", "Org")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Try to validate with second generator (different secret)
	_, err = tg2.ValidateToken(token)
	if err == nil {
		t.Error("Expected error when validating with different secret, got nil")
	}
}

func TestRefreshToken(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)

	// Generate original token
	originalToken, err := tg.GenerateToken("[email protected]", "admin", "John", "Doe", "Example Corp")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Get original claims for comparison
	originalClaims, _ := tg.ValidateToken(originalToken)

	// Wait a bit to ensure new token has different timestamps
	time.Sleep(1100 * time.Millisecond) // Sleep > 1 second to ensure different IssuedAt

	// Refresh token
	newToken, err := tg.RefreshToken(originalToken)
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}

	// Validate new token
	newClaims, _ := tg.ValidateToken(newToken)

	// New token should have later IssuedAt time
	if !newClaims.IssuedAt.After(originalClaims.IssuedAt.Time) {
		t.Error("Refreshed token should have later IssuedAt timestamp")
	}

	if originalClaims.UserID != newClaims.UserID {
		t.Error("Refreshed token should have same UserID")
	}

	if originalClaims.Role != newClaims.Role {
		t.Error("Refreshed token should have same Role")
	}
}

func TestIsAdmin(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)

	tests := []struct {
		name     string
		role     string
		wantBool bool
	}{
		{
			name:     "admin user",
			role:     "admin",
			wantBool: true,
		},
		{
			name:     "regular user",
			role:     "user",
			wantBool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := tg.GenerateToken("[email protected]", tt.role, "Test", "User", "Org")
			if err != nil {
				t.Fatalf("Failed to generate token: %v", err)
			}

			isAdmin, err := tg.IsAdmin(token)
			if err != nil {
				t.Fatalf("IsAdmin() error = %v", err)
			}

			if isAdmin != tt.wantBool {
				t.Errorf("IsAdmin() = %v, want %v", isAdmin, tt.wantBool)
			}
		})
	}
}

func TestIsAdmin_InvalidToken(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)

	_, err := tg.IsAdmin("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}
}
