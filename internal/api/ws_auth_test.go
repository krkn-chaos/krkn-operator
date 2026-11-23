package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

type logSocketChecker struct{ active bool }

func (c *logSocketChecker) IsUserActive(context.Context, string) (bool, error) {
	return c.active, nil
}

func (c *logSocketChecker) IsExternalIdentityActive(context.Context, *auth.Claims) (bool, error) {
	return c.active, nil
}

func TestLogSocketSessionStatus(t *testing.T) {
	checker := &logSocketChecker{active: true}
	h := &Handler{userStatusChecker: checker, externalIdentityChecker: checker}
	claims := &auth.Claims{UserID: "alice", RegisteredClaims: jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}
	if !h.socketSessionActive(t.Context(), claims) {
		t.Fatal("active local session rejected")
	}
	checker.active = false
	if h.socketSessionActive(t.Context(), claims) {
		t.Fatal("disabled local account retained log socket")
	}
	claims.Provider = "corp"
	if h.socketSessionActive(t.Context(), claims) {
		t.Fatal("disabled OIDC account retained log socket")
	}
	checker.active = true
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Second))
	if h.socketSessionActive(t.Context(), claims) {
		t.Fatal("expired session retained log socket")
	}
}

func TestWebsocketPrincipalRejectsInvalidSubprotocol(t *testing.T) {
	h := &Handler{}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/run/logs", nil)
	r.Header.Set("Sec-WebSocket-Protocol", "other.token")
	if _, _, err := h.websocketPrincipal(r); err == nil {
		t.Fatal("non-access-token subprotocol accepted")
	}
}
