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
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krknctl/pkg/resiliency"
)

// calculateResiliencyScore collects logs from all completed scenario runs,
// parses resiliency reports, and calculates the final aggregated score.
//
// This function:
// 1. Iterates through all scenario runs in the graph
// 2. For each completed run, fetches the pod logs
// 3. Parses the KRKN_RESILIENCY_REPORT_JSON marker from logs using krknctl
// 4. Aggregates all reports into a single final score using krknctl
// 5. Compares the calculated score against the baseline
// 6. Populates GraphRun.Status.ResiliencyScore (immutable once set)
func (r *KrknGraphRunReconciler) calculateResiliencyScore(
	ctx context.Context,
	graphRun *krknv1alpha1.KrknGraphRun,
	existingRuns map[string]*krknv1alpha1.KrknScenarioRun,
) error {
	logger := log.FromContext(ctx).WithName("calculate-resiliency-score")

	logger.Info("calculating resiliency score for graph run",
		"graphRun", graphRun.Name,
		"totalNodes", len(graphRun.Status.NodeStatuses))

	// Collect resiliency reports from all completed scenario runs
	var reports []resiliency.DetailedScenarioReport

	for _, nodeStatus := range graphRun.Status.NodeStatuses {
		// Only process completed nodes
		if nodeStatus.Phase != "Completed" {
			logger.V(1).Info("skipping node - not completed",
				"nodeID", nodeStatus.NodeID,
				"phase", nodeStatus.Phase)
			continue
		}

		// Get the scenario run for this node
		sanitizedNodeID := sanitizeNodeID(nodeStatus.NodeID)
		scenarioRun, found := existingRuns[sanitizedNodeID]
		if !found {
			logger.Info("scenario run not found for node",
				"nodeID", nodeStatus.NodeID,
				"sanitizedNodeID", sanitizedNodeID)
			continue
		}

		// Process all cluster jobs for this scenario run
		for _, jobStatus := range scenarioRun.Status.ClusterJobs {
			// Skip if pod name is not set
			if jobStatus.PodName == "" {
				logger.V(1).Info("pod name not set for cluster job",
					"nodeID", nodeStatus.NodeID,
					"clusterName", jobStatus.ClusterName)
				continue
			}

			// Fetch pod logs
			podLogs, err := r.fetchPodLogs(ctx, scenarioRun.Namespace, jobStatus.PodName)
			if err != nil {
				logger.Error(err, "failed to fetch pod logs",
					"nodeID", nodeStatus.NodeID,
					"podName", jobStatus.PodName,
					"clusterName", jobStatus.ClusterName)
				continue
			}

			// Parse resiliency report from logs using krknctl
			report, err := resiliency.ParseResiliencyReport(podLogs)
			if err != nil {
				logger.V(1).Info("no resiliency report found in pod logs",
					"nodeID", nodeStatus.NodeID,
					"podName", jobStatus.PodName,
					"clusterName", jobStatus.ClusterName,
					"error", err.Error())
				continue
			}

			logger.Info("parsed resiliency report from pod",
				"nodeID", nodeStatus.NodeID,
				"podName", jobStatus.PodName,
				"clusterName", jobStatus.ClusterName,
				"scenarios", len(report.OverallReport.Scenarios),
				"score", report.OverallReport.ResiliencyScore)

			reports = append(reports, *report)
		}
	}

	// Check if we found any reports
	if len(reports) == 0 {
		return fmt.Errorf("no resiliency reports found in any scenario run logs")
	}

	// Aggregate all reports using krknctl
	finalReport := resiliency.AggregateReports(reports)

	logger.Info("aggregated resiliency score calculated",
		"graphRun", graphRun.Name,
		"totalScenarios", len(finalReport.Scenarios),
		"finalScore", finalReport.ResiliencyScore,
		"passedSlos", finalReport.PassedSlos,
		"totalSlos", finalReport.TotalSlos)

	// Determine status based on baseline comparison
	status := "no-baseline"
	var message string

	if graphRun.Spec.ResiliencyScoreBaseline != nil {
		baseline := *graphRun.Spec.ResiliencyScoreBaseline
		if finalReport.ResiliencyScore >= baseline {
			status = "pass"
			message = fmt.Sprintf("Score %.2f meets baseline %.2f", finalReport.ResiliencyScore, baseline)
		} else {
			status = "fail"
			message = fmt.Sprintf("Score %.2f is below baseline %.2f", finalReport.ResiliencyScore, baseline)
		}
	} else {
		message = fmt.Sprintf("Score %.2f calculated (no baseline specified)", finalReport.ResiliencyScore)
	}

	// Populate immutable resiliency score result
	graphRun.Status.ResiliencyScore = &krknv1alpha1.ResiliencyScoreResult{
		Calculated: finalReport.ResiliencyScore,
		Baseline:   graphRun.Spec.ResiliencyScoreBaseline,
		Status:     status,
		Message:    message,
	}

	logger.Info("resiliency score result set",
		"graphRun", graphRun.Name,
		"calculated", finalReport.ResiliencyScore,
		"baseline", graphRun.Spec.ResiliencyScoreBaseline,
		"status", status,
		"message", message)

	return nil
}

// fetchPodLogs fetches logs from a specific pod.
// Returns the full log content as bytes for parsing by krknctl.
func (r *KrknGraphRunReconciler) fetchPodLogs(ctx context.Context, namespace, podName string) ([]byte, error) {
	logger := log.FromContext(ctx).WithName("fetch-pod-logs")

	logger.V(1).Info("fetching logs from pod",
		"pod", podName,
		"namespace", namespace)

	// Get pod logs
	req := r.Clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: "scenario", // krkn scenario container name
	})

	podLogs, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to stream logs for pod %s: %w", podName, err)
	}
	defer podLogs.Close()

	// Read all logs into memory
	logBytes, err := io.ReadAll(podLogs)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs for pod %s: %w", podName, err)
	}

	logger.V(1).Info("fetched pod logs",
		"pod", podName,
		"logSize", len(logBytes))

	return logBytes, nil
}
