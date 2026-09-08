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

package visualize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCreateRequest(t *testing.T) {
	tests := []struct {
		name    string
		request CreateVisualizeRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request with auto-detect prometheus",
			request: CreateVisualizeRequest{
				Name:                 "test-visualize",
				TargetClusters:       []string{"cluster-1"},
				Namespace:            "krkn-visualize",
				GrafanaPassword:      "admin123",
				AutoDetectPrometheus: true,
			},
			wantErr: false,
		},
		{
			name: "valid request with manual prometheus",
			request: CreateVisualizeRequest{
				Name:                  "test-visualize",
				TargetClusters:        []string{"cluster-1"},
				GrafanaPassword:       "admin123",
				AutoDetectPrometheus:  false,
				PrometheusURL:         "https://prometheus.example.com",
				PrometheusBearerToken: "token123",
			},
			wantErr: false,
		},
		{
			name: "valid with elasticsearch config",
			request: CreateVisualizeRequest{
				Name:                    "test-visualize",
				TargetClusters:          []string{"cluster-1"},
				ElasticsearchConfigName: "production-es",
				GrafanaPassword:         "admin123",
				AutoDetectPrometheus:    true,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			request: CreateVisualizeRequest{
				TargetClusters:  []string{"cluster-1"},
				GrafanaPassword: "admin123",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "invalid name format",
			request: CreateVisualizeRequest{
				Name:            "Test_Visualize",
				TargetClusters:  []string{"cluster-1"},
				GrafanaPassword: "admin123",
			},
			wantErr: true,
			errMsg:  "lowercase alphanumeric",
		},
		{
			name: "missing target clusters",
			request: CreateVisualizeRequest{
				Name:            "test-visualize",
				TargetClusters:  []string{},
				GrafanaPassword: "admin123",
			},
			wantErr: true,
			errMsg:  "at least one target cluster is required",
		},
		{
			name: "missing grafana password",
			request: CreateVisualizeRequest{
				Name:           "test-visualize",
				TargetClusters: []string{"cluster-1"},
			},
			wantErr: true,
			errMsg:  "grafanaPassword is required",
		},
		{
			name: "grafana password too short",
			request: CreateVisualizeRequest{
				Name:            "test-visualize",
				TargetClusters:  []string{"cluster-1"},
				GrafanaPassword: "abc",
			},
			wantErr: true,
			errMsg:  "at least 4 characters",
		},
		{
			name: "invalid namespace format",
			request: CreateVisualizeRequest{
				Name:            "test-visualize",
				TargetClusters:  []string{"cluster-1"},
				Namespace:       "Invalid_Namespace",
				GrafanaPassword: "admin123",
			},
			wantErr: true,
			errMsg:  "valid Kubernetes namespace name",
		},
		{
			name: "missing prometheus url when not auto-detecting",
			request: CreateVisualizeRequest{
				Name:                 "test-visualize",
				TargetClusters:       []string{"cluster-1"},
				GrafanaPassword:      "admin123",
				AutoDetectPrometheus: false,
			},
			wantErr: true,
			errMsg:  "prometheusUrl is required when autoDetectPrometheus is false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateRequest(&tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
