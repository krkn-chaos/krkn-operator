package controller

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	"github.com/krkn-chaos/krknctl/pkg/verify"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func newAIRunFixture(t *testing.T) (*KrknAIRunReconciler, client.WithWatch, *krknv1alpha1.KrknAIRun) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add core scheme: %v", err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add Krkn scheme: %v", err)
	}
	aiRun := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "run-one", Namespace: "operator", UID: types.UID("run-one-uid"),
			Annotations: map[string]string{"example.com/keep": "yes"},
		},
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
		&krknv1alpha1.KrknTargetRequest{
			ObjectMeta: metav1.ObjectMeta{Name: "self", Namespace: "operator"},
			Status: krknv1alpha1.KrknTargetRequestStatus{
				Status: "Completed",
				TargetData: map[string][]krknv1alpha1.ClusterTarget{
					"krkn-operator": {{ClusterName: "self", ClusterAPIURL: "https://api.self.example:6443"}},
				},
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "config-one", Namespace: "operator"},
			Data:       map[string]string{"krkn-ai.yaml": "genetic:\n  generations: 1\nscenario:\n  pod-scenarios:\n    enable: true\n"},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "self", Namespace: "operator"},
			Data:       map[string][]byte{"managed-clusters": []byte(managedClusters)},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "release-krkn-ai-orchestrator", Namespace: "operator"},
		},
	).WithStatusSubresource(aiRun).Build()
	runPodConfig, err := KrknAIRunPodConfigFromEnv(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	return &KrknAIRunReconciler{
		Client:                         fakeClient,
		APIReader:                      fakeClient,
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
		RunPodConfig:                   runPodConfig,
		imageVerifier: func(_ context.Context, image string) (verify.VerifiedImage, error) {
			return verify.VerifiedImage{Digest: image}, nil
		},
	}, fakeClient, aiRun
}

func TestReconcileProvisionsPublicRunLifecycle(t *testing.T) {
	reconciler, fakeClient, aiRun := newAIRunFixture(t)
	request := ctrl.Request{NamespacedName: types.NamespacedName{Name: aiRun.Name, Namespace: aiRun.Namespace}}

	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("initial Reconcile failed: %v", err)
	}
	var persisted krknv1alpha1.KrknAIRun
	if err := fakeClient.Get(context.Background(), request.NamespacedName, &persisted); err != nil {
		t.Fatal(err)
	}
	if !controllerutil.ContainsFinalizer(&persisted, krknAIRunFinalizer) {
		t.Fatal("cleanup finalizer was not installed before provisioning")
	}
	var pods corev1.PodList
	if err := fakeClient.List(context.Background(), &pods, client.InNamespace(aiRun.Namespace)); err != nil {
		t.Fatal(err)
	}
	if len(pods.Items) != 0 {
		t.Fatalf("workloads were provisioned before finalizer installation: %+v", pods.Items)
	}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("status initialization Reconcile failed: %v", err)
	}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("provisioning Reconcile failed: %v", err)
	}

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

func TestReconcileDisabledRejectsUnstartedRuns(t *testing.T) {
	for _, phase := range []string{"", aiRunPhasePending} {
		t.Run("phase="+phase, func(t *testing.T) {
			reconciler, fakeClient, aiRun := newAIRunFixture(t)
			reconciler.RejectNewRuns = true
			aiRun.Status.Phase = phase
			if err := fakeClient.Status().Update(context.Background(), aiRun); err != nil {
				t.Fatal(err)
			}
			key := client.ObjectKeyFromObject(aiRun)
			if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key}); err != nil {
				t.Fatal(err)
			}
			var persisted krknv1alpha1.KrknAIRun
			if err := fakeClient.Get(context.Background(), key, &persisted); err != nil {
				t.Fatal(err)
			}
			if persisted.Status.Phase != aiRunPhaseFailed || !strings.Contains(persisted.Status.FailureReason, "disabled") {
				t.Fatalf("unstarted run was not rejected: %+v", persisted.Status)
			}
			assertAIRunTargetIdentity(t, &persisted, "https://api.self.example:6443")
			var pods corev1.PodList
			if err := fakeClient.List(context.Background(), &pods); err != nil {
				t.Fatal(err)
			}
			if len(pods.Items) != 0 || persisted.Status.OrchestratorPodName != "" {
				t.Fatal("disabled installation provisioned an orchestrator")
			}
		})
	}
}

func TestReconcileDisabledCompletesExistingRun(t *testing.T) {
	reconciler, fakeClient, aiRun := newAIRunFixture(t)
	reconciler.RejectNewRuns = true
	controllerutil.AddFinalizer(aiRun, krknAIRunFinalizer)
	if err := fakeClient.Update(context.Background(), aiRun); err != nil {
		t.Fatal(err)
	}
	aiRun.Status.Phase = aiRunPhaseRunning
	aiRun.Status.OrchestratorPodName = "started-orchestrator"
	if err := fakeClient.Status().Update(context.Background(), aiRun); err != nil {
		t.Fatal(err)
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: aiRun.Status.OrchestratorPodName, Namespace: aiRun.Namespace},
		Status: corev1.PodStatus{Phase: corev1.PodSucceeded, ContainerStatuses: []corev1.ContainerStatus{
			{Name: "orchestrator", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0}}},
			{Name: "result-uploader", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0}}},
		}},
	}
	setAIRunPodOwner(pod, aiRun)
	if err := fakeClient.Create(context.Background(), pod); err != nil {
		t.Fatal(err)
	}
	key := client.ObjectKeyFromObject(aiRun)
	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key}); err != nil {
		t.Fatal(err)
	}
	var completed krknv1alpha1.KrknAIRun
	if err := fakeClient.Get(context.Background(), key, &completed); err != nil {
		t.Fatal(err)
	}
	if completed.Status.Phase != aiRunPhaseSucceeded || completed.Status.FailureReason != "" {
		t.Fatalf("disabling interrupted an existing run: %+v", completed.Status)
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
		var persisted krknv1alpha1.KrknAIRun
		if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: aiRun.Name, Namespace: aiRun.Namespace}, &persisted); err != nil {
			t.Fatal(err)
		}
		if !controllerutil.ContainsFinalizer(&persisted, krknAIRunFinalizer) {
			t.Fatal("terminal run was not protected by the cleanup finalizer")
		}
		assertAIRunTargetIdentity(t, &persisted, "https://api.self.example:6443")
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

func TestReconcileRetainsTargetIdentityBeforeProvisioningFailure(t *testing.T) {
	reconciler, fakeClient, aiRun := newAIRunFixture(t)
	reconciler.APIReader = fake.NewClientBuilder().WithScheme(reconciler.Scheme).Build()
	ctx := context.Background()
	key := client.ObjectKeyFromObject(aiRun)
	request := ctrl.Request{NamespacedName: key}

	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("initial Reconcile failed: %v", err)
	}
	var persisted krknv1alpha1.KrknAIRun
	if err := fakeClient.Get(ctx, key, &persisted); err != nil {
		t.Fatal(err)
	}
	assertAIRunTargetIdentity(t, &persisted, "https://api.self.example:6443")

	var targetRequest krknv1alpha1.KrknTargetRequest
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "self", Namespace: "operator"}, &targetRequest); err != nil {
		t.Fatal(err)
	}
	targetRequest.Status.TargetData["krkn-operator"][0].ClusterAPIURL = "https://changed.example:6443"
	if err := fakeClient.Update(ctx, &targetRequest); err != nil {
		t.Fatalf("change live target API URL: %v", err)
	}
	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("status initialization Reconcile failed: %v", err)
	}
	if err := fakeClient.Get(ctx, key, &persisted); err != nil {
		t.Fatal(err)
	}
	assertAIRunTargetIdentity(t, &persisted, "https://api.self.example:6443")

	targetRequest = krknv1alpha1.KrknTargetRequest{ObjectMeta: metav1.ObjectMeta{Name: "self", Namespace: "operator"}}
	if err := fakeClient.Delete(ctx, &targetRequest); err != nil {
		t.Fatalf("remove target request: %v", err)
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "self", Namespace: "operator"}}
	if err := fakeClient.Delete(ctx, secret); err != nil {
		t.Fatalf("remove target Secret: %v", err)
	}
	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("provisioning failure Reconcile failed: %v", err)
	}
	if err := fakeClient.Get(ctx, key, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Status.Phase != aiRunPhaseFailed || !strings.Contains(persisted.Status.FailureReason, "ServiceAccount") {
		t.Fatalf("provisioning did not record the expected genuine failure: %+v", persisted.Status)
	}
	assertAIRunTargetIdentity(t, &persisted, "https://api.self.example:6443")

	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("terminal Reconcile after target removal failed: %v", err)
	}
	if err := fakeClient.Get(ctx, key, &persisted); err != nil {
		t.Fatal(err)
	}
	assertAIRunTargetIdentity(t, &persisted, "https://api.self.example:6443")
}

func assertAIRunTargetIdentity(t *testing.T, aiRun *krknv1alpha1.KrknAIRun, apiURL string) {
	t.Helper()
	if got := aiRun.Annotations[krknv1alpha1.KrknAIRunTargetAPIURLAnnotation]; got != apiURL {
		t.Fatalf("retained target API URL = %q, want %q", got, apiURL)
	}
	provider, cluster, err := oneTarget(aiRun.Spec.TargetClusters)
	if err != nil {
		t.Fatal(err)
	}
	wantBinding := files.HashLogicalName(aiRun.Spec.TargetRequestID + "\x00" + provider + "\x00" + cluster)
	if got := aiRun.Annotations[krknv1alpha1.KrknAIRunTargetBindingAnnotation]; got != wantBinding {
		t.Fatalf("retained target binding = %q, want %q", got, wantBinding)
	}
	if got := aiRun.Annotations["example.com/keep"]; got != "yes" {
		t.Fatalf("unrelated annotation was not preserved: %q", got)
	}
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
	var uploaderOutputDir, uploadInterval string
	for _, env := range pod.Spec.Containers[1].Env {
		switch env.Name {
		case "KRKNAI_OUTPUT_DIR":
			uploaderOutputDir = env.Value
		case "KRKNAI_UPLOAD_INTERVAL_SECONDS":
			uploadInterval = env.Value
		}
	}
	if uploaderOutputDir != "/output" {
		t.Fatalf("uploader must receive the shared output parent directory, got %q", uploaderOutputDir)
	}
	if uploadInterval != "10" {
		t.Fatalf("operator uploader interval = %q, want 10 seconds", uploadInterval)
	}
	var orchestratorPodName string
	for _, env := range pod.Spec.Containers[0].Env {
		switch env.Name {
		case "KRKNAI_ORCHESTRATOR_POD_NAME":
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
	for _, container := range pod.Spec.Containers {
		for _, name := range []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory, corev1.ResourceEphemeralStorage} {
			request, limit := container.Resources.Requests[name], container.Resources.Limits[name]
			if request.Sign() <= 0 || limit.Sign() <= 0 || request.Cmp(limit) > 0 {
				t.Fatalf("container %s has unbounded or invalid %s resources", container.Name, name)
			}
		}
	}
	for _, volume := range pod.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil {
			t.Fatalf("run Pod must not mount a PVC: %+v", volume)
		}
		if volume.EmptyDir != nil && (volume.EmptyDir.SizeLimit == nil || volume.EmptyDir.SizeLimit.Sign() <= 0) {
			t.Fatalf("run volume %s has no positive size limit", volume.Name)
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
	configMap.Data = map[string]string{configKey: "genetic:\n  generations: 1\nscenario:\n  pod-scenarios:\n    enable: true\n"}
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
	setAIRunPodOwner(pod, aiRun)
	if err := fakeClient.Create(context.Background(), pod); err != nil {
		t.Fatalf("failed to create Pod: %v", err)
	}
	if _, err := reconciler.observeRun(context.Background(), aiRun); err != nil {
		t.Fatalf("observeRun failed: %v", err)
	}
	artifactsCommitted := meta.FindStatusCondition(aiRun.Status.Conditions, "ArtifactsCommitted")
	if aiRun.Status.Phase != aiRunPhaseSucceeded || artifactsCommitted == nil || artifactsCommitted.Status != metav1.ConditionTrue {
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
	setAIRunPodOwner(pod, aiRun)
	if err := fakeClient.Create(context.Background(), pod); err != nil {
		t.Fatalf("failed to create Pod: %v", err)
	}
	if _, err := reconciler.observeRun(context.Background(), aiRun); err != nil {
		t.Fatalf("observeRun failed: %v", err)
	}
	artifactsCommitted := meta.FindStatusCondition(aiRun.Status.Conditions, "ArtifactsCommitted")
	if aiRun.Status.Phase != aiRunPhaseFailed || aiRun.Status.FailureReason != "RunnerFailed" ||
		artifactsCommitted == nil || artifactsCommitted.Status != metav1.ConditionFalse {
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
	setAIRunPodOwner(pod, aiRun)
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

func TestObserveRunFailsEvictedPodWithIncompleteContainerStatus(t *testing.T) {
	for _, statuses := range [][]corev1.ContainerStatus{
		nil,
		{{Name: "orchestrator", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 137, Reason: "OOMKilled"}}}},
	} {
		t.Run(fmt.Sprintf("containers=%d", len(statuses)), func(t *testing.T) {
			reconciler, fakeClient, aiRun := newAIRunFixture(t)
			aiRun.Status.Phase = aiRunPhaseRunning
			aiRun.Status.OrchestratorPodName = "evicted-run"
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: aiRun.Status.OrchestratorPodName, Namespace: aiRun.Namespace},
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed, Reason: "Evicted", Message: "Pod exceeded local ephemeral storage limit",
					ContainerStatuses: statuses,
				},
			}
			setAIRunPodOwner(pod, aiRun)
			if err := fakeClient.Create(context.Background(), pod); err != nil {
				t.Fatal(err)
			}
			result, err := reconciler.observeRun(context.Background(), aiRun)
			if err != nil {
				t.Fatal(err)
			}
			var persisted krknv1alpha1.KrknAIRun
			if err := fakeClient.Get(context.Background(), client.ObjectKeyFromObject(aiRun), &persisted); err != nil {
				t.Fatal(err)
			}
			if result != (ctrl.Result{}) || persisted.Status.Phase != aiRunPhaseFailed ||
				!strings.Contains(persisted.Status.FailureReason, pod.Status.Reason) ||
				!strings.Contains(persisted.Status.FailureReason, pod.Status.Message) ||
				persisted.Status.CompletionTime == nil {
				t.Fatalf("evicted run did not terminate: result=%+v status=%+v", result, persisted.Status)
			}
			condition := meta.FindStatusCondition(persisted.Status.Conditions, "ArtifactsCommitted")
			if condition == nil || condition.Status == metav1.ConditionTrue {
				t.Fatalf("evicted run must not claim committed artifacts: %+v", condition)
			}
			finalized := meta.FindStatusCondition(persisted.Status.Conditions, "ArtifactsFinalized")
			if finalized == nil || finalized.Status != metav1.ConditionTrue {
				t.Fatalf("terminal Pod without uploader status should settle finalization: %+v", finalized)
			}
		})
	}
}
