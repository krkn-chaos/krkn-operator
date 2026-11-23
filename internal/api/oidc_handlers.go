package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/identity"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const oidcStateCookie = "krkn_oidc_state"

// oidcLoginConfig reads the current provider Secret on each login. This keeps
// enable/disable and credential rotation consistent across operator replicas.
func (h *Handler) oidcLoginConfig(ctx context.Context, id string) (identity.OIDCAuthenticator, identityProviderConfig, error) {
	config, err := h.providerStore.get(ctx, id)
	if err != nil {
		return nil, identityProviderConfig{}, err
	}
	if !config.Enabled {
		return nil, identityProviderConfig{}, fmt.Errorf("identity provider is disabled")
	}
	provider, err := identity.NewOIDCClient(config.ID, config.Issuer, config.ClientID, config.ClientSecret, config.RedirectURL, nil)
	return provider, config, err
}

// OIDCRouter handles /api/v1/auth/oidc/{providerID}/{login|callback}.
func (h *Handler) OIDCRouter(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, OIDCBasePath+"/"), "/")
	if len(parts) != 2 || !validProviderID(parts[0]) {
		http.NotFound(w, r)
		return
	}
	switch parts[1] {
	case "login":
		h.startOIDCLogin(w, r, parts[0])
	case "callback":
		h.completeOIDCLogin(w, r, parts[0])
	default:
		http.NotFound(w, r)
	}
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
	h.startOIDCLogin(w, r, "keycloak")
}

func (h *Handler) startOIDCLogin(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	provider, config, err := h.oidcLoginConfig(r.Context(), id)
	if err != nil {
		log.FromContext(r.Context()).Info("OIDC login configuration unavailable", "provider", id, "cause", err.Error())
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
	signed, err := tokenGen.SignOIDCProviderState(id, state, nonce, verifier)
	if err != nil {
		http.Error(w, "Login unavailable", http.StatusInternalServerError)
		return
	}
	callbackPath := OIDCBasePath + "/" + id + "/callback"
	http.SetCookie(w, &http.Cookie{Name: oidcStateCookie, Value: signed, Path: callbackPath,
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 300})
	authURL, err := provider.AuthCodeURL(r.Context(), state, nonce, verifier)
	if err != nil {
		http.Error(w, "Keycloak login unavailable", http.StatusServiceUnavailable)
		return
	}
	if !sameOIDCOrigin(config.Issuer, authURL) {
		http.Error(w, "Keycloak authorization endpoint unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Location", authURL)
	w.WriteHeader(http.StatusFound)
}

// sameOIDCOrigin prevents discovery metadata from sending browser state or
// PKCE challenges to an authorization endpoint outside the configured issuer.
func sameOIDCOrigin(issuer, authorizationURL string) bool {
	issuerURL, issuerErr := url.Parse(issuer)
	authURL, authErr := url.Parse(authorizationURL)
	return issuerErr == nil && authErr == nil && issuerURL.Host != "" && authURL.Host != "" &&
		authURL.User == nil && authURL.Fragment == "" && authURL.Scheme == issuerURL.Scheme &&
		strings.EqualFold(authURL.Host, issuerURL.Host)
}

// CompleteKeycloakLogin verifies Keycloak's ID token and issues only an
// operator browser session. No Keycloak token is sent to the console.
func (h *Handler) CompleteKeycloakLogin(w http.ResponseWriter, r *http.Request) {
	h.completeOIDCLogin(w, r, "keycloak")
}

func (h *Handler) completeOIDCLogin(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cookie, err := r.Cookie(oidcStateCookie)
	if err != nil {
		http.Error(w, "Login session expired or unavailable; start again from the login page", http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: oidcStateCookie, Path: OIDCBasePath + "/" + id + "/callback", MaxAge: -1,
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
	tokenGen, err := h.getTokenGenerator()
	if err != nil {
		http.Error(w, "Login unavailable", http.StatusInternalServerError)
		return
	}
	state, err := tokenGen.VerifyOIDCState(cookie.Value)
	if err != nil || state.Provider != id || r.URL.Query().Get("state") != state.State || r.URL.Query().Get("code") == "" {
		http.Error(w, "Invalid login response", http.StatusUnauthorized)
		return
	}
	provider, _, err := h.oidcLoginConfig(r.Context(), id)
	if err != nil {
		log.FromContext(r.Context()).Info("OIDC callback configuration unavailable", "provider", id, "cause", err.Error())
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
	config, err := h.providerStore.get(ctx, id)
	if err != nil || !config.Enabled || config.Issuer != key.Issuer {
		http.Error(w, "Identity provider unavailable", http.StatusServiceUnavailable)
		return
	}

	user, err := h.ensureExternalUser(ctx, key, principal)
	if err != nil {
		http.Error(w, "External account blocked or unavailable", http.StatusForbidden)
		return
	}
	if err := h.recordExternalLastLogin(ctx, user); err != nil {
		http.Error(w, "External account unavailable", http.StatusServiceUnavailable)
		return
	}
	userID := user.Spec.UserID
	session, err := tokenGen.GenerateSessionToken(auth.Claims{UserID: userID, Role: "user",
		Name: principal.GivenName, Surname: principal.FamilyName,
		Provider: key.Provider, ProviderEpoch: config.SessionEpoch, Groups: principal.Groups,
		ExternalIssuer: key.Issuer, ExternalSubject: key.Subject, AccountUID: string(user.UID)})
	if err != nil {
		http.Error(w, "Session unavailable", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: auth.SessionCookieName, Value: session, Path: "/",
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: int((8 * time.Hour).Seconds())})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) recordExternalLastLogin(ctx context.Context, user *krknv1alpha1.KrknUser) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var current krknv1alpha1.KrknUser
		if err := h.identityReader.Get(ctx, client.ObjectKeyFromObject(user), &current); err != nil {
			return err
		}
		if current.UID != user.UID || !current.Status.Active {
			return fmt.Errorf("OIDC account changed during login")
		}
		current.Status.LastLogin = metav1.Now()
		return h.client.Status().Update(ctx, &current)
	})
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
		}
	} else if err != nil {
		return nil, fmt.Errorf("read OIDC account: %w", err)
	}
	if user.Spec.UserID != userID || user.Spec.AuthProvider != key.Provider ||
		user.Spec.ExternalIssuer != key.Issuer || user.Spec.ExternalSubject != key.Subject || user.UID == "" {
		return nil, fmt.Errorf("OIDC account is mismatched")
	}
	// A failed first status update leaves a matching account with no creation
	// timestamp. Retrying that initial activation permits the next login to heal
	// it; an admin-disabled account has a timestamp and stays disabled.
	if !user.Status.Active && user.Status.Created.IsZero() {
		user.Status.Active = true
		user.Status.Created = metav1.Now()
		if err := h.client.Status().Update(ctx, &user); err != nil {
			return nil, fmt.Errorf("activate OIDC account: %w", err)
		}
	}
	if !user.Status.Active {
		return nil, fmt.Errorf("OIDC account is inactive")
	}
	// Recheck after account creation to close the block-versus-login race.
	if block, err := h.getOIDCBlock(ctx, key); err != nil {
		return nil, err
	} else if block != nil {
		return nil, fmt.Errorf("OIDC identity is blocked")
	}
	return &user, nil
}
