package api

import (
	"errors"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/krkn-chaos/krkn-operator/internal/crdmigration"
)

// rejectRunWritesDuringCRDMigration prevents new run objects from being
// persisted while the API server may still enforce the previous CRD schema.
func (h *Handler) rejectRunWritesDuringCRDMigration(w http.ResponseWriter, r *http.Request) bool {
	ctx := r.Context()
	if h.clientset != nil {
		_, err := h.clientset.CoreV1().ConfigMaps(h.namespace).Get(ctx, crdmigration.GuardConfigMapName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false
		}
		if err != nil {
			log.FromContext(ctx).Error(err, "unable to check custom resource migration guard", "namespace", h.namespace)
			writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{Error: "service_unavailable", Message: "Run operations are temporarily unavailable during CRD migration"})
			return true
		}
		writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{Error: "service_unavailable", Message: "Run operations are temporarily unavailable during CRD migration"})
		return true
	}
	if h.client == nil {
		log.FromContext(ctx).Error(errors.New("Kubernetes client is unavailable"), "unable to check custom resource migration guard", "namespace", h.namespace)
		writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{Error: "service_unavailable", Message: "Run operations are temporarily unavailable during CRD migration"})
		return true
	}
	guard := &corev1.ConfigMap{}
	err := h.client.Get(ctx, client.ObjectKey{Name: crdmigration.GuardConfigMapName, Namespace: h.namespace}, guard)
	if apierrors.IsNotFound(err) {
		return false
	}
	if err != nil {
		log.FromContext(ctx).Error(err, "unable to check custom resource migration guard", "namespace", h.namespace)
		writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{Error: "service_unavailable", Message: "Run operations are temporarily unavailable during CRD migration"})
		return true
	}
	writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{Error: "service_unavailable", Message: "Run operations are temporarily unavailable during CRD migration"})
	return true
}
