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

// Package auth provides authentication utilities for the krkn-operator ecosystem.
// This includes JWT token generation, validation, and password hashing utilities
// that can be shared across all operators in the ecosystem.
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT claims for krkn-operator authentication.
// It extends the standard JWT claims with user-specific information.
type Claims struct {
	UserID       string `json:"userId"`       // User's email address
	Role         string `json:"role"`         // User role: "user" or "admin"
	Name         string `json:"name"`         // User's first name
	Surname      string `json:"surname"`      // User's last name
	Organization string `json:"organization"` // User's organization
	jwt.RegisteredClaims
}

// TokenGenerator handles JWT token generation and validation.
type TokenGenerator struct {
	secretKey     []byte
	tokenDuration time.Duration
	issuer        string
}

// NewTokenGenerator creates a new JWT token generator.
//
// Parameters:
//   - secretKey: The secret key used to sign JWT tokens (should be at least 32 bytes)
//   - tokenDuration: How long tokens remain valid (e.g., 24 hours)
//   - issuer: The issuer claim for tokens (typically "krkn-operator")
//
// Returns a TokenGenerator instance.
func NewTokenGenerator(secretKey []byte, tokenDuration time.Duration, issuer string) *TokenGenerator {
	return &TokenGenerator{
		secretKey:     secretKey,
		tokenDuration: tokenDuration,
		issuer:        issuer,
	}
}

// GenerateToken creates a new JWT token for a user.
//
// Parameters:
//   - userID: The user's email address
//   - role: The user's role ("user" or "admin")
//   - name: The user's first name
//   - surname: The user's last name
//   - organization: The user's organization (optional)
//
// Returns the signed JWT token string or an error.
func (tg *TokenGenerator) GenerateToken(userID, role, name, surname, organization string) (string, error) {
	now := time.Now()
	expirationTime := now.Add(tg.tokenDuration)

	claims := &Claims{
		UserID:       userID,
		Role:         role,
		Name:         name,
		Surname:      surname,
		Organization: organization,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    tg.issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(tg.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken validates a JWT token and returns the claims.
//
// Parameters:
//   - tokenString: The JWT token string to validate
//
// Returns the claims if valid, or an error if the token is invalid or expired.
func (tg *TokenGenerator) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tg.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// RefreshToken generates a new token with the same claims but updated expiration.
//
// Parameters:
//   - tokenString: The current JWT token to refresh
//
// Returns a new JWT token with extended expiration or an error.
func (tg *TokenGenerator) RefreshToken(tokenString string) (string, error) {
	claims, err := tg.ValidateToken(tokenString)
	if err != nil {
		return "", fmt.Errorf("cannot refresh invalid token: %w", err)
	}

	// Generate new token with same user info
	return tg.GenerateToken(
		claims.UserID,
		claims.Role,
		claims.Name,
		claims.Surname,
		claims.Organization,
	)
}

// IsAdmin checks if the token belongs to an admin user.
//
// Parameters:
//   - tokenString: The JWT token to check
//
// Returns true if the user has admin role, false otherwise.
func (tg *TokenGenerator) IsAdmin(tokenString string) (bool, error) {
	claims, err := tg.ValidateToken(tokenString)
	if err != nil {
		return false, err
	}

	return claims.Role == "admin", nil
}
