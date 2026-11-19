package api

import (
	"net/http"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

// GetSession returns the current principal and a CSRF token for cookie sessions.
// The CSRF token is readable only by a same-origin caller; it is never sent in
// the HttpOnly session cookie's JavaScript-accessible storage.
func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method_not_allowed", Message: "Only GET is allowed"})
		return
	}
	claims := auth.GetClaimsFromContext(r.Context())
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized", Message: "Login required"})
		return
	}
	writeJSON(w, http.StatusOK, struct {
		UserID    string `json:"userId"`
		Role      string `json:"role"`
		Provider  string `json:"provider,omitempty"`
		CSRFToken string `json:"csrfToken,omitempty"`
	}{claims.UserID, claims.Role, claims.Provider, claims.CSRFToken})
}
