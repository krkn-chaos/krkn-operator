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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

const testOperatorNamespace = "krkn-operator-system"

// newProvider is a helper that builds a KrknOperatorTargetProvider CR for tests.
func newProvider(name, namespace, operatorName string, active bool) *krknv1alpha1.KrknOperatorTargetProvider {
	return &krknv1alpha1.KrknOperatorTargetProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderSpec{
			OperatorName: operatorName,
			Active:       active,
		},
	}
}

// newProvidersHandler builds a Handler backed by a fake client seeded with the
// given provider objects and scoped to the test operator namespace.
func newProvidersHandler(t *testing.T, objs ...client.Object) *Handler {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		Build()

	return &Handler{
		client:    fakeClient,
		namespace: testOperatorNamespace,
	}
}

// TestListProviders_ScopedToNamespace verifies that ListProviders only returns
// providers that live in the operator namespace, ignoring identically-shaped
// resources in other namespaces. This guards the namespace-scoping fix so a
// cluster with providers in multiple namespaces cannot leak them across tenants.
func TestListProviders_ScopedToNamespace(t *testing.T) {
	inNamespace := newProvider("provider-in", testOperatorNamespace, "operator-in", true)
	otherNamespace := newProvider("provider-out", "other-namespace", "operator-out", true)

	handler := newProvidersHandler(t, inNamespace, otherNamespace)

	req := httptest.NewRequest(http.MethodGet, ProvidersPath, nil)
	w := httptest.NewRecorder()

	handler.ListProviders(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp ListProvidersResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	require.Len(t, resp.Providers, 1, "only providers in the operator namespace should be returned")
	assert.Equal(t, "operator-in", resp.Providers[0].Name)
	assert.True(t, resp.Providers[0].Active)
}

// TestListProviders_Empty verifies that ListProviders returns an empty (but
// non-nil) list with a 200 status when no providers exist in the namespace.
func TestListProviders_Empty(t *testing.T) {
	handler := newProvidersHandler(t)

	req := httptest.NewRequest(http.MethodGet, ProvidersPath, nil)
	w := httptest.NewRecorder()

	handler.ListProviders(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp ListProvidersResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Providers)
	assert.Empty(t, resp.Providers)
}

// TestUpdateProviderStatus covers the request-validation and namespace-scoping
// behavior of the PATCH /api/v1/providers/{name} endpoint.
func TestUpdateProviderStatus(t *testing.T) {
	tests := []struct {
		name           string
		providerName   string
		body           string
		seed           []client.Object
		expectedStatus int
		expectActive   bool // only checked when expectedStatus == 200
	}{
		{
			name:         "deactivates existing provider",
			providerName: "operator-in",
			body:         `{"active": false}`,
			seed: []client.Object{
				newProvider("provider-in", testOperatorNamespace, "operator-in", true),
			},
			expectedStatus: http.StatusOK,
			expectActive:   false,
		},
		{
			name:         "activates existing provider",
			providerName: "operator-in",
			body:         `{"active": true}`,
			seed: []client.Object{
				newProvider("provider-in", testOperatorNamespace, "operator-in", false),
			},
			expectedStatus: http.StatusOK,
			expectActive:   true,
		},
		{
			name:         "provider in other namespace is not found",
			providerName: "operator-out",
			body:         `{"active": false}`,
			seed: []client.Object{
				newProvider("provider-out", "other-namespace", "operator-out", true),
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "missing provider name",
			providerName:   "",
			body:           `{"active": false}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid request body",
			providerName:   "operator-in",
			body:           `{not-json`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newProvidersHandler(t, tt.seed...)

			req := httptest.NewRequest(
				http.MethodPatch,
				ProvidersPath+"/"+tt.providerName,
				strings.NewReader(tt.body),
			)
			w := httptest.NewRecorder()

			handler.UpdateProviderStatus(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus != http.StatusOK {
				return
			}

			var resp UpdateProviderStatusResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, tt.providerName, resp.Name)
			assert.Equal(t, tt.expectActive, resp.Active)
		})
	}
}
