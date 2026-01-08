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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func TestGetClusters_Success(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

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

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(targetRequest).Build()
	handler := NewHandler(fakeClient, "default", "localhost:50051")

	// Test
	req := httptest.NewRequest("GET", "/clusters?id=test-request", nil)
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

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	handler := NewHandler(fakeClient, "default", "localhost:50051")

	// Test
	req := httptest.NewRequest("GET", "/clusters?id=non-existent", nil)
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

func TestGetClusters_NotCompleted(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-request",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{
			UUID: "test-uuid",
		},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status: "pending",
			TargetData: map[string][]krknv1alpha1.ClusterTarget{
				"operator-1": {
					{
						ClusterName:   "cluster-1",
						ClusterAPIURL: "https://api.cluster1.example.com",
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(targetRequest).Build()
	handler := NewHandler(fakeClient, "default", "localhost:50051")

	// Test
	req := httptest.NewRequest("GET", "/clusters?id=test-request", nil)
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

	expectedMessage := "KrknTargetRequest with id 'test-request' is not completed"
	if response.Message != expectedMessage {
		t.Errorf("Expected message '%s', got '%s'", expectedMessage, response.Message)
	}
}

func TestGetClusters_MissingID(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	handler := NewHandler(fakeClient, "default", "localhost:50051")

	// Test
	req := httptest.NewRequest("GET", "/clusters", nil)
	w := httptest.NewRecorder()
	handler.GetClusters(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error != "bad_request" {
		t.Errorf("Expected error 'bad_request', got '%s'", response.Error)
	}
}

func TestHealthCheck(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	handler := NewHandler(fakeClient, "default", "localhost:50051")

	// Test
	req := httptest.NewRequest("GET", "/health", nil)
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

func TestGetTargetByUUID_Success(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-uuid-123",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status: "Completed",
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(targetRequest).Build()
	handler := NewHandler(fakeClient, "default", "localhost:50051")

	// Test
	req := httptest.NewRequest("GET", "/targets/test-uuid-123", nil)
	w := httptest.NewRecorder()
	handler.GetTargetByUUID(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

func TestGetTargetByUUID_NotFound(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	handler := NewHandler(fakeClient, "default", "localhost:50051")

	// Test
	req := httptest.NewRequest("GET", "/targets/non-existent-uuid", nil)
	w := httptest.NewRecorder()
	handler.GetTargetByUUID(w, req)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGetTargetByUUID_NotCompleted(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-uuid-pending",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status: "Pending",
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(targetRequest).Build()
	handler := NewHandler(fakeClient, "default", "localhost:50051")

	// Test
	req := httptest.NewRequest("GET", "/targets/test-uuid-pending", nil)
	w := httptest.NewRecorder()
	handler.GetTargetByUUID(w, req)

	// Assert
	if w.Code != http.StatusContinue {
		t.Errorf("Expected status code %d, got %d", http.StatusContinue, w.Code)
	}
}

func TestGetTargetByUUID_MissingUUID(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	handler := NewHandler(fakeClient, "default", "localhost:50051")

	// Test - path is just /targets/ with no UUID
	req := httptest.NewRequest("GET", "/targets/", nil)
	w := httptest.NewRecorder()
	handler.GetTargetByUUID(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error != "bad_request" {
		t.Errorf("Expected error 'bad_request', got '%s'", response.Error)
	}
}

func TestPostTarget_Success(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	handler := NewHandler(fakeClient, "default", "localhost:50051")

	// Test
	req := httptest.NewRequest("POST", "/targets", nil)
	w := httptest.NewRecorder()
	handler.PostTarget(w, req)

	// Assert
	if w.Code != http.StatusProcessing {
		t.Errorf("Expected status code %d, got %d", http.StatusProcessing, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	uuid, exists := response["uuid"]
	if !exists {
		t.Error("Expected 'uuid' field in response")
	}

	if uuid == "" {
		t.Error("Expected non-empty UUID in response")
	}

	// The response is verified - UUID is returned correctly
	// The CR creation is handled by the fake client
}
