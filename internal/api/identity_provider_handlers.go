package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/identity"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
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
	HasCACert            bool   `json:"hasCACert"`
	SkipTLSVerify        bool   `json:"skipTLSVerify"`
	Enabled              bool   `json:"enabled"`
	Deleting             bool   `json:"deleting"`
	Source               string `json:"source"`
	HasClientSecret      bool   `json:"hasClientSecret"`
	HasAdminClientSecret bool   `json:"hasAdminClientSecret"`
}

func providerResponse(c identityProviderConfig) IdentityProviderResponse {
	return IdentityProviderResponse{ID: c.ID, Type: c.Type, DisplayName: c.DisplayName,
		Issuer: c.Issuer, ClientID: c.ClientID, RedirectURL: c.RedirectURL,
		AdminClientID: c.AdminClientID, HasCACert: c.CACert != "", SkipTLSVerify: c.SkipTLSVerify,
		Enabled: c.Enabled, Deleting: c.Deleting, Source: c.Source,
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
	CACert            string `json:"caCert"`
	SkipTLSVerify     bool   `json:"skipTLSVerify"`
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
	CACert            *string `json:"caCert,omitempty"`
	SkipTLSVerify     *bool   `json:"skipTLSVerify,omitempty"`
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
// and issuers are immutable; deleting a provider also clears its external group
// mappings, while user records and blacklist entries remain available.
func (h *Handler) IdentityProviders(w http.ResponseWriter, r *http.Request) {
	if !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{Error: "forbidden", Message: "Admin privileges required"})
		return
	}
	if r.URL.Path == IdentityProvidersPath {
		if r.Method == http.MethodPost {
			h.identityProviderMutationMu.Lock()
			defer h.identityProviderMutationMu.Unlock()
		}
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
				AdminClientSecret: req.AdminClientSecret, CACert: req.CACert,
				SkipTLSVerify: req.SkipTLSVerify, Enabled: req.Enabled, Source: "secret"}
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
			providerSecret := c.secret(h.namespace)
			if err := h.providerStore.client.Create(r.Context(), providerSecret); err != nil {
				if !apierrors.IsAlreadyExists(err) {
					writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot create identity provider"})
					return
				}
				var existing corev1.Secret
				key := client.ObjectKeyFromObject(providerSecret)
				if getErr := h.providerStore.reader.Get(r.Context(), key, &existing); getErr != nil {
					writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot inspect existing identity provider"})
					return
				}
				if existing.Labels[identityProviderTombstoneLabel] != "true" {
					writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "conflict", Message: "Provider ID already exists"})
					return
				}
				existing.Data = providerSecret.Data
				existing.Type = providerSecret.Type
				existing.Labels = providerSecret.Labels
				if err := h.providerStore.client.Update(r.Context(), &existing); err != nil {
					if apierrors.IsConflict(err) {
						writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "conflict", Message: "Provider tombstone changed concurrently; retry creation"})
					} else {
						writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot replace deleted identity provider"})
					}
					return
				}
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
	if r.Method == http.MethodDelete {
		h.deleteIdentityProvider(w, r, id)
		return
	}
	if r.Method != http.MethodPatch {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	h.identityProviderMutationMu.Lock()
	defer h.identityProviderMutationMu.Unlock()
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
	if secret.Labels[identityProviderTombstoneLabel] == "true" {
		http.NotFound(w, r)
		return
	}
	c, err := secretProvider(&secret)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Invalid stored provider"})
		return
	}
	if c.Deleting {
		writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "provider_deleting", Message: "Identity provider deletion is in progress"})
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
	if req.CACert != nil {
		c.CACert = *req.CACert
	}
	if req.SkipTLSVerify != nil {
		c.SkipTLSVerify = *req.SkipTLSVerify
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

func (h *Handler) deleteIdentityProvider(w http.ResponseWriter, r *http.Request, id string) {
	h.identityProviderMutationMu.Lock()
	defer h.identityProviderMutationMu.Unlock()

	var secret corev1.Secret
	key := client.ObjectKey{Name: identityProviderSecretName(id), Namespace: h.namespace}
	if err := h.providerStore.reader.Get(r.Context(), key, &secret); err != nil {
		if apierrors.IsNotFound(err) {
			if id == "keycloak" {
				if _, ok := legacyKeycloakConfig(); ok {
					writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "environment_provider", Message: "Environment providers are read-only and cannot be deleted"})
					return
				}
			}
			removed, cleanupErr := h.removeProviderGroupMappings(r.Context(), id)
			if cleanupErr != nil {
				writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot remove provider group mappings"})
				return
			}
			if removed == 0 {
				writeJSONError(w, http.StatusNotFound, ErrorResponse{Error: "not_found", Message: "Identity provider not found"})
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot read identity provider"})
		return
	}
	if secret.Labels[identityProviderTombstoneLabel] == "true" {
		if _, err := h.removeProviderGroupMappings(r.Context(), id); err != nil {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot remove provider group mappings"})
			return
		}
		if err := h.verifyProviderGroupMappingsRemoved(r.Context(), id); err != nil {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Provider group mappings remain after cleanup"})
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if secret.Labels[identityProviderLabel] != "true" {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{Error: "not_found", Message: "Identity provider not found"})
		return
	}
	if id == "keycloak" {
		if _, ok := legacyKeycloakConfig(); ok {
			writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "environment_provider", Message: "Deleting this override would reactivate the environment-configured provider"})
			return
		}
	}

	if err := h.markIdentityProviderDeleting(r.Context(), key); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot disable identity provider for deletion"})
		return
	}
	if err := h.providerStore.reader.Get(r.Context(), key, &secret); err != nil {
		if apierrors.IsNotFound(err) {
			if _, cleanupErr := h.removeProviderGroupMappings(r.Context(), id); cleanupErr != nil {
				writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot remove provider group mappings"})
				return
			}
			if verifyErr := h.verifyProviderGroupMappingsRemoved(r.Context(), id); verifyErr != nil {
				writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Provider group mappings remain after cleanup"})
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot read identity provider during deletion"})
		return
	}
	_, err := h.removeProviderGroupMappings(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot remove provider group mappings"})
		return
	}
	if err := h.verifyProviderGroupMappingsRemoved(r.Context(), id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Provider group mappings remain after cleanup"})
		return
	}
	if err := h.replaceProviderWithTombstone(r.Context(), key, secret); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Cannot remove identity provider configuration"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) replaceProviderWithTombstone(ctx context.Context, key client.ObjectKey, secret corev1.Secret) error {
	secret.Data = map[string][]byte{"deleting": []byte("true")}
	secret.Labels = map[string]string{
		identityProviderLabel:          "true",
		identityProviderTombstoneLabel: "true",
		"app.kubernetes.io/name":       "krkn-operator",
	}
	if err := h.providerStore.client.Update(ctx, &secret); err != nil {
		return fmt.Errorf("replace provider Secret %s with deletion tombstone: %w", key.Name, err)
	}
	return nil
}

func (h *Handler) markIdentityProviderDeleting(ctx context.Context, key client.ObjectKey) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var secret corev1.Secret
		if err := h.providerStore.reader.Get(ctx, key, &secret); err != nil {
			return err
		}
		if string(secret.Data["deleting"]) == "true" {
			return nil
		}
		epoch, err := randomURLValue()
		if err != nil {
			return err
		}
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data["enabled"] = []byte("false")
		secret.Data["deleting"] = []byte("true")
		secret.Data["sessionEpoch"] = []byte(epoch)
		return h.providerStore.client.Update(ctx, &secret)
	})
}

// removeProviderGroupMappings clears grants that reference a provider before
// its configuration is removed. User records and blacklist entries are kept.
func (h *Handler) removeProviderGroupMappings(ctx context.Context, providerID string) (int, error) {
	var groups krknv1alpha1.KrknUserGroupList
	if err := h.providerStore.reader.List(ctx, &groups, client.InNamespace(h.namespace)); err != nil {
		return 0, fmt.Errorf("list local groups: %w", err)
	}
	removed := 0
	for i := range groups.Items {
		groupName := groups.Items[i].Name
		containsProvider := false
		for _, mapping := range groups.Items[i].Spec.ExternalGroups {
			if mapping.Provider == providerID {
				containsProvider = true
				break
			}
		}
		if !containsProvider {
			continue
		}
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			var group krknv1alpha1.KrknUserGroup
			if err := h.providerStore.reader.Get(ctx, client.ObjectKey{Name: groupName, Namespace: h.namespace}, &group); err != nil {
				return err
			}
			mappings := make([]krknv1alpha1.ExternalGroupMapping, 0, len(group.Spec.ExternalGroups))
			for _, mapping := range group.Spec.ExternalGroups {
				if mapping.Provider != providerID {
					mappings = append(mappings, mapping)
				}
			}
			group.Spec.ExternalGroups = mappings
			return h.providerStore.client.Update(ctx, &group)
		}); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return removed, fmt.Errorf("remove provider mapping from group %q: %w", groupName, err)
		}
		removed++
	}
	return removed, nil
}

func (h *Handler) verifyProviderGroupMappingsRemoved(ctx context.Context, providerID string) error {
	var groups krknv1alpha1.KrknUserGroupList
	if err := h.providerStore.reader.List(ctx, &groups, client.InNamespace(h.namespace)); err != nil {
		return fmt.Errorf("list local groups after provider cleanup: %w", err)
	}
	for i := range groups.Items {
		for _, mapping := range groups.Items[i].Spec.ExternalGroups {
			if mapping.Provider == providerID {
				return fmt.Errorf("provider mapping remains in group %q", groups.Items[i].Name)
			}
		}
	}
	return nil
}
