package componentversion

import (
	"context"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPublisherReconcile(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	client := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&krknv1alpha1.KrknComponent{}).Build()
	manifest := krknv1alpha1.ComponentManifest{Version: "v1.1.0-rc4", Components: []krknv1alpha1.ComponentDescription{{Component: "api", Description: "API"}}}
	publisher := &Publisher{Client: client, Namespace: "krkn", Name: "krkn-operator", Manifest: manifest}
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "krkn", Name: "krkn-operator"}}

	if _, err := publisher.Reconcile(ctx, request); err != nil {
		t.Fatal(err)
	}
	var resource krknv1alpha1.KrknComponent
	if err := client.Get(ctx, request.NamespacedName, &resource); err != nil {
		t.Fatal(err)
	}
	if got := resource.Status.Components[publisher.Name].Version; got != manifest.Version {
		t.Fatalf("published version = %q, want %q", got, manifest.Version)
	}
	version := resource.ResourceVersion
	if _, err := publisher.Reconcile(ctx, request); err != nil {
		t.Fatal(err)
	}
	if err := client.Get(ctx, request.NamespacedName, &resource); err != nil {
		t.Fatal(err)
	}
	if resource.ResourceVersion != version {
		t.Fatal("idempotent reconcile updated the resource")
	}
	if err := client.Delete(ctx, &resource); err != nil {
		t.Fatal(err)
	}
	if _, err := publisher.Reconcile(ctx, request); err != nil {
		t.Fatal(err)
	}
	if err := client.Get(ctx, request.NamespacedName, &resource); err != nil {
		t.Fatal(err)
	}
	if got := resource.Status.Components[publisher.Name].Version; got != manifest.Version {
		t.Fatalf("version after deletion = %q, want %q", got, manifest.Version)
	}
}
