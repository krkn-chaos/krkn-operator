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

Assisted-by: Claude Sonnet 4.5 (claude-sonnet-4-5@20250929)
*/

package groupauth

import (
	"testing"
)

func TestGroupLabelKey(t *testing.T) {
	tests := []struct {
		name      string
		groupName string
		want      string
	}{
		{
			name:      "simple name",
			groupName: "dev-team",
			want:      "group.krkn.krkn-chaos.dev/dev-team",
		},
		{
			name:      "name with spaces",
			groupName: "dev team",
			want:      "group.krkn.krkn-chaos.dev/dev-team",
		},
		{
			name:      "name with special chars",
			groupName: "dev@team!",
			want:      "group.krkn.krkn-chaos.dev/dev-team",
		},
		{
			name:      "uppercase name",
			groupName: "DevTeam",
			want:      "group.krkn.krkn-chaos.dev/devteam",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroupLabelKey(tt.groupName)
			if got != tt.want {
				t.Errorf("GroupLabelKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractGroupNamesFromLabels(t *testing.T) {
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
				"group.krkn.krkn-chaos.dev/dev-team":  "true",
				"group.krkn.krkn-chaos.dev/ops-team":  "true",
				"group.krkn.krkn-chaos.dev/prod-team": "true",
			},
			want: []string{"dev-team", "ops-team", "prod-team"},
		},
		{
			name: "mixed labels",
			labels: map[string]string{
				"group.krkn.krkn-chaos.dev/dev-team": "true",
				"krkn.krkn-chaos.dev/role":           "admin",
				"app.kubernetes.io/name":             "krkn-operator",
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
			name: "only non-group labels",
			labels: map[string]string{
				"krkn.krkn-chaos.dev/role": "admin",
				"app.kubernetes.io/name":   "krkn-operator",
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractGroupNamesFromLabels(tt.labels)

			// Convert slices to maps for comparison (order doesn't matter)
			gotMap := make(map[string]bool)
			for _, g := range got {
				gotMap[g] = true
			}
			wantMap := make(map[string]bool)
			for _, g := range tt.want {
				wantMap[g] = true
			}

			if len(gotMap) != len(wantMap) {
				t.Errorf("ExtractGroupNamesFromLabels() returned %d groups, want %d", len(got), len(tt.want))
				return
			}

			for g := range wantMap {
				if !gotMap[g] {
					t.Errorf("ExtractGroupNamesFromLabels() missing group %s", g)
				}
			}
		})
	}
}

func TestSanitizeGroupName(t *testing.T) {
	tests := []struct {
		name      string
		groupName string
		want      string
	}{
		{
			name:      "already valid",
			groupName: "dev-team",
			want:      "dev-team",
		},
		{
			name:      "with spaces",
			groupName: "dev team",
			want:      "dev-team",
		},
		{
			name:      "with special characters",
			groupName: "dev@team!",
			want:      "dev-team",
		},
		{
			name:      "uppercase",
			groupName: "DevTeam",
			want:      "devteam",
		},
		{
			name:      "leading and trailing hyphens",
			groupName: "-dev-team-",
			want:      "dev-team",
		},
		{
			name:      "leading and trailing underscores",
			groupName: "_dev_team_",
			want:      "dev_team",
		},
		{
			name:      "mixed separators",
			groupName: "dev.team_name",
			want:      "dev.team_name",
		},
		{
			name:      "very long name - no truncation",
			groupName: "this-is-a-very-long-group-name-that-exceeds-the-kubernetes-label-limit-of-sixty-three-characters",
			want:      "this-is-a-very-long-group-name-that-exceeds-the-kubernetes-label-limit-of-sixty-three-characters",
		},
		{
			name:      "long name with trailing hyphens",
			groupName: "this-is-a-very-long-group-name-that-exceeds-kubernetes-label--",
			want:      "this-is-a-very-long-group-name-that-exceeds-kubernetes-label",
		},
		{
			name:      "multiple consecutive special chars",
			groupName: "dev@@@team!!!",
			want:      "dev-team",
		},
		{
			name:      "dots and underscores valid",
			groupName: "dev.ops_team",
			want:      "dev.ops_team",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeGroupName(tt.groupName)
			if got != tt.want {
				t.Errorf("SanitizeGroupName() = %q, want %q", got, tt.want)
			}

			// Verify result doesn't have leading/trailing invalid characters
			if len(got) > 0 {
				firstChar := got[0]
				lastChar := got[len(got)-1]
				if firstChar == '-' || firstChar == '_' || firstChar == '.' {
					t.Errorf("SanitizeGroupName() result has invalid leading char: %c", firstChar)
				}
				if lastChar == '-' || lastChar == '_' || lastChar == '.' {
					t.Errorf("SanitizeGroupName() result has invalid trailing char: %c", lastChar)
				}
			}
		})
	}
}
