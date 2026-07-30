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

package api

import (
	"testing"

	"github.com/krkn-chaos/krkn-operator/pkg/files"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Test for Bug Fix #1: buildFileInfo should exclude studioLayout.json
func TestBuildFileInfo_ExcludesStudioLayout(t *testing.T) {
	tests := []struct {
		name         string
		configMap    *corev1.ConfigMap
		wantFileName string
	}{
		{
			name: "workflow with studioLayout - should return workflow.json",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-workflow",
					Labels: map[string]string{
						files.FileIDLabel: "test-123",
					},
				},
				Data: map[string]string{
					"workflow.json":     `{"node1": {}}`,
					"studioLayout.json": `{"positions": {}}`,
				},
			},
			wantFileName: "workflow.json",
		},
		{
			name: "file without studioLayout - should return filename",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-file",
					Labels: map[string]string{
						files.FileIDLabel: "test-456",
					},
				},
				Data: map[string]string{
					"config.yaml": `key: value`,
				},
			},
			wantFileName: "config.yaml",
		},
		{
			name: "only studioLayout - should return empty",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-broken",
					Labels: map[string]string{
						files.FileIDLabel: "test-789",
					},
				},
				Data: map[string]string{
					"studioLayout.json": `{"positions": {}}`,
				},
			},
			wantFileName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileInfo := buildFileInfo(tt.configMap)
			if fileInfo.FileName != tt.wantFileName {
				t.Errorf("buildFileInfo() fileName = %v, want %v", fileInfo.FileName, tt.wantFileName)
			}
		})
	}
}

// Test for Bug Fix #2: WorkflowName fallback for backwards compatibility
func TestBuildFileResponse_WorkflowNameFallback(t *testing.T) {
	tests := []struct {
		name             string
		configMap        *corev1.ConfigMap
		wantWorkflowName string
	}{
		{
			name: "workflow with annotation - use annotation",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-workflow",
					Labels: map[string]string{
						files.FileIDLabel:      "test-123",
						files.FilePurposeLabel: "workflow-template",
					},
					Annotations: map[string]string{
						files.WorkflowNameAnnotation: "My Custom Workflow",
					},
				},
				Data: map[string]string{
					"workflow.json": `{"node1": {}}`,
				},
			},
			wantWorkflowName: "My Custom Workflow",
		},
		{
			name: "workflow without annotation - fallback to fileName",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-workflow-old",
					Labels: map[string]string{
						files.FileIDLabel:      "test-456",
						files.FilePurposeLabel: "workflow-template",
					},
					Annotations: map[string]string{},
				},
				Data: map[string]string{
					"workflow.json": `{"node1": {}}`,
				},
			},
			wantWorkflowName: "workflow.json",
		},
		{
			name: "regular file without annotation - no fallback",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-file",
					Labels: map[string]string{
						files.FileIDLabel: "test-789",
					},
					Annotations: map[string]string{},
				},
				Data: map[string]string{
					"config.yaml": `key: value`,
				},
			},
			wantWorkflowName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileResp := buildFileResponse(tt.configMap)
			if fileResp.WorkflowName != tt.wantWorkflowName {
				t.Errorf("buildFileResponse() WorkflowName = %v, want %v", fileResp.WorkflowName, tt.wantWorkflowName)
			}
		})
	}
}

// Test for Bug Fix #3: UpdateFileAnnotations pointer semantics
func TestUpdateFileAnnotations_PointerSemantics(t *testing.T) {
	existingAnnotations := map[string]string{
		files.WorkflowNameAnnotation: "Existing Workflow",
		files.DescriptionAnnotation:  "Existing Description",
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
			updated := files.UpdateFileAnnotations(
				existingAnnotations,
				"Test description",
				"test-user@example.com",
				tt.workflowName,
			)

			if tt.wantExists {
				if got, exists := updated[files.WorkflowNameAnnotation]; !exists {
					t.Errorf("UpdateFileAnnotations() workflowName annotation missing, want %v", tt.wantWorkflowName)
				} else if got != tt.wantWorkflowName {
					t.Errorf("UpdateFileAnnotations() workflowName = %v, want %v", got, tt.wantWorkflowName)
				}
			} else {
				if _, exists := updated[files.WorkflowNameAnnotation]; exists {
					t.Errorf("UpdateFileAnnotations() workflowName annotation should be deleted")
				}
			}
		})
	}
}
