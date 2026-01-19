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
)

func TestGetClusters_Success(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-request",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{
			UUID: "test-uuid",
		},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status: "Completed",
			TargetData: map[string][]krknv1alpha1.ClusterTarget{
				"operator-1": {
					{
						ClusterName:   "cluster-1",
						ClusterAPIURL: "https://api.cluster1.example.com",
					},
					{
						ClusterName:   "cluster-2",
						ClusterAPIURL: "https://api.cluster2.example.com",
					},
				},
			},
		},
	}

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(targetRequest).Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	// Test
	req := httptest.NewRequest("GET", "/api/v1/clusters?id=test-request", nil)
	w := httptest.NewRecorder()
	handler.GetClusters(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response ClustersResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "Completed" {
		t.Errorf("Expected status 'Completed', got '%s'", response.Status)
	}

	if len(response.TargetData) != 1 {
		t.Errorf("Expected 1 operator in TargetData, got %d", len(response.TargetData))
	}

	if len(response.TargetData["operator-1"]) != 2 {
		t.Errorf("Expected 2 clusters for operator-1, got %d", len(response.TargetData["operator-1"]))
	}
}

func TestGetClusters_NotFound(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	// Test
	req := httptest.NewRequest("GET", "/api/v1/clusters?id=non-existent", nil)
	w := httptest.NewRecorder()
	handler.GetClusters(w, req)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error != "not_found" {
		t.Errorf("Expected error 'not_found', got '%s'", response.Error)
	}
}

func TestHealthCheck(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	// Test
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	handler.HealthCheck(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response["status"])
	}
}

func TestPostTarget_LegacyEndpoint(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	// Test
	req := httptest.NewRequest("POST", "/api/v1/targets", nil)
	w := httptest.NewRecorder()
	handler.PostTarget(w, req)

	// Assert - should return 102 Processing
	if w.Code != http.StatusProcessing {
		t.Errorf("Expected status code %d (Processing), got %d", http.StatusProcessing, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["uuid"] == "" {
		t.Error("Expected uuid in response, got empty string")
	}

	// Verify that KrknTargetRequest CR was created
	var targetRequest krknv1alpha1.KrknTargetRequest
	err := fakeClient.Get(req.Context(), client.ObjectKey{
		Name:      response["uuid"],
		Namespace: "default",
	}, &targetRequest)

	if err != nil {
		t.Errorf("Failed to get created KrknTargetRequest: %v", err)
	}

	if targetRequest.Spec.UUID != response["uuid"] {
		t.Errorf("Expected UUID '%s', got '%s'", response["uuid"], targetRequest.Spec.UUID)
	}
}

func TestGetTargetByUUID_NotCompleted(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-uuid",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{
			UUID: "test-uuid",
		},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status: "Pending",
		},
	}

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(targetRequest).Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	// Test
	req := httptest.NewRequest("GET", "/api/v1/targets/test-uuid", nil)
	w := httptest.NewRecorder()
	handler.GetTargetByUUID(w, req)

	// Assert - should return 100 Continue
	if w.Code != http.StatusContinue {
		t.Errorf("Expected status code %d (Continue), got %d", http.StatusContinue, w.Code)
	}
}

func TestGetTargetByUUID_Completed(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-uuid",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{
			UUID: "test-uuid",
		},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status: "Completed",
		},
	}

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(targetRequest).Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	// Test
	req := httptest.NewRequest("GET", "/api/v1/targets/test-uuid", nil)
	w := httptest.NewRecorder()
	handler.GetTargetByUUID(w, req)

	// Assert - should return 200 OK
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d (OK), got %d", http.StatusOK, w.Code)
	}
}

func TestGetTargetByUUID_NotFound(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	// Test
	req := httptest.NewRequest("GET", "/api/v1/targets/non-existent-uuid", nil)
	w := httptest.NewRecorder()
	handler.GetTargetByUUID(w, req)

	// Assert - should return 404 Not Found
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d (Not Found), got %d", http.StatusNotFound, w.Code)
	}
}
