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

	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

func TestBuildLabels(t *testing.T) {
	tests := []struct {
		name           string
		provider       string
		groups         []string
		availableToAll bool
		wantProvider   string
		wantGroups     int
		wantAvailable  bool
	}{
		{
			name:         "aws no groups",
			provider:     ProviderAWS,
			groups:       nil,
			wantProvider: ProviderAWS,
		},
		{
			name:         "gcp with groups",
			provider:     ProviderGCP,
			groups:       []string{"team-a", "team-b"},
			wantProvider: ProviderGCP,
			wantGroups:   2,
		},
		{
			name:           "azure available to all",
			provider:       ProviderAzure,
			availableToAll: true,
			wantProvider:   ProviderAzure,
			wantAvailable:  true,
		},
		{
			name:           "openstack with groups and available to all",
			provider:       ProviderOpenStack,
			groups:         []string{"ops"},
			availableToAll: true,
			wantProvider:   ProviderOpenStack,
			wantGroups:     1,
			wantAvailable:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			labels := BuildLabels(tt.provider, tt.groups, tt.availableToAll)

			if labels[AppNameLabel] != AppName {
				t.Errorf("AppNameLabel = %q, want %q", labels[AppNameLabel], AppName)
			}
			if labels[AppComponentLabel] != ComponentCloudCredential {
				t.Errorf("AppComponentLabel = %q, want %q", labels[AppComponentLabel], ComponentCloudCredential)
			}
			if labels[ProviderTypeLabel] != tt.wantProvider {
				t.Errorf("ProviderTypeLabel = %q, want %q", labels[ProviderTypeLabel], tt.wantProvider)
			}

			if tt.wantAvailable {
				if labels[AvailableToAllLabel] != "true" {
					t.Error("expected AvailableToAllLabel to be 'true'")
				}
			} else {
				if _, ok := labels[AvailableToAllLabel]; ok {
					t.Error("expected AvailableToAllLabel to be absent")
				}
			}

			groupCount := 0
			for key, value := range labels {
				if len(key) > len(groupauth.GroupLabelPrefix) && key[:len(groupauth.GroupLabelPrefix)] == groupauth.GroupLabelPrefix && value == "true" {
					groupCount++
				}
			}
			if groupCount != tt.wantGroups {
				t.Errorf("group label count = %d, want %d", groupCount, tt.wantGroups)
			}
		})
	}
}

func TestBuildAnnotations(t *testing.T) {
	annotations := BuildAnnotations("test cloud creds", "admin@example.com")

	if annotations[CreatedByAnnotation] != "admin@example.com" {
		t.Errorf("CreatedBy = %q, want %q", annotations[CreatedByAnnotation], "admin@example.com")
	}
	if annotations[CreatedAtAnnotation] == "" {
		t.Error("expected CreatedAt to be set")
	}
	if annotations[DescriptionAnnotation] != "test cloud creds" {
		t.Errorf("Description = %q, want %q", annotations[DescriptionAnnotation], "test cloud creds")
	}
}

func TestBuildAnnotationsNoDescription(t *testing.T) {
	annotations := BuildAnnotations("", "admin@example.com")

	if _, ok := annotations[DescriptionAnnotation]; ok {
		t.Error("expected DescriptionAnnotation to be absent when description is empty")
	}
}

func TestUpdateAnnotations(t *testing.T) {
	existing := map[string]string{
		CreatedByAnnotation:   "original@example.com",
		CreatedAtAnnotation:   "2026-01-01T00:00:00Z",
		DescriptionAnnotation: "old desc",
	}

	updated := UpdateAnnotations(existing, "new desc", "updater@example.com")

	if updated[CreatedByAnnotation] != "original@example.com" {
		t.Error("expected CreatedBy to be preserved")
	}
	if updated[CreatedAtAnnotation] != "2026-01-01T00:00:00Z" {
		t.Error("expected CreatedAt to be preserved")
	}
	if updated[UpdatedByAnnotation] != "updater@example.com" {
		t.Errorf("UpdatedBy = %q, want %q", updated[UpdatedByAnnotation], "updater@example.com")
	}
	if updated[UpdatedAtAnnotation] == "" {
		t.Error("expected UpdatedAt to be set")
	}
	if updated[DescriptionAnnotation] != "new desc" {
		t.Errorf("Description = %q, want %q", updated[DescriptionAnnotation], "new desc")
	}
}

func TestUpdateAnnotationsClearsDescription(t *testing.T) {
	existing := map[string]string{
		DescriptionAnnotation: "old desc",
	}

	updated := UpdateAnnotations(existing, "", "updater@example.com")

	if _, ok := updated[DescriptionAnnotation]; ok {
		t.Error("expected DescriptionAnnotation to be removed when empty")
	}
}

func TestExtractGroupsFromLabels(t *testing.T) {
	labels := map[string]string{
		AppNameLabel:                      AppName,
		AppComponentLabel:                 ComponentCloudCredential,
		groupauth.GroupLabelKey("team-a"): "true",
		groupauth.GroupLabelKey("team-b"): "true",
		groupauth.GroupLabelKey("team-c"): "false",
	}

	groups := ExtractGroupsFromLabels(labels)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	groupSet := make(map[string]bool)
	for _, g := range groups {
		groupSet[g] = true
	}
	if !groupSet["team-a"] || !groupSet["team-b"] {
		t.Errorf("expected groups [team-a, team-b], got %v", groups)
	}
}
