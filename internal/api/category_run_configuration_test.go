package api

import (
	"slices"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCompareCategoryRunConfigurationsScenarioRuns(t *testing.T) {
	base := scenarioRunConfigurationFixture()

	t.Run("ignores registry credentials run metadata and file IDs", func(t *testing.T) {
		other := base.DeepCopy()
		private := false
		other.Spec.Scenario.Private = &private
		other.Spec.Scenario.RegistryName = "another-registry"
		other.Spec.CloudCredentialRef = "different-cloud-credential"
		other.Spec.OwnerUserID = "another-user@example.com"
		other.Spec.CustomRunName = "a differently named execution"
		other.Spec.TargetClusters = map[string][]string{"provider-b": {"cluster-b"}}
		other.Spec.ResiliencyScoreEnabled = true
		other.Spec.Files = []krknv1alpha1.FileMount{
			{Name: "metrics.yaml", Content: "Y29udGVudA==", MountPath: "/etc/metrics.yaml", FileID: "file-2"},
			{Name: "scenario.yaml", Content: "c2NlbmFyaW8=", MountPath: "/etc/scenario.yaml", FileID: "file-1"},
		}

		assertConfigurationDifferences(t, base, other, nil)
	})

	tests := []struct {
		name   string
		mutate func(*krknv1alpha1.KrknScenarioRun)
		want   []string
	}{
		{
			name: "scenario identity",
			mutate: func(run *krknv1alpha1.KrknScenarioRun) {
				run.Spec.Scenario.Name = "pod-disruption-v2"
			},
			want: []string{"scenario.name"},
		},
		{
			name: "kubeconfig mount path",
			mutate: func(run *krknv1alpha1.KrknScenarioRun) {
				run.Spec.KubeconfigPath = "/var/run/other-kubeconfig"
			},
			want: []string{"kubeconfigPath"},
		},
		{
			name: "retry count",
			mutate: func(run *krknv1alpha1.KrknScenarioRun) {
				run.Spec.MaxRetries++
			},
			want: []string{"maxRetries"},
		},
		{
			name: "retry policy",
			mutate: func(run *krknv1alpha1.KrknScenarioRun) {
				run.Spec.RetryBackoff = "fixed"
			},
			want: []string{"retryBackoff"},
		},
		{
			name: "retry delay",
			mutate: func(run *krknv1alpha1.KrknScenarioRun) {
				run.Spec.RetryDelay = "30s"
			},
			want: []string{"retryDelay"},
		},
		{
			name: "environment value",
			mutate: func(run *krknv1alpha1.KrknScenarioRun) {
				run.Spec.Environment["POD_COUNT"] = "3"
			},
			want: []string{"environment.POD_COUNT"},
		},
		{
			name: "environment key removed",
			mutate: func(run *krknv1alpha1.KrknScenarioRun) {
				delete(run.Spec.Environment, "POD_COUNT")
			},
			want: []string{"environment.POD_COUNT"},
		},
		{
			name: "file content",
			mutate: func(run *krknv1alpha1.KrknScenarioRun) {
				run.Spec.Files[0].Content = "bmV3IGNvbnRlbnQ="
			},
			want: []string{"files"},
		},
		{
			name: "file mount path",
			mutate: func(run *krknv1alpha1.KrknScenarioRun) {
				run.Spec.Files[0].MountPath = "/different/path"
			},
			want: []string{"files"},
		},
		{
			name: "file name",
			mutate: func(run *krknv1alpha1.KrknScenarioRun) {
				run.Spec.Files[0].Name = "different.yaml"
			},
			want: []string{"files"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			other := base.DeepCopy()
			test.mutate(other)
			assertConfigurationDifferences(t, base, other, test.want)
		})
	}
}

func TestCompareCategoryRunConfigurationsGraphRuns(t *testing.T) {
	base := graphRunConfigurationFixture()

	t.Run("ignores registry credentials comments images file IDs baseline and map order", func(t *testing.T) {
		other := base.DeepCopy()
		other.Spec.Graph = map[string]krknv1alpha1.GraphScenarioNode{
			"node-b": {
				Comment:            "changed comment",
				Scenario:           scenarioReference("network-latency", false, "other-registry"),
				Name:               "changed legacy display name",
				Image:              "ignored-image",
				RegistryName:       "another-registry",
				Env:                map[string]string{"LATENCY_MS": "100", "DIRECTION": "egress"},
				Volumes:            map[string]string{"different-metrics-file-id": "/etc/metrics.yaml"},
				DependsOn:          stringPointer("node-a"),
				CloudCredentialRef: "another-node-credential",
			},
			"node-a": {
				Scenario:           scenarioReference("pod-disruption", false, "another-registry"),
				Name:               "changed display name",
				Image:              "another-ignored-image",
				Env:                map[string]string{"POD_COUNT": "2"},
				Volumes:            map[string]string{"different-scenario-file-id": "/etc/scenario.yaml"},
				CloudCredentialRef: "another-node-credential",
			},
		}
		other.Spec.CloudCredentialRef = "another-graph-credential"
		other.Spec.ResiliencyScoreBaseline = floatPointer(99)
		other.Spec.ResiliencyScoreEnabled = !base.Spec.ResiliencyScoreEnabled

		assertConfigurationDifferences(t, base, other, nil)
	})

	tests := []struct {
		name   string
		mutate func(*krknv1alpha1.KrknGraphRun)
		want   []string
	}{
		{
			name: "node scenario identity",
			mutate: func(run *krknv1alpha1.KrknGraphRun) {
				node := run.Spec.Graph["node-a"]
				node.Scenario.Name = "pod-disruption-v2"
				run.Spec.Graph["node-a"] = node
			},
			want: []string{"graph.node-a.scenario.name"},
		},
		{
			name: "node environment parameter",
			mutate: func(run *krknv1alpha1.KrknGraphRun) {
				node := run.Spec.Graph["node-a"]
				node.Env["POD_COUNT"] = "3"
				run.Spec.Graph["node-a"] = node
			},
			want: []string{"graph.node-a.env.POD_COUNT"},
		},
		{
			name: "node volume reference",
			mutate: func(run *krknv1alpha1.KrknGraphRun) {
				node := run.Spec.Graph["node-a"]
				node.Volumes["scenario-file"] = "/etc/other.yaml"
				run.Spec.Graph["node-a"] = node
			},
			want: []string{"graph.node-a.volumes"},
		},
		{
			name: "resiliency score mount presence and path are ignored",
			mutate: func(run *krknv1alpha1.KrknGraphRun) {
				run.Spec.ResiliencyMountPath = "/var/run/other-metrics.yaml"
				node := run.Spec.Graph["node-b"]
				node.Volumes["metrics-file"] = run.Spec.ResiliencyMountPath
				run.Spec.Graph["node-b"] = node
			},
			want: nil,
		},
		{
			name: "resiliency score mount can be absent without splitting",
			mutate: func(run *krknv1alpha1.KrknGraphRun) {
				node := run.Spec.Graph["node-b"]
				node.Volumes = nil
				run.Spec.Graph["node-b"] = node
			},
			want: nil,
		},
		{
			name: "dependency relation",
			mutate: func(run *krknv1alpha1.KrknGraphRun) {
				node := run.Spec.Graph["node-b"]
				node.DependsOn = nil
				run.Spec.Graph["node-b"] = node
			},
			want: []string{"graph.node-b.depends_on"},
		},
		{
			name: "regenerated node IDs with remapped dependencies",
			mutate: func(run *krknv1alpha1.KrknGraphRun) {
				first := run.Spec.Graph["node-a"]
				second := run.Spec.Graph["node-b"]
				delete(run.Spec.Graph, "node-a")
				delete(run.Spec.Graph, "node-b")
				second.DependsOn = stringPointer("replayed-node-a")
				run.Spec.Graph["replayed-node-a"] = first
				run.Spec.Graph["replayed-node-b"] = second
			},
			want: nil,
		},
		{
			name: "regenerated node IDs with missing resiliency metrics mounts",
			mutate: func(run *krknv1alpha1.KrknGraphRun) {
				first := run.Spec.Graph["node-a"]
				second := run.Spec.Graph["node-b"]
				delete(run.Spec.Graph, "node-a")
				delete(run.Spec.Graph, "node-b")
				second.DependsOn = stringPointer("replayed-node-a")
				second.Volumes = nil
				run.Spec.Graph["replayed-node-a"] = first
				run.Spec.Graph["replayed-node-b"] = second
			},
			want: nil,
		},
		{
			name: "regenerated node IDs with changed dependency topology",
			mutate: func(run *krknv1alpha1.KrknGraphRun) {
				first := run.Spec.Graph["node-a"]
				second := run.Spec.Graph["node-b"]
				delete(run.Spec.Graph, "node-a")
				delete(run.Spec.Graph, "node-b")
				second.DependsOn = nil
				run.Spec.Graph["replayed-node-a"] = first
				run.Spec.Graph["replayed-node-b"] = second
			},
			want: []string{"graph"},
		},
		{
			name: "graph retry count",
			mutate: func(run *krknv1alpha1.KrknGraphRun) {
				run.Spec.MaxRetries++
			},
			want: []string{"maxRetries"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			other := base.DeepCopy()
			test.mutate(other)
			assertConfigurationDifferences(t, base, other, test.want)
		})
	}
}

func TestCompareCategoryRunConfigurationsRejectsDifferentKinds(t *testing.T) {
	assertConfigurationDifferences(t, scenarioRunConfigurationFixture(), graphRunConfigurationFixture(), []string{"type"})
	if got := compareCategoryRunConfigurations(nil, scenarioRunConfigurationFixture()); !slices.Equal(got, []string{"type"}) {
		t.Fatalf("configuration differences for nil input = %v, want [type]", got)
	}
}

func TestCompareCategoryRunConfigurationsUsesLegacyScenarioIdentity(t *testing.T) {
	left := scenarioRunConfigurationFixture()
	right := scenarioRunConfigurationFixture()
	left.Spec.Scenario.Name = ""
	left.Spec.ScenarioName = "legacy-scenario"
	right.Spec.Scenario.Name = "legacy-scenario"
	right.Spec.ScenarioName = "different-unused-legacy-name"
	assertConfigurationDifferences(t, left, right, nil)

	leftGraph := graphRunConfigurationFixture()
	rightGraph := graphRunConfigurationFixture()
	leftNode := leftGraph.Spec.Graph["node-a"]
	leftNode.Scenario.Name = ""
	leftNode.Name = "legacy-node-scenario"
	leftGraph.Spec.Graph["node-a"] = leftNode
	rightNode := rightGraph.Spec.Graph["node-a"]
	rightNode.Scenario.Name = "legacy-node-scenario"
	rightNode.Name = "different-unused-legacy-name"
	rightGraph.Spec.Graph["node-a"] = rightNode
	assertConfigurationDifferences(t, leftGraph, rightGraph, nil)
}

func scenarioRunConfigurationFixture() *krknv1alpha1.KrknScenarioRun {
	return &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "scenario-run"},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			Scenario:           scenarioReference("pod-disruption", true, "registry-a"),
			Environment:        map[string]string{"POD_COUNT": "2", "DURATION": "60"},
			Files:              []krknv1alpha1.FileMount{{Name: "scenario.yaml", Content: "c2NlbmFyaW8=", MountPath: "/etc/scenario.yaml", FileID: "file-1"}, {Name: "metrics.yaml", Content: "Y29udGVudA==", MountPath: "/etc/metrics.yaml", FileID: "file-3"}},
			CloudCredentialRef: "credential-a",
			OwnerUserID:        "first-user@example.com",
			CustomRunName:      "first execution",
			TargetClusters:     map[string][]string{"provider-a": {"cluster-a"}},
			MaxRetries:         2,
		},
	}
}

func graphRunConfigurationFixture() *krknv1alpha1.KrknGraphRun {
	return &krknv1alpha1.KrknGraphRun{
		ObjectMeta: metav1.ObjectMeta{Name: "graph-run"},
		Spec: krknv1alpha1.KrknGraphRunSpec{
			Graph: map[string]krknv1alpha1.GraphScenarioNode{
				"node-a": {
					Scenario: scenarioReference("pod-disruption", true, "registry-a"),
					Name:     "pod-disruption",
					Image:    "ignored-image",
					Env:      map[string]string{"POD_COUNT": "2"},
					Volumes:  map[string]string{"scenario-file": "/etc/scenario.yaml"},
				},
				"node-b": {
					Scenario:  scenarioReference("network-latency", true, "registry-a"),
					Env:       map[string]string{"LATENCY_MS": "100", "DIRECTION": "egress"},
					Volumes:   map[string]string{"metrics-file": "/etc/metrics.yaml"},
					DependsOn: stringPointer("node-a"),
				},
			},
			CloudCredentialRef:      "credential-a",
			ResiliencyMountPath:     "/etc/metrics.yaml",
			ResiliencyScoreBaseline: floatPointer(80),
			ResiliencyScoreEnabled:  true,
			MaxRetries:              3,
		},
	}
}

func scenarioReference(name string, private bool, registry string) krknv1alpha1.ScenarioReference {
	return krknv1alpha1.ScenarioReference{
		Name:         name,
		Private:      &private,
		RegistryName: registry,
	}
}

func stringPointer(value string) *string { return &value }

func floatPointer(value float64) *float64 { return &value }

func assertConfigurationDifferences(t *testing.T, left, right any, want []string) {
	t.Helper()
	leftObject, leftOK := left.(client.Object)
	rightObject, rightOK := right.(client.Object)
	if !leftOK || !rightOK {
		t.Fatalf("test fixtures must implement client.Object, got %T and %T", left, right)
	}
	got := compareCategoryRunConfigurations(leftObject, rightObject)
	if !slices.Equal(got, want) {
		t.Fatalf("configuration differences = %v, want %v", got, want)
	}
}
