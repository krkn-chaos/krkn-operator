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

package v1alpha1

import (
	"fmt"
	"strings"
)

const (
	// MaxNodeIDLength is the maximum recommended length for node IDs
	MaxNodeIDLength = 253
	// MaxLabelValueLength is the maximum length for Kubernetes label values (RFC 1123)
	MaxLabelValueLength = 63
)

// ValidateGraph validates that node IDs in the graph meet Kubernetes naming requirements
// and that there are no collisions after sanitization.
//
// Returns an error if validation fails, nil otherwise.
func (r *KrknGraphRun) ValidateGraph() error {
	if len(r.Spec.Graph) == 0 {
		return fmt.Errorf("graph cannot be empty")
	}

	// Track sanitized node IDs to detect collisions
	sanitizedIDs := make(map[string][]string) // sanitized -> original IDs
	var warnings []string

	for nodeID := range r.Spec.Graph {
		// Skip metadata nodes (starting with '_')
		if strings.HasPrefix(nodeID, "_") {
			continue
		}

		// Validate node ID length
		if len(nodeID) > MaxNodeIDLength {
			return fmt.Errorf("node ID '%s' exceeds maximum length of %d characters (current: %d)", nodeID, MaxNodeIDLength, len(nodeID))
		}

		// Validate node ID is not empty
		if nodeID == "" {
			return fmt.Errorf("node ID cannot be empty")
		}

		// Sanitize the node ID to see what it would become
		sanitized := SanitizeNodeIDForKubernetes(nodeID)

		// Validate that sanitization doesn't produce an empty or invalid result
		if sanitized == "" || sanitized == "empty" || sanitized == "node" {
			return fmt.Errorf("node ID '%s' sanitizes to invalid value '%s'; use alphanumeric characters, hyphens, underscores, or dots", nodeID, sanitized)
		}

		// Track for collision detection
		sanitizedIDs[sanitized] = append(sanitizedIDs[sanitized], nodeID)

		// Collect warnings for node IDs that will be modified
		if sanitized != nodeID {
			warnings = append(warnings, fmt.Sprintf("node ID '%s' will be sanitized to '%s'", nodeID, sanitized))
		}

		// Warn if close to length limit
		if len(nodeID) > MaxLabelValueLength {
			warnings = append(warnings, fmt.Sprintf("node ID '%s' exceeds recommended length of %d characters and will be truncated to '%s'", nodeID, MaxLabelValueLength, sanitized))
		}
	}

	// Check for collisions (multiple original IDs map to same sanitized ID)
	for sanitized, originals := range sanitizedIDs {
		if len(originals) > 1 {
			return fmt.Errorf("node ID collision detected: %v all sanitize to '%s'; use distinct node IDs that remain unique after lowercase conversion and special character replacement", originals, sanitized)
		}
	}

	// Validate that all DependsOn references exist in the graph
	// This prevents "phantom nodes" that would be created in resolved levels
	// but don't exist in spec.graph, causing controller failures
	for nodeID, node := range r.Spec.Graph {
		// Skip metadata nodes
		if strings.HasPrefix(nodeID, "_") {
			continue
		}

		if node.DependsOn != nil && *node.DependsOn != "" {
			dependsOnID := *node.DependsOn

			// Check if the referenced node exists
			if _, exists := r.Spec.Graph[dependsOnID]; !exists {
				return fmt.Errorf("node '%s' has invalid DependsOn reference '%s': referenced node does not exist in graph", nodeID, dependsOnID)
			}

			// Check if the referenced node is a metadata node (invalid dependency)
			if strings.HasPrefix(dependsOnID, "_") {
				return fmt.Errorf("node '%s' has invalid DependsOn reference '%s': cannot depend on metadata nodes (starting with '_')", nodeID, dependsOnID)
			}

			// Check for self-reference
			if dependsOnID == nodeID {
				return fmt.Errorf("node '%s' has invalid DependsOn reference: cannot depend on itself", nodeID)
			}
		}
	}

	// Return warnings as part of error message if there are any
	// (in a real webhook, these would be admission.Warnings)
	if len(warnings) > 0 {
		// Just log warnings, don't fail validation
		// In the future, these could be returned as webhook warnings
	}

	return nil
}

// SanitizeNodeIDForKubernetes sanitizes a node ID for use in Kubernetes resource names and labels.
// It replaces invalid characters with hyphens, converts to lowercase, truncates to 63 characters,
// and ensures it starts and ends with an alphanumeric character.
//
// Kubernetes label value requirements (RFC 1123):
// - Maximum 63 characters
// - Only lowercase alphanumeric characters, '-', '_', or '.'
// - Must start and end with an alphanumeric character
func SanitizeNodeIDForKubernetes(nodeID string) string {
	if nodeID == "" {
		return "empty"
	}

	// Convert to lowercase
	sanitized := strings.ToLower(nodeID)

	// Replace invalid characters with hyphens
	var builder strings.Builder
	builder.Grow(len(sanitized))
	for _, r := range sanitized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			builder.WriteRune(r)
		} else {
			builder.WriteRune('-')
		}
	}
	sanitized = builder.String()

	// Truncate to max label value length
	if len(sanitized) > MaxLabelValueLength {
		sanitized = sanitized[:MaxLabelValueLength]
	}

	// Ensure it starts with alphanumeric
	sanitized = strings.TrimLeft(sanitized, "-_.")
	if sanitized == "" {
		return "node"
	}

	// Ensure it ends with alphanumeric
	sanitized = strings.TrimRight(sanitized, "-_.")
	if sanitized == "" {
		return "node"
	}

	return sanitized
}
