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

// Package registry provides functionality for managing private container registries
// in the krkn-operator ecosystem. It handles conversion between Kubernetes Secrets
// and krknctl RegistryV2 configurations, manages labels and annotations for
// group-based access control, and defines API types for registry CRUD operations.
package registry

import "errors"

// DockerConfig represents the structure of a Docker config.json.
type DockerConfig struct {
	Auths map[string]AuthEntry `json:"auths"`
}

// AuthEntry represents an authentication entry in Docker config.
type AuthEntry struct {
	Auth string `json:"auth"`
}

// AITagMetadata contains registry-local tag selections used by Krkn-AI.
type AITagMetadata struct {
	// AIOrchestratorTag defaults to krkn-ai-operator when empty.
	AIOrchestratorTag string `json:"aiOrchestratorTag,omitempty"`
	// AIResultUploaderTag defaults to the krkn-ai-service image tag when empty.
	AIResultUploaderTag string `json:"aiResultUploaderTag,omitempty"`
	// WorkloadImageTags override helper tags; empty values use defaults.
	WorkloadImageTags map[string]string `json:"workloadImageTags,omitempty"`
}

// CreateRegistryRequest represents the request to create a registry.
type CreateRegistryRequest struct {
	Name               string   `json:"name"`
	RegistryURL        string   `json:"registryUrl"`
	ScenarioRepository string   `json:"scenarioRepository"`
	AuthType           string   `json:"authType"`
	Username           string   `json:"username"`
	Password           string   `json:"password"`
	SkipTLS            bool     `json:"skipTls,omitempty"`
	Insecure           bool     `json:"insecure,omitempty"`
	Description        string   `json:"description,omitempty"`
	Groups             []string `json:"groups,omitempty"`
	AvailableToAll     bool     `json:"availableToAll,omitempty"`
	AITagMetadata
}

// UpdateRegistryRequest represents a request to update a registry. Omitted AI
// image fields preserve their saved values; explicit empty values clear them.
type UpdateRegistryRequest struct {
	RegistryURL         string             `json:"registryUrl"`
	ScenarioRepository  string             `json:"scenarioRepository"`
	AuthType            string             `json:"authType"`
	Username            string             `json:"username,omitempty"`
	Password            string             `json:"password,omitempty"`
	SkipTLS             bool               `json:"skipTls,omitempty"`
	Insecure            bool               `json:"insecure,omitempty"`
	Description         string             `json:"description,omitempty"`
	Groups              []string           `json:"groups,omitempty"`
	AvailableToAll      bool               `json:"availableToAll,omitempty"`
	AIOrchestratorTag   *string            `json:"aiOrchestratorTag,omitempty"`
	AIResultUploaderTag *string            `json:"aiResultUploaderTag,omitempty"`
	WorkloadImageTags   *map[string]string `json:"workloadImageTags,omitempty"`
}

// MergeAITagMetadata applies only tag fields present in an update request.
func (r UpdateRegistryRequest) MergeAITagMetadata(current AITagMetadata) AITagMetadata {
	if r.AIOrchestratorTag != nil {
		current.AIOrchestratorTag = *r.AIOrchestratorTag
	}
	if r.AIResultUploaderTag != nil {
		current.AIResultUploaderTag = *r.AIResultUploaderTag
	}
	if r.WorkloadImageTags != nil {
		current.WorkloadImageTags = *r.WorkloadImageTags
	}
	return current
}

// ValidateAITagMetadata validates the fields explicitly supplied by an update.
func (r UpdateRegistryRequest) ValidateAITagMetadata() error {
	var metadata AITagMetadata
	if r.AIOrchestratorTag != nil {
		metadata.AIOrchestratorTag = *r.AIOrchestratorTag
	}
	if r.AIResultUploaderTag != nil {
		metadata.AIResultUploaderTag = *r.AIResultUploaderTag
	}
	if r.WorkloadImageTags != nil {
		metadata.WorkloadImageTags = *r.WorkloadImageTags
	}
	return ValidateAITagMetadata(metadata)
}

// RegistryResponse represents a registry in API responses.
type RegistryResponse struct {
	Name               string   `json:"name"`
	RegistryURL        string   `json:"registryUrl"`
	ScenarioRepository string   `json:"scenarioRepository"`
	AuthType           string   `json:"authType"`
	Description        string   `json:"description,omitempty"`
	Groups             []string `json:"groups,omitempty"`
	AvailableToAll     bool     `json:"availableToAll"`
	SkipTLS            bool     `json:"skipTls"`
	Insecure           bool     `json:"insecure"`
	CreatedAt          string   `json:"createdAt,omitempty"`
	CreatedBy          string   `json:"createdBy,omitempty"`
	UpdatedAt          string   `json:"updatedAt,omitempty"`
	UpdatedBy          string   `json:"updatedBy,omitempty"`
	AITagMetadata
}

// AIImageInventory is the non-secret registry inventory passed to Krkn-AI.
type AIImageInventory struct {
	RegistryName        string            `json:"registryName"`
	Repository          string            `json:"repository"`
	AvailableTags       []string          `json:"availableTags"`
	WorkloadImageTags   map[string]string `json:"workloadImageTags"`
	OrchestratorImage   string            `json:"orchestratorImage"`
	ResultUploaderImage string            `json:"resultUploaderImage"`
}

// ErrAIRegistryConfiguration identifies invalid registry metadata or missing
// required private images.
var ErrAIRegistryConfiguration = errors.New("invalid private Krkn-AI registry configuration")

// ListRegistriesResponse represents the response for listing registries.
type ListRegistriesResponse struct {
	Registries []RegistryResponse `json:"registries"`
	Total      int                `json:"total"`
}

// RegistryInfo represents minimal registry information for users.
type RegistryInfo struct {
	Name               string `json:"name"`
	RegistryURL        string `json:"registryUrl"`
	ScenarioRepository string `json:"scenarioRepository"`
	Description        string `json:"description,omitempty"`
	AITagMetadata
}

// AvailableRegistriesResponse represents the response for listing available registries.
type AvailableRegistriesResponse struct {
	Registries []RegistryInfo `json:"registries"`
}

// CreateRegistryResponse represents the response after creating a registry.
type CreateRegistryResponse struct {
	Message string `json:"message"`
	Name    string `json:"name"`
}

// UpdateRegistryResponse represents the response after updating a registry.
type UpdateRegistryResponse struct {
	Message string `json:"message"`
	Name    string `json:"name"`
}

// DeleteRegistryResponse represents the response after deleting a registry.
type DeleteRegistryResponse struct {
	Message string `json:"message"`
}
