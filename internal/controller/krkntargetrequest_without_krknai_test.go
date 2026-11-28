package controller

import (
	"context"
	"testing"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// withoutKrknAICRD answers a KrknAIRun list as the API server does when the
// Krkn-AI CRD is not installed, the default for Helm installs.
var withoutKrknAICRD = interceptor.Funcs{List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
	if _, ok := list.(*krknv1alpha1.KrknAIRunList); ok {
		return &meta.NoKindMatchError{
			GroupKind:        schema.GroupKind{Group: krknv1alpha1.GroupVersion.Group, Kind: "KrknAIRun"},
			SearchedVersions: []string{krknv1alpha1.GroupVersion.Version},
		}
	}
	return c.List(ctx, list, opts...)
}}

func TestCleanupOldTargetRequestsWithoutTheKrknAICRD(t *testing.T) {
	target := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "old-target",
			Namespace:         testOperatorNamespace,
			CreationTimestamp: metav1.NewTime(time.Now().Add(-2 * time.Hour)),
		},
		Status: krknv1alpha1.KrknTargetRequestStatus{Status: "Completed"},
	}
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	reconciler := &KrknTargetRequestReconciler{
		Client:            fake.NewClientBuilder().WithScheme(scheme).WithObjects(target).WithInterceptorFuncs(withoutKrknAICRD).Build(),
		Scheme:            scheme,
		OperatorName:      testOperatorName,
		OperatorNamespace: testOperatorNamespace,
	}

	deleted, err := reconciler.cleanupOldTargetRequests(context.Background())

	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	var remaining krknv1alpha1.KrknTargetRequest
	if err := reconciler.Get(context.Background(), types.NamespacedName{Name: target.Name, Namespace: target.Namespace}, &remaining); err == nil {
		t.Fatal("old completed target was not cleaned up")
	}
}
