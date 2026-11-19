package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/identity"
)

type assigningUIDClient struct{ client.Client }

func (c assigningUIDClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if user, ok := obj.(*krknv1alpha1.KrknUser); ok {
		user.UID = types.UID("created-uid")
	}
	return c.Client.Create(ctx, obj, opts...)
}

func TestKeycloakCallbackProvisionsUserAndRejectsBadNonce(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var issuer, nonce string
	badNonce := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/main/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer": issuer, "authorization_endpoint": issuer + "/protocol/openid-connect/auth",
				"token_endpoint": issuer + "/protocol/openid-connect/token", "jwks_uri": issuer + "/protocol/openid-connect/certs",
				"response_types_supported": []string{"code"}, "subject_types_supported": []string{"public"},
				"id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/realms/main/protocol/openid-connect/certs":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{
				"kty": "RSA", "kid": "test", "use": "sig", "alg": "RS256",
				"n": base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
				"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
			}}})
		case "/realms/main/protocol/openid-connect/token":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			if r.Form.Get("code_verifier") == "" {
				t.Error("missing PKCE verifier")
			}
			claimNonce := nonce
			if badNonce {
				claimNonce = "wrong"
			}
			idToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
				"iss": issuer, "sub": "keycloak-user-1", "aud": "operator-client",
				"iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix(),
				"nonce": claimNonce, "preferred_username": "tullio", "email": "tullio@example.com",
				"groups": []string{"/Ops"},
			})
			idToken.Header["kid"] = "test"
			signed, err := idToken.SignedString(privateKey)
			if err != nil {
				t.Error(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access", "token_type": "Bearer", "expires_in": 60, "id_token": signed})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	issuer = server.URL + "/realms/main"
	t.Setenv("KEYCLOAK_ISSUER_URL", issuer)
	t.Setenv("KEYCLOAK_OIDC_CLIENT_ID", "operator-client")
	t.Setenv("KEYCLOAK_OIDC_CLIENT_SECRET", "test-secret")
	t.Setenv("KEYCLOAK_OIDC_REDIRECT_URL", "https://operator.example"+KeycloakCallback)
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = krknv1alpha1.AddToScheme(scheme)
	k8s := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(TestJWTSecret("default")).WithStatusSubresource(&krknv1alpha1.KrknUser{}).Build()
	manager, err := auth.NewTestSecretManager(k8s, "default")
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(assigningUIDClient{k8s}, nil, "default", "", manager, KrknAIServiceConfig{})
	defer h.Shutdown()
	login := func() (*http.Cookie, string) {
		w := httptest.NewRecorder()
		h.StartKeycloakLogin(w, httptest.NewRequest(http.MethodGet, KeycloakStart, nil))
		if w.Code != http.StatusFound {
			t.Fatalf("start status=%d body=%s", w.Code, w.Body.String())
		}
		redirect, err := url.Parse(w.Header().Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		if redirect.Query().Get("code_challenge_method") != "S256" {
			t.Fatal("PKCE S256 missing")
		}
		nonce = redirect.Query().Get("nonce")
		if nonce == "" {
			t.Fatal("nonce missing")
		}
		for _, cookie := range w.Result().Cookies() {
			if cookie.Name == oidcStateCookie {
				return cookie, redirect.Query().Get("state")
			}
		}
		t.Fatal("state cookie missing")
		return nil, ""
	}
	stateCookie, state := login()
	req := httptest.NewRequest(http.MethodGet, KeycloakCallback+"?state="+url.QueryEscape(state)+"&code=code", nil)
	req.AddCookie(stateCookie)
	w := httptest.NewRecorder()
	h.CompleteKeycloakLogin(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("callback status=%d body=%s", w.Code, w.Body.String())
	}
	var session *http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == auth.SessionCookieName {
			session = cookie
		}
	}
	if session == nil || !session.HttpOnly || !session.Secure {
		t.Fatalf("session cookie=%v", session)
	}
	generator, err := manager.GetTokenGenerator()
	if err != nil {
		t.Fatal(err)
	}
	claims, err := generator.ValidateToken(session.Value)
	if err != nil {
		t.Fatal(err)
	}
	key := identity.ExternalIdentity{Provider: "keycloak", Issuer: issuer, Subject: "keycloak-user-1"}
	userID, _ := key.UserID()
	if claims.UserID != userID || claims.AccountUID != "created-uid" || len(claims.Groups) != 1 || claims.Groups[0] != "/Ops" {
		t.Fatalf("session claims=%+v", claims)
	}
	var user krknv1alpha1.KrknUser
	if err := k8s.Get(context.Background(), client.ObjectKey{Name: sanitizeUsername(userID), Namespace: "default"}, &user); err != nil {
		t.Fatal(err)
	}
	if user.Spec.ExternalUsername != "tullio" || user.Spec.PasswordSecretRef != "" {
		t.Fatalf("user spec=%+v", user.Spec)
	}

	badNonce = true
	stateCookie, state = login()
	req = httptest.NewRequest(http.MethodGet, KeycloakCallback+"?state="+url.QueryEscape(state)+"&code="+strconv.Itoa(2), nil)
	req.AddCookie(stateCookie)
	w = httptest.NewRecorder()
	h.CompleteKeycloakLogin(w, req)
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "Invalid ID token") {
		t.Fatalf("bad nonce status=%d body=%s", w.Code, w.Body.String())
	}
}
