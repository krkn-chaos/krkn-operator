/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	krknchaosv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// KrknTargetReconciler reconciles a KrknTarget object
type KrknTargetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=krkn-chaos.krkn-chaos.dev,resources=krkntargets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=krkn-chaos.krkn-chaos.dev,resources=krkntargets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=krkn-chaos.krkn-chaos.dev,resources=krkntargets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KrknTarget object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *KrknTargetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	krknTarget := &krknchaosv1alpha1.KrknTarget{}
	if err := r.Client.Get(ctx, req.NamespacedName, krknTarget); err != nil {
		if apierrors.IsNotFound(err) {
			logf.Log.Info("KrknTarget not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		logf.Log.Error(err, "Failed to get KrknTarget.")
	}
	// idempotent secret content synchronization
	err := r.synchronizeSecret(ctx, krknTarget)
	if err != nil {
		return ctrl.Result{}, err
	}

	conditions := &krknTarget.Status.Conditions
	if len(*conditions) == 0 {
		meta.SetStatusCondition(conditions, metav1.Condition{
			Type:    "App",
			Status:  metav1.ConditionUnknown,
			Reason:  "Initializing",
			Message: "Starting reconciliation",
		})
		logf.Log.Info("Condition", "Length", len(krknTarget.Status.Conditions))
		if err := r.Status().Update(ctx, krknTarget); err != nil {
			logf.Log.Error(err, "Failed to update KrknTarget status")
			return ctrl.Result{}, err
		}
		// Start the next cycle
		return ctrl.Result{}, nil
	}
	currentCondition := (*conditions)[0].Reason
	switch currentCondition {
	case "Initializing":
	case "Unavailable":
	case "Available":
	default:
		meta.SetStatusCondition(conditions, metav1.Condition{
			Type:    "App",
			Status:  metav1.ConditionUnknown,
			Reason:  "Initializing",
			Message: "Starting reconciliation",
		})
		if err := r.Status().Update(ctx, krknTarget); err != nil {
			logf.Log.Error(err, "Failed to update KrknTarget status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KrknTargetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&krknchaosv1alpha1.KrknTarget{}).Named(
		"krkntarget").Complete(r)
}

func (r *KrknTargetReconciler) synchronizeSecret(ctx context.Context,
	instance *krknchaosv1alpha1.KrknTarget) error {
	secretName := "krkn-target-" + instance.Spec.Name
	secretNamespace := utils.GetOperatorNamespace()
	action := instance.Spec.Action
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
		},

		Data: map[string][]byte{
			"token":       []byte(instance.Spec.Token),
			"apiEndpoint": []byte(instance.Spec.APIEndpoint),
			"name":        []byte(instance.Spec.Name),
		},
	}

	currentSecret := &corev1.Secret{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: secretName, Namespace: secretNamespace}, currentSecret)
	secretExists := err == nil

	switch action {
	case krknchaosv1alpha1.ActionCreate:
		if !secretExists {
			logf.Log.Info("Creating KrknTarget secret", "Namespace", secret.Namespace, "Name", secret.Name)
			return r.Client.Create(ctx, secret)
		}
		return nil

	case krknchaosv1alpha1.ActionUpdate:
		if !secretExists {
			logf.Log.Info("Creating KrknTarget secret", "Namespace", secret.Namespace, "Name", secret.Name)
			return r.Client.Create(ctx, secret)
		}
		if !reflect.DeepEqual(currentSecret.Data, secret.Data) {
			currentSecret.Data = secret.Data
			logf.Log.Info("Updating KrknTarget secret", "Namespace", secret.Namespace, "Name",
				secret.Name)
			return r.Client.Update(ctx, currentSecret)
		}
		return nil

	case krknchaosv1alpha1.ActionDelete:
		if secretExists {
			logf.Log.Info("Deleting KrknTarget secret", "Namespace", secret.Namespace, "Name",
				secret.Name)
			return r.Client.Delete(ctx, currentSecret)
		}
		return nil

	default:
		return fmt.Errorf("invalid KrknTarget Action: %s", action)
	}
}
