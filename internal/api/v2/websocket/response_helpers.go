package websocket

import (
	"math"
	"sort"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/api/jobstats"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WSUnifiedJobItem represents a single item in the unified jobs list for WebSocket responses.
// Uses the same typed envelope as REST UnifiedJobItem for frontend compatibility.
type WSUnifiedJobItem struct {
	Type        string                     `json:"type"` // "scenarioRun" or "graphRun"
	Name        string                     `json:"name"`
	CreatedAt   string                     `json:"createdAt"` // RFC3339 timestamp
	ScenarioRun *ScenarioRunStatusResponse `json:"scenarioRun,omitempty"`
	GraphRun    *GraphRunResponse          `json:"graphRun,omitempty"`
}

func (w WSUnifiedJobItem) JobType() string            { return w.Type }
func (w WSUnifiedJobItem) ScenarioSucceeded() int      { if w.ScenarioRun != nil { return w.ScenarioRun.SuccessfulJobs }; return 0 }
func (w WSUnifiedJobItem) ScenarioFailed() int         { if w.ScenarioRun != nil { return w.ScenarioRun.FailedJobs }; return 0 }
func (w WSUnifiedJobItem) ScenarioRunning() int        { if w.ScenarioRun != nil { return w.ScenarioRun.RunningJobs }; return 0 }
func (w WSUnifiedJobItem) ScenarioTotalTargets() int   { if w.ScenarioRun != nil { return w.ScenarioRun.TotalTargets }; return 0 }
func (w WSUnifiedJobItem) GraphTotal() int             { if w.GraphRun != nil { return w.GraphRun.Summary.TotalNodes }; return 0 }
func (w WSUnifiedJobItem) GraphCompleted() int         { if w.GraphRun != nil { return w.GraphRun.Summary.CompletedNodes }; return 0 }
func (w WSUnifiedJobItem) GraphFailed() int            { if w.GraphRun != nil { return w.GraphRun.Summary.FailedNodes }; return 0 }

// WSUnifiedJobsSnapshot represents the paginated jobs snapshot sent to WebSocket clients.
type WSUnifiedJobsSnapshot struct {
	Jobs  []WSUnifiedJobItem `json:"jobs"`
	Stats WSJobStatsSummary  `json:"stats"`
}

// buildUnifiedJobList merges standalone ScenarioRuns and GraphRuns into a unified sorted list.
func buildUnifiedJobList(scenarioRuns []krknv1alpha1.KrknScenarioRun, graphRuns []krknv1alpha1.KrknGraphRun) []WSUnifiedJobItem {
	jobs := make([]WSUnifiedJobItem, 0, len(scenarioRuns)+len(graphRuns))

	for i := range scenarioRuns {
		sr := &scenarioRuns[i]
		if sr.Labels["krkn.dev/graph-run"] != "" {
			continue
		}
		resp := buildScenarioRunResponse(sr)
		jobs = append(jobs, WSUnifiedJobItem{
			Type:        "scenarioRun",
			Name:        sr.Name,
			CreatedAt:   sr.CreationTimestamp.Format(time.RFC3339),
			ScenarioRun: &resp,
		})
	}

	for i := range graphRuns {
		gr := &graphRuns[i]
		resp := buildGraphRunResponse(gr)
		jobs = append(jobs, WSUnifiedJobItem{
			Type:      "graphRun",
			Name:      gr.Name,
			CreatedAt: gr.CreationTimestamp.Format(time.RFC3339),
			GraphRun:  &resp,
		})
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt > jobs[j].CreatedAt
	})

	return jobs
}

// paginateJobItems returns a page of items and pagination metadata.
func paginateJobItems(items []WSUnifiedJobItem, page, limit int) ([]WSUnifiedJobItem, WSPaginationMeta) {
	total := len(items)
	totalPages := 0
	if limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	meta := WSPaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	if limit <= 0 || page <= 0 {
		return items, meta
	}

	offset := (page - 1) * limit
	if offset >= total {
		return []WSUnifiedJobItem{}, meta
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return items[offset:end], meta
}

// computeWSJobStats computes aggregate job statistics from the full unified job list.
func computeWSJobStats(jobs []WSUnifiedJobItem) WSJobStatsSummary {
	s := jobstats.Compute(jobs)
	return WSJobStatsSummary(s)
}

// buildScenarioRunResponse builds the response with sanitized clusterJobs (no ClusterAPIURL).
// Used for both lightweight "run" broadcasts and "run" snapshots.
func buildScenarioRunResponse(run *krknv1alpha1.KrknScenarioRun) ScenarioRunStatusResponse {
	return ScenarioRunStatusResponse{
		ScenarioRunName:   run.Name,
		ScenarioName:      run.Spec.ScenarioName,
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
		CustomRunName:     run.Spec.CustomRunName,
		CreationTimestamp: run.CreationTimestamp.Format(time.RFC3339),
	}
}

// buildScenarioRunDetailResponse builds FULL response with sanitized clusterJobs for detail view.
func buildScenarioRunDetailResponse(run *krknv1alpha1.KrknScenarioRun) ScenarioRunStatusResponse {
	return ScenarioRunStatusResponse{
		ScenarioRunName:   run.Name,
		ScenarioName:      run.Spec.ScenarioName,
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
		CustomRunName:     run.Spec.CustomRunName,
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
		Name:         run.Name,
		Phase:        run.Status.Phase,
		Summary: GraphRunSummaryResponse{
			TotalNodes:     run.Status.Summary.TotalNodes,
			CompletedNodes: run.Status.Summary.CompletedNodes,
			RunningNodes:   run.Status.Summary.RunningNodes,
			FailedNodes:    run.Status.Summary.FailedNodes,
			PendingNodes:   run.Status.Summary.PendingNodes,
		},
		NodeStatuses:            nil,
		ResolvedLevels:          run.Status.ResolvedLevels,
		StartTime:               run.Status.StartTime,
		CompletionTime:          run.Status.CompletionTime,
		OwnerUserID:             run.Spec.OwnerUserID,
		CreationTimestamp:       run.CreationTimestamp.Format(time.RFC3339),
		ResiliencyScores:        convertGraphClusterScoresForSnapshot(run.Status.ResiliencyScores),
		ResiliencyScoreEnabled:  run.Spec.ResiliencyScoreEnabled,
		ResiliencyScoreBaseline: run.Spec.ResiliencyScoreBaseline,
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
