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
	"os/exec"
	"strings"

	"github.com/krkn-chaos/krkn-operator/pkg/visualize"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

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
		grafanaURL = r.detectGrafanaURL(ctx, targetNamespace)
		logger.Info("Detected Grafana URL", "url", grafanaURL)

	} else if job.Status.Failed > 0 {
		// Job failed
		logger.Error(nil, "Deployment job failed", "instance", instanceName)
		newStatus = visualize.StatusFailed
		errorMessage = "Deployment job failed"

		// Try to get error details from Job conditions
		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobFailed && cond.Message != "" {
				errorMessage = cond.Message
				break
			}
		}
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

// detectGrafanaURL attempts to detect the Grafana URL from the deployed resources
func (r *VisualizeJobController) detectGrafanaURL(ctx context.Context, namespace string) string {
	logger := log.FromContext(ctx)

	// Try OpenShift Route first
	cmd := exec.CommandContext(ctx, "oc", "get", "route", "grafana", "-n", namespace,
		"-o", "jsonpath={.spec.host}")
	if output, err := cmd.Output(); err == nil {
		host := strings.TrimSpace(string(output))
		if host != "" {
			return fmt.Sprintf("https://%s", host)
		}
	}

	// Try Kubernetes Ingress
	cmd = exec.CommandContext(ctx, "kubectl", "get", "ingress", "grafana", "-n", namespace,
		"-o", "jsonpath={.spec.rules[0].host}")
	if output, err := cmd.Output(); err == nil {
		host := strings.TrimSpace(string(output))
		if host != "" {
			return fmt.Sprintf("http://%s", host)
		}
	}

	// Try LoadBalancer Service
	cmd = exec.CommandContext(ctx, "kubectl", "get", "svc", "grafana", "-n", namespace,
		"-o", "jsonpath={.status.loadBalancer.ingress[0].hostname}")
	if output, err := cmd.Output(); err == nil {
		host := strings.TrimSpace(string(output))
		if host != "" {
			return fmt.Sprintf("http://%s:3000", host)
		}
	}

	// Fallback to service name
	logger.Info("Could not detect external Grafana URL, using service name", "namespace", namespace)
	return fmt.Sprintf("http://grafana.%s.svc.cluster.local:3000", namespace)
}

// SetupWithManager sets up the controller with the Manager
func (r *VisualizeJobController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.Job{}).
		Complete(r)
}
