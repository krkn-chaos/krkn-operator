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
	"encoding/json"
	"reflect"
	"testing"
)

func TestRedactSensitiveParameters(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "redacts password in flat object",
			input: `{"namespace":"default","password":"secret123"}`,
			want:  `{"namespace":"default","password":"***REDACTED***"}`,
		},
		{
			name:  "redacts multiple sensitive keys",
			input: `{"user":"admin","password":"secret","token":"abc123","api_key":"xyz"}`,
			want:  `{"api_key":"***REDACTED***","password":"***REDACTED***","token":"***REDACTED***","user":"admin"}`,
		},
		{
			name:  "redacts AWS credentials",
			input: `{"AWS_ACCESS_KEY_ID":"AKIAIOSFODNN7EXAMPLE","AWS_SECRET_ACCESS_KEY":"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"}`,
			want:  `{"AWS_ACCESS_KEY_ID":"***REDACTED***","AWS_SECRET_ACCESS_KEY":"***REDACTED***"}`,
		},
		{
			name:  "redacts nested sensitive keys",
			input: `{"config":{"namespace":"ns1","credentials":{"password":"secret","user":"admin"}}}`,
			want:  `{"config":{"credentials":"***REDACTED***","namespace":"ns1"}}`,
		},
		{
			name:  "redacts in arrays",
			input: `[{"config":{"password":"pass1"}},{"config":{"password":"pass2"}}]`,
			want:  `[{"config":{"password":"***REDACTED***"}},{"config":{"password":"***REDACTED***"}}]`,
		},
		{
			name:  "case insensitive matching",
			input: `{"Password":"secret","TOKEN":"abc","ApiKey":"xyz"}`,
			want:  `{"ApiKey":"***REDACTED***","Password":"***REDACTED***","TOKEN":"***REDACTED***"}`,
		},
		{
			name:  "preserves non-sensitive fields",
			input: `{"namespace":"default","action":"delete","count":5}`,
			want:  `{"action":"delete","count":5,"namespace":"default"}`,
		},
		{
			name:  "handles empty object",
			input: `{}`,
			want:  `{}`,
		},
		{
			name:  "handles empty array",
			input: `[]`,
			want:  `[]`,
		},
		{
			name:  "returns original for malformed JSON",
			input: `{invalid json`,
			want:  `{invalid json`,
		},
		{
			name:  "redacts OS_PASSWORD",
			input: `{"OS_USERNAME":"admin","OS_PASSWORD":"secret123"}`,
			want:  `{"OS_PASSWORD":"***REDACTED***","OS_USERNAME":"admin"}`,
		},
		{
			name:  "redacts Azure secrets",
			input: `{"AZURE_CLIENT_ID":"client123","AZURE_CLIENT_SECRET":"secret456"}`,
			want:  `{"AZURE_CLIENT_ID":"***REDACTED***","AZURE_CLIENT_SECRET":"***REDACTED***"}`,
		},
		{
			name:  "real scenario example with secrets",
			input: `{"scenario":{"namespace":"openshift-etcd","password":"prod_secret","config":{"timeout":30}}}`,
			want:  `{"scenario":{"config":{"timeout":30},"namespace":"openshift-etcd","password":"***REDACTED***"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := json.RawMessage(tt.input)
			got := redactSensitiveParameters(input)

			// For malformed JSON, check exact match
			if tt.input == tt.want {
				if string(got) != tt.want {
					t.Errorf("redactSensitiveParameters() = %s, want %s", got, tt.want)
				}
				return
			}

			// For valid JSON, compare as normalized JSON to ignore key ordering
			var gotObj, wantObj any
			if err := json.Unmarshal(got, &gotObj); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.want), &wantObj); err != nil {
				t.Fatalf("failed to unmarshal expected: %v", err)
			}
			if !reflect.DeepEqual(gotObj, wantObj) {
				t.Errorf("redactSensitiveParameters() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestIsSensitiveKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"password", true},
		{"Password", true},
		{"PASSWORD", true},
		{"token", true},
		{"api_key", true},
		{"AWS_SECRET_ACCESS_KEY", true},
		{"namespace", false},
		{"config", false},
		{"name", false},
		{"secret_key", true},
		{"bearer_token", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := isSensitiveKey(tt.key)
			if got != tt.want {
				t.Errorf("isSensitiveKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestFlattenRedactsParameters(t *testing.T) {
	// Verify that flatten() applies redaction to scenario parameters
	source := `{
		"run_uuid": "test123",
		"job_status": true,
		"scenarios": [{
			"scenario_type": "pod",
			"start_timestamp": 100,
			"end_timestamp": 200,
			"exit_status": 0,
			"parameters": {
				"namespace": "default",
				"password": "secret123",
				"api_key": "xyz789"
			}
		}]
	}`

	var src rawTelemetrySource
	if err := json.Unmarshal([]byte(source), &src); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	doc := src.flatten()

	if len(doc.Scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(doc.Scenarios))
	}

	var params map[string]any
	if err := json.Unmarshal(doc.Scenarios[0].Parameters, &params); err != nil {
		t.Fatalf("failed to unmarshal redacted parameters: %v", err)
	}

	// Check that sensitive keys are redacted
	if params["password"] != "***REDACTED***" {
		t.Errorf("password not redacted, got %v", params["password"])
	}
	if params["api_key"] != "***REDACTED***" {
		t.Errorf("api_key not redacted, got %v", params["api_key"])
	}
	// Check that non-sensitive keys are preserved
	if params["namespace"] != "default" {
		t.Errorf("namespace should be preserved, got %v", params["namespace"])
	}
}
