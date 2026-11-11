package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type failingWorkspaceGetClient struct{ client.Client }

func (c failingWorkspaceGetClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if _, ok := obj.(*krknv1alpha1.KrknWorkspace); ok {
		return errors.New("temporary workspace lookup failure")
	}
	return c.Client.Get(ctx, key, obj, opts...)
}

const testWorkspaceUID = "9e70715d-9154-4b9e-aa76-1ae6c6b2594e"
const testEntryUID = "e7b95e74-ffec-4574-8c7a-e102444decae"

func testWorkspacePVC() *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: "default", UID: testWorkspaceUID},
		Spec:       corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}},
		Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
	}
}

func TestListAvailableWorkspacePVCs(t *testing.T) {
	h := setupScenarioRunTestHandler("target", map[string]string{})
	ready := testWorkspacePVC()
	ready.Name = "ready-rwx"
	pending := testWorkspacePVC()
	pending.Name = "pending-rwx"
	pending.Status.Phase = corev1.ClaimPending
	readWriteOnce := testWorkspacePVC()
	readWriteOnce.Name = "bound-rwo"
	readWriteOnce.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	block := testWorkspacePVC()
	block.Name = "bound-block"
	mode := corev1.PersistentVolumeBlock
	block.Spec.VolumeMode = &mode
	otherNamespace := testWorkspacePVC()
	otherNamespace.Name = "other-namespace"
	otherNamespace.Namespace = "elsewhere"
	for _, pvc := range []*corev1.PersistentVolumeClaim{ready, pending, readWriteOnce, block, otherNamespace} {
		require.NoError(t, h.client.Create(t.Context(), pvc))
	}

	for _, tc := range []struct {
		method string
		role   string
		code   int
	}{
		{http.MethodGet, "admin", http.StatusOK},
		{http.MethodGet, "user", http.StatusForbidden},
		{http.MethodPost, "admin", http.StatusMethodNotAllowed},
	} {
		response := httptest.NewRecorder()
		h.WorkspaceRouter(response, newCategoryRequestAs(tc.method, WorkspacePVCsPath, "", tc.role, "person@example.com"))
		require.Equal(t, tc.code, response.Code, response.Body.String())
		if tc.code == http.StatusOK {
			var body struct {
				PVCs []availableWorkspacePVC `json:"pvcs"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
			require.Equal(t, []availableWorkspacePVC{{Name: "ready-rwx"}}, body.PVCs)
		}
	}
}

func TestScenarioRunWithoutPVCOrWorkspace(t *testing.T) {
	h := setupScenarioRunTestHandler("target", map[string]string{"cluster": "YXBpVmVyc2lvbjogdjEK"})

	for _, path := range []string{WorkspacePVCsPath, WorkspacesPath + "/available"} {
		response := httptest.NewRecorder()
		h.WorkspaceRouter(response, newCategoryRequestAs(http.MethodGet, path, "", "admin", "admin@example.com"))
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		if path == WorkspacePVCsPath {
			require.JSONEq(t, `{"pvcs":[]}`, response.Body.String())
		} else {
			require.JSONEq(t, `{"workspaces":[]}`, response.Body.String())
		}
	}

	request := `{"targetRequestId":"target","targetClusters":{"krkn-operator":["cluster"]},"scenario":{"name":"pod-delete","private":false},"rollbackEnabled":false}`
	response := httptest.NewRecorder()
	h.PostScenarioRun(response, newCategoryRequestAs(http.MethodPost, ScenariosRunPath, request, "admin", "admin@example.com"))
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var created ScenarioRunCreateResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &created))
	run := &krknv1alpha1.KrknScenarioRun{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: created.ScenarioRunName}, run))
	require.Nil(t, run.Spec.Workspace)
	require.False(t, run.Spec.RollbackEnabled)
}

func readyWorkspace(t *testing.T, h *Handler, group string, public bool) *krknv1alpha1.KrknWorkspace {
	t.Helper()
	entry := &krknv1alpha1.KrknWorkspace{ObjectMeta: metav1.ObjectMeta{Name: "friendly", Namespace: "default", UID: testEntryUID},
		Spec: krknv1alpha1.KrknWorkspaceSpec{ClaimName: "shared", ClaimUID: testWorkspaceUID, Group: group, AvailableToAll: public}}
	require.NoError(t, h.client.Create(t.Context(), entry))
	entry.Status.ObservedGeneration = entry.Generation
	entry.Status.Conditions = []metav1.Condition{{Type: "Available", Status: metav1.ConditionTrue,
		Reason: "PVCReady", Message: "ready", ObservedGeneration: entry.Generation, LastTransitionTime: metav1.Now()}}
	require.NoError(t, h.client.Update(t.Context(), entry))
	return entry
}

func TestWorkspaceReferenceVisibilityPreservesPinnedIdentity(t *testing.T) {
	h := setupScenarioRunTestHandler("target", map[string]string{})
	require.NoError(t, h.client.Create(t.Context(), testWorkspacePVC()))
	readyWorkspace(t, h, "team-a", false)
	require.NoError(t, h.client.Create(t.Context(), newCategoryTestGroup("team-a")))
	require.NoError(t, h.client.Create(t.Context(), newCategoryTestUser(t, "member@example.com", "team-a")))
	require.NoError(t, h.client.Create(t.Context(), newCategoryTestUser(t, "outsider@example.com")))
	ref := &krknv1alpha1.WorkspaceReference{Name: "friendly", WorkspaceUID: testEntryUID, ClaimName: "shared", ClaimUID: testWorkspaceUID}
	member := newCategoryRequestAs(http.MethodGet, WorkspacesPath, "", "user", "member@example.com")
	visible, err := h.visibleWorkspaceReference(member.Context(), ref)
	require.NoError(t, err)
	require.NotNil(t, visible)
	require.Equal(t, ref.Name, visible.Name)
	require.Equal(t, ref.WorkspaceUID, visible.WorkspaceUID)
	require.Equal(t, ref.ClaimUID, visible.ClaimUID)
	require.Empty(t, visible.ClaimName)
	require.Equal(t, "shared", ref.ClaimName, "response redaction must not mutate the run")
	outsider := newCategoryRequestAs(http.MethodGet, WorkspacesPath, "", "user", "outsider@example.com")
	visible, err = h.visibleWorkspaceReference(outsider.Context(), ref)
	require.NoError(t, err)
	require.Nil(t, visible)
	nodes := map[string]krknv1alpha1.GraphScenarioNode{"node": {Workspace: ref}}
	redacted, err := h.publicGraphWorkspaces(outsider.Context(), nodes)
	require.NoError(t, err)
	require.Nil(t, redacted["node"].Workspace)
	h.client = failingWorkspaceGetClient{h.client}
	_, err = h.visibleWorkspaceReference(member.Context(), ref)
	require.ErrorContains(t, err, "temporary workspace lookup failure")
	_, err = h.publicGraphWorkspaces(member.Context(), nodes)
	require.ErrorContains(t, err, "temporary workspace lookup failure")
}

func TestAvailableWorkspacesTrailingSlashStillFiltersUnavailable(t *testing.T) {
	h := setupScenarioRunTestHandler("target", map[string]string{})
	require.NoError(t, h.client.Create(t.Context(), testWorkspacePVC()))
	readyWorkspace(t, h, "", true)
	require.NoError(t, h.client.Create(t.Context(), &krknv1alpha1.KrknWorkspace{
		ObjectMeta: metav1.ObjectMeta{Name: "unavailable", Namespace: "default"},
		Spec:       krknv1alpha1.KrknWorkspaceSpec{ClaimName: "shared", ClaimUID: testWorkspaceUID, AvailableToAll: true},
	}))
	for _, path := range []string{WorkspacesPath + "/available", WorkspacesPath + "/available/"} {
		response := httptest.NewRecorder()
		h.WorkspaceRouter(response, newCategoryRequestAs(http.MethodGet, path, "", "admin", "admin@example.com"))
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var body struct {
			Workspaces []workspaceInfo `json:"workspaces"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.Len(t, body.Workspaces, 1)
		require.Equal(t, "friendly", body.Workspaces[0].Name)
	}
}

func TestWorkspaceResolutionHidesPVCIdentityFromNonAdmin(t *testing.T) {
	h := setupScenarioRunTestHandler("target", map[string]string{})
	require.NoError(t, h.client.Create(t.Context(), newCategoryTestUser(t, "person@example.com")))
	pvc := testWorkspacePVC()
	require.NoError(t, h.client.Create(t.Context(), pvc))
	readyWorkspace(t, h, "", true)
	pvc.Status.Phase = corev1.ClaimPending
	require.NoError(t, h.client.Status().Update(t.Context(), pvc))
	response := httptest.NewRecorder()
	request := newCategoryRequestAs(http.MethodPost, ScenariosRunPath, "", "user", "person@example.com")
	_, ok := h.resolveWorkspaceSelection(response, request, &krknv1alpha1.WorkspaceReference{Name: "friendly"})
	require.False(t, ok)
	require.Equal(t, http.StatusConflict, response.Code)
	require.NotContains(t, response.Body.String(), "shared")
	require.NotContains(t, response.Body.String(), testWorkspaceUID)
}

func TestWorkspacePublicationAndAccess(t *testing.T) {
	h := setupScenarioRunTestHandler("target", map[string]string{})
	require.NoError(t, h.client.Create(t.Context(), testWorkspacePVC()))
	require.NoError(t, h.client.Create(t.Context(), newCategoryTestGroup("team-a")))
	require.NoError(t, h.client.Create(t.Context(), newCategoryTestUser(t, "member@example.com", "team-a")))
	require.NoError(t, h.client.Create(t.Context(), newCategoryTestUser(t, "outsider@example.com", "team-b")))

	unauthorized := httptest.NewRecorder()
	h.WorkspaceRouter(unauthorized, newCategoryRequestAs(http.MethodPut, WorkspacesPath+"/friendly", `{"claimName":"shared","groups":["team-a"]}`, "user", "member@example.com"))
	require.Equal(t, http.StatusForbidden, unauthorized.Code)

	published := httptest.NewRecorder()
	h.WorkspaceRouter(published, newCategoryRequestAs(http.MethodPut, WorkspacesPath+"/friendly", `{"claimName":"shared","groups":["team-a"]}`, "admin", "admin@example.com"))
	require.Equal(t, http.StatusOK, published.Code, published.Body.String())
	require.NotContains(t, published.Body.String(), "displayName")
	legacyName := httptest.NewRecorder()
	h.WorkspaceRouter(legacyName, newCategoryRequestAs(http.MethodPut, WorkspacesPath+"/another", `{"claimName":"shared","displayName":"Duplicate"}`, "admin", "admin@example.com"))
	require.Equal(t, http.StatusBadRequest, legacyName.Code, legacyName.Body.String())
	entry := &krknv1alpha1.KrknWorkspace{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: "friendly"}, entry))
	require.Equal(t, "shared", entry.Spec.ClaimName)
	require.Equal(t, testWorkspaceUID, entry.Spec.ClaimUID)
	require.Equal(t, "team-a", entry.Spec.Group)
	entry.Status.ObservedGeneration = entry.Generation
	entry.Status.Conditions = []metav1.Condition{{Type: "Available", Status: metav1.ConditionTrue, Reason: "PVCReady",
		Message: "ready", ObservedGeneration: entry.Generation, LastTransitionTime: metav1.Now()}}
	require.NoError(t, h.client.Update(t.Context(), entry))

	for _, tc := range []struct {
		user  string
		count int
	}{
		{"member@example.com", 1}, {"outsider@example.com", 0},
	} {
		response := httptest.NewRecorder()
		h.WorkspaceRouter(response, newCategoryRequestAs(http.MethodGet, WorkspacesPath+"/available", "", "user", tc.user))
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var body struct {
			Workspaces []workspaceInfo `json:"workspaces"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.Len(t, body.Workspaces, tc.count)
		if tc.count != 0 {
			require.Equal(t, "friendly", body.Workspaces[0].Name)
			require.Empty(t, body.Workspaces[0].ClaimName, "users must not see the cluster PVC name")
		}
	}
	denied := httptest.NewRecorder()
	_, ok := h.resolveWorkspaceSelection(denied,
		newCategoryRequestAs(http.MethodPost, ScenariosRunPath, "", "user", "outsider@example.com"),
		&krknv1alpha1.WorkspaceReference{Name: "friendly"})
	require.False(t, ok)
	require.Equal(t, http.StatusForbidden, denied.Code)
}

func TestScenarioRunWorkspaceSelectionAndMissingReplayPVC(t *testing.T) {
	h := setupScenarioRunTestHandler("target", map[string]string{"cluster": "YXBpVmVyc2lvbjogdjEK"})
	pvc := testWorkspacePVC()
	require.NoError(t, h.client.Create(t.Context(), pvc))
	readyWorkspace(t, h, "", true)
	base := `{"targetRequestId":"target","targetClusters":{"krkn-operator":["cluster"]},"scenario":{"name":"pod-delete","private":false},"workspace":{"name":"friendly"}}`
	request := newCategoryRequestAs(http.MethodPost, ScenariosRunPath, base, "admin", "admin@example.com")
	response := httptest.NewRecorder()
	h.PostScenarioRun(response, request)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var created ScenarioRunCreateResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &created))
	run := &krknv1alpha1.KrknScenarioRun{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: created.ScenarioRunName}, run))
	require.Equal(t, "friendly", run.Spec.Workspace.Name)
	require.Equal(t, testEntryUID, run.Spec.Workspace.WorkspaceUID)
	require.Equal(t, testWorkspaceUID, run.Spec.Workspace.ClaimUID)
	overlapBody := `{"targetRequestId":"target","targetClusters":{"krkn-operator":["cluster"]},"scenario":{"name":"pod-delete","private":false},"workspace":{"name":"friendly"},"files":[{"name":"config.yaml","content":"e30=","mountPath":"/var/lib/krkn/workspace/config.yaml"}]}`
	overlapped := httptest.NewRecorder()
	h.PostScenarioRun(overlapped, newCategoryRequestAs(http.MethodPost, ScenariosRunPath, overlapBody, "admin", "admin@example.com"))
	require.Equal(t, http.StatusBadRequest, overlapped.Code, overlapped.Body.String())

	replay, err := h.reconstructScenarioRunPayload(request.Context(), run)
	require.NoError(t, err)
	require.Equal(t, run.Spec.Workspace, replay.Workspace)
	require.NoError(t, h.client.Delete(t.Context(), pvc))
	replayJSON, err := json.Marshal(replay)
	require.NoError(t, err)
	replayed := httptest.NewRecorder()
	h.PostScenarioRun(replayed, newCategoryRequestAs(http.MethodPost, ScenariosRunPath, string(replayJSON), "admin", "admin@example.com"))
	require.Equal(t, http.StatusConflict, replayed.Code, replayed.Body.String())
	var runs krknv1alpha1.KrknScenarioRunList
	require.NoError(t, h.client.List(t.Context(), &runs, client.InNamespace("default")))
	require.Len(t, runs.Items, 1, "a replay with a missing PVC must not create another run")

	replacement := testWorkspacePVC()
	replacement.UID = "dc7f90ee-56d0-4521-b014-d588f2af9341"
	require.NoError(t, h.client.Create(t.Context(), replacement))
	replayed = httptest.NewRecorder()
	h.PostScenarioRun(replayed, newCategoryRequestAs(http.MethodPost, ScenariosRunPath, string(replayJSON), "admin", "admin@example.com"))
	require.Equal(t, http.StatusConflict, replayed.Code, replayed.Body.String())
}

func TestScenarioRollbackRequiresWorkspaceAndPersistsSelection(t *testing.T) {
	h := setupScenarioRunTestHandler("target", map[string]string{"cluster": "YXBpVmVyc2lvbjogdjEK"})
	require.NoError(t, h.client.Create(t.Context(), testWorkspacePVC()))
	readyWorkspace(t, h, "", true)
	base := `{"targetRequestId":"target","targetClusters":{"krkn-operator":["cluster"]},"scenario":{"name":"pod-delete","private":false},"rollbackEnabled":true}`
	missing := httptest.NewRecorder()
	h.PostScenarioRun(missing, newCategoryRequestAs(http.MethodPost, ScenariosRunPath, base, "admin", "admin@example.com"))
	require.Equal(t, http.StatusBadRequest, missing.Code, missing.Body.String())

	selected := httptest.NewRecorder()
	h.PostScenarioRun(selected, newCategoryRequestAs(http.MethodPost, ScenariosRunPath,
		`{"targetRequestId":"target","targetClusters":{"krkn-operator":["cluster"]},"scenario":{"name":"pod-delete","private":false},"rollbackEnabled":true,"maxRetries":5,"workspace":{"name":"friendly"}}`, "admin", "admin@example.com"))
	require.Equal(t, http.StatusCreated, selected.Code, selected.Body.String())
	var created ScenarioRunCreateResponse
	require.NoError(t, json.Unmarshal(selected.Body.Bytes(), &created))
	run := &krknv1alpha1.KrknScenarioRun{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: created.ScenarioRunName}, run))
	require.True(t, run.Spec.RollbackEnabled)
	require.Zero(t, run.Spec.MaxRetries)
	replay, err := h.reconstructScenarioRunPayload(t.Context(), run)
	require.NoError(t, err)
	require.True(t, replay.RollbackEnabled)
	require.NotNil(t, replay.MaxRetries)
	require.Zero(t, *replay.MaxRetries)
	require.Equal(t, run.Spec.Workspace, replay.Workspace)
}

func TestGraphRunPinsNodeWorkspace(t *testing.T) {
	pvc := testWorkspacePVC()
	h := graphRunCategoryTestHandler(t, pvc)
	readyWorkspace(t, h, "", true)
	request := GraphRunCreateRequest{
		Graph: map[string]krknv1alpha1.GraphScenarioNode{
			"node-1": {
				Scenario:        publicScenarioReference("pod-delete"),
				Workspace:       &krknv1alpha1.WorkspaceReference{Name: "friendly"},
				RollbackEnabled: true,
			},
		},
		TargetRequestID: "graph-category-target",
		TargetClusters:  map[string][]string{"krkn-operator": {"test-cluster"}},
	}
	body, err := json.Marshal(request)
	require.NoError(t, err)
	response := httptest.NewRecorder()
	h.CreateGraphRun(response, newCategoryRequestAs(http.MethodPost, GraphRunsPath, string(body), "admin", "admin@example.com"))
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var runs krknv1alpha1.KrknGraphRunList
	require.NoError(t, h.client.List(t.Context(), &runs, client.InNamespace("default")))
	require.Len(t, runs.Items, 1)
	require.Equal(t, testWorkspaceUID, runs.Items[0].Spec.Graph["node-1"].Workspace.ClaimUID)
	require.Equal(t, testEntryUID, runs.Items[0].Spec.Graph["node-1"].Workspace.WorkspaceUID)
	require.True(t, runs.Items[0].Spec.Graph["node-1"].RollbackEnabled)
	var returned GraphRunDetailResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &returned))
	require.Empty(t, returned.Spec.Graph["node-1"].Workspace.ClaimName)
}

func TestGraphRollbackRequiresNodeWorkspace(t *testing.T) {
	h := graphRunCategoryTestHandler(t)
	request := GraphRunCreateRequest{
		Graph: map[string]krknv1alpha1.GraphScenarioNode{
			"node-1": {Scenario: publicScenarioReference("pod-delete"), RollbackEnabled: true},
		},
		TargetRequestID: "graph-category-target",
		TargetClusters:  map[string][]string{"krkn-operator": {"test-cluster"}},
	}
	body, err := json.Marshal(request)
	require.NoError(t, err)
	response := httptest.NewRecorder()
	h.CreateGraphRun(response, newCategoryRequestAs(http.MethodPost, GraphRunsPath, string(body), "admin", "admin@example.com"))
	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
}

func TestWorkspaceRejectsStaleStatusAndReplacedAlias(t *testing.T) {
	h := setupScenarioRunTestHandler("target", map[string]string{})
	require.NoError(t, h.client.Create(t.Context(), testWorkspacePVC()))
	entry := readyWorkspace(t, h, "", true)
	request := newCategoryRequestAs(http.MethodPost, ScenariosRunPath, "", "admin", "admin@example.com")
	ref, ok := h.resolveWorkspaceSelection(httptest.NewRecorder(), request, &krknv1alpha1.WorkspaceReference{Name: "friendly"})
	require.True(t, ok)
	entry.Generation++
	require.NoError(t, h.client.Update(t.Context(), entry))
	stale := httptest.NewRecorder()
	_, ok = h.resolveWorkspaceSelection(stale, request, ref)
	require.False(t, ok)
	require.Equal(t, http.StatusConflict, stale.Code)

	require.NoError(t, h.client.Delete(t.Context(), entry))
	replacement := entry.DeepCopy()
	replacement.UID = "30e74857-8e0e-4fa2-9344-789a44bfa0cf"
	replacement.ResourceVersion = ""
	replacement.Status = krknv1alpha1.KrknWorkspaceStatus{}
	require.NoError(t, h.client.Create(t.Context(), replacement))
	replaced := httptest.NewRecorder()
	_, ok = h.resolveWorkspaceSelection(replaced, request, ref)
	require.False(t, ok)
	require.Equal(t, http.StatusConflict, replaced.Code)
}
