package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	"github.com/krkn-chaos/krkn-operator/pkg/registry"
	"github.com/krkn-chaos/krkn-operator/pkg/workflows"
	"github.com/krkn-chaos/krknctl/pkg/provider"
	"github.com/krkn-chaos/krknctl/pkg/provider/models"
	"github.com/krkn-chaos/krknctl/pkg/typing"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// secretFieldState reports only whether a secret input has a value. The value
// itself must never be copied into an application read response.
type secretFieldState map[string]bool

type workflowSecretIndex map[string]map[string]struct{}

const secretInputFieldsAnnotation = "krkn.krkn-chaos.dev/secret-input-fields"

func encodeSecretIndex(index workflowSecretIndex) (string, error) {
	encoded, err := json.Marshal(index)
	return string(encoded), err
}

func indexFromSecretState(state map[string]map[string]bool) workflowSecretIndex {
	index := make(workflowSecretIndex, len(state))
	for nodeID, fields := range state {
		index[nodeID] = make(map[string]struct{}, len(fields))
		for variable := range fields {
			index[nodeID][variable] = struct{}{}
		}
	}
	return index
}

func decodeSecretIndex(encoded string) (workflowSecretIndex, error) {
	var index workflowSecretIndex
	if encoded == "" {
		return nil, nil
	}
	if err := json.Unmarshal([]byte(encoded), &index); err != nil {
		return nil, fmt.Errorf("invalid saved secret input metadata: %w", err)
	}
	return index, nil
}

func unionSecretIndexes(saved, current workflowSecretIndex) workflowSecretIndex {
	combined := make(workflowSecretIndex, len(saved)+len(current))
	for nodeID, fields := range saved {
		combined[nodeID] = make(map[string]struct{}, len(fields))
		for variable := range fields {
			combined[nodeID][variable] = struct{}{}
		}
	}
	for nodeID, fields := range current {
		if combined[nodeID] == nil {
			combined[nodeID] = make(map[string]struct{}, len(fields))
		}
		for variable := range fields {
			combined[nodeID][variable] = struct{}{}
		}
	}
	return combined
}

func (h *Handler) workflowIndexForRead(ctx context.Context, graph map[string]krknv1alpha1.GraphScenarioNode, layout map[string]interface{}, encoded string) (workflowSecretIndex, error) {
	saved, err := decodeSecretIndex(encoded)
	if err != nil {
		return nil, err
	}
	current, currentErr := h.secretIndexForWorkflow(ctx, graph, layout)
	if currentErr != nil && saved == nil {
		return nil, currentErr
	}
	return unionSecretIndexes(saved, current), nil
}

func (h *Handler) secretIndexForWorkflow(ctx context.Context, graph map[string]krknv1alpha1.GraphScenarioNode, layout map[string]interface{}) (workflowSecretIndex, error) {
	index := make(workflowSecretIndex, len(graph))
	for nodeID, node := range graph {
		if len(node.Env) == 0 && !studioNodeHasFormValues(layout, nodeID) {
			continue
		}
		normalizeLegacyGraphScenarioNode(&node)
		variables, err := h.secretVariablesForScenario(ctx, node.Scenario)
		if err != nil {
			return nil, fmt.Errorf("node %q: %w", nodeID, err)
		}
		index[nodeID] = variables
	}
	return index, nil
}

func studioNodeHasFormValues(layout map[string]interface{}, nodeID string) bool {
	nodes, ok := layout["nodes"].([]interface{})
	if !ok {
		return false
	}
	for _, rawNode := range nodes {
		node, ok := rawNode.(map[string]interface{})
		if !ok || node["nodeId"] != nodeID {
			continue
		}
		config, ok := node["config"].(map[string]interface{})
		if !ok {
			return false
		}
		for _, formName := range []string{"scenarioFormValues", "globalFormValues"} {
			if form, ok := config[formName].(map[string]interface{}); ok && len(form) > 0 {
				return true
			}
		}
	}
	return false
}

// redactStudioLayout removes secret values from the two form-value maps in
// each Studio node. The caller owns the decoded layout map.
func redactStudioLayout(layout map[string]interface{}, index workflowSecretIndex) {
	nodes, ok := layout["nodes"].([]interface{})
	if !ok {
		return
	}
	for _, rawNode := range nodes {
		node, ok := rawNode.(map[string]interface{})
		if !ok {
			continue
		}
		nodeID, _ := node["nodeId"].(string)
		variables := index[nodeID]
		config, ok := node["config"].(map[string]interface{})
		if !ok {
			continue
		}
		delete(config, "clearSecretFields")
		for _, formName := range []string{"scenarioFormValues", "globalFormValues"} {
			form, ok := config[formName].(map[string]interface{})
			if !ok {
				continue
			}
			for variable := range variables {
				delete(form, variable)
			}
		}
	}
}

// redactWorkflowResponse is the application read boundary for workflow secret
// inputs. The persisted graph may contain values, but neither graph nor Studio
// layout in the HTTP response does.
func (h *Handler) redactWorkflowResponse(ctx context.Context, response *workflows.WorkflowResponse) error {
	index, err := h.workflowIndexForRead(ctx, response.Graph, response.StudioLayout, response.SecretInputFieldsJSON)
	if err != nil {
		return err
	}
	response.SecretFields = make(map[string]map[string]bool, len(index))
	for nodeID, node := range response.Graph {
		variables := index[nodeID]
		if len(variables) == 0 {
			continue
		}
		visible, state := redactSecretEnvironment(node.Env, variables)
		node.Env = visible
		response.Graph[nodeID] = node
		response.SecretFields[nodeID] = map[string]bool(state)
	}
	redactStudioLayout(response.StudioLayout, index)
	return nil
}

func (h *Handler) redactGraphEnvironment(ctx context.Context, graph map[string]krknv1alpha1.GraphScenarioNode, encoded string) (map[string]krknv1alpha1.GraphScenarioNode, map[string]map[string]bool, error) {
	index, err := h.workflowIndexForRead(ctx, graph, nil, encoded)
	if err != nil {
		return nil, nil, err
	}
	visible := make(map[string]krknv1alpha1.GraphScenarioNode, len(graph))
	state := make(map[string]map[string]bool)
	for nodeID, node := range graph {
		if len(index[nodeID]) > 0 {
			var nodeState secretFieldState
			node.Env, nodeState = redactSecretEnvironment(node.Env, index[nodeID])
			state[nodeID] = map[string]bool(nodeState)
		}
		visible[nodeID] = node
	}
	return visible, state, nil
}

func (h *Handler) redactScenarioRunRequest(ctx context.Context, request *ScenarioRunRequest, encoded string) error {
	if len(request.Environment) == 0 {
		return nil
	}
	variables, err := h.secretVariablesForScenario(ctx, request.Scenario)
	if err != nil && encoded == "" {
		return err
	}
	var saved map[string]struct{}
	if encoded != "" {
		if decodeErr := json.Unmarshal([]byte(encoded), &saved); decodeErr != nil {
			return fmt.Errorf("invalid saved scenario secret metadata: %w", decodeErr)
		}
	}
	if variables == nil {
		variables = make(map[string]struct{})
	}
	for variable := range saved {
		variables[variable] = struct{}{}
	}
	visible, state := redactSecretEnvironment(request.Environment, variables)
	request.Environment = visible
	request.SecretFields = map[string]bool(state)
	return nil
}

func (h *Handler) mergeGraphRunSourceSecrets(ctx context.Context, request *GraphRunCreateRequest) error {
	if request.SourceWorkflowID == "" && request.SourceGraphRunName == "" {
		return nil
	}
	if request.SourceWorkflowID != "" && request.SourceGraphRunName != "" {
		return fmt.Errorf("only one graph run source may be specified")
	}
	var previousGraph map[string]krknv1alpha1.GraphScenarioNode
	var previousIndex workflowSecretIndex
	if request.SourceWorkflowID != "" {
		fileResponse, err := h.getFileByIDInternal(ctx, request.SourceWorkflowID)
		if err != nil || fileResponse.FilePurpose != files.FilePurposeWorkflow {
			return fmt.Errorf("source workflow is not available")
		}
		previousGraph, err = workflows.FromFileContent(fileResponse.Content)
		if err != nil {
			return fmt.Errorf("parse source workflow: %w", err)
		}
		previousIndex, err = decodeSecretIndex(fileResponse.SecretInputFieldsJSON)
		if err != nil {
			return err
		}
	} else {
		source := &krknv1alpha1.KrknGraphRun{}
		if err := h.client.Get(ctx, types.NamespacedName{Name: request.SourceGraphRunName, Namespace: h.namespace}, source); err != nil {
			return fmt.Errorf("source graph run is not available: %w", err)
		}
		claims := auth.GetClaimsFromContext(ctx)
		if claims == nil {
			return fmt.Errorf("authentication required to replay a graph run")
		}
		if !auth.IsAdmin(ctx) {
			allowed, err := h.checkGraphRunGroupAccess(ctx, claims.UserID, source, groupauth.ActionView)
			if err != nil || !allowed {
				return fmt.Errorf("source graph run is not accessible")
			}
		}
		previousGraph = source.Spec.Graph
		var decodeErr error
		previousIndex, decodeErr = decodeSecretIndex(source.Annotations[secretInputFieldsAnnotation])
		if decodeErr != nil {
			return decodeErr
		}
	}
	for nodeID, submitted := range request.Graph {
		previous, exists := previousGraph[nodeID]
		if !exists || len(previous.Env) == 0 {
			continue
		}
		normalizeLegacyGraphScenarioNode(&submitted)
		variables, err := h.secretVariablesForScenario(ctx, submitted.Scenario)
		if err != nil {
			return fmt.Errorf("read secret metadata for node %q: %w", nodeID, err)
		}
		if previousIndex != nil && !reflect.DeepEqual(previousIndex[nodeID], variables) {
			previous.Env = nil
		}
		merged, _ := mergeGraphNodeSecrets(previous, submitted, unionSecretIndexes(workflowSecretIndex{nodeID: previousIndex[nodeID]}, workflowSecretIndex{nodeID: variables})[nodeID])
		request.Graph[nodeID] = merged
	}
	return nil
}

func scenarioExecutionChanged(previous, submitted ScenarioRunRequest) bool {
	previous.TargetRequestID, submitted.TargetRequestID = "", ""
	previous.Environment, submitted.Environment = nil, nil
	previous.SecretFields, submitted.SecretFields = nil, nil
	previous.SourceScenarioRunName, submitted.SourceScenarioRunName = "", ""
	previous.ScenarioImage, submitted.ScenarioImage = "", ""
	previous.CustomRunName, submitted.CustomRunName = "", ""
	previous.Categories, submitted.Categories = nil, nil
	return !reflect.DeepEqual(previous, submitted)
}

func (h *Handler) mergeScenarioRunSourceSecrets(ctx context.Context, request *ScenarioRunRequest) error {
	if request.SourceScenarioRunName == "" {
		return nil
	}
	source, err := h.getScenarioRun(ctx, request.SourceScenarioRunName)
	if err != nil {
		return fmt.Errorf("source scenario run is unavailable: %w", err)
	}
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		return fmt.Errorf("authentication required to replay a scenario run")
	}
	if !auth.IsAdmin(ctx) {
		targetRequest := &krknv1alpha1.KrknTargetRequest{}
		if err := h.client.Get(ctx, client.ObjectKey{Name: source.Spec.TargetRequestID, Namespace: h.namespace}, targetRequest); err != nil {
			return fmt.Errorf("source scenario run is unavailable: %w", err)
		}
		if err := groupauth.ValidateScenarioRunAccess(ctx, h.client, claims.UserID, h.namespace, source.Spec.TargetClusters, targetRequest); err != nil {
			return fmt.Errorf("source scenario run is not accessible")
		}
	}
	previous, err := h.reconstructScenarioRunPayload(ctx, source)
	if err != nil {
		return fmt.Errorf("reconstruct source scenario run: %w", err)
	}
	variables, err := h.secretVariablesForScenario(ctx, request.Scenario)
	if err != nil {
		return fmt.Errorf("read scenario secret metadata: %w", err)
	}
	currentVariables := variables
	var saved map[string]struct{}
	if encoded := source.Annotations[secretInputFieldsAnnotation]; encoded != "" {
		if err := json.Unmarshal([]byte(encoded), &saved); err != nil {
			return fmt.Errorf("invalid source scenario secret metadata: %w", err)
		}
	}
	if saved != nil && !reflect.DeepEqual(saved, currentVariables) {
		previous.Environment = nil
	}
	variables = unionSecretIndexes(workflowSecretIndex{"scenario": saved}, workflowSecretIndex{"scenario": currentVariables})["scenario"]
	request.Environment, _ = mergeSecretEnvironment(previous.Environment, request.Environment, variables, scenarioExecutionChanged(*previous, *request))
	return nil
}

// safeFileResponse closes the alternate /files read path for workflow data.
func (h *Handler) safeFileResponse(ctx context.Context, response files.FileResponse) (files.FileResponse, error) {
	if response.FilePurpose != files.FilePurposeWorkflow {
		return response, nil
	}
	graph, err := workflows.FromFileContent(response.Content)
	if err != nil {
		return files.FileResponse{}, err
	}
	layout, err := workflows.StudioLayoutFromJSON(response.StudioLayout)
	if err != nil {
		return files.FileResponse{}, err
	}
	workflow := workflows.WorkflowResponse{Graph: graph, StudioLayout: layout, SecretInputFieldsJSON: response.SecretInputFieldsJSON}
	if err := h.redactWorkflowResponse(ctx, &workflow); err != nil {
		return files.FileResponse{}, err
	}
	response.Content, err = workflows.ToFileContent(workflow.Graph)
	if err != nil {
		return files.FileResponse{}, err
	}
	response.StudioLayout, err = workflows.StudioLayoutToJSON(workflow.StudioLayout)
	if err != nil {
		return files.FileResponse{}, err
	}
	return response, nil
}

// secretVariablesForScenario reads the authoritative input schema from the
// scenario image metadata. A caller-supplied secret flag is never trusted.
func (h *Handler) secretVariablesForScenario(ctx context.Context, scenario krknv1alpha1.ScenarioReference) (map[string]struct{}, error) {
	if err := scenario.Validate(); err != nil {
		return nil, err
	}
	mode := provider.Mode(provider.Quay)
	var registryConfig *models.RegistryV2
	if *scenario.Private {
		registrySecret, err := h.loadRegistrySecret(ctx, scenario.RegistryName)
		if err != nil {
			return nil, fmt.Errorf("load registry for scenario %q: %w", scenario.Name, err)
		}
		registryConfig, err = registry.ExtractRegistryV2FromSecret(registrySecret)
		if err != nil {
			return nil, fmt.Errorf("decode registry for scenario %q: %w", scenario.Name, err)
		}
		if registryConfig.Token != nil {
			registryConfig = &models.RegistryV2{
				RegistryURL:        registryConfig.RegistryURL,
				ScenarioRepository: registryConfig.ScenarioRepository,
				Token:              registryConfig.Token,
				SkipTLS:            registryConfig.SkipTLS,
				Insecure:           registryConfig.Insecure,
			}
		}
		mode = provider.Private
	}
	providerFactory := h.scenarioProviderFactory
	if providerFactory == nil {
		providerFactory = createScenarioProvider
	}
	scenarioProvider, err := providerFactory(mode)
	if err != nil {
		return nil, err
	}
	detail, err := scenarioProvider.GetScenarioDetail(scenario.Name, registryConfig)
	if err != nil {
		return nil, fmt.Errorf("read input metadata for scenario %q: %w", scenario.Name, err)
	}
	if detail == nil {
		return nil, fmt.Errorf("input metadata for scenario %q is missing", scenario.Name)
	}
	variables := make(map[string]struct{})
	addSecretVariables(variables, detail.Fields)
	globalDetail, err := scenarioProvider.GetGlobalEnvironment(registryConfig, scenario.Name)
	if err != nil && !errors.Is(err, provider.ErrLabelNotFound) {
		return nil, fmt.Errorf("read global input metadata for scenario %q: %w", scenario.Name, err)
	}
	if globalDetail != nil {
		addSecretVariables(variables, globalDetail.Fields)
	}
	return variables, nil
}

func addSecretVariables(destination map[string]struct{}, fields []typing.InputField) {
	for _, field := range fields {
		if field.Secret && field.Variable != nil && *field.Variable != "" {
			destination[*field.Variable] = struct{}{}
		}
	}
}

// redactSecretEnvironment removes secret inputs from a response environment.
// An absent value is represented by false in the returned state map.
func redactSecretEnvironment(environment map[string]string, secretVariables map[string]struct{}) (map[string]string, secretFieldState) {
	visible := make(map[string]string, len(environment))
	state := make(secretFieldState, len(secretVariables))
	for variable := range secretVariables {
		state[variable] = environment[variable] != ""
	}
	for variable, value := range environment {
		if _, secret := secretVariables[variable]; !secret {
			visible[variable] = value
		}
	}
	return visible, state
}

// mergeSecretEnvironment applies a full node/scenario edit. Changes to any
// cleartext execution input invalidate every previously saved secret for that
// node. Secret-only edits preserve sibling secrets; an explicitly empty secret
// deletes only that secret. A newly supplied value is always accepted.
func mergeSecretEnvironment(previous, submitted map[string]string, secretVariables map[string]struct{}, otherExecutionFieldsChanged bool) (map[string]string, bool) {
	previousVisible, _ := redactSecretEnvironment(previous, secretVariables)
	submittedVisible, _ := redactSecretEnvironment(submitted, secretVariables)
	invalidated := otherExecutionFieldsChanged || !reflect.DeepEqual(previousVisible, submittedVisible)

	merged := make(map[string]string, len(submitted)+len(previous))
	for variable, value := range submittedVisible {
		merged[variable] = value
	}
	if !invalidated {
		for variable := range secretVariables {
			if value := previous[variable]; value != "" {
				merged[variable] = value
			}
		}
	}
	for variable := range secretVariables {
		if value, supplied := submitted[variable]; supplied {
			if value == "" {
				delete(merged, variable)
			} else {
				merged[variable] = value
			}
		}
	}
	return merged, invalidated
}

// mergeGraphNodeSecrets compares the complete executable node, excluding its
// secret values. Canvas positions and workflow descriptions are not part of a
// GraphScenarioNode and therefore do not invalidate its secrets.
func mergeGraphNodeSecrets(previous, submitted krknv1alpha1.GraphScenarioNode, secretVariables map[string]struct{}) (krknv1alpha1.GraphScenarioNode, bool) {
	previousVisible := previous
	previousVisible.Env, _ = redactSecretEnvironment(previous.Env, secretVariables)
	submittedVisible := submitted
	submittedVisible.Env, _ = redactSecretEnvironment(submitted.Env, secretVariables)
	otherChanged := !reflect.DeepEqual(previousVisible, submittedVisible)
	submitted.Env, otherChanged = mergeSecretEnvironment(previous.Env, submitted.Env, secretVariables, otherChanged)
	return submitted, otherChanged
}
