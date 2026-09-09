package controller

import (
	"context"
	"encoding/base64"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newAIRunFixture(t *testing.T) (*KrknAIRunReconciler, client.Client, *krknv1alpha1.KrknAIRun) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add core scheme: %v", err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add Krkn scheme: %v", err)
	}
	aiRun := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "run-one", Namespace: "operator", UID: types.UID("run-one-uid")},
		Spec: krknv1alpha1.KrknAIRunSpec{
			TargetRequestID: "self",
			TargetClusters:  map[string][]string{"krkn-operator": {"self"}},
			ConfigMapName:   "config-one",
		},
	}
	kubeconfig := "apiVersion: v1\nkind: Config\n"
	managedClusters := `{"krkn-operator":{"self":{"kubeconfig":"` + base64.StdEncoding.EncodeToString([]byte(kubeconfig)) + `"}}}`
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		aiRun,
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "config-one", Namespace: "operator"},
			Data:       map[string]string{"krkn-ai.yaml": "genetic:\n  generations: 1\n"},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "self", Namespace: "operator"},
			Data:       map[string][]byte{"managed-clusters": []byte(managedClusters)},
		},
	).WithStatusSubresource(aiRun).Build()
	return &KrknAIRunReconciler{
		Client:                 fakeClient,
		APIReader:              fakeClient,
		Scheme:                 scheme,
		Namespace:              "operator",
		OrchestratorImage:      "registry.example/runner:test",
		ServiceImage:           "registry.example/service:test",
		ServiceURL:             "http://ai-service:8080",
		ServiceTokenSecretName: "ai-service-token",
	}, fakeClient, aiRun
}

func TestEnsureProvisionedUsesEmptyDirAndUploader(t *testing.T) {
	reconciler, fakeClient, aiRun := newAIRunFixture(t)
	if err := reconciler.ensureProvisioned(context.Background(), aiRun); err != nil {
		t.Fatalf("ensureProvisioned failed: %v", err)
	}
	var pod corev1.Pod
	if err := fakeClient.Get(context.Background(), client.ObjectKey{Name: "ai-run-run-one", Namespace: "operator"}, &pod); err != nil {
		t.Fatalf("failed to get run Pod: %v", err)
	}
	if len(pod.Spec.Containers) != 2 || pod.Spec.Containers[1].Name != "result-uploader" {
		t.Fatalf("expected orchestrator and result uploader containers: %+v", pod.Spec.Containers)
	}
	for _, volume := range pod.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil {
			t.Fatalf("run Pod must not mount a PVC: %+v", volume)
		}
	}
	var output *corev1.Volume
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].Name == "output" {
			output = &pod.Spec.Volumes[i]
		}
	}
	if output == nil || output.EmptyDir == nil {
		t.Fatalf("run output must be an EmptyDir: %+v", output)
	}
	for _, mount := range pod.Spec.Containers[1].VolumeMounts {
		if mount.Name == "config" || mount.Name == "kubeconfig" {
			t.Fatalf("uploader must not receive input credentials: %+v", pod.Spec.Containers[1].VolumeMounts)
		}
	}
}

func TestObserveRunSucceedsOnlyAfterArtifactsCommit(t *testing.T) {
	reconciler, fakeClient, aiRun := newAIRunFixture(t)
	aiRun.Status.OrchestratorPodName = "ai-run-run-one"
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: aiRun.Status.OrchestratorPodName, Namespace: aiRun.Namespace},
		Status: corev1.PodStatus{Phase: corev1.PodSucceeded, ContainerStatuses: []corev1.ContainerStatus{
			{Name: "orchestrator", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0}}},
			{Name: "result-uploader", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0}}},
		}},
	}
	if err := fakeClient.Create(context.Background(), pod); err != nil {
		t.Fatalf("failed to create Pod: %v", err)
	}
	if _, err := reconciler.observeRun(context.Background(), aiRun); err != nil {
		t.Fatalf("observeRun failed: %v", err)
	}
	if aiRun.Status.Phase != "Succeeded" || aiRun.Status.Conditions[0].Type != "ArtifactsCommitted" || aiRun.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected committed successful run, got %+v", aiRun.Status)
	}
}

func TestObserveRunPrefersOrchestratorFailure(t *testing.T) {
	reconciler, fakeClient, aiRun := newAIRunFixture(t)
	aiRun.Status.OrchestratorPodName = "ai-run-run-one"
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: aiRun.Status.OrchestratorPodName, Namespace: aiRun.Namespace},
		Status: corev1.PodStatus{Phase: corev1.PodFailed, ContainerStatuses: []corev1.ContainerStatus{
			{Name: "orchestrator", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, Reason: "RunnerFailed"}}},
			{Name: "result-uploader", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, Reason: "UploadFailed"}}},
		}},
	}
	if err := fakeClient.Create(context.Background(), pod); err != nil {
		t.Fatalf("failed to create Pod: %v", err)
	}
	if _, err := reconciler.observeRun(context.Background(), aiRun); err != nil {
		t.Fatalf("observeRun failed: %v", err)
	}
	if aiRun.Status.Phase != "Failed" || aiRun.Status.FailureReason != "RunnerFailed" || aiRun.Status.Conditions[0].Status != metav1.ConditionFalse {
		t.Fatalf("expected orchestrator failure and uncommitted artifacts, got %+v", aiRun.Status)
	}
}
