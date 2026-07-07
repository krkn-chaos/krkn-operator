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
)

func TestCreateGraphRun_ResiliencyScore(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Common test setup
	testTarget := &krknv1alpha1.KrknTargetRequest{
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
	}

	baseRequest := GraphRunCreateRequest{
		Graph: map[string]krknv1alpha1.GraphScenarioNode{
			"node-1": {
				Name:  "test-scenario",
				Image: "quay.io/krkn-chaos/krkn-hub:dummy-scenario",
			},
		},
		TargetRequestID: "test-target",
		TargetClusters: map[string][]string{
			"local-provider": {"local-cluster"},
		},
	}

	tests := []struct {
		name           string
		request        GraphRunCreateRequest
		headers        map[string]string
		expectStatus   int
		expectError    string
		validateResult func(t *testing.T, graphRun *krknv1alpha1.KrknGraphRun)
	}{
		{
			name:    "resiliency score enabled with baseline and mount path",
			request: baseRequest,
			headers: map[string]string{
				"X-Resiliency-Score":      "true",
				"X-Resiliency-Baseline":   "9.0",
				"X-Resiliency-Mount-Path": "/etc/kraken/metrics.yaml",
			},
			expectStatus: http.StatusCreated,
			validateResult: func(t *testing.T, graphRun *krknv1alpha1.KrknGraphRun) {
				if !graphRun.Spec.ResiliencyScoreEnabled {
					t.Error("ResiliencyScoreEnabled should be true")
				}
				if graphRun.Spec.ResiliencyMountPath != "/etc/kraken/metrics.yaml" {
					t.Errorf("Expected mount path '/etc/kraken/metrics.yaml', got '%s'", graphRun.Spec.ResiliencyMountPath)
				}
				if graphRun.Spec.ResiliencyScoreBaseline == nil || *graphRun.Spec.ResiliencyScoreBaseline != 9.0 {
					t.Errorf("Expected baseline 9.0, got %v", graphRun.Spec.ResiliencyScoreBaseline)
				}
			},
		},
		{
			name:    "resiliency score enabled without mount path (optional)",
			request: baseRequest,
			headers: map[string]string{
				"X-Resiliency-Score":    "true",
				"X-Resiliency-Baseline": "8.5",
			},
			expectStatus: http.StatusCreated,
			validateResult: func(t *testing.T, graphRun *krknv1alpha1.KrknGraphRun) {
				if !graphRun.Spec.ResiliencyScoreEnabled {
					t.Error("ResiliencyScoreEnabled should be true")
				}
				if graphRun.Spec.ResiliencyMountPath != "" {
					t.Errorf("Expected empty mount path, got '%s'", graphRun.Spec.ResiliencyMountPath)
				}
				if graphRun.Spec.ResiliencyScoreBaseline == nil || *graphRun.Spec.ResiliencyScoreBaseline != 8.5 {
					t.Errorf("Expected baseline 8.5, got %v", graphRun.Spec.ResiliencyScoreBaseline)
				}
			},
		},
		{
			name:    "resiliency score enabled but baseline missing - error",
			request: baseRequest,
			headers: map[string]string{
				"X-Resiliency-Score": "true",
			},
			expectStatus: http.StatusBadRequest,
			expectError:  "X-Resiliency-Baseline header is required",
		},
		{
			name:    "resiliency score enabled with invalid baseline - error",
			request: baseRequest,
			headers: map[string]string{
				"X-Resiliency-Score":    "true",
				"X-Resiliency-Baseline": "invalid",
			},
			expectStatus: http.StatusBadRequest,
			expectError:  "must be a valid number",
		},
		{
			name:    "resiliency score enabled with negative baseline - error",
			request: baseRequest,
			headers: map[string]string{
				"X-Resiliency-Score":    "true",
				"X-Resiliency-Baseline": "-5.0",
			},
			expectStatus: http.StatusBadRequest,
			expectError:  "must be a non-negative number",
		},
		{
			name:    "resiliency score enabled with relative mount path - error",
			request: baseRequest,
			headers: map[string]string{
				"X-Resiliency-Score":      "true",
				"X-Resiliency-Baseline":   "9.0",
				"X-Resiliency-Mount-Path": "relative/path/metrics.yaml",
			},
			expectStatus: http.StatusBadRequest,
			expectError:  "must be an absolute path",
		},
		{
			name:    "resiliency score disabled - baseline ignored",
			request: baseRequest,
			headers: map[string]string{
				"X-Resiliency-Score":    "false",
				"X-Resiliency-Baseline": "9.0",
			},
			expectStatus: http.StatusCreated,
			validateResult: func(t *testing.T, graphRun *krknv1alpha1.KrknGraphRun) {
				if graphRun.Spec.ResiliencyScoreEnabled {
					t.Error("ResiliencyScoreEnabled should be false")
				}
				if graphRun.Spec.ResiliencyScoreBaseline != nil {
					t.Error("Baseline should not be set when resiliency score is disabled")
				}
			},
		},
		{
			name:         "no resiliency headers - defaults to disabled",
			request:      baseRequest,
			headers:      map[string]string{},
			expectStatus: http.StatusCreated,
			validateResult: func(t *testing.T, graphRun *krknv1alpha1.KrknGraphRun) {
				if graphRun.Spec.ResiliencyScoreEnabled {
					t.Error("ResiliencyScoreEnabled should be false by default")
				}
				if graphRun.Spec.ResiliencyMountPath != "" {
					t.Error("ResiliencyMountPath should be empty by default")
				}
				if graphRun.Spec.ResiliencyScoreBaseline != nil {
					t.Error("ResiliencyScoreBaseline should be nil by default")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup fake clients
			k8sClient := fake.NewSimpleClientset()
			objects := []runtime.Object{testTarget}

			// Add admin user
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
			objects = append(objects, adminUser)

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

			// Add headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Add auth context (admin to bypass permission checks)
			ctx := context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
				UserID: "admin",
				Role:   "admin",
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
				if !contains(errResp.Message, tt.expectError) {
					t.Errorf("expected error containing '%s', got '%s'", tt.expectError, errResp.Message)
				}
			}

			// Validate result if provided
			if tt.validateResult != nil && rr.Code == http.StatusCreated {
				var resp GraphRunDetailResponse
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				// Fetch the created GraphRun CR to validate spec fields
				var graphRunList krknv1alpha1.KrknGraphRunList
				if err := ctrlClient.List(context.Background(), &graphRunList); err != nil {
					t.Fatalf("failed to list GraphRuns: %v", err)
				}

				if len(graphRunList.Items) != 1 {
					t.Fatalf("expected 1 GraphRun, got %d", len(graphRunList.Items))
				}

				tt.validateResult(t, &graphRunList.Items[0])
			}
		})
	}
}
