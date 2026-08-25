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

// Package wsorigin provides same-origin validation for WebSocket upgrade
// requests. It is shared by the v1 and v2 API WebSocket upgraders so the
// origin policy stays consistent and is defined in a single place.
package wsorigin

import (
	"net/http"
	"net/url"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// IsSameOrigin validates the Origin header of a WebSocket upgrade request
// against the request Host. It applies a safe default policy that prevents
// cross-site WebSocket hijacking (CSWSH) attacks:
//
//   - Requests without an Origin header are allowed (non-browser clients such
//     as CLI tools and test harnesses do not send Origin).
//   - Same-origin requests are allowed. Comparison is scheme-aware and
//     normalized: hostnames are compared case-insensitively and ports are
//     compared using their scheme defaults (80 for http/ws, 443 for
//     https/wss). This ensures equivalent representations match — e.g. an
//     Origin of "https://api.example.com" (implicit :443) is treated as the
//     same origin as a Host of "api.example.com:443", and IPv6 brackets and
//     letter case do not cause false rejections.
//   - Everything else (different host, different explicit port, malformed or
//     "null" Origin) is rejected.
//
// Note: r.Host is used as the trusted server identity. Forwarded host headers
// (e.g. X-Forwarded-Host) are intentionally NOT consulted because they are
// attacker-controllable; deployments that terminate TLS in front of the server
// should ensure the proxy preserves the Host header.
func IsSameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Non-browser clients (CLI, testing tools) don't send Origin.
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

	if reqHost == origHost && portsMatch(reqPort, origPort, origScheme) {
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
