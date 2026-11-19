package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// KeycloakDirectory lists realm groups through the Keycloak Admin REST API.
// The caller supplies a confidential client's credentials with read-only group
// access. The Keycloak OIDC groups mapper must emit paths for ClaimValue to
// match the values returned by this directory.
type KeycloakDirectory struct {
	provider    string
	groupsURL   string
	credentials clientcredentials.Config
	client      *http.Client
}

// NewKeycloakDirectory builds a group directory for an issuer URL ending in
// /realms/{realm}. clientSecret must come from a Kubernetes Secret. client may
// be nil to use the default HTTP client. No network request is made here.
func NewKeycloakDirectory(provider, issuer, clientID, clientSecret string, client *http.Client) (*KeycloakDirectory, error) {
	if provider == "" || clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("provider and service client credentials are required")
	}
	u, err := url.Parse(issuer)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("invalid Keycloak issuer URL")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[len(parts)-2] != "realms" || parts[len(parts)-1] == "" {
		return nil, fmt.Errorf("Keycloak issuer URL must end with /realms/{realm}")
	}
	realm := parts[len(parts)-1]
	basePath := "/" + strings.Join(parts[:len(parts)-2], "/")
	if basePath == "/" {
		basePath = ""
	}
	adminURL := *u
	adminURL.Path = basePath + "/admin/realms/" + url.PathEscape(realm) + "/groups"
	if client == nil {
		client = http.DefaultClient
	}
	return &KeycloakDirectory{
		provider:  provider,
		groupsURL: adminURL.String(),
		credentials: clientcredentials.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     strings.TrimRight(issuer, "/") + "/protocol/openid-connect/token",
		},
		client: client,
	}, nil
}

// Provider returns the configured provider name used in group mappings.
func (d *KeycloakDirectory) Provider() string { return d.provider }

type keycloakGroup struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Path      string          `json:"path"`
	SubGroups []keycloakGroup `json:"subGroups"`
}

// ListGroups searches the Keycloak Admin API and flattens returned subgroups.
// first and max are the Keycloak pagination offset and size; max is capped at 100.
func (d *KeycloakDirectory) ListGroups(ctx context.Context, query string, first, max int) ([]Group, error) {
	if first < 0 {
		return nil, fmt.Errorf("first must be non-negative")
	}
	if max <= 0 || max > 100 {
		max = 100
	}
	u, err := url.Parse(d.groupsURL)
	if err != nil {
		return nil, err
	}
	params := u.Query()
	params.Set("first", strconv.Itoa(first))
	params.Set("max", strconv.Itoa(max))
	params.Set("briefRepresentation", "false")
	if query != "" {
		params.Set("search", query)
	}
	u.RawQuery = params.Encode()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, d.client)
	httpClient := oauth2.NewClient(ctx, d.credentials.TokenSource(ctx))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Keycloak group request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Keycloak group request returned HTTP %d", resp.StatusCode)
	}
	var raw []keycloakGroup
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode Keycloak groups: %w", err)
	}
	groups := make([]Group, 0, len(raw))
	var add func(keycloakGroup)
	add = func(g keycloakGroup) {
		if g.ID != "" && g.Path != "" {
			groups = append(groups, Group{Provider: d.provider, ID: g.ID, Name: g.Name, Path: g.Path, ClaimValue: g.Path})
		}
		for _, child := range g.SubGroups {
			add(child)
		}
	}
	for _, g := range raw {
		add(g)
	}
	return groups, nil
}
