package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// newHandlerWithoutKrknAICRD answers a KrknAIRun list as the API server does
// when the Krkn-AI CRD is not installed, the default for Helm installs.
func newHandlerWithoutKrknAICRD(t *testing.T, objects ...client.Object) (*Handler, client.Client) {
	t.Helper()
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	k8sClient := fakeclient.NewClientBuilder().WithScheme(scheme).
		WithObjects(append(objects, TestJWTSecret("default"))...).
		WithInterceptorFuncs(interceptor.Funcs{List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			if _, ok := list.(*krknv1alpha1.KrknAIRunList); ok {
				return &meta.NoKindMatchError{
					GroupKind:        schema.GroupKind{Group: krknv1alpha1.GroupVersion.Group, Kind: "KrknAIRun"},
					SearchedVersions: []string{krknv1alpha1.GroupVersion.Version},
				}
			}
			return c.List(ctx, list, opts...)
		}}).Build()
	return NewTestHandler(k8sClient, fake.NewSimpleClientset(), "default", "localhost:50051"), k8sClient
}

func TestDeleteTargetByUUID_WithoutTheKrknAICRD(t *testing.T) {
	target := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "test-uuid-delete", Namespace: "default"},
		Spec:       krknv1alpha1.KrknTargetRequestSpec{UUID: "test-uuid-delete"},
		Status:     krknv1alpha1.KrknTargetRequestStatus{Status: "Completed"},
	}
	handler, k8sClient := newHandlerWithoutKrknAICRD(t, target)

	req := httptest.NewRequest(http.MethodDelete, TargetsPath+"/test-uuid-delete", nil).WithContext(createAdminContext())
	w := httptest.NewRecorder()
	handler.DeleteTargetByUUID(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d: %s", w.Code, http.StatusNoContent, w.Body.String())
	}
	var deleted krknv1alpha1.KrknTargetRequest
	err := k8sClient.Get(context.Background(), types.NamespacedName{Name: target.Name, Namespace: target.Namespace}, &deleted)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("target request still there or unexpected error: %v", err)
	}
}

func TestKrknAIConfigInUse_WithoutTheKrknAICRD(t *testing.T) {
	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: "ai-config", Namespace: "default",
		Labels: map[string]string{files.FilePurposeLabel: files.FilePurposeKrknAIConfig},
	}}
	handler, _ := newHandlerWithoutKrknAICRD(t, configMap)

	inUse, err := handler.krknAIConfigInUse(context.Background(), configMap)

	if err != nil {
		t.Fatalf("krknAIConfigInUse failed: %v", err)
	}
	if inUse {
		t.Fatal("a configuration is in use although no KrknAIRun can exist")
	}
}
