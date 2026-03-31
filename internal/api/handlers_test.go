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
	"strings"
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
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

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

	req := httptest.NewRequest("GET", ClustersPath+"?id=test-request", nil)
	w := httptest.NewRecorder()
	handler.GetClusters(w, req)

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
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("GET", ClustersPath+"?id=non-existent", nil)
	w := httptest.NewRecorder()
	handler.GetClusters(w, req)

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
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("GET", HealthPath, nil)
	w := httptest.NewRecorder()
	handler.HealthCheck(w, req)

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
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("POST", TargetsPath, nil)
	w := httptest.NewRecorder()
	handler.PostTarget(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status code %d (Processing), got %d", http.StatusAccepted, w.Code)
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
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

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

	req := httptest.NewRequest("GET", TargetsPath+"/test-uuid", nil)
	w := httptest.NewRecorder()
	handler.GetTargetByUUID(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status code %d (Processing), got %d", http.StatusAccepted, w.Code)
	}
}

func TestGetTargetByUUID_Completed(t *testing.T) {
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

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

	req := httptest.NewRequest("GET", TargetsPath+"/test-uuid", nil)
	w := httptest.NewRecorder()
	handler.GetTargetByUUID(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d (OK), got %d", http.StatusOK, w.Code)
	}
}

func TestGetTargetByUUID_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("GET", TargetsPath+"/non-existent-uuid", nil)
	w := httptest.NewRecorder()
	handler.GetTargetByUUID(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d (Not Found), got %d", http.StatusNotFound, w.Code)
	}
}

func setupScenarioRunTestHandler(targetRequestID string, clusters map[string]string) *Handler {
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	// Create managed-clusters structure
	managedClusters := map[string]map[string]map[string]string{
		"krkn-operator-acm": make(map[string]map[string]string),
	}

	for clusterName, kubeconfig := range clusters {
		managedClusters["krkn-operator-acm"][clusterName] = map[string]string{
			"kubeconfig": kubeconfig,
		}
	}

	managedClustersJSON, _ := json.Marshal(managedClusters)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      targetRequestID,
			Namespace: "default",
		},
		Data: map[string][]byte{
			"managed-clusters": managedClustersJSON,
		},
	}

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
	fakeClientset := fake.NewSimpleClientset()
	return NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")
}

func TestPostScenarioRun_SingleTarget_Success(t *testing.T) {
	targetRequestID := "test-request-id"
	clusterName := "test-cluster"
	kubeconfig := "YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmNsdXN0ZXJzOiBbXQpjb250ZXh0czogW10KdXNlcnM6IFtd"

	handler := setupScenarioRunTestHandler(targetRequestID, map[string]string{
		clusterName: kubeconfig,
	})

	// Test
	reqBody := `{
		"targetRequestID": "test-request-id",
		"targetClusters": {
			"krkn-operator": ["test-cluster"]
		},
		"scenarioImage": "quay.io/krkn/pod-scenarios:latest",
		"scenarioName": "pod-delete"
	}`

	req := httptest.NewRequest("POST", ScenariosRunPath, strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.PostScenarioRun(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response ScenarioRunCreateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.TotalTargets != 1 {
		t.Errorf("Expected TotalTargets=1, got %d", response.TotalTargets)
	}

	// TargetClusters is a map[string][]string (provider -> cluster names)
	totalClusters := 0
	for _, clusters := range response.TargetClusters {
		totalClusters += len(clusters)
	}

	if totalClusters != 1 {
		t.Fatalf("Expected 1 cluster in response, got %d", totalClusters)
	}

	// Verify cluster name exists in any provider
	foundCluster := false
	for _, clusters := range response.TargetClusters {
		for _, cluster := range clusters {
			if cluster == clusterName {
				foundCluster = true
				break
			}
		}
	}

	if !foundCluster {
		t.Errorf("Expected to find cluster '%s' in TargetClusters", clusterName)
	}

	if response.ScenarioRunName == "" {
		t.Error("Expected ScenarioRunName to be set")
	}

	if !strings.HasPrefix(response.ScenarioRunName, "pod-delete-") {
		t.Errorf("Expected ScenarioRunName to start with 'pod-delete-', got '%s'", response.ScenarioRunName)
	}
}

func TestPostScenarioRun_MissingTargetUUIDs(t *testing.T) {
	handler := setupScenarioRunTestHandler("test-id", map[string]string{})

	// Test
	reqBody := `{
		"scenarioImage": "quay.io/krkn/pod-scenarios:latest",
		"scenarioName": "pod-delete"
	}`

	req := httptest.NewRequest("POST", ScenariosRunPath, strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.PostScenarioRun(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error != "bad_request" {
		t.Errorf("Expected error='bad_request', got '%s'", response.Error)
	}
}

func TestPostScenarioRun_MultipleTargets_AllSuccess(t *testing.T) {
	kubeconfig := "YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmNsdXN0ZXJzOiBbXQpjb250ZXh0czogW10KdXNlcnM6IFtd"

	handler := setupScenarioRunTestHandler("test-request-id", map[string]string{
		"cluster-1": kubeconfig,
		"cluster-2": kubeconfig,
		"cluster-3": kubeconfig,
	})

	// Test
	reqBody := `{
		"targetRequestID": "test-request-id",
		"targetClusters": {
			"krkn-operator": ["cluster-1", "cluster-2", "cluster-3"]
		},
		"scenarioImage": "quay.io/krkn/pod-scenarios:latest",
		"scenarioName": "pod-delete"
	}`

	req := httptest.NewRequest("POST", ScenariosRunPath, strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.PostScenarioRun(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response ScenarioRunCreateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.TotalTargets != 3 {
		t.Errorf("Expected TotalTargets=3, got %d", response.TotalTargets)
	}

	// Count total clusters across all providers
	totalClusters := 0
	for _, clusters := range response.TargetClusters {
		totalClusters += len(clusters)
	}

	if totalClusters != 3 {
		t.Fatalf("Expected 3 clusters in response, got %d", totalClusters)
	}

	if response.ScenarioRunName == "" {
		t.Error("Expected ScenarioRunName to be set")
	}
}

func TestPostScenarioRun_MultipleTargets_PartialFailure(t *testing.T) {
	kubeconfig := "YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmNsdXN0ZXJzOiBbXQpjb250ZXh0czogW10KdXNlcnM6IFtd"

	handler := setupScenarioRunTestHandler("test-request-id", map[string]string{
		"cluster-1": kubeconfig,
		"cluster-2": kubeconfig,
		// "invalid" cluster is intentionally not included
	})

	reqBody := `{
		"targetRequestID": "test-request-id",
		"targetClusters": {
			"krkn-operator": ["cluster-1", "invalid", "cluster-2"]
		},
		"scenarioImage": "quay.io/krkn/pod-scenarios:latest",
		"scenarioName": "pod-delete"
	}`

	req := httptest.NewRequest("POST", ScenariosRunPath, strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.PostScenarioRun(w, req)

	// Note: With CRD-based approach, the CR is created successfully even if some clusters are invalid.
	// The controller will handle the failures when reconciling.
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}

	var response ScenarioRunCreateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.TotalTargets != 3 {
		t.Errorf("Expected TotalTargets=3, got %d", response.TotalTargets)
	}

	// Count total clusters across all providers
	totalClusters := 0
	for _, clusters := range response.TargetClusters {
		totalClusters += len(clusters)
	}

	if totalClusters != 3 {
		t.Fatalf("Expected 3 clusters in response, got %d", totalClusters)
	}

	if response.ScenarioRunName == "" {
		t.Error("Expected ScenarioRunName to be set")
	}
}

func TestPostScenarioRun_MultipleTargets_AllFailure(t *testing.T) {
	// Note: With CRD-based approach, the CR is created successfully even with invalid clusters.
	// The controller will handle the failures when reconciling.
	// Empty cluster map - all requests will fail at reconciliation time
	handler := setupScenarioRunTestHandler("test-request-id", map[string]string{})

	// Test
	reqBody := `{
		"targetRequestID": "test-request-id",
		"targetClusters": {
			"krkn-operator": ["invalid-1", "invalid-2"]
		},
		"scenarioImage": "quay.io/krkn/pod-scenarios:latest",
		"scenarioName": "pod-delete"
	}`

	req := httptest.NewRequest("POST", ScenariosRunPath, strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.PostScenarioRun(w, req)

	// CR creation succeeds even with invalid clusters
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}

	var response ScenarioRunCreateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.TotalTargets != 2 {
		t.Errorf("Expected TotalTargets=2, got %d", response.TotalTargets)
	}

	if response.ScenarioRunName == "" {
		t.Error("Expected ScenarioRunName to be set")
	}
}

func TestPostScenarioRun_Validation_ClusterNames(t *testing.T) {
	tests := []struct {
		name        string
		reqBody     string
		expectedErr string
	}{
		{
			name:        "Empty array",
			reqBody:     `{"targetRequestID": "test-id", "targetClusters": {"krkn-operator": []}, "scenarioImage": "img", "scenarioName": "test"}`,
			expectedErr: "provider 'krkn-operator' must have at least one cluster",
		},
		{
			name:        "Duplicates",
			reqBody:     `{"targetRequestID": "test-id", "targetClusters": {"krkn-operator": ["cluster1", "cluster1"]}, "scenarioImage": "img", "scenarioName": "test"}`,
			expectedErr: "cluster 'cluster1' appears in multiple providers",
		},
		{
			name:        "Empty string",
			reqBody:     `{"targetRequestID": "test-id", "targetClusters": {"krkn-operator": ["cluster1", ""]}, "scenarioImage": "img", "scenarioName": "test"}`,
			expectedErr: "cluster names cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := setupScenarioRunTestHandler("test-id", map[string]string{})

			req := httptest.NewRequest("POST", ScenariosRunPath, strings.NewReader(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler.PostScenarioRun(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
			}

			var response ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if !strings.Contains(response.Message, tt.expectedErr) {
				t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedErr, response.Message)
			}
		})
	}
}

func TestListScenarioRuns_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	scenarioRun1 := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run-1",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			ScenarioName: "pod-delete",
			TargetClusters: map[string][]string{
				"krkn-operator": {"cluster-1"},
			},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase:        "Running",
			TotalTargets: 1,
		},
	}

	scenarioRun2 := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run-2",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			ScenarioName: "node-drain",
			TargetClusters: map[string][]string{
				"krkn-operator": {"cluster-2"},
			},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase:        "Succeeded",
			TotalTargets: 1,
		},
	}

	scenarioRun3 := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run-3",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			ScenarioName: "pod-delete",
			TargetClusters: map[string][]string{
				"krkn-operator": {"cluster-3"},
			},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase:        "Failed",
			TotalTargets: 1,
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(scenarioRun1, scenarioRun2, scenarioRun3).
		Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("GET", ScenariosRunPath, nil)
	w := httptest.NewRecorder()
	handler.ListScenarioRuns(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response ScenarioRunListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.ScenarioRuns) != 3 {
		t.Errorf("Expected 3 scenario runs, got %d", len(response.ScenarioRuns))
	}

	// Verify scenario names are populated
	for _, run := range response.ScenarioRuns {
		if run.ScenarioName == "" {
			t.Errorf("Expected ScenarioName to be set for run %s", run.ScenarioRunName)
		}
	}
}

func TestListScenarioRuns_FilterByScenarioName(t *testing.T) {
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	scenarioRun1 := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run-1",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			ScenarioName: "pod-delete",
			TargetClusters: map[string][]string{
				"krkn-operator": {"cluster-1"},
			},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase:        "Running",
			TotalTargets: 1,
		},
	}

	scenarioRun2 := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run-2",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			ScenarioName: "node-drain",
			TargetClusters: map[string][]string{
				"krkn-operator": {"cluster-2"},
			},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase:        "Running",
			TotalTargets: 1,
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(scenarioRun1, scenarioRun2).
		Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("GET", ScenariosRunPath+"?scenarioName=pod-delete", nil)
	w := httptest.NewRecorder()
	handler.ListScenarioRuns(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response ScenarioRunListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.ScenarioRuns) != 1 {
		t.Errorf("Expected 1 scenario run, got %d", len(response.ScenarioRuns))
	}

	if response.ScenarioRuns[0].ScenarioName != "pod-delete" {
		t.Errorf("Expected ScenarioName='pod-delete', got '%s'", response.ScenarioRuns[0].ScenarioName)
	}
}

// NOTE: Tests for deleteTargetRequest were removed - KrknTargetRequest is now owned by ScenarioRun
// and will be automatically deleted via Kubernetes garbage collection when ScenarioRun is deleted.
