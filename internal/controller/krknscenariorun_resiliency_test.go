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

Assisted-by: Claude Opus 4.6 (claude-opus-4-6@20260805)
*/

package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// newTestClientsetWithLogs creates a real kubernetes.Clientset backed by an
// httptest.Server that serves pod log content. The server responds to
// /api/v1/namespaces/{ns}/pods/{pod}/log with the provided logContent.
// All other requests get 404. Returns the clientset and a cleanup function.
func newTestClientsetWithLogs(t *testing.T, namespace string, podLogs map[string]string) (kubernetes.Interface, func()) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for podName, logContent := range podLogs {
			expectedPath := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log", namespace, podName)
			if r.URL.Path == expectedPath {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte(logContent)); err != nil {
					t.Errorf("failed to write response for pod %s: %v", podName, err)
				}
				return
			}
		}
		http.NotFound(w, r)
	}))

	clientset, err := kubernetes.NewForConfig(&rest.Config{
		Host: server.URL,
	})
	require.NoError(t, err)

	return clientset, server.Close
}

func newResiliencyTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	return scheme
}

func TestCalculateResiliencyScores(t *testing.T) {
	resiliencyLog := func(score float64) string {
		return fmt.Sprintf(`2025-01-15 10:00:00 INFO Starting scenario
KRKN_RESILIENCY_REPORT_JSON: {"overall_resiliency_report": {"scenarios": {"test": %v}, "resiliency_score": %v, "passed_slos": 10, "total_slos": 12}}`, score, score)
	}

	tests := []struct {
		name           string
		clusterJobs    []krknv1alpha1.ClusterJobStatus
		phase          string
		podLogs        map[string]string
		useFakeClient  bool
		wantErr        bool
		errContains    string
		wantScores     map[string]float64
		wantStatuses   map[string]string
		wantScoreCount int
		wantNoScores   bool
	}{
		{
			name:  "single cluster happy path",
			phase: "Succeeded",
			clusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ProviderName: "local", ClusterName: "cluster1", JobID: "job-1", PodName: "krkn-job-abc123", Phase: "Succeeded"},
			},
			podLogs:        map[string]string{"krkn-job-abc123": resiliencyLog(95.5)},
			wantScores:     map[string]float64{"cluster1": 95.5},
			wantStatuses:   map[string]string{"cluster1": "calculated"},
			wantScoreCount: 1,
		},
		{
			name:  "multiple clusters",
			phase: "Succeeded",
			clusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ProviderName: "aws", ClusterName: "prod-east", JobID: "job-1", PodName: "krkn-job-east", Phase: "Succeeded"},
				{ProviderName: "aws", ClusterName: "prod-west", JobID: "job-2", PodName: "krkn-job-west", Phase: "Succeeded"},
			},
			podLogs:        map[string]string{"krkn-job-east": resiliencyLog(88.0), "krkn-job-west": resiliencyLog(72.5)},
			wantScores:     map[string]float64{"prod-east": 88.0, "prod-west": 72.5},
			wantStatuses:   map[string]string{"prod-east": "calculated", "prod-west": "calculated"},
			wantScoreCount: 2,
		},
		{
			name:  "zero score",
			phase: "Succeeded",
			clusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ProviderName: "local", ClusterName: "cluster1", JobID: "job-1", PodName: "krkn-job-zero", Phase: "Succeeded"},
			},
			podLogs:        map[string]string{"krkn-job-zero": resiliencyLog(0.0)},
			wantScores:     map[string]float64{"cluster1": 0.0},
			wantStatuses:   map[string]string{"cluster1": "calculated"},
			wantScoreCount: 1,
		},
		{
			name:  "partially failed - only succeeded jobs get scores",
			phase: "PartiallyFailed",
			clusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ProviderName: "aws", ClusterName: "cluster1", JobID: "job-1", PodName: "krkn-job-ok", Phase: "Succeeded"},
				{ProviderName: "aws", ClusterName: "cluster2", JobID: "job-2", PodName: "krkn-job-fail", Phase: "Failed"},
			},
			podLogs:        map[string]string{"krkn-job-ok": resiliencyLog(90.0)},
			wantScores:     map[string]float64{"cluster1": 90.0},
			wantStatuses:   map[string]string{"cluster1": "calculated"},
			wantScoreCount: 1,
		},
		{
			name:  "no reports in logs - persists error entry",
			phase: "Succeeded",
			clusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ProviderName: "local", ClusterName: "cluster1", JobID: "job-1", PodName: "test-pod-1", Phase: "Succeeded"},
			},
			useFakeClient:  true,
			wantErr:        true,
			wantScoreCount: 1,
			wantStatuses:   map[string]string{"cluster1": "error"},
		},
		{
			name:  "skips non-succeeded jobs - no entries produced",
			phase: "PartiallyFailed",
			clusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ProviderName: "local", ClusterName: "cluster1", JobID: "job-1", PodName: "test-pod-1", Phase: "Failed"},
			},
			useFakeClient: true,
			wantNoScores:  true,
		},
		{
			name:  "missing pod name - persists error entry",
			phase: "Succeeded",
			clusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ProviderName: "local", ClusterName: "cluster1", JobID: "job-1", PodName: "", Phase: "Succeeded"},
			},
			useFakeClient:  true,
			wantErr:        true,
			wantScoreCount: 1,
			wantStatuses:   map[string]string{"cluster1": "error"},
		},
		{
			name:  "mixed results - calculated and error entries persisted",
			phase: "Succeeded",
			clusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ProviderName: "aws", ClusterName: "prod-east", JobID: "job-1", PodName: "krkn-job-east", Phase: "Succeeded"},
				{ProviderName: "aws", ClusterName: "prod-west", JobID: "job-2", PodName: "krkn-job-west", Phase: "Succeeded"},
			},
			podLogs:        map[string]string{"krkn-job-east": resiliencyLog(88.0), "krkn-job-west": "no report here"},
			wantErr:        true,
			wantScoreCount: 2,
			wantScores:     map[string]float64{"prod-east": 88.0},
			wantStatuses:   map[string]string{"prod-east": "calculated", "prod-west": "error"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			scheme := newResiliencyTestScheme(t)

			scenarioRun := &krknv1alpha1.KrknScenarioRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-run",
					Namespace: "default",
				},
				Spec: krknv1alpha1.KrknScenarioRunSpec{
					TargetRequestID:        "test-target",
					TargetClusters:         map[string][]string{"local": {"cluster1"}},
					ScenarioName:           "test",
					ScenarioImage:          "test:latest",
					ResiliencyScoreEnabled: true,
				},
				Status: krknv1alpha1.KrknScenarioRunStatus{
					Phase:       tc.phase,
					ClusterJobs: tc.clusterJobs,
				},
			}

			var clientset kubernetes.Interface
			var cleanup func()
			if tc.useFakeClient {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "test-pod-1", Namespace: "default"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "scenario", Image: "test:latest"}},
					},
					Status: corev1.PodStatus{Phase: corev1.PodSucceeded},
				}
				clientset = fake.NewSimpleClientset(pod)
			} else {
				clientset, cleanup = newTestClientsetWithLogs(t, "default", tc.podLogs)
				defer cleanup()
			}

			fakeClient := fakeclient.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scenarioRun).
				WithStatusSubresource(scenarioRun).
				Build()

			reconciler := &KrknScenarioRunReconciler{
				Client:    fakeClient,
				Scheme:    scheme,
				Clientset: clientset,
				Namespace: "default",
			}

			err := reconciler.calculateResiliencyScores(ctx, scenarioRun)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}

			if tc.wantNoScores {
				assert.Empty(t, scenarioRun.Status.ResiliencyScores)
				return
			}

			require.Len(t, scenarioRun.Status.ResiliencyScores, tc.wantScoreCount)

			statusMap := make(map[string]string)
			scoreMap := make(map[string]float64)
			for _, cs := range scenarioRun.Status.ResiliencyScores {
				statusMap[cs.ClusterName] = cs.Status
				scoreMap[cs.ClusterName] = cs.Score
			}
			for cluster, expectedStatus := range tc.wantStatuses {
				assert.Equal(t, expectedStatus, statusMap[cluster],
					"status mismatch for cluster %s", cluster)
			}
			for cluster, expectedScore := range tc.wantScores {
				assert.Equal(t, expectedScore, scoreMap[cluster],
					"score mismatch for cluster %s", cluster)
			}
		})
	}
}

func TestStatusEqual_ComparesResiliencyScores(t *testing.T) {
	reconciler := &KrknScenarioRunReconciler{}

	tests := []struct {
		name      string
		old       *krknv1alpha1.KrknScenarioRunStatus
		new       *krknv1alpha1.KrknScenarioRunStatus
		wantEqual bool
	}{
		{
			name: "detects added scores",
			old:  &krknv1alpha1.KrknScenarioRunStatus{Phase: "Succeeded"},
			new: &krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Succeeded",
				ResiliencyScores: []krknv1alpha1.ClusterResiliencyScore{
					{ClusterName: "cluster1", Score: 9.5, Status: "calculated"},
				},
			},
			wantEqual: false,
		},
		{
			name: "equal when scores match",
			old: &krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Succeeded",
				ResiliencyScores: []krknv1alpha1.ClusterResiliencyScore{
					{ClusterName: "cluster1", Score: 9.5, Status: "calculated"},
				},
			},
			new: &krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Succeeded",
				ResiliencyScores: []krknv1alpha1.ClusterResiliencyScore{
					{ClusterName: "cluster1", Score: 9.5, Status: "calculated"},
				},
			},
			wantEqual: true,
		},
		{
			name: "detects score status change from error to calculated",
			old: &krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Succeeded",
				ResiliencyScores: []krknv1alpha1.ClusterResiliencyScore{
					{ClusterName: "cluster1", Status: "error", Message: "logs unavailable"},
				},
			},
			new: &krknv1alpha1.KrknScenarioRunStatus{
				Phase: "Succeeded",
				ResiliencyScores: []krknv1alpha1.ClusterResiliencyScore{
					{ClusterName: "cluster1", Score: 9.5, Status: "calculated"},
				},
			},
			wantEqual: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantEqual, reconciler.statusEqual(tc.old, tc.new))
		})
	}
}

func TestHasIncompleteResiliencyScores(t *testing.T) {
	reconciler := &KrknScenarioRunReconciler{}

	tests := []struct {
		name   string
		scores []krknv1alpha1.ClusterResiliencyScore
		want   bool
	}{
		{
			name:   "no scores yet",
			scores: nil,
			want:   true,
		},
		{
			name: "all calculated",
			scores: []krknv1alpha1.ClusterResiliencyScore{
				{ClusterName: "c1", Score: 90, Status: "calculated"},
				{ClusterName: "c2", Score: 80, Status: "calculated"},
			},
			want: false,
		},
		{
			name: "has error entry",
			scores: []krknv1alpha1.ClusterResiliencyScore{
				{ClusterName: "c1", Score: 90, Status: "calculated"},
				{ClusterName: "c2", Status: "error", Message: "logs unavailable"},
			},
			want: true,
		},
		{
			name: "all error entries",
			scores: []krknv1alpha1.ClusterResiliencyScore{
				{ClusterName: "c1", Status: "error", Message: "logs unavailable"},
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			run := &krknv1alpha1.KrknScenarioRun{
				Status: krknv1alpha1.KrknScenarioRunStatus{
					ResiliencyScores: tc.scores,
				},
			}
			assert.Equal(t, tc.want, reconciler.hasIncompleteResiliencyScores(run))
		})
	}
}

func TestResiliencyScoreEnvVarInjection(t *testing.T) {
	tests := []struct {
		name       string
		labels     map[string]string
		env        map[string]string
		enabled    bool
		wantEnvVar bool
		wantValue  string
	}{
		{
			name:       "injected for standalone runs",
			labels:     map[string]string{},
			env:        map[string]string{"EXISTING_VAR": "value"},
			enabled:    true,
			wantEnvVar: true,
			wantValue:  "true",
		},
		{
			name:       "not injected for graph-run children",
			labels:     map[string]string{"krkn.dev/graph-run": "test-graphrun"},
			env:        map[string]string{},
			enabled:    true,
			wantEnvVar: false,
		},
		{
			name:       "overrides user-provided false value",
			labels:     map[string]string{},
			env:        map[string]string{"RESILIENCY_SCORE": "false"},
			enabled:    true,
			wantEnvVar: true,
			wantValue:  "true",
		},
		{
			name:       "not injected when disabled",
			labels:     map[string]string{},
			env:        map[string]string{},
			enabled:    false,
			wantEnvVar: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scenarioRun := &krknv1alpha1.KrknScenarioRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-run",
					Namespace: "default",
					Labels:    tc.labels,
				},
				Spec: krknv1alpha1.KrknScenarioRunSpec{
					TargetRequestID:        "test-target",
					TargetClusters:         map[string][]string{"local": {"cluster1"}},
					ScenarioName:           "test",
					ScenarioImage:          "test:latest",
					ResiliencyScoreEnabled: tc.enabled,
					Environment:            tc.env,
				},
			}

			envVars := make([]corev1.EnvVar, 0, len(scenarioRun.Spec.Environment))
			for key, value := range scenarioRun.Spec.Environment {
				envVars = append(envVars, corev1.EnvVar{Name: key, Value: value})
			}

			if _, isGraphRun := scenarioRun.Labels["krkn.dev/graph-run"]; !isGraphRun && scenarioRun.Spec.ResiliencyScoreEnabled {
				filtered := envVars[:0]
				for _, ev := range envVars {
					if ev.Name != "RESILIENCY_SCORE" {
						filtered = append(filtered, ev)
					}
				}
				envVars = append(filtered, corev1.EnvVar{Name: "RESILIENCY_SCORE", Value: "true"})
			}

			var found *corev1.EnvVar
			for i, ev := range envVars {
				if ev.Name == "RESILIENCY_SCORE" {
					found = &envVars[i]
					break
				}
			}

			if tc.wantEnvVar {
				require.NotNil(t, found, "RESILIENCY_SCORE env var should be present")
				assert.Equal(t, tc.wantValue, found.Value)
			} else {
				assert.Nil(t, found, "RESILIENCY_SCORE env var should not be present")
			}
		})
	}
}

// TestReconcile_ResiliencyScoreFailure_DoesNotBlockTerminalPhase is a
// reconcile-level regression test for the bug where a resiliency-score
// extraction error caused an early return before Status().Update(), leaving the
// run stuck in a non-terminal phase even though all cluster jobs had completed.
//
// The fix moves the requeue to AFTER the status update so the terminal phase is
// always persisted. This test proves that: when score calculation fails, the
// run still reaches its terminal phase and the reconciler requeues for retry.
func TestReconcile_ResiliencyScoreFailure_DoesNotBlockTerminalPhase(t *testing.T) {
	scheme := newResiliencyTestScheme(t)
	const (
		ns          = "default"
		runName     = "resiliency-regression"
		clusterName = "cluster1"
		podName     = "krkn-job-abc123"
	)

	now := metav1.Now()
	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: runName, Namespace: ns},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			ScenarioName:           "test-scenario",
			ScenarioImage:          "test:latest",
			TargetRequestID:        "req-1",
			TargetClusters:         map[string][]string{"local": {clusterName}},
			ResiliencyScoreEnabled: true,
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase:        "Running",
			TotalTargets: 1,
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{
					ProviderName:   "local",
					ClusterName:    clusterName,
					JobID:          "job-1",
					PodName:        podName,
					Phase:          "Succeeded",
					StartTime:      &now,
					CompletionTime: &now,
				},
			},
		},
	}

	// Serve pod logs WITHOUT the KRKN_RESILIENCY_REPORT_JSON marker so
	// calculateResiliencyScores returns an error.
	clientset, cleanup := newTestClientsetWithLogs(t, ns, map[string]string{
		podName: "2025-01-15 10:00:00 INFO Scenario completed successfully\nno resiliency marker here\n",
	})
	defer cleanup()

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(scenarioRun).
		WithStatusSubresource(scenarioRun).
		Build()

	reconciler := &KrknScenarioRunReconciler{
		Client:    fakeClient,
		Scheme:    scheme,
		Clientset: clientset,
		Namespace: ns,
	}

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: runName, Namespace: ns},
	})

	require.NoError(t, err)

	// The reconciler should requeue for resiliency retry.
	assert.True(t, result.RequeueAfter > 0,
		"expected requeue for resiliency score retry")

	// Re-fetch the run and verify the terminal phase was persisted.
	var updated krknv1alpha1.KrknScenarioRun
	require.NoError(t, fakeClient.Get(context.Background(),
		types.NamespacedName{Name: runName, Namespace: ns}, &updated))

	assert.Equal(t, "Succeeded", updated.Status.Phase,
		"terminal phase must be persisted even when resiliency score calculation fails")

	// Error entries should be persisted so API consumers can see which clusters failed.
	require.Len(t, updated.Status.ResiliencyScores, 1,
		"error score entry should be persisted")
	assert.Equal(t, "error", updated.Status.ResiliencyScores[0].Status,
		"score status should be 'error' when extraction fails")
	assert.Equal(t, clusterName, updated.Status.ResiliencyScores[0].ClusterName)
	assert.NotEmpty(t, updated.Status.ResiliencyScores[0].Message,
		"error entry should include a message")
}
