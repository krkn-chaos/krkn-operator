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

import "time"

// WhoAmIInput is the (empty) input of the krkn_whoami tool.
type WhoAmIInput struct{}

// ClusterPermissionOutput describes the actions a robot may take on a cluster.
type ClusterPermissionOutput struct {
	ClusterAPIURL string   `json:"clusterApiUrl" jsonschema:"API URL of the target cluster"`
	Actions       []string `json:"actions" jsonschema:"sorted list of allowed actions (view, run, cancel)"`
}

// WhoAmIOutput is the output of the krkn_whoami tool.
type WhoAmIOutput struct {
	RobotName      string                    `json:"robotName" jsonschema:"name of the robot account"`
	Groups         []string                  `json:"groups" jsonschema:"group memberships of the robot"`
	TokenExpiresAt time.Time                 `json:"tokenExpiresAt" jsonschema:"expiry time of the current token"`
	Permissions    []ClusterPermissionOutput `json:"permissions" jsonschema:"per-cluster permissions of the robot"`
}

// JobOutput is a single cluster job of a scenario run, as visible to a robot.
type JobOutput struct {
	JobID           string     `json:"jobId" jsonschema:"unique identifier of the job"`
	ProviderName    string     `json:"providerName" jsonschema:"provider that owns the target cluster"`
	ClusterName     string     `json:"clusterName" jsonschema:"name of the target cluster"`
	Phase           string     `json:"phase" jsonschema:"current phase of the job"`
	PodName         string     `json:"podName" jsonschema:"name of the pod running the scenario"`
	StartTime       *time.Time `json:"startTime,omitempty" jsonschema:"when the job started"`
	CompletionTime  *time.Time `json:"completionTime,omitempty" jsonschema:"when the job completed"`
	RetryCount      int        `json:"retryCount" jsonschema:"number of retries used so far"`
	MaxRetries      int        `json:"maxRetries" jsonschema:"maximum number of retries"`
	CancelRequested bool       `json:"cancelRequested" jsonschema:"whether cancellation was requested"`
	FailureReason   string     `json:"failureReason,omitempty" jsonschema:"reason for failure"`
	Message         string     `json:"message,omitempty" jsonschema:"human-readable job message"`
}

// ClusterScoreOutput is a per-cluster resiliency score.
type ClusterScoreOutput struct {
	ClusterName string  `json:"clusterName" jsonschema:"name of the scored cluster"`
	Score       float64 `json:"score" jsonschema:"resiliency score (0-100)"`
	Status      string  `json:"status,omitempty" jsonschema:"score calculation status"`
	Message     string  `json:"message,omitempty" jsonschema:"score message"`
}

// ScenarioRunOutput is a scenario run as visible to a robot. Fields that
// would leak hidden clusters are omitted or recomputed when PartialView.
type ScenarioRunOutput struct {
	Name             string               `json:"name" jsonschema:"name of the scenario run"`
	ScenarioName     string               `json:"scenarioName" jsonschema:"name of the executed scenario"`
	Phase            string               `json:"phase" jsonschema:"overall phase of the run"`
	CustomRunName    string               `json:"customRunName,omitempty" jsonschema:"user-provided run label"`
	OwnerID          string               `json:"ownerId" jsonschema:"principal that owns the run"`
	GraphRunName     string               `json:"graphRunName,omitempty" jsonschema:"parent graph run, if any"`
	GraphNodeID      string               `json:"graphNodeId,omitempty" jsonschema:"parent graph node, if any"`
	CreatedAt        time.Time            `json:"createdAt" jsonschema:"creation timestamp"`
	PartialView      bool                 `json:"partialView" jsonschema:"true when some target clusters are hidden from this robot"`
	VisibleJobs      int                  `json:"visibleJobs" jsonschema:"number of jobs on clusters visible to this robot"`
	SucceededJobs    int                  `json:"succeededJobs" jsonschema:"succeeded jobs among visible ones"`
	FailedJobs       int                  `json:"failedJobs" jsonschema:"failed jobs among visible ones"`
	RunningJobs      int                  `json:"runningJobs" jsonschema:"running jobs among visible ones"`
	Jobs             []JobOutput          `json:"jobs,omitempty" jsonschema:"jobs on visible clusters"`
	ResiliencyScore  *float64             `json:"resiliencyScore,omitempty" jsonschema:"aggregate resiliency score, omitted on partial view"`
	ResiliencyScores []ClusterScoreOutput `json:"resiliencyScores,omitempty" jsonschema:"per-cluster scores, visible clusters only"`
	TargetClusters   map[string][]string  `json:"targetClusters,omitempty" jsonschema:"visible target clusters by provider"`
}

// GraphSummaryOutput aggregates node counts of a graph run.
type GraphSummaryOutput struct {
	TotalNodes     int `json:"totalNodes" jsonschema:"total nodes in the graph"`
	CompletedNodes int `json:"completedNodes" jsonschema:"completed nodes"`
	RunningNodes   int `json:"runningNodes" jsonschema:"running nodes"`
	FailedNodes    int `json:"failedNodes" jsonschema:"failed nodes"`
	PendingNodes   int `json:"pendingNodes" jsonschema:"pending nodes"`
}

// GraphNodeStatusOutput is a node's execution status within a graph run.
type GraphNodeStatusOutput struct {
	NodeID         string   `json:"nodeId" jsonschema:"unique node identifier"`
	NodeName       string   `json:"nodeName" jsonschema:"human-readable node name"`
	Phase          string   `json:"phase" jsonschema:"current node phase"`
	ScenarioRunRef string   `json:"scenarioRunRef,omitempty" jsonschema:"scenario run created for this node"`
	DependsOn      []string `json:"dependsOn,omitempty" jsonschema:"node IDs this node depends on"`
	Message        string   `json:"message,omitempty" jsonschema:"node message, cleared on partial view"`
}

// GraphScoreOutput is a per-cluster graph run resiliency score.
type GraphScoreOutput struct {
	ProviderName string   `json:"providerName,omitempty" jsonschema:"provider that owns the cluster"`
	ClusterName  string   `json:"clusterName" jsonschema:"name of the scored cluster"`
	Calculated   float64  `json:"calculated" jsonschema:"aggregated score, -1 while calculating"`
	Baseline     *float64 `json:"baseline,omitempty" jsonschema:"minimum acceptable score"`
	Status       string   `json:"status,omitempty" jsonschema:"score status"`
}

// GraphNodeDefinitionOutput is a redacted graph node definition. Environment
// values and file contents are never included; only their keys.
type GraphNodeDefinitionOutput struct {
	NodeID             string            `json:"nodeId" jsonschema:"unique node identifier"`
	Name               string            `json:"name,omitempty" jsonschema:"legacy scenario name"`
	ScenarioName       string            `json:"scenarioName,omitempty" jsonschema:"scenario name"`
	RegistryName       string            `json:"registryName,omitempty" jsonschema:"private registry name"`
	Private            *bool             `json:"private,omitempty" jsonschema:"whether the scenario is from a private registry"`
	EnvKeys            []string          `json:"envKeys,omitempty" jsonschema:"sorted environment variable names, values never included"`
	Volumes            map[string]string `json:"volumes,omitempty" jsonschema:"volume mounts (file ID to mount path)"`
	DependsOn          *string           `json:"dependsOn,omitempty" jsonschema:"parent node ID"`
	CloudCredentialRef string            `json:"cloudCredentialRef,omitempty" jsonschema:"cloud credential for this node"`
	ResiliencyWeight   float64           `json:"resiliencyWeight,omitempty" jsonschema:"node weight in the resiliency score"`
}

// GraphRunOutput is a graph run as visible to a robot.
type GraphRunOutput struct {
	Name             string                      `json:"name" jsonschema:"name of the graph run"`
	Phase            string                      `json:"phase" jsonschema:"overall phase"`
	OwnerID          string                      `json:"ownerId" jsonschema:"principal that owns the run"`
	TargetRequestID  string                      `json:"targetRequestId" jsonschema:"target request reference"`
	CreatedAt        time.Time                   `json:"createdAt" jsonschema:"creation timestamp"`
	StartTime        *time.Time                  `json:"startTime,omitempty" jsonschema:"start timestamp"`
	CompletionTime   *time.Time                  `json:"completionTime,omitempty" jsonschema:"completion timestamp"`
	PartialView      bool                        `json:"partialView" jsonschema:"true when some target clusters are hidden"`
	TargetClusters   map[string][]string         `json:"targetClusters,omitempty" jsonschema:"visible target clusters by provider"`
	Summary          GraphSummaryOutput          `json:"summary" jsonschema:"node count summary"`
	Nodes            []GraphNodeStatusOutput     `json:"nodes,omitempty" jsonschema:"node statuses"`
	Definition       []GraphNodeDefinitionOutput `json:"definition,omitempty" jsonschema:"redacted node definitions"`
	ResiliencyScores []GraphScoreOutput          `json:"resiliencyScores,omitempty" jsonschema:"per-cluster scores, visible clusters only"`
}

// FileRefOutput references a mounted file without its content.
type FileRefOutput struct {
	Name      string `json:"name" jsonschema:"name of the file"`
	MountPath string `json:"mountPath" jsonschema:"absolute mount path in the scenario pod"`
	FileID    string `json:"fileId,omitempty" jsonschema:"source ConfigMap UUID"`
}

// ScenarioRefOutput references the scenario of a run.
type ScenarioRefOutput struct {
	Name         string `json:"name" jsonschema:"scenario name"`
	RegistryName string `json:"registryName,omitempty" jsonschema:"private registry name"`
	Private      *bool  `json:"private,omitempty" jsonschema:"whether the scenario is private"`
}

// ScenarioRunConfigOutput is a redacted view of a scenario run configuration.
// Environment values and file contents are never included.
type ScenarioRunConfigOutput struct {
	Name                   string              `json:"name" jsonschema:"name of the scenario run"`
	TargetRequestID        string              `json:"targetRequestId" jsonschema:"target request reference"`
	TargetClusters         map[string][]string `json:"targetClusters,omitempty" jsonschema:"visible target clusters by provider"`
	Scenario               ScenarioRefOutput   `json:"scenario" jsonschema:"scenario reference"`
	EnvKeys                []string            `json:"envKeys,omitempty" jsonschema:"sorted environment variable names, values never included"`
	Files                  []FileRefOutput     `json:"files,omitempty" jsonschema:"mounted files, content never included"`
	CloudCredentialRef     string              `json:"cloudCredentialRef,omitempty" jsonschema:"cloud credential reference"`
	MaxRetries             int                 `json:"maxRetries" jsonschema:"maximum retries per job"`
	RetryBackoff           string              `json:"retryBackoff,omitempty" jsonschema:"retry backoff strategy"`
	RetryDelay             string              `json:"retryDelay,omitempty" jsonschema:"initial retry delay"`
	KubeconfigPath         string              `json:"kubeconfigPath,omitempty" jsonschema:"kubeconfig mount path"`
	CustomRunName          string              `json:"customRunName,omitempty" jsonschema:"user-provided run label"`
	ResiliencyScoreEnabled bool                `json:"resiliencyScoreEnabled,omitempty" jsonschema:"whether scoring is enabled"`
	PartialView            bool                `json:"partialView" jsonschema:"true when some target clusters are hidden"`
}

// ListClustersInput selects the target request to list clusters from.
type ListClustersInput struct {
	TargetRequestID string `json:"targetRequestId,omitempty" jsonschema:"existing target request UUID; a recent robot-owned one is reused when omitted"`
}

// ClusterOutput describes one target cluster and the robot's actions on it.
type ClusterOutput struct {
	ProviderName  string   `json:"providerName" jsonschema:"provider that owns the cluster"`
	ClusterName   string   `json:"clusterName" jsonschema:"name of the cluster"`
	ClusterAPIURL string   `json:"clusterApiUrl" jsonschema:"API URL of the cluster"`
	Online        *bool    `json:"online,omitempty" jsonschema:"whether the cluster API responded to liveness checks"`
	Actions       []string `json:"actions" jsonschema:"sorted actions the robot may perform"`
}

// ListClustersOutput is the cluster discovery result.
type ListClustersOutput struct {
	TargetRequestID string          `json:"targetRequestId" jsonschema:"target request used for discovery"`
	Status          string          `json:"status" jsonschema:"discovery status: Completed or Pending"`
	Clusters        []ClusterOutput `json:"clusters,omitempty" jsonschema:"clusters the robot has at least one action on"`
}

// ListScenarioRunsInput filters the scenario run list.
type ListScenarioRunsInput struct {
	Phase        string `json:"phase,omitempty" jsonschema:"filter by exact phase"`
	GraphRunName string `json:"graphRunName,omitempty" jsonschema:"filter by parent graph run"`
	OwnedByMe    bool   `json:"ownedByMe,omitempty" jsonschema:"only runs owned by this robot"`
	Limit        int    `json:"limit,omitempty" jsonschema:"page size, default 20, max 100"`
	Offset       int    `json:"offset,omitempty" jsonschema:"page offset"`
}

// ListScenarioRunsOutput is a page of visible scenario runs.
type ListScenarioRunsOutput struct {
	Runs   []ScenarioRunOutput `json:"runs" jsonschema:"visible runs, newest first"`
	Total  int                 `json:"total" jsonschema:"total visible after filters"`
	Limit  int                 `json:"limit" jsonschema:"effective page size"`
	Offset int                 `json:"offset" jsonschema:"effective page offset"`
}

// ScenarioRunRefInput identifies a scenario run.
type ScenarioRunRefInput struct {
	ScenarioRunName string `json:"scenarioRunName" jsonschema:"name of the scenario run"`
}

// JobLogsInput requests the logs of one job.
type JobLogsInput struct {
	ScenarioRunName string `json:"scenarioRunName" jsonschema:"name of the scenario run"`
	JobID           string `json:"jobId" jsonschema:"identifier of the job"`
	TailLines       int    `json:"tailLines,omitempty" jsonschema:"last N log lines, default 200, max 1000"`
}

// JobLogsOutput is a bounded, authorized log excerpt. Logs may contain
// sensitive output.
type JobLogsOutput struct {
	JobID     string `json:"jobId" jsonschema:"identifier of the job"`
	PodName   string `json:"podName" jsonschema:"pod the logs come from"`
	Truncated bool   `json:"truncated" jsonschema:"whether the excerpt hit the byte limit"`
	Logs      string `json:"logs" jsonschema:"log text; may contain sensitive output"`
}

// ReportStatusOutput is the report generation status of a run.
type ReportStatusOutput struct {
	ScenarioRunName string     `json:"scenarioRunName" jsonschema:"name of the scenario run"`
	Generated       bool       `json:"generated" jsonschema:"whether reports were generated"`
	HTMLAvailable   bool       `json:"htmlAvailable" jsonschema:"whether an HTML report exists"`
	PDFAvailable    bool       `json:"pdfAvailable" jsonschema:"whether a PDF report exists"`
	GeneratedAt     *time.Time `json:"generatedAt,omitempty" jsonschema:"when reports were generated"`
	Message         string     `json:"message,omitempty" jsonschema:"status message"`
}

// ListScenariosInput selects a registry for the scenario catalog.
type ListScenariosInput struct {
	RegistryName string `json:"registryName,omitempty" jsonschema:"saved private registry name; public when omitted"`
}

// ScenarioSummaryOutput is one catalog entry.
type ScenarioSummaryOutput struct {
	Name            string     `json:"name" jsonschema:"scenario tag name"`
	SignatureStatus string     `json:"signatureStatus,omitempty" jsonschema:"image signature state"`
	LastModified    *time.Time `json:"lastModified,omitempty" jsonschema:"last update time"`
}

// ListScenariosOutput is the scenario catalog.
type ListScenariosOutput struct {
	Scenarios []ScenarioSummaryOutput `json:"scenarios" jsonschema:"available scenario tags"`
}

// ScenarioDetailInput selects a scenario for detail inspection.
type ScenarioDetailInput struct {
	ScenarioName string `json:"scenarioName" jsonschema:"scenario tag name"`
	RegistryName string `json:"registryName,omitempty" jsonschema:"saved private registry name; public when omitted"`
}

// ScenarioInputFieldOutput describes one scenario input field.
type ScenarioInputFieldOutput struct {
	Name              *string `json:"name,omitempty" jsonschema:"field name"`
	ShortDescription  *string `json:"short_description,omitempty" jsonschema:"short description"`
	Description       *string `json:"description,omitempty" jsonschema:"full description"`
	Variable          *string `json:"variable,omitempty" jsonschema:"environment variable name"`
	Type              string  `json:"type" jsonschema:"field type as string"`
	Default           *string `json:"default,omitempty" jsonschema:"default value"`
	Validator         *string `json:"validator,omitempty" jsonschema:"validation rule"`
	ValidationMessage *string `json:"validation_message,omitempty" jsonschema:"validation error message"`
	Separator         *string `json:"separator,omitempty" jsonschema:"list separator"`
	AllowedValues     *string `json:"allowed_values,omitempty" jsonschema:"allowed values"`
	Required          bool    `json:"required,omitempty" jsonschema:"whether the field is required"`
	MountPath         *string `json:"mount_path,omitempty" jsonschema:"mount path for file-type fields"`
	Requires          *string `json:"requires,omitempty" jsonschema:"required companion field"`
	MutuallyExcludes  *string `json:"mutually_excludes,omitempty" jsonschema:"mutually exclusive field"`
	Secret            bool    `json:"secret,omitempty" jsonschema:"whether the value is secret"`
	Group             *string `json:"group,omitempty" jsonschema:"field group"`
}

// ScenarioDetailOutput mirrors the REST scenario detail response.
type ScenarioDetailOutput struct {
	Name            string                     `json:"name" jsonschema:"scenario tag name"`
	Digest          *string                    `json:"digest,omitempty" jsonschema:"image digest"`
	Size            *int64                     `json:"size,omitempty" jsonschema:"image size in bytes"`
	SignatureStatus string                     `json:"signature_status,omitempty" jsonschema:"image signature state"`
	LastModified    *time.Time                 `json:"last_modified,omitempty" jsonschema:"last update time"`
	Title           string                     `json:"title" jsonschema:"scenario title"`
	Description     string                     `json:"description" jsonschema:"scenario description"`
	Fields          []ScenarioInputFieldOutput `json:"fields" jsonschema:"input fields"`
}

// NamedResourceOutput is a name/description pair.
type NamedResourceOutput struct {
	Name        string `json:"name" jsonschema:"resource name"`
	Description string `json:"description,omitempty" jsonschema:"resource description"`
}

// FileResourceOutput is an available file resource.
type FileResourceOutput struct {
	FileID  string `json:"fileId" jsonschema:"file UUID"`
	Name    string `json:"name" jsonschema:"file name"`
	Purpose string `json:"purpose,omitempty" jsonschema:"system-level purpose classification"`
}

// AvailableResourcesOutput lists resources usable in runs.
type AvailableResourcesOutput struct {
	Registries       []NamedResourceOutput `json:"registries" jsonschema:"saved registries available to this robot"`
	CloudCredentials []NamedResourceOutput `json:"cloudCredentials" jsonschema:"cloud credentials available to this robot"`
	Files            []FileResourceOutput  `json:"files" jsonschema:"files available to this robot"`
}

// ListGraphRunsInput filters the graph run list.
type ListGraphRunsInput struct {
	Phase     string `json:"phase,omitempty" jsonschema:"filter by exact phase"`
	OwnedByMe bool   `json:"ownedByMe,omitempty" jsonschema:"only graph runs owned by this robot"`
	Limit     int    `json:"limit,omitempty" jsonschema:"page size, default 20, max 100"`
	Offset    int    `json:"offset,omitempty" jsonschema:"page offset"`
}

// ListGraphRunsOutput is a page of visible graph runs.
type ListGraphRunsOutput struct {
	GraphRuns []GraphRunOutput `json:"graphRuns" jsonschema:"visible graph runs, newest first"`
	Total     int              `json:"total" jsonschema:"total visible after filters"`
	Limit     int              `json:"limit" jsonschema:"effective page size"`
	Offset    int              `json:"offset" jsonschema:"effective page offset"`
}

// GraphRunRefInput identifies a graph run.
type GraphRunRefInput struct {
	GraphRunName string `json:"graphRunName" jsonschema:"name of the graph run"`
}

// WorkflowSummaryOutput is one entry of the workflow catalog.
type WorkflowSummaryOutput struct {
	WorkflowID   string   `json:"workflowId" jsonschema:"workflow UUID"`
	WorkflowName string   `json:"workflowName" jsonschema:"workflow name"`
	Description  string   `json:"description,omitempty" jsonschema:"workflow description"`
	NodeCount    int      `json:"nodeCount" jsonschema:"number of nodes"`
	Categories   []string `json:"categories,omitempty" jsonschema:"visible categories"`
}

// ListWorkflowsOutput is the accessible workflow catalog.
type ListWorkflowsOutput struct {
	Workflows []WorkflowSummaryOutput `json:"workflows" jsonschema:"workflows available to this robot"`
}

// WorkflowRefInput identifies a workflow.
type WorkflowRefInput struct {
	WorkflowID string `json:"workflowId" jsonschema:"workflow UUID"`
}

// WorkflowOutput is a redacted saved workflow. Robots cannot create, edit or
// delete workflows.
type WorkflowOutput struct {
	WorkflowID   string                      `json:"workflowId" jsonschema:"workflow UUID"`
	WorkflowName string                      `json:"workflowName" jsonschema:"workflow name"`
	Description  string                      `json:"description,omitempty" jsonschema:"workflow description"`
	Categories   []string                    `json:"categories,omitempty" jsonschema:"visible categories"`
	Nodes        []GraphNodeDefinitionOutput `json:"nodes" jsonschema:"redacted node definitions"`
}

// FileReferenceInput references a centrally-managed file to mount.
type FileReferenceInput struct {
	FileID    string `json:"fileId" jsonschema:"UUID of the saved file"`
	MountPath string `json:"mountPath" jsonschema:"absolute mount path in the scenario pod"`
}

// RunScenarioInput starts a new scenario run. Only saved files (by UUID) and
// saved registries (by name) can be referenced; inline file contents and
// registry credentials are never accepted.
type RunScenarioInput struct {
	TargetRequestID         string               `json:"targetRequestId" jsonschema:"UUID from krkn_list_clusters"`
	TargetClusters          map[string][]string  `json:"targetClusters" jsonschema:"target clusters by provider name"`
	ScenarioName            string               `json:"scenarioName" jsonschema:"scenario tag name"`
	RegistryName            string               `json:"registryName,omitempty" jsonschema:"saved private registry name"`
	PrivateRegistry         bool                 `json:"privateRegistry,omitempty" jsonschema:"whether the scenario comes from a private registry"`
	Environment             map[string]string    `json:"environment,omitempty" jsonschema:"environment variables for the scenario pod"`
	FileReferences          []FileReferenceInput `json:"fileReferences,omitempty" jsonschema:"saved files to mount, by UUID"`
	CustomRunName           string               `json:"customRunName,omitempty" jsonschema:"user-facing run label"`
	Categories              []string             `json:"categories,omitempty" jsonschema:"category names for the run"`
	MaxRetries              *int                 `json:"maxRetries,omitempty" jsonschema:"maximum retries per job"`
	CloudCredentialRef      string               `json:"cloudCredentialRef,omitempty" jsonschema:"saved cloud credential name"`
	ElasticsearchConfigName string               `json:"elasticsearchConfigName,omitempty" jsonschema:"saved Elasticsearch config name"`
	ResiliencyScoreEnabled  bool                 `json:"resiliencyScoreEnabled,omitempty" jsonschema:"enable resiliency scoring"`
	IdempotencyKey          string               `json:"idempotencyKey,omitempty" jsonschema:"optional client-chosen key; retrying the same key returns the existing run instead of starting a duplicate"`
}

// ReplayScenarioRunInput re-runs an existing scenario run.
type ReplayScenarioRunInput struct {
	ScenarioRunName string `json:"scenarioRunName" jsonschema:"name of the run to replay"`
	TargetRequestID string `json:"targetRequestId,omitempty" jsonschema:"fresh target request UUID; the original one is used when omitted"`
	CustomRunName   string `json:"customRunName,omitempty" jsonschema:"user-facing label for the new run"`
}

// CancelJobInput cancels one job of a run.
type CancelJobInput struct {
	ScenarioRunName string `json:"scenarioRunName" jsonschema:"name of the scenario run"`
	JobID           string `json:"jobId" jsonschema:"identifier of the job to cancel"`
}

// StartedJobOutput is the per-target result of starting a run.
type StartedJobOutput struct {
	ClusterName string `json:"clusterName" jsonschema:"target cluster name"`
	JobID       string `json:"jobId" jsonschema:"created job identifier"`
	Status      string `json:"status" jsonschema:"initial job status"`
	Success     bool   `json:"success" jsonschema:"whether the job was created"`
	Error       string `json:"error,omitempty" jsonschema:"creation error"`
}

// RunStartedOutput is the result of starting a scenario or graph run.
type RunStartedOutput struct {
	Kind string             `json:"kind" jsonschema:"scenario_run or graph_run"`
	Name string             `json:"name" jsonschema:"name of the created run"`
	Jobs []StartedJobOutput `json:"jobs,omitempty" jsonschema:"per-target job results"`
}

// ActionOutput is the result of a cancel or delete.
type ActionOutput struct {
	Message string `json:"message" jsonschema:"human-readable result"`
}

// RunWorkflowInput starts a graph run from a saved workflow.
type RunWorkflowInput struct {
	WorkflowID             string              `json:"workflowId" jsonschema:"UUID of the saved workflow"`
	TargetRequestID        string              `json:"targetRequestId" jsonschema:"UUID from krkn_list_clusters"`
	TargetClusters         map[string][]string `json:"targetClusters" jsonschema:"target clusters by provider name"`
	Categories             []string            `json:"categories,omitempty" jsonschema:"category names for the run"`
	CloudCredentialRef     string              `json:"cloudCredentialRef,omitempty" jsonschema:"saved cloud credential name"`
	MaxRetries             *int                `json:"maxRetries,omitempty" jsonschema:"maximum retries per node"`
	ResiliencyScoreEnabled bool                `json:"resiliencyScoreEnabled,omitempty" jsonschema:"enable resiliency scoring"`
	ResiliencyBaseline     *float64            `json:"resiliencyBaseline,omitempty" jsonschema:"minimum acceptable score, required when scoring is enabled"`
	ResiliencyMountPath    string              `json:"resiliencyMountPath,omitempty" jsonschema:"absolute path of the metrics file in node volumes"`
	IdempotencyKey         string              `json:"idempotencyKey,omitempty" jsonschema:"optional client-chosen key; retrying the same key returns the existing run instead of starting a duplicate"`
}

// ReplayGraphRunInput re-runs an existing graph run.
type ReplayGraphRunInput struct {
	GraphRunName    string `json:"graphRunName" jsonschema:"name of the graph run to replay"`
	TargetRequestID string `json:"targetRequestId,omitempty" jsonschema:"fresh target request UUID; the original one is used when omitted"`
}
