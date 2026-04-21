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

// NewTestHandler creates a Handler with an initialized SecretManager for testing
// IMPORTANT: The JWT secret MUST be included when creating the fake client:
//
//	client := fakeclient.NewClientBuilder().
//	    WithScheme(scheme).
//	    WithRuntimeObjects(user, secret, api.TestJWTSecret("default")).
//	    Build()
//	handler := api.NewTestHandler(client, clientset, "default", "localhost:50051")
//
// If the JWT secret doesn't exist, SecretManager will try to create it
func NewTestHandler(client client.Client, clientset kubernetes.Interface, namespace string, grpcServerAddr string) *Handler {
	// Try to create/initialize SecretManager for testing
	secretManager, err := auth.NewTestSecretManager(client, namespace)
	if err != nil {
		// If it fails, try creating the secret first
		ctx := context.Background()
		_ = client.Create(ctx, TestJWTSecret(namespace))

		// Retry
		secretManager, err = auth.NewTestSecretManager(client, namespace)
		if err != nil {
			// Still failing - panic with helpful message
			panic("NewTestHandler: failed to initialize SecretManager even after creating JWT secret. " +
				"Make sure to include TestJWTSecret(namespace) in WithRuntimeObjects(). Error: " + err.Error())
		}
	}

	return NewHandler(client, clientset, namespace, grpcServerAddr, secretManager)
}