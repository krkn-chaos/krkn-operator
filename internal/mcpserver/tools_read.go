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

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerReadTools registers the read-only MCP tools.
func registerReadTools(srv *mcp.Server, backend Backend) {
	registerWhoAmI(srv, backend)
	registerScenarioRunReadTools(srv, backend)
	registerGraphReadTools(srv, backend)
}

// registerWhoAmI registers the krkn_whoami tool.
func registerWhoAmI(srv *mcp.Server, backend Backend) {
	falsePtr := false
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_whoami",
		Description: "Show this robot's identity, groups, cluster permissions and token expiry.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: &falsePtr,
		},
	}, withAudit("krkn_whoami", func(WhoAmIInput) []any { return nil },
		func(ctx context.Context, req *mcp.CallToolRequest, in WhoAmIInput) (*mcp.CallToolResult, WhoAmIOutput, error) {
			principal, err := principalFromRequest(req)
			if err != nil {
				return handlerToolError[WhoAmIOutput](&BackendError{Code: CodeForbidden, Message: "unauthenticated"})
			}
			out, err := backend.WhoAmI(ctx, principal)
			if err != nil {
				return handlerToolError[WhoAmIOutput](err)
			}
			return nil, out, nil
		}))
}

// registerScenarioRunReadTools registers the scenario run investigation tools.
func registerScenarioRunReadTools(srv *mcp.Server, backend Backend) {
	falsePtr := false

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_list_clusters",
		Description: "Discover target clusters: reuses a recent robot-owned target request or creates a new one, then lists the clusters this robot has at least one action on.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: &falsePtr,
			IdempotentHint:  true,
		},
	}, withAudit("krkn_list_clusters", func(in ListClustersInput) []any {
		return []any{"targetRequestId", in.TargetRequestID}
	}, toolHandler(backend.ListClusters)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_list_scenario_runs",
		Description: "List scenario runs visible to this robot (view permission on at least one target cluster), newest first.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &falsePtr},
	}, withAudit("krkn_list_scenario_runs", func(in ListScenarioRunsInput) []any {
		return []any{"phase", in.Phase, "graphRunName", in.GraphRunName, "limit", in.Limit, "offset", in.Offset}
	}, toolHandler(backend.ListScenarioRuns)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_get_scenario_run",
		Description: "Get one scenario run with its visible jobs and recomputed counts. Hidden or missing runs both return not_found.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &falsePtr},
	}, withAudit("krkn_get_scenario_run", func(in ScenarioRunRefInput) []any {
		return []any{"scenarioRun", in.ScenarioRunName}
	}, toolHandler(backend.GetScenarioRun)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_get_job_logs",
		Description: "Get the last log lines of a scenario job on a cluster this robot can view. Logs may contain sensitive output.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &falsePtr},
	}, withAudit("krkn_get_job_logs", func(in JobLogsInput) []any {
		return []any{"scenarioRun", in.ScenarioRunName, "jobId", in.JobID}
	}, toolHandler(backend.GetJobLogs)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_get_scenario_run_config",
		Description: "Get the redacted configuration of a scenario run: environment keys and file references only, never values or contents.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &falsePtr},
	}, withAudit("krkn_get_scenario_run_config", func(in ScenarioRunRefInput) []any {
		return []any{"scenarioRun", in.ScenarioRunName}
	}, toolHandler(backend.GetScenarioRunConfig)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_get_report_status",
		Description: "Get whether reports were generated for a scenario run. Reports cannot be downloaded via MCP.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &falsePtr},
	}, withAudit("krkn_get_report_status", func(in ScenarioRunRefInput) []any {
		return []any{"scenarioRun", in.ScenarioRunName}
	}, toolHandler(backend.GetReportStatus)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_list_scenarios",
		Description: "List the scenario tags available in a registry (saved registry name only; never inline credentials).",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &falsePtr},
	}, withAudit("krkn_list_scenarios", func(in ListScenariosInput) []any {
		return []any{"registryName", in.RegistryName}
	}, toolHandler(backend.ListScenarios)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_get_scenario_detail",
		Description: "Get the input fields of a scenario so a run can be built.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &falsePtr},
	}, withAudit("krkn_get_scenario_detail", func(in ScenarioDetailInput) []any {
		return []any{"scenarioName", in.ScenarioName, "registryName", in.RegistryName}
	}, toolHandler(backend.GetScenarioDetail)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_list_available_resources",
		Description: "List saved registries, cloud credentials and files this robot may reference in runs.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &falsePtr},
	}, withAudit("krkn_list_available_resources", func(struct{}) []any { return nil },
		toolHandler(backend.ListAvailableResources)))
}

// toolHandler adapts a Backend method to a tool handler with principal
// extraction and client-safe error mapping.
func toolHandler[In, Out any](method func(context.Context, Principal, In) (Out, error)) mcp.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		principal, err := principalFromRequest(req)
		if err != nil {
			return handlerToolError[Out](&BackendError{Code: CodeForbidden, Message: "unauthenticated"})
		}
		out, err := method(ctx, principal, in)
		if err != nil {
			return handlerToolError[Out](err)
		}
		return nil, out, nil
	}
}

// registerGraphReadTools registers the graph run and workflow read tools.
func registerGraphReadTools(srv *mcp.Server, backend Backend) {
	falsePtr := false

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_list_graph_runs",
		Description: "List graph runs visible to this robot (view permission on at least one resolvable target cluster), newest first.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &falsePtr},
	}, withAudit("krkn_list_graph_runs", func(in ListGraphRunsInput) []any {
		return []any{"phase", in.Phase, "limit", in.Limit, "offset", in.Offset}
	}, toolHandler(backend.ListGraphRuns)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_get_graph_run",
		Description: "Get one graph run with node statuses, redacted definitions and visible scores. Hidden or missing runs both return not_found.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &falsePtr},
	}, withAudit("krkn_get_graph_run", func(in GraphRunRefInput) []any {
		return []any{"graphRun", in.GraphRunName}
	}, toolHandler(backend.GetGraphRun)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_list_workflows",
		Description: "List saved workflow templates accessible to this robot.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &falsePtr},
	}, withAudit("krkn_list_workflows", func(struct{}) []any { return nil },
		toolHandler(backend.ListWorkflows)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_get_workflow",
		Description: "Get a saved workflow's redacted node definitions. Robots can read and execute workflows, not create, edit or delete them.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &falsePtr},
	}, withAudit("krkn_get_workflow", func(in WorkflowRefInput) []any {
		return []any{"workflowId", in.WorkflowID}
	}, toolHandler(backend.GetWorkflow)))
}
