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

Assisted-by: Claude Sonnet 4.5 (claude-sonnet-4-5@20250929)
*/

package controller

import (
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	krknctlconfig "github.com/krkn-chaos/krknctl/pkg/config"
)

func getTestConfig(t *testing.T) *krknctlconfig.Config {
	t.Helper()
	config, err := krknctlconfig.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load krknctl config: %v", err)
	}
	return &config
}

func TestBuildContainerImage_PublicRegistry(t *testing.T) {
	config := getTestConfig(t)

	// Test default public Quay registry (clean tag)
	spec := &krknv1alpha1.KrknScenarioRunSpec{
		ScenarioImage: "node-cpu-hog",
	}

	image, err := buildContainerImage(spec, config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := "quay.io/krkn-chaos/krkn-hub:node-cpu-hog"
	if image != expected {
		t.Errorf("Expected image '%s', got '%s'", expected, image)
	}
}

func TestBuildContainerImage_PublicRegistryWithPrefix(t *testing.T) {
	config := getTestConfig(t)

	// Test legacy frontend format with krkn-hub: prefix (should be stripped)
	spec := &krknv1alpha1.KrknScenarioRunSpec{
		ScenarioImage: "krkn-hub:node-cpu-hog",
	}

	image, err := buildContainerImage(spec, config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should strip the prefix and construct correct path
	expected := "quay.io/krkn-chaos/krkn-hub:node-cpu-hog"
	if image != expected {
		t.Errorf("Expected image '%s', got '%s'", expected, image)
	}
}

func TestBuildContainerImage_PrivateRegistryWithName(t *testing.T) {
	config := getTestConfig(t)

	// Test private registry with RegistryName (saved registry)
	spec := &krknv1alpha1.KrknScenarioRunSpec{
		RegistryName:       "my-private-registry",
		RegistryURL:        "quay.io",
		ScenarioRepository: "rh_ee_tsebasti/krkn-hub-private",
		ScenarioImage:      "node-memory-hog",
	}

	image, err := buildContainerImage(spec, config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := "quay.io/rh_ee_tsebasti/krkn-hub-private:node-memory-hog"
	if image != expected {
		t.Errorf("Expected image '%s', got '%s'", expected, image)
	}
}

func TestBuildContainerImage_PrivateRegistryInline(t *testing.T) {
	config := getTestConfig(t)

	// Test private registry with inline credentials (no RegistryName)
	spec := &krknv1alpha1.KrknScenarioRunSpec{
		RegistryURL:        "registry.example.com",
		ScenarioRepository: "myorg/scenarios",
		ScenarioImage:      "custom-scenario",
		Username:           "user",
		Password:           "pass",
	}

	image, err := buildContainerImage(spec, config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := "registry.example.com/myorg/scenarios:custom-scenario"
	if image != expected {
		t.Errorf("Expected image '%s', got '%s'", expected, image)
	}
}

func TestBuildContainerImage_PrivateRegistryWithToken(t *testing.T) {
	config := getTestConfig(t)

	// Test private registry with token
	spec := &krknv1alpha1.KrknScenarioRunSpec{
		RegistryURL:        "quay.io",
		ScenarioRepository: "private-org/private-repo",
		ScenarioImage:      "secret-scenario",
		Token:              "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
	}

	image, err := buildContainerImage(spec, config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := "quay.io/private-org/private-repo:secret-scenario"
	if image != expected {
		t.Errorf("Expected image '%s', got '%s'", expected, image)
	}
}

func TestBuildContainerImage_DifferentPublicScenarios(t *testing.T) {
	config := getTestConfig(t)

	scenarios := []struct {
		name     string
		scenario string
		expected string
	}{
		{
			name:     "node-cpu-hog",
			scenario: "node-cpu-hog",
			expected: "quay.io/krkn-chaos/krkn-hub:node-cpu-hog",
		},
		{
			name:     "pod-scenarios",
			scenario: "pod-scenarios",
			expected: "quay.io/krkn-chaos/krkn-hub:pod-scenarios",
		},
		{
			name:     "zone-outage",
			scenario: "zone-outage",
			expected: "quay.io/krkn-chaos/krkn-hub:zone-outage",
		},
		{
			name:     "node-cpu-hog-with-prefix",
			scenario: "krkn-hub:node-cpu-hog",
			expected: "quay.io/krkn-chaos/krkn-hub:node-cpu-hog",
		},
		{
			name:     "pod-scenarios-with-prefix",
			scenario: "krkn-hub:pod-scenarios",
			expected: "quay.io/krkn-chaos/krkn-hub:pod-scenarios",
		},
	}

	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			spec := &krknv1alpha1.KrknScenarioRunSpec{
				ScenarioImage: tc.scenario,
			}

			image, err := buildContainerImage(spec, config)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if image != tc.expected {
				t.Errorf("Expected image '%s', got '%s'", tc.expected, image)
			}
		})
	}
}

func TestBuildContainerImage_PartialRegistryInfo(t *testing.T) {
	config := getTestConfig(t)

	// Test with only RegistryURL but no ScenarioRepository - should use public registry
	spec := &krknv1alpha1.KrknScenarioRunSpec{
		RegistryURL:   "quay.io",
		ScenarioImage: "node-cpu-hog",
	}

	image, err := buildContainerImage(spec, config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should fall back to public registry
	expected := "quay.io/krkn-chaos/krkn-hub:node-cpu-hog"
	if image != expected {
		t.Errorf("Expected image '%s', got '%s'", expected, image)
	}
}

func TestBuildContainerImage_OnlyScenarioRepository(t *testing.T) {
	config := getTestConfig(t)

	// Test with only ScenarioRepository but no RegistryURL - should use public registry
	spec := &krknv1alpha1.KrknScenarioRunSpec{
		ScenarioRepository: "rh_ee_tsebasti/krkn-hub-private",
		ScenarioImage:      "node-cpu-hog",
	}

	image, err := buildContainerImage(spec, config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should fall back to public registry
	expected := "quay.io/krkn-chaos/krkn-hub:node-cpu-hog"
	if image != expected {
		t.Errorf("Expected image '%s', got '%s'", expected, image)
	}
}
