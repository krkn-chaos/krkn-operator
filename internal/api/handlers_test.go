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

	req := httptest.NewRequest("GET", "/api/v1/clusters?id=test-request", nil)
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

	req := httptest.NewRequest("GET", "/api/v1/clusters?id=non-existent", nil)
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

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
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

	req := httptest.NewRequest("POST", "/api/v1/targets", nil)
	w := httptest.NewRecorder()
	handler.PostTarget(w, req)

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

	req := httptest.NewRequest("GET", "/api/v1/targets/test-uuid", nil)
	w := httptest.NewRecorder()
	handler.GetTargetByUUID(w, req)

	if w.Code != http.StatusContinue {
		t.Errorf("Expected status code %d (Continue), got %d", http.StatusContinue, w.Code)
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

	req := httptest.NewRequest("GET", "/api/v1/targets/test-uuid", nil)
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

	req := httptest.NewRequest("GET", "/api/v1/targets/non-existent-uuid", nil)
	w := httptest.NewRecorder()
	handler.GetTargetByUUID(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d (Not Found), got %d", http.StatusNotFound, w.Code)
	}
}

func setupScenarioRunTestHandler(targets ...*krknv1alpha1.KrknOperatorTarget) *Handler {
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	objects := make([]client.Object, 0, len(targets)*2)
	secretDataJSON := `{"kubeconfig":"YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmNsdXN0ZXJzOiBbXQpjb250ZXh0czogW10KdXNlcnM6IFtd"}`

	for _, target := range targets {
		objects = append(objects, target)
		objects = append(objects, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      target.Spec.SecretUUID,
				Namespace: "default",
			},
			Data: map[string][]byte{
				"kubeconfig": []byte(secretDataJSON),
			},
		})
	}

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
	fakeClientset := fake.NewSimpleClientset()
	return NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")
}

func TestPostScenarioRun_SingleTarget_Success(t *testing.T) {
	target := &krknv1alpha1.KrknOperatorTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "single-uuid",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknOperatorTargetSpec{
			ClusterName: "test-cluster",
			SecretType:  "kubeconfig",
			SecretUUID:   "single-uuid-kubeconfig",
		},
		Status: krknv1alpha1.KrknOperatorTargetStatus{
			Ready: true,
		},
	}

	handler := setupScenarioRunTestHandler(target)

	// Test
	reqBody := `{
		"targetUUIDs": ["single-uuid"],
		"scenarioImage": "quay.io/krkn/pod-scenarios:latest",
		"scenarioName": "pod-delete"
	}`

	req := httptest.NewRequest("POST", "/api/v1/scenarios/run", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.PostScenarioRun(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response ScenarioRunResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.TotalTargets != 1 {
		t.Errorf("Expected TotalTargets=1, got %d", response.TotalTargets)
	}

	if response.SuccessfulJobs != 1 {
		t.Errorf("Expected SuccessfulJobs=1, got %d", response.SuccessfulJobs)
	}

	if response.FailedJobs != 0 {
		t.Errorf("Expected FailedJobs=0, got %d", response.FailedJobs)
	}

	if len(response.Jobs) != 1 {
		t.Fatalf("Expected 1 job in response, got %d", len(response.Jobs))
	}

	job := response.Jobs[0]
	if job.TargetUUID != "single-uuid" {
		t.Errorf("Expected TargetUUID='single-uuid', got '%s'", job.TargetUUID)
	}

	if !job.Success {
		t.Errorf("Expected Success=true, got false. Error: %s", job.Error)
	}

	if job.JobId == "" {
		t.Error("Expected JobId to be set")
	}

	if job.PodName == "" {
		t.Error("Expected PodName to be set")
	}

	if job.Status != "Pending" {
		t.Errorf("Expected Status='Pending', got '%s'", job.Status)
	}
}

func TestPostScenarioRun_MissingTargetUUIDs(t *testing.T) {
	handler := setupScenarioRunTestHandler()

	// Test
	reqBody := `{
		"scenarioImage": "quay.io/krkn/pod-scenarios:latest",
		"scenarioName": "pod-delete"
	}`

	req := httptest.NewRequest("POST", "/api/v1/scenarios/run", strings.NewReader(reqBody))
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
	target1 := &krknv1alpha1.KrknOperatorTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "uuid1",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknOperatorTargetSpec{
			ClusterName: "cluster-1",
			SecretType:  "kubeconfig",
			SecretUUID:   "uuid1-kubeconfig",
		},
		Status: krknv1alpha1.KrknOperatorTargetStatus{
			Ready: true,
		},
	}

	target2 := &krknv1alpha1.KrknOperatorTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "uuid2",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknOperatorTargetSpec{
			ClusterName: "cluster-2",
			SecretType:  "kubeconfig",
			SecretUUID:   "uuid2-kubeconfig",
		},
		Status: krknv1alpha1.KrknOperatorTargetStatus{
			Ready: true,
		},
	}

	target3 := &krknv1alpha1.KrknOperatorTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "uuid3",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknOperatorTargetSpec{
			ClusterName: "cluster-3",
			SecretType:  "kubeconfig",
			SecretUUID:   "uuid3-kubeconfig",
		},
		Status: krknv1alpha1.KrknOperatorTargetStatus{
			Ready: true,
		},
	}

	handler := setupScenarioRunTestHandler(target1, target2, target3)

	// Test
	reqBody := `{
		"targetUUIDs": ["uuid1", "uuid2", "uuid3"],
		"scenarioImage": "quay.io/krkn/pod-scenarios:latest",
		"scenarioName": "pod-delete"
	}`

	req := httptest.NewRequest("POST", "/api/v1/scenarios/run", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.PostScenarioRun(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response ScenarioRunResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.TotalTargets != 3 {
		t.Errorf("Expected TotalTargets=3, got %d", response.TotalTargets)
	}

	if response.SuccessfulJobs != 3 {
		t.Errorf("Expected SuccessfulJobs=3, got %d", response.SuccessfulJobs)
	}

	if response.FailedJobs != 0 {
		t.Errorf("Expected FailedJobs=0, got %d", response.FailedJobs)
	}

	if len(response.Jobs) != 3 {
		t.Fatalf("Expected 3 jobs in response, got %d", len(response.Jobs))
	}

	for i, job := range response.Jobs {
		if !job.Success {
			t.Errorf("Job %d failed: %s", i, job.Error)
		}
	}
}

func TestPostScenarioRun_MultipleTargets_PartialFailure(t *testing.T) {
	target1 := &krknv1alpha1.KrknOperatorTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "valid-1",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknOperatorTargetSpec{
			ClusterName: "cluster-1",
			SecretType:  "kubeconfig",
			SecretUUID:   "valid-1-kubeconfig",
		},
		Status: krknv1alpha1.KrknOperatorTargetStatus{
			Ready: true,
		},
	}

	target2 := &krknv1alpha1.KrknOperatorTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "valid-2",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknOperatorTargetSpec{
			ClusterName: "cluster-2",
			SecretType:  "kubeconfig",
			SecretUUID:   "valid-2-kubeconfig",
		},
		Status: krknv1alpha1.KrknOperatorTargetStatus{
			Ready: true,
		},
	}

	handler := setupScenarioRunTestHandler(target1, target2)

	reqBody := `{
		"targetUUIDs": ["valid-1", "invalid", "valid-2"],
		"scenarioImage": "quay.io/krkn/pod-scenarios:latest",
		"scenarioName": "pod-delete"
	}`

	req := httptest.NewRequest("POST", "/api/v1/scenarios/run", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.PostScenarioRun(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}

	var response ScenarioRunResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.TotalTargets != 3 {
		t.Errorf("Expected TotalTargets=3, got %d", response.TotalTargets)
	}

	if response.SuccessfulJobs != 2 {
		t.Errorf("Expected SuccessfulJobs=2, got %d", response.SuccessfulJobs)
	}

	if response.FailedJobs != 1 {
		t.Errorf("Expected FailedJobs=1, got %d", response.FailedJobs)
	}

	if response.Jobs[1].Success {
		t.Error("Expected jobs[1] (invalid) to fail")
	}

	if response.Jobs[1].TargetUUID != "invalid" {
		t.Errorf("Expected jobs[1].TargetUUID='invalid', got '%s'", response.Jobs[1].TargetUUID)
	}

	if !response.Jobs[0].Success {
		t.Errorf("Expected jobs[0] to succeed, got error: %s", response.Jobs[0].Error)
	}

	if !response.Jobs[2].Success {
		t.Errorf("Expected jobs[2] to succeed, got error: %s", response.Jobs[2].Error)
	}
}

func TestPostScenarioRun_MultipleTargets_AllFailure(t *testing.T) {
	handler := setupScenarioRunTestHandler()

	// Test
	reqBody := `{
		"targetUUIDs": ["invalid-1", "invalid-2"],
		"scenarioImage": "quay.io/krkn/pod-scenarios:latest",
		"scenarioName": "pod-delete"
	}`

	req := httptest.NewRequest("POST", "/api/v1/scenarios/run", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.PostScenarioRun(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var response ScenarioRunResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.SuccessfulJobs != 0 {
		t.Errorf("Expected SuccessfulJobs=0, got %d", response.SuccessfulJobs)
	}

	if response.FailedJobs != 2 {
		t.Errorf("Expected FailedJobs=2, got %d", response.FailedJobs)
	}
}

func TestPostScenarioRun_Validation_TargetUUIDs(t *testing.T) {
	tests := []struct {
		name        string
		reqBody     string
		expectedErr string
	}{
		{
			name:        "Empty array",
			reqBody:     `{"targetUUIDs": [], "scenarioImage": "img", "scenarioName": "test"}`,
			expectedErr: "targetUUIDs is required and must contain at least one UUID",
		},
		{
			name:        "Duplicates",
			reqBody:     `{"targetUUIDs": ["uuid1", "uuid1"], "scenarioImage": "img", "scenarioName": "test"}`,
			expectedErr: "targetUUIDs contains duplicates",
		},
		{
			name:        "Empty string",
			reqBody:     `{"targetUUIDs": ["uuid1", ""], "scenarioImage": "img", "scenarioName": "test"}`,
			expectedErr: "targetUUIDs cannot contain empty strings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := setupScenarioRunTestHandler()

			req := httptest.NewRequest("POST", "/api/v1/scenarios/run", strings.NewReader(tt.reqBody))
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

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-job-1",
			Namespace: "default",
			Labels: map[string]string{
				"app":                "krkn-scenario",
				"krkn-job-id":        "job-1",
				"krkn-target-uuid":   "uuid1",
				"krkn-scenario-name": "pod-delete",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-job-2",
			Namespace: "default",
			Labels: map[string]string{
				"app":                "krkn-scenario",
				"krkn-job-id":        "job-2",
				"krkn-target-uuid":   "uuid2",
				"krkn-scenario-name": "node-drain",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	pod3 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-job-3",
			Namespace: "default",
			Labels: map[string]string{
				"app":                "krkn-scenario",
				"krkn-job-id":        "job-3",
				"krkn-target-uuid":   "uuid3",
				"krkn-scenario-name": "pod-delete",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodSucceeded,
		},
	}

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(pod1, pod2, pod3).Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("GET", "/api/v1/scenarios/run", nil)
	w := httptest.NewRecorder()
	handler.ListScenarioRuns(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response JobsListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.Jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(response.Jobs))
	}

	// Verify targetUUID is populated
	for _, job := range response.Jobs {
		if job.TargetUUID == "" {
			t.Errorf("Expected TargetUUID to be set for job %s", job.JobId)
		}
	}
}

func TestListScenarioRuns_FilterByTargetUUID(t *testing.T) {
	scheme := runtime.NewScheme()
	krknv1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-job-1",
			Namespace: "default",
			Labels: map[string]string{
				"app":                "krkn-scenario",
				"krkn-job-id":        "job-1",
				"krkn-target-uuid":   "uuid1",
				"krkn-scenario-name": "pod-delete",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-job-2",
			Namespace: "default",
			Labels: map[string]string{
				"app":                "krkn-scenario",
				"krkn-job-id":        "job-2",
				"krkn-target-uuid":   "uuid2",
				"krkn-scenario-name": "node-drain",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(pod1, pod2).Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("GET", "/api/v1/scenarios/run?targetUUID=uuid1", nil)
	w := httptest.NewRecorder()
	handler.ListScenarioRuns(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response JobsListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.Jobs) != 1 {
		t.Errorf("Expected 1 job, got %d", len(response.Jobs))
	}

	if response.Jobs[0].TargetUUID != "uuid1" {
		t.Errorf("Expected TargetUUID='uuid1', got '%s'", response.Jobs[0].TargetUUID)
	}
}
