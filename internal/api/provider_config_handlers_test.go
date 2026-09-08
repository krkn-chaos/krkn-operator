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
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// providerConfigTestScheme builds a runtime scheme registered with the types the
// provider-config handlers operate on. Extracted to avoid duplicating the same
// scheme setup across every handler test.
func providerConfigTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add krkn scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add corev1 scheme: %v", err)
	}
	return scheme
}

// decodeErrorResponse unmarshals a sanitized ErrorResponse from a recorder body,
// failing the test if the payload is not valid JSON.
func decodeErrorResponse(t *testing.T, body []byte) ErrorResponse {
	t.Helper()
	var resp ErrorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to decode error response %q: %v", string(body), err)
	}
	return resp
}

func TestPostProviderConfig_ReturnsAcceptedWithUUID(t *testing.T) {
	scheme := providerConfigTestScheme(t)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodPost, ProviderConfigPath, nil)
	w := httptest.NewRecorder()

	handler.PostProviderConfig(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusAccepted, w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	uuid, ok := resp["uuid"]
	if !ok || uuid == "" {
		t.Errorf("expected a non-empty uuid in response, got %v", resp)
	}

	// The CR must actually have been created with the returned UUID as its name.
	var config krknv1alpha1.KrknOperatorTargetProviderConfig
	if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: uuid, Namespace: "default"}, &config); err != nil {
		t.Errorf("expected provider config CR %q to be created: %v", uuid, err)
	}
}

func TestPostProviderConfig_FailureReturnsSanitizedError(t *testing.T) {
	scheme := providerConfigTestScheme(t)
	internalErr := errors.New("etcdserver: leader changed - secret internal detail")
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				return internalErr
			},
		}).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodPost, ProviderConfigPath, nil)
	w := httptest.NewRecorder()

	handler.PostProviderConfig(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusInternalServerError, w.Code, w.Body.String())
	}

	body := w.Body.String()
	if strings.Contains(body, "etcdserver") || strings.Contains(body, "secret internal detail") {
		t.Errorf("response leaked internal error details: %s", body)
	}
	resp := decodeErrorResponse(t, w.Body.Bytes())
	if resp.Message != "Failed to create KrknOperatorTargetProviderConfig" {
		t.Errorf("expected generic message, got %q", resp.Message)
	}
}

func TestGetProviderConfigByUUID_NotFoundReturns404(t *testing.T) {
	scheme := providerConfigTestScheme(t)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, ProviderConfigPath+"/missing-uuid", nil)
	w := httptest.NewRecorder()

	handler.GetProviderConfigByUUID(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
	resp := decodeErrorResponse(t, w.Body.Bytes())
	if resp.Error != "not_found" {
		t.Errorf("expected error 'not_found', got %q", resp.Error)
	}
}

func TestGetProviderConfigByUUID_InternalErrorReturnsSanitized500(t *testing.T) {
	scheme := providerConfigTestScheme(t)
	internalErr := errors.New("connection refused to 10.0.0.5:6443 - secret internal detail")
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return internalErr
			},
		}).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, ProviderConfigPath+"/some-uuid", nil)
	w := httptest.NewRecorder()

	handler.GetProviderConfigByUUID(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusInternalServerError, w.Code, w.Body.String())
	}

	body := w.Body.String()
	if strings.Contains(body, "connection refused") || strings.Contains(body, "secret internal detail") {
		t.Errorf("response leaked internal error details: %s", body)
	}
	resp := decodeErrorResponse(t, w.Body.Bytes())
	if resp.Message != "Failed to fetch KrknOperatorTargetProviderConfig" {
		t.Errorf("expected generic message, got %q", resp.Message)
	}
}

func TestGetProviderConfigByUUID_PendingReturnsAccepted(t *testing.T) {
	scheme := providerConfigTestScheme(t)
	config := &krknv1alpha1.KrknOperatorTargetProviderConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "pending-uuid", Namespace: "default"},
		Spec:       krknv1alpha1.KrknOperatorTargetProviderConfigSpec{UUID: "pending-uuid"},
		Status:     krknv1alpha1.KrknOperatorTargetProviderConfigStatus{Status: "Pending"},
	}
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(config).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, ProviderConfigPath+"/pending-uuid", nil)
	w := httptest.NewRecorder()

	handler.GetProviderConfigByUUID(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusAccepted, w.Code, w.Body.String())
	}
}

func TestGetProviderConfigByUUID_CompletedReturnsConfigData(t *testing.T) {
	scheme := providerConfigTestScheme(t)
	config := &krknv1alpha1.KrknOperatorTargetProviderConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "done-uuid", Namespace: "default"},
		Spec:       krknv1alpha1.KrknOperatorTargetProviderConfigSpec{UUID: "done-uuid"},
		Status: krknv1alpha1.KrknOperatorTargetProviderConfigStatus{
			Status: "Completed",
			ConfigData: map[string]krknv1alpha1.ProviderConfigData{
				"krkn-operator": {ConfigMap: "krkn-operator-config", Namespace: "default"},
			},
		},
	}
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(config).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, ProviderConfigPath+"/done-uuid", nil)
	w := httptest.NewRecorder()

	handler.GetProviderConfigByUUID(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["uuid"] != "done-uuid" {
		t.Errorf("expected uuid 'done-uuid', got %v", resp["uuid"])
	}
	if resp["status"] != "Completed" {
		t.Errorf("expected status 'Completed', got %v", resp["status"])
	}
	if _, ok := resp["config_data"]; !ok {
		t.Errorf("expected config_data in response, got %v", resp)
	}
}

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

	handler := NewTestHandler(fakeClient, fakeClientset, "default", "localhost:50051")

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

	handler := NewTestHandler(fakeClient, fakeClientset, "default", "localhost:50051")

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
