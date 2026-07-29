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
*/

package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// managedClustersSecret builds the "managed-clusters" Secret that
// getKubeconfigFromProvider expects, wiring one provider/cluster to a dummy
// (but validly base64-encoded) kubeconfig.
func managedClustersSecret(name, provider, cluster string) *corev1.Secret {
	kubeconfig := base64.StdEncoding.EncodeToString([]byte("dummy-kubeconfig"))
	mc := map[string]map[string]map[string]string{
		provider: {cluster: {"kubeconfig": kubeconfig}},
	}
	mcBytes, _ := json.Marshal(mc)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Data:       map[string][]byte{"managed-clusters": mcBytes},
	}
}

// TestCreateClusterJob_WiresActiveDeadlineSeconds proves the end-to-end wiring:
// spec.duration on the CR must land on the created pod's ActiveDeadlineSeconds,
// and an empty duration must leave the pod unbounded (nil).
func TestCreateClusterJob_WiresActiveDeadlineSeconds(t *testing.T) {
	tests := []struct {
		name         string
		duration     string
		wantDeadline *int64
	}{
		{name: "duration sets deadline", duration: "30s", wantDeadline: ptrInt64(30)},
		{name: "empty leaves pod unbounded", duration: "", wantDeadline: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = krknv1alpha1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)

			scenarioRun := &krknv1alpha1.KrknScenarioRun{
				ObjectMeta: metav1.ObjectMeta{Name: "test-scenario", Namespace: "default"},
				Spec: krknv1alpha1.KrknScenarioRunSpec{
					ScenarioName:    "test-scenario",
					TargetRequestID: "test-uuid",
					Duration:        tt.duration,
				},
			}
			secret := managedClustersSecret("test-uuid", "test-provider", "test-cluster")

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(scenarioRun, secret).Build()
			reconciler := &KrknScenarioRunReconciler{
				Client:    fakeClient,
				Scheme:    scheme,
				Namespace: "default",
			}

			if err := reconciler.createClusterJob(context.Background(), scenarioRun, "test-provider", "test-cluster"); err != nil {
				t.Fatalf("createClusterJob returned error: %v", err)
			}

			if len(scenarioRun.Status.ClusterJobs) != 1 {
				t.Fatalf("expected 1 cluster job, got %d", len(scenarioRun.Status.ClusterJobs))
			}

			var pod corev1.Pod
			if err := fakeClient.Get(context.Background(), types.NamespacedName{
				Name:      scenarioRun.Status.ClusterJobs[0].PodName,
				Namespace: "default",
			}, &pod); err != nil {
				t.Fatalf("failed to fetch created pod: %v", err)
			}

			got := pod.Spec.ActiveDeadlineSeconds
			switch {
			case tt.wantDeadline == nil && got != nil:
				t.Fatalf("expected no deadline, got %d", *got)
			case tt.wantDeadline != nil && got == nil:
				t.Fatalf("expected deadline %d, got nil", *tt.wantDeadline)
			case tt.wantDeadline != nil && got != nil && *got != *tt.wantDeadline:
				t.Fatalf("expected deadline %d, got %d", *tt.wantDeadline, *got)
			}
		})
	}
}

// TestUpdateClusterJobStatuses_DeadlineExceeded_NotRetried proves that a pod
// killed for exceeding its activeDeadlineSeconds is a terminal failure: it is
// not retried, and the deadline is surfaced as the failure reason/message.
func TestUpdateClusterJobStatuses_DeadlineExceeded_NotRetried(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	now := metav1.Now()
	const deadlineMsg = "Pod was active on the node longer than the specified deadline"

	// Kubernetes marks a deadline-killed pod Failed with a pod-level reason.
	deadlinePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod-deadline", Namespace: "default"},
		Status: corev1.PodStatus{
			Phase:   corev1.PodFailed,
			Reason:  "DeadlineExceeded",
			Message: deadlineMsg,
		},
	}

	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "test-scenario", Namespace: "default"},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			ScenarioName:    "test-scenario",
			TargetRequestID: "test-uuid",
			Duration:        "30s",
			MaxRetries:      3,
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{
					ProviderName: "test-provider",
					ClusterName:  "test-cluster",
					JobID:        "job-deadline",
					PodName:      "pod-deadline",
					Phase:        "Running",
					StartTime:    &now,
					RetryCount:   0,
					MaxRetries:   3,
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(scenarioRun, deadlinePod).Build()
	reconciler := &KrknScenarioRunReconciler{
		Client:    fakeClient,
		Scheme:    scheme,
		Namespace: "default",
	}

	if err := reconciler.updateClusterJobStatuses(context.Background(), scenarioRun); err != nil {
		t.Fatalf("updateClusterJobStatuses returned error: %v", err)
	}

	job := scenarioRun.Status.ClusterJobs[0]
	if job.Phase != "Failed" {
		t.Errorf("expected phase 'Failed', got '%s'", job.Phase)
	}
	if job.FailureReason != "DeadlineExceeded" {
		t.Errorf("expected FailureReason 'DeadlineExceeded', got '%s'", job.FailureReason)
	}
	// Not retried: a retry would have set phase 'Retrying' and bumped RetryCount.
	if job.RetryCount != 0 {
		t.Errorf("expected RetryCount 0 (no retry), got %d", job.RetryCount)
	}
	if job.Message != deadlineMsg {
		t.Errorf("expected deadline message surfaced, got '%s'", job.Message)
	}
	if job.CompletionTime == nil {
		t.Error("expected CompletionTime to be set")
	}
}
