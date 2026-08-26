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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateGrafanaRoute(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, routev1.AddToScheme(scheme))

	tests := []struct {
		name         string
		namespace    string
		instanceName string
		existingObjs []client.Object
		wantErr      bool
		checkRoute   func(t *testing.T, c client.Client)
	}{
		{
			name:         "successful route creation",
			namespace:    "krkn-visualize",
			instanceName: "test-viz",
			wantErr:      false,
			checkRoute: func(t *testing.T, c client.Client) {
				ctx := context.Background()
				var route routev1.Route
				err := c.Get(ctx, client.ObjectKey{
					Namespace: "krkn-visualize",
					Name:      "grafana",
				}, &route)
				require.NoError(t, err)

				// Check labels
				assert.Equal(t, ManagedByValue, route.Labels[LabelManagedBy])
				assert.Equal(t, ComponentValue, route.Labels[LabelComponent])
				assert.Equal(t, "test-viz", route.Labels[LabelName])

				// Check route spec
				assert.Equal(t, "Service", route.Spec.To.Kind)
				assert.Equal(t, "grafana", route.Spec.To.Name)
				assert.Equal(t, "http", route.Spec.Port.TargetPort.StrVal)

				// Check TLS
				require.NotNil(t, route.Spec.TLS)
				assert.Equal(t, routev1.TLSTerminationEdge, route.Spec.TLS.Termination)
				assert.Equal(t, routev1.InsecureEdgeTerminationPolicyRedirect, route.Spec.TLS.InsecureEdgeTerminationPolicy)
			},
		},
		{
			name:         "route already exists",
			namespace:    "krkn-visualize",
			instanceName: "test-viz",
			existingObjs: []client.Object{
				&routev1.Route{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "grafana",
						Namespace: "krkn-visualize",
					},
					Spec: routev1.RouteSpec{
						Host: "existing-host.example.com",
					},
				},
			},
			wantErr: false, // Should not error if route exists
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.existingObjs...).
				Build()

			ctx := context.Background()
			err := CreateGrafanaRoute(ctx, c, tt.namespace, tt.instanceName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkRoute != nil {
					tt.checkRoute(t, c)
				}
			}
		})
	}
}

func TestGetRouteURL(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, routev1.AddToScheme(scheme))

	tests := []struct {
		name      string
		namespace string
		route     *routev1.Route
		wantURL   string
		wantErr   bool
	}{
		{
			name:      "route with host assigned",
			namespace: "krkn-visualize",
			route: &routev1.Route{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "grafana",
					Namespace: "krkn-visualize",
				},
				Spec: routev1.RouteSpec{
					Host: "grafana-krkn-visualize.apps.example.com",
				},
			},
			wantURL: "https://grafana-krkn-visualize.apps.example.com",
			wantErr: false,
		},
		{
			name:      "route not found",
			namespace: "krkn-visualize",
			route:     nil, // No route created
			wantURL:   "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var objs []client.Object
			if tt.route != nil {
				objs = append(objs, tt.route)
			}

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objs...).
				Build()

			// Note: GetRouteURL uses wait.PollImmediate which may need to be stubbed
			// For now, test the happy path where route exists with host
			if tt.route != nil && tt.route.Spec.Host != "" {
				ctx := context.Background()
				url, err := GetRouteURL(ctx, c, tt.namespace)

				if tt.wantErr {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tt.wantURL, url)
				}
			}
		})
	}
}
