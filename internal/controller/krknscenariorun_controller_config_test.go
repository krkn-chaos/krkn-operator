package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLoadKrknctlConfigFromConfigMap(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add core scheme: %v", err)
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknctl-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"config.json": `{"quay_host":"registry.example.com","quay_org":"chaos","quay_scenario_registry":"scenarios","kubeconfig_path":"/custom/kubeconfig"}`,
		},
	}
	reconciler := &KrknScenarioRunReconciler{
		Client:               fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap).Build(),
		Namespace:            "default",
		KrknctlConfigMapName: "krknctl-config",
	}

	config, err := reconciler.loadKrknctlConfig(context.Background())
	if err != nil {
		t.Fatalf("expected ConfigMap config to load: %v", err)
	}
	if config.QuayHost != "registry.example.com" {
		t.Fatalf("expected ConfigMap quay host, got %q", config.QuayHost)
	}
	if config.KubeconfigPath != "/custom/kubeconfig" {
		t.Fatalf("expected ConfigMap kubeconfig path, got %q", config.KubeconfigPath)
	}
}

func TestLoadKrknctlConfigUsesEmbeddedDefaults(t *testing.T) {
	reconciler := &KrknScenarioRunReconciler{}

	config, err := reconciler.loadKrknctlConfig(context.Background())
	if err != nil {
		t.Fatalf("expected embedded config to load: %v", err)
	}
	if config.QuayHost == "" || config.QuayOrg == "" || config.QuayScenarioRegistry == "" {
		t.Fatalf("embedded config is missing image defaults: %+v", config)
	}
}

func TestLoadKrknctlConfigRejectsMissingConfigMapKey(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add core scheme: %v", err)
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "krknctl-config", Namespace: "default"},
		Data:       map[string]string{"other.json": "{}"},
	}
	reconciler := &KrknScenarioRunReconciler{
		Client:               fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap).Build(),
		Namespace:            "default",
		KrknctlConfigMapName: "krknctl-config",
	}

	if _, err := reconciler.loadKrknctlConfig(context.Background()); err == nil {
		t.Fatal("expected missing config.json key to fail")
	}
}
