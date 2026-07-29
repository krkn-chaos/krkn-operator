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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

// TestGetScenarioReplay_Success tests successful replay with admin user
func TestGetScenarioReplay_Success(t *testing.T) {
	// Setup fake client with pod and scenarioRun
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = krknv1alpha1.AddToScheme(scheme)

	jobID := "test-job-123"
	scenarioRunName := "dummy-scenario-abc123"

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-job-" + jobID,
			Namespace: "krkn-operator-system",
			Labels: map[string]string{
				"krkn-job-id": jobID,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "krkn.krkn-chaos.dev/v1alpha1",
					Kind:       "KrknScenarioRun",
					Name:       scenarioRunName,
					UID:        "test-uid",
				},
			},
		},
	}

	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scenarioRunName,
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "target-123",
			TargetClusters: map[string][]string{
				"krkn-operator": {"cluster1"},
			},
			ScenarioName:  "dummy-scenario",
			ScenarioImage: "quay.io/krkn-chaos/krkn-hub:dummy-scenario",
			Environment: map[string]string{
				"EXIT_STATUS": "0",
			},
			Files: []krknv1alpha1.FileMount{
				{
					Name:      "config.yaml",
					Content:   "base64content",
					MountPath: "/etc/krkn/config.yaml",
					FileID:    "file-uuid-456",
				},
			},
			RegistryName: "my-registry",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod, scenarioRun).
		Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/run/replay/"+jobID, nil)

	// Add admin claims to context
	claims := &auth.Claims{
		UserID: "admin@test.com",
		Role:   "admin",
	}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Execute
	handler.GetScenarioReplay(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var payload ScenarioRunRequest
	err := json.Unmarshal(w.Body.Bytes(), &payload)
	require.NoError(t, err)

	assert.Equal(t, "target-123", payload.TargetRequestID)
	assert.Equal(t, "dummy-scenario", payload.ScenarioName)
	assert.Equal(t, "quay.io/krkn-chaos/krkn-hub:dummy-scenario", payload.ScenarioImage)
	assert.Equal(t, map[string][]string{"krkn-operator": {"cluster1"}}, payload.TargetClusters)
	assert.Equal(t, map[string]string{"EXIT_STATUS": "0"}, payload.Environment)

	// Verify fileReferences reconstructed
	require.Len(t, payload.FileReferences, 1)
	assert.Equal(t, "file-uuid-456", payload.FileReferences[0].FileID)
	assert.Equal(t, "/etc/krkn/config.yaml", payload.FileReferences[0].MountPath)

	// Verify registryName
	require.NotNil(t, payload.RegistryName)
	assert.Equal(t, "my-registry", *payload.RegistryName)
}

// TestGetScenarioReplay_JobNotFound tests 404 when job doesn't exist
func TestGetScenarioReplay_JobNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = krknv1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/run/replay/nonexistent", nil)
	claims := &auth.Claims{
		UserID: "admin@test.com",
		Role:   "admin",
	}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.GetScenarioReplay(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "not_found", errResp.Error)
}

// TestGetScenarioReplay_ScenarioRunNotFound tests 404 when ScenarioRun doesn't exist
func TestGetScenarioReplay_ScenarioRunNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = krknv1alpha1.AddToScheme(scheme)

	jobID := "test-job-123"

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-job-" + jobID,
			Namespace: "krkn-operator-system",
			Labels: map[string]string{
				"krkn-job-id": jobID,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "krkn.krkn-chaos.dev/v1alpha1",
					Kind:       "KrknScenarioRun",
					Name:       "deleted-scenario",
					UID:        "test-uid",
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod).
		Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/run/replay/"+jobID, nil)
	claims := &auth.Claims{
		UserID: "admin@test.com",
		Role: "admin",
	}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.GetScenarioReplay(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "not_found", errResp.Error)
}

// TestGetScenarioReplay_NoOwnerReference tests 400 when pod has no ScenarioRun owner
func TestGetScenarioReplay_NoOwnerReference(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = krknv1alpha1.AddToScheme(scheme)

	jobID := "test-job-123"

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-job-" + jobID,
			Namespace: "krkn-operator-system",
			Labels: map[string]string{
				"krkn-job-id": jobID,
			},
			// No ownerReferences!
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod).
		Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/run/replay/"+jobID, nil)
	claims := &auth.Claims{
		UserID: "admin@test.com",
		Role: "admin",
	}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.GetScenarioReplay(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "bad_request", errResp.Error)
}

// TestGetScenarioReplay_Unauthorized tests 401 when no auth claims
func TestGetScenarioReplay_Unauthorized(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = krknv1alpha1.AddToScheme(scheme)

	jobID := "test-job-123"
	scenarioRunName := "dummy-scenario-abc123"

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-job-" + jobID,
			Namespace: "krkn-operator-system",
			Labels: map[string]string{
				"krkn-job-id": jobID,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "krkn.krkn-chaos.dev/v1alpha1",
					Kind:       "KrknScenarioRun",
					Name:       scenarioRunName,
					UID:        "test-uid",
				},
			},
		},
	}

	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scenarioRunName,
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "target-123",
			TargetClusters: map[string][]string{
				"krkn-operator": {"cluster1"},
			},
			ScenarioName:  "dummy-scenario",
			ScenarioImage: "quay.io/krkn-chaos/krkn-hub:dummy-scenario",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod, scenarioRun).
		Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/run/replay/"+jobID, nil)
	// No claims in context!

	w := httptest.NewRecorder()

	handler.GetScenarioReplay(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var errResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", errResp.Error)
}

// TestGetScenarioReplay_WithInlineFiles tests replay with inline files (no fileId)
func TestGetScenarioReplay_WithInlineFiles(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = krknv1alpha1.AddToScheme(scheme)

	jobID := "test-job-456"
	scenarioRunName := "scenario-with-inline"

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-job-" + jobID,
			Namespace: "krkn-operator-system",
			Labels: map[string]string{
				"krkn-job-id": jobID,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "krkn.krkn-chaos.dev/v1alpha1",
					Kind:       "KrknScenarioRun",
					Name:       scenarioRunName,
					UID:        "test-uid",
				},
			},
		},
	}

	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scenarioRunName,
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "target-789",
			TargetClusters: map[string][]string{
				"krkn-operator": {"cluster2"},
			},
			ScenarioName:  "test-scenario",
			ScenarioImage: "quay.io/test:latest",
			Files: []krknv1alpha1.FileMount{
				{
					Name:      "inline.yaml",
					Content:   "inline-base64",
					MountPath: "/tmp/inline.yaml",
					// No FileID - this was an inline file
				},
				{
					Name:      "reference.yaml",
					Content:   "ref-base64",
					MountPath: "/tmp/ref.yaml",
					FileID:    "uuid-789", // This has FileID
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod, scenarioRun).
		Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/run/replay/"+jobID, nil)
	claims := &auth.Claims{
		UserID: "admin@test.com",
		Role: "admin",
	}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.GetScenarioReplay(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var payload ScenarioRunRequest
	err := json.Unmarshal(w.Body.Bytes(), &payload)
	require.NoError(t, err)

	// Should have 1 inline file
	require.Len(t, payload.Files, 1)
	assert.Equal(t, "inline.yaml", payload.Files[0].Name)
	assert.Equal(t, "inline-base64", payload.Files[0].Content)
	assert.Equal(t, "/tmp/inline.yaml", payload.Files[0].MountPath)

	// Should have 1 fileReference
	require.Len(t, payload.FileReferences, 1)
	assert.Equal(t, "uuid-789", payload.FileReferences[0].FileID)
	assert.Equal(t, "/tmp/ref.yaml", payload.FileReferences[0].MountPath)
}

// Test helper functions

func TestFindPodByJobID_MultiplePodsError(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	jobID := "duplicate-job"

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-1",
			Namespace: "krkn-operator-system",
			Labels:    map[string]string{"krkn-job-id": jobID},
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-2",
			Namespace: "krkn-operator-system",
			Labels:    map[string]string{"krkn-job-id": jobID},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod1, pod2).
		Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	_, err := handler.findPodByJobID(context.Background(), jobID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple pods found")
}

func TestExtractScenarioRunNameFromPod(t *testing.T) {
	handler := &Handler{}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "krkn.krkn-chaos.dev/v1alpha1",
					Kind:       "KrknScenarioRun",
					Name:       "scenario-run-123",
					UID:        "uid-123",
				},
			},
		},
	}

	name, err := handler.extractScenarioRunNameFromPod(pod)
	require.NoError(t, err)
	assert.Equal(t, "scenario-run-123", name)
}

func TestExtractScenarioRunNameFromPod_NoOwnerRef(t *testing.T) {
	handler := &Handler{}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
	}

	_, err := handler.extractScenarioRunNameFromPod(pod)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no KrknScenarioRun owner reference found")
}
