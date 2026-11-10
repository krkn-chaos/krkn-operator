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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/mcpserver"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
)

// graphFixture builds a graph run gN targeting the given clusters.
func graphFixture(name string, clusters map[string][]string) *krknv1alpha1.KrknGraphRun {
	now := time.Now().UTC()
	return &krknv1alpha1.KrknGraphRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(now.Add(-time.Hour)),
		},
		Spec: krknv1alpha1.KrknGraphRunSpec{
			Graph: map[string]krknv1alpha1.GraphScenarioNode{
				"node-1": {
					Scenario: krknv1alpha1.ScenarioReference{Name: "s1", Private: &[]bool{false}[0]},
					Env:      map[string]string{"API_KEY": "zzz"},
				},
			},
			TargetRequestID: "tr-" + name,
			TargetClusters:  clusters,
			OwnerUserID:     "robot:ci-bot",
		},
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
			NodeStatuses: []krknv1alpha1.NodeStatus{
				{NodeID: "node-1", NodeName: "n1", Phase: "Running", Message: "cluster B exploded"},
			},
			ResiliencyScores: []krknv1alpha1.GraphClusterScore{
				{ClusterName: "cluster-a", Calculated: 90},
				{ClusterName: "cluster-b", Calculated: 10},
			},
		},
	}
}

func graphTargetRequest(name string) *krknv1alpha1.KrknTargetRequest {
	return &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec:       krknv1alpha1.KrknTargetRequestSpec{UUID: name},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status: "Completed",
			TargetData: map[string][]krknv1alpha1.ClusterTarget{
				"krkn-operator": {
					{ClusterName: "cluster-a", ClusterAPIURL: urlA},
					{ClusterName: "cluster-b", ClusterAPIURL: urlB},
				},
			},
		},
	}
}

func newMCPGraphsTestHandler(t *testing.T, objs ...runtime.Object) *Handler {
	t.Helper()
	h, _, _ := newMCPRunsTestHandler(t, objs...)
	return h
}

func TestMCPListGraphRuns_FiltersInvisible(t *testing.T) {
	g1 := graphFixture("g1", map[string][]string{"krkn-operator": {"cluster-a", "cluster-b"}})
	g2 := graphFixture("g2", map[string][]string{"krkn-operator": {"cluster-b"}})
	h := newMCPGraphsTestHandler(t, g1, g2, graphTargetRequest("tr-g1"), graphTargetRequest("tr-g2"))
	backend := &mcpBackend{h: h}

	out, err := backend.ListGraphRuns(context.Background(), robotPrincipalFixture(), mcpserver.ListGraphRunsInput{})
	if err != nil {
		t.Fatalf("ListGraphRuns failed: %v", err)
	}
	if out.Total != 1 {
		t.Fatalf("Total = %d, want 1 (g2 targets only hidden cluster B)", out.Total)
	}
	if len(out.GraphRuns) != 1 || out.GraphRuns[0].Name != "g1" {
		t.Fatalf("graph runs = %+v, want [g1]", out.GraphRuns)
	}
	if len(out.GraphRuns[0].Nodes) != 0 || out.GraphRuns[0].Definition != nil || out.GraphRuns[0].ResiliencyScores != nil {
		t.Error("list items must have nil Nodes, Definition and ResiliencyScores")
	}
}

func TestMCPGetGraphRun_PartialViewRedaction(t *testing.T) {
	g1 := graphFixture("g1", map[string][]string{"krkn-operator": {"cluster-a", "cluster-b"}})
	h := newMCPGraphsTestHandler(t, g1, graphTargetRequest("tr-g1"))
	backend := &mcpBackend{h: h}

	out, err := backend.GetGraphRun(context.Background(), robotPrincipalFixture(), mcpserver.GraphRunRefInput{GraphRunName: "g1"})
	if err != nil {
		t.Fatalf("GetGraphRun failed: %v", err)
	}
	if !out.PartialView {
		t.Error("PartialView = false, want true")
	}
	for _, node := range out.Nodes {
		if node.Message != "" {
			t.Errorf("node Message = %q, want cleared on partial view", node.Message)
		}
	}
	if clusters := out.TargetClusters["krkn-operator"]; len(clusters) != 1 || clusters[0] != "cluster-a" {
		t.Errorf("TargetClusters = %v, want only cluster-a", out.TargetClusters)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if containsStr(string(raw), "zzz") {
		t.Errorf("graph run leaks env values: %s", raw)
	}
	if containsStr(string(raw), "cluster B exploded") {
		t.Errorf("graph run leaks hidden-cluster node message: %s", raw)
	}
}

func TestGraphVisibility_TargetRequestMissing(t *testing.T) {
	g3 := graphFixture("g3", map[string][]string{"krkn-operator": {"cluster-a"}})
	// No KrknTargetRequest for g3.
	h := newMCPGraphsTestHandler(t, g3)
	backend := &mcpBackend{h: h}

	out, err := backend.ListGraphRuns(context.Background(), robotPrincipalFixture(), mcpserver.ListGraphRunsInput{})
	if err != nil {
		t.Fatalf("ListGraphRuns failed: %v", err)
	}
	if out.Total != 0 {
		t.Errorf("Total = %d, want 0 (unresolvable target means not visible)", out.Total)
	}

	_, err = backend.GetGraphRun(context.Background(), robotPrincipalFixture(), mcpserver.GraphRunRefInput{GraphRunName: "g3"})
	_ = isBackendCode(t, err, mcpserver.CodeNotFound)
}

// workflowConfigMap builds a workflow file ConfigMap.
func workflowConfigMap(fileID string, groups []string, availableToAll bool) *corev1.ConfigMap {
	labels := map[string]string{
		"app.kubernetes.io/name":                 "krkn-operator",
		"app.kubernetes.io/component":            "file",
		"files.krkn.krkn-chaos.dev/file-id":      fileID,
		"files.krkn.krkn-chaos.dev/file-purpose": "workflow-template",
	}
	if availableToAll {
		labels["files.krkn.krkn-chaos.dev/available-to-all"] = "true"
	}
	for _, g := range groups {
		labels["group.krkn.krkn-chaos.dev/"+g] = "true"
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "file-" + fileID,
			Namespace:   "default",
			Labels:      labels,
			Annotations: map[string]string{"files.krkn.krkn-chaos.dev/created-by": "admin@x.io"},
		},
		Data: map[string]string{
			"workflow.json":     `{"node-1": {"name": "s1", "image": "x:latest", "env": {"API_KEY": "zzz"}}}`,
			"studioLayout.json": `{"positions": {}}`,
		},
	}
}

func TestMCPGetWorkflow_RedactsEnv(t *testing.T) {
	wf := workflowConfigMap("wf-sre", []string{"sre"}, false)
	h := newMCPGraphsTestHandler(t, wf)
	backend := &mcpBackend{h: h}

	out, err := backend.GetWorkflow(context.Background(), robotPrincipalFixture(), mcpserver.WorkflowRefInput{WorkflowID: "wf-sre"})
	if err != nil {
		t.Fatalf("GetWorkflow failed: %v", err)
	}
	if len(out.Nodes) != 1 {
		t.Fatalf("nodes = %d, want 1", len(out.Nodes))
	}
	if len(out.Nodes[0].EnvKeys) != 1 || out.Nodes[0].EnvKeys[0] != "API_KEY" {
		t.Errorf("EnvKeys = %v, want [API_KEY]", out.Nodes[0].EnvKeys)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if containsStr(string(raw), "zzz") {
		t.Errorf("workflow output leaks env values: %s", raw)
	}
	if containsStr(string(raw), "studioLayout") || containsStr(string(raw), "positions") {
		t.Errorf("workflow output leaks StudioLayout: %s", raw)
	}
}

func TestMCPGetWorkflow_NotAccessible(t *testing.T) {
	wf := workflowConfigMap("wf-ops", []string{"ops"}, false)
	h := newMCPGraphsTestHandler(t, wf)
	backend := &mcpBackend{h: h}

	_, err := backend.GetWorkflow(context.Background(), robotPrincipalFixture(), mcpserver.WorkflowRefInput{WorkflowID: "wf-ops"})
	_ = isBackendCode(t, err, mcpserver.CodeNotFound)
}

func TestMCPListWorkflows_OnlyAccessible(t *testing.T) {
	public := workflowConfigMap("wf-public", nil, true)
	sre := workflowConfigMap("wf-sre", []string{"sre"}, false)
	ops := workflowConfigMap("wf-ops", []string{"ops"}, false)
	h := newMCPGraphsTestHandler(t, public, sre, ops)
	backend := &mcpBackend{h: h}

	out, err := backend.ListWorkflows(context.Background(), robotPrincipalFixture(), struct{}{})
	if err != nil {
		t.Fatalf("ListWorkflows failed: %v", err)
	}
	ids := make(map[string]bool, len(out.Workflows))
	for _, wf := range out.Workflows {
		ids[wf.WorkflowID] = true
	}
	if !ids["wf-public"] || !ids["wf-sre"] {
		t.Errorf("accessible workflows missing: %v", ids)
	}
	if ids["wf-ops"] {
		t.Errorf("ops-scoped workflow leaked to sre robot: %v", ids)
	}
	_ = files.FilePurposeWorkflow
}
