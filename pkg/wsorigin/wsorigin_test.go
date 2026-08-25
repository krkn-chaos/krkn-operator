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

func TestSetAllowedOrigins(t *testing.T) {
	// Reset the global allow-list after the test so other tests keep the
	// strict same-origin-only default.
	t.Cleanup(func() { SetAllowedOrigins(nil) })

	invalid := SetAllowedOrigins([]string{
		"http://localhost:3000",
		" https://console.example.com ", // trimmed
		"",                              // skipped
		"not-a-valid-origin",            // invalid: no scheme/host
	})

	if len(invalid) != 1 || invalid[0] != "not-a-valid-origin" {
		t.Fatalf("expected exactly the malformed entry to be reported invalid, got %v", invalid)
	}

	tests := []struct {
		name     string
		host     string
		origin   string
		expected bool
	}{
		{name: "same-origin still allowed", host: "localhost:8080", origin: "http://localhost:8080", expected: true},
		{name: "configured cross-origin allowed", host: "localhost:8080", origin: "http://localhost:3000", expected: true},
		{name: "configured origin implicit default port", host: "localhost:8080", origin: "https://console.example.com", expected: true},
		{name: "configured origin case-insensitive", host: "localhost:8080", origin: "http://LOCALHOST:3000", expected: true},
		{name: "unconfigured cross-origin still rejected", host: "localhost:8080", origin: "http://evil.example.com", expected: false},
		{name: "configured host wrong port rejected", host: "localhost:8080", origin: "http://localhost:3001", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			req.Host = tt.host
			req.Header.Set("Origin", tt.origin)

			if got := IsSameOrigin(req); got != tt.expected {
				t.Errorf("IsSameOrigin() = %v, want %v (host=%q, origin=%q)",
					got, tt.expected, tt.host, tt.origin)
			}
		})
	}
}

func TestSetAllowedOrigins_EmptyRestoresStrict(t *testing.T) {
	SetAllowedOrigins([]string{"http://localhost:3000"})
	SetAllowedOrigins(nil) // restore strict policy

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Host = "localhost:8080"
	req.Header.Set("Origin", "http://localhost:3000")

	if IsSameOrigin(req) {
		t.Error("expected cross-origin request to be rejected after allow-list cleared")
	}
}
