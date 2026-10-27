package api

import (
	"net/http"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krkncomponents,verbs=get;list;watch

// ListComponentsResponse contains version manifests keyed by operator name.
type ListComponentsResponse struct {
	Components map[string]krknv1alpha1.ComponentManifest `json:"components"`
}

// ListComponents returns the version manifests published in the operator namespace.
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
	writeJSON(w, http.StatusOK, ListComponentsResponse{Components: components})
}
