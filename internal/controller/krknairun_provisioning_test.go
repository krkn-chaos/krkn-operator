package controller

import (
	"context"
	"errors"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func newPendingAIRunFixture(t *testing.T) (*KrknAIRunReconciler, client.WithWatch, *krknv1alpha1.KrknAIRun) {
	t.Helper()
	r, c, run := newAIRunFixture(t)
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(run)}
	for range 2 {
		if _, err := r.Reconcile(context.Background(), request); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.Get(context.Background(), request.NamespacedName, run); err != nil {
		t.Fatal(err)
	}
	return r, c, run
}

func TestReconcileIgnoresLegacyOrchestratorImage(t *testing.T) {
	for _, tc := range []struct {
		name  string
		image string
		phase string
	}{
		{name: "installation image", image: "registry.example/trusted:release", phase: aiRunPhaseProvisioning},
		{name: "missing installation image", phase: aiRunPhaseFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, c, run := newPendingAIRunFixture(t)
			r.OrchestratorImage = tc.image
			ctx := context.Background()
			key := client.ObjectKeyFromObject(run)
			legacyRun := &unstructured.Unstructured{}
			legacyRun.SetGroupVersionKind(krknv1alpha1.GroupVersion.WithKind("KrknAIRun"))
			if err := c.Get(ctx, key, legacyRun); err != nil {
				t.Fatal(err)
			}
			if err := unstructured.SetNestedField(legacyRun.Object, "registry.example/untrusted:latest", "spec", "orchestratorImage"); err != nil {
				t.Fatal(err)
			}
			if err := c.Update(ctx, legacyRun); err != nil {
				t.Fatal(err)
			}
			if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key}); err != nil {
				t.Fatal(err)
			}
			if err := c.Get(ctx, key, run); err != nil {
				t.Fatal(err)
			}
			if run.Status.Phase != tc.phase {
				t.Fatalf("run phase = %q, want %q", run.Status.Phase, tc.phase)
			}
			if tc.image == "" {
				var pods corev1.PodList
				if err := c.List(ctx, &pods, client.InNamespace(run.Namespace)); err != nil {
					t.Fatal(err)
				}
				var credentials corev1.ConfigMapList
				if err := c.List(ctx, &credentials, client.InNamespace(run.Namespace),
					client.MatchingLabels{"krkn.dev/ai-run": aiRunLabelValue(run.Name)}); err != nil {
					t.Fatal(err)
				}
				if len(pods.Items) != 0 || len(credentials.Items) != 0 {
					t.Fatal("missing installation image provisioned a Pod or target credentials")
				}
				return
			}
			var pod corev1.Pod
			if err := c.Get(ctx, client.ObjectKey{Name: run.Status.OrchestratorPodName, Namespace: run.Namespace}, &pod); err != nil {
				t.Fatal(err)
			}
			for _, container := range pod.Spec.Containers {
				if container.Name == "orchestrator" {
					if container.Image != tc.image {
						t.Fatalf("orchestrator image = %q, want %q", container.Image, tc.image)
					}
					return
				}
			}
			t.Fatal("orchestrator container is missing")
		})
	}
}

func TestReconcileRetriesOperationalProvisioningErrors(t *testing.T) {
	conflict := apierrors.NewConflict(schema.GroupResource{
		Group: "krkn.krkn-chaos.dev", Resource: "krknairuns",
	}, "run-one", errors.New("resource version changed"))
	cases := []struct {
		name         string
		stage        string
		err          error
		acceptCreate bool
	}{
		{name: "service account unavailable", stage: "service account read", err: apierrors.NewServiceUnavailable("temporarily unavailable")},
		{name: "secret transport error", stage: "secret read", err: errors.New("connection reset by peer")},
		{name: "config read timeout", stage: "config read", err: apierrors.NewTimeoutError("read timed out", 0)},
		{name: "kubeconfig create throttled", stage: "kubeconfig create", err: apierrors.NewTooManyRequests("throttled", 0)},
		{name: "pod create unavailable", stage: "pod create", err: apierrors.NewServiceUnavailable("temporarily unavailable")},
		{name: "pod accepted before timeout", stage: "pod create", err: apierrors.NewTimeoutError("response lost", 0), acceptCreate: true},
		{name: "pod name reservation conflict", stage: "reserve pod name", err: conflict},
		{name: "provisioning status conflict", stage: "provisioning status", err: conflict},
		{name: "provisioning status unavailable", stage: "provisioning status", err: apierrors.NewServiceUnavailable("temporarily unavailable")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, c, run := newPendingAIRunFixture(t)
			ctx := context.Background()
			failed := false
			wrapped := interceptor.NewClient(c, interceptor.Funcs{
				Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					stage := ""
					switch obj.(type) {
					case *corev1.ServiceAccount:
						stage = "service account read"
					case *corev1.Secret:
						stage = "secret read"
					case *corev1.ConfigMap:
						stage = "config read"
					}
					if !failed && stage == tc.stage {
						failed = true
						return tc.err
					}
					return c.Get(ctx, key, obj, opts...)
				},
				Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					stage := "kubeconfig create"
					if _, ok := obj.(*corev1.Pod); ok {
						stage = "pod create"
					}
					if !failed && stage == tc.stage {
						if tc.acceptCreate {
							if err := c.Create(ctx, obj, opts...); err != nil {
								return err
							}
						}
						failed = true
						return tc.err
					}
					return c.Create(ctx, obj, opts...)
				},
				SubResourceUpdate: func(ctx context.Context, c client.Client, subresource string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
					stage := "reserve pod name"
					if obj.(*krknv1alpha1.KrknAIRun).Status.Phase == aiRunPhaseProvisioning {
						stage = "provisioning status"
					}
					if !failed && stage == tc.stage {
						failed = true
						return tc.err
					}
					return c.SubResource(subresource).Update(ctx, obj, opts...)
				},
			})
			r.Client = wrapped
			r.APIReader = wrapped
			request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(run)}
			result, err := r.Reconcile(ctx, request)
			if !errors.Is(err, tc.err) || result != (ctrl.Result{}) {
				t.Fatalf("operational failure result=%+v error=%v", result, err)
			}
			if err := c.Get(ctx, request.NamespacedName, run); err != nil {
				t.Fatal(err)
			}
			if run.Status.Phase != aiRunPhasePending || run.Status.CompletionTime != nil || run.Status.FailureReason != "" {
				t.Fatalf("operational failure became terminal: %+v", run.Status)
			}
			reservedName := run.Status.OrchestratorPodName
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("retry failed: %v", err)
			}
			if err := c.Get(ctx, request.NamespacedName, run); err != nil {
				t.Fatal(err)
			}
			if run.Status.Phase != aiRunPhaseProvisioning || run.Status.CompletionTime != nil {
				t.Fatalf("retry did not resume provisioning: %+v", run.Status)
			}
			if reservedName != "" && run.Status.OrchestratorPodName != reservedName {
				t.Fatalf("retry replaced reserved Pod %q with %q", reservedName, run.Status.OrchestratorPodName)
			}
			var pods corev1.PodList
			selector := client.MatchingLabels{"krkn.dev/ai-run": aiRunLabelValue(run.Name)}
			if err := c.List(ctx, &pods, client.InNamespace(run.Namespace), selector); err != nil {
				t.Fatal(err)
			}
			if len(pods.Items) != 1 || pods.Items[0].Name != run.Status.OrchestratorPodName {
				t.Fatalf("retry did not retain a single orchestrator Pod: %+v", pods.Items)
			}
			var configMaps corev1.ConfigMapList
			if err := c.List(ctx, &configMaps, client.InNamespace(run.Namespace), selector); err != nil {
				t.Fatal(err)
			}
			if len(configMaps.Items) != 1 || configMaps.Items[0].Name != aiResourceName("ai", run.Name, "kubeconfig") {
				t.Fatalf("retry did not retain a single kubeconfig ConfigMap: %+v", configMaps.Items)
			}
		})
	}
}

func TestReconcileProvisioningUsesUncachedInputs(t *testing.T) {
	r, c, run := newPendingAIRunFixture(t)
	r.Client = interceptor.NewClient(c, interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			switch obj.(type) {
			case *corev1.ConfigMap:
				return apierrors.NewNotFound(corev1.Resource("configmaps"), key.Name)
			case *corev1.Secret:
				return apierrors.NewNotFound(corev1.Resource("secrets"), key.Name)
			}
			return c.Get(ctx, key, obj, opts...)
		},
	})
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(run)}
	if _, err := r.Reconcile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(context.Background(), request.NamespacedName, run); err != nil {
		t.Fatal(err)
	}
	if run.Status.Phase != aiRunPhaseProvisioning {
		t.Fatalf("stale cache prevented provisioning: %+v", run.Status)
	}
	var pod corev1.Pod
	if err := c.Get(context.Background(), client.ObjectKey{Name: run.Status.OrchestratorPodName, Namespace: run.Namespace}, &pod); err != nil {
		t.Fatalf("orchestrator was not created: %v", err)
	}
}

func TestReconcileFailsPermanentProvisioningErrors(t *testing.T) {
	cases := []struct {
		name       string
		configure  func(*KrknAIRunReconciler, *krknv1alpha1.KrknAIRun)
		secretData map[string][]byte
		configData map[string]string
		missing    client.Object
		createErr  error
	}{
		{name: "multiple targets", configure: func(_ *KrknAIRunReconciler, run *krknv1alpha1.KrknAIRun) {
			run.Spec.TargetClusters = map[string][]string{"provider": {"one", "two"}}
		}},
		{name: "missing target request", configure: func(_ *KrknAIRunReconciler, run *krknv1alpha1.KrknAIRun) {
			run.Spec.TargetRequestID = ""
		}},
		{name: "missing config reference", configure: func(_ *KrknAIRunReconciler, run *krknv1alpha1.KrknAIRun) {
			run.Spec.ConfigMapName = ""
		}},
		{name: "missing orchestrator image", configure: func(r *KrknAIRunReconciler, _ *krknv1alpha1.KrknAIRun) {
			r.OrchestratorImage = ""
		}},
		{name: "missing artifact service settings", configure: func(r *KrknAIRunReconciler, _ *krknv1alpha1.KrknAIRun) {
			r.ServiceURL = ""
		}},
		{name: "missing service account", missing: &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "release-krkn-ai-orchestrator", Namespace: "operator"}}},
		{name: "missing credentials", missing: &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "self", Namespace: "operator"}}},
		{name: "missing config", missing: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "config-one", Namespace: "operator"}}},
		{name: "missing managed clusters", secretData: map[string][]byte{}},
		{name: "malformed managed clusters", secretData: map[string][]byte{"managed-clusters": []byte("not-json")}},
		{name: "unknown provider", secretData: map[string][]byte{"managed-clusters": []byte(`{"other":{}}`)}},
		{name: "unknown cluster", secretData: map[string][]byte{"managed-clusters": []byte(`{"krkn-operator":{}}`)}},
		{name: "invalid kubeconfig encoding", secretData: map[string][]byte{"managed-clusters": []byte(`{"krkn-operator":{"self":{"kubeconfig":"not-base64!"}}}`)}},
		{name: "missing config key", configData: map[string]string{}},
		{name: "blank config", configData: map[string]string{"krkn-ai.yaml": " \n\t"}},
		{name: "invalid Pod rejected", createErr: apierrors.NewInvalid(schema.GroupKind{Kind: "Pod"}, "run-one", field.ErrorList{
			field.Invalid(field.NewPath("spec", "containers"), nil, "invalid configuration"),
		})},
		{name: "bad Pod request", createErr: apierrors.NewBadRequest("invalid Pod request")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, c, run := newPendingAIRunFixture(t)
			ctx := context.Background()
			if tc.configure != nil {
				tc.configure(r, run)
				if err := c.Update(ctx, run); err != nil {
					t.Fatal(err)
				}
			}
			if tc.missing != nil {
				if err := c.Delete(ctx, tc.missing); err != nil {
					t.Fatal(err)
				}
			}
			if tc.secretData != nil {
				var secret corev1.Secret
				if err := c.Get(ctx, client.ObjectKey{Name: run.Spec.TargetRequestID, Namespace: run.Namespace}, &secret); err != nil {
					t.Fatal(err)
				}
				secret.Data = tc.secretData
				if err := c.Update(ctx, &secret); err != nil {
					t.Fatal(err)
				}
			}
			if tc.configData != nil {
				var config corev1.ConfigMap
				if err := c.Get(ctx, client.ObjectKey{Name: run.Spec.ConfigMapName, Namespace: run.Namespace}, &config); err != nil {
					t.Fatal(err)
				}
				config.Data = tc.configData
				if err := c.Update(ctx, &config); err != nil {
					t.Fatal(err)
				}
			}
			if tc.createErr != nil {
				r.Client = interceptor.NewClient(c, interceptor.Funcs{
					Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
						if _, ok := obj.(*corev1.Pod); ok {
							return tc.createErr
						}
						return c.Create(ctx, obj, opts...)
					},
				})
			}
			request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(run)}
			result, err := r.Reconcile(ctx, request)
			if err != nil || result != (ctrl.Result{}) {
				t.Fatalf("configuration failure result=%+v error=%v", result, err)
			}
			if err := c.Get(ctx, request.NamespacedName, run); err != nil {
				t.Fatal(err)
			}
			if run.Status.Phase != aiRunPhaseFailed || run.Status.CompletionTime == nil {
				t.Fatalf("configuration error was not terminal: %+v", run.Status)
			}
			var pods corev1.PodList
			if err := c.List(ctx, &pods, client.InNamespace(run.Namespace)); err != nil {
				t.Fatal(err)
			}
			if len(pods.Items) != 0 {
				t.Fatalf("invalid configuration launched workloads: %+v", pods.Items)
			}
		})
	}
}
