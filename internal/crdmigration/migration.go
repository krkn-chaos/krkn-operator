// Package crdmigration migrates stored Krkn custom resources around CRD schema updates.
package crdmigration

import (
	"context"
	"encoding/json"
	"errors"
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
	// AnnotationKey stores the retryable legacy-field snapshot for one custom resource.
	AnnotationKey = "migration.krkn-chaos.dev/api-v1"
	// GuardConfigMapName pauses run reconciliation and API writes during CRD synchronization.
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
	Version             int                     `json:"version"`
	Kind                string                  `json:"kind"`
	Fields              []string                `json:"fields,omitempty"`
	Warnings            []string                `json:"warnings,omitempty"`
	Blockers            []string                `json:"blockers,omitempty"`
	Scenario            map[string]interface{}  `json:"scenario,omitempty"`
	GraphScenarios      map[string]interface{}  `json:"graphScenarios,omitempty"`
	SpecDefaults        map[string]interface{}  `json:"specDefaults,omitempty"`
	RemoveSpecFields    []string                `json:"removeSpecFields,omitempty"`
	ClusterJobImages    []clusterJobImage       `json:"clusterJobImages,omitempty"`
	ClusterJobRetries   []clusterJobRetries     `json:"clusterJobRetries,omitempty"`
	ClusterJobProviders []clusterJobProvider    `json:"clusterJobProviders,omitempty"`
	ResiliencyStatuses  []resiliencyScoreStatus `json:"resiliencyStatuses,omitempty"`
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

type clusterJobProvider struct {
	Index        int    `json:"index"`
	ClusterName  string `json:"clusterName"`
	JobID        string `json:"jobId"`
	ProviderName string `json:"providerName"`
}

const patchRetries = 3

type migrationBlockedError struct{ reason string }

func (e migrationBlockedError) Error() string { return e.reason }

type resiliencyScoreStatus struct {
	Index        int    `json:"index"`
	ClusterName  string `json:"clusterName"`
	Status       string `json:"status"`
	ProviderName string `json:"providerName,omitempty"`
}

type migrationResult struct {
	changed      bool
	blocked      bool
	warningCount int
}

type migrationStats struct {
	scanned             int
	changed             int
	unchanged           int
	objectsWithWarnings int
	warnings            int
	blocked             int
}

type graphScenarioReferences map[string]map[string]interface{}

// Stage snapshots legacy values needed by migration in annotations that survive
// CRD schema pruning. It is safe to retry after a failed stage or rollout.
// It returns an error for malformed resource data, invalid staged annotations,
// annotation size limits, or Kubernetes API failures; existing snapshots remain
// in place when a stage attempt fails.
func Stage(ctx context.Context, client dynamic.Interface, namespace string, logger logr.Logger) error {
	graphReferences, err := loadGraphScenarioReferences(ctx, client, namespace, logger, "stage")
	if err != nil {
		return err
	}
	return forEachResource(ctx, client, namespace, logger, "stage", func(resource resourceDefinition, item *unstructured.Unstructured) (migrationResult, error) {
		annotations := item.GetAnnotations()
		plan, err := planResourceWithGraphReferences(resource.kind, item, graphReferences)
		if err != nil {
			return migrationResult{}, err
		}
		previousEncoded := annotations[AnnotationKey]
		if previousEncoded != "" {
			previousPlan := migrationPlan{}
			if err := json.Unmarshal([]byte(previousEncoded), &previousPlan); err != nil {
				return migrationResult{}, fmt.Errorf("decode staged migration for %s/%s: %w", item.GetNamespace(), item.GetName(), err)
			}
			plan = mergeMigrationPlans(previousPlan, plan)
		}
		if len(plan.Fields) == 0 && len(plan.Warnings) == 0 && len(plan.Blockers) == 0 {
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
		patch, err := json.Marshal(map[string]interface{}{"metadata": map[string]interface{}{
			"resourceVersion": item.GetResourceVersion(),
			"annotations":     map[string]string{AnnotationKey: string(encoded)},
		}})
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
		logger.Info(message, "kind", resource.kind, "namespace", item.GetNamespace(), "name", item.GetName(), "fields", plan.Fields, "warnings", plan.Warnings, "blockers", plan.Blockers)
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
	merged.Blockers = uniqueSortedStrings(append(merged.Blockers, current.Blockers...))
	merged.RemoveSpecFields = uniqueSortedStrings(append(merged.RemoveSpecFields, current.RemoveSpecFields...))
	merged.ClusterJobImages = mergeClusterJobImages(merged.ClusterJobImages, current.ClusterJobImages)
	merged.ClusterJobRetries = mergeClusterJobRetries(merged.ClusterJobRetries, current.ClusterJobRetries)
	merged.ClusterJobProviders = mergeClusterJobProviders(merged.ClusterJobProviders, current.ClusterJobProviders)
	merged.ResiliencyStatuses = mergeResiliencyStatuses(merged.ResiliencyStatuses, current.ResiliencyStatuses)
	if merged.Scenario != nil {
		missingIdentity := "scenario identity is absent; spec.scenario.name cannot be reconstructed"
		if current.Scenario != nil || containsString(current.Blockers, missingIdentity) {
			merged.Blockers = removeString(merged.Blockers, missingIdentity)
			merged.Warnings = removeWarning(merged.Warnings, missingIdentity)
		}
		inlineCredentials := "inline registry credentials have no registryName and cannot be transferred to the current API"
		privateWithoutRegistry := "private scenario has no registryName; a saved registry reference cannot be reconstructed"
		if current.Scenario != nil {
			merged.Blockers = removeString(merged.Blockers, inlineCredentials)
			merged.Blockers = removeString(merged.Blockers, privateWithoutRegistry)
			merged.Warnings = removeWarning(merged.Warnings, inlineCredentials)
			merged.Warnings = removeWarning(merged.Warnings, privateWithoutRegistry)
		} else if merged.Scenario["registryName"] != nil {
			if !containsString(current.Blockers, inlineCredentials) {
				merged.Blockers = removeString(merged.Blockers, inlineCredentials)
				merged.Warnings = removeWarning(merged.Warnings, inlineCredentials)
			}
			if !containsString(current.Blockers, privateWithoutRegistry) {
				merged.Blockers = removeString(merged.Blockers, privateWithoutRegistry)
				merged.Warnings = removeWarning(merged.Warnings, privateWithoutRegistry)
			}
		}
		publicRegistryConflict := "existing public scenario reference conflicts with a legacy registryName"
		if !containsString(current.Warnings, publicRegistryConflict) {
			merged.Warnings = removeWarning(merged.Warnings, publicRegistryConflict)
		}
	}
	for _, blocker := range append([]string(nil), merged.Blockers...) {
		if strings.HasPrefix(blocker, "status.clusterJobs[") && !containsString(current.Blockers, blocker) {
			// Recheck required fields while rebuilding the status patch; retaining an
			// old blocker here would prevent that bounded, retryable check.
			merged.Blockers = removeString(merged.Blockers, blocker)
		}
		if blocker == "graph identity is absent; spec.graph cannot be reconstructed" && current.GraphScenarios != nil && !containsString(current.Blockers, blocker) {
			merged.Blockers = removeString(merged.Blockers, blocker)
		}
	}
	if containsString(merged.Fields, "KrknGraphRun.spec.graph node -> spec.scenario") {
		merged.Fields = removeString(merged.Fields, "spec.scenarioName -> spec.scenario.name")
		merged.Fields = removeString(merged.Fields, "legacy registry identity -> spec.scenario.private")
		merged.Fields = removeString(merged.Fields, "spec.scenario")
		merged.Warnings = removeWarning(merged.Warnings, "existing public scenario reference conflicts with a legacy registryName")
	}
	for nodeID, rawReference := range merged.GraphScenarios {
		reference, ok := rawReference.(map[string]interface{})
		if !ok || reference["name"] == "" {
			continue
		}
		missingIdentity := "graph node " + nodeID + " has no scenario identity"
		merged.Blockers = removeString(merged.Blockers, missingIdentity)
		merged.Warnings = removeWarning(merged.Warnings, missingIdentity)
		privateWithoutRegistry := "graph node " + nodeID + " is private but has no registryName"
		if reference["registryName"] != nil && !containsString(current.Blockers, privateWithoutRegistry) {
			merged.Blockers = removeString(merged.Blockers, privateWithoutRegistry)
			merged.Warnings = removeWarning(merged.Warnings, privateWithoutRegistry)
		}
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

func mergeClusterJobProviders(previous, current []clusterJobProvider) []clusterJobProvider {
	merged := map[string]clusterJobProvider{}
	for _, item := range append(previous, current...) {
		key := fmt.Sprintf("%d\x00%s\x00%s", item.Index, item.ClusterName, item.JobID)
		merged[key] = item
	}
	result := make([]clusterJobProvider, 0, len(merged))
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
// Objects with unrecoverable fields remain annotated and blocked for manual
// repair; callers can retry after repairing the object or retrying the upgrade.
// It returns an error for unsupported payloads and Kubernetes API failures.
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
			currentPlan, err := planResourceWithGraphReferences(resource.kind, item, graphReferences)
			if err != nil {
				return migrationResult{}, err
			}
			plan = mergeMigrationPlans(plan, currentPlan)
		} else {
			plan, err = planResourceWithGraphReferences(resource.kind, item, graphReferences)
			if err != nil {
				return migrationResult{}, err
			}
		}
		if annotations[AnnotationKey] != "" && resource.kind == "KrknScenarioRun" {
			applyGraphScenarioReference(&plan, item, graphReferences)
		}
		if plan.Version != 1 || plan.Kind != resource.kind {
			return migrationResult{}, fmt.Errorf("unsupported migration payload for %s/%s (version %d, kind %q)", item.GetNamespace(), item.GetName(), plan.Version, plan.Kind)
		}
		if len(plan.Blockers) > 0 {
			if annotations[AnnotationKey] == "" {
				if err := writeMigrationAnnotation(ctx, client, resource, item, plan); err != nil {
					return migrationResult{}, err
				}
			}
			logger.Info("custom resource migration is blocked and remains staged", "kind", resource.kind, "namespace", item.GetNamespace(), "name", item.GetName(), "reasons", plan.Blockers)
			return migrationResult{blocked: true, warningCount: len(plan.Warnings)}, nil
		}
		if len(plan.Fields) == 0 && len(plan.Warnings) == 0 && annotations[AnnotationKey] == "" {
			logger.Info("custom resource already uses the current API", "kind", resource.kind, "namespace", item.GetNamespace(), "name", item.GetName())
			return migrationResult{}, nil
		}

		stagedAnnotation := annotations[AnnotationKey]
		item, err = patchMigrationResourceWithRetry(ctx, client, resource, item, "status", stagedAnnotation, func(current *unstructured.Unstructured) (map[string]interface{}, error) {
			plan, err = refreshedMigrationPlan(ctx, client, resource, current, stagedAnnotation, "complete")
			if err != nil {
				return nil, err
			}
			if len(plan.Blockers) > 0 {
				return nil, migrationBlockedError{reason: strings.Join(plan.Blockers, "; ")}
			}
			statusPatch, err := buildStatusPatch(current, &plan)
			if err != nil || len(statusPatch) == 0 {
				return nil, err
			}
			return map[string]interface{}{"status": statusPatch}, nil
		})
		if err != nil {
			var blocked migrationBlockedError
			if errors.As(err, &blocked) {
				if stagedAnnotation == "" {
					if err := writeMigrationAnnotation(ctx, client, resource, item, plan); err != nil {
						return migrationResult{}, err
					}
				}
				logger.Info("custom resource migration is blocked and remains staged", "kind", resource.kind, "namespace", item.GetNamespace(), "name", item.GetName(), "reason", blocked.reason)
				return migrationResult{blocked: true, warningCount: len(plan.Warnings)}, nil
			}
			return migrationResult{}, err
		}

		item, err = patchMigrationResourceWithRetry(ctx, client, resource, item, "", stagedAnnotation, func(current *unstructured.Unstructured) (map[string]interface{}, error) {
			plan, err = refreshedMigrationPlan(ctx, client, resource, current, stagedAnnotation, "complete")
			if err != nil {
				return nil, err
			}
			if len(plan.Blockers) > 0 {
				return nil, migrationBlockedError{reason: strings.Join(plan.Blockers, "; ")}
			}
			rootPatch, err := buildRootPatch(current, &plan)
			if err != nil {
				return nil, err
			}
			if stagedAnnotation != "" {
				metadataPatch, _ := rootPatch["metadata"].(map[string]interface{})
				if metadataPatch == nil {
					metadataPatch = map[string]interface{}{}
					rootPatch["metadata"] = metadataPatch
				}
				metadataPatch["annotations"] = map[string]interface{}{AnnotationKey: nil}
			}
			return rootPatch, nil
		})
		if err != nil {
			var blocked migrationBlockedError
			if errors.As(err, &blocked) {
				if stagedAnnotation == "" {
					if err := writeMigrationAnnotation(ctx, client, resource, item, plan); err != nil {
						return migrationResult{}, err
					}
				}
				logger.Info("custom resource migration is blocked and remains staged", "kind", resource.kind, "namespace", item.GetNamespace(), "name", item.GetName(), "reason", blocked.reason)
				return migrationResult{blocked: true, warningCount: len(plan.Warnings)}, nil
			}
			return migrationResult{}, err
		}

		logger.Info("completed custom resource migration", "kind", resource.kind, "namespace", item.GetNamespace(), "name", item.GetName(), "fields", plan.Fields, "warnings", plan.Warnings)
		return migrationResult{changed: true, warningCount: len(plan.Warnings)}, nil
	})
}

func refreshedMigrationPlan(ctx context.Context, client dynamic.Interface, resource resourceDefinition, item *unstructured.Unstructured, expectedAnnotation, phase string) (migrationPlan, error) {
	annotations := item.GetAnnotations()
	encoded := annotations[AnnotationKey]
	if encoded != expectedAnnotation {
		return migrationPlan{}, apierrors.NewConflict(resource.gvr.GroupResource(), item.GetName(), errors.New("staged migration annotation changed during completion"))
	}
	graphReferences, err := loadGraphScenarioReferences(ctx, client, item.GetNamespace(), logr.Discard(), phase)
	if err != nil {
		return migrationPlan{}, err
	}
	current, err := planResourceWithGraphReferences(resource.kind, item, graphReferences)
	if err != nil {
		return migrationPlan{}, err
	}
	if encoded != "" {
		previous := migrationPlan{}
		if err := json.Unmarshal([]byte(encoded), &previous); err != nil {
			return migrationPlan{}, fmt.Errorf("decode staged migration for %s/%s: %w", item.GetNamespace(), item.GetName(), err)
		}
		current = mergeMigrationPlans(previous, current)
		if resource.kind == "KrknScenarioRun" {
			applyGraphScenarioReference(&current, item, graphReferences)
		}
	}
	return current, nil
}

func patchMigrationResourceWithRetry(
	ctx context.Context,
	client dynamic.Interface,
	resource resourceDefinition,
	item *unstructured.Unstructured,
	subresource string,
	expectedAnnotation string,
	build func(*unstructured.Unstructured) (map[string]interface{}, error),
) (*unstructured.Unstructured, error) {
	current := item
	resourceClient := client.Resource(resource.gvr).Namespace(item.GetNamespace())
	for attempt := 0; attempt < patchRetries; attempt++ {
		patchObject, err := build(current)
		if err != nil {
			return current, err
		}
		if len(patchObject) == 0 {
			return current, nil
		}
		metadataPatch, _ := patchObject["metadata"].(map[string]interface{})
		if metadataPatch == nil {
			metadataPatch = map[string]interface{}{}
			patchObject["metadata"] = metadataPatch
		}
		metadataPatch["resourceVersion"] = current.GetResourceVersion()
		encoded, err := json.Marshal(patchObject)
		if err != nil {
			return current, fmt.Errorf("encode %s migration patch for %s/%s: %w", resource.kind, current.GetNamespace(), current.GetName(), err)
		}
		var updated *unstructured.Unstructured
		if subresource == "" {
			updated, err = resourceClient.Patch(ctx, current.GetName(), types.MergePatchType, encoded, metav1.PatchOptions{})
		} else {
			updated, err = resourceClient.Patch(ctx, current.GetName(), types.MergePatchType, encoded, metav1.PatchOptions{}, subresource)
		}
		if err == nil {
			if updated != nil {
				return updated, nil
			}
			return current, nil
		}
		if !apierrors.IsConflict(err) || attempt == patchRetries-1 {
			return current, fmt.Errorf("patch %s migration for %s/%s after %d attempt(s): %w", resource.kind, current.GetNamespace(), current.GetName(), attempt+1, err)
		}
		current, err = resourceClient.Get(ctx, item.GetName(), metav1.GetOptions{})
		if err != nil {
			return item, fmt.Errorf("refetch %s/%s after migration conflict: %w", item.GetNamespace(), item.GetName(), err)
		}
		if current.GetAnnotations()[AnnotationKey] != expectedAnnotation {
			return current, apierrors.NewConflict(resource.gvr.GroupResource(), current.GetName(), errors.New("staged migration annotation changed during completion"))
		}
	}
	return current, fmt.Errorf("patch %s migration for %s/%s exhausted retries", resource.kind, item.GetNamespace(), item.GetName())
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
				if result.blocked {
					stats.blocked++
				} else if result.changed {
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
		"blocked", stats.blocked,
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
			graph, found, err := migrationNestedMap(graphRun.Object, "spec", "graph")
			if err != nil {
				return nil, fmt.Errorf("read KrknGraphRun %s/%s graph for child migration: %w", graphRun.GetNamespace(), graphRun.GetName(), err)
			}
			if !found {
				continue
			}
			for nodeID, rawNode := range graph {
				node, ok := rawNode.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("KrknGraphRun %s/%s spec.graph.%s must be an object, got %T", graphRun.GetNamespace(), graphRun.GetName(), nodeID, rawNode)
				}
				ref, err := graphNodeScenarioReference(node)
				if err != nil {
					return nil, fmt.Errorf("read KrknGraphRun %s/%s spec.graph.%s: %w", graphRun.GetNamespace(), graphRun.GetName(), nodeID, err)
				}
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

func graphNodeScenarioReference(node map[string]interface{}) (map[string]interface{}, error) {
	current, found, err := migrationNestedMap(node, "scenario")
	if err != nil {
		return nil, err
	}
	if !found {
		current = map[string]interface{}{}
	}
	name, err := migrationString(current, "name", "scenario")
	if err != nil {
		return nil, err
	}
	if name == "" {
		name, err = migrationString(node, "name", "node")
		if err != nil {
			return nil, err
		}
	}
	if name == "" {
		return nil, nil
	}
	registryName, err := migrationString(current, "registryName", "scenario")
	if err != nil {
		return nil, err
	}
	if registryName == "" {
		registryName, err = migrationString(node, "registryName", "node")
		if err != nil {
			return nil, err
		}
	}
	private, found := current["private"].(bool)
	if rawPrivate, exists := current["private"]; exists && rawPrivate != nil && !found {
		return nil, fmt.Errorf("scenario.private must be a boolean, got %T", rawPrivate)
	}
	if !found {
		private = registryName != ""
	}
	ref := map[string]interface{}{"name": name, "private": private}
	if private && registryName != "" {
		ref["registryName"] = registryName
	}
	return ref, nil
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

func planResourceWithGraphReferences(kind string, item *unstructured.Unstructured, graphReferences graphScenarioReferences) (migrationPlan, error) {
	plan, err := planResource(kind, item)
	if err != nil {
		return migrationPlan{}, err
	}
	if kind == "KrknScenarioRun" {
		applyGraphScenarioReference(&plan, item, graphReferences)
	}
	return plan, nil
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
		plan.Blockers = removeString(plan.Blockers, "scenario identity is absent; spec.scenario.name cannot be reconstructed")
		plan.Warnings = removeWarning(plan.Warnings, "private scenario has no registryName; a saved registry reference cannot be reconstructed")
		plan.Blockers = removeString(plan.Blockers, "private scenario has no registryName; a saved registry reference cannot be reconstructed")
		plan.Warnings = removeWarning(plan.Warnings, "existing public scenario reference conflicts with a legacy registryName")
		if reference["registryName"] != nil {
			plan.Warnings = removeWarning(plan.Warnings, "inline registry credentials have no registryName and cannot be transferred to the current API")
			plan.Blockers = removeString(plan.Blockers, "inline registry credentials have no registryName and cannot be transferred to the current API")
		}
		if reference["private"] == true && reference["registryName"] == nil {
			warning := "private scenario has no registryName; a saved registry reference cannot be reconstructed"
			plan.Warnings = append(plan.Warnings, warning)
			plan.Blockers = append(plan.Blockers, warning)
		}
		sort.Strings(plan.Fields)
		sort.Strings(plan.Warnings)
		plan.Blockers = uniqueSortedStrings(plan.Blockers)
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

func migrationNestedMap(object map[string]interface{}, path ...string) (map[string]interface{}, bool, error) {
	value, found, err := unstructured.NestedMap(object, path...)
	if err != nil {
		return nil, false, fmt.Errorf("read %s: %w", strings.Join(path, "."), err)
	}
	return value, found, nil
}

func migrationNestedSlice(object map[string]interface{}, path ...string) ([]interface{}, bool, error) {
	value, found, err := unstructured.NestedSlice(object, path...)
	if err != nil {
		return nil, false, fmt.Errorf("read %s: %w", strings.Join(path, "."), err)
	}
	return value, found, nil
}

func migrationString(object map[string]interface{}, field string, path string) (string, error) {
	value, exists := object[field]
	if !exists || value == nil {
		return "", nil
	}
	result, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s.%s must be a string, got %T", path, field, value)
	}
	return result, nil
}

func planResource(kind string, item *unstructured.Unstructured) (migrationPlan, error) {
	plan := migrationPlan{Version: 1, Kind: kind}
	spec, found, err := migrationNestedMap(item.Object, "spec")
	if err != nil {
		return migrationPlan{}, fmt.Errorf("custom resource %s/%s has malformed spec: %w", item.GetNamespace(), item.GetName(), err)
	}
	if !found {
		return migrationPlan{}, fmt.Errorf("custom resource %s/%s has no spec", item.GetNamespace(), item.GetName())
	}
	status, statusFound, err := migrationNestedMap(item.Object, "status")
	if err != nil {
		return migrationPlan{}, fmt.Errorf("custom resource %s/%s has malformed status: %w", item.GetNamespace(), item.GetName(), err)
	}
	if !statusFound {
		status = map[string]interface{}{}
	}
	switch kind {
	case "KrknScenarioRun":
		err = planScenarioRun(&plan, spec, status)
	case "KrknGraphRun":
		err = planGraphRun(&plan, spec)
	default:
		plan.Warnings = append(plan.Warnings, "no migration registered for resource kind")
	}
	if err != nil {
		return migrationPlan{}, fmt.Errorf("plan %s/%s migration: %w", item.GetNamespace(), item.GetName(), err)
	}
	sort.Strings(plan.Fields)
	sort.Strings(plan.Warnings)
	plan.Blockers = uniqueSortedStrings(plan.Blockers)
	return plan, nil
}

func planScenarioRun(plan *migrationPlan, spec, status map[string]interface{}) error {
	current, currentFound, err := migrationNestedMap(spec, "scenario")
	if err != nil {
		return fmt.Errorf("read spec.scenario: %w", err)
	}
	if !currentFound {
		current = map[string]interface{}{}
	}
	name, err := migrationString(current, "name", "spec.scenario")
	if err != nil {
		return err
	}
	registryName, err := migrationString(current, "registryName", "spec.scenario")
	if err != nil {
		return err
	}
	if registryName == "" {
		registryName, err = migrationString(spec, "registryName", "spec")
		if err != nil {
			return err
		}
	}
	inlineCredentials, err := hasInlineRegistryCredentials(spec)
	if err != nil {
		return err
	}
	if name == "" {
		name, err = migrationString(spec, "scenarioName", "spec")
		if err != nil {
			return err
		}
		if name != "" {
			plan.Fields = append(plan.Fields, "spec.scenarioName -> spec.scenario.name")
		}
	}
	if inlineCredentials && registryName == "" {
		warning := "inline registry credentials have no registryName and cannot be transferred to the current API"
		plan.Warnings = append(plan.Warnings, warning)
		plan.Blockers = append(plan.Blockers, warning)
	}
	if name == "" {
		warning := "scenario identity is absent; spec.scenario.name cannot be reconstructed"
		plan.Warnings = append(plan.Warnings, warning)
		plan.Blockers = append(plan.Blockers, warning)
	} else {
		private, hasPrivate := current["private"].(bool)
		if rawPrivate, exists := current["private"]; exists && rawPrivate != nil && !hasPrivate {
			return fmt.Errorf("spec.scenario.private must be a boolean, got %T", rawPrivate)
		}
		if !hasPrivate {
			private = registryName != ""
			plan.Fields = append(plan.Fields, "legacy registry identity -> spec.scenario.private")
		}
		if private && registryName == "" {
			warning := "private scenario has no registryName; a saved registry reference cannot be reconstructed"
			plan.Warnings = append(plan.Warnings, warning)
			plan.Blockers = append(plan.Blockers, warning)
		}
		if !private && registryName != "" {
			plan.Warnings = append(plan.Warnings, "existing public scenario reference conflicts with a legacy registryName")
		}
		ref := map[string]interface{}{"name": name, "private": private}
		if private && registryName != "" {
			ref["registryName"] = registryName
		}
		if len(plan.Blockers) == 0 {
			// Carry valid current references into a refreshed plan so they replace stale staged values.
			plan.Scenario = ref
		}
		if len(current) == 0 || current["name"] != ref["name"] || current["private"] != ref["private"] || current["registryName"] != ref["registryName"] {
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

	jobs, _, err := migrationNestedSlice(status, "clusterJobs")
	if err != nil {
		return err
	}
	for index, rawJob := range jobs {
		job, ok := rawJob.(map[string]interface{})
		if !ok {
			return fmt.Errorf("status.clusterJobs[%d] must be an object, got %T", index, rawJob)
		}
		image, err := migrationString(job, "scenarioImage", fmt.Sprintf("status.clusterJobs[%d]", index))
		if err != nil {
			return err
		}
		legacyImage, err := migrationString(job, "containerImage", fmt.Sprintf("status.clusterJobs[%d]", index))
		if err != nil {
			return err
		}
		clusterName, err := migrationString(job, "clusterName", fmt.Sprintf("status.clusterJobs[%d]", index))
		if err != nil {
			return err
		}
		jobID, err := migrationString(job, "jobId", fmt.Sprintf("status.clusterJobs[%d]", index))
		if err != nil {
			return err
		}
		if image == "" && legacyImage != "" {
			plan.ClusterJobImages = append(plan.ClusterJobImages, clusterJobImage{Index: index, ClusterName: clusterName, JobID: jobID, Image: legacyImage})
			plan.Fields = append(plan.Fields, "status.clusterJobs[].containerImage -> scenarioImage")
		}
		podName, err := migrationString(job, "podName", fmt.Sprintf("status.clusterJobs[%d]", index))
		if err != nil {
			return err
		}
		if podName != "" {
			configured, _ := job["maxRetriesConfigured"].(bool)
			if rawConfigured, exists := job["maxRetriesConfigured"]; exists && rawConfigured != nil {
				if _, ok := rawConfigured.(bool); !ok {
					return fmt.Errorf("status.clusterJobs[%d].maxRetriesConfigured must be a boolean", index)
				}
			}
			if !configured {
				maxRetries, found, err := unstructured.NestedInt64(job, "maxRetries")
				if err != nil {
					return fmt.Errorf("read status.clusterJobs[%d].maxRetries: %w", index, err)
				}
				if !found || maxRetries == 0 {
					if specRetries, specFound, err := unstructured.NestedInt64(spec, "maxRetries"); err != nil {
						return fmt.Errorf("read spec.maxRetries: %w", err)
					} else if specFound {
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

	providersByCluster, ambiguousProviders, err := providersByCluster(spec, jobs)
	if err != nil {
		return err
	}
	for index, rawJob := range jobs {
		job := rawJob.(map[string]interface{})
		clusterName, _ := job["clusterName"].(string)
		jobID, _ := job["jobId"].(string)
		providerName, _ := job["providerName"].(string)
		if providerName == "" && clusterName != "" && !ambiguousProviders[clusterName] && providersByCluster[clusterName] != "" {
			providerName = providersByCluster[clusterName]
			plan.ClusterJobProviders = append(plan.ClusterJobProviders, clusterJobProvider{Index: index, ClusterName: clusterName, JobID: jobID, ProviderName: providerName})
			plan.Fields = append(plan.Fields, "spec.targetClusters -> status.clusterJobs[].providerName")
		}
	}
	jobsNeedUpdate := len(plan.ClusterJobImages) > 0 || len(plan.ClusterJobRetries) > 0 || len(plan.ClusterJobProviders) > 0
	if jobsNeedUpdate {
		for index, rawJob := range jobs {
			job := rawJob.(map[string]interface{})
			clusterName, _ := job["clusterName"].(string)
			jobID, _ := job["jobId"].(string)
			phase, err := migrationString(job, "phase", fmt.Sprintf("status.clusterJobs[%d]", index))
			if err != nil {
				return err
			}
			providerName, _ := job["providerName"].(string)
			if providerName == "" {
				for _, inferred := range plan.ClusterJobProviders {
					if inferred.Index == index && inferred.ClusterName == clusterName && inferred.JobID == jobID {
						providerName = inferred.ProviderName
						break
					}
				}
			}
			if clusterName == "" || jobID == "" || phase == "" || providerName == "" {
				plan.Blockers = append(plan.Blockers, fmt.Sprintf("status.clusterJobs[%d] lacks required identity or phase fields; refusing to write a partial status list", index))
			}
		}
	}
	scores, _, err := migrationNestedSlice(status, "resiliencyScores")
	if err != nil {
		return err
	}
	for index, rawScore := range scores {
		score, ok := rawScore.(map[string]interface{})
		if !ok {
			return fmt.Errorf("status.resiliencyScores[%d] must be an object, got %T", index, rawScore)
		}
		clusterName, err := migrationString(score, "clusterName", fmt.Sprintf("status.resiliencyScores[%d]", index))
		if err != nil {
			return err
		}
		statusValue, err := migrationString(score, "status", fmt.Sprintf("status.resiliencyScores[%d]", index))
		if err != nil {
			return err
		}
		providerName, err := migrationString(score, "providerName", fmt.Sprintf("status.resiliencyScores[%d]", index))
		if err != nil {
			return err
		}
		if providerName == "" {
			if ambiguousProviders[clusterName] {
				plan.Warnings = append(plan.Warnings, fmt.Sprintf("resiliency score cluster %q maps to multiple providers; providerName was not inferred", clusterName))
			} else {
				providerName = providersByCluster[clusterName]
			}
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
	if len(plan.ClusterJobProviders) > 0 {
		plan.Fields = append(plan.Fields, "legacy cluster job providers preserved")
	}
	plan.Blockers = uniqueSortedStrings(plan.Blockers)
	return nil
}

func providersByCluster(spec map[string]interface{}, jobs []interface{}) (map[string]string, map[string]bool, error) {
	providers := map[string]string{}
	ambiguous := map[string]bool{}
	add := func(clusterName, providerName string) {
		addProvider(&providers, &ambiguous, clusterName, providerName)
	}
	targetClusters, exists := spec["targetClusters"]
	if exists && targetClusters != nil {
		byProvider, ok := targetClusters.(map[string]interface{})
		if !ok {
			return nil, nil, fmt.Errorf("spec.targetClusters must be an object, got %T", targetClusters)
		}
		for providerName, rawClusters := range byProvider {
			clusters, ok := rawClusters.([]interface{})
			if !ok {
				return nil, nil, fmt.Errorf("spec.targetClusters.%s must be an array, got %T", providerName, rawClusters)
			}
			for index, rawCluster := range clusters {
				clusterName, ok := rawCluster.(string)
				if !ok {
					return nil, nil, fmt.Errorf("spec.targetClusters.%s[%d] must be a string, got %T", providerName, index, rawCluster)
				}
				add(clusterName, providerName)
			}
		}
	}
	for index, rawJob := range jobs {
		job, ok := rawJob.(map[string]interface{})
		if !ok {
			return nil, nil, fmt.Errorf("status.clusterJobs[%d] must be an object, got %T", index, rawJob)
		}
		clusterName, err := migrationString(job, "clusterName", fmt.Sprintf("status.clusterJobs[%d]", index))
		if err != nil {
			return nil, nil, err
		}
		providerName, err := migrationString(job, "providerName", fmt.Sprintf("status.clusterJobs[%d]", index))
		if err != nil {
			return nil, nil, err
		}
		if clusterName != "" && providerName != "" {
			add(clusterName, providerName)
		}
	}
	return providers, ambiguous, nil
}

func addProvider(providers *map[string]string, ambiguous *map[string]bool, clusterName, providerName string) {
	if clusterName == "" || providerName == "" || (*ambiguous)[clusterName] {
		return
	}
	if existing := (*providers)[clusterName]; existing != "" && existing != providerName {
		delete(*providers, clusterName)
		(*ambiguous)[clusterName] = true
		return
	}
	(*providers)[clusterName] = providerName
}

func planGraphRun(plan *migrationPlan, spec map[string]interface{}) error {
	graph, ok, err := migrationNestedMap(spec, "graph")
	if err != nil {
		return err
	}
	if !ok {
		plan.Blockers = append(plan.Blockers, "graph identity is absent; spec.graph cannot be reconstructed")
		plan.GraphScenarios = map[string]interface{}{}
	}
	if ok {
		migrated := map[string]interface{}{}
		for nodeID, rawNode := range graph {
			node, ok := rawNode.(map[string]interface{})
			if !ok {
				return fmt.Errorf("spec.graph.%s must be an object, got %T", nodeID, rawNode)
			}
			current, currentFound, err := migrationNestedMap(node, "scenario")
			if err != nil {
				return fmt.Errorf("spec.graph.%s: %w", nodeID, err)
			}
			if !currentFound {
				current = map[string]interface{}{}
			}
			name, err := migrationString(current, "name", "spec.graph."+nodeID+".scenario")
			if err != nil {
				return err
			}
			if name == "" {
				name, err = migrationString(node, "name", "spec.graph."+nodeID)
				if err != nil {
					return err
				}
				if name != "" {
					plan.Fields = append(plan.Fields, "spec.graph."+nodeID+".name -> scenario.name")
				}
			}
			if name == "" && strings.HasPrefix(nodeID, "_") {
				continue
			}
			if name == "" {
				warning := "graph node " + nodeID + " has no scenario identity"
				plan.Warnings = append(plan.Warnings, warning)
				plan.Blockers = append(plan.Blockers, warning)
				continue
			}
			private, hasPrivate := current["private"].(bool)
			if rawPrivate, exists := current["private"]; exists && rawPrivate != nil && !hasPrivate {
				return fmt.Errorf("spec.graph.%s.scenario.private must be a boolean", nodeID)
			}
			registryName, err := migrationString(current, "registryName", "spec.graph."+nodeID+".scenario")
			if err != nil {
				return err
			}
			if registryName == "" {
				registryName, err = migrationString(node, "registryName", "spec.graph."+nodeID)
				if err != nil {
					return err
				}
			}
			if !hasPrivate {
				private = registryName != ""
			}
			if private && registryName == "" {
				warning := "graph node " + nodeID + " is private but has no registryName"
				plan.Warnings = append(plan.Warnings, warning)
				plan.Blockers = append(plan.Blockers, warning)
			}
			if !private && registryName != "" {
				plan.Warnings = append(plan.Warnings, "graph node "+nodeID+" has a public scenario reference and a legacy registryName")
			}
			ref := map[string]interface{}{"name": name, "private": private}
			if private && registryName != "" {
				ref["registryName"] = registryName
			}
			migrated[nodeID] = ref
			if len(current) == 0 || current["name"] != ref["name"] || current["private"] != ref["private"] || current["registryName"] != ref["registryName"] {
				plan.Fields = append(plan.Fields, "spec.graph."+nodeID+".scenario backfilled")
			}
		}
		plan.GraphScenarios = migrated
	}
	if _, exists := spec["maxRetries"]; !exists {
		plan.SpecDefaults = map[string]interface{}{"maxRetries": int64(3)}
		plan.Fields = append(plan.Fields, "spec.maxRetries defaulted to 3")
	}
	plan.Blockers = uniqueSortedStrings(plan.Blockers)
	return nil
}

func buildStatusPatch(item *unstructured.Unstructured, plan *migrationPlan) (map[string]interface{}, error) {
	status, found, err := migrationNestedMap(item.Object, "status")
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	var jobsChanged, scoresChanged bool
	jobs, _, err := migrationNestedSlice(status, "clusterJobs")
	if err != nil {
		return nil, err
	}
	for index, rawJob := range jobs {
		job, ok := rawJob.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("status.clusterJobs[%d] must be an object, got %T", index, rawJob)
		}
		for _, image := range plan.ClusterJobImages {
			if index == image.Index && sameJob(job, image.ClusterName, image.JobID) {
				if existing, _ := job["scenarioImage"].(string); existing == "" {
					job["scenarioImage"] = image.Image
					jobsChanged = true
				}
				if _, exists := job["containerImage"]; exists {
					delete(job, "containerImage")
					jobsChanged = true
				}
			}
		}
		for _, retries := range plan.ClusterJobRetries {
			if index == retries.Index && sameJob(job, retries.ClusterName, retries.JobID) {
				if configured, _ := job["maxRetriesConfigured"].(bool); !configured {
					job["maxRetries"] = retries.MaxRetries
					job["maxRetriesConfigured"] = true
					jobsChanged = true
				}
			}
		}
		for _, provider := range plan.ClusterJobProviders {
			if index == provider.Index && sameJob(job, provider.ClusterName, provider.JobID) {
				if existing, _ := job["providerName"].(string); existing == "" {
					job["providerName"] = provider.ProviderName
					jobsChanged = true
				}
			}
		}
	}
	statusPatch := map[string]interface{}{}
	if jobsChanged {
		// Updating a JSON list replaces the entire list. Ensure every element can
		// pass the required fields in the currently installed CRD before patching.
		for index, rawJob := range jobs {
			job := rawJob.(map[string]interface{})
			for _, required := range []string{"clusterName", "jobId", "phase", "providerName"} {
				value, _ := job[required].(string)
				if value == "" {
					return nil, migrationBlockedError{reason: fmt.Sprintf("status.clusterJobs[%d].%s is required by the current CRD; refusing to patch the full job list", index, required)}
				}
			}
		}
		statusPatch["clusterJobs"] = jobs
	}

	scores, _, err := migrationNestedSlice(status, "resiliencyScores")
	if err != nil {
		return nil, err
	}
	for index, rawScore := range scores {
		score, ok := rawScore.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("status.resiliencyScores[%d] must be an object, got %T", index, rawScore)
		}
		clusterName, err := migrationString(score, "clusterName", fmt.Sprintf("status.resiliencyScores[%d]", index))
		if err != nil {
			return nil, err
		}
		for _, migration := range plan.ResiliencyStatuses {
			if migration.Index != index || migration.ClusterName != clusterName {
				continue
			}
			statusValue, err := migrationString(score, "status", fmt.Sprintf("status.resiliencyScores[%d]", index))
			if err != nil {
				return nil, err
			}
			if statusValue == "" {
				score["status"] = migration.Status
				scoresChanged = true
			}
			providerName, err := migrationString(score, "providerName", fmt.Sprintf("status.resiliencyScores[%d]", index))
			if err != nil {
				return nil, err
			}
			if providerName == "" && migration.ProviderName != "" {
				score["providerName"] = migration.ProviderName
				scoresChanged = true
			}
		}
	}
	if scoresChanged {
		statusPatch["resiliencyScores"] = scores
	}
	if len(statusPatch) == 0 {
		return nil, nil
	}
	return statusPatch, nil
}

func buildRootPatch(item *unstructured.Unstructured, plan *migrationPlan) (map[string]interface{}, error) {
	patch := map[string]interface{}{}
	specPatch := map[string]interface{}{}
	if plan.Scenario != nil {
		specPatch["scenario"] = plan.Scenario
	}
	if len(plan.GraphScenarios) > 0 {
		graph, found, err := migrationNestedMap(item.Object, "spec", "graph")
		if err != nil {
			return nil, migrationBlockedError{reason: "staged graph references cannot be applied because spec.graph is malformed: " + err.Error()}
		}
		if !found {
			return nil, migrationBlockedError{reason: "staged graph references cannot be applied because spec.graph is missing"}
		}
		for nodeID, reference := range plan.GraphScenarios {
			rawNode, exists := graph[nodeID]
			if !exists {
				return nil, migrationBlockedError{reason: fmt.Sprintf("staged graph reference for node %q cannot be applied because the node is missing", nodeID)}
			}
			node, ok := rawNode.(map[string]interface{})
			if !ok {
				return nil, migrationBlockedError{reason: fmt.Sprintf("staged graph reference for node %q cannot be applied because the node is malformed", nodeID)}
			}
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
	return patch, nil
}

func sameJob(job map[string]interface{}, clusterName, jobID string) bool {
	cluster, _ := job["clusterName"].(string)
	id, _ := job["jobId"].(string)
	return cluster == clusterName && id == jobID
}

func hasInlineRegistryCredentials(spec map[string]interface{}) (bool, error) {
	hasCredentials := false
	for _, field := range []string{"token", "username", "password"} {
		value, err := migrationString(spec, field, "spec")
		if err != nil {
			return false, err
		}
		if strings.TrimSpace(value) != "" {
			hasCredentials = true
		}
	}
	return hasCredentials, nil
}

func writeMigrationAnnotation(ctx context.Context, client dynamic.Interface, resource resourceDefinition, item *unstructured.Unstructured, plan migrationPlan) error {
	encoded, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("encode blocked migration for %s/%s: %w", item.GetNamespace(), item.GetName(), err)
	}
	if err := checkAnnotationSize(item, string(encoded)); err != nil {
		return err
	}
	patch, err := json.Marshal(map[string]interface{}{"metadata": map[string]interface{}{
		"resourceVersion": item.GetResourceVersion(),
		"annotations":     map[string]string{AnnotationKey: string(encoded)},
	}})
	if err != nil {
		return fmt.Errorf("encode blocked migration marker for %s/%s: %w", item.GetNamespace(), item.GetName(), err)
	}
	if _, err := client.Resource(resource.gvr).Namespace(item.GetNamespace()).Patch(ctx, item.GetName(), types.MergePatchType, patch, metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("preserve blocked migration for %s/%s: %w", item.GetNamespace(), item.GetName(), err)
	}
	return nil
}
