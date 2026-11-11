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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/mcpserver"
)

// TestMCPEndToEnd_RealClient drives the real server wiring end to end: the
// full HTTP handler stack, a real issued robot token, a real MCP client
// session, and revocation taking effect on the next call.
func TestMCPEndToEnd_RealClient(t *testing.T) {
	t.Setenv(MCPEnabledEnv, "true")

	handler, _, _ := newMCPRunsTestHandler(t, targetRequestTR1())

	issued, err := handler.robotTokens.Issue(context.Background(), "ci-bot")
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	mcpSrv := httptest.NewServer(handler.newMCPHTTPHandler())
	defer mcpSrv.Close()

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "e2e-client", Version: "v"}, nil)
	session, err := mcpClient.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:   mcpSrv.URL,
		HTTPClient: &http.Client{Transport: bearerTransport{token: issued.Token}},
	}, nil)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	// krkn_whoami
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "krkn_whoami", Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("krkn_whoami failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("krkn_whoami returned an error result")
	}
	var whoami mcpserver.WhoAmIOutput
	if err := unmarshalStructured(result, &whoami); err != nil {
		t.Fatalf("decode whoami: %v", err)
	}
	if whoami.RobotName != "ci-bot" || len(whoami.Groups) != 1 || whoami.Groups[0] != "sre" {
		t.Errorf("whoami = %+v", whoami)
	}

	// krkn_list_scenario_runs
	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "krkn_list_scenario_runs", Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("krkn_list_scenario_runs failed: %v", err)
	}
	var list mcpserver.ListScenarioRunsOutput
	if err := unmarshalStructured(result, &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if list.Total != 1 || list.Runs[0].Name != "r1" {
		t.Errorf("list = %+v, want only r1", list)
	}

	// krkn_get_scenario_run
	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "krkn_get_scenario_run", Arguments: map[string]any{"scenarioRunName": "r1"},
	})
	if err != nil {
		t.Fatalf("krkn_get_scenario_run failed: %v", err)
	}
	var run mcpserver.ScenarioRunOutput
	if err := unmarshalStructured(result, &run); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	if run.Name != "r1" || !run.PartialView || run.VisibleJobs != 1 {
		t.Errorf("run = %+v", run)
	}

	// Revoke; the very next call must fail.
	if err := handler.robotTokens.Revoke(context.Background(), "ci-bot"); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "krkn_whoami", Arguments: map[string]any{},
	})
	if err == nil {
		t.Fatal("CallTool after revocation succeeded, want failure")
	}
}

// unmarshalStructured decodes a tool result's structured content.
func unmarshalStructured(result *mcp.CallToolResult, target any) error {
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

// callTool invokes a tool through a real client session, marshalling typed
// input through JSON so the call path matches what a real MCP client sends.
func callTool[T any](t *testing.T, session *mcp.ClientSession, name string, in T) *mcp.CallToolResult {
	t.Helper()
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal %s args: %v", name, err)
	}
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		t.Fatalf("unmarshal %s args: %v", name, err)
	}
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s failed: %v", name, err)
	}
	if result.IsError {
		t.Fatalf("%s returned an error result: %s", name, result.Content[0].(interface{ GetText() string }).GetText())
	}
	return result
}

// TestMCPEndToEnd_WriteTools_RealClient drives the mutation tools through a
// real MCP client session: submission with an idempotency key (and a retry
// returning the same run), job cancellation and run deletion, all through the
// full HTTP handler stack with a real issued robot token.
func TestMCPEndToEnd_WriteTools_RealClient(t *testing.T) {
	t.Setenv(MCPEnabledEnv, "true")

	handler, c, _ := newMCPRunsTestHandler(t, targetRequestTR1())
	baseline := countScenarioRuns(t, c)

	issued, err := handler.robotTokens.Issue(context.Background(), "ci-bot")
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	mcpSrv := httptest.NewServer(handler.newMCPHTTPHandler())
	defer mcpSrv.Close()

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "e2e-write-client", Version: "v"}, nil)
	session, err := mcpClient.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:   mcpSrv.URL,
		HTTPClient: &http.Client{Transport: bearerTransport{token: issued.Token}},
	}, nil)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer func() { _ = session.Close() }()

	// krkn_run_scenario with an idempotency key, via the real session.
	in := mcpWriteInput()
	in.IdempotencyKey = "e2e-run-1"
	var started mcpserver.RunStartedOutput
	if err := unmarshalStructured(callTool(t, session, "krkn_run_scenario", in), &started); err != nil {
		t.Fatalf("decode run result: %v", err)
	}
	if started.Kind != "scenario_run" || started.Name == "" {
		t.Fatalf("started = %+v, want scenario_run with a name", started)
	}

	// Retrying the same key through the session returns the same run.
	var retried mcpserver.RunStartedOutput
	if err := unmarshalStructured(callTool(t, session, "krkn_run_scenario", in), &retried); err != nil {
		t.Fatalf("decode retry result: %v", err)
	}
	if retried.Name != started.Name {
		t.Errorf("retry returned run %q, want the existing run %q", retried.Name, started.Name)
	}
	if after := countScenarioRuns(t, c); after != baseline+1 {
		t.Errorf("scenario run count = %d, want %d (no duplicate from retry)", after, baseline+1)
	}

	// krkn_cancel_job on the seeded run r1 / job-a (cluster-a, cancel granted).
	var cancelled mcpserver.ActionOutput
	if err := unmarshalStructured(callTool(t, session, "krkn_cancel_job",
		mcpserver.CancelJobInput{ScenarioRunName: "r1", JobID: jobAFixture}), &cancelled); err != nil {
		t.Fatalf("decode cancel result: %v", err)
	}
	if cancelled.Message == "" {
		t.Error("cancel returned an empty message")
	}
	run := &krknv1alpha1.KrknScenarioRun{}
	if err := c.Get(context.Background(), krknRunKey("r1"), run); err != nil {
		t.Fatal(err)
	}
	cancelRequested := false
	for _, job := range run.Status.ClusterJobs {
		if job.JobID == jobAFixture && job.CancelRequested {
			cancelRequested = true
		}
	}
	if !cancelRequested {
		t.Error("CancelRequested not set on job-a after krkn_cancel_job")
	}

	// krkn_delete_scenario_run on a run confined to cluster-a.
	deletable := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-del", Namespace: "default"},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "tr1",
			TargetClusters:  map[string][]string{"krkn-operator": {"cluster-a"}},
			Scenario:        krknv1alpha1.ScenarioReference{Name: "scenario-x"},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ClusterName: "cluster-a", ClusterAPIURL: urlA, JobID: "job-e2e", Phase: "Succeeded"},
			},
		},
	}
	if err := c.Create(context.Background(), deletable); err != nil {
		t.Fatal(err)
	}
	var deleted mcpserver.ActionOutput
	if err := unmarshalStructured(callTool(t, session, "krkn_delete_scenario_run",
		mcpserver.ScenarioRunRefInput{ScenarioRunName: "e2e-del"}), &deleted); err != nil {
		t.Fatalf("decode delete result: %v", err)
	}
	if err := c.Get(context.Background(), krknRunKey("e2e-del"), &krknv1alpha1.KrknScenarioRun{}); err == nil {
		t.Error("e2e-del still exists after krkn_delete_scenario_run")
	}

	// A write the robot is not permitted to perform must surface as a tool
	// error result, not a transport failure: r1 spans cluster-b (no cancel).
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "krkn_delete_scenario_run",
		Arguments: map[string]any{"scenarioRunName": "r1"},
	})
	if err != nil {
		t.Fatalf("forbidden delete failed at transport level: %v", err)
	}
	if !result.IsError {
		t.Error("deleting r1 (spans a cluster without cancel) was not reported as a tool error")
	}
}

// krknRunKey builds the object key for a scenario run in the test namespace.
func krknRunKey(name string) client.ObjectKey {
	return client.ObjectKey{Name: name, Namespace: "default"}
}

// TestMCPServerVersion_NonEmpty asserts the handshake version is never empty:
// tagged builds report the module version, local builds fall back to "dev".
func TestMCPServerVersion_NonEmpty(t *testing.T) {
	h, _, _ := newMCPRunsTestHandler(t)
	if v := h.mcpServerVersion(); v == "" {
		t.Error("mcpServerVersion() returned an empty string")
	}
}
