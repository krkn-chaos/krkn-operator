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

package provider

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

const (
	testNamespace    = "krkn-operator-system"
	testProviderName = "test-operator"
)

func setupTestClient(objs ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(&krknv1alpha1.KrknOperatorTargetProvider{}).
		Build()
}

func TestEnsureProvider_CreatesNewProvider(t *testing.T) {
	fakeClient := setupTestClient()
	providerReg := NewProviderRegistrationWithConfig(fakeClient, Config{
		ProviderName:      testProviderName,
		HeartbeatInterval: 30 * time.Second,
		Namespace:         testNamespace,
	})
	ctx := context.Background()

	err := providerReg.ensureProvider(ctx)
	if err != nil {
		t.Fatalf("ensureProvider failed: %v", err)
	}

	// Verify provider was created
	var provider krknv1alpha1.KrknOperatorTargetProvider
	if err := fakeClient.Get(ctx, types.NamespacedName{
		Name:      testProviderName,
		Namespace: testNamespace,
	}, &provider); err != nil {
		t.Fatalf("Failed to get provider: %v", err)
	}

	if provider.Spec.OperatorName != testProviderName {
		t.Errorf("Expected operator name %s, got %s", testProviderName, provider.Spec.OperatorName)
	}

	if !provider.Spec.Active {
		t.Error("Expected provider to be active")
	}
}

func TestEnsureProvider_UpdatesExistingProvider(t *testing.T) {
	existingProvider := &krknv1alpha1.KrknOperatorTargetProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testProviderName,
			Namespace: testNamespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderSpec{
			OperatorName: testProviderName,
			Active:       false, // Inactive
		},
	}

	fakeClient := setupTestClient(existingProvider)
	providerReg := NewProviderRegistrationWithConfig(fakeClient, Config{
		ProviderName:      testProviderName,
		HeartbeatInterval: 30 * time.Second,
		Namespace:         testNamespace,
	})
	ctx := context.Background()

	err := providerReg.ensureProvider(ctx)
	if err != nil {
		t.Fatalf("ensureProvider failed: %v", err)
	}

	// Verify provider was updated to active
	var provider krknv1alpha1.KrknOperatorTargetProvider
	if err := fakeClient.Get(ctx, types.NamespacedName{
		Name:      testProviderName,
		Namespace: testNamespace,
	}, &provider); err != nil {
		t.Fatalf("Failed to get provider: %v", err)
	}

	if !provider.Spec.Active {
		t.Error("Expected provider to be updated to active")
	}
}

func TestUpdateHeartbeat_UpdatesTimestamp(t *testing.T) {
	// Use a timestamp from 1 minute ago to ensure difference
	oldTime := metav1.NewTime(metav1.Now().Add(-1 * 60 * 1000000000)) // 1 minute ago
	provider := &krknv1alpha1.KrknOperatorTargetProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testProviderName,
			Namespace: testNamespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderSpec{
			OperatorName: testProviderName,
			Active:       true,
		},
		Status: krknv1alpha1.KrknOperatorTargetProviderStatus{
			Timestamp: oldTime,
		},
	}

	fakeClient := setupTestClient(provider)
	providerReg := NewProviderRegistrationWithConfig(fakeClient, Config{
		ProviderName:      testProviderName,
		HeartbeatInterval: 30 * time.Second,
		Namespace:         testNamespace,
	})
	ctx := context.Background()

	err := providerReg.updateHeartbeat(ctx)
	if err != nil {
		t.Fatalf("updateHeartbeat failed: %v", err)
	}

	// Verify timestamp was updated
	var updated krknv1alpha1.KrknOperatorTargetProvider
	if err := fakeClient.Get(ctx, types.NamespacedName{
		Name:      testProviderName,
		Namespace: testNamespace,
	}, &updated); err != nil {
		t.Fatalf("Failed to get provider: %v", err)
	}

	if updated.Status.Timestamp.Time.Equal(oldTime.Time) {
		t.Error("Expected timestamp to be updated")
	}

	if !updated.Status.Timestamp.Time.After(oldTime.Time) {
		t.Errorf("Expected timestamp to be newer than old timestamp (old: %v, new: %v)",
			oldTime.Time, updated.Status.Timestamp.Time)
	}
}

func TestDeactivateProvider_SetsActiveToFalse(t *testing.T) {
	provider := &krknv1alpha1.KrknOperatorTargetProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testProviderName,
			Namespace: testNamespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderSpec{
			OperatorName: testProviderName,
			Active:       true,
		},
	}

	fakeClient := setupTestClient(provider)
	providerReg := NewProviderRegistrationWithConfig(fakeClient, Config{
		ProviderName:      testProviderName,
		HeartbeatInterval: 30 * time.Second,
		Namespace:         testNamespace,
	})
	ctx := context.Background()

	err := providerReg.deactivateProvider(ctx)
	if err != nil {
		t.Fatalf("deactivateProvider failed: %v", err)
	}

	// Verify provider was deactivated
	var updated krknv1alpha1.KrknOperatorTargetProvider
	if err := fakeClient.Get(ctx, types.NamespacedName{
		Name:      testProviderName,
		Namespace: testNamespace,
	}, &updated); err != nil {
		t.Fatalf("Failed to get provider: %v", err)
	}

	if updated.Spec.Active {
		t.Error("Expected provider to be deactivated")
	}
}

func TestDeactivateProvider_HandlesNotFound(t *testing.T) {
	fakeClient := setupTestClient()
	providerReg := NewProviderRegistrationWithConfig(fakeClient, Config{
		ProviderName:      testProviderName,
		HeartbeatInterval: 30 * time.Second,
		Namespace:         testNamespace,
	})
	ctx := context.Background()

	// Should not error if provider doesn't exist
	err := providerReg.deactivateProvider(ctx)
	if err != nil {
		t.Fatalf("deactivateProvider should not fail when provider not found: %v", err)
	}
}

func TestNeedLeaderElection_ReturnsTrue(t *testing.T) {
	fakeClient := setupTestClient()
	providerReg := NewProviderRegistrationWithConfig(fakeClient, Config{
		ProviderName:      testProviderName,
		HeartbeatInterval: 30 * time.Second,
		Namespace:         testNamespace,
	})

	if !providerReg.NeedLeaderElection() {
		t.Error("Expected NeedLeaderElection to return true")
	}
}

func TestNewProviderRegistration_BackwardsCompatibility(t *testing.T) {
	fakeClient := setupTestClient()
	providerReg := NewProviderRegistration(fakeClient, testNamespace)

	// Verify defaults are set correctly
	if providerReg.providerName != "krkn-operator" {
		t.Errorf("Expected default provider name 'krkn-operator', got %s", providerReg.providerName)
	}

	if providerReg.heartbeatInterval != 30*time.Second {
		t.Errorf("Expected default heartbeat interval 30s, got %v", providerReg.heartbeatInterval)
	}

	if providerReg.namespace != testNamespace {
		t.Errorf("Expected namespace %s, got %s", testNamespace, providerReg.namespace)
	}
}

func TestNewProviderRegistrationWithConfig_CustomValues(t *testing.T) {
	fakeClient := setupTestClient()
	customInterval := 60 * time.Second
	providerReg := NewProviderRegistrationWithConfig(fakeClient, Config{
		ProviderName:      "custom-operator",
		HeartbeatInterval: customInterval,
		Namespace:         testNamespace,
	})

	if providerReg.providerName != "custom-operator" {
		t.Errorf("Expected provider name 'custom-operator', got %s", providerReg.providerName)
	}

	if providerReg.heartbeatInterval != customInterval {
		t.Errorf("Expected heartbeat interval %v, got %v", customInterval, providerReg.heartbeatInterval)
	}
}

func TestNewProviderRegistrationWithConfig_DefaultValues(t *testing.T) {
	fakeClient := setupTestClient()
	providerReg := NewProviderRegistrationWithConfig(fakeClient, Config{
		Namespace: testNamespace,
		// ProviderName and HeartbeatInterval not set
	})

	if providerReg.providerName != "krkn-operator" {
		t.Errorf("Expected default provider name 'krkn-operator', got %s", providerReg.providerName)
	}

	if providerReg.heartbeatInterval != 30*time.Second {
		t.Errorf("Expected default heartbeat interval 30s, got %v", providerReg.heartbeatInterval)
	}
}
