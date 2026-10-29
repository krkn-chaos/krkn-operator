package controller

import (
	"context"
	"testing"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestScenarioRunReconcileDoesNotProvisionWhileDeleting(t *testing.T) {
	scheme := newTestScheme(t)
	now := metav1.NewTime(time.Now())
	scenarioRun := &krknv1alpha1.KrknScenarioRun{ObjectMeta: metav1.ObjectMeta{
		Name: "deleting-run", Namespace: "default", UID: types.UID("scenario-run-uid"),
		DeletionTimestamp: &now, Finalizers: []string{"krkn.krkn-chaos.dev/test-finalizer"},
	}}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(scenarioRun).
		WithStatusSubresource(scenarioRun).Build()
	reconciler := &KrknScenarioRunReconciler{Client: fakeClient, Scheme: scheme}

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: scenarioRun.Name, Namespace: scenarioRun.Namespace},
	})
	if err != nil || result != (ctrl.Result{}) {
		t.Fatalf("deleting ScenarioRun reconcile result=%+v error=%v", result, err)
	}
	var persisted krknv1alpha1.KrknScenarioRun
	if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: scenarioRun.Name, Namespace: scenarioRun.Namespace}, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Status.Phase != "" {
		t.Fatalf("deleting ScenarioRun status was initialized: %+v", persisted.Status)
	}
	var pods corev1.PodList
	if err := fakeClient.List(context.Background(), &pods); err != nil {
		t.Fatal(err)
	}
	if len(pods.Items) != 0 {
		t.Fatalf("deleting ScenarioRun provisioned Pods: %+v", pods.Items)
	}
}
