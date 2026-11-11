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

// Package mcpserver hosts the krkn-operator MCP endpoint: the SDK server,
// tool schemas, robot bearer-token authentication and audit logging.
// It depends only on the Backend interface implemented in internal/api,
// so all authorization and data access stay in the existing REST layer.
package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Principal identifies the authenticated caller of an MCP tool.
type Principal struct {
	RobotName string
}

// RobotIdentity is the verified identity behind a bearer token.
type RobotIdentity struct {
	Name      string
	ExpiresAt time.Time
}

// TokenVerifier validates a robot bearer token.
type TokenVerifier func(ctx context.Context, token string) (RobotIdentity, error)

// ErrInvalidCredentials is returned by a TokenVerifier when the token is not
// valid. It maps to a 401; any other verifier error maps to a generic 500.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrorCode is a stable, client-safe error code for tool failures.
type ErrorCode string

// Error codes surfaced to MCP clients.
const (
	CodeNotFound    ErrorCode = "not_found"
	CodeForbidden   ErrorCode = "forbidden"
	CodeInvalidArg  ErrorCode = "invalid_argument"
	CodeConflict    ErrorCode = "conflict"
	CodeUnavailable ErrorCode = "unavailable"
	CodeInternal    ErrorCode = "internal"
)

// BackendError is a client-safe error carrying a stable code.
type BackendError struct {
	Code    ErrorCode
	Message string
}

func (e *BackendError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Backend is the interface between the MCP server and the krkn-operator API
// layer. It grows one method per tool family in later tasks.
type Backend interface {
	WhoAmI(ctx context.Context, p Principal) (WhoAmIOutput, error)
	ListClusters(ctx context.Context, p Principal, in ListClustersInput) (ListClustersOutput, error)
	ListScenarioRuns(ctx context.Context, p Principal, in ListScenarioRunsInput) (ListScenarioRunsOutput, error)
	GetScenarioRun(ctx context.Context, p Principal, in ScenarioRunRefInput) (ScenarioRunOutput, error)
	GetJobLogs(ctx context.Context, p Principal, in JobLogsInput) (JobLogsOutput, error)
	GetScenarioRunConfig(ctx context.Context, p Principal, in ScenarioRunRefInput) (ScenarioRunConfigOutput, error)
	GetReportStatus(ctx context.Context, p Principal, in ScenarioRunRefInput) (ReportStatusOutput, error)
	ListScenarios(ctx context.Context, p Principal, in ListScenariosInput) (ListScenariosOutput, error)
	GetScenarioDetail(ctx context.Context, p Principal, in ScenarioDetailInput) (ScenarioDetailOutput, error)
	ListAvailableResources(ctx context.Context, p Principal, in struct{}) (AvailableResourcesOutput, error)
	ListGraphRuns(ctx context.Context, p Principal, in ListGraphRunsInput) (ListGraphRunsOutput, error)
	GetGraphRun(ctx context.Context, p Principal, in GraphRunRefInput) (GraphRunOutput, error)
	ListWorkflows(ctx context.Context, p Principal, in struct{}) (ListWorkflowsOutput, error)
	GetWorkflow(ctx context.Context, p Principal, in WorkflowRefInput) (WorkflowOutput, error)
	RunScenario(ctx context.Context, p Principal, in RunScenarioInput) (RunStartedOutput, error)
	ReplayScenarioRun(ctx context.Context, p Principal, in ReplayScenarioRunInput) (RunStartedOutput, error)
	CancelJob(ctx context.Context, p Principal, in CancelJobInput) (ActionOutput, error)
	DeleteCompleteScenarioRun(ctx context.Context, p Principal, in ScenarioRunRefInput) (ActionOutput, error)
	RunWorkflow(ctx context.Context, p Principal, in RunWorkflowInput) (RunStartedOutput, error)
	ReplayGraphRun(ctx context.Context, p Principal, in ReplayGraphRunInput) (RunStartedOutput, error)
	DeleteGraphRun(ctx context.Context, p Principal, in GraphRunRefInput) (ActionOutput, error)
}
