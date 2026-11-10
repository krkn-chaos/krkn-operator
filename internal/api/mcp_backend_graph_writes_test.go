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

package api

import (
	"context"
	"encoding/json"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/mcpserver"
)

const (
	// graphTR1ID is the target request used by graph write tests.
	graphTR1ID = "tr1"
	// robotCIBotPrincipal is the principal ID of the test robot.
	robotCIBotPrincipal = "robot:ci-bot"
)

func TestMCPRunWorkflow_CreatesGraphRunOwnedByRobot(t *testing.T) {
	wf := workflowConfigMap("wf-sre", []string{"sre"}, false)
	h, c, _ := newMCPRunsTestHandler(t, targetRequestTR1(), wf)
	backend := &mcpBackend{h: h}

	out, err := backend.RunWorkflow(context.Background(), robotPrincipalFixture(), mcpserver.RunWorkflowInput{
		WorkflowID:      "wf-sre",
		TargetRequestID: graphTR1ID,
		TargetClusters:  map[string][]string{"krkn-operator": {"cluster-a"}},
	})
	if err != nil {
		t.Fatalf("RunWorkflow failed: %v", err)
	}
	if out.Kind != "graph_run" || out.Name == "" {
		t.Fatalf("out = %+v", out)
	}
	created := &krknv1alpha1.KrknGraphRun{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: out.Name, Namespace: "default"}, created); err != nil {
		t.Fatalf("graph run not created: %v", err)
	}
	if created.Spec.OwnerUserID != robotCIBotPrincipal {
		t.Errorf("OwnerUserID = %q, want robot:ci-bot", created.Spec.OwnerUserID)
	}
	if len(created.Spec.Graph) != 1 {
		t.Errorf("graph nodes = %d, want 1", len(created.Spec.Graph))
	}
}

func TestMCPRunWorkflow_InaccessibleWorkflow(t *testing.T) {
	wf := workflowConfigMap("wf-ops", []string{"ops"}, false)
	h, _, _ := newMCPRunsTestHandler(t, targetRequestTR1(), wf)
	backend := &mcpBackend{h: h}

	_, err := backend.RunWorkflow(context.Background(), robotPrincipalFixture(), mcpserver.RunWorkflowInput{
		WorkflowID:      "wf-ops",
		TargetRequestID: graphTR1ID,
		TargetClusters:  map[string][]string{"krkn-operator": {"cluster-a"}},
	})
	_ = isBackendCode(t, err, mcpserver.CodeNotFound)
}

func TestMCPRunWorkflow_DeniedWithoutRunOnAllTargets(t *testing.T) {
	wf := workflowConfigMap("wf-sre", []string{"sre"}, false)
	h, c, _ := newMCPRunsTestHandler(t, targetRequestTR1(), wf)
	backend := &mcpBackend{h: h}

	_, err := backend.RunWorkflow(context.Background(), robotPrincipalFixture(), mcpserver.RunWorkflowInput{
		WorkflowID:      "wf-sre",
		TargetRequestID: graphTR1ID,
		TargetClusters:  map[string][]string{"krkn-operator": {"cluster-a", "cluster-b"}},
	})
	_ = isBackendCode(t, err, mcpserver.CodeForbidden)

	list := &krknv1alpha1.KrknGraphRunList{}
	if err := c.List(context.Background(), list, client.InNamespace("default")); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 0 {
		t.Errorf("graph runs = %d, want 0 (no CR created)", len(list.Items))
	}
}

func TestMCPRunWorkflow_ResiliencyNeedsBaseline(t *testing.T) {
	wf := workflowConfigMap("wf-sre", []string{"sre"}, false)
	h, c, _ := newMCPRunsTestHandler(t, targetRequestTR1(), wf)
	backend := &mcpBackend{h: h}

	_, err := backend.RunWorkflow(context.Background(), robotPrincipalFixture(), mcpserver.RunWorkflowInput{
		WorkflowID:             "wf-sre",
		TargetRequestID:        graphTR1ID,
		TargetClusters:         map[string][]string{"krkn-operator": {"cluster-a"}},
		ResiliencyScoreEnabled: true,
	})
	_ = isBackendCode(t, err, mcpserver.CodeInvalidArg)

	list := &krknv1alpha1.KrknGraphRunList{}
	if err := c.List(context.Background(), list, client.InNamespace("default")); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 0 {
		t.Errorf("graph runs = %d, want 0 (no CR created)", len(list.Items))
	}
}

func TestMCPReplayGraphRun_ResubmitsSameGraph(t *testing.T) {
	g1 := graphFixture("g1", map[string][]string{"krkn-operator": {"cluster-a"}})
	g1.Spec.TargetRequestID = graphTR1ID
	node := g1.Spec.Graph["node-1"]
	node.Env = map[string]string{"PW": "s3cr3t"}
	g1.Spec.Graph["node-1"] = node
	h, c, _ := newMCPRunsTestHandler(t, g1, targetRequestTR1())
	backend := &mcpBackend{h: h}

	out, err := backend.ReplayGraphRun(context.Background(), robotPrincipalFixture(), mcpserver.ReplayGraphRunInput{GraphRunName: "g1"})
	if err != nil {
		t.Fatalf("ReplayGraphRun failed: %v", err)
	}
	if out.Kind != "graph_run" || out.Name == "" || out.Name == "g1" {
		t.Fatalf("out = %+v", out)
	}
	created := &krknv1alpha1.KrknGraphRun{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: out.Name, Namespace: "default"}, created); err != nil {
		t.Fatalf("replayed graph run not created: %v", err)
	}
	if len(created.Spec.Graph) != 1 || created.Spec.Graph["node-1"].Env["PW"] != "s3cr3t" {
		t.Errorf("replayed graph differs: %+v", created.Spec.Graph)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if containsStr(string(raw), "s3cr3t") {
		t.Errorf("replay output leaks env values: %s", raw)
	}
}

func TestMCPReplayGraphRun_StaleTargetRequest(t *testing.T) {
	g1 := graphFixture("g1", map[string][]string{"krkn-operator": {"cluster-a"}})
	h, _, _ := newMCPRunsTestHandler(t, g1) // no TR
	backend := &mcpBackend{h: h}

	_, err := backend.ReplayGraphRun(context.Background(), robotPrincipalFixture(), mcpserver.ReplayGraphRunInput{GraphRunName: "g1"})
	be := isBackendCode(t, err, mcpserver.CodeInvalidArg)
	if !containsStr(be.Message, "krkn_list_clusters") {
		t.Errorf("message %q does not point at krkn_list_clusters", be.Message)
	}
}

func TestMCPDeleteGraphRun_OwnerAndCancelAll(t *testing.T) {
	g1 := graphFixture("g1", map[string][]string{"krkn-operator": {"cluster-a"}})
	g1.Spec.TargetRequestID = graphTR1ID
	h, _, _ := newMCPRunsTestHandler(t, g1, targetRequestTR1())
	backend := &mcpBackend{h: h}

	if _, err := backend.DeleteGraphRun(context.Background(), robotPrincipalFixture(), mcpserver.GraphRunRefInput{GraphRunName: "g1"}); err != nil {
		t.Fatalf("DeleteGraphRun failed: %v", err)
	}
	if err := h.client.Get(context.Background(), client.ObjectKey{Name: "g1", Namespace: "default"}, &krknv1alpha1.KrknGraphRun{}); err == nil {
		t.Error("g1 still exists after delete")
	}
}

func TestMCPDeleteGraphRun_NotOwner(t *testing.T) {
	g1 := graphFixture("g1", map[string][]string{"krkn-operator": {"cluster-a"}})
	g1.Spec.TargetRequestID = graphTR1ID
	g1.Spec.OwnerUserID = "alice@x.io"
	h, _, _ := newMCPRunsTestHandler(t, g1, targetRequestTR1())
	backend := &mcpBackend{h: h}

	_, err := backend.DeleteGraphRun(context.Background(), robotPrincipalFixture(), mcpserver.GraphRunRefInput{GraphRunName: "g1"})
	be := isBackendCode(t, err, mcpserver.CodeForbidden)
	if !containsStr(be.Message, "robots can only delete graph runs they created") {
		t.Errorf("message = %q", be.Message)
	}
}

func TestMCPDeleteGraphRun_MissingCancelOnOneTarget(t *testing.T) {
	// Robot gains view (but not cancel) on B via a second group.
	viewers := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "viewers", Namespace: "default"},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: "viewers",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				urlB: {Actions: []string{"view"}},
			},
		},
	}
	g1 := graphFixture("g1", map[string][]string{"krkn-operator": {"cluster-a", "cluster-b"}})
	g1.Spec.TargetRequestID = graphTR1ID
	h, _, _ := newMCPRunsTestHandler(t, viewers, g1, targetRequestTR1())
	// ci-bot is in sre; also add it to viewers (view on B, no cancel).
	robot := &krknv1alpha1.KrknRobot{}
	if err := h.client.Get(context.Background(), client.ObjectKey{Name: "ci-bot", Namespace: "default"}, robot); err != nil {
		t.Fatal(err)
	}
	robot.Labels["group.krkn.krkn-chaos.dev/viewers"] = "true"
	if err := h.client.Update(context.Background(), robot); err != nil {
		t.Fatal(err)
	}
	backend := &mcpBackend{h: h}

	_, err := backend.DeleteGraphRun(context.Background(), robotPrincipalFixture(), mcpserver.GraphRunRefInput{GraphRunName: "g1"})
	be := isBackendCode(t, err, mcpserver.CodeForbidden)
	if !containsStr(be.Message, "cancel") {
		t.Errorf("message = %q, want cancel mention", be.Message)
	}
}

func TestDeleteGraphRun_UnresolvableTargetDenied(t *testing.T) {
	g1 := graphFixture("g1", map[string][]string{"krkn-operator": {"cluster-a"}})
	h, _, _ := newMCPRunsTestHandler(t, g1) // TR deleted
	backend := &mcpBackend{h: h}

	_, err := backend.DeleteGraphRun(context.Background(), robotPrincipalFixture(), mcpserver.GraphRunRefInput{GraphRunName: "g1"})
	be := isBackendCode(t, err, mcpserver.CodeForbidden)
	if !containsStr(be.Message, "cannot verify cancel permission") {
		t.Errorf("message = %q, want unresolvable-target denial", be.Message)
	}
}

func TestMCPReplayGraphRun_HiddenRunNotFound(t *testing.T) {
	// A graph run whose only target the robot cannot view is not_found even
	// though its target request exists (no existence oracle via replay).
	g2 := graphFixture("g2", map[string][]string{"krkn-operator": {"cluster-b"}})
	g2.Spec.TargetRequestID = graphTR1ID
	h, _, _ := newMCPRunsTestHandler(t, g2, targetRequestTR1())
	backend := &mcpBackend{h: h}

	_, err := backend.ReplayGraphRun(context.Background(), robotPrincipalFixture(), mcpserver.ReplayGraphRunInput{GraphRunName: "g2"})
	_ = isBackendCode(t, err, mcpserver.CodeNotFound)
}
