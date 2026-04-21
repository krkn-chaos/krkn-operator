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

package api

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

// TestJWTSecret returns a corev1.Secret with the test JWT key
// Add this to WithRuntimeObjects() when creating fake clients for tests
func TestJWTSecret(namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      auth.JWTSecretName,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			auth.JWTSecretKey: []byte("test-jwt-secret-key-32-bytes!!!!"), // Exactly 32 bytes
		},
	}
}

// NewTestHandler creates a Handler with a SecretManager for testing
//
// For tests that use JWT authentication (Login, Register, etc.):
//
//	scheme := runtime.NewScheme()
//	_ = krknv1alpha1.AddToScheme(scheme)
//	_ = corev1.AddToScheme(scheme)  // REQUIRED for JWT tests
//	client := fakeclient.NewClientBuilder().
//	    WithScheme(scheme).
//	    WithRuntimeObjects(user, secret, api.TestJWTSecret("default")).
//	    Build()
//
// For tests that don't use JWT (authorization tests, etc.):
//
//	scheme := runtime.NewScheme()
//	_ = krknv1alpha1.AddToScheme(scheme)
//	client := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
//	// SecretManager will not be initialized, but handler will still work
//	// for endpoints that don't require JWT generation/validation
func NewTestHandler(client client.Client, clientset kubernetes.Interface, namespace string, grpcServerAddr string) *Handler {
	// Try to create/initialize SecretManager for testing
	// This will fail if the scheme doesn't include corev1 (Secret type not registered)
	secretManager, err := auth.NewTestSecretManager(client, namespace)
	if err != nil {
		// Check if this is a scheme issue (Secret type not registered)
		if strings.Contains(err.Error(), "no kind is registered for the type") {
			// Scheme doesn't support Secret - create an uninitialized SecretManager
			// Tests that don't use JWT will work fine
			// Tests that try to use JWT will get a clear error when they call GetTokenGenerator()
			secretManager = auth.NewSecretManager(client, namespace, TokenDuration, "krkn-operator")
		} else {
			// Try creating the secret first (maybe it just doesn't exist)
			ctx := context.Background()
			_ = client.Create(ctx, TestJWTSecret(namespace))

			// Retry
			secretManager, err = auth.NewTestSecretManager(client, namespace)
			if err != nil {
				// Still failing - create uninitialized manager
				// This allows tests to run but JWT operations will fail with clear errors
				secretManager = auth.NewSecretManager(client, namespace, TokenDuration, "krkn-operator")
			}
		}
	}

	return NewHandler(client, clientset, namespace, grpcServerAddr, secretManager)
}
