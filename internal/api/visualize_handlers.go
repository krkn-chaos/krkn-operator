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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/visualize"
	routeclient "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		Data: map[string]string{
			visualize.DataKeyGrafanaPassword: req.GrafanaPassword,
		},
	}

	// Store Prometheus bearer token if provided
	if req.PrometheusBearerToken != "" {
		configMap.Data[visualize.DataKeyPrometheusBearerToken] = req.PrometheusBearerToken
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
		// Create background context (parent context will be cancelled when request completes)
		bgCtx := context.Background()
		bgLogger := logger.WithValues("async", true)

		deployer := visualize.NewGrafanaDeployer(h.client, h.namespace)
		grafanaURL, err := deployer.DeployGrafana(bgCtx, deployConfig)

		// Fetch the ConfigMap again (it may have been updated)
		var updatedConfigMap corev1.ConfigMap
		if getErr := h.client.Get(bgCtx, client.ObjectKey{
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

		if updateErr := h.client.Update(bgCtx, &updatedConfigMap); updateErr != nil {
			bgLogger.Error(updateErr, "Failed to update ConfigMap with deployment result")
		}
	}()

	logger.Info("Started Grafana deployment in background", "instance", req.Name)

	writeJSON(w, http.StatusCreated, visualize.CreateVisualizeResponse{
		Message: "krkn-visualize installation started",
		Name:    req.Name,
	})
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
		instances = append(instances, instance)
	}

	writeJSON(w, http.StatusOK, visualize.ListVisualizeInstancesResponse{
		Instances: instances,
		Total:     len(instances),
	})
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
		bgCtx := context.Background()
		bgLogger := logger.WithValues("async", true)

		if targetNamespace != "" {
			// Delete Grafana deployment and namespace
			deployer := visualize.NewGrafanaDeployer(h.client, h.namespace)
			if err := deployer.DeleteGrafana(bgCtx, targetNamespace); err != nil {
				bgLogger.Error(err, "Failed to delete Grafana deployment", "namespace", targetNamespace)
			} else {
				bgLogger.Info("Deleted Grafana deployment and namespace", "namespace", targetNamespace)
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

	writeJSON(w, http.StatusOK, visualize.StatusResponse{
		Status:       configMap.Annotations[visualize.AnnotationStatus],
		GrafanaURL:   configMap.Annotations[visualize.AnnotationGrafanaURL],
		ErrorMessage: configMap.Annotations[visualize.AnnotationErrorMessage],
	})
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
		deployment, err := h.clientset.AppsV1().Deployments(targetNamespace).Get(ctx, "grafana", metav1.GetOptions{})
		if err == nil {
			report.WriteString(fmt.Sprintf("✓ Deployment: %d/%d replicas ready\n",
				deployment.Status.ReadyReplicas, *deployment.Spec.Replicas))
		} else if apierrors.IsNotFound(err) {
			report.WriteString("✗ Deployment: Not found\n")
		} else {
			report.WriteString(fmt.Sprintf("⚠ Deployment: Error (%v)\n", err))
		}

		// Check Service using direct API call
		service, err := h.clientset.CoreV1().Services(targetNamespace).Get(ctx, "grafana", metav1.GetOptions{})
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
		route, err := routeClient.Routes(targetNamespace).Get(ctx, "grafana", metav1.GetOptions{})
		if err == nil {
			if route.Spec.Host != "" {
				grafanaURL = fmt.Sprintf("https://%s", route.Spec.Host)
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
func (h *Handler) getRouteClient() (*routeclient.RouteV1Client, error) {
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
