package api

import "testing"

func TestSameOIDCOrigin(t *testing.T) {
	issuer := "https://keycloak.example/realms/krkn"
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"issuer authorization endpoint", "https://keycloak.example/realms/krkn/protocol/openid-connect/auth?state=one", true},
		{"different host", "https://other.example/authorize?state=one", false},
		{"subdomain", "https://keycloak.example.evil.test/authorize", false},
		{"different port", "https://keycloak.example:8443/authorize", false},
		{"scheme downgrade", "http://keycloak.example/authorize", false},
		{"userinfo", "https://user@keycloak.example/authorize", false},
		{"relative URL", "/authorize", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sameOIDCOrigin(issuer, tc.url); got != tc.want {
				t.Fatalf("sameOIDCOrigin(%q, %q) = %v, want %v", issuer, tc.url, got, tc.want)
			}
		})
	}
}
