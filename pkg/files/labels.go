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
	"crypto/sha256"
	"encoding/hex"
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
	// FileIDLabel stores the UUID identifier for the file
	FileIDLabel = "files.krkn.krkn-chaos.dev/file-id"
	// FilePurposeLabel identifies the purpose/type of file (e.g., "workflow-template")
	FilePurposeLabel = "files.krkn.krkn-chaos.dev/file-purpose"
	// LogicalNameHashLabel stores a SHA256 prefix of the logical name for efficient server-side dedup queries
	LogicalNameHashLabel = "files.krkn.krkn-chaos.dev/logical-name-hash"

	// DescriptionAnnotation stores the file description
	DescriptionAnnotation = "files.krkn.krkn-chaos.dev/description"
	// WorkflowNameAnnotation stores the user-defined workflow name (for workflow templates)
	WorkflowNameAnnotation = "files.krkn.krkn-chaos.dev/workflow-name"
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

	// FilePurposeFile is the filePurpose value for generic user files
	FilePurposeFile = "file"
	// FilePurposeWorkflow is the filePurpose value for workflow graph templates
	FilePurposeWorkflow = "workflow-template"
	// FilePurposeResiliency is the filePurpose value for resiliency scoring metric definitions
	FilePurposeResiliency = "resiliency-score"
)

// HashLogicalName returns a truncated SHA256 hex digest of the logical name,
// safe for use as a Kubernetes label value (max 63 chars, RFC 1123).
func HashLogicalName(name string) string {
	h := sha256.Sum256([]byte(name))
	return hex.EncodeToString(h[:16])
}

// BuildFileLabels creates the labels map for a file ConfigMap
func BuildFileLabels(fileID, fileType string, groups []string, availableToAll bool, filePurpose, logicalName string) map[string]string {
	labels := map[string]string{
		AppNameLabel:      AppName,
		AppComponentLabel: ComponentFile,
		FileIDLabel:       fileID,
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

	// Add file purpose label if specified
	if filePurpose != "" {
		labels[FilePurposeLabel] = filePurpose
	}

	// Add logical name hash for efficient server-side dedup queries
	if logicalName != "" {
		labels[LogicalNameHashLabel] = HashLogicalName(logicalName)
	}

	return labels
}

// ExtractFileIDFromLabels extracts the file ID from file ConfigMap labels
func ExtractFileIDFromLabels(labels map[string]string) string {
	return labels[FileIDLabel]
}

// BuildFileAnnotations creates the annotations map for a file ConfigMap.
//
// Parameters:
//   - description: Optional human-readable description of the file
//   - createdBy: Email of the user creating the file (required, used for audit trail)
//   - workflowName: User-defined workflow name (only for workflow templates, empty for regular files)
//
// Annotations created:
//   - files.krkn.krkn-chaos.dev/created-by: Always set to createdBy
//   - files.krkn.krkn-chaos.dev/created-at: Always set to current UTC timestamp (RFC3339)
//   - files.krkn.krkn-chaos.dev/description: Set only if description is non-empty
//   - files.krkn.krkn-chaos.dev/workflow-name: Set only if workflowName is non-empty
//
// Example usage:
//
//	// Regular file (no workflow name)
//	annotations := BuildFileAnnotations(
//	    "Configuration for production",
//	    "admin@example.com",
//	    "", // empty for non-workflow files
//	)
//
//	// Workflow template
//	annotations := BuildFileAnnotations(
//	    "Pod chaos workflow",
//	    "admin@example.com",
//	    "My Pod Chaos Workflow", // workflow name for workflow templates
//	)
func BuildFileAnnotations(
	description string,
	createdBy string,
	workflowName string,
) map[string]string {
	annotations := map[string]string{
		CreatedByAnnotation: createdBy,
		CreatedAtAnnotation: time.Now().UTC().Format(time.RFC3339),
	}

	if description != "" {
		annotations[DescriptionAnnotation] = description
	}

	if workflowName != "" {
		annotations[WorkflowNameAnnotation] = workflowName
	}

	return annotations
}

// UpdateFileAnnotations updates the annotations for a file ConfigMap.
// workflowName pointer semantics:
//   - nil: field was omitted in request, preserve existing annotation
//   - non-nil empty string: explicitly set to empty, delete annotation
//   - non-nil non-empty: update annotation with new value
func UpdateFileAnnotations(
	existing map[string]string,
	description string,
	updatedBy string,
	workflowName *string,
) map[string]string {
	// Keep existing annotations and update specific ones
	updated := make(map[string]string)
	for k, v := range existing {
		updated[k] = v
	}

	updated[UpdatedByAnnotation] = updatedBy
	updated[UpdatedAtAnnotation] = time.Now().UTC().Format(time.RFC3339)

	if description != "" {
		updated[DescriptionAnnotation] = description
	} else {
		delete(updated, DescriptionAnnotation)
	}

	// Only update workflowName if explicitly provided (non-nil pointer)
	if workflowName != nil {
		if *workflowName != "" {
			updated[WorkflowNameAnnotation] = *workflowName
		} else {
			delete(updated, WorkflowNameAnnotation)
		}
	}
	// If workflowName is nil, preserve existing annotation (don't touch it)

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

// ExtractFilePurposeFromLabels extracts the file purpose from file ConfigMap labels
// Returns empty string if no file purpose label is found
func ExtractFilePurposeFromLabels(labels map[string]string) string {
	return labels[FilePurposeLabel]
}

// ValidFilePurposes returns all valid filePurpose values
func ValidFilePurposes() []string {
	return []string{FilePurposeFile, FilePurposeWorkflow, FilePurposeResiliency}
}

// IsValidFilePurpose checks if a filePurpose value is valid
func IsValidFilePurpose(purpose string) bool {
	for _, valid := range ValidFilePurposes() {
		if purpose == valid {
			return true
		}
	}
	return false
}
