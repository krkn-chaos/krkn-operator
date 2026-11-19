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

package visualize

import (
	"context"
	"fmt"
	"testing"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// routeWithHost pre-seeds the Grafana Route for one instance. The fake client
// has no OpenShift router to assign spec.host, so DeployGrafana's route-host
// wait would otherwise time out; pre-seeding also exercises the
// AlreadyExists-tolerated path of CreateGrafanaRoute.
func routeWithHost(instanceName, namespace string) *routev1.Route {
	name := ResourceName(instanceName, "grafana")
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       routev1.RouteSpec{Host: name + ".apps.example.com"},
	}
}

func TestDeployGrafana(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, routev1.AddToScheme(scheme))

	tests := []struct {
		name           string
		config         DeploymentConfig
		existingObjs   []client.Object
		wantErr        bool
		checkResources func(t *testing.T, c client.Client)
	}{
		{
			name: "successful deployment",
			config: DeploymentConfig{
				InstanceName:    "test-viz",
				TargetCluster:   "test-cluster",
				Namespace:       "krkn-visualize",
				GrafanaPassword: "admin123",
			},
			existingObjs: []client.Object{routeWithHost("test-viz", "krkn-visualize")},
			wantErr:      false,
			checkResources: func(t *testing.T, c client.Client) {
				ctx := context.Background()

				// Check namespace
				var ns corev1.Namespace
				err := c.Get(ctx, client.ObjectKey{Name: "krkn-visualize"}, &ns)
				assert.NoError(t, err)
				assert.Equal(t, ManagedByValue, ns.Labels[LabelManagedBy])

				// Check deployment
				var deployment appsv1.Deployment
				err = c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: ResourceName("test-viz", "grafana")}, &deployment)
				assert.NoError(t, err)
				assert.Equal(t, int32(1), *deployment.Spec.Replicas)
				assert.Len(t, deployment.Spec.Template.Spec.Containers, 2)
				assert.Equal(t, fmt.Sprintf("http://%s:8081/render", ResourceName("test-viz", "grafana-renderer")), deployment.Spec.Template.Spec.Containers[0].Env[0].Value)
				syncer := deployment.Spec.Template.Spec.Containers[1]
				assert.Equal(t, "dashboard-syncer", syncer.Name)
				assert.Equal(t, "quay.io/krkn-chaos/visualize-syncer:opensearch-latest", syncer.Image)
				assert.Equal(t, "GRAFANA_ADMIN_PASSWORD", syncer.Env[0].Name)
				assert.Equal(t, ResourceName("test-viz", "grafana-admin"), syncer.Env[0].ValueFrom.SecretKeyRef.Name)
				assert.Equal(t, "http://admin:$(GRAFANA_ADMIN_PASSWORD)@localhost:3000", syncer.Env[1].Value)
				assert.Equal(t, "/visualize/rendered/Chaos:/visualize/rendered/k8s", syncer.Env[2].Value)

				var rendererDeployment appsv1.Deployment
				err = c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: ResourceName("test-viz", "grafana-renderer")}, &rendererDeployment)
				assert.NoError(t, err)
				assert.Equal(t, "grafana/grafana-image-renderer:latest", rendererDeployment.Spec.Template.Spec.Containers[0].Image)

				var rendererService corev1.Service
				err = c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: ResourceName("test-viz", "grafana-renderer")}, &rendererService)
				assert.NoError(t, err)
				assert.Equal(t, int32(8081), rendererService.Spec.Ports[0].Port)

				// Check service
				var service corev1.Service
				err = c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: ResourceName("test-viz", "grafana")}, &service)
				assert.NoError(t, err)
				assert.Equal(t, corev1.ServiceTypeClusterIP, service.Spec.Type)

				// Check secret. The fake client does not perform the API
				// server's StringData -> Data normalization, so the values
				// stay in StringData.
				var secret corev1.Secret
				err = c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: ResourceName("test-viz", "grafana-admin")}, &secret)
				assert.NoError(t, err)
				assert.Equal(t, "admin123", secret.StringData["admin-password"])
				assert.Len(t, secret.StringData["renderer-token"], 64)

				// Check datasources Secret
				var datasources corev1.Secret
				err = c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: ResourceName("test-viz", "grafana-datasources")}, &datasources)
				assert.NoError(t, err)
			},
		},
		{
			name: "deployment with Elasticsearch",
			config: DeploymentConfig{
				InstanceName:          "test-viz-es",
				TargetCluster:         "test-cluster",
				Namespace:             "krkn-visualize",
				GrafanaPassword:       "admin123",
				ElasticsearchURL:      "https://es.example.com:9200",
				ElasticsearchUsername: "elastic",
				ElasticsearchPassword: "elastic123",
			},
			existingObjs: []client.Object{routeWithHost("test-viz-es", "krkn-visualize")},
			wantErr:      false,
			checkResources: func(t *testing.T, c client.Client) {
				ctx := context.Background()

				// Check datasources Secret contains ES config
				var datasources corev1.Secret
				err := c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: ResourceName("test-viz-es", "grafana-datasources")}, &datasources)
				assert.NoError(t, err)
				assert.Contains(t, string(datasources.Data["datasources.yaml"]), "Elasticsearch")
				assert.Contains(t, string(datasources.Data["datasources.yaml"]), "https://es.example.com:9200")
			},
		},
		{
			name: "deployment with Prometheus",
			config: DeploymentConfig{
				InstanceName:    "test-viz-prom",
				TargetCluster:   "test-cluster",
				Namespace:       "krkn-visualize",
				GrafanaPassword: "admin123",
				PrometheusURL:   "https://prometheus.example.com:9090",
			},
			existingObjs: []client.Object{routeWithHost("test-viz-prom", "krkn-visualize")},
			wantErr:      false,
			checkResources: func(t *testing.T, c client.Client) {
				ctx := context.Background()

				// Check datasources Secret contains Prometheus config
				var datasources corev1.Secret
				err := c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: ResourceName("test-viz-prom", "grafana-datasources")}, &datasources)
				assert.NoError(t, err)
				assert.Contains(t, string(datasources.Data["datasources.yaml"]), "Prometheus")
				assert.Contains(t, string(datasources.Data["datasources.yaml"]), "https://prometheus.example.com:9090")
			},
		},
		{
			name: "namespace already exists",
			config: DeploymentConfig{
				InstanceName:    "test-viz",
				TargetCluster:   "test-cluster",
				Namespace:       "krkn-visualize",
				GrafanaPassword: "admin123",
			},
			existingObjs: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "krkn-visualize",
					},
				},
				routeWithHost("test-viz", "krkn-visualize"),
			},
			wantErr: false, // Should succeed even if namespace exists
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.existingObjs...).
				Build()

			deployer := NewGrafanaDeployer(c, "krkn-operator")
			ctx := context.Background()

			_, err := deployer.DeployGrafana(ctx, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkResources != nil {
					tt.checkResources(t, c)
				}
			}
		})
	}
}

func TestCreateGrafanaRenderer(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	oldDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: ResourceName("test-viz", "grafana-renderer"), Namespace: "krkn-visualize"},
		Spec:       appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "renderer", Image: "grafana/grafana-image-renderer:latest"}}}}},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(oldDeployment).Build()
	deployer := NewGrafanaDeployer(c, "krkn-operator")
	config := DeploymentConfig{InstanceName: "test-viz", Namespace: "krkn-visualize"}

	require.NoError(t, deployer.createGrafanaRenderer(context.Background(), config))

	var deployment appsv1.Deployment
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Namespace: config.Namespace, Name: ResourceName(config.InstanceName, "grafana-renderer")}, &deployment))
	require.Len(t, deployment.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "grafana/grafana-image-renderer:latest", deployment.Spec.Template.Spec.Containers[0].Image)
	assert.Equal(t, int32(8081), deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
	require.Len(t, deployment.Spec.Template.Spec.Containers[0].Env, 1)
	assert.Equal(t, "AUTH_TOKEN", deployment.Spec.Template.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, ResourceName(config.InstanceName, "grafana-admin"), deployment.Spec.Template.Spec.Containers[0].Env[0].ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "renderer-token", deployment.Spec.Template.Spec.Containers[0].Env[0].ValueFrom.SecretKeyRef.Key)

	var service corev1.Service
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Namespace: config.Namespace, Name: ResourceName(config.InstanceName, "grafana-renderer")}, &service))
	require.Len(t, service.Spec.Ports, 1)
	assert.Equal(t, int32(8081), service.Spec.Ports[0].Port)
}

func TestCreateGrafanaDeploymentIncludesDashboardSyncer(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	oldDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: ResourceName("test-viz", "grafana"), Namespace: "krkn-visualize"},
		Spec:       appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "grafana", Image: "grafana/grafana:latest"}}}}},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(oldDeployment).Build()
	deployer := NewGrafanaDeployer(c, "krkn-operator")
	config := DeploymentConfig{InstanceName: "test-viz", Namespace: "krkn-visualize"}

	require.NoError(t, deployer.createGrafanaDeployment(context.Background(), config))

	var deployment appsv1.Deployment
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Namespace: config.Namespace, Name: ResourceName(config.InstanceName, "grafana")}, &deployment))
	require.Len(t, deployment.Spec.Template.Spec.Containers, 2)
	assert.Equal(t, "grafana", deployment.Spec.Template.Spec.Containers[0].Name)
	var grafanaRendererTokenEnv *corev1.EnvVar
	for i := range deployment.Spec.Template.Spec.Containers[0].Env {
		if deployment.Spec.Template.Spec.Containers[0].Env[i].Name == "GF_RENDERING_RENDERER_TOKEN" {
			grafanaRendererTokenEnv = &deployment.Spec.Template.Spec.Containers[0].Env[i]
			break
		}
	}
	require.NotNil(t, grafanaRendererTokenEnv)
	assert.Equal(t, ResourceName(config.InstanceName, "grafana-admin"), grafanaRendererTokenEnv.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "renderer-token", grafanaRendererTokenEnv.ValueFrom.SecretKeyRef.Key)
	assert.Equal(t, "dashboard-syncer", deployment.Spec.Template.Spec.Containers[1].Name)
	assert.Equal(t, "quay.io/krkn-chaos/visualize-syncer:opensearch-latest", deployment.Spec.Template.Spec.Containers[1].Image)
}

func TestCreateGrafanaSecretAddsRendererTokenToExistingSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: ResourceName("test-viz", "grafana-admin"), Namespace: "krkn-visualize"},
		Data:       map[string][]byte{"admin-password": []byte("admin123")},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	deployer := NewGrafanaDeployer(c, "krkn-operator")
	config := DeploymentConfig{InstanceName: "test-viz", Namespace: "krkn-visualize", GrafanaPassword: "admin123"}

	require.NoError(t, deployer.createGrafanaSecret(context.Background(), config))

	var updated corev1.Secret
	require.NoError(t, c.Get(context.Background(), client.ObjectKeyFromObject(existing), &updated))
	assert.Equal(t, "admin123", string(updated.Data["admin-password"]))
	assert.Len(t, updated.Data["renderer-token"], 64)
}

func TestDeleteGrafana(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, routev1.AddToScheme(scheme))

	tests := []struct {
		name         string
		namespace    string
		existingObjs []client.Object
		wantErr      bool
		checkDeleted func(t *testing.T, c client.Client)
	}{
		{
			name:      "successful deletion",
			namespace: "krkn-visualize",
			existingObjs: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "krkn-visualize",
					},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ResourceName("test-viz", "grafana"),
						Namespace: "krkn-visualize",
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ResourceName("test-viz", "grafana"),
						Namespace: "krkn-visualize",
					},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ResourceName("test-viz", "grafana-renderer"),
						Namespace: "krkn-visualize",
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ResourceName("test-viz", "grafana-renderer"),
						Namespace: "krkn-visualize",
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-viz-grafana-pod",
						Namespace: "krkn-visualize",
						Labels: map[string]string{
							LabelManagedBy: ManagedByValue,
							LabelComponent: ComponentValue,
							LabelName:      "test-viz",
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other-instance-pod",
						Namespace: "krkn-visualize",
						Labels: map[string]string{
							LabelManagedBy: ManagedByValue,
							LabelComponent: ComponentValue,
							LabelName:      "other-instance",
						},
					},
				},
			},
			wantErr: false,
			checkDeleted: func(t *testing.T, c client.Client) {
				ctx := context.Background()

				// Shared namespace must remain intact.
				var ns corev1.Namespace
				err := c.Get(ctx, client.ObjectKey{Name: "krkn-visualize"}, &ns)
				assert.NoError(t, err)

				var deletedPod corev1.Pod
				err = c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: "test-viz-grafana-pod"}, &deletedPod)
				assert.True(t, apierrors.IsNotFound(err))

				var rendererDeployment appsv1.Deployment
				err = c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: ResourceName("test-viz", "grafana-renderer")}, &rendererDeployment)
				assert.True(t, apierrors.IsNotFound(err))
				var rendererService corev1.Service
				err = c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: ResourceName("test-viz", "grafana-renderer")}, &rendererService)
				assert.True(t, apierrors.IsNotFound(err))

				var otherPod corev1.Pod
				err = c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: "other-instance-pod"}, &otherPod)
				assert.NoError(t, err)
			},
		},
		{
			name:      "namespace doesn't exist",
			namespace: "nonexistent",
			wantErr:   false, // Should not error if namespace doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.existingObjs...).
				Build()

			deployer := NewGrafanaDeployer(c, "krkn-operator")
			ctx := context.Background()

			err := deployer.DeleteGrafana(ctx, tt.namespace, "test-viz")

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkDeleted != nil {
					tt.checkDeleted(t, c)
				}
			}
		})
	}
}

func TestDeleteNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name        string
		namespace   *corev1.Namespace
		wantDeleted bool
	}{
		{
			name: "deletes namespace created by krkn-operator",
			namespace: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: "krkn-visualize",
				Labels: map[string]string{
					LabelManagedBy: ManagedByValue,
					LabelComponent: ComponentValue,
				},
			}},
			wantDeleted: true,
		},
		{
			name: "deletes existing namespace without ownership labels",
			namespace: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: "krkn-visualize",
			}},
			wantDeleted: true,
		},
		{
			name:        "missing namespace is already cleaned up",
			wantDeleted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var objects []client.Object
			if tt.namespace != nil {
				objects = append(objects, tt.namespace)
			}
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
			deployer := NewGrafanaDeployer(c, "krkn-operator")

			err := deployer.DeleteNamespace(context.Background(), "krkn-visualize")
			require.NoError(t, err)

			var got corev1.Namespace
			err = c.Get(context.Background(), client.ObjectKey{Name: "krkn-visualize"}, &got)
			if tt.wantDeleted {
				assert.True(t, apierrors.IsNotFound(err))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateDatasourcesSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name     string
		config   DeploymentConfig
		wantYAML []string // Strings that should be in the YAML
	}{
		{
			name: "Elasticsearch datasource",
			config: DeploymentConfig{
				InstanceName:          "test",
				Namespace:             "krkn-visualize",
				ElasticsearchURL:      "https://es.example.com:9200",
				ElasticsearchUsername: "elastic",
				ElasticsearchPassword: "password",
			},
			wantYAML: []string{
				"name: Elasticsearch",
				"type: elasticsearch",
				"url: https://es.example.com:9200",
				"basicAuth: true",
				"basicAuthUser: elastic",
			},
		},
		{
			name: "Prometheus datasource",
			config: DeploymentConfig{
				InstanceName:  "test",
				Namespace:     "krkn-visualize",
				PrometheusURL: "https://prom.example.com:9090",
			},
			wantYAML: []string{
				"name: Prometheus",
				"type: prometheus",
				"url: https://prom.example.com:9090",
			},
		},
		{
			name: "Both datasources",
			config: DeploymentConfig{
				InstanceName:     "test",
				Namespace:        "krkn-visualize",
				ElasticsearchURL: "https://es.example.com:9200",
				PrometheusURL:    "https://prom.example.com:9090",
			},
			wantYAML: []string{
				"Elasticsearch",
				"Prometheus",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).Build()
			deployer := NewGrafanaDeployer(c, "krkn-operator")
			ctx := context.Background()

			err := deployer.createDatasourcesSecret(ctx, tt.config)
			require.NoError(t, err)

			// Verify Secret
			var secret corev1.Secret
			err = c.Get(ctx, client.ObjectKey{
				Namespace: tt.config.Namespace,
				Name:      ResourceName(tt.config.InstanceName, "grafana-datasources"),
			}, &secret)
			require.NoError(t, err)

			yaml := string(secret.Data["datasources.yaml"])
			for _, want := range tt.wantYAML {
				assert.Contains(t, yaml, want)
			}
		})
	}
}
