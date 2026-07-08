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

package filetypes

import (
	"fmt"
	"strings"
)

// Label constants for file type management
const (
	// FileTypeLabelPrefix is the prefix for file type labels on ConfigMaps
	// Example: file-type.krkn.krkn-chaos.dev/config=true
	FileTypeLabelPrefix = "file-type.krkn.krkn-chaos.dev/"
)

// BuildFileTypeLabel creates a label key for a file type.
// The type name is sanitized to ensure it's a valid Kubernetes label.
//
// Example:
//
//	BuildFileTypeLabel("config") -> "file-type.krkn.krkn-chaos.dev/config"
//	BuildFileTypeLabel("my-scripts") -> "file-type.krkn.krkn-chaos.dev/my-scripts"
func BuildFileTypeLabel(typeName string) string {
	sanitized := sanitizeTypeName(typeName)
	return FileTypeLabelPrefix + sanitized
}

// ExtractFileTypeFromLabels extracts the file type name from ConfigMap labels.
// Returns empty string if no file type label is found.
//
// This function looks for labels with the FileTypeLabelPrefix and value "true".
// Only one file type label should exist per ConfigMap (enforced by ValidateSingleFileType).
func ExtractFileTypeFromLabels(labels map[string]string) string {
	for key, value := range labels {
		if strings.HasPrefix(key, FileTypeLabelPrefix) && value == "true" {
			// Extract type name from label key
			return strings.TrimPrefix(key, FileTypeLabelPrefix)
		}
	}
	return ""
}

// ValidateSingleFileType ensures only one file-type label exists in the label set.
// Returns an error if multiple file type labels are found.
//
// This enforces the business rule that each file can have only one type.
func ValidateSingleFileType(labels map[string]string) error {
	count := 0
	var foundTypes []string

	for key := range labels {
		if strings.HasPrefix(key, FileTypeLabelPrefix) {
			count++
			typeName := strings.TrimPrefix(key, FileTypeLabelPrefix)
			foundTypes = append(foundTypes, typeName)
		}
	}

	if count > 1 {
		return fmt.Errorf("file can have only one type, found %d: %v", count, foundTypes)
	}

	return nil
}

// sanitizeTypeName sanitizes a file type name to be a valid Kubernetes label value.
// Kubernetes labels must:
// - Be 63 characters or less
// - Contain only alphanumeric characters, dashes, underscores, and dots
// - Start and end with an alphanumeric character
func sanitizeTypeName(typeName string) string {
	// Convert to lowercase for consistency
	sanitized := strings.ToLower(typeName)

	// Replace invalid characters with dashes
	sanitized = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '-'
	}, sanitized)

	// Trim leading/trailing dashes
	sanitized = strings.Trim(sanitized, "-")

	// Ensure max length of 63 characters (Kubernetes label limit)
	if len(sanitized) > 63 {
		sanitized = sanitized[:63]
		// Trim any trailing dash that may have been introduced by truncation
		sanitized = strings.TrimSuffix(sanitized, "-")
	}

	return sanitized
}
