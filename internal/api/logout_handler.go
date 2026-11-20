package api

import (
	"net/http"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

// LogoutSession removes the browser session cookie. Stateless Bearer JWTs are
// unaffected. For immediate operator-side revocation, admins can disable or
// block the account; simply removing a cookie cannot revoke a copied token.
func (h *Handler) LogoutSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method_not_allowed", Message: "Only POST is allowed"})
		return
	}
	http.SetCookie(w, &http.Cookie{Name: auth.SessionCookieName, Value: "", Path: "/",
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}
