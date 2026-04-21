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
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InitializeForTesting manually initializes a SecretManager for testing
// This bypasses the normal Start() flow which is blocking and requires a running manager
// Use this in tests that need JWT functionality
func (s *SecretManager) InitializeForTesting(ctx context.Context) error {
	secret, err := s.loadOrCreateSecret(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.secret = secret
	s.ready = true
	s.mu.Unlock()

	return nil
}

// NewTestSecretManager creates and initializes a SecretManager for testing
// The secret is loaded/created immediately and the manager is marked as ready
func NewTestSecretManager(client client.Client, namespace string) (*SecretManager, error) {
	sm := NewSecretManager(client, namespace, 24*time.Hour, "test-issuer")

	ctx := context.Background()
	if err := sm.InitializeForTesting(ctx); err != nil {
		return nil, err
	}

	return sm, nil
}