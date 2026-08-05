package websocket

import (
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// buildScenarioRunResponse builds the response with sanitized clusterJobs (no ClusterAPIURL).
// Used for both lightweight "run" broadcasts and "run" snapshots.
func buildScenarioRunResponse(run *krknv1alpha1.KrknScenarioRun) ScenarioRunStatusResponse {
	return ScenarioRunStatusResponse{
		ScenarioRunName:   run.Name,
		Phase:             run.Status.Phase,
		TotalTargets:      run.Status.TotalTargets,
		SuccessfulJobs:    run.Status.SuccessfulJobs,
		FailedJobs:        run.Status.FailedJobs,
		RunningJobs:       run.Status.RunningJobs,
		ClusterJobs:       sanitizedClusterJobs(run.Status.ClusterJobs),
		OwnerUserID:       run.Spec.OwnerUserID,
		RegistryName:      run.Spec.RegistryName,
		GraphRunName:      run.Labels["krkn.dev/graph-run"],
		GraphNodeID:       run.Labels["krkn.dev/graph-node"],
		CreationTimestamp: run.CreationTimestamp.Format(time.RFC3339),
	}
}

// buildScenarioRunDetailResponse builds FULL response with sanitized clusterJobs for detail view.
func buildScenarioRunDetailResponse(run *krknv1alpha1.KrknScenarioRun) ScenarioRunStatusResponse {
	return ScenarioRunStatusResponse{
		ScenarioRunName:   run.Name,
		Phase:             run.Status.Phase,
		TotalTargets:      run.Status.TotalTargets,
		SuccessfulJobs:    run.Status.SuccessfulJobs,
		FailedJobs:        run.Status.FailedJobs,
		RunningJobs:       run.Status.RunningJobs,
		ClusterJobs:       sanitizedClusterJobs(run.Status.ClusterJobs),
		OwnerUserID:       run.Spec.OwnerUserID,
		RegistryName:      run.Spec.RegistryName,
		GraphRunName:      run.Labels["krkn.dev/graph-run"],
		GraphNodeID:       run.Labels["krkn.dev/graph-node"],
		CreationTimestamp: run.CreationTimestamp.Format(time.RFC3339),
	}
}

// sanitizedClusterJobs returns an interface{}-typed nil when there are no jobs,
// avoiding the Go typed-nil-in-interface pitfall with omitempty.
func sanitizedClusterJobs(jobs []krknv1alpha1.ClusterJobStatus) interface{} {
	converted := convertClusterJobs(jobs)
	if converted == nil {
		return nil
	}
	return converted
}

// convertClusterJobs converts raw CRD ClusterJobStatus to sanitized ClusterJobResponse,
// omitting ClusterAPIURL and LastRetryTime to match the REST API contract.
func convertClusterJobs(jobs []krknv1alpha1.ClusterJobStatus) []ClusterJobResponse {
	if len(jobs) == 0 {
		return nil
	}
	result := make([]ClusterJobResponse, len(jobs))
	for i, job := range jobs {
		result[i] = ClusterJobResponse{
			ProviderName:    job.ProviderName,
			ClusterName:     job.ClusterName,
			JobID:           job.JobID,
			PodName:         job.PodName,
			ContainerImage:  job.ContainerImage,
			Phase:           job.Phase,
			StartTime:       convertMetaTime(job.StartTime),
			CompletionTime:  convertMetaTime(job.CompletionTime),
			Message:         job.Message,
			RetryCount:      job.RetryCount,
			MaxRetries:      job.MaxRetries,
			CancelRequested: job.CancelRequested,
			FailureReason:   job.FailureReason,
		}
	}
	return result
}

func convertMetaTime(t *metav1.Time) *time.Time {
	if t == nil {
		return nil
	}
	result := t.Time
	return &result
}

// buildGraphRunResponse builds the SAME response as REST API
func buildGraphRunResponse(run *krknv1alpha1.KrknGraphRun) GraphRunResponse {
	return GraphRunResponse{
		GraphRunName: run.Name,
		Phase:        run.Status.Phase,
		Summary: GraphRunSummaryResponse{
			TotalNodes:     run.Status.Summary.TotalNodes,
			CompletedNodes: run.Status.Summary.CompletedNodes,
			RunningNodes:   run.Status.Summary.RunningNodes,
			FailedNodes:    run.Status.Summary.FailedNodes,
			PendingNodes:   run.Status.Summary.PendingNodes,
		},
		NodeStatuses:      nil,
		ResolvedLevels:    run.Status.ResolvedLevels,
		StartTime:         run.Status.StartTime,
		CompletionTime:    run.Status.CompletionTime,
		OwnerUserID:       run.Spec.OwnerUserID,
		CreationTimestamp: run.CreationTimestamp.Format(time.RFC3339),
		ResiliencyScores:  convertGraphClusterScoresForSnapshot(run.Status.ResiliencyScores),
	}
}

func convertGraphClusterScoresForSnapshot(scores []krknv1alpha1.GraphClusterScore) []GraphClusterScoreResponse {
	if scores == nil {
		return nil
	}
	result := make([]GraphClusterScoreResponse, len(scores))
	for i, score := range scores {
		result[i] = GraphClusterScoreResponse{
			ProviderName:      score.ProviderName,
			ClusterName:       score.ClusterName,
			Calculated:        score.Calculated,
			Baseline:          score.Baseline,
			Status:            score.Status,
			Message:           score.Message,
			NodeContributions: score.NodeContributions,
		}
	}
	return result
}
