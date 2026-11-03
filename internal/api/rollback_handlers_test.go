package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/workspace"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const rollbackSourceUID = "7fa25dbb-7278-4104-90c7-e7744b68e5af"
const rollbackAttemptID = "4f322276-81e4-4d70-9f94-9951ded10e75"
const priorRollbackAttemptID = "5e34f888-ed3a-4b57-82b8-8c59ab5dc4ab"

func rollbackTestHandler(t *testing.T) *Handler {
	t.Helper()
	h := setupScenarioRunTestHandler("target", map[string]string{})
	require.NoError(t, h.client.Create(t.Context(), testWorkspacePVC()))
	readyWorkspace(t, h, "", true)
	public := false
	run := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "source-run", Namespace: "default", UID: types.UID(rollbackSourceUID)},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			Scenario:        krknv1alpha1.ScenarioReference{Name: "pod-delete", Private: &public},
			TargetRequestID: "target", TargetClusters: map[string][]string{"provider": {"cluster"}},
			RollbackEnabled: true,
			Workspace:       &krknv1alpha1.WorkspaceReference{Name: "friendly", WorkspaceUID: testEntryUID, ClaimName: "shared", ClaimUID: testWorkspaceUID},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{ClusterJobs: []krknv1alpha1.ClusterJobStatus{{
			ProviderName: "provider", ClusterName: "cluster", ClusterAPIURL: "https://cluster.example.com",
			JobID: rollbackAttemptID, ScenarioImage: "quay.io/krkn-chaos/krkn-hub:pod-delete", Phase: "Failed",
		}}},
	}
	require.NoError(t, h.client.Create(t.Context(), run))
	require.NoError(t, h.client.Create(t.Context(), &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "krkn-job-" + rollbackAttemptID + "-kubeconfig", Namespace: "default",
			Labels:          map[string]string{"krkn-job-id": rollbackAttemptID, "krkn-cluster-name": "cluster"},
			OwnerReferences: []metav1.OwnerReference{{APIVersion: krknv1alpha1.GroupVersion.String(), Kind: "KrknScenarioRun", Name: run.Name, UID: run.UID}},
		},
		Data: map[string]string{"config": "apiVersion: v1"},
	}))
	return h
}

func rollbackRequest(h *Handler, method, role, jobID string) *httptest.ResponseRecorder {
	path := ScenariosRunPath + "/source-run/rollback/" + jobID
	response := httptest.NewRecorder()
	h.ScenariosRunRouter(response, newCategoryRequestAs(method, path, "", role, "person@example.com"))
	return response
}

func decodeRollbackResponse(t *testing.T, response *httptest.ResponseRecorder) RollbackAvailability {
	t.Helper()
	var result RollbackAvailability
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
	return result
}

func TestScenarioRollbackRequiresRetainedFilesAndReusesOriginalAttempt(t *testing.T) {
	h := rollbackTestHandler(t)
	first := rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())
	require.Equal(t, "checking", decodeRollbackResponse(t, first).State)

	probe := &corev1.Pod{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: rollbackProbeName(rollbackAttemptID)}, probe))
	require.True(t, probe.Spec.Volumes[0].PersistentVolumeClaim.ReadOnly)
	require.Equal(t, workspace.MountPath, probe.Spec.Containers[0].VolumeMounts[0].MountPath)
	expectedSubPath, err := workspace.SubPath(types.UID(rollbackSourceUID), "provider", "cluster", rollbackAttemptID)
	require.NoError(t, err)
	require.Equal(t, expectedSubPath, probe.Spec.Containers[0].VolumeMounts[0].SubPath)

	probe.Status.Phase = corev1.PodSucceeded
	probe.Status.ContainerStatuses = []corev1.ContainerStatus{{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Message: "absent"}}}}
	require.NoError(t, h.client.Status().Update(t.Context(), probe))
	absent := rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)
	require.Equal(t, "absent", decodeRollbackResponse(t, absent).State)
	require.Equal(t, http.StatusConflict, rollbackRequest(h, http.MethodPost, "admin", rollbackAttemptID).Code)

	require.NoError(t, h.client.Get(t.Context(), client.ObjectKeyFromObject(probe), probe))
	probe.Status.ContainerStatuses[0].State.Terminated.Message = "available"
	require.NoError(t, h.client.Status().Update(t.Context(), probe))
	available := rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)
	require.Equal(t, "available", decodeRollbackResponse(t, available).State)
	started := rollbackRequest(h, http.MethodPost, "admin", rollbackAttemptID)
	require.Equal(t, http.StatusAccepted, started.Code, started.Body.String())
	require.Equal(t, "running", decodeRollbackResponse(t, started).State)
	rollback := &krknv1alpha1.KrknScenarioRun{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: rollbackRunName(rollbackAttemptID)}, rollback))
	require.Equal(t, "rollback", rollback.Spec.Scenario.Name)
	require.Equal(t, rollbackAttemptID, rollback.Spec.Environment["UUID"])
	require.Equal(t, workspace.MountPath, rollback.Spec.Environment["ROLLBACK_VERSIONS_DIRECTORY"])
	require.Equal(t, rollbackSourceUID, rollback.Spec.RollbackSource.RunUID)
	require.Equal(t, rollbackAttemptID, rollback.Spec.RollbackSource.JobID)
	require.Equal(t, "https://cluster.example.com", rollback.Spec.RollbackSource.ClusterAPIURL)
	require.Equal(t, http.StatusOK, rollbackRequest(h, http.MethodPost, "admin", rollbackAttemptID).Code)

	rollback.Status.Phase = "Failed"
	require.NoError(t, h.client.Update(t.Context(), rollback))
	probe = &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: rollbackProbeName(rollbackAttemptID), Namespace: "default", Labels: map[string]string{"krkn-scenario-run-uid": rollbackSourceUID}}}
	require.NoError(t, h.client.Create(t.Context(), probe))
	probe.Status.Phase = corev1.PodSucceeded
	probe.Status.ContainerStatuses = []corev1.ContainerStatus{{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Message: "available"}}}}
	require.NoError(t, h.client.Status().Update(t.Context(), probe))
	failed := rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)
	require.Equal(t, "failed", decodeRollbackResponse(t, failed).State)
	retry := rollbackRequest(h, http.MethodPost, "admin", rollbackAttemptID)
	require.Equal(t, http.StatusAccepted, retry.Code, retry.Body.String())
	require.Equal(t, rollbackRunAttemptName(rollbackAttemptID, 2), decodeRollbackResponse(t, retry).RollbackRunName)
}

func TestScenarioRollbackAuthorizationAndMissingPVC(t *testing.T) {
	h := rollbackTestHandler(t)
	require.NoError(t, h.client.Create(t.Context(), newCategoryTestUser(t, "person@example.com")))
	require.Equal(t, http.StatusForbidden, rollbackRequest(h, http.MethodGet, "user", rollbackAttemptID).Code)
	require.Equal(t, http.StatusForbidden, rollbackRequest(h, http.MethodPost, "user", rollbackAttemptID).Code)
	pvc := &corev1.PersistentVolumeClaim{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: "shared"}, pvc))
	require.NoError(t, h.client.Delete(t.Context(), pvc))
	response := rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "unavailable", decodeRollbackResponse(t, response).State)
	require.Equal(t, http.StatusConflict, rollbackRequest(h, http.MethodPost, "admin", rollbackAttemptID).Code)
	replacement := testWorkspacePVC()
	replacement.UID = "8f53d5ab-fca8-464f-877c-9bb72dcdb319"
	require.NoError(t, h.client.Create(t.Context(), replacement))
	require.Equal(t, "unavailable", decodeRollbackResponse(t, rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)).State)
}

func TestScenarioRollbackViewerCannotExecute(t *testing.T) {
	h := rollbackTestHandler(t)
	group := newCategoryTestGroup("viewers")
	group.Spec.ClusterPermissions = map[string]krknv1alpha1.ClusterPermissionSet{
		"https://cluster.example.com": {Actions: []string{"view"}},
	}
	require.NoError(t, h.client.Create(t.Context(), group))
	require.NoError(t, h.client.Create(t.Context(), newCategoryTestUser(t, "person@example.com", "viewers")))
	response := rollbackRequest(h, http.MethodGet, "user", rollbackAttemptID)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.False(t, decodeRollbackResponse(t, response).CanExecute)
	require.Equal(t, http.StatusForbidden, rollbackRequest(h, http.MethodPost, "user", rollbackAttemptID).Code)
}

func TestScenarioRollbackInspectsEarlierRetryAttempt(t *testing.T) {
	h := rollbackTestHandler(t)
	run := &krknv1alpha1.KrknScenarioRun{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: "source-run"}, run))
	run.Status.ClusterJobs[0].AttemptHistory = []krknv1alpha1.ScenarioAttemptStatus{{JobID: priorRollbackAttemptID, ScenarioImage: "quay.io/krkn-chaos/krkn-hub:pod-delete"}}
	require.NoError(t, h.client.Update(t.Context(), run))
	require.NoError(t, h.client.Create(t.Context(), &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "krkn-job-" + priorRollbackAttemptID + "-kubeconfig", Namespace: "default",
			Labels:          map[string]string{"krkn-job-id": priorRollbackAttemptID, "krkn-cluster-name": "cluster"},
			OwnerReferences: []metav1.OwnerReference{{APIVersion: krknv1alpha1.GroupVersion.String(), Kind: "KrknScenarioRun", Name: run.Name, UID: run.UID}}},
		Data: map[string]string{"config": "apiVersion: v1"},
	}))
	response := rollbackRequest(h, http.MethodGet, "admin", priorRollbackAttemptID)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Equal(t, "checking", decodeRollbackResponse(t, response).State)
	probe := &corev1.Pod{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: rollbackProbeName(priorRollbackAttemptID)}, probe))
	expected, err := workspace.SubPath(run.UID, "provider", "cluster", priorRollbackAttemptID)
	require.NoError(t, err)
	require.Equal(t, expected, probe.Spec.Containers[0].VolumeMounts[0].SubPath)
}

func TestScenarioRollbackCannotUseMissingOriginalKubeconfig(t *testing.T) {
	h := rollbackTestHandler(t)
	configMap := &corev1.ConfigMap{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: "krkn-job-" + rollbackAttemptID + "-kubeconfig"}, configMap))
	require.NoError(t, h.client.Delete(t.Context(), configMap))
	first := rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)
	require.Equal(t, http.StatusOK, first.Code)
	require.False(t, decodeRollbackResponse(t, first).CanExecute)
	probe := &corev1.Pod{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: rollbackProbeName(rollbackAttemptID)}, probe))
	probe.Status.Phase = corev1.PodSucceeded
	probe.Status.ContainerStatuses = []corev1.ContainerStatus{{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Message: "available"}}}}
	require.NoError(t, h.client.Status().Update(t.Context(), probe))
	available := rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)
	require.Equal(t, "available", decodeRollbackResponse(t, available).State)
	require.False(t, decodeRollbackResponse(t, available).CanExecute)
	require.Equal(t, http.StatusConflict, rollbackRequest(h, http.MethodPost, "admin", rollbackAttemptID).Code)
}
