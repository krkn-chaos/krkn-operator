// Package crdmigration migrates stored Krkn custom resources around CRD schema updates.
package crdmigration

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

const (
	AnnotationKey      = "migration.krkn-chaos.dev/api-v1"
	GuardConfigMapName = "krkn-operator-crd-migration"
	pageSize           = int64(500)
	maxAnnotationsSize = 256 * 1024
)

var (
	resources = []resourceDefinition{
		{kind: "KrknScenarioRun", gvr: schema.GroupVersionResource{Group: "krkn.krkn-chaos.dev", Version: "v1alpha1", Resource: "krknscenarioruns"}},
		{kind: "KrknGraphRun", gvr: schema.GroupVersionResource{Group: "krkn.krkn-chaos.dev", Version: "v1alpha1", Resource: "krkngraphruns"}},
	}
)

type resourceDefinition struct {
	kind string
	gvr  schema.GroupVersionResource
}

type migrationPlan struct {
	Version            int                     `json:"version"`
	Kind               string                  `json:"kind"`
	Fields             []string                `json:"fields,omitempty"`
	Warnings           []string                `json:"warnings,omitempty"`
	Scenario           map[string]interface{}  `json:"scenario,omitempty"`
	GraphScenarios     map[string]interface{}  `json:"graphScenarios,omitempty"`
	SpecDefaults       map[string]interface{}  `json:"specDefaults,omitempty"`
	RemoveSpecFields   []string                `json:"removeSpecFields,omitempty"`
	ClusterJobImages   []clusterJobImage       `json:"clusterJobImages,omitempty"`
	ClusterJobRetries  []clusterJobRetries     `json:"clusterJobRetries,omitempty"`
	ResiliencyStatuses []resiliencyScoreStatus `json:"resiliencyStatuses,omitempty"`
}

type clusterJobImage struct {
	Index       int    `json:"index"`
	ClusterName string `json:"clusterName"`
	JobID       string `json:"jobId"`
	Image       string `json:"image"`
}

type clusterJobRetries struct {
	Index       int    `json:"index"`
	ClusterName string `json:"clusterName"`
	JobID       string `json:"jobId"`
	MaxRetries  int64  `json:"maxRetries"`
}

type resiliencyScoreStatus struct {
	Index        int    `json:"index"`
	ClusterName  string `json:"clusterName"`
	Status       string `json:"status"`
	ProviderName string `json:"providerName,omitempty"`
}

type migrationResult struct {
	changed      bool
	warningCount int
}

type migrationStats struct {
	scanned             int
	changed             int
	unchanged           int
	objectsWithWarnings int
	warnings            int
}

type graphScenarioReferences map[string]map[string]interface{}

// Stage snapshots all legacy values needed by the migration into annotations,
// which remain valid across the old and new CRD schemas.
func Stage(ctx context.Context, client dynamic.Interface, namespace string, logger logr.Logger) error {
	graphReferences, err := loadGraphScenarioReferences(ctx, client, namespace, logger, "stage")
	if err != nil {
		return err
	}
	return forEachResource(ctx, client, namespace, logger, "stage", func(resource resourceDefinition, item *unstructured.Unstructured) (migrationResult, error) {
		annotations := item.GetAnnotations()
		plan := planResourceWithGraphReferences(resource.kind, item, graphReferences)
		previousEncoded := annotations[AnnotationKey]
		if previousEncoded != "" {
			previousPlan := migrationPlan{}
			if err := json.Unmarshal([]byte(previousEncoded), &previousPlan); err != nil {
				return migrationResult{}, fmt.Errorf("decode staged migration for %s/%s: %w", item.GetNamespace(), item.GetName(), err)
			}
			plan = mergeMigrationPlans(previousPlan, plan)
		}
		if len(plan.Fields) == 0 && len(plan.Warnings) == 0 {
			if previousEncoded != "" {
				logger.Info("custom resource migration remains staged", "kind", resource.kind, "namespace", item.GetNamespace(), "name", item.GetName())
				return migrationResult{}, nil
			}
			logger.Info("custom resource requires no migration", "kind", resource.kind, "namespace", item.GetNamespace(), "name", item.GetName())
			return migrationResult{}, nil
		}
		encoded, err := json.Marshal(plan)
		if err != nil {
			return migrationResult{}, fmt.Errorf("encode migration for %s/%s: %w", item.GetNamespace(), item.GetName(), err)
		}
		if err := checkAnnotationSize(item, string(encoded)); err != nil {
			return migrationResult{}, err
		}
		if string(encoded) == previousEncoded {
			logger.Info("custom resource migration snapshot is current", "kind", resource.kind, "namespace", item.GetNamespace(), "name", item.GetName())
			return migrationResult{}, nil
		}
		patch, err := json.Marshal(map[string]interface{}{"metadata": map[string]interface{}{"annotations": map[string]string{AnnotationKey: string(encoded)}}})
		if err != nil {
			return migrationResult{}, fmt.Errorf("encode migration annotation for %s/%s: %w", item.GetNamespace(), item.GetName(), err)
		}
		if _, err := client.Resource(resource.gvr).Namespace(item.GetNamespace()).Patch(ctx, item.GetName(), types.MergePatchType, patch, metav1.PatchOptions{}); err != nil {
			return migrationResult{}, fmt.Errorf("stage migration for %s/%s: %w", item.GetNamespace(), item.GetName(), err)
		}
		message := "staged custom resource migration"
		if previousEncoded != "" {
			message = "refreshed staged custom resource migration"
		}
		logger.Info(message, "kind", resource.kind, "namespace", item.GetNamespace(), "name", item.GetName(), "fields", plan.Fields, "warnings", plan.Warnings)
		return migrationResult{changed: true, warningCount: len(plan.Warnings)}, nil
	})
}

func mergeMigrationPlans(previous, current migrationPlan) migrationPlan {
	merged := previous
	merged.Version = current.Version
	merged.Kind = current.Kind
	if current.Scenario != nil {
		merged.Scenario = current.Scenario
	}
	if current.GraphScenarios != nil {
		if merged.GraphScenarios == nil {
			merged.GraphScenarios = map[string]interface{}{}
		}
		for nodeID, reference := range current.GraphScenarios {
			merged.GraphScenarios[nodeID] = reference
		}
	}
	if current.SpecDefaults != nil {
		if merged.SpecDefaults == nil {
			merged.SpecDefaults = map[string]interface{}{}
		}
		for field, value := range current.SpecDefaults {
			merged.SpecDefaults[field] = value
		}
	}
	merged.Fields = uniqueSortedStrings(append(merged.Fields, current.Fields...))
	merged.Warnings = uniqueSortedStrings(append(merged.Warnings, current.Warnings...))
	merged.RemoveSpecFields = uniqueSortedStrings(append(merged.RemoveSpecFields, current.RemoveSpecFields...))
	merged.ClusterJobImages = mergeClusterJobImages(merged.ClusterJobImages, current.ClusterJobImages)
	merged.ClusterJobRetries = mergeClusterJobRetries(merged.ClusterJobRetries, current.ClusterJobRetries)
	merged.ResiliencyStatuses = mergeResiliencyStatuses(merged.ResiliencyStatuses, current.ResiliencyStatuses)
	if merged.Scenario != nil {
		merged.Warnings = removeWarning(merged.Warnings, "scenario identity is absent; spec.scenario.name cannot be reconstructed")
		if merged.Scenario["registryName"] != nil {
			merged.Warnings = removeWarning(merged.Warnings, "inline registry credentials have no registryName and cannot be transferred to the current API")
			merged.Warnings = removeWarning(merged.Warnings, "private scenario has no registryName; a saved registry reference cannot be reconstructed")
		}
	}
	if containsString(merged.Fields, "KrknGraphRun.spec.graph node -> spec.scenario") {
		merged.Fields = removeString(merged.Fields, "spec.scenarioName -> spec.scenario.name")
		merged.Fields = removeString(merged.Fields, "legacy registry identity -> spec.scenario.private")
		merged.Fields = removeString(merged.Fields, "spec.scenario")
		merged.Warnings = removeWarning(merged.Warnings, "existing public scenario reference conflicts with a legacy registryName")
	}
	return merged
}

func mergeClusterJobImages(previous, current []clusterJobImage) []clusterJobImage {
	merged := map[string]clusterJobImage{}
	for _, item := range append(previous, current...) {
		key := fmt.Sprintf("%d\x00%s\x00%s", item.Index, item.ClusterName, item.JobID)
		merged[key] = item
	}
	result := make([]clusterJobImage, 0, len(merged))
	for _, item := range merged {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Index < result[j].Index })
	return result
}

func mergeClusterJobRetries(previous, current []clusterJobRetries) []clusterJobRetries {
	merged := map[string]clusterJobRetries{}
	for _, item := range append(previous, current...) {
		key := fmt.Sprintf("%d\x00%s\x00%s", item.Index, item.ClusterName, item.JobID)
		merged[key] = item
	}
	result := make([]clusterJobRetries, 0, len(merged))
	for _, item := range merged {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Index < result[j].Index })
	return result
}

func mergeResiliencyStatuses(previous, current []resiliencyScoreStatus) []resiliencyScoreStatus {
	merged := map[string]resiliencyScoreStatus{}
	for _, item := range append(previous, current...) {
		key := fmt.Sprintf("%d\x00%s", item.Index, item.ClusterName)
		merged[key] = item
	}
	result := make([]resiliencyScoreStatus, 0, len(merged))
	for _, item := range merged {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Index < result[j].Index })
	return result
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	unique := values[:0]
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			unique = append(unique, value)
		}
	}
	sort.Strings(unique)
	return unique
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// Complete applies staged values after all target CRD schemas are Established.
// It also detects old fields retained by objects from an earlier partial upgrade.
func Complete(ctx context.Context, client dynamic.Interface, namespace string, logger logr.Logger) error {
	graphReferences, err := loadGraphScenarioReferences(ctx, client, namespace, logger, "complete")
	if err != nil {
		return err
	}
	return forEachResource(ctx, client, namespace, logger, "complete", func(resource resourceDefinition, item *unstructured.Unstructured) (migrationResult, error) {
		plan := migrationPlan{}
		annotations := item.GetAnnotations()
		if encoded := annotations[AnnotationKey]; encoded != "" {
			if err := json.Unmarshal([]byte(encoded), &plan); err != nil {
				return migrationResult{}, fmt.Errorf("decode staged migration for %s/%s: %w", item.GetNamespace(), item.GetName(), err)
			}
		} else {
			plan = planResourceWithGraphReferences(resource.kind, item, graphReferences)
		}
		if annotations[AnnotationKey] != "" && resource.kind == "KrknScenarioRun" {
			applyGraphScenarioReference(&plan, item, graphReferences)
		}
		if plan.Version != 1 || plan.Kind != resource.kind {
			return migrationResult{}, fmt.Errorf("unsupported migration payload for %s/%s (version %d, kind %q)", item.GetNamespace(), item.GetName(), plan.Version, plan.Kind)
		}
		if len(plan.Fields) == 0 && len(plan.Warnings) == 0 && annotations[AnnotationKey] == "" {
			logger.Info("custom resource already uses the current API", "kind", resource.kind, "namespace", item.GetNamespace(), "name", item.GetName())
			return migrationResult{}, nil
		}

		statusPatch := buildStatusPatch(item, &plan)
		if len(statusPatch) > 0 {
			encodedStatus, err := json.Marshal(map[string]interface{}{
				"metadata": map[string]interface{}{"resourceVersion": item.GetResourceVersion()},
				"status":   statusPatch,
			})
			if err != nil {
				return migrationResult{}, fmt.Errorf("encode status migration for %s/%s: %w", item.GetNamespace(), item.GetName(), err)
			}
			updated, err := client.Resource(resource.gvr).Namespace(item.GetNamespace()).Patch(ctx, item.GetName(), types.MergePatchType, encodedStatus, metav1.PatchOptions{}, "status")
			if err != nil {
				return migrationResult{}, fmt.Errorf("migrate status for %s/%s: %w", item.GetNamespace(), item.GetName(), err)
			}
			if updated != nil {
				item = updated
			}
		}

		rootPatch := buildRootPatch(item, &plan)
		if annotations[AnnotationKey] != "" {
			metadataPatch, _ := rootPatch["metadata"].(map[string]interface{})
			if metadataPatch == nil {
				metadataPatch = map[string]interface{}{}
				rootPatch["metadata"] = metadataPatch
			}
			metadataPatch["annotations"] = map[string]interface{}{AnnotationKey: nil}
		}
		if len(rootPatch) > 0 {
			encodedRoot, err := json.Marshal(rootPatch)
			if err != nil {
				return migrationResult{}, fmt.Errorf("encode spec migration for %s/%s: %w", item.GetNamespace(), item.GetName(), err)
			}
			if _, err := client.Resource(resource.gvr).Namespace(item.GetNamespace()).Patch(ctx, item.GetName(), types.MergePatchType, encodedRoot, metav1.PatchOptions{}); err != nil {
				return migrationResult{}, fmt.Errorf("migrate spec for %s/%s: %w", item.GetNamespace(), item.GetName(), err)
			}
		}

		logger.Info("completed custom resource migration", "kind", resource.kind, "namespace", item.GetNamespace(), "name", item.GetName(), "fields", plan.Fields, "warnings", plan.Warnings)
		return migrationResult{changed: true, warningCount: len(plan.Warnings)}, nil
	})
}

func forEachResource(ctx context.Context, client dynamic.Interface, namespace string, logger logr.Logger, phase string, migrate func(resourceDefinition, *unstructured.Unstructured) (migrationResult, error)) error {
	for _, resource := range resources {
		stats := migrationStats{}
		continueToken := ""
		skipped := false
		for {
			list, err := client.Resource(resource.gvr).Namespace(namespace).List(ctx, metav1.ListOptions{Limit: pageSize, Continue: continueToken})
			if apierrors.IsNotFound(err) {
				logger.Info("custom resource kind is not installed; migration skipped", "phase", phase, "kind", resource.kind, "namespace", namespace)
				skipped = true
				break
			}
			if err != nil {
				wrapped := fmt.Errorf("list %s resources in namespace %q for migration: %w", resource.kind, namespace, err)
				logger.Error(wrapped, "custom resource migration failed", "phase", phase, "kind", resource.kind, "namespace", namespace)
				logSummary(logger, phase, resource.kind, namespace, stats, 1, false)
				return wrapped
			}
			for index := range list.Items {
				item := &list.Items[index]
				stats.scanned++
				result, err := migrate(resource, item)
				if err != nil {
					logger.Error(err, "custom resource migration failed", "phase", phase, "kind", resource.kind, "namespace", item.GetNamespace(), "name", item.GetName())
					logSummary(logger, phase, resource.kind, namespace, stats, 1, false)
					return err
				}
				if result.changed {
					stats.changed++
				} else {
					stats.unchanged++
				}
				if result.warningCount > 0 {
					stats.objectsWithWarnings++
					stats.warnings += result.warningCount
				}
			}
			continueToken = list.GetContinue()
			if continueToken == "" {
				break
			}
		}
		logSummary(logger, phase, resource.kind, namespace, stats, 0, skipped)
	}
	return nil
}

func logSummary(logger logr.Logger, phase, kind, namespace string, stats migrationStats, failed int, skipped bool) {
	logger.Info("custom resource migration phase summary",
		"phase", phase,
		"kind", kind,
		"namespace", namespace,
		"scanned", stats.scanned,
		"changed", stats.changed,
		"unchanged", stats.unchanged,
		"objectsWithWarnings", stats.objectsWithWarnings,
		"warnings", stats.warnings,
		"failed", failed,
		"skipped", skipped,
	)
}

func checkAnnotationSize(item *unstructured.Unstructured, migrationValue string) error {
	annotationBytes := len(AnnotationKey) + len(migrationValue)
	for key, value := range item.GetAnnotations() {
		if key == AnnotationKey {
			continue
		}
		annotationBytes += len(key) + len(value)
	}
	if annotationBytes > maxAnnotationsSize {
		return fmt.Errorf("migration snapshot for %s/%s exceeds Kubernetes annotation size limit (%d bytes)", item.GetNamespace(), item.GetName(), maxAnnotationsSize)
	}
	return nil
}

// loadGraphScenarioReferences indexes the authoritative scenario identity for
// each graph node referenced by a child ScenarioRun's labels.
func loadGraphScenarioReferences(ctx context.Context, client dynamic.Interface, namespace string, logger logr.Logger, phase string) (graphScenarioReferences, error) {
	references := graphScenarioReferences{}
	ambiguous := map[string]bool{}
	continueToken := ""
	for {
		list, err := client.Resource(resources[1].gvr).Namespace(namespace).List(ctx, metav1.ListOptions{Limit: pageSize, Continue: continueToken})
		if apierrors.IsNotFound(err) {
			logger.Info("graph-run CRD is not installed; graph-child scenario lookup unavailable", "phase", phase, "namespace", namespace)
			return references, nil
		}
		if err != nil {
			return nil, fmt.Errorf("list KrknGraphRun resources in namespace %q for graph-child scenario migration: %w", namespace, err)
		}
		for index := range list.Items {
			graphRun := &list.Items[index]
			graph, found, _ := unstructured.NestedMap(graphRun.Object, "spec", "graph")
			if !found {
				continue
			}
			for nodeID, rawNode := range graph {
				node, _ := rawNode.(map[string]interface{})
				ref := graphNodeScenarioReference(node)
				if ref == nil {
					continue
				}
				key := graphReferenceKey(graphRun.GetName(), sanitizeGraphNodeID(nodeID))
				if _, exists := references[key]; exists {
					delete(references, key)
					ambiguous[key] = true
					logger.Info("graph node labels are ambiguous; scenario identity lookup skipped", "phase", phase, "namespace", namespace, "graphRun", graphRun.GetName(), "nodeLabel", sanitizeGraphNodeID(nodeID))
					continue
				}
				if !ambiguous[key] {
					references[key] = ref
				}
			}
		}
		continueToken = list.GetContinue()
		if continueToken == "" {
			break
		}
	}
	logger.Info("loaded graph node scenario identities", "phase", phase, "namespace", namespace, "references", len(references), "ambiguous", len(ambiguous))
	return references, nil
}

func graphNodeScenarioReference(node map[string]interface{}) map[string]interface{} {
	current, _ := node["scenario"].(map[string]interface{})
	name, _ := current["name"].(string)
	if name == "" {
		name, _ = node["name"].(string)
	}
	if name == "" {
		return nil
	}
	registryName, _ := current["registryName"].(string)
	if registryName == "" {
		registryName, _ = node["registryName"].(string)
	}
	private, found := current["private"].(bool)
	if !found {
		private = registryName != ""
	}
	ref := map[string]interface{}{"name": name, "private": private}
	if private && registryName != "" {
		ref["registryName"] = registryName
	}
	return ref
}

func graphReferenceKey(graphRunName, nodeLabel string) string {
	return graphRunName + "\x00" + nodeLabel
}

func sanitizeGraphNodeID(nodeID string) string {
	if nodeID == "" {
		return "empty"
	}
	var builder strings.Builder
	builder.Grow(len(nodeID))
	for _, char := range strings.ToLower(nodeID) {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '-' || char == '_' || char == '.' {
			builder.WriteRune(char)
		} else {
			builder.WriteRune('-')
		}
	}
	sanitized := builder.String()
	if len(sanitized) > 63 {
		sanitized = sanitized[:63]
	}
	sanitized = strings.TrimLeft(sanitized, "-_.")
	if sanitized == "" {
		return "node"
	}
	sanitized = strings.TrimRight(sanitized, "-_.")
	if sanitized == "" {
		return "node"
	}
	return sanitized
}

func planResourceWithGraphReferences(kind string, item *unstructured.Unstructured, graphReferences graphScenarioReferences) migrationPlan {
	plan := planResource(kind, item)
	if kind != "KrknScenarioRun" {
		return plan
	}
	applyGraphScenarioReference(&plan, item, graphReferences)
	return plan
}

func applyGraphScenarioReference(plan *migrationPlan, item *unstructured.Unstructured, graphReferences graphScenarioReferences) {
	labels := item.GetLabels()
	graphRunName := labels["krkn.dev/graph-run"]
	nodeLabel := labels["krkn.dev/graph-node"]
	if graphRunName == "" || nodeLabel == "" {
		return
	}
	if reference := graphReferences[graphReferenceKey(graphRunName, nodeLabel)]; reference != nil {
		plan.Scenario = reference
		plan.Fields = removeString(plan.Fields, "spec.scenarioName -> spec.scenario.name")
		plan.Fields = removeString(plan.Fields, "legacy registry identity -> spec.scenario.private")
		plan.Fields = removeString(plan.Fields, "spec.scenario")
		plan.Fields = append(plan.Fields, "KrknGraphRun.spec.graph node -> spec.scenario")
		plan.Warnings = removeWarning(plan.Warnings, "scenario identity is absent; spec.scenario.name cannot be reconstructed")
		plan.Warnings = removeWarning(plan.Warnings, "private scenario has no registryName; a saved registry reference cannot be reconstructed")
		plan.Warnings = removeWarning(plan.Warnings, "existing public scenario reference conflicts with a legacy registryName")
		if reference["registryName"] != nil {
			plan.Warnings = removeWarning(plan.Warnings, "inline registry credentials have no registryName and cannot be transferred to the current API")
		}
		if reference["private"] == true && reference["registryName"] == nil {
			plan.Warnings = append(plan.Warnings, "private scenario has no registryName; a saved registry reference cannot be reconstructed")
		}
		sort.Strings(plan.Fields)
		sort.Strings(plan.Warnings)
	}
}

func removeWarning(warnings []string, target string) []string {
	return removeString(warnings, target)
}

func removeString(values []string, target string) []string {
	filtered := values[:0]
	for _, value := range values {
		if value != target {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func planResource(kind string, item *unstructured.Unstructured) migrationPlan {
	plan := migrationPlan{Version: 1, Kind: kind}
	spec, _, _ := unstructured.NestedMap(item.Object, "spec")
	status, _, _ := unstructured.NestedMap(item.Object, "status")
	switch kind {
	case "KrknScenarioRun":
		planScenarioRun(&plan, spec, status)
	case "KrknGraphRun":
		planGraphRun(&plan, spec)
	default:
		plan.Warnings = append(plan.Warnings, "no migration registered for resource kind")
	}
	sort.Strings(plan.Fields)
	sort.Strings(plan.Warnings)
	return plan
}

func planScenarioRun(plan *migrationPlan, spec, status map[string]interface{}) {
	current, _ := spec["scenario"].(map[string]interface{})
	name, _ := current["name"].(string)
	registryName, _ := current["registryName"].(string)
	if registryName == "" {
		registryName, _ = spec["registryName"].(string)
	}
	inlineCredentials := hasInlineRegistryCredentials(spec)
	if name == "" {
		name, _ = spec["scenarioName"].(string)
		if name != "" {
			plan.Fields = append(plan.Fields, "spec.scenarioName -> spec.scenario.name")
		}
	}
	if inlineCredentials && registryName == "" {
		plan.Warnings = append(plan.Warnings, "inline registry credentials have no registryName and cannot be transferred to the current API")
	}
	if name == "" {
		plan.Warnings = append(plan.Warnings, "scenario identity is absent; spec.scenario.name cannot be reconstructed")
	} else {
		private, hasPrivate := current["private"].(bool)
		if !hasPrivate {
			private = registryName != "" || inlineCredentials
			plan.Fields = append(plan.Fields, "legacy registry identity -> spec.scenario.private")
		}
		if private && registryName == "" {
			plan.Warnings = append(plan.Warnings, "private scenario has no registryName; a saved registry reference cannot be reconstructed")
		}
		if !private && registryName != "" {
			plan.Warnings = append(plan.Warnings, "existing public scenario reference conflicts with a legacy registryName")
		}
		ref := map[string]interface{}{"name": name, "private": private}
		if private && registryName != "" {
			ref["registryName"] = registryName
		}
		if len(current) == 0 || current["name"] != ref["name"] || current["private"] != ref["private"] || current["registryName"] != ref["registryName"] {
			plan.Scenario = ref
			if len(current) == 0 {
				plan.Fields = append(plan.Fields, "spec.scenario")
			}
		}
	}

	defaults := map[string]interface{}{}
	if _, exists := spec["maxRetries"]; !exists {
		defaults["maxRetries"] = int64(3)
		plan.Fields = append(plan.Fields, "spec.maxRetries defaulted to 3")
	}
	if _, exists := spec["retryBackoff"]; !exists {
		defaults["retryBackoff"] = "exponential"
		plan.Fields = append(plan.Fields, "spec.retryBackoff defaulted to exponential")
	}
	if _, exists := spec["retryDelay"]; !exists {
		defaults["retryDelay"] = "10s"
		plan.Fields = append(plan.Fields, "spec.retryDelay defaulted to 10s")
	}
	if len(defaults) > 0 {
		plan.SpecDefaults = defaults
	}

	oldSpecFields := []string{"scenarioName", "scenarioImage", "registryURL", "scenarioRepository", "token", "username", "password", "registryName"}
	for _, field := range oldSpecFields {
		if _, exists := spec[field]; exists {
			plan.RemoveSpecFields = append(plan.RemoveSpecFields, field)
			if field == "token" || field == "username" || field == "password" {
				plan.Fields = append(plan.Fields, "legacy registry credentials removed")
			} else {
				plan.Fields = append(plan.Fields, "spec."+field+" removed after migration")
			}
		}
	}
	sort.Strings(plan.RemoveSpecFields)

	jobs, _, _ := unstructured.NestedSlice(status, "clusterJobs")
	for index, rawJob := range jobs {
		job, _ := rawJob.(map[string]interface{})
		image, _ := job["scenarioImage"].(string)
		legacyImage, _ := job["containerImage"].(string)
		clusterName, _ := job["clusterName"].(string)
		jobID, _ := job["jobId"].(string)
		if image == "" && legacyImage != "" {
			plan.ClusterJobImages = append(plan.ClusterJobImages, clusterJobImage{Index: index, ClusterName: clusterName, JobID: jobID, Image: legacyImage})
			plan.Fields = append(plan.Fields, "status.clusterJobs[].containerImage -> scenarioImage")
		}
		if podName, _ := job["podName"].(string); podName != "" {
			configured, _ := job["maxRetriesConfigured"].(bool)
			if !configured {
				maxRetries, found, _ := unstructured.NestedInt64(job, "maxRetries")
				if !found || maxRetries == 0 {
					if specRetries, specFound, _ := unstructured.NestedInt64(spec, "maxRetries"); specFound {
						maxRetries, found = specRetries, true
					}
				}
				if !found {
					maxRetries = 3
				}
				plan.ClusterJobRetries = append(plan.ClusterJobRetries, clusterJobRetries{Index: index, ClusterName: clusterName, JobID: jobID, MaxRetries: maxRetries})
				plan.Fields = append(plan.Fields, "status.clusterJobs[].maxRetriesConfigured backfilled")
			}
		}
	}

	jobProviders := make(map[string]string, len(jobs))
	for _, rawJob := range jobs {
		job, _ := rawJob.(map[string]interface{})
		clusterName, _ := job["clusterName"].(string)
		providerName, _ := job["providerName"].(string)
		if clusterName != "" && providerName != "" {
			jobProviders[clusterName] = providerName
		}
	}
	scores, _, _ := unstructured.NestedSlice(status, "resiliencyScores")
	for index, rawScore := range scores {
		score, _ := rawScore.(map[string]interface{})
		clusterName, _ := score["clusterName"].(string)
		statusValue, _ := score["status"].(string)
		providerName, _ := score["providerName"].(string)
		if providerName == "" {
			providerName = jobProviders[clusterName]
		}
		if statusValue == "" {
			statusValue = "calculated"
			plan.Fields = append(plan.Fields, "status.resiliencyScores[].status defaulted to calculated")
		}
		if score["providerName"] == nil && providerName != "" {
			plan.Fields = append(plan.Fields, "status.clusterJobs[].providerName -> resiliencyScores[].providerName")
		}
		if score["status"] == nil || score["status"] == "" || score["providerName"] == nil && providerName != "" {
			plan.ResiliencyStatuses = append(plan.ResiliencyStatuses, resiliencyScoreStatus{Index: index, ClusterName: clusterName, Status: statusValue, ProviderName: providerName})
		}
	}
	if len(plan.ClusterJobImages) > 0 {
		plan.Fields = append(plan.Fields, "legacy cluster job images preserved")
	}
	if len(plan.ClusterJobRetries) > 0 {
		plan.Fields = append(plan.Fields, "legacy retry settings preserved")
	}
}

func planGraphRun(plan *migrationPlan, spec map[string]interface{}) {
	graph, ok, _ := unstructured.NestedMap(spec, "graph")
	if ok {
		migrated := map[string]interface{}{}
		for nodeID, rawNode := range graph {
			node, _ := rawNode.(map[string]interface{})
			current, _ := node["scenario"].(map[string]interface{})
			name, _ := current["name"].(string)
			if name == "" {
				name, _ = node["name"].(string)
				if name != "" {
					plan.Fields = append(plan.Fields, "spec.graph."+nodeID+".name -> scenario.name")
				}
			}
			if name == "" && strings.HasPrefix(nodeID, "_") {
				continue
			}
			if name == "" {
				plan.Warnings = append(plan.Warnings, "graph node "+nodeID+" has no scenario identity")
				continue
			}
			private, hasPrivate := current["private"].(bool)
			registryName, _ := current["registryName"].(string)
			if registryName == "" {
				registryName, _ = node["registryName"].(string)
			}
			if !hasPrivate {
				private = registryName != ""
			}
			if private && registryName == "" {
				plan.Warnings = append(plan.Warnings, "graph node "+nodeID+" is private but has no registryName")
			}
			if !private && registryName != "" {
				plan.Warnings = append(plan.Warnings, "graph node "+nodeID+" has a public scenario reference and a legacy registryName")
			}
			ref := map[string]interface{}{"name": name, "private": private}
			if private && registryName != "" {
				ref["registryName"] = registryName
			}
			if len(current) == 0 || current["name"] != ref["name"] || current["private"] != ref["private"] || current["registryName"] != ref["registryName"] {
				migrated[nodeID] = ref
				plan.Fields = append(plan.Fields, "spec.graph."+nodeID+".scenario backfilled")
			}
		}
		if len(migrated) > 0 {
			plan.GraphScenarios = migrated
		}
	}
	if _, exists := spec["maxRetries"]; !exists {
		plan.SpecDefaults = map[string]interface{}{"maxRetries": int64(3)}
		plan.Fields = append(plan.Fields, "spec.maxRetries defaulted to 3")
	}
}

func buildStatusPatch(item *unstructured.Unstructured, plan *migrationPlan) map[string]interface{} {
	status, found, _ := unstructured.NestedMap(item.Object, "status")
	if !found {
		return nil
	}
	var changed bool
	jobs, _, _ := unstructured.NestedSlice(status, "clusterJobs")
	for index, rawJob := range jobs {
		job, _ := rawJob.(map[string]interface{})
		for _, image := range plan.ClusterJobImages {
			if index == image.Index && sameJob(job, image.ClusterName, image.JobID) {
				if existing, _ := job["scenarioImage"].(string); existing == "" {
					job["scenarioImage"] = image.Image
					changed = true
				}
				if _, exists := job["containerImage"]; exists {
					delete(job, "containerImage")
					changed = true
				}
			}
		}
		for _, retries := range plan.ClusterJobRetries {
			if index == retries.Index && sameJob(job, retries.ClusterName, retries.JobID) {
				if configured, _ := job["maxRetriesConfigured"].(bool); !configured {
					job["maxRetries"] = retries.MaxRetries
					job["maxRetriesConfigured"] = true
					changed = true
				}
			}
		}
	}
	if len(jobs) > 0 {
		status["clusterJobs"] = jobs
	}

	scores, _, _ := unstructured.NestedSlice(status, "resiliencyScores")
	for index, rawScore := range scores {
		score, _ := rawScore.(map[string]interface{})
		clusterName, _ := score["clusterName"].(string)
		for _, migration := range plan.ResiliencyStatuses {
			if migration.Index != index || migration.ClusterName != clusterName {
				continue
			}
			if statusValue, _ := score["status"].(string); statusValue == "" {
				score["status"] = migration.Status
				changed = true
			}
			if providerName, _ := score["providerName"].(string); providerName == "" && migration.ProviderName != "" {
				score["providerName"] = migration.ProviderName
				changed = true
			}
		}
	}
	if len(scores) > 0 {
		status["resiliencyScores"] = scores
	}
	if !changed {
		return nil
	}
	return status
}

func buildRootPatch(item *unstructured.Unstructured, plan *migrationPlan) map[string]interface{} {
	patch := map[string]interface{}{}
	specPatch := map[string]interface{}{}
	if plan.Scenario != nil {
		specPatch["scenario"] = plan.Scenario
	}
	if plan.GraphScenarios != nil {
		graph, _, _ := unstructured.NestedMap(item.Object, "spec", "graph")
		for nodeID, reference := range plan.GraphScenarios {
			node, _ := graph[nodeID].(map[string]interface{})
			node["scenario"] = reference
		}
		specPatch["graph"] = graph
	}
	for key, value := range plan.SpecDefaults {
		specPatch[key] = value
	}
	if len(plan.RemoveSpecFields) > 0 {
		for _, key := range plan.RemoveSpecFields {
			specPatch[key] = nil
		}
	}
	if len(specPatch) > 0 {
		patch["spec"] = specPatch
	}
	if len(patch) > 0 {
		patch["metadata"] = map[string]interface{}{"resourceVersion": item.GetResourceVersion()}
	}
	return patch
}

func sameJob(job map[string]interface{}, clusterName, jobID string) bool {
	cluster, _ := job["clusterName"].(string)
	id, _ := job["jobId"].(string)
	return cluster == clusterName && id == jobID
}

func hasInlineRegistryCredentials(spec map[string]interface{}) bool {
	for _, field := range []string{"token", "username", "password"} {
		value, _ := spec[field].(string)
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}
