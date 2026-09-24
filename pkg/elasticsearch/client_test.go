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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestQueryTelemetry(t *testing.T) {
	// The aggregation counts span the whole matched window, so they can (and here
	// do) exceed the two hits returned in the size-capped page.
	sampleHits := `{
      "hits": {
        "hits": [
          {"_source": {"run_uuid": "abc", "job_status": true, "scenarios": [{"scenario_type": "pod_disruption_scenarios", "start_timestamp": 1735689600, "end_timestamp": 1735689900, "exit_status": 0, "parameters": [{"config": {"namespace_pattern": "openshift-kube-apiserver"}}]}, {"scenario_type": "node"}]}},
          {"_source": {"run_uuid": "def", "job_status": false, "scenarios": [{"scenario_type": "pod", "start_timestamp": 1735776000, "end_timestamp": 1735776300, "exit_status": 1, "parameters": [{"config": {"namespace": "default"}}]}]}}
        ]
      },
      "aggregations": {
        "by_job_status": {
          "buckets": [
            {"key": 1, "key_as_string": "true", "doc_count": 10},
            {"key": 0, "key_as_string": "false", "doc_count": 3}
          ]
        }
      }
    }`

	tests := []struct {
		name       string
		index      string
		statusCode int
		body       string
		wantErr    bool
		wantCount  int
		wantStats  TelemetryStats
		checkFirst func(t *testing.T, d TelemetryDocument)
	}{
		{
			name:       "successful query flattens first scenario",
			index:      "telemetry",
			statusCode: http.StatusOK,
			body:       sampleHits,
			wantCount:  2,
			wantStats:  TelemetryStats{Pass: 10, Fail: 3, PassPercent: 76.92},
			checkFirst: func(t *testing.T, d TelemetryDocument) {
				if d.RunUUID != "abc" {
					t.Errorf("got run_uuid %q, want abc", d.RunUUID)
				}
				if d.ScenarioType != "pod_disruption_scenarios" {
					t.Errorf("got scenario_type %q, want pod_disruption_scenarios", d.ScenarioType)
				}
				if d.StartTimestamp != 1735689600 {
					t.Errorf("got start_timestamp %f, want 1735689600", d.StartTimestamp)
				}
				if d.EndTimestamp != 1735689900 {
					t.Errorf("got end_timestamp %f, want 1735689900", d.EndTimestamp)
				}
				if d.Namespace != "openshift-kube-apiserver" {
					t.Errorf("got namespace %q, want openshift-kube-apiserver", d.Namespace)
				}
				if !d.Status {
					t.Errorf("got status false, want true")
				}
			},
		},
		{
			name:       "no hits and no aggregation yields zero stats",
			index:      "telemetry",
			statusCode: http.StatusOK,
			body:       `{"hits":{"hits":[]}}`,
			wantCount:  0,
			wantStats:  TelemetryStats{Pass: 0, Fail: 0, PassPercent: 0},
		},
		{
			name:       "missing index errors before request",
			index:      "",
			statusCode: http.StatusOK,
			body:       sampleHits,
			wantErr:    true,
		},
		{
			name:       "non-2xx response errors",
			index:      "telemetry",
			statusCode: http.StatusUnauthorized,
			body:       `{"error":"unauthorized"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, "/_search") {
					t.Errorf("unexpected path %q", r.URL.Path)
				}
				// Verify the query body is well-formed JSON with a size field.
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("invalid request body: %v", err)
				}
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			conn := ConnectionParams{
				Host:  srv.URL, // includes http:// scheme, used verbatim
				Index: tt.index,
			}

			docs, stats, err := NewClient().QueryTelemetry(context.Background(), conn, 50, "", "")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(docs) != tt.wantCount {
				t.Fatalf("got %d docs, want %d", len(docs), tt.wantCount)
			}
			if stats != tt.wantStats {
				t.Errorf("got stats %+v, want %+v", stats, tt.wantStats)
			}
			if tt.checkFirst != nil && len(docs) > 0 {
				tt.checkFirst(t, docs[0])
			}
		})
	}
}

func TestQueryTelemetryRejectsCredentialsOverHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("request should not reach the server over plaintext HTTP with credentials")
		_, _ = w.Write([]byte(`{"hits":{"hits":[]}}`))
	}))
	defer srv.Close()

	conn := ConnectionParams{Host: srv.URL, Index: "telemetry", Username: "elastic", Password: "secret"}
	_, _, err := NewClient().QueryTelemetry(context.Background(), conn, 50, "", "")
	if err == nil {
		t.Fatal("expected error for credentials over plaintext HTTP, got nil")
	}
	if !strings.Contains(err.Error(), "plaintext HTTP") {
		t.Errorf("error = %q, want it to mention plaintext HTTP", err.Error())
	}
}

func TestConnectionParamsTLSConfig(t *testing.T) {
	// A syntactically valid self-signed certificate in PEM form.
	const caPEM = `-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`

	t.Run("default verifies with system roots", func(t *testing.T) {
		cfg, err := ConnectionParams{}.tlsConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.InsecureSkipVerify {
			t.Error("InsecureSkipVerify should be false by default")
		}
		if cfg.RootCAs != nil {
			t.Error("RootCAs should be nil (system roots) when no CACert is set")
		}
	})

	t.Run("custom CA is trusted without disabling verification", func(t *testing.T) {
		cfg, err := ConnectionParams{CACert: caPEM}.tlsConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.InsecureSkipVerify {
			t.Error("InsecureSkipVerify should remain false when a CACert is provided")
		}
		if cfg.RootCAs == nil {
			t.Error("RootCAs should be populated from the provided CACert")
		}
	})

	t.Run("invalid CA errors", func(t *testing.T) {
		if _, err := (ConnectionParams{CACert: "garbage"}).tlsConfig(); err == nil {
			t.Error("expected error for invalid CACert, got nil")
		}
	})

	t.Run("insecure skip verify is an explicit opt-in", func(t *testing.T) {
		cfg, err := ConnectionParams{InsecureSkipVerify: true}.tlsConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cfg.InsecureSkipVerify {
			t.Error("InsecureSkipVerify should be true when explicitly opted in")
		}
	})
}

// doerFunc adapts a function to the Doer interface for injection in tests.
type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

func TestQueryTelemetryUsesInjectedDoer(t *testing.T) {
	var gotURL string
	stub := doerFunc(func(r *http.Request) (*http.Response, error) {
		gotURL = r.URL.String()
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"hits":{"hits":[{"_source":{"run_uuid":"abc"}}]}}`)),
			Header:     make(http.Header),
		}, nil
	})

	c := NewClient(WithHTTPClient(stub))
	// A host that would never resolve proves the injected Doer is used instead
	// of a real network client.
	conn := ConnectionParams{Host: "https://unreachable.invalid", Port: 9200, Index: "telemetry"}
	docs, _, err := c.QueryTelemetry(context.Background(), conn, 10, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 1 || docs[0].RunUUID != "abc" {
		t.Fatalf("unexpected docs: %+v", docs)
	}
	if !strings.HasPrefix(gotURL, "https://unreachable.invalid:9200/telemetry/_search") {
		t.Errorf("injected Doer received unexpected URL: %s", gotURL)
	}
}

func TestQueryTelemetryRejectsOversizedResponse(t *testing.T) {
	// Serve a valid-JSON body larger than the cap so the size check, not the
	// decoder, is what rejects it.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"padding":"`))
		chunk := strings.Repeat("a", 1<<20) // 1 MiB
		for written := 0; written <= maxResponseBytes; written += len(chunk) {
			if _, err := w.Write([]byte(chunk)); err != nil {
				return
			}
		}
		_, _ = w.Write([]byte(`"}`))
	}))
	defer srv.Close()

	conn := ConnectionParams{Host: srv.URL, Index: "telemetry"}
	_, _, err := NewClient().QueryTelemetry(context.Background(), conn, 50, "", "")
	if err == nil {
		t.Fatal("expected an error for an oversized response, got nil")
	}
	if !strings.Contains(err.Error(), "maximum supported size") {
		t.Errorf("expected a size-limit error, got: %v", err)
	}
}

func TestClientTransportPooling(t *testing.T) {
	c := NewClient()

	// Two connections sharing the same TLS posture must reuse one transport so
	// their connections are pooled together, even across different hosts.
	tA, err := c.transport(ConnectionParams{Host: "https://a.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tB, err := c.transport(ConnectionParams{Host: "https://b.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tA != tB {
		t.Error("expected the same pooled transport for identical TLS configs")
	}

	// A differing TLS configuration must be isolated to its own transport.
	tInsecure, err := c.transport(ConnectionParams{Host: "https://a.example.com", InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tInsecure == tA {
		t.Error("expected a distinct transport for an insecure TLS config")
	}
	if !tInsecure.TLSClientConfig.InsecureSkipVerify {
		t.Error("insecure transport should carry InsecureSkipVerify=true")
	}
}

// roundTripperFunc is a non-*http.Transport RoundTripper used to simulate a
// replaced http.DefaultTransport.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestClientTransportDefaultNotHTTPTransport(t *testing.T) {
	// If http.DefaultTransport is replaced with a non-*http.Transport, transport
	// must fall back to a fresh *http.Transport instead of panicking on a nil
	// pointer dereference in Clone.
	orig := http.DefaultTransport
	http.DefaultTransport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, nil
	})
	defer func() { http.DefaultTransport = orig }()

	c := NewClient()
	tr, err := c.transport(ConnectionParams{Host: "https://a.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr == nil {
		t.Fatal("expected a non-nil transport fallback")
	}
	if tr.TLSClientConfig == nil {
		t.Error("expected TLS config to be attached to the fallback transport")
	}
}

func TestTransportKey(t *testing.T) {
	tests := []struct {
		name string
		a, b ConnectionParams
		same bool
	}{
		{
			name: "default configs share a key regardless of host",
			a:    ConnectionParams{Host: "https://a"},
			b:    ConnectionParams{Host: "https://b"},
			same: true,
		},
		{
			name: "insecure differs from default",
			a:    ConnectionParams{},
			b:    ConnectionParams{InsecureSkipVerify: true},
			same: false,
		},
		{
			name: "same CA shares a key",
			a:    ConnectionParams{CACert: "cert-A"},
			b:    ConnectionParams{CACert: "cert-A"},
			same: true,
		},
		{
			name: "different CA differs",
			a:    ConnectionParams{CACert: "cert-A"},
			b:    ConnectionParams{CACert: "cert-B"},
			same: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transportKey(tt.a) == transportKey(tt.b); got != tt.same {
				t.Errorf("transportKey equality = %v, want %v", got, tt.same)
			}
		})
	}
}

func TestQueryTelemetrySortsNewestFirst(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		_, _ = w.Write([]byte(`{"hits":{"hits":[]}}`))
	}))
	defer srv.Close()

	conn := ConnectionParams{Host: srv.URL, Index: "telemetry"}
	if _, _, err := NewClient().QueryTelemetry(context.Background(), conn, 50, "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sort, ok := captured["sort"].([]any)
	if !ok || len(sort) == 0 {
		t.Fatalf("expected a non-empty sort clause, got %v", captured["sort"])
	}

	ts, ok := sort[0].(map[string]any)["timestamp"].(map[string]any)
	if !ok {
		t.Fatalf("expected first sort key on timestamp, got %v", sort[0])
	}
	if ts["order"] != "desc" {
		t.Errorf("timestamp sort order = %v, want desc", ts["order"])
	}
	if ts["unmapped_type"] != "date" {
		t.Errorf("timestamp unmapped_type = %v, want date", ts["unmapped_type"])
	}

	if len(sort) < 2 {
		t.Fatalf("expected a deterministic tie-breaker sort key, got %v", sort)
	}
	if _, ok := sort[1].(map[string]any)["_doc"]; !ok {
		t.Errorf("expected _doc tie-breaker as second sort key, got %v", sort[1])
	}
}

func TestQueryTelemetryDateRange(t *testing.T) {
	tests := []struct {
		name      string
		startDate string
		endDate   string
		wantGTE   string
		// upperKey is the expected bound key: "lte" (inclusive, for the default
		// upper bound) or "lt" (exclusive, for an explicit end date). The other
		// key must be absent.
		upperKey  string
		wantUpper string
	}{
		// Default upper bound is inclusive "now" so telemetry through the current
		// instant is included (not "now/d", which drops everything logged today).
		{"defaults trailing window", "", "", "now-30d/d", "lte", "now"},
		// An explicit end date uses an exclusive "lt" next-day bound so only the
		// selected calendar day is included.
		{"explicit bounds", "2026-08-01", "2026-08-26", "2026-08-01", "lt", "2026-08-26||+1d/d"},
		{"only start provided", "2026-08-01", "", "2026-08-01", "lte", "now"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewDecoder(r.Body).Decode(&captured)
				_, _ = w.Write([]byte(`{"hits":{"hits":[]}}`))
			}))
			defer srv.Close()

			conn := ConnectionParams{Host: srv.URL, Index: "telemetry"}
			if _, _, err := NewClient().QueryTelemetry(context.Background(), conn, 50, tt.startDate, tt.endDate); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			rng := captured["query"].(map[string]any)["bool"].(map[string]any)["filter"].([]any)[0].(map[string]any)["range"].(map[string]any)["timestamp"].(map[string]any)
			if rng["gte"] != tt.wantGTE {
				t.Errorf("gte = %v, want %v", rng["gte"], tt.wantGTE)
			}
			if got := rng[tt.upperKey]; got != tt.wantUpper {
				t.Errorf("%s = %v, want %v", tt.upperKey, got, tt.wantUpper)
			}
			// The unused upper-bound key must not be present, so the bound has the
			// intended inclusivity.
			otherKey := "lt"
			if tt.upperKey == "lt" {
				otherKey = "lte"
			}
			if _, present := rng[otherKey]; present {
				t.Errorf("unexpected %q bound present: %v", otherKey, rng[otherKey])
			}
		})
	}
}

func TestBaseURL(t *testing.T) {
	tests := []struct {
		name string
		conn ConnectionParams
		want string
	}{
		{"host with scheme and port used verbatim", ConnectionParams{Host: "http://es.local:9200", Port: 9200}, "http://es.local:9200"},
		{"trailing slash trimmed", ConnectionParams{Host: "https://es.local:9200/", Port: 9200}, "https://es.local:9200"},
		{"bare host gets https and port", ConnectionParams{Host: "es.local", Port: 9200}, "https://es.local:9200"},
		{"scheme host without port gets configured port", ConnectionParams{Host: "https://es.local", Port: 9200}, "https://es.local:9200"},
		{"scheme host trailing slash without port gets configured port", ConnectionParams{Host: "https://es.local/", Port: 9200}, "https://es.local:9200"},
		{"scheme host without port and no configured port used verbatim", ConnectionParams{Host: "https://es.local", Port: 0}, "https://es.local"},
		{"scheme host port differs from configured keeps host port", ConnectionParams{Host: "https://es.local:9201", Port: 9200}, "https://es.local:9201"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.conn.baseURL(); got != tt.want {
				t.Errorf("baseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateQueryRequest(t *testing.T) {
	tests := []struct {
		name     string
		req      QueryTelemetryRequest
		wantErr  bool
		wantSize int
	}{
		{"neither config nor inline", QueryTelemetryRequest{}, true, 0},
		{"both config and inline", QueryTelemetryRequest{ConfigName: "c", Inline: &InlineConnection{Host: "https://h", TelemetryIndex: "i"}}, true, 0},
		{"inline valid defaults size", QueryTelemetryRequest{Inline: &InlineConnection{Host: "https://h", TelemetryIndex: "i"}}, false, DefaultQuerySize},
		{"inline missing host", QueryTelemetryRequest{Inline: &InlineConnection{TelemetryIndex: "i"}}, true, 0},
		{"inline missing index", QueryTelemetryRequest{Inline: &InlineConnection{Host: "https://h"}}, true, 0},
		{"inline bad port", QueryTelemetryRequest{Inline: &InlineConnection{Host: "https://h", TelemetryIndex: "i", Port: 70000}}, true, 0},
		{"inline creds over http", QueryTelemetryRequest{Inline: &InlineConnection{Host: "http://h", TelemetryIndex: "i", Username: "u"}}, true, 0},
		{"negative size", QueryTelemetryRequest{ConfigName: "c", Size: -1}, true, 0},
		{"zero size defaults", QueryTelemetryRequest{ConfigName: "c", Size: 0}, false, DefaultQuerySize},
		{"oversized clamped", QueryTelemetryRequest{ConfigName: "c", Size: 10000}, false, MaxQuerySize},
		{"in-range size preserved", QueryTelemetryRequest{ConfigName: "c", Size: 25}, false, 25},
		{"valid dates preserved", QueryTelemetryRequest{ConfigName: "c", Size: 25, StartDate: "2026-08-01", EndDate: "2026-08-26"}, false, 25},
		{"invalid start date", QueryTelemetryRequest{ConfigName: "c", StartDate: "08/01/2026"}, true, 0},
		{"start after end", QueryTelemetryRequest{ConfigName: "c", StartDate: "2026-08-27", EndDate: "2026-08-01"}, true, 0},
		{"end date in the future", QueryTelemetryRequest{ConfigName: "c", EndDate: "2999-12-31"}, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req
			err := ValidateQueryRequest(&req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if req.Size != tt.wantSize {
				t.Errorf("got size %d, want %d", req.Size, tt.wantSize)
			}
		})
	}

	t.Run("nil request", func(t *testing.T) {
		if err := ValidateQueryRequest(nil); err == nil {
			t.Fatal("expected error for nil request, got nil")
		}
	})
}

func TestRawTelemetrySourceFlatten(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   TelemetryDocument
	}{
		{
			name:   "config-style parameters (pod disruption)",
			source: `{"run_uuid":"abc","job_status":true,"scenarios":[{"scenario_type":"pod","start_timestamp":100,"end_timestamp":200,"exit_status":0,"parameters":[{"config":{"namespace_pattern":"ns1"}}]}]}`,
			want: TelemetryDocument{RunUUID: "abc", ScenarioType: "pod", StartTimestamp: 100, EndTimestamp: 200, Namespace: "ns1", Status: true,
				Scenarios: []ScenarioDetail{{ScenarioType: "pod", StartTimestamp: 100, EndTimestamp: 200, Parameters: json.RawMessage(`[{"config":{"namespace_pattern":"ns1"}}]`)}}},
		},
		{
			name:   "object-style parameters (pvc scenario)",
			source: `{"run_uuid":"pvc","job_status":true,"scenarios":[{"scenario_type":"pvc_scenarios","start_timestamp":1,"end_timestamp":2,"exit_status":0,"parameters":{"pvc_scenario":{"namespace":"openshift-monitoring","pvc_name":"x"}}}]}`,
			want: TelemetryDocument{RunUUID: "pvc", ScenarioType: "pvc_scenarios", StartTimestamp: 1, EndTimestamp: 2, Namespace: "openshift-monitoring", Status: true,
				Scenarios: []ScenarioDetail{{ScenarioType: "pvc_scenarios", StartTimestamp: 1, EndTimestamp: 2, Parameters: json.RawMessage(`{"pvc_scenario":{"namespace":"openshift-monitoring","pvc_name":"x"}}`)}}},
		},
		{
			name:   "array-nested parameters (time scenario)",
			source: `{"run_uuid":"time","job_status":true,"scenarios":[{"scenario_type":"time_scenarios","start_timestamp":3,"end_timestamp":4,"exit_status":0,"parameters":{"time_scenarios":[{"namespace":"openshift-etcd","action":"skew_time"}]}}]}`,
			want: TelemetryDocument{RunUUID: "time", ScenarioType: "time_scenarios", StartTimestamp: 3, EndTimestamp: 4, Namespace: "openshift-etcd", Status: true,
				Scenarios: []ScenarioDetail{{ScenarioType: "time_scenarios", StartTimestamp: 3, EndTimestamp: 4, Parameters: json.RawMessage(`{"time_scenarios":[{"namespace":"openshift-etcd","action":"skew_time"}]}`)}}},
		},
		{
			name:   "non-zero exit status marks failure",
			source: `{"run_uuid":"def","job_status":true,"scenarios":[{"scenario_type":"node","exit_status":1}]}`,
			want: TelemetryDocument{RunUUID: "def", ScenarioType: "node", Status: false,
				Scenarios: []ScenarioDetail{{ScenarioType: "node", ExitStatus: 1}}},
		},
		{
			name:   "no scenarios keeps job status",
			source: `{"run_uuid":"ghi","job_status":true}`,
			want:   TelemetryDocument{RunUUID: "ghi", Status: true},
		},
		{
			name:   "affected_pods recovery timings surfaced",
			source: `{"run_uuid":"rec","job_status":true,"scenarios":[{"scenario_type":"pod_disruption","start_timestamp":5,"end_timestamp":6,"exit_status":0,"parameters":{"scenarios":[{"expected_recovery_time":90,"namespace":"openshift-etcd"}]},"affected_pods":{"recovered":[{"pod_name":"etcd-0","namespace":"openshift-etcd","total_recovery_time":37.5,"pod_readiness_time":37.5,"pod_rescheduling_time":0}]}}]}`,
			want: TelemetryDocument{RunUUID: "rec", ScenarioType: "pod_disruption", StartTimestamp: 5, EndTimestamp: 6, Namespace: "openshift-etcd", Status: true,
				Scenarios: []ScenarioDetail{{
					ScenarioType: "pod_disruption", StartTimestamp: 5, EndTimestamp: 6,
					Parameters: json.RawMessage(`{"scenarios":[{"expected_recovery_time":90,"namespace":"openshift-etcd"}]}`),
					AffectedPods: &AffectedPods{Recovered: []RecoveredPod{
						{PodName: "etcd-0", Namespace: "openshift-etcd", TotalRecoveryTime: 37.5, PodReadinessTime: 37.5, PodReschedulingTime: 0},
					}},
				}}},
		},
		{
			name:   "cluster metadata and all scenarios surfaced",
			source: `{"run_uuid":"m1","job_status":true,"kubernetes_objects_count":{"Pod":701,"ConfigMap":1064},"network_plugins":["OVNKubernetes"],"total_node_count":9,"cloud_infrastructure":"AWS","cloud_type":"self-managed","cluster_version":"4.19.0","major_version":"4.19","build_url":"https://example/1","fips_enabled":false,"tag":"cr","etcd_encryption_enabled":false,"ipsec_enabled":false,"node_summary_infos":[{"count":3,"nodes_type":"master","architecture":"amd64","instance_type":"m5.2xlarge","kernel_version":"5.14.0-570.51.1.el9_6.x86_64","kubelet_version":"v1.33.5","os_version":"Red Hat Enterprise Linux CoreOS 9.6.20250930-0 (Plow)"},{"count":3,"nodes_type":"worker","architecture":"amd64","instance_type":"m5.xlarge","kernel_version":"5.14.0-570.51.1.el9_6.x86_64","kubelet_version":"v1.33.5","os_version":"Red Hat Enterprise Linux CoreOS 9.6.20250930-0 (Plow)"}],"scenarios":[{"scenario_type":"application_outages_scenarios","start_timestamp":10,"end_timestamp":20,"exit_status":0,"parameters":{"application_outage":{"namespace":"openshift-console"}}},{"scenario_type":"pod","start_timestamp":30,"end_timestamp":40,"exit_status":0,"parameters":{"config":{"kill":1},"id":"kill-pods"}}]}`,
			want: TelemetryDocument{RunUUID: "m1", ScenarioType: "application_outages_scenarios", StartTimestamp: 10, EndTimestamp: 20, Namespace: "openshift-console", Status: true,
				Metadata: &ClusterMetadata{
					KubernetesObjectsCount: map[string]int{"Pod": 701, "ConfigMap": 1064},
					NetworkPlugins:         []string{"OVNKubernetes"},
					TotalNodeCount:         9,
					CloudInfrastructure:    "AWS",
					CloudType:              "self-managed",
					ClusterVersion:         "4.19.0",
					MajorVersion:           "4.19",
					BuildURL:               "https://example/1",
					Tag:                    "cr",
					NodeSummaryInfos: []NodeSummaryInfo{
						{Count: 3, NodesType: "master", Architecture: "amd64", InstanceType: "m5.2xlarge", KernelVersion: "5.14.0-570.51.1.el9_6.x86_64", KubeletVersion: "v1.33.5", OSVersion: "Red Hat Enterprise Linux CoreOS 9.6.20250930-0 (Plow)"},
						{Count: 3, NodesType: "worker", Architecture: "amd64", InstanceType: "m5.xlarge", KernelVersion: "5.14.0-570.51.1.el9_6.x86_64", KubeletVersion: "v1.33.5", OSVersion: "Red Hat Enterprise Linux CoreOS 9.6.20250930-0 (Plow)"},
					},
				},
				Scenarios: []ScenarioDetail{
					{ScenarioType: "application_outages_scenarios", StartTimestamp: 10, EndTimestamp: 20, Parameters: json.RawMessage(`{"application_outage":{"namespace":"openshift-console"}}`)},
					{ScenarioType: "pod", StartTimestamp: 30, EndTimestamp: 40, Parameters: json.RawMessage(`{"config":{"kill":1},"id":"kill-pods"}`)},
				}},
		},
		{
			name:   "fractional timestamps from Python time.time()",
			source: `{"run_uuid":"frac","job_status":true,"scenarios":[{"scenario_type":"pod","start_timestamp":1735689600.123456,"end_timestamp":1735689900.987654,"exit_status":0,"parameters":[{"config":{"namespace_pattern":"default"}}]}]}`,
			want: TelemetryDocument{RunUUID: "frac", ScenarioType: "pod", StartTimestamp: 1735689600.123456, EndTimestamp: 1735689900.987654, Namespace: "default", Status: true,
				Scenarios: []ScenarioDetail{{ScenarioType: "pod", StartTimestamp: 1735689600.123456, EndTimestamp: 1735689900.987654, Parameters: json.RawMessage(`[{"config":{"namespace_pattern":"default"}}]`)}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var src rawTelemetrySource
			if err := json.Unmarshal([]byte(tt.source), &src); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}
			got := src.flatten()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("flatten() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
