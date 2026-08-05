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

import "testing"

func TestValidateCreateRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateElasticsearchConfigRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid minimal request",
			req:     CreateElasticsearchConfigRequest{Name: "my-es", Host: "https://es.example.com"},
			wantErr: false,
		},
		{
			name:    "valid full request",
			req:     CreateElasticsearchConfigRequest{Name: "prod-es", Host: "https://es.example.com", Port: 9200, Username: "elastic", Password: "secret"},
			wantErr: false,
		},
		{
			name:    "missing name",
			req:     CreateElasticsearchConfigRequest{Host: "https://es.example.com"},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name:    "missing host",
			req:     CreateElasticsearchConfigRequest{Name: "my-es"},
			wantErr: true,
			errMsg:  "host is required",
		},
		{
			name:    "negative port",
			req:     CreateElasticsearchConfigRequest{Name: "my-es", Host: "https://es.example.com", Port: -1},
			wantErr: true,
			errMsg:  "port must be between 0 and 65535",
		},
		{
			name:    "port above max",
			req:     CreateElasticsearchConfigRequest{Name: "my-es", Host: "https://es.example.com", Port: 65536},
			wantErr: true,
			errMsg:  "port must be between 0 and 65535",
		},
		{
			name:    "zero port is valid (uses default)",
			req:     CreateElasticsearchConfigRequest{Name: "my-es", Host: "https://es.example.com", Port: 0},
			wantErr: false,
		},
		{
			name:    "max valid port",
			req:     CreateElasticsearchConfigRequest{Name: "my-es", Host: "https://es.example.com", Port: 65535},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateRequest(&tt.req)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateCreateRequest() expected error %q but got nil", tt.errMsg)
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("ValidateCreateRequest() error = %q, want %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateCreateRequest() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateUpdateRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateElasticsearchConfigRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid minimal request",
			req:     UpdateElasticsearchConfigRequest{Host: "https://es.example.com"},
			wantErr: false,
		},
		{
			name:    "valid full request",
			req:     UpdateElasticsearchConfigRequest{Host: "https://es.example.com", Port: 9300, Username: "elastic", Password: "new-secret"},
			wantErr: false,
		},
		{
			name:    "missing host",
			req:     UpdateElasticsearchConfigRequest{Port: 9200},
			wantErr: true,
			errMsg:  "host is required",
		},
		{
			name:    "negative port",
			req:     UpdateElasticsearchConfigRequest{Host: "https://es.example.com", Port: -1},
			wantErr: true,
			errMsg:  "port must be between 0 and 65535",
		},
		{
			name:    "port above max",
			req:     UpdateElasticsearchConfigRequest{Host: "https://es.example.com", Port: 70000},
			wantErr: true,
			errMsg:  "port must be between 0 and 65535",
		},
		{
			name:    "zero port is valid",
			req:     UpdateElasticsearchConfigRequest{Host: "https://es.example.com", Port: 0},
			wantErr: false,
		},
		{
			name:    "password can be empty on update (keeps existing)",
			req:     UpdateElasticsearchConfigRequest{Host: "https://es.example.com", Username: "elastic", Password: ""},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateRequest(&tt.req)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateUpdateRequest() expected error %q but got nil", tt.errMsg)
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("ValidateUpdateRequest() error = %q, want %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateUpdateRequest() unexpected error: %v", err)
				}
			}
		})
	}
}
