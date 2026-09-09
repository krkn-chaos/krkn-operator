package api

import (
	"encoding/json"
	"net/http"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/signatureverification"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SignatureVerificationSettingsHandler handles the operator-wide image
// signature verification setting. GET is available to every authenticated
// user; PATCH is restricted to administrators.
//
// @Summary Get or update image signature verification
// @Description GET reports whether image signature verification is enabled. PATCH updates the setting and requires an administrator.
// @Tags operator
// @Accept json
// @Produce json
// @Param request body SignatureVerificationSettingsRequest false "Signature verification setting"
// @Success 200 {object} SignatureVerificationSettingsResponse
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 403 {object} ErrorResponse "Admin privileges required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /operator/signature-verification [get]
// @Router /operator/signature-verification [patch]
func (h *Handler) SignatureVerificationSettingsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		enabled, err := signatureverification.GetEnabled(ctx, h.client, h.namespace)
		if err != nil {
			log.FromContext(ctx).Error(err, "failed to read image signature verification setting")
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to read image signature verification setting"})
			return
		}
		writeJSON(w, http.StatusOK, SignatureVerificationSettingsResponse{Enabled: enabled})
	case http.MethodPatch:
		if !auth.IsAdmin(ctx) {
			writeJSONError(w, http.StatusForbidden, ErrorResponse{Error: "forbidden", Message: "This operation requires admin privileges"})
			return
		}
		var request SignatureVerificationSettingsRequest
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&request); err != nil || request.Enabled == nil {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "enabled must be a boolean"})
			return
		}
		if err := signatureverification.SetEnabled(ctx, h.client, h.namespace, *request.Enabled); err != nil {
			log.FromContext(ctx).Error(err, "failed to update image signature verification setting", "enabled", *request.Enabled)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to update image signature verification setting"})
			return
		}
		writeJSON(w, http.StatusOK, SignatureVerificationSettingsResponse{Enabled: *request.Enabled})
	default:
		w.Header().Set("Allow", "GET, PATCH")
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method_not_allowed", Message: "Only GET and PATCH are allowed"})
	}
}
