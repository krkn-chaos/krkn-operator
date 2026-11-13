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

Assisted-by: Claude Sonnet 4.5 (claude-sonnet-4-5@20250929)
*/

// Package controller implements Kubernetes controllers for krkn-operator custom resources.
// It reconciles KrknScenarioRun, KrknTargetRequest, and KrknOperatorTargetProviderConfig resources.
package controller

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/kubeconfig"

	"github.com/google/uuid"
	"github.com/krkn-chaos/krkn-operator/pkg/cloudcreds"
	"github.com/krkn-chaos/krkn-operator/pkg/workspace"
	krknctlconfig "github.com/krkn-chaos/krknctl/pkg/config"
	krknctlprovider "github.com/krkn-chaos/krknctl/pkg/provider"
	krknctlmodels "github.com/krkn-chaos/krknctl/pkg/provider/models"
	"github.com/krkn-chaos/krknctl/pkg/verify"
)

// KrknScenarioRunReconciler reconciles a KrknScenarioRun object
type KrknScenarioRunReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Clientset kubernetes.Interface
	Namespace string
	// signatureVerifier is injectable for controller unit tests; production
	// reconciliation uses krknctl's verifier when it is nil.
	signatureVerifier imageSignatureVerifier
	imageResolver     imageResolver
}

// Check storage before starting a chaos scenario. Some RWX backends allow the
// PVC root to be written but create kubelet subPath directories with ownership
// that prevents UID 1001 from writing inside them.
const rollbackWorkspaceWriteCheck = `if ! (umask 077; probe="$ROLLBACK_VERSIONS_DIRECTORY/.krkn-workspace-write-probe"; : > "$probe" && rm "$probe"); then echo "rollback workspace is not writable: $ROLLBACK_VERSIONS_DIRECTORY" >&2; exit 1; fi; `

// Public Krkn Hub images render config.yaml from their image template. Refuse
// rollback-enabled execution if that template does not consume the environment
// variable, since Krkn otherwise falls back to container-local storage.
func rollbackConfigTemplateCheck(templatePath string) string {
	return fmt.Sprintf(`if ! grep -E '^[[:space:]]*rollback_versions_directory:' %q | grep -Eq '\$(\{ROLLBACK_VERSIONS_DIRECTORY\}|ROLLBACK_VERSIONS_DIRECTORY)'; then echo 'scenario image does not map ROLLBACK_VERSIONS_DIRECTORY into config.yaml' >&2; exit 1; fi; `, templatePath)
}

// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknscenarioruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknscenarioruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknscenarioruns/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/log,verbs=get
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krkntargetrequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknoperatortargets,verbs=get;list;watch

// preparedJobResources holds all resources prepared for creating a scenario pod
type preparedJobResources struct {
	jobID             string
	clusterName       string
	clusterAPIURL     string
	containerImage    string
	kubeconfigPath    string
	volumes           []corev1.Volume
	volumeMounts      []corev1.VolumeMount
	envVars           []corev1.EnvVar
	imagePullSecrets  []corev1.LocalObjectReference
	createdConfigMaps []string // names of created ConfigMaps for cleanup
	createdSecrets    []string // names of created Secrets for cleanup
}

// clusterTarget holds the per-cluster parameters needed to create a scenario job.
type clusterTarget struct {
	providerName     string
	clusterName      string
	clusterAPIURL    string
	existingJobIndex int // -1 for new jobs, ≥ 0 for retry jobs
	jobID            string
	statusIndex      int
}

// jobOutcome captures the result of a parallel job-creation attempt.
type jobOutcome struct {
	target  clusterTarget
	jobID   string
	podName string
	image   string
	err     error
}

const (
	scenarioCancellationAnnotation = "krkn.dev/cancel-requested"
	scenarioCancellationPath       = "/var/run/krkn-cancellation/requested"
)

func (r *KrknScenarioRunReconciler) ownerCancellationRequested(
	ctx context.Context,
	scenarioRun *krknv1alpha1.KrknScenarioRun,
) (bool, error) {
	for _, owner := range scenarioRun.OwnerReferences {
		if owner.APIVersion != krknv1alpha1.GroupVersion.String() || owner.Kind != "KrknAIRun" {
			continue
		}
		var parent krknv1alpha1.KrknAIRun
		if err := r.Get(ctx, types.NamespacedName{Name: owner.Name, Namespace: scenarioRun.Namespace}, &parent); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, fmt.Errorf("read parent KrknAIRun %q cancellation intent: %w", owner.Name, err)
		}
		if parent.UID != owner.UID {
			return false, nil
		}
		return parent.Spec.CancelRequested ||
			parent.Status.Phase == aiRunPhaseCancelling ||
			parent.Status.Phase == aiRunPhaseCancelled, nil
	}
	return false, nil
}

var errScenarioRunCancelled = errors.New("scenario cancellation requested")

func (r *KrknScenarioRunReconciler) cancellationRequested(
	ctx context.Context,
	scenarioRun *krknv1alpha1.KrknScenarioRun,
	clusterName string,
) (bool, error) {
	current := scenarioRun
	var latest krknv1alpha1.KrknScenarioRun
	err := r.Get(ctx, types.NamespacedName{Name: scenarioRun.Name, Namespace: scenarioRun.Namespace}, &latest)
	if err == nil {
		if scenarioRun.UID != "" && latest.UID != scenarioRun.UID {
			return true, nil
		}
		current = &latest
	} else if apierrors.IsNotFound(err) {
		if scenarioRun.UID != "" {
			return true, nil
		}
	} else {
		return false, err
	}
	if !current.DeletionTimestamp.IsZero() ||
		current.Spec.CancelRequested || current.Status.Phase == "Cancelling" || current.Status.Phase == "Cancelled" {
		return true, nil
	}
	if clusterName != "" {
		for _, job := range current.Status.ClusterJobs {
			if job.ClusterName == clusterName && job.CancelRequested {
				return true, nil
			}
		}
	}
	return r.ownerCancellationRequested(ctx, current)
}

// getOwnerLabel returns the sanitized owner label value for a scenario run.
// If the scenario run has no OwnerUserID set, returns an empty string.
// The label value is sanitized to comply with Kubernetes label requirements (RFC 1123).
func getOwnerLabel(scenarioRun *krknv1alpha1.KrknScenarioRun) string {
	if scenarioRun.Spec.OwnerUserID == "" {
		return ""
	}
	sanitized := strings.ReplaceAll(scenarioRun.Spec.OwnerUserID, "@", "-")
	sanitized = strings.ReplaceAll(sanitized, ".", "-")
	return strings.ToLower(sanitized)
}

// buildContainerImage resolves a public scenario reference through krknctl.
// Private references must be resolved with buildContainerImageFromReference
// after loading the operator-managed registry Secret.
func buildContainerImage(spec *krknv1alpha1.KrknScenarioRunSpec, config *krknctlconfig.Config) (string, error) {
	if err := spec.Scenario.Validate(); err != nil {
		return "", fmt.Errorf("invalid scenario reference: %w", err)
	}
	if *spec.Scenario.Private {
		return "", fmt.Errorf("private registry configuration is required")
	}
	return buildContainerImageFromReference(spec.Scenario, config, nil)
}

// buildContainerImageFromReference resolves a scenario through krknctl. A
// private registry configuration is required for private references.
func buildContainerImageFromReference(
	reference krknv1alpha1.ScenarioReference,
	config *krknctlconfig.Config,
	privateRegistry *krknctlmodels.RegistryV2,
) (string, error) {
	if err := reference.Validate(); err != nil {
		return "", fmt.Errorf("invalid scenario reference: %w", err)
	}
	scenarioTag := krknctlmodels.ScenarioTag{Name: reference.Name}
	if *reference.Private {
		if privateRegistry == nil {
			return "", fmt.Errorf("private registry configuration is required")
		}
		return krknctlprovider.ImageReference(privateRegistry.GetPrivateRegistryURI(), scenarioTag), nil
	}

	imageURI, err := config.GetQuayImageURI()
	if err != nil {
		return "", fmt.Errorf("failed to build public scenario image URI: %w", err)
	}
	return krknctlprovider.ImageReference(imageURI, scenarioTag), nil
}

// Reconcile handles the reconciliation loop for KrknScenarioRun
func (r *KrknScenarioRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop started",
		"scenarioRun", req.Name,
		"namespace", req.Namespace)

	// Fetch the KrknScenarioRun instance
	var scenarioRun krknv1alpha1.KrknScenarioRun
	if err := r.Get(ctx, req.NamespacedName, &scenarioRun); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("scenarioRun not found, probably deleted", "scenarioRun", req.Name)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch KrknScenarioRun")
		return ctrl.Result{}, err
	}
	if !scenarioRun.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}
	if !scenarioRun.Spec.CancelRequested &&
		(scenarioRun.Status.Phase == "Cancelling" || scenarioRun.Status.Phase == "Cancelled") {
		scenarioRun.Spec.CancelRequested = true
		if err := r.Update(ctx, &scenarioRun); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, nil
			}
			return ctrl.Result{}, fmt.Errorf("persist scenario cancellation phase: %w", err)
		}
		return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, nil
	}
	if !scenarioRun.Spec.CancelRequested {
		parentCancelled, err := r.ownerCancellationRequested(ctx, &scenarioRun)
		if err != nil {
			return ctrl.Result{}, err
		}
		if parentCancelled {
			scenarioRun.Spec.CancelRequested = true
			if err := r.Update(ctx, &scenarioRun); err != nil {
				if apierrors.IsConflict(err) {
					return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, nil
				}
				return ctrl.Result{}, fmt.Errorf("persist parent cancellation intent: %w", err)
			}
			return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, nil
		}
	}
	pendingMigration, err := migrationPending(ctx, r.Client, scenarioRun.Namespace, scenarioRun.Annotations)
	if err != nil {
		return ctrl.Result{}, err
	}
	if pendingMigration {
		logger.Info("skipping ScenarioRun while the CRD migration is pending", "scenarioRun", scenarioRun.Name)
		return ctrl.Result{RequeueAfter: migrationGuardRequeue}, nil
	}
	reference, legacy, referenceErr := scenarioRun.Spec.ResolveScenarioReference()
	if referenceErr == nil && legacy {
		scenarioRun.Spec.Scenario = reference
		if err := r.Update(ctx, &scenarioRun); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, nil
			}
			return ctrl.Result{}, fmt.Errorf("migrate legacy scenario identity: %w", err)
		}
	}

	// Initialize status if first reconcile
	if scenarioRun.Status.Phase == "" {
		// Calculate total targets
		totalTargets := 0
		for _, clusters := range scenarioRun.Spec.TargetClusters {
			totalTargets += len(clusters)
		}

		logger.Info("initializing scenarioRun status",
			"scenarioRun", scenarioRun.Name,
			"totalTargets", totalTargets,
			"targetClusters", scenarioRun.Spec.TargetClusters)

		scenarioRun.Status.Phase = "Pending"
		scenarioRun.Status.TotalTargets = totalTargets
		scenarioRun.Status.ClusterJobs = make([]krknv1alpha1.ClusterJobStatus, 0)
		if err := r.Status().Update(ctx, &scenarioRun); err != nil {
			// If it's a conflict error, just requeue - the object was modified concurrently
			if apierrors.IsConflict(err) {
				logger.Info("conflict on status initialization, will retry on next reconcile")
				return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, nil
			}
			logger.Error(err, "failed to initialize status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, nil
	}

	// Snapshot status BEFORE any job creation so that appended failure entries are
	// detected as changes and persisted to the API server at the end of the reconcile.
	originalStatus := scenarioRun.Status.DeepCopy()
	if updated := backfillLegacyGraphResiliencyScoreStatuses(&scenarioRun); updated > 0 {
		logger.Info("backfilled legacy graph resiliency score statuses",
			"scenarioRun", scenarioRun.Name,
			"updatedScores", updated)
	}

	// Collect targets that need job creation (serial — reads Status.ClusterJobs safely).
	// getClusterAPIURL is called once per cluster here so the error-handler below can
	// reuse the result without a second API round-trip to KrknTargetRequest.
	var targets []clusterTarget
	for providerName, clusterNames := range scenarioRun.Spec.TargetClusters {
		if scenarioRun.Spec.CancelRequested {
			continue
		}
		for _, clusterName := range clusterNames {
			jobCancelled := false
			for _, job := range scenarioRun.Status.ClusterJobs {
				if job.ClusterName == clusterName && job.CancelRequested {
					jobCancelled = true
					break
				}
			}
			if jobCancelled {
				continue
			}
			if r.jobExistsForCluster(&scenarioRun, clusterName) {
				logger.V(1).Info("job already exists for cluster, skipping",
					"provider", providerName,
					"cluster", clusterName,
					"scenarioRun", scenarioRun.Name)
				continue
			}
			existingJobIndex := -1
			for i, job := range scenarioRun.Status.ClusterJobs {
				if job.ClusterName == clusterName && job.Phase == "Retrying" {
					existingJobIndex = i
					break
				}
			}
			clusterAPIURL := r.getClusterAPIURL(ctx, &scenarioRun, providerName, clusterName)
			if scenarioRun.Spec.RollbackSource != nil {
				clusterAPIURL = scenarioRun.Spec.RollbackSource.ClusterAPIURL
			}
			targets = append(targets, clusterTarget{
				providerName:     providerName,
				clusterName:      clusterName,
				clusterAPIURL:    clusterAPIURL,
				existingJobIndex: existingJobIndex,
				statusIndex:      -1,
			})
			logger.Info("creating job for cluster",
				"provider", providerName,
				"cluster", clusterName,
				"scenarioRun", scenarioRun.Name)
		}
	}

	// Reserve each target in status before creating any Kubernetes resources. Pod and
	// ConfigMap watches can enqueue another reconcile before resource creation returns;
	// the reservation makes that reconcile observe the target as already claimed.
	if len(targets) > 0 {
		now := metav1.Now()
		for i := range targets {
			targets[i].jobID = uuid.New().String()
			reserved := krknv1alpha1.ClusterJobStatus{
				ProviderName:         targets[i].providerName,
				ClusterName:          targets[i].clusterName,
				ClusterAPIURL:        targets[i].clusterAPIURL,
				JobID:                targets[i].jobID,
				Phase:                "Creating",
				StartTime:            &now,
				MaxRetries:           effectiveScenarioMaxRetries(&scenarioRun),
				MaxRetriesConfigured: true,
			}
			if targets[i].existingJobIndex >= 0 {
				targets[i].statusIndex = targets[i].existingJobIndex
				previous := scenarioRun.Status.ClusterJobs[targets[i].statusIndex]
				reserved.AttemptHistory = previous.AttemptHistory
				reserved.RetryCount = previous.RetryCount
				reserved.MaxRetries = previous.MaxRetries
				reserved.MaxRetriesConfigured = previous.MaxRetriesConfigured
				reserved.LastRetryTime = previous.LastRetryTime
				if scenarioRun.Spec.RollbackEnabled {
					reserved.MaxRetries = 0
					reserved.MaxRetriesConfigured = true
				}
				scenarioRun.Status.ClusterJobs[targets[i].statusIndex] = reserved
			} else {
				scenarioRun.Status.ClusterJobs = append(scenarioRun.Status.ClusterJobs, reserved)
				targets[i].statusIndex = len(scenarioRun.Status.ClusterJobs) - 1
			}
		}
		if err := r.Status().Update(ctx, &scenarioRun); err != nil {
			if apierrors.IsConflict(err) {
				logger.Info("conflict reserving cluster jobs, will retry on next reconcile")
				return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, nil
			}
			logger.Error(err, "failed to reserve cluster jobs")
			return ctrl.Result{}, err
		}
	}

	// Prepare resources and create pods in parallel — each goroutine only creates
	// Kubernetes resources (ConfigMaps, Secrets, Pod); no shared state is mutated.
	// Status writes happen serially below after g.Wait().
	outcomes := make([]jobOutcome, len(targets))
	g, gCtx := errgroup.WithContext(ctx)
	for i, target := range targets {
		i, target := i, target
		g.Go(func() error {
			resources, err := r.prepareJobResources(gCtx, &scenarioRun, target.providerName, target.clusterName, target.clusterAPIURL, target.jobID)
			if err != nil {
				outcomes[i] = jobOutcome{target: target, err: err}
				return nil
			}
			podName, err := r.submitScenarioPod(gCtx, &scenarioRun, resources)
			if err != nil {
				outcomes[i] = jobOutcome{target: target, err: err}
				return nil
			}
			outcomes[i] = jobOutcome{
				target:  target,
				jobID:   resources.jobID,
				podName: podName,
				image:   resources.containerImage,
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return ctrl.Result{}, err
	}

	// Apply outcomes serially to avoid concurrent slice mutations.
	// Track the failure in status so jobExistsForCluster returns true on
	// subsequent reconciles and we don't retry on every reconcile cycle.
	// Without this, partial resource creates (ConfigMaps) followed by cleanup
	// deletes drive an infinite reconcile loop via the Owns() watch.
	//
	// ClusterAPIURL must be populated even on failure: the API layer uses it to
	// authorize and filter jobs for non-admin users. Without it, non-admins
	// cannot see the failed job (and the run may appear stuck in "not processed").
	jobsCreated := 0
	var signatureFailure error
	for _, outcome := range outcomes {
		if outcome.err != nil {
			if errors.Is(outcome.err, errScenarioRunCancelled) {
				scenarioRun.Spec.CancelRequested = true
				job := &scenarioRun.Status.ClusterJobs[outcome.target.statusIndex]
				job.JobID = ""
				job.PodName = ""
				job.CancelRequested = true
				job.Phase = "Cancelled"
				setCancellationCleanupOutcome(job, "not-required", "")
				r.setCompletionTime(job)
				continue
			}
			message := "Job creation failed"
			failureReason := "JobCreationFailed"
			var invalidSignatureErr *InvalidImageSignatureError
			if errors.As(outcome.err, &invalidSignatureErr) {
				signatureFailure = outcome.err
				message = outcome.err.Error()
				failureReason = "InvalidImageSignature"
				logger.Error(outcome.err, "scenario run failed because the scenario image has no valid signature",
					"scenarioRun", scenarioRun.Name,
					"image", invalidSignatureErr.Image)
			}
			if errors.Is(outcome.err, workspace.ErrUnavailable) || errors.Is(outcome.err, workspace.ErrReplaced) {
				failureReason = "WorkspaceUnavailable"
				message = outcome.err.Error()
			}
			logger.Error(outcome.err, "failed to create cluster job",
				"provider", outcome.target.providerName,
				"cluster", outcome.target.clusterName,
				"scenarioRun", scenarioRun.Name)
			now := metav1.Now()
			failedJob := krknv1alpha1.ClusterJobStatus{
				ProviderName:   outcome.target.providerName,
				ClusterName:    outcome.target.clusterName,
				ClusterAPIURL:  outcome.target.clusterAPIURL,
				JobID:          uuid.New().String(),
				Phase:          "Failed",
				Message:        message,
				FailureReason:  failureReason,
				StartTime:      &now,
				CompletionTime: &now,
			}
			job := &scenarioRun.Status.ClusterJobs[outcome.target.statusIndex]
			failedJob.AttemptHistory = job.AttemptHistory
			failedJob.RetryCount = job.RetryCount
			failedJob.MaxRetries = job.MaxRetries
			failedJob.MaxRetriesConfigured = job.MaxRetriesConfigured
			failedJob.LastRetryTime = job.LastRetryTime
			scenarioRun.Status.ClusterJobs[outcome.target.statusIndex] = failedJob
		} else {
			now := metav1.Now()
			idx := outcome.target.statusIndex
			job := &scenarioRun.Status.ClusterJobs[idx]
			job.JobID = outcome.jobID
			job.PodName = outcome.podName
			job.ScenarioImage = outcome.image
			job.ClusterAPIURL = outcome.target.clusterAPIURL
			job.Phase = "Pending"
			job.StartTime = &now
			job.CompletionTime = nil
			job.Message = ""
			job.FailureReason = ""
			logger.Info("updated cluster job in status",
				"cluster", outcome.target.clusterName,
				"newJobId", outcome.jobID,
				"retryAttempt", job.RetryCount)
			jobsCreated++
		}
	}

	if jobsCreated > 0 {
		logger.Info("jobs created in this reconcile loop",
			"count", jobsCreated,
			"scenarioRun", scenarioRun.Name)
	}

	logger.V(1).Info("updating cluster job statuses",
		"scenarioRun", scenarioRun.Name,
		"totalJobs", len(scenarioRun.Status.ClusterJobs))

	// Update status for all jobs
	if err := r.updateClusterJobStatuses(ctx, &scenarioRun); err != nil {
		logger.Error(err, "failed to update cluster job statuses")
		return ctrl.Result{}, err
	}

	// Calculate overall status
	r.calculateOverallStatus(&scenarioRun)
	if r.hasCancellationRequests(&scenarioRun) {
		r.setCancellationConditions(&scenarioRun)
	}
	if signatureFailure != nil {
		scenarioRun.Status.Phase = "Failed"
		scenarioRun.Status.Message = signatureFailure.Error()
	}

	// Calculate resiliency scores for standalone runs (not graph-run children)
	// once the run reaches a terminal phase. Per-cluster entries are persisted
	// with status "calculated" or "error"; the reconciler retries while any
	// cluster has "error" status. Best-effort: never blocks terminal phase
	// persistence.
	resiliencyRequeue := false
	if _, isGraphRun := scenarioRun.Labels["krkn.dev/graph-run"]; !isGraphRun {
		if scenarioRun.Spec.ResiliencyScoreEnabled && r.hasIncompleteResiliencyScores(&scenarioRun) {
			isTerminal := scenarioRun.Status.Phase == "Succeeded" ||
				scenarioRun.Status.Phase == "Failed" ||
				scenarioRun.Status.Phase == "PartiallyFailed"
			if isTerminal {
				if err := r.calculateResiliencyScores(ctx, &scenarioRun); err != nil {
					logger.Error(err, "failed to calculate resiliency scores, will retry")
					resiliencyRequeue = true
				}
			}
		}
	}

	logger.Info("reconcile loop completed",
		"scenarioRun", scenarioRun.Name,
		"phase", scenarioRun.Status.Phase,
		"totalTargets", scenarioRun.Status.TotalTargets,
		"successfulJobs", scenarioRun.Status.SuccessfulJobs,
		"failedJobs", scenarioRun.Status.FailedJobs,
		"runningJobs", scenarioRun.Status.RunningJobs)

	// Update status only if it has changed
	statusChanged := !r.statusEqual(originalStatus, &scenarioRun.Status)

	logger.V(1).Info("status comparison result",
		"scenarioRun", scenarioRun.Name,
		"statusChanged", statusChanged,
		"oldPhase", originalStatus.Phase,
		"newPhase", scenarioRun.Status.Phase,
		"oldRunning", originalStatus.RunningJobs,
		"newRunning", scenarioRun.Status.RunningJobs)

	if statusChanged {
		// Log what changed
		changes := r.detectStatusChanges(originalStatus, &scenarioRun.Status)
		logger.Info("status changed, updating CR",
			"scenarioRun", scenarioRun.Name,
			"changes", changes)

		if err := r.Status().Update(ctx, &scenarioRun); err != nil {
			// If it's a conflict error, just requeue - the object was modified concurrently
			// and will be re-fetched on next reconcile with fresh data
			if apierrors.IsConflict(err) {
				logger.Info("conflict on status update, will retry on next reconcile")
				return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, nil
			}
			logger.Error(err, "failed to update status")
			return ctrl.Result{}, err
		}
	} else {
		logger.V(1).Info("status unchanged, skipping update",
			"scenarioRun", scenarioRun.Name,
			"phase", scenarioRun.Status.Phase,
			"runningJobs", scenarioRun.Status.RunningJobs)
	}

	// Requeue if run is still active. Phase "Running" covers both running and pending
	// jobs (calculateOverallStatus sets Running when pendingJobs > 0).
	if scenarioRun.Status.Phase == "Running" {
		logger.V(1).Info("requeuing because run still active",
			"scenarioRun", scenarioRun.Name,
			"phase", scenarioRun.Status.Phase,
			"runningJobs", scenarioRun.Status.RunningJobs)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	if scenarioRun.Status.Phase == "Cancelling" ||
		(r.hasCancellationRequests(&scenarioRun) && r.cancellationCleanupPending(&scenarioRun)) {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Requeue to retry resiliency score collection after status has been persisted.
	if resiliencyRequeue {
		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

// validateWorkspacePVC checks the pinned claim at each transition that could
// start a scenario pod. A missing or replaced claim must never fall back to an
// unmounted scenario execution.
func (r *KrknScenarioRunReconciler) validateWorkspacePVC(ctx context.Context, ref *krknv1alpha1.WorkspaceReference) error {
	if ref == nil {
		return nil
	}
	return workspace.Validate(ctx, r.Client, workspace.Binding{Namespace: r.Namespace, Reference: *ref})
}

// prepareJobResources prepares all resources needed for creating a scenario pod.
// This includes ConfigMaps, Secrets, Volumes, EnvVars, and determining the container image.
// Returns preparedJobResources or an error if preparation fails.
func (r *KrknScenarioRunReconciler) prepareJobResources(
	ctx context.Context,
	scenarioRun *krknv1alpha1.KrknScenarioRun,
	providerName string,
	clusterName string,
	clusterAPIURL string,
	jobID string,
) (*preparedJobResources, error) {
	logger := log.FromContext(ctx)
	if cancelled, err := r.cancellationRequested(ctx, scenarioRun, clusterName); err != nil {
		return nil, err
	} else if cancelled {
		return nil, errScenarioRunCancelled
	}
	if err := r.validateAIChildRegistry(ctx, scenarioRun); err != nil {
		return nil, err
	}

	// jobID is reserved in KrknScenarioRun status before resource creation.
	if scenarioRun.Spec.RollbackEnabled && scenarioRun.Spec.Workspace == nil {
		return nil, fmt.Errorf("rollback requires a workspace")
	}
	if scenarioRun.Spec.RollbackSource != nil && scenarioRun.Spec.Workspace == nil {
		return nil, fmt.Errorf("rollback source requires a workspace")
	}
	var workspaceSubPath string
	var originalKubeconfig string
	if ref := scenarioRun.Spec.Workspace; ref != nil {
		if err := r.validateWorkspacePVC(ctx, ref); err != nil {
			return nil, err
		}
		var err error
		if source := scenarioRun.Spec.RollbackSource; source != nil {
			original := &krknv1alpha1.KrknScenarioRun{}
			if err := r.Get(ctx, types.NamespacedName{Namespace: r.Namespace, Name: source.RunName}, original); err != nil {
				return nil, fmt.Errorf("load rollback source run: %w", err)
			}
			if string(original.UID) != source.RunUID || !original.Spec.RollbackEnabled || original.Spec.Workspace == nil || *original.Spec.Workspace != *ref {
				return nil, fmt.Errorf("rollback source workspace identity changed")
			}
			found := false
			for _, targetJob := range original.Status.ClusterJobs {
				if targetJob.ProviderName != providerName || targetJob.ClusterName != clusterName {
					continue
				}
				if targetJob.ClusterAPIURL == "" || targetJob.ClusterAPIURL != source.ClusterAPIURL {
					return nil, fmt.Errorf("rollback source target identity changed")
				}
				if targetJob.JobID == source.JobID {
					found = true
				}
				for _, attempt := range targetJob.AttemptHistory {
					if attempt.JobID == source.JobID {
						found = true
					}
				}
			}
			if !found || scenarioRun.Spec.Scenario.Name != "workload-rollback" {
				return nil, fmt.Errorf("rollback source attempt is invalid")
			}
			originalKubeconfig, err = kubeconfig.LoadOriginalAttempt(ctx, r.Client, r.Namespace, original, source.JobID, clusterName)
			if err != nil {
				return nil, err
			}
			workspaceSubPath, err = workspace.SubPath(original.UID, providerName, clusterName, source.JobID)
		} else {
			workspaceSubPath, err = workspace.SubPath(scenarioRun.UID, providerName, clusterName, jobID)
		}
		if err != nil {
			return nil, fmt.Errorf("prepare workspace path: %w", err)
		}
	}

	// Load the embedded krknctl defaults for the scenario pod.
	krknctlCfg, err := krknctlconfig.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load krknctl config: %w", err)
	}
	verifySignatures, err := imageSignatureVerificationEnabled(ctx, r.Client, r.Namespace)
	if err != nil {
		return nil, err
	}

	// Set default kubeconfig path if not provided
	kubeconfigPath := scenarioRun.Spec.KubeconfigPath
	if kubeconfigPath == "" {
		kubeconfigPath = krknctlCfg.KubeconfigPath
	}
	if workspaceSubPath != "" && workspace.ConflictsWithMount(kubeconfigPath) {
		return nil, fmt.Errorf("kubeconfig mount %q overlaps workspace mount", kubeconfigPath)
	}

	logger.Info("getting kubeconfig for cluster",
		"provider", providerName,
		"cluster", clusterName,
		"targetRequestId", scenarioRun.Spec.TargetRequestID)

	var kubeconfigDecoded []byte
	if originalKubeconfig != "" {
		kubeconfigDecoded = []byte(originalKubeconfig)
	} else {
		// Get kubeconfig from managed-clusters Secret (works for ALL providers)
		kubeconfigBase64, err := r.getKubeconfigFromProvider(ctx, scenarioRun.Spec.TargetRequestID, providerName, clusterName)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubeconfig from provider %s: %w", providerName, err)
		}
		// Decode kubeconfig for ConfigMap
		kubeconfigDecoded, err = base64.StdEncoding.DecodeString(kubeconfigBase64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode kubeconfig: %w", err)
		}
	}

	if clusterAPIURL == "" {
		logger.Error(nil, "ClusterAPIURL not found for cluster - job will not be visible to non-admins",
			"providerName", providerName,
			"clusterName", clusterName,
			"targetRequestId", scenarioRun.Spec.TargetRequestID)
	} else {
		logger.V(1).Info("Extracted ClusterAPIURL for job",
			"clusterName", clusterName,
			"clusterAPIURL", clusterAPIURL)
	}

	// Create ConfigMap for kubeconfig
	kubeconfigConfigMapName := kubeconfig.AttemptConfigMapName(jobID)
	kubeconfigLabels := map[string]string{
		"krkn-job-id":         jobID,
		"krkn-scenario-run":   scenarioRun.Name,
		"krkn-scenario-name":  scenarioRun.Spec.Scenario.Name,
		"krkn-cluster-name":   clusterName,
		"krkn-target-request": scenarioRun.Spec.TargetRequestID,
	}
	if ownerLabel := getOwnerLabel(scenarioRun); ownerLabel != "" {
		kubeconfigLabels["krkn.krkn-chaos.dev/owner-user"] = ownerLabel
	}
	kubeconfigConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubeconfigConfigMapName,
			Namespace: r.Namespace,
			Labels:    kubeconfigLabels,
		},
		Data: map[string]string{
			"config": string(kubeconfigDecoded),
		},
	}

	// Set owner reference for automatic cleanup
	if err := controllerutil.SetControllerReference(scenarioRun, kubeconfigConfigMap, r.Scheme); err != nil {
		return nil, fmt.Errorf("failed to set owner reference on kubeconfig ConfigMap: %w", err)
	}

	if cancelled, err := r.cancellationRequested(ctx, scenarioRun, clusterName); err != nil {
		return nil, err
	} else if cancelled {
		return nil, errScenarioRunCancelled
	}
	if err := r.Create(ctx, kubeconfigConfigMap); err != nil {
		return nil, fmt.Errorf("failed to create kubeconfig ConfigMap: %w", err)
	}

	// Track created resources for cleanup on error
	createdConfigMaps := []string{kubeconfigConfigMapName}
	var createdSecrets []string

	// Cleanup helper (used only in this function on error)
	cleanup := func() {
		for _, cmName := range createdConfigMaps {
			_ = r.Delete(ctx, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cmName,
					Namespace: r.Namespace,
				},
			})
		}
		for _, secretName := range createdSecrets {
			_ = r.Delete(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: r.Namespace,
				},
			})
		}
	}

	// Create ConfigMaps for user-provided files
	for _, file := range scenarioRun.Spec.Files {
		// Sanitize filename for ConfigMap name
		sanitizedName := strings.ReplaceAll(file.Name, "/", "-")
		sanitizedName = strings.ReplaceAll(sanitizedName, ".", "-")
		configMapName := fmt.Sprintf("krkn-job-%s-file-%s", jobID, sanitizedName)

		// Decode base64 content
		fileContent, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("failed to decode file content for '%s': %w", file.Name, err)
		}

		fileLabels := map[string]string{
			"krkn-job-id":         jobID,
			"krkn-scenario-run":   scenarioRun.Name,
			"krkn-scenario-name":  scenarioRun.Spec.Scenario.Name,
			"krkn-cluster-name":   clusterName,
			"krkn-target-request": scenarioRun.Spec.TargetRequestID,
		}
		if ownerLabel := getOwnerLabel(scenarioRun); ownerLabel != "" {
			fileLabels["krkn.krkn-chaos.dev/owner-user"] = ownerLabel
		}
		fileConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: r.Namespace,
				Labels:    fileLabels,
			},
			Data: map[string]string{
				file.Name: string(fileContent),
			},
		}

		// Set owner reference
		if err := controllerutil.SetControllerReference(scenarioRun, fileConfigMap, r.Scheme); err != nil {
			cleanup()
			return nil, fmt.Errorf("failed to set owner reference on file ConfigMap: %w", err)
		}

		if cancelled, err := r.cancellationRequested(ctx, scenarioRun, clusterName); err != nil {
			cleanup()
			return nil, err
		} else if cancelled {
			cleanup()
			return nil, errScenarioRunCancelled
		}
		if err := r.Create(ctx, fileConfigMap); err != nil {
			cleanup()
			return nil, fmt.Errorf("failed to create file ConfigMap: %w", err)
		}

		createdConfigMaps = append(createdConfigMaps, configMapName)
	}

	// Resolve the image exclusively from the scenario reference. Graph runs use
	// this same path; no caller-provided image is trusted.
	var privateRegistry *krknctlmodels.RegistryV2
	if scenarioRun.Spec.Scenario.Private == nil {
		cleanup()
		return nil, fmt.Errorf("scenario.private is required")
	}
	privateRegistry, err = resolvePrivateRegistry(ctx, r.Client, scenarioRun.Namespace, scenarioRun.Spec.Scenario)
	if err != nil {
		cleanup()
		return nil, err
	}
	resolver := r.imageResolver
	if resolver == nil {
		resolver = resolveImmutableImage
	}
	containerImage, err := resolver(ctx, &krknctlCfg, scenarioRun.Spec.Scenario, privateRegistry)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to resolve scenario image: %w", err)
	}
	verifier := r.signatureVerifier
	if verifier == nil {
		verifier = resolveImageSignature
	}
	signatureStatus, signatureErr := verifier(ctx, &krknctlCfg, scenarioRun.Spec.Scenario, privateRegistry, containerImage)
	if !verifySignatures {
		if signatureErr != nil || signatureStatus != verify.SignatureSigned {
			logger.Info("WARNING: image signature verification result skipped; executing unverified image",
				"scenarioRun", scenarioRun.Name,
				"image", containerImage,
				"verificationStatus", signatureStatus,
				"verificationError", signatureErr)
		}
	} else if signatureErr != nil {
		logger.Error(signatureErr, "failed to verify scenario image signature; refusing to create pod",
			"scenarioRun", scenarioRun.Name,
			"image", containerImage)
		cleanup()
		return nil, signatureErr
	} else if signatureStatus != verify.SignatureSigned {
		err := &InvalidImageSignatureError{Image: containerImage, Status: signatureStatus}
		logger.Error(err, "scenario image signature is invalid; refusing to create pod",
			"scenarioRun", scenarioRun.Name,
			"image", containerImage)
		cleanup()
		return nil, err
	}
	logger.V(1).Info("resolved scenario image through krknctl", "image", containerImage)

	// Handle private registry authentication
	var imagePullSecrets []corev1.LocalObjectReference

	if scenarioRun.Spec.Scenario.Private != nil && *scenarioRun.Spec.Scenario.Private {
		// Use saved private registry
		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{
			Name: scenarioRun.Spec.Scenario.RegistryName,
		})
	}

	// Build volumes and volume mounts
	volumes := []corev1.Volume{
		{
			Name: "kubeconfig",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: kubeconfigConfigMapName,
					},
				},
			},
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "kubeconfig",
			MountPath: kubeconfigPath,
			SubPath:   "config",
		},
	}

	// Add file mounts
	// Skip first ConfigMap (kubeconfig), file ConfigMaps start from index 1
	fileConfigMaps := createdConfigMaps[1:]
	for i, file := range scenarioRun.Spec.Files {
		if workspaceSubPath != "" && workspace.ConflictsWithMount(file.MountPath) {
			cleanup()
			return nil, fmt.Errorf("file mount %q overlaps workspace mount", file.MountPath)
		}
		volumeName := fmt.Sprintf("file-%d", i)

		volumes = append(volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fileConfigMaps[i],
					},
				},
			},
		})

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: file.MountPath,
			SubPath:   file.Name,
		})
	}
	if workspaceSubPath != "" {
		volume, mount, err := workspace.Mount(workspace.Binding{Namespace: r.Namespace, Reference: *scenarioRun.Spec.Workspace},
			workspace.MountOptions{VolumeName: "workspace", MountPath: workspace.MountPath, SubPath: workspaceSubPath})
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("prepare workspace mount: %w", err)
		}
		volumes = append(volumes, volume)
		volumeMounts = append(volumeMounts, mount)
	}

	// Add writable tmp volume
	volumes = append(volumes, corev1.Volume{
		Name: "tmp",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "tmp",
		MountPath: "/tmp",
	})

	// Convert environment map to EnvVar slice
	envVars := make([]corev1.EnvVar, 0, len(scenarioRun.Spec.Environment))
	for key, value := range scenarioRun.Spec.Environment {
		if scenarioRun.Spec.RollbackEnabled && (key == "ROLLBACK_VERSIONS_DIRECTORY" || key == "UUID") {
			continue
		}
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}
	if scenarioRun.Spec.RollbackEnabled {
		envVars = append(envVars, corev1.EnvVar{
			Name: "ROLLBACK_VERSIONS_DIRECTORY", Value: workspace.MountPath,
		})
		envVars = append(envVars, corev1.EnvVar{Name: "UUID", Value: jobID})
	}

	// Inject RESILIENCY_SCORE env var for standalone runs (not graph-run children).
	// Strip any user-provided value first to prevent conflicts, matching graph-run behavior.
	if _, isGraphRun := scenarioRun.Labels["krkn.dev/graph-run"]; !isGraphRun && scenarioRun.Spec.ResiliencyScoreEnabled {
		filtered := envVars[:0]
		for _, ev := range envVars {
			if ev.Name != "RESILIENCY_SCORE" {
				filtered = append(filtered, ev)
			}
		}
		envVars = append(filtered, corev1.EnvVar{
			Name:  "RESILIENCY_SCORE",
			Value: "true",
		})
	}

	// Inject cloud credentials via SecretKeyRef if specified
	envVars, volumes, volumeMounts, injErr := r.appendCloudCredentialInjection(
		ctx, scenarioRun.Spec.CloudCredentialRef, envVars, volumes, volumeMounts,
	)
	if injErr != nil {
		return nil, injErr
	}

	return &preparedJobResources{
		jobID:             jobID,
		clusterName:       clusterName,
		clusterAPIURL:     clusterAPIURL,
		containerImage:    containerImage,
		kubeconfigPath:    kubeconfigPath,
		volumes:           volumes,
		volumeMounts:      volumeMounts,
		envVars:           envVars,
		imagePullSecrets:  imagePullSecrets,
		createdConfigMaps: createdConfigMaps,
		createdSecrets:    createdSecrets,
	}, nil
}

// appendCloudCredentialInjection loads the named cloud-credential Secret and
// appends SecretKeyRef env vars (plus any volumes/mounts) to the job resources.
// When cloudCredentialRef is empty, inputs are returned unchanged.
func (r *KrknScenarioRunReconciler) appendCloudCredentialInjection(
	ctx context.Context,
	cloudCredentialRef string,
	envVars []corev1.EnvVar,
	volumes []corev1.Volume,
	volumeMounts []corev1.VolumeMount,
) ([]corev1.EnvVar, []corev1.Volume, []corev1.VolumeMount, error) {
	if cloudCredentialRef == "" {
		return envVars, volumes, volumeMounts, nil
	}

	var credSecret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Name:      cloudCredentialRef,
		Namespace: r.Namespace,
	}, &credSecret); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load cloud credential '%s': %w", cloudCredentialRef, err)
	}

	credEnvVars, credVolumes, credMounts, err := cloudcreds.InjectCredentials(&credSecret)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to inject cloud credentials: %w", err)
	}

	envVars = append(envVars, credEnvVars...)
	volumes = append(volumes, credVolumes...)
	volumeMounts = append(volumeMounts, credMounts...)
	return envVars, volumes, volumeMounts, nil
}

// cleanupPreparedResources deletes ConfigMaps and Secrets created during resource preparation.
// This is called when executeScenarioPod fails to prevent resource leaks.
func (r *KrknScenarioRunReconciler) cleanupPreparedResources(ctx context.Context, resources *preparedJobResources) {
	logger := log.FromContext(ctx)

	for _, cmName := range resources.createdConfigMaps {
		if err := r.Delete(ctx, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cmName,
				Namespace: r.Namespace,
			},
		}); err != nil {
			if !apierrors.IsNotFound(err) {
				logger.Error(err, "failed to delete ConfigMap during cleanup",
					"configMapName", cmName, "jobID", resources.jobID)
			}
		}
	}

	for _, secretName := range resources.createdSecrets {
		if err := r.Delete(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: r.Namespace,
			},
		}); err != nil {
			if !apierrors.IsNotFound(err) {
				logger.Error(err, "failed to delete Secret during cleanup",
					"secretName", secretName, "jobID", resources.jobID)
			}
		}
	}
}

// submitScenarioPod builds and creates the scenario Pod from prepared resources.
// On failure it cleans up all prepared ConfigMaps and Secrets and returns an error.
// Status updates are the caller's responsibility.
func (r *KrknScenarioRunReconciler) submitScenarioPod(
	ctx context.Context,
	scenarioRun *krknv1alpha1.KrknScenarioRun,
	resources *preparedJobResources,
) (string, error) {
	if cancelled, err := r.cancellationRequested(ctx, scenarioRun, resources.clusterName); err != nil {
		r.cleanupPreparedResources(ctx, resources)
		return "", err
	} else if cancelled {
		r.cleanupPreparedResources(ctx, resources)
		return "", errScenarioRunCancelled
	}
	var runAsUser int64 = 1001
	var runAsGroup int64 = 1001
	var fsGroup int64 = 1001

	podName := fmt.Sprintf("krkn-job-%s", resources.jobID)
	podLabels := map[string]string{
		"app":                 "krkn-scenario",
		"krkn-job-id":         resources.jobID,
		"krkn-scenario-run":   scenarioRun.Name,
		"krkn-scenario-name":  scenarioRun.Spec.Scenario.Name,
		"krkn-cluster-name":   resources.clusterName,
		"krkn-target-request": scenarioRun.Spec.TargetRequestID,
	}
	if ownerLabel := getOwnerLabel(scenarioRun); ownerLabel != "" {
		podLabels["krkn.krkn-chaos.dev/owner-user"] = ownerLabel
	}
	for _, key := range []string{
		"krkn.dev/ai-run",
		"krkn.dev/orchestrator-pod",
		"krkn.dev/scenario-id",
		"krkn.dev/generation-id",
		"krkn.dev/scenario-name",
	} {
		if value := scenarioRun.Labels[key]; value != "" {
			podLabels[key] = value
		}
	}
	podLabels["krkn.dev/component"] = "scenario"
	scenarioCommand := "if [ -x /home/krkn/kraken/containers/setup-ssh.sh ]; then /home/krkn/kraken/containers/setup-ssh.sh || exit $?; fi; " +
		"set -m; " +
		"/home/krkn/run.sh & scenario_pid=$!; " +
		"( while kill -0 -- \"-$scenario_pid\" 2>/dev/null; do " +
		"if [ \"$(cat " + scenarioCancellationPath + " 2>/dev/null)\" = \"true\" ]; then " +
		"kill -TERM -- \"-$scenario_pid\" 2>/dev/null || kill -TERM \"$scenario_pid\" 2>/dev/null; " +
		"grace=120; while [ \"$grace\" -gt 0 ] && kill -0 -- \"-$scenario_pid\" 2>/dev/null; do sleep 1; grace=$((grace - 1)); done; " +
		"if kill -0 -- \"-$scenario_pid\" 2>/dev/null; then echo 'CANCELLATION_FORCE_STOPPED grace_seconds=120'; kill -KILL -- \"-$scenario_pid\" 2>/dev/null; fi; exit 0; fi; sleep 1; done ) & " +
		"watcher_pid=$!; wait \"$scenario_pid\"; krkn_exit=$?; " +
		"wait \"$watcher_pid\" 2>/dev/null; " +
		"if [ -f /home/krkn/kraken/kraken.report.html ]; then echo '===KRKN_REPORT_HTML_START==='; base64 /home/krkn/kraken/kraken.report.html; echo '===KRKN_REPORT_HTML_END==='; fi; " +
		"if [ -f /home/krkn/kraken/kraken.report.pdf ]; then echo '===KRKN_REPORT_PDF_START==='; base64 /home/krkn/kraken/kraken.report.pdf; echo '===KRKN_REPORT_PDF_END==='; fi; " +
		"exit $krkn_exit"
	if scenarioRun.Spec.RollbackEnabled {
		scenarioCommand = rollbackWorkspaceWriteCheck + scenarioCommand
		if scenarioRun.Spec.Scenario.Private != nil && !*scenarioRun.Spec.Scenario.Private {
			scenarioCommand = rollbackConfigTemplateCheck("/home/krkn/kraken/config/config.yaml.template") + scenarioCommand
		}
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: r.Namespace,
			Labels:    podLabels,
			Annotations: map[string]string{
				scenarioCancellationAnnotation: "false",
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "krkn-operator-krkn-scenario-runner",
			RestartPolicy:      corev1.RestartPolicyNever,
			ImagePullSecrets:   resources.imagePullSecrets,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser:  &runAsUser,
				RunAsGroup: &runAsGroup,
				FSGroup:    &fsGroup,
			},
			Containers: []corev1.Container{
				{
					Name:    "scenario",
					Image:   resources.containerImage,
					Command: []string{"/bin/bash", "-c"},
					Args:    []string{scenarioCommand},
					Env:     resources.envVars,
					VolumeMounts: append(append([]corev1.VolumeMount(nil), resources.volumeMounts...), corev1.VolumeMount{
						Name: "cancellation", MountPath: "/var/run/krkn-cancellation", ReadOnly: true,
					}),
					ImagePullPolicy: corev1.PullAlways,
				},
			},
			Volumes: append(append([]corev1.Volume(nil), resources.volumes...), corev1.Volume{
				Name: "cancellation",
				VolumeSource: corev1.VolumeSource{DownwardAPI: &corev1.DownwardAPIVolumeSource{
					Items: []corev1.DownwardAPIVolumeFile{{
						Path: "requested",
						FieldRef: &corev1.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "metadata.annotations['" + scenarioCancellationAnnotation + "']",
						},
					}},
				}},
			}),
		},
	}
	if err := controllerutil.SetControllerReference(scenarioRun, pod, r.Scheme); err != nil {
		r.cleanupPreparedResources(ctx, resources)
		return "", fmt.Errorf("failed to set owner reference on pod: %w", err)
	}
	if ref := scenarioRun.Spec.Workspace; ref != nil {
		if err := r.validateWorkspacePVC(ctx, ref); err != nil {
			r.cleanupPreparedResources(ctx, resources)
			return "", err
		}
	}

	if cancelled, err := r.cancellationRequested(ctx, scenarioRun, resources.clusterName); err != nil {
		r.cleanupPreparedResources(ctx, resources)
		return "", err
	} else if cancelled {
		r.cleanupPreparedResources(ctx, resources)
		return "", errScenarioRunCancelled
	}
	if err := r.Create(ctx, pod); err != nil {
		r.cleanupPreparedResources(ctx, resources)
		return "", fmt.Errorf("failed to create pod: %w", err)
	}

	return podName, nil
}

// createClusterJob creates all resources needed for a single cluster scenario job.
// This method orchestrates the job creation by preparing resources and executing the pod.
func (r *KrknScenarioRunReconciler) createClusterJob(
	ctx context.Context,
	scenarioRun *krknv1alpha1.KrknScenarioRun,
	providerName string,
	clusterName string,
) error {
	if cancelled, err := r.cancellationRequested(ctx, scenarioRun, clusterName); err != nil {
		return err
	} else if cancelled {
		return errScenarioRunCancelled
	}
	// Check if this is a retry case
	existingJobIndex := -1
	for i, job := range scenarioRun.Status.ClusterJobs {
		if job.ClusterName == clusterName && job.Phase == "Retrying" {
			existingJobIndex = i
			break
		}
	}

	// Reserve the retry before creating resources so a second reconcile cannot
	// submit another pod for the same cluster.
	jobID := uuid.New().String()
	now := metav1.Now()
	if existingJobIndex < 0 {
		scenarioRun.Status.ClusterJobs = append(scenarioRun.Status.ClusterJobs, krknv1alpha1.ClusterJobStatus{
			ProviderName:         providerName,
			ClusterName:          clusterName,
			ClusterAPIURL:        r.getClusterAPIURL(ctx, scenarioRun, providerName, clusterName),
			JobID:                jobID,
			Phase:                "Creating",
			StartTime:            &now,
			MaxRetries:           scenarioRun.Spec.MaxRetries,
			MaxRetriesConfigured: true,
		})
		existingJobIndex = len(scenarioRun.Status.ClusterJobs) - 1
	} else {
		job := &scenarioRun.Status.ClusterJobs[existingJobIndex]
		job.JobID = jobID
		job.PodName = ""
		job.Phase = "Creating"
		job.StartTime = &now
		job.CompletionTime = nil
		job.Message = ""
		job.FailureReason = ""
	}
	if err := r.Status().Update(ctx, scenarioRun); err != nil {
		return err
	}

	// Step 1: Prepare all resources (ConfigMaps, Secrets, Volumes, EnvVars, container image)
	clusterAPIURL := r.getClusterAPIURL(ctx, scenarioRun, providerName, clusterName)
	resources, err := r.prepareJobResources(ctx, scenarioRun, providerName, clusterName, clusterAPIURL, jobID)
	if err != nil {
		return err
	}

	// Step 2: Create the pod
	logger := log.FromContext(ctx)
	podName, err := r.submitScenarioPod(ctx, scenarioRun, resources)
	if err != nil {
		return err
	}

	// Step 3: Complete the reserved status entry.
	job := &scenarioRun.Status.ClusterJobs[existingJobIndex]
	job.JobID = resources.jobID
	job.PodName = podName
	job.ScenarioImage = resources.containerImage
	job.Phase = "Pending"
	job.StartTime = &now
	job.CompletionTime = nil
	job.Message = ""
	logger.Info("updated retry job in status",
		"cluster", resources.clusterName,
		"newJobId", resources.jobID,
		"retryAttempt", job.RetryCount)
	return nil
}

func (r *KrknScenarioRunReconciler) requestPodCancellation(ctx context.Context, pod *corev1.Pod) error {
	if pod.Annotations[scenarioCancellationAnnotation] == "true" {
		return nil
	}
	before := pod.DeepCopy()
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	pod.Annotations[scenarioCancellationAnnotation] = "true"
	return r.Patch(ctx, pod, client.MergeFrom(before))
}

func cancellationCleanupReceipt(logs []byte) (outcome, detail string, found bool) {
	lines := strings.Split(string(logs), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "CANCELLATION_FORCE_STOPPED ") {
			return "unknown", line, true
		}
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "CANCELLATION_CLEANUP ") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "CANCELLATION_CLEANUP "))
		for _, field := range fields {
			if !strings.HasPrefix(field, "outcome=") {
				continue
			}
			outcome = strings.TrimPrefix(field, "outcome=")
			if outcome != "complete" && outcome != "failed" && outcome != "unknown" {
				return "unknown", "", true
			}
			for j, detailField := range fields {
				if strings.HasPrefix(detailField, "detail=") {
					detail = strings.TrimSpace(strings.TrimPrefix(strings.Join(fields[j:], " "), "detail="))
					break
				}
			}
			return outcome, detail, true
		}
		return "unknown", "", true
	}
	return "", "", false
}

func setCancellationCleanupOutcome(job *krknv1alpha1.ClusterJobStatus, outcome, detail string) {
	switch outcome {
	case "complete":
		job.FailureReason = "CancellationCleanupComplete"
		if detail == "" {
			detail = "Scenario cleanup completed after cancellation."
		}
	case "failed":
		job.FailureReason = "CancellationCleanupFailed"
		if detail == "" {
			detail = "Scenario cleanup reported failure after cancellation."
		}
	case "not-required":
		job.FailureReason = "CancellationCleanupNotRequired"
		if detail == "" {
			detail = "Scenario execution had not started."
		}
	default:
		job.FailureReason = "CancellationCleanupUnknown"
		if detail == "" {
			detail = "No definitive scenario cleanup receipt was observed."
		}
	}
	job.Message = detail
}

func (r *KrknScenarioRunReconciler) reconcileCancelledJob(
	ctx context.Context,
	scenarioRun *krknv1alpha1.KrknScenarioRun,
	job *krknv1alpha1.ClusterJobStatus,
) error {
	if job.Phase == "Succeeded" || job.Phase == "MaxRetriesExceeded" {
		return nil
	}
	if job.PodName == "" && job.JobID != "" {
		var pods corev1.PodList
		if err := r.List(ctx, &pods,
			client.InNamespace(r.Namespace),
			client.MatchingLabels{"krkn-job-id": job.JobID},
		); err != nil {
			return fmt.Errorf("find scenario Pod for cancelled job %q: %w", job.JobID, err)
		}
		ownedUIDs := map[types.UID]struct{}{scenarioRun.UID: {}}
		for i := range pods.Items {
			pod := &pods.Items[i]
			if hasScenarioRunOwner(pod.OwnerReferences, ownedUIDs) {
				job.PodName = pod.Name
				break
			}
		}
	}
	if job.PodName == "" {
		job.Phase = "Cancelled"
		if job.JobID == "" {
			setCancellationCleanupOutcome(job, "not-required", "No scenario job or Pod was reserved.")
		} else {
			setCancellationCleanupOutcome(job, "unknown", "No owned scenario Pod was found for the reserved job; container startup cannot be ruled out.")
		}
		r.setCompletionTime(job)
		return nil
	}
	var pod corev1.Pod
	err := r.Get(ctx, types.NamespacedName{Name: job.PodName, Namespace: r.Namespace}, &pod)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		job.Phase = "Cancelled"
		setCancellationCleanupOutcome(job, "unknown", "Scenario Pod disappeared before its cleanup receipt could be observed.")
		r.setCompletionTime(job)
		return nil
	}
	ownedUIDs := map[types.UID]struct{}{scenarioRun.UID: {}}
	if !hasScenarioRunOwner(pod.OwnerReferences, ownedUIDs) {
		job.Phase = "Cancelling"
		setCancellationCleanupOutcome(job, "unknown", "Refusing to observe or delete a same-name Pod with a different scenario-run owner UID.")
		return nil
	}
	if pod.Status.Phase == corev1.PodPending && !podHasStartedContainer(&pod) {
		if pod.UID == "" {
			job.Phase = "Cancelling"
			job.Message = "Cannot safely delete the never-started scenario Pod because its UID is unavailable."
			return nil
		}
		uid := pod.UID
		if err := r.Delete(ctx, &pod, client.Preconditions{UID: &uid}); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete never-started scenario Pod %q: %w", pod.Name, err)
		}
		job.Phase = "Cancelled"
		setCancellationCleanupOutcome(job, "not-required", "The scenario container never started; target cleanup was not required.")
		r.setCompletionTime(job)
		return nil
	}
	if pod.Status.Phase != corev1.PodFailed && pod.Status.Phase != corev1.PodSucceeded {
		if err := r.requestPodCancellation(ctx, &pod); err != nil {
			return fmt.Errorf("request graceful cancellation of scenario Pod %q: %w", pod.Name, err)
		}
		job.Phase = "Cancelling"
		job.Message = "Cancellation requested; waiting for the scenario process and cleanup receipt."
		return nil
	}

	job.Phase = "Cancelled"
	r.setCompletionTime(job)
	outcome, detail, receiptFound := "", "", false
	if r.Clientset != nil {
		logs, logErr := r.fetchScenarioPodLogs(ctx, job.PodName)
		if logErr == nil {
			outcome, detail, receiptFound = cancellationCleanupReceipt(logs)
		}
	}
	if !receiptFound {
		setCancellationCleanupOutcome(job, "unknown", detail)
	} else {
		setCancellationCleanupOutcome(job, outcome, detail)
	}
	if scenarioRun.Status.ReportStatus == nil && r.Clientset != nil {
		if err := r.extractAndStoreReports(ctx, scenarioRun, job); err != nil {
			return err
		}
	}
	return nil
}

// updateClusterJobStatuses updates the status of all cluster jobs by querying their pods
func (r *KrknScenarioRunReconciler) updateClusterJobStatuses(
	ctx context.Context,
	scenarioRun *krknv1alpha1.KrknScenarioRun,
) error {
	logger := log.FromContext(ctx)

	for i := range scenarioRun.Status.ClusterJobs {
		job := &scenarioRun.Status.ClusterJobs[i]
		if scenarioRun.Spec.RollbackEnabled {
			job.MaxRetries = 0
			job.MaxRetriesConfigured = true
		} else if scenarioRun.Labels["krkn.dev/ai-run"] != "" && !job.MaxRetriesConfigured {
			job.MaxRetries = scenarioRun.Spec.MaxRetries
			job.MaxRetriesConfigured = true
		} else {
			backfillLegacyMaxRetries(job, scenarioRun.Spec.MaxRetries)
		}
		if scenarioRun.Spec.CancelRequested {
			job.CancelRequested = true
		}
		if job.CancelRequested {
			if job.Phase == "Succeeded" || job.Phase == "MaxRetriesExceeded" ||
				(job.Phase == "Cancelled" && job.FailureReason != "CancellationCleanupUnknown") {
				continue
			}
			if err := r.reconcileCancelledJob(ctx, scenarioRun, job); err != nil {
				return err
			}
			continue
		}

		// A Creating entry is a durable reservation written before resource
		// creation. It has no pod yet, so leave it untouched until the creator
		// completes or the reservation timeout below expires.
		if job.Phase == "Creating" {
			if job.StartTime == nil || time.Since(job.StartTime.Time) < 30*time.Second {
				continue
			}
			job.Phase = "Failed"
			job.Message = "Job creation timed out"
			job.FailureReason = "JobCreationTimeout"
			now := metav1.Now()
			job.CompletionTime = &now
			continue
		}

		logger.V(1).Info("checking job status",
			"cluster", job.ClusterName,
			"jobID", job.JobID,
			"currentPhase", job.Phase,
			"podName", job.PodName)

		// Skip terminal jobs
		if job.Phase == "Succeeded" || job.Phase == "Cancelled" || job.Phase == "MaxRetriesExceeded" {
			// A report extraction error is returned from reconciliation without
			// setting ReportStatus, so a later reconciliation can retry it.
			if job.Phase == "Succeeded" && scenarioRun.Status.ReportStatus == nil {
				if err := r.extractAndStoreReports(ctx, scenarioRun, job); err != nil {
					return err
				}
			}
			logger.V(1).Info("skipping terminal job",
				"cluster", job.ClusterName,
				"jobID", job.JobID,
				"phase", job.Phase)
			continue
		}

		// Skip Failed jobs unless they need retry processing
		if job.Phase == "Failed" && job.RetryCount >= job.MaxRetries && !job.CancelRequested {
			if scenarioRun.Status.ReportStatus == nil && r.Clientset != nil && job.PodName != "" {
				if err := r.extractAndStoreReports(ctx, scenarioRun, job); err != nil {
					return err
				}
			}
			logger.V(1).Info("skipping failed job that exceeded retries",
				"cluster", job.ClusterName,
				"jobID", job.JobID,
				"retryCount", job.RetryCount,
				"maxRetries", job.MaxRetries)
			continue
		}

		// Fetch pod
		var pod corev1.Pod
		err := r.Get(ctx, types.NamespacedName{
			Name:      job.PodName,
			Namespace: r.Namespace,
		}, &pod)

		if err != nil {
			if apierrors.IsNotFound(err) {
				// IMPORTANT: Don't mark as Failed if pod was just created
				// Kubernetes might not have created the pod yet
				if job.Phase == "Pending" {
					// Calculate time since job start
					if job.StartTime != nil {
						timeSinceStart := time.Since(job.StartTime.Time)
						if timeSinceStart < 30*time.Second {
							// Pod not found but job is recent - this is normal, keep waiting
							logger.V(1).Info("pod not found but job is recent, keeping Pending status",
								"cluster", job.ClusterName,
								"jobID", job.JobID,
								"podName", job.PodName,
								"timeSinceStart", timeSinceStart.String())
							continue
						}
					}
				}

				// Pod genuinely not found - this is an error
				logger.Info("pod not found for job",
					"cluster", job.ClusterName,
					"jobID", job.JobID,
					"podName", job.PodName,
					"currentPhase", job.Phase)

				job.Phase = "Failed"
				job.Message = "Pod not found"
				job.FailureReason = "PodNotFound"
				now := metav1.Now()
				job.CompletionTime = &now
			} else {
				logger.Error(err, "error fetching pod",
					"cluster", job.ClusterName,
					"jobID", job.JobID,
					"podName", job.PodName)
			}
			continue
		}

		logger.V(1).Info("pod found",
			"cluster", job.ClusterName,
			"jobID", job.JobID,
			"podName", job.PodName,
			"podPhase", pod.Status.Phase)

		// Backfill container image if not set (for legacy runs)
		if job.ScenarioImage == "" && len(pod.Spec.Containers) > 0 {
			job.ScenarioImage = pod.Spec.Containers[0].Image
			logger.V(1).Info("backfilled scenario image from pod spec",
				"cluster", job.ClusterName,
				"jobID", job.JobID,
				"scenarioImage", job.ScenarioImage)
		}

		// Update job status based on pod phase
		previousPhase := job.Phase
		switch pod.Status.Phase {
		case corev1.PodPending:
			if err := r.validateWorkspacePVC(ctx, scenarioRun.Spec.Workspace); err != nil {
				if !errors.Is(err, workspace.ErrUnavailable) && !errors.Is(err, workspace.ErrReplaced) {
					return err
				}
				if deleteErr := r.Delete(ctx, &pod); deleteErr != nil && !apierrors.IsNotFound(deleteErr) {
					return fmt.Errorf("delete pending pod after workspace loss: %w", deleteErr)
				}
				job.Phase = "Failed"
				job.Message = err.Error()
				job.FailureReason = "WorkspaceUnavailable"
				job.RetryCount = job.MaxRetries
				r.setCompletionTime(job)
				continue
			}
			job.Phase = "Pending"
			if previousPhase != "Pending" {
				logger.Info("job phase transition",
					"cluster", job.ClusterName,
					"jobID", job.JobID,
					"from", previousPhase,
					"to", "Pending")
			}
		case corev1.PodRunning:
			job.Phase = "Running"
			if previousPhase != "Running" {
				logger.Info("job phase transition",
					"cluster", job.ClusterName,
					"jobID", job.JobID,
					"from", previousPhase,
					"to", "Running")
			}
		case corev1.PodSucceeded:
			job.Phase = "Succeeded"
			r.setCompletionTime(job)
			logger.Info("job succeeded",
				"cluster", job.ClusterName,
				"jobID", job.JobID,
				"duration", job.CompletionTime.Sub(job.StartTime.Time).String())
			if scenarioRun.Status.ReportStatus == nil {
				if err := r.extractAndStoreReports(ctx, scenarioRun, job); err != nil {
					return err
				}
			}
		case corev1.PodFailed:
			job.Phase = "Failed"
			job.Message = r.extractPodErrorMessage(&pod)
			job.FailureReason = r.extractFailureReason(&pod)
			r.setCompletionTime(job)

			// Retry logic
			logger.Info("pod failed, checking retry eligibility",
				"cluster", job.ClusterName,
				"jobID", job.JobID,
				"retryCount", job.RetryCount,
				"maxRetries", job.MaxRetries,
				"cancelRequested", job.CancelRequested,
				"failureReason", job.FailureReason)

			maxRetries := job.MaxRetries
			if maxRetries == 0 && job.MaxRetriesConfigured {
				job.Phase = "MaxRetriesExceeded"
				if scenarioRun.Labels["krkn.dev/ai-run"] != "" {
					logger.Info("Krkn-AI scenario retries disabled",
						"cluster", job.ClusterName,
						"jobID", job.JobID)
				} else {
					logger.Info("job retries disabled",
						"cluster", job.ClusterName,
						"jobID", job.JobID)
				}
				continue
			}

			if r.shouldRetryJob(job, maxRetries) {
				// Calculate backoff delay
				delay := r.calculateRetryDelay(job.RetryCount,
					scenarioRun.Spec.RetryBackoff,
					scenarioRun.Spec.RetryDelay)

				// Check if enough time has passed since last retry
				now := metav1.Now()
				if job.LastRetryTime != nil {
					elapsed := now.Sub(job.LastRetryTime.Time)
					if elapsed < delay {
						logger.Info("waiting for retry backoff",
							"cluster", job.ClusterName,
							"jobID", job.JobID,
							"elapsed", elapsed.String(),
							"requiredDelay", delay.String())
						// Don't retry yet, will check again on next reconcile
						continue
					}
				}

				// Retry!
				if job.JobID != "" {
					job.AttemptHistory = append(job.AttemptHistory, krknv1alpha1.ScenarioAttemptStatus{JobID: job.JobID, ScenarioImage: job.ScenarioImage})
				}
				job.Phase = "Retrying"
				job.RetryCount++
				job.LastRetryTime = &now

				logger.Info("retrying failed job",
					"cluster", job.ClusterName,
					"previousJobId", job.JobID,
					"retryAttempt", job.RetryCount,
					"maxRetries", maxRetries)

				// Validate required fields before retry
				if job.ProviderName == "" {
					logger.Error(nil, "cannot retry job: ProviderName is empty",
						"cluster", job.ClusterName,
						"jobID", job.JobID)
					job.Phase = "Failed"
					job.Message = "Retry failed: ProviderName is empty"
					job.FailureReason = "InvalidJobState"
					r.setCompletionTime(job)
					continue
				}

				if job.ClusterName == "" {
					logger.Error(nil, "cannot retry job: ClusterName is empty",
						"jobID", job.JobID)
					job.Phase = "Failed"
					job.Message = "Retry failed: ClusterName is empty"
					job.FailureReason = "InvalidJobState"
					r.setCompletionTime(job)
					continue
				}

				// Create new pod (will get new jobID)
				if err := r.createClusterJob(ctx, scenarioRun, job.ProviderName, job.ClusterName); err != nil {
					if errors.Is(err, errScenarioRunCancelled) {
						scenarioRun.Spec.CancelRequested = true
						job.CancelRequested = true
						job.Phase = "Cancelled"
						setCancellationCleanupOutcome(job, "not-required", "")
						r.setCompletionTime(job)
						continue
					}
					logger.Error(err, "failed to create retry job",
						"cluster", job.ClusterName,
						"retryAttempt", job.RetryCount)
					job.Phase = "Failed"
					job.Message = "Retry failed: " + err.Error()
					r.setCompletionTime(job)
				}
			} else if job.CancelRequested {
				job.Phase = "Cancelled"
				logger.Info("job marked as cancelled, no retry",
					"cluster", job.ClusterName,
					"jobID", job.JobID)
			} else {
				job.Phase = "MaxRetriesExceeded"
				logger.Info("job exceeded max retries",
					"cluster", job.ClusterName,
					"jobID", job.JobID,
					"retryCount", job.RetryCount,
					"maxRetries", maxRetries)
			}

			// A failed terminal run may still contain report markers in its logs.
			// Extract only after retries are exhausted or cancellation is final.
			if (job.Phase == "MaxRetriesExceeded" || job.Phase == "Cancelled" || job.Phase == "Failed") && scenarioRun.Status.ReportStatus == nil {
				if err := r.extractAndStoreReports(ctx, scenarioRun, job); err != nil {
					return err
				}
			}
		case corev1.PodUnknown:
			job.Phase = "Failed"
			job.Message = "Pod in unknown state"
			job.FailureReason = "PodUnknown"
			r.setCompletionTime(job)
			logger.Info("pod in unknown state",
				"cluster", job.ClusterName,
				"jobID", job.JobID,
				"podName", job.PodName)
			if scenarioRun.Status.ReportStatus == nil {
				if err := r.extractAndStoreReports(ctx, scenarioRun, job); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// setCompletionTime sets the completion time if not already set
func (r *KrknScenarioRunReconciler) setCompletionTime(job *krknv1alpha1.ClusterJobStatus) {
	if job.CompletionTime == nil {
		now := metav1.Now()
		job.CompletionTime = &now
	}
}

// extractPodErrorMessage extracts error message from pod status
func (r *KrknScenarioRunReconciler) extractPodErrorMessage(pod *corev1.Pod) string {
	if len(pod.Status.ContainerStatuses) == 0 {
		return ""
	}

	containerStatus := pod.Status.ContainerStatuses[0]
	if terminated := containerStatus.State.Terminated; terminated != nil {
		return terminated.Reason + ": " + terminated.Message
	}
	if waiting := containerStatus.State.Waiting; waiting != nil {
		return waiting.Reason + ": " + waiting.Message
	}
	return ""
}

// extractFailureReason extracts a categorized failure reason from pod
func (r *KrknScenarioRunReconciler) extractFailureReason(pod *corev1.Pod) string {
	if len(pod.Status.ContainerStatuses) == 0 {
		return "PodNotScheduled"
	}

	cs := pod.Status.ContainerStatuses[0]
	if cs.State.Terminated != nil {
		reason := cs.State.Terminated.Reason
		exitCode := cs.State.Terminated.ExitCode

		// Categorize common failures
		if exitCode == 137 {
			return "OOMKilled"
		}
		if exitCode == 143 {
			return "SIGTERM"
		}
		if reason == "Error" {
			return "ContainerError"
		}
		return reason
	}

	if cs.State.Waiting != nil {
		return cs.State.Waiting.Reason
	}

	return "Unknown"
}

// effectiveScenarioMaxRetries prevents rollback attempts from being retried,
// including when a run was created directly as a custom resource.
func effectiveScenarioMaxRetries(run *krknv1alpha1.KrknScenarioRun) int {
	if run.Spec.RollbackEnabled {
		return 0
	}
	return run.Spec.MaxRetries
}

// backfillLegacyMaxRetries restores the retry limit for jobs written by older
// controllers, which persisted zero before the field was introduced.
func backfillLegacyMaxRetries(job *krknv1alpha1.ClusterJobStatus, scenarioMaxRetries int) {
	if job.MaxRetriesConfigured {
		return
	}
	if job.MaxRetries != 0 {
		job.MaxRetriesConfigured = true
		return
	}
	if job.PodName == "" {
		return
	}
	job.MaxRetries = scenarioMaxRetries
	if job.MaxRetries == 0 {
		job.MaxRetries = 3
	}
	job.MaxRetriesConfigured = true
}

// shouldRetryJob determines if a failed job should be retried
func (r *KrknScenarioRunReconciler) shouldRetryJob(job *krknv1alpha1.ClusterJobStatus, maxRetries int) bool {
	// Don't retry if user cancelled
	if job.CancelRequested {
		return false
	}
	// Signature failures are terminal until the image or verification setting
	// changes; retrying would only repeat the same trust decision.
	if job.FailureReason == "InvalidImageSignature" {
		return false
	}

	// Don't retry if phase is already terminal
	if job.Phase == "Succeeded" || job.Phase == "Cancelled" || job.Phase == "MaxRetriesExceeded" {
		return false
	}

	// Check retry count against max. A zero on an unconfigured legacy value
	// retains the historic default; configured zero explicitly disables retries.
	if maxRetries == 0 && !job.MaxRetriesConfigured {
		maxRetries = 3
	}
	return job.RetryCount < maxRetries
}

// calculateRetryDelay calculates backoff delay based on retry count
func (r *KrknScenarioRunReconciler) calculateRetryDelay(retryCount int, backoffType, delayStr string) time.Duration {
	baseDelay := 10 * time.Second
	if delayStr != "" {
		if d, err := time.ParseDuration(delayStr); err == nil {
			baseDelay = d
		}
	}

	if backoffType == "exponential" {
		// Exponential: 10s, 20s, 40s, ...
		return baseDelay * time.Duration(1<<retryCount)
	}

	// Fixed: always same delay
	return baseDelay
}

// jobExistsForCluster checks if a job already exists for the given cluster
func (r *KrknScenarioRunReconciler) jobExistsForCluster(scenarioRun *krknv1alpha1.KrknScenarioRun, clusterName string) bool {
	for _, job := range scenarioRun.Status.ClusterJobs {
		if job.ClusterName == clusterName {
			// Don't count jobs in "Retrying" phase as existing,
			// since we need to create a new pod for them
			if job.Phase == "Retrying" {
				return false
			}
			return true
		}
	}
	return false
}

// getClusterAPIURL retrieves the ClusterAPIURL for a specific provider/cluster pair from the
// KrknTargetRequest referenced by the scenarioRun. Returns an empty string if the URL cannot
// be found, which callers must handle (job visibility to non-admins depends on this field).
func (r *KrknScenarioRunReconciler) getClusterAPIURL(
	ctx context.Context,
	scenarioRun *krknv1alpha1.KrknScenarioRun,
	providerName string,
	clusterName string,
) string {
	var targetRequest krknv1alpha1.KrknTargetRequest
	if err := r.Get(ctx, types.NamespacedName{
		Name:      scenarioRun.Spec.TargetRequestID,
		Namespace: r.Namespace,
	}, &targetRequest); err != nil {
		return ""
	}
	if providerTargets, exists := targetRequest.Status.TargetData[providerName]; exists {
		for _, target := range providerTargets {
			if target.ClusterName == clusterName {
				return target.ClusterAPIURL
			}
		}
	}
	return ""
}

// calculateOverallStatus computes the overall phase and counters
func (r *KrknScenarioRunReconciler) calculateOverallStatus(scenarioRun *krknv1alpha1.KrknScenarioRun) {
	var successfulJobs, failedJobs, cancelledJobs, runningJobs, pendingJobs int
	for _, job := range scenarioRun.Status.ClusterJobs {
		switch job.Phase {
		case "Succeeded":
			successfulJobs++
		case "Failed", "MaxRetriesExceeded":
			failedJobs++
		case "Cancelled":
			cancelledJobs++
		case "Running", "Retrying", "Cancelling":
			runningJobs++
		case "Pending", "Creating":
			pendingJobs++
		}
	}
	scenarioRun.Status.SuccessfulJobs = successfulJobs
	scenarioRun.Status.FailedJobs = failedJobs
	scenarioRun.Status.CancelledJobs = cancelledJobs
	scenarioRun.Status.RunningJobs = runningJobs

	totalJobs := len(scenarioRun.Status.ClusterJobs)
	if scenarioRun.Spec.CancelRequested {
		if runningJobs > 0 || pendingJobs > 0 {
			scenarioRun.Status.Phase = "Cancelling"
		} else if cancelledJobs > 0 || totalJobs == 0 {
			scenarioRun.Status.Phase = "Cancelled"
		} else if successfulJobs == totalJobs {
			scenarioRun.Status.Phase = "Succeeded"
		} else if failedJobs == totalJobs {
			scenarioRun.Status.Phase = "Failed"
		} else {
			scenarioRun.Status.Phase = "PartiallyFailed"
		}
		return
	}
	if totalJobs == 0 {
		scenarioRun.Status.Phase = "Pending"
	} else if runningJobs > 0 || pendingJobs > 0 {
		scenarioRun.Status.Phase = "Running"
	} else if cancelledJobs == totalJobs {
		scenarioRun.Status.Phase = "Cancelled"
	} else if failedJobs == totalJobs {
		scenarioRun.Status.Phase = "Failed"
	} else if successfulJobs == totalJobs {
		scenarioRun.Status.Phase = "Succeeded"
	} else {
		scenarioRun.Status.Phase = "PartiallyFailed"
	}
}

func (r *KrknScenarioRunReconciler) hasCancellationRequests(scenarioRun *krknv1alpha1.KrknScenarioRun) bool {
	if scenarioRun.Spec.CancelRequested {
		return true
	}
	for _, job := range scenarioRun.Status.ClusterJobs {
		if job.CancelRequested || job.Phase == "Cancelled" || job.Phase == "Cancelling" {
			return true
		}
	}
	return false
}

func (r *KrknScenarioRunReconciler) setCancellationConditions(scenarioRun *krknv1alpha1.KrknScenarioRun) {
	executionStopped := true
	for _, job := range scenarioRun.Status.ClusterJobs {
		switch job.Phase {
		case "Creating", "Pending", "Running", "Retrying", "Cancelling":
			executionStopped = false
		}
	}
	executionStatus, executionReason, executionMessage := metav1.ConditionTrue, "ExecutionStopped", "All active scenario jobs have stopped."
	if !executionStopped {
		executionStatus, executionReason, executionMessage = metav1.ConditionFalse, "CancellationPending", "Waiting for active scenario jobs to stop."
	}
	meta.SetStatusCondition(&scenarioRun.Status.Conditions, metav1.Condition{
		Type: "ExecutionStopped", Status: executionStatus, Reason: executionReason,
		Message: executionMessage, ObservedGeneration: scenarioRun.Generation,
	})
	cleanupUnknown, cleanupFailed := false, false
	cleanupMessage := "No cancelled scenario job has unresolved cleanup."
	for _, job := range scenarioRun.Status.ClusterJobs {
		if job.Phase == "Cancelling" ||
			(job.CancelRequested && job.Phase != "Succeeded" && job.Phase != "Failed" && job.Phase != "MaxRetriesExceeded" && job.Phase != "Cancelled") {
			cleanupUnknown = true
			cleanupMessage = "Waiting for active scenario cleanup to finish."
			continue
		}
		if job.Phase != "Cancelled" {
			continue
		}
		switch job.FailureReason {
		case "CancellationCleanupComplete", "CancellationCleanupNotRequired":
		case "CancellationCleanupFailed":
			cleanupFailed = true
			cleanupMessage = job.Message
		default:
			cleanupUnknown = true
			cleanupMessage = job.Message
		}
	}
	cleanupStatus, cleanupReason := metav1.ConditionTrue, "CleanupComplete"
	if cleanupUnknown {
		cleanupStatus, cleanupReason = metav1.ConditionUnknown, "CleanupPending"
	} else if cleanupFailed {
		cleanupStatus, cleanupReason = metav1.ConditionFalse, "CleanupFailed"
	}
	meta.SetStatusCondition(&scenarioRun.Status.Conditions, metav1.Condition{
		Type: "CleanupComplete", Status: cleanupStatus, Reason: cleanupReason,
		Message: cleanupMessage, ObservedGeneration: scenarioRun.Generation,
	})
}

func (r *KrknScenarioRunReconciler) cancellationCleanupPending(scenarioRun *krknv1alpha1.KrknScenarioRun) bool {
	condition := meta.FindStatusCondition(scenarioRun.Status.Conditions, "CleanupComplete")
	return condition == nil || condition.Status == metav1.ConditionUnknown
}

// getKubeconfigFromProvider retrieves kubeconfig from a provider-specific Secret.
func (r *KrknScenarioRunReconciler) getKubeconfigFromProvider(ctx context.Context, targetID string, providerName string, clusterName string) (string, error) {
	return resolveManagedClusterKubeconfig(ctx, r.Client, r.Namespace, targetID, providerName, clusterName)
}

// hasIncompleteResiliencyScores returns true when resiliency score calculation
// should be (re-)attempted: either no scores exist yet, or at least one cluster
// entry has status "error".
func (r *KrknScenarioRunReconciler) hasIncompleteResiliencyScores(scenarioRun *krknv1alpha1.KrknScenarioRun) bool {
	if len(scenarioRun.Status.ResiliencyScores) == 0 {
		return true
	}
	for _, s := range scenarioRun.Status.ResiliencyScores {
		if s.Status == "error" {
			return true
		}
	}
	return false
}

// statusEqual compares two KrknScenarioRunStatus to determine if they are equal
// This is a semantic comparison that handles pointer fields correctly
func (r *KrknScenarioRunReconciler) statusEqual(old, new *krknv1alpha1.KrknScenarioRunStatus) bool {
	// Compare scalar fields
	if old.Phase != new.Phase {
		return false
	}
	if old.Message != new.Message {
		return false
	}
	if old.TotalTargets != new.TotalTargets {
		return false
	}
	if old.SuccessfulJobs != new.SuccessfulJobs {
		return false
	}
	if old.FailedJobs != new.FailedJobs {
		return false
	}
	if old.RunningJobs != new.RunningJobs {
		return false
	}

	// Compare ClusterJobs array length
	if len(old.ClusterJobs) != len(new.ClusterJobs) {
		return false
	}

	// Compare each job
	for i := range old.ClusterJobs {
		if !r.jobStatusEqual(&old.ClusterJobs[i], &new.ClusterJobs[i]) {
			return false
		}
	}

	// Compare ResiliencyScores
	if !reflect.DeepEqual(old.ResiliencyScores, new.ResiliencyScores) {
		return false
	}

	// Compare Conditions array
	if !reflect.DeepEqual(old.Conditions, new.Conditions) {
		return false
	}

	return true
}

// jobStatusEqual compares two ClusterJobStatus semantically
func (r *KrknScenarioRunReconciler) jobStatusEqual(old, new *krknv1alpha1.ClusterJobStatus) bool {
	// Compare scalar fields
	if old.ProviderName != new.ProviderName ||
		old.ClusterName != new.ClusterName ||
		old.ClusterAPIURL != new.ClusterAPIURL ||
		old.JobID != new.JobID ||
		old.PodName != new.PodName ||
		old.ScenarioImage != new.ScenarioImage ||
		!reflect.DeepEqual(old.AttemptHistory, new.AttemptHistory) ||
		old.Phase != new.Phase ||
		old.Message != new.Message ||
		old.RetryCount != new.RetryCount ||
		old.MaxRetries != new.MaxRetries ||
		old.CancelRequested != new.CancelRequested ||
		old.FailureReason != new.FailureReason {
		return false
	}

	// Compare time pointers - check if both nil or both have same value
	if !timeEqual(old.StartTime, new.StartTime) ||
		!timeEqual(old.CompletionTime, new.CompletionTime) ||
		!timeEqual(old.LastRetryTime, new.LastRetryTime) {
		return false
	}

	return true
}

// timeEqual compares two metav1.Time pointers semantically
func timeEqual(t1, t2 *metav1.Time) bool {
	if t1 == nil && t2 == nil {
		return true
	}
	if t1 == nil || t2 == nil {
		return false
	}
	// Compare the actual time values, not the pointers
	return t1.Time.Equal(t2.Time)
}

// detectStatusChanges returns a human-readable description of what changed between two statuses
func (r *KrknScenarioRunReconciler) detectStatusChanges(old, new *krknv1alpha1.KrknScenarioRunStatus) string {
	var changes []string

	// Phase change
	if old.Phase != new.Phase {
		changes = append(changes, fmt.Sprintf("phase:%s→%s", old.Phase, new.Phase))
	}

	// Job count changes
	addCountChange := func(name string, oldVal, newVal int) {
		if oldVal != newVal {
			changes = append(changes, fmt.Sprintf("%s:%d→%d", name, oldVal, newVal))
		}
	}
	addCountChange("successful", old.SuccessfulJobs, new.SuccessfulJobs)
	addCountChange("failed", old.FailedJobs, new.FailedJobs)
	addCountChange("running", old.RunningJobs, new.RunningJobs)

	// Job phase changes
	phaseChanges := countJobPhaseChanges(old.ClusterJobs, new.ClusterJobs)
	if phaseChanges > 0 {
		changes = append(changes, fmt.Sprintf("%d job phase changes", phaseChanges))
	}

	// New jobs
	if newJobs := len(new.ClusterJobs) - len(old.ClusterJobs); newJobs > 0 {
		changes = append(changes, fmt.Sprintf("+%d new jobs", newJobs))
	}

	if len(changes) == 0 {
		return "unknown changes"
	}
	return strings.Join(changes, ", ")
}

// countJobPhaseChanges counts how many jobs changed phase between old and new
func countJobPhaseChanges(oldJobs, newJobs []krknv1alpha1.ClusterJobStatus) int {
	count := 0
	minLen := len(oldJobs)
	if len(newJobs) < minLen {
		minLen = len(newJobs)
	}
	for i := 0; i < minLen; i++ {
		if oldJobs[i].Phase != newJobs[i].Phase {
			count++
		}
	}
	return count
}

// fetchScenarioPodLogs fetches logs from a completed scenario pod.
// A log-fetch error is returned to controller-runtime so reconciliation can
// retry without blocking a worker inside a local backoff loop.
func (r *KrknScenarioRunReconciler) fetchScenarioPodLogs(ctx context.Context, podName string) ([]byte, error) {
	logger := log.FromContext(ctx).WithName("fetch-scenario-logs")

	logger.V(1).Info("fetching logs from scenario pod", "pod", podName)
	if r.Clientset == nil {
		return nil, fmt.Errorf("cannot fetch logs for pod %s: Kubernetes client is not configured", podName)
	}

	req := r.Clientset.CoreV1().Pods(r.Namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: "scenario",
	})

	podLogs, err := req.Stream(ctx)
	if err != nil {
		logger.V(1).Info("pod logs not available; reconciliation will retry", "pod", podName, "error", err.Error())
		return nil, fmt.Errorf("failed to open logs for pod %s: %w", podName, err)
	}
	defer podLogs.Close()

	logBytes, err := io.ReadAll(podLogs)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs for pod %s: %w", podName, err)
	}

	logger.V(1).Info("successfully fetched pod logs", "pod", podName, "logSize", len(logBytes))
	return logBytes, nil
}

// extractDelimitedContent finds content between start and end markers in logs.
// Returns the content string and a boolean indicating if it was found.
func extractDelimitedContent(logs, startMarker, endMarker string) (string, bool) {
	startIdx := strings.LastIndex(logs, startMarker)
	if startIdx == -1 {
		return "", false
	}
	startIdx += len(startMarker)

	endIdx := strings.LastIndex(logs, endMarker)
	if endIdx == -1 || endIdx <= startIdx {
		return "", false
	}

	return logs[startIdx:endIdx], true
}

// decodeReportPayloads decodes the report payloads extracted from pod logs.
// Invalid payloads are reported individually so one valid report can still be
// stored when the other format is corrupt.
func decodeReportPayloads(htmlB64, pdfB64 string) (map[string]string, map[string][]byte, []string) {
	data := make(map[string]string)
	binaryData := make(map[string][]byte)
	var messages []string

	if htmlB64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(htmlB64))
		if err != nil {
			messages = append(messages, fmt.Sprintf("failed to decode HTML report: %v", err))
		} else {
			data["summary.html"] = string(decoded)
		}
	}

	if pdfB64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(pdfB64))
		if err != nil {
			messages = append(messages, fmt.Sprintf("failed to decode PDF report: %v", err))
		} else {
			binaryData["summary.pdf"] = decoded
		}
	}

	return data, binaryData, messages
}

func targetClusterCount(targetClusters map[string][]string) int {
	count := 0
	for _, clusters := range targetClusters {
		count += len(clusters)
	}
	return count
}

const maxReportConfigMapDataBytes int64 = 900 * 1024

func validateReportPayloadSize(data map[string]string, binaryData map[string][]byte) error {
	var size int64
	for _, value := range data {
		size += int64(len(value))
	}
	for _, value := range binaryData {
		size += int64(len(value))
	}
	if size > maxReportConfigMapDataBytes {
		return fmt.Errorf("report payload is %d bytes; the supported ConfigMap limit is %d bytes", size, maxReportConfigMapDataBytes)
	}
	return nil
}

// extractAndStoreReports extracts base64-encoded reports from pod logs and stores them in a ConfigMap.
func (r *KrknScenarioRunReconciler) extractAndStoreReports(ctx context.Context, scenarioRun *krknv1alpha1.KrknScenarioRun, job *krknv1alpha1.ClusterJobStatus) error {
	logger := log.FromContext(ctx)
	clusterCount := targetClusterCount(scenarioRun.Spec.TargetClusters)
	if clusterCount > 1 {
		message := fmt.Sprintf("Report generation is not supported for multi-cluster scenario run %q; this run targets %d clusters", scenarioRun.Name, clusterCount)
		scenarioRun.Status.ReportStatus = &krknv1alpha1.ReportStatus{Message: message}
		logger.Info(message)
		return nil
	}

	logs, err := r.fetchScenarioPodLogs(ctx, job.PodName)
	if err != nil {
		logger.Error(err, "unable to read logs for report extraction; reconciliation will retry", "pod", job.PodName)
		return err
	}

	logsStr := string(logs)
	htmlB64, htmlFound := extractDelimitedContent(logsStr, "===KRKN_REPORT_HTML_START===", "===KRKN_REPORT_HTML_END===")
	pdfB64, pdfFound := extractDelimitedContent(logsStr, "===KRKN_REPORT_PDF_START===", "===KRKN_REPORT_PDF_END===")

	if !htmlFound && !pdfFound {
		message := fmt.Sprintf("No reports found for scenario run %q in completed pod %q; expected kraken.report.html or kraken.report.pdf", scenarioRun.Name, job.PodName)
		scenarioRun.Status.ReportStatus = &krknv1alpha1.ReportStatus{Message: message}
		logger.V(1).Info(message, "pod", job.PodName)
		return nil
	}

	cmName := fmt.Sprintf("krkn-report-%s", scenarioRun.Name)
	now := metav1.Now()

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: r.Namespace,
			Labels: map[string]string{
				"app":               "krkn-operator",
				"component":         "reports",
				"krkn-scenario-run": scenarioRun.Name,
			},
			Annotations: map[string]string{
				"krkn.krkn-chaos.dev/report-job-id": job.JobID,
			},
		},
		Data:       make(map[string]string),
		BinaryData: make(map[string][]byte),
	}

	decodedHTML, decodedPDF, decodeMessages := decodeReportPayloads(htmlB64, pdfB64)
	for name, value := range decodedHTML {
		configMap.Data[name] = value
	}
	for name, value := range decodedPDF {
		configMap.BinaryData[name] = value
	}

	if len(configMap.Data) == 0 && len(configMap.BinaryData) == 0 {
		message := fmt.Sprintf("Report generation failed for scenario run %q in pod %q: %s", scenarioRun.Name, job.PodName, strings.Join(decodeMessages, "; "))
		scenarioRun.Status.ReportStatus = &krknv1alpha1.ReportStatus{Message: message}
		logger.Info(message, "pod", job.PodName)
		return nil
	}

	if err := validateReportPayloadSize(configMap.Data, configMap.BinaryData); err != nil {
		message := fmt.Sprintf("Report generation failed for scenario run %q: %v", scenarioRun.Name, err)
		scenarioRun.Status.ReportStatus = &krknv1alpha1.ReportStatus{Message: message}
		logger.Info(message, "pod", job.PodName)
		return nil
	}

	if err := controllerutil.SetControllerReference(scenarioRun, configMap, r.Scheme); err != nil {
		scenarioRun.Status.ReportStatus = &krknv1alpha1.ReportStatus{Message: fmt.Sprintf("Report generation failed for scenario run %q: unable to set ownership on ConfigMap %q: %v", scenarioRun.Name, cmName, err)}
		logger.Error(err, "failed to set owner reference on report ConfigMap", "configMapName", cmName)
		return err
	}

	if err := r.Create(ctx, configMap); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			scenarioRun.Status.ReportStatus = &krknv1alpha1.ReportStatus{Message: fmt.Sprintf("Report generation failed for scenario run %q: unable to create ConfigMap %q: %v", scenarioRun.Name, cmName, err)}
			logger.Error(err, "failed to create report ConfigMap", "configMapName", cmName)
			return err
		}
		var existing corev1.ConfigMap
		if err := r.Get(ctx, types.NamespacedName{Name: cmName, Namespace: r.Namespace}, &existing); err != nil {
			scenarioRun.Status.ReportStatus = &krknv1alpha1.ReportStatus{Message: fmt.Sprintf("Report generation failed for scenario run %q: unable to read existing ConfigMap %q: %v", scenarioRun.Name, cmName, err)}
			logger.Error(err, "failed to get existing report ConfigMap", "configMapName", cmName)
			return err
		}
		existing.Data = configMap.Data
		existing.BinaryData = configMap.BinaryData
		existing.Annotations = configMap.Annotations
		if err := r.Update(ctx, &existing); err != nil {
			scenarioRun.Status.ReportStatus = &krknv1alpha1.ReportStatus{Message: fmt.Sprintf("Report generation failed for scenario run %q: unable to update ConfigMap %q: %v", scenarioRun.Name, cmName, err)}
			logger.Error(err, "failed to update existing report ConfigMap", "configMapName", cmName)
			return err
		}
	}

	scenarioRun.Status.ReportStatus = &krknv1alpha1.ReportStatus{
		Generated:     true,
		HTMLAvailable: len(configMap.Data) > 0 && configMap.Data["summary.html"] != "",
		PDFAvailable:  len(configMap.BinaryData) > 0 && len(configMap.BinaryData["summary.pdf"]) > 0,
		GeneratedAt:   &now,
		Location:      cmName,
		FileSize:      int64(len(configMap.Data["summary.html"]) + len(configMap.BinaryData["summary.pdf"])),
		Message:       strings.Join(decodeMessages, "; "),
	}

	logger.Info("extracted and stored reports in ConfigMap", "configMapName", cmName, "htmlAvailable", scenarioRun.Status.ReportStatus.HTMLAvailable, "pdfAvailable", scenarioRun.Status.ReportStatus.PDFAvailable)
	return nil
}

// SetupWithManager sets up the controller with the Manager
func (r *KrknScenarioRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Only watch Pod events to track job completion — ConfigMap/Secret changes
	// don't affect reconciliation logic and including them caused infinite loops
	// when partial resource creates were cleaned up on pod-creation failure.
	// Owner references on ConfigMaps/Secrets still ensure GC when the CR is deleted.
	return ctrl.NewControllerManagedBy(mgr).
		For(&krknv1alpha1.KrknScenarioRun{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
