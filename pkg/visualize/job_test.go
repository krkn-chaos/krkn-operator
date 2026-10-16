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

	want := "'krknctl' 'visualize' '--grafana-password' 'secret; touch /tmp/pwned $(id) `id` '\\''quoted'\\''"
	if got != want {
		t.Fatalf("joinArgs() = %q, want %q", got, want)
	}
}

func TestBuildCleanupJobQuotesTargetNamespace(t *testing.T) {
	job := BuildCleanupJob("instance", "operator", "target; touch /tmp/pwned")
	script := job.Spec.Template.Spec.Containers[0].Args[0]

	if want := "'target; touch /tmp/pwned'"; !strings.Contains(script, "--namespace "+want) {
		t.Fatalf("cleanup script does not safely quote target namespace: %s", script)
	}
}
