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

package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func validIdentity() RobotIdentity {
	return RobotIdentity{Name: "ci-bot", ExpiresAt: time.Now().Add(time.Hour)}
}

func TestMCPHTTP_NoTokenUnauthorized(t *testing.T) {
	srv := httptest.NewServer(NewHandler(&stubBackend{}, staticVerifier{identity: validIdentity()}.verify, "test"))
	defer srv.Close()

	resp, err := http.Post(srv.URL, "application/json", bytes.NewReader([]byte(`{"jsonrpc":"2.0","method":"initialize","id":1}`)))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestMCPHTTP_InvalidTokenUnauthorized(t *testing.T) {
	srv := httptest.NewServer(NewHandler(&stubBackend{}, staticVerifier{err: ErrInvalidCredentials}.verify, "test"))
	defer srv.Close()

	client := &http.Client{Transport: bearerRoundTripper{token: "krkn_robot_ci-bot.bogus"}}
	resp, err := client.Post(srv.URL, "application/json", bytes.NewReader([]byte(`{"jsonrpc":"2.0","method":"initialize","id":1}`)))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
	var body bytes.Buffer
	_, _ = body.ReadFrom(resp.Body)
	if strings.Contains(body.String(), "backend") {
		t.Errorf("401 body leaks backend detail: %s", body.String())
	}
}

func TestMCPHTTP_BackendErrorIsGeneric(t *testing.T) {
	srv := httptest.NewServer(NewHandler(&stubBackend{}, staticVerifier{err: errors.New("etcd 10.0.0.5 timeout")}.verify, "test"))
	defer srv.Close()

	client := &http.Client{Transport: bearerRoundTripper{token: "krkn_robot_ci-bot.x"}}
	resp, err := client.Post(srv.URL, "application/json", bytes.NewReader([]byte(`{"jsonrpc":"2.0","method":"initialize","id":1}`)))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
	var body bytes.Buffer
	_, _ = body.ReadFrom(resp.Body)
	if strings.Contains(body.String(), "10.0.0.5") {
		t.Errorf("500 body leaks infrastructure detail: %s", body.String())
	}
}

func TestMCPHTTP_ListToolsAndWhoAmI(t *testing.T) {
	backend := &stubBackend{whoami: WhoAmIOutput{
		RobotName: "ci-bot",
		Groups:    []string{"sre"},
	}}
	session := newTestSession(t, backend, staticVerifier{identity: validIdentity()}.verify)

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	found := false
	for _, tool := range tools.Tools {
		if tool.Name == "krkn_whoami" {
			found = true
		}
	}
	if !found {
		t.Fatalf("krkn_whoami not listed: %+v", tools.Tools)
	}

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "krkn_whoami",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("krkn_whoami returned error: %+v", result)
	}
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var out WhoAmIOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}
	if out.RobotName != "ci-bot" {
		t.Errorf("RobotName = %q, want ci-bot", out.RobotName)
	}
}

func TestMCPHTTP_GETNotAllowed(t *testing.T) {
	srv := httptest.NewServer(NewHandler(&stubBackend{}, staticVerifier{identity: validIdentity()}.verify, "test"))
	defer srv.Close()

	client := &http.Client{Transport: bearerRoundTripper{token: "krkn_robot_ci-bot.x"}}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405 (stateless mode)", resp.StatusCode)
	}
}

func TestMCPHTTP_RevokedTokenRejected(t *testing.T) {
	flip := &flipVerifier{identity: validIdentity()}
	backend := &stubBackend{whoami: WhoAmIOutput{RobotName: "ci-bot"}}
	session := newTestSession(t, backend, flip.verify)

	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "krkn_whoami", Arguments: map[string]any{}}); err != nil {
		t.Fatalf("first CallTool failed: %v", err)
	}

	flip.invalid = true
	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "krkn_whoami", Arguments: map[string]any{}})
	if err == nil {
		t.Fatal("CallTool after revocation succeeded, want transport error")
	}
}

// flipVerifier flips from valid to invalid.
type flipVerifier struct {
	identity RobotIdentity
	invalid  bool
}

func (f *flipVerifier) verify(ctx context.Context, token string) (RobotIdentity, error) {
	if f.invalid {
		return RobotIdentity{}, ErrInvalidCredentials
	}
	return f.identity, nil
}

func TestToolError_MapsCodes(t *testing.T) {
	result := toolError(&BackendError{Code: "not_found", Message: "x"})
	if !result.IsError {
		t.Fatal("IsError = false, want true")
	}
	if len(result.Content) != 1 {
		t.Fatalf("content entries = %d, want 1", len(result.Content))
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content[0] type = %T, want TextContent", result.Content[0])
	}
	if text.Text != "not_found: x" {
		t.Errorf("text = %q, want not_found: x", text.Text)
	}

	result = toolError(errors.New("boom"))
	if !result.IsError {
		t.Fatal("IsError = false, want true")
	}
	text, ok = result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content[0] type = %T, want TextContent", result.Content[0])
	}
	if text.Text != "internal: internal error" {
		t.Errorf("text = %q, want internal: internal error", text.Text)
	}
}

func TestPrincipalFromRequest_MissingFailsClosed(t *testing.T) {
	req := &mcp.CallToolRequest{}
	if _, err := principalFromRequest(req); err == nil {
		t.Fatal("principalFromRequest with no token info returned nil error")
	}
}

func TestMCPHTTP_ReadToolsListed(t *testing.T) {
	session := newTestSession(t, &stubBackend{}, staticVerifier{identity: validIdentity()}.verify)

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	wantReadOnly := []string{
		"krkn_whoami",
		"krkn_list_scenario_runs",
		"krkn_get_scenario_run",
		"krkn_get_job_logs",
		"krkn_get_scenario_run_config",
		"krkn_get_report_status",
		"krkn_list_scenarios",
		"krkn_get_scenario_detail",
		"krkn_list_available_resources",
	}
	byName := make(map[string]*mcp.Tool, len(tools.Tools))
	for i := range tools.Tools {
		byName[tools.Tools[i].Name] = tools.Tools[i]
	}
	for _, name := range wantReadOnly {
		tool, ok := byName[name]
		if !ok {
			t.Errorf("tool %q not listed", name)
			continue
		}
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("tool %q must have ReadOnlyHint true", name)
		}
	}
	// krkn_list_clusters may create a discovery CR.
	tool, ok := byName["krkn_list_clusters"]
	if !ok {
		t.Fatal("krkn_list_clusters not listed")
	}
	if tool.Annotations == nil || tool.Annotations.ReadOnlyHint {
		t.Error("krkn_list_clusters must have ReadOnlyHint false")
	}
	if tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint {
		t.Error("krkn_list_clusters must have DestructiveHint false")
	}
	if !tool.Annotations.IdempotentHint {
		t.Error("krkn_list_clusters must have IdempotentHint true")
	}
}

func TestMCPRunScenario_NoInlineFiles(t *testing.T) {
	session := newTestSession(t, &stubBackend{}, staticVerifier{identity: validIdentity()}.verify)

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	for i := range tools.Tools {
		if tools.Tools[i].Name != "krkn_run_scenario" {
			continue
		}
		raw, err := json.Marshal(tools.Tools[i].InputSchema)
		if err != nil {
			t.Fatalf("marshal schema: %v", err)
		}
		if strings.Contains(string(raw), "\"files\"") {
			t.Errorf("krkn_run_scenario schema accepts inline files: %s", raw)
		}
		if !strings.Contains(string(raw), "fileReferences") {
			t.Errorf("krkn_run_scenario schema lacks fileReferences: %s", raw)
		}
		return
	}
	t.Fatal("krkn_run_scenario not listed")
}

func TestMCPHTTP_WriteToolsAnnotated(t *testing.T) {
	session := newTestSession(t, &stubBackend{}, staticVerifier{identity: validIdentity()}.verify)

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	want := []string{
		"krkn_run_scenario",
		"krkn_replay_scenario_run",
		"krkn_cancel_job",
		"krkn_delete_scenario_run",
		"krkn_run_workflow",
		"krkn_replay_graph_run",
		"krkn_delete_graph_run",
	}
	byName := make(map[string]*mcp.Tool, len(tools.Tools))
	for i := range tools.Tools {
		byName[tools.Tools[i].Name] = tools.Tools[i]
	}
	for _, name := range want {
		tool, ok := byName[name]
		if !ok {
			t.Errorf("write tool %q not listed", name)
			continue
		}
		if tool.Annotations == nil || tool.Annotations.ReadOnlyHint {
			t.Errorf("tool %q must have ReadOnlyHint false", name)
		}
		if tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint {
			t.Errorf("tool %q must have DestructiveHint true", name)
		}
	}
}
