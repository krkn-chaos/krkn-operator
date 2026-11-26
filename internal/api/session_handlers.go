package api

import (
	"net/http"
	"sort"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

// GetSession returns the current principal's display names and groups together
// with a CSRF token for cookie sessions.
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
	groups, err := groupauth.GetUserGroups(r.Context(), h.client, claims.UserID, h.namespace)
	if err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{Error: "groups_unavailable", Message: "Cannot resolve current group mappings"})
		return
	}
	mappedGroups := make([]string, 0, len(groups))
	for _, group := range groups {
		mappedGroups = append(mappedGroups, group.Name)
	}
	sort.Strings(mappedGroups)
	writeJSON(w, http.StatusOK, struct {
		UserID       string   `json:"userId"`
		Name         string   `json:"name"`
		Surname      string   `json:"surname"`
		Role         string   `json:"role"`
		Provider     string   `json:"provider,omitempty"`
		MappedGroups []string `json:"mappedGroups"`
		CSRFToken    string   `json:"csrfToken,omitempty"`
	}{claims.UserID, claims.Name, claims.Surname, claims.Role, claims.Provider, mappedGroups, claims.CSRFToken})
}
