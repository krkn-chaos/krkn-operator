package kubeconfig

import (
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLoadOriginalAttempt(t *testing.T) {
	const jobID = "attempt-id"
	source := &krknv1alpha1.KrknScenarioRun{ObjectMeta: metav1.ObjectMeta{Name: "source-run", Namespace: "operator", UID: types.UID("source-uid")}}
	base := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: AttemptConfigMapName(jobID), Namespace: "operator",
			Labels:          map[string]string{"krkn-job-id": jobID, "krkn-cluster-name": "cluster-a"},
			OwnerReferences: []metav1.OwnerReference{{Kind: "KrknScenarioRun", Name: source.Name, UID: source.UID}},
		},
		Data: map[string]string{"config": "apiVersion: v1"},
	}
	for _, tc := range []struct {
		name   string
		change func(*corev1.ConfigMap)
		valid  bool
	}{
		{name: "valid", valid: true},
		{name: "wrong owner UID", change: func(cm *corev1.ConfigMap) { cm.OwnerReferences[0].UID = "other" }},
		{name: "wrong owner name", change: func(cm *corev1.ConfigMap) { cm.OwnerReferences[0].Name = "other" }},
		{name: "wrong owner kind", change: func(cm *corev1.ConfigMap) { cm.OwnerReferences[0].Kind = "Other" }},
		{name: "wrong attempt label", change: func(cm *corev1.ConfigMap) { cm.Labels["krkn-job-id"] = "other" }},
		{name: "wrong cluster label", change: func(cm *corev1.ConfigMap) { cm.Labels["krkn-cluster-name"] = "other" }},
		{name: "empty config", change: func(cm *corev1.ConfigMap) { cm.Data["config"] = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cm := base.DeepCopy()
			if tc.change != nil {
				tc.change(cm)
			}
			scheme := runtime.NewScheme()
			require.NoError(t, corev1.AddToScheme(scheme))
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cm).Build()
			config, err := LoadOriginalAttempt(t.Context(), k8sClient, "operator", source, jobID, "cluster-a")
			if tc.valid {
				require.NoError(t, err)
				require.Equal(t, "apiVersion: v1", config)
			} else {
				require.ErrorContains(t, err, "identity is invalid")
				require.Empty(t, config)
			}
		})
	}
}
