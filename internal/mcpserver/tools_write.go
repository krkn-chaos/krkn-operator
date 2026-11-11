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
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// destructiveSuffix must end every write tool description.
const destructiveSuffix = "This performs a real chaos/destructive operation; confirm with the user before calling."

// registerWriteTools registers the mutation tools.
func registerWriteTools(srv *mcp.Server, backend Backend) {
	truePtr := true
	falsePtr := false

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_run_scenario",
		Description: "Start a new scenario run on the given target clusters. " + destructiveSuffix,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: &truePtr, OpenWorldHint: &falsePtr},
	}, withAudit("krkn_run_scenario", describeRunScenario, toolHandler(backend.RunScenario)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_replay_scenario_run",
		Description: "Re-run an existing scenario run with its original configuration, optionally with a fresh target request. " + destructiveSuffix,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: &truePtr, OpenWorldHint: &falsePtr},
	}, withAudit("krkn_replay_scenario_run", func(in ReplayScenarioRunInput) []any {
		return []any{"scenarioRun", in.ScenarioRunName, "targetRequestId", in.TargetRequestID}
	}, toolHandler(backend.ReplayScenarioRun)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_cancel_job",
		Description: "Cancel one job of a scenario run (requires cancel permission on that job's cluster). " + destructiveSuffix,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: &truePtr, IdempotentHint: true},
	}, withAudit("krkn_cancel_job", func(in CancelJobInput) []any {
		return []any{"scenarioRun", in.ScenarioRunName, "jobId", in.JobID}
	}, toolHandler(backend.CancelJob)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_delete_scenario_run",
		Description: "Delete a scenario run and its jobs (requires cancel permission on every job's cluster). " + destructiveSuffix,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: &truePtr, IdempotentHint: true},
	}, withAudit("krkn_delete_scenario_run", func(in ScenarioRunRefInput) []any {
		return []any{"scenarioRun", in.ScenarioRunName}
	}, toolHandler(backend.DeleteCompleteScenarioRun)))

	registerGraphWriteTools(srv, backend)
}

// describeRunScenario returns audit identifiers, never environment values.
func describeRunScenario(in RunScenarioInput) []any {
	clusters := make([]string, 0, len(in.TargetClusters))
	for _, names := range in.TargetClusters {
		clusters = append(clusters, names...)
	}
	return []any{
		"scenarioName", in.ScenarioName,
		"registryName", in.RegistryName,
		"targetClusters", strings.Join(clusters, ","),
	}
}

// registerGraphWriteTools registers the graph-level mutation tools.
func registerGraphWriteTools(srv *mcp.Server, backend Backend) {
	truePtr := true
	falsePtr := false

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_run_workflow",
		Description: "Start a graph run from a saved workflow on the given target clusters. " + destructiveSuffix,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: &truePtr, OpenWorldHint: &falsePtr},
	}, withAudit("krkn_run_workflow", func(in RunWorkflowInput) []any {
		return []any{"workflowId", in.WorkflowID, "targetClusters", clusterNamesJoined(in.TargetClusters)}
	}, toolHandler(backend.RunWorkflow)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_replay_graph_run",
		Description: "Re-run an existing graph run with its original graph, optionally with a fresh target request. " + destructiveSuffix,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: &truePtr, OpenWorldHint: &falsePtr},
	}, withAudit("krkn_replay_graph_run", func(in ReplayGraphRunInput) []any {
		return []any{"graphRun", in.GraphRunName, "targetRequestId", in.TargetRequestID}
	}, toolHandler(backend.ReplayGraphRun)))

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "krkn_delete_graph_run",
		Description: "Delete a graph run this robot created (requires cancel permission on every target cluster). " + destructiveSuffix,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: &truePtr, IdempotentHint: true},
	}, withAudit("krkn_delete_graph_run", func(in GraphRunRefInput) []any {
		return []any{"graphRun", in.GraphRunName}
	}, toolHandler(backend.DeleteGraphRun)))
}

// clusterNamesJoined flattens target clusters to a comma-separated list.
func clusterNamesJoined(targets map[string][]string) string {
	names := make([]string, 0, len(targets))
	for _, list := range targets {
		names = append(names, list...)
	}
	return strings.Join(names, ",")
}
