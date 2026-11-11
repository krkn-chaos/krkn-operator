package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/kubeconfig"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	"github.com/krkn-chaos/krkn-operator/pkg/workspace"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// RollbackState is the observable state of an attempt's rollback action.
type RollbackState string

const (
	RollbackStateChecking    RollbackState = "checking"
	RollbackStateAvailable   RollbackState = "available"
	RollbackStateAbsent      RollbackState = "absent"
	RollbackStateUnavailable RollbackState = "unavailable"
	RollbackStateRunning     RollbackState = "running"
	RollbackStateSucceeded   RollbackState = "succeeded"
	RollbackStateFailed      RollbackState = "failed"
	RollbackStateError       RollbackState = "error"
)

// RollbackAvailability describes files in one pinned scenario attempt.
type RollbackAvailability struct {
	JobID           string        `json:"jobId"`
	State           RollbackState `json:"state"`
	CanExecute      bool          `json:"canExecute"`
	RollbackRunName string        `json:"rollbackRunName,omitempty"`
	Message         string        `json:"message,omitempty"`
}

func (h *Handler) canExecuteRollback(ctx context.Context, job *krknv1alpha1.ClusterJobStatus) (bool, error) {
	if job.ClusterAPIURL == "" {
		return false, nil
	}
	if auth.IsAdmin(ctx) {
		return true, nil
	}
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil || job.ClusterAPIURL == "" {
		return false, nil
	}
	return groupauth.HasClusterPermission(ctx, h.client, claims.UserID, h.namespace, job.ClusterAPIURL, groupauth.ActionRun)
}

func (h *Handler) validateOriginalKubeconfig(ctx context.Context, run *krknv1alpha1.KrknScenarioRun, job *krknv1alpha1.ClusterJobStatus, jobID string) error {
	_, err := kubeconfig.LoadOriginalAttempt(ctx, h.client, h.namespace, run, jobID, job.ClusterName)
	return err
}

var rollbackInspectorImagePattern = regexp.MustCompile(`^[^@[:space:]]+@sha256:[a-f0-9]{64}$`)

func (h *Handler) rollbackInspectorConfig() (string, []corev1.LocalObjectReference, error) {
	if !rollbackInspectorImagePattern.MatchString(h.rollbackInspectorImage) {
		return "", nil, errors.New("rollback inspector image must be pinned to a SHA-256 digest")
	}
	var pullSecrets []corev1.LocalObjectReference
	if h.rollbackInspectorPullSecrets != "" {
		if err := json.Unmarshal([]byte(h.rollbackInspectorPullSecrets), &pullSecrets); err != nil {
			return "", nil, fmt.Errorf("decode rollback inspector pull secrets: %w", err)
		}
		for _, secret := range pullSecrets {
			if secret.Name == "" {
				return "", nil, errors.New("rollback inspector pull secret name is empty")
			}
		}
	}
	return h.rollbackInspectorImage, pullSecrets, nil
}

func rollbackProbeName(jobID string) string { return "krkn-rb-probe-" + jobID }
func rollbackRunName(jobID string) string   { return "krkn-rollback-" + jobID }

func rollbackRunAttemptName(jobID string, iteration int) string {
	if iteration == 1 {
		return rollbackRunName(jobID)
	}
	return rollbackRunName(jobID) + "-" + strconv.Itoa(iteration)
}

func (h *Handler) latestRollbackRun(ctx context.Context, runUID, jobID string) (*krknv1alpha1.KrknScenarioRun, int, error) {
	list := &krknv1alpha1.KrknScenarioRunList{}
	if err := h.client.List(ctx, list, client.InNamespace(h.namespace), client.MatchingLabels{"krkn-rollback-attempt": jobID}); err != nil {
		return nil, 0, err
	}
	var latest *krknv1alpha1.KrknScenarioRun
	maxIteration := 0
	for i := range list.Items {
		run := &list.Items[i]
		if run.Spec.RollbackSource == nil || run.Spec.RollbackSource.RunUID != runUID || run.Spec.RollbackSource.JobID != jobID {
			return nil, 0, errors.New("rollback run identity conflicts with this attempt")
		}
		iteration, err := strconv.Atoi(run.Labels["krkn-rollback-iteration"])
		if err != nil || iteration < 1 || run.Name != rollbackRunAttemptName(jobID, iteration) {
			return nil, 0, errors.New("rollback run iteration is invalid")
		}
		if iteration > maxIteration {
			latest, maxIteration = run, iteration
		}
	}
	return latest, maxIteration, nil
}

func (h *Handler) loadRollbackAttempt(w http.ResponseWriter, r *http.Request, runName, jobID string, action groupauth.Action) (*krknv1alpha1.KrknScenarioRun, *krknv1alpha1.ClusterJobStatus, bool) {
	if _, err := uuid.Parse(jobID); err != nil || runName == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "Invalid rollback run or attempt ID"})
		return nil, nil, false
	}
	run := &krknv1alpha1.KrknScenarioRun{}
	if err := h.client.Get(r.Context(), client.ObjectKey{Namespace: h.namespace, Name: runName}, run); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{Error: "not_found", Message: "Scenario run not found"})
		} else {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to load scenario run"})
		}
		return nil, nil, false
	}
	for i := range run.Status.ClusterJobs {
		job := &run.Status.ClusterJobs[i]
		image := ""
		if job.JobID == jobID {
			image = job.ScenarioImage
		} else {
			for _, attempt := range job.AttemptHistory {
				if attempt.JobID == jobID {
					image = attempt.ScenarioImage
					break
				}
			}
		}
		// An attempt without an image never reached a scenario Pod.
		if image == "" {
			continue
		}
		if !h.checkJobAccess(w, r, job, action, "rollback") {
			return nil, nil, false
		}
		return run, job, true
	}
	writeJSONError(w, http.StatusNotFound, ErrorResponse{Error: "not_found", Message: "Scenario attempt not found"})
	return nil, nil, false
}

func (h *Handler) validateRollbackWorkspace(ctx context.Context, run *krknv1alpha1.KrknScenarioRun) error {
	ref := run.Spec.Workspace
	if !run.Spec.RollbackEnabled || ref == nil {
		return workspace.ErrUnavailable
	}
	return workspace.Validate(ctx, h.client, workspace.Binding{Namespace: h.namespace, Reference: *ref})
}

func (h *Handler) rollbackProbe(ctx context.Context, run *krknv1alpha1.KrknScenarioRun, job *krknv1alpha1.ClusterJobStatus, jobID string) (RollbackState, string, error) {
	image, pullSecrets, err := h.rollbackInspectorConfig()
	if err != nil {
		return RollbackStateUnavailable, "Rollback inspector image is unavailable", nil
	}
	pod := &corev1.Pod{}
	key := client.ObjectKey{Namespace: h.namespace, Name: rollbackProbeName(jobID)}
	err = h.client.Get(ctx, key, pod)
	if err == nil {
		if pod.Labels["krkn-scenario-run-uid"] != string(run.UID) {
			return RollbackStateError, "", errors.New("rollback probe belongs to another run")
		}
		if len(pod.Spec.Containers) != 1 || pod.Spec.Containers[0].Image != image || len(pod.Spec.Containers[0].Command) != 2 || pod.Spec.Containers[0].Command[0] != "/manager" || pod.Spec.Containers[0].Command[1] != "inspect-rollback" {
			if pod.DeletionTimestamp == nil {
				if err := h.client.Delete(ctx, pod); err != nil && !apierrors.IsNotFound(err) {
					return RollbackStateError, "", fmt.Errorf("replace stale rollback probe: %w", err)
				}
			}
			return RollbackStateChecking, "", nil
		}
		// A missing read-only subPath leaves the container in CreateContainerConfigError.
		// Keep the failed probe so polling cannot delete and recreate it forever.
		for _, container := range pod.Status.ContainerStatuses {
			if container.State.Waiting != nil && container.State.Waiting.Reason == "CreateContainerConfigError" {
				return RollbackStateUnavailable, "Rollback unavailable: inspection pod could not start", nil
			}
		}
		if pod.Status.Phase == corev1.PodFailed {
			return RollbackStateUnavailable, "Rollback unavailable: inspection pod failed", nil
		}
		if !pod.CreationTimestamp.IsZero() && pod.Status.Phase != corev1.PodSucceeded && time.Since(pod.CreationTimestamp.Time) > 2*time.Minute {
			return RollbackStateUnavailable, "Rollback unavailable: inspection timed out", nil
		}
		if !pod.CreationTimestamp.IsZero() && pod.Status.Phase == corev1.PodSucceeded && time.Since(pod.CreationTimestamp.Time) > 30*time.Second {
			if err := h.client.Delete(ctx, pod); err != nil && !apierrors.IsNotFound(err) {
				return RollbackStateError, "", fmt.Errorf("refresh rollback probe: %w", err)
			}
			return RollbackStateChecking, "", nil
		}
		switch pod.Status.Phase {
		case corev1.PodSucceeded:
			if len(pod.Status.ContainerStatuses) == 0 || pod.Status.ContainerStatuses[0].State.Terminated == nil {
				return RollbackStateError, "", errors.New("rollback probe has no result")
			}
			result := pod.Status.ContainerStatuses[0].State.Terminated.Message
			if result == "available" || result == "absent" {
				return RollbackState(result), "", nil
			}
			return RollbackStateError, "", errors.New("rollback probe returned an invalid result")
		default:
			return RollbackStateChecking, "", nil
		}
	}
	if !apierrors.IsNotFound(err) {
		return RollbackStateError, "", fmt.Errorf("load rollback probe: %w", err)
	}
	subPath, err := workspace.SubPath(run.UID, job.ProviderName, job.ClusterName, jobID)
	if err != nil {
		return RollbackStateError, "", err
	}
	volume, mount, err := workspace.Mount(workspace.Binding{Namespace: h.namespace, Reference: *run.Spec.Workspace},
		workspace.MountOptions{VolumeName: "workspace", MountPath: workspace.MountPath, SubPath: subPath, ReadOnly: true})
	if err != nil {
		return RollbackStateError, "", fmt.Errorf("prepare rollback inspection mount: %w", err)
	}
	falseValue := false
	trueValue := true
	deadline := int64(60)
	uid := int64(1001)
	pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: key.Name, Namespace: h.namespace,
			Labels:          map[string]string{"krkn-scenario-run-uid": string(run.UID), "krkn-rollback-attempt": jobID},
			OwnerReferences: []metav1.OwnerReference{{APIVersion: krknv1alpha1.GroupVersion.String(), Kind: "KrknScenarioRun", Name: run.Name, UID: run.UID}},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "krkn-operator-krkn-scenario-runner",
			RestartPolicy:      corev1.RestartPolicyNever, AutomountServiceAccountToken: &falseValue,
			ImagePullSecrets:      pullSecrets,
			ActiveDeadlineSeconds: &deadline,
			SecurityContext:       &corev1.PodSecurityContext{RunAsUser: &uid, RunAsGroup: &uid, FSGroup: &uid, RunAsNonRoot: &trueValue, SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault}},
			Volumes:               []corev1.Volume{volume},
			Containers: []corev1.Container{{
				Name: "inspect", Image: image, Command: []string{"/manager", "inspect-rollback"},
				Env:             []corev1.EnvVar{{Name: "UUID", Value: jobID}, {Name: "ROLLBACK_VERSIONS_DIRECTORY", Value: workspace.MountPath}},
				VolumeMounts:    []corev1.VolumeMount{mount},
				SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &falseValue, ReadOnlyRootFilesystem: &trueValue, Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}}},
			}},
		},
	}
	if err := h.client.Create(ctx, pod); err != nil && !apierrors.IsAlreadyExists(err) {
		return RollbackStateError, "", fmt.Errorf("create rollback probe: %w", err)
	}
	return RollbackStateChecking, "", nil
}

// HandleScenarioRollback inspects retained files or launches a rollback run for one attempt.
// @Summary Inspect or execute rollback for a scenario attempt
// @Description GET checks the pinned workspace for Krkn rollback files. POST starts a rollback run only when files are available.
// @Tags scenarios
// @Produce json
// @Param runName path string true "Scenario run name"
// @Param jobID path string true "Original attempt ID"
// @Success 200 {object} RollbackAvailability
// @Success 202 {object} RollbackAvailability
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Security BearerAuth
// @Router /scenarios/run/{runName}/rollback/{jobID} [get]
// @Router /scenarios/run/{runName}/rollback/{jobID} [post]
func (h *Handler) HandleScenarioRollback(w http.ResponseWriter, r *http.Request, runName, jobID string) {
	action := groupauth.ActionView
	if r.Method == http.MethodPost {
		action = groupauth.ActionRun
	} else if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method_not_allowed", Message: "Method not allowed"})
		return
	}
	run, job, ok := h.loadRollbackAttempt(w, r, runName, jobID, action)
	if !ok {
		return
	}
	canExecute := false
	if r.Method == http.MethodGet {
		var err error
		canExecute, err = h.canExecuteRollback(r.Context(), job)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to check rollback permission"})
			return
		}
	}
	existing, iteration, err := h.latestRollbackRun(r.Context(), string(run.UID), jobID)
	if err != nil {
		log.FromContext(r.Context()).Error(err, "Failed to load rollback run", "runName", runName, "jobID", jobID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to load rollback run"})
		return
	}
	if existing != nil {
		switch existing.Status.Phase {
		case "Succeeded":
			writeJSON(w, http.StatusOK, RollbackAvailability{JobID: jobID, State: RollbackStateSucceeded, RollbackRunName: existing.Name})
			return
		case "", "Pending", "Running":
			writeJSON(w, http.StatusOK, RollbackAvailability{JobID: jobID, State: RollbackStateRunning, RollbackRunName: existing.Name})
			return
		}
	}
	if job.Phase == "Running" || job.Phase == "Pending" || job.Phase == "Retrying" {
		if r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, RollbackAvailability{JobID: jobID, State: RollbackStateUnavailable, Message: "Scenario attempt is still active"})
		} else {
			writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "scenario_active", Message: "Scenario attempt is still active"})
		}
		return
	}
	if r.Method == http.MethodGet && !canExecute {
		writeJSON(w, http.StatusOK, RollbackAvailability{JobID: jobID, State: RollbackStateUnavailable, Message: "Cluster run permission is required"})
		return
	}
	if err := h.validateRollbackWorkspace(r.Context(), run); err != nil {
		if r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, RollbackAvailability{JobID: jobID, State: RollbackStateUnavailable, Message: "Workspace PVC is unavailable"})
		} else {
			writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "workspace_unavailable", Message: "Workspace PVC is unavailable"})
		}
		return
	}
	if run.Spec.Scenario.Private != nil && *run.Spec.Scenario.Private {
		secret, err := h.loadRegistrySecret(r.Context(), run.Spec.Scenario.RegistryName)
		if err != nil {
			log.FromContext(r.Context()).Error(err, "Failed to load original scenario registry for rollback", "registryName", run.Spec.Scenario.RegistryName)
			if r.Method == http.MethodGet {
				writeJSON(w, http.StatusOK, RollbackAvailability{JobID: jobID, State: RollbackStateUnavailable, Message: "The original scenario registry is unavailable"})
			} else {
				writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "rollback_registry_unavailable", Message: "The original scenario registry is unavailable"})
			}
			return
		}
		if !h.canAccessRegistry(r.Context(), secret) {
			if r.Method == http.MethodGet {
				writeJSON(w, http.StatusOK, RollbackAvailability{JobID: jobID, State: RollbackStateUnavailable, Message: "Access to the original scenario registry is required"})
			} else {
				writeJSONError(w, http.StatusForbidden, ErrorResponse{Error: "forbidden", Message: "Access to the original scenario registry is required"})
			}
			return
		}
	}
	kubeconfigErr := h.validateOriginalKubeconfig(r.Context(), run, job, jobID)
	if r.Method == http.MethodPost && kubeconfigErr != nil {
		writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "target_unavailable", Message: "Original attempt kubeconfig is unavailable"})
		return
	}
	if kubeconfigErr != nil {
		canExecute = false
	}
	state, probeMessage, err := h.rollbackProbe(r.Context(), run, job, jobID)
	if err != nil {
		log.FromContext(r.Context()).Error(err, "Failed to inspect rollback files", "runName", runName, "jobID", jobID)
		if r.Method == http.MethodPost {
			writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{Error: "rollback_inspection_failed", Message: "Rollback inspection failed"})
		} else {
			writeJSON(w, http.StatusOK, RollbackAvailability{JobID: jobID, State: RollbackStateError, Message: "Rollback inspection failed"})
		}
		return
	}
	if r.Method == http.MethodGet {
		response := RollbackAvailability{JobID: jobID, State: state, CanExecute: canExecute, Message: probeMessage}
		if kubeconfigErr != nil && state == RollbackStateAvailable {
			response.Message = "Original attempt kubeconfig is unavailable"
		} else if !canExecute && state == RollbackStateAvailable {
			response.Message = "Cluster run permission is required"
		}
		if existing != nil {
			response.RollbackRunName = existing.Name
			if existing.Status.Phase == "Failed" || existing.Status.Phase == "PartiallyFailed" || existing.Status.Phase == "MaxRetriesExceeded" {
				switch state {
				case "available":
					response.State = "failed"
					response.Message = "Previous rollback failed; files are still available"
				case "absent":
					response.State = "unavailable"
					response.Message = "Previous rollback failed and files are no longer available"
				}
			}
		}
		if response.State != RollbackStateAvailable && response.State != RollbackStateFailed {
			response.CanExecute = false
		}
		if response.CanExecute && (response.State == RollbackStateAvailable || response.State == RollbackStateFailed) {
			imageCheck := h.checkRollbackImage(r.Context(), run, true)
			if !imageCheck.available() {
				response.CanExecute = false
				response.Message = imageCheck.message
			}
		}
		writeJSON(w, http.StatusOK, response)
		return
	}
	if state != RollbackStateAvailable {
		message := probeMessage
		if message == "" {
			message = "Rollback files are not available"
		}
		writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "rollback_unavailable", Message: message})
		return
	}
	if job.ClusterAPIURL == "" {
		writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "target_unavailable", Message: "Original target identity is unavailable"})
		return
	}
	imageCheck := h.checkRollbackImage(r.Context(), run, false)
	if !imageCheck.available() {
		writeJSONError(w, imageCheck.status, ErrorResponse{Error: imageCheck.code, Message: imageCheck.message})
		return
	}
	// Resolve the rollback workload from the same registry as the source scenario.
	private := run.Spec.Scenario.Private != nil && *run.Spec.Scenario.Private
	registryName := ""
	if private {
		registryName = run.Spec.Scenario.RegistryName
	}
	claims := auth.GetClaimsFromContext(r.Context())
	owner := ""
	if claims != nil {
		owner = claims.UserID
	}
	rollbackName := rollbackRunAttemptName(jobID, iteration+1)
	rollback := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: rollbackName, Namespace: h.namespace, Labels: map[string]string{"krkn-rollback-source": run.Name, "krkn-rollback-attempt": jobID, "krkn-rollback-iteration": strconv.Itoa(iteration + 1)}},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			Scenario:        krknv1alpha1.ScenarioReference{Name: rollbackImageName, Private: &private, RegistryName: registryName},
			TargetRequestID: run.Spec.TargetRequestID,
			TargetClusters:  map[string][]string{job.ProviderName: {job.ClusterName}},
			OwnerUserID:     owner,
			KubeconfigPath:  "/home/krkn/.kube/config",
			Environment:     map[string]string{"UUID": jobID, "ROLLBACK_VERSIONS_DIRECTORY": workspace.MountPath},
			Workspace:       run.Spec.Workspace.DeepCopy(),
			RollbackSource:  &krknv1alpha1.RollbackSource{RunName: run.Name, RunUID: string(run.UID), JobID: jobID, ClusterAPIURL: job.ClusterAPIURL},
		},
	}
	if err := h.client.Create(r.Context(), rollback); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to create rollback run"})
			return
		}
		created := &krknv1alpha1.KrknScenarioRun{}
		if getErr := h.client.Get(r.Context(), client.ObjectKey{Namespace: h.namespace, Name: rollbackName}, created); getErr != nil || created.Spec.RollbackSource == nil || created.Spec.RollbackSource.RunUID != string(run.UID) || created.Spec.RollbackSource.JobID != jobID {
			writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "conflict", Message: "Rollback attempt was created concurrently with a different identity"})
			return
		}
	}
	if probe := rollbackProbeName(jobID); probe != "" {
		_ = h.client.Delete(r.Context(), &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: probe, Namespace: h.namespace}})
	}
	writeJSON(w, http.StatusAccepted, RollbackAvailability{JobID: jobID, State: RollbackStateRunning, RollbackRunName: rollbackName})
}
