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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/kubeconfig"
)

// setupTestHandler creates a test Handler with fake clients
func setupTestHandler() *Handler {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	fakeClientset := fake.NewSimpleClientset()

	return &Handler{
		client:         fakeClient,
		clientset:      fakeClientset,
		namespace:      "test-namespace",
		grpcServerAddr: "localhost:50051",
	}
}

func TestCreateTarget_WithKubeconfig(t *testing.T) {
	handler := setupTestHandler()

	// Generate a valid kubeconfig for testing
	validKubeconfig, err := kubeconfig.GenerateFromToken(
		"test-cluster",
		"https://api.test.com:6443",
		"",
		"test-token",
		true,
	)
	if err != nil {
		t.Fatalf("Failed to generate test kubeconfig: %v", err)
	}

	reqBody := CreateTargetRequest{
		ClusterName: "test-cluster",
		SecretType:  "kubeconfig",
		Kubeconfig:  validKubeconfig,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/operator/targets", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.CreateTarget(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response CreateTargetResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.UUID == "" {
		t.Error("Expected UUID in response, got empty string")
	}

	// Verify that KrknOperatorTarget CR was created
	var target krknv1alpha1.KrknOperatorTarget
	err = handler.client.Get(req.Context(), client.ObjectKey{
		Name:      response.UUID,
		Namespace: handler.namespace,
	}, &target)

	if err != nil {
		t.Errorf("Failed to get created target: %v", err)
	}

	if target.Spec.ClusterName != "test-cluster" {
		t.Errorf("Expected cluster name 'test-cluster', got '%s'", target.Spec.ClusterName)
	}

	if target.Spec.SecretType != "kubeconfig" {
		t.Errorf("Expected secret type 'kubeconfig', got '%s'", target.Spec.SecretType)
	}

	// Verify that Secret was created
	var secret corev1.Secret
	err = handler.client.Get(req.Context(), client.ObjectKey{
		Name:      target.Spec.SecretUUID,
		Namespace: handler.namespace,
	}, &secret)

	if err != nil {
		t.Errorf("Failed to get created secret: %v", err)
	}

	if secret.Data["kubeconfig"] == nil {
		t.Error("Expected kubeconfig in secret data")
	}
}

func TestCreateTarget_WithToken(t *testing.T) {
	handler := setupTestHandler()

	reqBody := CreateTargetRequest{
		ClusterName:   "test-cluster",
		SecretType:    "token",
		ClusterAPIURL: "https://api.test.com:6443",
		Token:         "test-token-123",
		CABundle:      "",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/operator/targets", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.CreateTarget(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response CreateTargetResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify that KrknOperatorTarget was created with InsecureSkipTLSVerify=true
	var target krknv1alpha1.KrknOperatorTarget
	err := handler.client.Get(req.Context(), client.ObjectKey{
		Name:      response.UUID,
		Namespace: handler.namespace,
	}, &target)

	if err != nil {
		t.Errorf("Failed to get created target: %v", err)
	}

	if !target.Spec.InsecureSkipTLSVerify {
		t.Error("Expected InsecureSkipTLSVerify to be true when CABundle is empty")
	}

	if target.Spec.ClusterAPIURL != "https://api.test.com:6443" {
		t.Errorf("Expected API URL 'https://api.test.com:6443', got '%s'", target.Spec.ClusterAPIURL)
	}
}

func TestCreateTarget_WithCredentials(t *testing.T) {
	handler := setupTestHandler()

	reqBody := CreateTargetRequest{
		ClusterName:   "test-cluster",
		SecretType:    "credentials",
		ClusterAPIURL: "https://api.test.com:6443",
		Username:      "admin",
		Password:      "secret123",
		CABundle:      "LS0tLS1CRUdJTi...",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/operator/targets", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.CreateTarget(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response CreateTargetResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify that target was created with CABundle
	var target krknv1alpha1.KrknOperatorTarget
	err := handler.client.Get(req.Context(), client.ObjectKey{
		Name:      response.UUID,
		Namespace: handler.namespace,
	}, &target)

	if err != nil {
		t.Errorf("Failed to get created target: %v", err)
	}

	if target.Spec.CABundle != "LS0tLS1CRUdJTi..." {
		t.Error("Expected CABundle to be set")
	}

	if target.Spec.InsecureSkipTLSVerify {
		t.Error("Expected InsecureSkipTLSVerify to be false when CABundle is provided")
	}
}

func TestCreateTarget_MissingRequiredFields(t *testing.T) {
	handler := setupTestHandler()

	tests := []struct {
		name       string
		reqBody    CreateTargetRequest
		wantStatus int
		wantError  string
	}{
		{
			name: "missing cluster name",
			reqBody: CreateTargetRequest{
				SecretType: "kubeconfig",
				Kubeconfig: "test",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "clusterName is required",
		},
		{
			name: "missing secret type",
			reqBody: CreateTargetRequest{
				ClusterName: "test-cluster",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "secretType is required",
		},
		{
			name: "invalid secret type",
			reqBody: CreateTargetRequest{
				ClusterName: "test-cluster",
				SecretType:  "invalid",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "secretType must be one of",
		},
		{
			name: "missing kubeconfig for kubeconfig type",
			reqBody: CreateTargetRequest{
				ClusterName: "test-cluster",
				SecretType:  "kubeconfig",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "kubeconfig is required",
		},
		{
			name: "missing token for token type",
			reqBody: CreateTargetRequest{
				ClusterName:   "test-cluster",
				SecretType:    "token",
				ClusterAPIURL: "https://api.test.com:6443",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "token is required",
		},
		{
			name: "missing clusterAPIURL for token type",
			reqBody: CreateTargetRequest{
				ClusterName: "test-cluster",
				SecretType:  "token",
				Token:       "test-token",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "clusterAPIURL is required",
		},
		{
			name: "missing username for credentials type",
			reqBody: CreateTargetRequest{
				ClusterName:   "test-cluster",
				SecretType:    "credentials",
				ClusterAPIURL: "https://api.test.com:6443",
				Password:      "secret",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "username and password are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/operator/targets", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.CreateTarget(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}

			var errResp ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("Failed to unmarshal error response: %v", err)
			}

			if errResp.Message == "" || !contains(errResp.Message, tt.wantError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.wantError, errResp.Message)
			}
		})
	}
}

func TestCreateTarget_DuplicateClusterName(t *testing.T) {
	handler := setupTestHandler()

	// Create first target
	validKubeconfig, _ := kubeconfig.GenerateFromToken("test-cluster", "https://api.test.com:6443", "", "token", true)

	existingTarget := &krknv1alpha1.KrknOperatorTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-uuid",
			Namespace: handler.namespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetSpec{
			UUID:          "existing-uuid",
			ClusterName:   "test-cluster",
			ClusterAPIURL: "https://api.test.com:6443",
			SecretType:    "kubeconfig",
			SecretUUID:    "secret-uuid",
		},
	}

	if err := handler.client.Create(nil, existingTarget); err != nil {
		t.Fatalf("Failed to create existing target: %v", err)
	}

	// Try to create another target with the same cluster name
	reqBody := CreateTargetRequest{
		ClusterName: "test-cluster",
		SecretType:  "kubeconfig",
		Kubeconfig:  validKubeconfig,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/operator/targets", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.CreateTarget(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
	}

	var errResp ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)

	if !contains(errResp.Message, "already exists") {
		t.Errorf("Expected 'already exists' error, got '%s'", errResp.Message)
	}
}

func TestListTargets(t *testing.T) {
	handler := setupTestHandler()

	// Create some test targets
	targets := []krknv1alpha1.KrknOperatorTarget{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "target-1",
				Namespace: handler.namespace,
			},
			Spec: krknv1alpha1.KrknOperatorTargetSpec{
				UUID:          "target-1",
				ClusterName:   "cluster-1",
				ClusterAPIURL: "https://api1.test.com:6443",
				SecretType:    "kubeconfig",
			},
			Status: krknv1alpha1.KrknOperatorTargetStatus{
				Ready: true,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "target-2",
				Namespace: handler.namespace,
			},
			Spec: krknv1alpha1.KrknOperatorTargetSpec{
				UUID:          "target-2",
				ClusterName:   "cluster-2",
				ClusterAPIURL: "https://api2.test.com:6443",
				SecretType:    "token",
			},
			Status: krknv1alpha1.KrknOperatorTargetStatus{
				Ready: true,
			},
		},
	}

	for _, target := range targets {
		if err := handler.client.Create(nil, &target); err != nil {
			t.Fatalf("Failed to create test target: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/operator/targets", nil)
	w := httptest.NewRecorder()

	handler.ListTargets(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response ListTargetsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.Targets) != 2 {
		t.Errorf("Expected 2 targets, got %d", len(response.Targets))
	}

	// Verify target fields
	foundCluster1 := false
	for _, target := range response.Targets {
		if target.ClusterName == "cluster-1" {
			foundCluster1 = true
			if target.ClusterAPIURL != "https://api1.test.com:6443" {
				t.Errorf("Expected API URL 'https://api1.test.com:6443', got '%s'", target.ClusterAPIURL)
			}
			if !target.Ready {
				t.Error("Expected target to be ready")
			}
		}
	}

	if !foundCluster1 {
		t.Error("Expected to find cluster-1 in response")
	}
}

func TestGetTarget(t *testing.T) {
	handler := setupTestHandler()

	// Create a test target
	target := &krknv1alpha1.KrknOperatorTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-uuid",
			Namespace: handler.namespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetSpec{
			UUID:          "test-uuid",
			ClusterName:   "test-cluster",
			ClusterAPIURL: "https://api.test.com:6443",
			SecretType:    "kubeconfig",
		},
		Status: krknv1alpha1.KrknOperatorTargetStatus{
			Ready: true,
		},
	}

	if err := handler.client.Create(nil, target); err != nil {
		t.Fatalf("Failed to create test target: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/operator/targets/test-uuid", nil)
	w := httptest.NewRecorder()

	handler.GetTarget(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response TargetResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.UUID != "test-uuid" {
		t.Errorf("Expected UUID 'test-uuid', got '%s'", response.UUID)
	}

	if response.ClusterName != "test-cluster" {
		t.Errorf("Expected cluster name 'test-cluster', got '%s'", response.ClusterName)
	}
}

func TestGetTarget_NotFound(t *testing.T) {
	handler := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/operator/targets/non-existent-uuid", nil)
	w := httptest.NewRecorder()

	handler.GetTarget(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDeleteTarget(t *testing.T) {
	handler := setupTestHandler()

	// Create target and secret
	secretUUID := "secret-uuid"
	targetUUID := "target-uuid"

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretUUID,
			Namespace: handler.namespace,
		},
		Data: map[string][]byte{
			"kubeconfig": []byte(`{"kubeconfig":"test"}`),
		},
	}

	target := &krknv1alpha1.KrknOperatorTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      targetUUID,
			Namespace: handler.namespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetSpec{
			UUID:          targetUUID,
			ClusterName:   "test-cluster",
			ClusterAPIURL: "https://api.test.com:6443",
			SecretType:    "kubeconfig",
			SecretUUID:    secretUUID,
		},
	}

	if err := handler.client.Create(nil, secret); err != nil {
		t.Fatalf("Failed to create secret: %v", err)
	}

	if err := handler.client.Create(nil, target); err != nil {
		t.Fatalf("Failed to create target: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/operator/targets/"+targetUUID, nil)
	w := httptest.NewRecorder()

	handler.DeleteTarget(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify target was deleted
	var deletedTarget krknv1alpha1.KrknOperatorTarget
	err := handler.client.Get(nil, client.ObjectKey{
		Name:      targetUUID,
		Namespace: handler.namespace,
	}, &deletedTarget)

	if err == nil {
		t.Error("Expected target to be deleted, but it still exists")
	}

	// Verify secret was deleted
	var deletedSecret corev1.Secret
	err = handler.client.Get(nil, client.ObjectKey{
		Name:      secretUUID,
		Namespace: handler.namespace,
	}, &deletedSecret)

	if err == nil {
		t.Error("Expected secret to be deleted, but it still exists")
	}
}

func TestUpdateTarget(t *testing.T) {
	handler := setupTestHandler()

	// Create initial target and secret
	secretUUID := "secret-uuid"
	targetUUID := "target-uuid"

	kubeconfigData, _ := kubeconfig.MarshalSecretData("initial-kubeconfig")
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretUUID,
			Namespace: handler.namespace,
		},
		Data: map[string][]byte{
			"kubeconfig": kubeconfigData,
		},
	}

	target := &krknv1alpha1.KrknOperatorTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      targetUUID,
			Namespace: handler.namespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetSpec{
			UUID:          targetUUID,
			ClusterName:   "test-cluster",
			ClusterAPIURL: "https://api.test.com:6443",
			SecretType:    "kubeconfig",
			SecretUUID:    secretUUID,
		},
	}

	if err := handler.client.Create(nil, secret); err != nil {
		t.Fatalf("Failed to create secret: %v", err)
	}

	if err := handler.client.Create(nil, target); err != nil {
		t.Fatalf("Failed to create target: %v", err)
	}

	// Update with new token-based auth
	updateReq := UpdateTargetRequest{
		CreateTargetRequest: CreateTargetRequest{
			ClusterName:   "updated-cluster",
			SecretType:    "token",
			ClusterAPIURL: "https://api.updated.com:6443",
			Token:         "new-token",
		},
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/operator/targets/"+targetUUID, bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.UpdateTarget(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify target was updated
	var updatedTarget krknv1alpha1.KrknOperatorTarget
	err := handler.client.Get(nil, client.ObjectKey{
		Name:      targetUUID,
		Namespace: handler.namespace,
	}, &updatedTarget)

	if err != nil {
		t.Fatalf("Failed to get updated target: %v", err)
	}

	if updatedTarget.Spec.ClusterName != "updated-cluster" {
		t.Errorf("Expected cluster name 'updated-cluster', got '%s'", updatedTarget.Spec.ClusterName)
	}

	if updatedTarget.Spec.SecretType != "token" {
		t.Errorf("Expected secret type 'token', got '%s'", updatedTarget.Spec.SecretType)
	}

	if updatedTarget.Spec.ClusterAPIURL != "https://api.updated.com:6443" {
		t.Errorf("Expected API URL 'https://api.updated.com:6443', got '%s'", updatedTarget.Spec.ClusterAPIURL)
	}

	// Verify secret was updated
	var updatedSecret corev1.Secret
	err = handler.client.Get(nil, client.ObjectKey{
		Name:      secretUUID,
		Namespace: handler.namespace,
	}, &updatedSecret)

	if err != nil {
		t.Fatalf("Failed to get updated secret: %v", err)
	}

	// Extract and verify kubeconfig was regenerated
	newKubeconfig, err := kubeconfig.UnmarshalSecretData(updatedSecret.Data["kubeconfig"])
	if err != nil {
		t.Fatalf("Failed to unmarshal updated kubeconfig: %v", err)
	}

	if newKubeconfig == "initial-kubeconfig" {
		t.Error("Expected kubeconfig to be updated, but it's still the initial value")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
