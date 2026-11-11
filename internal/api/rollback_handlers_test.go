package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/registry"
	"github.com/krkn-chaos/krkn-operator/pkg/signatureverification"
	"github.com/krkn-chaos/krkn-operator/pkg/workspace"
	"github.com/krkn-chaos/krknctl/pkg/provider"
	"github.com/krkn-chaos/krknctl/pkg/provider/models"
	"github.com/krkn-chaos/krknctl/pkg/verify"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const rollbackSourceUID = "7fa25dbb-7278-4104-90c7-e7744b68e5af"
const rollbackAttemptID = "4f322276-81e4-4d70-9f94-9951ded10e75"
const priorRollbackAttemptID = "5e34f888-ed3a-4b57-82b8-8c59ab5dc4ab"

var testRollbackInspectorImage = "quay.io/krkn-chaos/krkn-operator@sha256:" + strings.Repeat("a", 64)

func rollbackTestHandler(t *testing.T) *Handler {
	t.Helper()
	h := setupScenarioRunTestHandler("target", map[string]string{})
	h.rollbackInspectorImage = testRollbackInspectorImage
	h.rollbackInspectorPullSecrets = `[{"name":"operator-pull-secret"}]`
	digest := "sha256:rollback-test"
	mock := &mockScenarioProvider{
		signature: verify.SignatureSigned,
	}
	h.scenarioProviderFactory = func(provider.Mode) (provider.ScenarioDataProvider, error) { return mock, nil }
	h.rollbackDigestResolver = func(context.Context, bool, *models.RegistryV2) (string, error) { return digest, nil }
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

func rollbackAvailableProbe(t *testing.T, h *Handler) {
	t.Helper()
	pod := testRollbackProbePod(rollbackAttemptID)
	require.NoError(t, h.client.Create(t.Context(), pod))
	pod.Status.Phase = corev1.PodSucceeded
	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Message: "available"}}}}
	require.NoError(t, h.client.Status().Update(t.Context(), pod))
}

func testRollbackProbePod(jobID string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: rollbackProbeName(jobID), Namespace: "default",
			Labels: map[string]string{"krkn-scenario-run-uid": rollbackSourceUID},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name: "inspect", Image: testRollbackInspectorImage,
			Command: []string{"/manager", "inspect-rollback"},
		}}},
	}
}

func rollbackRegistrySecret(t *testing.T, name string, availableToAll bool) *corev1.Secret {
	t.Helper()
	dockerConfig, err := registry.BuildDockerConfigJSON("registry.example.test", "test-user", "test-password")
	require.NoError(t, err)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: "default",
			Labels:      registry.BuildRegistryLabels(registry.AuthTypePassword, nil, availableToAll),
			Annotations: registry.BuildRegistryAnnotations("registry.example.test", "scenarios", "", false, false, "admin"),
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{corev1.DockerConfigJsonKey: dockerConfig},
	}
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
	require.Equal(t, "krkn-operator-krkn-scenario-runner", probe.Spec.ServiceAccountName)
	require.NotNil(t, probe.Spec.AutomountServiceAccountToken)
	require.False(t, *probe.Spec.AutomountServiceAccountToken)
	require.Equal(t, testRollbackInspectorImage, probe.Spec.Containers[0].Image)
	require.Equal(t, []string{"/manager", "inspect-rollback"}, probe.Spec.Containers[0].Command)
	require.Equal(t, []corev1.LocalObjectReference{{Name: "operator-pull-secret"}}, probe.Spec.ImagePullSecrets)
	require.True(t, *probe.Spec.SecurityContext.RunAsNonRoot)
	require.True(t, *probe.Spec.Containers[0].SecurityContext.ReadOnlyRootFilesystem)
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
	require.Equal(t, "workload-rollback", rollback.Spec.Scenario.Name)
	require.NotNil(t, rollback.Spec.Scenario.Private)
	require.False(t, *rollback.Spec.Scenario.Private)
	require.Equal(t, rollbackAttemptID, rollback.Spec.Environment["UUID"])
	require.Equal(t, workspace.MountPath, rollback.Spec.Environment["ROLLBACK_VERSIONS_DIRECTORY"])
	require.Equal(t, rollbackSourceUID, rollback.Spec.RollbackSource.RunUID)
	require.Equal(t, rollbackAttemptID, rollback.Spec.RollbackSource.JobID)
	require.Equal(t, "https://cluster.example.com", rollback.Spec.RollbackSource.ClusterAPIURL)
	require.Equal(t, http.StatusOK, rollbackRequest(h, http.MethodPost, "admin", rollbackAttemptID).Code)

	rollback.Status.Phase = "Failed"
	require.NoError(t, h.client.Update(t.Context(), rollback))
	probe = testRollbackProbePod(rollbackAttemptID)
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

func TestScenarioRollbackProbeCannotInspectWorkspace(t *testing.T) {
	for _, tc := range []struct {
		name    string
		phase   corev1.PodPhase
		waiting string
		age     time.Duration
		message string
	}{
		{name: "mount configuration error", phase: corev1.PodPending, waiting: "CreateContainerConfigError", message: "could not start"},
		{name: "pending timeout", phase: corev1.PodPending, age: 3 * time.Minute, message: "timed out"},
		{name: "failed deadline", phase: corev1.PodFailed, age: 3 * time.Minute, message: "failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := rollbackTestHandler(t)
			pod := testRollbackProbePod(rollbackAttemptID)
			if tc.age > 0 {
				pod.CreationTimestamp = metav1.NewTime(time.Now().Add(-tc.age))
			}
			require.NoError(t, h.client.Create(t.Context(), pod))
			pod.Status.Phase = tc.phase
			if tc.waiting != "" {
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "inspect", State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{Reason: tc.waiting},
				}}}
			}
			require.NoError(t, h.client.Status().Update(t.Context(), pod))

			for range 2 {
				response := rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)
				require.Equal(t, http.StatusOK, response.Code, response.Body.String())
				result := decodeRollbackResponse(t, response)
				require.Equal(t, "unavailable", result.State)
				require.False(t, result.CanExecute)
				require.Contains(t, result.Message, tc.message)
			}
			require.Equal(t, http.StatusConflict, rollbackRequest(h, http.MethodPost, "admin", rollbackAttemptID).Code)
			require.NoError(t, h.client.Get(t.Context(), client.ObjectKeyFromObject(pod), &corev1.Pod{}))
		})
	}
}

func TestScenarioRollbackProbeRequiresPinnedOperatorImage(t *testing.T) {
	for _, image := range []string{"", "quay.io/krkn-chaos/krkn-operator:latest", "quay.io/krkn-chaos/krkn-operator@sha256:bad"} {
		t.Run(image, func(t *testing.T) {
			h := rollbackTestHandler(t)
			h.rollbackInspectorImage = image
			response := rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)
			require.Equal(t, http.StatusOK, response.Code)
			result := decodeRollbackResponse(t, response)
			require.Equal(t, "unavailable", result.State)
			require.False(t, result.CanExecute)
			probe := &corev1.Pod{}
			require.True(t, apierrors.IsNotFound(h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: rollbackProbeName(rollbackAttemptID)}, probe)))
			require.Equal(t, http.StatusConflict, rollbackRequest(h, http.MethodPost, "admin", rollbackAttemptID).Code)
		})
	}
}

func TestRollbackInspectorConfigAcceptsReleaseImageAndRejectsInvalidSecrets(t *testing.T) {
	h := rollbackTestHandler(t)
	h.rollbackInspectorImage = "quay.io/krkn-chaos/krkn-operator:v1.2.0@sha256:" + strings.Repeat("a", 64)
	image, secrets, err := h.rollbackInspectorConfig()
	require.NoError(t, err)
	require.Equal(t, h.rollbackInspectorImage, image)
	require.Equal(t, []corev1.LocalObjectReference{{Name: "operator-pull-secret"}}, secrets)
	h.rollbackInspectorPullSecrets = `[{"name":""}]`
	_, _, err = h.rollbackInspectorConfig()
	require.Error(t, err)
}

func TestScenarioRollbackReplacesLegacyProbeBeforeReadingResult(t *testing.T) {
	h := rollbackTestHandler(t)
	legacy := testRollbackProbePod(rollbackAttemptID)
	legacy.Spec.Containers[0].Image = "quay.io/krkn-chaos/krkn-hub:pod-delete"
	require.NoError(t, h.client.Create(t.Context(), legacy))
	legacy.Status.Phase = corev1.PodSucceeded
	legacy.Status.ContainerStatuses = []corev1.ContainerStatus{{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Message: "available"}}}}
	require.NoError(t, h.client.Status().Update(t.Context(), legacy))
	response := rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)
	require.Equal(t, "checking", decodeRollbackResponse(t, response).State)
	require.True(t, apierrors.IsNotFound(h.client.Get(t.Context(), client.ObjectKeyFromObject(legacy), &corev1.Pod{})))
	response = rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)
	require.Equal(t, "checking", decodeRollbackResponse(t, response).State)
	probe := &corev1.Pod{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKeyFromObject(legacy), probe))
	require.Equal(t, testRollbackInspectorImage, probe.Spec.Containers[0].Image)
}

func TestScenarioRollbackUsesSourcePrivateRegistry(t *testing.T) {
	h := rollbackTestHandler(t)
	require.NoError(t, h.client.Create(t.Context(), rollbackRegistrySecret(t, "source-registry", true)))
	resolvedPrivate := false
	h.rollbackDigestResolver = func(_ context.Context, private bool, config *models.RegistryV2) (string, error) {
		resolvedPrivate = true
		require.True(t, private)
		require.NotNil(t, config)
		require.Equal(t, "registry.example.test", config.RegistryURL)
		return "sha256:private-test", nil
	}
	run := &krknv1alpha1.KrknScenarioRun{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: "source-run"}, run))
	private := true
	run.Spec.Scenario.Private = &private
	run.Spec.Scenario.RegistryName = "source-registry"
	require.NoError(t, h.client.Update(t.Context(), run))

	require.Equal(t, http.StatusOK, rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID).Code)
	probe := &corev1.Pod{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: rollbackProbeName(rollbackAttemptID)}, probe))
	require.Equal(t, []corev1.LocalObjectReference{{Name: "operator-pull-secret"}}, probe.Spec.ImagePullSecrets)
	probe.Status.Phase = corev1.PodSucceeded
	probe.Status.ContainerStatuses = []corev1.ContainerStatus{{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Message: "available"}}}}
	require.NoError(t, h.client.Status().Update(t.Context(), probe))

	response := rollbackRequest(h, http.MethodPost, "admin", rollbackAttemptID)
	require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	require.True(t, resolvedPrivate)
	rollback := &krknv1alpha1.KrknScenarioRun{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: rollbackRunName(rollbackAttemptID)}, rollback))
	require.Equal(t, "workload-rollback", rollback.Spec.Scenario.Name)
	require.NotNil(t, rollback.Spec.Scenario.Private)
	require.True(t, *rollback.Spec.Scenario.Private)
	require.Equal(t, "source-registry", rollback.Spec.Scenario.RegistryName)
}

func TestScenarioRollbackRejectsMissingOrUntrustedImage(t *testing.T) {
	tests := []struct {
		name       string
		provider   *mockScenarioProvider
		digestErr  error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "missing tag",
			provider:   &mockScenarioProvider{},
			digestErr:  registry.ErrTagNotFound,
			wantStatus: http.StatusConflict,
			wantCode:   "rollback_image_unavailable",
		},
		{
			name:       "unsigned image",
			provider:   &mockScenarioProvider{signature: verify.SignatureUnsigned},
			wantStatus: http.StatusConflict,
			wantCode:   "rollback_image_untrusted",
		},
		{
			name:       "untrusted image",
			provider:   &mockScenarioProvider{signature: verify.SignatureUntrusted},
			wantStatus: http.StatusConflict,
			wantCode:   "rollback_image_untrusted",
		},
		{
			name:       "registry error",
			provider:   &mockScenarioProvider{},
			digestErr:  errors.New("registry unavailable"),
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   "rollback_image_check_failed",
		},
		{
			name:       "signature check error",
			provider:   &mockScenarioProvider{signatureErr: errors.New("signature registry unavailable")},
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   "rollback_image_check_failed",
		},
		{
			name:       "unknown signature status",
			provider:   &mockScenarioProvider{signature: verify.SignatureUnknown},
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   "rollback_image_check_failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := rollbackTestHandler(t)
			h.scenarioProviderFactory = func(provider.Mode) (provider.ScenarioDataProvider, error) { return tt.provider, nil }
			if tt.digestErr != nil {
				h.rollbackDigestResolver = func(context.Context, bool, *models.RegistryV2) (string, error) { return "", tt.digestErr }
			}
			rollbackAvailableProbe(t, h)
			available := rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)
			require.Equal(t, http.StatusOK, available.Code)
			result := decodeRollbackResponse(t, available)
			require.Equal(t, "available", result.State)
			require.False(t, result.CanExecute)
			require.NotEmpty(t, result.Message)

			started := rollbackRequest(h, http.MethodPost, "admin", rollbackAttemptID)
			require.Equal(t, tt.wantStatus, started.Code, started.Body.String())
			var failure ErrorResponse
			require.NoError(t, json.Unmarshal(started.Body.Bytes(), &failure))
			require.Equal(t, tt.wantCode, failure.Error)
			run := &krknv1alpha1.KrknScenarioRun{}
			require.True(t, apierrors.IsNotFound(h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: rollbackRunName(rollbackAttemptID)}, run)))
		})
	}
}

func TestScenarioRollbackCachesGETImageCheckButRechecksPOST(t *testing.T) {
	h := rollbackTestHandler(t)
	checks := 0
	h.rollbackDigestResolver = func(context.Context, bool, *models.RegistryV2) (string, error) {
		checks++
		return "sha256:rollback-test", nil
	}
	rollbackAvailableProbe(t, h)
	for range 2 {
		response := rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)
		require.Equal(t, http.StatusOK, response.Code)
		require.True(t, decodeRollbackResponse(t, response).CanExecute)
	}
	require.Equal(t, 1, checks)
	require.Equal(t, http.StatusAccepted, rollbackRequest(h, http.MethodPost, "admin", rollbackAttemptID).Code)
	require.Equal(t, 2, checks)
}

func TestScenarioRollbackPrivateRegistryRequiresAccess(t *testing.T) {
	h := rollbackTestHandler(t)
	run := &krknv1alpha1.KrknScenarioRun{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: "source-run"}, run))
	private := true
	run.Spec.Scenario.Private = &private
	run.Spec.Scenario.RegistryName = "source-registry"
	require.NoError(t, h.client.Update(t.Context(), run))
	require.NoError(t, h.client.Create(t.Context(), rollbackRegistrySecret(t, "source-registry", false)))
	group := newCategoryTestGroup("rollback-runners")
	group.Spec.ClusterPermissions = map[string]krknv1alpha1.ClusterPermissionSet{
		"https://cluster.example.com": {Actions: []string{"view", "run"}},
	}
	require.NoError(t, h.client.Create(t.Context(), group))
	require.NoError(t, h.client.Create(t.Context(), newCategoryTestUser(t, "person@example.com", "rollback-runners")))
	response := rollbackRequest(h, http.MethodGet, "user", rollbackAttemptID)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	result := decodeRollbackResponse(t, response)
	require.Equal(t, "unavailable", result.State)
	require.False(t, result.CanExecute)
	require.Contains(t, result.Message, "registry")
	probe := &corev1.Pod{}
	err := h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: rollbackProbeName(rollbackAttemptID)}, probe)
	require.True(t, apierrors.IsNotFound(err), "registry access must be checked before creating a probe: %v", err)
	require.Equal(t, http.StatusForbidden, rollbackRequest(h, http.MethodPost, "user", rollbackAttemptID).Code)
}

func TestScenarioRollbackPrivateRegistryMissingWorkload(t *testing.T) {
	h := rollbackTestHandler(t)
	run := &krknv1alpha1.KrknScenarioRun{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: "source-run"}, run))
	private := true
	run.Spec.Scenario.Private = &private
	run.Spec.Scenario.RegistryName = "source-registry"
	require.NoError(t, h.client.Update(t.Context(), run))
	require.NoError(t, h.client.Create(t.Context(), rollbackRegistrySecret(t, "source-registry", true)))
	h.rollbackDigestResolver = func(_ context.Context, isPrivate bool, config *models.RegistryV2) (string, error) {
		require.True(t, isPrivate)
		require.NotNil(t, config)
		return "", registry.ErrTagNotFound
	}
	rollbackAvailableProbe(t, h)
	response := rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)
	require.Equal(t, http.StatusOK, response.Code)
	result := decodeRollbackResponse(t, response)
	require.Equal(t, "available", result.State)
	require.False(t, result.CanExecute)
	require.Contains(t, result.Message, rollbackImageName)
	started := rollbackRequest(h, http.MethodPost, "admin", rollbackAttemptID)
	require.Equal(t, http.StatusConflict, started.Code)
	var failure ErrorResponse
	require.NoError(t, json.Unmarshal(started.Body.Bytes(), &failure))
	require.Equal(t, "rollback_image_unavailable", failure.Error)
}

func TestScenarioRollbackRespectsDisabledSignatureVerification(t *testing.T) {
	h := rollbackTestHandler(t)
	require.NoError(t, h.client.Create(t.Context(), &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: signatureverification.ConfigMapName, Namespace: "default"},
		Data:       map[string]string{signatureverification.EnabledKey: "false"},
	}))
	mock := &mockScenarioProvider{signature: verify.SignatureUnsigned}
	h.scenarioProviderFactory = func(provider.Mode) (provider.ScenarioDataProvider, error) { return mock, nil }
	rollbackAvailableProbe(t, h)
	response := rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)
	require.Equal(t, http.StatusOK, response.Code)
	require.True(t, decodeRollbackResponse(t, response).CanExecute)
	require.Equal(t, http.StatusAccepted, rollbackRequest(h, http.MethodPost, "admin", rollbackAttemptID).Code)
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
	result := decodeRollbackResponse(t, response)
	require.Equal(t, "unavailable", result.State)
	require.False(t, result.CanExecute)
	require.Equal(t, http.StatusConflict, rollbackRequest(h, http.MethodPost, "admin", rollbackAttemptID).Code)
	replacement := testWorkspacePVC()
	replacement.UID = "8f53d5ab-fca8-464f-877c-9bb72dcdb319"
	require.NoError(t, h.client.Create(t.Context(), replacement))
	result = decodeRollbackResponse(t, rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID))
	require.Equal(t, "unavailable", result.State)
	require.False(t, result.CanExecute)
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
	require.Equal(t, "unavailable", decodeRollbackResponse(t, response).State)
	probe := &corev1.Pod{}
	require.True(t, apierrors.IsNotFound(h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: rollbackProbeName(rollbackAttemptID)}, probe)))
	require.Equal(t, http.StatusForbidden, rollbackRequest(h, http.MethodPost, "user", rollbackAttemptID).Code)
}

func TestScenarioRollbackSucceededIsIdempotent(t *testing.T) {
	h := rollbackTestHandler(t)
	rollback := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: rollbackRunName(rollbackAttemptID), Namespace: "default",
			Labels: map[string]string{"krkn-rollback-attempt": rollbackAttemptID, "krkn-rollback-iteration": "1"}},
		Spec: krknv1alpha1.KrknScenarioRunSpec{RollbackSource: &krknv1alpha1.RollbackSource{
			RunName: "source-run", RunUID: rollbackSourceUID, JobID: rollbackAttemptID}},
		Status: krknv1alpha1.KrknScenarioRunStatus{Phase: "Succeeded"},
	}
	require.NoError(t, h.client.Create(t.Context(), rollback))
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		response := rollbackRequest(h, method, "admin", rollbackAttemptID)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		result := decodeRollbackResponse(t, response)
		require.Equal(t, "succeeded", result.State)
		require.False(t, result.CanExecute)
		require.Equal(t, rollback.Name, result.RollbackRunName)
	}
	probe := &corev1.Pod{}
	require.True(t, apierrors.IsNotFound(h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: rollbackProbeName(rollbackAttemptID)}, probe)))
	var runs krknv1alpha1.KrknScenarioRunList
	require.NoError(t, h.client.List(t.Context(), &runs, client.InNamespace("default")))
	require.Len(t, runs.Items, 2)
}

func TestScenarioRollbackFailedWithoutFilesIsUnavailable(t *testing.T) {
	h := rollbackTestHandler(t)
	rollback := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: rollbackRunName(rollbackAttemptID), Namespace: "default",
			Labels: map[string]string{"krkn-rollback-attempt": rollbackAttemptID, "krkn-rollback-iteration": "1"}},
		Spec: krknv1alpha1.KrknScenarioRunSpec{RollbackSource: &krknv1alpha1.RollbackSource{
			RunName: "source-run", RunUID: rollbackSourceUID, JobID: rollbackAttemptID}},
		Status: krknv1alpha1.KrknScenarioRunStatus{Phase: "Failed"},
	}
	require.NoError(t, h.client.Create(t.Context(), rollback))
	probe := testRollbackProbePod(rollbackAttemptID)
	require.NoError(t, h.client.Create(t.Context(), probe))
	probe.Status.Phase = corev1.PodSucceeded
	probe.Status.ContainerStatuses = []corev1.ContainerStatus{{State: corev1.ContainerState{
		Terminated: &corev1.ContainerStateTerminated{Message: "absent"}}}}
	require.NoError(t, h.client.Status().Update(t.Context(), probe))
	response := rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)
	require.Equal(t, http.StatusOK, response.Code)
	result := decodeRollbackResponse(t, response)
	require.Equal(t, "unavailable", result.State)
	require.False(t, result.CanExecute)
	require.Contains(t, result.Message, "failed")
}

func TestScenarioRollbackRetryingDoesNotInspect(t *testing.T) {
	h := rollbackTestHandler(t)
	run := &krknv1alpha1.KrknScenarioRun{}
	require.NoError(t, h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: "source-run"}, run))
	run.Status.ClusterJobs[0].Phase = "Retrying"
	require.NoError(t, h.client.Update(t.Context(), run))
	response := rollbackRequest(h, http.MethodGet, "admin", rollbackAttemptID)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "unavailable", decodeRollbackResponse(t, response).State)
	require.False(t, decodeRollbackResponse(t, response).CanExecute)
	probe := &corev1.Pod{}
	require.True(t, apierrors.IsNotFound(h.client.Get(t.Context(), client.ObjectKey{Namespace: "default", Name: rollbackProbeName(rollbackAttemptID)}, probe)))
	require.Equal(t, http.StatusConflict, rollbackRequest(h, http.MethodPost, "admin", rollbackAttemptID).Code)
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
	require.Equal(t, testRollbackInspectorImage, probe.Spec.Containers[0].Image)
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
