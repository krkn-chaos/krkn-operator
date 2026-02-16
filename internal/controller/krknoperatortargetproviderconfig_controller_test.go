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

package controller

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

const (
	testConfigName = "test-config"
	testConfigUUID = "test-config-uuid-123"
)

func setupTestConfigReconciler(objs ...client.Object) *KrknOperatorTargetProviderConfigReconciler {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(&krknv1alpha1.KrknOperatorTargetProviderConfig{}, &krknv1alpha1.KrknOperatorTargetProvider{}).
		Build()

	return &KrknOperatorTargetProviderConfigReconciler{
		Client:            fakeClient,
		Scheme:            scheme,
		OperatorNamespace: testOperatorNamespace,
	}
}

func TestConfigReconcile_SetsUUIDLabel(t *testing.T) {
	config := &krknv1alpha1.KrknOperatorTargetProviderConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigName,
			Namespace: testOperatorNamespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderConfigSpec{
			UUID: testConfigUUID,
		},
	}

	reconciler := setupTestConfigReconciler(config)
	ctx := context.Background()

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testConfigName,
			Namespace: testOperatorNamespace,
		},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify UUID label was set
	var updated krknv1alpha1.KrknOperatorTargetProviderConfig
	if err := reconciler.Get(ctx, types.NamespacedName{
		Name:      testConfigName,
		Namespace: testOperatorNamespace,
	}, &updated); err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	if label, exists := updated.Labels["krkn.krkn-chaos.dev/uuid"]; !exists || label != testConfigUUID {
		t.Errorf("Expected UUID label to be %s, got %s (exists: %v)", testConfigUUID, label, exists)
	}
}

func TestConfigReconcile_InitializesStatus(t *testing.T) {
	config := &krknv1alpha1.KrknOperatorTargetProviderConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigName,
			Namespace: testOperatorNamespace,
			Labels: map[string]string{
				"krkn.krkn-chaos.dev/uuid": testConfigUUID,
			},
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderConfigSpec{
			UUID: testConfigUUID,
		},
	}

	reconciler := setupTestConfigReconciler(config)
	ctx := context.Background()

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testConfigName,
			Namespace: testOperatorNamespace,
		},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify status was initialized
	var updated krknv1alpha1.KrknOperatorTargetProviderConfig
	if err := reconciler.Get(ctx, types.NamespacedName{
		Name:      testConfigName,
		Namespace: testOperatorNamespace,
	}, &updated); err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	if updated.Status.Status != "pending" {
		t.Errorf("Expected status to be 'pending', got %s", updated.Status.Status)
	}

	if updated.Status.Created == nil {
		t.Error("Expected Created timestamp to be set")
	}

	// ConfigData map is initialized lazily when first provider contributes
	// so we don't strictly require it to be non-nil here
}

func TestConfigReconcile_MarksCompleted(t *testing.T) {
	config := &krknv1alpha1.KrknOperatorTargetProviderConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigName,
			Namespace: testOperatorNamespace,
			Labels: map[string]string{
				"krkn.krkn-chaos.dev/uuid": testConfigUUID,
			},
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderConfigSpec{
			UUID: testConfigUUID,
		},
		Status: krknv1alpha1.KrknOperatorTargetProviderConfigStatus{
			Status:  "pending",
			Created: &metav1.Time{Time: time.Now()},
			ConfigData: map[string]krknv1alpha1.ProviderConfigData{
				"krkn-operator": {
					ConfigMap:    "krkn-operator-config",
					ConfigSchema: `{"type": "object"}`,
				},
			},
		},
	}

	provider := &krknv1alpha1.KrknOperatorTargetProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testOperatorName,
			Namespace: testOperatorNamespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderSpec{
			OperatorName: testOperatorName,
			Active:       true,
		},
	}

	reconciler := setupTestConfigReconciler(config, provider)
	ctx := context.Background()

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testConfigName,
			Namespace: testOperatorNamespace,
		},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify request was marked as completed
	var updated krknv1alpha1.KrknOperatorTargetProviderConfig
	if err := reconciler.Get(ctx, types.NamespacedName{
		Name:      testConfigName,
		Namespace: testOperatorNamespace,
	}, &updated); err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	if updated.Status.Status != "Completed" {
		t.Errorf("Expected status to be 'Completed', got %s", updated.Status.Status)
	}

	if updated.Status.Completed == nil {
		t.Error("Expected Completed timestamp to be set")
	}
}

func TestConfigReconcile_WaitsForAllProviders(t *testing.T) {
	config := &krknv1alpha1.KrknOperatorTargetProviderConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigName,
			Namespace: testOperatorNamespace,
			Labels: map[string]string{
				"krkn.krkn-chaos.dev/uuid": testConfigUUID,
			},
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderConfigSpec{
			UUID: testConfigUUID,
		},
		Status: krknv1alpha1.KrknOperatorTargetProviderConfigStatus{
			Status:  "pending",
			Created: &metav1.Time{Time: time.Now()},
			ConfigData: map[string]krknv1alpha1.ProviderConfigData{
				"krkn-operator": {
					ConfigMap:    "krkn-operator-config",
					ConfigSchema: `{"type": "object"}`,
				},
			},
		},
	}

	provider1 := &krknv1alpha1.KrknOperatorTargetProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-operator",
			Namespace: testOperatorNamespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderSpec{
			OperatorName: "krkn-operator",
			Active:       true,
		},
	}

	provider2 := &krknv1alpha1.KrknOperatorTargetProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-operator-acm",
			Namespace: testOperatorNamespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderSpec{
			OperatorName: "krkn-operator-acm",
			Active:       true,
		},
	}

	reconciler := setupTestConfigReconciler(config, provider1, provider2)
	ctx := context.Background()

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testConfigName,
			Namespace: testOperatorNamespace,
		},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify request is NOT completed (waiting for second provider)
	var updated krknv1alpha1.KrknOperatorTargetProviderConfig
	if err := reconciler.Get(ctx, types.NamespacedName{
		Name:      testConfigName,
		Namespace: testOperatorNamespace,
	}, &updated); err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	if updated.Status.Status != "pending" {
		t.Errorf("Expected status to remain 'pending', got %s", updated.Status.Status)
	}

	// Now add second provider's config data
	updated.Status.ConfigData["krkn-operator-acm"] = krknv1alpha1.ProviderConfigData{
		ConfigMap:    "acm-config",
		ConfigSchema: `{"type": "object"}`,
	}
	if err := reconciler.Status().Update(ctx, &updated); err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// Reconcile again
	_, err = reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testConfigName,
			Namespace: testOperatorNamespace,
		},
	})

	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	// Now verify it's completed
	if err := reconciler.Get(ctx, types.NamespacedName{
		Name:      testConfigName,
		Namespace: testOperatorNamespace,
	}, &updated); err != nil {
		t.Fatalf("Failed to get config after second reconcile: %v", err)
	}

	if updated.Status.Status != "Completed" {
		t.Errorf("Expected status to be 'Completed', got %s", updated.Status.Status)
	}
}

func TestConfigReconcile_HandlesEmptySchema(t *testing.T) {
	config := &krknv1alpha1.KrknOperatorTargetProviderConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigName,
			Namespace: testOperatorNamespace,
			Labels: map[string]string{
				"krkn.krkn-chaos.dev/uuid": testConfigUUID,
			},
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderConfigSpec{
			UUID: testConfigUUID,
		},
		Status: krknv1alpha1.KrknOperatorTargetProviderConfigStatus{
			Status:  "pending",
			Created: &metav1.Time{Time: time.Now()},
			ConfigData: map[string]krknv1alpha1.ProviderConfigData{
				"krkn-operator": {
					ConfigMap:    "krkn-operator-config",
					ConfigSchema: "", // Empty schema is valid
				},
			},
		},
	}

	provider := &krknv1alpha1.KrknOperatorTargetProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testOperatorName,
			Namespace: testOperatorNamespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderSpec{
			OperatorName: testOperatorName,
			Active:       true,
		},
	}

	reconciler := setupTestConfigReconciler(config, provider)
	ctx := context.Background()

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testConfigName,
			Namespace: testOperatorNamespace,
		},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify request was marked as completed even with empty schema
	var updated krknv1alpha1.KrknOperatorTargetProviderConfig
	if err := reconciler.Get(ctx, types.NamespacedName{
		Name:      testConfigName,
		Namespace: testOperatorNamespace,
	}, &updated); err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	if updated.Status.Status != "Completed" {
		t.Errorf("Expected status to be 'Completed', got %s", updated.Status.Status)
	}
}

func TestConfigReconcile_SkipsInactiveProviders(t *testing.T) {
	config := &krknv1alpha1.KrknOperatorTargetProviderConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigName,
			Namespace: testOperatorNamespace,
			Labels: map[string]string{
				"krkn.krkn-chaos.dev/uuid": testConfigUUID,
			},
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderConfigSpec{
			UUID: testConfigUUID,
		},
		Status: krknv1alpha1.KrknOperatorTargetProviderConfigStatus{
			Status:  "pending",
			Created: &metav1.Time{Time: time.Now()},
			ConfigData: map[string]krknv1alpha1.ProviderConfigData{
				"krkn-operator": {
					ConfigMap:    "krkn-operator-config",
					ConfigSchema: `{"type": "object"}`,
				},
			},
		},
	}

	activeProvider := &krknv1alpha1.KrknOperatorTargetProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-operator",
			Namespace: testOperatorNamespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderSpec{
			OperatorName: "krkn-operator",
			Active:       true,
		},
	}

	inactiveProvider := &krknv1alpha1.KrknOperatorTargetProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-operator-inactive",
			Namespace: testOperatorNamespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderSpec{
			OperatorName: "krkn-operator-inactive",
			Active:       false, // Inactive
		},
	}

	reconciler := setupTestConfigReconciler(config, activeProvider, inactiveProvider)
	ctx := context.Background()

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testConfigName,
			Namespace: testOperatorNamespace,
		},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify request was marked as completed (inactive provider ignored)
	var updated krknv1alpha1.KrknOperatorTargetProviderConfig
	if err := reconciler.Get(ctx, types.NamespacedName{
		Name:      testConfigName,
		Namespace: testOperatorNamespace,
	}, &updated); err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	if updated.Status.Status != "Completed" {
		t.Errorf("Expected status to be 'Completed' (inactive provider ignored), got %s", updated.Status.Status)
	}
}

func TestConfigReconcile_SkipsAlreadyCompleted(t *testing.T) {
	now := metav1.Now()
	config := &krknv1alpha1.KrknOperatorTargetProviderConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testConfigName,
			Namespace: testOperatorNamespace,
			Labels: map[string]string{
				"krkn.krkn-chaos.dev/uuid": testConfigUUID,
			},
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderConfigSpec{
			UUID: testConfigUUID,
		},
		Status: krknv1alpha1.KrknOperatorTargetProviderConfigStatus{
			Status:    "Completed",
			Created:   &now,
			Completed: &now,
			ConfigData: map[string]krknv1alpha1.ProviderConfigData{
				"krkn-operator": {
					ConfigMap:    "krkn-operator-config",
					ConfigSchema: `{"type": "object"}`,
				},
			},
		},
	}

	reconciler := setupTestConfigReconciler(config)
	ctx := context.Background()

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testConfigName,
			Namespace: testOperatorNamespace,
		},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify request was not modified
	var updated krknv1alpha1.KrknOperatorTargetProviderConfig
	if err := reconciler.Get(ctx, types.NamespacedName{
		Name:      testConfigName,
		Namespace: testOperatorNamespace,
	}, &updated); err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	if updated.Status.Status != "Completed" {
		t.Errorf("Expected status to remain 'Completed', got %s", updated.Status.Status)
	}
}
