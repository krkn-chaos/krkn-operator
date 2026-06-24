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

package files

import (
	"testing"
	"time"
)

func TestBuildFileLabels(t *testing.T) {
	tests := []struct {
		name           string
		fileType       string
		groups         []string
		availableToAll bool
		want           map[string]string
	}{
		{
			name:           "with groups",
			fileType:       "",
			groups:         []string{"dev-team", "qa-team"},
			availableToAll: false,
			want: map[string]string{
				AppNameLabel:                         AppName,
				AppComponentLabel:                    ComponentFile,
				"group.krkn.krkn-chaos.dev/dev-team": "true",
				"group.krkn.krkn-chaos.dev/qa-team":  "true",
			},
		},
		{
			name:           "available to all",
			fileType:       "",
			groups:         []string{},
			availableToAll: true,
			want: map[string]string{
				AppNameLabel:        AppName,
				AppComponentLabel:   ComponentFile,
				AvailableToAllLabel: "true",
			},
		},
		{
			name:           "no groups, not public",
			fileType:       "",
			groups:         []string{},
			availableToAll: false,
			want: map[string]string{
				AppNameLabel:      AppName,
				AppComponentLabel: ComponentFile,
			},
		},
		{
			name:           "single group",
			fileType:       "",
			groups:         []string{"ops-team"},
			availableToAll: false,
			want: map[string]string{
				AppNameLabel:                         AppName,
				AppComponentLabel:                    ComponentFile,
				"group.krkn.krkn-chaos.dev/ops-team": "true",
			},
		},
		{
			name:           "with file type",
			fileType:       "config",
			groups:         []string{},
			availableToAll: false,
			want: map[string]string{
				AppNameLabel:                           AppName,
				AppComponentLabel:                      ComponentFile,
				"file-type.krkn.krkn-chaos.dev/config": "true",
			},
		},
		{
			name:           "with file type and groups",
			fileType:       "script",
			groups:         []string{"dev-team"},
			availableToAll: false,
			want: map[string]string{
				AppNameLabel:                           AppName,
				AppComponentLabel:                      ComponentFile,
				"file-type.krkn.krkn-chaos.dev/script": "true",
				"group.krkn.krkn-chaos.dev/dev-team":   "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFileLabels(tt.fileType, tt.groups, tt.availableToAll)

			// Check all expected labels are present
			for key, wantValue := range tt.want {
				gotValue, exists := got[key]
				if !exists {
					t.Errorf("BuildFileLabels() missing key %s", key)
					continue
				}
				if gotValue != wantValue {
					t.Errorf("BuildFileLabels()[%s] = %v, want %v", key, gotValue, wantValue)
				}
			}

			// Check no extra labels
			if len(got) != len(tt.want) {
				t.Errorf("BuildFileLabels() has %d labels, want %d", len(got), len(tt.want))
			}
		})
	}
}

func TestBuildFileAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		description string
		createdBy   string
	}{
		{
			name:        "full configuration",
			description: "Application configuration file",
			createdBy:   "admin@example.com",
		},
		{
			name:        "minimal configuration",
			description: "",
			createdBy:   "user@example.com",
		},
		{
			name:        "with description",
			description: "Service settings",
			createdBy:   "ops@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFileAnnotations(
				tt.description,
				tt.createdBy,
			)

			// Check required annotations
			if got[CreatedByAnnotation] != tt.createdBy {
				t.Errorf("CreatedByAnnotation = %v, want %v", got[CreatedByAnnotation], tt.createdBy)
			}

			// Check description
			if tt.description != "" {
				if got[DescriptionAnnotation] != tt.description {
					t.Errorf("DescriptionAnnotation = %v, want %v", got[DescriptionAnnotation], tt.description)
				}
			} else {
				if _, exists := got[DescriptionAnnotation]; exists {
					t.Errorf("DescriptionAnnotation should not exist for empty description")
				}
			}

			// Check timestamps exist
			if _, exists := got[CreatedAtAnnotation]; !exists {
				t.Error("CreatedAtAnnotation missing")
			}

			// Verify timestamp format
			createdAt := got[CreatedAtAnnotation]
			if _, err := time.Parse(time.RFC3339, createdAt); err != nil {
				t.Errorf("CreatedAtAnnotation has invalid RFC3339 format: %v", err)
			}
		})
	}
}

func TestUpdateFileAnnotations(t *testing.T) {
	existing := map[string]string{
		DescriptionAnnotation: "Old description",
		CreatedByAnnotation:   "admin@example.com",
		CreatedAtAnnotation:   "2025-01-01T00:00:00Z",
	}

	tests := []struct {
		name        string
		existing    map[string]string
		description string
		updatedBy   string
	}{
		{
			name:        "update all fields",
			existing:    existing,
			description: "New description",
			updatedBy:   "user@example.com",
		},
		{
			name:        "remove description",
			existing:    existing,
			description: "",
			updatedBy:   "admin@example.com",
		},
		{
			name:        "update description only",
			existing:    existing,
			description: "Same description",
			updatedBy:   "ops@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UpdateFileAnnotations(
				tt.existing,
				tt.description,
				tt.updatedBy,
			)

			// Check updated fields
			if got[UpdatedByAnnotation] != tt.updatedBy {
				t.Errorf("UpdatedByAnnotation = %v, want %v", got[UpdatedByAnnotation], tt.updatedBy)
			}

			// Check original fields preserved
			if got[CreatedByAnnotation] != existing[CreatedByAnnotation] {
				t.Errorf("CreatedByAnnotation should be preserved")
			}
			if got[CreatedAtAnnotation] != existing[CreatedAtAnnotation] {
				t.Errorf("CreatedAtAnnotation should be preserved")
			}

			// Check description handling
			if tt.description != "" {
				if got[DescriptionAnnotation] != tt.description {
					t.Errorf("DescriptionAnnotation = %v, want %v", got[DescriptionAnnotation], tt.description)
				}
			} else {
				if _, exists := got[DescriptionAnnotation]; exists {
					t.Errorf("DescriptionAnnotation should be removed when empty")
				}
			}

			// Check updated timestamp exists
			if _, exists := got[UpdatedAtAnnotation]; !exists {
				t.Error("UpdatedAtAnnotation missing")
			}

			// Verify timestamp format
			updatedAt := got[UpdatedAtAnnotation]
			if _, err := time.Parse(time.RFC3339, updatedAt); err != nil {
				t.Errorf("UpdatedAtAnnotation has invalid RFC3339 format: %v", err)
			}
		})
	}
}

func TestExtractGroupsFromLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   []string
	}{
		{
			name: "single group",
			labels: map[string]string{
				"group.krkn.krkn-chaos.dev/dev-team": "true",
			},
			want: []string{"dev-team"},
		},
		{
			name: "multiple groups",
			labels: map[string]string{
				"group.krkn.krkn-chaos.dev/dev-team": "true",
				"group.krkn.krkn-chaos.dev/qa-team":  "true",
				"group.krkn.krkn-chaos.dev/ops-team": "true",
			},
			want: []string{"dev-team", "qa-team", "ops-team"},
		},
		{
			name: "mixed labels",
			labels: map[string]string{
				"group.krkn.krkn-chaos.dev/dev-team":   "true",
				"app.kubernetes.io/name":               "krkn-operator",
				"files.krkn.krkn-chaos.dev/mount-path": "/etc/config",
			},
			want: []string{"dev-team"},
		},
		{
			name: "group with false value",
			labels: map[string]string{
				"group.krkn.krkn-chaos.dev/dev-team": "false",
			},
			want: []string{},
		},
		{
			name:   "no groups",
			labels: map[string]string{},
			want:   []string{},
		},
		{
			name: "available to all without groups",
			labels: map[string]string{
				AppNameLabel:        AppName,
				AppComponentLabel:   ComponentFile,
				AvailableToAllLabel: "true",
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractGroupsFromLabels(tt.labels)

			// Convert to maps for comparison (order doesn't matter)
			gotMap := make(map[string]bool)
			for _, g := range got {
				gotMap[g] = true
			}
			wantMap := make(map[string]bool)
			for _, g := range tt.want {
				wantMap[g] = true
			}

			if len(gotMap) != len(wantMap) {
				t.Errorf("ExtractGroupsFromLabels() returned %d groups, want %d", len(got), len(tt.want))
				return
			}

			for g := range wantMap {
				if !gotMap[g] {
					t.Errorf("ExtractGroupsFromLabels() missing group %s", g)
				}
			}
		})
	}
}
