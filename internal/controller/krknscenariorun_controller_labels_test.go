package controller

import (
	"context"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSubmitScenarioPodCopiesAIRunLabels(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add core scheme: %v", err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add Krkn scheme: %v", err)
	}

	reconciler := &KrknScenarioRunReconciler{
		Client:    fake.NewClientBuilder().WithScheme(scheme).Build(),
		Scheme:    scheme,
		Namespace: "operator",
	}
	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scenario-run",
			Namespace: "operator",
			Labels: map[string]string{
				"krkn.dev/ai-run":           "ai-run",
				"krkn.dev/orchestrator-pod": "ai-run-a1b2c3d4",
				"krkn.dev/scenario-id":      "7",
				"krkn.dev/generation-id":    "3",
				"krkn.dev/scenario-name":    "dummy-scenario",
			},
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			Scenario: krknv1alpha1.ScenarioReference{Name: "dummy-scenario"},
		},
	}
	resources := &preparedJobResources{jobID: "job-id", clusterName: "self"}

	podName, err := reconciler.submitScenarioPod(context.Background(), scenarioRun, resources)
	if err != nil {
		t.Fatalf("submitScenarioPod failed: %v", err)
	}
	var pod corev1.Pod
	if err := reconciler.Get(context.Background(), client.ObjectKey{Name: podName, Namespace: "operator"}, &pod); err != nil {
		t.Fatalf("failed to get created scenario Pod: %v", err)
	}

	for key, want := range map[string]string{
		"krkn.dev/ai-run":           "ai-run",
		"krkn.dev/orchestrator-pod": "ai-run-a1b2c3d4",
		"krkn.dev/scenario-id":      "7",
		"krkn.dev/generation-id":    "3",
		"krkn.dev/scenario-name":    "dummy-scenario",
		"krkn.dev/component":        "scenario",
	} {
		if got := pod.Labels[key]; got != want {
			t.Errorf("label %q = %q, want %q", key, got, want)
		}
	}
}
