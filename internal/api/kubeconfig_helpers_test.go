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
				clusterName: {"kubeconfig": "legacy-kubeconfig"},
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
