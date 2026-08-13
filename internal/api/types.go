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
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/api/jobstats"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
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

// DuplicateFileError is returned when a file with the same logical name already exists
type DuplicateFileError struct {
	Name       string
	ExistingID string
}

// Error implements the error interface for DuplicateFileError.
func (e *DuplicateFileError) Error() string {
	return fmt.Sprintf("a file with name '%s' already exists", e.Name)
}

// ScenariosRequest represents the optional request body for POST /scenarios
// If empty, defaults to quay.io; if registryName provided, uses named registry
type ScenariosRequest struct {
	// RegistryName is the name of a saved registry (optional)
	// If omitted, defaults to quay.io public registry
	RegistryName *string `json:"registryName,omitempty"`
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
	Requires          *string `json:"requires,omitempty"`
	MutuallyExcludes  *string `json:"mutually_excludes,omitempty"`
	Secret            bool    `json:"secret,omitempty"`
	Group             *string `json:"group,omitempty"`
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
	// TargetRequestID is the UUID of the KrknTargetRequest (required)
	TargetRequestID string `json:"targetRequestId"`
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
	// Files is an array of file objects to mount in the container (optional, legacy inline file mount)
	Files []FileMount `json:"files,omitempty"`
	// FileReferences are references to centrally-managed files by UUID (optional)
	FileReferences []files.FileReference `json:"fileReferences,omitempty"`
	// CustomRunName is a user-provided label for the run (optional)
	CustomRunName string `json:"customRunName,omitempty"`
	// ElasticsearchConfigName, if set, names a saved Elasticsearch config Secret whose
	// credentials (ES_PASSWORD, and any ES_* vars not already in Environment) are
	// injected server-side so the password is never transmitted by the client.
	ElasticsearchConfigName string `json:"elasticsearchConfigName,omitempty"`
	// Private registry configuration (optional)
	ScenariosRequest
}

// TargetJobResult represents the result of creating a job for a specific target
type TargetJobResult struct {
	// ClusterName is the name of the target cluster
	ClusterName string `json:"clusterName"`
	// JobID is the unique job identifier
	JobID string `json:"jobId"`
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
	// JobID is the unique job identifier
	JobID string `json:"jobId"`
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
	// OwnerUserID is the email address of the user who created this scenario run
	OwnerUserID string `json:"ownerUserId,omitempty"`
	// CustomRunName is the user-provided label for the run, if supplied
	CustomRunName string `json:"customRunName,omitempty"`
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
	// OwnerUserID is the email address of the user who created this scenario run
	OwnerUserID string `json:"ownerUserId,omitempty"`
	// RegistryName is the name of the private registry used (empty for public registry)
	RegistryName string `json:"registryName,omitempty"`
	// GraphRunName is the name of the parent KrknGraphRun (if this scenario run is part of a graph run)
	GraphRunName string `json:"graphRunName,omitempty"`
	// GraphNodeID is the node ID in the graph (if this scenario run is part of a graph run)
	GraphNodeID string `json:"graphNodeId,omitempty"`
	// CustomRunName is the user-provided label for this run
	CustomRunName string `json:"customRunName,omitempty"`
	// ResiliencyScore is the individual resiliency score for this scenario run node
	ResiliencyScore *float64 `json:"resiliencyScore,omitempty"`
	// ResiliencyScores contains per-cluster resiliency scores
	ResiliencyScores []ClusterResiliencyScoreResponse `json:"resiliencyScores,omitempty"`
}

// ClusterJobStatusResponse represents the status of a job for a specific cluster
type ClusterJobStatusResponse struct {
	// ProviderName is the name of the provider that owns this cluster
	ProviderName string `json:"providerName"`
	// ClusterName is the name of the target cluster
	ClusterName string `json:"clusterName"`
	// JobID is the unique identifier for this job
	JobID string `json:"jobId"`
	// PodName is the name of the pod running the scenario
	PodName string `json:"podName,omitempty"`
	// ContainerImage is the full container image path being run
	ContainerImage string `json:"containerImage,omitempty"`
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
	// OwnerUserID is the email address of the user who created this scenario run
	OwnerUserID string `json:"ownerUserId,omitempty"`
	// GraphRunName is the name of the parent KrknGraphRun (if this scenario run is part of a graph run)
	GraphRunName string `json:"graphRunName,omitempty"`
	// GraphNodeID is the node ID in the graph (if this scenario run is part of a graph run)
	GraphNodeID string `json:"graphNodeId,omitempty"`
	// CustomRunName is the user-provided label for this run
	CustomRunName string `json:"customRunName,omitempty"`
	// ResiliencyScore is the individual resiliency score for this scenario run node
	ResiliencyScore *float64 `json:"resiliencyScore,omitempty"`
	// ResiliencyScores contains per-cluster resiliency scores
	ResiliencyScores []ClusterResiliencyScoreResponse `json:"resiliencyScores,omitempty"`
}

// ScenarioRunListResponse represents the response for GET /scenarios/run
type ScenarioRunListResponse struct {
	// ScenarioRuns is the list of scenario runs
	ScenarioRuns []ScenarioRunListItem `json:"scenarioRuns"`
	// Pagination contains pagination metadata
	Pagination PaginationMeta `json:"pagination"`
}

// PaginationMeta contains pagination metadata for paginated responses.
type PaginationMeta struct {
	// Page is the current page number (1-based), 0 if unpaginated
	Page int `json:"page"`
	// Limit is the number of items per page, 0 if unpaginated
	Limit int `json:"limit"`
	// Total is the total number of items matching the query
	Total int `json:"total"`
	// TotalPages is the total number of pages
	TotalPages int `json:"totalPages"`
}

// UnifiedJobItem represents a single item in the unified jobs list.
// It wraps either a ScenarioRun or a GraphRun with a type discriminator.
type UnifiedJobItem struct {
	// Type is the resource type: "scenarioRun" or "graphRun"
	Type string `json:"type"`
	// Name is the resource name
	Name string `json:"name"`
	// CreatedAt is the creation timestamp (used for sorting)
	CreatedAt time.Time `json:"createdAt"`
	// ScenarioRun contains the scenario run data (when Type == "scenarioRun")
	ScenarioRun *ScenarioRunListItem `json:"scenarioRun,omitempty"`
	// GraphRun contains the graph run data (when Type == "graphRun")
	GraphRun *GraphRunListItem `json:"graphRun,omitempty"`
}

func (u UnifiedJobItem) JobType() string { return u.Type }
func (u UnifiedJobItem) ScenarioSucceeded() int {
	if u.ScenarioRun != nil {
		return u.ScenarioRun.SuccessfulJobs
	}
	return 0
}
func (u UnifiedJobItem) ScenarioFailed() int {
	if u.ScenarioRun != nil {
		return u.ScenarioRun.FailedJobs
	}
	return 0
}
func (u UnifiedJobItem) ScenarioRunning() int {
	if u.ScenarioRun != nil {
		return u.ScenarioRun.RunningJobs
	}
	return 0
}
func (u UnifiedJobItem) ScenarioTotalTargets() int {
	if u.ScenarioRun != nil {
		return u.ScenarioRun.TotalTargets
	}
	return 0
}
func (u UnifiedJobItem) GraphTotal() int {
	if u.GraphRun != nil {
		return u.GraphRun.Summary.TotalNodes
	}
	return 0
}
func (u UnifiedJobItem) GraphCompleted() int {
	if u.GraphRun != nil {
		return u.GraphRun.Summary.CompletedNodes
	}
	return 0
}
func (u UnifiedJobItem) GraphFailed() int {
	if u.GraphRun != nil {
		return u.GraphRun.Summary.FailedNodes
	}
	return 0
}

// JobStatsSummary contains aggregate job statistics computed across all runs (not just the current page).
type JobStatsSummary = jobstats.Summary

// UnifiedJobsResponse represents the response for GET /api/v2/jobs
type UnifiedJobsResponse struct {
	// Jobs is the list of unified job items
	Jobs []UnifiedJobItem `json:"jobs"`
	// Pagination contains pagination metadata
	Pagination PaginationMeta `json:"pagination"`
	// Stats contains aggregate job statistics across all runs
	Stats JobStatsSummary `json:"stats"`
}

// ActiveRunsOverviewResponse represents the response for GET /api/v1/dashboard/active-runs
// It provides an overview of currently running scenario runs
type ActiveRunsOverviewResponse struct {
	// TotalActiveRuns is the total number of scenario runs in Running state
	TotalActiveRuns int `json:"totalActiveRuns"`
	// TotalClusters is the total number of unique clusters with active runs
	TotalClusters int `json:"totalClusters"`
	// ClusterRuns is a map of cluster name to list of scenario run names running on that cluster
	ClusterRuns map[string][]string `json:"clusterRuns"`
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

// Authentication types

// IsRegisteredResponse represents the response for GET /auth/is-registered
type IsRegisteredResponse struct {
	// Registered indicates if at least one admin user exists
	Registered bool `json:"registered"`
}

// RegisterRequest represents the request body for POST /auth/register
type RegisterRequest struct {
	// UserID is the email address of the user (required)
	UserID string `json:"userId"`
	// Password is the plaintext password (required, min 8 characters)
	Password string `json:"password"`
	// Name is the first name of the user (required)
	Name string `json:"name"`
	// Surname is the last name of the user (required)
	Surname string `json:"surname"`
	// Organization is the user's organization (optional)
	Organization string `json:"organization,omitempty"`
	// Role is either "user" or "admin" (required)
	Role string `json:"role"`
}

// RegisterResponse represents the response for POST /auth/register
type RegisterResponse struct {
	// Message contains a success message
	Message string `json:"message"`
	// UserID is the registered user's email
	UserID string `json:"userId"`
	// Role is the user's role
	Role string `json:"role"`
}

// LoginRequest represents the request body for POST /auth/login
type LoginRequest struct {
	// UserID is the email address of the user (required)
	UserID string `json:"userId"`
	// Password is the plaintext password (required)
	Password string `json:"password"`
}

// LoginResponse represents the response for POST /auth/login
type LoginResponse struct {
	// Token is the JWT authentication token
	Token string `json:"token"`
	// ExpiresAt is the token expiration timestamp
	ExpiresAt string `json:"expiresAt"`
	// UserID is the authenticated user's email
	UserID string `json:"userId"`
	// Role is the user's role
	Role string `json:"role"`
	// Name is the user's first name
	Name string `json:"name"`
	// Surname is the user's last name
	Surname string `json:"surname"`
}

// User CRUD types

// UserResponse represents a user in API responses (no password)
type UserResponse struct {
	// UserID is the email address of the user
	UserID string `json:"userId"`
	// Name is the first name of the user
	Name string `json:"name"`
	// Surname is the last name of the user
	Surname string `json:"surname"`
	// Organization is the user's organization (optional)
	Organization string `json:"organization,omitempty"`
	// Role is either "user" or "admin"
	Role string `json:"role"`
	// Active indicates if the user account is active
	Active bool `json:"active"`
	// Created is when the user was created
	Created *time.Time `json:"created,omitempty"`
	// LastLogin is when the user last logged in
	LastLogin *time.Time `json:"lastLogin,omitempty"`
}

// ListUsersResponse represents the response for GET /api/v1/users
type ListUsersResponse struct {
	// Users is the array of user objects
	Users []UserResponse `json:"users"`
	// Total is the total number of users matching the filter
	Total int `json:"total"`
	// Page is the current page number
	Page int `json:"page"`
	// Limit is the number of items per page
	Limit int `json:"limit"`
}

// CreateUserRequest represents the request body for POST /api/v1/users
type CreateUserRequest struct {
	// UserID is the email address of the user (required)
	UserID string `json:"userId"`
	// Password is the plaintext password (required, min 8 characters)
	Password string `json:"password"`
	// Name is the first name of the user (required)
	Name string `json:"name"`
	// Surname is the last name of the user (required)
	Surname string `json:"surname"`
	// Organization is the user's organization (optional)
	Organization string `json:"organization,omitempty"`
	// Role is either "user" or "admin" (required)
	Role string `json:"role"`
}

// CreateUserResponse represents the response for POST /api/v1/users
type CreateUserResponse struct {
	// Message contains a success message
	Message string `json:"message"`
	// UserID is the created user's email
	UserID string `json:"userId"`
	// Role is the user's role
	Role string `json:"role"`
}

// UpdateUserRequest represents the request body for PATCH /api/v1/users/:userId
type UpdateUserRequest struct {
	// Name is the first name (optional)
	Name *string `json:"name,omitempty"`
	// Surname is the last name (optional)
	Surname *string `json:"surname,omitempty"`
	// Organization is the user's organization (optional)
	Organization *string `json:"organization,omitempty"`
	// Role is either "user" or "admin" (admin only, optional)
	Role *string `json:"role,omitempty"`
	// Active indicates if the user account is active (admin only, optional)
	Active *bool `json:"active,omitempty"`
}

// UpdateUserResponse represents the response for PATCH /api/v1/users/:userId
type UpdateUserResponse struct {
	// Message contains a success message
	Message string `json:"message"`
	// User is the updated user object
	User UserResponse `json:"user"`
}

// DeleteUserResponse represents the response for DELETE /api/v1/users/:userId
type DeleteUserResponse struct {
	// Message contains a success message
	Message string `json:"message"`
}

// ChangePasswordRequest represents the request body for PATCH /api/v1/users/:userId/password
type ChangePasswordRequest struct {
	// CurrentPassword is the user's current password (required when changing own password)
	CurrentPassword string `json:"currentPassword,omitempty"`
	// NewPassword is the new password (required)
	NewPassword string `json:"newPassword"`
}

// ChangePasswordResponse represents the response for PATCH /api/v1/users/:userId/password
type ChangePasswordResponse struct {
	// Message contains a success message
	Message string `json:"message"`
}

// UserGroup CRUD types

// ClusterPermissionSet defines the actions allowed on a cluster
type ClusterPermissionSet struct {
	// Actions is the list of allowed actions: "view", "run", "cancel"
	Actions []string `json:"actions"`
}

// UserGroupResponse represents a user group in API responses
type UserGroupResponse struct {
	// Name is the group name
	Name string `json:"name"`
	// Description is the group description (optional)
	Description string `json:"description,omitempty"`
	// ClusterPermissions is a map of clusterAPIURL to permitted actions
	ClusterPermissions map[string]ClusterPermissionSet `json:"clusterPermissions"`
	// MemberCount is the number of users in this group (calculated dynamically)
	MemberCount int `json:"memberCount"`
	// CreatedAt is when the group was created
	CreatedAt *time.Time `json:"createdAt,omitempty"`
}

// ListUserGroupsResponse represents the response for GET /api/v1/groups
type ListUserGroupsResponse struct {
	// Groups is the array of user group objects
	Groups []UserGroupResponse `json:"groups"`
	// Total is the total number of groups
	Total int `json:"total"`
}

// CreateUserGroupRequest represents the request body for POST /api/v1/groups
type CreateUserGroupRequest struct {
	// Name is the group name (required)
	Name string `json:"name"`
	// Description is the group description (optional)
	Description string `json:"description,omitempty"`
	// ClusterPermissions is a map of clusterAPIURL to permitted actions (required, min 1)
	ClusterPermissions map[string]ClusterPermissionSet `json:"clusterPermissions"`
	// DiscoveryUUID is the optional UUID of a KrknTargetRequest to delete after group creation
	DiscoveryUUID string `json:"discoveryUuid,omitempty"`
}

// CreateUserGroupResponse represents the response for POST /api/v1/groups
type CreateUserGroupResponse struct {
	// Message contains a success message
	Message string `json:"message"`
	// Name is the created group's name
	Name string `json:"name"`
}

// UpdateUserGroupRequest represents the request body for PATCH /api/v1/groups/:groupName
type UpdateUserGroupRequest struct {
	// Description is the group description (optional)
	Description *string `json:"description,omitempty"`
	// ClusterPermissions is a map of clusterAPIURL to permitted actions (optional)
	ClusterPermissions map[string]ClusterPermissionSet `json:"clusterPermissions,omitempty"`
	// DiscoveryUUID is the optional UUID of a KrknTargetRequest to delete after group update
	DiscoveryUUID string `json:"discoveryUuid,omitempty"`
}

// UpdateUserGroupResponse represents the response for PATCH /api/v1/groups/:groupName
type UpdateUserGroupResponse struct {
	// Message contains a success message
	Message string `json:"message"`
	// Group is the updated group object
	Group UserGroupResponse `json:"group"`
}

// DeleteUserGroupResponse represents the response for DELETE /api/v1/groups/:groupName
type DeleteUserGroupResponse struct {
	// Message contains a success message
	Message string `json:"message"`
}

// AddGroupMemberRequest represents the request body for POST /api/v1/groups/:groupName/members
type AddGroupMemberRequest struct {
	// UserID is the email address of the user to add (required)
	UserID string `json:"userId"`
}

// AddGroupMemberResponse represents the response for POST /api/v1/groups/:groupName/members
type AddGroupMemberResponse struct {
	// Message contains a success message
	Message string `json:"message"`
	// UserID is the added user's email
	UserID string `json:"userId"`
	// GroupName is the group name
	GroupName string `json:"groupName"`
}

// RemoveGroupMemberResponse represents the response for DELETE /api/v1/groups/:groupName/members/:userId
type RemoveGroupMemberResponse struct {
	// Message contains a success message
	Message string `json:"message"`
}

// ListGroupMembersResponse represents the response for GET /api/v1/groups/:groupName/members
type ListGroupMembersResponse struct {
	// Members is the array of user objects in this group
	Members []UserResponse `json:"members"`
	// Total is the total number of members
	Total int `json:"total"`
	// GroupName is the group name
	GroupName string `json:"groupName"`
}

// Graph Run API types

// GraphRunCreateRequest represents the request body for POST /api/v1/graphruns
type GraphRunCreateRequest struct {
	// Graph is the dependency graph of scenarios to execute
	Graph map[string]krknv1alpha1.GraphScenarioNode `json:"graph"`
	// TargetRequestID is the reference to the KrknTargetRequest CR
	TargetRequestID string `json:"targetRequestId"`
	// TargetClusters is a map of provider-name to list of cluster names
	TargetClusters map[string][]string `json:"targetClusters"`
}

// GraphRunListItem represents a single item in the graph runs list
type GraphRunListItem struct {
	Name                    string                      `json:"name"`
	Namespace               string                      `json:"namespace"`
	CreationTimestamp       time.Time                   `json:"creationTimestamp"`
	Phase                   string                      `json:"phase"`
	OwnerUserID             string                      `json:"ownerUserId"`
	TargetRequestID         string                      `json:"targetRequestId"`
	Summary                 GraphRunSummaryResponse     `json:"summary"`
	StartTime               *metav1.Time                `json:"startTime,omitempty"`
	CompletionTime          *metav1.Time                `json:"completionTime,omitempty"`
	ResiliencyScoreEnabled  bool                        `json:"resiliencyScoreEnabled,omitempty"`
	ResiliencyScoreBaseline *float64                    `json:"resiliencyScoreBaseline,omitempty"`
	ResiliencyScores        []GraphClusterScoreResponse `json:"resiliencyScores,omitempty"`
}

// GraphRunListResponse represents the response for GET /api/v1/graphruns
type GraphRunListResponse struct {
	GraphRuns []GraphRunListItem `json:"graphRuns"`
}

// GraphRunDetailResponse represents the detailed response for a single graph run
type GraphRunDetailResponse struct {
	Name              string                 `json:"name"`
	Namespace         string                 `json:"namespace"`
	CreationTimestamp time.Time              `json:"creationTimestamp"`
	Spec              GraphRunSpecResponse   `json:"spec"`
	Status            GraphRunStatusResponse `json:"status"`
}

// GraphRunSpecResponse represents the spec section of a graph run
type GraphRunSpecResponse struct {
	Graph                   map[string]krknv1alpha1.GraphScenarioNode `json:"graph"`
	TargetRequestID         string                                    `json:"targetRequestId"`
	TargetClusters          map[string][]string                       `json:"targetClusters"`
	OwnerUserID             string                                    `json:"ownerUserId"`
	ResiliencyScoreEnabled  bool                                      `json:"resiliencyScoreEnabled,omitempty"`
	ResiliencyMountPath     string                                    `json:"resiliencyMountPath,omitempty"`
	ResiliencyScoreBaseline *float64                                  `json:"resiliencyScoreBaseline,omitempty"`
}

// GraphRunStatusResponse represents the status section of a graph run
type GraphRunStatusResponse struct {
	Phase            string                      `json:"phase"`
	Summary          GraphRunSummaryResponse     `json:"summary"`
	NodeStatuses     []NodeStatusResponse        `json:"nodeStatuses"`
	ResolvedLevels   [][]string                  `json:"resolvedLevels"`
	StartTime        *metav1.Time                `json:"startTime,omitempty"`
	CompletionTime   *metav1.Time                `json:"completionTime,omitempty"`
	ResiliencyScores []GraphClusterScoreResponse `json:"resiliencyScores,omitempty"`
}

// GraphRunSummaryResponse represents aggregate statistics
type GraphRunSummaryResponse struct {
	TotalNodes     int `json:"totalNodes"`
	CompletedNodes int `json:"completedNodes"`
	RunningNodes   int `json:"runningNodes"`
	FailedNodes    int `json:"failedNodes"`
	PendingNodes   int `json:"pendingNodes"`
}

// NodeStatusResponse represents the status of a single node in the graph
type NodeStatusResponse struct {
	NodeID             string                           `json:"nodeId"`
	NodeName           string                           `json:"nodeName"`
	Phase              string                           `json:"phase"`
	ScenarioRunRef     string                           `json:"scenarioRunRef,omitempty"`
	StartTime          *metav1.Time                     `json:"startTime,omitempty"`
	CompletionTime     *metav1.Time                     `json:"completionTime,omitempty"`
	DependsOn          []string                         `json:"dependsOn,omitempty"`
	Message            string                           `json:"message,omitempty"`
	ResiliencyScores   []ClusterResiliencyScoreResponse `json:"resiliencyScores,omitempty"`
	ResiliencyScoreAvg *float64                         `json:"resiliencyScoreAvg,omitempty"`
}

// ClusterResiliencyScoreResponse represents the resiliency score for a specific cluster
type ClusterResiliencyScoreResponse struct {
	ClusterName string  `json:"clusterName"`
	Score       float64 `json:"score"`
}

// GraphClusterScoreResponse represents the aggregated resiliency score for a cluster in a graph run
type GraphClusterScoreResponse struct {
	ProviderName      string             `json:"providerName,omitempty"`
	ClusterName       string             `json:"clusterName"`
	Calculated        float64            `json:"calculated"`
	Baseline          *float64           `json:"baseline,omitempty"`
	Status            string             `json:"status"`
	Message           string             `json:"message,omitempty"`
	NodeContributions map[string]float64 `json:"nodeContributions,omitempty"`
}
