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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/krkn-chaos/krknctl/pkg/provider/models"
	corev1 "k8s.io/api/core/v1"
)

// ExtractRegistryV2FromSecret extracts RegistryV2 configuration from a Kubernetes Secret
func ExtractRegistryV2FromSecret(secret *corev1.Secret) (*models.RegistryV2, error) {
	// Validate secret type
	if secret.Type != corev1.SecretTypeDockerConfigJson {
		return nil, fmt.Errorf("invalid secret type: expected %s, got %s",
			corev1.SecretTypeDockerConfigJson, secret.Type)
	}

	// Validate it's a registry secret
	if secret.Labels[AppComponentLabel] != ComponentRegistry {
		return nil, fmt.Errorf("secret is not a registry secret (missing component label)")
	}

	// Extract registry URL from annotations
	registryURL, ok := secret.Annotations[RegistryURLAnnotation]
	if !ok || registryURL == "" {
		return nil, fmt.Errorf("missing required annotation: %s", RegistryURLAnnotation)
	}

	// Extract scenario repository from annotations
	scenarioRepo, ok := secret.Annotations[ScenarioRepositoryAnnotation]
	if !ok || scenarioRepo == "" {
		return nil, fmt.Errorf("missing required annotation: %s", ScenarioRepositoryAnnotation)
	}

	// Extract and decode .dockerconfigjson
	dockerConfigJSON, ok := secret.Data[corev1.DockerConfigJsonKey]
	if !ok {
		return nil, fmt.Errorf("missing .dockerconfigjson data field")
	}

	var dockerConfig DockerConfig
	if err := json.Unmarshal(dockerConfigJSON, &dockerConfig); err != nil {
		return nil, fmt.Errorf("failed to parse .dockerconfigjson: %w", err)
	}

	// Get auth entry for the registry URL
	authEntry, ok := dockerConfig.Auths[registryURL]
	if !ok {
		return nil, fmt.Errorf("no auth entry found for registry %s", registryURL)
	}

	// Decode the auth field (base64)
	// Standard format is always base64(username:password)
	decodedAuth, err := base64.StdEncoding.DecodeString(authEntry.Auth)
	if err != nil {
		return nil, fmt.Errorf("failed to decode auth field: %w", err)
	}

	// Split username:password (standard format)
	parts := strings.SplitN(string(decodedAuth), ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid auth format: expected base64(username:password)")
	}

	username := parts[0]
	password := parts[1]

	registry := &models.RegistryV2{
		RegistryURL:        registryURL,
		ScenarioRepository: scenarioRepo,
		Username:           &username,
		Password:           &password,
		SkipTLS:            secret.Annotations[SkipTLSAnnotation] == "true",
		Insecure:           secret.Annotations[InsecureAnnotation] == "true",
	}

	// Check auth type label to determine if password should also populate Token field
	authType := secret.Labels[AuthTypeLabel]
	if authType == AuthTypeToken {
		// For token auth, password field contains the actual token for API Bearer auth
		registry.Token = &password
	}

	return registry, nil
}

// BuildDockerConfigJSON creates a .dockerconfigjson data field with username:password auth
func BuildDockerConfigJSON(registryURL, username, password string) ([]byte, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password are required")
	}

	// Standard format: base64(username:password)
	authStr := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

	dockerConfig := DockerConfig{
		Auths: map[string]AuthEntry{
			registryURL: {
				Auth: authStr,
			},
		},
	}

	return json.Marshal(dockerConfig)
}

// ValidateRegistrySecret validates that a Secret is a valid registry Secret
func ValidateRegistrySecret(secret *corev1.Secret) error {
	// Check secret type
	if secret.Type != corev1.SecretTypeDockerConfigJson {
		return fmt.Errorf("invalid secret type: expected %s", corev1.SecretTypeDockerConfigJson)
	}

	// Check required labels
	requiredLabels := map[string]string{
		AppNameLabel:      AppName,
		AppComponentLabel: ComponentRegistry,
	}

	for key, expectedValue := range requiredLabels {
		if value, ok := secret.Labels[key]; !ok || value != expectedValue {
			return fmt.Errorf("missing or invalid label: %s (expected %s)", key, expectedValue)
		}
	}

	// Check required annotations
	requiredAnnotations := []string{
		RegistryURLAnnotation,
		ScenarioRepositoryAnnotation,
		SkipTLSAnnotation,
		InsecureAnnotation,
		CreatedByAnnotation,
		CreatedAtAnnotation,
	}

	for _, key := range requiredAnnotations {
		if _, ok := secret.Annotations[key]; !ok {
			return fmt.Errorf("missing required annotation: %s", key)
		}
	}

	// Check .dockerconfigjson data field
	dockerConfigJSON, ok := secret.Data[corev1.DockerConfigJsonKey]
	if !ok {
		return fmt.Errorf("missing .dockerconfigjson data field")
	}

	// Validate it's valid JSON
	var dockerConfig DockerConfig
	if err := json.Unmarshal(dockerConfigJSON, &dockerConfig); err != nil {
		return fmt.Errorf("invalid .dockerconfigjson: %w", err)
	}

	// Check auth entry exists for registry URL
	registryURL := secret.Annotations[RegistryURLAnnotation]
	authEntry, ok := dockerConfig.Auths[registryURL]
	if !ok {
		return fmt.Errorf("no auth entry found for registry %s", registryURL)
	}

	// Validate auth can be decoded and is in username:password format
	decodedAuth, err := base64.StdEncoding.DecodeString(authEntry.Auth)
	if err != nil {
		return fmt.Errorf("invalid base64 encoding in auth field: %w", err)
	}

	// Validate standard format username:password
	if !strings.Contains(string(decodedAuth), ":") {
		return fmt.Errorf("auth must be in format base64(username:password)")
	}

	return nil
}
