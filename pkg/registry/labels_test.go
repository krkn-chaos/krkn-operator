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

package registry

import (
	"testing"
	"time"
)

func TestBuildRegistryLabels(t *testing.T) {
	tests := []struct {
		name           string
		authType       string
		groups         []string
		availableToAll bool
		want           map[string]string
	}{
		{
			name:           "token auth with groups",
			authType:       AuthTypeToken,
			groups:         []string{"dev-team", "qa-team"},
			availableToAll: false,
			want: map[string]string{
				AppNameLabel:                         AppName,
				AppComponentLabel:                    ComponentRegistry,
				AuthTypeLabel:                        AuthTypeToken,
				"group.krkn.krkn-chaos.dev/dev-team": "true",
				"group.krkn.krkn-chaos.dev/qa-team":  "true",
			},
		},
		{
			name:           "password auth available to all",
			authType:       AuthTypePassword,
			groups:         []string{},
			availableToAll: true,
			want: map[string]string{
				AppNameLabel:        AppName,
				AppComponentLabel:   ComponentRegistry,
				AuthTypeLabel:       AuthTypePassword,
				AvailableToAllLabel: "true",
			},
		},
		{
			name:           "token auth no groups",
			authType:       AuthTypeToken,
			groups:         []string{},
			availableToAll: false,
			want: map[string]string{
				AppNameLabel:      AppName,
				AppComponentLabel: ComponentRegistry,
				AuthTypeLabel:     AuthTypeToken,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildRegistryLabels(tt.authType, tt.groups, tt.availableToAll)

			// Check all expected labels are present
			for key, wantValue := range tt.want {
				gotValue, exists := got[key]
				if !exists {
					t.Errorf("BuildRegistryLabels() missing key %s", key)
					continue
				}
				if gotValue != wantValue {
					t.Errorf("BuildRegistryLabels()[%s] = %v, want %v", key, gotValue, wantValue)
				}
			}

			// Check no extra labels
			if len(got) != len(tt.want) {
				t.Errorf("BuildRegistryLabels() has %d labels, want %d", len(got), len(tt.want))
			}
		})
	}
}

func TestBuildRegistryAnnotations(t *testing.T) {
	tests := []struct {
		name         string
		registryURL  string
		scenarioRepo string
		description  string
		skipTLS      bool
		insecure     bool
		createdBy    string
	}{
		{
			name:         "full configuration",
			registryURL:  "registry.example.com",
			scenarioRepo: "krkn-chaos/krkn-hub",
			description:  "Production registry",
			skipTLS:      true,
			insecure:     false,
			createdBy:    "admin@example.com",
		},
		{
			name:         "minimal configuration",
			registryURL:  "registry.io",
			scenarioRepo: "org/repo",
			description:  "",
			skipTLS:      false,
			insecure:     false,
			createdBy:    "user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildRegistryAnnotations(
				tt.registryURL,
				tt.scenarioRepo,
				tt.description,
				tt.skipTLS,
				tt.insecure,
				tt.createdBy,
			)

			// Check required annotations
			if got[RegistryURLAnnotation] != tt.registryURL {
				t.Errorf("RegistryURLAnnotation = %v, want %v", got[RegistryURLAnnotation], tt.registryURL)
			}
			if got[ScenarioRepositoryAnnotation] != tt.scenarioRepo {
				t.Errorf("ScenarioRepositoryAnnotation = %v, want %v", got[ScenarioRepositoryAnnotation], tt.scenarioRepo)
			}
			if got[CreatedByAnnotation] != tt.createdBy {
				t.Errorf("CreatedByAnnotation = %v, want %v", got[CreatedByAnnotation], tt.createdBy)
			}

			// Check boolean annotations
			skipTLSStr := "false"
			if tt.skipTLS {
				skipTLSStr = "true"
			}
			if got[SkipTLSAnnotation] != skipTLSStr {
				t.Errorf("SkipTLSAnnotation = %v, want %v", got[SkipTLSAnnotation], skipTLSStr)
			}

			insecureStr := "false"
			if tt.insecure {
				insecureStr = "true"
			}
			if got[InsecureAnnotation] != insecureStr {
				t.Errorf("InsecureAnnotation = %v, want %v", got[InsecureAnnotation], insecureStr)
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

func TestUpdateRegistryAnnotations(t *testing.T) {
	existing := map[string]string{
		RegistryURLAnnotation:        "old-registry.io",
		ScenarioRepositoryAnnotation: "old/repo",
		CreatedByAnnotation:          "admin@example.com",
		CreatedAtAnnotation:          "2025-01-01T00:00:00Z",
	}

	tests := []struct {
		name         string
		existing     map[string]string
		registryURL  string
		scenarioRepo string
		description  string
		skipTLS      bool
		insecure     bool
		updatedBy    string
	}{
		{
			name:         "update all fields",
			existing:     existing,
			registryURL:  "new-registry.io",
			scenarioRepo: "new/repo",
			description:  "Updated description",
			skipTLS:      true,
			insecure:     true,
			updatedBy:    "user@example.com",
		},
		{
			name:         "remove description",
			existing:     existing,
			registryURL:  "registry.io",
			scenarioRepo: "org/repo",
			description:  "",
			skipTLS:      false,
			insecure:     false,
			updatedBy:    "admin@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UpdateRegistryAnnotations(
				tt.existing,
				tt.registryURL,
				tt.scenarioRepo,
				tt.description,
				tt.skipTLS,
				tt.insecure,
				tt.updatedBy,
			)

			// Check updated fields
			if got[RegistryURLAnnotation] != tt.registryURL {
				t.Errorf("RegistryURLAnnotation = %v, want %v", got[RegistryURLAnnotation], tt.registryURL)
			}
			if got[ScenarioRepositoryAnnotation] != tt.scenarioRepo {
				t.Errorf("ScenarioRepositoryAnnotation = %v, want %v", got[ScenarioRepositoryAnnotation], tt.scenarioRepo)
			}
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
				"group.krkn.krkn-chaos.dev/dev-team":     "true",
				"app.kubernetes.io/name":                 "krkn-operator",
				"registry.krkn.krkn-chaos.dev/auth-type": "token",
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
