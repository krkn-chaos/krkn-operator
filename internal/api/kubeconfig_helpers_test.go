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
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func newKubeconfigHelperHandler(
	t *testing.T,
	targetID string,
	managedClusters map[string]map[string]map[string]string,
	targetData map[string][]krknv1alpha1.ClusterTarget,
) *Handler {
	t.Helper()

	managedClustersJSON, err := json.Marshal(managedClusters)
	if err != nil {
		t.Fatalf("marshal managed clusters: %v", err)
	}

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("register core API types: %v", err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("register Krkn API types: %v", err)
	}

	const namespace = "operator-system"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: targetID, Namespace: namespace},
		Data:       map[string][]byte{"managed-clusters": managedClustersJSON},
	}
	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{Name: targetID, Namespace: namespace},
		Status:     krknv1alpha1.KrknTargetRequestStatus{TargetData: targetData},
	}
	client := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(secret, targetRequest).Build()
	return &Handler{client: client, namespace: namespace}
}

func TestLegacyTargetKubeconfigUsesRequestedProvider(t *testing.T) {
	const targetID = "target-request"
	const clusterName = "shared-cluster"
	handler := newKubeconfigHelperHandler(t, targetID,
		map[string]map[string]map[string]string{
			"krkn-operator": {
				clusterName: {"cluster-api": "https://operator.cluster", "kubeconfig": "operator-kubeconfig"},
			},
			"krkn-operator-acm": {
				clusterName: {"cluster-api": "https://acm.cluster", "kubeconfig": "acm-kubeconfig"},
			},
		},
		map[string][]krknv1alpha1.ClusterTarget{
			"krkn-operator": {
				{ClusterName: clusterName, ClusterAPIURL: "https://operator.cluster"},
			},
			"krkn-operator-acm": {
				{ClusterName: clusterName, ClusterAPIURL: "https://acm.cluster"},
			},
		},
	)

	got, err := handler.getKubeconfig(context.Background(), "", targetID, "krkn-operator", clusterName)
	if err != nil {
		t.Fatalf("get kubeconfig: %v", err)
	}
	if got != "operator-kubeconfig" {
		t.Fatalf("kubeconfig = %q, want %q", got, "operator-kubeconfig")
	}

	apiURL, err := handler.getClusterAPIURL(context.Background(), "", targetID, "krkn-operator", clusterName)
	if err != nil {
		t.Fatalf("get cluster API URL: %v", err)
	}
	if apiURL != "https://operator.cluster" {
		t.Fatalf("cluster API URL = %q, want %q", apiURL, "https://operator.cluster")
	}
}

func TestLegacyTargetKubeconfigSupportsUniqueLegacyProvider(t *testing.T) {
	const targetID = "target-request"
	const clusterName = "prod"
	handler := newKubeconfigHelperHandler(t, targetID,
		map[string]map[string]map[string]string{
			"krkn-operator-acm": {
				clusterName: {"cluster-api": "https://prod.cluster", "kubeconfig": "legacy-kubeconfig"},
			},
		},
		map[string][]krknv1alpha1.ClusterTarget{
			"krkn-operator-acm": {
				{ClusterName: clusterName, ClusterAPIURL: "https://prod.cluster"},
			},
		},
	)

	got, err := handler.getKubeconfig(context.Background(), "", targetID, "", clusterName)
	if err != nil {
		t.Fatalf("get kubeconfig without provider: %v", err)
	}
	if got != "legacy-kubeconfig" {
		t.Fatalf("kubeconfig = %q, want %q", got, "legacy-kubeconfig")
	}
}

func TestLegacyTargetKubeconfigHandlesIdenticalDuplicateEntries(t *testing.T) {
	const targetID = "target-request"
	const clusterName = "shared-cluster"
	const clusterAPIURL = "https://shared.cluster"
	entry := func(kubeconfig string) map[string]string {
		return map[string]string{
			"cluster-name": clusterName,
			"cluster-api":  clusterAPIURL,
			"kubeconfig":   kubeconfig,
		}
	}

	tests := []struct {
		name           string
		acmKubeconfig  string
		wantKubeconfig string
		wantConflict   bool
	}{
		{
			name:           "identical kubeconfigs",
			acmKubeconfig:  "shared-kubeconfig",
			wantKubeconfig: "shared-kubeconfig",
		},
		{
			name:          "conflicting kubeconfigs",
			acmKubeconfig: "different-kubeconfig",
			wantConflict:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newKubeconfigHelperHandler(t, targetID,
				map[string]map[string]map[string]string{
					"krkn-operator": {
						clusterName: entry("shared-kubeconfig"),
					},
					"krkn-operator-acm": {
						clusterName: entry(tt.acmKubeconfig),
					},
				},
				map[string][]krknv1alpha1.ClusterTarget{
					"krkn-operator": {
						{ClusterName: clusterName, ClusterAPIURL: clusterAPIURL},
					},
					"krkn-operator-acm": {
						{ClusterName: clusterName, ClusterAPIURL: clusterAPIURL},
					},
				},
			)

			apiURL, err := handler.getClusterAPIURL(context.Background(), "", targetID, "", clusterName)
			if err != nil || apiURL != clusterAPIURL {
				t.Fatalf("authorized API URL = %q, error = %v; want %q", apiURL, err, clusterAPIURL)
			}

			got, err := handler.getKubeconfig(context.Background(), "", targetID, "", clusterName)
			if tt.wantConflict {
				if err == nil || !strings.Contains(err.Error(), "conflicting kubeconfigs") {
					t.Fatalf("kubeconfig = %q, error = %v; want conflicting kubeconfigs error", got, err)
				}
				return
			}
			if err != nil || got != tt.wantKubeconfig {
				t.Fatalf("kubeconfig = %q, error = %v; want %q", got, err, tt.wantKubeconfig)
			}
		})
	}
}

func TestLegacyTargetKubeconfigRejectsAmbiguousProvider(t *testing.T) {
	const targetID = "target-request"
	const clusterName = "shared-cluster"
	handler := newKubeconfigHelperHandler(t, targetID,
		map[string]map[string]map[string]string{
			"krkn-operator": {
				clusterName: {"kubeconfig": "operator-kubeconfig"},
			},
			"krkn-operator-acm": {
				clusterName: {"kubeconfig": "acm-kubeconfig"},
			},
		},
		map[string][]krknv1alpha1.ClusterTarget{
			"krkn-operator": {
				{ClusterName: clusterName, ClusterAPIURL: "https://operator.cluster"},
			},
			"krkn-operator-acm": {
				{ClusterName: clusterName, ClusterAPIURL: "https://acm.cluster"},
			},
		},
	)

	_, err := handler.getKubeconfig(context.Background(), "", targetID, "", clusterName)
	if !errors.Is(err, errAmbiguousClusterProvider) {
		t.Fatalf("error = %v, want ambiguous provider error", err)
	}
}

func TestLegacyKubeconfigRejectsStaleManagedClusterEntries(t *testing.T) {
	const targetID = "target-request"
	const clusterName = "shared-cluster"
	const authorizedProvider = "krkn-operator"
	const authorizedURL = "https://authorized.cluster"

	tests := []struct {
		name          string
		operatorName  string
		targetURL     string
		managedValues map[string]map[string]map[string]string
	}{
		{
			name:      "same name under stale provider",
			targetURL: authorizedURL,
			managedValues: map[string]map[string]map[string]string{
				"krkn-operator-acm": {
					clusterName: {"cluster-api": "https://stale.cluster", "kubeconfig": "stale-kubeconfig"},
				},
			},
		},
		{
			name:         "same provider with stale API URL",
			operatorName: authorizedProvider,
			targetURL:    authorizedURL,
			managedValues: map[string]map[string]map[string]string{
				authorizedProvider: {
					clusterName: {"cluster-api": "https://stale.cluster", "kubeconfig": "stale-kubeconfig"},
				},
			},
		},
		{
			name:         "missing API URL",
			operatorName: authorizedProvider,
			managedValues: map[string]map[string]map[string]string{
				authorizedProvider: {
					clusterName: {"kubeconfig": "stale-kubeconfig"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newKubeconfigHelperHandler(t, targetID, tt.managedValues,
				map[string][]krknv1alpha1.ClusterTarget{
					authorizedProvider: {
						{ClusterName: clusterName, ClusterAPIURL: tt.targetURL},
					},
				},
			)

			gotURL, err := handler.getClusterAPIURL(context.Background(), "", targetID, tt.operatorName, clusterName)
			if err != nil {
				t.Fatalf("resolve authorized cluster URL: %v", err)
			}
			if gotURL != tt.targetURL {
				t.Fatalf("authorized URL = %q, want %q", gotURL, tt.targetURL)
			}

			got, err := handler.getKubeconfig(context.Background(), "", targetID, tt.operatorName, clusterName)
			if err == nil || got != "" {
				t.Fatalf("kubeconfig = %q, error = %v; want stale entry rejected", got, err)
			}
		})
	}
}

func TestLegacyTargetClusterAPIURLRejectsAmbiguousProvider(t *testing.T) {
	const targetID = "target-request"
	const clusterName = "shared-cluster"
	handler := newKubeconfigHelperHandler(t, targetID, nil,
		map[string][]krknv1alpha1.ClusterTarget{
			"krkn-operator": {
				{ClusterName: clusterName, ClusterAPIURL: "https://operator.cluster"},
			},
			"krkn-operator-acm": {
				{ClusterName: clusterName, ClusterAPIURL: "https://acm.cluster"},
			},
		},
	)

	_, err := handler.getClusterAPIURL(context.Background(), "", targetID, "", clusterName)
	if !errors.Is(err, errAmbiguousClusterProvider) {
		t.Fatalf("error = %v, want ambiguous provider error", err)
	}
}

func TestLegacyTargetClusterAPIURLPreservesEmptyURL(t *testing.T) {
	const targetID = "target-request"
	const clusterName = "legacy-cluster"
	handler := newKubeconfigHelperHandler(t, targetID, nil,
		map[string][]krknv1alpha1.ClusterTarget{
			"krkn-operator-acm": {{ClusterName: clusterName}},
		},
	)

	apiURL, err := handler.getClusterAPIURL(context.Background(), "", targetID, "", clusterName)
	if err != nil || apiURL != "" {
		t.Fatalf("API URL = %q, error = %v; want empty URL without error", apiURL, err)
	}
}

func TestLegacyEndpointsForwardOperatorName(t *testing.T) {
	const targetID = "target-request"
	const clusterName = "shared-cluster"
	handler := newKubeconfigHelperHandler(t, targetID,
		map[string]map[string]map[string]string{
			"krkn-operator": {
				clusterName: {"kubeconfig": "operator-kubeconfig"},
			},
		},
		nil,
	)

	tests := []struct {
		name    string
		request *http.Request
		serve   func(http.ResponseWriter, *http.Request)
	}{
		{
			name:    "nodes",
			request: httptest.NewRequest(http.MethodGet, "/nodes?id=target-request&cluster-name=shared-cluster&operator-name=missing", nil),
			serve:   handler.GetNodes,
		},
		{
			name:    "terminal",
			request: httptest.NewRequest(http.MethodPost, "/terminal", strings.NewReader(`{"uuid":"target-request","cluster_id":"shared-cluster","operator_name":"missing","command":"kubectl get pods"}`)),
			serve:   handler.ExecuteTerminal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			tt.serve(response, tt.request)
			if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "provider 'missing'") {
				t.Fatalf("status = %d, body = %s; want provider-specific 404", response.Code, response.Body.String())
			}
		})
	}
}
