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
)

func TestValidateFileContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "valid JSON",
			content: `{"key": "value", "number": 123}`,
			wantErr: false,
		},
		{
			name:    "valid YAML",
			content: "key: value\nnumber: 123\n",
			wantErr: false,
		},
		{
			name:    "valid JSON array",
			content: `[1, 2, 3]`,
			wantErr: false,
		},
		{
			name:    "valid YAML list",
			content: "- item1\n- item2\n- item3\n",
			wantErr: false,
		},
		{
			name:    "complex YAML",
			content: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\ndata:\n  key: value\n",
			wantErr: false,
		},
		{
			name:    "empty string",
			content: "",
			wantErr: true,
		},
		{
			name:    "invalid content - plain text",
			content: "This is just plain text, not JSON or YAML",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			content: `{"key": "value"`,
			wantErr: true,
		},
		{
			name:    "script content",
			content: "#!/bin/bash\necho 'hello'",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileContent(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFileContent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateFileGroups(t *testing.T) {
	tests := []struct {
		name           string
		groups         []string
		availableToAll bool
		wantErr        bool
		errContains    string
	}{
		{
			name:           "no groups, not public",
			groups:         []string{},
			availableToAll: false,
			wantErr:        false,
		},
		{
			name:           "no groups, public",
			groups:         []string{},
			availableToAll: true,
			wantErr:        false,
		},
		{
			name:           "single group, not public",
			groups:         []string{"team-a"},
			availableToAll: false,
			wantErr:        false,
		},
		{
			name:           "multiple groups (violates 1:1)",
			groups:         []string{"team-a", "team-b"},
			availableToAll: false,
			wantErr:        true,
			errContains:    "at most 1 group",
		},
		{
			name:           "group AND public (mutually exclusive)",
			groups:         []string{"team-a"},
			availableToAll: true,
			wantErr:        true,
			errContains:    "cannot be both",
		},
		{
			name:           "three groups (violates 1:1)",
			groups:         []string{"team-a", "team-b", "team-c"},
			availableToAll: false,
			wantErr:        true,
			errContains:    "at most 1 group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileGroups(tt.groups, tt.availableToAll)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFileGroups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateFileGroups() error = %v, should contain %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestValidateUserFilePermissions(t *testing.T) {
	tests := []struct {
		name           string
		isAdmin        bool
		groups         []string
		availableToAll bool
		userGroups     []string
		wantErr        bool
		errContains    string
	}{
		{
			name:           "admin - public file",
			isAdmin:        true,
			groups:         []string{},
			availableToAll: true,
			userGroups:     []string{},
			wantErr:        false,
		},
		{
			name:           "admin - group file (any group)",
			isAdmin:        true,
			groups:         []string{"team-a"},
			availableToAll: false,
			userGroups:     []string{"team-b"}, // Admin can assign to any group
			wantErr:        false,
		},
		{
			name:           "admin - private file (no groups, not public)",
			isAdmin:        true,
			groups:         []string{},
			availableToAll: false,
			userGroups:     []string{},
			wantErr:        false,
		},
		{
			name:           "user - public file",
			isAdmin:        false,
			groups:         []string{},
			availableToAll: true,
			userGroups:     []string{"team-a"},
			wantErr:        false,
		},
		{
			name:           "user - own group file",
			isAdmin:        false,
			groups:         []string{"team-a"},
			availableToAll: false,
			userGroups:     []string{"team-a"},
			wantErr:        false,
		},
		{
			name:           "user - other group file (forbidden)",
			isAdmin:        false,
			groups:         []string{"team-b"},
			availableToAll: false,
			userGroups:     []string{"team-a"},
			wantErr:        true,
			errContains:    "you can only assign files to your own group",
		},
		{
			name:           "user - private file (no group, not public - forbidden)",
			isAdmin:        false,
			groups:         []string{},
			availableToAll: false,
			userGroups:     []string{"team-a"},
			wantErr:        true,
			errContains:    "file must be either public or assigned to your group",
		},
		{
			name:           "user - group AND public (mutually exclusive)",
			isAdmin:        false,
			groups:         []string{"team-a"},
			availableToAll: true,
			userGroups:     []string{"team-a"},
			wantErr:        true,
			errContains:    "cannot be both",
		},
		{
			name:           "user - no groups, multiple options",
			isAdmin:        false,
			groups:         []string{},
			availableToAll: true,
			userGroups:     []string{"team-a", "team-b"},
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserFilePermissions(tt.isAdmin, tt.groups, tt.availableToAll, tt.userGroups)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserFilePermissions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateUserFilePermissions() error = %v, should contain %q", err, tt.errContains)
				}
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
