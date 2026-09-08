/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package wsorigin provides Origin validation for WebSocket upgrade requests.
// It is shared by the v1 and v2 API WebSocket upgraders so the origin policy
// stays consistent and is defined in a single place.
//
// Origin enforcement is OPT-IN. By default (no allow-list configured) all
// origins are accepted, which restores the pre-hardening behavior and avoids
// breaking deployments where the browser origin legitimately differs from the
// API Host (a console served on a separate origin, or one behind a proxy that
// rewrites the Host header). This is safe because WebSocket authentication uses
// a JWT carried in the Sec-WebSocket-Protocol subprotocol — not ambient cookies
// — so cross-site WebSocket hijacking (CSWSH) is not exploitable: a malicious
// cross-origin page cannot read the origin-scoped token and therefore cannot
// authenticate. Configuring an allow-list turns on same-origin + allow-list
// enforcement as optional defense-in-depth.
package wsorigin

import (
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// allowedOrigins holds the set of explicitly-permitted origins (in canonical
// "scheme://host:port" form). When empty, origin enforcement is disabled and
// all origins are accepted. It is stored atomically so it can be configured
// once at startup and read concurrently by upgrade handlers without a data
// race.
var allowedOrigins atomic.Pointer[map[string]struct{}]

// SetAllowedOrigins enables origin enforcement and configures the origins that
// IsAllowedOrigin will accept (in addition to same-origin requests). Each entry
// is a full origin such as "http://localhost:3000". This is intended to be
// called once at startup.
//
// Passing no (or only empty) origins leaves enforcement disabled, i.e. all
// origins are accepted. Entries that cannot be parsed as an origin with both a
// scheme and host are ignored and returned in the invalid slice so the caller
// can log them.
func SetAllowedOrigins(origins []string) (invalid []string) {
	set := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		u, err := url.Parse(o)
		if err != nil || u.Scheme == "" || u.Hostname() == "" {
			invalid = append(invalid, o)
			continue
		}
		set[canonicalOrigin(u)] = struct{}{}
	}
	allowedOrigins.Store(&set)
	return invalid
}

// canonicalOrigin renders a URL as a normalized "scheme://host:port" string:
// scheme and host are lowercased and the scheme's default port is filled in
// when absent, so equivalent representations compare equal.
func canonicalOrigin(u *url.URL) string {
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if port == "" {
		port = defaultPort(scheme)
	}
	return scheme + "://" + host + ":" + port
}

// IsAllowedOrigin reports whether a WebSocket upgrade request may proceed based
// on its Origin header. The policy is:
//
//   - Requests without an Origin header are allowed (non-browser clients such
//     as CLI tools and test harnesses do not send Origin).
//   - When no allow-list is configured (the default), all origins are allowed.
//     Enforcement is opt-in; see the package documentation for why this is safe
//     given the JWT-in-subprotocol authentication model.
//   - When an allow-list IS configured, only same-origin requests and origins
//     on the list are allowed; everything else (different host, different
//     explicit port, malformed or "null" Origin) is rejected. Same-origin
//     comparison is scheme-aware and normalized: hostnames are compared
//     case-insensitively and ports are compared using their scheme defaults
//     (80 for http/ws, 443 for https/wss), so equivalent representations match
//     and IPv6 brackets or letter case do not cause false rejections.
//
// Note: r.Host is used as the trusted server identity. Forwarded host headers
// (e.g. X-Forwarded-Host) are intentionally NOT consulted because they are
// attacker-controllable; deployments that terminate TLS in front of the server
// should ensure the proxy preserves the Host header.
func IsAllowedOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Non-browser clients (CLI, testing tools) don't send Origin.
		return true
	}

	set := allowedOrigins.Load()
	if set == nil || len(*set) == 0 {
		// Enforcement is opt-in: with no allow-list configured, accept all
		// origins (auth is via a JWT subprotocol, not ambient cookies).
		return true
	}

	originURL, err := url.Parse(origin)
	if err != nil || originURL.Host == "" {
		log.Log.WithName("websocket-origin").Info(
			"Rejected WebSocket connection: malformed Origin header",
			"origin", origin,
			"client_ip", r.RemoteAddr,
		)
		return false
	}

	// Request scheme is unknown from the Host header alone (TLS may be
	// terminated by an upstream proxy), so the request port defaults are
	// resolved against the Origin's scheme below.
	reqHost, reqPort := splitHostPort(r.Host)
	origHost := strings.ToLower(originURL.Hostname())
	origScheme := strings.ToLower(originURL.Scheme)
	origPort := originURL.Port()
	if origPort == "" {
		origPort = defaultPort(origScheme)
	}

	// Same-origin is always allowed when enforcing.
	if reqHost == origHost && portsMatch(reqPort, origPort, origScheme) {
		return true
	}

	// Otherwise the origin must be on the configured allow-list.
	if _, ok := (*set)[canonicalOrigin(originURL)]; ok {
		return true
	}

	log.Log.WithName("websocket-origin").Info(
		"Rejected cross-origin WebSocket connection",
		"origin", origin,
		"host", r.Host,
		"client_ip", r.RemoteAddr,
	)
	return false
}

// splitHostPort splits an authority ("host" or "host:port") into a
// lowercased hostname and an (optional) port, transparently handling IPv6
// bracket notation. The port is empty when the authority carries none.
func splitHostPort(authority string) (host, port string) {
	// Parsing as the authority component of a URL correctly strips IPv6
	// brackets and separates the port without relying on error-prone manual
	// string splitting.
	u, err := url.Parse("//" + strings.TrimSpace(authority))
	if err != nil {
		return strings.ToLower(strings.TrimSpace(authority)), ""
	}
	return strings.ToLower(u.Hostname()), u.Port()
}

// portsMatch reports whether the request port and origin port refer to the
// same effective endpoint. When the request Host omits a port (common behind
// TLS terminators that listen on the standard port), it is considered a match
// as long as the Origin targets the default port for its scheme.
func portsMatch(reqPort, origPort, origScheme string) bool {
	if reqPort == "" {
		return origPort == defaultPort(origScheme)
	}
	return reqPort == origPort
}

// defaultPort returns the well-known port for the given URL scheme, or an
// empty string for unrecognized schemes.
func defaultPort(scheme string) string {
	switch scheme {
	case "https", "wss":
		return "443"
	case "http", "ws":
		return "80"
	default:
		return ""
	}
}
