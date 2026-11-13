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

package registry

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

// Label and annotation keys for registry Secrets
const (
	// AppNameLabel is the standard app name label
	AppNameLabel = "app.kubernetes.io/name"
	// AppComponentLabel is the standard component label
	AppComponentLabel = "app.kubernetes.io/component"

	// AuthTypeLabel indicates the authentication type (token or password)
	AuthTypeLabel = "registry.krkn.krkn-chaos.dev/auth-type"
	// AvailableToAllLabel marks registries accessible by all users
	AvailableToAllLabel = "registry.krkn.krkn-chaos.dev/available-to-all"

	// RegistryURLAnnotation stores the registry URL
	RegistryURLAnnotation = "registry.krkn.krkn-chaos.dev/registry-url"
	// ScenarioRepositoryAnnotation stores the scenario repository path
	ScenarioRepositoryAnnotation = "registry.krkn.krkn-chaos.dev/scenario-repository"
	// DescriptionAnnotation stores the registry description
	DescriptionAnnotation = "registry.krkn.krkn-chaos.dev/description"
	// SkipTLSAnnotation indicates if TLS verification should be skipped
	SkipTLSAnnotation = "registry.krkn.krkn-chaos.dev/skip-tls"
	// InsecureAnnotation indicates if insecure connections are allowed
	InsecureAnnotation = "registry.krkn.krkn-chaos.dev/insecure"
	// CreatedByAnnotation stores the email of the admin who created the registry
	CreatedByAnnotation = "registry.krkn.krkn-chaos.dev/created-by"
	// CreatedAtAnnotation stores the creation timestamp
	CreatedAtAnnotation = "registry.krkn.krkn-chaos.dev/created-at"
	// UpdatedByAnnotation stores the email of the admin who last updated the registry
	UpdatedByAnnotation = "registry.krkn.krkn-chaos.dev/updated-by"
	// UpdatedAtAnnotation stores the last update timestamp
	UpdatedAtAnnotation = "registry.krkn.krkn-chaos.dev/updated-at"

	// AIOrchestratorTagAnnotation stores the private Krkn-AI orchestrator tag.
	AIOrchestratorTagAnnotation = "registry.krkn.krkn-chaos.dev/ai-orchestrator-tag"
	// AIResultUploaderTagAnnotation stores the private Krkn-AI result uploader tag.
	AIResultUploaderTagAnnotation = "registry.krkn.krkn-chaos.dev/ai-result-uploader-tag"
	// WorkloadImageTagsAnnotation stores workload role-to-tag overrides as JSON.
	WorkloadImageTagsAnnotation = "registry.krkn.krkn-chaos.dev/workload-image-tags"

	// AppName is the value for AppNameLabel
	AppName = "krkn-operator"
	// ComponentRegistry is the value for AppComponentLabel
	ComponentRegistry = "registry"

	// AuthTypeToken indicates token-based authentication (Bearer token for API calls)
	AuthTypeToken = "token"
	// AuthTypePassword indicates username/password-based authentication
	AuthTypePassword = "password"
)

// BuildRegistryLabels creates the labels map for a registry Secret
func BuildRegistryLabels(authType string, groups []string, availableToAll bool) map[string]string {
	labels := map[string]string{
		AppNameLabel:      AppName,
		AppComponentLabel: ComponentRegistry,
		AuthTypeLabel:     authType,
	}

	// Add available-to-all label if specified
	if availableToAll {
		labels[AvailableToAllLabel] = "true"
	}

	// Add group labels
	for _, groupName := range groups {
		groupLabel := groupauth.GroupLabelKey(groupName)
		labels[groupLabel] = "true"
	}

	return labels
}

// BuildRegistryAnnotations creates registry configuration annotations.
func BuildRegistryAnnotations(
	registryURL string,
	scenarioRepo string,
	description string,
	skipTLS bool,
	insecure bool,
	createdBy string,
	aiTags ...AITagMetadata,
) map[string]string {
	annotations := map[string]string{
		RegistryURLAnnotation:        registryURL,
		ScenarioRepositoryAnnotation: scenarioRepo,
		SkipTLSAnnotation:            fmt.Sprintf("%t", skipTLS),
		InsecureAnnotation:           fmt.Sprintf("%t", insecure),
		CreatedByAnnotation:          createdBy,
		CreatedAtAnnotation:          time.Now().UTC().Format(time.RFC3339),
	}
	if description != "" {
		annotations[DescriptionAnnotation] = description
	}
	if len(aiTags) != 0 {
		storeAITagMetadata(annotations, aiTags[0])
	}
	return annotations
}

// UpdateRegistryAnnotations updates registry configuration annotations.
func UpdateRegistryAnnotations(
	existing map[string]string,
	registryURL string,
	scenarioRepo string,
	description string,
	skipTLS bool,
	insecure bool,
	updatedBy string,
	aiTags ...AITagMetadata,
) map[string]string {
	updated := make(map[string]string, len(existing)+3)
	for k, v := range existing {
		updated[k] = v
	}
	updated[RegistryURLAnnotation] = registryURL
	updated[ScenarioRepositoryAnnotation] = scenarioRepo
	updated[SkipTLSAnnotation] = fmt.Sprintf("%t", skipTLS)
	updated[InsecureAnnotation] = fmt.Sprintf("%t", insecure)
	updated[UpdatedByAnnotation] = updatedBy
	updated[UpdatedAtAnnotation] = time.Now().UTC().Format(time.RFC3339)
	if description != "" {
		updated[DescriptionAnnotation] = description
	} else {
		delete(updated, DescriptionAnnotation)
	}
	if len(aiTags) != 0 {
		storeAITagMetadata(updated, aiTags[0])
	}
	return updated
}

func storeAITagMetadata(annotations map[string]string, metadata AITagMetadata) {
	setAnnotation(annotations, AIOrchestratorTagAnnotation, metadata.AIOrchestratorTag)
	setAnnotation(annotations, AIResultUploaderTagAnnotation, metadata.AIResultUploaderTag)
	if len(metadata.WorkloadImageTags) == 0 {
		delete(annotations, WorkloadImageTagsAnnotation)
		return
	}
	encoded, err := json.Marshal(metadata.WorkloadImageTags)
	if err == nil {
		annotations[WorkloadImageTagsAnnotation] = string(encoded)
	}
}

func setAnnotation(annotations map[string]string, key, value string) {
	if value == "" {
		delete(annotations, key)
		return
	}
	annotations[key] = value
}

// ExtractGroupsFromLabels extracts group names from registry Secret labels
func ExtractGroupsFromLabels(labels map[string]string) []string {
	groups := []string{}

	for key, value := range labels {
		// Check if it's a group label with value "true"
		if strings.HasPrefix(key, groupauth.GroupLabelPrefix) && value == "true" {
			// Extract group name from label key
			groupName := strings.TrimPrefix(key, groupauth.GroupLabelPrefix)
			groups = append(groups, groupName)
		}
	}

	return groups
}
