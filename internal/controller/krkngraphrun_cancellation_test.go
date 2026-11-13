package controller

import (
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGraphRunCancelledChildDoesNotLeaveGraphRunning(t *testing.T) {
	reconciler := &KrknGraphRunReconciler{}
	graph := &krknv1alpha1.KrknGraphRun{
		Status: krknv1alpha1.KrknGraphRunStatus{
			NodeStatuses: []krknv1alpha1.NodeStatus{{NodeID: "node", Phase: "Running"}},
		},
	}
	child := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "child"},
		Status:     krknv1alpha1.KrknScenarioRunStatus{Phase: "Cancelling"},
	}
	if _, err := reconciler.updateNodeStatusFromRun(graph, "node", child); err != nil {
		t.Fatal(err)
	}
	reconciler.calculateGlobalStatus(graph)
	if graph.Status.Phase != "Running" || graph.Status.Summary.RunningNodes != 1 {
		t.Fatalf("in-progress cancellation was prematurely terminal: %+v", graph.Status)
	}
	child.Status.Phase = "Cancelled"
	if _, err := reconciler.updateNodeStatusFromRun(graph, "node", child); err != nil {
		t.Fatal(err)
	}
	reconciler.calculateGlobalStatus(graph)
	if graph.Status.Phase != "Failed" || graph.Status.Summary.FailedNodes != 1 || graph.Status.Summary.RunningNodes != 0 {
		t.Fatalf("cancelled child left graph running or successful: %+v", graph.Status)
	}
}
