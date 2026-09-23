package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	krknctlprovider "github.com/krkn-chaos/krknctl/pkg/provider"
	krknctlmodels "github.com/krkn-chaos/krknctl/pkg/provider/models"
	"github.com/krkn-chaos/krknctl/pkg/verify"
)

func TestScenarioTagForImagePreservesVerifiedDigest(t *testing.T) {
	reference := publicReference("cpu-hog")
	image := "quay.io/krkn-chaos/krkn-hub-multiarch@sha256:verified"
	tag := scenarioTagForImage(reference, image)
	if tag.Digest == nil || *tag.Digest != "sha256:verified" {
		t.Fatalf("scenarioTagForImage() digest = %v, want sha256:verified", tag.Digest)
	}
	if got := krknctlprovider.ImageReference("quay.io/krkn-chaos/krkn-hub-multiarch", tag); got != image {
		t.Fatalf("image reference = %q, want %q", got, image)
	}
}

func TestResolvePrivateImageDigestUsesManifestDigest(t *testing.T) {
	const expectedDigest = "sha256:private-manifest"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/scenarios/manifests/cpu-hog" {
			t.Fatalf("manifest path = %q", r.URL.Path)
		}
		if user, password, ok := r.BasicAuth(); !ok || user != "user" || password != "password" {
			t.Fatalf("unexpected registry authentication")
		}
		w.Header().Set("Docker-Content-Digest", expectedDigest)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	user, password := "user", "password"
	registry := &krknctlmodels.RegistryV2{
		RegistryURL:        strings.TrimPrefix(server.URL, "http://"),
		ScenarioRepository: "scenarios",
		Insecure:           true,
		Username:           &user,
		Password:           &password,
	}
	digest, err := resolvePrivateImageDigest(context.Background(), "cpu-hog", registry)
	if err != nil {
		t.Fatalf("resolvePrivateImageDigest() error = %v", err)
	}
	if digest != expectedDigest {
		t.Fatalf("digest = %q, want %q", digest, expectedDigest)
	}
}

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
