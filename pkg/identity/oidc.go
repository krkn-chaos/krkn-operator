package identity

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCPrincipal is a verified user identity returned by an OIDC authenticator.
// Groups are the exact values emitted by the provider's groups claim.
type OIDCPrincipal struct {
	Identity   ExternalIdentity
	Username   string
	Email      string
	GivenName  string
	FamilyName string
	Groups     []string
}

// OIDCAuthenticator is the public login contract for OIDC providers. The API
// layer owns state, nonce, PKCE verifier storage and operator session issuance;
// implementations own provider discovery, code exchange and ID token checking.
type OIDCAuthenticator interface {
	Name() string
	AuthCodeURL(ctx context.Context, state, nonce, verifier string) (string, error)
	VerifyCode(ctx context.Context, code, verifier, expectedNonce string) (*OIDCPrincipal, error)
}

// OIDCClient implements OIDCAuthenticator for any compliant OIDC issuer.
// Keycloak and Dex differ in configuration and group claim mapping, not flow.
type OIDCClient struct {
	name, issuer, clientID, clientSecret, redirectURL string
	httpClient                                        *http.Client
}

// NewOIDCClient configures a provider without making a network request.
// clientSecret may be empty for a public PKCE client; confidential deployments
// should supply it from a Kubernetes Secret.
func NewOIDCClient(name, issuer, clientID, clientSecret, redirectURL string, client *http.Client) (*OIDCClient, error) {
	if name == "" || issuer == "" || clientID == "" || redirectURL == "" {
		return nil, fmt.Errorf("incomplete OIDC provider configuration")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &OIDCClient{name: name, issuer: issuer, clientID: clientID, clientSecret: clientSecret, redirectURL: redirectURL, httpClient: client}, nil
}

// Name returns the provider alias used in external group mappings.
func (c *OIDCClient) Name() string { return c.name }

func (c *OIDCClient) config(ctx context.Context) (*oidc.Provider, *oauth2.Config, error) {
	ctx, cancel := context.WithTimeout(oidc.ClientContext(ctx, c.httpClient), 10*time.Second)
	defer cancel()
	provider, err := oidc.NewProvider(ctx, c.issuer)
	if err != nil {
		return nil, nil, fmt.Errorf("discover OIDC issuer: %w", err)
	}
	return provider, &oauth2.Config{ClientID: c.clientID, ClientSecret: c.clientSecret,
		RedirectURL: c.redirectURL, Endpoint: provider.Endpoint(), Scopes: []string{oidc.ScopeOpenID, "profile", "email"}}, nil
}

// AuthCodeURL starts Authorization Code + PKCE using caller-generated state,
// nonce and verifier. The caller must store and validate these values securely.
func (c *OIDCClient) AuthCodeURL(ctx context.Context, state, nonce, verifier string) (string, error) {
	if state == "" || nonce == "" || verifier == "" {
		return "", fmt.Errorf("incomplete OIDC login transaction")
	}
	_, config, err := c.config(ctx)
	if err != nil {
		return "", err
	}
	return config.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)), nil
}

type oidcClaims struct {
	Username   string   `json:"preferred_username"`
	Email      string   `json:"email"`
	GivenName  string   `json:"given_name"`
	FamilyName string   `json:"family_name"`
	Groups     []string `json:"groups"`
}

// VerifyCode exchanges the code with PKCE, verifies the ID token's issuer,
// signature, audience, expiry and nonce, then returns the stable identity.
func (c *OIDCClient) VerifyCode(ctx context.Context, code, verifier, expectedNonce string) (*OIDCPrincipal, error) {
	if code == "" || verifier == "" || expectedNonce == "" {
		return nil, fmt.Errorf("incomplete OIDC callback")
	}
	ctx, cancel := context.WithTimeout(oidc.ClientContext(ctx, c.httpClient), 10*time.Second)
	defer cancel()
	provider, config, err := c.config(ctx)
	if err != nil {
		return nil, err
	}
	tokens, err := config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("OIDC code exchange: %w", err)
	}
	rawID, ok := tokens.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("missing OIDC ID token")
	}
	id, err := provider.Verifier(&oidc.Config{ClientID: c.clientID}).Verify(ctx, rawID)
	if err != nil || id.Nonce != expectedNonce || id.Subject == "" {
		return nil, fmt.Errorf("invalid OIDC ID token or nonce")
	}
	var claims oidcClaims
	if err := id.Claims(&claims); err != nil {
		return nil, fmt.Errorf("decode OIDC claims: %w", err)
	}
	username := claims.Username
	if username == "" {
		username = claims.Email
	}
	if username == "" {
		username = id.Subject
	}
	return &OIDCPrincipal{Identity: ExternalIdentity{Provider: c.name, Issuer: id.Issuer, Subject: id.Subject},
		Username: username, Email: claims.Email, GivenName: claims.GivenName,
		FamilyName: claims.FamilyName, Groups: claims.Groups}, nil
}
