package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/identity"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// RegisterGroupDirectory adds a provider directory before the server starts.
// Provider names are unique; replacing an existing name is intentional.
func (h *Handler) RegisterGroupDirectory(directory identity.GroupDirectory) {
	h.groupDirectories[directory.Provider()] = directory
}

// IdentityProvidersRouter sends group discovery to its adapter and all other
// provider paths to the admin configuration API.
func (h *Handler) IdentityProvidersRouter(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/groups") {
		h.ListExternalGroups(w, r)
		return
	}
	h.IdentityProviders(w, r)
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
	logger := log.FromContext(r.Context()).WithName("api")
	if !exists {
		config, err := h.providerStore.get(r.Context(), parts[0])
		if err != nil {
			if errors.Is(err, errIdentityProviderNotFound) {
				http.NotFound(w, r)
			} else {
				logger.Info("Cannot read identity provider for group discovery", "provider", parts[0], "cause", err.Error())
				writeJSON(w, http.StatusBadGateway, ErrorResponse{Error: "provider_unavailable", Message: "Cannot read identity provider"})
			}
			return
		}
		if config.AdminClientID == "" || config.AdminClientSecret == "" {
			logger.Info("Group discovery credentials are not configured", "provider", parts[0])
			writeJSON(w, http.StatusServiceUnavailable, ErrorResponse{Error: "provider_unavailable", Message: "Group discovery credentials are not configured"})
			return
		}
		directory, err = identity.NewKeycloakDirectory(config.ID, config.Issuer,
			config.AdminClientID, config.AdminClientSecret, nil)
		if err != nil {
			logger.Info("Invalid group discovery configuration", "provider", parts[0], "cause", err.Error())
			writeJSON(w, http.StatusBadGateway, ErrorResponse{Error: "provider_unavailable", Message: "Invalid group discovery configuration"})
			return
		}
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
	var page identity.GroupPage
	if paged, ok := directory.(identity.PagedGroupDirectory); ok {
		page, err = paged.ListGroupsPage(r.Context(), r.URL.Query().Get("search"), first, max)
	} else {
		page.Groups, err = directory.ListGroups(r.Context(), r.URL.Query().Get("search"), first, max)
		page.NextFirst = first + len(page.Groups)
		page.HasMore = len(page.Groups) == max
	}
	if err != nil {
		logger.Info("Failed to list external groups", "provider", parts[0], "cause", err.Error())
		writeJSON(w, http.StatusBadGateway, ErrorResponse{Error: "provider_unavailable", Message: "Failed to list external groups"})
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Groups    []identity.Group `json:"groups"`
		NextFirst int              `json:"nextFirst"`
		HasMore   bool             `json:"hasMore"`
	}{Groups: page.Groups, NextFirst: page.NextFirst, HasMore: page.HasMore})
}
