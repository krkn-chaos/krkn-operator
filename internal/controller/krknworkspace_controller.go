package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/workspace"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// KrknWorkspaceReconciler observes an externally managed PVC and reports its
// metadata availability without changing the PVC or its filesystem.
type KrknWorkspaceReconciler struct {
	client.Client
	Namespace string
}

// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknworkspaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknworkspaces/status,verbs=get;update;patch

func (r *KrknWorkspaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if req.Namespace != r.Namespace {
		return ctrl.Result{}, nil
	}
	entry := &krknv1alpha1.KrknWorkspace{}
	if err := r.Get(ctx, req.NamespacedName, entry); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	before := entry.Status.DeepCopy()
	condition := metav1.Condition{Type: "Available", Status: metav1.ConditionFalse,
		ObservedGeneration: entry.Generation}
	if entry.Spec.ClaimName == "" || entry.Spec.ClaimUID == "" || (entry.Spec.AvailableToAll && entry.Spec.Group != "") {
		condition.Reason, condition.Message = "InvalidSpec", "Workspace claim identity or visibility is invalid"
	} else {
		pvc := &corev1.PersistentVolumeClaim{}
		err := r.Get(ctx, types.NamespacedName{Namespace: r.Namespace, Name: entry.Spec.ClaimName}, pvc)
		switch {
		case apierrors.IsNotFound(err):
			condition.Reason, condition.Message = "PVCMissing", "Referenced PVC is missing"
		case err != nil:
			return ctrl.Result{}, fmt.Errorf("load workspace PVC %q: %w", entry.Spec.ClaimName, err)
		default:
			if err := workspace.ValidatePVC(pvc, types.UID(entry.Spec.ClaimUID)); err != nil {
				condition.Reason = "PVCUnavailable"
				if errors.Is(err, workspace.ErrReplaced) {
					condition.Reason = "PVCReplaced"
				}
				condition.Message = err.Error()
			} else {
				condition.Status, condition.Reason, condition.Message = metav1.ConditionTrue, "PVCReady", "Referenced PVC is Bound, Filesystem, and ReadWriteMany"
			}
		}
	}
	entry.Status.ObservedGeneration = entry.Generation
	meta.SetStatusCondition(&entry.Status.Conditions, condition)
	if reflect.DeepEqual(*before, entry.Status) {
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}
	if err := r.Status().Update(ctx, entry); err != nil {
		return ctrl.Result{}, fmt.Errorf("update workspace status: %w", err)
	}
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *KrknWorkspaceReconciler) mapPVC(ctx context.Context, object client.Object) []reconcile.Request {
	if object.GetNamespace() != r.Namespace {
		return nil
	}
	var entries krknv1alpha1.KrknWorkspaceList
	if err := r.List(ctx, &entries, client.InNamespace(r.Namespace)); err != nil {
		log.FromContext(ctx).Error(err, "list workspaces for PVC event", "pvc", object.GetName())
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for _, entry := range entries.Items {
		if entry.Spec.ClaimName == object.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: r.Namespace, Name: entry.Name}})
		}
	}
	return requests
}

func (r *KrknWorkspaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&krknv1alpha1.KrknWorkspace{}).
		Watches(&corev1.PersistentVolumeClaim{}, handler.EnqueueRequestsFromMapFunc(r.mapPVC)).
		WithEventFilter(NewNamespaceFilter(r.Namespace)).
		Complete(r)
}
