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

	out, err := backend.CancelJob(context.Background(), robotPrincipalFixture(), mcpserver.CancelJobInput{ScenarioRunName: "r1", JobID: "job-a"})
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
		if job.JobID == "job-a" && job.CancelRequested {
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

	_, err = backend.CancelJob(context.Background(), mcpserver.Principal{RobotName: "viewer-bot"}, mcpserver.CancelJobInput{ScenarioRunName: "r1", JobID: "job-a"})
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
