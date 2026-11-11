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
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	logr "github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/mcpserver"
	"github.com/krkn-chaos/krkn-operator/pkg/cloudcreds"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	"github.com/krkn-chaos/krkn-operator/pkg/registry"
)

// opsFileConfigMap builds a file ConfigMap scoped to group ops.
func opsFileConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "file-ops-file",
			Namespace: "default",
			Labels: map[string]string{
				files.AppNameLabel:              files.AppName,
				files.AppComponentLabel:         files.ComponentFile,
				files.FileIDLabel:               "ops-file",
				"group.krkn.krkn-chaos.dev/ops": "true",
			},
		},
		Data: map[string]string{"data.yaml": "x"},
	}
}

// opsRegistrySecret builds a private registry secret scoped to group ops.
func opsRegistrySecret() *corev1.Secret {
	dockerCfg := `{"auths":{"https://reg.example.invalid":{"auth":"dXNlcjpwYXNz"}}}`
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ops-registry",
			Namespace: "default",
			Labels: map[string]string{
				registry.AppNameLabel:           registry.AppName,
				registry.AppComponentLabel:      registry.ComponentRegistry,
				"group.krkn.krkn-chaos.dev/ops": "true",
			},
			Annotations: map[string]string{
				registry.RegistryURLAnnotation:        "https://reg.example.invalid",
				registry.ScenarioRepositoryAnnotation: "scenarios",
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: []byte(dockerCfg),
		},
	}
}

// opsCloudCredentialSecret builds a cloud credential secret scoped to ops.
func opsCloudCredentialSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ops-cred",
			Namespace: "default",
			Labels: map[string]string{
				cloudcreds.AppNameLabel:         cloudcreds.AppName,
				cloudcreds.AppComponentLabel:    cloudcreds.ComponentCloudCredential,
				"group.krkn.krkn-chaos.dev/ops": "true",
			},
		},
		Data: map[string][]byte{"key": []byte("v")},
	}
}

// targetRequestTR1 is the Completed TR with clusters A and B.
func targetRequestTR1() *krknv1alpha1.KrknTargetRequest {
	return &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "tr1", Namespace: "default"},
		Spec:       krknv1alpha1.KrknTargetRequestSpec{UUID: "tr1"},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status: "Completed",
			TargetData: map[string][]krknv1alpha1.ClusterTarget{
				"krkn-operator": {
					{ClusterName: "cluster-a", ClusterAPIURL: urlA},
					{ClusterName: "cluster-b", ClusterAPIURL: urlB},
				},
			},
		},
	}
}

func mcpWriteInput() mcpserver.RunScenarioInput {
	return mcpserver.RunScenarioInput{
		TargetRequestID: "tr1",
		TargetClusters:  map[string][]string{"krkn-operator": {"cluster-a"}},
		ScenarioName:    "scenario-x",
	}
}

func countScenarioRuns(t *testing.T, c client.Client) int {
	t.Helper()
	list := &krknv1alpha1.KrknScenarioRunList{}
	if err := c.List(context.Background(), list, client.InNamespace("default")); err != nil {
		t.Fatalf("list runs: %v", err)
	}
	return len(list.Items)
}

func TestMCPRunScenario_AllowedOnRunCluster(t *testing.T) {
	h, c, _ := newMCPRunsTestHandler(t, targetRequestTR1())
	backend := &mcpBackend{h: h}

	out, err := backend.RunScenario(context.Background(), robotPrincipalFixture(), mcpWriteInput())
	if err != nil {
		t.Fatalf("RunScenario failed: %v", err)
	}
	if out.Kind != "scenario_run" || out.Name == "" {
		t.Errorf("out = %+v, want Kind scenario_run and a name", out)
	}

	created := &krknv1alpha1.KrknScenarioRun{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: out.Name, Namespace: "default"}, created); err != nil {
		t.Fatalf("created run not found: %v", err)
	}
	if created.Spec.OwnerUserID != "robot:ci-bot" {
		t.Errorf("OwnerUserID = %q, want robot:ci-bot", created.Spec.OwnerUserID)
	}
	if created.Labels["krkn.krkn-chaos.dev/owner-user"] != "robot.ci-bot" {
		t.Errorf("owner-user label = %q, want robot.ci-bot", created.Labels["krkn.krkn-chaos.dev/owner-user"])
	}
}

// TestMCPRunScenario_IdempotencyKeyDeduplicates covers retry safety: the same
// client-supplied idempotency key returns the previously created run instead
// of starting a duplicate experiment, a different key creates a new run, and
// an invalid key is rejected before any side effect.
func TestMCPRunScenario_IdempotencyKeyDeduplicates(t *testing.T) {
	h, c, _ := newMCPRunsTestHandler(t, targetRequestTR1())
	backend := &mcpBackend{h: h}
	baseline := countScenarioRuns(t, c)

	in := mcpWriteInput()
	in.IdempotencyKey = "deploy-check-1"
	first, err := backend.RunScenario(context.Background(), robotPrincipalFixture(), in)
	if err != nil {
		t.Fatalf("first RunScenario failed: %v", err)
	}

	// Retry with the same key: same run, no duplicate.
	second, err := backend.RunScenario(context.Background(), robotPrincipalFixture(), in)
	if err != nil {
		t.Fatalf("retry RunScenario failed: %v", err)
	}
	if second.Name != first.Name {
		t.Errorf("retry returned run %q, want the existing run %q", second.Name, first.Name)
	}
	if after := countScenarioRuns(t, c); after != baseline+1 {
		t.Errorf("scenario run count = %d, want %d (no duplicate)", after, baseline+1)
	}

	// A different key creates a new run.
	in.IdempotencyKey = "deploy-check-2"
	third, err := backend.RunScenario(context.Background(), robotPrincipalFixture(), in)
	if err != nil {
		t.Fatalf("RunScenario with a new key failed: %v", err)
	}
	if third.Name == first.Name {
		t.Errorf("new key returned the existing run %q, want a new run", third.Name)
	}
	if after := countScenarioRuns(t, c); after != baseline+2 {
		t.Errorf("scenario run count = %d, want %d", after, baseline+2)
	}

	// An invalid key is rejected without side effects.
	in.IdempotencyKey = "bad key!"
	if _, err := backend.RunScenario(context.Background(), robotPrincipalFixture(), in); err == nil {
		t.Error("invalid idempotency key accepted")
	} else {
		_ = isBackendCode(t, err, mcpserver.CodeInvalidArg)
	}
	if after := countScenarioRuns(t, c); after != baseline+2 {
		t.Errorf("scenario run count = %d, want %d (invalid key must not create a run)", after, baseline+2)
	}
}

// TestMCPRunScenario_IdempotencyKeyScopedToRobot proves a key is scoped to
// the submitting principal: another robot with equivalent permissions using
// the same key gets its own run, not the first robot's.
func TestMCPRunScenario_IdempotencyKeyScopedToRobot(t *testing.T) {
	h, c, _ := newMCPRunsTestHandler(t, targetRequestTR1())
	backend := &mcpBackend{h: h}

	// A second robot with run permission on cluster-a via a different group.
	otherGroup := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "qa", Namespace: "default"},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: "qa",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				urlA: {Actions: []string{"view", "run", "cancel"}},
			},
		},
	}
	if err := c.Create(context.Background(), otherGroup); err != nil {
		t.Fatal(err)
	}
	otherRobot := robotFixture("qa-bot", false)
	otherRobot.Labels = map[string]string{"group.krkn.krkn-chaos.dev/qa": "true"}
	if err := c.Create(context.Background(), otherRobot); err != nil {
		t.Fatal(err)
	}

	in := mcpWriteInput()
	in.IdempotencyKey = "shared-key"
	first, err := backend.RunScenario(context.Background(), robotPrincipalFixture(), in)
	if err != nil {
		t.Fatalf("ci-bot RunScenario failed: %v", err)
	}
	second, err := backend.RunScenario(context.Background(), mcpserver.Principal{RobotName: "qa-bot"}, in)
	if err != nil {
		t.Fatalf("qa-bot RunScenario failed: %v", err)
	}
	if second.Name == first.Name {
		t.Errorf("qa-bot's run %q reused ci-bot's run; idempotency keys must be scoped per robot", second.Name)
	}
}

// TestMCPRunScenario_IdempotencyKeyConcurrent proves a concurrent duplicate
// submission with the same key cannot create two runs: the deterministic run
// name makes Kubernetes reject the second create with AlreadyExists, which the
// backend maps to the existing run.
func TestMCPRunScenario_IdempotencyKeyConcurrent(t *testing.T) {
	h, c, _ := newMCPRunsTestHandler(t, targetRequestTR1())
	backend := &mcpBackend{h: h}
	baseline := countScenarioRuns(t, c)

	in := mcpWriteInput()
	in.IdempotencyKey = "race-key"

	var wg sync.WaitGroup
	results := make([]mcpserver.RunStartedOutput, 2)
	errs := make([]error, 2)
	for i := range 2 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = backend.RunScenario(context.Background(), robotPrincipalFixture(), in)
		}(i)
	}
	wg.Wait()

	for i := range 2 {
		if errs[i] != nil {
			t.Fatalf("concurrent RunScenario %d failed: %v", i, errs[i])
		}
	}
	if results[0].Name != results[1].Name {
		t.Errorf("concurrent submissions returned different runs %q and %q, want the same run", results[0].Name, results[1].Name)
	}
	if after := countScenarioRuns(t, c); after != baseline+1 {
		t.Errorf("scenario run count = %d, want %d (exactly one run from concurrent duplicates)", after, baseline+1)
	}
}

// TestMCPRunScenario_IdempotencyKeyConflictReturnsExisting covers the recovery
// path when the label lookup misses but the deterministic name collides: the
// create returns 409 and the backend returns the robot's existing run instead
// of an error (and never creates a second run).
func TestMCPRunScenario_IdempotencyKeyConflictReturnsExisting(t *testing.T) {
	h, c, _ := newMCPRunsTestHandler(t, targetRequestTR1())
	backend := &mcpBackend{h: h}

	// An existing run under the deterministic name, owned by this robot, with
	// no idempotency label (as after a label-patch failure).
	key := "recover-key"
	existing := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: idempotentRunName("ci-bot", key), Namespace: "default"},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "tr1",
			TargetClusters:  map[string][]string{"krkn-operator": {"cluster-a"}},
			Scenario:        krknv1alpha1.ScenarioReference{Name: "scenario-x"},
			OwnerUserID:     "robot:ci-bot",
		},
	}
	if err := c.Create(context.Background(), existing); err != nil {
		t.Fatal(err)
	}
	baseline := countScenarioRuns(t, c)

	in := mcpWriteInput()
	in.IdempotencyKey = key
	out, err := backend.RunScenario(context.Background(), robotPrincipalFixture(), in)
	if err != nil {
		t.Fatalf("RunScenario after name conflict failed: %v", err)
	}
	if out.Name != existing.Name {
		t.Errorf("out.Name = %q, want the existing run %q", out.Name, existing.Name)
	}
	if after := countScenarioRuns(t, c); after != baseline {
		t.Errorf("scenario run count = %d, want %d (no duplicate on conflict)", after, baseline)
	}
}

// TestMCPRunScenario_IdempotencyKeyConflictNotOtherOwnersRun proves the 409
// recovery path never hands back another principal's run: a run under the
// derived name but owned by someone else propagates the conflict error.
func TestMCPRunScenario_IdempotencyKeyConflictNotOtherOwnersRun(t *testing.T) {
	h, c, _ := newMCPRunsTestHandler(t, targetRequestTR1())
	backend := &mcpBackend{h: h}

	key := "foreign-key"
	foreign := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: idempotentRunName("ci-bot", key), Namespace: "default"},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "tr1",
			TargetClusters:  map[string][]string{"krkn-operator": {"cluster-a"}},
			Scenario:        krknv1alpha1.ScenarioReference{Name: "scenario-x"},
			OwnerUserID:     "someone-else",
		},
	}
	if err := c.Create(context.Background(), foreign); err != nil {
		t.Fatal(err)
	}

	in := mcpWriteInput()
	in.IdempotencyKey = key
	_, err := backend.RunScenario(context.Background(), robotPrincipalFixture(), in)
	_ = isBackendCode(t, err, mcpserver.CodeConflict)
}

// TestMCPRunScenario_IdempotencyKeyReturnsOldest proves duplicate keys resolve
// to the first-created run by creation timestamp, not by lexicographic name
// order (run names are "<scenario>-<uuid8>", so name order is arbitrary).
func TestMCPRunScenario_IdempotencyKeyReturnsOldest(t *testing.T) {
	now := time.Now().UTC()
	// "aaa-run" sorts first by name but was created later; "zzz-run" is older.
	newer := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "aaa-run",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(now),
			Labels:            map[string]string{mcpIdempotencyKeyLabel: "dupe-key"},
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "tr1",
			TargetClusters:  map[string][]string{"krkn-operator": {"cluster-a"}},
			Scenario:        krknv1alpha1.ScenarioReference{Name: "scenario-x"},
			OwnerUserID:     "robot:ci-bot",
		},
	}
	older := newer.DeepCopy()
	older.Name = "zzz-run"
	older.CreationTimestamp = metav1.NewTime(now.Add(-time.Hour))
	h, _, _ := newMCPRunsTestHandler(t, targetRequestTR1(), newer, older)
	backend := &mcpBackend{h: h}

	in := mcpWriteInput()
	in.IdempotencyKey = "dupe-key"
	out, err := backend.RunScenario(context.Background(), robotPrincipalFixture(), in)
	if err != nil {
		t.Fatalf("RunScenario failed: %v", err)
	}
	if out.Name != "zzz-run" {
		t.Errorf("idempotent retry returned %q, want the oldest run %q (not name-sorted %q)", out.Name, "zzz-run", "aaa-run")
	}
}

// TestMCPRunScenario_IdempotencyKeyWithCustomRunName covers the combination of
// a client-supplied customRunName and an idempotency key: the user's name is
// preserved and a retry with the same key and name returns the same run.
func TestMCPRunScenario_IdempotencyKeyWithCustomRunName(t *testing.T) {
	h, c, _ := newMCPRunsTestHandler(t, targetRequestTR1())
	backend := &mcpBackend{h: h}
	baseline := countScenarioRuns(t, c)

	in := mcpWriteInput()
	in.IdempotencyKey = "named-key"
	in.CustomRunName = "my-named-run"
	first, err := backend.RunScenario(context.Background(), robotPrincipalFixture(), in)
	if err != nil {
		t.Fatalf("first RunScenario failed: %v", err)
	}
	if first.Name != "my-named-run" {
		t.Errorf("run name = %q, want the client-supplied customRunName", first.Name)
	}
	second, err := backend.RunScenario(context.Background(), robotPrincipalFixture(), in)
	if err != nil {
		t.Fatalf("retry RunScenario failed: %v", err)
	}
	if second.Name != first.Name {
		t.Errorf("retry returned run %q, want the existing run %q", second.Name, first.Name)
	}
	if after := countScenarioRuns(t, c); after != baseline+1 {
		t.Errorf("scenario run count = %d, want %d (no duplicate)", after, baseline+1)
	}
}

// TestMCPPermissions_ReflectGroupLabelChangeImmediately proves permission
// changes take effect on the next request with no caching: removing the
// robot's group label hides runs that were visible a moment before.
func TestMCPPermissions_ReflectGroupLabelChangeImmediately(t *testing.T) {
	h, c, _ := newMCPRunsTestHandler(t)
	backend := &mcpBackend{h: h}

	out, err := backend.ListScenarioRuns(context.Background(), robotPrincipalFixture(), mcpserver.ListScenarioRunsInput{})
	if err != nil {
		t.Fatalf("ListScenarioRuns failed: %v", err)
	}
	if out.Total != 1 || out.Runs[0].Name != "r1" {
		t.Fatalf("before label change: runs = %+v, want only r1", out.Runs)
	}

	// Strip the sre group label from the robot, as an admin removing
	// membership would.
	robot := &krknv1alpha1.KrknRobot{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "ci-bot", Namespace: "default"}, robot); err != nil {
		t.Fatal(err)
	}
	updated := robot.DeepCopy()
	delete(updated.Labels, "group.krkn.krkn-chaos.dev/sre")
	if err := c.Patch(context.Background(), updated, client.MergeFrom(robot)); err != nil {
		t.Fatal(err)
	}

	after, err := backend.ListScenarioRuns(context.Background(), robotPrincipalFixture(), mcpserver.ListScenarioRunsInput{})
	if err != nil {
		t.Fatalf("ListScenarioRuns after label change failed: %v", err)
	}
	if after.Total != 0 {
		t.Errorf("after removing the group label: Total = %d, want 0 (permissions must not be cached)", after.Total)
	}
}

func TestMCPRunScenario_DeniedWithoutRun(t *testing.T) {
	h, c, _ := newMCPRunsTestHandler(t, targetRequestTR1())
	backend := &mcpBackend{h: h}

	in := mcpWriteInput()
	in.TargetClusters = map[string][]string{"krkn-operator": {"cluster-a", "cluster-b"}}
	_, err := backend.RunScenario(context.Background(), robotPrincipalFixture(), in)
	be := isBackendCode(t, err, mcpserver.CodeForbidden)

	if after := countScenarioRuns(t, c); after != 2 {
		t.Errorf("scenario run count = %d, want 2 (no new CR; message %q)", after, be.Message)
	}
}

func TestMCPRunScenario_ProtectedFileDenied(t *testing.T) {
	h, c, _ := newMCPRunsTestHandler(t, targetRequestTR1(), opsFileConfigMap())
	backend := &mcpBackend{h: h}

	in := mcpWriteInput()
	in.FileReferences = []mcpserver.FileReferenceInput{{FileID: "ops-file", MountPath: "/tmp/f"}}
	_, err := backend.RunScenario(context.Background(), robotPrincipalFixture(), in)
	_ = isBackendCode(t, err, mcpserver.CodeForbidden)
	if after := countScenarioRuns(t, c); after != 2 {
		t.Errorf("scenario run count = %d, want 2 (no new CR)", after)
	}
}

func TestMCPRunScenario_ProtectedRegistryDenied(t *testing.T) {
	h, c, _ := newMCPRunsTestHandler(t, targetRequestTR1(), opsRegistrySecret())
	backend := &mcpBackend{h: h}

	in := mcpWriteInput()
	in.PrivateRegistry = true
	in.RegistryName = "ops-registry"
	_, err := backend.RunScenario(context.Background(), robotPrincipalFixture(), in)
	be := isBackendCode(t, err, mcpserver.CodeForbidden)
	_ = be
	if after := countScenarioRuns(t, c); after != 2 {
		t.Errorf("scenario run count = %d, want 2 (no new CR)", after)
	}
}

func TestMCPRunScenario_ProtectedCloudCredentialDenied(t *testing.T) {
	h, c, _ := newMCPRunsTestHandler(t, targetRequestTR1(), opsCloudCredentialSecret())
	backend := &mcpBackend{h: h}

	in := mcpWriteInput()
	in.CloudCredentialRef = "ops-cred"
	_, err := backend.RunScenario(context.Background(), robotPrincipalFixture(), in)
	_ = isBackendCode(t, err, mcpserver.CodeForbidden)
	if after := countScenarioRuns(t, c); after != 2 {
		t.Errorf("scenario run count = %d, want 2 (no new CR)", after)
	}
}

func TestMCPReplayScenarioRun_SubmitsWithoutReturningSecrets(t *testing.T) {
	// r1 replays a run that targets cluster-a only, with a secret env value.
	runA := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "r1a", Namespace: "default"},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "tr1",
			TargetClusters:  map[string][]string{"krkn-operator": {"cluster-a"}},
			Scenario:        krknv1alpha1.ScenarioReference{Name: "scenario-x", Private: &[]bool{false}[0]},
			Environment:     map[string]string{"PASSWORD": "s3cr3t"},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ClusterName: "cluster-a", ClusterAPIURL: urlA, JobID: "job-r1a", Phase: "Succeeded"},
			},
		},
	}
	h, c, _ := newMCPRunsTestHandler(t, targetRequestTR1(), runA)
	backend := &mcpBackend{h: h}
	before := countScenarioRuns(t, c)

	out, err := backend.ReplayScenarioRun(context.Background(), robotPrincipalFixture(), mcpserver.ReplayScenarioRunInput{ScenarioRunName: "r1a"})
	if err != nil {
		t.Fatalf("ReplayScenarioRun failed: %v", err)
	}
	if countScenarioRuns(t, c) != before+1 {
		t.Fatalf("no new run created by replay")
	}

	created := &krknv1alpha1.KrknScenarioRun{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: out.Name, Namespace: "default"}, created); err != nil {
		t.Fatalf("replayed run not found: %v", err)
	}
	if created.Spec.Environment["PASSWORD"] != "s3cr3t" {
		t.Errorf("replayed run lost original env (controller needs it)")
	}
	if created.Spec.TargetClusters["krkn-operator"][0] != "cluster-a" {
		t.Errorf("replayed run targets = %v, want cluster-a only", created.Spec.TargetClusters)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if containsStr(string(raw), "s3cr3t") {
		t.Errorf("replay output leaks env values: %s", raw)
	}
}

func TestMCPReplayScenarioRun_HiddenNotFound(t *testing.T) {
	h, _, _ := newMCPRunsTestHandler(t, targetRequestTR1())
	backend := &mcpBackend{h: h}

	_, err := backend.ReplayScenarioRun(context.Background(), robotPrincipalFixture(), mcpserver.ReplayScenarioRunInput{ScenarioRunName: "r2"})
	_ = isBackendCode(t, err, mcpserver.CodeNotFound)
}

func TestMCPReplayScenarioRun_StaleTargetRequest(t *testing.T) {
	// No tr1 exists: the original target request is gone.
	h, _, _ := newMCPRunsTestHandler(t)
	backend := &mcpBackend{h: h}

	_, err := backend.ReplayScenarioRun(context.Background(), robotPrincipalFixture(), mcpserver.ReplayScenarioRunInput{ScenarioRunName: "r1"})
	be := isBackendCode(t, err, mcpserver.CodeInvalidArg)
	if !strings.Contains(be.Message, "krkn_list_clusters") {
		t.Errorf("message %q does not point at krkn_list_clusters", be.Message)
	}
}

func TestMCPCancelJob_RequiresCancelOnJobCluster(t *testing.T) {
	h, _, _ := newMCPRunsTestHandler(t, targetRequestTR1())
	backend := &mcpBackend{h: h}

	out, err := backend.CancelJob(context.Background(), robotPrincipalFixture(), mcpserver.CancelJobInput{ScenarioRunName: "r1", JobID: jobAFixture})
	if err != nil {
		t.Fatalf("CancelJob on cluster-a failed: %v", err)
	}
	if out.Message == "" {
		t.Error("empty message")
	}
	run := &krknv1alpha1.KrknScenarioRun{}
	if err := h.client.Get(context.Background(), client.ObjectKey{Name: "r1", Namespace: "default"}, run); err != nil {
		t.Fatal(err)
	}
	cancelled := false
	for _, job := range run.Status.ClusterJobs {
		if job.JobID == jobAFixture && job.CancelRequested {
			cancelled = true
		}
	}
	if !cancelled {
		t.Error("CancelRequested not set on job-a")
	}

	// A view-only robot must be denied.
	viewOnly := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "viewers", Namespace: "default"},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: "viewers",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				urlA: {Actions: []string{"view"}},
			},
		},
	}
	if err := h.client.Create(context.Background(), viewOnly); err != nil {
		t.Fatal(err)
	}
	viewerRobot := robotFixture("viewer-bot", false)
	viewerRobot.Labels = map[string]string{"group.krkn.krkn-chaos.dev/viewers": "true"}
	if err := h.client.Create(context.Background(), viewerRobot); err != nil {
		t.Fatal(err)
	}

	_, err = backend.CancelJob(context.Background(), mcpserver.Principal{RobotName: "viewer-bot"}, mcpserver.CancelJobInput{ScenarioRunName: "r1", JobID: jobAFixture})
	_ = isBackendCode(t, err, mcpserver.CodeForbidden)
}

func TestMCPDeleteScenarioRun_RequiresCancelOnAll(t *testing.T) {
	h, _, _ := newMCPRunsTestHandler(t, targetRequestTR1())
	backend := &mcpBackend{h: h}

	// r1 spans A (cancel ok) and B (no permission) -> forbidden.
	_, err := backend.DeleteCompleteScenarioRun(context.Background(), robotPrincipalFixture(), mcpserver.ScenarioRunRefInput{ScenarioRunName: "r1"})
	_ = isBackendCode(t, err, mcpserver.CodeForbidden)

	// A run only on A is deleted.
	onlyA := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "r3", Namespace: "default"},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "tr1",
			TargetClusters:  map[string][]string{"krkn-operator": {"cluster-a"}},
			Scenario:        krknv1alpha1.ScenarioReference{Name: "scenario-x"},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ClusterName: "cluster-a", ClusterAPIURL: urlA, JobID: "job-a3", Phase: "Succeeded"},
			},
		},
	}
	if err := h.client.Create(context.Background(), onlyA); err != nil {
		t.Fatal(err)
	}
	if _, err := backend.DeleteCompleteScenarioRun(context.Background(), robotPrincipalFixture(), mcpserver.ScenarioRunRefInput{ScenarioRunName: "r3"}); err != nil {
		t.Fatalf("DeleteCompleteScenarioRun(r3) failed: %v", err)
	}
	err = h.client.Get(context.Background(), client.ObjectKey{Name: "r3", Namespace: "default"}, &krknv1alpha1.KrknScenarioRun{})
	if err == nil {
		t.Error("r3 still exists after delete")
	}
}

func TestMCPAudit_WriteLogsNoEnv(t *testing.T) {
	var captured []string
	sink := funcr.New(func(prefix, args string) {
		captured = append(captured, args)
	}, funcr.Options{})
	restore := setGlobalLogger(sink)
	defer restore()

	h, _, _ := newMCPRunsTestHandler(t, targetRequestTR1())
	backend := &mcpBackend{h: h}

	// Drive the tool through the real MCP handler so the audit wrapper runs.
	verify := func(ctx context.Context, token string) (mcpserver.RobotIdentity, error) {
		return mcpserver.RobotIdentity{Name: "ci-bot", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}
	srv := httptest.NewServer(mcpserver.NewHandler(backend, verify, "test"))
	defer srv.Close()

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "audit-test", Version: "v"}, nil)
	session, err := mcpClient.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:   srv.URL,
		HTTPClient: &http.Client{Transport: bearerTransport{token: "krkn_robot_ci-bot.x"}},
	}, nil)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer func() { _ = session.Close() }()

	in := mcpWriteInput()
	in.Environment = map[string]string{"PW": "topsecretvalue"}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		t.Fatalf("unmarshal args: %v", err)
	}
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "krkn_run_scenario", Arguments: args}); err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	found := false
	for _, line := range captured {
		if strings.Contains(line, "krkn_run_scenario") && strings.Contains(line, "ci-bot") {
			found = true
			if strings.Contains(line, "topsecretvalue") {
				t.Errorf("audit line leaks env value: %s", line)
			}
		}
	}
	if !found {
		t.Errorf("no audit line for krkn_run_scenario found in %d lines: %v", len(captured), captured)
	}
}

// bearerTransport adds an Authorization header to every request.
type bearerTransport struct {
	token string
}

func (bt bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+bt.token)
	return http.DefaultTransport.RoundTrip(clone)
}

var _ runtime.Object = (*krknv1alpha1.KrknScenarioRun)(nil)

// setGlobalLogger swaps controller-runtime's global logger and returns a
// restore function.
func setGlobalLogger(sink logr.Logger) func() {
	old := logr.New(log.Log.GetSink())
	log.SetLogger(sink)
	return func() { log.SetLogger(old) }
}

func TestMCPRunScenario_ForbiddenMessageGeneric(t *testing.T) {
	h, _, _ := newMCPRunsTestHandler(t, targetRequestTR1())
	backend := &mcpBackend{h: h}

	in := mcpWriteInput()
	in.TargetClusters = map[string][]string{"krkn-operator": {"cluster-a", "cluster-b"}}
	_, err := backend.RunScenario(context.Background(), robotPrincipalFixture(), in)
	be := isBackendCode(t, err, mcpserver.CodeForbidden)
	if containsStr(be.Message, "cluster-b") || containsStr(be.Message, urlB) {
		t.Errorf("forbidden message leaks hidden cluster: %q", be.Message)
	}
}

func TestMCPReplayScenarioRun_HiddenTargetNoLeak(t *testing.T) {
	// r1 spans A (visible) and B (hidden): replay is denied and the message
	// must not name cluster-b or its URL.
	h, _, _ := newMCPRunsTestHandler(t, targetRequestTR1())
	backend := &mcpBackend{h: h}

	_, err := backend.ReplayScenarioRun(context.Background(), robotPrincipalFixture(), mcpserver.ReplayScenarioRunInput{ScenarioRunName: "r1"})
	be := isBackendCode(t, err, mcpserver.CodeForbidden)
	if containsStr(be.Message, "cluster-b") || containsStr(be.Message, urlB) {
		t.Errorf("replay denial leaks hidden cluster: %q", be.Message)
	}
}
