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
	"time"

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
	// graphRunKind is the kind reported for started graph runs.
	graphRunKind = "graph_run"
	// zzzRunFixture sorts first by name but is the oldest run (created earlier).
	zzzRunFixture = "zzz-run"
	// aaaRunFixture sorts last by name but is the newer run.
	aaaRunFixture = "aaa-run"
)

// countGraphRuns counts graph runs in the test namespace.
func countGraphRuns(t *testing.T, c client.Client) int {
	t.Helper()
	list := &krknv1alpha1.KrknGraphRunList{}
	if err := c.List(context.Background(), list, client.InNamespace("default")); err != nil {
		t.Fatalf("list graph runs: %v", err)
	}
	return len(list.Items)
}

// TestMCPRunWorkflow_IdempotencyKeyDeduplicates is the graph-run variant of
// the scenario-run idempotency test: the same key returns the existing graph
// run, a different key creates a new one.
func TestMCPRunWorkflow_IdempotencyKeyDeduplicates(t *testing.T) {
	wf := workflowConfigMap("wf-sre", []string{"sre"}, false)
	h, c, _ := newMCPRunsTestHandler(t, targetRequestTR1(), wf)
	backend := &mcpBackend{h: h}
	baseline := countGraphRuns(t, c)

	in := mcpserver.RunWorkflowInput{
		WorkflowID:      "wf-sre",
		TargetRequestID: graphTR1ID,
		TargetClusters:  map[string][]string{"krkn-operator": {"cluster-a"}},
		IdempotencyKey:  "wf-deploy-1",
	}
	first, err := backend.RunWorkflow(context.Background(), robotPrincipalFixture(), in)
	if err != nil {
		t.Fatalf("first RunWorkflow failed: %v", err)
	}

	second, err := backend.RunWorkflow(context.Background(), robotPrincipalFixture(), in)
	if err != nil {
		t.Fatalf("retry RunWorkflow failed: %v", err)
	}
	if second.Name != first.Name {
		t.Errorf("retry returned graph run %q, want the existing run %q", second.Name, first.Name)
	}
	if after := countGraphRuns(t, c); after != baseline+1 {
		t.Errorf("graph run count = %d, want %d (no duplicate)", after, baseline+1)
	}

	in.IdempotencyKey = "wf-deploy-2"
	third, err := backend.RunWorkflow(context.Background(), robotPrincipalFixture(), in)
	if err != nil {
		t.Fatalf("RunWorkflow with a new key failed: %v", err)
	}
	if third.Name == first.Name {
		t.Errorf("new key returned the existing graph run %q, want a new run", third.Name)
	}
	if after := countGraphRuns(t, c); after != baseline+2 {
		t.Errorf("graph run count = %d, want %d", after, baseline+2)
	}
}

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
	if out.Kind != graphRunKind || out.Name == "" {
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
	if out.Kind != graphRunKind || out.Name == "" || out.Name == "g1" {
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
	// The run's target request is gone, so visibility cannot be verified.
	// The visibility check runs first and fails closed: not_found, not the
	// ownership/forbidden path, so the run's existence is not leaked.
	g1 := graphFixture("g1", map[string][]string{"krkn-operator": {"cluster-a"}})
	h, _, _ := newMCPRunsTestHandler(t, g1) // TR deleted
	backend := &mcpBackend{h: h}

	_, err := backend.DeleteGraphRun(context.Background(), robotPrincipalFixture(), mcpserver.GraphRunRefInput{GraphRunName: "g1"})
	_ = isBackendCode(t, err, mcpserver.CodeNotFound)
}

// TestMCPDeleteGraphRun_VisibilityBeforeOwnership proves deletion is not an
// existence oracle: a graph run the robot cannot see is not_found even though
// the ownership check (which would answer forbidden) never runs.
func TestMCPDeleteGraphRun_VisibilityBeforeOwnership(t *testing.T) {
	// g2 targets only cluster-b, which ci-bot cannot view, and was created by
	// another principal. Before the reorder this returned forbidden ("robots
	// can only delete graph runs they created"), revealing that g2 exists.
	g2 := graphFixture("g2", map[string][]string{"krkn-operator": {"cluster-b"}})
	g2.Spec.TargetRequestID = graphTR1ID
	g2.Spec.OwnerUserID = "alice@x.io"
	h, _, _ := newMCPRunsTestHandler(t, g2, targetRequestTR1())
	backend := &mcpBackend{h: h}

	_, err := backend.DeleteGraphRun(context.Background(), robotPrincipalFixture(), mcpserver.GraphRunRefInput{GraphRunName: "g2"})
	be := isBackendCode(t, err, mcpserver.CodeNotFound)
	if containsStr(be.Message, "robots can only delete") {
		t.Errorf("message = %q leaks the ownership denial for a hidden run", be.Message)
	}
}

// TestMCPReplayGraphRun_NonOwnerFreshTargetRequestNotFound closes the
// visibility bypass: a robot that does not own the run cannot replay it by
// supplying a fresh targetRequestId when the original target request is gone
// (the graph's contents, including env, must stay unreadable), while the
// owning robot can still replay with a fresh target request.
func TestMCPReplayGraphRun_NonOwnerFreshTargetRequestNotFound(t *testing.T) {
	// g1 is owned by ci-bot; its original target request does not exist, but a
	// fresh completed one (tr1) does.
	g1 := graphFixture("g1", map[string][]string{"krkn-operator": {"cluster-a"}})
	g1.Spec.TargetRequestID = "tr-gone"
	h, c, _ := newMCPRunsTestHandler(t, g1, targetRequestTR1())
	backend := &mcpBackend{h: h}

	// qa-bot has no view on any cluster and does not own g1: not_found, and
	// no graph run may be created.
	otherRobot := robotFixture("qa-bot", false)
	if err := c.Create(context.Background(), otherRobot); err != nil {
		t.Fatal(err)
	}
	before := countGraphRuns(t, c)
	_, err := backend.ReplayGraphRun(context.Background(), mcpserver.Principal{RobotName: "qa-bot"},
		mcpserver.ReplayGraphRunInput{GraphRunName: "g1", TargetRequestID: graphTR1ID})
	_ = isBackendCode(t, err, mcpserver.CodeNotFound)
	if after := countGraphRuns(t, c); after != before {
		t.Errorf("graph run count = %d, want %d (non-owner replay must not create a run)", after, before)
	}

	// The owning robot replays successfully with the fresh target request.
	out, err := backend.ReplayGraphRun(context.Background(), robotPrincipalFixture(),
		mcpserver.ReplayGraphRunInput{GraphRunName: "g1", TargetRequestID: graphTR1ID})
	if err != nil {
		t.Fatalf("owner ReplayGraphRun with fresh target request failed: %v", err)
	}
	if out.Kind != graphRunKind || out.Name == "" || out.Name == "g1" {
		t.Errorf("out = %+v, want a new graph run", out)
	}
}

// TestMCPRunWorkflow_IdempotencyKeyReturnsOldest proves duplicate graph-run
// keys resolve to the first-created run by creation timestamp, not by
// lexicographic name order.
func TestMCPRunWorkflow_IdempotencyKeyReturnsOldest(t *testing.T) {
	now := time.Now().UTC()
	// aaaRunFixture sorts first by name but was created later; zzzRunFixture is older.
	newer := graphFixture(aaaRunFixture, map[string][]string{"krkn-operator": {"cluster-a"}})
	newer.Spec.TargetRequestID = graphTR1ID
	newer.Labels = map[string]string{mcpIdempotencyKeyLabel: "dupe-key"}
	newer.CreationTimestamp = metav1.NewTime(now)
	older := graphFixture(zzzRunFixture, map[string][]string{"krkn-operator": {"cluster-a"}})
	older.Spec.TargetRequestID = graphTR1ID
	older.Labels = map[string]string{mcpIdempotencyKeyLabel: "dupe-key"}
	older.CreationTimestamp = metav1.NewTime(now.Add(-time.Hour))
	h, _, _ := newMCPRunsTestHandler(t, newer, older, targetRequestTR1())
	backend := &mcpBackend{h: h}

	out, err := backend.RunWorkflow(context.Background(), robotPrincipalFixture(), mcpserver.RunWorkflowInput{
		TargetRequestID: graphTR1ID,
		TargetClusters:  map[string][]string{"krkn-operator": {"cluster-a"}},
		IdempotencyKey:  "dupe-key",
		WorkflowID:      "wf-unset",
	})
	if err != nil {
		t.Fatalf("RunWorkflow failed: %v", err)
	}
	if out.Name != zzzRunFixture {
		t.Errorf("idempotent retry returned %q, want the oldest run %q (not name-sorted %q)", out.Name, zzzRunFixture, aaaRunFixture)
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
