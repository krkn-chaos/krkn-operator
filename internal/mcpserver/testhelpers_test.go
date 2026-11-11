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
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// bearerRoundTripper adds an Authorization: Bearer header to every request.
type bearerRoundTripper struct {
	token string
	base  http.RoundTripper
}

func (rt bearerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+rt.token)
	base := rt.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(clone)
}

// testBearerToken is the fixed token used by test sessions.
const testBearerToken = "krkn_robot_ci-bot.secret"

// newTestSession starts an httptest server around NewHandler and connects a
// client session to it with a bearer token. The server is closed via t.Cleanup.
func newTestSession(t *testing.T, backend Backend, verify TokenVerifier) *mcp.ClientSession {
	t.Helper()
	srv := httptest.NewServer(NewHandler(backend, verify, "test"))
	t.Cleanup(srv.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: srv.URL,
		HTTPClient: &http.Client{
			Transport: bearerRoundTripper{token: testBearerToken},
		},
	}, nil)
	if err != nil {
		t.Fatalf("client connect failed: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// stubBackend lets tests control WhoAmI without the full api package. All
// other tools return not-implemented.
type stubBackend struct {
	whoami WhoAmIOutput
	err    error
}

func (s *stubBackend) WhoAmI(ctx context.Context, p Principal) (WhoAmIOutput, error) {
	return s.whoami, s.err
}

func (s *stubBackend) ListClusters(ctx context.Context, p Principal, in ListClustersInput) (ListClustersOutput, error) {
	return ListClustersOutput{}, errNotImplemented
}

func (s *stubBackend) ListScenarioRuns(ctx context.Context, p Principal, in ListScenarioRunsInput) (ListScenarioRunsOutput, error) {
	return ListScenarioRunsOutput{}, errNotImplemented
}

func (s *stubBackend) GetScenarioRun(ctx context.Context, p Principal, in ScenarioRunRefInput) (ScenarioRunOutput, error) {
	return ScenarioRunOutput{}, errNotImplemented
}

func (s *stubBackend) GetJobLogs(ctx context.Context, p Principal, in JobLogsInput) (JobLogsOutput, error) {
	return JobLogsOutput{}, errNotImplemented
}

func (s *stubBackend) GetScenarioRunConfig(ctx context.Context, p Principal, in ScenarioRunRefInput) (ScenarioRunConfigOutput, error) {
	return ScenarioRunConfigOutput{}, errNotImplemented
}

func (s *stubBackend) GetReportStatus(ctx context.Context, p Principal, in ScenarioRunRefInput) (ReportStatusOutput, error) {
	return ReportStatusOutput{}, errNotImplemented
}

func (s *stubBackend) ListScenarios(ctx context.Context, p Principal, in ListScenariosInput) (ListScenariosOutput, error) {
	return ListScenariosOutput{}, errNotImplemented
}

func (s *stubBackend) GetScenarioDetail(ctx context.Context, p Principal, in ScenarioDetailInput) (ScenarioDetailOutput, error) {
	return ScenarioDetailOutput{}, errNotImplemented
}

func (s *stubBackend) ListAvailableResources(ctx context.Context, p Principal, in struct{}) (AvailableResourcesOutput, error) {
	return AvailableResourcesOutput{}, errNotImplemented
}

// errNotImplemented marks unstubbed backend methods in tests.
var errNotImplemented = &BackendError{Code: CodeInternal, Message: "not implemented in test stub"}

// staticVerifier returns a fixed identity or error.
type staticVerifier struct {
	identity RobotIdentity
	err      error
}

func (v staticVerifier) verify(ctx context.Context, token string) (RobotIdentity, error) {
	return v.identity, v.err
}

func (s *stubBackend) ListGraphRuns(ctx context.Context, p Principal, in ListGraphRunsInput) (ListGraphRunsOutput, error) {
	return ListGraphRunsOutput{}, errNotImplemented
}

func (s *stubBackend) GetGraphRun(ctx context.Context, p Principal, in GraphRunRefInput) (GraphRunOutput, error) {
	return GraphRunOutput{}, errNotImplemented
}

func (s *stubBackend) ListWorkflows(ctx context.Context, p Principal, in struct{}) (ListWorkflowsOutput, error) {
	return ListWorkflowsOutput{}, errNotImplemented
}

func (s *stubBackend) GetWorkflow(ctx context.Context, p Principal, in WorkflowRefInput) (WorkflowOutput, error) {
	return WorkflowOutput{}, errNotImplemented
}

func (s *stubBackend) RunScenario(ctx context.Context, p Principal, in RunScenarioInput) (RunStartedOutput, error) {
	return RunStartedOutput{}, errNotImplemented
}

func (s *stubBackend) ReplayScenarioRun(ctx context.Context, p Principal, in ReplayScenarioRunInput) (RunStartedOutput, error) {
	return RunStartedOutput{}, errNotImplemented
}

func (s *stubBackend) CancelJob(ctx context.Context, p Principal, in CancelJobInput) (ActionOutput, error) {
	return ActionOutput{}, errNotImplemented
}

func (s *stubBackend) DeleteCompleteScenarioRun(ctx context.Context, p Principal, in ScenarioRunRefInput) (ActionOutput, error) {
	return ActionOutput{}, errNotImplemented
}

func (s *stubBackend) RunWorkflow(ctx context.Context, p Principal, in RunWorkflowInput) (RunStartedOutput, error) {
	return RunStartedOutput{}, errNotImplemented
}

func (s *stubBackend) ReplayGraphRun(ctx context.Context, p Principal, in ReplayGraphRunInput) (RunStartedOutput, error) {
	return RunStartedOutput{}, errNotImplemented
}

func (s *stubBackend) DeleteGraphRun(ctx context.Context, p Principal, in GraphRunRefInput) (ActionOutput, error) {
	return ActionOutput{}, errNotImplemented
}
