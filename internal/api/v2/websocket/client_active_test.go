package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

type socketStatusChecker struct {
	active bool
	calls  int
}

func (c *socketStatusChecker) IsUserActive(context.Context, string) (bool, error) {
	c.calls++
	return c.active, nil
}

func (c *socketStatusChecker) IsExternalIdentityActive(context.Context, *auth.Claims) (bool, error) {
	c.calls++
	return c.active, nil
}

func TestClientActiveChecksLocalAndExternalAccounts(t *testing.T) {
	for _, provider := range []string{"", "corp"} {
		t.Run(provider, func(t *testing.T) {
			checker := &socketStatusChecker{active: true}
			h := &Handler{userChecker: checker, externalChecker: checker}
			client := &Client{claims: &auth.Claims{UserID: "alice", Provider: provider,
				RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}}
			if !h.clientActive(client) || !h.clientActive(client) || checker.calls != 1 {
				t.Fatalf("active status was not cached: calls=%d", checker.calls)
			}
			client.statusValidUntil = time.Now().Add(-time.Second)
			checker.active = false
			if h.clientActive(client) || checker.calls != 2 {
				t.Fatalf("disabled account retained WebSocket access: calls=%d", checker.calls)
			}
			client.claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Second))
			if h.clientActive(client) || checker.calls != 2 {
				t.Fatal("expired session reached the account checker")
			}
		})
	}
}
