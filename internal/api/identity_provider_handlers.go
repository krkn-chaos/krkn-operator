package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/identity"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IdentityProviderResponse contains only public configuration and credential
// presence flags. The API never returns client secrets to the browser.
type IdentityProviderResponse struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	DisplayName          string `json:"displayName"`
	Issuer               string `json:"issuer"`
	ClientID             string `json:"clientId"`
	RedirectURL          string `json:"redirectUrl"`
	AdminClientID        string `json:"adminClientId,omitempty"`
	Enabled              bool   `json:"enabled"`
	Source               string `json:"source"`
	HasClientSecret      bool   `json:"hasClientSecret"`
	HasAdminClientSecret bool   `json:"hasAdminClientSecret"`
}

func providerResponse(c identityProviderConfig) IdentityProviderResponse {
	return IdentityProviderResponse{ID: c.ID, Type: c.Type, DisplayName: c.DisplayName,
		Issuer: c.Issuer, ClientID: c.ClientID, RedirectURL: c.RedirectURL,
		AdminClientID: c.AdminClientID, Enabled: c.Enabled, Source: c.Source,
		HasClientSecret: c.ClientSecret != "", HasAdminClientSecret: c.AdminClientSecret != ""}
}

type createIdentityProviderRequest struct {
	ID                string `json:"id"`
	Type              string `json:"type"`
	DisplayName       string `json:"displayName"`
	Issuer            string `json:"issuer"`
	ClientID          string `json:"clientId"`
	ClientSecret      string `json:"clientSecret"`
	RedirectURL       string `json:"redirectUrl"`
	AdminClientID     string `json:"adminClientId"`
	AdminClientSecret string `json:"adminClientSecret"`
	Enabled           bool   `json:"enabled"`
}

type patchIdentityProviderRequest struct {
	DisplayName       *string `json:"displayName,omitempty"`
	Issuer            *string `json:"issuer,omitempty"`
	ClientID          *string `json:"clientId,omitempty"`
	ClientSecret      *string `json:"clientSecret,omitempty"`
	RedirectURL       *string `json:"redirectUrl,omitempty"`
	AdminClientID     *string `json:"adminClientId,omitempty"`
	AdminClientSecret *string `json:"adminClientSecret,omitempty"`
	Enabled           *bool   `json:"enabled,omitempty"`
}

func decodeProviderJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

// PublicIdentityProviders lists enabled login choices without credentials.
func (h *Handler) PublicIdentityProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	configs, _, err := h.providerStore.list(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{Error: "provider_unavailable", Message: "Cannot list identity providers"})
		return
	}
	type loginChoice struct {
		ID          string `json:"id"`
		Type        string `json:"type"`
		DisplayName string `json:"displayName"`
		LoginURL    string `json:"loginUrl"`
	}
	providers := make([]loginChoice, 0, len(configs))
	for _, c := range configs {
		if c.Enabled {
			providers = append(providers, loginChoice{ID: c.ID, Type: c.Type, DisplayName: c.DisplayName,
				LoginURL: OIDCBasePath + "/" + c.ID + "/login"})
		}
	}
	sort.Slice(providers, func(i, j int) bool { return providers[i].ID < providers[j].ID })
	writeJSON(w, http.StatusOK, struct {
		Providers []loginChoice `json:"providers"`
	}{providers})
}

// IdentityProviders manages Secret-backed Keycloak instances for admins. IDs
// and issuers are immutable; disabling is used instead of deleting an identity
// namespace that may still be referenced by users, blocks and group mappings.
func (h *Handler) IdentityProviders(w http.ResponseWriter, r *http.Request) {
	if !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{Error: "forbidden", Message: "Admin privileges required"})
		return
	}
	if r.URL.Path == IdentityProvidersPath {
		switch r.Method {
		case http.MethodGet:
			configs, invalid, err := h.providerStore.list(r.Context())
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot list identity providers"})
				return
			}
			providers := make([]IdentityProviderResponse, 0, len(configs))
			for _, c := range configs {
				providers = append(providers, providerResponse(c))
			}
			sort.Slice(providers, func(i, j int) bool { return providers[i].ID < providers[j].ID })
			writeJSON(w, http.StatusOK, struct {
				Providers        []IdentityProviderResponse `json:"providers"`
				InvalidProviders []invalidIdentityProvider  `json:"invalidProviders"`
			}{providers, invalid})
		case http.MethodPost:
			var req createIdentityProviderRequest
			if err := decodeProviderJSON(r, &req); err != nil {
				writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "Invalid provider request"})
				return
			}
			c := identityProviderConfig{ID: req.ID, Type: req.Type, DisplayName: req.DisplayName,
				Issuer: req.Issuer, ClientID: req.ClientID, ClientSecret: req.ClientSecret,
				RedirectURL: req.RedirectURL, AdminClientID: req.AdminClientID,
				AdminClientSecret: req.AdminClientSecret, Enabled: req.Enabled, Source: "secret"}
			epoch, err := randomURLValue()
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot create session epoch"})
				return
			}
			c.SessionEpoch = epoch
			if err := validateProviderConfig(c); err != nil {
				writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "validation_error", Message: err.Error()})
				return
			}
			if c.AdminClientID != "" {
				if _, err := identity.NewKeycloakDirectory(c.ID, c.Issuer, c.AdminClientID, c.AdminClientSecret, nil); err != nil {
					writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "validation_error", Message: err.Error()})
					return
				}
			}
			if c.ID == "keycloak" {
				if legacy, ok := legacyKeycloakConfig(); ok && legacy.Issuer != c.Issuer {
					writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "conflict", Message: "The legacy keycloak ID is bound to another issuer"})
					return
				}
			}
			if err := h.providerStore.client.Create(r.Context(), c.secret(h.namespace)); err != nil {
				if apierrors.IsAlreadyExists(err) {
					writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "conflict", Message: "Provider ID already exists"})
					return
				}
				writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot create identity provider"})
				return
			}
			writeJSON(w, http.StatusCreated, providerResponse(c))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	id := strings.TrimPrefix(r.URL.Path, IdentityProvidersPath+"/")
	if !validProviderID(id) || strings.Contains(id, "/") || r.URL.Path != IdentityProvidersPath+"/"+id {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodGet {
		c, err := h.providerStore.get(r.Context(), id)
		if err != nil {
			if errors.Is(err, errIdentityProviderNotFound) {
				http.NotFound(w, r)
			} else {
				writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot read identity provider"})
			}
			return
		}
		writeJSON(w, http.StatusOK, providerResponse(c))
		return
	}
	if r.Method != http.MethodPatch {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var secret corev1.Secret
	if err := h.providerStore.reader.Get(r.Context(), client.ObjectKey{Name: identityProviderSecretName(id), Namespace: h.namespace}, &secret); err != nil {
		if apierrors.IsNotFound(err) {
			if id == "keycloak" {
				if _, ok := legacyKeycloakConfig(); ok {
					writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "environment_provider", Message: "Environment providers are read-only; create a Secret-backed provider with the same ID to override"})
					return
				}
			}
			http.NotFound(w, r)
		} else {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot read identity provider"})
		}
		return
	}
	c, err := secretProvider(&secret)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Invalid stored provider"})
		return
	}
	var req patchIdentityProviderRequest
	if err := decodeProviderJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "Invalid provider request"})
		return
	}
	if req.Issuer != nil && *req.Issuer != c.Issuer {
		writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "immutable_issuer", Message: "Issuer cannot be changed; use a new provider ID"})
		return
	}
	if req.DisplayName != nil {
		c.DisplayName = *req.DisplayName
	}
	if req.ClientID != nil {
		c.ClientID = *req.ClientID
	}
	if req.ClientSecret != nil {
		c.ClientSecret = *req.ClientSecret
	}
	if req.RedirectURL != nil {
		c.RedirectURL = *req.RedirectURL
	}
	if req.AdminClientID != nil {
		c.AdminClientID = *req.AdminClientID
	}
	if req.AdminClientSecret != nil {
		c.AdminClientSecret = *req.AdminClientSecret
	}
	if req.Enabled != nil && *req.Enabled != c.Enabled {
		c.Enabled = *req.Enabled
		epoch, err := randomURLValue()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot rotate session epoch"})
			return
		}
		c.SessionEpoch = epoch
	}
	if err := validateProviderConfig(c); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "validation_error", Message: err.Error()})
		return
	}
	if c.AdminClientID != "" {
		if _, err := identity.NewKeycloakDirectory(c.ID, c.Issuer, c.AdminClientID, c.AdminClientSecret, nil); err != nil {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "validation_error", Message: err.Error()})
			return
		}
	}
	secret.Data = c.secret(h.namespace).Data
	if err := h.providerStore.client.Update(r.Context(), &secret); err != nil {
		if apierrors.IsConflict(err) {
			writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "conflict", Message: "Provider changed concurrently; reload and retry"})
			return
		}
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot update identity provider"})
		return
	}
	writeJSON(w, http.StatusOK, providerResponse(c))
}
