package krknaiserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/krkn-chaos/krkn-operator/pkg/registry"
)

func TestDiscoveryDeadlineDoesNotShortenOrdinaryRequests(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(75 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
		}
	}))
	defer service.Close()
	client := New(service.URL, "token")
	client.discoveryClient.Timeout = 25 * time.Millisecond

	_, err := client.DoDiscovery(context.Background(), http.MethodPost, "/v1/discoveries", nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("discovery error = %v, want deadline exceeded", err)
	}

	response, err := client.Do(context.Background(), http.MethodPost, "/v1/configs/validate", nil)
	if err != nil {
		t.Fatalf("ordinary request inherited the short discovery deadline: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("ordinary request status = %d, want 200", response.StatusCode)
	}
}

func TestImageRequirementsReturnsCanonicalCatalog(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/images/requirements" || r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("unexpected image requirements request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"scenarioTags":["scenario-v1"],"workloadImageTags":{"hog":"workload-krkn-hog"}}`))
	}))
	defer service.Close()

	requirements, err := New(service.URL, "token").ImageRequirements(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(requirements.ScenarioTags) != 1 || requirements.ScenarioTags[0] != "scenario-v1" || requirements.WorkloadImageTags["hog"] != "workload-krkn-hog" {
		t.Fatalf("unexpected image requirements: %+v", requirements)
	}
}

func TestValidateConfigImagesSendsInventoryAndReturnsTypedValidationError(t *testing.T) {
	var calls int
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/v1/configs/validate" || r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("unexpected image validation request: %s %s", r.Method, r.URL.Path)
		}
		var request struct {
			ConfigYAML     string                     `json:"configYaml"`
			ImageInventory *registry.AIImageInventory `json:"imageInventory"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode validation request: %v", err)
		}
		if request.ConfigYAML != runnableConfigYAML || request.ImageInventory == nil || request.ImageInventory.RegistryName != "private" {
			t.Errorf("unexpected validation payload: %+v", request)
		}
		if calls == 1 {
			_, _ = w.Write([]byte(`{"valid":true}`))
			return
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":[{"path":"scenario","message":"unknown image tag"}],"secret":"must-not-escape"}`))
	}))
	defer service.Close()

	inventory := &registry.AIImageInventory{RegistryName: "private", Repository: "registry.example/org/repo"}
	client := New(service.URL, "token")
	if err := client.ValidateConfigImages(context.Background(), runnableConfigYAML, inventory); err != nil {
		t.Fatalf("valid config error = %v", err)
	}
	err := client.ValidateConfigImages(context.Background(), runnableConfigYAML, inventory)
	var validation *ConfigValidationError
	if !errors.Is(err, ErrInvalidConfig) || !errors.As(err, &validation) || len(validation.Errors) != 1 || validation.Errors[0].Path != "scenario" {
		t.Fatalf("typed validation error = %#v", err)
	}
	if err.Error() == "" || err.Error() == "must-not-escape" {
		t.Fatalf("unexpected validation error message: %v", err)
	}
}
