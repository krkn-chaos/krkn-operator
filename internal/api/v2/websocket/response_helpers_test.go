package websocket

import (
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvertClusterJobs_OmitsClusterAPIURL(t *testing.T) {
	now := metav1.Now()
	jobs := []krknv1alpha1.ClusterJobStatus{
		{
			ProviderName:    "prov-1",
			ClusterName:     "cluster-1",
			ClusterAPIURL:   "https://secret-api.example.com:6443",
			JobID:           "job-1",
			PodName:         "pod-1",
			ContainerImage:  "quay.io/krkn/scenario:latest",
			Phase:           "Succeeded",
			StartTime:       &now,
			CompletionTime:  &now,
			Message:         "completed",
			RetryCount:      1,
			MaxRetries:      3,
			CancelRequested: false,
			FailureReason:   "",
		},
	}

	result := convertClusterJobs(jobs)

	if len(result) != 1 {
		t.Fatalf("Expected 1 job, got %d", len(result))
	}

	r := result[0]
	if r.ProviderName != "prov-1" {
		t.Errorf("Expected ProviderName 'prov-1', got '%s'", r.ProviderName)
	}
	if r.ClusterName != "cluster-1" {
		t.Errorf("Expected ClusterName 'cluster-1', got '%s'", r.ClusterName)
	}
	if r.JobID != "job-1" {
		t.Errorf("Expected JobID 'job-1', got '%s'", r.JobID)
	}
	if r.Phase != "Succeeded" {
		t.Errorf("Expected Phase 'Succeeded', got '%s'", r.Phase)
	}
	if r.StartTime == nil {
		t.Error("Expected StartTime to be set")
	}
	if r.CompletionTime == nil {
		t.Error("Expected CompletionTime to be set")
	}
	if r.Message != "completed" {
		t.Errorf("Expected Message 'completed', got '%s'", r.Message)
	}
	if r.RetryCount != 1 {
		t.Errorf("Expected RetryCount 1, got %d", r.RetryCount)
	}
}

func TestConvertClusterJobs_NilInput(t *testing.T) {
	result := convertClusterJobs(nil)
	if result != nil {
		t.Errorf("Expected nil for nil input, got %v", result)
	}
}

func TestConvertClusterJobs_EmptyInput(t *testing.T) {
	result := convertClusterJobs([]krknv1alpha1.ClusterJobStatus{})
	if result != nil {
		t.Errorf("Expected nil for empty input, got %v", result)
	}
}

func TestBuildScenarioRunResponse_IncludesClusterJobs(t *testing.T) {
	run := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "run-1",
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase:        "Running",
			TotalTargets: 1,
			RunningJobs:  1,
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{
					ClusterName:   "cluster-1",
					ClusterAPIURL: "https://secret.example.com",
					JobID:         "job-1",
					Phase:         "Running",
				},
			},
		},
	}

	resp := buildScenarioRunResponse(run)

	jobs, ok := resp.ClusterJobs.([]ClusterJobResponse)
	if !ok {
		t.Fatalf("Expected ClusterJobs to be []ClusterJobResponse, got %T", resp.ClusterJobs)
	}
	if len(jobs) != 1 {
		t.Fatalf("Expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Phase != "Running" {
		t.Errorf("Expected phase 'Running', got '%s'", jobs[0].Phase)
	}
}

func TestBuildScenarioRunResponse_NoClusterJobs(t *testing.T) {
	run := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "run-1",
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Pending",
		},
	}

	resp := buildScenarioRunResponse(run)

	if resp.ClusterJobs != nil {
		t.Errorf("Expected nil ClusterJobs for run with no jobs, got %v", resp.ClusterJobs)
	}
}
