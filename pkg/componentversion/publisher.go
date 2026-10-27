package componentversion

import (
	"context"
	"fmt"
	"reflect"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krkncomponents,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krkncomponents/status,verbs=get;update;patch

// Publisher reconciles one KrknComponent resource owned by one operator binary.
// Its manifest is injected when the binary starts.
type Publisher struct {
	// Client reads and writes the namespaced KrknComponent resource.
	Client client.Client
	// Namespace is the namespace in which this operator publishes its resource.
	Namespace string
	// Name is both the resource name and its key in the status map.
	Name string
	// Manifest is the validated version manifest embedded in this operator build.
	Manifest krknv1alpha1.ComponentManifest
}

// SetupWithManager watches the publisher's KrknComponent resource. It returns
// an error if the controller cannot be registered, including when the CRD is
// not yet served by the API server.
func (p *Publisher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&krknv1alpha1.KrknComponent{}).
		Complete(p)
}

// NeedLeaderElection ensures only the elected manager bootstraps the resource.
func (p *Publisher) NeedLeaderElection() bool { return true }

// Start bootstraps the resource so reconciliation can resume on later changes.
// Kubernetes API failures are logged and retried with bounded exponential
// backoff instead of stopping the manager. It returns ctx.Err() on shutdown.
func (p *Publisher) Start(ctx context.Context) error {
	const maxDelay = 30 * time.Second
	for delay := time.Second; ; {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := p.ensure(ctx); err == nil {
			break
		} else {
			log.FromContext(ctx).Error(err, "unable to publish component version; retrying", "namespace", p.Namespace, "name", p.Name)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		if delay < maxDelay {
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}
	<-ctx.Done()
	return ctx.Err()
}

// Reconcile restores this publisher's manifest after updates or deletion. It
// returns Kubernetes read, create, or status-update errors so controller-runtime
// can retry them; concurrent writes are retried with bounded backoff here.
func (p *Publisher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if req.Namespace != p.Namespace || req.Name != p.Name {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, p.ensure(ctx)
}

func (p *Publisher) ensure(ctx context.Context) error {
	key := types.NamespacedName{Namespace: p.Namespace, Name: p.Name}
	operation := "get"
	err := retry.OnError(retry.DefaultRetry, func(err error) bool {
		return apierrors.IsConflict(err) || apierrors.IsAlreadyExists(err) || apierrors.IsNotFound(err)
	}, func() error {
		var resource krknv1alpha1.KrknComponent
		operation = "get"
		if err := p.Client.Get(ctx, key, &resource); err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			resource = krknv1alpha1.KrknComponent{ObjectMeta: metav1.ObjectMeta{
				Name: p.Name, Namespace: p.Namespace,
			}}
			operation = "create"
			if err := p.Client.Create(ctx, &resource); err != nil {
				return err
			}
		}
		want := map[string]krknv1alpha1.ComponentManifest{p.Name: p.Manifest}
		if reflect.DeepEqual(resource.Status.Components, want) {
			return nil
		}
		resource.Status.Components = want
		operation = "update status"
		if err := p.Client.Status().Update(ctx, &resource); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("%s KrknComponent %s: %w", operation, key, err)
	}
	return nil
}
