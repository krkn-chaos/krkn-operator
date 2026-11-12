/*
Copyright 2026.

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
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/krkn-chaos/krknctl/pkg/provider"
	"github.com/krkn-chaos/krknctl/pkg/provider/models"
	"github.com/krkn-chaos/krknctl/pkg/verify"
)

func TestPostScenariosCachesPublicCatalog(t *testing.T) {
	handler := setupFilesTestHandler()
	digest := "sha256:cpu-hog"
	mock := &mockScenarioProvider{
		tags:      []models.ScenarioTag{{Name: "cpu-hog", Digest: &digest}},
		signature: verify.SignatureSigned,
		details: map[string]*models.ScenarioDetail{
			"cpu-hog": {ScenarioTag: models.ScenarioTag{Name: "cpu-hog"}, IsAScenario: true},
		},
	}
	var factoryCalls atomic.Int32
	handler.scenarioProviderFactory = func(provider.Mode) (provider.ScenarioDataProvider, error) {
		factoryCalls.Add(1)
		return mock, nil
	}

	first := postScenariosForCacheTest(t, handler)
	second := postScenariosForCacheTest(t, handler)

	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("expected both requests to succeed, got %d and %d", first.Code, second.Code)
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("cached response differs from initial response: first %s, second %s", first.Body.String(), second.Body.String())
	}
	if got := factoryCalls.Load(); got != 1 {
		t.Errorf("expected one provider creation, got %d", got)
	}
	if got := mock.registryImageCalls.Load(); got != 1 {
		t.Errorf("expected one registry listing, got %d", got)
	}
	if got := mock.signatureCalls.Load(); got != 1 {
		t.Errorf("expected one signature verification, got %d", got)
	}
	if got := mock.detailCalls.Load(); got != 1 {
		t.Errorf("expected one detail lookup, got %d", got)
	}
}

func TestPostScenariosCachesCatalogWhenTagHasNoScenarioLabels(t *testing.T) {
	handler := setupFilesTestHandler()
	mock := &mockScenarioProvider{
		tags: []models.ScenarioTag{
			{Name: "cpu-hog"},
			{Name: "base-image"},
		},
		signature: verify.SignatureSigned,
		details: map[string]*models.ScenarioDetail{
			"cpu-hog": {ScenarioTag: models.ScenarioTag{Name: "cpu-hog"}, IsAScenario: true},
		},
		err: map[string]error{
			"base-image": provider.ErrLabelNotFound,
		},
	}
	var factoryCalls atomic.Int32
	handler.scenarioProviderFactory = func(provider.Mode) (provider.ScenarioDataProvider, error) {
		factoryCalls.Add(1)
		return mock, nil
	}

	first := postScenariosForCacheTest(t, handler)
	second := postScenariosForCacheTest(t, handler)

	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("expected both requests to succeed, got %d and %d", first.Code, second.Code)
	}
	if got := factoryCalls.Load(); got != 1 {
		t.Errorf("expected the catalog with expected label exclusions to be cached, provider created %d times", got)
	}
	if got := mock.detailCalls.Load(); got != 2 {
		t.Errorf("expected both tags to be classified once, got %d detail lookups", got)
	}
}

func TestPublicSignatureCacheReusesStatusByDigest(t *testing.T) {
	digest := "sha256:cpu-hog"
	mock := &mockScenarioProvider{signature: verify.SignatureSigned}
	cachedProvider := &publicSignatureCachingProvider{
		ScenarioDataProvider: mock,
		cache:                &publicSignatureCache{},
	}
	tag := models.ScenarioTag{Name: "cpu-hog", Digest: &digest}

	for i := 0; i < 2; i++ {
		status, err := cachedProvider.GetImageSignatureStatus(context.Background(), nil, tag)
		if err != nil {
			t.Fatalf("signature lookup %d failed: %v", i+1, err)
		}
		if status != verify.SignatureSigned {
			t.Fatalf("signature lookup %d returned %q, want signed", i+1, status)
		}
	}
	if got := mock.signatureCalls.Load(); got != 1 {
		t.Errorf("expected one underlying signature verification, got %d", got)
	}
}

func postScenariosForCacheTest(t *testing.T, handler *Handler) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, ScenariosPath, nil)
	response := httptest.NewRecorder()
	handler.PostScenarios(response, request)
	return response
}
