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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/visualize"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateVisualizeInstance(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	tests := []struct {
		name           string
		request        visualize.CreateVisualizeRequest
		isAdmin        bool
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name: "successful creation",
			request: visualize.CreateVisualizeRequest{
				Name:                 "test-visualize",
				TargetClusters:       []string{"test-cluster"},
				Namespace:            "krkn-visualize",
				GrafanaPassword:      "admin123",
				AutoDetectPrometheus: true,
			},
			isAdmin:        true,
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				var resp visualize.CreateVisualizeResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "test-visualize", resp.Name)
				assert.Contains(t, resp.Message, "installation started")
			},
		},
		{
			name: "missing required fields",
			request: visualize.CreateVisualizeRequest{
				Name: "test-visualize",
			},
			isAdmin:        true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "non-admin forbidden",
			request: visualize.CreateVisualizeRequest{
				Name:            "test-visualize",
				TargetClusters:  []string{"test-cluster"},
				GrafanaPassword: "admin123",
			},
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "with elasticsearch config",
			request: visualize.CreateVisualizeRequest{
				Name:                    "test-with-es",
				TargetClusters:          []string{"test-cluster"},
				ElasticsearchConfigName: "production-es",
				GrafanaPassword:         "admin123",
				AutoDetectPrometheus:    true,
			},
			isAdmin:        true,
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(scheme).Build()
			handler := &Handler{
				client:    client,
				namespace: "krkn-operator",
			}

			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, VisualizePath, bytes.NewReader(body))

			ctx := req.Context()
			if tt.isAdmin {
				ctx = context.WithValue(ctx, auth.UserClaimsKey, &auth.Claims{UserID: "admin", Role: "admin"})
			} else {
				ctx = context.WithValue(ctx, auth.UserClaimsKey, &auth.Claims{UserID: "user", Role: "user"})
			}
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.CreateVisualizeInstance(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr.Body.Bytes())
			}
		})
	}
}

func TestListVisualizeInstances(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	// Create test ConfigMaps
	testInstances := []runtime.Object{
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "instance-1",
				Namespace: "krkn-operator",
				Labels: map[string]string{
					visualize.LabelManagedBy: visualize.ManagedByValue,
					visualize.LabelComponent: visualize.ComponentValue,
				},
				Annotations: map[string]string{
					visualize.AnnotationTargetCluster: "cluster-1",
					visualize.AnnotationNamespace:     "krkn-visualize",
					visualize.AnnotationStatus:        visualize.StatusReady,
					visualize.AnnotationGrafanaURL:    "https://grafana.example.com",
				},
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "instance-2",
				Namespace: "krkn-operator",
				Labels: map[string]string{
					visualize.LabelManagedBy: visualize.ManagedByValue,
					visualize.LabelComponent: visualize.ComponentValue,
				},
				Annotations: map[string]string{
					visualize.AnnotationTargetCluster: "cluster-2",
					visualize.AnnotationNamespace:     "krkn-visualize",
					visualize.AnnotationStatus:        visualize.StatusPending,
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(testInstances...).Build()
	handler := &Handler{
		client:    client,
		namespace: "krkn-operator",
	}

	req := httptest.NewRequest(http.MethodGet, VisualizePath, nil)
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{UserID: "user", Role: "user"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ListVisualizeInstances(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp visualize.ListVisualizeInstancesResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Total)
	assert.Len(t, resp.Instances, 2)
}

func TestGetVisualizeInstance(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	testInstance := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "krkn-operator",
			Labels: map[string]string{
				visualize.LabelManagedBy: visualize.ManagedByValue,
				visualize.LabelComponent: visualize.ComponentValue,
			},
			Annotations: map[string]string{
				visualize.AnnotationTargetCluster: "test-cluster",
				visualize.AnnotationNamespace:     "krkn-visualize",
				visualize.AnnotationStatus:        visualize.StatusReady,
				visualize.AnnotationGrafanaURL:    "https://grafana.example.com",
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(testInstance).Build()
	handler := &Handler{
		client:    client,
		namespace: "krkn-operator",
	}

	req := httptest.NewRequest(http.MethodGet, VisualizePath+"/test-instance", nil)
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{UserID: "user", Role: "user"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.GetVisualizeInstance(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp visualize.VisualizeInstanceResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "test-instance", resp.Name)
	assert.Equal(t, "test-cluster", resp.TargetCluster)
	assert.Equal(t, visualize.StatusReady, resp.Status)
}

func TestDeleteVisualizeInstance(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	testInstance := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "krkn-operator",
			Labels: map[string]string{
				visualize.LabelManagedBy: visualize.ManagedByValue,
				visualize.LabelComponent: visualize.ComponentValue,
			},
			Annotations: map[string]string{
				visualize.AnnotationTargetCluster: "test-cluster",
				visualize.AnnotationNamespace:     "krkn-visualize",
				visualize.AnnotationStatus:        visualize.StatusReady,
			},
		},
	}

	tests := []struct {
		name           string
		instanceName   string
		isAdmin        bool
		expectedStatus int
	}{
		{
			name:           "successful deletion",
			instanceName:   "test-instance",
			isAdmin:        true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "non-admin forbidden",
			instanceName:   "test-instance",
			isAdmin:        false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "not found",
			instanceName:   "nonexistent",
			isAdmin:        true,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(testInstance).Build()
			handler := &Handler{
				client:    client,
				namespace: "krkn-operator",
			}

			req := httptest.NewRequest(http.MethodDelete, VisualizePath+"/"+tt.instanceName, nil)
			ctx := req.Context()
			if tt.isAdmin {
				ctx = context.WithValue(ctx, auth.UserClaimsKey, &auth.Claims{UserID: "admin", Role: "admin"})
			} else {
				ctx = context.WithValue(ctx, auth.UserClaimsKey, &auth.Claims{UserID: "user", Role: "user"})
			}
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.DeleteVisualizeInstance(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestGetVisualizeInstanceLogs(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, routev1.AddToScheme(scheme))

	testInstance := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "krkn-operator",
			Labels: map[string]string{
				visualize.LabelManagedBy: visualize.ManagedByValue,
				visualize.LabelComponent: visualize.ComponentValue,
			},
			Annotations: map[string]string{
				visualize.AnnotationTargetCluster: "test-cluster",
				visualize.AnnotationNamespace:     "krkn-visualize",
				visualize.AnnotationStatus:        visualize.StatusReady,
				visualize.AnnotationGrafanaURL:    "https://grafana.example.com",
			},
		},
	}

	grafanaDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana",
			Namespace: "krkn-visualize",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: func() *int32 { r := int32(1); return &r }(),
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 1,
		},
	}

	grafanaService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana",
			Namespace: "krkn-visualize",
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "10.0.0.1",
			Type:      corev1.ServiceTypeClusterIP,
		},
	}

	grafanaRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana",
			Namespace: "krkn-visualize",
		},
		Spec: routev1.RouteSpec{
			Host: "grafana-krkn-visualize.apps.example.com",
		},
	}

	tests := []struct {
		name           string
		instanceName   string
		existingObjs   []runtime.Object
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:         "successful logs retrieval with all resources",
			instanceName: "test-instance",
			existingObjs: []runtime.Object{
				testInstance,
				grafanaDeployment,
				grafanaService,
				grafanaRoute,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp visualize.LogsResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Contains(t, resp.Logs, "test-instance")
				assert.Contains(t, resp.Logs, "krkn-visualize")
				assert.Contains(t, resp.Logs, "Ready")
				assert.Contains(t, resp.Logs, "✓ Deployment: 1/1 replicas ready")
				assert.Contains(t, resp.Logs, "✓ Service")
				assert.Contains(t, resp.Logs, "✓ Route: grafana-krkn-visualize.apps.example.com")
				assert.Contains(t, resp.Logs, "https://grafana-krkn-visualize.apps.example.com")
				assert.Equal(t, visualize.StatusReady, resp.Status)
			},
		},
		{
			name:           "instance not found",
			instanceName:   "nonexistent",
			existingObjs:   []runtime.Object{},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:         "deployment without Grafana resources",
			instanceName: "test-instance",
			existingObjs: []runtime.Object{
				testInstance,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp visualize.LogsResponse
				require.NoError(t, json.Unmarshal(body, &resp))
				assert.Contains(t, resp.Logs, "✗ Deployment: Not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(tt.existingObjs...).
				Build()

			handler := &Handler{
				client:    client,
				namespace: "krkn-operator",
			}

			req := httptest.NewRequest(http.MethodGet, VisualizePath+"/"+tt.instanceName+"/logs", nil)
			ctx := context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{UserID: "user", Role: "user"})
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.GetVisualizeInstanceLogs(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr.Body.Bytes())
			}
		})
	}
}
