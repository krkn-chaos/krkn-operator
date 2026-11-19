package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/identity"
)

// RegisterGroupDirectory adds a provider directory before the server starts.
// Provider names are unique; replacing an existing name is intentional.
func (h *Handler) RegisterGroupDirectory(directory identity.GroupDirectory) {
	h.groupDirectories[directory.Provider()] = directory
}

// ListExternalGroups handles GET /api/v1/identity-providers/{provider}/groups.
// Only operator admins may browse external groups. Provider failures are not
// hidden behind empty results, so the UI can distinguish an outage from no groups.
func (h *Handler) ListExternalGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method_not_allowed", Message: "Only GET is allowed"})
		return
	}
	if !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{Error: "forbidden", Message: "Admin privileges required"})
		return
	}
	path := strings.TrimPrefix(r.URL.Path, IdentityProvidersPath+"/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "groups" {
		http.NotFound(w, r)
		return
	}
	directory, exists := h.groupDirectories[parts[0]]
	if !exists {
		http.NotFound(w, r)
		return
	}
	first, err := strconv.Atoi(r.URL.Query().Get("first"))
	if r.URL.Query().Get("first") == "" {
		first, err = 0, nil
	}
	if err != nil || first < 0 {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "Invalid first offset"})
		return
	}
	max, err := strconv.Atoi(r.URL.Query().Get("max"))
	if r.URL.Query().Get("max") == "" {
		max, err = 50, nil
	}
	if err != nil || max < 1 || max > 100 {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "max must be between 1 and 100"})
		return
	}
	groups, err := directory.ListGroups(r.Context(), r.URL.Query().Get("search"), first, max)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, ErrorResponse{Error: "provider_unavailable", Message: "Failed to list external groups"})
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Groups []identity.Group `json:"groups"`
	}{Groups: groups})
}
