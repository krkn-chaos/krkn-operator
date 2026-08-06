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

func TestFilterScenariosByIsAScenario(t *testing.T) {
	digest := "sha256:abc123"
	size := int64(1024)

	tests := []struct {
		name          string
		mock          *mockScenarioProvider
		tags          *[]models.ScenarioTag
		expectedNames []string
	}{
		{
			name: "only returns scenarios with IsAScenario true",
			mock: &mockScenarioProvider{
				details: map[string]*models.ScenarioDetail{
					"cpu-hog":    {ScenarioTag: models.ScenarioTag{Name: "cpu-hog"}, IsAScenario: true},
					"memory-hog": {ScenarioTag: models.ScenarioTag{Name: "memory-hog"}, IsAScenario: true},
					"base-image": {ScenarioTag: models.ScenarioTag{Name: "base-image"}, IsAScenario: false},
				},
			},
			tags: &[]models.ScenarioTag{
				{Name: "cpu-hog"},
				{Name: "memory-hog"},
				{Name: "base-image"},
			},
			expectedNames: []string{"cpu-hog", "memory-hog"},
		},
		{
			name:          "nil tags returns empty slice",
			mock:          &mockScenarioProvider{details: map[string]*models.ScenarioDetail{}},
			tags:          nil,
			expectedNames: []string{},
		},
		{
			name:          "empty tags returns empty slice",
			mock:          &mockScenarioProvider{details: map[string]*models.ScenarioDetail{}},
			tags:          &[]models.ScenarioTag{},
			expectedNames: []string{},
		},
		{
			name: "all filtered out returns empty slice",
			mock: &mockScenarioProvider{
				details: map[string]*models.ScenarioDetail{
					"base-image": {ScenarioTag: models.ScenarioTag{Name: "base-image"}, IsAScenario: false},
					"tooling":    {ScenarioTag: models.ScenarioTag{Name: "tooling"}, IsAScenario: false},
				},
			},
			tags: &[]models.ScenarioTag{
				{Name: "base-image"},
				{Name: "tooling"},
			},
			expectedNames: []string{},
		},
		{
			name: "detail error skips scenario",
			mock: &mockScenarioProvider{
				details: map[string]*models.ScenarioDetail{
					"cpu-hog": {ScenarioTag: models.ScenarioTag{Name: "cpu-hog"}, IsAScenario: true},
				},
				err: map[string]error{
					"broken-tag": fmt.Errorf("manifest not found"),
				},
			},
			tags: &[]models.ScenarioTag{
				{Name: "cpu-hog"},
				{Name: "broken-tag"},
			},
			expectedNames: []string{"cpu-hog"},
		},
		{
			name: "detail returns nil skips scenario",
			mock: &mockScenarioProvider{
				details: map[string]*models.ScenarioDetail{
					"cpu-hog": {ScenarioTag: models.ScenarioTag{Name: "cpu-hog"}, IsAScenario: true},
				},
			},
			tags: &[]models.ScenarioTag{
				{Name: "cpu-hog"},
				{Name: "unknown-tag"},
			},
			expectedNames: []string{"cpu-hog"},
		},
		{
			name: "preserves tag metadata and input order",
			mock: &mockScenarioProvider{
				details: map[string]*models.ScenarioDetail{
					"cpu-hog":    {ScenarioTag: models.ScenarioTag{Name: "cpu-hog"}, IsAScenario: true, Fields: []typing.InputField{}},
					"memory-hog": {ScenarioTag: models.ScenarioTag{Name: "memory-hog"}, IsAScenario: true},
				},
			},
			tags: &[]models.ScenarioTag{
				{Name: "cpu-hog", Digest: &digest, Size: &size},
				{Name: "memory-hog"},
			},
			expectedNames: []string{"cpu-hog", "memory-hog"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterScenariosByIsAScenario(context.Background(), tt.mock, tt.tags, nil)

			if len(result) != len(tt.expectedNames) {
				t.Fatalf("expected %d scenarios, got %d", len(tt.expectedNames), len(result))
			}

			for i, expectedName := range tt.expectedNames {
				if result[i].Name != expectedName {
					t.Errorf("result[%d]: expected name %q, got %q", i, expectedName, result[i].Name)
				}
			}
		})
	}
}

func TestFilterScenariosByIsAScenario_PreservesTagMetadata(t *testing.T) {
	digest := "sha256:abc123"
	size := int64(1024)

	mock := &mockScenarioProvider{
		details: map[string]*models.ScenarioDetail{
			"cpu-hog": {
				ScenarioTag: models.ScenarioTag{Name: "cpu-hog", Digest: &digest, Size: &size},
				IsAScenario: true,
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
	if result[0].Digest == nil || *result[0].Digest != digest {
		t.Error("expected digest to be preserved")
	}
	if result[0].Size == nil || *result[0].Size != size {
		t.Error("expected size to be preserved")
	}
}

func TestFilterScenariosByIsAScenario_CancelledContext(t *testing.T) {
	mock := &mockScenarioProvider{
		details: map[string]*models.ScenarioDetail{
			"cpu-hog": {ScenarioTag: models.ScenarioTag{Name: "cpu-hog"}, IsAScenario: true},
		},
	}
	tags := &[]models.ScenarioTag{
		{Name: "cpu-hog"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := filterScenariosByIsAScenario(ctx, mock, tags, nil)

	if len(result) > 1 {
		t.Errorf("expected at most 1 result with cancelled context, got %d", len(result))
	}
}
