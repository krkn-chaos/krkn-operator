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

package api

import (
	"net/http"
	"sort"
	"strconv"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/api/jobstats"
	kvstore "github.com/krkn-chaos/krkn-operator/pkg/configstore"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const defaultPageSize = 20

// getDefaultPageSize reads the default page size from the configstore.
func getDefaultPageSize() int {
	if val, ok := kvstore.Get().GetValue("jobs.defaultPageSize"); ok {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			return n
		}
	}
	return defaultPageSize
}

// ListJobs handles GET /api/v2/jobs
// Returns a unified, paginated list of standalone ScenarioRuns and GraphRuns sorted by creation time (newest first).
//
// @Summary List all jobs (unified view)
// @Description Returns a merged list of standalone ScenarioRuns and GraphRuns, sorted by creation time descending
// @Tags jobs
// @Produce json
// @Param page query int false "Page number (1-based). Omit for all results."
// @Param limit query int false "Items per page. Defaults to jobs.defaultPageSize from ConfigMap."
// @Success 200 {object} UnifiedJobsResponse "Paginated list of jobs"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v2/jobs [get]
func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("list-jobs")

	// List ScenarioRuns
	var scenarioRuns krknv1alpha1.KrknScenarioRunList
	if err := h.client.List(ctx, &scenarioRuns, client.InNamespace(h.namespace)); err != nil {
		logger.Error(err, "Failed to list scenario runs")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list scenario runs",
		})
		return
	}

	// List GraphRuns
	var graphRuns krknv1alpha1.KrknGraphRunList
	if err := h.client.List(ctx, &graphRuns, client.InNamespace(h.namespace)); err != nil {
		logger.Error(err, "Failed to list graph runs")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list graph runs",
		})
		return
	}

	// Apply group permission filters
	filteredScenarioRuns := h.FilterScenarioRunsByGroupPermission(scenarioRuns.Items, ctx)
	filteredGraphRuns := h.FilterGraphRunsByGroupPermission(graphRuns.Items, ctx)

	// Build unified list
	jobs := BuildUnifiedJobList(filteredScenarioRuns, filteredGraphRuns)

	// Compute aggregate stats from the full list before pagination
	stats := ComputeJobStats(jobs)

	// Parse pagination params
	page, limit := ParsePaginationParams(r, getDefaultPageSize())

	var response UnifiedJobsResponse
	if page == 0 {
		// No pagination requested — return all items
		response = UnifiedJobsResponse{
			Jobs: jobs,
			Pagination: PaginationMeta{
				Total: len(jobs),
			},
			Stats: stats,
		}
	} else {
		paginated, meta := PaginateSlice(jobs, page, limit)
		response = UnifiedJobsResponse{
			Jobs:       paginated,
			Pagination: meta,
			Stats:      stats,
		}
	}

	logger.Info("Listed jobs", "total", response.Pagination.Total, "page", response.Pagination.Page, "returned", len(response.Jobs))
	writeJSON(w, http.StatusOK, response)
}

// ComputeJobStats computes aggregate job statistics from the full unified job list.
func ComputeJobStats(jobs []UnifiedJobItem) JobStatsSummary {
	return jobstats.Compute(jobs)
}

// BuildUnifiedJobList merges standalone ScenarioRuns and GraphRuns into a single
// sorted list of UnifiedJobItems (newest first).
func BuildUnifiedJobList(scenarioRuns []krknv1alpha1.KrknScenarioRun, graphRuns []krknv1alpha1.KrknGraphRun) []UnifiedJobItem {
	jobs := make([]UnifiedJobItem, 0, len(scenarioRuns)+len(graphRuns))

	for i := range scenarioRuns {
		sr := &scenarioRuns[i]
		// Skip ScenarioRuns that are part of a GraphRun
		if sr.Labels["krkn.dev/graph-run"] != "" {
			continue
		}

		item := ScenarioRunListItem{
			ScenarioRunName:        sr.Name,
			ScenarioName:           sr.Spec.ScenarioName,
			Phase:                  sr.Status.Phase,
			TotalTargets:           sr.Status.TotalTargets,
			SuccessfulJobs:         sr.Status.SuccessfulJobs,
			FailedJobs:             sr.Status.FailedJobs,
			RunningJobs:            sr.Status.RunningJobs,
			CreatedAt:              sr.CreationTimestamp.Time,
			OwnerUserID:            sr.Spec.OwnerUserID,
			GraphRunName:           sr.Labels["krkn.dev/graph-run"],
			GraphNodeID:            sr.Labels["krkn.dev/graph-node"],
			CustomRunName:          sr.Spec.CustomRunName,
			ResiliencyScoreEnabled: sr.Spec.ResiliencyScoreEnabled,
			ResiliencyScore:        averageResiliencyScore(sr.Status.ResiliencyScores),
			ResiliencyScores:       convertClusterResiliencyScores(sr.Status.ResiliencyScores),
		}
		jobs = append(jobs, UnifiedJobItem{
			Type:        "scenarioRun",
			Name:        sr.Name,
			CreatedAt:   sr.CreationTimestamp.Time,
			ScenarioRun: &item,
		})
	}

	for i := range graphRuns {
		gr := &graphRuns[i]
		item := GraphRunListItem{
			Name:              gr.Name,
			Namespace:         gr.Namespace,
			CreationTimestamp: gr.CreationTimestamp.Time,
			Phase:             gr.Status.Phase,
			OwnerUserID:       gr.Spec.OwnerUserID,
			TargetRequestID:   gr.Spec.TargetRequestID,
			Summary: GraphRunSummaryResponse{
				TotalNodes:     gr.Status.Summary.TotalNodes,
				CompletedNodes: gr.Status.Summary.CompletedNodes,
				RunningNodes:   gr.Status.Summary.RunningNodes,
				FailedNodes:    gr.Status.Summary.FailedNodes,
				PendingNodes:   gr.Status.Summary.PendingNodes,
			},
			StartTime:               gr.Status.StartTime,
			CompletionTime:          gr.Status.CompletionTime,
			ResiliencyScoreEnabled:  gr.Spec.ResiliencyScoreEnabled,
			ResiliencyScoreBaseline: gr.Spec.ResiliencyScoreBaseline,
			ResiliencyScores:        convertGraphClusterScores(gr.Status.ResiliencyScores),
		}
		jobs = append(jobs, UnifiedJobItem{
			Type:      "graphRun",
			Name:      gr.Name,
			CreatedAt: gr.CreationTimestamp.Time,
			GraphRun:  &item,
		})
	}

	// Sort by creation time descending (newest first)
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
	})

	return jobs
}
