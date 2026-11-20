package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

// websocketPrincipal uses middleware claims for registered routes and retains
// legacy JWT subprotocol authentication for direct handler consumers.
func (h *Handler) websocketPrincipal(r *http.Request) (*auth.Claims, string, error) {
	protocol := r.Header.Get("Sec-WebSocket-Protocol")
	if claims := auth.GetClaimsFromContext(r.Context()); claims != nil {
		return claims, protocol, nil
	}
	parts := strings.SplitN(protocol, ".", 2)
	if len(parts) != 2 || parts[0] != "access_token" || parts[1] == "" {
		return nil, "", fmt.Errorf("expected access_token.<jwt> WebSocket subprotocol")
	}
	generator, err := h.getTokenGenerator()
	if err != nil {
		return nil, "", err
	}
	claims, err := generator.ValidateToken(parts[1])
	if err != nil || claims.TokenUse != "" {
		return nil, "", fmt.Errorf("invalid or expired WebSocket token")
	}
	return claims, protocol, nil
}

func websocketResponseHeader(protocol string) http.Header {
	if protocol == "" {
		return nil
	}
	return http.Header{"Sec-WebSocket-Protocol": []string{protocol}}
}

// watchExternalSocket closes an OIDC socket after account deletion, block, or
// session expiry. This also covers passive log streams without incoming messages.
func (h *Handler) watchExternalSocket(conn *websocket.Conn, claims *auth.Claims) func() {
	if claims == nil || claims.Provider == "" {
		return func() {}
	}
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
					_ = conn.Close()
					return
				}
				if h.externalIdentityChecker == nil {
					_ = conn.Close()
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				active, err := h.externalIdentityChecker.IsExternalIdentityActive(ctx, claims)
				cancel()
				if err != nil || !active {
					_ = conn.Close()
					return
				}
			}
		}
	}()
	return func() { close(stop) }
}
