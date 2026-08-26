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
	"errors"
	"fmt"

	"github.com/krkn-chaos/krkn-operator/pkg/visualize"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// +kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;create;delete
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;create;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=list;delete;deletecollection
// +kubebuilder:rbac:groups="",resources=services,verbs=create;delete
// +kubebuilder:rbac:groups="",resources=configmaps;secrets,verbs=create;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=create;delete

// VisualizeJobController watches krkn-visualize deployment Jobs
// and updates the corresponding ConfigMap instance status
type VisualizeJobController struct {
	client.Client
	Namespace string // Operator namespace where ConfigMaps are stored
}

// Reconcile handles Job status changes
func (r *VisualizeJobController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("visualize-job-controller").WithValues("job", req.NamespacedName)

	// Fetch the Job
	var job batchv1.Job
	if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
		if apierrors.IsNotFound(err) {
			// Job deleted, nothing to do
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Job")
		return ctrl.Result{}, err
	}

	// Only watch our deployment jobs (check labels)
	if job.Labels[visualize.LabelComponent] != visualize.ComponentValue {
		return ctrl.Result{}, nil
	}

	// Get instance name from Job labels
	instanceName := job.Labels["app.kubernetes.io/instance"]
	if instanceName == "" {
		logger.Info("Job missing instance label, skipping")
		return ctrl.Result{}, nil
	}

	// Fetch the corresponding ConfigMap
	var cm corev1.ConfigMap
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: r.Namespace,
		Name:      instanceName,
	}, &cm); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("ConfigMap not found, instance may have been deleted", "instance", instanceName)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get ConfigMap", "instance", instanceName)
		return ctrl.Result{}, err
	}

	// Check current status - only update if still Installing
	currentStatus := cm.Annotations[visualize.AnnotationStatus]
	if currentStatus != visualize.StatusInstalling {
		// Already updated or not our job, skip
		return ctrl.Result{}, nil
	}

	// Check Job completion status
	var newStatus string
	var grafanaURL string
	var errorMessage string

	if job.Status.Succeeded > 0 {
		// Job succeeded!
		logger.Info("Deployment job succeeded", "instance", instanceName)
		newStatus = visualize.StatusReady

		// Detect Grafana URL
		targetNamespace := cm.Annotations[visualize.AnnotationNamespace]
		detectedURL, err := r.detectGrafanaURL(ctx, targetNamespace, instanceName)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("waiting for Grafana route for instance %q: %w", instanceName, err)
		}
		grafanaURL = detectedURL
		logger.Info("Detected Grafana URL", "url", grafanaURL)

	} else if job.Status.Failed > 0 {
		// Job failed
		newStatus = visualize.StatusFailed
		errorMessage = "Deployment job failed"

		// Try to get error details from Job conditions
		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobFailed && cond.Message != "" {
				errorMessage = cond.Message
				break
			}
		}
		logger.Error(errors.New(errorMessage), "Deployment job failed", "instance", instanceName)
	} else {
		// Job still running, check back later
		return ctrl.Result{}, nil
	}

	// Update ConfigMap annotations
	cm.Annotations = visualize.UpdateAnnotations(
		cm.Annotations,
		newStatus,
		grafanaURL,
		errorMessage,
	)

	if err := r.Update(ctx, &cm); err != nil {
		logger.Error(err, "Failed to update ConfigMap status")
		return ctrl.Result{}, err
	}

	logger.Info("Updated instance status",
		"instance", instanceName,
		"status", newStatus,
		"grafanaURL", grafanaURL,
	)

	return ctrl.Result{}, nil
}

// detectGrafanaURL waits for the per-instance OpenShift Route to receive a host.
func (r *VisualizeJobController) detectGrafanaURL(ctx context.Context, namespace, instanceName string) (string, error) {
	routeName := visualize.ResourceName(instanceName, "grafana")
	return visualize.GetRouteURL(ctx, r.Client, namespace, routeName)
}

// SetupWithManager sets up the controller with the Manager
func (r *VisualizeJobController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.Job{}).
		Complete(r)
}
