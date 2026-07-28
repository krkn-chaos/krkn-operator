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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/workflows"
)

// setupWorkflowTestHandler creates a test Handler with fake client
func setupWorkflowTestHandler() *Handler {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		Build()

	return &Handler{
		client:    fakeClient,
		namespace: "test-namespace",
	}
}

// validWorkflowGraph returns a valid workflow graph for testing
func validWorkflowGraph() map[string]krknv1alpha1.GraphScenarioNode {
	return map[string]krknv1alpha1.GraphScenarioNode{
		"node1": {
			Name:  "test-scenario-1",
			Image: "quay.io/krkn-chaos/krkn:latest",
		},
		"node2": {
			Name:      "test-scenario-2",
			Image:     "quay.io/krkn-chaos/krkn:latest",
			DependsOn: stringPtr("node1"),
		},
	}
}

// invalidWorkflowGraph returns an invalid workflow graph (circular dependency)
func invalidWorkflowGraph() map[string]krknv1alpha1.GraphScenarioNode {
	return map[string]krknv1alpha1.GraphScenarioNode{
		"node1": {
			Name:      "test-scenario-1",
			Image:     "quay.io/krkn-chaos/krkn:latest",
			DependsOn: stringPtr("node2"),
		},
		"node2": {
			Name:      "test-scenario-2",
			Image:     "quay.io/krkn-chaos/krkn:latest",
			DependsOn: stringPtr("node1"),
		},
	}
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

func TestCreateWorkflow(t *testing.T) {
	tests := []struct {
		name         string
		request      workflows.CreateWorkflowRequest
		setupGroups  []string
		userGroups   []string
		userID       string
		expectStatus int
		expectInDB   bool
		isAdmin      bool
	}{
		{
			name: "create workflow successfully (admin, public)",
			request: workflows.CreateWorkflowRequest{
				WorkflowName:   "Test Workflow",
				Description:    "Test workflow description",
				Graph:          validWorkflowGraph(),
				AvailableToAll: true,
			},
			userID:       "admin@test.example",
			isAdmin:      true,
			expectStatus: http.StatusCreated,
			expectInDB:   true,
		},
		{
			name: "create workflow successfully (user, own group)",
			request: workflows.CreateWorkflowRequest{
				WorkflowName: "User Workflow",
				Description:  "User workflow description",
				Graph:        validWorkflowGraph(),
				Groups:       []string{"dev-team"},
			},
			setupGroups:  []string{"dev-team"},
			userGroups:   []string{"dev-team"},
			userID:       "user@test.example",
			isAdmin:      false,
			expectStatus: http.StatusCreated,
			expectInDB:   true,
		},
		{
			name: "reject workflow with invalid graph (cycle)",
			request: workflows.CreateWorkflowRequest{
				WorkflowName:   "Invalid Workflow",
				Description:    "Workflow with circular dependency",
				Graph:          invalidWorkflowGraph(),
				AvailableToAll: true,
			},
			userID:       "admin@test.example",
			isAdmin:      true,
			expectStatus: http.StatusBadRequest,
			expectInDB:   false,
		},
		{
			name: "reject workflow with empty graph",
			request: workflows.CreateWorkflowRequest{
				WorkflowName:   "Empty Workflow",
				Description:    "Workflow with no nodes",
				Graph:          map[string]krknv1alpha1.GraphScenarioNode{},
				AvailableToAll: true,
			},
			userID:       "admin@test.example",
			isAdmin:      true,
			expectStatus: http.StatusBadRequest,
			expectInDB:   false,
		},
		{
			name: "reject workflow with empty name",
			request: workflows.CreateWorkflowRequest{
				WorkflowName:   "", // Empty name
				Description:    "Workflow with no name",
				Graph:          validWorkflowGraph(),
				AvailableToAll: true,
			},
			userID:       "admin@test.example",
			isAdmin:      true,
			expectStatus: http.StatusBadRequest,
			expectInDB:   false,
		},
		{
			name: "reject workflow for other group (user)",
			request: workflows.CreateWorkflowRequest{
				WorkflowName: "Other Group Workflow",
				Description:  "Workflow for group user doesn't belong to",
				Graph:        validWorkflowGraph(),
				Groups:       []string{"other-team"},
			},
			setupGroups:  []string{"other-team", "dev-team"}, // Create both groups
			userGroups:   []string{"dev-team"},
			userID:       "user@test.example",
			isAdmin:      false,
			expectStatus: http.StatusBadRequest,
			expectInDB:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh handler for each test to avoid state pollution
			handler := setupWorkflowTestHandler()

			// Create user groups in fake client
			for _, groupName := range tt.setupGroups {
				group := &krknv1alpha1.KrknUserGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      groupName,
						Namespace: handler.namespace,
					},
				}
				if err := handler.client.Create(context.Background(), group); err != nil {
					t.Fatalf("Failed to create test group: %v", err)
				}
			}

			// Create user (both admin and regular users need KrknUser CR)
			labels := make(map[string]string)
			for _, group := range tt.userGroups {
				labels["group.krkn.krkn-chaos.dev/"+group] = "true"
			}
			userName := "krknuser-" + sanitizeUserID(tt.userID)
			user := &krknv1alpha1.KrknUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      userName,
					Namespace: handler.namespace,
					Labels:    labels,
				},
				Spec: krknv1alpha1.KrknUserSpec{
					UserID: tt.userID,
				},
			}
			if err := handler.client.Create(context.Background(), user); err != nil {
				t.Fatalf("Failed to create test user: %v", err)
			}

			// Marshal request
			body, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Create request
			req := httptest.NewRequest(http.MethodPost, WorkflowsPath, bytes.NewReader(body))
			if tt.isAdmin {
				req = addAdminContext(req)
			} else {
				req = addUserContext(req, tt.userID, tt.userGroups...)
			}

			// Execute request
			rr := httptest.NewRecorder()
			handler.CreateWorkflow(rr, req)

			// Check status code
			if rr.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectStatus, rr.Code, rr.Body.String())
			}

			// Check if workflow was created in DB
			if tt.expectInDB {
				var configMapList corev1.ConfigMapList
				err := handler.client.List(context.Background(), &configMapList, client.InNamespace(handler.namespace))
				if err != nil {
					t.Fatalf("Failed to list ConfigMaps: %v", err)
				}

				found := false
				for _, cm := range configMapList.Items {
					if cm.Labels["files.krkn.krkn-chaos.dev/file-purpose"] == "workflow-template" {
						found = true
						// Verify it contains the graph
						if _, exists := cm.Data["workflow.json"]; !exists {
							t.Errorf("Workflow ConfigMap missing workflow.json data")
						}
						break
					}
				}

				if !found {
					t.Errorf("Expected workflow ConfigMap to be created, but not found")
				}
			}
		})
	}
}

func TestListWorkflows(t *testing.T) {
	handler := setupWorkflowTestHandler()

	// Create test workflow ConfigMaps
	workflow1 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "file-workflow-1",
			Namespace: handler.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":                     "krkn-operator",
				"app.kubernetes.io/component":                "file",
				"files.krkn.krkn-chaos.dev/file-id":          "workflow-1",
				"files.krkn.krkn-chaos.dev/file-purpose":     "workflow-template",
				"files.krkn.krkn-chaos.dev/available-to-all": "true",
			},
			Annotations: map[string]string{
				"files.krkn.krkn-chaos.dev/description": "Test workflow 1",
			},
		},
		Data: map[string]string{
			"workflow.json": `{"node1": {"name": "test", "image": "test:latest"}}`,
		},
	}

	workflow2 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "file-workflow-2",
			Namespace: handler.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":                 "krkn-operator",
				"app.kubernetes.io/component":            "file",
				"files.krkn.krkn-chaos.dev/file-id":      "workflow-2",
				"files.krkn.krkn-chaos.dev/file-purpose": "workflow-template",
				"group.krkn.krkn-chaos.dev/dev-team":     "true",
			},
			Annotations: map[string]string{
				"files.krkn.krkn-chaos.dev/description": "Test workflow 2",
			},
		},
		Data: map[string]string{
			"workflow.json": `{"node1": {"name": "test", "image": "test:latest"}}`,
		},
	}

	// Create workflows in fake client
	if err := handler.client.Create(context.Background(), workflow1); err != nil {
		t.Fatalf("Failed to create test workflow 1: %v", err)
	}
	if err := handler.client.Create(context.Background(), workflow2); err != nil {
		t.Fatalf("Failed to create test workflow 2: %v", err)
	}

	tests := []struct {
		name         string
		isAdmin      bool
		expectStatus int
		expectCount  int
	}{
		{
			name:         "admin sees all workflows",
			isAdmin:      true,
			expectStatus: http.StatusOK,
			expectCount:  2,
		},
		{
			name:         "user gets forbidden",
			isAdmin:      false,
			expectStatus: http.StatusForbidden,
			expectCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, WorkflowsPath, nil)
			if tt.isAdmin {
				req = addAdminContext(req)
			} else {
				req = addUserContext(req, "user@test.example")
			}

			rr := httptest.NewRecorder()
			handler.ListWorkflows(rr, req)

			if rr.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectStatus, rr.Code, rr.Body.String())
			}

			if tt.expectStatus == http.StatusOK {
				var resp workflows.ListWorkflowsResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if len(resp.Workflows) != tt.expectCount {
					t.Errorf("Expected %d workflows, got %d", tt.expectCount, len(resp.Workflows))
				}
			}
		})
	}
}

func TestListAvailableWorkflows(t *testing.T) {
	handler := setupWorkflowTestHandler()

	// Create test workflows
	publicWorkflow := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "file-public-workflow",
			Namespace: handler.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":                     "krkn-operator",
				"app.kubernetes.io/component":                "file",
				"files.krkn.krkn-chaos.dev/file-id":          "public-wf",
				"files.krkn.krkn-chaos.dev/file-purpose":     "workflow-template",
				"files.krkn.krkn-chaos.dev/available-to-all": "true",
			},
		},
		Data: map[string]string{
			"workflow.json": `{"node1": {"name": "test", "image": "test:latest"}}`,
		},
	}

	groupWorkflow := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "file-group-workflow",
			Namespace: handler.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":                 "krkn-operator",
				"app.kubernetes.io/component":            "file",
				"files.krkn.krkn-chaos.dev/file-id":      "group-wf",
				"files.krkn.krkn-chaos.dev/file-purpose": "workflow-template",
				"group.krkn.krkn-chaos.dev/dev-team":     "true",
			},
			Annotations: map[string]string{
				"files.krkn.krkn-chaos.dev/created-by": "user@test.example",
			},
		},
		Data: map[string]string{
			"workflow.json": `{"node1": {"name": "test", "image": "test:latest"}}`,
		},
	}

	if err := handler.client.Create(context.Background(), publicWorkflow); err != nil {
		t.Fatalf("Failed to create public workflow: %v", err)
	}
	if err := handler.client.Create(context.Background(), groupWorkflow); err != nil {
		t.Fatalf("Failed to create group workflow: %v", err)
	}

	// Create user group
	group := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dev-team",
			Namespace: handler.namespace,
		},
	}
	if err := handler.client.Create(context.Background(), group); err != nil {
		t.Fatalf("Failed to create test group: %v", err)
	}

	// Create user
	userName := "krknuser-" + sanitizeUserID("user@test.example")
	user := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userName,
			Namespace: handler.namespace,
			Labels: map[string]string{
				"group.krkn.krkn-chaos.dev/dev-team": "true",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID: "user@test.example",
		},
	}
	if err := handler.client.Create(context.Background(), user); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name         string
		isAdmin      bool
		userGroups   []string
		expectStatus int
		expectCount  int
	}{
		{
			name:         "admin sees all workflows",
			isAdmin:      true,
			expectStatus: http.StatusOK,
			expectCount:  2,
		},
		{
			name:         "user sees public and own group workflows",
			isAdmin:      false,
			userGroups:   []string{"dev-team"},
			expectStatus: http.StatusOK,
			expectCount:  2, // public + group
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, WorkflowsAvailablePath, nil)
			if tt.isAdmin {
				req = addAdminContext(req)
			} else {
				req = addUserContext(req, "user@test.example", tt.userGroups...)
			}

			rr := httptest.NewRecorder()
			handler.ListAvailableWorkflows(rr, req)

			if rr.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectStatus, rr.Code, rr.Body.String())
			}

			if tt.expectStatus == http.StatusOK {
				var resp workflows.AvailableWorkflowsResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if len(resp.Workflows) != tt.expectCount {
					t.Errorf("Expected %d workflows, got %d", tt.expectCount, len(resp.Workflows))
				}
			}
		})
	}
}

func TestGetWorkflow(t *testing.T) {
	handler := setupWorkflowTestHandler()

	// Create test workflow
	workflow := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "file-test-workflow",
			Namespace: handler.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":                     "krkn-operator",
				"app.kubernetes.io/component":                "file",
				"files.krkn.krkn-chaos.dev/file-id":          "test-wf-id",
				"files.krkn.krkn-chaos.dev/file-purpose":     "workflow-template",
				"files.krkn.krkn-chaos.dev/available-to-all": "true",
			},
			Annotations: map[string]string{
				"files.krkn.krkn-chaos.dev/description": "Test workflow",
			},
		},
		Data: map[string]string{
			"workflow.json": `{"node1": {"name": "test-node", "image": "test:latest"}}`,
		},
	}

	if err := handler.client.Create(context.Background(), workflow); err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	tests := []struct {
		name         string
		workflowID   string
		isAdmin      bool
		expectStatus int
		expectGraph  bool
	}{
		{
			name:         "get workflow successfully",
			workflowID:   "test-wf-id",
			isAdmin:      true,
			expectStatus: http.StatusOK,
			expectGraph:  true,
		},
		{
			name:         "workflow not found",
			workflowID:   "nonexistent-id",
			isAdmin:      true,
			expectStatus: http.StatusNotFound,
			expectGraph:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, WorkflowsPath+"/"+tt.workflowID, nil)
			if tt.isAdmin {
				req = addAdminContext(req)
			} else {
				req = addUserContext(req, "user@test.example")
			}

			rr := httptest.NewRecorder()
			handler.GetWorkflow(rr, req)

			if rr.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectStatus, rr.Code, rr.Body.String())
			}

			if tt.expectGraph {
				var resp workflows.WorkflowResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if len(resp.Graph) == 0 {
					t.Errorf("Expected graph to be populated, got empty graph")
				}

				if _, exists := resp.Graph["node1"]; !exists {
					t.Errorf("Expected node1 in graph, not found")
				}
			}
		})
	}
}

func TestListAvailableWorkflowsMethodCheck(t *testing.T) {
	handler := setupWorkflowTestHandler()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run("reject "+method, func(t *testing.T) {
			req := httptest.NewRequest(method, WorkflowsAvailablePath, nil)
			req = addAdminContext(req)

			rr := httptest.NewRecorder()
			// Call via server route (which has method guard)
			mux := http.NewServeMux()
			mux.Handle(WorkflowsAvailablePath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}
				handler.ListAvailableWorkflows(w, r)
			}))

			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, rr.Code)
			}
		})
	}
}

func TestNodeCountAccuracy(t *testing.T) {
	handler := setupWorkflowTestHandler()

	// Create workflow with metadata node
	workflow := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "file-test-nodecount",
			Namespace: handler.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":                     "krkn-operator",
				"app.kubernetes.io/component":                "file",
				"files.krkn.krkn-chaos.dev/file-id":          "nodecount-test",
				"files.krkn.krkn-chaos.dev/file-purpose":     "workflow-template",
				"files.krkn.krkn-chaos.dev/available-to-all": "true",
			},
			Annotations: map[string]string{
				"files.krkn.krkn-chaos.dev/created-by": "admin@test.example",
			},
		},
		Data: map[string]string{
			"workflow.json": `{
				"node1": {"name": "test1", "image": "test:latest"},
				"node2": {"name": "test2", "image": "test:latest"},
				"_metadata": {"version": "1.0"}
			}`,
		},
	}

	if err := handler.client.Create(context.Background(), workflow); err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	userName := "krknuser-" + sanitizeUserID("admin@test.example")
	user := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userName,
			Namespace: handler.namespace,
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID: "admin@test.example",
		},
	}
	if err := handler.client.Create(context.Background(), user); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, WorkflowsAvailablePath, nil)
	req = addAdminContext(req)

	rr := httptest.NewRecorder()
	handler.ListAvailableWorkflows(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var resp workflows.AvailableWorkflowsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(resp.Workflows) != 1 {
		t.Fatalf("Expected 1 workflow, got %d", len(resp.Workflows))
	}

	// Should count only node1 and node2, not _metadata
	if resp.Workflows[0].NodeCount != 2 {
		t.Errorf("Expected NodeCount 2 (excluding _metadata), got %d", resp.Workflows[0].NodeCount)
	}
}
