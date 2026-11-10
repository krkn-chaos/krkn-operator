package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetKubeconfigFromTargetRequest_UsesConfiguredOperator(t *testing.T) {
	managedClusters := map[string]map[string]map[string]string{
		"krkn-operator": {"workload": {"kubeconfig": "expected-kubeconfig"}},
	}
	data, err := json.Marshal(managedClusters)
	if err != nil {
		t.Fatal(err)
	}
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "target-uuid", Namespace: "test-namespace"},
		Data:       map[string][]byte{"managed-clusters": data},
	}
	client := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
	handler := NewTestHandler(client, fake.NewSimpleClientset(), "test-namespace", "localhost:50051")

	kubeconfig, err := handler.getKubeconfigFromTargetRequest(context.Background(), "target-uuid", "workload")
	if err != nil {
		t.Fatal(err)
	}
	if kubeconfig != "expected-kubeconfig" {
		t.Fatalf("kubeconfig = %q, want expected-kubeconfig", kubeconfig)
	}
}

func TestGetKubeconfigFromTargetRequest_RejectsDuplicateClusterNames(t *testing.T) {
	managedClusters := map[string]map[string]map[string]string{
		"krkn-operator":     {"workload": {"kubeconfig": "first"}},
		"krkn-operator-acm": {"workload": {"kubeconfig": "second"}},
	}
	data, err := json.Marshal(managedClusters)
	if err != nil {
		t.Fatal(err)
	}
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "target-uuid", Namespace: "test-namespace"},
		Data:       map[string][]byte{"managed-clusters": data},
	}
	client := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
	handler := NewTestHandler(client, fake.NewSimpleClientset(), "test-namespace", "localhost:50051")

	_, err = handler.getKubeconfigFromTargetRequest(context.Background(), "target-uuid", "workload")
	if err == nil || !strings.Contains(err.Error(), "multiple operators") {
		t.Fatalf("expected ambiguous cluster error, got %v", err)
	}
}
