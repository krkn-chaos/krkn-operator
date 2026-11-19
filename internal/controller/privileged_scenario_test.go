package controller

import (
	"context"
	"errors"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	krknctlconfig "github.com/krkn-chaos/krknctl/pkg/config"
	krknctlmodels "github.com/krkn-chaos/krknctl/pkg/provider/models"
	"github.com/krkn-chaos/krknctl/pkg/verify"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestPrivilegedLabel(t *testing.T) {
	config, err := krknctlconfig.LoadConfig()
	require.NoError(t, err)
	for _, tc := range []struct {
		name    string
		labels  map[string]string
		want    bool
		wantErr bool
	}{
		{name: "label absent", labels: map[string]string{"krknctl.is_a_scenario": "true"}, want: false},
		{name: "true", labels: map[string]string{"krknctl.privileged": "true"}, want: true},
		{name: "false", labels: map[string]string{"krknctl.privileged": "false"}, want: false},
		{name: "not a boolean", labels: map[string]string{"krknctl.privileged": "yes"}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := privilegedLabel(&config, tc.labels)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

// newPrivilegedTestReconciler returns a reconciler for one scenario run whose
// image resolves with the given privileged flag.
func newPrivilegedTestReconciler(t *testing.T, privileged bool, funcs interceptor.Funcs) (*KrknScenarioRunReconciler, *krknv1alpha1.KrknScenarioRun) {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: "default"},
		Data:       map[string][]byte{"managed-clusters": []byte(`{"provider":{"cluster":{"kubeconfig":"Y29uZmln"}}}`)},
	}
	public := false
	run := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "run", Namespace: "default", UID: "9e70715d-9154-4b9e-aa76-1ae6c6b2594e"},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "target",
			Scenario:        krknv1alpha1.ScenarioReference{Name: "node-io-hog", Private: &public},
		},
	}
	r := &KrknScenarioRunReconciler{
		Client: fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(secret, run).WithInterceptorFuncs(funcs).Build(),
		Scheme: scheme, Namespace: "default",
		imageResolver: func(context.Context, *krknctlconfig.Config, krknv1alpha1.ScenarioReference, *krknctlmodels.RegistryV2) (resolvedImage, error) {
			return resolvedImage{image: "quay.io/example/node-io-hog@sha256:abc", privileged: privileged}, nil
		},
		signatureVerifier: func(context.Context, *krknctlconfig.Config, krknv1alpha1.ScenarioReference, *krknctlmodels.RegistryV2, string) (verify.SignatureStatus, error) {
			return verify.SignatureSigned, nil
		},
	}
	return r, run
}

func TestScenarioPodIsPrivilegedOnlyWhenTheScenarioRequestsIt(t *testing.T) {
	for _, privileged := range []bool{true, false} {
		t.Run(map[bool]string{true: "privileged", false: "unprivileged"}[privileged], func(t *testing.T) {
			r, run := newPrivilegedTestReconciler(t, privileged, interceptor.Funcs{})
			resources, err := r.prepareJobResources(t.Context(), run, "provider", "cluster", "https://cluster.example.com", "11111111-1111-4111-8111-111111111111")
			require.NoError(t, err)
			_, err = r.submitScenarioPod(t.Context(), run, resources)
			require.NoError(t, err)

			var pods corev1.PodList
			require.NoError(t, r.List(t.Context(), &pods))
			require.Len(t, pods.Items, 1)
			pod := pods.Items[0]
			if privileged {
				require.NotNil(t, pod.Spec.Containers[0].SecurityContext)
				require.Equal(t, true, *pod.Spec.Containers[0].SecurityContext.Privileged)
			} else {
				require.Nil(t, pod.Spec.Containers[0].SecurityContext)
			}
			// The Pod-level identity stays the same either way.
			require.Equal(t, int64(1001), *pod.Spec.SecurityContext.RunAsUser)
			require.Equal(t, int64(1001), *pod.Spec.SecurityContext.RunAsGroup)
			require.Equal(t, int64(1001), *pod.Spec.SecurityContext.FSGroup)
		})
	}
}

func TestPrivilegedScenarioPodRejectedByClusterPolicy(t *testing.T) {
	rejectPods := interceptor.Funcs{Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
		if _, ok := obj.(*corev1.Pod); ok {
			return apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, obj.GetName(),
				errors.New(`violates PodSecurity "restricted:latest": privileged`))
		}
		return c.Create(ctx, obj, opts...)
	}}
	for _, privileged := range []bool{true, false} {
		r, run := newPrivilegedTestReconciler(t, privileged, rejectPods)
		resources, err := r.prepareJobResources(t.Context(), run, "provider", "cluster", "https://cluster.example.com", "11111111-1111-4111-8111-111111111111")
		require.NoError(t, err)

		_, err = r.submitScenarioPod(t.Context(), run, resources)

		require.True(t, apierrors.IsForbidden(err), "error: %v", err)
		if privileged {
			require.ErrorContains(t, err, `scenario "node-io-hog" needs a privileged container`)
			require.ErrorContains(t, err, `"krkn-operator-krkn-scenario-runner"`)
		} else {
			require.ErrorContains(t, err, "failed to create pod")
			require.NotContains(t, err.Error(), "privileged container")
		}
	}
}
