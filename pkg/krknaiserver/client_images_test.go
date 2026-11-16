package krknaiserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/krkn-chaos/krkn-operator/pkg/registry"
)

func TestImageRequirementsUsesAuthenticatedServiceRoute(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/requirements" || r.Method != http.MethodGet {
			t.Errorf("request = %s %s, want GET /v1/images/requirements", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("authorization = %q, want service bearer token", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"scenarioTags":["pod-delete"],"workloadImageTags":{"hog":"workload-krkn-hog"}}`))
	}))
	defer service.Close()

	got, err := New(service.URL, "token").ImageRequirements(context.Background())
	if err != nil {
		t.Fatalf("ImageRequirements() error = %v", err)
	}
	if len(got.ScenarioTags) != 1 || got.ScenarioTags[0] != "pod-delete" || got.WorkloadImageTags["hog"] != "workload-krkn-hog" {
		t.Fatalf("ImageRequirements() = %+v", got)
	}
}

func TestValidateConfigImagesForwardsInventoryAndClassifiesValidation(t *testing.T) {
	inventory := &registry.AIImageInventory{
		RegistryName: "private", Repository: "registry.example:5000/org/repo",
		AvailableTags: []string{"pod-delete"}, WorkloadImageTags: map[string]string{"hog": "workload-krkn-hog"},
		OrchestratorImage:   "registry.example:5000/org/repo@sha256:abc",
		ResultUploaderImage: "registry.example:5000/org/repo@sha256:def",
	}
	var calls int
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/v1/configs/validate" || r.Method != http.MethodPost {
			t.Errorf("request = %s %s, want POST /v1/configs/validate", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("authorization = %q, want service bearer token", r.Header.Get("Authorization"))
		}
		var body struct {
			ConfigYAML     string                    `json:"configYaml"`
			RegistryName   string                    `json:"registryName"`
			ImageInventory registry.AIImageInventory `json:"imageInventory"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body.ConfigYAML != runnableConfigYAML || body.RegistryName != inventory.RegistryName || body.ImageInventory.RegistryName != inventory.RegistryName || body.ImageInventory.OrchestratorImage != inventory.OrchestratorImage {
			t.Errorf("validation payload = %+v", body)
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":[{"path":"scenario","message":"missing private image"}]}`))
	}))
	defer service.Close()

	err := New(service.URL, "token").ValidateConfigImages(context.Background(), runnableConfigYAML, inventory)
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("ValidateConfigImages() error = %v, want ErrInvalidConfig", err)
	}
	var validation *ConfigValidationError
	if !errors.As(err, &validation) || len(validation.Errors) != 1 || validation.Errors[0].Path != "scenario" {
		t.Fatalf("validation details = %#v", validation)
	}
	if calls != 1 {
		t.Fatalf("validation calls = %d, want 1", calls)
	}
}

func TestValidateConfigImagesDoesNotExposeServiceErrorBody(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("secret response body"))
	}))
	defer service.Close()

	err := New(service.URL, "token").ValidateConfigImages(context.Background(), runnableConfigYAML, &registry.AIImageInventory{})
	if err == nil || strings.Contains(err.Error(), "secret response body") {
		t.Fatalf("ValidateConfigImages() leaked service body: %v", err)
	}
}
