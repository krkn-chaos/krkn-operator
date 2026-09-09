package signatureverification

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetEnabledDefaultsToTrue(t *testing.T) {
	client := fakeClient(t)
	enabled, err := GetEnabled(context.Background(), client, "default")
	if err != nil {
		t.Fatalf("GetEnabled returned error: %v", err)
	}
	if !enabled {
		t.Fatal("expected verification to be enabled by default")
	}
}

func TestSetEnabledPersistsAndReadsValue(t *testing.T) {
	client := fakeClient(t)
	if err := SetEnabled(context.Background(), client, "default", false); err != nil {
		t.Fatalf("SetEnabled returned error: %v", err)
	}
	enabled, err := GetEnabled(context.Background(), client, "default")
	if err != nil {
		t.Fatalf("GetEnabled returned error: %v", err)
	}
	if enabled {
		t.Fatal("expected persisted setting to be disabled")
	}

	if err := SetEnabled(context.Background(), client, "default", true); err != nil {
		t.Fatalf("SetEnabled update returned error: %v", err)
	}
	enabled, err = GetEnabled(context.Background(), client, "default")
	if err != nil || !enabled {
		t.Fatalf("expected persisted setting to be enabled, enabled=%v err=%v", enabled, err)
	}
}

func TestGetEnabledRejectsInvalidValue(t *testing.T) {
	client := fakeClient(t, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: ConfigMapName, Namespace: "default"},
		Data:       map[string]string{EnabledKey: "not-a-bool"},
	})
	if _, err := GetEnabled(context.Background(), client, "default"); err == nil {
		t.Fatal("expected invalid setting to return an error")
	}
}

func fakeClient(t *testing.T, objects ...runtime.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}
	return fakeclient.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()
}
