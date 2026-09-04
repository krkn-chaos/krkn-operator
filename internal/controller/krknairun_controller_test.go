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
			PVCName:             "shared-rwx",
			OrchestratorPodName: "ai-run-run-one",
		},
	}
	resultsPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "shared-rwx", Namespace: "operator"},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		aiRun,
		resultsPVC,
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
		APIReader:         fakeClient,
		Scheme:            scheme,
		Namespace:         "operator",
		OrchestratorImage: "registry.example/krkn-ai-operator:test",
		ResultsPVCName:    "shared-rwx",
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
	var resultsVolume *corev1.Volume
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].Name == "results" {
			resultsVolume = &pod.Spec.Volumes[i]
			break
		}
	}
	if resultsVolume == nil || resultsVolume.PersistentVolumeClaim == nil ||
		resultsVolume.PersistentVolumeClaim.ClaimName != "shared-rwx" {
		t.Fatalf("orchestrator pod does not mount shared results PVC: %+v", resultsVolume)
	}
	var runUID string
	for _, env := range pod.Spec.Containers[0].Env {
		if env.Name == "KRKNAI_RUN_UID" {
			runUID = env.Value
			break
		}
	}
	if runUID != string(aiRun.UID) {
		t.Fatalf("unexpected KRKNAI_RUN_UID: %q", runUID)
	}
	var storedPVC corev1.PersistentVolumeClaim
	if err := fakeClient.Get(context.Background(), client.ObjectKey{
		Name: "shared-rwx", Namespace: "operator",
	}, &storedPVC); err != nil {
		t.Fatalf("failed to get shared results PVC: %v", err)
	}
	if len(storedPVC.OwnerReferences) != 0 {
		t.Fatalf("shared results PVC unexpectedly owned by KrknAIRun: %+v", storedPVC.OwnerReferences)
	}
	var generatedPVC corev1.PersistentVolumeClaim
	if err := fakeClient.Get(context.Background(), client.ObjectKey{
		Name: "ai-run-one-results", Namespace: "operator",
	}, &generatedPVC); !apierrors.IsNotFound(err) {
		t.Fatalf("unexpected per-run results PVC: %v", err)
	}
}

func newResultsPVCFixture(t *testing.T, includePVC bool, accessModes []corev1.PersistentVolumeAccessMode) (*KrknAIRunReconciler, client.Client, *krknv1alpha1.KrknAIRun) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add core scheme: %v", err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add krkn scheme: %v", err)
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
	managedClusters := `{"krkn-operator":{"self":{"kubeconfig":"` +
		base64.StdEncoding.EncodeToString([]byte(kubeconfig)) + `"}}}`
	objects := []client.Object{
		aiRun,
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "config-one", Namespace: "operator"},
			Data:       map[string]string{"krkn-ai.yaml": "genetic:\n  generations: 1\n"},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "self", Namespace: "operator"},
			Data:       map[string][]byte{"managed-clusters": []byte(managedClusters)},
		},
	}
	if includePVC {
		objects = append(objects, &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "shared-rwx", Namespace: "operator"},
			Spec:       corev1.PersistentVolumeClaimSpec{AccessModes: accessModes},
		})
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).WithStatusSubresource(aiRun).Build()
	reconciler := &KrknAIRunReconciler{
		Client:            fakeClient,
		APIReader:         fakeClient,
		Scheme:            scheme,
		Namespace:         "operator",
		ResultsPVCName:    "shared-rwx",
		OrchestratorImage: "registry.example/krkn-ai-operator:test",
	}
	return reconciler, fakeClient, aiRun
}

func TestEnsureProvisionedRejectsMissingResultsPVC(t *testing.T) {
	reconciler, fakeClient, aiRun := newResultsPVCFixture(t, false, nil)

	err := reconciler.ensureProvisioned(context.Background(), aiRun)
	if err == nil || err.Error() != `results PVC "shared-rwx" not found in namespace "operator"` {
		t.Fatalf("expected missing results PVC error, got %v", err)
	}
	var generatedConfig corev1.ConfigMap
	if err := fakeClient.Get(context.Background(), client.ObjectKey{
		Name: "ai-run-run-one-config", Namespace: "operator",
	}, &generatedConfig); !apierrors.IsNotFound(err) {
		t.Fatalf("unexpected generated ConfigMap: %v", err)
	}
	var pod corev1.Pod
	if err := fakeClient.Get(context.Background(), client.ObjectKey{
		Name: "ai-run-run-one", Namespace: "operator",
	}, &pod); !apierrors.IsNotFound(err) {
		t.Fatalf("unexpected orchestrator Pod: %v", err)
	}
}

func TestEnsureProvisionedRejectsNonRWXResultsPVC(t *testing.T) {
	reconciler, fakeClient, aiRun := newResultsPVCFixture(
		t, true, []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
	)

	err := reconciler.ensureProvisioned(context.Background(), aiRun)
	if err == nil || err.Error() != `results PVC "shared-rwx" must declare ReadWriteMany access mode` {
		t.Fatalf("expected non-RWX results PVC error, got %v", err)
	}
	var generatedConfig corev1.ConfigMap
	if err := fakeClient.Get(context.Background(), client.ObjectKey{
		Name: "ai-run-run-one-config", Namespace: "operator",
	}, &generatedConfig); !apierrors.IsNotFound(err) {
		t.Fatalf("unexpected generated ConfigMap: %v", err)
	}
	var pod corev1.Pod
	if err := fakeClient.Get(context.Background(), client.ObjectKey{
		Name: "ai-run-run-one", Namespace: "operator",
	}, &pod); !apierrors.IsNotFound(err) {
		t.Fatalf("unexpected orchestrator Pod: %v", err)
	}
}

func TestEnsureProvisionedUsesExplicitPVCName(t *testing.T) {
	reconciler, fakeClient, aiRun := newResultsPVCFixture(
		t, true, []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
	)
	aiRun.Spec.Storage = &krknv1alpha1.KrknAIRunStorageSpec{PVCName: "shared-rwx"}
	aiRun.Status = krknv1alpha1.KrknAIRunStatus{
		Phase: "Provisioning", PVCName: "shared-rwx", OrchestratorPodName: "ai-run-run-one",
	}
	if err := reconciler.ensureProvisioned(context.Background(), aiRun); err != nil {
		t.Fatalf("ensureProvisioned failed: %v", err)
	}
	if aiRun.Status.PVCName != "shared-rwx" {
		t.Fatalf("unexpected status PVC name: %q", aiRun.Status.PVCName)
	}
	var pod corev1.Pod
	if err := fakeClient.Get(context.Background(), client.ObjectKey{
		Name: "ai-run-run-one", Namespace: "operator",
	}, &pod); err != nil {
		t.Fatalf("failed to get orchestrator pod: %v", err)
	}
	var resultsVolume *corev1.Volume
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].Name == "results" {
			resultsVolume = &pod.Spec.Volumes[i]
			break
		}
	}
	if resultsVolume == nil || resultsVolume.PersistentVolumeClaim == nil ||
		resultsVolume.PersistentVolumeClaim.ClaimName != "shared-rwx" {
		t.Fatalf("unexpected pod PVC volume: %+v", resultsVolume)
	}
	var selectedPVC corev1.PersistentVolumeClaim
	if err := fakeClient.Get(context.Background(), client.ObjectKey{
		Name: "shared-rwx", Namespace: "operator",
	}, &selectedPVC); err != nil {
		t.Fatalf("failed to get selected PVC: %v", err)
	}
	if len(selectedPVC.OwnerReferences) != 0 {
		t.Fatalf("explicit PVC unexpectedly owned by KrknAIRun: %+v", selectedPVC.OwnerReferences)
	}
}

func TestEnsureProvisionedCreatesDedicatedResultsPVC(t *testing.T) {
	reconciler, fakeClient, aiRun := newResultsPVCFixture(t, false, nil)
	reconciler.ResultsStorageMode = "dedicated"
	reconciler.ResultsStorageClassName = "gp3-csi"
	aiRun.Status = krknv1alpha1.KrknAIRunStatus{
		Phase: "Provisioning", PVCName: "ai-run-one-results", OrchestratorPodName: "ai-run-run-one",
	}
	if err := reconciler.ensureProvisioned(context.Background(), aiRun); err != nil {
		t.Fatalf("ensureProvisioned failed: %v", err)
	}

	var pvc corev1.PersistentVolumeClaim
	if err := fakeClient.Get(context.Background(), client.ObjectKey{
		Name: "ai-run-one-results", Namespace: "operator",
	}, &pvc); err != nil {
		t.Fatalf("failed to get dedicated results PVC: %v", err)
	}
	if len(pvc.Spec.AccessModes) != 1 || pvc.Spec.AccessModes[0] != corev1.ReadWriteOnce {
		t.Fatalf("unexpected access modes: %v", pvc.Spec.AccessModes)
	}
	if pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName != "gp3-csi" {
		t.Fatalf("unexpected storage class: %v", pvc.Spec.StorageClassName)
	}
	if !metav1.IsControlledBy(&pvc, aiRun) {
		t.Fatalf("dedicated results PVC is not owned by KrknAIRun")
	}
	if aiRun.Status.PVCName != "ai-run-one-results" {
		t.Fatalf("unexpected status PVC name: %q", aiRun.Status.PVCName)
	}
}

func TestEnsureProvisionedRunStorageOverridesInstallationMode(t *testing.T) {
	reconciler, fakeClient, aiRun := newResultsPVCFixture(t, false, nil)
	aiRun.Spec.Storage = &krknv1alpha1.KrknAIRunStorageSpec{
		StorageClassName: "gp3-csi",
		AccessMode:       string(corev1.ReadWriteOnce),
		Size:             "3Gi",
	}
	aiRun.Status = krknv1alpha1.KrknAIRunStatus{
		Phase: "Provisioning", PVCName: "ai-run-one-results", OrchestratorPodName: "ai-run-run-one",
	}

	if err := reconciler.ensureProvisioned(context.Background(), aiRun); err != nil {
		t.Fatalf("ensureProvisioned failed: %v", err)
	}

	var pvc corev1.PersistentVolumeClaim
	if err := fakeClient.Get(context.Background(), client.ObjectKey{
		Name: "ai-run-one-results", Namespace: "operator",
	}, &pvc); err != nil {
		t.Fatalf("failed to get run-specific results PVC: %v", err)
	}
	if pvc.Spec.AccessModes[0] != corev1.ReadWriteOnce ||
		pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName != "gp3-csi" {
		t.Fatalf("run storage did not override installation mode: %+v", pvc.Spec)
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
