package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	corev1 "k8s.io/api/core/v1"
)

func TestKrknAIDiscoveryRejectsNoRunnableScenarios(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
	}{{"none available", false}, {"scenario available", true}} {
		t.Run(tc.name, func(t *testing.T) {
			config := "scenario:\n  pod-scenarios:\n    enable: false\n"
			if tc.enabled {
				config = "scenario:\n  pod-scenarios:\n    enable: true\n"
			}
			service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{"configYaml": config, "warnings": []string{"missing helper image"}})
			}))
			defer service.Close()
			h := newKrknAITestHandler(t, service.URL)
			response := httptest.NewRecorder()
			h.KrknAIRouter(response, adminKrknAIRequest(http.MethodPost, KrknAIPath+"/discoveries", `{"targetRequestId":"target","targetClusters":{"provider":["cluster"]}}`))
			if tc.enabled {
				if response.Code != http.StatusOK {
					t.Fatalf("valid discovery: %d %s", response.Code, response.Body.String())
				}
			} else if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "No runnable scenarios") {
				t.Fatalf("empty discovery: %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestKrknAIRejectsEmptyScenarioConfigSaveAndStart(t *testing.T) {
	h := newKrknAITestHandler(t, "http://unused")
	for _, request := range []struct{ path, body string }{
		{"/configs/validate", `{"configYaml":"scenario: {}\n"}`},
		{"/configs", `{"name":"empty-config","configYaml":"scenario: {}\n","targetRequestId":"target","targetClusters":{"provider":["cluster"]}}`},
	} {
		response := httptest.NewRecorder()
		h.KrknAIRouter(response, adminKrknAIRequest(http.MethodPost, KrknAIPath+request.path, request.body))
		if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "Enable at least one runnable scenario") {
			t.Fatalf("empty config %s: %d %s", request.path, response.Code, response.Body.String())
		}
	}
	var maps corev1.ConfigMapList
	if err := h.client.List(t.Context(), &maps); err != nil {
		t.Fatal(err)
	}
	if len(maps.Items) != 0 {
		t.Fatal("empty configuration was persisted")
	}

	save := httptest.NewRecorder()
	h.KrknAIRouter(save, adminKrknAIRequest(http.MethodPost, KrknAIPath+"/configs", `{"name":"valid-config","configYaml":"scenario:\n  pod-scenarios:\n    enable: true\n","targetRequestId":"target","targetClusters":{"provider":["cluster"]}}`))
	if save.Code != http.StatusCreated {
		t.Fatalf("valid save: %d %s", save.Code, save.Body.String())
	}
	var created struct {
		ConfigID string `json:"configId"`
	}
	if err := json.Unmarshal(save.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	config, err := h.loadKrknAIConfigMapByID(t.Context(), created.ConfigID)
	if err != nil {
		t.Fatal(err)
	}
	config.Data[files.KrknAIConfigFileName] = "scenario: {}\n"
	if err := h.client.Update(t.Context(), config); err != nil {
		t.Fatal(err)
	}
	start := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{"name": "empty-run", "configId": created.ConfigID, "targetRequestId": "target", "targetClusters": map[string][]string{"provider": {"cluster"}}})
	h.KrknAIRouter(start, adminKrknAIRequest(http.MethodPost, KrknAIPath+"/runs", string(body)))
	if start.Code != http.StatusUnprocessableEntity {
		t.Fatalf("empty start: %d %s", start.Code, start.Body.String())
	}
	var runs krknv1alpha1.KrknAIRunList
	if err := h.client.List(t.Context(), &runs); err != nil {
		t.Fatal(err)
	}
	if len(runs.Items) != 0 {
		t.Fatal("empty configuration created a run")
	}
}
