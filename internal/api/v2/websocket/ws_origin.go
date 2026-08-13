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

package websocket

import (
	"net/http"
	"net/url"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// checkWebSocketOriginV2 validates the Origin header for WebSocket upgrade requests.
// It applies a safe default policy:
//   - Requests without an Origin header are allowed (non-browser clients like CLI tools).
//   - Same-origin requests are allowed (Origin host matches request Host).
//   - Cross-origin requests are rejected.
//
// This prevents cross-site WebSocket hijacking (CSWSH) attacks where a malicious
// web page attempts to establish a WebSocket connection using the victim's credentials.
func checkWebSocketOriginV2(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Non-browser clients (CLI, testing tools) don't send Origin
		return true
	}

	// Parse the origin URL to extract the host
	originURL, err := url.Parse(origin)
	if err != nil {
		logger := log.Log.WithName("websocket-v2-origin")
		logger.Info("Rejected WebSocket connection: malformed Origin header",
			"origin", origin,
			"client_ip", r.RemoteAddr,
		)
		return false
	}

	// Allow same-origin requests
	host := r.Host
	if originURL.Host == host {
		return true
	}

	logger := log.Log.WithName("websocket-v2-origin")
	logger.Info("Rejected cross-origin WebSocket connection",
		"origin", origin,
		"host", host,
		"client_ip", r.RemoteAddr,
	)
	return false
}
