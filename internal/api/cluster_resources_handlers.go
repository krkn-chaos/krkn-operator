/*
Copyright 2026.

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

package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	"github.com/krkn-chaos/krkn-operator/pkg/terminal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type clusterResourceItem struct {
	Metadata struct {
		Name      string            `json:"name"`
		Namespace string            `json:"namespace,omitempty"`
		Labels    map[string]string `json:"labels,omitempty"`
	} `json:"metadata"`
	Spec *struct {
		Containers []struct {
			Name string `json:"name"`
		} `json:"containers,omitempty"`
	} `json:"spec,omitempty"`
}

type clusterResourcesResponse struct {
	Items []clusterResourceItem `json:"items"`
}

// GetClusterResources lists limited resource metadata for scenario parameter suggestions.
//
// @Summary List cluster resources for scenario parameter suggestions
// @Tags clusters
// @Produce json
// @Param id query string true "KrknTargetRequest ID"
// @Param cluster-name query string true "Cluster name"
// @Param operator-name query string true "Target provider name"
// @Param resource query string true "namespaces, nodes, pods, pvcs, vmi, or serviceaccounts"
// @Param namespace query string false "Namespace for namespaced resources"
// @Param all-namespaces query boolean false "List namespaced resources across all namespaces"
// @Success 200 {object} clusterResourcesResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 405 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 502 {object} ErrorResponse
// @Failure 504 {object} ErrorResponse
// @Security BearerAuth
// @Router /cluster-resources [get]
func (h *Handler) GetClusterResources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method_not_allowed", Message: "GET required"})
		return
	}

	query := r.URL.Query()
	targetID := query.Get("id")
	clusterName := query.Get("cluster-name")
	operatorName := query.Get("operator-name")
	resource := query.Get("resource")
	namespace := query.Get("namespace")
	allNamespaces := query.Get("all-namespaces")
	if targetID == "" || clusterName == "" || operatorName == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: "id, cluster-name, and operator-name are required"})
		return
	}

	namespaced := false
	switch resource {
	case "namespaces", "nodes":
	case "pods", "pvcs", "vmi", "serviceaccounts":
		namespaced = true
	default:
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: "Unsupported resource"})
		return
	}
	namespaceRequested := namespace != ""
	allRequested := allNamespaces == "true"
	if (allNamespaces != "" && !allRequested) ||
		(namespaced && namespaceRequested == allRequested) ||
		(!namespaced && (namespaceRequested || allRequested)) ||
		(namespaceRequested && len(validation.IsDNS1123Label(namespace)) != 0) {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: "Invalid namespace scope"})
		return
	}

	ctx := r.Context()
	claims := auth.GetClaimsFromContext(ctx)
	if claims != nil && !auth.IsAdmin(ctx) {
		apiURL, err := h.getClusterAPIURL(ctx, "", targetID, operatorName, clusterName)
		if err != nil {
			h.writeClusterResourceLookupError(w, r, err)
			return
		}
		allowed, err := groupauth.HasClusterPermission(ctx, h.client, claims.UserID, h.namespace, apiURL, groupauth.ActionRun)
		if err != nil {
			log.FromContext(ctx).Error(err, "Failed to check cluster resource permission")
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to check cluster permission"})
			return
		}
		if !allowed {
			writeJSONError(w, http.StatusForbidden, ErrorResponse{Error: "forbidden", Message: "Run permission required for this cluster"})
			return
		}
	}

	kubeconfig, err := h.getKubeconfig(ctx, "", targetID, operatorName, clusterName)
	if err != nil {
		h.writeClusterResourceLookupError(w, r, err)
		return
	}

	command := &terminal.ParsedCommand{
		Command: "kubectl", Subcommand: "get", Args: []string{resource},
		Flags: map[string]string{"o": "json"},
	}
	if namespace != "" {
		command.Flags["n"] = namespace
	} else if allNamespaces == "true" {
		command.BooleanFlags = []string{"A"}
	}
	result, err := h.executeKubectlViaGRPC(ctx, kubeconfig, command)
	if err != nil {
		log.FromContext(ctx).Error(err, "Cluster resource provider call failed", "resource", resource)
		if status.Code(err) == codes.DeadlineExceeded || errors.Is(err, context.DeadlineExceeded) {
			writeJSONError(w, http.StatusGatewayTimeout, ErrorResponse{Error: "provider_timeout", Message: "Cluster resource lookup timed out"})
		} else {
			writeJSONError(w, http.StatusBadGateway, ErrorResponse{Error: "provider_error", Message: "Cluster resource lookup failed"})
		}
		return
	}
	if result.Error != "" || result.ExitCode != 0 {
		log.FromContext(ctx).Info("Cluster resource command failed", "resource", resource, "providerError", result.Error, "exitCode", result.ExitCode)
		if result.Error == "timeout" {
			writeJSONError(w, http.StatusGatewayTimeout, ErrorResponse{Error: "provider_timeout", Message: "Cluster resource lookup timed out"})
		} else {
			writeJSONError(w, http.StatusBadGateway, ErrorResponse{Error: "provider_error", Message: "Cluster resource lookup failed"})
		}
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(result.StdoutBase64)
	if err != nil {
		log.FromContext(ctx).Error(err, "Invalid cluster resource encoding", "resource", resource)
		writeJSONError(w, http.StatusBadGateway, ErrorResponse{Error: "provider_error", Message: "Invalid cluster resource response"})
		return
	}
	var payload struct {
		Items *[]clusterResourceItem `json:"items"`
	}
	if err := json.Unmarshal(decoded, &payload); err != nil {
		log.FromContext(ctx).Error(err, "Invalid cluster resource data", "resource", resource)
		writeJSONError(w, http.StatusBadGateway, ErrorResponse{Error: "provider_error", Message: "Invalid cluster resource response"})
		return
	}
	if payload.Items == nil {
		log.FromContext(ctx).Info("Cluster resource response omitted items", "resource", resource)
		writeJSONError(w, http.StatusBadGateway, ErrorResponse{Error: "provider_error", Message: "Invalid cluster resource response"})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, clusterResourcesResponse{Items: *payload.Items})
}

func (h *Handler) writeClusterResourceLookupError(w http.ResponseWriter, r *http.Request, err error) {
	log.FromContext(r.Context()).Error(err, "Cluster resource target lookup failed")
	if errors.Is(err, errAmbiguousClusterProvider) {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "ambiguous_cluster", Message: "Cluster provider is ambiguous"})
	} else if apierrors.IsNotFound(err) || strings.Contains(err.Error(), "not found") {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{Error: "not_found", Message: "Target or provider not found"})
	} else {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cluster resource lookup failed"})
	}
}
