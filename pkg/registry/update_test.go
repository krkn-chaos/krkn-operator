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
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestUpdateWithoutCredentials verifies that updating a registry without providing credentials
// keeps the existing credentials intact
func TestUpdateWithoutCredentials(t *testing.T) {
	// Create initial secret with OAuth token
	authStr := base64.StdEncoding.EncodeToString([]byte("$oauthtoken:original-token"))
	dockerConfig := DockerConfig{
		Auths: map[string]AuthEntry{
			"registry.example.com": {
				Auth: authStr,
			},
		},
	}
	dockerConfigJSON, _ := json.Marshal(dockerConfig)

	originalSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-registry",
			Namespace: "default",
			Labels: map[string]string{
				AppNameLabel:      AppName,
				AppComponentLabel: ComponentRegistry,
			},
			Annotations: map[string]string{
				RegistryURLAnnotation:        "registry.example.com",
				ScenarioRepositoryAnnotation: "krkn-chaos/krkn-hub",
				SkipTLSAnnotation:            "false",
				InsecureAnnotation:           "false",
				CreatedByAnnotation:          "admin",
				CreatedAtAnnotation:          "2025-01-01T00:00:00Z",
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfigJSON,
		},
	}

	// Extract existing registry config
	existingRegistry, err := ExtractRegistryV2FromSecret(originalSecret)
	if err != nil {
		t.Fatalf("Failed to extract existing registry: %v", err)
	}

	// Simulate update without providing new credentials - use existing
	newDockerConfigJSON, err := BuildDockerConfigJSON(
		"registry.example.com",
		*existingRegistry.Username,
		*existingRegistry.Password,
	)
	if err != nil {
		t.Fatalf("Failed to build new dockerconfigjson: %v", err)
	}

	// Verify the new config contains the original credentials
	var newDockerConfig DockerConfig
	if err := json.Unmarshal(newDockerConfigJSON, &newDockerConfig); err != nil {
		t.Fatalf("Failed to unmarshal new config: %v", err)
	}

	newAuthEntry, exists := newDockerConfig.Auths["registry.example.com"]
	if !exists {
		t.Fatal("Auth entry not found in new config")
	}

	decodedAuth, err := base64.StdEncoding.DecodeString(newAuthEntry.Auth)
	if err != nil {
		t.Fatalf("Failed to decode auth: %v", err)
	}

	if string(decodedAuth) != "$oauthtoken:original-token" {
		t.Errorf("Credentials changed! Got %v, want $oauthtoken:original-token", string(decodedAuth))
	}
}

// TestUpdateWithNewCredentials verifies that updating a registry with new credentials
// replaces the existing ones
func TestUpdateWithNewCredentials(t *testing.T) {
	// Create initial secret
	authStr := base64.StdEncoding.EncodeToString([]byte("$oauthtoken:original-token"))
	dockerConfig := DockerConfig{
		Auths: map[string]AuthEntry{
			"registry.example.com": {
				Auth: authStr,
			},
		},
	}
	dockerConfigJSON, _ := json.Marshal(dockerConfig)

	originalSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-registry",
			Namespace: "default",
			Labels: map[string]string{
				AppNameLabel:      AppName,
				AppComponentLabel: ComponentRegistry,
			},
			Annotations: map[string]string{
				RegistryURLAnnotation:        "registry.example.com",
				ScenarioRepositoryAnnotation: "krkn-chaos/krkn-hub",
				SkipTLSAnnotation:            "false",
				InsecureAnnotation:           "false",
				CreatedByAnnotation:          "admin",
				CreatedAtAnnotation:          "2025-01-01T00:00:00Z",
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfigJSON,
		},
	}

	// Extract existing registry config (simulating what the handler does)
	_, err := ExtractRegistryV2FromSecret(originalSecret)
	if err != nil {
		t.Fatalf("Failed to extract existing registry: %v", err)
	}

	// Simulate update WITH new credentials
	newDockerConfigJSON, err := BuildDockerConfigJSON(
		"registry.example.com",
		"$oauthtoken",
		"new-token",
	)
	if err != nil {
		t.Fatalf("Failed to build new dockerconfigjson: %v", err)
	}

	// Verify the new config contains the new credentials
	var newDockerConfig DockerConfig
	if err := json.Unmarshal(newDockerConfigJSON, &newDockerConfig); err != nil {
		t.Fatalf("Failed to unmarshal new config: %v", err)
	}

	newAuthEntry, exists := newDockerConfig.Auths["registry.example.com"]
	if !exists {
		t.Fatal("Auth entry not found in new config")
	}

	decodedAuth, err := base64.StdEncoding.DecodeString(newAuthEntry.Auth)
	if err != nil {
		t.Fatalf("Failed to decode auth: %v", err)
	}

	if string(decodedAuth) != "$oauthtoken:new-token" {
		t.Errorf("Credentials not updated! Got %v, want $oauthtoken:new-token", string(decodedAuth))
	}
}

// TestUpdatePasswordAuthWithoutCredentials tests updating password-based registry without credentials
func TestUpdatePasswordAuthWithoutCredentials(t *testing.T) {
	// Create initial secret with password auth
	authStr := base64.StdEncoding.EncodeToString([]byte("original-user:original-pass"))
	dockerConfig := DockerConfig{
		Auths: map[string]AuthEntry{
			"registry.io": {
				Auth: authStr,
			},
		},
	}
	dockerConfigJSON, _ := json.Marshal(dockerConfig)

	originalSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-registry",
			Namespace: "default",
			Labels: map[string]string{
				AppNameLabel:      AppName,
				AppComponentLabel: ComponentRegistry,
			},
			Annotations: map[string]string{
				RegistryURLAnnotation:        "registry.io",
				ScenarioRepositoryAnnotation: "org/repo",
				SkipTLSAnnotation:            "false",
				InsecureAnnotation:           "false",
				CreatedByAnnotation:          "admin",
				CreatedAtAnnotation:          "2025-01-01T00:00:00Z",
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfigJSON,
		},
	}

	// Extract existing registry config
	existingRegistry, err := ExtractRegistryV2FromSecret(originalSecret)
	if err != nil {
		t.Fatalf("Failed to extract existing registry: %v", err)
	}

	// Verify existing credentials were extracted correctly
	if existingRegistry.Username == nil || *existingRegistry.Username != "original-user" {
		t.Errorf("Existing username = %v, want original-user", existingRegistry.Username)
	}
	if existingRegistry.Password == nil || *existingRegistry.Password != "original-pass" {
		t.Errorf("Existing password = %v, want original-pass", existingRegistry.Password)
	}

	// Simulate update without providing new credentials - use existing ones
	newDockerConfigJSON, err := BuildDockerConfigJSON(
		"registry.io",
		*existingRegistry.Username,
		*existingRegistry.Password,
	)
	if err != nil {
		t.Fatalf("Failed to build new dockerconfigjson: %v", err)
	}

	// Verify the new config contains the original credentials
	var newDockerConfig DockerConfig
	if err := json.Unmarshal(newDockerConfigJSON, &newDockerConfig); err != nil {
		t.Fatalf("Failed to unmarshal new config: %v", err)
	}

	newAuthEntry, exists := newDockerConfig.Auths["registry.io"]
	if !exists {
		t.Fatal("Auth entry not found in new config")
	}

	decodedAuth, err := base64.StdEncoding.DecodeString(newAuthEntry.Auth)
	if err != nil {
		t.Fatalf("Failed to decode auth: %v", err)
	}

	if string(decodedAuth) != "original-user:original-pass" {
		t.Errorf("Credentials changed! Got %v, want original-user:original-pass", string(decodedAuth))
	}
}
