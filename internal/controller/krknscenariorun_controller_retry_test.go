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
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func TestUpdateClusterJobStatuses_ProviderNameEmpty(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	now := metav1.Now()

	// Create a failed pod to trigger retry logic
	failedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-123",
			Namespace: "default",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
		},
	}

	// Create ScenarioRun with job that has empty ProviderName
	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-scenario",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			ScenarioName:    "test-scenario",
			TargetRequestId: "test-uuid",
			MaxRetries:      3,
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{
					ProviderName:  "", // Empty ProviderName - should be caught
					ClusterName:   "cluster1",
					JobId:         "job-123",
					PodName:       "pod-123",
					Phase:         "Running", // Start as Running so it transitions to Failed
					StartTime:     &now,
					RetryCount:    0,
					MaxRetries:    3,
					LastRetryTime: nil,
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(scenarioRun, failedPod).Build()

	reconciler := &KrknScenarioRunReconciler{
		Client:    fakeClient,
		Scheme:    scheme,
		Namespace: "default",
	}

	ctx := context.Background()
	err := reconciler.updateClusterJobStatuses(ctx, scenarioRun)

	// Should not error even with empty ProviderName
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Job should be marked as Failed with appropriate message
	job := scenarioRun.Status.ClusterJobs[0]
	if job.Phase != "Failed" {
		t.Errorf("Expected job phase 'Failed', got '%s'", job.Phase)
	}

	expectedMsg := "Retry failed: ProviderName is empty"
	if job.Message != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, job.Message)
	}

	if job.FailureReason != "InvalidJobState" {
		t.Errorf("Expected FailureReason 'InvalidJobState', got '%s'", job.FailureReason)
	}

	if job.CompletionTime == nil {
		t.Error("Expected CompletionTime to be set")
	}
}

func TestUpdateClusterJobStatuses_ClusterNameEmpty(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	now := metav1.Now()

	// Create a failed pod to trigger retry logic
	failedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-456",
			Namespace: "default",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
		},
	}

	// Create ScenarioRun with job that has empty ClusterName
	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-scenario",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			ScenarioName:    "test-scenario",
			TargetRequestId: "test-uuid",
			MaxRetries:      3,
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{
					ProviderName:  "krkn-operator",
					ClusterName:   "", // Empty ClusterName - should be caught
					JobId:         "job-456",
					PodName:       "pod-456",
					Phase:         "Running", // Start as Running so it transitions to Failed
					StartTime:     &now,
					RetryCount:    0,
					MaxRetries:    3,
					LastRetryTime: nil,
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(scenarioRun, failedPod).Build()

	reconciler := &KrknScenarioRunReconciler{
		Client:    fakeClient,
		Scheme:    scheme,
		Namespace: "default",
	}

	ctx := context.Background()
	err := reconciler.updateClusterJobStatuses(ctx, scenarioRun)

	// Should not error even with empty ClusterName
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Job should be marked as Failed with appropriate message
	job := scenarioRun.Status.ClusterJobs[0]
	if job.Phase != "Failed" {
		t.Errorf("Expected job phase 'Failed', got '%s'", job.Phase)
	}

	expectedMsg := "Retry failed: ClusterName is empty"
	if job.Message != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, job.Message)
	}

	if job.FailureReason != "InvalidJobState" {
		t.Errorf("Expected FailureReason 'InvalidJobState', got '%s'", job.FailureReason)
	}

	if job.CompletionTime == nil {
		t.Error("Expected CompletionTime to be set")
	}
}
