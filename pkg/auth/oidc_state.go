package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// OIDCState records the login transaction in a short-lived, signed cookie.
// It is not a user session and cannot be used as an API credential.
type OIDCState struct {
	Provider string `json:"provider,omitempty"`
	State    string `json:"state"`
	Nonce    string `json:"nonce"`
	Verifier string `json:"verifier"`
	jwt.RegisteredClaims
}

// SignOIDCState signs a five-minute login transaction without server storage.
func (tg *TokenGenerator) SignOIDCState(state, nonce, verifier string) (string, error) {
	return tg.SignOIDCProviderState("", state, nonce, verifier)
}

// SignOIDCProviderState also binds a login transaction to one provider ID.
func (tg *TokenGenerator) SignOIDCProviderState(provider, state, nonce, verifier string) (string, error) {
	if state == "" || nonce == "" || verifier == "" {
		return "", fmt.Errorf("incomplete OIDC state")
	}
	now := time.Now()
	claims := OIDCState{Provider: provider, State: state, Nonce: nonce, Verifier: verifier,
		RegisteredClaims: jwt.RegisteredClaims{Issuer: tg.issuer, Subject: "oidc-state",
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute))}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(tg.secretKey)
}

// VerifyOIDCState validates the signed login transaction and its purpose.
func (tg *TokenGenerator) VerifyOIDCState(raw string) (*OIDCState, error) {
	claims := &OIDCState{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected algorithm")
		}
		return tg.secretKey, nil
	}, jwt.WithIssuer(tg.issuer))
	if err != nil || !token.Valid || claims.Subject != "oidc-state" || claims.State == "" || claims.Nonce == "" || claims.Verifier == "" {
		return nil, fmt.Errorf("invalid OIDC state")
	}
	return claims, nil
}
