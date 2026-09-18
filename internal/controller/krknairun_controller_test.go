package controller

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
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
	apiReader := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "release-krkn-ai-orchestrator", Namespace: "operator"},
		},
	).Build()
	return &KrknAIRunReconciler{
		Client:                         fakeClient,
		APIReader:                      apiReader,
		Scheme:                         scheme,
		Namespace:                      "operator",
		OrchestratorImage:              "registry.example/runner:test",
		OrchestratorImagePullPolicy:    corev1.PullIfNotPresent,
		ServiceImage:                   "registry.example/service:test",
		ServiceImagePullPolicy:         corev1.PullNever,
		ServiceURL:                     "http://ai-service:8080",
		ServiceTokenSecretName:         "ai-service-token",
		OrchestratorServiceAccountName: "release-krkn-ai-orchestrator",
		ImagePullSecrets:               []corev1.LocalObjectReference{{Name: "private-registry"}},
	}, fakeClient, aiRun
}

func TestReconcileProvisionsPublicRunLifecycle(t *testing.T) {
	reconciler, fakeClient, aiRun := newAIRunFixture(t)
	request := ctrl.Request{NamespacedName: types.NamespacedName{Name: aiRun.Name, Namespace: aiRun.Namespace}}

	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("initial Reconcile failed: %v", err)
	}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("provisioning Reconcile failed: %v", err)
	}

	var persisted krknv1alpha1.KrknAIRun
	if err := fakeClient.Get(context.Background(), request.NamespacedName, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Status.Phase != aiRunPhaseProvisioning || persisted.Status.OrchestratorPodName == "" {
		t.Fatalf("run was not provisioned: %+v", persisted.Status)
	}
	var pod corev1.Pod
	if err := fakeClient.Get(context.Background(), types.NamespacedName{
		Name: persisted.Status.OrchestratorPodName, Namespace: aiRun.Namespace,
	}, &pod); err != nil {
		t.Fatalf("orchestrator Pod was not created: %v", err)
	}
}

func TestReconcileTerminalAndUnsupportedPhases(t *testing.T) {
	t.Run("terminal run does not requeue", func(t *testing.T) {
		reconciler, fakeClient, aiRun := newAIRunFixture(t)
		aiRun.Status.Phase = aiRunPhaseSucceeded
		if err := fakeClient.Status().Update(context.Background(), aiRun); err != nil {
			t.Fatal(err)
		}
		result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{Name: aiRun.Name, Namespace: aiRun.Namespace},
		})
		if err != nil || result != (ctrl.Result{}) {
			t.Fatalf("terminal Reconcile result=%+v error=%v", result, err)
		}
	})

	t.Run("unsupported phase is recorded as failure", func(t *testing.T) {
		reconciler, fakeClient, aiRun := newAIRunFixture(t)
		aiRun.Status.Phase = "Unknown"
		if err := fakeClient.Status().Update(context.Background(), aiRun); err != nil {
			t.Fatal(err)
		}
		if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{Name: aiRun.Name, Namespace: aiRun.Namespace},
		}); err != nil {
			t.Fatal(err)
		}
		var persisted krknv1alpha1.KrknAIRun
		if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: aiRun.Name, Namespace: aiRun.Namespace}, &persisted); err != nil {
			t.Fatal(err)
		}
		if persisted.Status.Phase != aiRunPhaseFailed || !strings.Contains(persisted.Status.FailureReason, "unsupported phase") {
			t.Fatalf("unsupported phase was not recorded: %+v", persisted.Status)
		}
	})
}

func TestAIResourceNameKeepsLongRunNamesDistinct(t *testing.T) {
	prefix := strings.Repeat("a", 70)
	first := aiResourceName("ai", prefix+"-one", "kubeconfig")
	second := aiResourceName("ai", prefix+"-two", "kubeconfig")
	if len(first) > 63 || len(second) > 63 || first == second {
		t.Fatalf("generated names must be bounded and distinct: %q %q", first, second)
	}
}

func assertAIRunPodRuntimeConfig(
	t *testing.T,
	pod *corev1.Pod,
	aiRun *krknv1alpha1.KrknAIRun,
	podName string,
) {
	t.Helper()
	if pod.Labels["krkn.dev/ai-run"] != aiRunLabelValue(aiRun.Name) ||
		pod.Labels["krkn.dev/component"] != "orchestrator" {
		t.Fatalf("unexpected orchestrator labels: %+v", pod.Labels)
	}
	if pod.Spec.ServiceAccountName != "release-krkn-ai-orchestrator" {
		t.Fatalf("orchestrator ServiceAccount = %q", pod.Spec.ServiceAccountName)
	}
	if len(pod.Spec.ImagePullSecrets) != 1 || pod.Spec.ImagePullSecrets[0].Name != "private-registry" {
		t.Fatalf("image pull secrets were not propagated: %+v", pod.Spec.ImagePullSecrets)
	}
	if pod.Spec.Containers[0].ImagePullPolicy != corev1.PullIfNotPresent ||
		pod.Spec.Containers[1].ImagePullPolicy != corev1.PullNever {
		t.Fatalf("image pull policies were not propagated: %+v", pod.Spec.Containers)
	}
	if len(pod.Spec.Containers) != 2 || pod.Spec.Containers[1].Name != "result-uploader" {
		t.Fatalf("expected orchestrator and result uploader containers: %+v", pod.Spec.Containers)
	}
	var uploaderOutputDir string
	for _, env := range pod.Spec.Containers[1].Env {
		if env.Name == "KRKNAI_OUTPUT_DIR" {
			uploaderOutputDir = env.Value
		}
	}
	if uploaderOutputDir != "/output" {
		t.Fatalf("uploader must receive the shared output parent directory, got %q", uploaderOutputDir)
	}
	var orchestratorPodName string
	for _, env := range pod.Spec.Containers[0].Env {
		if env.Name == "KRKNAI_ORCHESTRATOR_POD_NAME" {
			orchestratorPodName = env.Value
		}
	}
	if orchestratorPodName != podName {
		t.Fatalf("orchestrator must receive its Pod name, got %q want %q", orchestratorPodName, podName)
	}
}

func TestEnsureProvisionedUsesEmptyDirAndUploader(t *testing.T) {
	reconciler, fakeClient, aiRun := newAIRunFixture(t)
	if err := reconciler.ensureProvisioned(context.Background(), aiRun); err != nil {
		t.Fatalf("ensureProvisioned failed: %v", err)
	}
	var persisted krknv1alpha1.KrknAIRun
	if err := fakeClient.Get(context.Background(), client.ObjectKey{Name: aiRun.Name, Namespace: aiRun.Namespace}, &persisted); err != nil {
		t.Fatalf("failed to get persisted run: %v", err)
	}
	podName := persisted.Status.OrchestratorPodName
	if !strings.HasPrefix(podName, aiRun.Name+"-") {
		t.Fatalf("orchestrator Pod must use the run name and a unique suffix, got %q", podName)
	}
	var pod corev1.Pod
	if err := fakeClient.Get(context.Background(), client.ObjectKey{Name: podName, Namespace: "operator"}, &pod); err != nil {
		t.Fatalf("failed to get run Pod: %v", err)
	}
	assertAIRunPodRuntimeConfig(t, &pod, aiRun, podName)
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
	var config *corev1.Volume
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].Name == "config" {
			config = &pod.Spec.Volumes[i]
		}
	}
	if config == nil || config.ConfigMap == nil || config.ConfigMap.Name != "config-one" {
		t.Fatalf("run Pod must mount its source config directly: %+v", config)
	}
	var copiedConfig corev1.ConfigMap
	err := fakeClient.Get(context.Background(), client.ObjectKey{Name: "ai-run-run-one-config", Namespace: "operator"}, &copiedConfig)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("run must not create a copied config ConfigMap: %v", err)
	}
	for _, mount := range pod.Spec.Containers[1].VolumeMounts {
		if mount.Name == "config" || mount.Name == "kubeconfig" {
			t.Fatalf("uploader must not receive input credentials: %+v", pod.Spec.Containers[1].VolumeMounts)
		}
	}
}

func TestEnsureProvisionedMountsSelectedConfigKey(t *testing.T) {
	reconciler, fakeClient, aiRun := newAIRunFixture(t)
	const configKey = "operator-config.yaml"
	aiRun.Spec.ConfigMapKey = configKey
	var configMap corev1.ConfigMap
	if err := fakeClient.Get(context.Background(), client.ObjectKey{Name: aiRun.Spec.ConfigMapName, Namespace: aiRun.Namespace}, &configMap); err != nil {
		t.Fatalf("failed to get source ConfigMap: %v", err)
	}
	configMap.Data = map[string]string{configKey: "genetic:\n  generations: 1\n"}
	if err := fakeClient.Update(context.Background(), &configMap); err != nil {
		t.Fatalf("failed to update source ConfigMap: %v", err)
	}
	if err := reconciler.ensureProvisioned(context.Background(), aiRun); err != nil {
		t.Fatalf("ensureProvisioned failed: %v", err)
	}
	var persisted krknv1alpha1.KrknAIRun
	if err := fakeClient.Get(context.Background(), client.ObjectKey{Name: aiRun.Name, Namespace: aiRun.Namespace}, &persisted); err != nil {
		t.Fatalf("failed to get persisted run: %v", err)
	}
	var pod corev1.Pod
	if err := fakeClient.Get(context.Background(), client.ObjectKey{Name: persisted.Status.OrchestratorPodName, Namespace: aiRun.Namespace}, &pod); err != nil {
		t.Fatalf("failed to get run Pod: %v", err)
	}
	for _, mount := range pod.Spec.Containers[0].VolumeMounts {
		if mount.Name == "config" {
			if mount.SubPath != configKey {
				t.Fatalf("config mount subpath = %q, want %q", mount.SubPath, configKey)
			}
			return
		}
	}
	t.Fatal("orchestrator has no config mount")
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
	if aiRun.Status.Phase != aiRunPhaseSucceeded || aiRun.Status.Conditions[0].Type != "ArtifactsCommitted" || aiRun.Status.Conditions[0].Status != metav1.ConditionTrue {
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
	if aiRun.Status.Phase != aiRunPhaseFailed || aiRun.Status.FailureReason != "RunnerFailed" || aiRun.Status.Conditions[0].Status != metav1.ConditionFalse {
		t.Fatalf("expected orchestrator failure and uncommitted artifacts, got %+v", aiRun.Status)
	}
}

func TestObserveRunWaitsForUploaderAfterOrchestratorFailure(t *testing.T) {
	reconciler, fakeClient, aiRun := newAIRunFixture(t)
	aiRun.Status.OrchestratorPodName = "ai-run-run-one"
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: aiRun.Status.OrchestratorPodName, Namespace: aiRun.Namespace},
		Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{
			{Name: "orchestrator", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, Reason: "RunnerFailed"}}},
			{Name: "result-uploader", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
		}},
	}
	if err := fakeClient.Create(context.Background(), pod); err != nil {
		t.Fatalf("failed to create Pod: %v", err)
	}
	result, err := reconciler.observeRun(context.Background(), aiRun)
	if err != nil {
		t.Fatalf("observeRun failed: %v", err)
	}
	if result.RequeueAfter == 0 || aiRun.Status.Phase != aiRunPhaseRunning {
		t.Fatalf("controller must wait for diagnostic artifact upload, result=%+v status=%+v", result, aiRun.Status)
	}
	var retained corev1.Pod
	if err := fakeClient.Get(context.Background(), client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, &retained); err != nil {
		t.Fatalf("orchestrator Pod was deleted before artifact upload: %v", err)
	}
}

func TestObserveRunFailsWhenOrchestratorPodDisappears(t *testing.T) {
	reconciler, _, aiRun := newAIRunFixture(t)
	aiRun.Status.Phase = aiRunPhaseRunning
	aiRun.Status.OrchestratorPodName = "missing-pod"

	result, err := reconciler.observeRun(context.Background(), aiRun)
	if err != nil {
		t.Fatalf("observeRun failed: %v", err)
	}
	if result != (ctrl.Result{}) || aiRun.Status.Phase != aiRunPhaseFailed ||
		!strings.Contains(aiRun.Status.FailureReason, "disappeared") {
		t.Fatalf("missing Pod must fail the run, result=%+v status=%+v", result, aiRun.Status)
	}
}
