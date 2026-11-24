package visualize

import (
	"strings"
	"testing"
)

func TestJoinArgsShellQuotesEveryArgument(t *testing.T) {
	got := joinArgs([]string{
		"krknctl",
		"visualize",
		"--grafana-password",
		"secret; touch /tmp/pwned $(id) `id` 'quoted'",
	})

	want := "'krknctl' 'visualize' '--grafana-password' 'secret; touch /tmp/pwned $(id) `id` '\\''quoted'\\'''"
	if got != want {
		t.Fatalf("joinArgs() = %q, want %q", got, want)
	}
}

func TestBuildCleanupJobQuotesTargetNamespace(t *testing.T) {
	job := BuildCleanupJob("instance", "operator", "target; touch /tmp/pwned")
	script := job.Spec.Template.Spec.Containers[0].Args[0]

	// joinArgs quotes every argument, so the flag and the value are each
	// singly quoted.
	if want := "'target; touch /tmp/pwned'"; !strings.Contains(script, "'--namespace' "+want) {
		t.Fatalf("cleanup script does not safely quote target namespace: %s", script)
	}
}

func TestBuildDeployImageJobSetsGrafanaImage(t *testing.T) {
	config := DeploymentConfig{
		InstanceName:         "test-viz",
		TargetCluster:        "test-cluster",
		Namespace:            "krkn-visualize",
		GrafanaPassword:      "admin123",
		AutoDetectPrometheus: true,
	}
	job := BuildDeployImageJob("test-viz", "krkn-operator", "kubeconfig-secret", config)

	container := job.Spec.Template.Spec.Containers[0]
	var grafanaImage string
	for _, env := range container.Env {
		if env.Name == "GRAFANA_IMAGE" {
			grafanaImage = env.Value
		}
	}
	if grafanaImage == "" {
		t.Fatal("deploy image job does not set GRAFANA_IMAGE env var")
	}
	// The upstream deploy script defaults to Grafana 10.4.0, whose TV kiosk
	// mode hides dashboard variables. The operator must pin 11.x or newer.
	if !strings.HasPrefix(grafanaImage, "grafana/grafana:11.") {
		t.Fatalf("GRAFANA_IMAGE must be Grafana 11.x or newer, got %q", grafanaImage)
	}
}
