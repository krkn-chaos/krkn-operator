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

	client := mcp.NewClient(&mcp.Implementation{Name: "e2e-client", Version: "v"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
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
