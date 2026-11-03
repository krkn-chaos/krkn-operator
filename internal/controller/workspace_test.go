package controller

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/workspace"
	krknctlconfig "github.com/krkn-chaos/krknctl/pkg/config"
	krknctlmodels "github.com/krkn-chaos/krknctl/pkg/provider/models"
	"github.com/krkn-chaos/krknctl/pkg/verify"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPrepareJobResourcesRejectsUnavailableWorkspaceBeforeCreatingPod(t *testing.T) {
	for _, tc := range []struct {
		name string
		pvc  *corev1.PersistentVolumeClaim
		want error
	}{
		{name: "missing", want: workspace.ErrUnavailable},
		{name: "replaced", want: workspace.ErrReplaced, pvc: &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: "shared", Namespace: "default", UID: "dc7f90ee-56d0-4521-b014-d588f2af9341",
			},
			Spec:   corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}},
			Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, corev1.AddToScheme(scheme))
			require.NoError(t, krknv1alpha1.AddToScheme(scheme))
			objects := []client.Object{&krknv1alpha1.KrknWorkspace{
				ObjectMeta: metav1.ObjectMeta{Name: "friendly", Namespace: "default", UID: "e7b95e74-ffec-4574-8c7a-e102444decae"},
				Spec:       krknv1alpha1.KrknWorkspaceSpec{ClaimName: "shared", ClaimUID: "9e70715d-9154-4b9e-aa76-1ae6c6b2594e"},
			}}
			if tc.pvc != nil {
				objects = append(objects, tc.pvc)
			}
			r := &KrknScenarioRunReconciler{
				Client: fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build(),
				Scheme: scheme, Namespace: "default",
			}
			run := &krknv1alpha1.KrknScenarioRun{
				ObjectMeta: metav1.ObjectMeta{UID: "9e70715d-9154-4b9e-aa76-1ae6c6b2594e"},
				Spec: krknv1alpha1.KrknScenarioRunSpec{Workspace: &krknv1alpha1.WorkspaceReference{
					Name: "friendly", WorkspaceUID: "e7b95e74-ffec-4574-8c7a-e102444decae",
					ClaimName: "shared", ClaimUID: "9e70715d-9154-4b9e-aa76-1ae6c6b2594e",
				}},
			}
			_, err := r.prepareJobResources(t.Context(), run, "provider", "cluster", "")
			require.True(t, errors.Is(err, tc.want), "error: %v", err)
			var pods corev1.PodList
			require.NoError(t, r.List(t.Context(), &pods))
			require.Empty(t, pods.Items)
		})
	}
}

func TestScenarioPodMountsOnlyAssignedWorkspaceDirectory(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "shared", Namespace: "default", UID: "dc7f90ee-56d0-4521-b014-d588f2af9341",
		},
		Spec:   corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}},
		Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: "default"},
		Data:       map[string][]byte{"managed-clusters": []byte(`{"provider":{"cluster":{"kubeconfig":"Y29uZmln"}}}`)},
	}
	private := false
	run := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "run", Namespace: "default", UID: "9e70715d-9154-4b9e-aa76-1ae6c6b2594e",
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "target",
			Scenario:        krknv1alpha1.ScenarioReference{Name: "pod-delete", Private: &private},
			Workspace:       &krknv1alpha1.WorkspaceReference{Name: "friendly", WorkspaceUID: "e7b95e74-ffec-4574-8c7a-e102444decae", ClaimName: "shared", ClaimUID: string(pvc.UID)},
			RollbackEnabled: true,
			Environment:     map[string]string{"ROLLBACK_VERSIONS_DIRECTORY": "/wrong/path", "UUID": "wrong-uuid"},
		},
	}
	entry := &krknv1alpha1.KrknWorkspace{ObjectMeta: metav1.ObjectMeta{Name: "friendly", Namespace: "default", UID: "e7b95e74-ffec-4574-8c7a-e102444decae"},
		Spec: krknv1alpha1.KrknWorkspaceSpec{ClaimName: "shared", ClaimUID: string(pvc.UID)}}
	r := &KrknScenarioRunReconciler{
		Client: fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(pvc, secret, run, entry).Build(),
		Scheme: scheme, Namespace: "default",
		imageResolver: func(context.Context, *krknctlconfig.Config, krknv1alpha1.ScenarioReference, *krknctlmodels.RegistryV2) (string, error) {
			return "quay.io/example/scenario@sha256:abc", nil
		},
		signatureVerifier: func(context.Context, *krknctlconfig.Config, krknv1alpha1.ScenarioReference, *krknctlmodels.RegistryV2, string) (verify.SignatureStatus, error) {
			return verify.SignatureSigned, nil
		},
	}
	resources, err := r.prepareJobResources(t.Context(), run, "provider", "cluster", "https://cluster.example.com")
	require.NoError(t, err)
	expected, err := workspace.SubPath(run.UID, "provider", "cluster", resources.jobID)
	require.NoError(t, err)
	_, err = r.submitScenarioPod(t.Context(), run, resources)
	require.NoError(t, err)
	var pods corev1.PodList
	require.NoError(t, r.List(t.Context(), &pods))
	require.Len(t, pods.Items, 1)
	pod := pods.Items[0]
	require.Equal(t, "shared", pod.Spec.Volumes[len(pod.Spec.Volumes)-2].PersistentVolumeClaim.ClaimName)
	var workspaceMounts []corev1.VolumeMount
	for _, mount := range pod.Spec.Containers[0].VolumeMounts {
		if mount.Name == "workspace" {
			workspaceMounts = append(workspaceMounts, mount)
		}
	}
	require.Len(t, workspaceMounts, 1)
	require.Equal(t, workspace.MountPath, workspaceMounts[0].MountPath)
	require.Equal(t, expected, workspaceMounts[0].SubPath)
	var rollbackPaths []string
	for _, env := range pod.Spec.Containers[0].Env {
		if env.Name == "ROLLBACK_VERSIONS_DIRECTORY" {
			rollbackPaths = append(rollbackPaths, env.Value)
		}
	}
	require.Equal(t, []string{workspace.MountPath}, rollbackPaths)
	var runUUIDs []string
	for _, env := range pod.Spec.Containers[0].Env {
		if env.Name == "UUID" {
			runUUIDs = append(runUUIDs, env.Value)
		}
	}
	require.Equal(t, []string{resources.jobID}, runUUIDs)
	require.Contains(t, pod.Spec.Containers[0].Args[0], rollbackWorkspaceWriteCheck)
	require.Contains(t, pod.Spec.Containers[0].Args[0], rollbackConfigTemplateCheck("/home/krkn/kraken/config/config.yaml.template"))
	require.Empty(t, pod.Spec.InitContainers)
	require.EqualValues(t, 1001, *pod.Spec.SecurityContext.RunAsUser)
	require.EqualValues(t, 1001, *pod.Spec.SecurityContext.RunAsGroup)
	require.EqualValues(t, 1001, *pod.Spec.SecurityContext.FSGroup)

	run.Spec.RollbackEnabled = false
	run.Spec.Environment = nil
	workspaceOnly, err := r.prepareJobResources(t.Context(), run, "provider", "cluster", "https://cluster.example.com")
	require.NoError(t, err)
	for _, env := range workspaceOnly.envVars {
		require.NotEqual(t, "ROLLBACK_VERSIONS_DIRECTORY", env.Name)
	}
	mounted := false
	for _, mount := range workspaceOnly.volumeMounts {
		if mount.Name == "workspace" && mount.MountPath == workspace.MountPath {
			mounted = true
		}
	}
	require.True(t, mounted, "workspace remains mounted without rollback")

	original := &krknv1alpha1.KrknScenarioRun{}
	require.NoError(t, r.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: "run"}, original))
	original.Status.ClusterJobs = []krknv1alpha1.ClusterJobStatus{{ProviderName: "provider", ClusterName: "cluster", ClusterAPIURL: "https://cluster.example.com", JobID: resources.jobID, Phase: "Failed"}}
	require.NoError(t, r.Update(t.Context(), original))
	rollback := run.DeepCopy()
	rollback.Name = "rollback-run"
	rollback.UID = "8a44fb1b-1c03-4f17-af55-674e096b356e"
	rollback.Spec.Scenario.Name = "rollback"
	rollback.Spec.RollbackSource = &krknv1alpha1.RollbackSource{RunName: "run", RunUID: string(original.UID), JobID: resources.jobID, ClusterAPIURL: "https://cluster.example.com"}
	changedTarget := &corev1.Secret{}
	require.NoError(t, r.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: "target"}, changedTarget))
	changedTarget.Data["managed-clusters"] = []byte(`{"provider":{"cluster":{"kubeconfig":"bmV3"}}}`)
	require.NoError(t, r.Update(t.Context(), changedTarget))
	rollbackResources, err := r.prepareJobResources(t.Context(), rollback, "provider", "cluster", "https://cluster.example.com")
	require.NoError(t, err)
	rollbackConfig := &corev1.ConfigMap{}
	require.NoError(t, r.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: "krkn-job-" + rollbackResources.jobID + "-kubeconfig"}, rollbackConfig))
	require.Equal(t, "config", rollbackConfig.Data["config"], "rollback uses the original attempt kubeconfig")
	for _, mount := range rollbackResources.volumeMounts {
		if mount.Name == "workspace" {
			require.Equal(t, expected, mount.SubPath, "rollback mounts the original attempt directory")
		}
	}
	rollback.Spec.RollbackSource.RunUID = "replaced-run-uid"
	_, err = r.prepareJobResources(t.Context(), rollback, "provider", "cluster", "https://cluster.example.com")
	require.ErrorContains(t, err, "workspace identity changed")
}

func TestPrepareJobResourcesRequiresWorkspaceForRollback(t *testing.T) {
	r := &KrknScenarioRunReconciler{}
	run := &krknv1alpha1.KrknScenarioRun{Spec: krknv1alpha1.KrknScenarioRunSpec{RollbackEnabled: true}}
	_, err := r.prepareJobResources(t.Context(), run, "provider", "cluster", "")
	require.ErrorContains(t, err, "rollback requires a workspace")
}

func TestRollbackWorkspaceWriteCheck(t *testing.T) {
	for _, tc := range []struct {
		name     string
		dir      string
		wantFail bool
	}{
		{name: "writable", dir: t.TempDir()},
		{name: "missing directory", dir: filepath.Join(t.TempDir(), "missing"), wantFail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.CommandContext(t.Context(), "/bin/bash", "-c", rollbackWorkspaceWriteCheck+"echo SCENARIO_STARTED")
			cmd.Env = append(os.Environ(), "ROLLBACK_VERSIONS_DIRECTORY="+tc.dir)
			output, err := cmd.CombinedOutput()
			if tc.wantFail {
				require.Error(t, err)
				require.Contains(t, string(output), "rollback workspace is not writable")
				require.NotContains(t, string(output), "SCENARIO_STARTED")
				return
			}
			require.NoError(t, err, string(output))
			require.Contains(t, string(output), "SCENARIO_STARTED")
			_, err = os.Stat(filepath.Join(tc.dir, ".krkn-workspace-write-probe"))
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	}
}

func TestRollbackConfigTemplateCheck(t *testing.T) {
	for _, tc := range []struct {
		name     string
		template string
		wantFail bool
	}{
		{name: "plain variable", template: "kraken:\n  rollback_versions_directory: $ROLLBACK_VERSIONS_DIRECTORY\n"},
		{name: "braced variable", template: "kraken:\n  rollback_versions_directory: ${ROLLBACK_VERSIONS_DIRECTORY}\n"},
		{name: "missing mapping", template: "kraken:\n  chaos_scenarios: []\n", wantFail: true},
		{name: "hardcoded path", template: "kraken:\n  rollback_versions_directory: /tmp/rollback\n", wantFail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			templatePath := filepath.Join(t.TempDir(), "config.yaml.template")
			require.NoError(t, os.WriteFile(templatePath, []byte(tc.template), 0o600))
			cmd := exec.CommandContext(t.Context(), "/bin/bash", "-c", rollbackConfigTemplateCheck(templatePath)+"echo SCENARIO_STARTED")
			output, err := cmd.CombinedOutput()
			if tc.wantFail {
				require.Error(t, err)
				require.Contains(t, string(output), "scenario image does not map ROLLBACK_VERSIONS_DIRECTORY")
				require.NotContains(t, string(output), "SCENARIO_STARTED")
				return
			}
			require.NoError(t, err, string(output))
			require.Contains(t, string(output), "SCENARIO_STARTED")
		})
	}
}

func TestPendingScenarioPodFailsWhenWorkspaceDisappears(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "scenario-pod", Namespace: "default"},
		Status:     corev1.PodStatus{Phase: corev1.PodPending},
	}
	r := &KrknScenarioRunReconciler{
		Client: fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build(),
		Scheme: scheme, Namespace: "default",
	}
	now := metav1.Now()
	run := &krknv1alpha1.KrknScenarioRun{
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			MaxRetries: 3,
			Workspace:  &krknv1alpha1.WorkspaceReference{Name: "friendly", WorkspaceUID: "e7b95e74-ffec-4574-8c7a-e102444decae", ClaimName: "shared", ClaimUID: "9e70715d-9154-4b9e-aa76-1ae6c6b2594e"},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{ClusterJobs: []krknv1alpha1.ClusterJobStatus{{
			JobID: "af3caf28-bb79-40a3-a00c-05ecbe209d10", PodName: "scenario-pod",
			Phase: "Pending", StartTime: &now, MaxRetries: 3, MaxRetriesConfigured: true,
		}}},
	}
	require.NoError(t, r.updateClusterJobStatuses(t.Context(), run))
	job := run.Status.ClusterJobs[0]
	require.Equal(t, "Failed", job.Phase)
	require.Equal(t, "WorkspaceUnavailable", job.FailureReason)
	require.Equal(t, job.MaxRetries, job.RetryCount)
	require.NotNil(t, job.CompletionTime)
	err := r.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: "scenario-pod"}, &corev1.Pod{})
	require.True(t, apierrors.IsNotFound(err), "pod should be removed: %v", err)
}
