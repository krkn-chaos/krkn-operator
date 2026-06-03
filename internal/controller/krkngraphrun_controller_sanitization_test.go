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

package controller

import (
	"strings"
	"testing"
)

func TestSanitizeNodeID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		verify   func(t *testing.T, result string)
	}{
		{
			name:     "valid lowercase alphanumeric",
			input:    "node1",
			expected: "node1",
		},
		{
			name:     "uppercase converted to lowercase",
			input:    "NodeOne",
			expected: "nodeone",
		},
		{
			name:     "valid with hyphens",
			input:    "node-one",
			expected: "node-one",
		},
		{
			name:     "valid with underscores",
			input:    "node_one",
			expected: "node_one",
		},
		{
			name:     "valid with dots",
			input:    "node.one",
			expected: "node.one",
		},
		{
			name:     "invalid characters replaced with hyphens",
			input:    "node@one#two",
			expected: "node-one-two",
		},
		{
			name:     "spaces replaced with hyphens",
			input:    "node one two",
			expected: "node-one-two",
		},
		{
			name:  "truncated to 63 characters",
			input: "very-long-node-id-that-exceeds-the-maximum-kubernetes-label-value-length-of-sixty-three-characters",
			verify: func(t *testing.T, result string) {
				if len(result) > MaxLabelValueLength {
					t.Errorf("Expected length <= %d, got %d", MaxLabelValueLength, len(result))
				}
			},
		},
		{
			name:     "starts with hyphen trimmed",
			input:    "-node-one",
			expected: "node-one",
		},
		{
			name:     "ends with hyphen trimmed",
			input:    "node-one-",
			expected: "node-one",
		},
		{
			name:     "starts and ends with invalid chars trimmed",
			input:    "---node---",
			expected: "node",
		},
		{
			name:     "empty string returns default",
			input:    "",
			expected: "empty",
		},
		{
			name:     "only invalid characters returns default",
			input:    "---",
			expected: "node",
		},
		{
			name:     "mixed case with special characters",
			input:    "Node@One#Two",
			expected: "node-one-two",
		},
		{
			name:     "kubernetes-like node name",
			input:    "scenario.execution.pod-disruption",
			expected: "scenario.execution.pod-disruption",
		},
		{
			name:  "verify alphanumeric start and end after truncation",
			input: strings.Repeat("a", 60) + "---",
			verify: func(t *testing.T, result string) {
				if len(result) == 0 {
					t.Error("Result should not be empty")
					return
				}
				first := rune(result[0])
				last := rune(result[len(result)-1])
				if !isAlphanumeric(first) {
					t.Errorf("Result must start with alphanumeric, got: %c", first)
				}
				if !isAlphanumeric(last) {
					t.Errorf("Result must end with alphanumeric, got: %c", last)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeNodeID(tt.input)

			// Run custom verification if provided
			if tt.verify != nil {
				tt.verify(t, result)
				return
			}

			// Otherwise check exact match
			if result != tt.expected {
				t.Errorf("sanitizeNodeID(%q) = %q, expected %q", tt.input, result, tt.expected)
			}

			// Verify result meets Kubernetes label requirements
			if len(result) > MaxLabelValueLength {
				t.Errorf("Result length %d exceeds max %d", len(result), MaxLabelValueLength)
			}

			if len(result) > 0 {
				// Must start and end with alphanumeric
				first := rune(result[0])
				last := rune(result[len(result)-1])
				if !isAlphanumeric(first) {
					t.Errorf("Result must start with alphanumeric, got: %c in %q", first, result)
				}
				if !isAlphanumeric(last) {
					t.Errorf("Result must end with alphanumeric, got: %c in %q", last, result)
				}

				// All characters must be valid
				for i, r := range result {
					if !isValidLabelChar(r) {
						t.Errorf("Invalid character at position %d: %c in %q", i, r, result)
					}
				}
			}
		})
	}
}

// Helper function to check if a rune is alphanumeric
func isAlphanumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// Helper function to check if a rune is valid in a Kubernetes label value
func isValidLabelChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.'
}
