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
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
)

func TestCreateGraphRun_WithFileReferences(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tests := []struct {
		name         string
		request      GraphRunCreateRequest
		setupFiles   []*corev1.ConfigMap
		setupTarget  *krknv1alpha1.KrknTargetRequest
		userID       string
		isAdmin      bool
		expectStatus int
		expectError  string
	}{
		{
			name: "valid file reference - public file",
			request: GraphRunCreateRequest{
				Graph: map[string]krknv1alpha1.GraphScenarioNode{
					"node-1": {
						Name:  "test-scenario",
						Image: "quay.io/krkn-chaos/krkn-hub:dummy-scenario",
						Volumes: map[string]string{
							"550e8400-e29b-41d4-a716-446655440001": "/config/test.yaml",
						},
					},
				},
				TargetRequestID: "test-target",
				TargetClusters: map[string][]string{
					"local-provider": {"local-cluster"},
				},
			},
			setupFiles: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "file-550e8400-e29b-41d4-a716-446655440001",
						Namespace: "default",
						Labels: map[string]string{
							files.AppNameLabel:        "krkn-operator",
							files.AppComponentLabel:   "file",
							files.FileIDLabel:         "550e8400-e29b-41d4-a716-446655440001",
							files.AvailableToAllLabel: "true",
						},
					},
					Data: map[string]string{
						"test.yaml": "key: value",
					},
				},
			},
			setupTarget: &krknv1alpha1.KrknTargetRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-target",
					Namespace: "default",
				},
				Spec: krknv1alpha1.KrknTargetRequestSpec{
					UUID: "test-uuid",
				},
				Status: krknv1alpha1.KrknTargetRequestStatus{
					Status: "Completed",
					TargetData: map[string][]krknv1alpha1.ClusterTarget{
						"local-provider": {
							{
								ClusterName:   "local-cluster",
								ClusterAPIURL: "https://local-cluster:6443",
							},
						},
					},
				},
			},
			userID:       "admin",
			isAdmin:      true,
			expectStatus: http.StatusCreated,
		},
		{
			name: "file not found",
			request: GraphRunCreateRequest{
				Graph: map[string]krknv1alpha1.GraphScenarioNode{
					"node-1": {
						Name:  "test-scenario",
						Image: "quay.io/krkn-chaos/krkn-hub:dummy-scenario",
						Volumes: map[string]string{
							"non-existent-uuid": "/config/test.yaml",
						},
					},
				},
				TargetRequestID: "test-target",
				TargetClusters: map[string][]string{
					"local-provider": {"local-cluster"},
				},
			},
			setupFiles: []*corev1.ConfigMap{},
			setupTarget: &krknv1alpha1.KrknTargetRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-target",
					Namespace: "default",
				},
				Spec: krknv1alpha1.KrknTargetRequestSpec{
					UUID: "test-uuid",
				},
				Status: krknv1alpha1.KrknTargetRequestStatus{
					Status: "Completed",
					TargetData: map[string][]krknv1alpha1.ClusterTarget{
						"local-provider": {
							{
								ClusterName:   "local-cluster",
								ClusterAPIURL: "https://local-cluster:6443",
							},
						},
					},
				},
			},
			userID:       "admin",
			isAdmin:      true,
			expectStatus: http.StatusBadRequest,
			expectError:  "not found",
		},
		{
			name: "multiple files in multiple nodes",
			request: GraphRunCreateRequest{
				Graph: map[string]krknv1alpha1.GraphScenarioNode{
					"node-1": {
						Name:  "test-scenario",
						Image: "quay.io/krkn-chaos/krkn-hub:dummy-scenario",
						Volumes: map[string]string{
							"550e8400-e29b-41d4-a716-446655440005": "/config/test1.yaml",
						},
					},
					"node-2": {
						Name:  "test-scenario",
						Image: "quay.io/krkn-chaos/krkn-hub:dummy-scenario",
						Volumes: map[string]string{
							"550e8400-e29b-41d4-a716-446655440006": "/config/test2.yaml",
						},
					},
				},
				TargetRequestID: "test-target",
				TargetClusters: map[string][]string{
					"local-provider": {"local-cluster"},
				},
			},
			setupFiles: []*corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "file-550e8400-e29b-41d4-a716-446655440005",
						Namespace: "default",
						Labels: map[string]string{
							files.AppNameLabel:        "krkn-operator",
							files.AppComponentLabel:   "file",
							files.FileIDLabel:         "550e8400-e29b-41d4-a716-446655440005",
							files.AvailableToAllLabel: "true",
						},
					},
					Data: map[string]string{
						"test1.yaml": "key: value1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "file-550e8400-e29b-41d4-a716-446655440006",
						Namespace: "default",
						Labels: map[string]string{
							files.AppNameLabel:        "krkn-operator",
							files.AppComponentLabel:   "file",
							files.FileIDLabel:         "550e8400-e29b-41d4-a716-446655440006",
							files.AvailableToAllLabel: "true",
						},
					},
					Data: map[string]string{
						"test2.yaml": "key: value2",
					},
				},
			},
			setupTarget: &krknv1alpha1.KrknTargetRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-target",
					Namespace: "default",
				},
				Spec: krknv1alpha1.KrknTargetRequestSpec{
					UUID: "test-uuid",
				},
				Status: krknv1alpha1.KrknTargetRequestStatus{
					Status: "Completed",
					TargetData: map[string][]krknv1alpha1.ClusterTarget{
						"local-provider": {
							{
								ClusterName:   "local-cluster",
								ClusterAPIURL: "https://local-cluster:6443",
							},
						},
					},
				},
			},
			userID:       "admin",
			isAdmin:      true,
			expectStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup fake clients
			k8sClient := fake.NewSimpleClientset()
			objects := []runtime.Object{}
			for _, file := range tt.setupFiles {
				objects = append(objects, file)
			}
			if tt.setupTarget != nil {
				objects = append(objects, tt.setupTarget)
			}

			// Add KrknUser CRs
			user1 := &krknv1alpha1.KrknUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "krknuser-user1",
					Namespace: "default",
				},
				Spec: krknv1alpha1.KrknUserSpec{
					UserID: "user1",
					Role:   "user",
				},
			}
			adminUser := &krknv1alpha1.KrknUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "krknuser-admin",
					Namespace: "default",
				},
				Spec: krknv1alpha1.KrknUserSpec{
					UserID: "admin",
					Role:   "admin",
				},
			}
			objects = append(objects, user1, adminUser)

			ctrlClient := fakeclient.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objects...).
				Build()

			// Create handler
			handler := &Handler{
				client:    ctrlClient,
				clientset: k8sClient,
				namespace: "default",
			}

			// Create request
			reqBody, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, GraphRunsPath, bytes.NewReader(reqBody))

			// Add auth context
			role := "user"
			if tt.isAdmin {
				role = "admin"
			}
			ctx := context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
				UserID: tt.userID,
				Role:   role,
			})
			req = req.WithContext(ctx)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			handler.CreateGraphRun(rr, req)

			// Check status code
			if rr.Code != tt.expectStatus {
				t.Errorf("expected status %d, got %d. Response: %s", tt.expectStatus, rr.Code, rr.Body.String())
			}

			// Check error message if expected
			if tt.expectError != "" {
				var errResp ErrorResponse
				_ = json.NewDecoder(rr.Body).Decode(&errResp)
				if errResp.Message == "" || !contains(errResp.Message, tt.expectError) {
					t.Errorf("expected error containing '%s', got '%s'", tt.expectError, errResp.Message)
				}
			}

			// For successful cases, verify GraphRun was created
			if tt.expectStatus == http.StatusCreated {
				var resp GraphRunDetailResponse
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				// Verify graph structure matches request
				if len(resp.Spec.Graph) != len(tt.request.Graph) {
					t.Errorf("expected %d nodes in graph, got %d", len(tt.request.Graph), len(resp.Spec.Graph))
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
