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

package elasticsearch

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestValidateDestinationIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"loopback v4", "127.0.0.1", true},
		{"loopback v6", "::1", true},
		{"private 10", "10.0.0.5", true},
		{"private 172", "172.16.0.1", true},
		{"private 192", "192.168.1.1", true},
		{"metadata link-local", "169.254.169.254", true},
		{"link-local v6", "fe80::1", true},
		{"ula v6", "fc00::1", true},
		{"cgnat", "100.64.0.1", true},
		{"unspecified v4", "0.0.0.0", true},
		{"unspecified v6", "::", true},
		{"multicast", "224.0.0.1", true},
		{"mapped internal", "::ffff:127.0.0.1", true},
		{"public v4", "93.184.216.34", false},
		{"public v6", "2606:2800:220:1:248:1893:25c8:1946", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("bad test IP %q", tt.ip)
			}
			err := validateDestinationIP(ip)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected rejection for %s, got nil", tt.ip)
				}
				if !errors.Is(err, ErrDestinationNotAllowed) {
					t.Errorf("error should wrap ErrDestinationNotAllowed, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Errorf("expected %s to be allowed, got: %v", tt.ip, err)
			}
		})
	}
}

func TestValidateDestinationURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"https allowed port", "https://es.example.com:9200", false},
		{"https 443", "https://es.example.com:443", false},
		{"http allowed port", "http://es.example.com:9200", false},
		{"disallowed port", "https://es.example.com:22", true},
		{"missing port", "https://es.example.com", true},
		{"file scheme", "file://es.example.com:9200", true},
		{"gopher scheme", "gopher://es.example.com:9200", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.raw)
			if err != nil {
				t.Fatalf("bad test URL %q: %v", tt.raw, err)
			}
			err = validateDestinationURL(u)
			if tt.wantErr && err == nil {
				t.Fatalf("expected rejection for %s, got nil", tt.raw)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected %s to be allowed, got: %v", tt.raw, err)
			}
		})
	}
}

// TestQueryTelemetryRejectsInlineDestinationBeforeOutbound is the required
// regression test: a restricted (inline) connection targeting a disallowed
// address must be rejected before any outbound request is attempted. The
// injected Doer fails the test if it is ever invoked.
func TestQueryTelemetryRejectsInlineDestinationBeforeOutbound(t *testing.T) {
	tests := []struct {
		name string
		host string
		port int
	}{
		{"loopback", "http://127.0.0.1", 9200},
		{"private", "https://10.0.0.1", 9200},
		{"metadata", "http://169.254.169.254", 9200},
		{"disallowed port", "https://93.184.216.34", 22},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := doerFunc(func(r *http.Request) (*http.Response, error) {
				t.Fatalf("outbound request must not be made; got %s", r.URL)
				return nil, nil
			})
			c := NewClient(WithHTTPClient(stub))
			conn := ConnectionParams{
				Host:                tt.host,
				Port:                tt.port,
				Index:               "telemetry",
				RestrictDestination: true,
			}
			_, _, err := c.QueryTelemetry(context.Background(), conn, 10, "", "")
			if err == nil {
				t.Fatal("expected a destination error, got nil")
			}
			if !errors.Is(err, ErrDestinationNotAllowed) {
				t.Errorf("error should wrap ErrDestinationNotAllowed, got: %v", err)
			}
		})
	}
}

// TestQueryTelemetryAllowsUnrestrictedLoopback proves the guard does not apply to
// saved (admin) configs: an unrestricted connection may reach a loopback test
// server, which is how in-cluster private clusters remain reachable.
func TestQueryTelemetryAllowsUnrestrictedLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hits":{"hits":[{"_source":{"run_uuid":"abc"}}]}}`))
	}))
	defer srv.Close()

	conn := ConnectionParams{Host: srv.URL, Index: "telemetry"} // RestrictDestination false
	if _, _, err := NewClient().QueryTelemetry(context.Background(), conn, 10, "", ""); err != nil {
		t.Fatalf("unrestricted loopback query should succeed, got: %v", err)
	}
}

func TestGuardedCheckRedirect(t *testing.T) {
	// A redirect to a disallowed scheme is rejected. (A loopback IP on an allowed
	// port passes this scheme/port check but is still blocked at dial time by
	// guardedDialContext, which validates the resolved address.)
	badScheme, _ := http.NewRequest(http.MethodGet, "file:///etc/passwd", nil)
	if err := guardedCheckRedirect(badScheme, nil); err == nil {
		t.Error("redirect to a disallowed scheme should be rejected")
	}

	// Disallowed port on redirect is rejected.
	badPort, _ := http.NewRequest(http.MethodGet, "https://es.example.com:22/x", nil)
	if err := guardedCheckRedirect(badPort, nil); err == nil {
		t.Error("redirect to a disallowed port should be rejected")
	}

	// A permitted redirect target passes the per-hop scheme/port check.
	ok, _ := http.NewRequest(http.MethodGet, "https://es.example.com:9200/x", nil)
	if err := guardedCheckRedirect(ok, nil); err != nil {
		t.Errorf("permitted redirect target should pass, got: %v", err)
	}

	// Too many hops is rejected regardless of target.
	via := make([]*http.Request, maxInlineRedirects)
	if err := guardedCheckRedirect(ok, via); err == nil {
		t.Error("exceeding the redirect cap should be rejected")
	}
}
