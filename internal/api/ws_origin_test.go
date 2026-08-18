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

package api

import (
	"net/http/httptest"
	"testing"
)

func TestCheckWebSocketOrigin(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		origin   string
		expected bool
	}{
		{
			name:     "no origin header - allowed (non-browser client)",
			host:     "localhost:8080",
			origin:   "",
			expected: true,
		},
		{
			name:     "same origin - allowed",
			host:     "localhost:8080",
			origin:   "http://localhost:8080",
			expected: true,
		},
		{
			name:     "same origin HTTPS - allowed",
			host:     "api.example.com",
			origin:   "https://api.example.com",
			expected: true,
		},
		{
			name:     "cross origin - rejected",
			host:     "api.example.com",
			origin:   "https://evil.example.com",
			expected: false,
		},
		{
			name:     "cross origin different port - rejected",
			host:     "localhost:8080",
			origin:   "http://localhost:3000",
			expected: false,
		},
		{
			name:     "malformed origin - rejected",
			host:     "localhost:8080",
			origin:   "://not-a-valid-url",
			expected: false,
		},
		{
			name:     "origin with path - allowed if host matches",
			host:     "api.example.com",
			origin:   "https://api.example.com/some/path",
			expected: true,
		},
		{
			name:     "subdomain mismatch - rejected",
			host:     "api.example.com",
			origin:   "https://other.api.example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			req.Host = tt.host
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			result := checkWebSocketOrigin(req)
			if result != tt.expected {
				t.Errorf("checkWebSocketOrigin() = %v, want %v (host=%q, origin=%q)",
					result, tt.expected, tt.host, tt.origin)
			}
		})
	}
}
