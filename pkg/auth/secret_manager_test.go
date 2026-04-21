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

package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestSecretManager_LoadExistingSecret verifies that SecretManager loads an existing JWT secret
func TestSecretManager_LoadExistingSecret(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      JWTSecretName,
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			JWTSecretKey: []byte("existing-secret-key-32-bytes!!!!"), // Exactly 32 bytes
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existingSecret).
		Build()

	manager := NewSecretManager(k8sClient, "test-namespace", 24*time.Hour, "test-issuer")

	// Test - load the secret
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	secret, err := manager.loadOrCreateSecret(ctx)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, []byte("existing-secret-key-32-bytes!!!!"), secret)
	assert.False(t, manager.IsReady()) // Not ready until Start() is called
}

// TestSecretManager_CreateNewSecret verifies that SecretManager creates a new JWT secret if it doesn't exist
func TestSecretManager_CreateNewSecret(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	manager := NewSecretManager(k8sClient, "test-namespace", 24*time.Hour, "test-issuer")

	// Test - create the secret
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	secret, err := manager.loadOrCreateSecret(ctx)

	// Verify
	require.NoError(t, err)
	assert.Len(t, secret, JWTSecretLength)

	// Verify secret was created in Kubernetes
	createdSecret := &corev1.Secret{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      JWTSecretName,
		Namespace: "test-namespace",
	}, createdSecret)
	require.NoError(t, err)
	assert.Equal(t, secret, createdSecret.Data[JWTSecretKey])
}

// TestSecretManager_RaceCondition verifies that SecretManager handles race condition when multiple replicas try to create the secret
func TestSecretManager_RaceCondition(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	// Simulate race: first replica creates the secret
	ctx := context.Background()
	firstSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      JWTSecretName,
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			JWTSecretKey: []byte("first-replica-created-this-key!!"), // Exactly 32 bytes
		},
	}
	err := k8sClient.Create(ctx, firstSecret)
	require.NoError(t, err)

	// Second replica tries to load/create
	manager := NewSecretManager(k8sClient, "test-namespace", 24*time.Hour, "test-issuer")
	secret, err := manager.loadOrCreateSecret(ctx)

	// Verify - should load the secret created by first replica
	require.NoError(t, err)
	assert.Equal(t, []byte("first-replica-created-this-key!!"), secret)
}

// TestSecretManager_GetTokenGenerator verifies that GetTokenGenerator returns a valid TokenGenerator
func TestSecretManager_GetTokenGenerator(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	manager := NewSecretManager(k8sClient, "test-namespace", 24*time.Hour, "test-issuer")

	// Test - should fail before Start()
	_, err := manager.GetTokenGenerator()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet loaded")

	// Load the secret manually
	ctx := context.Background()
	secret, err := manager.loadOrCreateSecret(ctx)
	require.NoError(t, err)

	// Mark as ready
	manager.mu.Lock()
	manager.secret = secret
	manager.ready = true
	manager.mu.Unlock()

	// Test - should succeed after ready
	tokenGen, err := manager.GetTokenGenerator()
	require.NoError(t, err)
	assert.NotNil(t, tokenGen)
}

// TestSecretManager_NeedLeaderElection verifies that SecretManager does not require leader election
func TestSecretManager_NeedLeaderElection(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	manager := NewSecretManager(k8sClient, "test-namespace", 24*time.Hour, "test-issuer")

	// Test
	needsLeaderElection := manager.NeedLeaderElection()

	// Verify
	assert.False(t, needsLeaderElection,
		"SecretManager should not require leader election (all replicas need the same secret)")
}

// TestSecretManager_IsReady verifies the IsReady state management
func TestSecretManager_IsReady(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	manager := NewSecretManager(k8sClient, "test-namespace", 24*time.Hour, "test-issuer")

	// Test - not ready initially
	assert.False(t, manager.IsReady())

	// Load secret
	ctx := context.Background()
	secret, err := manager.loadOrCreateSecret(ctx)
	require.NoError(t, err)

	// Still not ready (only Start() sets ready flag)
	assert.False(t, manager.IsReady())

	// Simulate Start() completing
	manager.mu.Lock()
	manager.secret = secret
	manager.ready = true
	manager.mu.Unlock()

	// Now ready
	assert.True(t, manager.IsReady())
}

// TestSecretManager_SecretTooShort verifies that SecretManager rejects secrets shorter than required length
func TestSecretManager_SecretTooShort(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	shortSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      JWTSecretName,
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			JWTSecretKey: []byte("too-short"), // Less than 32 bytes
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(shortSecret).
		Build()

	manager := NewSecretManager(k8sClient, "test-namespace", 24*time.Hour, "test-issuer")

	// Test
	ctx := context.Background()
	_, err := manager.loadOrCreateSecret(ctx)

	// Verify - should fail with clear error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

// TestSecretManager_MissingSecretKey verifies that SecretManager rejects secrets without the required key
func TestSecretManager_MissingSecretKey(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	invalidSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      JWTSecretName,
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			"wrong-key": []byte("some-secret-data-here-32-bytes"),
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(invalidSecret).
		Build()

	manager := NewSecretManager(k8sClient, "test-namespace", 24*time.Hour, "test-issuer")

	// Test
	ctx := context.Background()
	_, err := manager.loadOrCreateSecret(ctx)

	// Verify - should fail with clear error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestGenerateRandomSecret verifies that generateRandomSecret produces cryptographically secure random data
func TestGenerateRandomSecret(t *testing.T) {
	// Generate multiple secrets
	secret1, err1 := generateRandomSecret(32)
	secret2, err2 := generateRandomSecret(32)

	// Verify
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Len(t, secret1, 32)
	assert.Len(t, secret2, 32)
	assert.NotEqual(t, secret1, secret2, "Generated secrets should be different")
}

// TestSecretManager_MultiReplicaConsistency documents the expected multi-replica behavior
func TestSecretManager_MultiReplicaConsistency(t *testing.T) {
	t.Run("all replicas load the same secret", func(t *testing.T) {
		// Setup - simulate shared Kubernetes cluster
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)

		sharedClient := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		// Replica 1 creates the secret
		replica1 := NewSecretManager(sharedClient, "test-namespace", 24*time.Hour, "test-issuer")
		ctx := context.Background()
		secret1, err := replica1.loadOrCreateSecret(ctx)
		require.NoError(t, err)

		// Replica 2 loads the same secret
		replica2 := NewSecretManager(sharedClient, "test-namespace", 24*time.Hour, "test-issuer")
		secret2, err := replica2.loadOrCreateSecret(ctx)
		require.NoError(t, err)

		// Verify both replicas have the same secret
		assert.Equal(t, secret1, secret2,
			"All replicas must use the same JWT secret to prevent auth inconsistencies")
	})
}
