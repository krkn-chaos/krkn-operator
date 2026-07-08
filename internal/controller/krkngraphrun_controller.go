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

package controller

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/graph"
)

const (
	// GraphRunLabelKey is the label key for identifying KrknScenarioRuns created by a KrknGraphRun
	GraphRunLabelKey = "krkn.dev/graph-run"

	// GraphNodeLabelKey is the label key for identifying which node a KrknScenarioRun represents
	GraphNodeLabelKey = "krkn.dev/graph-node"

	// FinalizerName is the finalizer for KrknGraphRun cleanup
	FinalizerName = "krkn.dev/graph-run-finalizer"

	// MaxLabelValueLength is the maximum length for Kubernetes label values (RFC 1123)
	MaxLabelValueLength = 63
)

// KrknGraphRunReconciler reconciles a KrknGraphRun object
type KrknGraphRunReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Clientset kubernetes.Interface
	Namespace string
}

// sanitizeNodeID sanitizes a node ID for use in Kubernetes resource names and label values.
// It replaces invalid characters with hyphens, converts to lowercase, truncates to 63 characters,
// and ensures it starts and ends with an alphanumeric character.
//
// Kubernetes label value requirements (RFC 1123):
// - Maximum 63 characters
// - Only lowercase alphanumeric characters, '-', '_', or '.'
// - Must start and end with an alphanumeric character
func sanitizeNodeID(nodeID string) string {
	if nodeID == "" {
		return "empty"
	}

	// Convert to lowercase
	sanitized := strings.ToLower(nodeID)

	// Replace invalid characters with hyphens
	// Valid: alphanumeric, '-', '_', '.'
	var builder strings.Builder
	builder.Grow(len(sanitized))
	for _, r := range sanitized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			builder.WriteRune(r)
		} else {
			builder.WriteRune('-')
		}
	}
	sanitized = builder.String()

	// Truncate to max label value length
	if len(sanitized) > MaxLabelValueLength {
		sanitized = sanitized[:MaxLabelValueLength]
	}

	// Ensure it starts with alphanumeric
	sanitized = strings.TrimLeft(sanitized, "-_.")
	if sanitized == "" {
		return "node"
	}

	// Ensure it ends with alphanumeric
	sanitized = strings.TrimRight(sanitized, "-_.")
	if sanitized == "" {
		return "node"
	}

	return sanitized
}

// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krkngraphruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krkngraphruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krkngraphruns/finalizers,verbs=update
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknscenarioruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknscenarioruns/status,verbs=get

// Reconcile handles the reconciliation loop for KrknGraphRun
func (r *KrknGraphRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile loop started", "graphRun", req.Name, "namespace", req.Namespace)

	// 1. Fetch the KrknGraphRun instance
	var graphRun krknv1alpha1.KrknGraphRun
	if err := r.Get(ctx, req.NamespacedName, &graphRun); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("graphRun not found, probably deleted", "graphRun", req.Name)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch KrknGraphRun")
		return ctrl.Result{}, err
	}

	// 2. Handle deletion with finalizer
	if !graphRun.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&graphRun, FinalizerName) {
			logger.Info("removing finalizer after cleanup", "graphRun", graphRun.Name)
			controllerutil.RemoveFinalizer(&graphRun, FinalizerName)
			if err := r.Update(ctx, &graphRun); err != nil {
				return ctrl.Result{}, err
			}
		}
		// Owner references will cascade delete KrknScenarioRuns
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&graphRun, FinalizerName) {
		controllerutil.AddFinalizer(&graphRun, FinalizerName)
		if err := r.Update(ctx, &graphRun); err != nil {
			return ctrl.Result{}, err
		}
		// Requeue to continue processing with finalizer added
		return ctrl.Result{Requeue: true}, nil
	}

	// 3. Initialize status if first reconcile
	if graphRun.Status.Phase == "" {
		if err := r.initializeStatus(ctx, &graphRun); err != nil {
			logger.Error(err, "failed to initialize status")
			// If it's a conflict error, just requeue - the object was modified concurrently
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// 4. Skip reconciliation if GraphRun is in a terminal state
	if r.isTerminalPhase(graphRun.Status.Phase) {
		logger.V(1).Info("GraphRun is in terminal state, skipping reconciliation",
			"phase", graphRun.Status.Phase)
		return ctrl.Result{}, nil
	}

	// 5. Resolve the dependency graph if not already done
	if len(graphRun.Status.ResolvedLevels) == 0 {
		if err := r.resolveGraph(ctx, &graphRun); err != nil {
			logger.Error(err, "failed to resolve graph")
			// If it's a conflict error, just requeue - the object was modified concurrently
			// and will be re-fetched on next reconcile with fresh data
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			// For other errors, mark as Failed
			return r.updateStatusWithError(ctx, &graphRun, err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// 5. Query existing KrknScenarioRuns for this graph
	existingRuns, err := r.getExistingScenarioRuns(ctx, &graphRun)
	if err != nil {
		logger.Error(err, "failed to query existing scenario runs")
		return ctrl.Result{}, err
	}

	// 6. Process each level in order
	result, statusAlreadyUpdated, err := r.processLevels(ctx, &graphRun, existingRuns)
	if err != nil {
		logger.Error(err, "failed to process graph levels")
		return r.updateStatusWithError(ctx, &graphRun, err)
	}

	// 7. Calculate global phase and summary (if not already updated by fail-fast)
	if !statusAlreadyUpdated {
		r.calculateGlobalStatus(&graphRun)

		// 7.1. Calculate resiliency score if enabled and GraphRun is in terminal state
		if graphRun.Spec.ResiliencyScoreEnabled &&
		   r.isTerminalPhase(graphRun.Status.Phase) &&
		   graphRun.Status.ResiliencyScore == nil {
			if err := r.calculateResiliencyScore(ctx, &graphRun, existingRuns); err != nil {
				logger.Error(err, "failed to calculate resiliency score", "graphRun", graphRun.Name)
				// Don't fail the reconcile, just log the error
			}
		}

		// 8. Persist all status updates in a single call
		if err := r.Status().Update(ctx, &graphRun); err != nil {
			if apierrors.IsConflict(err) {
				logger.Info("conflict on final status update, will retry on next reconcile")
				return ctrl.Result{Requeue: true}, nil
			}
			logger.Error(err, "failed to update global status")
			return ctrl.Result{}, err
		}
	}

	logger.Info("reconcile loop completed",
		"graphRun", graphRun.Name,
		"phase", graphRun.Status.Phase,
		"totalNodes", graphRun.Status.Summary.TotalNodes,
		"completedNodes", graphRun.Status.Summary.CompletedNodes)

	return result, nil
}

// initializeStatus initializes the GraphRun status on first reconcile
func (r *KrknGraphRunReconciler) initializeStatus(ctx context.Context, graphRun *krknv1alpha1.KrknGraphRun) error {
	logger := log.FromContext(ctx)

	graphRun.Status.Phase = "Pending"
	graphRun.Status.StartTime = &metav1.Time{Time: time.Now()}
	graphRun.Status.NodeStatuses = []krknv1alpha1.NodeStatus{}

	// Count total nodes (filter out metadata nodes starting with '_')
	totalNodes := 0
	for nodeID := range graphRun.Spec.Graph {
		if !strings.HasPrefix(nodeID, "_") {
			totalNodes++
		}
	}

	graphRun.Status.Summary = krknv1alpha1.GraphRunSummary{
		TotalNodes:     totalNodes,
		PendingNodes:   totalNodes,
		CompletedNodes: 0,
		RunningNodes:   0,
		FailedNodes:    0,
	}

	logger.Info("initializing graphRun status",
		"graphRun", graphRun.Name,
		"totalNodes", totalNodes)

	// Update with conflict retry
	if err := r.Status().Update(ctx, graphRun); err != nil {
		if apierrors.IsConflict(err) {
			logger.Info("conflict in initializeStatus, will retry on next reconcile")
			return fmt.Errorf("conflict updating status: %w", err)
		}
		return err
	}
	return nil
}

// resolveGraph resolves the dependency graph into topological levels
func (r *KrknGraphRunReconciler) resolveGraph(ctx context.Context, graphRun *krknv1alpha1.KrknGraphRun) error {
	logger := log.FromContext(ctx)

	logger.Info("resolving dependency graph", "graphRun", graphRun.Name)

	levels, err := graph.ResolveGraph(graphRun.Spec.Graph)
	if err != nil {
		return fmt.Errorf("failed to resolve graph: %w", err)
	}

	graphRun.Status.ResolvedLevels = levels

	// Initialize node statuses (filter out metadata nodes starting with '_')
	nodeStatuses := make([]krknv1alpha1.NodeStatus, 0, len(graphRun.Spec.Graph))
	for nodeID, node := range graphRun.Spec.Graph {
		// Skip metadata nodes (same filter as in pkg/graph/ResolveGraph)
		if strings.HasPrefix(nodeID, "_") {
			continue
		}

		var dependsOn []string
		if node.DependsOn != nil {
			dependsOn = []string{*node.DependsOn}
		}

		nodeStatuses = append(nodeStatuses, krknv1alpha1.NodeStatus{
			NodeID:    nodeID,
			NodeName:  node.Name,
			Phase:     "Pending",
			DependsOn: dependsOn,
		})
	}
	graphRun.Status.NodeStatuses = nodeStatuses

	logger.Info("graph resolved successfully",
		"graphRun", graphRun.Name,
		"levels", len(levels),
		"nodes", len(nodeStatuses))

	// Update with conflict retry
	if err := r.Status().Update(ctx, graphRun); err != nil {
		if apierrors.IsConflict(err) {
			logger.Info("conflict in resolveGraph, will retry on next reconcile")
			return fmt.Errorf("conflict updating status: %w", err)
		}
		return err
	}
	return nil
}

// getExistingScenarioRuns queries for KrknScenarioRuns created by this graph.
// Returns a map keyed by sanitized node ID (as stored in labels).
func (r *KrknGraphRunReconciler) getExistingScenarioRuns(ctx context.Context, graphRun *krknv1alpha1.KrknGraphRun) (map[string]*krknv1alpha1.KrknScenarioRun, error) {
	var runList krknv1alpha1.KrknScenarioRunList
	if err := r.List(ctx, &runList,
		client.InNamespace(graphRun.Namespace),
		client.MatchingLabels{GraphRunLabelKey: graphRun.Name},
	); err != nil {
		return nil, err
	}

	// Map uses sanitized node IDs as keys (from labels)
	runs := make(map[string]*krknv1alpha1.KrknScenarioRun)
	for i := range runList.Items {
		sanitizedNodeID := runList.Items[i].Labels[GraphNodeLabelKey]
		if sanitizedNodeID != "" {
			runs[sanitizedNodeID] = &runList.Items[i]
		}
	}

	return runs, nil
}

// processLevels processes each level of the dependency graph.
// Returns (result, statusAlreadyUpdated, error).
func (r *KrknGraphRunReconciler) processLevels(
	ctx context.Context,
	graphRun *krknv1alpha1.KrknGraphRun,
	existingRuns map[string]*krknv1alpha1.KrknScenarioRun,
) (ctrl.Result, bool, error) {
	logger := log.FromContext(ctx)

	for levelIdx, level := range graphRun.Status.ResolvedLevels {
		logger.Info("processing level", "graphRun", graphRun.Name, "level", levelIdx, "nodes", level)

		// Check if previous levels are complete
		if levelIdx > 0 {
			prevLevelComplete, hasFailures := r.checkPreviousLevel(graphRun, levelIdx-1)
			if hasFailures {
				// Fail-fast: mark dependent nodes as blocked (already updates status)
				return r.handleFailFast(ctx, graphRun, levelIdx)
			}
			if !prevLevelComplete {
				logger.Info("previous level not complete, requeuing",
					"graphRun", graphRun.Name,
					"level", levelIdx-1)
				return ctrl.Result{RequeueAfter: 10 * time.Second}, false, nil
			}
		}

		// Process each node in this level (modifies status in memory)
		for _, nodeID := range level {
			_, err := r.processNode(ctx, graphRun, nodeID, existingRuns)
			if err != nil {
				return ctrl.Result{}, false, err
			}
		}
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, false, nil
}

// checkPreviousLevel checks if all nodes in the previous level are complete
func (r *KrknGraphRunReconciler) checkPreviousLevel(graphRun *krknv1alpha1.KrknGraphRun, levelIdx int) (complete bool, hasFailures bool) {
	if levelIdx < 0 || levelIdx >= len(graphRun.Status.ResolvedLevels) {
		return true, false
	}

	nodesInLevel := graphRun.Status.ResolvedLevels[levelIdx]
	allComplete := true
	anyFailed := false

	for _, nodeID := range nodesInLevel {
		nodeStatus := r.findNodeStatus(graphRun, nodeID)
		if nodeStatus == nil {
			allComplete = false
			continue
		}

		if nodeStatus.Phase == "Failed" {
			anyFailed = true
		}

		if nodeStatus.Phase != "Completed" && nodeStatus.Phase != "Failed" {
			allComplete = false
		}
	}

	return allComplete, anyFailed
}

// findNodeStatus finds a node status by node ID
func (r *KrknGraphRunReconciler) findNodeStatus(graphRun *krknv1alpha1.KrknGraphRun, nodeID string) *krknv1alpha1.NodeStatus {
	for i := range graphRun.Status.NodeStatuses {
		if graphRun.Status.NodeStatuses[i].NodeID == nodeID {
			return &graphRun.Status.NodeStatuses[i]
		}
	}
	return nil
}

// processNode processes a single node in the graph.
// Returns true if the node status was modified.
func (r *KrknGraphRunReconciler) processNode(
	ctx context.Context,
	graphRun *krknv1alpha1.KrknGraphRun,
	nodeID string,
	existingRuns map[string]*krknv1alpha1.KrknScenarioRun,
) (bool, error) {
	// Sanitize node ID for lookup (existingRuns uses sanitized IDs as keys)
	sanitizedNodeID := sanitizeNodeID(nodeID)

	// Check if scenario run already exists
	scenarioRun, exists := existingRuns[sanitizedNodeID]
	if !exists {
		// Create new scenario run
		return r.createScenarioRun(ctx, graphRun, nodeID)
	}

	// Update node status from existing scenario run
	return r.updateNodeStatusFromRun(graphRun, nodeID, scenarioRun)
}

// createScenarioRun creates a KrknScenarioRun for a graph node.
// Returns true if the node status was modified.
func (r *KrknGraphRunReconciler) createScenarioRun(
	ctx context.Context,
	graphRun *krknv1alpha1.KrknGraphRun,
	nodeID string,
) (bool, error) {
	logger := log.FromContext(ctx)

	node, ok := graphRun.Spec.Graph[nodeID]
	if !ok {
		return false, fmt.Errorf("node %s not found in graph", nodeID)
	}

	// Translate Volumes (file UUID -> mount path) to FileMount objects
	// Volumes format: {"<file-uuid>": "/mount/path"}
	fileMounts, err := r.translateVolumesToFileMounts(ctx, node.Volumes)
	if err != nil {
		logger.Error(err, "failed to translate volumes to file mounts",
			"nodeID", nodeID,
			"volumes", node.Volumes)
		return false, fmt.Errorf("failed to translate volumes: %w", err)
	}

	// Map node to scenario run spec
	spec, err := graph.MapScenarioNodeToScenarioRunSpec(
		node,
		node.Name,
		graphRun.Spec.TargetRequestID,
		graphRun.Spec.TargetClusters,
		graphRun.Spec.OwnerUserID,
	)
	if err != nil {
		return false, fmt.Errorf("failed to map node to scenario run spec: %w", err)
	}

	// Add translated file mounts to spec
	if len(fileMounts) > 0 {
		spec.Files = fileMounts
		logger.V(1).Info("added file mounts to scenario run spec",
			"nodeID", nodeID,
			"fileCount", len(fileMounts))
	}

	// Handle resiliency score environment variables (controller-reserved)
	// These env vars are exclusively managed by the controller and should never
	// come from user-defined node.Env. Strip them first to prevent leakage.
	logger.Info("checking resiliency score configuration",
		"nodeID", nodeID,
		"resiliencyScoreEnabled", graphRun.Spec.ResiliencyScoreEnabled,
		"resiliencyMountPath", graphRun.Spec.ResiliencyMountPath)

	// Initialize environment map if nil
	if spec.Environment == nil {
		spec.Environment = make(map[string]string)
	}

	// Remove any user-provided RESILIENCY_* env vars (reserved by controller)
	delete(spec.Environment, "RESILIENCY_SCORE")
	delete(spec.Environment, "RESILIENCY_FILE")

	// Add resiliency env vars ONLY if enabled
	if graphRun.Spec.ResiliencyScoreEnabled {
		// Set RESILIENCY_SCORE=true to enable resiliency scoring in the scenario pod
		spec.Environment["RESILIENCY_SCORE"] = "true"
		logger.Info("added RESILIENCY_SCORE environment variable",
			"nodeID", nodeID,
			"value", "true")

		// If a resiliency mount path is specified, check if this node has a file mounted at that path
		if graphRun.Spec.ResiliencyMountPath != "" {
			// Search for a file mount matching the resiliency mount path
			for _, fileMount := range fileMounts {
				if fileMount.MountPath == graphRun.Spec.ResiliencyMountPath {
					// Found the resiliency metrics file - set RESILIENCY_FILE env var
					spec.Environment["RESILIENCY_FILE"] = graphRun.Spec.ResiliencyMountPath
					logger.V(1).Info("configured resiliency file for node",
						"nodeID", nodeID,
						"resiliencyFile", graphRun.Spec.ResiliencyMountPath)
					break
				}
			}
		}

		logger.V(1).Info("enabled resiliency score for node",
			"nodeID", nodeID,
			"resiliencyEnabled", true,
			"resiliencyFile", spec.Environment["RESILIENCY_FILE"])
	}

	// Sanitize node ID for use in Kubernetes resource names and labels
	sanitizedNodeID := sanitizeNodeID(nodeID)

	// Create scenario run
	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", graphRun.Name, sanitizedNodeID),
			Namespace: graphRun.Namespace,
			Labels: map[string]string{
				GraphRunLabelKey:  graphRun.Name,
				GraphNodeLabelKey: sanitizedNodeID,
			},
		},
		Spec: spec,
	}

	// Set owner reference for cascade deletion
	if err := controllerutil.SetControllerReference(graphRun, scenarioRun, r.Scheme); err != nil {
		return false, err
	}

	logger.Info("creating scenario run for node",
		"graphRun", graphRun.Name,
		"nodeID", nodeID,
		"scenarioRun", scenarioRun.Name)

	if err := r.Create(ctx, scenarioRun); err != nil {
		return false, err
	}

	// Update node status in memory
	return r.setNodeStatus(graphRun, nodeID, "Running", scenarioRun.Name, nil, nil)
}

// updateNodeStatusFromRun updates a node's status from its scenario run.
// Returns true if the node status was modified.
func (r *KrknGraphRunReconciler) updateNodeStatusFromRun(
	graphRun *krknv1alpha1.KrknGraphRun,
	nodeID string,
	scenarioRun *krknv1alpha1.KrknScenarioRun,
) (bool, error) {
	phase := "Running"
	var startTime, completionTime *metav1.Time

	// Map scenario run phase to node phase
	switch scenarioRun.Status.Phase {
	case "Succeeded":
		phase = "Completed"
	case "Failed", "PartiallyFailed":
		phase = "Failed"
	case "Running":
		phase = "Running"
	case "Pending":
		phase = "Pending"
	}

	// Get timing information from first cluster job (approximation)
	if len(scenarioRun.Status.ClusterJobs) > 0 {
		firstJob := scenarioRun.Status.ClusterJobs[0]
		startTime = firstJob.StartTime
		completionTime = firstJob.CompletionTime
	}

	return r.setNodeStatus(graphRun, nodeID, phase, scenarioRun.Name, startTime, completionTime)
}

// updateNodeStatus updates a node's status in the graph run
// setNodeStatus updates a node's status in memory without persisting to the API server.
// Returns true if the status was modified, false if unchanged.
func (r *KrknGraphRunReconciler) setNodeStatus(
	graphRun *krknv1alpha1.KrknGraphRun,
	nodeID string,
	phase string,
	scenarioRunRef string,
	startTime, completionTime *metav1.Time,
) (bool, error) {
	for i := range graphRun.Status.NodeStatuses {
		if graphRun.Status.NodeStatuses[i].NodeID == nodeID {
			modified := false

			if graphRun.Status.NodeStatuses[i].Phase != phase {
				graphRun.Status.NodeStatuses[i].Phase = phase
				modified = true
			}

			if graphRun.Status.NodeStatuses[i].ScenarioRunRef != scenarioRunRef {
				graphRun.Status.NodeStatuses[i].ScenarioRunRef = scenarioRunRef
				modified = true
			}

			if startTime != nil && (graphRun.Status.NodeStatuses[i].StartTime == nil ||
				!graphRun.Status.NodeStatuses[i].StartTime.Equal(startTime)) {
				graphRun.Status.NodeStatuses[i].StartTime = startTime
				modified = true
			}

			if completionTime != nil && (graphRun.Status.NodeStatuses[i].CompletionTime == nil ||
				!graphRun.Status.NodeStatuses[i].CompletionTime.Equal(completionTime)) {
				graphRun.Status.NodeStatuses[i].CompletionTime = completionTime
				modified = true
			}

			return modified, nil
		}
	}
	return false, fmt.Errorf("node status not found for %s", nodeID)
}

// handleFailFast marks dependent nodes as blocked when fail-fast is triggered.
// Returns true to indicate it has already updated the status.
func (r *KrknGraphRunReconciler) handleFailFast(
	ctx context.Context,
	graphRun *krknv1alpha1.KrknGraphRun,
	fromLevel int,
) (ctrl.Result, bool, error) {
	logger := log.FromContext(ctx)
	logger.Info("fail-fast triggered, marking dependent nodes as blocked",
		"graphRun", graphRun.Name,
		"fromLevel", fromLevel)

	// Mark all nodes in subsequent levels as blocked (batch in memory)
	for levelIdx := fromLevel; levelIdx < len(graphRun.Status.ResolvedLevels); levelIdx++ {
		for _, nodeID := range graphRun.Status.ResolvedLevels[levelIdx] {
			if _, err := r.setNodeStatus(graphRun, nodeID, "Blocked", "", nil, nil); err != nil {
				logger.Error(err, "failed to set node status to blocked during fail-fast",
					"nodeID", nodeID, "graphRun", graphRun.Name)
			}
		}
	}

	// Calculate global status before updating
	r.calculateGlobalStatus(graphRun)

	if err := r.Status().Update(ctx, graphRun); err != nil {
		if apierrors.IsConflict(err) {
			logger.Info("conflict in handleFailFast, will retry on next reconcile")
			return ctrl.Result{Requeue: true}, true, nil
		}
		return ctrl.Result{}, true, err
	}

	return ctrl.Result{}, true, nil
}

// calculateGlobalStatus calculates the global phase and summary based on node statuses.
// Updates the graphRun status in memory without persisting to the API server.
func (r *KrknGraphRunReconciler) calculateGlobalStatus(graphRun *krknv1alpha1.KrknGraphRun) {
	// Count nodes from NodeStatuses, not Spec.Graph (which may include filtered metadata nodes)
	totalNodes := len(graphRun.Status.NodeStatuses)
	completedNodes := 0
	runningNodes := 0
	failedNodes := 0
	pendingNodes := 0

	for _, nodeStatus := range graphRun.Status.NodeStatuses {
		switch nodeStatus.Phase {
		case "Completed":
			completedNodes++
		case "Running":
			runningNodes++
		case "Failed":
			failedNodes++
		case "Pending", "Blocked":
			pendingNodes++
		}
	}

	// Calculate global phase
	phase := "Pending"
	if failedNodes > 0 {
		if completedNodes > 0 {
			phase = "PartiallyFailed"
		} else {
			phase = "Failed"
		}
	} else if completedNodes == totalNodes {
		phase = "Completed"
		if graphRun.Status.CompletionTime == nil {
			graphRun.Status.CompletionTime = &metav1.Time{Time: time.Now()}
		}
	} else if runningNodes > 0 || completedNodes > 0 {
		// If any work has started (running or completed), we're in "Running" phase
		phase = "Running"
	}

	graphRun.Status.Phase = phase
	graphRun.Status.Summary = krknv1alpha1.GraphRunSummary{
		TotalNodes:     totalNodes,
		CompletedNodes: completedNodes,
		RunningNodes:   runningNodes,
		FailedNodes:    failedNodes,
		PendingNodes:   pendingNodes,
	}
}

// updateStatusWithError updates status with error and returns result
func (r *KrknGraphRunReconciler) updateStatusWithError(
	ctx context.Context,
	graphRun *krknv1alpha1.KrknGraphRun,
	err error,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	graphRun.Status.Phase = "Failed"
	if updateErr := r.Status().Update(ctx, graphRun); updateErr != nil {
		logger.Error(updateErr, "failed to update status to Failed",
			"graphRun", graphRun.Name, "originalError", err)
		// Return the original error even if status update fails
	}
	return ctrl.Result{}, err
}

// isTerminalPhase returns true if the phase indicates the GraphRun has completed execution
func (r *KrknGraphRunReconciler) isTerminalPhase(phase string) bool {
	return phase == "Completed" || phase == "Failed" || phase == "PartiallyFailed"
}

// translateVolumesToFileMounts translates GraphScenarioNode.Volumes (UUID->mountPath) to FileMount objects.
// Volumes format: {"<file-uuid>": "/mount/path"}
// This function loads file ConfigMaps by UUID and encodes content as base64 for FileMount.
func (r *KrknGraphRunReconciler) translateVolumesToFileMounts(
	ctx context.Context,
	volumes map[string]string,
) ([]krknv1alpha1.FileMount, error) {
	logger := log.FromContext(ctx)

	if len(volumes) == 0 {
		return nil, nil
	}

	fileMounts := make([]krknv1alpha1.FileMount, 0, len(volumes))

	for fileID, mountPath := range volumes {
		// Validate mount path is absolute
		if !filepath.IsAbs(mountPath) {
			return nil, fmt.Errorf("mount path '%s' for file '%s' must be absolute", mountPath, fileID)
		}

		// Load file ConfigMap by UUID
		fileConfigMap, err := r.loadFileConfigMapByID(ctx, fileID)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("file with ID '%s' not found", fileID)
			}
			return nil, fmt.Errorf("failed to load file ConfigMap '%s': %w", fileID, err)
		}

		// Extract file content from ConfigMap (first data key)
		var fileName, content string
		for k, v := range fileConfigMap.Data {
			fileName = k
			content = v
			break
		}

		if fileName == "" {
			return nil, fmt.Errorf("file ConfigMap '%s' has no data", fileID)
		}

		// Base64 encode content for FileMount
		encodedContent := base64.StdEncoding.EncodeToString([]byte(content))

		// Create FileMount
		fileMounts = append(fileMounts, krknv1alpha1.FileMount{
			Name:      fileName,
			Content:   encodedContent,
			MountPath: mountPath,
		})

		logger.V(1).Info("translated volume to file mount",
			"fileID", fileID,
			"fileName", fileName,
			"mountPath", mountPath)
	}

	return fileMounts, nil
}

// loadFileConfigMapByID loads a file ConfigMap by file ID (UUID) using label selector.
// Returns error if not found or if multiple ConfigMaps have the same file ID.
func (r *KrknGraphRunReconciler) loadFileConfigMapByID(
	ctx context.Context,
	fileID string,
) (*corev1.ConfigMap, error) {
	// List ConfigMaps with the file ID label in the operator's namespace
	var configMapList corev1.ConfigMapList
	err := r.List(ctx, &configMapList,
		client.InNamespace(r.Namespace),
		client.MatchingLabels{
			"app.kubernetes.io/name":            "krkn-operator",
			"app.kubernetes.io/component":       "file",
			"files.krkn.krkn-chaos.dev/file-id": fileID,
		},
	)

	if err != nil {
		return nil, err
	}

	if len(configMapList.Items) == 0 {
		return nil, apierrors.NewNotFound(corev1.Resource("configmap"), fileID)
	}

	if len(configMapList.Items) > 1 {
		return nil, fmt.Errorf("multiple ConfigMaps found with file ID '%s'", fileID)
	}

	return &configMapList.Items[0], nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KrknGraphRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&krknv1alpha1.KrknGraphRun{}).
		Owns(&krknv1alpha1.KrknScenarioRun{}).
		Complete(r)
}
