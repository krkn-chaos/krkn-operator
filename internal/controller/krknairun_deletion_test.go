package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func setAIRunDeleting(t *testing.T, fakeClient client.Client, aiRun *krknv1alpha1.KrknAIRun) {
	t.Helper()
	if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: aiRun.Name, Namespace: aiRun.Namespace}, aiRun); err != nil {
		t.Fatalf("read KrknAIRun before deletion: %v", err)
	}
	controllerutil.AddFinalizer(aiRun, krknAIRunFinalizer)
	if err := fakeClient.Update(context.Background(), aiRun); err != nil {
		t.Fatalf("mark KrknAIRun deleting: %v", err)
	}
	if err := fakeClient.Delete(context.Background(), aiRun); err != nil {
		t.Fatalf("request KrknAIRun deletion: %v", err)
	}
}

func runDeletionToCompletion(t *testing.T, reconciler *KrknAIRunReconciler, fakeClient client.Client, aiRun *krknv1alpha1.KrknAIRun) error {
	t.Helper()
	request := ctrl.Request{NamespacedName: types.NamespacedName{Name: aiRun.Name, Namespace: aiRun.Namespace}}
	for range 20 {
		_, err := reconciler.Reconcile(context.Background(), request)
		if err != nil {
			return err
		}
		var persisted krknv1alpha1.KrknAIRun
		err = fakeClient.Get(context.Background(), request.NamespacedName, &persisted)
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
	}
	return context.DeadlineExceeded
}

func TestReconcileDeletionCleansOnlyUIDOwnedResources(t *testing.T) {
	var calls atomic.Int32
	var gotPath, gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		gotPath = r.URL.Path
		gotAuthorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	reconciler, fakeClient, aiRun := newAIRunFixture(t)
	reconciler.ServiceURL = server.URL
	if err := fakeClient.Create(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: reconciler.ServiceTokenSecretName, Namespace: aiRun.Namespace},
		Data:       map[string][]byte{"token": []byte("test-token")},
	}); err != nil {
		t.Fatal(err)
	}
	aiRun.Status.Phase = aiRunPhaseRunning
	if err := fakeClient.Status().Update(context.Background(), aiRun); err != nil {
		t.Fatal(err)
	}
	setAIRunDeleting(t, fakeClient, aiRun)

	runOwner := *metav1.NewControllerRef(aiRun, krknv1alpha1.GroupVersion.WithKind("KrknAIRun"))
	child := &krknv1alpha1.KrknScenarioRun{ObjectMeta: metav1.ObjectMeta{
		Name: "child-run", Namespace: aiRun.Namespace, UID: types.UID("child-run-uid"),
		Labels: map[string]string{"krkn.dev/ai-run": aiRun.Name}, OwnerReferences: []metav1.OwnerReference{runOwner},
	}}
	if err := fakeClient.Create(context.Background(), child); err != nil {
		t.Fatal(err)
	}
	childOwner := *metav1.NewControllerRef(child, krknv1alpha1.GroupVersion.WithKind("KrknScenarioRun"))
	ownedPod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name: "child-pod", Namespace: aiRun.Namespace, UID: types.UID("child-pod-uid"),
		Labels: map[string]string{"krkn.dev/ai-run": aiRun.Name}, OwnerReferences: []metav1.OwnerReference{childOwner},
	}}
	ownedConfig := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: "child-config", Namespace: aiRun.Namespace, UID: types.UID("child-config-uid"), OwnerReferences: []metav1.OwnerReference{childOwner},
	}}
	orchestratorPod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name: aiOrchestratorPodName(aiRun.Name), Namespace: aiRun.Namespace, UID: types.UID("orchestrator-pod-uid"),
		Labels: map[string]string{"krkn.dev/ai-run": aiRun.Name}, OwnerReferences: []metav1.OwnerReference{runOwner},
	}}
	runConfig := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: "run-kubeconfig", Namespace: aiRun.Namespace, UID: types.UID("run-config-uid"), OwnerReferences: []metav1.OwnerReference{runOwner},
	}}
	foreignRun := &krknv1alpha1.KrknScenarioRun{ObjectMeta: metav1.ObjectMeta{
		Name: "foreign-child", Namespace: aiRun.Namespace, UID: types.UID("foreign-child-uid"),
		Labels: map[string]string{"krkn.dev/ai-run": aiRun.Name},
	}}
	foreignChildOwner := *metav1.NewControllerRef(foreignRun, krknv1alpha1.GroupVersion.WithKind("KrknScenarioRun"))
	foreignPod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name: "foreign-pod", Namespace: aiRun.Namespace, UID: types.UID("foreign-pod-uid"),
		Labels: map[string]string{"krkn.dev/ai-run": aiRun.Name}, OwnerReferences: []metav1.OwnerReference{foreignChildOwner},
	}}
	foreignRunPodOwner := metav1.OwnerReference{
		APIVersion: krknv1alpha1.GroupVersion.String(), Kind: "KrknAIRun", Name: aiRun.Name, UID: "different-run-uid",
	}
	foreignRunPod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name: "foreign-run-pod", Namespace: aiRun.Namespace, UID: types.UID("foreign-run-pod-uid"),
		Labels: map[string]string{"krkn.dev/ai-run": aiRun.Name}, OwnerReferences: []metav1.OwnerReference{foreignRunPodOwner},
	}}
	for _, object := range []client.Object{ownedPod, ownedConfig, orchestratorPod, runConfig, foreignRun, foreignPod, foreignRunPod} {
		if err := fakeClient.Create(context.Background(), object); err != nil {
			t.Fatal(err)
		}
	}

	if err := runDeletionToCompletion(t, reconciler, fakeClient, aiRun); err != nil {
		t.Fatalf("deletion did not complete: %v", err)
	}
	if calls.Load() != 1 || gotPath != "/v1/runs/run-one-uid" || gotAuthorization != "Bearer test-token" {
		t.Fatalf("artifact DELETE calls=%d path=%q authorization=%q", calls.Load(), gotPath, gotAuthorization)
	}
	for _, name := range []string{"child-pod", orchestratorPod.Name} {
		var pod corev1.Pod
		if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: aiRun.Namespace}, &pod); !apierrors.IsNotFound(err) {
			t.Errorf("owned Pod %q remains: %v", name, err)
		}
	}
	var childAfter krknv1alpha1.KrknScenarioRun
	if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: child.Name, Namespace: child.Namespace}, &childAfter); !apierrors.IsNotFound(err) {
		t.Errorf("owned KrknScenarioRun remains: %v", err)
	}
	for _, name := range []string{"child-config", "run-kubeconfig"} {
		var configMap corev1.ConfigMap
		if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: aiRun.Namespace}, &configMap); !apierrors.IsNotFound(err) {
			t.Errorf("owned ConfigMap %q remains: %v", name, err)
		}
	}
	var sourceConfig corev1.ConfigMap
	if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: "config-one", Namespace: aiRun.Namespace}, &sourceConfig); err != nil {
		t.Errorf("input ConfigMap was removed: %v", err)
	}
	var secret corev1.Secret
	if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: reconciler.ServiceTokenSecretName, Namespace: aiRun.Namespace}, &secret); err != nil {
		t.Errorf("service token Secret was removed: %v", err)
	}
	for _, key := range []types.NamespacedName{
		{Name: "foreign-pod", Namespace: aiRun.Namespace},
		{Name: "foreign-run-pod", Namespace: aiRun.Namespace},
		{Name: "foreign-child", Namespace: aiRun.Namespace},
	} {
		var object client.Object = &corev1.Pod{}
		if key.Name == "foreign-child" {
			object = &krknv1alpha1.KrknScenarioRun{}
		}
		if err := fakeClient.Get(context.Background(), key, object); err != nil {
			t.Errorf("unrelated resource %q was removed: %v", key.Name, err)
		}
	}
}

func TestReconcileDeletionRetainsFinalizerForArtifactServiceFailures(t *testing.T) {
	for _, statusCode := range []int{http.StatusInternalServerError, http.StatusNotFound, http.StatusMethodNotAllowed} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			var responseCode atomic.Int32
			responseCode.Store(int32(statusCode))
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.WriteHeader(int(responseCode.Load()))
			}))
			defer server.Close()
			reconciler, fakeClient, aiRun := newAIRunFixture(t)
			reconciler.ServiceURL = server.URL
			if err := fakeClient.Create(context.Background(), &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: reconciler.ServiceTokenSecretName, Namespace: aiRun.Namespace},
				Data:       map[string][]byte{"token": []byte("test-token")},
			}); err != nil {
				t.Fatal(err)
			}
			setAIRunDeleting(t, fakeClient, aiRun)

			err := runDeletionToCompletion(t, reconciler, fakeClient, aiRun)
			if err == nil || calls.Load() != 1 {
				t.Fatalf("service status %d: error=%v calls=%d", statusCode, err, calls.Load())
			}
			var persisted krknv1alpha1.KrknAIRun
			if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: aiRun.Name, Namespace: aiRun.Namespace}, &persisted); err != nil {
				t.Fatal(err)
			}
			if !controllerutil.ContainsFinalizer(&persisted, krknAIRunFinalizer) {
				t.Fatal("cleanup finalizer removed after an unsuccessful artifact DELETE")
			}
			condition := meta.FindStatusCondition(persisted.Status.Conditions, krknAIRunDeletionCondition)
			if condition == nil || condition.Reason != "CleanupFailed" {
				t.Fatalf("cleanup failure condition not recorded: %+v", condition)
			}

			responseCode.Store(http.StatusNoContent)
			if err := runDeletionToCompletion(t, reconciler, fakeClient, aiRun); err != nil {
				t.Fatalf("cleanup did not recover after service returned 204: %v", err)
			}
			if calls.Load() != 2 {
				t.Fatalf("expected retry after successful 204, got %d calls", calls.Load())
			}
			_, err = reconciler.Reconcile(context.Background(), ctrl.Request{
				NamespacedName: types.NamespacedName{Name: aiRun.Name, Namespace: aiRun.Namespace},
			})
			if err != nil || calls.Load() != 2 {
				t.Fatalf("completed deletion was not idempotent: error=%v calls=%d", err, calls.Load())
			}
		})
	}
}

func TestReconcileDeletionWaitsForOrchestratorAndScenarioPods(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	reconciler, fakeClient, aiRun := newAIRunFixture(t)
	reconciler.ServiceURL = server.URL
	if err := fakeClient.Create(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: reconciler.ServiceTokenSecretName, Namespace: aiRun.Namespace},
		Data:       map[string][]byte{"token": []byte("test-token")},
	}); err != nil {
		t.Fatal(err)
	}
	setAIRunDeleting(t, fakeClient, aiRun)
	runOwner := *metav1.NewControllerRef(aiRun, krknv1alpha1.GroupVersion.WithKind("KrknAIRun"))
	orchestratorPod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name: aiOrchestratorPodName(aiRun.Name), Namespace: aiRun.Namespace, UID: types.UID("orchestrator-pod-uid"),
		OwnerReferences: []metav1.OwnerReference{runOwner},
	}}
	if err := fakeClient.Create(context.Background(), orchestratorPod); err != nil {
		t.Fatal(err)
	}
	child := &krknv1alpha1.KrknScenarioRun{ObjectMeta: metav1.ObjectMeta{
		Name: "child-run", Namespace: aiRun.Namespace, UID: types.UID("child-run-uid"), OwnerReferences: []metav1.OwnerReference{runOwner},
	}}
	if err := fakeClient.Create(context.Background(), child); err != nil {
		t.Fatal(err)
	}
	childOwner := *metav1.NewControllerRef(child, krknv1alpha1.GroupVersion.WithKind("KrknScenarioRun"))
	childPod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name: "child-pod", Namespace: aiRun.Namespace, UID: types.UID("child-pod-uid"), OwnerReferences: []metav1.OwnerReference{childOwner},
	}}
	if err := fakeClient.Create(context.Background(), childPod); err != nil {
		t.Fatal(err)
	}
	heldPods := &holdChildPodDeleteClient{Client: fakeClient, heldPodName: orchestratorPod.Name}
	reconciler.Client = heldPods
	request := ctrl.Request{NamespacedName: types.NamespacedName{Name: aiRun.Name, Namespace: aiRun.Namespace}}
	for range 3 {
		if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
			t.Fatal(err)
		}
	}
	var childStillPresent krknv1alpha1.KrknScenarioRun
	if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: child.Name, Namespace: child.Namespace}, &childStillPresent); err != nil {
		t.Fatalf("child was deleted before orchestrator stopped: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatal("artifact purge started while the orchestrator Pod remained")
	}
	if err := fakeClient.Delete(context.Background(), orchestratorPod); err != nil {
		t.Fatal(err)
	}
	heldPods.heldPodName = "child-pod"
	for range 8 {
		if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("artifact purge started while a scenario producer Pod remained")
	}
	var persisted krknv1alpha1.KrknAIRun
	if err := fakeClient.Get(context.Background(), request.NamespacedName, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Status.Phase != "" {
		t.Fatalf("deleting KrknAIRun entered normal provisioning phase %q", persisted.Status.Phase)
	}
	var generatedConfig corev1.ConfigMap
	if err := fakeClient.Get(context.Background(), types.NamespacedName{
		Name: aiResourceName("ai", aiRun.Name, "kubeconfig"), Namespace: aiRun.Namespace,
	}, &generatedConfig); !apierrors.IsNotFound(err) {
		t.Fatalf("deleting KrknAIRun provisioned a kubeconfig ConfigMap: %v", err)
	}
	if !controllerutil.ContainsFinalizer(&persisted, krknAIRunFinalizer) {
		t.Fatal("cleanup finalizer removed while a scenario producer Pod remained")
	}
	if err := fakeClient.Delete(context.Background(), childPod); err != nil {
		t.Fatal(err)
	}
	heldPods.heldPodName = ""
	if err := runDeletionToCompletion(t, reconciler, fakeClient, aiRun); err != nil {
		t.Fatalf("cleanup did not resume after producer stopped: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected exactly one artifact purge after producers stopped, got %d calls", calls.Load())
	}
}

type holdChildPodDeleteClient struct {
	client.Client
	heldPodName string
}

func (c *holdChildPodDeleteClient) Delete(ctx context.Context, object client.Object, options ...client.DeleteOption) error {
	if pod, ok := object.(*corev1.Pod); ok && pod.Name == c.heldPodName {
		return nil
	}
	return c.Client.Delete(ctx, object, options...)
}

func TestReconcileDeletesTerminalRun(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	reconciler, fakeClient, aiRun := newAIRunFixture(t)
	reconciler.ServiceURL = server.URL
	if err := fakeClient.Create(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: reconciler.ServiceTokenSecretName, Namespace: aiRun.Namespace},
		Data:       map[string][]byte{"token": []byte("test-token")},
	}); err != nil {
		t.Fatal(err)
	}
	aiRun.Status.Phase = aiRunPhaseSucceeded
	if err := fakeClient.Status().Update(context.Background(), aiRun); err != nil {
		t.Fatal(err)
	}
	setAIRunDeleting(t, fakeClient, aiRun)
	if err := runDeletionToCompletion(t, reconciler, fakeClient, aiRun); err != nil {
		t.Fatalf("terminal run deletion did not complete: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected artifact deletion for terminal run, got %d calls", calls.Load())
	}
}
