package controller

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnsureProvisionedUsesRunConfigMap(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add core scheme: %v", err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add krkn scheme: %v", err)
	}

	configYAML := "genetic:\n  generations: 1\n"
	kubeconfig := "apiVersion: v1\nkind: Config\n"
	managedClusters := `{"krkn-operator":{"self":{"kubeconfig":"` +
		base64.StdEncoding.EncodeToString([]byte(kubeconfig)) + `"}}}`
	aiRun := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run-one",
			Namespace: "operator",
			UID:       types.UID("run-one-uid"),
		},
		Spec: krknv1alpha1.KrknAIRunSpec{
			TargetRequestID: "self",
			TargetClusters:  map[string][]string{"krkn-operator": {"self"}},
			ConfigMapName:   "config-one",
		},
		Status: krknv1alpha1.KrknAIRunStatus{
			Phase:               "Provisioning",
			PVCName:             "ai-run-one-results",
			OrchestratorPodName: "ai-run-run-one",
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		aiRun,
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "config-one", Namespace: "operator"},
			Data:       map[string]string{"krkn-ai.yaml": configYAML},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "self", Namespace: "operator"},
			Data:       map[string][]byte{"managed-clusters": []byte(managedClusters)},
		},
	).Build()
	reconciler := &KrknAIRunReconciler{
		Client:            fakeClient,
		Scheme:            scheme,
		Namespace:         "operator",
		OrchestratorImage: "registry.example/krkn-ai-operator:test",
	}

	if err := reconciler.ensureProvisioned(context.Background(), aiRun); err != nil {
		t.Fatalf("ensureProvisioned failed: %v", err)
	}

	var generatedConfig corev1.ConfigMap
	if err := fakeClient.Get(context.Background(), client.ObjectKey{
		Name: "ai-run-one-config", Namespace: "operator",
	}, &generatedConfig); err != nil {
		t.Fatalf("failed to get generated config ConfigMap: %v", err)
	}
	if generatedConfig.Data["krkn-ai.yaml"] != configYAML {
		t.Fatalf("generated config did not preserve source data: %q", generatedConfig.Data["krkn-ai.yaml"])
	}
	if len(generatedConfig.OwnerReferences) != 1 || generatedConfig.OwnerReferences[0].Name != aiRun.Name {
		t.Fatalf("generated config is not owned by KrknAIRun: %+v", generatedConfig.OwnerReferences)
	}

	var pod corev1.Pod
	if err := fakeClient.Get(context.Background(), client.ObjectKey{
		Name: "ai-run-run-one", Namespace: "operator",
	}, &pod); err != nil {
		t.Fatalf("failed to get orchestrator pod: %v", err)
	}
	if pod.Spec.Containers[0].Env[1].Value != "/input/krkn-ai.yaml" {
		t.Fatalf("unexpected config file path: %q", pod.Spec.Containers[0].Env[1].Value)
	}
	var configVolume *corev1.Volume
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].Name == "config" {
			configVolume = &pod.Spec.Volumes[i]
			break
		}
	}
	if configVolume == nil || configVolume.ConfigMap == nil || configVolume.ConfigMap.Name != "ai-run-one-config" {
		t.Fatalf("orchestrator pod does not mount generated config: %+v", configVolume)
	}
}

func TestEnsureProvisionedRejectsMissingRunConfigMapKey(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add core scheme: %v", err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add krkn scheme: %v", err)
	}
	aiRun := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "run-one", Namespace: "operator"},
		Spec: krknv1alpha1.KrknAIRunSpec{
			TargetRequestID: "self",
			TargetClusters:  map[string][]string{"krkn-operator": {"self"}},
			ConfigMapName:   "config-one",
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		aiRun,
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "config-one", Namespace: "operator"},
			Data:       map[string]string{"other.yaml": "config"},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "self", Namespace: "operator"},
			Data: map[string][]byte{
				"managed-clusters": []byte(`{"krkn-operator":{"self":{"kubeconfig":""}}}`),
			},
		},
	).Build()
	reconciler := &KrknAIRunReconciler{Client: fakeClient, Scheme: scheme, Namespace: "operator"}

	err := reconciler.ensureProvisioned(context.Background(), aiRun)
	if err == nil || !strings.Contains(err.Error(), `does not contain non-empty key "krkn-ai.yaml"`) {
		t.Fatalf("expected missing config key error, got %v", err)
	}
}
