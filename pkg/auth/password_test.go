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
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "securepassword123",
			wantErr:  false,
		},
		{
			name:     "minimum length password",
			password: "12345678", // exactly 8 characters
			wantErr:  false,
		},
		{
			name:     "long password",
			password: "this-is-a-very-long-and-secure-password-with-many-characters",
			wantErr:  false,
		},
		{
			name:     "password with special characters",
			password: "P@ssw0rd!#$",
			wantErr:  false,
		},
		{
			name:     "too short password",
			password: "short",
			wantErr:  true,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)

			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if hash == "" {
					t.Error("Expected hash to be generated, got empty string")
				}

				if hash == tt.password {
					t.Error("Hash should not be the same as plaintext password")
				}

				// Bcrypt hashes start with "$2a$" or "$2b$"
				if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
					t.Errorf("Hash doesn't appear to be a valid bcrypt hash: %s", hash)
				}
			}
		})
	}
}

func TestHashPassword_Uniqueness(t *testing.T) {
	password := "testpassword123"

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Bcrypt adds salt, so hashes of the same password should be different
	if hash1 == hash2 {
		t.Error("Two hashes of the same password should be different (bcrypt uses salt)")
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "correctpassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{
			name:     "correct password",
			password: password,
			hash:     hash,
			want:     true,
		},
		{
			name:     "incorrect password",
			password: "wrongpassword",
			hash:     hash,
			want:     false,
		},
		{
			name:     "empty password",
			password: "",
			hash:     hash,
			want:     false,
		},
		{
			name:     "case sensitive - wrong case",
			password: "CORRECTPASSWORD123",
			hash:     hash,
			want:     false,
		},
		{
			name:     "password with extra characters",
			password: password + "extra",
			hash:     hash,
			want:     false,
		},
		{
			name:     "invalid hash",
			password: password,
			hash:     "not-a-valid-hash",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifyPassword(tt.password, tt.hash)

			if result != tt.want {
				t.Errorf("VerifyPassword() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestVerifyPassword_MultiplePasswords(t *testing.T) {
	passwords := []string{
		"password1",
		"password2",
		"password3",
	}

	hashes := make([]string, len(passwords))
	for i, pwd := range passwords {
		hash, err := HashPassword(pwd)
		if err != nil {
			t.Fatalf("Failed to hash password %s: %v", pwd, err)
		}
		hashes[i] = hash
	}

	// Verify correct passwords
	for i, pwd := range passwords {
		if !VerifyPassword(pwd, hashes[i]) {
			t.Errorf("Password %s should match its hash", pwd)
		}
	}

	// Verify cross-verification fails
	for i, pwd := range passwords {
		for j, hash := range hashes {
			if i != j {
				if VerifyPassword(pwd, hash) {
					t.Errorf("Password %s should not match hash of password %s", pwd, passwords[j])
				}
			}
		}
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password - minimum length",
			password: "12345678",
			wantErr:  false,
		},
		{
			name:     "valid password - longer",
			password: "thisisalongpassword",
			wantErr:  false,
		},
		{
			name:     "valid password - with special chars",
			password: "P@ssw0rd!",
			wantErr:  false,
		},
		{
			name:     "invalid - too short",
			password: "short",
			wantErr:  true,
		},
		{
			name:     "invalid - exactly 7 chars",
			password: "1234567",
			wantErr:  true,
		},
		{
			name:     "invalid - empty",
			password: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHashAndVerify_Integration(t *testing.T) {
	password := "integration-test-password-123"

	// Hash the password
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() failed: %v", err)
	}

	// Verify correct password
	if !VerifyPassword(password, hash) {
		t.Error("VerifyPassword() should return true for correct password")
	}

	// Verify incorrect password
	if VerifyPassword("wrong-password", hash) {
		t.Error("VerifyPassword() should return false for incorrect password")
	}
}

func TestMinPasswordLength(t *testing.T) {
	if MinPasswordLength != 8 {
		t.Errorf("MinPasswordLength = %d, want 8", MinPasswordLength)
	}
}
