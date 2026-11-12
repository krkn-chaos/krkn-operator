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
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
			{Name: "cpu-hog", Digest: digestPointer("sha256:cpu-hog")},
			{Name: "base-image", Digest: digestPointer("sha256:base-image")},
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

func digestPointer(digest string) *string {
	return &digest
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

func TestPublicSignatureCacheDoesNotCacheUnknown(t *testing.T) {
	digest := "sha256:cpu-hog"
	var calls atomic.Int32
	mock := &mockScenarioProvider{signatureFn: func(context.Context, *models.RegistryV2, models.ScenarioTag) (verify.SignatureStatus, error) {
		if calls.Add(1) == 1 {
			return verify.SignatureUnknown, nil
		}
		return verify.SignatureSigned, nil
	}}
	cachedProvider := &publicSignatureCachingProvider{ScenarioDataProvider: mock, cache: &publicSignatureCache{}}
	tag := models.ScenarioTag{Name: "cpu-hog", Digest: &digest}

	first, err := cachedProvider.GetImageSignatureStatus(context.Background(), nil, tag)
	if err != nil || first != verify.SignatureUnknown {
		t.Fatalf("first signature lookup = (%q, %v), want unknown without error", first, err)
	}
	second, err := cachedProvider.GetImageSignatureStatus(context.Background(), nil, tag)
	if err != nil || second != verify.SignatureSigned {
		t.Fatalf("second signature lookup = (%q, %v), want signed without error", second, err)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("expected unknown result to be retried, got %d provider calls", got)
	}
}

func TestPublicSignatureCacheDoesNotReuseMutableTag(t *testing.T) {
	var calls atomic.Int32
	mock := &mockScenarioProvider{signatureFn: func(context.Context, *models.RegistryV2, models.ScenarioTag) (verify.SignatureStatus, error) {
		if calls.Add(1) == 1 {
			return verify.SignatureSigned, nil
		}
		return verify.SignatureUnsigned, nil
	}}
	cachedProvider := &publicSignatureCachingProvider{ScenarioDataProvider: mock, cache: &publicSignatureCache{}}
	tag := models.ScenarioTag{Name: "latest"}

	first, err := cachedProvider.GetImageSignatureStatus(context.Background(), nil, tag)
	if err != nil || first != verify.SignatureSigned {
		t.Fatalf("first signature lookup = (%q, %v), want signed without error", first, err)
	}
	second, err := cachedProvider.GetImageSignatureStatus(context.Background(), nil, tag)
	if err != nil || second != verify.SignatureUnsigned {
		t.Fatalf("second signature lookup = (%q, %v), want unsigned without error", second, err)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("expected mutable tag to be verified each time, got %d provider calls", got)
	}
}

func TestPublicSignatureCacheLoadOutlivesFirstWaiter(t *testing.T) {
	cache := &publicSignatureCache{}
	firstCtx, cancelFirst := context.WithCancel(context.Background())
	defer cancelFirst()
	loadStarted := make(chan struct{})
	releaseLoad := make(chan struct{})
	var loadCalls atomic.Int32
	load := func(context.Context) (verify.SignatureStatus, error) {
		loadCalls.Add(1)
		close(loadStarted)
		<-releaseLoad
		return verify.SignatureSigned, nil
	}

	firstResult := make(chan error, 1)
	go func() {
		_, err := cache.getOrLoad(firstCtx, "digest:sha256:cpu-hog", load)
		firstResult <- err
	}()
	<-loadStarted

	secondResult := make(chan struct {
		status verify.SignatureStatus
		err    error
	}, 1)
	go func() {
		status, err := cache.getOrLoad(context.Background(), "digest:sha256:cpu-hog", load)
		secondResult <- struct {
			status verify.SignatureStatus
			err    error
		}{status: status, err: err}
	}()
	time.Sleep(20 * time.Millisecond)
	cancelFirst()
	if err := <-firstResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("first waiter error = %v, want context canceled", err)
	}
	close(releaseLoad)
	result := <-secondResult
	if result.err != nil || result.status != verify.SignatureSigned {
		t.Fatalf("remaining waiter got (%q, %v), want signed without error", result.status, result.err)
	}
	if got := loadCalls.Load(); got != 1 {
		t.Errorf("expected one shared signature lookup, got %d", got)
	}
}

func TestPostScenariosRetriesUnknownSignatureBeforeCachingCatalog(t *testing.T) {
	handler := setupFilesTestHandler()
	digest := "sha256:cpu-hog"
	var signatureCalls atomic.Int32
	mock := &mockScenarioProvider{
		tags: []models.ScenarioTag{{Name: "cpu-hog", Digest: &digest}},
		details: map[string]*models.ScenarioDetail{
			"cpu-hog": {ScenarioTag: models.ScenarioTag{Name: "cpu-hog"}, IsAScenario: true},
		},
		signatureFn: func(context.Context, *models.RegistryV2, models.ScenarioTag) (verify.SignatureStatus, error) {
			if signatureCalls.Add(1) == 1 {
				return verify.SignatureUnknown, nil
			}
			return verify.SignatureSigned, nil
		},
	}
	handler.scenarioProviderFactory = func(provider.Mode) (provider.ScenarioDataProvider, error) { return mock, nil }

	first := postScenariosForCacheTest(t, handler)
	second := postScenariosForCacheTest(t, handler)
	third := postScenariosForCacheTest(t, handler)
	if first.Code != http.StatusOK || second.Code != http.StatusOK || third.Code != http.StatusOK {
		t.Fatalf("expected all requests to succeed, got %d, %d, %d", first.Code, second.Code, third.Code)
	}
	if !strings.Contains(second.Body.String(), `"signature_status":"signed"`) {
		t.Errorf("expected successful verification on retry to be visible, got %s", second.Body.String())
	}
	if got := mock.signatureCalls.Load(); got != 2 {
		t.Errorf("expected unknown to retry and signed catalog to cache, got %d signature calls", got)
	}
}

func TestPostScenariosDoesNotCacheNilDetail(t *testing.T) {
	handler := setupFilesTestHandler()
	digest := "sha256:cpu-hog"
	var detailCalls atomic.Int32
	mock := &mockScenarioProvider{
		tags:      []models.ScenarioTag{{Name: "cpu-hog", Digest: &digest}},
		signature: verify.SignatureSigned,
		detailFn: func(string, *models.RegistryV2) (*models.ScenarioDetail, error) {
			if detailCalls.Add(1) == 1 {
				return nil, nil
			}
			return &models.ScenarioDetail{ScenarioTag: models.ScenarioTag{Name: "cpu-hog"}, IsAScenario: true}, nil
		},
	}
	handler.scenarioProviderFactory = func(provider.Mode) (provider.ScenarioDataProvider, error) { return mock, nil }

	first := postScenariosForCacheTest(t, handler)
	second := postScenariosForCacheTest(t, handler)
	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("expected both requests to succeed, got %d and %d", first.Code, second.Code)
	}
	if !strings.Contains(second.Body.String(), "cpu-hog") {
		t.Errorf("expected reappearing scenario to be returned on retry, got %s", second.Body.String())
	}
	if got := mock.detailCalls.Load(); got != 2 {
		t.Errorf("expected nil detail to force a retry, got %d detail calls", got)
	}
}

func TestFetchScenarioCatalogReturnsTimeoutInsteadOfPartialCatalog(t *testing.T) {
	digest := "sha256:cpu-hog"
	started := make(chan struct{})
	release := make(chan struct{})
	mock := &mockScenarioProvider{
		tags:      []models.ScenarioTag{{Name: "cpu-hog", Digest: &digest}},
		signature: verify.SignatureSigned,
		detailFn: func(string, *models.RegistryV2) (*models.ScenarioDetail, error) {
			close(started)
			<-release
			return &models.ScenarioDetail{ScenarioTag: models.ScenarioTag{Name: "cpu-hog"}, IsAScenario: true}, nil
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, _, err := fetchScenarioCatalog(ctx, mock, nil)
		result <- err
	}()
	<-started
	err := <-result
	close(release)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("fetch error = %v, want context deadline exceeded", err)
	}
}

func TestGetRegistryImagesWithContextReturnsWhenProviderStalls(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	mock := &mockScenarioProvider{registryImagesFn: func(*models.RegistryV2) (*[]models.ScenarioTag, error) {
		close(started)
		<-release
		return nil, nil
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := getRegistryImagesWithContext(ctx, mock, nil)
		result <- err
	}()
	<-started
	err := <-result
	close(release)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("registry listing error = %v, want context deadline exceeded", err)
	}
}

func TestScenarioCatalogErrorIncludesOperationAndUnwrapsCause(t *testing.T) {
	cause := errors.New("registry unavailable")
	err := &scenarioCatalogError{operation: "listing scenarios", err: cause}
	if !errors.Is(err, cause) {
		t.Fatal("errors.Is did not find the underlying catalog error")
	}
	if !strings.Contains(err.Error(), "listing scenarios") {
		t.Fatalf("error string %q does not include operation context", err)
	}
}

func postScenariosForCacheTest(t *testing.T, handler *Handler) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, ScenariosPath, nil)
	response := httptest.NewRecorder()
	handler.PostScenarios(response, request)
	return response
}
