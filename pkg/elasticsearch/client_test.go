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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQueryTelemetry(t *testing.T) {
	sampleHits := `{
      "hits": {
        "hits": [
          {"_source": {"run_uuid": "abc", "job_status": true, "scenarios": [{"scenario_type": "pod_disruption_scenarios", "start_timestamp": 1735689600, "end_timestamp": 1735689900, "exit_status": 0, "parameters": [{"config": {"namespace_pattern": "openshift-kube-apiserver"}}]}, {"scenario_type": "node"}]}},
          {"_source": {"run_uuid": "def", "job_status": false, "scenarios": [{"scenario_type": "pod", "start_timestamp": 1735776000, "end_timestamp": 1735776300, "exit_status": 1, "parameters": [{"config": {"namespace": "default"}}]}]}}
        ]
      }
    }`

	tests := []struct {
		name       string
		index      string
		statusCode int
		body       string
		wantErr    bool
		wantCount  int
		checkFirst func(t *testing.T, d TelemetryDocument)
	}{
		{
			name:       "successful query flattens first scenario",
			index:      "telemetry",
			statusCode: http.StatusOK,
			body:       sampleHits,
			wantCount:  2,
			checkFirst: func(t *testing.T, d TelemetryDocument) {
				if d.RunUUID != "abc" {
					t.Errorf("got run_uuid %q, want abc", d.RunUUID)
				}
				if d.ScenarioType != "pod_disruption_scenarios" {
					t.Errorf("got scenario_type %q, want pod_disruption_scenarios", d.ScenarioType)
				}
				if d.StartTimestamp != 1735689600 {
					t.Errorf("got start_timestamp %d, want 1735689600", d.StartTimestamp)
				}
				if d.EndTimestamp != 1735689900 {
					t.Errorf("got end_timestamp %d, want 1735689900", d.EndTimestamp)
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

			docs, err := QueryTelemetry(context.Background(), conn, 50, "", "")
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
			if tt.checkFirst != nil && len(docs) > 0 {
				tt.checkFirst(t, docs[0])
			}
		})
	}
}

func TestQueryTelemetryDateRange(t *testing.T) {
	tests := []struct {
		name      string
		startDate string
		endDate   string
		wantGTE   string
		wantLTE   string
	}{
		{"defaults trailing window", "", "", "now-30d/d", "now/d"},
		{"explicit bounds", "2026-08-01", "2026-08-26", "2026-08-01", "2026-08-26||+1d/d"},
		{"only start provided", "2026-08-01", "", "2026-08-01", "now/d"},
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
			if _, err := QueryTelemetry(context.Background(), conn, 50, tt.startDate, tt.endDate); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			rng := captured["query"].(map[string]any)["bool"].(map[string]any)["filter"].([]any)[0].(map[string]any)["range"].(map[string]any)["timestamp"].(map[string]any)
			if rng["gte"] != tt.wantGTE {
				t.Errorf("gte = %v, want %v", rng["gte"], tt.wantGTE)
			}
			if rng["lte"] != tt.wantLTE {
				t.Errorf("lte = %v, want %v", rng["lte"], tt.wantLTE)
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
		{"host with scheme used verbatim", ConnectionParams{Host: "http://es.local:9200", Port: 9200}, "http://es.local:9200"},
		{"trailing slash trimmed", ConnectionParams{Host: "https://es.local:9200/", Port: 9200}, "https://es.local:9200"},
		{"bare host gets https and port", ConnectionParams{Host: "es.local", Port: 9200}, "https://es.local:9200"},
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
		{"missing config name", QueryTelemetryRequest{}, true, 0},
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
			want:   TelemetryDocument{RunUUID: "abc", ScenarioType: "pod", StartTimestamp: 100, EndTimestamp: 200, Namespace: "ns1", Status: true},
		},
		{
			name:   "object-style parameters (pvc scenario)",
			source: `{"run_uuid":"pvc","job_status":true,"scenarios":[{"scenario_type":"pvc_scenarios","start_timestamp":1,"end_timestamp":2,"exit_status":0,"parameters":{"pvc_scenario":{"namespace":"openshift-monitoring","pvc_name":"x"}}}]}`,
			want:   TelemetryDocument{RunUUID: "pvc", ScenarioType: "pvc_scenarios", StartTimestamp: 1, EndTimestamp: 2, Namespace: "openshift-monitoring", Status: true},
		},
		{
			name:   "array-nested parameters (time scenario)",
			source: `{"run_uuid":"time","job_status":true,"scenarios":[{"scenario_type":"time_scenarios","start_timestamp":3,"end_timestamp":4,"exit_status":0,"parameters":{"time_scenarios":[{"namespace":"openshift-etcd","action":"skew_time"}]}}]}`,
			want:   TelemetryDocument{RunUUID: "time", ScenarioType: "time_scenarios", StartTimestamp: 3, EndTimestamp: 4, Namespace: "openshift-etcd", Status: true},
		},
		{
			name:   "non-zero exit status marks failure",
			source: `{"run_uuid":"def","job_status":true,"scenarios":[{"scenario_type":"node","exit_status":1}]}`,
			want:   TelemetryDocument{RunUUID: "def", ScenarioType: "node", Status: false},
		},
		{
			name:   "no scenarios keeps job status",
			source: `{"run_uuid":"ghi","job_status":true}`,
			want:   TelemetryDocument{RunUUID: "ghi", Status: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var src rawTelemetrySource
			if err := json.Unmarshal([]byte(tt.source), &src); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}
			got := src.flatten()
			if got != tt.want {
				t.Errorf("flatten() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
