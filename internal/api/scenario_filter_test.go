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
	"context"
	"fmt"
	"testing"

	"github.com/krkn-chaos/krknctl/pkg/provider"
	"github.com/krkn-chaos/krknctl/pkg/provider/models"
	"github.com/krkn-chaos/krknctl/pkg/typing"
)

type mockScenarioProvider struct {
	details map[string]*models.ScenarioDetail
	err     map[string]error
}

func (m *mockScenarioProvider) GetRegistryImages(_ *models.RegistryV2) (*[]models.ScenarioTag, error) {
	return nil, nil
}

func (m *mockScenarioProvider) GetGlobalEnvironment(_ *models.RegistryV2, _ string) (*models.ScenarioDetail, error) {
	return nil, nil
}

func (m *mockScenarioProvider) GetScenarioDetail(scenario string, _ *models.RegistryV2) (*models.ScenarioDetail, error) {
	if e, ok := m.err[scenario]; ok {
		return nil, e
	}
	detail, ok := m.details[scenario]
	if !ok {
		return nil, nil
	}
	return detail, nil
}

func (m *mockScenarioProvider) ScaffoldScenarios(_ []string, _ bool, _ *models.RegistryV2, _ bool, _ *provider.ScaffoldSeed) (*string, error) {
	return nil, nil
}

func TestFilterScenariosByIsAScenario_OnlyReturnsScenarios(t *testing.T) {
	mock := &mockScenarioProvider{
		details: map[string]*models.ScenarioDetail{
			"cpu-hog": {
				ScenarioTag: models.ScenarioTag{Name: "cpu-hog"},
				IsAScenario: true,
				Title:       "CPU Hog",
			},
			"memory-hog": {
				ScenarioTag: models.ScenarioTag{Name: "memory-hog"},
				IsAScenario: true,
				Title:       "Memory Hog",
			},
			"base-image": {
				ScenarioTag: models.ScenarioTag{Name: "base-image"},
				IsAScenario: false,
				Title:       "Base Image",
			},
		},
	}
	tags := &[]models.ScenarioTag{
		{Name: "cpu-hog"},
		{Name: "memory-hog"},
		{Name: "base-image"},
	}

	result := filterScenariosByIsAScenario(context.Background(), mock, tags, nil)

	if len(result) != 2 {
		t.Fatalf("expected 2 scenarios, got %d", len(result))
	}
	names := map[string]bool{}
	for _, s := range result {
		names[s.Name] = true
	}
	if !names["cpu-hog"] {
		t.Error("expected cpu-hog in filtered results")
	}
	if !names["memory-hog"] {
		t.Error("expected memory-hog in filtered results")
	}
	if names["base-image"] {
		t.Error("base-image should have been filtered out")
	}
}

func TestFilterScenariosByIsAScenario_NilTags(t *testing.T) {
	mock := &mockScenarioProvider{details: map[string]*models.ScenarioDetail{}}
	result := filterScenariosByIsAScenario(context.Background(), mock, nil, nil)

	if len(result) != 0 {
		t.Fatalf("expected 0 scenarios for nil tags, got %d", len(result))
	}
}

func TestFilterScenariosByIsAScenario_EmptyTags(t *testing.T) {
	mock := &mockScenarioProvider{details: map[string]*models.ScenarioDetail{}}
	tags := &[]models.ScenarioTag{}
	result := filterScenariosByIsAScenario(context.Background(), mock, tags, nil)

	if len(result) != 0 {
		t.Fatalf("expected 0 scenarios for empty tags, got %d", len(result))
	}
}

func TestFilterScenariosByIsAScenario_AllFilteredOut(t *testing.T) {
	mock := &mockScenarioProvider{
		details: map[string]*models.ScenarioDetail{
			"base-image": {
				ScenarioTag: models.ScenarioTag{Name: "base-image"},
				IsAScenario: false,
			},
			"tooling": {
				ScenarioTag: models.ScenarioTag{Name: "tooling"},
				IsAScenario: false,
			},
		},
	}
	tags := &[]models.ScenarioTag{
		{Name: "base-image"},
		{Name: "tooling"},
	}

	result := filterScenariosByIsAScenario(context.Background(), mock, tags, nil)

	if len(result) != 0 {
		t.Fatalf("expected 0 scenarios when all are filtered out, got %d", len(result))
	}
}

func TestFilterScenariosByIsAScenario_DetailErrorSkipsScenario(t *testing.T) {
	mock := &mockScenarioProvider{
		details: map[string]*models.ScenarioDetail{
			"cpu-hog": {
				ScenarioTag: models.ScenarioTag{Name: "cpu-hog"},
				IsAScenario: true,
			},
		},
		err: map[string]error{
			"broken-tag": fmt.Errorf("manifest not found"),
		},
	}
	tags := &[]models.ScenarioTag{
		{Name: "cpu-hog"},
		{Name: "broken-tag"},
	}

	result := filterScenariosByIsAScenario(context.Background(), mock, tags, nil)

	if len(result) != 1 {
		t.Fatalf("expected 1 scenario (broken-tag skipped), got %d", len(result))
	}
	if result[0].Name != "cpu-hog" {
		t.Errorf("expected cpu-hog, got %s", result[0].Name)
	}
}

func TestFilterScenariosByIsAScenario_DetailReturnsNilSkipsScenario(t *testing.T) {
	mock := &mockScenarioProvider{
		details: map[string]*models.ScenarioDetail{
			"cpu-hog": {
				ScenarioTag: models.ScenarioTag{Name: "cpu-hog"},
				IsAScenario: true,
			},
		},
	}
	tags := &[]models.ScenarioTag{
		{Name: "cpu-hog"},
		{Name: "unknown-tag"},
	}

	result := filterScenariosByIsAScenario(context.Background(), mock, tags, nil)

	if len(result) != 1 {
		t.Fatalf("expected 1 scenario (unknown-tag skipped), got %d", len(result))
	}
}

func TestFilterScenariosByIsAScenario_PreservesTagMetadata(t *testing.T) {
	digest := "sha256:abc123"
	size := int64(1024)

	mock := &mockScenarioProvider{
		details: map[string]*models.ScenarioDetail{
			"cpu-hog": {
				ScenarioTag: models.ScenarioTag{
					Name:   "cpu-hog",
					Digest: &digest,
					Size:   &size,
				},
				IsAScenario: true,
				Fields:      []typing.InputField{},
			},
		},
	}
	tags := &[]models.ScenarioTag{
		{Name: "cpu-hog", Digest: &digest, Size: &size},
	}

	result := filterScenariosByIsAScenario(context.Background(), mock, tags, nil)

	if len(result) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(result))
	}
	if result[0].Name != "cpu-hog" {
		t.Errorf("expected name cpu-hog, got %s", result[0].Name)
	}
	if result[0].Digest == nil || *result[0].Digest != digest {
		t.Error("expected digest to be preserved")
	}
	if result[0].Size == nil || *result[0].Size != size {
		t.Error("expected size to be preserved")
	}
}
