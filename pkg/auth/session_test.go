package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type externalCheckerFunc func(context.Context, *Claims) (bool, error)

func (f externalCheckerFunc) IsExternalIdentityActive(ctx context.Context, claims *Claims) (bool, error) {
	return f(ctx, claims)
}

func TestCookieSessionRequiresActiveExternalIdentityAndCSRF(t *testing.T) {
	generator := NewTokenGenerator([]byte("test-jwt-secret-key-32-bytes!!!!"), 24*time.Hour, "krkn-operator")
	token, err := generator.GenerateSessionToken(Claims{UserID: "oidc-user", Provider: "keycloak", ExternalIssuer: "issuer", ExternalSubject: "sub", AccountUID: "uid"})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := generator.ValidateToken(token)
	if err != nil {
		t.Fatal(err)
	}
	active := true
	middleware := NewMiddleware(generator)
	middleware.SetExternalIdentityChecker(externalCheckerFunc(func(_ context.Context, got *Claims) (bool, error) {
		if got.AccountUID != "uid" {
			t.Fatalf("account UID = %q", got.AccountUID)
		}
		return active, nil
	}))
	handler := middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	request := func(method, csrf string, cookie bool) int {
		r := httptest.NewRequest(method, "/api/v1/groups", nil)
		if cookie {
			r.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
		} else {
			r.Header.Set(AuthorizationHeader, BearerPrefix+token)
		}
		if csrf != "" {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Code
	}
	if got := request(http.MethodGet, "", false); got != http.StatusUnauthorized {
		t.Fatalf("session accepted as bearer: %d", got)
	}
	if got := request(http.MethodGet, "", true); got != http.StatusNoContent {
		t.Fatalf("active cookie status=%d", got)
	}
	if got := request(http.MethodPatch, "", true); got != http.StatusForbidden {
		t.Fatalf("missing CSRF status=%d", got)
	}
	if got := request(http.MethodPatch, claims.CSRFToken, true); got != http.StatusNoContent {
		t.Fatalf("valid CSRF status=%d", got)
	}
	active = false
	if got := request(http.MethodGet, "", true); got != http.StatusUnauthorized {
		t.Fatalf("revoked session status=%d", got)
	}
}
