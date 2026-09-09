package controller

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krknctl/pkg/verify"
)

func TestGraphRunInvalidImageSignatureFailsWithoutRetry(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add scheme: %v", err)
	}
	graphRun := &krknv1alpha1.KrknGraphRun{
		ObjectMeta: metav1.ObjectMeta{Name: "graph-run", Namespace: "default"},
	}
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(graphRun).Build()
	reconciler := &KrknGraphRunReconciler{Client: fakeClient}

	result, err := reconciler.updateStatusWithError(context.Background(), graphRun, &InvalidImageSignatureError{
		Image:  "quay.io/krkn-chaos/krkn-hub:cpu-hog",
		Status: verify.SignatureUnsigned,
	})
	if err != nil {
		t.Fatalf("expected terminal signature failure not to requeue, got %v", err)
	}
	if result != (ctrl.Result{}) {
		t.Fatalf("expected empty reconcile result, got %+v", result)
	}
	if graphRun.Status.Phase != "Failed" || graphRun.Status.Message == "" {
		t.Fatalf("expected failed phase and message, got phase=%q message=%q", graphRun.Status.Phase, graphRun.Status.Message)
	}
}

func TestScenarioRunInvalidImageSignatureIsNotRetried(t *testing.T) {
	reconciler := &KrknScenarioRunReconciler{}
	job := &krknv1alpha1.ClusterJobStatus{Phase: "Failed", FailureReason: "InvalidImageSignature"}
	if reconciler.shouldRetryJob(job, 3) {
		t.Fatal("expected invalid signature failure not to be retried")
	}
}
