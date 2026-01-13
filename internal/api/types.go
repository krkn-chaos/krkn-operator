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
