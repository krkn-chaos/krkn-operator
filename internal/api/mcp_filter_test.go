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
	"reflect"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

const (
	urlA = "https://api.a:6443"
	urlB = "https://api.b:6443"
	// jobAFixture is the job on cluster-a in the seeded run r1.
	jobAFixture = "job-a"
)

// permsFor builds clusterPerms for the given URL→actions.
func permsFor(perms map[string][]groupauth.Action) clusterPerms {
	cp := make(clusterPerms, len(perms))
	for u, actions := range perms {
		set := make(map[groupauth.Action]bool, len(actions))
		for _, a := range actions {
			set[a] = true
		}
		cp[u] = set
	}
	return cp
}

func filterTestRun() *krknv1alpha1.KrknScenarioRun {
	now := metav1.NewTime(time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC))
	private := true
	return &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "r1",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "tr1",
			TargetClusters:  map[string][]string{"krkn-operator": {"cluster-a", "cluster-b"}},
			Scenario:        krknv1alpha1.ScenarioReference{Name: "scenario-x", Private: &private},
			Environment:     map[string]string{"PASSWORD": "hunter2"},
			Files:           []krknv1alpha1.FileMount{{Name: "f", Content: "c2VjcmV0", MountPath: "/tmp/f", FileID: "uuid-1"}},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "PartiallyFailed",
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ProviderName: "krkn-operator", ClusterName: "cluster-a", ClusterAPIURL: urlA, JobID: "job-a", Phase: "Succeeded"},
				{ProviderName: "krkn-operator", ClusterName: "cluster-b", ClusterAPIURL: urlB, JobID: "job-b", Phase: "Failed"},
			},
			ResiliencyScores: []krknv1alpha1.ClusterResiliencyScore{
				{ClusterName: "cluster-a", Score: 80},
				{ClusterName: "cluster-b", Score: 20},
			},
		},
	}
}

func TestFilterScenarioRun_FullView(t *testing.T) {
	run := filterTestRun()
	perms := permsFor(map[string][]groupauth.Action{
		urlA: {groupauth.ActionView},
		urlB: {groupauth.ActionView},
	})
	out, ok := filterScenarioRun(run, perms, true, nil)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if out.PartialView {
		t.Error("PartialView = true, want false")
	}
	if out.ResiliencyScore == nil {
		t.Error("ResiliencyScore = nil, want preserved on full view")
	}
	if len(out.Jobs) != 2 {
		t.Errorf("jobs = %d, want 2", len(out.Jobs))
	}
}

func TestFilterScenarioRun_PartialView(t *testing.T) {
	run := filterTestRun()
	perms := permsFor(map[string][]groupauth.Action{
		urlA: {groupauth.ActionView},
	})
	out, ok := filterScenarioRun(run, perms, true, nil)
	if !ok {
		t.Fatal("ok = false, want true (one cluster visible)")
	}
	if !out.PartialView {
		t.Error("PartialView = false, want true")
	}
	if out.VisibleJobs != 1 {
		t.Errorf("VisibleJobs = %d, want 1", out.VisibleJobs)
	}
	if out.SucceededJobs != 1 {
		t.Errorf("SucceededJobs = %d, want 1 (recomputed from visible only)", out.SucceededJobs)
	}
	if out.FailedJobs != 0 {
		t.Errorf("FailedJobs = %d, want 0 (hidden job's failure must not leak)", out.FailedJobs)
	}
	if len(out.Jobs) != 1 || out.Jobs[0].ClusterName != "cluster-a" {
		t.Errorf("jobs = %+v, want only cluster-a", out.Jobs)
	}
	if len(out.ResiliencyScores) != 1 || out.ResiliencyScores[0].ClusterName != "cluster-a" {
		t.Errorf("ResiliencyScores = %+v, want only cluster-a", out.ResiliencyScores)
	}
	if out.ResiliencyScore != nil {
		t.Error("ResiliencyScore must be nil on partial view")
	}
	if _, ok := out.TargetClusters["krkn-operator"]; !ok || len(out.TargetClusters["krkn-operator"]) != 1 || out.TargetClusters["krkn-operator"][0] != "cluster-a" {
		t.Errorf("TargetClusters = %v, want only cluster-a", out.TargetClusters)
	}
}

func TestFilterScenarioRun_NoneVisible(t *testing.T) {
	run := filterTestRun()
	perms := permsFor(nil)
	if _, ok := filterScenarioRun(run, perms, true, nil); ok {
		t.Error("ok = true, want false when nothing is visible")
	}
}

func TestFilterScenarioRun_DoesNotMutateInput(t *testing.T) {
	run := filterTestRun()
	before := run.DeepCopy()
	perms := permsFor(map[string][]groupauth.Action{urlA: {groupauth.ActionView}})
	if _, ok := filterScenarioRun(run, perms, true, nil); !ok {
		t.Fatal("ok = false, want true")
	}
	if !reflect.DeepEqual(run, before) {
		t.Error("filterScenarioRun mutated the input CR")
	}
}

func graphRunFixture() *krknv1alpha1.KrknGraphRun {
	now := metav1.NewTime(time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC))
	dep := "node-1"
	return &krknv1alpha1.KrknGraphRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "g1",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: krknv1alpha1.KrknGraphRunSpec{
			Graph: map[string]krknv1alpha1.GraphScenarioNode{
				"node-1": {Scenario: krknv1alpha1.ScenarioReference{Name: "s1"}, Env: map[string]string{"TOKEN": "abc"}},
				"node-2": {Scenario: krknv1alpha1.ScenarioReference{Name: "s2"}, DependsOn: &dep},
			},
			TargetRequestID: "tr1",
			TargetClusters:  map[string][]string{"krkn-operator": {"cluster-a", "cluster-b"}},
			OwnerUserID:     "robot:ci-bot",
		},
		Status: krknv1alpha1.KrknGraphRunStatus{
			Phase: "Running",
			NodeStatuses: []krknv1alpha1.NodeStatus{
				{NodeID: "node-1", NodeName: "n1", Phase: "Completed", Message: "cluster B exploded"},
				{NodeID: "node-2", NodeName: "n2", Phase: "Pending"},
			},
			ResiliencyScores: []krknv1alpha1.GraphClusterScore{
				{ClusterName: "cluster-a", Calculated: 90},
				{ClusterName: "cluster-b", Calculated: 10},
			},
		},
	}
}

func TestFilterGraphRun_PartialView(t *testing.T) {
	gr := graphRunFixture()
	perms := permsFor(map[string][]groupauth.Action{urlA: {groupauth.ActionView}})
	urls := map[string]string{"cluster-a": urlA, "cluster-b": urlB}

	out, ok := filterGraphRun(gr, urls, perms)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if !out.PartialView {
		t.Error("PartialView = false, want true")
	}
	for _, node := range out.Nodes {
		if node.Message != "" {
			t.Errorf("node %s Message = %q, want cleared on partial view", node.NodeID, node.Message)
		}
	}
	if clusters := out.TargetClusters["krkn-operator"]; len(clusters) != 1 || clusters[0] != "cluster-a" {
		t.Errorf("TargetClusters = %v, want only cluster-a", out.TargetClusters)
	}
	if len(out.ResiliencyScores) != 1 || out.ResiliencyScores[0].ClusterName != "cluster-a" {
		t.Errorf("ResiliencyScores = %+v, want only cluster-a", out.ResiliencyScores)
	}
}

func TestRedactScenarioConfig_NoSecrets(t *testing.T) {
	run := filterTestRun()
	perms := permsFor(map[string][]groupauth.Action{urlA: {groupauth.ActionView}})

	out := redactScenarioConfig(run, perms, nil)
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(raw) == "" {
		t.Fatal("empty output")
	}
	for _, secret := range []string{"hunter2", "c2VjcmV0"} {
		if containsStr(string(raw), secret) {
			t.Errorf("redacted config leaks %q: %s", secret, raw)
		}
	}
	if len(out.EnvKeys) != 1 || out.EnvKeys[0] != "PASSWORD" {
		t.Errorf("EnvKeys = %v, want [PASSWORD]", out.EnvKeys)
	}
}

func containsStr(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}

func TestRedactGraphNodes_NoEnvValues(t *testing.T) {
	gr := graphRunFixture()
	out := redactGraphNodes(gr.Spec.Graph)
	if len(out) != 2 {
		t.Fatalf("nodes = %d, want 2", len(out))
	}
	if out[0].NodeID != "node-1" || out[1].NodeID != "node-2" {
		t.Errorf("nodes not sorted by ID: %+v", out)
	}
	if len(out[0].EnvKeys) != 1 || out[0].EnvKeys[0] != "TOKEN" {
		t.Errorf("EnvKeys = %v, want [TOKEN]", out[0].EnvKeys)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if containsStr(string(raw), "abc") {
		t.Errorf("redacted graph nodes leak env values: %s", raw)
	}
}

func TestGraphTargetURLs_TargetRequestMissing(t *testing.T) {
	h, _ := newRobotsTestHandler(t)
	gr := graphRunFixture()
	gr.Spec.TargetRequestID = "gone"

	if _, err := h.graphTargetURLs(context.Background(), gr); err == nil {
		t.Error("graphTargetURLs with missing TR returned nil error, want error")
	}
}

func TestFilterScenarioRun_NoJobsVisibleViaTargetRequest(t *testing.T) {
	now := metav1.NewTime(time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC))
	run := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "fresh", Namespace: "default", CreationTimestamp: now},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "tr1",
			TargetClusters:  map[string][]string{"krkn-operator": {"cluster-a", "cluster-b"}},
			Scenario:        krknv1alpha1.ScenarioReference{Name: "scenario-x"},
		},
	}
	perms := permsFor(map[string][]groupauth.Action{urlA: {groupauth.ActionView}})
	urls := map[string]string{"cluster-a": urlA, "cluster-b": urlB}

	out, ok := filterScenarioRun(run, perms, true, urls)
	if !ok {
		t.Fatal("ok = false, want true (view on cluster-a via target request)")
	}
	if !out.PartialView {
		t.Error("PartialView = false, want true (cluster-b hidden)")
	}
	if clusters := out.TargetClusters["krkn-operator"]; len(clusters) != 1 || clusters[0] != "cluster-a" {
		t.Errorf("TargetClusters = %v, want only cluster-a", out.TargetClusters)
	}

	// No viewable target at all -> not visible.
	noPerms := permsFor(nil)
	if _, ok := filterScenarioRun(run, noPerms, true, urls); ok {
		t.Error("ok = true, want false when no target is viewable")
	}
}
