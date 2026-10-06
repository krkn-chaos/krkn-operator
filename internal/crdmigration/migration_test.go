package crdmigration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestStageAndCompleteScenarioRunMigration(t *testing.T) {
	ctx := context.Background()
	legacy := scenarioRunFixture("operator", "legacy-run")
	client := newMigrationClient(legacy)
	var logs strings.Builder
	logger := migrationTestLogger(&logs)

	if err := Stage(ctx, client, "operator", logger); err != nil {
		t.Fatalf("stage migration: %v", err)
	}
	staged, err := client.Resource(resources[0].gvr).Namespace("operator").Get(ctx, legacy.GetName(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get staged ScenarioRun: %v", err)
	}
	annotation := staged.GetAnnotations()[AnnotationKey]
	if annotation == "" {
		t.Fatal("Stage did not write a migration annotation")
	}
	if strings.Contains(annotation, "do-not-log-this-token") {
		t.Fatal("migration annotation contains inline credentials")
	}

	// Simulate the API server pruning legacy fields while applying the new schema.
	spec, _, _ := unstructured.NestedMap(staged.Object, "spec")
	for _, field := range []string{"scenarioName", "scenarioImage", "registryName", "token", "username", "password"} {
		delete(spec, field)
	}
	if err := unstructured.SetNestedMap(staged.Object, spec, "spec"); err != nil {
		t.Fatalf("set pruned spec: %v", err)
	}
	status, _, _ := unstructured.NestedMap(staged.Object, "status")
	jobs, _, _ := unstructured.NestedSlice(status, "clusterJobs")
	job := jobs[0].(map[string]interface{})
	delete(job, "containerImage")
	jobs[0] = job
	status["clusterJobs"] = jobs
	if err := unstructured.SetNestedMap(staged.Object, status, "status"); err != nil {
		t.Fatalf("set pruned status: %v", err)
	}
	if _, err := client.Resource(resources[0].gvr).Namespace("operator").Update(ctx, staged, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("update pruned ScenarioRun: %v", err)
	}

	if err := Complete(ctx, client, "operator", logger); err != nil {
		t.Fatalf("complete migration: %v", err)
	}
	got, err := client.Resource(resources[0].gvr).Namespace("operator").Get(ctx, legacy.GetName(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get migrated ScenarioRun: %v", err)
	}
	scenario, found, err := unstructured.NestedMap(got.Object, "spec", "scenario")
	if err != nil || !found {
		t.Fatalf("get migrated scenario: found=%t err=%v", found, err)
	}
	if scenario["name"] != "scenario-a" || scenario["private"] != true || scenario["registryName"] != "saved-registry" {
		t.Fatalf("migrated scenario = %#v", scenario)
	}
	for _, field := range []string{"scenarioName", "scenarioImage", "registryName", "token", "username", "password"} {
		if _, found := got.Object["spec"].(map[string]interface{})[field]; found {
			t.Errorf("legacy spec field %q remains", field)
		}
	}
	if retries, found, _ := unstructured.NestedInt64(got.Object, "spec", "maxRetries"); !found || retries != 5 {
		t.Errorf("spec.maxRetries = %d, found=%t; want 5", retries, found)
	}
	if backoff, _, _ := unstructured.NestedString(got.Object, "spec", "retryBackoff"); backoff != "exponential" {
		t.Errorf("spec.retryBackoff = %q, want exponential", backoff)
	}
	if delay, _, _ := unstructured.NestedString(got.Object, "spec", "retryDelay"); delay != "10s" {
		t.Errorf("spec.retryDelay = %q, want 10s", delay)
	}
	if got.GetAnnotations()[AnnotationKey] != "" {
		t.Error("migration annotation was not cleared after completion")
	}

	migratedStatus, _, _ := unstructured.NestedMap(got.Object, "status")
	migratedJobs, _, _ := unstructured.NestedSlice(migratedStatus, "clusterJobs")
	migratedJob := migratedJobs[0].(map[string]interface{})
	if migratedJob["scenarioImage"] != "quay.io/krkn/scenario:scenario-a" || migratedJob["maxRetries"] != int64(2) || migratedJob["maxRetriesConfigured"] != true {
		t.Errorf("migrated cluster job = %#v", migratedJob)
	}
	scores, _, _ := unstructured.NestedSlice(migratedStatus, "resiliencyScores")
	score := scores[0].(map[string]interface{})
	if score["status"] != "calculated" || score["providerName"] != "provider-a" {
		t.Errorf("migrated resiliency score = %#v", score)
	}
	if strings.Contains(logs.String(), "do-not-log-this-token") {
		t.Fatal("migration logs contain inline credentials")
	}
	if !strings.Contains(logs.String(), "custom resource migration phase summary") || !strings.Contains(logs.String(), "objectsWithWarnings") {
		t.Fatalf("migration logs are missing per-phase summary fields: %s", logs.String())
	}

	patchesBeforeRetry := countPatchActions(client.Actions())
	if err := Complete(ctx, client, "operator", logger); err != nil {
		t.Fatalf("repeat completed migration: %v", err)
	}
	if got := countPatchActions(client.Actions()); got != patchesBeforeRetry {
		t.Errorf("idempotent repeat issued %d additional patches", got-patchesBeforeRetry)
	}
}

func TestStageRefreshesSnapshotForLegacyFieldsWrittenDuringRollout(t *testing.T) {
	ctx := context.Background()
	legacy := scenarioRunFixture("operator", "legacy-run")
	client := newMigrationClient(legacy)
	if err := Stage(ctx, client, "operator", logr.Discard()); err != nil {
		t.Fatalf("stage initial migration: %v", err)
	}

	staged, err := client.Resource(resources[0].gvr).Namespace("operator").Get(ctx, legacy.GetName(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get staged ScenarioRun: %v", err)
	}
	status, _, _ := unstructured.NestedMap(staged.Object, "status")
	jobs, _, _ := unstructured.NestedSlice(status, "clusterJobs")
	jobs = append(jobs, map[string]interface{}{
		"clusterName":    "cluster-b",
		"jobId":          "job-b",
		"podName":        "scenario-pod-b",
		"containerImage": "quay.io/krkn/scenario:scenario-b",
		"maxRetries":     int64(4),
	})
	status["clusterJobs"] = jobs
	if err := unstructured.SetNestedMap(staged.Object, status, "status"); err != nil {
		t.Fatalf("set updated legacy status: %v", err)
	}
	if _, err := client.Resource(resources[0].gvr).Namespace("operator").Update(ctx, staged, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("update legacy ScenarioRun during rollout: %v", err)
	}
	if err := Stage(ctx, client, "operator", logr.Discard()); err != nil {
		t.Fatalf("refresh staged migration: %v", err)
	}
	refreshed, err := client.Resource(resources[0].gvr).Namespace("operator").Get(ctx, legacy.GetName(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get refreshed ScenarioRun: %v", err)
	}
	plan := migrationPlan{}
	if err := json.Unmarshal([]byte(refreshed.GetAnnotations()[AnnotationKey]), &plan); err != nil {
		t.Fatalf("decode refreshed migration plan: %v", err)
	}
	if len(plan.ClusterJobImages) != 2 {
		t.Fatalf("staged cluster job images = %#v, want both jobs", plan.ClusterJobImages)
	}
	if plan.ClusterJobImages[1].Image != "quay.io/krkn/scenario:scenario-b" {
		t.Fatalf("refreshed second cluster job image = %q", plan.ClusterJobImages[1].Image)
	}
}

func TestGraphRunMigrationBackfillsScenarioReferences(t *testing.T) {
	ctx := context.Background()
	graphRun := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "krkn.krkn-chaos.dev/v1alpha1",
		"kind":       "KrknGraphRun",
		"metadata":   map[string]interface{}{"name": "legacy-graph", "namespace": "operator"},
		"spec": map[string]interface{}{
			"graph": map[string]interface{}{
				"node-a": map[string]interface{}{"name": "scenario-a", "registryName": "saved-registry", "image": "ignored-image"},
				"_start": map[string]interface{}{"_comment": "metadata only"},
			},
			"targetRequestId": "request-a",
			"targetClusters":  map[string]interface{}{"provider-a": []interface{}{"cluster-a"}},
		},
	}}
	client := newMigrationClient(graphRun)
	if err := Stage(ctx, client, "operator", logr.Discard()); err != nil {
		t.Fatalf("stage graph migration: %v", err)
	}
	if err := Complete(ctx, client, "operator", logr.Discard()); err != nil {
		t.Fatalf("complete graph migration: %v", err)
	}
	got, err := client.Resource(resources[1].gvr).Namespace("operator").Get(ctx, graphRun.GetName(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get migrated GraphRun: %v", err)
	}
	graph, _, _ := unstructured.NestedMap(got.Object, "spec", "graph")
	node := graph["node-a"].(map[string]interface{})
	scenario := node["scenario"].(map[string]interface{})
	if scenario["name"] != "scenario-a" || scenario["private"] != true || scenario["registryName"] != "saved-registry" {
		t.Errorf("migrated graph scenario = %#v", scenario)
	}
	if retries, found, _ := unstructured.NestedInt64(got.Object, "spec", "maxRetries"); !found || retries != 3 {
		t.Errorf("spec.maxRetries = %d, found=%t; want default 3", retries, found)
	}
	if got.GetAnnotations()[AnnotationKey] != "" {
		t.Error("migration annotation was not cleared from GraphRun")
	}
}

func TestScenarioRunWithUnmappableInlineCredentialsIsReportedWithoutGuessing(t *testing.T) {
	item := scenarioRunFixture("operator", "unmappable-run")
	spec, _, _ := unstructured.NestedMap(item.Object, "spec")
	delete(spec, "scenarioName")
	spec["scenarioImage"] = "quay.io/krkn/scenarios:possibly-not-the-tag"
	delete(spec, "registryName")
	spec["token"] = "do-not-log-this-token"
	if err := unstructured.SetNestedMap(item.Object, spec, "spec"); err != nil {
		t.Fatalf("set incomplete legacy spec: %v", err)
	}
	plan := planResource("KrknScenarioRun", item)
	if plan.Scenario != nil {
		t.Fatalf("migration guessed a scenario reference: %#v", plan.Scenario)
	}
	if !containsSubstring(plan.Warnings, "scenario identity is absent") || !containsSubstring(plan.Warnings, "inline registry credentials") {
		t.Fatalf("unmappable legacy fields were not reported: %#v", plan.Warnings)
	}
	encoded, err := jsonMarshal(plan)
	if err != nil {
		t.Fatalf("marshal migration plan: %v", err)
	}
	if strings.Contains(encoded, "do-not-log-this-token") {
		t.Fatal("migration plan contains inline credentials")
	}
}

func TestGraphChildScenarioIdentityIsRecoveredFromParentGraph(t *testing.T) {
	ctx := context.Background()
	graphRun := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "krkn.krkn-chaos.dev/v1alpha1",
		"kind":       "KrknGraphRun",
		"metadata":   map[string]interface{}{"name": "parent-graph", "namespace": "operator"},
		"spec": map[string]interface{}{"graph": map[string]interface{}{
			"node-a": map[string]interface{}{"name": "scenario-from-parent", "registryName": "saved-registry"},
		}},
	}}
	child := scenarioRunFixture("operator", "parent-graph-node-a")
	child.SetLabels(map[string]string{"krkn.dev/graph-run": "parent-graph", "krkn.dev/graph-node": "node-a"})
	spec, _, _ := unstructured.NestedMap(child.Object, "spec")
	spec["scenarioName"] = "stale-child-scenario"
	delete(spec, "registryName")
	if err := unstructured.SetNestedMap(child.Object, spec, "spec"); err != nil {
		t.Fatalf("set pruned child spec: %v", err)
	}
	client := newMigrationClient(child, graphRun)

	if err := Stage(ctx, client, "operator", logr.Discard()); err != nil {
		t.Fatalf("stage graph-child migration: %v", err)
	}
	staged, err := client.Resource(resources[0].gvr).Namespace("operator").Get(ctx, child.GetName(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get staged child: %v", err)
	}
	plan := migrationPlan{}
	if err := json.Unmarshal([]byte(staged.GetAnnotations()[AnnotationKey]), &plan); err != nil {
		t.Fatalf("decode staged plan: %v", err)
	}
	if plan.Scenario["name"] != "scenario-from-parent" || plan.Scenario["registryName"] != "saved-registry" || plan.Scenario["private"] != true {
		t.Fatalf("staged scenario reference = %#v", plan.Scenario)
	}
	if containsSubstring(plan.Warnings, "scenario identity is absent") || containsSubstring(plan.Warnings, "inline registry credentials have no registryName") {
		t.Fatalf("parent graph identity did not clear recovered-field warnings: %#v", plan.Warnings)
	}

	if err := Complete(ctx, client, "operator", logr.Discard()); err != nil {
		t.Fatalf("complete graph-child migration: %v", err)
	}
	got, err := client.Resource(resources[0].gvr).Namespace("operator").Get(ctx, child.GetName(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get migrated child: %v", err)
	}
	reference, found, err := unstructured.NestedMap(got.Object, "spec", "scenario")
	if err != nil || !found {
		t.Fatalf("get migrated scenario reference: found=%t err=%v", found, err)
	}
	if reference["name"] != "scenario-from-parent" || reference["registryName"] != "saved-registry" || reference["private"] != true {
		t.Fatalf("migrated scenario reference = %#v", reference)
	}
}

func TestCompleteMigrationCanResumeAfterStatusPatchFailure(t *testing.T) {
	ctx := context.Background()
	legacy := scenarioRunFixture("operator", "retry-run")
	client := newMigrationClient(legacy)
	if err := Stage(ctx, client, "operator", logr.Discard()); err != nil {
		t.Fatalf("stage migration: %v", err)
	}

	failNextStatusPatch := true
	client.PrependReactor("patch", "krknscenarioruns", func(action ktesting.Action) (bool, runtime.Object, error) {
		if action.GetSubresource() == "status" && failNextStatusPatch {
			failNextStatusPatch = false
			return true, nil, errors.New("temporary status patch failure")
		}
		return false, nil, nil
	})
	if err := Complete(ctx, client, "operator", logr.Discard()); err == nil {
		t.Fatal("expected first completion to fail")
	}
	staged, err := client.Resource(resources[0].gvr).Namespace("operator").Get(ctx, legacy.GetName(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get staged resource after failed completion: %v", err)
	}
	if staged.GetAnnotations()[AnnotationKey] == "" {
		t.Fatal("failed completion cleared the retry marker")
	}
	if err := Complete(ctx, client, "operator", logr.Discard()); err != nil {
		t.Fatalf("retry completion: %v", err)
	}
	got, err := client.Resource(resources[0].gvr).Namespace("operator").Get(ctx, legacy.GetName(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get migrated resource after retry: %v", err)
	}
	if got.GetAnnotations()[AnnotationKey] != "" {
		t.Error("successful retry did not clear the migration marker")
	}
}

func TestStageRejectsSnapshotThatExceedsAnnotationLimit(t *testing.T) {
	ctx := context.Background()
	legacy := scenarioRunFixture("operator", "large-run")
	legacy.SetAnnotations(map[string]string{"existing.example.dev/large": strings.Repeat("x", maxAnnotationsSize)})
	client := newMigrationClient(legacy)
	err := Stage(ctx, client, "operator", logr.Discard())
	if err == nil || !strings.Contains(err.Error(), "annotation size limit") {
		t.Fatalf("Stage error = %v, want annotation size limit", err)
	}
	if patches := countPatchActions(client.Actions()); patches != 0 {
		t.Fatalf("Stage issued %d patches despite oversized snapshot", patches)
	}
}

func TestMigrationIsScopedToOperatorNamespace(t *testing.T) {
	ctx := context.Background()
	operatorRun := scenarioRunFixture("operator", "operator-run")
	otherRun := scenarioRunFixture("other", "other-run")
	client := newMigrationClient(operatorRun, otherRun)
	if err := Stage(ctx, client, "operator", logr.Discard()); err != nil {
		t.Fatalf("stage migration: %v", err)
	}
	operatorObject, err := client.Resource(resources[0].gvr).Namespace("operator").Get(ctx, operatorRun.GetName(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get operator namespace object: %v", err)
	}
	otherObject, err := client.Resource(resources[0].gvr).Namespace("other").Get(ctx, otherRun.GetName(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get other namespace object: %v", err)
	}
	if operatorObject.GetAnnotations()[AnnotationKey] == "" {
		t.Error("operator namespace object was not staged")
	}
	if otherObject.GetAnnotations()[AnnotationKey] != "" {
		t.Error("migration crossed into a namespace outside the operator's Role")
	}
}

func TestMigrationSkipsKindsNotInstalledYet(t *testing.T) {
	client := newMigrationClient()
	client.PrependReactor("list", resources[0].gvr.Resource, func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    resources[0].gvr.Group,
			Resource: resources[0].gvr.Resource,
		}, "")
	})
	if err := Stage(context.Background(), client, "operator", logr.Discard()); err != nil {
		t.Fatalf("Stage should skip an uninstalled CRD: %v", err)
	}
	if err := Complete(context.Background(), client, "operator", logr.Discard()); err != nil {
		t.Fatalf("Complete should skip a CRD absent during a fresh install: %v", err)
	}
}

func scenarioRunFixture(namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "krkn.krkn-chaos.dev/v1alpha1",
		"kind":       "KrknScenarioRun",
		"metadata":   map[string]interface{}{"name": name, "namespace": namespace},
		"spec": map[string]interface{}{
			"scenarioName":    "scenario-a",
			"scenarioImage":   "quay.io/krkn/scenario:scenario-a",
			"registryName":    "saved-registry",
			"token":           "do-not-log-this-token",
			"username":        "legacy-user",
			"password":        "legacy-password",
			"maxRetries":      int64(5),
			"targetRequestId": "request-a",
			"targetClusters":  map[string]interface{}{"provider-a": []interface{}{"cluster-a"}},
		},
		"status": map[string]interface{}{
			"clusterJobs": []interface{}{map[string]interface{}{
				"providerName":   "provider-a",
				"clusterName":    "cluster-a",
				"jobId":          "job-a",
				"podName":        "scenario-pod",
				"containerImage": "quay.io/krkn/scenario:scenario-a",
				"maxRetries":     int64(2),
			}},
			"resiliencyScores": []interface{}{map[string]interface{}{
				"clusterName": "cluster-a",
				"score":       float64(82.5),
			}},
		},
	}}
}

func newMigrationClient(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
	listKinds := map[schema.GroupVersionResource]string{
		resources[0].gvr: "KrknScenarioRunList",
		resources[1].gvr: "KrknGraphRunList",
	}
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds, objects...)
}

func migrationTestLogger(output *strings.Builder) logr.Logger {
	return funcr.New(func(prefix, args string) {
		fmt.Fprintf(output, "%s%s\n", prefix, args)
	}, funcr.Options{})
}

func countPatchActions(actions []ktesting.Action) int {
	count := 0
	for _, action := range actions {
		if action.GetVerb() == "patch" {
			count++
		}
	}
	return count
}

func containsSubstring(values []string, substring string) bool {
	for _, value := range values {
		if strings.Contains(value, substring) {
			return true
		}
	}
	return false
}

func jsonMarshal(value interface{}) (string, error) {
	encoded, err := json.Marshal(value)
	return string(encoded), err
}
