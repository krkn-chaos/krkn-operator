package websocket

import (
	"encoding/json"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestScenarioRunWebSocketResponsesIncludeRollbackIdentity(t *testing.T) {
	run := krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "node-cpu-hog-953e3861"},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			RollbackEnabled: true,
			RollbackSource:  &krknv1alpha1.RollbackSource{RunName: "source-run", JobID: "attempt"},
		},
	}
	jobs := buildUnifiedJobList([]krknv1alpha1.KrknScenarioRun{run}, nil)
	if len(jobs) != 1 || jobs[0].ScenarioRun == nil {
		t.Fatalf("expected one scenario run in jobs snapshot, got %+v", jobs)
	}
	for name, response := range map[string]ScenarioRunStatusResponse{
		"jobs":       *jobs[0].ScenarioRun,
		"run-detail": buildScenarioRunDetailResponse(&run),
	} {
		encoded, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("marshal %s response: %v", name, err)
		}
		var payload map[string]any
		if err := json.Unmarshal(encoded, &payload); err != nil {
			t.Fatalf("decode %s response: %v", name, err)
		}
		if payload["rollbackEnabled"] != true {
			t.Errorf("%s response omitted rollbackEnabled: %s", name, encoded)
		}
		if payload["isRollback"] != true {
			t.Errorf("%s response omitted isRollback: %s", name, encoded)
		}
	}
}

func TestGroupRollbackJobsWebSocketKeepsFlatViewAndChecksUID(t *testing.T) {
	parent := krknv1alpha1.KrknScenarioRun{ObjectMeta: metav1.ObjectMeta{Name: "source", UID: "source-uid"}}
	child := krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "rollback"},
		Spec:       krknv1alpha1.KrknScenarioRunSpec{RollbackSource: &krknv1alpha1.RollbackSource{RunName: parent.Name, RunUID: string(parent.UID), JobID: "attempt"}},
		Status:     krknv1alpha1.KrknScenarioRunStatus{Phase: "Succeeded"},
	}
	wrongUID := child.DeepCopy()
	wrongUID.Name = "old-rollback"
	wrongUID.Spec.RollbackSource.RunUID = "old-uid"
	runs := []krknv1alpha1.KrknScenarioRun{parent, child, *wrongUID}
	flat := buildUnifiedJobList(runs, nil)
	grouped := groupRollbackJobs(flat, runs)
	if len(grouped) != 2 || len(flat) != 3 {
		t.Fatalf("expected two grouped and three flat jobs, got %d and %d", len(grouped), len(flat))
	}
	for _, job := range grouped {
		if job.Name == "rollback" {
			t.Fatal("linked rollback remained at top level")
		}
		if job.Name == "source" && (len(job.RollbackRuns) != 1 || job.RollbackRuns[0].SourceJobID != "attempt") {
			t.Fatalf("source has wrong rollback children: %+v", job.RollbackRuns)
		}
	}
	for _, job := range flat {
		if len(job.RollbackRuns) != 0 {
			t.Fatal("grouping mutated the flat Jobs view")
		}
	}
}

func TestConvertClusterJobs_OmitsClusterAPIURL(t *testing.T) {
	now := metav1.Now()
	jobs := []krknv1alpha1.ClusterJobStatus{
		{
			ProviderName:    "prov-1",
			ClusterName:     "cluster-1",
			ClusterAPIURL:   "https://secret-api.example.com:6443",
			JobID:           "job-1",
			PodName:         "pod-1",
			ScenarioImage:   "quay.io/krkn/scenario:latest",
			AttemptHistory:  []krknv1alpha1.ScenarioAttemptStatus{{JobID: "previous-job", ScenarioImage: "quay.io/krkn/scenario:previous"}},
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
	if len(r.AttemptHistory) != 1 || r.AttemptHistory[0].JobID != "previous-job" {
		t.Errorf("Expected previous attempt in WebSocket response, got %+v", r.AttemptHistory)
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
