package backup

import (
	"context"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestBackupRestoresOIDCProviderAndBlockSecrets(t *testing.T) {
	const namespace = "krkn-system"
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	provider := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "krkn-identity-provider-corp", Namespace: namespace,
		Labels: map[string]string{"krkn.krkn-chaos.dev/identity-provider": "true", "app.kubernetes.io/name": "krkn-operator"}},
		Data: map[string][]byte{"clientSecret": []byte("test-only-secret")}}
	block := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "krkn-oidc-block-abc", Namespace: namespace,
		Labels: map[string]string{"krkn.krkn-chaos.dev/oidc-block": "true", "app.kubernetes.io/name": "krkn-operator"}},
		Data: map[string][]byte{"subject": []byte("blocked-subject")}}
	source := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(provider, block).Build()
	archive, err := CreateBackup(context.Background(), source, BackupConfig{
		Namespace: namespace, OutputDir: t.TempDir(), BackupName: "oidc-settings",
	})
	if err != nil {
		t.Fatal(err)
	}
	destination := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	if err := RestoreBackup(context.Background(), destination, RestoreConfig{Namespace: namespace, BackupPath: archive}); err != nil {
		t.Fatal(err)
	}
	for name, check := range map[string]struct{ key, want string }{
		provider.Name: {"clientSecret", "test-only-secret"},
		block.Name:    {"subject", "blocked-subject"},
	} {
		var restored corev1.Secret
		if err := destination.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, &restored); err != nil {
			t.Fatal(err)
		}
		if string(restored.Data[check.key]) != check.want {
			t.Fatalf("restored Secret %s lost %s", name, check.key)
		}
	}
}

func TestRestoreRejectsUnrelatedSecretWithOIDCLabel(t *testing.T) {
	secret := &unstructured.Unstructured{}
	secret.SetName("unrelated-secret")
	secret.SetLabels(map[string]string{"krkn.krkn-chaos.dev/oidc-block": "true", "app.kubernetes.io/name": "krkn-operator"})
	if isOperatorManagedSecret(secret) {
		t.Fatal("unrelated Secret accepted for OIDC restore")
	}
}
