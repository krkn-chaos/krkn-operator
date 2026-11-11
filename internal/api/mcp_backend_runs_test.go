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
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/mcpserver"
	"github.com/krkn-chaos/krkn-operator/pkg/cloudcreds"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	"github.com/krkn-chaos/krkn-operator/pkg/registry"
)

// newMCPRunsTestHandler builds the standard Task 8 fixture: group sre grants
// view+run+cancel on URL A; URL B is unlisted; robot ci-bot is in sre; run r1
// has jobs on A and B; run r2 only on B.
func newMCPRunsTestHandler(t *testing.T, objs ...runtime.Object) (*Handler, client.Client, *fake.Clientset) {
	t.Helper()
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	now := time.Now().UTC()
	private := false

	group := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "sre", Namespace: "default"},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: "sre",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				urlA: {Actions: []string{"view", "run", "cancel"}},
			},
		},
	}
	robot := robotFixture("ci-bot", false)
	robot.Labels = map[string]string{"group.krkn.krkn-chaos.dev/sre": "true"}

	r1 := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "r1",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(now.Add(-time.Hour)),
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "tr1",
			TargetClusters:  map[string][]string{"krkn-operator": {"cluster-a", "cluster-b"}},
			Scenario:        krknv1alpha1.ScenarioReference{Name: "scenario-x", Private: &private},
			Environment:     map[string]string{"PASSWORD": "s3cr3t"},
			Files:           []krknv1alpha1.FileMount{{Name: "f", Content: "c2VjcmV0", MountPath: "/tmp/f"}},
			OwnerUserID:     "robot:ci-bot",
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "PartiallyFailed",
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ProviderName: "krkn-operator", ClusterName: "cluster-a", ClusterAPIURL: urlA, JobID: jobAFixture, Phase: "Succeeded", PodName: "pod-a"},
				{ProviderName: "krkn-operator", ClusterName: "cluster-b", ClusterAPIURL: urlB, JobID: "job-b", Phase: "Failed", PodName: "pod-b"},
			},
			ResiliencyScores: []krknv1alpha1.ClusterResiliencyScore{
				{ClusterName: "cluster-a", Score: 80},
				{ClusterName: "cluster-b", Score: 20},
			},
			ReportStatus: &krknv1alpha1.ReportStatus{
				Generated: true,
				Location:  "krkn-report-r1",
			},
		},
	}
	r2 := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "r2",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "tr1",
			TargetClusters:  map[string][]string{"krkn-operator": {"cluster-b"}},
			Scenario:        krknv1alpha1.ScenarioReference{Name: "scenario-x", Private: &private},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Succeeded",
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ProviderName: "krkn-operator", ClusterName: "cluster-b", ClusterAPIURL: urlB, JobID: "job-b2", Phase: "Succeeded"},
			},
		},
	}

	all := append([]runtime.Object{group, robot, r1, r2}, objs...)
	c := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(all...).
		WithStatusSubresource(&krknv1alpha1.KrknRobot{}, &krknv1alpha1.KrknScenarioRun{}).
		Build()

	clientset := fake.NewSimpleClientset()
	h := NewTestHandler(c, clientset, "default", "localhost:50051")
	return h, c, clientset
}

func robotPrincipalFixture() mcpserver.Principal {
	return mcpserver.Principal{RobotName: "ci-bot"}
}

func isBackendCode(t *testing.T, err error, code mcpserver.ErrorCode) *mcpserver.BackendError {
	t.Helper()
	var be *mcpserver.BackendError
	if !errorsAsBackend(err, &be) {
		t.Fatalf("err %v (%T) is not a BackendError", err, err)
	}
	if be.Code != code {
		t.Fatalf("code = %s, want %s (message %q)", be.Code, code, be.Message)
	}
	return be
}

func errorsAsBackend(err error, target **mcpserver.BackendError) bool {
	for err != nil {
		if be, ok := err.(*mcpserver.BackendError); ok {
			*target = be
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

func TestMCPListScenarioRuns_FiltersInvisibleRuns(t *testing.T) {
	h, _, _ := newMCPRunsTestHandler(t)
	backend := &mcpBackend{h: h}

	out, err := backend.ListScenarioRuns(context.Background(), robotPrincipalFixture(), mcpserver.ListScenarioRunsInput{})
	if err != nil {
		t.Fatalf("ListScenarioRuns failed: %v", err)
	}
	if out.Total != 1 {
		t.Fatalf("Total = %d, want 1 (r2 must be invisible)", out.Total)
	}
	if len(out.Runs) != 1 || out.Runs[0].Name != "r1" {
		t.Fatalf("runs = %+v, want [r1]", out.Runs)
	}
	if !out.Runs[0].PartialView {
		t.Error("r1.PartialView = false, want true")
	}
}

func TestMCPListScenarioRuns_LimitClamp(t *testing.T) {
	h, _, _ := newMCPRunsTestHandler(t)
	backend := &mcpBackend{h: h}

	out, err := backend.ListScenarioRuns(context.Background(), robotPrincipalFixture(), mcpserver.ListScenarioRunsInput{Limit: 500})
	if err != nil {
		t.Fatalf("ListScenarioRuns failed: %v", err)
	}
	if out.Limit != 100 {
		t.Errorf("Limit = %d, want 100 (clamped)", out.Limit)
	}
}

func TestMCPGetScenarioRun_HiddenIsNotFound(t *testing.T) {
	h, _, _ := newMCPRunsTestHandler(t)
	backend := &mcpBackend{h: h}

	_, err := backend.GetScenarioRun(context.Background(), robotPrincipalFixture(), mcpserver.ScenarioRunRefInput{ScenarioRunName: "r2"})
	if err == nil {
		t.Fatalf("GetScenarioRun(r2) succeeded, want not_found")
	}
	hiddenErr := isBackendCode(t, err, mcpserver.CodeNotFound)
	if hiddenErr.Message != `scenario run "r2" not found` {
		t.Errorf("hidden run message = %q", hiddenErr.Message)
	}

	_, err = backend.GetScenarioRun(context.Background(), robotPrincipalFixture(), mcpserver.ScenarioRunRefInput{ScenarioRunName: "no-such-run"})
	if err == nil {
		t.Fatal("GetScenarioRun(no-such-run) succeeded, want not_found")
	}
	missingErr := isBackendCode(t, err, mcpserver.CodeNotFound)

	wantFormat := `scenario run "no-such-run" not found`
	if missingErr.Message != wantFormat {
		t.Errorf("missing run message = %q, want %q (same shape as hidden: existence must not leak)", missingErr.Message, wantFormat)
	}
}

func TestMCPGetJobLogs_HiddenJobNotFound(t *testing.T) {
	h, _, _ := newMCPRunsTestHandler(t)
	backend := &mcpBackend{h: h}

	_, err := backend.GetJobLogs(context.Background(), robotPrincipalFixture(), mcpserver.JobLogsInput{
		ScenarioRunName: "r1",
		JobID:           "job-b",
	})
	_ = isBackendCode(t, err, mcpserver.CodeNotFound)
}

func TestMCPGetJobLogs_Bounds(t *testing.T) {
	h, _, clientset := newMCPRunsTestHandler(t)

	var captured *corev1.PodLogOptions
	clientset.PrependReactor("get", "pods/log", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if generic, ok := action.(k8stesting.GenericAction); ok {
			if opts, ok := generic.GetValue().(*corev1.PodLogOptions); ok {
				captured = opts
			}
		}
		return true, &runtime.Unknown{Raw: []byte("fake logs")}, nil
	})

	backend := &mcpBackend{h: h}
	out, err := backend.GetJobLogs(context.Background(), robotPrincipalFixture(), mcpserver.JobLogsInput{
		ScenarioRunName: "r1",
		JobID:           jobAFixture,
		TailLines:       5000,
	})
	if err != nil {
		t.Fatalf("GetJobLogs failed: %v", err)
	}
	if out.JobID != jobAFixture || out.PodName != "pod-a" {
		t.Errorf("out = %+v", out)
	}
	if captured == nil {
		t.Fatal("PodLogOptions were not captured")
	}
	if captured.TailLines != nil {
		t.Errorf("TailLines = %v, want nil (the tail is applied client-side, after report-block filtering)", captured.TailLines)
	}
	if captured.LimitBytes == nil || *captured.LimitBytes != 262144 {
		t.Errorf("LimitBytes = %v, want 262144", captured.LimitBytes)
	}
	if captured.Container != "scenario" {
		t.Errorf("Container = %q, want scenario", captured.Container)
	}
}

// TestMCPGetJobLogs_FiltersReportBlocksBeforeTail mirrors the REST log path:
// the ===KRKN_REPORT_*=== blocks (base64 report payloads) must be stripped
// BEFORE the tail window is applied, so a tail starting mid-block can never
// leak encoded report content.
func TestMCPGetJobLogs_FiltersReportBlocksBeforeTail(t *testing.T) {
	h, _, clientset := newMCPRunsTestHandler(t)

	var captured *corev1.PodLogOptions
	clientset.PrependReactor("get", "pods/log", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if generic, ok := action.(k8stesting.GenericAction); ok {
			if opts, ok := generic.GetValue().(*corev1.PodLogOptions); ok {
				captured = opts
			}
		}
		logs := strings.Join([]string{
			"line-1",
			"line-2",
			"===KRKN_REPORT_HTML_START===",
			"ZXhhbXBsZS1iYXNlNjQtcmVwb3J0LXBheWxvYWQ=",
			"===KRKN_REPORT_HTML_END===",
			"line-3",
		}, "\n") + "\n"
		return true, &runtime.Unknown{Raw: []byte(logs)}, nil
	})

	backend := &mcpBackend{h: h}
	out, err := backend.GetJobLogs(context.Background(), robotPrincipalFixture(), mcpserver.JobLogsInput{
		ScenarioRunName: "r1",
		JobID:           jobAFixture,
		TailLines:       2,
	})
	if err != nil {
		t.Fatalf("GetJobLogs failed: %v", err)
	}
	if captured == nil || captured.TailLines != nil {
		t.Fatalf("TailLines sent to Kubernetes = %v, want nil", captured)
	}
	if out.Logs != "line-2\nline-3\n" {
		t.Errorf("Logs = %q, want the last two report-free lines", out.Logs)
	}
	if containsStr(out.Logs, "KRKN_REPORT") || containsStr(out.Logs, "ZXhhbXBsZS1iYXNlNjQtcmVwb3J0LXBheWxvYWQ") {
		t.Errorf("logs leak report block content: %q", out.Logs)
	}
}

func TestMCPGetJobLogs_NegativeTail(t *testing.T) {
	h, _, _ := newMCPRunsTestHandler(t)
	backend := &mcpBackend{h: h}

	_, err := backend.GetJobLogs(context.Background(), robotPrincipalFixture(), mcpserver.JobLogsInput{
		ScenarioRunName: "r1",
		JobID:           jobAFixture,
		TailLines:       -1,
	})
	_ = isBackendCode(t, err, mcpserver.CodeInvalidArg)
}

// TestMCPGetScenarioRun_ClusterNameCollisionAcrossProviders guards against the
// authorization bypass where flattening TargetData into a name->URL map let a
// duplicate cluster name resolve to a provider the robot can view. The run has
// no jobs yet, so visibility is decided via its declared targets.
func TestMCPGetScenarioRun_ClusterNameCollisionAcrossProviders(t *testing.T) {
	rColl := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "r-coll",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(time.Now().UTC()),
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "tr-coll",
			// cluster-a under prov-b resolves to URL B (no view); the same
			// name under prov-a resolves to URL A (view granted).
			TargetClusters: map[string][]string{"prov-b": {"cluster-a"}},
			Scenario:       krknv1alpha1.ScenarioReference{Name: "scenario-x"},
		},
	}
	h, _, _ := newMCPRunsTestHandler(t, rColl, collisionTargetRequest("tr-coll"))
	backend := &mcpBackend{h: h}

	out, err := backend.ListScenarioRuns(context.Background(), robotPrincipalFixture(), mcpserver.ListScenarioRunsInput{})
	if err != nil {
		t.Fatalf("ListScenarioRuns failed: %v", err)
	}
	// r1 is legitimately visible; r-coll must not be.
	if out.Total != 1 || len(out.Runs) != 1 || out.Runs[0].Name != "r1" {
		t.Errorf("runs = %+v, want only r1 (r-coll must resolve within prov-b, whose URL the robot cannot view)", out.Runs)
	}

	_, err = backend.GetScenarioRun(context.Background(), robotPrincipalFixture(), mcpserver.ScenarioRunRefInput{ScenarioRunName: "r-coll"})
	_ = isBackendCode(t, err, mcpserver.CodeNotFound)
}

// viewOnlyFixtures builds a viewers group granting view (but not run) on URL A
// plus a robot in it.
func viewOnlyFixtures() []runtime.Object {
	group := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "viewers", Namespace: "default"},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: "viewers",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				urlA: {Actions: []string{"view"}},
			},
		},
	}
	viewBot := robotFixture("view-bot", false)
	viewBot.Labels = map[string]string{"group.krkn.krkn-chaos.dev/viewers": "true"}
	return []runtime.Object{group, viewBot}
}

// runnableScenarioRun builds a run confined to cluster-a (URL A) with a job,
// sensitive environment values and the given cloud credential reference.
func runnableScenarioRun(name, cloudCredRef string) *krknv1alpha1.KrknScenarioRun {
	return &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(time.Now().UTC().Add(-time.Minute)),
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID:    "tr1",
			TargetClusters:     map[string][]string{"krkn-operator": {"cluster-a"}},
			Scenario:           krknv1alpha1.ScenarioReference{Name: "scenario-x"},
			Environment:        map[string]string{"PASSWORD": "hunter2"},
			CloudCredentialRef: cloudCredRef,
			OwnerUserID:        "robot:ci-bot",
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			Phase: "Running",
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ProviderName: "krkn-operator", ClusterName: "cluster-a", ClusterAPIURL: urlA, JobID: "job-" + name, Phase: "Running", PodName: "pod-" + name},
			},
		},
	}
}

// restrictedCloudCredential builds a cloud credential Secret scoped to the ops
// group (not sre), so the ci-bot robot cannot use it.
func restrictedCloudCredential(name string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels: map[string]string{
				cloudcreds.AppNameLabel:                   cloudcreds.AppName,
				cloudcreds.AppComponentLabel:              cloudcreds.ComponentCloudCredential,
				"group.krkn.krkn-chaos.dev/ops":           "true",
				"cloudcreds.krkn.krkn-chaos.dev/provider": "aws",
			},
		},
		Data: map[string][]byte{"key": []byte("hunter2")},
	}
}

// publicCloudCredential builds an available-to-all cloud credential Secret.
func publicCloudCredential(name string) *corev1.Secret {
	secret := restrictedCloudCredential(name)
	secret.Labels[cloudcreds.AvailableToAllLabel] = "true"
	return secret
}

// TestMCPGetScenarioRunConfig_RequiresRunPermission mirrors the REST policy:
// krkn_get_scenario_run_config must require run permission on ALL target
// clusters, not just view on any visible one.
func TestMCPGetScenarioRunConfig_RequiresRunPermission(t *testing.T) {
	objs := append(viewOnlyFixtures(),
		targetRequestTR1(),
		runnableScenarioRun("r-run", "cred-ops"),
		runnableScenarioRun("r-run-pub", "cred-public"),
		restrictedCloudCredential("cred-ops"),
		publicCloudCredential("cred-public"),
	)
	h, _, _ := newMCPRunsTestHandler(t, objs...)
	backend := &mcpBackend{h: h}
	ctx := context.Background()

	// View-only robot: forbidden even though the run is visible.
	_, err := backend.GetScenarioRunConfig(ctx, mcpserver.Principal{RobotName: "view-bot"}, mcpserver.ScenarioRunRefInput{ScenarioRunName: "r-run"})
	be := isBackendCode(t, err, mcpserver.CodeForbidden)
	if containsStr(be.Message, urlA) || containsStr(be.Message, urlB) {
		t.Errorf("forbidden message leaks cluster URLs: %q", be.Message)
	}

	// Run permission on cluster-a but not on cluster-b (r1 spans both): forbidden.
	_, err = backend.GetScenarioRunConfig(ctx, robotPrincipalFixture(), mcpserver.ScenarioRunRefInput{ScenarioRunName: "r1"})
	_ = isBackendCode(t, err, mcpserver.CodeForbidden)

	// Run permission on every target: allowed, with a redacted config.
	out, err := backend.GetScenarioRunConfig(ctx, robotPrincipalFixture(), mcpserver.ScenarioRunRefInput{ScenarioRunName: "r-run"})
	if err != nil {
		t.Fatalf("GetScenarioRunConfig(r-run) failed: %v", err)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if containsStr(string(raw), "hunter2") {
		t.Errorf("config leaks secret values: %s", raw)
	}
	if out.CloudCredentialRef != "" {
		t.Errorf("CloudCredentialRef = %q, want empty (credential not accessible to the robot)", out.CloudCredentialRef)
	}

	// An accessible credential reference is preserved.
	pubOut, err := backend.GetScenarioRunConfig(ctx, robotPrincipalFixture(), mcpserver.ScenarioRunRefInput{ScenarioRunName: "r-run-pub"})
	if err != nil {
		t.Fatalf("GetScenarioRunConfig(r-run-pub) failed: %v", err)
	}
	if pubOut.CloudCredentialRef != "cred-public" {
		t.Errorf("CloudCredentialRef = %q, want cred-public (accessible credential must be preserved)", pubOut.CloudCredentialRef)
	}
}

func TestMCPGetScenarioRunConfig_Redacted(t *testing.T) {
	h, _, _ := newMCPRunsTestHandler(t, targetRequestTR1(), runnableScenarioRun("r-run", ""), restrictedCloudCredential("cred-ops"))
	backend := &mcpBackend{h: h}

	out, err := backend.GetScenarioRunConfig(context.Background(), robotPrincipalFixture(), mcpserver.ScenarioRunRefInput{ScenarioRunName: "r-run"})
	if err != nil {
		t.Fatalf("GetScenarioRunConfig failed: %v", err)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if containsStr(string(raw), "hunter2") {
		t.Errorf("config leaks secrets: %s", raw)
	}
	if len(out.EnvKeys) != 1 || out.EnvKeys[0] != "PASSWORD" {
		t.Errorf("EnvKeys = %v, want [PASSWORD]", out.EnvKeys)
	}
}

func TestMCPGetReportStatus_NoLocation(t *testing.T) {
	h, _, _ := newMCPRunsTestHandler(t)
	backend := &mcpBackend{h: h}

	out, err := backend.GetReportStatus(context.Background(), robotPrincipalFixture(), mcpserver.ScenarioRunRefInput{ScenarioRunName: "r1"})
	if err != nil {
		t.Fatalf("GetReportStatus failed: %v", err)
	}
	if !out.Generated {
		t.Error("Generated = false, want true")
	}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if containsStr(string(raw), "krkn-report-r1") {
		t.Errorf("report status leaks Location: %s", raw)
	}
}

func completedTRForRobot(name string, age time.Duration) *krknv1alpha1.KrknTargetRequest {
	now := time.Now().UTC()
	return &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(now.Add(-age)),
			Labels:            map[string]string{"krkn.krkn-chaos.dev/owner-user": "robot.ci-bot"},
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{UUID: name},
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

func TestMCPListClusters_ReusesRecentTargetRequest(t *testing.T) {
	tr := completedTRForRobot("tr-recent", 2*time.Minute)
	h, c, _ := newMCPRunsTestHandler(t, tr)
	backend := &mcpBackend{h: h}

	before := countTargetRequests(t, c)
	out, err := backend.ListClusters(context.Background(), robotPrincipalFixture(), mcpserver.ListClustersInput{})
	if err != nil {
		t.Fatalf("ListClusters failed: %v", err)
	}
	if out.TargetRequestID != "tr-recent" {
		t.Errorf("TargetRequestID = %q, want tr-recent (reused)", out.TargetRequestID)
	}
	if after := countTargetRequests(t, c); after != before {
		t.Errorf("target request count changed %d -> %d, want no new TR created", before, after)
	}
}

func countTargetRequests(t *testing.T, c client.Client) int {
	t.Helper()
	list := &krknv1alpha1.KrknTargetRequestList{}
	if err := c.List(context.Background(), list, client.InNamespace("default")); err != nil {
		t.Fatalf("list TRs: %v", err)
	}
	return len(list.Items)
}

func TestMCPListClusters_FiltersByActions(t *testing.T) {
	tr := completedTRForRobot("tr-recent", 2*time.Minute)
	h, _, _ := newMCPRunsTestHandler(t, tr)
	backend := &mcpBackend{h: h}

	out, err := backend.ListClusters(context.Background(), robotPrincipalFixture(), mcpserver.ListClustersInput{})
	if err != nil {
		t.Fatalf("ListClusters failed: %v", err)
	}
	if out.Status != "Completed" {
		t.Errorf("Status = %q, want Completed", out.Status)
	}
	if len(out.Clusters) != 1 {
		t.Fatalf("clusters = %+v, want only cluster-a", out.Clusters)
	}
	cluster := out.Clusters[0]
	if cluster.ClusterName != "cluster-a" {
		t.Errorf("cluster = %q, want cluster-a", cluster.ClusterName)
	}
	if len(cluster.Actions) != 3 || cluster.Actions[0] != "cancel" || cluster.Actions[1] != "run" || cluster.Actions[2] != "view" {
		t.Errorf("Actions = %v, want [cancel run view]", cluster.Actions)
	}
}

func TestMCPListClusters_PendingAfterTimeout(t *testing.T) {
	oldTimeout := mcpTargetPollTimeout
	mcpTargetPollTimeout = 50 * time.Millisecond
	defer func() { mcpTargetPollTimeout = oldTimeout }()

	h, _, _ := newMCPRunsTestHandler(t)
	backend := &mcpBackend{h: h}

	out, err := backend.ListClusters(context.Background(), robotPrincipalFixture(), mcpserver.ListClustersInput{})
	if err != nil {
		t.Fatalf("ListClusters failed: %v", err)
	}
	if out.Status != "Pending" {
		t.Errorf("Status = %q, want Pending after poll timeout", out.Status)
	}
	if len(out.Clusters) != 0 {
		t.Errorf("clusters = %+v, want none while pending", out.Clusters)
	}
}

// TestMCPListClusters_ReturnsClustersOnFirstCompletedCall covers the case the
// pre-completed fixtures never hit: discovery finishes while the very first
// krkn_list_clusters call is still polling, and that same call must return the
// discovered clusters instead of an empty Completed result.
func TestMCPListClusters_ReturnsClustersOnFirstCompletedCall(t *testing.T) {
	oldInterval := mcpTargetPollInterval
	oldTimeout := mcpTargetPollTimeout
	mcpTargetPollInterval = 5 * time.Millisecond
	mcpTargetPollTimeout = 5 * time.Second
	defer func() {
		mcpTargetPollInterval = oldInterval
		mcpTargetPollTimeout = oldTimeout
	}()

	tr := completedTRForRobot("tr-slow", 0)
	tr.Status.Status = "Pending"
	tr.Status.TargetData = nil
	h, c, _ := newMCPRunsTestHandler(t, tr)

	// Discovery completes shortly after polling starts.
	go func() {
		time.Sleep(20 * time.Millisecond)
		updated := &krknv1alpha1.KrknTargetRequest{}
		if err := c.Get(context.Background(), client.ObjectKey{Name: "tr-slow", Namespace: "default"}, updated); err != nil {
			return
		}
		updated.Status.Status = "Completed"
		updated.Status.TargetData = map[string][]krknv1alpha1.ClusterTarget{
			"krkn-operator": {
				{ClusterName: "cluster-a", ClusterAPIURL: urlA},
				{ClusterName: "cluster-b", ClusterAPIURL: urlB},
			},
		}
		_ = c.Update(context.Background(), updated)
	}()

	backend := &mcpBackend{h: h}
	out, err := backend.ListClusters(context.Background(), robotPrincipalFixture(), mcpserver.ListClustersInput{TargetRequestID: "tr-slow"})
	if err != nil {
		t.Fatalf("ListClusters failed: %v", err)
	}
	if out.Status != "Completed" {
		t.Fatalf("Status = %q, want Completed", out.Status)
	}
	if len(out.Clusters) != 1 || out.Clusters[0].ClusterName != "cluster-a" {
		t.Errorf("clusters = %+v, want [cluster-a] on the same first call that observed completion", out.Clusters)
	}
}

func TestMCPListAvailableResources_NoSecretFields(t *testing.T) {
	// Registry secret carrying credentials, file ConfigMap and cloud
	// credential secret carrying sensitive data: none of it may reach the
	// tool output.
	dockerCfg := `{"auths":{"https://reg.example.invalid":{"auth":"dXNlcjpwYXNzd29yZA=="}}}`
	registrySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "reg-one",
			Namespace: "default",
			Labels: map[string]string{
				registry.AppNameLabel:                           registry.AppName,
				registry.AppComponentLabel:                      registry.ComponentRegistry,
				"registry.krkn.krkn-chaos.dev/available-to-all": "true",
			},
			Annotations: map[string]string{
				"registry.krkn.krkn-chaos.dev/registry-url":        "https://reg.example.invalid",
				"registry.krkn.krkn-chaos.dev/scenario-repository": "scenarios",
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: []byte(dockerCfg),
		},
	}
	fileCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "file-uuid-1",
			Namespace: "default",
			Labels: map[string]string{
				files.AppNameLabel:        files.AppName,
				files.AppComponentLabel:   files.ComponentFile,
				files.FileIDLabel:         "uuid-1",
				files.AvailableToAllLabel: "true",
			},
		},
		Data: map[string]string{"kubeconfig": "super-secret-kubeconfig-content"},
	}
	cloudSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cred-one",
			Namespace: "default",
			Labels: map[string]string{
				"app.kubernetes.io/name":                          "krkn-operator",
				"app.kubernetes.io/component":                     "cloud-credential",
				"cloudcreds.krkn.krkn-chaos.dev/available-to-all": "true",
			},
		},
		Data: map[string][]byte{"password": []byte("hunter2")},
	}

	h, _, _ := newMCPRunsTestHandler(t, registrySecret, fileCM, cloudSecret)
	backend := &mcpBackend{h: h}

	out, err := backend.ListAvailableResources(context.Background(), robotPrincipalFixture(), struct{}{})
	if err != nil {
		t.Fatalf("ListAvailableResources failed: %v", err)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	lower := strings.ToLower(string(raw))
	for _, keyword := range []string{"password", "token", "secret", "kubeconfig"} {
		if strings.Contains(lower, keyword) {
			t.Errorf("available resources output contains %q: %s", keyword, raw)
		}
	}
}
