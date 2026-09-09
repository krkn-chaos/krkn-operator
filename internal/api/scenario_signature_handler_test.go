package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krkn-chaos/krknctl/pkg/provider"
	"github.com/krkn-chaos/krknctl/pkg/provider/models"
)

func TestPostScenariosPropagatesSignatureStatus(t *testing.T) {
	handler := setupFilesTestHandler()
	mock := &mockScenarioProvider{
		tags:      []models.ScenarioTag{{Name: "cpu-hog"}},
		signature: "signed",
		details: map[string]*models.ScenarioDetail{
			"cpu-hog": {ScenarioTag: models.ScenarioTag{Name: "cpu-hog"}, IsAScenario: true},
		},
	}
	handler.scenarioProviderFactory = func(provider.Mode) (provider.ScenarioDataProvider, error) {
		return mock, nil
	}

	request := httptest.NewRequest(http.MethodPost, ScenariosPath, nil)
	response := httptest.NewRecorder()
	handler.PostScenarios(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	var body ScenariosResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Scenarios) != 1 || body.Scenarios[0].SignatureStatus != "signed" {
		t.Fatalf("expected signed scenario result, got %#v", body.Scenarios)
	}
}

func TestPostScenariosReturnsErrorWhenSignatureVerificationFails(t *testing.T) {
	handler := setupFilesTestHandler()
	mock := &mockScenarioProvider{
		tags: []models.ScenarioTag{{Name: "cpu-hog"}},
	}
	handler.scenarioProviderFactory = func(provider.Mode) (provider.ScenarioDataProvider, error) {
		return mock, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodPost, ScenariosPath, nil).WithContext(ctx)
	response := httptest.NewRecorder()
	handler.PostScenarios(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d: %s", response.Code, response.Body.String())
	}
}
