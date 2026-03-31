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

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func TestUpdateProviderConfigValues_CreatesNativeKeyValueFormat(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Create KrknOperatorTargetProviderConfig with schema
	config := &krknv1alpha1.KrknOperatorTargetProviderConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-uuid",
			Namespace: "default",
			Labels: map[string]string{
				"krkn.krkn-chaos.dev/uuid": "test-uuid",
			},
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderConfigSpec{
			UUID: "test-uuid",
		},
		Status: krknv1alpha1.KrknOperatorTargetProviderConfigStatus{
			Status: "Completed",
			ConfigData: map[string]krknv1alpha1.ProviderConfigData{
				"krkn-operator": {
					ConfigMap:    "krkn-operator-config",
					Namespace:    "default",
					ConfigSchema: `[{"name":"Test Field","variable":"TEST_KEY","type":"string","required":"false"}]`,
				},
			},
		},
	}

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(config).Build()
	fakeClientset := fake.NewSimpleClientset()

	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	// Create request
	reqBody := ProviderConfigUpdateRequest{
		ProviderName: "krkn-operator",
		Values: map[string]string{
			"TEST_KEY": "test_value",
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, ProviderConfigPath+"/test-uuid", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	handler.UpdateProviderConfigValues(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify ConfigMap was created with native key-value format
	var configMap corev1.ConfigMap
	err := fakeClient.Get(context.Background(), types.NamespacedName{
		Name:      "krkn-operator-config",
		Namespace: "default",
	}, &configMap)

	if err != nil {
		t.Fatalf("Failed to get ConfigMap: %v", err)
	}

	// Verify native key-value format (no "config.yaml" key)
	if _, exists := configMap.Data["config.yaml"]; exists {
		t.Error("ConfigMap should not have 'config.yaml' key in native format")
	}

	// Verify TEST_KEY exists with correct value
	value, exists := configMap.Data["TEST_KEY"]
	if !exists {
		t.Error("Expected key 'TEST_KEY' not found in ConfigMap.Data")
	}
	if value != "test_value" {
		t.Errorf("Expected value 'test_value', got '%s'", value)
	}
}

func TestUpdateProviderConfigValues_UpdatesExistingConfigMap(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Create existing ConfigMap
	existingConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-operator-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"EXISTING_KEY": "existing_value",
		},
	}

	// Create KrknOperatorTargetProviderConfig with schema
	config := &krknv1alpha1.KrknOperatorTargetProviderConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-uuid",
			Namespace: "default",
			Labels: map[string]string{
				"krkn.krkn-chaos.dev/uuid": "test-uuid",
			},
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderConfigSpec{
			UUID: "test-uuid",
		},
		Status: krknv1alpha1.KrknOperatorTargetProviderConfigStatus{
			Status: "Completed",
			ConfigData: map[string]krknv1alpha1.ProviderConfigData{
				"krkn-operator": {
					ConfigMap:    "krkn-operator-config",
					Namespace:    "default",
					ConfigSchema: `[{"name":"Test Field","variable":"TEST_KEY","type":"string","required":"false"}]`,
				},
			},
		},
	}

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).
		WithObjects(config, existingConfigMap).Build()
	fakeClientset := fake.NewSimpleClientset()

	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	// Create request with new value
	reqBody := ProviderConfigUpdateRequest{
		ProviderName: "krkn-operator",
		Values: map[string]string{
			"TEST_KEY": "test_value",
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, ProviderConfigPath+"/test-uuid", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	handler.UpdateProviderConfigValues(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify ConfigMap was updated
	var configMap corev1.ConfigMap
	err := fakeClient.Get(context.Background(), types.NamespacedName{
		Name:      "krkn-operator-config",
		Namespace: "default",
	}, &configMap)

	if err != nil {
		t.Fatalf("Failed to get ConfigMap: %v", err)
	}

	// Verify new key exists
	newValue, exists := configMap.Data["TEST_KEY"]
	if !exists {
		t.Error("Expected key 'TEST_KEY' not found after update")
	}
	if newValue != "test_value" {
		t.Errorf("Expected value 'test_value', got '%s'", newValue)
	}

	// Verify existing key is preserved (WriteConfigMapData does merge)
	existingValue, exists := configMap.Data["EXISTING_KEY"]
	if !exists {
		t.Error("Existing key 'EXISTING_KEY' should be preserved")
	}
	if existingValue != "existing_value" {
		t.Errorf("Expected existing value 'existing_value', got '%s'", existingValue)
	}

	// Verify no config.yaml key
	if _, exists := configMap.Data["config.yaml"]; exists {
		t.Error("ConfigMap should not have 'config.yaml' key in native format")
	}
}
