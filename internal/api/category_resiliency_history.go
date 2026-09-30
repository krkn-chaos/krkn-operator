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
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

const (
	categoryHistoryScenarioRunType = "scenario-runs"
	categoryHistoryGraphRunType    = "graph-runs"
	categoryHistoryGraphRunLabel   = "krkn.dev/graph-run"
)

// CategoryResiliencyHistoryResponse contains resiliency score datapoints grouped
// by cluster and dynamic behavior-configuration groups.
type CategoryResiliencyHistoryResponse struct {
	Clusters            map[string][]CategoryResiliencyDataPoint `json:"clusters"`
	ConfigurationGroups map[string]CategoryConfigurationGroup    `json:"configurationGroups"`
}

// CategoryResiliencyDataPoint is one cluster's score from a category run.
type CategoryResiliencyDataPoint struct {
	Date    time.Time `json:"date"`
	RunID   string    `json:"runId"`
	RunType string    `json:"runType"`
	Score   float64   `json:"score"`
	// Baseline is the run's minimum acceptable resiliency score, when configured.
	Baseline             *float64 `json:"baseline,omitempty"`
	ConfigurationGroupID string   `json:"configurationGroupId"`
	ProviderName         string   `json:"providerName,omitempty"`
}

// CategoryConfigurationGroup describes a set of runs with equivalent
// behavior-affecting configuration. The map key is the ID placed on datapoints.
type CategoryConfigurationGroup struct {
	RunType             string   `json:"runType"`
	RepresentativeRunID string   `json:"representativeRunId"`
	ScenarioNames       []string `json:"scenarioNames,omitempty"`
	// ParameterProfileFingerprint identifies the unordered multiset of environment key/value pairs.
	ParameterProfileFingerprint string `json:"parameterProfileFingerprint"`
	// ParameterProfileName is a deterministic readable alias for the parameter profile.
	ParameterProfileName string `json:"parameterProfileName"`
}

type categoryHistoryScore struct {
	clusterName  string
	providerName string
	score        float64
	baseline     *float64
}

type categoryHistoryRun struct {
	runType   string
	runID     string
	createdAt time.Time
	scenario  *krknv1alpha1.KrknScenarioRun
	graph     *krknv1alpha1.KrknGraphRun
	scores    []categoryHistoryScore
	groupID   string
}

// GetCategoryResiliencyHistory handles GET /api/v2/categories/{category}/resiliency-history.
//
// @Summary Get category resiliency score history
// @Description Return per-cluster resiliency scores for category-associated scenario and graph runs, grouped by behavior-affecting configuration.
// @Tags categories
// @Produce json
// @Param category path string true "Category name"
// @Success 200 {object} CategoryResiliencyHistoryResponse
// @Failure 400 {object} ErrorResponse "Invalid category name"
// @Failure 401 {object} ErrorResponse "Authentication required"
// @Failure 403 {object} ErrorResponse "The caller cannot access the category"
// @Failure 404 {object} ErrorResponse "Category not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v2/categories/{category}/resiliency-history [get]
func (h *Handler) GetCategoryResiliencyHistory(w http.ResponseWriter, r *http.Request, categoryName string) {
	ctx := r.Context()
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
		return
	}
	if err := validateCategoryName(categoryName); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	logger := log.FromContext(ctx).WithName("category-resiliency-history").WithValues("category", categoryName)
	category := &krknv1alpha1.KrknCategory{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: categoryName, Namespace: h.namespace}, category); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Category %q not found", categoryName),
			})
			return
		}
		logger.Error(err, "Failed to load category")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to load category",
		})
		return
	}

	if !auth.IsAdmin(ctx) {
		visible, err := h.canViewCategory(ctx, category, claims.UserID)
		if err != nil {
			logger.Error(err, "Failed to check category visibility")
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to validate category access",
			})
			return
		}
		if !visible {
			writeJSONError(w, http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have permission to view this category",
			})
			return
		}
	}

	var userGroups []krknv1alpha1.KrknUserGroup
	if !auth.IsAdmin(ctx) {
		groups, err := groupauth.GetUserGroups(ctx, h.client, claims.UserID, h.namespace)
		if err != nil {
			logger.Error(err, "Failed to load cluster visibility permissions")
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to validate cluster visibility",
			})
			return
		}
		userGroups = groups
	}

	history, err := h.loadCategoryResiliencyHistory(ctx, categoryName, userGroups, nil, nil)
	if err != nil {
		logger.Error(err, "Failed to load category resiliency history")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to load category resiliency history",
		})
		return
	}
	writeJSON(w, http.StatusOK, history)
}

// loadCategoryResiliencyHistory returns a category's scored runs. selectedClusters
// is nil for the category-specific endpoint and restricts datapoints for aggregate
// queries. Category and cluster visibility must be checked by the caller first.
func (h *Handler) loadCategoryResiliencyHistory(
	ctx context.Context,
	categoryName string,
	userGroups []krknv1alpha1.KrknUserGroup,
	selectedClusters map[string]struct{},
	selectedProviders map[string]map[string]struct{},
) (CategoryResiliencyHistoryResponse, error) {
	histories, err := h.loadCategoryResiliencyHistories(ctx, []string{categoryName}, userGroups, selectedClusters, selectedProviders)
	if err != nil {
		return CategoryResiliencyHistoryResponse{}, err
	}
	return histories[categoryName], nil
}

// loadCategoryResiliencyHistories loads each run type once for a multi-category
// query, then buckets the scored runs by their category labels. A single-category
// query keeps the narrower label selector used by the existing endpoint.
func (h *Handler) loadCategoryResiliencyHistories(
	ctx context.Context,
	categoryNames []string,
	userGroups []krknv1alpha1.KrknUserGroup,
	selectedClusters map[string]struct{},
	selectedProviders map[string]map[string]struct{},
) (map[string]CategoryResiliencyHistoryResponse, error) {
	categorySet := make(map[string]struct{}, len(categoryNames))
	categoryRuns := make(map[string][]categoryHistoryRun, len(categoryNames))
	for _, categoryName := range categoryNames {
		categorySet[categoryName] = struct{}{}
		categoryRuns[categoryName] = nil
	}

	categoryListOptions := []client.ListOption{client.InNamespace(h.namespace)}
	if len(categoryNames) == 1 {
		categoryListOptions = append(categoryListOptions, client.MatchingLabels{
			krknv1alpha1.CategoryEntityLabelPrefix + categoryNames[0]: "true",
		})
	}

	graphRuns := &krknv1alpha1.KrknGraphRunList{}
	if err := h.client.List(ctx, graphRuns, categoryListOptions...); err != nil {
		return nil, fmt.Errorf("list category graph runs: %w", err)
	}
	categoryGraphRuns := make([]krknv1alpha1.KrknGraphRun, 0, len(graphRuns.Items))
	needsGraphChildRuns := false
	for i := range graphRuns.Items {
		if !runHasAnyCategory(graphRuns.Items[i].Labels, categorySet) {
			continue
		}
		categoryGraphRuns = append(categoryGraphRuns, graphRuns.Items[i])
		if !auth.IsAdmin(ctx) && len(filterCategoryHistoryScoresByCluster(
			graphHistoryScores(&graphRuns.Items[i]), selectedClusters, selectedProviders,
		)) > 0 {
			needsGraphChildRuns = true
		}
	}

	// Graph child runs do not carry category labels. Load them in one additional
	// read only when non-admin graph scores need per-cluster authorization.
	scenarioListOptions := categoryListOptions
	if len(categoryNames) == 1 && needsGraphChildRuns {
		scenarioListOptions = []client.ListOption{client.InNamespace(h.namespace)}
	}
	scenarioRuns := &krknv1alpha1.KrknScenarioRunList{}
	if err := h.client.List(ctx, scenarioRuns, scenarioListOptions...); err != nil {
		return nil, fmt.Errorf("list category scenario runs: %w", err)
	}

	categoryScenarioRuns := make([]krknv1alpha1.KrknScenarioRun, 0, len(scenarioRuns.Items))
	scenarioRunsByName := make(map[string]*krknv1alpha1.KrknScenarioRun, len(scenarioRuns.Items))
	scenarioRunsByGraph := make(map[string][]*krknv1alpha1.KrknScenarioRun)
	for i := range scenarioRuns.Items {
		scenarioRunsByName[scenarioRuns.Items[i].Name] = &scenarioRuns.Items[i]
		if graphRunName := scenarioRuns.Items[i].Labels[categoryHistoryGraphRunLabel]; graphRunName != "" {
			scenarioRunsByGraph[graphRunName] = append(scenarioRunsByGraph[graphRunName], &scenarioRuns.Items[i])
		}
		if runHasAnyCategory(scenarioRuns.Items[i].Labels, categorySet) {
			categoryScenarioRuns = append(categoryScenarioRuns, scenarioRuns.Items[i])
		}
	}
	scenarioRuns.Items = categoryScenarioRuns
	graphRuns.Items = categoryGraphRuns

	for i := range scenarioRuns.Items {
		run := &scenarioRuns.Items[i]
		scores := scenarioHistoryScores(run)
		scores = filterCategoryHistoryScoresByCluster(scores, selectedClusters, selectedProviders)
		if !auth.IsAdmin(ctx) {
			scores = filterScenarioHistoryScoresByPermission(run, scores, userGroups)
		}
		if len(scores) == 0 {
			continue
		}
		historyRun := categoryHistoryRun{
			runType:   categoryHistoryScenarioRunType,
			runID:     run.Name,
			createdAt: run.CreationTimestamp.Time,
			scenario:  run,
			scores:    scores,
		}
		for categoryName := range categorySet {
			if run.Labels[krknv1alpha1.CategoryEntityLabelPrefix+categoryName] == "true" {
				categoryRuns[categoryName] = append(categoryRuns[categoryName], historyRun)
			}
		}
	}
	for i := range graphRuns.Items {
		run := &graphRuns.Items[i]
		scores := graphHistoryScores(run)
		scores = filterCategoryHistoryScoresByCluster(scores, selectedClusters, selectedProviders)
		if !auth.IsAdmin(ctx) && len(scores) > 0 {
			var targetRequest krknv1alpha1.KrknTargetRequest
			targetRequestFound := true
			if err := h.client.Get(ctx, client.ObjectKey{Name: run.Spec.TargetRequestID, Namespace: h.namespace}, &targetRequest); err != nil {
				if !apierrors.IsNotFound(err) {
					return nil, fmt.Errorf("load graph run target details for %s: %w", run.Name, err)
				}
				targetRequestFound = false
			}
			scores = filterGraphHistoryScoresByPermission(run, scores, userGroups, scenarioRunsByName, scenarioRunsByGraph[run.Name], &targetRequest, targetRequestFound)
		}
		if len(scores) == 0 {
			continue
		}
		historyRun := categoryHistoryRun{
			runType:   categoryHistoryGraphRunType,
			runID:     run.Name,
			createdAt: run.CreationTimestamp.Time,
			graph:     run,
			scores:    scores,
		}
		for categoryName := range categorySet {
			if run.Labels[krknv1alpha1.CategoryEntityLabelPrefix+categoryName] == "true" {
				categoryRuns[categoryName] = append(categoryRuns[categoryName], historyRun)
			}
		}
	}

	histories := make(map[string]CategoryResiliencyHistoryResponse, len(categoryNames))
	for _, categoryName := range categoryNames {
		histories[categoryName] = buildCategoryResiliencyHistory(categoryRuns[categoryName])
	}
	return histories, nil
}

func runHasAnyCategory(labels map[string]string, categories map[string]struct{}) bool {
	for categoryName := range categories {
		if labels[krknv1alpha1.CategoryEntityLabelPrefix+categoryName] == "true" {
			return true
		}
	}
	return false
}

func filterCategoryHistoryScoresByCluster(
	scores []categoryHistoryScore,
	selectedClusters map[string]struct{},
	selectedProviders map[string]map[string]struct{},
) []categoryHistoryScore {
	if selectedClusters == nil {
		return scores
	}
	filtered := make([]categoryHistoryScore, 0, len(scores))
	for _, score := range scores {
		if _, selected := selectedClusters[score.clusterName]; !selected {
			continue
		}
		if providers, hasProviderFilter := selectedProviders[score.clusterName]; hasProviderFilter {
			if _, selected := providers[score.providerName]; !selected || score.providerName == "" {
				continue
			}
		}
		filtered = append(filtered, score)
	}
	return filtered
}

func filterScenarioHistoryScoresByPermission(
	run *krknv1alpha1.KrknScenarioRun,
	scores []categoryHistoryScore,
	userGroups []krknv1alpha1.KrknUserGroup,
) []categoryHistoryScore {
	visibleScores := make([]categoryHistoryScore, 0, len(scores))
	for _, score := range scores {
		providerName := score.providerName
		providers := make(map[string]struct{})
		for _, job := range run.Status.ClusterJobs {
			if job.ClusterName != score.clusterName {
				continue
			}
			providers[job.ProviderName] = struct{}{}
		}
		// Older scores lack a provider. Only expose them when their cluster name
		// maps to one provider in the run.
		if providerName == "" {
			if len(providers) != 1 {
				continue
			}
			for name := range providers {
				providerName = name
			}
		}
		foundCluster := false
		clusterVisible := true
		for _, job := range run.Status.ClusterJobs {
			if job.ClusterName != score.clusterName || job.ProviderName != providerName {
				continue
			}
			foundCluster = true
			if job.ClusterAPIURL == "" || !groupauth.CanPerformAction(userGroups, job.ClusterAPIURL, groupauth.ActionView) {
				clusterVisible = false
				break
			}
		}
		if foundCluster && clusterVisible {
			score.providerName = providerName
			visibleScores = append(visibleScores, score)
		}
	}
	return visibleScores
}

func filterGraphHistoryScoresByPermission(
	run *krknv1alpha1.KrknGraphRun,
	scores []categoryHistoryScore,
	userGroups []krknv1alpha1.KrknUserGroup,
	scenarioRunsByName map[string]*krknv1alpha1.KrknScenarioRun,
	graphChildRuns []*krknv1alpha1.KrknScenarioRun,
	targetRequest *krknv1alpha1.KrknTargetRequest,
	targetRequestFound bool,
) []categoryHistoryScore {
	visibleScores := make([]categoryHistoryScore, 0, len(scores))
	childRunSet := make(map[string]*krknv1alpha1.KrknScenarioRun, len(graphChildRuns)+len(run.Status.NodeStatuses))
	for _, childRun := range graphChildRuns {
		childRunSet[childRun.Name] = childRun
	}
	for _, nodeStatus := range run.Status.NodeStatuses {
		if childRun := scenarioRunsByName[nodeStatus.ScenarioRunRef]; childRun != nil {
			childRunSet[childRun.Name] = childRun
		}
	}
	for _, score := range scores {
		providers := make([]string, 0, 1)
		if score.providerName != "" {
			if containsString(run.Spec.TargetClusters[score.providerName], score.clusterName) {
				providers = append(providers, score.providerName)
			}
		} else {
			for providerName, clusterNames := range run.Spec.TargetClusters {
				if containsString(clusterNames, score.clusterName) {
					providers = append(providers, providerName)
				}
			}
		}
		// ProviderName is optional on historical scores. Without it, only a
		// uniquely attributable target can be safely checked and exposed.
		if len(providers) != 1 {
			continue
		}
		providerName := providers[0]
		foundCluster := false
		clusterURLFound := false
		clusterVisible := true
		for _, childRun := range childRunSet {
			for _, job := range childRun.Status.ClusterJobs {
				if job.ProviderName != providerName || job.ClusterName != score.clusterName {
					continue
				}
				foundCluster = true
				if job.ClusterAPIURL == "" {
					continue
				}
				clusterURLFound = true
				if !groupauth.CanPerformAction(userGroups, job.ClusterAPIURL, groupauth.ActionView) {
					clusterVisible = false
					break
				}
			}
			if !clusterVisible {
				break
			}
		}
		if !clusterURLFound && targetRequestFound {
			for _, target := range targetRequest.Status.TargetData[providerName] {
				if target.ClusterName != score.clusterName {
					continue
				}
				foundCluster = true
				if target.ClusterAPIURL == "" || !groupauth.CanPerformAction(userGroups, target.ClusterAPIURL, groupauth.ActionView) {
					clusterVisible = false
					break
				}
			}
		}
		if foundCluster && clusterVisible {
			score.providerName = providerName
			visibleScores = append(visibleScores, score)
		}
	}
	return visibleScores
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func scenarioHistoryScores(run *krknv1alpha1.KrknScenarioRun) []categoryHistoryScore {
	scores := make([]categoryHistoryScore, 0, len(run.Status.ResiliencyScores))
	for _, score := range run.Status.ResiliencyScores {
		if score.Status != "calculated" || score.ClusterName == "" {
			continue
		}
		providerName := score.ProviderName
		providers := make(map[string]struct{})
		for _, job := range run.Status.ClusterJobs {
			if job.ClusterName == score.ClusterName {
				providers[job.ProviderName] = struct{}{}
			}
		}
		if providerName == "" && len(providers) == 1 {
			for name := range providers {
				providerName = name
			}
		}
		scores = append(scores, categoryHistoryScore{
			clusterName:  score.ClusterName,
			providerName: providerName,
			score:        score.Score,
		})
	}
	return scores
}

func graphHistoryScores(run *krknv1alpha1.KrknGraphRun) []categoryHistoryScore {
	scores := make([]categoryHistoryScore, 0, len(run.Status.ResiliencyScores))
	for _, score := range run.Status.ResiliencyScores {
		if score.ClusterName == "" || score.Calculated < 0 {
			continue
		}
		switch score.Status {
		case "pass", "fail", "no-baseline":
		default:
			continue
		}
		baseline := score.Baseline
		if baseline == nil {
			baseline = run.Spec.ResiliencyScoreBaseline
		}
		providerName := score.ProviderName
		if providerName == "" {
			providers := make([]string, 0, 1)
			for candidate, clusterNames := range run.Spec.TargetClusters {
				if containsString(clusterNames, score.ClusterName) {
					providers = append(providers, candidate)
				}
			}
			if len(providers) == 1 {
				providerName = providers[0]
			}
		}
		scores = append(scores, categoryHistoryScore{
			clusterName:  score.ClusterName,
			providerName: providerName,
			score:        score.Calculated,
			baseline:     baseline,
		})
	}
	return scores
}

func buildCategoryResiliencyHistory(runs []categoryHistoryRun) CategoryResiliencyHistoryResponse {
	scoredRuns := make([]categoryHistoryRun, 0, len(runs))
	for _, run := range runs {
		if len(run.scores) > 0 {
			scoredRuns = append(scoredRuns, run)
		}
	}
	runs = scoredRuns

	sort.Slice(runs, func(i, j int) bool {
		if !runs[i].createdAt.Equal(runs[j].createdAt) {
			return runs[i].createdAt.Before(runs[j].createdAt)
		}
		if runs[i].runType != runs[j].runType {
			return runs[i].runType < runs[j].runType
		}
		return runs[i].runID < runs[j].runID
	})

	response := CategoryResiliencyHistoryResponse{
		Clusters:            make(map[string][]CategoryResiliencyDataPoint),
		ConfigurationGroups: make(map[string]CategoryConfigurationGroup),
	}
	profilesByConfiguration := make(map[string]int, len(runs))
	for i := range runs {
		signature, valid := categoryBehaviorConfigurationSignature(runs[i])
		profileIndex, matched := profilesByConfiguration[signature]
		if !valid {
			matched = false
		}
		if matched {
			runs[i].groupID = runs[profileIndex].groupID
		} else {
			runs[i].groupID = fmt.Sprintf("%s/%s", runs[i].runType, runs[i].runID)
			if valid {
				profilesByConfiguration[signature] = i
			}
			parameterProfileFingerprint := categoryParameterProfileFingerprint(runs[i])
			response.ConfigurationGroups[runs[i].groupID] = CategoryConfigurationGroup{
				RunType:                     runs[i].runType,
				RepresentativeRunID:         runs[i].runID,
				ScenarioNames:               categoryRunScenarioNames(runs[i]),
				ParameterProfileFingerprint: parameterProfileFingerprint,
				ParameterProfileName:        categoryParameterProfileName(parameterProfileFingerprint),
			}
		}

		for _, score := range runs[i].scores {
			if score.clusterName == "" {
				continue
			}
			response.Clusters[score.clusterName] = append(response.Clusters[score.clusterName], CategoryResiliencyDataPoint{
				Date:                 runs[i].createdAt,
				RunID:                runs[i].runID,
				RunType:              runs[i].runType,
				Score:                score.score,
				Baseline:             score.baseline,
				ConfigurationGroupID: runs[i].groupID,
				ProviderName:         score.providerName,
			})
		}
	}

	for clusterName := range response.Clusters {
		sort.Slice(response.Clusters[clusterName], func(i, j int) bool {
			left, right := response.Clusters[clusterName][i], response.Clusters[clusterName][j]
			if !left.Date.Equal(right.Date) {
				return left.Date.Before(right.Date)
			}
			if left.RunType != right.RunType {
				return left.RunType < right.RunType
			}
			return left.RunID < right.RunID
		})
	}
	return response
}

func categoryBehaviorConfigurationSignature(run categoryHistoryRun) (string, bool) {
	switch {
	case run.scenario != nil:
		spec := run.scenario.Spec
		scenarioReference := spec.Scenario
		scenarioReference.Name = scenarioIdentity(spec.Scenario.Name, spec.ScenarioName)
		files := make([]scenarioFileBehavior, 0, len(spec.Files))
		for _, file := range spec.Files {
			files = append(files, scenarioFileBehavior{name: file.Name, content: file.Content, mountPath: file.MountPath})
		}
		sort.Slice(files, func(i, j int) bool {
			if files[i].name != files[j].name {
				return files[i].name < files[j].name
			}
			if files[i].content != files[j].content {
				return files[i].content < files[j].content
			}
			return files[i].mountPath < files[j].mountPath
		})
		configuration := struct {
			RunType         string
			ScenarioName    string
			Scenario        krknv1alpha1.ScenarioReference
			KubeconfigPath  string
			MaxRetries      int
			RetryBackoff    string
			RetryDelay      string
			CloudCredential string
			Environment     map[string]string
			Files           []scenarioFileBehavior
		}{
			RunType:         run.runType,
			ScenarioName:    scenarioIdentity(spec.Scenario.Name, spec.ScenarioName),
			Scenario:        scenarioReference,
			KubeconfigPath:  spec.KubeconfigPath,
			MaxRetries:      spec.MaxRetries,
			RetryBackoff:    spec.RetryBackoff,
			RetryDelay:      spec.RetryDelay,
			CloudCredential: spec.CloudCredentialRef,
			Environment:     spec.Environment,
			Files:           files,
		}
		encoded, err := json.Marshal(configuration)
		return string(encoded), err == nil
	case run.graph != nil:
		graphSignature, valid := graphConfigurationSignature(run.graph.Spec.Graph, run.graph.Spec.ResiliencyMountPath)
		if !valid {
			return "", false
		}
		configuration := struct {
			RunType         string
			Graph           string
			MaxRetries      int
			CloudCredential string
		}{
			RunType:         run.runType,
			Graph:           graphSignature,
			MaxRetries:      run.graph.Spec.MaxRetries,
			CloudCredential: run.graph.Spec.CloudCredentialRef,
		}
		encoded, err := json.Marshal(configuration)
		return string(encoded), err == nil
	default:
		return "", false
	}
}

func categoryRunScenarioNames(run categoryHistoryRun) []string {
	names := make([]string, 0)
	if run.scenario != nil {
		if name := scenarioIdentity(run.scenario.Spec.Scenario.Name, run.scenario.Spec.ScenarioName); name != "" {
			names = append(names, name)
		}
	}
	if run.graph != nil {
		for _, node := range run.graph.Spec.Graph {
			if name := scenarioIdentity(node.Scenario.Name, node.Name); name != "" {
				names = append(names, name)
			}
		}
		sort.Strings(names)
	}
	return names
}
