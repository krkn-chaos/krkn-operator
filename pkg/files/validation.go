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
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// ValidateFileContent validates that content is valid JSON or YAML.
// Content must be a structured format (object/map or array/list), not just plain text.
// Returns an error if content is neither valid JSON nor valid YAML.
func ValidateFileContent(content string) error {
	if content == "" {
		return fmt.Errorf("content cannot be empty")
	}

	// Try JSON first (must be object or array, not primitive)
	var jsonData interface{}
	if err := json.Unmarshal([]byte(content), &jsonData); err == nil {
		// Valid JSON - check it's structured (not just a string/number/bool)
		switch jsonData.(type) {
		case map[string]interface{}, []interface{}:
			return nil
		default:
			// Primitive JSON value (string, number, bool) - not accepted
			return fmt.Errorf("content must be a JSON object or array, not a primitive value")
		}
	}

	// Try YAML (must be map or list, not scalar)
	var yamlData interface{}
	if err := yaml.Unmarshal([]byte(content), &yamlData); err == nil {
		// Valid YAML - check it's structured (not just a scalar string)
		switch yamlData.(type) {
		case map[string]interface{}, []interface{}:
			return nil
		default:
			// Scalar YAML value (plain text) - not accepted
			return fmt.Errorf("content must be a YAML map or list, not plain text")
		}
	}

	return fmt.Errorf("content must be valid JSON or YAML format")
}

// ValidateFileGroups validates file group assignment rules.
// - Maximum 1 group per file (1:1 relationship)
// - Cannot have both groups and availableToAll=true
func ValidateFileGroups(groups []string, availableToAll bool) error {
	// Check 1:1 relationship (max 1 group)
	if len(groups) > 1 {
		return fmt.Errorf("file can be assigned to at most 1 group, got %d groups", len(groups))
	}

	// Check mutual exclusivity
	if len(groups) > 0 && availableToAll {
		return fmt.Errorf("file cannot be both assigned to a group and available to all")
	}

	return nil
}

// ValidateUserFilePermissions validates file permissions based on user role and group membership.
// - Admin: can assign files to any group or make public
// - User: can assign files to their own group or make public
// - userGroups: list of groups the current user belongs to
func ValidateUserFilePermissions(isAdmin bool, groups []string, availableToAll bool, userGroups []string) error {
	if isAdmin {
		// Admin can do anything (subject to ValidateFileGroups constraints)
		return nil
	}

	// Non-admin users can create public files OR files for their own group
	if len(groups) == 0 {
		// No group assignment - must be public
		if !availableToAll {
			return fmt.Errorf("file must be either public or assigned to your group")
		}
		return nil
	}

	// User is assigning to a group - check they belong to it
	if len(groups) > 0 {
		assignedGroup := groups[0] // We know it's max 1 from ValidateFileGroups
		userBelongsToGroup := false
		for _, userGroup := range userGroups {
			if userGroup == assignedGroup {
				userBelongsToGroup = true
				break
			}
		}

		if !userBelongsToGroup {
			return fmt.Errorf("you can only assign files to your own group (you belong to: %v)", userGroups)
		}

		// If assigning to group, must not be public
		if availableToAll {
			return fmt.Errorf("file cannot be both assigned to a group and available to all")
		}
	}

	return nil
}
