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

Assisted-by: Claude Sonnet 4.6 (claude-sonnet-4-6)
*/

package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// newTestScheme builds a scheme with all types needed for controller tests.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add krknv1alpha1 to scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add corev1 to scheme: %v", err)
	}
	return scheme
}

// newTargetRequest builds a KrknTargetRequest that maps providerName/clusterName -> clusterAPIURL.
func newTargetRequest(name, namespace, providerName, clusterName, clusterAPIURL string) *krknv1alpha1.KrknTargetRequest {
	return &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status: "completed",
			TargetData: map[string][]krknv1alpha1.ClusterTarget{
				providerName: {
					{ClusterName: clusterName, ClusterAPIURL: clusterAPIURL},
				},
			},
		},
	}
}

// --- Unit tests for getClusterAPIURL ---

func TestGetClusterAPIURL_Found(t *testing.T) {
	scheme := newTestScheme(t)
	const (
		ns           = "default"
		targetID     = "req-1"
		providerName = "aws"
		clusterName  = "prod-cluster"
		wantAPIURL   = "https://api.prod-cluster.example.com:6443"
	)

	targetRequest := newTargetRequest(targetID, ns, providerName, clusterName, wantAPIURL)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(targetRequest).
		WithStatusSubresource(targetRequest).Build()

	// Patch status separately because fake client ignores status in WithObjects.
	if err := fakeClient.Status().Update(context.Background(), targetRequest); err != nil {
		t.Fatalf("failed to set target request status: %v", err)
	}

	reconciler := &KrknScenarioRunReconciler{
		Client:    fakeClient,
		Scheme:    scheme,
		Namespace: ns,
	}

	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "run-1", Namespace: ns},
		Spec:       krknv1alpha1.KrknScenarioRunSpec{TargetRequestID: targetID},
	}

	got := reconciler.getClusterAPIURL(context.Background(), scenarioRun, providerName, clusterName)
	if got != wantAPIURL {
		t.Errorf("getClusterAPIURL() = %q, want %q", got, wantAPIURL)
	}
}

func TestGetClusterAPIURL_TargetRequestNotFound(t *testing.T) {
	scheme := newTestScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconciler := &KrknScenarioRunReconciler{
		Client:    fakeClient,
		Scheme:    scheme,
		Namespace: "default",
	}
	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "run-1", Namespace: "default"},
		Spec:       krknv1alpha1.KrknScenarioRunSpec{TargetRequestID: "missing-request"},
	}

	got := reconciler.getClusterAPIURL(context.Background(), scenarioRun, "aws", "cluster-1")
	if got != "" {
		t.Errorf("getClusterAPIURL() = %q, want empty string", got)
	}
}

func TestGetClusterAPIURL_ProviderNotFound(t *testing.T) {
	scheme := newTestScheme(t)
	const ns = "default"

	targetRequest := newTargetRequest("req-1", ns, "aws", "cluster-1", "https://api.example.com")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(targetRequest).
		WithStatusSubresource(targetRequest).Build()
	if err := fakeClient.Status().Update(context.Background(), targetRequest); err != nil {
		t.Fatalf("failed to set target request status: %v", err)
	}

	reconciler := &KrknScenarioRunReconciler{Client: fakeClient, Scheme: scheme, Namespace: ns}
	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "run-1", Namespace: ns},
		Spec:       krknv1alpha1.KrknScenarioRunSpec{TargetRequestID: "req-1"},
	}

	got := reconciler.getClusterAPIURL(context.Background(), scenarioRun, "gcp", "cluster-1")
	if got != "" {
		t.Errorf("getClusterAPIURL() = %q, want empty string when provider absent", got)
	}
}

func TestGetClusterAPIURL_ClusterNotFound(t *testing.T) {
	scheme := newTestScheme(t)
	const ns = "default"

	targetRequest := newTargetRequest("req-1", ns, "aws", "cluster-1", "https://api.example.com")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(targetRequest).
		WithStatusSubresource(targetRequest).Build()
	if err := fakeClient.Status().Update(context.Background(), targetRequest); err != nil {
		t.Fatalf("failed to set target request status: %v", err)
	}

	reconciler := &KrknScenarioRunReconciler{Client: fakeClient, Scheme: scheme, Namespace: ns}
	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "run-1", Namespace: ns},
		Spec:       krknv1alpha1.KrknScenarioRunSpec{TargetRequestID: "req-1"},
	}

	got := reconciler.getClusterAPIURL(context.Background(), scenarioRun, "aws", "other-cluster")
	if got != "" {
		t.Errorf("getClusterAPIURL() = %q, want empty string when cluster absent", got)
	}
}

// --- Reconcile integration test ---

// TestReconcile_FailedJobStatus_IncludesClusterAPIURL verifies that when createClusterJob
// fails (here: no kubeconfig Secret exists), the Failed ClusterJobStatus appended to the run
// still carries ClusterAPIURL so that non-admin users can see the failure.
//
// Before the fix this field was always empty for creation failures, causing the API layer to
// skip the job during permission checks and making the run appear stuck or invisible.
func TestReconcile_FailedJobStatus_IncludesClusterAPIURL(t *testing.T) {
	scheme := newTestScheme(t)
	const (
		ns           = "default"
		runName      = "test-run"
		targetID     = "req-1"
		providerName = "aws"
		clusterName  = "prod-cluster"
		wantAPIURL   = "https://api.prod-cluster.example.com:6443"
	)

	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: runName, Namespace: ns},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			ScenarioName:    "pod-disruption",
			TargetRequestID: targetID,
			TargetClusters:  map[string][]string{providerName: {clusterName}},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			// Pre-initialised so the reconciler skips the init block and goes straight to
			// job creation. Without this it would try to Status().Update() during init,
			// which resets TotalTargets before the test assertions run.
			Phase:       "Pending",
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{},
		},
	}

	targetRequest := newTargetRequest(targetID, ns, providerName, clusterName, wantAPIURL)

	// No Secret → getKubeconfigFromProvider fails → createClusterJob fails → error handler runs.
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&krknv1alpha1.KrknScenarioRun{}, &krknv1alpha1.KrknTargetRequest{}).
		WithObjects(scenarioRun, targetRequest).
		Build()

	// Populate KrknTargetRequest.Status (WithObjects ignores status by default for subresources).
	if err := fakeClient.Status().Update(context.Background(), targetRequest); err != nil {
		t.Fatalf("failed to seed KrknTargetRequest status: %v", err)
	}

	reconciler := &KrknScenarioRunReconciler{
		Client:    fakeClient,
		Scheme:    scheme,
		Namespace: ns,
	}

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: runName, Namespace: ns},
	})
	// Reconcile is allowed to return an error (e.g. from a failed status update in
	// the test environment), but the in-cluster object should reflect the failed job.
	// We intentionally do not t.Fatal here — the status check below is the real assertion.
	_ = err

	// Re-fetch the run to inspect persisted status.
	var updated krknv1alpha1.KrknScenarioRun
	if fetchErr := fakeClient.Get(context.Background(),
		types.NamespacedName{Name: runName, Namespace: ns},
		&updated,
	); fetchErr != nil {
		t.Fatalf("failed to re-fetch KrknScenarioRun after reconcile: %v", fetchErr)
	}

	if len(updated.Status.ClusterJobs) == 0 {
		t.Fatal("expected at least one ClusterJobStatus after job-creation failure, got none")
	}

	job := updated.Status.ClusterJobs[0]
	if job.Phase != "Failed" {
		t.Errorf("job.Phase = %q, want %q", job.Phase, "Failed")
	}
	if job.ClusterAPIURL != wantAPIURL {
		t.Errorf("job.ClusterAPIURL = %q, want %q — non-admin users will not see this failed job",
			job.ClusterAPIURL, wantAPIURL)
	}
	if job.ProviderName != providerName {
		t.Errorf("job.ProviderName = %q, want %q", job.ProviderName, providerName)
	}
	if job.ClusterName != clusterName {
		t.Errorf("job.ClusterName = %q, want %q", job.ClusterName, clusterName)
	}
	if job.FailureReason != "JobCreationFailed" {
		t.Errorf("job.FailureReason = %q, want %q", job.FailureReason, "JobCreationFailed")
	}
	if job.CompletionTime == nil {
		t.Error("job.CompletionTime should be set on creation failure")
	}
}
