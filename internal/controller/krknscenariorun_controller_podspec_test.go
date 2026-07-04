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

package controller

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func TestBuildJobLabels(t *testing.T) {
	tests := []struct {
		name        string
		jobID       string
		clusterName string
		scenarioRun *krknv1alpha1.KrknScenarioRun
		extra       map[string]string
		expected    map[string]string
	}{
		{
			name:        "base labels without owner or extra",
			jobID:       "job-123",
			clusterName: "cluster-a",
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				ObjectMeta: metav1.ObjectMeta{Name: "run-1"},
				Spec: krknv1alpha1.KrknScenarioRunSpec{
					ScenarioName:    "node-cpu-hog",
					TargetRequestID: "tr-1",
				},
			},
			extra: nil,
			expected: map[string]string{
				"krkn-job-id":         "job-123",
				"krkn-scenario-run":   "run-1",
				"krkn-scenario-name":  "node-cpu-hog",
				"krkn-cluster-name":   "cluster-a",
				"krkn-target-request": "tr-1",
			},
		},
		{
			name:        "with owner label",
			jobID:       "job-456",
			clusterName: "cluster-b",
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				ObjectMeta: metav1.ObjectMeta{Name: "run-2"},
				Spec: krknv1alpha1.KrknScenarioRunSpec{
					ScenarioName:    "pod-scenarios",
					TargetRequestID: "tr-2",
					OwnerUserID:     "User.Name@Example.COM",
				},
			},
			extra: nil,
			expected: map[string]string{
				"krkn-job-id":                    "job-456",
				"krkn-scenario-run":              "run-2",
				"krkn-scenario-name":             "pod-scenarios",
				"krkn-cluster-name":              "cluster-b",
				"krkn-target-request":            "tr-2",
				"krkn.krkn-chaos.dev/owner-user": "user-name-example-com",
			},
		},
		{
			name:        "with extra app label (pod case)",
			jobID:       "job-789",
			clusterName: "cluster-c",
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				ObjectMeta: metav1.ObjectMeta{Name: "run-3"},
				Spec: krknv1alpha1.KrknScenarioRunSpec{
					ScenarioName:    "zone-outage",
					TargetRequestID: "tr-3",
				},
			},
			extra: map[string]string{"app": "krkn-scenario"},
			expected: map[string]string{
				"app":                 "krkn-scenario",
				"krkn-job-id":         "job-789",
				"krkn-scenario-run":   "run-3",
				"krkn-scenario-name":  "zone-outage",
				"krkn-cluster-name":   "cluster-c",
				"krkn-target-request": "tr-3",
			},
		},
		{
			name:        "with owner and extra app label",
			jobID:       "job-999",
			clusterName: "cluster-d",
			scenarioRun: &krknv1alpha1.KrknScenarioRun{
				ObjectMeta: metav1.ObjectMeta{Name: "run-4"},
				Spec: krknv1alpha1.KrknScenarioRunSpec{
					ScenarioName:    "container-scenarios",
					TargetRequestID: "tr-4",
					OwnerUserID:     "owner@krkn.dev",
				},
			},
			extra: map[string]string{"app": "krkn-scenario"},
			expected: map[string]string{
				"app":                            "krkn-scenario",
				"krkn-job-id":                    "job-999",
				"krkn-scenario-run":              "run-4",
				"krkn-scenario-name":             "container-scenarios",
				"krkn-cluster-name":              "cluster-d",
				"krkn-target-request":            "tr-4",
				"krkn.krkn-chaos.dev/owner-user": "owner-krkn-dev",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildJobLabels(tc.jobID, tc.clusterName, tc.scenarioRun, tc.extra)

			if len(got) != len(tc.expected) {
				t.Fatalf("expected %d labels, got %d: %v", len(tc.expected), len(got), got)
			}
			for k, want := range tc.expected {
				if got[k] != want {
					t.Errorf("label %q: expected %q, got %q", k, want, got[k])
				}
			}
		})
	}
}

func TestBuildScenarioPod(t *testing.T) {
	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "my-run"},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			ScenarioName:    "node-cpu-hog",
			TargetRequestID: "tr-42",
			OwnerUserID:     "someone@example.com",
			Environment: map[string]string{
				"CHAOS_DURATION": "60",
			},
			Files: []krknv1alpha1.FileMount{
				{
					Name:      "config.yaml",
					Content:   "",
					MountPath: "/opt/config.yaml",
				},
			},
		},
	}

	imagePullSecrets := []corev1.LocalObjectReference{
		{Name: "krkn-job-abc-registry"},
	}

	pod := buildScenarioPod(
		"abc",
		"cluster-x",
		"quay.io/krkn-chaos/krkn-hub:node-cpu-hog",
		"krkn-job-abc-kubeconfig",
		"/home/krkn/.kube/config",
		[]string{"krkn-job-abc-file-config-yaml"},
		imagePullSecrets,
		"krkn-operator-system",
		scenarioRun,
	)

	// Pod name and namespace
	if pod.Name != "krkn-job-abc" {
		t.Errorf("expected pod name 'krkn-job-abc', got %q", pod.Name)
	}
	if pod.Namespace != "krkn-operator-system" {
		t.Errorf("expected namespace 'krkn-operator-system', got %q", pod.Namespace)
	}

	// Labels: base set + app + owner
	wantLabels := map[string]string{
		"app":                            "krkn-scenario",
		"krkn-job-id":                    "abc",
		"krkn-scenario-run":              "my-run",
		"krkn-scenario-name":             "node-cpu-hog",
		"krkn-cluster-name":              "cluster-x",
		"krkn-target-request":            "tr-42",
		"krkn.krkn-chaos.dev/owner-user": "someone-example-com",
	}
	if len(pod.Labels) != len(wantLabels) {
		t.Errorf("expected %d labels, got %d: %v", len(wantLabels), len(pod.Labels), pod.Labels)
	}
	for k, want := range wantLabels {
		if pod.Labels[k] != want {
			t.Errorf("pod label %q: expected %q, got %q", k, want, pod.Labels[k])
		}
	}

	// Container: image, name, pull policy
	if len(pod.Spec.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(pod.Spec.Containers))
	}
	container := pod.Spec.Containers[0]
	if container.Name != "scenario" {
		t.Errorf("expected container name 'scenario', got %q", container.Name)
	}
	if container.Image != "quay.io/krkn-chaos/krkn-hub:node-cpu-hog" {
		t.Errorf("unexpected container image: %q", container.Image)
	}
	if container.ImagePullPolicy != corev1.PullAlways {
		t.Errorf("expected ImagePullPolicy PullAlways, got %q", container.ImagePullPolicy)
	}

	// Env presence
	foundEnv := false
	for _, e := range container.Env {
		if e.Name == "CHAOS_DURATION" && e.Value == "60" {
			foundEnv = true
		}
	}
	if !foundEnv {
		t.Errorf("expected env CHAOS_DURATION=60 to be present, got %v", container.Env)
	}

	// Restart policy
	if pod.Spec.RestartPolicy != corev1.RestartPolicyNever {
		t.Errorf("expected RestartPolicyNever, got %q", pod.Spec.RestartPolicy)
	}

	// ServiceAccount
	if pod.Spec.ServiceAccountName != "krkn-operator-krkn-scenario-runner" {
		t.Errorf("unexpected service account: %q", pod.Spec.ServiceAccountName)
	}

	// ImagePullSecrets passed through
	if len(pod.Spec.ImagePullSecrets) != 1 || pod.Spec.ImagePullSecrets[0].Name != "krkn-job-abc-registry" {
		t.Errorf("unexpected imagePullSecrets: %v", pod.Spec.ImagePullSecrets)
	}

	// SecurityContext UID/GID/FSGroup = 1001
	sc := pod.Spec.SecurityContext
	if sc == nil {
		t.Fatal("expected pod SecurityContext to be set")
	}
	if sc.RunAsUser == nil || *sc.RunAsUser != 1001 {
		t.Errorf("expected RunAsUser 1001, got %v", sc.RunAsUser)
	}
	if sc.RunAsGroup == nil || *sc.RunAsGroup != 1001 {
		t.Errorf("expected RunAsGroup 1001, got %v", sc.RunAsGroup)
	}
	if sc.FSGroup == nil || *sc.FSGroup != 1001 {
		t.Errorf("expected FSGroup 1001, got %v", sc.FSGroup)
	}

	// Volumes: kubeconfig, file-0, tmp
	wantVolumes := map[string]bool{"kubeconfig": false, "file-0": false, "tmp": false}
	for _, v := range pod.Spec.Volumes {
		if _, ok := wantVolumes[v.Name]; ok {
			wantVolumes[v.Name] = true
		}
	}
	for name, found := range wantVolumes {
		if !found {
			t.Errorf("expected volume %q to be present, volumes: %v", name, pod.Spec.Volumes)
		}
	}
	// kubeconfig volume references the resolved ConfigMap name
	for _, v := range pod.Spec.Volumes {
		if v.Name == "kubeconfig" {
			if v.ConfigMap == nil || v.ConfigMap.Name != "krkn-job-abc-kubeconfig" {
				t.Errorf("kubeconfig volume should reference 'krkn-job-abc-kubeconfig', got %v", v.ConfigMap)
			}
		}
		if v.Name == "file-0" {
			if v.ConfigMap == nil || v.ConfigMap.Name != "krkn-job-abc-file-config-yaml" {
				t.Errorf("file-0 volume should reference 'krkn-job-abc-file-config-yaml', got %v", v.ConfigMap)
			}
		}
	}

	// VolumeMounts: kubeconfig (with mount path + subpath), file-0, tmp
	wantMounts := map[string]bool{"kubeconfig": false, "file-0": false, "tmp": false}
	for _, m := range container.VolumeMounts {
		if _, ok := wantMounts[m.Name]; ok {
			wantMounts[m.Name] = true
		}
		if m.Name == "kubeconfig" {
			if m.MountPath != "/home/krkn/.kube/config" || m.SubPath != "config" {
				t.Errorf("unexpected kubeconfig mount: %+v", m)
			}
		}
		if m.Name == "tmp" && m.MountPath != "/tmp" {
			t.Errorf("expected tmp mount path '/tmp', got %q", m.MountPath)
		}
		if m.Name == "file-0" {
			if m.MountPath != "/opt/config.yaml" || m.SubPath != "config.yaml" {
				t.Errorf("unexpected file-0 mount: %+v", m)
			}
		}
	}
	for name, found := range wantMounts {
		if !found {
			t.Errorf("expected volume mount %q to be present, mounts: %v", name, container.VolumeMounts)
		}
	}
}
