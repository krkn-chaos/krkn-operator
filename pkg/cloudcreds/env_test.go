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

package cloudcreds

import (
	"testing"
)

func TestIsCloudEnvVar(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"CLOUD_TYPE", true},
		{"DISKS", true},
		{"AWS_SECRET_ACCESS_KEY", true},
		{"AZURE_CLIENT_SECRET", true},
		{"OS_PASSWORD", true},
		{"GOOGLE_APPLICATION_CREDENTIALS", true},
		{"BMC_PASSWORD", true},
		{"VSPHERE_PASSWORD", true},
		{"IBMC_APIKEY", true},
		{"TIMEOUT", false},
		{"ES_PASSWORD", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsCloudEnvVar(tt.key); got != tt.want {
			t.Errorf("IsCloudEnvVar(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestStripCloudEnvVars(t *testing.T) {
	if StripCloudEnvVars(nil) != nil {
		t.Fatal("expected nil for nil input")
	}

	in := map[string]string{
		"TIMEOUT":              "300",
		"AWS_SECRET_ACCESS_KEY": "plaintext",
		"CLOUD_TYPE":           "aws",
		"NODE_NAME":            "worker-1",
	}
	out := StripCloudEnvVars(in)
	if len(out) != 2 {
		t.Fatalf("expected 2 remaining keys, got %d: %v", len(out), out)
	}
	if out["TIMEOUT"] != "300" || out["NODE_NAME"] != "worker-1" {
		t.Errorf("unexpected stripped map: %v", out)
	}
	if _, ok := in["AWS_SECRET_ACCESS_KEY"]; !ok {
		t.Error("StripCloudEnvVars must not mutate the input map")
	}
}
