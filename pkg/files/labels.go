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
	"strings"
	"time"

	"github.com/krkn-chaos/krkn-operator/pkg/filetypes"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

// Label and annotation keys for file ConfigMaps
const (
	// AppNameLabel is the standard app name label
	AppNameLabel = "app.kubernetes.io/name"
	// AppComponentLabel is the standard component label
	AppComponentLabel = "app.kubernetes.io/component"

	// AvailableToAllLabel marks files accessible by all users
	AvailableToAllLabel = "files.krkn.krkn-chaos.dev/available-to-all"

	// MountPathAnnotation stores the path where the file should be mounted
	MountPathAnnotation = "files.krkn.krkn-chaos.dev/mount-path"
	// DescriptionAnnotation stores the file description
	DescriptionAnnotation = "files.krkn.krkn-chaos.dev/description"
	// CreatedByAnnotation stores the email of the admin who created the file
	CreatedByAnnotation = "files.krkn.krkn-chaos.dev/created-by"
	// CreatedAtAnnotation stores the creation timestamp
	CreatedAtAnnotation = "files.krkn.krkn-chaos.dev/created-at"
	// UpdatedByAnnotation stores the email of the admin who last updated the file
	UpdatedByAnnotation = "files.krkn.krkn-chaos.dev/updated-by"
	// UpdatedAtAnnotation stores the last update timestamp
	UpdatedAtAnnotation = "files.krkn.krkn-chaos.dev/updated-at"

	// AppName is the value for AppNameLabel
	AppName = "krkn-operator"
	// ComponentFile is the value for AppComponentLabel
	ComponentFile = "file"
)

// BuildFileLabels creates the labels map for a file ConfigMap
func BuildFileLabels(fileType string, groups []string, availableToAll bool) map[string]string {
	labels := map[string]string{
		AppNameLabel:      AppName,
		AppComponentLabel: ComponentFile,
	}

	// Add file type label if specified
	if fileType != "" {
		typeLabel := filetypes.BuildFileTypeLabel(fileType)
		labels[typeLabel] = "true"
	}

	// Add available-to-all label if specified
	if availableToAll {
		labels[AvailableToAllLabel] = "true"
	}

	// Add group labels
	for _, groupName := range groups {
		groupLabel := groupauth.GroupLabelKey(groupName)
		labels[groupLabel] = "true"
	}

	return labels
}

// BuildFileAnnotations creates the annotations map for a file ConfigMap
func BuildFileAnnotations(
	mountPath string,
	description string,
	createdBy string,
) map[string]string {
	annotations := map[string]string{
		MountPathAnnotation: mountPath,
		CreatedByAnnotation: createdBy,
		CreatedAtAnnotation: time.Now().UTC().Format(time.RFC3339),
	}

	if description != "" {
		annotations[DescriptionAnnotation] = description
	}

	return annotations
}

// UpdateFileAnnotations updates the annotations for a file ConfigMap
func UpdateFileAnnotations(
	existing map[string]string,
	mountPath string,
	description string,
	updatedBy string,
) map[string]string {
	// Keep existing annotations and update specific ones
	updated := make(map[string]string)
	for k, v := range existing {
		updated[k] = v
	}

	updated[MountPathAnnotation] = mountPath
	updated[UpdatedByAnnotation] = updatedBy
	updated[UpdatedAtAnnotation] = time.Now().UTC().Format(time.RFC3339)

	if description != "" {
		updated[DescriptionAnnotation] = description
	} else {
		delete(updated, DescriptionAnnotation)
	}

	return updated
}

// ExtractGroupsFromLabels extracts group names from file ConfigMap labels
func ExtractGroupsFromLabels(labels map[string]string) []string {
	groups := []string{}

	for key, value := range labels {
		// Check if it's a group label with value "true"
		if strings.HasPrefix(key, groupauth.GroupLabelPrefix) && value == "true" {
			// Extract group name from label key
			groupName := strings.TrimPrefix(key, groupauth.GroupLabelPrefix)
			groups = append(groups, groupName)
		}
	}

	return groups
}

// ExtractFileTypeFromLabels extracts the file type from file ConfigMap labels
// Returns empty string if no file type label is found
func ExtractFileTypeFromLabels(labels map[string]string) string {
	return filetypes.ExtractFileTypeFromLabels(labels)
}
