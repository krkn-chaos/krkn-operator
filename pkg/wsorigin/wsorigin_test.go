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

package wsorigin

import (
	"net/http/httptest"
	"testing"
)

func TestIsSameOrigin(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		origin   string
		expected bool
	}{
		{name: "no origin allowed", host: "localhost:8080", origin: "", expected: true},
		{name: "exact same origin", host: "localhost:8080", origin: "http://localhost:8080", expected: true},
		{name: "https implicit port vs bare host", host: "api.example.com", origin: "https://api.example.com", expected: true},
		{name: "https implicit port vs host :443", host: "api.example.com:443", origin: "https://api.example.com", expected: true},
		{name: "https explicit :443 vs bare host", host: "api.example.com", origin: "https://api.example.com:443", expected: true},
		{name: "http implicit port vs host :80", host: "api.example.com:80", origin: "http://api.example.com", expected: true},
		{name: "case-insensitive hostname", host: "API.Example.COM:8080", origin: "http://api.example.com:8080", expected: true},
		{name: "ipv6 same origin", host: "[::1]:8080", origin: "http://[::1]:8080", expected: true},
		{name: "different host rejected", host: "api.example.com", origin: "https://evil.example.com", expected: false},
		{name: "different explicit port rejected", host: "localhost:8080", origin: "http://localhost:3000", expected: false},
		{name: "non-default origin port vs bare host rejected", host: "api.example.com", origin: "https://api.example.com:8443", expected: false},
		{name: "malformed origin rejected", host: "localhost:8080", origin: "://nope", expected: false},
		{name: "null origin rejected", host: "api.example.com", origin: "null", expected: false},
		{name: "empty-host origin rejected", host: "api.example.com", origin: "https://", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			req.Host = tt.host
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			if got := IsSameOrigin(req); got != tt.expected {
				t.Errorf("IsSameOrigin() = %v, want %v (host=%q, origin=%q)",
					got, tt.expected, tt.host, tt.origin)
			}
		})
	}
}
