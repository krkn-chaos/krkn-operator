package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/identity"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const oidcStateCookie = "krkn_oidc_state"

// keycloakLoginConfig is intentionally distinct from the Admin API service
// client. Only the public OIDC login client receives user authentication codes.
func keycloakLoginConfig() (identity.OIDCAuthenticator, error) {
	if os.Getenv("KEYCLOAK_OIDC_CLIENT_SECRET") == "" {
		return nil, fmt.Errorf("Keycloak OIDC client secret is required")
	}
	return identity.NewOIDCClient("keycloak", os.Getenv("KEYCLOAK_ISSUER_URL"),
		os.Getenv("KEYCLOAK_OIDC_CLIENT_ID"), os.Getenv("KEYCLOAK_OIDC_CLIENT_SECRET"),
		os.Getenv("KEYCLOAK_OIDC_REDIRECT_URL"), nil)
}

func randomURLValue() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// StartKeycloakLogin starts Authorization Code + PKCE with state and nonce.
func (h *Handler) StartKeycloakLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	provider, err := keycloakLoginConfig()
	if err != nil {
		http.Error(w, "Keycloak login unavailable", http.StatusServiceUnavailable)
		return
	}
	state, err := randomURLValue()
	if err != nil {
		http.Error(w, "Login unavailable", http.StatusInternalServerError)
		return
	}
	nonce, err := randomURLValue()
	if err != nil {
		http.Error(w, "Login unavailable", http.StatusInternalServerError)
		return
	}
	verifier := oauth2.GenerateVerifier()
	tokenGen, err := h.getTokenGenerator()
	if err != nil {
		http.Error(w, "Login unavailable", http.StatusInternalServerError)
		return
	}
	signed, err := tokenGen.SignOIDCState(state, nonce, verifier)
	if err != nil {
		http.Error(w, "Login unavailable", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: oidcStateCookie, Value: signed, Path: KeycloakCallback,
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 300})
	authURL, err := provider.AuthCodeURL(r.Context(), state, nonce, verifier)
	if err != nil {
		http.Error(w, "Keycloak login unavailable", http.StatusServiceUnavailable)
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

// CompleteKeycloakLogin verifies Keycloak's ID token and issues only an
// operator browser session. No Keycloak token is sent to the console.
func (h *Handler) CompleteKeycloakLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cookie, err := r.Cookie(oidcStateCookie)
	if err != nil {
		http.Error(w, "Missing login state", http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: oidcStateCookie, Path: KeycloakCallback, MaxAge: -1,
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
	tokenGen, err := h.getTokenGenerator()
	if err != nil {
		http.Error(w, "Login unavailable", http.StatusInternalServerError)
		return
	}
	state, err := tokenGen.VerifyOIDCState(cookie.Value)
	if err != nil || r.URL.Query().Get("state") != state.State || r.URL.Query().Get("code") == "" {
		http.Error(w, "Invalid login response", http.StatusUnauthorized)
		return
	}
	provider, err := keycloakLoginConfig()
	if err != nil {
		http.Error(w, "Keycloak login unavailable", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	principal, err := provider.VerifyCode(ctx, r.URL.Query().Get("code"), state.Verifier, state.Nonce)
	if err != nil {
		http.Error(w, "Invalid ID token or login response", http.StatusUnauthorized)
		return
	}
	key := principal.Identity

	user, err := h.ensureExternalUser(ctx, key, principal)
	if err != nil {
		http.Error(w, "External account blocked or unavailable", http.StatusForbidden)
		return
	}
	userID := user.Spec.UserID
	session, err := tokenGen.GenerateSessionToken(auth.Claims{UserID: userID, Role: "user",
		Name: principal.GivenName, Surname: principal.FamilyName,
		Provider: key.Provider, Groups: principal.Groups,
		ExternalIssuer: key.Issuer, ExternalSubject: key.Subject, AccountUID: string(user.UID)})
	if err != nil {
		http.Error(w, "Session unavailable", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: auth.SessionCookieName, Value: session, Path: "/",
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: int((8 * time.Hour).Seconds())})
	http.Redirect(w, r, "/", http.StatusFound)
}

// ensureExternalUser creates a user account on first OIDC login. Blocks are
// checked before creation, and an existing record must match issuer + subject.
func (h *Handler) ensureExternalUser(ctx context.Context, key identity.ExternalIdentity, claims *identity.OIDCPrincipal) (*krknv1alpha1.KrknUser, error) {
	if block, err := h.getOIDCBlock(ctx, key); err != nil {
		return nil, err
	} else if block != nil {
		return nil, fmt.Errorf("OIDC identity is blocked")
	}
	userID, err := key.UserID()
	if err != nil {
		return nil, err
	}
	name := sanitizeUsername(userID)
	var user krknv1alpha1.KrknUser
	err = h.identityReader.Get(ctx, client.ObjectKey{Name: name, Namespace: h.namespace}, &user)
	if apierrors.IsNotFound(err) {
		user = krknv1alpha1.KrknUser{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: h.namespace,
			Labels: map[string]string{UserAccountLabel: "true", AdminRoleLabel: "user"}},
			Spec: krknv1alpha1.KrknUserSpec{UserID: userID, Name: claims.GivenName,
				Surname: claims.FamilyName, Role: "user", AuthProvider: key.Provider,
				ExternalIssuer: key.Issuer, ExternalSubject: key.Subject,
				ExternalUsername: claims.Username, ExternalEmail: claims.Email}}
		if err := h.client.Create(ctx, &user); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return nil, fmt.Errorf("create OIDC account: %w", err)
			}
			if err := h.identityReader.Get(ctx, client.ObjectKey{Name: name, Namespace: h.namespace}, &user); err != nil {
				return nil, fmt.Errorf("read concurrently created OIDC account: %w", err)
			}
		} else {
			user.Status.Active = true
			user.Status.Created = metav1.Now()
			if err := h.client.Status().Update(ctx, &user); err != nil {
				return nil, fmt.Errorf("activate OIDC account: %w", err)
			}
		}
	} else if err != nil {
		return nil, fmt.Errorf("read OIDC account: %w", err)
	}
	if user.Spec.UserID != userID || user.Spec.AuthProvider != key.Provider ||
		user.Spec.ExternalIssuer != key.Issuer || user.Spec.ExternalSubject != key.Subject ||
		!user.Status.Active || user.UID == "" {
		return nil, fmt.Errorf("OIDC account is inactive or mismatched")
	}
	// Recheck after account creation to close the block-versus-login race.
	if block, err := h.getOIDCBlock(ctx, key); err != nil {
		return nil, err
	} else if block != nil {
		return nil, fmt.Errorf("OIDC identity is blocked")
	}
	return &user, nil
}
