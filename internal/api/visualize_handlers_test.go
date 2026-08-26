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
	"time"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/visualize"
	routev1 "github.com/openshift/api/route/v1"
	routefake "github.com/openshift/client-go/route/clientset/versioned/fake"
	routeclient "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		visualizeRouteURL: func(_ context.Context, targetCluster, namespace, routeName string) (string, error) {
			assert.Equal(t, "cluster-1", targetCluster)
			assert.Equal(t, "krkn-visualize", namespace)
			assert.Equal(t, visualize.UpstreamRouteName, routeName)
			return "http://route-host.example.com", nil
		},
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
	for _, instance := range resp.Instances {
		if instance.Name == "instance-1" {
			assert.Equal(t, "http://route-host.example.com", instance.GrafanaURL)
		}
	}
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

func TestGetVisualizeInstanceStatusRejectsUnrelatedConfigMap(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unrelated",
			Namespace: "krkn-operator",
			Annotations: map[string]string{
				visualize.AnnotationStatus: visualize.StatusReady,
			},
		},
	}).Build()
	handler := &Handler{client: client, namespace: "krkn-operator"}

	req := httptest.NewRequest(http.MethodGet, VisualizePath+"/unrelated/status", nil)
	rr := httptest.NewRecorder()
	handler.GetVisualizeInstanceStatus(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDeleteVisualizeInstance(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, routev1.AddToScheme(scheme))
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
		name            string
		instanceName    string
		isAdmin         bool
		namespaceExists bool
		otherInstance   bool
		wantNamespace   bool
		expectedStatus  int
	}{
		{
			name:            "successful deletion",
			instanceName:    "test-instance",
			isAdmin:         true,
			namespaceExists: true,
			expectedStatus:  http.StatusOK,
		},
		{
			name:            "keeps namespace while another instance uses it",
			instanceName:    "test-instance",
			isAdmin:         true,
			namespaceExists: true,
			otherInstance:   true,
			wantNamespace:   true,
			expectedStatus:  http.StatusOK,
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
			objects := []runtime.Object{testInstance.DeepCopy()}
			var targetObjects []client.Object
			if tt.namespaceExists {
				targetObjects = append(targetObjects, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
					Name: "krkn-visualize",
				}})
			}
			if tt.otherInstance {
				objects = append(objects, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
					Name:      "other-instance",
					Namespace: "krkn-operator",
					Labels:    visualize.BuildLabels(),
					Annotations: map[string]string{
						visualize.AnnotationNamespace: "krkn-visualize",
						visualize.AnnotationStatus:    visualize.StatusReady,
					},
				}})
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()
			targetClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(targetObjects...).Build()
			handler := &Handler{
				client:    fakeClient,
				namespace: "krkn-operator",
				visualizeTargetClientFactory: func(_ context.Context, targetCluster string) (client.Client, error) {
					assert.Equal(t, "test-cluster", targetCluster)
					return targetClient, nil
				},
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

			if tt.isAdmin && tt.expectedStatus == http.StatusOK && tt.namespaceExists {
				require.Eventually(t, func() bool {
					var cm corev1.ConfigMap
					return apierrors.IsNotFound(fakeClient.Get(context.Background(), client.ObjectKey{
						Namespace: "krkn-operator",
						Name:      "test-instance",
					}, &cm))
				}, time.Second, 10*time.Millisecond)

				var ns corev1.Namespace
				err := targetClient.Get(context.Background(), client.ObjectKey{Name: "krkn-visualize"}, &ns)
				if tt.wantNamespace {
					assert.NoError(t, err)
				} else {
					assert.True(t, apierrors.IsNotFound(err))
				}
			}
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

	unrelatedConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unrelated",
			Namespace: "krkn-operator",
			Annotations: map[string]string{
				visualize.AnnotationNamespace: "sensitive-target",
			},
		},
	}

	grafanaDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      visualize.ResourceName("test-instance", "grafana"),
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
			Name:      visualize.ResourceName("test-instance", "grafana"),
			Namespace: "krkn-visualize",
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "10.0.0.1",
			Type:      corev1.ServiceTypeClusterIP,
		},
	}

	grafanaRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      visualize.ResourceName("test-instance", "grafana"),
			Namespace: "krkn-visualize",
		},
		Spec: routev1.RouteSpec{
			Host: "test-instance-grafana.apps.example.com",
		},
	}

	tests := []struct {
		name         string
		instanceName string
		existingObjs []runtime.Object
		// grafanaResources seeds the injected clientset/route fakes with the
		// Grafana Deployment, Service and Route fixtures.
		grafanaResources bool
		expectedStatus   int
		checkResponse    func(t *testing.T, body []byte)
	}{
		{
			name:             "successful logs retrieval with all resources",
			instanceName:     "test-instance",
			grafanaResources: true,
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
				assert.Contains(t, resp.Logs, "✓ Route: test-instance-grafana.apps.example.com")
				assert.Contains(t, resp.Logs, "http://test-instance-grafana.apps.example.com")
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
			name:           "unrelated ConfigMap is not exposed",
			instanceName:   "unrelated",
			existingObjs:   []runtime.Object{unrelatedConfigMap},
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

			// The logs handler inspects Deployment/Service state through a
			// clientset and the Route through an OpenShift route client; both
			// are injected as fakes, seeded only when the subtest expects the
			// Grafana resources to exist.
			clientset := k8sfake.NewSimpleClientset()
			routeClientSet := routefake.NewSimpleClientset()
			if tt.grafanaResources {
				clientset = k8sfake.NewSimpleClientset(grafanaDeployment, grafanaService)
				routeClientSet = routefake.NewSimpleClientset(grafanaRoute)
			}

			handler := &Handler{
				client:    client,
				clientset: clientset,
				namespace: "krkn-operator",
				visualizeRouteClientFactory: func() (routeclient.RouteV1Interface, error) {
					return routeClientSet.RouteV1(), nil
				},
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

// newDashboardsTestHandler builds a Handler with one Ready visualize instance
// whose Grafana route resolves to the given URL.
func newDashboardsTestHandler(t *testing.T, grafanaURL string) *Handler {
	t.Helper()
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
				visualize.AnnotationGrafanaURL:    "https://grafana-annotation.example.com",
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(testInstance).Build()
	return &Handler{
		client:    client,
		namespace: "krkn-operator",
		visualizeRouteURL: func(_ context.Context, targetCluster, namespace, routeName string) (string, error) {
			assert.Equal(t, "test-cluster", targetCluster)
			assert.Equal(t, "krkn-visualize", namespace)
			assert.Equal(t, visualize.UpstreamRouteName, routeName)
			return grafanaURL, nil
		},
	}
}

func TestGetVisualizeInstanceDashboards(t *testing.T) {
	grafanaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/search", r.URL.Path)
		assert.Equal(t, "dash-db", r.URL.Query().Get("type"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"uid": "pod-scenarios", "title": "Pod Scenarios - Chaos Engineering", "url": "/d/pod-scenarios/pod-scenarios", "folderTitle": "Chaos", "type": "dash-db"},
			{"uid": "node-overview", "title": "Node Overview", "url": "/d/node-overview/node-overview", "folderTitle": "General", "type": "dash-db"}
		]`))
	}))
	defer grafanaServer.Close()

	handler := newDashboardsTestHandler(t, grafanaServer.URL)

	req := httptest.NewRequest(http.MethodGet, VisualizePath+"/test-instance/dashboards", nil)
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{UserID: "user", Role: "user"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.GetVisualizeInstanceDashboards(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp visualize.DashboardsResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Total)
	require.Len(t, resp.Dashboards, 2)
	assert.Equal(t, "pod-scenarios", resp.Dashboards[0].UID)
	assert.Equal(t, "Pod Scenarios - Chaos Engineering", resp.Dashboards[0].Title)
	assert.Equal(t, "/d/pod-scenarios/pod-scenarios", resp.Dashboards[0].URL)
	assert.Equal(t, "Chaos", resp.Dashboards[0].FolderTitle)
}

func TestGetVisualizeInstanceDashboardsNotReady(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	testInstance := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "installing-instance",
			Namespace: "krkn-operator",
			Labels: map[string]string{
				visualize.LabelManagedBy: visualize.ManagedByValue,
				visualize.LabelComponent: visualize.ComponentValue,
			},
			Annotations: map[string]string{
				visualize.AnnotationTargetCluster: "test-cluster",
				visualize.AnnotationNamespace:     "krkn-visualize",
				visualize.AnnotationStatus:        visualize.StatusInstalling,
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(testInstance).Build()
	handler := &Handler{client: client, namespace: "krkn-operator"}

	req := httptest.NewRequest(http.MethodGet, VisualizePath+"/installing-instance/dashboards", nil)
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{UserID: "user", Role: "user"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.GetVisualizeInstanceDashboards(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
}

func TestGetVisualizeInstanceDashboardsGrafanaError(t *testing.T) {
	grafanaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer grafanaServer.Close()

	handler := newDashboardsTestHandler(t, grafanaServer.URL)

	req := httptest.NewRequest(http.MethodGet, VisualizePath+"/test-instance/dashboards", nil)
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{UserID: "user", Role: "user"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.GetVisualizeInstanceDashboards(rr, req)

	assert.Equal(t, http.StatusBadGateway, rr.Code)
}

func TestVisualizeRouterRoutesDashboards(t *testing.T) {
	handler := newDashboardsTestHandler(t, "https://unused.example.com")

	req := httptest.NewRequest(http.MethodGet, VisualizePath+"/test-instance/dashboards", nil)
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{UserID: "user", Role: "user"})
	req = req.WithContext(ctx)

	// Route resolution fails (unused URL is not reachable), which must surface
	// as a 502 from the dashboards handler, proving the router dispatched to
	// GetVisualizeInstanceDashboards rather than GetVisualizeInstance.
	rr := httptest.NewRecorder()
	handler.VisualizeRouter(rr, req)

	assert.Equal(t, http.StatusBadGateway, rr.Code)
}
