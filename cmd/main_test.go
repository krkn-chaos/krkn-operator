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

package main

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/krkn-chaos/krkn-operator/pkg/configstore"
	"github.com/krkn-chaos/krkn-operator/pkg/provider"
)

func TestResolveMigrationNamespacePrefersHookNamespace(t *testing.T) {
	values := map[string]string{"POD_NAMESPACE": "release-ns", "KRKN_NAMESPACE": "override-ns"}
	got := resolveMigrationNamespace(func(key string) string { return values[key] })
	if got != "release-ns" {
		t.Fatalf("namespace = %q, want POD_NAMESPACE release namespace", got)
	}
	delete(values, "POD_NAMESPACE")
	if got := resolveMigrationNamespace(func(key string) string { return values[key] }); got != "override-ns" {
		t.Fatalf("namespace = %q, want KRKN_NAMESPACE override", got)
	}
	values = map[string]string{}
	if got := resolveMigrationNamespace(func(key string) string { return values[key] }); got != "krkn-operator-system" {
		t.Fatalf("namespace = %q, want default", got)
	}
}

func TestRunCRDMigrationMode(t *testing.T) {
	markerErr := errors.New("migration failed")
	t.Run("stage mode", func(t *testing.T) {
		called := false
		handled, err := runCRDMigrationMode(true, false, "release", "", "/crds", func(namespace string) error {
			called = namespace == "release"
			return markerErr
		}, func(string, string, string) error { t.Fatal("sync callback called in stage mode"); return nil })
		if !handled || !called || !errors.Is(err, markerErr) {
			t.Fatalf("handled=%t called=%t err=%v", handled, called, err)
		}
	})
	t.Run("sync mode", func(t *testing.T) {
		called := false
		handled, err := runCRDMigrationMode(false, true, "release", "operator", "/crds", func(string) error { t.Fatal("stage callback called in sync mode"); return nil }, func(namespace, deployment, directory string) error {
			called = namespace == "release" && deployment == "operator" && directory == "/crds"
			return markerErr
		})
		if !handled || !called || !errors.Is(err, markerErr) {
			t.Fatalf("handled=%t called=%t err=%v", handled, called, err)
		}
	})
	if handled, err := runCRDMigrationMode(false, true, "release", "", "/crds", nil, nil); !handled || err == nil {
		t.Fatalf("missing deployment: handled=%t err=%v", handled, err)
	}
	if handled, err := runCRDMigrationMode(true, true, "release", "operator", "/crds", nil, nil); handled || err == nil {
		t.Fatalf("conflicting modes: handled=%t err=%v", handled, err)
	}
	if handled, err := runCRDMigrationMode(false, false, "release", "", "/crds", nil, nil); handled || err != nil {
		t.Fatalf("normal mode: handled=%t err=%v", handled, err)
	}
}

func TestConfigStoreInitializer_Start_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	// Create ConfigMap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-operator-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"API_PORT":     "8080",
			"API_ENABLED":  "true",
			"CONFIG_VALUE": "test_value",
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap).Build()

	initializer := NewConfigStoreInitializer(fakeClient, "default")

	// Clear kvstore before test
	store := kvstore.Get()
	snapshot := store.Snapshot()
	for k := range snapshot {
		store.Delete(k)
	}

	ctx := context.Background()
	err := initializer.Start(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify kvstore was populated
	value, ok := store.GetValue("API_PORT")
	if !ok {
		t.Error("Expected API_PORT to be in kvstore")
	}
	if value != "8080" {
		t.Errorf("Expected API_PORT='8080', got '%s'", value)
	}

	value, ok = store.GetValue("CONFIG_VALUE")
	if !ok {
		t.Error("Expected CONFIG_VALUE to be in kvstore")
	}
	if value != "test_value" {
		t.Errorf("Expected CONFIG_VALUE='test_value', got '%s'", value)
	}

	var updatedConfigMap corev1.ConfigMap
	if err := fakeClient.Get(ctx, client.ObjectKey{
		Name:      "krkn-operator-config",
		Namespace: "default",
	}, &updatedConfigMap); err != nil {
		t.Fatalf("Failed to get migrated ConfigMap: %v", err)
	}
	if got := updatedConfigMap.Labels[provider.ProviderConfigLabel]; got != provider.ProviderConfigLabelValue {
		t.Errorf("Expected provider configuration label %q, got %q", provider.ProviderConfigLabelValue, got)
	}

	// Clean up
	for k := range store.Snapshot() {
		store.Delete(k)
	}
}

func TestConfigStoreInitializer_Start_ConfigMapNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	// No ConfigMap created
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	initializer := NewConfigStoreInitializer(fakeClient, "default")

	// Clear kvstore before test
	store := kvstore.Get()
	snapshot := store.Snapshot()
	for k := range snapshot {
		store.Delete(k)
	}

	ctx := context.Background()
	err := initializer.Start(ctx)

	// Should not return error when ConfigMap not found
	if err != nil {
		t.Errorf("Expected no error when ConfigMap not found, got %v", err)
	}

	// kvstore should only contain pagination defaults
	snapshot = store.Snapshot()
	if val, ok := snapshot["jobs.defaultPageSize"]; !ok || val != "20" {
		t.Errorf("Expected jobs.defaultPageSize=20, got %v", snapshot)
	}
}

func TestConfigStoreInitializer_NeedLeaderElection(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	initializer := NewConfigStoreInitializer(fakeClient, "default")

	// Should return false because kvstore initialization should happen on all replicas
	if initializer.NeedLeaderElection() {
		t.Error("Expected NeedLeaderElection() to return false")
	}
}
