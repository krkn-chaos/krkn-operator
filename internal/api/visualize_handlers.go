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

// Package api provides HTTP handlers for krkn-visualize management.
// This file implements handlers for creating, listing, and deleting krkn-visualize
// instances, which are stored as Kubernetes ConfigMaps with deployment metadata.
package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/visualize"
	routev1 "github.com/openshift/api/route/v1"
	routeclient "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CreateVisualizeInstance handles POST /api/v1/visualize
// Creates a new krkn-visualize installation request (admin only)
func (h *Handler) CreateVisualizeInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("create-visualize-instance")

	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	var req visualize.CreateVisualizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := visualize.ValidateCreateRequest(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	// Check if instance already exists
	exists, err := h.visualizeInstanceExists(ctx, req.Name)
	if err != nil {
		logger.Error(err, "Failed to check for existing visualize instance", "name", req.Name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to check for existing instance",
		})
		return
	}
	if exists {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: fmt.Sprintf("krkn-visualize instance '%s' already exists", req.Name),
		})
		return
	}

	claims := auth.GetClaimsFromContext(ctx)
	createdBy := ""
	if claims != nil {
		createdBy = claims.UserID
	}

	namespace := req.Namespace
	if namespace == "" {
		namespace = visualize.DefaultNamespace
	}

	// For simplicity, we'll create one instance per target cluster
	// If multiple targets provided, create the first one (or iterate to create multiple)
	targetCluster := req.TargetClusters[0]
	var existingInstances corev1.ConfigMapList
	if err := h.client.List(ctx, &existingInstances,
		client.InNamespace(h.namespace),
		client.MatchingLabels{visualize.LabelComponent: visualize.ComponentValue},
	); err != nil {
		logger.Error(err, "Failed to check for existing visualize namespace")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to check for existing krkn-visualize namespace",
		})
		return
	}
	for i := range existingInstances.Items {
		existing := &existingInstances.Items[i]
		if existing.Annotations[visualize.AnnotationTargetCluster] == targetCluster &&
			existing.Annotations[visualize.AnnotationNamespace] == namespace {
			writeJSONError(w, http.StatusConflict, ErrorResponse{
				Error:   "conflict",
				Message: fmt.Sprintf("target cluster %q already has a krkn-visualize installation in namespace %q", targetCluster, namespace),
			})
			return
		}
	}

	labels := visualize.BuildLabels()
	annotations := visualize.BuildAnnotations(
		targetCluster,
		namespace,
		req.ElasticsearchConfigName,
		visualize.StatusPending,
		"", // grafanaURL - will be set after deployment
		"", // errorMessage
		createdBy,
		req.AutoDetectPrometheus,
		req.PrometheusURL,
	)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   h.namespace,
			Labels:      labels,
			Annotations: annotations,
		},
	}

	if err := h.client.Create(ctx, configMap); err != nil {
		logger.Error(err, "Failed to create visualize instance ConfigMap", "name", req.Name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create krkn-visualize instance",
		})
		return
	}

	logger.Info("Created krkn-visualize instance ConfigMap", "name", req.Name, "targetCluster", targetCluster, "createdBy", createdBy)

	// Fetch Elasticsearch config if specified
	var esURL, esUsername, esPassword string
	if req.ElasticsearchConfigName != "" {
		var esSecret corev1.Secret
		if err := h.client.Get(ctx, client.ObjectKey{
			Namespace: h.namespace,
			Name:      req.ElasticsearchConfigName,
		}, &esSecret); err == nil {
			// Extract ES config from Secret
			host := esSecret.Annotations["krkn.io/es-host"]
			port := esSecret.Annotations["krkn.io/es-port"]
			if host != "" && port != "" {
				esURL = fmt.Sprintf("%s:%s", host, port)
			}
			esUsername = string(esSecret.Data["username"])
			esPassword = string(esSecret.Data["password"])
			logger.Info("Fetched Elasticsearch config", "configName", req.ElasticsearchConfigName, "esURL", esURL)
		} else {
			logger.Error(err, "Failed to fetch Elasticsearch config", "configName", req.ElasticsearchConfigName)
			// Continue anyway - deployment will proceed without ES config
		}
	}

	// Build deployment configuration
	deployConfig := visualize.DeploymentConfig{
		InstanceName:          req.Name,
		TargetCluster:         targetCluster,
		Namespace:             namespace,
		GrafanaPassword:       req.GrafanaPassword,
		ElasticsearchURL:      esURL,
		ElasticsearchUsername: esUsername,
		ElasticsearchPassword: esPassword,
		PrometheusURL:         req.PrometheusURL,
		PrometheusBearerToken: req.PrometheusBearerToken,
		AutoDetectPrometheus:  req.AutoDetectPrometheus,
	}

	// Update ConfigMap status to Installing
	configMap.Annotations = visualize.UpdateAnnotations(
		configMap.Annotations,
		visualize.StatusInstalling,
		"",
		"",
	)
	if err := h.client.Update(ctx, configMap); err != nil {
		logger.Error(err, "Failed to update ConfigMap status to Installing")
	}

	// Deploy Grafana asynchronously
	go func() {
		// Detach from the request's cancellation (the request completes before
		// deployment finishes) while keeping request-scoped values.
		operationCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Minute)
		defer cancel()
		bgLogger := logger.WithValues("async", true)

		targetClient, err := h.getVisualizeTargetClient(operationCtx, targetCluster)
		var grafanaURL string
		if err == nil {
			err = h.runVisualizeDeployImage(operationCtx, configMap, deployConfig)
		}
		if err == nil {
			// The upstream deploy script creates the Grafana Route in the selected
			// target cluster. Read its spec.host from that same cluster.
			grafanaURL, err = visualize.GetRouteURL(operationCtx, targetClient, namespace, visualize.UpstreamRouteName)
		}
		if err == nil {
			// Enable iframe embedding so the console's Visualize page can show
			// the deployed dashboards. Failure is not fatal: dashboards remain
			// reachable via the external Grafana link.
			if embedErr := visualize.EnableGrafanaEmbedding(operationCtx, targetClient, namespace); embedErr != nil {
				bgLogger.Error(embedErr, "Failed to enable Grafana embedding", "namespace", namespace)
			}
		}

		// Fetch the ConfigMap again (it may have been updated)
		statusCtx, statusCancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer statusCancel()
		var updatedConfigMap corev1.ConfigMap
		if getErr := h.client.Get(statusCtx, client.ObjectKey{
			Namespace: h.namespace,
			Name:      req.Name,
		}, &updatedConfigMap); getErr != nil {
			bgLogger.Error(getErr, "Failed to fetch ConfigMap for status update")
			return
		}

		if err != nil {
			bgLogger.Error(err, "Grafana deployment failed")
			updatedConfigMap.Annotations = visualize.UpdateAnnotations(
				updatedConfigMap.Annotations,
				visualize.StatusFailed,
				"",
				fmt.Sprintf("Deployment failed: %v", err),
			)
		} else {
			bgLogger.Info("Grafana deployment succeeded", "url", grafanaURL)
			updatedConfigMap.Annotations = visualize.UpdateAnnotations(
				updatedConfigMap.Annotations,
				visualize.StatusReady,
				grafanaURL,
				"",
			)
		}

		if updateErr := h.client.Update(statusCtx, &updatedConfigMap); updateErr != nil {
			bgLogger.Error(updateErr, "Failed to update ConfigMap with deployment result")
		}
	}()

	logger.Info("Started Grafana deployment in background", "instance", req.Name)

	writeJSON(w, http.StatusCreated, visualize.CreateVisualizeResponse{
		Message: "krkn-visualize installation started",
		Name:    req.Name,
	})
}

// runVisualizeDeployImage runs the deployment image published by the
// krkn-chaos/visualize repository. The Job itself runs in the operator
// namespace, but its kubeconfig is the selected target's, so deploy.sh applies
// resources to that target cluster.
func (h *Handler) runVisualizeDeployImage(ctx context.Context, owner *corev1.ConfigMap, config visualize.DeploymentConfig) error {
	// The first argument accepts either a target UUID or the cluster name
	// submitted by the visualize install form.
	kubeconfigBase64, err := h.getKubeconfig(ctx, config.TargetCluster, "", "", "")
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig for selected target %q: %w", config.TargetCluster, err)
	}
	kubeconfigBytes, err := base64.StdEncoding.DecodeString(kubeconfigBase64)
	if err != nil {
		return fmt.Errorf("failed to decode kubeconfig for selected target %q: %w", config.TargetCluster, err)
	}

	kubeconfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "visualize-kubeconfig-" + config.InstanceName + "-",
			Namespace:    h.namespace,
			Labels: map[string]string{
				visualize.LabelManagedBy:     visualize.ManagedByValue,
				"app.kubernetes.io/instance": config.InstanceName,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"config":            kubeconfigBytes,
			"grafana-password":  []byte(config.GrafanaPassword),
			"prometheus-bearer": []byte(config.PrometheusBearerToken),
			"es-username":       []byte(config.ElasticsearchUsername),
			"es-password":       []byte(config.ElasticsearchPassword),
		},
	}
	if owner != nil && owner.UID != "" {
		kubeconfigSecret.OwnerReferences = []metav1.OwnerReference{{
			APIVersion: "v1",
			Kind:       "ConfigMap",
			Name:       owner.Name,
			UID:        owner.UID,
		}}
	}
	if err := h.client.Create(ctx, kubeconfigSecret); err != nil {
		return fmt.Errorf("failed to create temporary target kubeconfig Secret: %w", err)
	}
	defer func() {
		if err := h.client.Delete(context.Background(), kubeconfigSecret); err != nil && !apierrors.IsNotFound(err) {
			log.FromContext(ctx).Error(err, "Failed to remove temporary target kubeconfig Secret", "secret", kubeconfigSecret.Name)
		}
	}()

	job := visualize.BuildDeployImageJob(config.InstanceName, h.namespace, kubeconfigSecret.Name, config)
	if err := h.client.Create(ctx, job); err != nil {
		return fmt.Errorf("failed to create krkn-visualize deploy Job: %w", err)
	}
	log.FromContext(ctx).Info("Started upstream krkn-visualize deploy image", "job", job.Name, "targetCluster", config.TargetCluster)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		var current batchv1.Job
		if err := h.client.Get(ctx, client.ObjectKeyFromObject(job), &current); err != nil {
			return fmt.Errorf("failed to read krkn-visualize deploy Job %q: %w", job.Name, err)
		}
		if current.Status.Succeeded > 0 {
			return nil
		}
		if current.Status.Failed > 0 {
			message := "Job reported failure"
			for _, condition := range current.Status.Conditions {
				if condition.Type == batchv1.JobFailed && condition.Message != "" {
					message = condition.Message
					break
				}
			}
			return fmt.Errorf("krkn-visualize deploy Job %q failed: %s", job.Name, message)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for krkn-visualize deploy Job %q: %w", job.Name, ctx.Err())
		case <-ticker.C:
		}
	}
}

// ListVisualizeInstances handles GET /api/v1/visualize
// Lists all krkn-visualize instances (authenticated users)
func (h *Handler) ListVisualizeInstances(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("list-visualize-instances")

	var configMapList corev1.ConfigMapList
	if err := h.client.List(ctx, &configMapList,
		client.InNamespace(h.namespace),
		client.MatchingLabels{
			visualize.LabelManagedBy: visualize.ManagedByValue,
			visualize.LabelComponent: visualize.ComponentValue,
		},
	); err != nil {
		logger.Error(err, "Failed to list visualize instances")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve krkn-visualize instances",
		})
		return
	}

	instances := make([]visualize.VisualizeInstanceResponse, 0, len(configMapList.Items))
	for _, cm := range configMapList.Items {
		instance := visualize.ParseInstanceResponse(cm.Name, cm.Annotations)
		if instance.Status == visualize.StatusReady {
			routeURL, err := h.getVisualizeRouteURL(ctx, instance.TargetCluster, instance.Namespace, visualize.UpstreamRouteName)
			if err != nil {
				logger.Error(err, "Failed to resolve Grafana route host", "instance", cm.Name)
				instance.GrafanaURL = ""
			} else {
				instance.GrafanaURL = routeURL
			}
		}
		instances = append(instances, instance)
	}

	writeJSON(w, http.StatusOK, visualize.ListVisualizeInstancesResponse{
		Instances: instances,
		Total:     len(instances),
	})
}

func (h *Handler) getVisualizeRouteURL(ctx context.Context, targetCluster, namespace, routeName string) (string, error) {
	if h.visualizeRouteURL != nil {
		return h.visualizeRouteURL(ctx, targetCluster, namespace, routeName)
	}
	targetClient, err := h.getVisualizeTargetClient(ctx, targetCluster)
	if err != nil {
		return "", err
	}
	return visualize.GetRouteURL(ctx, targetClient, namespace, routeName)
}

// getVisualizeTargetClient builds an uncached Kubernetes client from the
// selected target's stored kubeconfig. The operator's manager client is
// connected to the host cluster and must only be used for operator resources.
func (h *Handler) getVisualizeTargetClient(ctx context.Context, targetCluster string) (client.Client, error) {
	if h.visualizeTargetClientFactory != nil {
		return h.visualizeTargetClientFactory(ctx, targetCluster)
	}
	kubeconfigBase64, err := h.getKubeconfig(ctx, targetCluster, "", "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig for selected target %q: %w", targetCluster, err)
	}
	kubeconfigBytes, err := base64.StdEncoding.DecodeString(kubeconfigBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode kubeconfig for selected target %q: %w", targetCluster, err)
	}
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig for selected target %q: %w", targetCluster, err)
	}
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add Kubernetes APIs to target client scheme: %w", err)
	}
	if err := routev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add OpenShift Route API to target client scheme: %w", err)
	}
	targetClient, err := client.New(restConfig, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client for selected target %q: %w", targetCluster, err)
	}
	return targetClient, nil
}

// GetVisualizeInstance handles GET /api/v1/visualize/:name
// Gets a specific krkn-visualize instance (authenticated users)
func (h *Handler) GetVisualizeInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("get-visualize-instance")

	// Extract instance name from path: /api/v1/visualize/{name}
	name := strings.TrimPrefix(r.URL.Path, VisualizePath+"/")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Instance name is required",
		})
		return
	}

	var configMap corev1.ConfigMap
	if err := h.client.Get(ctx, client.ObjectKey{
		Namespace: h.namespace,
		Name:      name,
	}, &configMap); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("krkn-visualize instance '%s' not found", name),
			})
			return
		}
		logger.Error(err, "Failed to get visualize instance", "name", name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve instance",
		})
		return
	}

	// Verify it's a visualize ConfigMap
	if configMap.Labels[visualize.LabelComponent] != visualize.ComponentValue {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: fmt.Sprintf("krkn-visualize instance '%s' not found", name),
		})
		return
	}

	instance := visualize.ParseInstanceResponse(configMap.Name, configMap.Annotations)
	writeJSON(w, http.StatusOK, instance)
}

// DeleteVisualizeInstance handles DELETE /api/v1/visualize/:name
// Deletes a krkn-visualize instance (admin only)
func (h *Handler) DeleteVisualizeInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("delete-visualize-instance")

	if !auth.IsAdmin(ctx) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Extract instance name from path
	name := strings.TrimPrefix(r.URL.Path, VisualizePath+"/")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Instance name is required",
		})
		return
	}

	var configMap corev1.ConfigMap
	if err := h.client.Get(ctx, client.ObjectKey{
		Namespace: h.namespace,
		Name:      name,
	}, &configMap); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("krkn-visualize instance '%s' not found", name),
			})
			return
		}
		logger.Error(err, "Failed to get visualize instance for deletion", "name", name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete instance",
		})
		return
	}

	// Verify it's a visualize ConfigMap
	if configMap.Labels[visualize.LabelComponent] != visualize.ComponentValue {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: fmt.Sprintf("krkn-visualize instance '%s' not found", name),
		})
		return
	}

	// Get target namespace from annotations
	targetNamespace := configMap.Annotations[visualize.AnnotationNamespace]

	// Set status to "Deleting" immediately
	configMap.Annotations = visualize.UpdateAnnotations(
		configMap.Annotations,
		visualize.StatusDeleting,
		"",
		"",
	)
	if err := h.client.Update(ctx, &configMap); err != nil {
		logger.Error(err, "Failed to update status to Deleting")
	}

	// Respond immediately
	writeJSON(w, http.StatusOK, visualize.DeleteVisualizeResponse{
		Message: fmt.Sprintf("krkn-visualize instance '%s' deletion started", name),
	})

	// Delete Grafana resources and ConfigMap asynchronously
	go func() {
		// Detach from the request's cancellation while keeping request-scoped values.
		bgCtx := context.WithoutCancel(ctx)
		bgLogger := logger.WithValues("async", true)

		if targetNamespace != "" {
			// Delete this instance's resources before considering namespace cleanup.
			targetCluster := configMap.Annotations[visualize.AnnotationTargetCluster]
			targetClient, err := h.getVisualizeTargetClient(bgCtx, targetCluster)
			if err != nil {
				bgLogger.Error(err, "Failed to create client for selected target cluster", "targetCluster", targetCluster)
				return
			}
			deployer := visualize.NewGrafanaDeployer(targetClient, h.namespace)
			if err := deployer.DeleteGrafana(bgCtx, targetNamespace, name); err != nil {
				bgLogger.Error(err, "Failed to delete Grafana deployment", "namespace", targetNamespace)
				return
			}
			bgLogger.Info("Deleted Grafana instance resources", "namespace", targetNamespace, "instance", name)

			// A namespace may be shared by several visualize instances. Keep it
			// while any other instance in it is not already being deleted.
			var instances corev1.ConfigMapList
			if err := h.client.List(bgCtx, &instances,
				client.InNamespace(h.namespace),
				client.MatchingLabels{visualize.LabelComponent: visualize.ComponentValue},
			); err != nil {
				bgLogger.Error(err, "Failed to check for other visualize instances", "namespace", targetNamespace)
				return
			}
			namespaceInUse := false
			for i := range instances.Items {
				instance := &instances.Items[i]
				if instance.Name == name ||
					instance.Annotations[visualize.AnnotationNamespace] != targetNamespace ||
					instance.Annotations[visualize.AnnotationStatus] == visualize.StatusDeleting {
					continue
				}
				namespaceInUse = true
				break
			}
			if !namespaceInUse {
				if err := deployer.DeleteNamespace(bgCtx, targetNamespace); err != nil {
					bgLogger.Error(err, "Failed to delete visualize namespace", "namespace", targetNamespace)
					return
				}
			}
		}

		// Delete the ConfigMap tracking this instance
		var cm corev1.ConfigMap
		if err := h.client.Get(bgCtx, client.ObjectKey{
			Namespace: h.namespace,
			Name:      name,
		}, &cm); err == nil {
			if err := h.client.Delete(bgCtx, &cm); err != nil {
				bgLogger.Error(err, "Failed to delete visualize instance ConfigMap", "name", name)
			} else {
				bgLogger.Info("Deleted krkn-visualize instance successfully", "name", name)
			}
		}
	}()
}

// GetVisualizeInstanceStatus handles GET /api/v1/visualize/:name/status
// Gets the status of a krkn-visualize instance (authenticated users)
func (h *Handler) GetVisualizeInstanceStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("get-visualize-status")

	// Extract instance name from path: /api/v1/visualize/{name}/status
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, VisualizePath+"/"), "/")
	if len(pathParts) < 1 {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Instance name is required",
		})
		return
	}
	name := pathParts[0]

	var configMap corev1.ConfigMap
	if err := h.client.Get(ctx, client.ObjectKey{
		Namespace: h.namespace,
		Name:      name,
	}, &configMap); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("krkn-visualize instance '%s' not found", name),
			})
			return
		}
		logger.Error(err, "Failed to get visualize instance status", "name", name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve instance status",
		})
		return
	}
	if configMap.Labels[visualize.LabelComponent] != visualize.ComponentValue {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: fmt.Sprintf("krkn-visualize instance '%s' not found", name),
		})
		return
	}

	writeJSON(w, http.StatusOK, visualize.StatusResponse{
		Status:       configMap.Annotations[visualize.AnnotationStatus],
		GrafanaURL:   configMap.Annotations[visualize.AnnotationGrafanaURL],
		ErrorMessage: configMap.Annotations[visualize.AnnotationErrorMessage],
	})
}

// GetVisualizeInstanceDashboards handles GET /api/v1/visualize/:name/dashboards
// Lists the dashboards installed in the instance's Grafana (authenticated users).
//
// The request is proxied server-side to Grafana's search API so the console can
// list dashboards without hitting CORS restrictions or exposing credentials.
// Anonymous Viewer auth is enabled on the deployed Grafana, so no credentials
// are required to read the search API.
func (h *Handler) GetVisualizeInstanceDashboards(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("get-visualize-dashboards")

	// Extract instance name from path: /api/v1/visualize/{name}/dashboards
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, VisualizePath+"/"), "/")
	if len(pathParts) < 1 || pathParts[0] == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Instance name is required",
		})
		return
	}
	name := pathParts[0]

	var configMap corev1.ConfigMap
	if err := h.client.Get(ctx, client.ObjectKey{
		Namespace: h.namespace,
		Name:      name,
	}, &configMap); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("krkn-visualize instance '%s' not found", name),
			})
			return
		}
		logger.Error(err, "Failed to get visualize instance", "name", name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve instance",
		})
		return
	}
	if configMap.Labels[visualize.LabelComponent] != visualize.ComponentValue {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: fmt.Sprintf("krkn-visualize instance '%s' not found", name),
		})
		return
	}

	if configMap.Annotations[visualize.AnnotationStatus] != visualize.StatusReady {
		writeJSONError(w, http.StatusConflict, ErrorResponse{
			Error:   "conflict",
			Message: fmt.Sprintf("krkn-visualize instance '%s' is not ready (status: %s)", name, configMap.Annotations[visualize.AnnotationStatus]),
		})
		return
	}

	// Resolve the Grafana URL. Prefer the live route host; fall back to the
	// URL recorded at deploy time when the route cannot be read.
	grafanaURL, err := h.getVisualizeRouteURL(ctx, configMap.Annotations[visualize.AnnotationTargetCluster], configMap.Annotations[visualize.AnnotationNamespace], visualize.UpstreamRouteName)
	if err != nil {
		grafanaURL = configMap.Annotations[visualize.AnnotationGrafanaURL]
	}
	if grafanaURL == "" {
		writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{
			Error:   "service_unavailable",
			Message: fmt.Sprintf("No Grafana URL available for krkn-visualize instance '%s'", name),
		})
		return
	}

	dashboards, err := h.fetchGrafanaDashboards(ctx, grafanaURL)
	if err != nil {
		logger.Error(err, "Failed to query Grafana dashboards", "name", name, "grafanaURL", grafanaURL)
		writeJSONError(w, http.StatusBadGateway, ErrorResponse{
			Error:   "bad_gateway",
			Message: fmt.Sprintf("Failed to query Grafana for instance '%s': %v", name, err),
		})
		return
	}

	writeJSON(w, http.StatusOK, visualize.DashboardsResponse{
		Dashboards: dashboards,
		Total:      len(dashboards),
	})
}

// fetchGrafanaDashboards queries Grafana's search API for dashboards.
func (h *Handler) fetchGrafanaDashboards(ctx context.Context, grafanaURL string) ([]visualize.GrafanaDashboard, error) {
	searchURL := strings.TrimSuffix(grafanaURL, "/") + "/api/search?type=dash-db"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build Grafana search request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to reach Grafana: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("grafana search returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var dashboards []visualize.GrafanaDashboard
	if err := json.NewDecoder(resp.Body).Decode(&dashboards); err != nil {
		return nil, fmt.Errorf("failed to decode Grafana search response: %w", err)
	}
	return dashboards, nil
}

// GetVisualizeInstanceLogs handles GET /api/v1/visualize/:name/logs
// Gets the deployment status and resource information for a krkn-visualize instance
func (h *Handler) GetVisualizeInstanceLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("get-visualize-logs")

	// Extract instance name from path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, VisualizePath+"/"), "/")
	if len(pathParts) < 1 {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Instance name is required",
		})
		return
	}
	name := pathParts[0]

	// Get instance ConfigMap
	var configMap corev1.ConfigMap
	if err := h.client.Get(ctx, client.ObjectKey{
		Namespace: h.namespace,
		Name:      name,
	}, &configMap); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("krkn-visualize instance '%s' not found", name),
			})
			return
		}
		logger.Error(err, "Failed to get instance")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve instance",
		})
		return
	}
	if configMap.Labels[visualize.LabelComponent] != visualize.ComponentValue {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: fmt.Sprintf("krkn-visualize instance '%s' not found", name),
		})
		return
	}

	// Extract metadata
	targetCluster := configMap.Annotations[visualize.AnnotationTargetCluster]
	targetNamespace := configMap.Annotations[visualize.AnnotationNamespace]
	status := configMap.Annotations[visualize.AnnotationStatus]
	errorMessage := configMap.Annotations[visualize.AnnotationErrorMessage]

	// Build status report
	var report strings.Builder
	report.WriteString("=== krkn-visualize Deployment Status ===\n\n")
	report.WriteString(fmt.Sprintf("Instance: %s\n", name))
	report.WriteString(fmt.Sprintf("Target Cluster: %s\n", targetCluster))
	report.WriteString(fmt.Sprintf("Namespace: %s\n", targetNamespace))
	report.WriteString(fmt.Sprintf("Status: %s\n\n", status))

	if errorMessage != "" {
		report.WriteString(fmt.Sprintf("ERROR: %s\n\n", errorMessage))
	}

	// Check Grafana resources
	// NOTE: We use h.clientset for direct API calls instead of h.client (cached client)
	// because the operator's cache only watches specific namespaces, not the target namespace
	report.WriteString("=== Grafana Resources ===\n\n")

	// Check Deployment using direct API call
	if h.clientset != nil {
		deployment, err := h.clientset.AppsV1().Deployments(targetNamespace).Get(ctx, visualize.ResourceName(name, "grafana"), metav1.GetOptions{})
		if err == nil {
			report.WriteString(fmt.Sprintf("✓ Deployment: %d/%d replicas ready\n",
				deployment.Status.ReadyReplicas, *deployment.Spec.Replicas))
		} else if apierrors.IsNotFound(err) {
			report.WriteString("✗ Deployment: Not found\n")
		} else {
			report.WriteString(fmt.Sprintf("⚠ Deployment: Error (%v)\n", err))
		}

		// Check Service using direct API call
		service, err := h.clientset.CoreV1().Services(targetNamespace).Get(ctx, visualize.ResourceName(name, "grafana"), metav1.GetOptions{})
		if err == nil {
			report.WriteString(fmt.Sprintf("✓ Service: %s (%s)\n", service.Spec.ClusterIP, service.Spec.Type))
		} else if apierrors.IsNotFound(err) {
			report.WriteString("✗ Service: Not found\n")
		} else {
			report.WriteString(fmt.Sprintf("⚠ Service: Error (%v)\n", err))
		}
	} else {
		report.WriteString("⚠ Cannot check Deployment/Service: clientset not available\n")
	}

	// Check Route using OpenShift client (direct API call)
	var grafanaURL string
	if routeClient, err := h.getRouteClient(); err == nil {
		route, err := routeClient.Routes(targetNamespace).Get(ctx, visualize.ResourceName(name, "grafana"), metav1.GetOptions{})
		if err == nil {
			if url, urlErr := visualize.RouteURL(route); urlErr == nil {
				grafanaURL = url
				report.WriteString(fmt.Sprintf("✓ Route: %s\n", route.Spec.Host))
			} else {
				report.WriteString("⚠ Route: Found but host not assigned yet\n")
			}
		} else if apierrors.IsNotFound(err) {
			report.WriteString("✗ Route: Not found\n")
		} else {
			report.WriteString(fmt.Sprintf("⚠ Route: Error (%v)\n", err))
		}
	} else {
		report.WriteString(fmt.Sprintf("⚠ Route: Cannot create client (%v)\n", err))
	}

	if grafanaURL != "" {
		report.WriteString(fmt.Sprintf("\n=== Access Grafana ===\n\nURL: %s\nUsername: admin\nPassword: <configured-password>\n", grafanaURL))
	}

	// Get Grafana pod logs
	if h.clientset != nil && targetNamespace != "" {
		report.WriteString("\n=== Grafana Pod Logs ===\n\n")

		// List pods with grafana label
		podList, err := h.clientset.CoreV1().Pods(targetNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=grafana",
		})

		if err != nil {
			report.WriteString(fmt.Sprintf("Failed to list pods: %v\n", err))
		} else if len(podList.Items) == 0 {
			report.WriteString("No Grafana pods found\n")
		} else {
			// Get logs from the first pod
			pod := podList.Items[0]
			report.WriteString(fmt.Sprintf("Pod: %s (Phase: %s)\n\n", pod.Name, pod.Status.Phase))

			// Get pod logs (last 100 lines)
			logOpts := &corev1.PodLogOptions{
				TailLines: func() *int64 { lines := int64(100); return &lines }(),
			}

			req := h.clientset.CoreV1().Pods(targetNamespace).GetLogs(pod.Name, logOpts)
			podLogs, err := req.Stream(ctx)
			if err != nil {
				report.WriteString(fmt.Sprintf("Failed to get pod logs: %v\n", err))
			} else {
				defer podLogs.Close()
				logBytes, err := io.ReadAll(podLogs)
				if err != nil {
					report.WriteString(fmt.Sprintf("Failed to read pod logs: %v\n", err))
				} else {
					report.WriteString(string(logBytes))
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, visualize.LogsResponse{
		Logs:   report.String(),
		Status: status,
	})
}

// getRouteClient creates an OpenShift route client for direct API access
func (h *Handler) getRouteClient() (routeclient.RouteV1Interface, error) {
	if h.visualizeRouteClientFactory != nil {
		return h.visualizeRouteClientFactory()
	}
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get rest config: %w", err)
	}
	return routeclient.NewForConfig(cfg)
}

// VisualizeRouter handles routing for /api/v1/visualize endpoints
func (h *Handler) VisualizeRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Exact match: /api/v1/visualize -> list all instances
		if r.URL.Path == VisualizePath || r.URL.Path == VisualizePath+"/" {
			h.ListVisualizeInstances(w, r)
			return
		}
		// Check if this is a logs request: /api/v1/visualize/{name}/logs
		if strings.HasSuffix(r.URL.Path, "/logs") {
			h.GetVisualizeInstanceLogs(w, r)
			return
		}
		// Check if this is a status request: /api/v1/visualize/{name}/status
		if strings.HasSuffix(r.URL.Path, "/status") {
			h.GetVisualizeInstanceStatus(w, r)
			return
		}
		// Check if this is a dashboards request: /api/v1/visualize/{name}/dashboards
		if strings.HasSuffix(r.URL.Path, "/dashboards") {
			h.GetVisualizeInstanceDashboards(w, r)
			return
		}
		// Otherwise: /api/v1/visualize/{name} -> get specific instance
		h.GetVisualizeInstance(w, r)
	case http.MethodPost:
		h.CreateVisualizeInstance(w, r)
	case http.MethodDelete:
		h.DeleteVisualizeInstance(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: fmt.Sprintf("Method %s not allowed", r.Method),
		})
	}
}

// visualizeInstanceExists checks if a visualize instance ConfigMap exists
func (h *Handler) visualizeInstanceExists(ctx context.Context, name string) (bool, error) {
	var configMap corev1.ConfigMap
	err := h.client.Get(ctx, client.ObjectKey{
		Namespace: h.namespace,
		Name:      name,
	}, &configMap)

	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	// Verify it's a visualize ConfigMap
	if configMap.Labels[visualize.LabelComponent] != visualize.ComponentValue {
		return false, nil
	}

	return true, nil
}
