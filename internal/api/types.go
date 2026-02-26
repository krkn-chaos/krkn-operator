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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// ClustersResponse represents the response for GET /clusters endpoint
type ClustersResponse struct {
	// TargetData contains a map of operator-name to list of cluster targets
	TargetData map[string][]krknv1alpha1.ClusterTarget `json:"targetData"`
	// Status represents the current state of the request (pending, completed)
	Status string `json:"status"`
}

// NodesResponse represents the response for GET /nodes endpoint
type NodesResponse struct {
	// Nodes contains the list of node names in the cluster
	Nodes []string `json:"nodes"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// ScenariosRequest represents the optional request body for POST /scenarios
// If provided, uses private registry; if nil/empty, defaults to quay.io
type ScenariosRequest struct {
	// Username for private registry authentication (optional)
	Username *string `json:"username,omitempty"`
	// Password for private registry authentication (optional)
	Password *string `json:"password,omitempty"`
	// Token for private registry authentication (optional, alternative to username/password)
	Token *string `json:"token,omitempty"`
	// RegistryURL is the private registry URL (required if using private registry)
	RegistryURL string `json:"registryUrl,omitempty"`
	// ScenarioRepository is the scenario repository name (required if using private registry)
	ScenarioRepository string `json:"scenarioRepository,omitempty"`
	// SkipTLS skips TLS verification for private registry
	SkipTLS bool `json:"skipTls,omitempty"`
	// Insecure allows insecure connections to private registry
	Insecure bool `json:"insecure,omitempty"`
}

// ScenarioTag represents a scenario available in the registry
type ScenarioTag struct {
	// Name is the scenario tag/version name
	Name string `json:"name"`
	// Digest is the image digest (optional)
	Digest *string `json:"digest,omitempty"`
	// Size is the image size in bytes (optional)
	Size *int64 `json:"size,omitempty"`
	// LastModified is when the scenario was last updated (optional)
	LastModified *time.Time `json:"lastModified,omitempty"`
}

// ScenariosResponse represents the response for POST /scenarios endpoint
type ScenariosResponse struct {
	// Scenarios contains the list of available scenario tags
	Scenarios []ScenarioTag `json:"scenarios"`
}

// InputFieldResponse represents a scenario input field with Type as string
// This is a wrapper around krknctl typing.InputField to ensure Type is serialized as string
type InputFieldResponse struct {
	Name              *string `json:"name"`
	ShortDescription  *string `json:"short_description,omitempty"`
	Description       *string `json:"description,omitempty"`
	Variable          *string `json:"variable"`
	Type              string  `json:"type"` // String representation instead of int64 enum
	Default           *string `json:"default,omitempty"`
	Validator         *string `json:"validator,omitempty"`
	ValidationMessage *string `json:"validation_message,omitempty"`
	Separator         *string `json:"separator,omitempty"`
	AllowedValues     *string `json:"allowed_values,omitempty"`
	Required          bool    `json:"required,omitempty"`
	MountPath         *string `json:"mount_path,omitempty"`
	Requires          *string `json:"requires,omitempty"`
	MutuallyExcludes  *string `json:"mutually_excludes,omitempty"`
	Secret            bool    `json:"secret,omitempty"`
}

// ScenarioDetailResponse represents the response for POST /scenarios/detail/{scenario_name}
// This wraps krknctl models.ScenarioDetail to ensure Type fields are strings
type ScenarioDetailResponse struct {
	Name         string               `json:"name"`
	Digest       *string              `json:"digest,omitempty"`
	Size         *int64               `json:"size,omitempty"`
	LastModified *time.Time           `json:"last_modified,omitempty"`
	Title        string               `json:"title"`
	Description  string               `json:"description"`
	Fields       []InputFieldResponse `json:"fields"`
}

// GlobalsRequest represents the request body for POST /scenarios/globals
type GlobalsRequest struct {
	ScenariosRequest
	// ScenarioNames is the list of scenario names to get global environments for
	ScenarioNames []string `json:"scenarioNames"`
}

// GlobalsResponse represents the response for POST /scenarios/globals endpoint
type GlobalsResponse struct {
	// Globals is a map of scenario name to global environment details
	Globals map[string]ScenarioDetailResponse `json:"globals"`
}

// FileMount represents a file to be mounted in the scenario pod
type FileMount struct {
	// Name is the file name
	Name string `json:"name"`
	// Content is the base64-encoded file content
	Content string `json:"content"`
	// MountPath is the absolute path where the file should be mounted
	MountPath string `json:"mountPath"`
}

// ScenarioRunRequest represents the request body for POST /scenarios/run
type ScenarioRunRequest struct {
	// TargetRequestId is the UUID of the KrknTargetRequest (required)
	TargetRequestId string `json:"targetRequestId"`
	// TargetClusters is a map of provider-name to list of cluster names
	// Example: {"krkn-operator": ["cluster1", "cluster2"], "krkn-operator-acm": ["cluster3"]}
	TargetClusters map[string][]string `json:"targetClusters"`

	// ScenarioImage is the container image to run
	ScenarioImage string `json:"scenarioImage"`
	// ScenarioName is the name of the scenario being executed
	ScenarioName string `json:"scenarioName"`
	// KubeconfigPath is the path where kubeconfig should be mounted (optional, default: /home/krkn/.kube/config)
	KubeconfigPath string `json:"kubeconfigPath,omitempty"`
	// Environment is a map of environment variables to pass to the container (optional)
	Environment map[string]string `json:"environment,omitempty"`
	// Files is an array of file objects to mount in the container (optional)
	Files []FileMount `json:"files,omitempty"`
	// Private registry configuration (optional)
	ScenariosRequest
}

// TargetJobResult represents the result of creating a job for a specific target
type TargetJobResult struct {
	// ClusterName is the name of the target cluster
	ClusterName string `json:"clusterName"`
	// JobId is the unique job identifier
	JobId string `json:"jobId"`
	// Status is the initial job status (usually "Pending" or "Failed")
	Status string `json:"status"`
	// PodName is the Kubernetes pod name
	PodName string `json:"podName"`
	// Success indicates if the job was created successfully
	Success bool `json:"success"`
	// Error contains error message if Success is false
	Error string `json:"error,omitempty"`
}

// ScenarioRunResponse represents the response for POST /scenarios/run
type ScenarioRunResponse struct {
	// Jobs is the array of job results for each target
	Jobs []TargetJobResult `json:"jobs"`
	// TotalTargets is the total number of targets requested
	TotalTargets int `json:"totalTargets"`
	// SuccessfulJobs is the number of jobs created successfully
	SuccessfulJobs int `json:"successfulJobs"`
	// FailedJobs is the number of jobs that failed to create
	FailedJobs int `json:"failedJobs"`
}

// JobStatusResponse represents the response for GET /scenarios/run/{jobId}
type JobStatusResponse struct {
	// JobId is the unique job identifier
	JobId string `json:"jobId"`
	// ClusterName is the target cluster name
	ClusterName string `json:"clusterName"`
	// ScenarioName is the scenario name
	ScenarioName string `json:"scenarioName"`
	// Status is the current job status (Pending, Running, Succeeded, Failed, Stopped)
	Status string `json:"status"`
	// PodName is the Kubernetes pod name
	PodName string `json:"podName"`
	// StartTime is when the job started (optional)
	StartTime *time.Time `json:"startTime,omitempty"`
	// CompletionTime is when the job completed (optional)
	CompletionTime *time.Time `json:"completionTime,omitempty"`
	// Message is additional status message or error details (optional)
	Message string `json:"message,omitempty"`
}

// JobsListResponse represents the response for GET /scenarios/run
type JobsListResponse struct {
	// Jobs is the array of job status objects
	Jobs []JobStatusResponse `json:"jobs"`
}

// CreateTargetRequest represents the request body for POST /api/v1/targets
type CreateTargetRequest struct {
	// ClusterName is the name of the target cluster (required)
	ClusterName string `json:"clusterName"`

	// ClusterAPIURL is the Kubernetes API server URL (optional if kubeconfig provided)
	ClusterAPIURL string `json:"clusterAPIURL,omitempty"`

	// SecretType specifies the authentication method: "kubeconfig", "token", or "credentials"
	SecretType string `json:"secretType"`

	// CABundle is the base64-encoded CA certificate bundle (optional)
	CABundle string `json:"caBundle,omitempty"`

	// Credentials - provide ONE of the following based on SecretType:

	// Kubeconfig (base64-encoded) - for SecretType="kubeconfig"
	Kubeconfig string `json:"kubeconfig,omitempty"`

	// Token - for SecretType="token"
	Token string `json:"token,omitempty"`

	// Username - for SecretType="credentials"
	Username string `json:"username,omitempty"`

	// Password - for SecretType="credentials"
	Password string `json:"password,omitempty"`
}

// CreateTargetResponse represents the response for POST /api/v1/targets
type CreateTargetResponse struct {
	// UUID is the unique identifier for the created target
	UUID string `json:"uuid"`

	// Message contains additional information
	Message string `json:"message,omitempty"`
}

// TargetResponse represents a single target in responses
type TargetResponse struct {
	// UUID is the unique identifier
	UUID string `json:"uuid"`

	// ClusterName is the name of the target cluster
	ClusterName string `json:"clusterName"`

	// ClusterAPIURL is the Kubernetes API server URL
	ClusterAPIURL string `json:"clusterAPIURL"`

	// SecretType is the authentication method
	SecretType string `json:"secretType"`

	// Ready indicates if the target is ready
	Ready bool `json:"ready"`

	// CreatedAt is the creation timestamp
	CreatedAt *time.Time `json:"createdAt,omitempty"`
}

// ListTargetsResponse represents the response for GET /api/v1/targets
type ListTargetsResponse struct {
	// Targets is the array of target objects
	Targets []TargetResponse `json:"targets"`
}

// UpdateTargetRequest represents the request body for PUT /api/v1/targets/{uuid}
type UpdateTargetRequest struct {
	CreateTargetRequest
}

// ScenarioRunCreateResponse represents the response for POST /scenarios/run (new CRD-based approach)
type ScenarioRunCreateResponse struct {
	// ScenarioRunName is the name of the created KrknScenarioRun CR
	ScenarioRunName string `json:"scenarioRunName"`
	// TargetClusters is a map of provider-name to list of cluster names
	TargetClusters map[string][]string `json:"targetClusters"`
	// TotalTargets is the total number of target clusters
	TotalTargets int `json:"totalTargets"`
}

// ScenarioRunStatusResponse represents the response for GET /scenarios/run/{scenarioRunName} (new CRD-based approach)
type ScenarioRunStatusResponse struct {
	// ScenarioRunName is the name of the KrknScenarioRun CR
	ScenarioRunName string `json:"scenarioRunName"`
	// Phase is the overall phase of the scenario run
	Phase string `json:"phase"`
	// TotalTargets is the total number of target clusters
	TotalTargets int `json:"totalTargets"`
	// SuccessfulJobs is the number of successfully completed jobs
	SuccessfulJobs int `json:"successfulJobs"`
	// FailedJobs is the number of failed jobs
	FailedJobs int `json:"failedJobs"`
	// RunningJobs is the number of currently running jobs
	RunningJobs int `json:"runningJobs"`
	// ClusterJobs contains the status of each cluster job
	ClusterJobs []ClusterJobStatusResponse `json:"clusterJobs"`
}

// ClusterJobStatusResponse represents the status of a job for a specific cluster
type ClusterJobStatusResponse struct {
	// ProviderName is the name of the provider that owns this cluster
	ProviderName string `json:"providerName"`
	// ClusterName is the name of the target cluster
	ClusterName string `json:"clusterName"`
	// JobId is the unique identifier for this job
	JobId string `json:"jobId"`
	// PodName is the name of the pod running the scenario
	PodName string `json:"podName,omitempty"`
	// Phase is the current phase of the job
	Phase string `json:"phase"`
	// StartTime is when the job started
	StartTime *time.Time `json:"startTime,omitempty"`
	// CompletionTime is when the job completed
	CompletionTime *time.Time `json:"completionTime,omitempty"`
	// Message contains additional information about the job status
	Message string `json:"message,omitempty"`
	// RetryCount is the number of times this job has been retried
	RetryCount int `json:"retryCount,omitempty"`
	// MaxRetries is the maximum number of retries allowed
	MaxRetries int `json:"maxRetries,omitempty"`
	// CancelRequested indicates if cancellation was requested
	CancelRequested bool `json:"cancelRequested,omitempty"`
	// FailureReason contains the categorized failure reason
	FailureReason string `json:"failureReason,omitempty"`
}

// ScenarioRunListItem represents a single scenario run in the list view
type ScenarioRunListItem struct {
	// ScenarioRunName is the name of the KrknScenarioRun CR
	ScenarioRunName string `json:"scenarioRunName"`
	// ScenarioName is the name of the scenario being executed
	ScenarioName string `json:"scenarioName"`
	// Phase is the overall phase of the scenario run
	Phase string `json:"phase"`
	// TotalTargets is the total number of target clusters
	TotalTargets int `json:"totalTargets"`
	// SuccessfulJobs is the number of successfully completed jobs
	SuccessfulJobs int `json:"successfulJobs"`
	// FailedJobs is the number of failed jobs
	FailedJobs int `json:"failedJobs"`
	// RunningJobs is the number of currently running jobs
	RunningJobs int `json:"runningJobs"`
	// CreatedAt is the creation timestamp
	CreatedAt time.Time `json:"createdAt"`
}

// ScenarioRunListResponse represents the response for GET /scenarios/run
type ScenarioRunListResponse struct {
	// ScenarioRuns is the list of scenario runs
	ScenarioRuns []ScenarioRunListItem `json:"scenarioRuns"`
}

// ProviderConfigUpdateRequest is the request body for POST /api/v1/provider-config/{uuid}
type ProviderConfigUpdateRequest struct {
	// ProviderName is the name of the provider whose config to update
	ProviderName string `json:"provider_name"`
	// Values is a map of configuration keys to values (all values are strings)
	Values map[string]string `json:"values"`
}

// ProviderConfigUpdateResponse is the response for successful config updates
type ProviderConfigUpdateResponse struct {
	// Message contains a success message
	Message string `json:"message"`
	// UpdatedFields is the list of fields that were updated
	UpdatedFields []string `json:"updatedFields,omitempty"`
}

// ProviderResponse represents a single provider in the list
type ProviderResponse struct {
	// Name is the operator name
	Name string `json:"name"`
	// Active indicates if the provider is active
	Active bool `json:"active"`
	// LastHeartbeat is the timestamp of the last heartbeat
	LastHeartbeat *metav1.Time `json:"lastHeartbeat,omitempty"`
}

// ListProvidersResponse is the response for GET /api/v1/providers
type ListProvidersResponse struct {
	// Providers is the list of registered providers
	Providers []ProviderResponse `json:"providers"`
}

// UpdateProviderStatusRequest is the request body for PATCH /api/v1/providers/{name}
type UpdateProviderStatusRequest struct {
	// Active sets the provider active status
	Active bool `json:"active"`
}

// UpdateProviderStatusResponse is the response for successful provider status updates
type UpdateProviderStatusResponse struct {
	// Message contains a success message
	Message string `json:"message"`
	// Name is the provider name
	Name string `json:"name"`
	// Active is the new active status
	Active bool `json:"active"`
}
