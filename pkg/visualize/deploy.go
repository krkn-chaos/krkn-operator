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

package visualize

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DeploymentConfig contains all configuration needed to deploy krkn-visualize
type DeploymentConfig struct {
	InstanceName          string
	TargetCluster         string
	Namespace             string
	GrafanaPassword       string
	ElasticsearchURL      string
	ElasticsearchUsername string
	ElasticsearchPassword string
	PrometheusURL         string
	PrometheusBearerToken string
	AutoDetectPrometheus  bool
	KubeCommand           string // "kubectl" or "oc"
}

// DeployKrknVisualize executes the krkn-visualize deployment using krknctl
// This function should be called asynchronously (e.g., in a goroutine or Job)
func DeployKrknVisualize(ctx context.Context, config DeploymentConfig) (string, error) {
	logger := log.FromContext(ctx).WithName("deploy-krkn-visualize").WithValues("instance", config.InstanceName)

	logger.Info("Starting krkn-visualize deployment using krknctl",
		"targetCluster", config.TargetCluster,
		"namespace", config.Namespace,
	)

	// Build krknctl visualize command
	args := []string{"visualize"}

	// Required flags
	args = append(args,
		"--namespace", config.Namespace,
		"--grafana-password", config.GrafanaPassword,
	)

	// Add Elasticsearch config if provided
	if config.ElasticsearchURL != "" {
		args = append(args,
			"--es-url", config.ElasticsearchURL,
			"--es-username", config.ElasticsearchUsername,
			"--es-password", config.ElasticsearchPassword,
		)
	}

	// Add Prometheus config
	if config.AutoDetectPrometheus {
		args = append(args, "--auto-detect-prometheus")
	} else if config.PrometheusURL != "" {
		args = append(args, "--prometheus-url", config.PrometheusURL)
		if config.PrometheusBearerToken != "" {
			args = append(args, "--prometheus-token", config.PrometheusBearerToken)
		}
	}

	// Execute krknctl command
	cmd := exec.CommandContext(ctx, "krknctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(err, "krknctl visualize command failed", "output", string(output))
		return "", fmt.Errorf("krknctl visualize failed: %w - output: %s", err, string(output))
	}

	logger.Info("krknctl visualize completed successfully", "output", string(output))

	// Detect Grafana URL after deployment
	grafanaURL, err := detectGrafanaURL(ctx, config.Namespace, config.KubeCommand)
	if err != nil {
		logger.Error(err, "Failed to detect Grafana URL")
		return "", fmt.Errorf("failed to detect Grafana URL: %w", err)
	}

	logger.Info("krkn-visualize deployment completed", "grafanaURL", grafanaURL)
	return grafanaURL, nil
}

// isOpenShift checks if the target cluster is OpenShift
func isOpenShift(ctx context.Context) bool {
	// Check if OpenShift-specific resources exist
	cmd := exec.CommandContext(ctx, "kubectl", "api-resources", "--api-group=route.openshift.io")
	err := cmd.Run()
	return err == nil
}

// detectGrafanaURL detects the Grafana dashboard URL after deployment
func detectGrafanaURL(ctx context.Context, namespace, kubeCmd string) (string, error) {
	logger := log.FromContext(ctx).WithName("detect-grafana-url")

	// For OpenShift, get the Route
	if kubeCmd == "oc" {
		cmd := exec.CommandContext(ctx, "oc", "get", "route", "grafana", "-n", namespace,
			"-o", "jsonpath={.spec.host}")
		output, err := cmd.Output()
		if err != nil {
			logger.Error(err, "Failed to get OpenShift route")
			return "", err
		}
		host := strings.TrimSpace(string(output))
		if host != "" {
			return fmt.Sprintf("https://%s", host), nil
		}
	}

	// For Kubernetes, try to get LoadBalancer or NodePort service
	cmd := exec.CommandContext(ctx, "kubectl", "get", "svc", "grafana", "-n", namespace,
		"-o", "jsonpath={.status.loadBalancer.ingress[0].hostname}")
	output, err := cmd.Output()
	if err != nil {
		logger.Error(err, "Failed to get service")
		return "", err
	}

	host := strings.TrimSpace(string(output))
	if host != "" {
		return fmt.Sprintf("http://%s:3000", host), nil
	}

	// Fallback: return placeholder
	return fmt.Sprintf("http://grafana.%s.svc.cluster.local:3000", namespace), nil
}

// CleanupKrknVisualize removes a krkn-visualize deployment using krknctl
func CleanupKrknVisualize(ctx context.Context, namespace string) error {
	logger := log.FromContext(ctx).WithName("cleanup-krkn-visualize").WithValues("namespace", namespace)

	logger.Info("Starting krkn-visualize cleanup")

	// Use krknctl to cleanup
	cmd := exec.CommandContext(ctx, "krknctl", "visualize", "--cleanup", "--namespace", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(err, "krknctl cleanup failed", "output", string(output))
		return fmt.Errorf("krknctl cleanup failed: %w - output: %s", err, string(output))
	}

	logger.Info("krkn-visualize cleanup completed", "output", string(output))
	return nil
}
