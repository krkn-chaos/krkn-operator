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

func TestMetadataPresenceTracking(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   *ClusterMetadata
	}{
		{
			name:   "explicit false values preserved",
			source: `{"run_uuid":"test","job_status":true,"fips_enabled":false,"etcd_encryption_enabled":false,"ipsec_enabled":false,"scenarios":[]}`,
			want: &ClusterMetadata{
				FIPSEnabled:           boolPtr(false),
				EtcdEncryptionEnabled: boolPtr(false),
				IPSecEnabled:          boolPtr(false),
			},
		},
		{
			name:   "explicit zero node count preserved",
			source: `{"run_uuid":"test","job_status":true,"total_node_count":0,"scenarios":[]}`,
			want: &ClusterMetadata{
				TotalNodeCount: intPtr(0),
			},
		},
		{
			name:   "explicit true values preserved",
			source: `{"run_uuid":"test","job_status":true,"fips_enabled":true,"etcd_encryption_enabled":true,"ipsec_enabled":true,"scenarios":[]}`,
			want: &ClusterMetadata{
				FIPSEnabled:           boolPtr(true),
				EtcdEncryptionEnabled: boolPtr(true),
				IPSecEnabled:          boolPtr(true),
			},
		},
		{
			name:   "absent fields result in nil",
			source: `{"run_uuid":"test","job_status":true,"scenarios":[]}`,
			want:   nil,
		},
		{
			name:   "mix of present and absent fields",
			source: `{"run_uuid":"test","job_status":true,"fips_enabled":false,"total_node_count":5,"scenarios":[]}`,
			want: &ClusterMetadata{
				FIPSEnabled:    boolPtr(false),
				TotalNodeCount: intPtr(5),
			},
		},
		{
			name:   "explicit false with other metadata present",
			source: `{"run_uuid":"test","job_status":true,"cluster_version":"4.19.0","fips_enabled":false,"scenarios":[]}`,
			want: &ClusterMetadata{
				ClusterVersion: "4.19.0",
				FIPSEnabled:    boolPtr(false),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var src rawTelemetrySource
			if err := json.Unmarshal([]byte(tt.source), &src); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			got := src.metadata()

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("metadata() mismatch:\ngot:  %+v\nwant: %+v", got, tt.want)
				if got != nil && tt.want != nil {
					if got.FIPSEnabled != nil && tt.want.FIPSEnabled != nil {
						t.Logf("FIPSEnabled: got=%v, want=%v", *got.FIPSEnabled, *tt.want.FIPSEnabled)
					}
					if got.TotalNodeCount != nil && tt.want.TotalNodeCount != nil {
						t.Logf("TotalNodeCount: got=%v, want=%v", *got.TotalNodeCount, *tt.want.TotalNodeCount)
					}
				}
			}
		})
	}
}

func TestMetadataJSONMarshaling(t *testing.T) {
	// Verify that pointer fields with false/zero are included in JSON output
	metadata := &ClusterMetadata{
		ClusterVersion:        "4.19.0",
		FIPSEnabled:           boolPtr(false),
		EtcdEncryptionEnabled: boolPtr(true),
		TotalNodeCount:        intPtr(0),
	}

	jsonBytes, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Verify explicit false is present in JSON
	if !contains(jsonStr, `"fips_enabled":false`) {
		t.Errorf("expected fips_enabled:false in JSON, got: %s", jsonStr)
	}

	// Verify explicit true is present in JSON
	if !contains(jsonStr, `"etcd_encryption_enabled":true`) {
		t.Errorf("expected etcd_encryption_enabled:true in JSON, got: %s", jsonStr)
	}

	// Verify explicit zero is present in JSON
	if !contains(jsonStr, `"total_node_count":0`) {
		t.Errorf("expected total_node_count:0 in JSON, got: %s", jsonStr)
	}

	// Verify absent field (IPSecEnabled) is not in JSON
	if contains(jsonStr, `"ipsec_enabled"`) {
		t.Errorf("expected ipsec_enabled to be omitted, got: %s", jsonStr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
