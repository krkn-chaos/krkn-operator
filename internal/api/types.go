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
	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"time"
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
	// TargetId is the UUID of the KrknTargetRequest CR
	TargetId string `json:"targetId"`
	// ClusterName is the name of the target cluster
	ClusterName string `json:"clusterName"`
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

// ScenarioRunResponse represents the response for POST /scenarios/run
type ScenarioRunResponse struct {
	// JobId is the unique job identifier
	JobId string `json:"jobId"`
	// Status is the initial job status (usually "Pending")
	Status string `json:"status"`
	// PodName is the Kubernetes pod name
	PodName string `json:"podName"`
}

// JobStatusResponse represents the response for GET /scenarios/run/{jobId}
type JobStatusResponse struct {
	// JobId is the unique job identifier
	JobId string `json:"jobId"`
	// TargetId is the KrknTargetRequest UUID
	TargetId string `json:"targetId"`
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
