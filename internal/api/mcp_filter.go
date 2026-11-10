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
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/mcpserver"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

// filterScenarioRun builds the robot-visible view of a scenario run. ok is
// false when no target cluster is visible. For runs that already have jobs,
// visibility is decided per job cluster; for runs with no jobs yet (for
// example immediately after creation), urlsByCluster resolves the declared
// targets through the run's target request. All output is copied; the input
// CR is never mutated.
func filterScenarioRun(run *krknv1alpha1.KrknScenarioRun, perms clusterPerms, includeJobs bool, urlsByCluster map[string]string) (mcpserver.ScenarioRunOutput, bool) {
	visibleJobs := make([]krknv1alpha1.ClusterJobStatus, 0, len(run.Status.ClusterJobs))
	for _, job := range run.Status.ClusterJobs {
		if job.ClusterAPIURL != "" && perms.can(job.ClusterAPIURL, groupauth.ActionView) {
			visibleJobs = append(visibleJobs, job)
		}
	}

	visibleTargets := visibleJobClusterNames(visibleJobs)
	if len(run.Status.ClusterJobs) == 0 {
		// No jobs yet: decide visibility from the declared targets.
		visibleTargets = viewableTargetNames(run.Spec.TargetClusters, urlsByCluster, perms)
		if len(visibleTargets) == 0 {
			return mcpserver.ScenarioRunOutput{}, false
		}
	} else if len(visibleJobs) == 0 {
		return mcpserver.ScenarioRunOutput{}, false
	}

	partial := len(visibleJobs) < len(run.Status.ClusterJobs)
	if len(run.Status.ClusterJobs) == 0 {
		partial = len(visibleTargets) < countGraphTargets(run.Spec.TargetClusters)
	}
	out := mcpserver.ScenarioRunOutput{
		Name:          run.Name,
		ScenarioName:  run.Spec.Scenario.Name,
		Phase:         run.Status.Phase,
		CustomRunName: run.Spec.CustomRunName,
		OwnerID:       run.Spec.OwnerUserID,
		GraphRunName:  run.Labels["krkn.dev/graph-run"],
		GraphNodeID:   run.Labels["krkn.dev/graph-node"],
		CreatedAt:     run.CreationTimestamp.Time,
		PartialView:   partial,
		VisibleJobs:   len(visibleJobs),
	}

	out.SucceededJobs, out.FailedJobs, out.RunningJobs = countVisibleJobs(visibleJobs)

	if includeJobs && len(visibleJobs) > 0 {
		out.Jobs = make([]mcpserver.JobOutput, 0, len(visibleJobs))
		for _, job := range visibleJobs {
			out.Jobs = append(out.Jobs, jobToOutput(job))
		}
	}

	out.ResiliencyScores = filterClusterScores(run.Status.ResiliencyScores, visibleTargets)
	if !partial {
		out.ResiliencyScore = averageScores(run.Status.ResiliencyScores)
	}
	out.TargetClusters = filterTargetClusters(run.Spec.TargetClusters, visibleTargets, partial)

	return out, true
}

// viewableTargetNames returns the declared target clusters whose resolved URL
// the robot can view.
func viewableTargetNames(targets map[string][]string, urlsByCluster map[string]string, perms clusterPerms) map[string]bool {
	visible := make(map[string]bool)
	for _, clusters := range targets {
		for _, name := range clusters {
			url, ok := urlsByCluster[name]
			if ok && url != "" && perms.can(url, groupauth.ActionView) {
				visible[name] = true
			}
		}
	}
	return visible
}

// countVisibleJobs recomputes phase counts from visible jobs only, using the
// same phase strings the controller's calculateOverallStatus uses.
func countVisibleJobs(jobs []krknv1alpha1.ClusterJobStatus) (succeeded, failed, running int) {
	for _, job := range jobs {
		switch job.Phase {
		case "Succeeded":
			succeeded++
		case "Failed", "Cancelled", "MaxRetriesExceeded":
			failed++
		case "Running", "Retrying":
			running++
		}
	}
	return succeeded, failed, running
}

// jobToOutput copies a job status into its MCP output shape.
func jobToOutput(job krknv1alpha1.ClusterJobStatus) mcpserver.JobOutput {
	return mcpserver.JobOutput{
		JobID:           job.JobID,
		ProviderName:    job.ProviderName,
		ClusterName:     job.ClusterName,
		Phase:           job.Phase,
		PodName:         job.PodName,
		StartTime:       metaTimePtr(job.StartTime),
		CompletionTime:  metaTimePtr(job.CompletionTime),
		RetryCount:      job.RetryCount,
		MaxRetries:      job.MaxRetries,
		CancelRequested: job.CancelRequested,
		FailureReason:   job.FailureReason,
		Message:         job.Message,
	}
}

// metaTimePtr converts a metav1.Time pointer to a time.Time pointer copy.
func metaTimePtr(t *metav1.Time) *time.Time {
	if t == nil {
		return nil
	}
	copied := t.Time
	return &copied
}

// visibleJobClusterNames returns the set of cluster names of visible jobs.
func visibleJobClusterNames(jobs []krknv1alpha1.ClusterJobStatus) map[string]bool {
	names := make(map[string]bool, len(jobs))
	for _, job := range jobs {
		names[job.ClusterName] = true
	}
	return names
}

// filterClusterScores keeps only scores of visible clusters, returning a copy.
func filterClusterScores(scores []krknv1alpha1.ClusterResiliencyScore, visible map[string]bool) []mcpserver.ClusterScoreOutput {
	if len(scores) == 0 {
		return nil
	}
	out := make([]mcpserver.ClusterScoreOutput, 0, len(scores))
	for _, score := range scores {
		if !visible[score.ClusterName] {
			continue
		}
		out = append(out, mcpserver.ClusterScoreOutput{
			ClusterName: score.ClusterName,
			Score:       score.Score,
			Status:      score.Status,
			Message:     score.Message,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// averageScores computes the mean of all per-cluster scores, or nil.
func averageScores(scores []krknv1alpha1.ClusterResiliencyScore) *float64 {
	if len(scores) == 0 {
		return nil
	}
	sum := 0.0
	for _, score := range scores {
		sum += score.Score
	}
	avg := sum / float64(len(scores))
	return &avg
}

// filterTargetClusters keeps only visible cluster names. On a partial view
// the visible set is explicit; on a full view all targets are kept.
func filterTargetClusters(targets map[string][]string, visible map[string]bool, partial bool) map[string][]string {
	if targets == nil {
		return nil
	}
	out := make(map[string][]string, len(targets))
	for provider, clusters := range targets {
		kept := make([]string, 0, len(clusters))
		for _, name := range clusters {
			if !partial || visible[name] {
				kept = append(kept, name)
			}
		}
		if len(kept) > 0 {
			out[provider] = kept
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// filterGraphRun builds the robot-visible view of a graph run. ok is false
// when no target cluster is visible. The input CR is never mutated.
func filterGraphRun(gr *krknv1alpha1.KrknGraphRun, urlsByCluster map[string]string, perms clusterPerms) (mcpserver.GraphRunOutput, bool) {
	visibleClusters := make(map[string]bool)
	for _, clusters := range gr.Spec.TargetClusters {
		for _, name := range clusters {
			if url, ok := urlsByCluster[name]; ok && perms.can(url, groupauth.ActionView) {
				visibleClusters[name] = true
			}
		}
	}
	if len(visibleClusters) == 0 {
		return mcpserver.GraphRunOutput{}, false
	}
	partial := len(visibleClusters) < countGraphTargets(gr.Spec.TargetClusters)

	out := mcpserver.GraphRunOutput{
		Name:            gr.Name,
		Phase:           gr.Status.Phase,
		OwnerID:         gr.Spec.OwnerUserID,
		TargetRequestID: gr.Spec.TargetRequestID,
		CreatedAt:       gr.CreationTimestamp.Time,
		StartTime:       metaTimePtr(gr.Status.StartTime),
		CompletionTime:  metaTimePtr(gr.Status.CompletionTime),
		PartialView:     partial,
		TargetClusters:  filterTargetClusters(gr.Spec.TargetClusters, visibleClusters, partial),
		Summary: mcpserver.GraphSummaryOutput{
			TotalNodes:     gr.Status.Summary.TotalNodes,
			CompletedNodes: gr.Status.Summary.CompletedNodes,
			RunningNodes:   gr.Status.Summary.RunningNodes,
			FailedNodes:    gr.Status.Summary.FailedNodes,
			PendingNodes:   gr.Status.Summary.PendingNodes,
		},
	}

	if len(gr.Status.NodeStatuses) > 0 {
		out.Nodes = make([]mcpserver.GraphNodeStatusOutput, 0, len(gr.Status.NodeStatuses))
		for _, node := range gr.Status.NodeStatuses {
			nodeOut := mcpserver.GraphNodeStatusOutput{
				NodeID:         node.NodeID,
				NodeName:       node.NodeName,
				Phase:          node.Phase,
				ScenarioRunRef: node.ScenarioRunRef,
				DependsOn:      copyStringSlice(node.DependsOn),
			}
			if !partial {
				nodeOut.Message = node.Message
			}
			out.Nodes = append(out.Nodes, nodeOut)
		}
	}

	if len(gr.Status.ResiliencyScores) > 0 {
		out.ResiliencyScores = make([]mcpserver.GraphScoreOutput, 0, len(gr.Status.ResiliencyScores))
		for _, score := range gr.Status.ResiliencyScores {
			if !visibleClusters[score.ClusterName] {
				continue
			}
			graphScore := mcpserver.GraphScoreOutput{
				ProviderName: score.ProviderName,
				ClusterName:  score.ClusterName,
				Calculated:   score.Calculated,
				Status:       score.Status,
			}
			if score.Baseline != nil {
				b := *score.Baseline
				graphScore.Baseline = &b
			}
			out.ResiliencyScores = append(out.ResiliencyScores, graphScore)
		}
	}

	return out, true
}

// countGraphTargets counts all target clusters of a graph run.
func countGraphTargets(targets map[string][]string) int {
	total := 0
	for _, clusters := range targets {
		total += len(clusters)
	}
	return total
}

// copyStringSlice returns a copy of a string slice.
func copyStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

// redactScenarioConfig builds the redacted configuration view of a run:
// environment values and file contents are never included.
func redactScenarioConfig(run *krknv1alpha1.KrknScenarioRun, perms clusterPerms, urlsByCluster map[string]string) mcpserver.ScenarioRunConfigOutput {
	out := mcpserver.ScenarioRunConfigOutput{
		Name:            run.Name,
		TargetRequestID: run.Spec.TargetRequestID,
		Scenario: mcpserver.ScenarioRefOutput{
			Name:         run.Spec.Scenario.Name,
			RegistryName: run.Spec.Scenario.RegistryName,
			Private:      boolPtrCopy(run.Spec.Scenario.Private),
		},
		CloudCredentialRef:     run.Spec.CloudCredentialRef,
		MaxRetries:             run.Spec.MaxRetries,
		RetryBackoff:           run.Spec.RetryBackoff,
		RetryDelay:             run.Spec.RetryDelay,
		KubeconfigPath:         run.Spec.KubeconfigPath,
		CustomRunName:          run.Spec.CustomRunName,
		ResiliencyScoreEnabled: run.Spec.ResiliencyScoreEnabled,
	}

	if len(run.Spec.Environment) > 0 {
		out.EnvKeys = make([]string, 0, len(run.Spec.Environment))
		for key := range run.Spec.Environment {
			out.EnvKeys = append(out.EnvKeys, key)
		}
		sort.Strings(out.EnvKeys)
	}

	if len(run.Spec.Files) > 0 {
		out.Files = make([]mcpserver.FileRefOutput, 0, len(run.Spec.Files))
		for _, file := range run.Spec.Files {
			out.Files = append(out.Files, mcpserver.FileRefOutput{
				Name:      file.Name,
				MountPath: file.MountPath,
				FileID:    file.FileID,
			})
		}
	}

	// Target-cluster filtering mirrors filterScenarioRun, based on job URLs.
	visible := make(map[string]bool)
	allVisible := true
	for _, job := range run.Status.ClusterJobs {
		if job.ClusterAPIURL != "" && perms.can(job.ClusterAPIURL, groupauth.ActionView) {
			visible[job.ClusterName] = true
		} else {
			allVisible = false
		}
	}
	if len(run.Status.ClusterJobs) == 0 {
		visibleTargets := viewableTargetNames(run.Spec.TargetClusters, urlsByCluster, perms)
		out.PartialView = len(visibleTargets) < countGraphTargets(run.Spec.TargetClusters)
		out.TargetClusters = filterTargetClusters(run.Spec.TargetClusters, visibleTargets, out.PartialView)
	} else {
		out.PartialView = !allVisible
		out.TargetClusters = filterTargetClusters(run.Spec.TargetClusters, visible, !allVisible)
	}

	return out
}

// copyStringMap deep-copies a map[string]string.
func copyStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// boolPtrCopy copies a *bool.
func boolPtrCopy(b *bool) *bool {
	if b == nil {
		return nil
	}
	copied := *b
	return &copied
}

// redactGraphNodes converts graph node definitions to their redacted output
// shape, sorted by node ID. Env values are never included.
func redactGraphNodes(graph map[string]krknv1alpha1.GraphScenarioNode) []mcpserver.GraphNodeDefinitionOutput {
	if len(graph) == 0 {
		return nil
	}
	ids := make([]string, 0, len(graph))
	for id := range graph {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	out := make([]mcpserver.GraphNodeDefinitionOutput, 0, len(ids))
	for _, id := range ids {
		node := graph[id]
		nodeOut := mcpserver.GraphNodeDefinitionOutput{
			NodeID:             id,
			Name:               node.Name,
			ScenarioName:       node.Scenario.Name,
			RegistryName:       node.Scenario.RegistryName,
			Private:            boolPtrCopy(node.Scenario.Private),
			Volumes:            copyStringMap(node.Volumes),
			DependsOn:          node.DependsOn,
			CloudCredentialRef: node.CloudCredentialRef,
			ResiliencyWeight:   node.ResiliencyWeight,
		}
		if len(node.Env) > 0 {
			nodeOut.EnvKeys = make([]string, 0, len(node.Env))
			for key := range node.Env {
				nodeOut.EnvKeys = append(nodeOut.EnvKeys, key)
			}
			sort.Strings(nodeOut.EnvKeys)
		}
		out = append(out, nodeOut)
	}
	return out
}
