package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/krkn-chaos/krkn-operator/pkg/signatureverification"
	"github.com/krkn-chaos/krknctl/pkg/verify"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestReconcileAIImageSignatures(t *testing.T) {
	cases := []struct {
		name      string
		component string
		verifyErr error
		disabled  bool
		retryable bool
	}{
		{name: "signed images"},
		{name: "unsigned orchestrator", component: "orchestrator", verifyErr: verify.ErrUnsigned},
		{name: "untrusted orchestrator", component: "orchestrator", verifyErr: verify.ErrInvalidSignature},
		{name: "unsigned uploader", component: "result-uploader", verifyErr: verify.ErrUnsigned},
		{name: "untrusted uploader", component: "result-uploader", verifyErr: verify.ErrInvalidSignature},
		{name: "orchestrator registry unavailable", component: "orchestrator", verifyErr: verify.ErrRegistryUnreachable, retryable: true},
		{name: "uploader registry unavailable", component: "result-uploader", verifyErr: verify.ErrRegistryUnreachable, retryable: true},
		{name: "disabled allows unsigned orchestrator", component: "orchestrator", verifyErr: verify.ErrUnsigned, disabled: true},
		{name: "disabled allows untrusted uploader", component: "result-uploader", verifyErr: verify.ErrInvalidSignature, disabled: true},
		{name: "disabled allows unavailable verification", component: "result-uploader", verifyErr: verify.ErrRegistryUnreachable, disabled: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, c, run := newPendingAIRunFixture(t)
			ctx := context.Background()
			if tc.disabled {
				if err := signatureverification.SetEnabled(ctx, c, run.Namespace, false); err != nil {
					t.Fatal(err)
				}
			}
			images := map[string]string{
				"orchestrator":    r.OrchestratorImage,
				"result-uploader": r.ServiceImage,
			}
			digests := map[string]string{
				r.OrchestratorImage: "registry.example/runner@sha256:" + strings.Repeat("a", 64),
				r.ServiceImage:      "registry.example/service@sha256:" + strings.Repeat("b", 64),
			}
			verificationErr := tc.verifyErr
			r.imageVerifier = func(_ context.Context, image string) (verify.VerifiedImage, error) {
				if image == images[tc.component] && verificationErr != nil {
					return verify.VerifiedImage{}, fmt.Errorf("verification failed: %w", verificationErr)
				}
				return verify.VerifiedImage{Digest: digests[image]}, nil
			}
			request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(run)}
			result, err := r.Reconcile(ctx, request)
			if tc.retryable {
				if !errors.Is(err, tc.verifyErr) {
					t.Fatalf("verification error = %v, want %v", err, tc.verifyErr)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if err := c.Get(ctx, request.NamespacedName, run); err != nil {
				t.Fatal(err)
			}
			blocked := tc.verifyErr != nil && !tc.disabled
			if blocked {
				var pods corev1.PodList
				if err := c.List(ctx, &pods, client.InNamespace(run.Namespace)); err != nil {
					t.Fatal(err)
				}
				var credentials corev1.ConfigMapList
				if err := c.List(ctx, &credentials, client.InNamespace(run.Namespace),
					client.MatchingLabels{"krkn.dev/ai-run": aiRunLabelValue(run.Name)}); err != nil {
					t.Fatal(err)
				}
				if len(pods.Items) != 0 || len(credentials.Items) != 0 || run.Status.OrchestratorPodName != "" {
					t.Fatal("failed verification provisioned a Pod or target credentials")
				}
				if !tc.retryable {
					if result != (ctrl.Result{}) || run.Status.Phase != aiRunPhaseFailed || run.Status.CompletionTime == nil ||
						!strings.Contains(run.Status.FailureReason, images[tc.component]) {
						t.Fatalf("trust failure was not recorded as terminal: %+v", run.Status)
					}
					return
				}
				if run.Status.Phase != aiRunPhasePending || run.Status.CompletionTime != nil || run.Status.FailureReason != "" {
					t.Fatalf("operational failure became terminal: %+v", run.Status)
				}
				verificationErr = nil
				if _, err := r.Reconcile(ctx, request); err != nil {
					t.Fatalf("verification retry failed: %v", err)
				}
				if err := c.Get(ctx, request.NamespacedName, run); err != nil {
					t.Fatal(err)
				}
			}
			if run.Status.Phase != aiRunPhaseProvisioning {
				t.Fatalf("run phase = %q, want %q", run.Status.Phase, aiRunPhaseProvisioning)
			}
			var pod corev1.Pod
			if err := c.Get(ctx, client.ObjectKey{Name: run.Status.OrchestratorPodName, Namespace: run.Namespace}, &pod); err != nil {
				t.Fatal(err)
			}
			for _, container := range pod.Spec.Containers {
				wantImage := digests[images[container.Name]]
				if tc.disabled && container.Name == tc.component {
					wantImage = images[container.Name]
				}
				if container.Image != wantImage {
					t.Fatalf("%s image = %q, want %q", container.Name, container.Image, wantImage)
				}
			}
		})
	}
}
