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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func TestPrepareConfiguration_CreatesNativeKeyValueFormat(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconciler := &ProviderConfigContributorReconciler{
		Client:            fakeClient,
		Scheme:            scheme,
		OperatorName:      "krkn-operator",
		OperatorNamespace: "default",
	}

	ctx := context.Background()

	// Call prepareConfiguration
	configMapName, _, err := reconciler.prepareConfiguration(ctx)
	if err != nil {
		t.Fatalf("prepareConfiguration failed: %v", err)
	}

	if configMapName != "krkn-operator-config" {
		t.Errorf("Expected configMapName 'krkn-operator-config', got '%s'", configMapName)
	}

	// Verify ConfigMap was created with native key-value format
	var configMap corev1.ConfigMap
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: "default",
	}, &configMap)

	if err != nil {
		t.Fatalf("Failed to get ConfigMap: %v", err)
	}

	// Verify native key-value format (no "config.yaml" key)
	if _, exists := configMap.Data["config.yaml"]; exists {
		t.Error("ConfigMap should not have 'config.yaml' key in native format")
	}

	// Verify expected keys exist
	expectedKeys := map[string]string{
		"API_PORT":                    "8080",
		"API_ENABLED":                 "true",
		"SCENARIOS_DEFAULT_TIMEOUT":   "600s",
		"PROVIDER_HEARTBEAT_INTERVAL": "30s",
	}

	for key, expectedValue := range expectedKeys {
		value, exists := configMap.Data[key]
		if !exists {
			t.Errorf("Expected key '%s' not found in ConfigMap.Data", key)
			continue
		}
		if value != expectedValue {
			t.Errorf("Key '%s': expected value '%s', got '%s'", key, expectedValue, value)
		}
	}

	// Verify no extra keys
	if len(configMap.Data) != len(expectedKeys) {
		t.Errorf("Expected %d keys, got %d. ConfigMap.Data: %v",
			len(expectedKeys), len(configMap.Data), configMap.Data)
	}
}

func TestPrepareConfiguration_UpdatesExistingConfigMap(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Create existing ConfigMap with old data
	existingConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-operator-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"OLD_KEY": "old_value",
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingConfigMap).Build()

	reconciler := &ProviderConfigContributorReconciler{
		Client:            fakeClient,
		Scheme:            scheme,
		OperatorName:      "krkn-operator",
		OperatorNamespace: "default",
	}

	ctx := context.Background()

	// Call prepareConfiguration
	_, _, err := reconciler.prepareConfiguration(ctx)
	if err != nil {
		t.Fatalf("prepareConfiguration failed: %v", err)
	}

	// Verify ConfigMap was updated
	var configMap corev1.ConfigMap
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      "krkn-operator-config",
		Namespace: "default",
	}, &configMap)

	if err != nil {
		t.Fatalf("Failed to get ConfigMap: %v", err)
	}

	// Verify new keys exist
	if _, exists := configMap.Data["API_PORT"]; !exists {
		t.Error("Expected key 'API_PORT' not found after update")
	}

	// Verify old key was overwritten
	if _, exists := configMap.Data["OLD_KEY"]; exists {
		t.Error("Old key 'OLD_KEY' should have been removed after update")
	}
}
