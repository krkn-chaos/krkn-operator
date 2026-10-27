package componentversion

import (
	"context"
	"errors"
	"testing"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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

func TestPublisherRecoversFromConcurrentCreateAndDelete(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	var createRace, deleteRace bool
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&krknv1alpha1.KrknComponent{}).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if !createRace {
					createRace = true
					if err := c.Create(ctx, obj.DeepCopyObject().(client.Object), opts...); err != nil {
						return err
					}
					return apierrors.NewAlreadyExists(schema.GroupResource{Resource: "krkncomponents"}, obj.GetName())
				}
				return c.Create(ctx, obj, opts...)
			},
			SubResourceUpdate: func(ctx context.Context, c client.Client, name string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				if !deleteRace {
					deleteRace = true
					if err := c.Delete(ctx, obj); err != nil {
						return err
					}
					return apierrors.NewNotFound(schema.GroupResource{Resource: "krkncomponents"}, obj.GetName())
				}
				return c.Status().Update(ctx, obj, opts...)
			},
		}).Build()
	publisher := &Publisher{Client: k8sClient, Namespace: "krkn", Name: "operator", Manifest: krknv1alpha1.ComponentManifest{Version: "v1.1.0-rc4", Components: []krknv1alpha1.ComponentDescription{{Component: "api", Description: "API"}}}}
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: publisher.Namespace, Name: publisher.Name}}
	if _, err := publisher.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile after create/delete races: %v", err)
	}
	if !createRace || !deleteRace {
		t.Fatalf("create race = %t, delete race = %t", createRace, deleteRace)
	}
	var resource krknv1alpha1.KrknComponent
	if err := k8sClient.Get(context.Background(), request.NamespacedName, &resource); err != nil {
		t.Fatal(err)
	}
	if got := resource.Status.Components[publisher.Name].Version; got != publisher.Manifest.Version {
		t.Fatalf("version after races = %q", got)
	}
}

func TestPublisherReconcileReturnsTransientErrors(t *testing.T) {
	for _, failedOperation := range []string{"create", "status"} {
		t.Run(failedOperation, func(t *testing.T) {
			scheme := runtime.NewScheme()
			if err := krknv1alpha1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			transient := errors.New("temporary API failure")
			failOnce := true
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).
				WithStatusSubresource(&krknv1alpha1.KrknComponent{}).
				WithInterceptorFuncs(interceptor.Funcs{
					Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
						if failedOperation == "create" && failOnce {
							failOnce = false
							return transient
						}
						return c.Create(ctx, obj, opts...)
					},
					SubResourceUpdate: func(ctx context.Context, c client.Client, name string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
						if failedOperation == "status" && failOnce {
							failOnce = false
							return transient
						}
						return c.Status().Update(ctx, obj, opts...)
					},
				}).Build()
			publisher := &Publisher{Client: k8sClient, Namespace: "krkn", Name: "operator", Manifest: krknv1alpha1.ComponentManifest{Version: "v1.1.0", Components: []krknv1alpha1.ComponentDescription{{Component: "api", Description: "API"}}}}
			request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: publisher.Namespace, Name: publisher.Name}}
			if _, err := publisher.Reconcile(context.Background(), request); !errors.Is(err, transient) {
				t.Fatalf("first reconcile error = %v, want transient error", err)
			}
			if _, err := publisher.Reconcile(context.Background(), request); err != nil {
				t.Fatalf("retry reconcile: %v", err)
			}
		})
	}
}

func TestPublisherStartRetriesAndReturnsCancellation(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	failed := make(chan struct{})
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&krknv1alpha1.KrknComponent{}).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				select {
				case <-failed:
					return c.Get(ctx, key, obj, opts...)
				default:
					close(failed)
					return errors.New("temporary API failure")
				}
			},
		}).Build()
	publisher := &Publisher{Client: k8sClient, Namespace: "krkn", Name: "operator", Manifest: krknv1alpha1.ComponentManifest{Version: "v1.1.0", Components: []krknv1alpha1.ComponentDescription{{Component: "api", Description: "API"}}}}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- publisher.Start(ctx) }()
	select {
	case <-failed:
	case <-time.After(3 * time.Second):
		cancel()
		t.Fatal("publisher did not attempt initial publication")
	}
	request := types.NamespacedName{Namespace: publisher.Namespace, Name: publisher.Name}
	deadline := time.Now().Add(3 * time.Second)
	for {
		var resource krknv1alpha1.KrknComponent
		if err := k8sClient.Get(context.Background(), request, &resource); err == nil && resource.Status.Components[publisher.Name].Version == publisher.Manifest.Version {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("publisher did not recover from transient API failure")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("Start() error = %v, want context cancellation", err)
	}
}
