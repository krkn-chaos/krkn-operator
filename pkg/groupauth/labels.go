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
	"regexp"
	"strings"
)

// GroupLabelKey returns the label key for a group name
// Example: "dev-team" -> "group.krkn.krkn-chaos.dev/dev-team"
func GroupLabelKey(groupName string) string {
	sanitized := SanitizeGroupName(groupName)
	return GroupLabelPrefix + sanitized
}

// ExtractGroupNamesFromLabels extracts group names from KrknUser labels
// Returns a list of group names the user belongs to
func ExtractGroupNamesFromLabels(labels map[string]string) []string {
	groups := make([]string, 0)

	for key, value := range labels {
		if strings.HasPrefix(key, GroupLabelPrefix) && value == "true" {
			groupName := strings.TrimPrefix(key, GroupLabelPrefix)
			groups = append(groups, groupName)
		}
	}

	return groups
}

// SanitizeGroupName sanitizes a group name to be valid as a Kubernetes label name
// - Replaces invalid characters with hyphens
// - Converts to lowercase
// - Ensures it starts/ends with alphanumeric
// - Truncates to 63 characters max
func SanitizeGroupName(groupName string) string {
	// Replace invalid characters with hyphens
	sanitized := regexp.MustCompile(`[^a-zA-Z0-9\-_.]+`).ReplaceAllString(groupName, "-")

	// Trim leading/trailing hyphens, underscores, and dots
	sanitized = strings.Trim(sanitized, "-_.")

	// Convert to lowercase
	sanitized = strings.ToLower(sanitized)

	// Truncate to 63 characters (Kubernetes label name limit)
	if len(sanitized) > 63 {
		sanitized = sanitized[:63]
		// Re-trim in case truncation added trailing invalid chars
		sanitized = strings.TrimRight(sanitized, "-_.")
	}

	return sanitized
}
