package api

import (
	"context"
	"net/http"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krkncomponents,verbs=get;list;watch

// ListComponentsResponse contains version manifests keyed by component name.
type ListComponentsResponse struct {
	Components map[string]krknv1alpha1.ComponentManifest `json:"components"`
}

// ListComponents returns installed component versions and their subcomponents.
//
// @Summary List installed component versions
// @Description Return the exact build tags and described features published by Krkn operators
// @Tags components
// @Produce json
// @Success 200 {object} ListComponentsResponse
// @Failure 405 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /components [get]
func (h *Handler) ListComponents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method_not_allowed", Message: "Only GET is supported"})
		return
	}
	var list krknv1alpha1.KrknComponentList
	namespace := h.componentNamespace
	if namespace == "" {
		namespace = h.namespace
	}
	if err := h.client.List(r.Context(), &list, client.InNamespace(namespace)); err != nil {
		log.FromContext(r.Context()).Error(err, "failed to list KrknComponent resources")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to list components"})
		return
	}
	components := make(map[string]krknv1alpha1.ComponentManifest)
	for _, resource := range list.Items {
		for name, manifest := range resource.Status.Components {
			if name == resource.Name {
				components[name] = manifest
			}
		}
	}
	if operator, ok := components["krkn-operator"]; h.krknAIEnabled && ok {
		operator.Components = append(operator.Components, krknv1alpha1.ComponentDescription{
			Component:   "krkn-ai-operator",
			Description: "Orchestrates Krkn-AI's genetic algorithm loop to run, evaluate, and evolve chaos scenarios across generations.",
		})
		components["krkn-operator"] = operator
	}
	if h.krknAIEnabled && h.artifactClient != nil {
		// Limit this optional lookup to a short UI wait; failures omit only the service.
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		version, err := h.artifactClient.Version(ctx)
		cancel()
		if err != nil {
			log.FromContext(r.Context()).Error(err, "unable to read Krkn-AI component version")
		} else {
			components["krkn-ai-service"] = krknv1alpha1.ComponentManifest{
				Version: version,
				Components: []krknv1alpha1.ComponentDescription{
					{Component: "rest-api", Description: "Provides authenticated REST endpoints to upload, retrieve, and manage Krkn-AI run results and artifacts."},
				},
			}
		}
	}
	writeJSON(w, http.StatusOK, ListComponentsResponse{Components: components})
}
