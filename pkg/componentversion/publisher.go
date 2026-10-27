package componentversion

import (
	"context"
	"fmt"
	"reflect"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krkncomponents,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krkncomponents/status,verbs=get;update;patch

// Publisher reconciles one KrknComponent resource owned by one operator binary.
// Its manifest is injected when the binary starts.
type Publisher struct {
	Client    client.Client
	Namespace string
	Name      string
	Manifest  krknv1alpha1.ComponentManifest
}

// SetupWithManager watches the publisher's KrknComponent resource.
func (p *Publisher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&krknv1alpha1.KrknComponent{}).
		Complete(p)
}

// NeedLeaderElection ensures only the elected manager bootstraps the resource.
func (p *Publisher) NeedLeaderElection() bool { return true }

// Start bootstraps the resource so reconciliation can resume on later changes.
func (p *Publisher) Start(ctx context.Context) error {
	if err := p.ensure(ctx); err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}

// Reconcile restores this publisher's manifest after updates or deletion.
func (p *Publisher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if req.Namespace != p.Namespace || req.Name != p.Name {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, p.ensure(ctx)
}

func (p *Publisher) ensure(ctx context.Context) error {
	key := types.NamespacedName{Namespace: p.Namespace, Name: p.Name}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var resource krknv1alpha1.KrknComponent
		if err := p.Client.Get(ctx, key, &resource); err != nil {
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("get KrknComponent %s: %w", key, err)
			}
			resource = krknv1alpha1.KrknComponent{ObjectMeta: metav1.ObjectMeta{
				Name: p.Name, Namespace: p.Namespace,
			}}
			if err := p.Client.Create(ctx, &resource); err != nil {
				return fmt.Errorf("create KrknComponent %s: %w", key, err)
			}
		}
		want := map[string]krknv1alpha1.ComponentManifest{p.Name: p.Manifest}
		if reflect.DeepEqual(resource.Status.Components, want) {
			return nil
		}
		resource.Status.Components = want
		if err := p.Client.Status().Update(ctx, &resource); err != nil {
			return fmt.Errorf("update KrknComponent %s status: %w", key, err)
		}
		return nil
	})
}
