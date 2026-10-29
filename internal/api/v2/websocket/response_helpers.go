package websocket

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/api/jobstats"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type categoryVisibilityChecker interface {
	VisibleCategoryNames(context.Context) (map[string]struct{}, error)
}

func visibleCategoryNames(ctx context.Context, checker AuthorizationChecker) (map[string]struct{}, error) {
	categoryChecker, ok := checker.(categoryVisibilityChecker)
	if !ok {
		return map[string]struct{}{}, nil
	}
	visible, err := categoryChecker.VisibleCategoryNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("get visible category names: %w", err)
	}
	return visible, nil
}

// WSUnifiedJobItem represents a single item in the unified jobs list for WebSocket responses.
// Uses the same typed envelope as REST UnifiedJobItem for frontend compatibility.
type WSUnifiedJobItem struct {
	Type        string                     `json:"type"` // "scenarioRun" or "graphRun"
	Name        string                     `json:"name"`
	CreatedAt   string                     `json:"createdAt"` // RFC3339 timestamp
	Categories  []string                   `json:"categories,omitempty"`
	ScenarioRun *ScenarioRunStatusResponse `json:"scenarioRun,omitempty"`
	GraphRun    *GraphRunResponse          `json:"graphRun,omitempty"`
}

func (w WSUnifiedJobItem) JobType() string { return w.Type }
func (w WSUnifiedJobItem) ScenarioSucceeded() int {
	if w.ScenarioRun != nil {
		return w.ScenarioRun.SuccessfulJobs
	}
	return 0
}
func (w WSUnifiedJobItem) ScenarioFailed() int {
	if w.ScenarioRun != nil {
		return w.ScenarioRun.FailedJobs
	}
	return 0
}
func (w WSUnifiedJobItem) ScenarioRunning() int {
	if w.ScenarioRun != nil {
		return w.ScenarioRun.RunningJobs
	}
	return 0
}
func (w WSUnifiedJobItem) ScenarioTotalTargets() int {
	if w.ScenarioRun != nil {
		return w.ScenarioRun.TotalTargets
	}
	return 0
}
func (w WSUnifiedJobItem) GraphTotal() int {
	if w.GraphRun != nil {
		return w.GraphRun.Summary.TotalNodes
	}
	return 0
}
func (w WSUnifiedJobItem) GraphCompleted() int {
	if w.GraphRun != nil {
		return w.GraphRun.Summary.CompletedNodes
	}
	return 0
}
func (w WSUnifiedJobItem) GraphFailed() int {
	if w.GraphRun != nil {
		return w.GraphRun.Summary.FailedNodes
	}
	return 0
}

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
			Categories:  runCategoryNames(sr.Labels),
			ScenarioRun: &resp,
		})
	}

	for i := range graphRuns {
		gr := &graphRuns[i]
		resp := buildGraphRunResponse(gr)
		jobs = append(jobs, WSUnifiedJobItem{
			Type:       "graphRun",
			Name:       gr.Name,
			CreatedAt:  gr.CreationTimestamp.Format(time.RFC3339),
			Categories: runCategoryNames(gr.Labels),
			GraphRun:   &resp,
		})
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt > jobs[j].CreatedAt
	})

	return jobs
}

func runCategoryNames(labels map[string]string) []string {
	names := make([]string, 0, len(labels))
	for key, value := range labels {
		if value != "true" || !strings.HasPrefix(key, krknv1alpha1.CategoryEntityLabelPrefix) {
			continue
		}
		names = append(names, strings.TrimPrefix(key, krknv1alpha1.CategoryEntityLabelPrefix))
	}
	sort.Strings(names)
	return names
}

func filterJobCategoriesByVisibility(jobs []WSUnifiedJobItem, visible map[string]struct{}) {
	for i := range jobs {
		visibleNames := jobs[i].Categories[:0]
		for _, name := range jobs[i].Categories {
			if _, ok := visible[name]; ok {
				visibleNames = append(visibleNames, name)
			}
		}
		if len(visibleNames) == 0 {
			jobs[i].Categories = nil
		} else {
			jobs[i].Categories = visibleNames
		}
	}
}

// filterJobsByCategories returns jobs that match at least one requested category.
// An empty selection leaves the job list unchanged.
func filterJobsByCategories(jobs []WSUnifiedJobItem, categories []string) []WSUnifiedJobItem {
	if len(categories) == 0 {
		return jobs
	}

	selected := make(map[string]struct{}, len(categories))
	for _, category := range categories {
		if category != "" {
			selected[category] = struct{}{}
		}
	}
	if len(selected) == 0 {
		return jobs
	}

	filtered := jobs[:0]
	for _, job := range jobs {
		for _, category := range job.Categories {
			if _, ok := selected[category]; ok {
				filtered = append(filtered, job)
				break
			}
		}
	}
	return filtered
}

// sortUnifiedJobItems sorts the full filtered jobs slice before pagination. An
// omitted or unsupported sort keeps buildUnifiedJobList's createdAt-descending
// order, preserving behavior for older WebSocket clients.
func sortUnifiedJobItems(jobs []WSUnifiedJobItem, sortBy, sortOrder string) {
	if (sortOrder != "asc" && sortOrder != "desc") || !isJobsSortField(sortBy) {
		return
	}

	sort.SliceStable(jobs, func(i, j int) bool {
		comparison := compareJobsByField(jobs[i], jobs[j], sortBy)
		if sortOrder == "desc" {
			return comparison > 0
		}
		return comparison < 0
	})
}

func isJobsSortField(sortBy string) bool {
	switch sortBy {
	case "createdAt", "status", "name", "owner":
		return true
	default:
		return false
	}
}

func compareJobsByField(left, right WSUnifiedJobItem, sortBy string) int {
	switch sortBy {
	case "createdAt":
		leftTime, leftErr := time.Parse(time.RFC3339, left.CreatedAt)
		rightTime, rightErr := time.Parse(time.RFC3339, right.CreatedAt)
		if leftErr != nil {
			leftTime = time.Time{}
		}
		if rightErr != nil {
			rightTime = time.Time{}
		}
		if leftTime.Before(rightTime) {
			return -1
		}
		if leftTime.After(rightTime) {
			return 1
		}
	case "status":
		return compareNaturalStrings(jobSortStatus(left), jobSortStatus(right))
	case "name":
		return compareNaturalStrings(jobSortName(left), jobSortName(right))
	case "owner":
		return compareNaturalStrings(jobSortOwner(left), jobSortOwner(right))
	}
	return 0
}

func jobSortStatus(job WSUnifiedJobItem) string {
	if job.GraphRun != nil {
		if job.GraphRun.Phase == "Completed" {
			return "Succeeded"
		}
		return job.GraphRun.Phase
	}
	if job.ScenarioRun != nil {
		return job.ScenarioRun.Phase
	}
	return "Pending"
}

func jobSortName(job WSUnifiedJobItem) string {
	if job.GraphRun != nil || job.Type == "graphRun" {
		return job.Name
	}
	if job.ScenarioRun == nil {
		return job.Name
	}
	if customName := strings.TrimSpace(job.ScenarioRun.CustomRunName); customName != "" {
		return customName
	}
	return job.ScenarioRun.ScenarioRunName
}

func jobSortOwner(job WSUnifiedJobItem) string {
	if job.GraphRun != nil {
		return job.GraphRun.OwnerUserID
	}
	if job.ScenarioRun != nil {
		return job.ScenarioRun.OwnerUserID
	}
	return ""
}

// compareNaturalStrings mirrors the console's case-insensitive, numeric-aware
// run-name sorting without adding a locale/collation dependency.
func compareNaturalStrings(left, right string) int {
	a := []rune(strings.ToLower(left))
	b := []rune(strings.ToLower(right))
	for i, j := 0, 0; i < len(a) && j < len(b); {
		if isASCIIDigit(a[i]) && isASCIIDigit(b[j]) {
			aEnd, bEnd := i, j
			for aEnd < len(a) && isASCIIDigit(a[aEnd]) {
				aEnd++
			}
			for bEnd < len(b) && isASCIIDigit(b[bEnd]) {
				bEnd++
			}
			aNumber := strings.TrimLeft(string(a[i:aEnd]), "0")
			bNumber := strings.TrimLeft(string(b[j:bEnd]), "0")
			if len(aNumber) < len(bNumber) {
				return -1
			}
			if len(aNumber) > len(bNumber) {
				return 1
			}
			if cmp := strings.Compare(aNumber, bNumber); cmp != 0 {
				return cmp
			}
			i, j = aEnd, bEnd
			continue
		}
		if a[i] < b[j] {
			return -1
		}
		if a[i] > b[j] {
			return 1
		}
		i++
		j++
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

func isASCIIDigit(value rune) bool {
	return value >= '0' && value <= '9'
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
		ScenarioRunName:        run.Name,
		ScenarioName:           run.Spec.Scenario.Name,
		Phase:                  run.Status.Phase,
		TotalTargets:           run.Status.TotalTargets,
		SuccessfulJobs:         run.Status.SuccessfulJobs,
		FailedJobs:             run.Status.FailedJobs,
		RunningJobs:            run.Status.RunningJobs,
		ClusterJobs:            sanitizedClusterJobs(run.Status.ClusterJobs),
		OwnerUserID:            run.Spec.OwnerUserID,
		RegistryName:           run.Spec.Scenario.RegistryName,
		GraphRunName:           run.Labels["krkn.dev/graph-run"],
		GraphNodeID:            run.Labels["krkn.dev/graph-node"],
		CustomRunName:          run.Spec.CustomRunName,
		CreationTimestamp:      run.CreationTimestamp.Format(time.RFC3339),
		ResiliencyScoreEnabled: run.Spec.ResiliencyScoreEnabled,
		ResiliencyScore:        averageResiliencyScore(run.Status.ResiliencyScores),
		ResiliencyScores:       convertClusterResiliencyScoresWS(run.Status.ResiliencyScores),
	}
}

// buildScenarioRunDetailResponse builds FULL response with sanitized clusterJobs for detail view.
func buildScenarioRunDetailResponse(run *krknv1alpha1.KrknScenarioRun) ScenarioRunStatusResponse {
	return ScenarioRunStatusResponse{
		ScenarioRunName:        run.Name,
		ScenarioName:           run.Spec.Scenario.Name,
		Phase:                  run.Status.Phase,
		TotalTargets:           run.Status.TotalTargets,
		SuccessfulJobs:         run.Status.SuccessfulJobs,
		FailedJobs:             run.Status.FailedJobs,
		RunningJobs:            run.Status.RunningJobs,
		ClusterJobs:            sanitizedClusterJobs(run.Status.ClusterJobs),
		OwnerUserID:            run.Spec.OwnerUserID,
		RegistryName:           run.Spec.Scenario.RegistryName,
		GraphRunName:           run.Labels["krkn.dev/graph-run"],
		GraphNodeID:            run.Labels["krkn.dev/graph-node"],
		CustomRunName:          run.Spec.CustomRunName,
		CreationTimestamp:      run.CreationTimestamp.Format(time.RFC3339),
		ResiliencyScoreEnabled: run.Spec.ResiliencyScoreEnabled,
		ResiliencyScore:        averageResiliencyScore(run.Status.ResiliencyScores),
		ResiliencyScores:       convertClusterResiliencyScoresWS(run.Status.ResiliencyScores),
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
			ScenarioImage:   job.ScenarioImage,
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

func averageResiliencyScore(scores []krknv1alpha1.ClusterResiliencyScore) *float64 {
	var sum float64
	var count int
	for _, s := range scores {
		if s.Status == "calculated" {
			sum += s.Score
			count++
		}
	}
	if count == 0 {
		return nil
	}
	avg := sum / float64(count)
	return &avg
}

func convertClusterResiliencyScoresWS(scores []krknv1alpha1.ClusterResiliencyScore) []ClusterResiliencyScoreResponse {
	if scores == nil {
		return nil
	}
	result := make([]ClusterResiliencyScoreResponse, len(scores))
	for i, s := range scores {
		result[i] = ClusterResiliencyScoreResponse{
			ClusterName: s.ClusterName,
			Score:       s.Score,
			Status:      s.Status,
			Message:     s.Message,
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
