package websocket

import (
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// buildScenarioRunResponse builds the SAME response as REST API (lightweight, no clusterJobs)
func buildScenarioRunResponse(run *krknv1alpha1.KrknScenarioRun) ScenarioRunStatusResponse {
	return ScenarioRunStatusResponse{
		ScenarioRunName: run.Name,
		Phase:           run.Status.Phase,
		TotalTargets:    run.Status.TotalTargets,
		SuccessfulJobs:  run.Status.SuccessfulJobs,
		FailedJobs:      run.Status.FailedJobs,
		RunningJobs:     run.Status.RunningJobs,
		ClusterJobs:     nil, // Omitted for list view
		OwnerUserID:     run.Spec.OwnerUserID,
		RegistryName:    run.Spec.RegistryName,
		GraphRunName:    run.Labels["krkn.dev/graph-run"],
		GraphNodeID:     run.Labels["krkn.dev/graph-node"],
		CreationTimestamp: run.CreationTimestamp.Format(time.RFC3339),
	}
}

// buildScenarioRunDetailResponse builds FULL response with clusterJobs for detail view
func buildScenarioRunDetailResponse(run *krknv1alpha1.KrknScenarioRun) ScenarioRunStatusResponse {
	return ScenarioRunStatusResponse{
		ScenarioRunName:  run.Name,
		Phase:            run.Status.Phase,
		TotalTargets:     run.Status.TotalTargets,
		SuccessfulJobs:   run.Status.SuccessfulJobs,
		FailedJobs:       run.Status.FailedJobs,
		RunningJobs:      run.Status.RunningJobs,
		ClusterJobs:      run.Status.ClusterJobs, // Include full clusterJobs for detail
		OwnerUserID:      run.Spec.OwnerUserID,
		RegistryName:     run.Spec.RegistryName,
		GraphRunName:     run.Labels["krkn.dev/graph-run"],
		GraphNodeID:      run.Labels["krkn.dev/graph-node"],
		CreationTimestamp: run.CreationTimestamp.Format(time.RFC3339),
	}
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
		CreationTimestamp:  run.CreationTimestamp.Format(time.RFC3339),
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
