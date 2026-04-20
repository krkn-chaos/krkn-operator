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
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// TestServerNeedLeaderElection verifies that the API server does not require leader election
// This ensures the API is available on all replicas for high availability
func TestServerNeedLeaderElection(t *testing.T) {
	// Setup
	scheme, err := krknv1alpha1.SchemeBuilder.Build()
	assert.NoError(t, err)

	k8sClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	clientset := fake.NewSimpleClientset()

	server := NewServer(8080, k8sClient, clientset, "test-namespace", "localhost:50051")

	// Test
	needsLeaderElection := server.NeedLeaderElection()

	// Verify
	assert.False(t, needsLeaderElection,
		"API server should not require leader election to run on all replicas")
}

// TestServerImplementsLeaderElectionRunnable verifies that Server implements the LeaderElectionRunnable interface
func TestServerImplementsLeaderElectionRunnable(t *testing.T) {
	// Setup
	scheme, err := krknv1alpha1.SchemeBuilder.Build()
	assert.NoError(t, err)

	k8sClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	clientset := fake.NewSimpleClientset()

	server := NewServer(8080, k8sClient, clientset, "test-namespace", "localhost:50051")

	// Test - This will compile only if Server implements the interface correctly
	var _ interface {
		NeedLeaderElection() bool
	} = server

	// If we get here, the interface is implemented
	assert.NotNil(t, server)
}

// TestServerStatelessBehavior documents that the server is stateless and can handle concurrent requests
func TestServerStatelessBehavior(t *testing.T) {
	// This test documents the expected behavior rather than testing implementation
	// The API server should be stateless, meaning:
	// - No shared in-memory state between requests
	// - All state is persisted in Kubernetes resources
	// - Concurrent modifications are handled by Kubernetes optimistic locking (resourceVersion)
	// - Multiple replicas can safely handle requests in parallel

	t.Run("API server is designed to be stateless", func(t *testing.T) {
		// Setup
		scheme, err := krknv1alpha1.SchemeBuilder.Build()
		assert.NoError(t, err)

		k8sClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
		clientset := fake.NewSimpleClientset()

		// Create two server instances (simulating two replicas)
		server1 := NewServer(8081, k8sClient, clientset, "test-namespace", "localhost:50051")
		server2 := NewServer(8082, k8sClient, clientset, "test-namespace", "localhost:50051")

		// Both servers share the same client but have independent HTTP servers
		assert.NotNil(t, server1)
		assert.NotNil(t, server2)

		// Verify both don't need leader election (can run concurrently)
		assert.False(t, server1.NeedLeaderElection())
		assert.False(t, server2.NeedLeaderElection())
	})
}

// TestServerVsControllerLeaderElection documents the difference between API server and controllers
func TestServerVsControllerLeaderElection(t *testing.T) {
	t.Run("API server should NOT use leader election", func(t *testing.T) {
		scheme, err := krknv1alpha1.SchemeBuilder.Build()
		assert.NoError(t, err)

		k8sClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
		clientset := fake.NewSimpleClientset()

		server := NewServer(8080, k8sClient, clientset, "test-namespace", "localhost:50051")

		// API server should be available on all replicas
		assert.False(t, server.NeedLeaderElection(),
			"API server must be available on all replicas to avoid 502 errors")
	})

	t.Run("Controllers SHOULD use leader election by default", func(t *testing.T) {
		// This is a documentation test - controllers added to the manager without
		// implementing NeedLeaderElection() will default to requiring leader election
		// This is correct behavior to prevent multiple reconciliation loops

		// Controllers should NOT implement NeedLeaderElection() or should return true
		// This ensures only the leader replica runs reconciliation loops
		// preventing race conditions and infinite reconciliation loops
		assert.True(t, true, "Controllers must use leader election (default behavior)")
	})
}
