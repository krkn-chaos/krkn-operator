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

// TestIsAllowedOrigin_NoEnforcement verifies the default (opt-in) behavior:
// with no allow-list configured, every origin is accepted, including ones that
// differ from the Host. This is the non-breaking default.
func TestIsAllowedOrigin_NoEnforcement(t *testing.T) {
	// Ensure no allow-list is configured for this test.
	SetAllowedOrigins(nil)

	tests := []struct {
		name   string
		host   string
		origin string
	}{
		{name: "no origin", host: "localhost:8080", origin: ""},
		{name: "same origin", host: "localhost:8080", origin: "http://localhost:8080"},
		{name: "cross origin still allowed", host: "localhost:8080", origin: "http://localhost:3000"},
		{name: "foreign origin allowed", host: "api.example.com", origin: "https://evil.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			req.Host = tt.host
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if !IsAllowedOrigin(req) {
				t.Errorf("IsAllowedOrigin() = false, want true with enforcement off (host=%q, origin=%q)",
					tt.host, tt.origin)
			}
		})
	}
}

// TestIsAllowedOrigin_Enforced verifies that once an allow-list is configured,
// same-origin and allow-listed origins are accepted and everything else is
// rejected, with scheme-aware normalization.
func TestIsAllowedOrigin_Enforced(t *testing.T) {
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
		{name: "no origin allowed", host: "localhost:8080", origin: "", expected: true},
		{name: "same origin allowed", host: "localhost:8080", origin: "http://localhost:8080", expected: true},
		{name: "same origin implicit https port", host: "api.example.com", origin: "https://api.example.com", expected: true},
		{name: "same origin ipv6", host: "[::1]:8080", origin: "http://[::1]:8080", expected: true},
		{name: "same origin case-insensitive", host: "API.Example.COM:8080", origin: "http://api.example.com:8080", expected: true},
		{name: "allow-listed origin", host: "localhost:8080", origin: "http://localhost:3000", expected: true},
		{name: "allow-listed implicit default port", host: "localhost:8080", origin: "https://console.example.com", expected: true},
		{name: "allow-listed case-insensitive", host: "localhost:8080", origin: "http://LOCALHOST:3000", expected: true},
		{name: "unlisted cross-origin rejected", host: "localhost:8080", origin: "http://evil.example.com", expected: false},
		{name: "allow-listed host wrong port rejected", host: "localhost:8080", origin: "http://localhost:3001", expected: false},
		{name: "malformed origin rejected", host: "localhost:8080", origin: "://nope", expected: false},
		{name: "null origin rejected", host: "api.example.com", origin: "null", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			req.Host = tt.host
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if got := IsAllowedOrigin(req); got != tt.expected {
				t.Errorf("IsAllowedOrigin() = %v, want %v (host=%q, origin=%q)",
					got, tt.expected, tt.host, tt.origin)
			}
		})
	}
}

// TestSetAllowedOrigins_EmptyDisablesEnforcement verifies that clearing the
// allow-list returns to the accept-all default.
func TestSetAllowedOrigins_EmptyDisablesEnforcement(t *testing.T) {
	SetAllowedOrigins([]string{"http://localhost:3000"})

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Host = "localhost:8080"
	req.Header.Set("Origin", "http://evil.example.com")
	if IsAllowedOrigin(req) {
		t.Fatal("expected unlisted origin to be rejected while enforcing")
	}

	SetAllowedOrigins(nil) // disable enforcement
	if !IsAllowedOrigin(req) {
		t.Error("expected all origins to be allowed after clearing the allow-list")
	}
}
