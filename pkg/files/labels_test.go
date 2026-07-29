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

package files

import (
	"testing"
)

// TestUpdateFileAnnotations_PointerSemantics tests that workflowName pointer
// correctly distinguishes between omitted (nil), explicitly empty (""), and set values
func TestUpdateFileAnnotations_PointerSemantics(t *testing.T) {
	existingAnnotations := map[string]string{
		WorkflowNameAnnotation: "Existing Workflow",
		DescriptionAnnotation:  "Existing Description",
		CreatedByAnnotation:    "original@example.com",
		CreatedAtAnnotation:    "2026-01-01T00:00:00Z",
	}

	tests := []struct {
		name             string
		workflowName     *string
		wantWorkflowName string
		wantExists       bool
	}{
		{
			name:             "nil pointer - preserve existing annotation",
			workflowName:     nil,
			wantWorkflowName: "Existing Workflow",
			wantExists:       true,
		},
		{
			name:             "empty string pointer - delete annotation",
			workflowName:     stringPtr(""),
			wantWorkflowName: "",
			wantExists:       false,
		},
		{
			name:             "non-empty string pointer - update annotation",
			workflowName:     stringPtr("Updated Workflow"),
			wantWorkflowName: "Updated Workflow",
			wantExists:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of existing annotations for each test
			existing := make(map[string]string)
			for k, v := range existingAnnotations {
				existing[k] = v
			}

			updated := UpdateFileAnnotations(
				existing,
				"Test description",
				"test-user@example.com",
				tt.workflowName,
			)

			if tt.wantExists {
				if got, exists := updated[WorkflowNameAnnotation]; !exists {
					t.Errorf("UpdateFileAnnotations() workflowName annotation missing, want %v", tt.wantWorkflowName)
				} else if got != tt.wantWorkflowName {
					t.Errorf("UpdateFileAnnotations() workflowName = %v, want %v", got, tt.wantWorkflowName)
				}
			} else {
				if _, exists := updated[WorkflowNameAnnotation]; exists {
					t.Errorf("UpdateFileAnnotations() workflowName annotation should be deleted")
				}
			}

			// Verify other required annotations are set
			if _, exists := updated[UpdatedByAnnotation]; !exists {
				t.Error("UpdateFileAnnotations() UpdatedByAnnotation missing")
			}
			if _, exists := updated[UpdatedAtAnnotation]; !exists {
				t.Error("UpdateFileAnnotations() UpdatedAtAnnotation missing")
			}
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
