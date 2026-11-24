package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestPostScenarioRunPersistsResiliencyBaseline(t *testing.T) {
	handler := setupScenarioRunTestHandler("test-request-id", map[string]string{
		"test-cluster": "YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmNsdXN0ZXJzOiBbXQpjb250ZXh0czogW10KdXNlcnM6IFtd",
	})
	request := `{"targetRequestId":"test-request-id","targetClusters":{"krkn-operator":["test-cluster"]},"scenario":{"name":"pod-delete","private":false},"resiliencyScoreEnabled":true,"resiliencyScoreBaseline":80.5}`
	response := httptest.NewRecorder()
	handler.PostScenarioRun(response, httptest.NewRequest(http.MethodPost, ScenariosRunPath, strings.NewReader(request)))
	if response.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want 201: %s", response.Code, response.Body.String())
	}

	var created ScenarioRunCreateResponse
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	var run krknv1alpha1.KrknScenarioRun
	if err := handler.client.Get(context.Background(), client.ObjectKey{Name: created.ScenarioRunName, Namespace: "default"}, &run); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(run.Spec)
	if err != nil {
		t.Fatal(err)
	}
	var spec map[string]any
	if err := json.Unmarshal(encoded, &spec); err != nil {
		t.Fatal(err)
	}
	if got := spec["resiliencyScoreBaseline"]; got != 80.5 {
		t.Fatalf("persisted baseline = %v, want 80.5", got)
	}

	statusResponse := httptest.NewRecorder()
	handler.GetScenarioRunStatus(statusResponse, httptest.NewRequest(http.MethodGet, ScenariosRunPath+"/"+created.ScenarioRunName, nil))
	if statusResponse.Code != http.StatusCreated && statusResponse.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", statusResponse.Code, statusResponse.Body.String())
	}
	var status map[string]any
	if err := json.Unmarshal(statusResponse.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if got := status["resiliencyScoreBaseline"]; got != 80.5 {
		t.Fatalf("status baseline = %v, want 80.5", got)
	}

	listResponse := httptest.NewRecorder()
	handler.ListScenarioRuns(listResponse, httptest.NewRequest(http.MethodGet, ScenariosRunPath, nil))
	if listResponse.Code != http.StatusOK {
		t.Fatalf("GET list = %d: %s", listResponse.Code, listResponse.Body.String())
	}
	var list struct {
		ScenarioRuns []map[string]any `json:"scenarioRuns"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.ScenarioRuns) != 1 || list.ScenarioRuns[0]["resiliencyScoreBaseline"] != 80.5 {
		t.Fatalf("list baseline = %+v, want 80.5", list.ScenarioRuns)
	}
}

func TestPostScenarioRunBaselineValidation(t *testing.T) {
	cases := []struct {
		name     string
		fields   string
		wantCode int
		wantBase *float64
	}{
		{name: "negative baseline", fields: `,"resiliencyScoreEnabled":true,"resiliencyScoreBaseline":-2`, wantCode: http.StatusBadRequest},
		{name: "zero baseline", fields: `,"resiliencyScoreEnabled":true,"resiliencyScoreBaseline":0`, wantCode: http.StatusCreated, wantBase: floatPointer(0)},
		{name: "legacy enabled without baseline", fields: `,"resiliencyScoreEnabled":true`, wantCode: http.StatusCreated},
		{name: "disabled baseline ignored", fields: `,"resiliencyScoreBaseline":80`, wantCode: http.StatusCreated},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := setupScenarioRunTestHandler("test-request-id", map[string]string{
				"test-cluster": "YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmNsdXN0ZXJzOiBbXQpjb250ZXh0czogW10KdXNlcnM6IFtd",
			})
			request := fmt.Sprintf(`{"targetRequestId":"test-request-id","targetClusters":{"krkn-operator":["test-cluster"]},"scenario":{"name":"pod-delete","private":false}%s}`, tc.fields)
			response := httptest.NewRecorder()
			handler.PostScenarioRun(response, httptest.NewRequest(http.MethodPost, ScenariosRunPath, strings.NewReader(request)))
			if response.Code != tc.wantCode {
				t.Fatalf("POST status = %d, want %d: %s", response.Code, tc.wantCode, response.Body.String())
			}
			if tc.wantCode != http.StatusCreated {
				return
			}
			var created ScenarioRunCreateResponse
			if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
				t.Fatal(err)
			}
			var run krknv1alpha1.KrknScenarioRun
			if err := handler.client.Get(context.Background(), client.ObjectKey{Name: created.ScenarioRunName, Namespace: "default"}, &run); err != nil {
				t.Fatal(err)
			}
			if tc.wantBase == nil && run.Spec.ResiliencyScoreBaseline != nil || tc.wantBase != nil && (run.Spec.ResiliencyScoreBaseline == nil || *run.Spec.ResiliencyScoreBaseline != *tc.wantBase) {
				t.Fatalf("persisted baseline = %v, want %v", run.Spec.ResiliencyScoreBaseline, tc.wantBase)
			}
		})
	}
}
