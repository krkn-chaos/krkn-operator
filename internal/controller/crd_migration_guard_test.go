package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/crdmigration"
)

func TestScenarioRunReconcileSkipsPendingMigration(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core API scheme: %v", err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add API scheme: %v", err)
	}
	run := &krknv1alpha1.KrknScenarioRun{ObjectMeta: metav1.ObjectMeta{
		Name:      "scenario-run",
		Namespace: "default",
	}}
	guard := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: crdmigration.GuardConfigMapName, Namespace: run.Namespace}}
	client := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(run, guard).Build()
	reconciler := &KrknScenarioRunReconciler{Client: client}

	key := types.NamespacedName{Name: run.Name, Namespace: run.Namespace}
	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key}); err != nil {
		t.Fatalf("reconcile staged ScenarioRun: %v", err)
	}
	got := &krknv1alpha1.KrknScenarioRun{}
	if err := client.Get(context.Background(), key, got); err != nil {
		t.Fatalf("get ScenarioRun: %v", err)
	}
	if got.Status.Phase != "" {
		t.Fatalf("staged ScenarioRun phase = %q, want no reconciliation", got.Status.Phase)
	}
}

func TestGraphRunReconcileSkipsPendingMigration(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core API scheme: %v", err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add API scheme: %v", err)
	}
	run := &krknv1alpha1.KrknGraphRun{ObjectMeta: metav1.ObjectMeta{
		Name:      "graph-run",
		Namespace: "default",
	}}
	guard := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: crdmigration.GuardConfigMapName, Namespace: run.Namespace}}
	client := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(run, guard).Build()
	reconciler := &KrknGraphRunReconciler{Client: client}

	key := types.NamespacedName{Name: run.Name, Namespace: run.Namespace}
	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key}); err != nil {
		t.Fatalf("reconcile staged GraphRun: %v", err)
	}
	got := &krknv1alpha1.KrknGraphRun{}
	if err := client.Get(context.Background(), key, got); err != nil {
		t.Fatalf("get GraphRun: %v", err)
	}
	if len(got.Finalizers) != 0 {
		t.Fatalf("staged GraphRun finalizers = %v, want no reconciliation", got.Finalizers)
	}
}
