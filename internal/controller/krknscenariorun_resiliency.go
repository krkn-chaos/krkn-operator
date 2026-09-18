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

Assisted-by: Claude Opus 4.6 (claude-opus-4-6@20260805)
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

// calculateResiliencyScores collects logs from all completed cluster jobs in a
// standalone ScenarioRun, parses resiliency reports, and populates
// Status.ResiliencyScores. This mirrors the logic in the GraphRun reconciler's
// calculateResiliencyScore but operates on a single ScenarioRun rather than
// aggregating across graph nodes.
func (r *KrknScenarioRunReconciler) calculateResiliencyScores(
	ctx context.Context,
	scenarioRun *krknv1alpha1.KrknScenarioRun,
) error {
	logger := log.FromContext(ctx).WithName("calculate-resiliency-scores")

	logger.Info("calculating resiliency scores for standalone scenario run",
		"scenarioRun", scenarioRun.Name,
		"totalJobs", len(scenarioRun.Status.ClusterJobs))

	var clusterScores []krknv1alpha1.ClusterResiliencyScore

	for _, jobStatus := range scenarioRun.Status.ClusterJobs {
		if jobStatus.Phase != "Succeeded" {
			logger.V(1).Info("skipping non-succeeded job",
				"clusterName", jobStatus.ClusterName,
				"phase", jobStatus.Phase)
			continue
		}

		if jobStatus.PodName == "" {
			logger.V(1).Info("pod name not set for cluster job",
				"clusterName", jobStatus.ClusterName)
			continue
		}

		podLogs, err := r.fetchScenarioRunPodLogs(ctx, scenarioRun.Namespace, jobStatus.PodName)
		if err != nil {
			logger.Error(err, "failed to fetch pod logs",
				"podName", jobStatus.PodName,
				"clusterName", jobStatus.ClusterName)
			continue
		}

		report, err := resiliency.ParseResiliencyReport(podLogs)
		if err != nil {
			logger.V(1).Info("no resiliency report found in pod logs",
				"podName", jobStatus.PodName,
				"clusterName", jobStatus.ClusterName,
				"error", err.Error())
			continue
		}

		logger.Info("parsed resiliency report from pod",
			"podName", jobStatus.PodName,
			"clusterName", jobStatus.ClusterName,
			"score", report.OverallReport.ResiliencyScore)

		clusterScores = append(clusterScores, krknv1alpha1.ClusterResiliencyScore{
			ClusterName: jobStatus.ClusterName,
			Score:       report.OverallReport.ResiliencyScore,
		})
	}

	if len(clusterScores) == 0 {
		return fmt.Errorf("no resiliency reports found in any pod logs")
	}

	scenarioRun.Status.ResiliencyScores = clusterScores

	logger.Info("resiliency scores calculated for standalone scenario run",
		"scenarioRun", scenarioRun.Name,
		"clusterCount", len(clusterScores))

	return nil
}

// fetchScenarioRunPodLogs fetches logs from a specific pod with exponential backoff retry.
func (r *KrknScenarioRunReconciler) fetchScenarioRunPodLogs(ctx context.Context, namespace, podName string) ([]byte, error) {
	logger := log.FromContext(ctx).WithName("fetch-pod-logs")

	logger.V(1).Info("fetching logs from pod",
		"pod", podName,
		"namespace", namespace)

	var logBytes []byte

	retryErr := wait.ExponentialBackoff(wait.Backoff{
		Duration: 1 * time.Second,
		Factor:   2.0,
		Steps:    3,
		Cap:      10 * time.Second,
	}, func() (bool, error) {
		req := r.Clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
			Container: "scenario",
		})

		podLogs, err := req.Stream(ctx)
		if err != nil {
			logger.V(1).Info("pod logs not yet available, will retry",
				"pod", podName,
				"error", err.Error())
			return false, nil
		}
		defer podLogs.Close()

		logBytes, err = io.ReadAll(podLogs)
		if err != nil {
			return false, fmt.Errorf("failed to read logs: %w", err)
		}

		logger.V(1).Info("successfully fetched pod logs",
			"pod", podName,
			"logSize", len(logBytes))

		return true, nil
	})

	if retryErr != nil {
		return nil, fmt.Errorf("failed to fetch logs for pod %s after retries: %w", podName, retryErr)
	}

	return logBytes, nil
}
