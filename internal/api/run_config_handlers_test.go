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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

// --- ScenarioRun Config Tests ---

func TestGetScenarioRunConfig_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))

	scenarioRunName := "my-scenario-run"

	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scenarioRunName,
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "target-123",
			TargetClusters: map[string][]string{
				"krkn-operator": {"cluster1", "cluster2"},
			},
			ScenarioName:  "dummy-scenario",
			ScenarioImage: "quay.io/krkn-chaos/krkn-hub:dummy-scenario",
			Environment: map[string]string{
				"EXIT_STATUS": "0",
			},
			KubeconfigPath: "/home/krkn/.kube/config",
			Files: []krknv1alpha1.FileMount{
				{
					Name:      "config.yaml",
					Content:   "base64content",
					MountPath: "/etc/krkn/config.yaml",
					FileID:    "file-uuid-001",
				},
			},
			RegistryName: "my-registry",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(scenarioRun).
		Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/run/"+scenarioRunName+"/config", nil)
	claims := &auth.Claims{
		UserID: "admin@test.com",
		Role:   "admin",
	}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetScenarioRunConfig(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var payload ScenarioRunRequest
	err := json.Unmarshal(w.Body.Bytes(), &payload)
	require.NoError(t, err)

	assert.Equal(t, "target-123", payload.TargetRequestID)
	assert.Equal(t, "dummy-scenario", payload.ScenarioName)
	assert.Equal(t, "quay.io/krkn-chaos/krkn-hub:dummy-scenario", payload.ScenarioImage)
	assert.Equal(t, map[string][]string{"krkn-operator": {"cluster1", "cluster2"}}, payload.TargetClusters)
	assert.Equal(t, map[string]string{"EXIT_STATUS": "0"}, payload.Environment)
	assert.Equal(t, "/home/krkn/.kube/config", payload.KubeconfigPath)

	require.Len(t, payload.FileReferences, 1)
	assert.Equal(t, "file-uuid-001", payload.FileReferences[0].FileID)
	assert.Equal(t, "/etc/krkn/config.yaml", payload.FileReferences[0].MountPath)

	require.NotNil(t, payload.RegistryName)
	assert.Equal(t, "my-registry", *payload.RegistryName)
}

func TestGetScenarioRunConfig_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/run/nonexistent/config", nil)
	claims := &auth.Claims{
		UserID: "admin@test.com",
		Role:   "admin",
	}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetScenarioRunConfig(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "not_found", errResp.Error)
}

func TestGetScenarioRunConfig_Unauthorized(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))

	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-run",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "target-123",
			TargetClusters:  map[string][]string{"krkn-operator": {"cluster1"}},
			ScenarioName:    "dummy-scenario",
			ScenarioImage:   "quay.io/test:latest",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(scenarioRun).
		Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/run/test-run/config", nil)

	w := httptest.NewRecorder()
	handler.GetScenarioRunConfig(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var errResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", errResp.Error)
}

func TestGetScenarioRunConfig_WithInlineAndRefFiles(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))

	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mixed-files-run",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "target-456",
			TargetClusters:  map[string][]string{"krkn-operator": {"cluster1"}},
			ScenarioName:    "test-scenario",
			ScenarioImage:   "quay.io/test:latest",
			Files: []krknv1alpha1.FileMount{
				{
					Name:      "inline.yaml",
					Content:   "inline-content",
					MountPath: "/tmp/inline.yaml",
				},
				{
					Name:      "ref.yaml",
					Content:   "ref-content",
					MountPath: "/tmp/ref.yaml",
					FileID:    "uuid-ref-001",
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(scenarioRun).
		Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/run/mixed-files-run/config", nil)
	claims := &auth.Claims{
		UserID: "admin@test.com",
		Role:   "admin",
	}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetScenarioRunConfig(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var payload ScenarioRunRequest
	err := json.Unmarshal(w.Body.Bytes(), &payload)
	require.NoError(t, err)

	require.Len(t, payload.Files, 1)
	assert.Equal(t, "inline.yaml", payload.Files[0].Name)

	require.Len(t, payload.FileReferences, 1)
	assert.Equal(t, "uuid-ref-001", payload.FileReferences[0].FileID)
}

// --- GraphRun Config Tests ---

func TestGetGraphRunConfig_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))

	graphRunName := "my-graph-run"

	graphRun := &krknv1alpha1.KrknGraphRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      graphRunName,
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknGraphRunSpec{
			Graph: map[string]krknv1alpha1.GraphScenarioNode{
				"node-1": {
					Name:  "scenario-a",
					Image: "quay.io/krkn-chaos/krkn-hub:scenario-a",
					Env:   map[string]string{"EXIT_STATUS": "0"},
				},
				"node-2": {
					Name:      "scenario-b",
					Image:     "quay.io/krkn-chaos/krkn-hub:scenario-b",
					DependsOn: strPtr("node-1"),
				},
			},
			TargetRequestID: "target-graph-123",
			TargetClusters: map[string][]string{
				"krkn-operator": {"cluster1"},
			},
			OwnerUserID: "owner@test.com",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(graphRun).
		Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/graphruns/"+graphRunName+"/config", nil)
	claims := &auth.Claims{
		UserID: "admin@test.com",
		Role:   "admin",
	}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetGraphRunConfig(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var payload GraphRunCreateRequest
	err := json.Unmarshal(w.Body.Bytes(), &payload)
	require.NoError(t, err)

	assert.Equal(t, "target-graph-123", payload.TargetRequestID)
	assert.Equal(t, map[string][]string{"krkn-operator": {"cluster1"}}, payload.TargetClusters)

	require.Len(t, payload.Graph, 2)
	assert.Equal(t, "scenario-a", payload.Graph["node-1"].Name)
	assert.Equal(t, "scenario-b", payload.Graph["node-2"].Name)
	assert.Equal(t, "node-1", *payload.Graph["node-2"].DependsOn)
}

func TestGetGraphRunConfig_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/graphruns/nonexistent/config", nil)
	claims := &auth.Claims{
		UserID: "admin@test.com",
		Role:   "admin",
	}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetGraphRunConfig(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "not_found", errResp.Error)
}

func TestGetGraphRunConfig_Unauthorized(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))

	graphRun := &krknv1alpha1.KrknGraphRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-graph",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknGraphRunSpec{
			Graph: map[string]krknv1alpha1.GraphScenarioNode{
				"node-1": {Name: "s1", Image: "img:1"},
			},
			TargetRequestID: "target-123",
			TargetClusters:  map[string][]string{"krkn-operator": {"cluster1"}},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(graphRun).
		Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/graphruns/test-graph/config", nil)

	w := httptest.NewRecorder()
	handler.GetGraphRunConfig(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var errResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", errResp.Error)
}

// --- Router integration tests (v1) ---

func TestScenariosRunRouter_ConfigEndpoint(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))

	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "router-test-run",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "target-rt",
			TargetClusters:  map[string][]string{"krkn-operator": {"c1"}},
			ScenarioName:    "scenario-rt",
			ScenarioImage:   "quay.io/test:rt",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(scenarioRun).
		Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/run/router-test-run/config", nil)
	claims := &auth.Claims{
		UserID: "admin@test.com",
		Role:   "admin",
	}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ScenariosRunRouter(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var payload ScenarioRunRequest
	err := json.Unmarshal(w.Body.Bytes(), &payload)
	require.NoError(t, err)
	assert.Equal(t, "scenario-rt", payload.ScenarioName)
}

func TestGraphRunsRouter_ConfigEndpoint(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))

	graphRun := &krknv1alpha1.KrknGraphRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "router-test-graph",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknGraphRunSpec{
			Graph: map[string]krknv1alpha1.GraphScenarioNode{
				"n1": {Name: "s1", Image: "img:1"},
			},
			TargetRequestID: "target-rt",
			TargetClusters:  map[string][]string{"krkn-operator": {"c1"}},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(graphRun).
		Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/graphruns/router-test-graph/config", nil)
	claims := &auth.Claims{
		UserID: "admin@test.com",
		Role:   "admin",
	}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GraphRunsRouter(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var payload GraphRunCreateRequest
	err := json.Unmarshal(w.Body.Bytes(), &payload)
	require.NoError(t, err)
	assert.Equal(t, "target-rt", payload.TargetRequestID)
}

// --- Router integration tests (v2 path normalization) ---

func TestScenariosRunRouter_ConfigEndpoint_V2Path(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))

	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "v2-test-run",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "target-v2",
			TargetClusters:  map[string][]string{"krkn-operator": {"c1"}},
			ScenarioName:    "scenario-v2",
			ScenarioImage:   "quay.io/test:v2",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(scenarioRun).
		Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/scenarios/run/v2-test-run/config", nil)
	claims := &auth.Claims{
		UserID: "admin@test.com",
		Role:   "admin",
	}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ScenariosRunRouter(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var payload ScenarioRunRequest
	err := json.Unmarshal(w.Body.Bytes(), &payload)
	require.NoError(t, err)
	assert.Equal(t, "scenario-v2", payload.ScenarioName)
}

func TestGraphRunsRouter_ConfigEndpoint_V2Path(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))

	graphRun := &krknv1alpha1.KrknGraphRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "v2-test-graph",
			Namespace: "krkn-operator-system",
		},
		Spec: krknv1alpha1.KrknGraphRunSpec{
			Graph: map[string]krknv1alpha1.GraphScenarioNode{
				"n1": {Name: "s1", Image: "img:1"},
			},
			TargetRequestID: "target-v2",
			TargetClusters:  map[string][]string{"krkn-operator": {"c1"}},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(graphRun).
		Build()

	handler := &Handler{
		client:    fakeClient,
		namespace: "krkn-operator-system",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/graphruns/v2-test-graph/config", nil)
	claims := &auth.Claims{
		UserID: "admin@test.com",
		Role:   "admin",
	}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GraphRunsRouter(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var payload GraphRunCreateRequest
	err := json.Unmarshal(w.Body.Bytes(), &payload)
	require.NoError(t, err)
	assert.Equal(t, "target-v2", payload.TargetRequestID)
}

// --- Method not allowed tests ---

func TestScenariosRunRouter_ConfigMethodNotAllowed(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	handler := &Handler{client: fakeClient, namespace: "krkn-operator-system"}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/run/some-run/config", nil)
	claims := &auth.Claims{UserID: "admin@test.com", Role: "admin"}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ScenariosRunRouter(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestGraphRunsRouter_ConfigMethodNotAllowed(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	handler := &Handler{client: fakeClient, namespace: "krkn-operator-system"}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/graphruns/some-graph/config", nil)
	claims := &auth.Claims{UserID: "admin@test.com", Role: "admin"}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GraphRunsRouter(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}