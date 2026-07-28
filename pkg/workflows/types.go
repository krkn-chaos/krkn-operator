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

// Package workflows provides types and utilities for workflow template management.
package workflows

import (
	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// CreateWorkflowRequest represents a request to save a new workflow template.
type CreateWorkflowRequest struct {
	// WorkflowName is a human-readable name for the workflow
	WorkflowName string `json:"workflowName"`
	// Description is an optional description of the workflow
	Description string `json:"description,omitempty"`
	// Graph is the workflow graph definition (map of node ID to GraphScenarioNode)
	Graph map[string]krknv1alpha1.GraphScenarioNode `json:"graph"`
	// FileType is an optional user-defined category (e.g., "pod-chaos", "network-chaos")
	FileType string `json:"fileType,omitempty"`
	// Groups is a list of group names that can access this workflow (max 1)
	Groups []string `json:"groups,omitempty"`
	// AvailableToAll makes the workflow accessible to all users
	AvailableToAll bool `json:"availableToAll,omitempty"`
}

// UpdateWorkflowRequest represents a request to update an existing workflow template.
type UpdateWorkflowRequest struct {
	// WorkflowName is a human-readable name for the workflow
	WorkflowName string `json:"workflowName"`
	// Description is an optional description of the workflow
	Description string `json:"description,omitempty"`
	// Graph is the workflow graph definition (map of node ID to GraphScenarioNode)
	Graph map[string]krknv1alpha1.GraphScenarioNode `json:"graph"`
	// FileType is an optional user-defined category (e.g., "pod-chaos", "network-chaos")
	FileType string `json:"fileType,omitempty"`
	// Groups is a list of group names that can access this workflow (max 1)
	Groups []string `json:"groups,omitempty"`
	// AvailableToAll makes the workflow accessible to all users
	AvailableToAll bool `json:"availableToAll,omitempty"`
}

// WorkflowResponse represents a workflow template in API responses.
type WorkflowResponse struct {
	// WorkflowID is the UUID identifier for this workflow
	WorkflowID string `json:"workflowId"`
	// WorkflowName is a human-readable name for the workflow
	WorkflowName string `json:"workflowName"`
	// Description is an optional description of the workflow
	Description string `json:"description,omitempty"`
	// Graph is the workflow graph definition
	Graph map[string]krknv1alpha1.GraphScenarioNode `json:"graph"`
	// FileType is the user-defined category
	FileType string `json:"fileType,omitempty"`
	// Groups is a list of group names that can access this workflow
	Groups []string `json:"groups,omitempty"`
	// AvailableToAll indicates if the workflow is accessible to all users
	AvailableToAll bool `json:"availableToAll"`
	// CreatedAt is the timestamp when the workflow was created
	CreatedAt string `json:"createdAt,omitempty"`
	// CreatedBy is the email of the user who created the workflow
	CreatedBy string `json:"createdBy,omitempty"`
	// UpdatedAt is the timestamp when the workflow was last updated
	UpdatedAt string `json:"updatedAt,omitempty"`
	// UpdatedBy is the email of the user who last updated the workflow
	UpdatedBy string `json:"updatedBy,omitempty"`
}

// WorkflowInfo represents minimal workflow information for user-facing lists.
type WorkflowInfo struct {
	// WorkflowID is the UUID identifier for this workflow
	WorkflowID string `json:"workflowId"`
	// WorkflowName is a human-readable name for the workflow
	WorkflowName string `json:"workflowName"`
	// Description is an optional description of the workflow
	Description string `json:"description,omitempty"`
	// FileType is the user-defined category
	FileType string `json:"fileType,omitempty"`
	// NodeCount is the number of nodes in the workflow graph
	NodeCount int `json:"nodeCount"`
}

// ListWorkflowsResponse is the response for list workflows requests (admin only).
type ListWorkflowsResponse struct {
	// Workflows is the list of workflow templates
	Workflows []WorkflowResponse `json:"workflows"`
	// Total is the total number of workflows returned
	Total int `json:"total"`
}

// AvailableWorkflowsResponse is the response for available workflows requests.
type AvailableWorkflowsResponse struct {
	// Workflows is the list of workflows available to the current user
	Workflows []WorkflowInfo `json:"workflows"`
}

// CreateWorkflowResponse is the response for create workflow requests.
type CreateWorkflowResponse struct {
	// Message is a human-readable status message
	Message string `json:"message"`
	// WorkflowID is the generated UUID for this workflow
	WorkflowID string `json:"workflowId"`
}

// UpdateWorkflowResponse is the response for update workflow requests.
type UpdateWorkflowResponse struct {
	// Message is a human-readable status message
	Message string `json:"message"`
	// WorkflowID is the UUID for this workflow
	WorkflowID string `json:"workflowId"`
}

// DeleteWorkflowResponse is the response for delete workflow requests.
type DeleteWorkflowResponse struct {
	// Message is a human-readable status message
	Message string `json:"message"`
}
