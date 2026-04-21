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
*/

// Package auth provides JWT secret management with multi-replica safety
package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// JWTSecretName is the name of the Kubernetes Secret containing the JWT signing key
	JWTSecretName = "krkn-operator-jwt" // #nosec G101 -- This is a Secret name (metadata), not credentials; actual secret value is randomly generated and stored in Secret.Data[JWTSecretKey]
	// JWTSecretKey is the key in the Secret data map
	JWTSecretKey = "jwt-secret"
	// JWTSecretLength is the length of the JWT secret in bytes (256 bits)
	JWTSecretLength = 32
)

// SecretManager manages the JWT signing secret across multiple operator replicas
// It ensures all replicas use the same secret from Kubernetes, preventing auth inconsistencies
type SecretManager struct {
	client        client.Client
	namespace     string
	tokenDuration time.Duration
	issuer        string

	mu     sync.RWMutex
	secret []byte
	ready  bool
}

// NewSecretManager creates a new JWT secret manager
//
// Parameters:
//   - client: Kubernetes client for reading/creating secrets
//   - namespace: Namespace where the JWT secret is stored
//   - tokenDuration: Duration for which tokens are valid
//   - issuer: Token issuer name
//
// Returns a new SecretManager instance
func NewSecretManager(client client.Client, namespace string, tokenDuration time.Duration, issuer string) *SecretManager {
	return &SecretManager{
		client:        client,
		namespace:     namespace,
		tokenDuration: tokenDuration,
		issuer:        issuer,
	}
}

// Start implements manager.Runnable interface
// It loads or creates the JWT secret after the manager cache is ready
func (s *SecretManager) Start(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("jwt-secret-manager")
	logger.Info("🔐 Starting JWT secret manager", "namespace", s.namespace)

	// Load or create the secret with retry logic
	secret, err := s.loadOrCreateSecretWithRetry(ctx, 10, time.Second)
	if err != nil {
		logger.Error(err, "❌ FATAL: Failed to load or create JWT secret after all retries")
		return fmt.Errorf("fatal: could not initialize JWT secret: %w", err)
	}

	// Store the secret
	s.mu.Lock()
	s.secret = secret
	s.ready = true
	s.mu.Unlock()

	logger.Info("✅ JWT secret loaded successfully - authentication system ready",
		"secretLength", len(secret),
		"namespace", s.namespace)

	// Keep running to allow graceful shutdown
	<-ctx.Done()
	logger.Info("JWT secret manager stopped")
	return nil
}

// NeedLeaderElection implements manager.LeaderElectionRunnable
// Returns false because all replicas need to load the same secret
func (s *SecretManager) NeedLeaderElection() bool {
	return false
}

// GetTokenGenerator returns a TokenGenerator using the loaded secret
// Returns error if the secret is not yet ready
func (s *SecretManager) GetTokenGenerator() (*TokenGenerator, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.ready {
		return nil, fmt.Errorf("JWT secret not yet loaded, authentication system not ready")
	}

	return NewTokenGenerator(s.secret, s.tokenDuration, s.issuer), nil
}

// IsReady returns true if the JWT secret has been loaded
func (s *SecretManager) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ready
}

// loadOrCreateSecretWithRetry attempts to load or create the JWT secret with retry logic
func (s *SecretManager) loadOrCreateSecretWithRetry(ctx context.Context, maxRetries int, initialBackoff time.Duration) ([]byte, error) {
	logger := log.FromContext(ctx).WithName("jwt-secret-manager")
	backoff := initialBackoff

	logger.Info("🔄 Starting JWT secret load/create with retry",
		"maxRetries", maxRetries,
		"initialBackoff", initialBackoff)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		logger.Info("Attempting to load/create JWT secret",
			"attempt", attempt,
			"maxRetries", maxRetries)

		secret, err := s.loadOrCreateSecret(ctx)
		if err == nil {
			logger.Info("✅ Successfully loaded/created JWT secret",
				"attempt", attempt,
				"secretLength", len(secret))
			return secret, nil
		}

		if attempt == maxRetries {
			logger.Error(err, "❌ Failed to load/create JWT secret after all retries",
				"attempts", maxRetries,
				"lastError", err.Error())
			return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, err)
		}

		logger.Info("⚠️ Failed to load/create JWT secret, will retry",
			"attempt", attempt,
			"maxRetries", maxRetries,
			"backoff", backoff,
			"error", err.Error())

		select {
		case <-ctx.Done():
			logger.Info("Context cancelled during retry wait")
			return nil, ctx.Err()
		case <-time.After(backoff):
			// Exponential backoff (capped at 30s)
			backoff = backoff * 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}

	return nil, fmt.Errorf("unreachable")
}

// loadOrCreateSecret attempts to load the JWT secret from Kubernetes or create it if it doesn't exist
func (s *SecretManager) loadOrCreateSecret(ctx context.Context) ([]byte, error) {
	logger := log.FromContext(ctx).WithName("jwt-secret-manager")

	secret := &corev1.Secret{}
	secretKey := types.NamespacedName{
		Namespace: s.namespace,
		Name:      JWTSecretName,
	}

	// Try to get existing secret
	logger.V(1).Info("Attempting to get JWT secret from Kubernetes",
		"name", JWTSecretName,
		"namespace", s.namespace)

	err := s.client.Get(ctx, secretKey, secret)
	if err == nil {
		// Secret exists
		logger.Info("📥 Found existing JWT secret in Kubernetes",
			"name", JWTSecretName,
			"namespace", s.namespace,
			"creationTimestamp", secret.CreationTimestamp)

		jwtSecret, ok := secret.Data[JWTSecretKey]
		if !ok {
			logger.Error(nil, "❌ JWT secret exists but missing required key",
				"secretName", JWTSecretName,
				"expectedKey", JWTSecretKey,
				"availableKeys", getSecretKeys(secret.Data))
			return nil, fmt.Errorf("jwt-secret key '%s' not found in secret", JWTSecretKey)
		}

		if len(jwtSecret) < JWTSecretLength {
			logger.Error(nil, "❌ JWT secret too short",
				"actualLength", len(jwtSecret),
				"expectedLength", JWTSecretLength)
			return nil, fmt.Errorf("jwt-secret is too short (%d bytes, expected %d)", len(jwtSecret), JWTSecretLength)
		}

		logger.Info("✅ Successfully loaded existing JWT secret",
			"secretLength", len(jwtSecret))
		return jwtSecret, nil
	}

	if !apierrors.IsNotFound(err) {
		logger.Error(err, "❌ Failed to get JWT secret from Kubernetes",
			"name", JWTSecretName,
			"namespace", s.namespace,
			"errorType", fmt.Sprintf("%T", err))
		return nil, fmt.Errorf("failed to get JWT secret: %w", err)
	}

	// Secret doesn't exist, create it
	logger.Info("📝 JWT secret not found, creating new secret",
		"name", JWTSecretName,
		"namespace", s.namespace)

	randomSecret, err := generateRandomSecret(JWTSecretLength)
	if err != nil {
		logger.Error(err, "❌ Failed to generate random secret")
		return nil, fmt.Errorf("failed to generate random secret: %w", err)
	}

	logger.V(1).Info("Generated random JWT secret",
		"length", len(randomSecret))

	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      JWTSecretName,
			Namespace: s.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "krkn-operator",
				"app.kubernetes.io/component": "authentication",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			JWTSecretKey: randomSecret,
		},
	}

	logger.Info("Creating new JWT secret in Kubernetes",
		"name", JWTSecretName,
		"namespace", s.namespace)

	if err := s.client.Create(ctx, newSecret); err != nil {
		// Handle race condition - another replica may have created it
		if apierrors.IsAlreadyExists(err) {
			logger.Info("⚠️ JWT secret already exists (race condition - created by another replica), retrieving it",
				"name", JWTSecretName,
				"namespace", s.namespace)

			// Retry Get to load the secret created by the other replica
			if getErr := s.client.Get(ctx, secretKey, secret); getErr != nil {
				logger.Error(getErr, "❌ JWT secret exists but failed to retrieve it after race condition")
				return nil, fmt.Errorf("JWT secret exists but failed to retrieve it: %w", getErr)
			}

			jwtSecret, ok := secret.Data[JWTSecretKey]
			if !ok {
				logger.Error(nil, "❌ JWT secret created by other replica is missing required key",
					"expectedKey", JWTSecretKey,
					"availableKeys", getSecretKeys(secret.Data))
				return nil, fmt.Errorf("jwt-secret key not found in existing secret")
			}

			if len(jwtSecret) < JWTSecretLength {
				logger.Error(nil, "❌ JWT secret created by other replica is too short",
					"actualLength", len(jwtSecret),
					"expectedLength", JWTSecretLength)
				return nil, fmt.Errorf("existing jwt-secret is too short (%d bytes, expected %d)", len(jwtSecret), JWTSecretLength)
			}

			logger.Info("✅ Successfully loaded JWT secret created by another replica",
				"secretLength", len(jwtSecret))
			return jwtSecret, nil
		}

		logger.Error(err, "❌ Failed to create JWT secret in Kubernetes",
			"name", JWTSecretName,
			"namespace", s.namespace)
		return nil, fmt.Errorf("failed to create JWT secret: %w", err)
	}

	logger.Info("✅ Successfully created new JWT secret",
		"name", JWTSecretName,
		"namespace", s.namespace,
		"secretLength", len(randomSecret))
	return randomSecret, nil
}

// generateRandomSecret generates a cryptographically secure random secret
func generateRandomSecret(length int) ([]byte, error) {
	secret := make([]byte, length)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("failed to read random bytes: %w", err)
	}
	return secret, nil
}

// getSecretKeys returns the keys available in a secret data map (for logging)
func getSecretKeys(data map[string][]byte) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}
