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
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DeployImage = "quay.io/krkn-chaos/krkn-visualize:latest"

// DeployGrafanaImage overrides the Grafana image the upstream deploy script
// installs (its default is 10.4.0). Grafana 11+ is required for the console's
// embedded dashboards: in TV kiosk mode (kiosk=tv), Grafana 10.4 hides the
// dashboard variables row by design, while 11.x keeps both the variables and
// the time range picker visible alongside the hidden navigation chrome.
const DeployGrafanaImage = "grafana/grafana:11.6.0"

// BuildDeployImageJob runs the upstream krkn-visualize deploy image. The image
// executes the deploy script from the visualize repository and uses the
// selected target's kubeconfig mounted at its documented default path.
func BuildDeployImageJob(instanceName, operatorNamespace, kubeconfigSecret string, config DeploymentConfig) *batchv1.Job {
	backoffLimit := int32(1)
	ttlSecondsAfterFinished := int32(3600)
	mode := int32(0444)
	automountToken := false
	env := []corev1.EnvVar{
		{Name: "K8S_CMD", Value: "oc"},
		{Name: "GRAFANA_IMAGE", Value: DeployGrafanaImage},
		{Name: "GRAFANA_PASSWORD", ValueFrom: secretKeyRef(kubeconfigSecret, "grafana-password")},
		{Name: "NAMESPACE", Value: config.Namespace},
		{Name: "KUBECONFIG", Value: "/kubeconfig"},
		{Name: "PROMETHEUS_URL", Value: config.PrometheusURL},
		{Name: "PROMETHEUS_BEARER", ValueFrom: secretKeyRef(kubeconfigSecret, "prometheus-bearer")},
		{Name: "ES_URL", Value: config.ElasticsearchURL},
		{Name: "ES_USERNAME", ValueFrom: secretKeyRef(kubeconfigSecret, "es-username")},
		{Name: "ES_PASSWORD", ValueFrom: secretKeyRef(kubeconfigSecret, "es-password")},
	}
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "visualize-deploy-" + instanceName + "-",
			Namespace:    operatorNamespace,
			Labels: map[string]string{
				LabelManagedBy:               ManagedByValue,
				"app.kubernetes.io/instance": instanceName,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: &automountToken,
					// The deploy image calls the selected cluster's API. Use the
					// node resolver instead of the host cluster DNS Service, which
					// may not be able to resolve target-cluster API names.
					DNSPolicy: corev1.DNSDefault,
					Containers: []corev1.Container{{
						Name:  "visualize-deploy",
						Image: DeployImage,
						Env:   env,
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "target-kubeconfig",
							MountPath: "/kubeconfig",
							SubPath:   "config",
							ReadOnly:  true,
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "target-kubeconfig",
						VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
							SecretName:  kubeconfigSecret,
							Items:       []corev1.KeyToPath{{Key: "config", Path: "config"}},
							DefaultMode: &mode,
						}},
					}},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
}

func secretKeyRef(secretName, key string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
		Key:                  key,
	}}
}

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
	if !config.AutoDetectPrometheus && config.PrometheusURL != "" {
		krknctlArgs = append(krknctlArgs, "--prometheus-url", config.PrometheusURL)
	}
	if config.PrometheusBearerToken != "" {
		krknctlArgs = append(krknctlArgs, "--prometheus-bearer-token", config.PrometheusBearerToken)
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
` + joinArgs([]string{"krknctl", "visualize", "--delete", "--namespace", targetNamespace})

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

// joinArgs joins command arguments into a shell command string, quoting every
// argument so request-derived values cannot be interpreted by the shell.
func joinArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
	}
	return strings.Join(quoted, " ")
}
