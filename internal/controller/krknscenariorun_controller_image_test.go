/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package controller

import (
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	krknctlconfig "github.com/krkn-chaos/krknctl/pkg/config"
	krknctlmodels "github.com/krkn-chaos/krknctl/pkg/provider/models"
)

func getTestConfig(t *testing.T) *krknctlconfig.Config {
	t.Helper()
	config, err := krknctlconfig.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load krknctl config: %v", err)
	}
	return &config
}

func publicReference(name string) krknv1alpha1.ScenarioReference {
	private := false
	return krknv1alpha1.ScenarioReference{Name: name, Private: &private}
}

func privateReference(name, registryName string) krknv1alpha1.ScenarioReference {
	private := true
	return krknv1alpha1.ScenarioReference{Name: name, Private: &private, RegistryName: registryName}
}

func TestBuildContainerImage_PublicRegistryUsesKrknctl(t *testing.T) {
	image, err := buildContainerImage(&krknv1alpha1.KrknScenarioRunSpec{
		Scenario: publicReference("dummy-scenario"),
	}, getTestConfig(t))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if want := "quay.io/krkn-chaos/krkn-hub-multiarch:dummy-scenario"; image != want {
		t.Fatalf("expected image %q, got %q", want, image)
	}
}

func TestBuildContainerImage_RejectsPrivateWithoutResolvedRegistry(t *testing.T) {
	_, err := buildContainerImage(&krknv1alpha1.KrknScenarioRunSpec{
		Scenario: privateReference("dummy-scenario", "private"),
	}, getTestConfig(t))
	if err == nil {
		t.Fatal("expected private registry configuration error")
	}
}

func TestBuildContainerImageFromReference_PrivateRegistry(t *testing.T) {
	privateRegistry := &krknctlmodels.RegistryV2{
		RegistryURL:        "registry.example.com",
		ScenarioRepository: "chaos/scenarios",
	}
	image, err := buildContainerImageFromReference(
		privateReference("dummy-scenario", "private"),
		getTestConfig(t),
		privateRegistry,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if want := "registry.example.com/chaos/scenarios:dummy-scenario"; image != want {
		t.Fatalf("expected image %q, got %q", want, image)
	}
}

func TestBuildContainerImageFromReferenceRejectsInvalidPublicRegistry(t *testing.T) {
	private := false
	_, err := buildContainerImageFromReference(
		krknv1alpha1.ScenarioReference{Name: "dummy-scenario", Private: &private, RegistryName: "unexpected"},
		getTestConfig(t),
		nil,
	)
	if err == nil {
		t.Fatal("expected invalid public/private registry combination error")
	}
}
