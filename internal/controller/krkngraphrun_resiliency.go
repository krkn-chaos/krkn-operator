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
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
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
// 6. Populates GraphRun.Status.ResiliencyScores (immutable once set)
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

		// Track cluster scores for this scenario run
		var clusterScores []krknv1alpha1.ClusterResiliencyScore

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

			logger.V(1).Info("parsed resiliency report from pod",
				"nodeID", nodeStatus.NodeID,
				"podName", jobStatus.PodName,
				"clusterName", jobStatus.ClusterName,
				"score", report.OverallReport.ResiliencyScore)

			reports = append(reports, *report)

			// Store score per cluster for this node
			clusterScore := krknv1alpha1.ClusterResiliencyScore{
				ClusterName: jobStatus.ClusterName,
				Score:       report.OverallReport.ResiliencyScore,
			}
			clusterScores = append(clusterScores, clusterScore)
		}

		// Update the scenario run with per-cluster scores
		if len(clusterScores) > 0 && len(scenarioRun.Status.ResiliencyScores) == 0 {
			scenarioRun.Status.ResiliencyScores = clusterScores
			if err := r.Status().Update(ctx, scenarioRun); err != nil {
				logger.Error(err, "failed to update scenario run with resiliency scores",
					"nodeID", nodeStatus.NodeID,
					"scenarioRun", scenarioRun.Name,
					"clusterCount", len(clusterScores))
				// Don't fail the entire calculation, just log the error
			} else {
				logger.Info("updated scenario run with per-cluster resiliency scores",
					"nodeID", nodeStatus.NodeID,
					"scenarioRun", scenarioRun.Name,
					"clusterCount", len(clusterScores))
			}
		}
	}

	// Check if we found any reports
	if len(reports) == 0 {
		return fmt.Errorf("no resiliency reports found in any scenario run logs")
	}

	// Aggregate scores PER CLUSTER
	// Build map: cluster -> list of (nodeID, score)
	clusterNodeScores := make(map[string]map[string]float64)

	for _, nodeStatus := range graphRun.Status.NodeStatuses {
		if nodeStatus.Phase != "Completed" {
			continue
		}

		sanitizedNodeID := sanitizeNodeID(nodeStatus.NodeID)
		scenarioRun, found := existingRuns[sanitizedNodeID]
		if !found {
			continue
		}

		// Add this node's scores to each cluster
		for _, clusterScore := range scenarioRun.Status.ResiliencyScores {
			if clusterNodeScores[clusterScore.ClusterName] == nil {
				clusterNodeScores[clusterScore.ClusterName] = make(map[string]float64)
			}
			clusterNodeScores[clusterScore.ClusterName][nodeStatus.NodeID] = clusterScore.Score
		}
	}

	// Calculate final GraphClusterScores
	var graphClusterScores []krknv1alpha1.GraphClusterScore

	for clusterName, nodeScores := range clusterNodeScores {
		// Calculate average score for this cluster
		var sum float64
		for _, score := range nodeScores {
			sum += score
		}
		avgScore := sum / float64(len(nodeScores))

		// Determine status based on baseline
		status := "no-baseline"
		var message string

		if graphRun.Spec.ResiliencyScoreBaseline != nil {
			baseline := *graphRun.Spec.ResiliencyScoreBaseline
			if avgScore >= baseline {
				status = "pass"
				message = fmt.Sprintf("Score %.2f meets baseline %.2f", avgScore, baseline)
			} else {
				status = "fail"
				message = fmt.Sprintf("Score %.2f is below baseline %.2f", avgScore, baseline)
			}
		} else {
			message = fmt.Sprintf("Score %.2f calculated (no baseline specified)", avgScore)
		}

		graphClusterScores = append(graphClusterScores, krknv1alpha1.GraphClusterScore{
			ClusterName:       clusterName,
			Calculated:        avgScore,
			Baseline:          graphRun.Spec.ResiliencyScoreBaseline,
			Status:            status,
			Message:           message,
			NodeContributions: nodeScores,
		})

		logger.Info("calculated cluster resiliency score",
			"graphRun", graphRun.Name,
			"clusterName", clusterName,
			"avgScore", avgScore,
			"nodeCount", len(nodeScores),
			"status", status)
	}

	// Populate immutable resiliency score results
	graphRun.Status.ResiliencyScores = graphClusterScores

	logger.Info("resiliency scores calculated for all clusters",
		"graphRun", graphRun.Name,
		"clusterCount", len(graphClusterScores))

	return nil
}

// fetchPodLogs fetches logs from a specific pod with exponential backoff retry.
// Returns the full log content as bytes for parsing by krknctl.
//
// Retry strategy:
// - 3 attempts with exponential backoff (1s, 2s, 4s)
// - Total max wait time: ~7 seconds
// - Handles transient API failures (e.g., pod logs not yet available)
func (r *KrknGraphRunReconciler) fetchPodLogs(ctx context.Context, namespace, podName string) ([]byte, error) {
	logger := log.FromContext(ctx).WithName("fetch-pod-logs")

	logger.V(1).Info("fetching logs from pod",
		"pod", podName,
		"namespace", namespace)

	var logBytes []byte

	// Retry with exponential backoff
	retryErr := wait.ExponentialBackoff(wait.Backoff{
		Duration: 1 * time.Second,
		Factor:   2.0,
		Steps:    3,
		Cap:      10 * time.Second,
	}, func() (bool, error) {
		// Get pod logs request
		req := r.Clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
			Container: "scenario", // krkn scenario container name
		})

		podLogs, err := req.Stream(ctx)
		if err != nil {
			logger.V(1).Info("pod logs not yet available, will retry",
				"pod", podName,
				"error", err.Error())
			return false, nil // Retry on stream error
		}
		defer podLogs.Close()

		// Read all logs into memory
		logBytes, err = io.ReadAll(podLogs)
		if err != nil {
			// Terminal error on read failure
			return false, fmt.Errorf("failed to read logs: %w", err)
		}

		logger.V(1).Info("successfully fetched pod logs",
			"pod", podName,
			"logSize", len(logBytes))

		return true, nil // Success
	})

	if retryErr != nil {
		return nil, fmt.Errorf("failed to fetch logs for pod %s after retries: %w", podName, retryErr)
	}

	return logBytes, nil
}
