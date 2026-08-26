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
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildDeploymentJob creates a Kubernetes Job that deploys krkn-visualize using krknctl
func BuildDeploymentJob(instanceName, operatorNamespace string, config DeploymentConfig) *batchv1.Job {
	// Build krknctl command arguments
	krknctlArgs := []string{
		"krknctl", "visualize",
		"--namespace", config.Namespace,
		"--grafana-password", config.GrafanaPassword,
	}

	// Add Elasticsearch config if provided
	if config.ElasticsearchURL != "" {
		krknctlArgs = append(krknctlArgs,
			"--es-url", config.ElasticsearchURL,
			"--es-username", config.ElasticsearchUsername,
			"--es-password", config.ElasticsearchPassword,
		)
	}

	// Add Prometheus config
	if config.AutoDetectPrometheus {
		krknctlArgs = append(krknctlArgs, "--auto-detect-prometheus")
	} else if config.PrometheusURL != "" {
		krknctlArgs = append(krknctlArgs, "--prometheus-url", config.PrometheusURL)
		if config.PrometheusBearerToken != "" {
			krknctlArgs = append(krknctlArgs, "--prometheus-token", config.PrometheusBearerToken)
		}
	}

	// Build the full installation and execution script
	installScript := `#!/bin/bash
set -e

echo "=== Installing krknctl ==="
curl -fsSL https://raw.githubusercontent.com/krkn-chaos/krknctl/refs/heads/main/install.sh | bash

echo ""
echo "=== Running krknctl visualize ==="
export PATH="/root/.local/bin:$PATH"
` + joinArgs(krknctlArgs)

	backoffLimit := int32(2)
	ttlSecondsAfterFinished := int32(3600) // Clean up after 1 hour

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "visualize-deploy-" + instanceName,
			Namespace: operatorNamespace,
			Labels: map[string]string{
				LabelManagedBy:               ManagedByValue,
				LabelComponent:               ComponentValue,
				"app.kubernetes.io/instance": instanceName,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						LabelManagedBy: ManagedByValue,
						LabelComponent: ComponentValue,
						LabelName:      instanceName,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "default",
					Containers: []corev1.Container{
						{
							Name:    "krknctl",
							Image:   "ubuntu:22.04", // Use Ubuntu base image with curl and bash
							Command: []string{"/bin/bash", "-c"},
							Args: []string{
								installScript,
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
}

// BuildCleanupJob creates a Kubernetes Job that removes krkn-visualize using krknctl
func BuildCleanupJob(instanceName, operatorNamespace, targetNamespace string) *batchv1.Job {
	// Build the cleanup script
	cleanupScript := `#!/bin/bash
set -e

echo "=== Installing krknctl ==="
curl -fsSL https://raw.githubusercontent.com/krkn-chaos/krknctl/refs/heads/main/install.sh | bash

echo ""
echo "=== Running krknctl cleanup ==="
export PATH="/root/.local/bin:$PATH"
krknctl visualize --cleanup --namespace ` + targetNamespace

	backoffLimit := int32(1)
	ttlSecondsAfterFinished := int32(300) // Clean up after 5 minutes

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "visualize-cleanup-" + instanceName,
			Namespace: operatorNamespace,
			Labels: map[string]string{
				LabelManagedBy:               ManagedByValue,
				LabelComponent:               ComponentValue,
				"app.kubernetes.io/instance": instanceName,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						LabelManagedBy: ManagedByValue,
						LabelComponent: ComponentValue,
						LabelName:      instanceName,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "default",
					Containers: []corev1.Container{
						{
							Name:    "krknctl",
							Image:   "ubuntu:22.04", // Use Ubuntu base image with curl and bash
							Command: []string{"/bin/bash", "-c"},
							Args: []string{
								cleanupScript,
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
}

// joinArgs joins command arguments into a shell command string
func joinArgs(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		// Quote arguments that might contain spaces or special characters
		if containsSpaces(arg) {
			result += "\"" + arg + "\""
		} else {
			result += arg
		}
	}
	return result
}

// containsSpaces checks if a string contains spaces
func containsSpaces(s string) bool {
	for _, r := range s {
		if r == ' ' {
			return true
		}
	}
	return false
}
