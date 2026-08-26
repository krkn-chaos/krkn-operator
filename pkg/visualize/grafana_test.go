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
			wantErr: false,
			checkResources: func(t *testing.T, c client.Client) {
				ctx := context.Background()

				// Check namespace
				var ns corev1.Namespace
				err := c.Get(ctx, client.ObjectKey{Name: "krkn-visualize"}, &ns)
				assert.NoError(t, err)
				assert.Equal(t, ManagedByValue, ns.Labels[LabelManagedBy])

				// Check deployment
				var deployment appsv1.Deployment
				err = c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: "grafana"}, &deployment)
				assert.NoError(t, err)
				assert.Equal(t, int32(1), *deployment.Spec.Replicas)

				// Check service
				var service corev1.Service
				err = c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: "grafana"}, &service)
				assert.NoError(t, err)
				assert.Equal(t, corev1.ServiceTypeClusterIP, service.Spec.Type)

				// Check secret
				var secret corev1.Secret
				err = c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: "grafana-admin"}, &secret)
				assert.NoError(t, err)
				assert.Equal(t, "admin123", string(secret.Data["admin-password"]))

				// Check datasources ConfigMap
				var cm corev1.ConfigMap
				err = c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: "grafana-datasources"}, &cm)
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
			wantErr: false,
			checkResources: func(t *testing.T, c client.Client) {
				ctx := context.Background()

				// Check datasources ConfigMap contains ES config
				var cm corev1.ConfigMap
				err := c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: "grafana-datasources"}, &cm)
				assert.NoError(t, err)
				assert.Contains(t, cm.Data["datasources.yaml"], "Elasticsearch")
				assert.Contains(t, cm.Data["datasources.yaml"], "https://es.example.com:9200")
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
			wantErr: false,
			checkResources: func(t *testing.T, c client.Client) {
				ctx := context.Background()

				// Check datasources ConfigMap contains Prometheus config
				var cm corev1.ConfigMap
				err := c.Get(ctx, client.ObjectKey{Namespace: "krkn-visualize", Name: "grafana-datasources"}, &cm)
				assert.NoError(t, err)
				assert.Contains(t, cm.Data["datasources.yaml"], "Prometheus")
				assert.Contains(t, cm.Data["datasources.yaml"], "https://prometheus.example.com:9090")
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

func TestDeleteGrafana(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))

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
						Name:      "grafana",
						Namespace: "krkn-visualize",
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "grafana",
						Namespace: "krkn-visualize",
					},
				},
			},
			wantErr: false,
			checkDeleted: func(t *testing.T, c client.Client) {
				ctx := context.Background()

				// Namespace should be deleted (or in terminating state)
				var ns corev1.Namespace
				err := c.Get(ctx, client.ObjectKey{Name: "krkn-visualize"}, &ns)
				assert.True(t, apierrors.IsNotFound(err) || ns.DeletionTimestamp != nil)
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

			err := deployer.DeleteGrafana(ctx, tt.namespace)

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

func TestCreateDatasourcesConfigMap(t *testing.T) {
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

			err := deployer.createDatasourcesConfigMap(ctx, tt.config)
			require.NoError(t, err)

			// Verify ConfigMap
			var cm corev1.ConfigMap
			err = c.Get(ctx, client.ObjectKey{
				Namespace: tt.config.Namespace,
				Name:      "grafana-datasources",
			}, &cm)
			require.NoError(t, err)

			yaml := cm.Data["datasources.yaml"]
			for _, want := range tt.wantYAML {
				assert.Contains(t, yaml, want)
			}
		})
	}
}
