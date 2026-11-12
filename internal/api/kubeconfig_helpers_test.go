package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
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

	kubeconfig, err := handler.getKubeconfigFromTargetRequest(context.Background(), "target-uuid", "workload", "")
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

	_, err = handler.getKubeconfigFromTargetRequest(context.Background(), "target-uuid", "workload", "")
	if err == nil || !strings.Contains(err.Error(), "multiple operators") {
		t.Fatalf("expected ambiguous cluster error, got %v", err)
	}
}

func TestGetClusterAPIURL_RejectsDuplicateClusterNames(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	target := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "target-uuid", Namespace: "test-namespace"},
		Status: krknv1alpha1.KrknTargetRequestStatus{TargetData: map[string][]krknv1alpha1.ClusterTarget{
			"provider-a": {{ClusterName: "shared", ClusterAPIURL: "https://a"}},
			"provider-b": {{ClusterName: "shared", ClusterAPIURL: "https://b"}},
		}},
	}
	handler := NewTestHandler(fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(target).Build(), fake.NewSimpleClientset(), "test-namespace", "localhost:50051")
	_, err := handler.getClusterAPIURL(context.Background(), "", "target-uuid", "shared", "")
	if err == nil || !strings.Contains(err.Error(), "multiple operators") {
		t.Fatalf("expected ambiguous cluster error, got %v", err)
	}
}

func TestGetClusterAPIURL_PreservesEmptyURLForUniqueLegacyCluster(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	target := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "target-uuid", Namespace: "test-namespace"},
		Status: krknv1alpha1.KrknTargetRequestStatus{TargetData: map[string][]krknv1alpha1.ClusterTarget{
			"provider-a": {{ClusterName: "shared"}},
		}},
	}
	handler := NewTestHandler(fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(target).Build(), fake.NewSimpleClientset(), "test-namespace", "localhost:50051")
	url, err := handler.getClusterAPIURL(context.Background(), "", "target-uuid", "shared", "")
	if err != nil || url != "" {
		t.Fatalf("url = %q, error = %v; want empty URL without error", url, err)
	}
}

func TestLegacyLookup_SelectsProvider(t *testing.T) {
	data, err := json.Marshal(map[string]map[string]map[string]string{
		"provider-a": {"shared": {"kubeconfig": "config-a"}},
		"provider-b": {"shared": {"kubeconfig": "config-b"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "target-uuid", Namespace: "test-namespace"}, Data: map[string][]byte{"managed-clusters": data}}
	target := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "target-uuid", Namespace: "test-namespace"},
		Status: krknv1alpha1.KrknTargetRequestStatus{TargetData: map[string][]krknv1alpha1.ClusterTarget{
			"provider-a": {{ClusterName: "shared", ClusterAPIURL: "https://a"}},
			"provider-b": {{ClusterName: "shared", ClusterAPIURL: "https://b"}},
		}},
	}
	handler := NewTestHandler(fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(secret, target).Build(), fake.NewSimpleClientset(), "test-namespace", "localhost:50051")
	for _, tc := range []struct{ provider, config, url string }{
		{"provider-a", "config-a", "https://a"},
		{"provider-b", "config-b", "https://b"},
	} {
		t.Run(tc.provider, func(t *testing.T) {
			config, err := handler.getKubeconfigFromTargetRequest(context.Background(), "target-uuid", "shared", tc.provider)
			if err != nil || config != tc.config {
				t.Fatalf("config = %q, error = %v; want %q", config, err, tc.config)
			}
			url, err := handler.getClusterAPIURL(context.Background(), "", "target-uuid", "shared", tc.provider)
			if err != nil || url != tc.url {
				t.Fatalf("url = %q, error = %v; want %q", url, err, tc.url)
			}
		})
	}
	if _, err := handler.getKubeconfigFromTargetRequest(context.Background(), "target-uuid", "shared", "missing"); err == nil {
		t.Fatal("expected unknown provider to fail")
	}
}

func TestLegacyEndpoints_ForwardProvider(t *testing.T) {
	data, err := json.Marshal(map[string]map[string]map[string]string{
		"provider-a": {"shared": {"kubeconfig": "config-a"}},
		"provider-b": {"shared": {"kubeconfig": "config-b"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "target-uuid", Namespace: "test-namespace"}, Data: map[string][]byte{"managed-clusters": data}}
	handler := NewTestHandler(fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build(), fake.NewSimpleClientset(), "test-namespace", "localhost:50051")
	for _, tc := range []struct {
		name    string
		request *http.Request
		serve   func(http.ResponseWriter, *http.Request)
	}{
		{"nodes", httptest.NewRequest(http.MethodGet, "/nodes?id=target-uuid&cluster-name=shared&provider=missing", nil), handler.GetNodes},
		{"terminal", httptest.NewRequest(http.MethodPost, "/terminal", strings.NewReader(`{"uuid":"target-uuid","cluster_id":"shared","provider":"missing","command":"kubectl get pods"}`)), handler.ExecuteTerminal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			tc.serve(response, tc.request)
			if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "provider 'missing'") {
				t.Fatalf("status = %d, body = %s; want provider-specific 404", response.Code, response.Body.String())
			}
		})
	}
}
