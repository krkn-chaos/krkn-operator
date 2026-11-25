package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	"github.com/krkn-chaos/krkn-operator/pkg/workflows"
	"github.com/krkn-chaos/krknctl/pkg/provider"
	"github.com/krkn-chaos/krknctl/pkg/provider/models"
	"github.com/krkn-chaos/krknctl/pkg/typing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMergeGraphNodeSecrets(t *testing.T) {
	secretVariables := map[string]struct{}{"TOKEN_A": {}, "TOKEN_B": {}}
	previous := krknv1alpha1.GraphScenarioNode{
		Env: map[string]string{"URL": "https://service.example", "TOKEN_A": "alpha", "TOKEN_B": "bravo"},
	}
	tests := []struct {
		name        string
		submitted   krknv1alpha1.GraphScenarioNode
		wantEnv     map[string]string
		invalidated bool
	}{
		{
			name:      "secret replacement preserves sibling",
			submitted: krknv1alpha1.GraphScenarioNode{Env: map[string]string{"URL": "https://service.example", "TOKEN_A": "replacement"}},
			wantEnv:   map[string]string{"URL": "https://service.example", "TOKEN_A": "replacement", "TOKEN_B": "bravo"},
		},
		{
			name:      "explicit secret removal preserves sibling",
			submitted: krknv1alpha1.GraphScenarioNode{Env: map[string]string{"URL": "https://service.example", "TOKEN_A": ""}},
			wantEnv:   map[string]string{"URL": "https://service.example", "TOKEN_B": "bravo"},
		},
		{
			name:        "cleartext change invalidates only old secrets",
			submitted:   krknv1alpha1.GraphScenarioNode{Env: map[string]string{"URL": "https://other.example"}},
			wantEnv:     map[string]string{"URL": "https://other.example"},
			invalidated: true,
		},
		{
			name:        "new secret supplied alongside cleartext change is accepted",
			submitted:   krknv1alpha1.GraphScenarioNode{Env: map[string]string{"URL": "https://other.example", "TOKEN_A": "new"}},
			wantEnv:     map[string]string{"URL": "https://other.example", "TOKEN_A": "new"},
			invalidated: true,
		},
		{
			name:        "non-environment execution change invalidates node secrets",
			submitted:   krknv1alpha1.GraphScenarioNode{Env: map[string]string{"URL": "https://service.example"}, RollbackEnabled: true},
			wantEnv:     map[string]string{"URL": "https://service.example"},
			invalidated: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, invalidated := mergeGraphNodeSecrets(previous, tt.submitted, secretVariables)
			if invalidated != tt.invalidated || !reflect.DeepEqual(got.Env, tt.wantEnv) {
				t.Fatalf("got env=%v invalidated=%t; want env=%v invalidated=%t", got.Env, invalidated, tt.wantEnv, tt.invalidated)
			}
			if !reflect.DeepEqual(previous.Env, map[string]string{"URL": "https://service.example", "TOKEN_A": "alpha", "TOKEN_B": "bravo"}) {
				t.Fatal("previous node was mutated")
			}
		})
	}
}

func TestWorkflowSecretEditInvalidatesOnlyChangedNode(t *testing.T) {
	h := setupWorkflowTestHandler()
	admin := &krknv1alpha1.KrknUser{ObjectMeta: metav1.ObjectMeta{Name: mustSanitizeUserIDForResourceName(t, "admin@test.example"), Namespace: h.namespace}, Spec: krknv1alpha1.KrknUserSpec{UserID: "admin@test.example"}}
	if err := h.client.Create(context.Background(), admin); err != nil {
		t.Fatal(err)
	}
	publicReader := &krknv1alpha1.KrknUser{ObjectMeta: metav1.ObjectMeta{Name: mustSanitizeUserIDForResourceName(t, "reader@test.example"), Namespace: h.namespace}, Spec: krknv1alpha1.KrknUserSpec{UserID: "reader@test.example"}}
	if err := h.client.Create(context.Background(), publicReader); err != nil {
		t.Fatal(err)
	}
	mock := &mockScenarioProvider{details: map[string]*models.ScenarioDetail{
		"test-scenario-1": {ScenarioTag: models.ScenarioTag{Name: "test-scenario-1"}, Fields: []typing.InputField{{Variable: stringPtr("TOKEN"), Secret: true}}},
		"test-scenario-2": {ScenarioTag: models.ScenarioTag{Name: "test-scenario-2"}, Fields: []typing.InputField{{Variable: stringPtr("TOKEN"), Secret: true}}},
	}}
	h.scenarioProviderFactory = func(provider.Mode) (provider.ScenarioDataProvider, error) {
		return mock, nil
	}
	graph := validWorkflowGraph()
	for nodeID, token := range map[string]string{"node1": "first-private", "node2": "second-private"} {
		node := graph[nodeID]
		node.Env = map[string]string{"URL": "https://service.example", "TOKEN": token}
		graph[nodeID] = node
	}
	createBody, _ := json.Marshal(workflows.CreateWorkflowRequest{WorkflowName: "secrets", Graph: graph, AvailableToAll: true})
	create := httptest.NewRecorder()
	h.CreateWorkflow(create, addAdminContext(httptest.NewRequest(http.MethodPost, WorkflowsPath, bytes.NewReader(createBody))))
	if create.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", create.Code, create.Body.String())
	}
	var created workflows.CreateWorkflowResponse
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	path := WorkflowsPath + "/" + created.WorkflowID
	get := httptest.NewRecorder()
	h.GetWorkflow(get, addAdminContext(httptest.NewRequest(http.MethodGet, path, nil)))
	if get.Code != http.StatusOK || strings.Contains(get.Body.String(), "first-private") || strings.Contains(get.Body.String(), "second-private") {
		t.Fatalf("workflow GET leaked or failed: %d %s", get.Code, get.Body.String())
	}
	publicGet := httptest.NewRecorder()
	h.GetWorkflow(publicGet, addUserContext(httptest.NewRequest(http.MethodGet, path, nil), "reader@test.example"))
	if publicGet.Code != http.StatusOK || strings.Contains(publicGet.Body.String(), "first-private") || strings.Contains(publicGet.Body.String(), "second-private") {
		t.Fatalf("public workflow GET leaked or failed: %d %s", publicGet.Code, publicGet.Body.String())
	}
	var visible workflows.WorkflowResponse
	if err := json.Unmarshal(get.Body.Bytes(), &visible); err != nil {
		t.Fatal(err)
	}
	if !visible.SecretFields["node1"]["TOKEN"] || !visible.SecretFields["node2"]["TOKEN"] {
		t.Fatalf("missing configured state: %#v", visible.SecretFields)
	}
	changed := visible.Graph["node1"]
	changed.Env["URL"] = "https://other.example"
	visible.Graph["node1"] = changed
	updateBody, _ := json.Marshal(workflows.UpdateWorkflowRequest{WorkflowName: "secrets", Graph: visible.Graph, AvailableToAll: true})
	update := httptest.NewRecorder()
	h.UpdateWorkflow(update, addAdminContext(httptest.NewRequest(http.MethodPut, path, bytes.NewReader(updateBody))))
	if update.Code != http.StatusOK {
		t.Fatalf("update: %d %s", update.Code, update.Body.String())
	}
	file, err := h.getFileByIDInternal(addAdminContext(httptest.NewRequest(http.MethodGet, path, nil)).Context(), created.WorkflowID)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := workflows.FromFileContent(file.Content)
	if err != nil {
		t.Fatal(err)
	}
	if stored["node1"].Env["TOKEN"] != "" || stored["node2"].Env["TOKEN"] != "second-private" {
		t.Fatalf("incorrect per-node invalidation: first=%v second=%v", stored["node1"].Env, stored["node2"].Env)
	}
	// A mutable image tag can change its input classification. A subsequent
	// update must not silently reuse the old secret under the new schema.
	mock.details["test-scenario-2"].Fields = nil
	update = httptest.NewRecorder()
	h.UpdateWorkflow(update, addAdminContext(httptest.NewRequest(http.MethodPut, path, bytes.NewReader(updateBody))))
	if update.Code != http.StatusOK {
		t.Fatalf("metadata-change update: %d %s", update.Code, update.Body.String())
	}
	file, err = h.getFileByIDInternal(addAdminContext(httptest.NewRequest(http.MethodGet, path, nil)).Context(), created.WorkflowID)
	if err != nil {
		t.Fatal(err)
	}
	stored, err = workflows.FromFileContent(file.Content)
	if err != nil {
		t.Fatal(err)
	}
	if stored["node2"].Env["TOKEN"] != "" {
		t.Fatal("old secret reused after metadata classification changed")
	}
}

func TestSavedSecretMetadataRedactsAfterScenarioMetadataChanges(t *testing.T) {
	index := workflowSecretIndex{"first": {"TOKEN": {}}}
	encoded, err := encodeSecretIndex(index)
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{scenarioProviderFactory: func(provider.Mode) (provider.ScenarioDataProvider, error) {
		return nil, errors.New("scenario tag unavailable")
	}}
	graph := map[string]krknv1alpha1.GraphScenarioNode{
		"first": {Scenario: krknv1alpha1.ScenarioReference{Name: "scenario"}, Env: map[string]string{"TOKEN": "private-value", "URL": "https://service.example"}},
	}
	response := workflows.WorkflowResponse{
		Graph:                 graph,
		SecretInputFieldsJSON: encoded,
		StudioLayout: map[string]interface{}{"nodes": []interface{}{map[string]interface{}{
			"nodeId": "first", "config": map[string]interface{}{
				"scenarioFormValues": map[string]interface{}{"TOKEN": "private-value", "URL": "https://service.example"},
				"clearSecretFields":  []interface{}{"TOKEN"},
			},
		}}},
	}
	if err := h.redactWorkflowResponse(context.Background(), &response); err != nil {
		t.Fatal(err)
	}
	if _, leaked := response.Graph["first"].Env["TOKEN"]; leaked {
		t.Fatal("secret leaked through workflow graph")
	}
	if !response.SecretFields["first"]["TOKEN"] {
		t.Fatal("configured state was not reported")
	}
	config := response.StudioLayout["nodes"].([]interface{})[0].(map[string]interface{})["config"].(map[string]interface{})
	if _, leaked := config["scenarioFormValues"].(map[string]interface{})["TOKEN"]; leaked {
		t.Fatal("secret leaked through Studio layout")
	}
	if _, retained := config["clearSecretFields"]; retained {
		t.Fatal("transient secret deletion marker was persisted in Studio layout")
	}
	content, err := workflows.ToFileContent(graph)
	if err != nil {
		t.Fatal(err)
	}
	file, err := h.safeFileResponse(context.Background(), files.FileResponse{FilePurpose: files.FilePurposeWorkflow, Content: content, SecretInputFieldsJSON: encoded})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(file.Content, "private-value") {
		t.Fatal("secret leaked through generic file response")
	}
}

func TestSavedScenarioSecretMetadataRedactsAfterMetadataChanges(t *testing.T) {
	h := &Handler{scenarioProviderFactory: func(provider.Mode) (provider.ScenarioDataProvider, error) {
		return nil, errors.New("scenario tag unavailable")
	}}
	request := ScenarioRunRequest{
		Scenario:    krknv1alpha1.ScenarioReference{Name: "scenario"},
		Environment: map[string]string{"TOKEN": "private-value", "URL": "https://service.example"},
	}
	if err := h.redactScenarioRunRequest(context.Background(), &request, `{"TOKEN":{}}`); err != nil {
		t.Fatal(err)
	}
	if _, leaked := request.Environment["TOKEN"]; leaked {
		t.Fatal("secret leaked through scenario run config")
	}
	if !request.SecretFields["TOKEN"] || request.Environment["URL"] != "https://service.example" {
		t.Fatalf("incorrect redacted scenario response: %#v", request)
	}
}

func TestRedactSecretEnvironment(t *testing.T) {
	visible, state := redactSecretEnvironment(
		map[string]string{"URL": "https://service.example", "TOKEN_A": "alpha"},
		map[string]struct{}{"TOKEN_A": {}, "TOKEN_B": {}},
	)
	if !reflect.DeepEqual(visible, map[string]string{"URL": "https://service.example"}) {
		t.Fatalf("visible environment contains unexpected values: %v", visible)
	}
	if !reflect.DeepEqual(state, secretFieldState{"TOKEN_A": true, "TOKEN_B": false}) {
		t.Fatalf("unexpected secret state: %v", state)
	}
}

func TestSecretInputMetadataDoesNotReturnDefault(t *testing.T) {
	fields := convertInputFields([]typing.InputField{
		{Variable: stringPtr("TOKEN"), Default: stringPtr("private-default"), Secret: true},
		{Variable: stringPtr("URL"), Default: stringPtr("https://service.example")},
	})
	if fields[0].Default != nil || fields[1].Default == nil || *fields[1].Default != "https://service.example" {
		t.Fatalf("unexpected defaults in response: %#v", fields)
	}
}
