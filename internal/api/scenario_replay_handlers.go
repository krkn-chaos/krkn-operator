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

package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

// GetScenarioReplay handles GET /api/v1/scenarios/run/replay/{jobId}
// Retrieves a completed scenario job and reconstructs the payload for replay
//
// @Summary Replay a scenario from a completed job
// @Description Retrieve scenario configuration from a completed job and return payload ready for re-execution via POST /scenarios/run
// @Tags scenarios
// @Produce json
// @Param jobId path string true "Job ID (krkn-job-id label value)"
// @Success 200 {object} ScenarioRunRequest "Scenario configuration ready for replay"
// @Failure 400 {object} ErrorResponse "Invalid job ID or job not found"
// @Failure 403 {object} ErrorResponse "Insufficient permissions"
// @Failure 404 {object} ErrorResponse "Job or ScenarioRun not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /scenarios/run/replay/{jobId} [get]
func (h *Handler) GetScenarioReplay(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	// Extract jobId from URL path
	// Path format: /api/v1/scenarios/run/replay/{jobId}
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/scenarios/run/replay/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Job ID is required",
		})
		return
	}
	jobID := pathParts[0]

	logger.V(1).Info("Replay request received", "jobId", jobID)

	// Step 1: Find pod by label krkn-job-id={jobId}
	pod, err := h.findPodByJobID(ctx, jobID)
	if err != nil {
		logger.Error(err, "Failed to find pod by job ID", "jobId", jobID)
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: fmt.Sprintf("No pod found with job ID '%s'", jobID),
		})
		return
	}

	logger.V(1).Info("Found pod for job", "jobId", jobID, "podName", pod.Name)

	// Step 2: Extract KrknScenarioRun name from pod ownerReferences
	scenarioRunName, err := h.extractScenarioRunNameFromPod(pod)
	if err != nil {
		logger.Error(err, "Failed to extract ScenarioRun name from pod", "podName", pod.Name)
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: fmt.Sprintf("Pod '%s' does not have a KrknScenarioRun owner reference", pod.Name),
		})
		return
	}

	logger.V(1).Info("Extracted ScenarioRun name", "scenarioRunName", scenarioRunName)

	// Step 3: Read KrknScenarioRun CR
	scenarioRun, err := h.getScenarioRun(ctx, scenarioRunName)
	if err != nil {
		logger.Error(err, "Failed to get ScenarioRun", "scenarioRunName", scenarioRunName)
		if client.IgnoreNotFound(err) == nil {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("ScenarioRun '%s' not found", scenarioRunName),
			})
		} else {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to retrieve ScenarioRun",
			})
		}
		return
	}

	logger.V(1).Info("Retrieved ScenarioRun", "scenarioRunName", scenarioRunName)

	// Step 4: Validate RBAC permissions
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User authentication required",
		})
		return
	}

	// Admin bypass
	if !auth.IsAdmin(ctx) {
		// Regular user: must have 'run' permission on target clusters
		// Fetch KrknTargetRequest to get cluster details
		targetRequest := &krknv1alpha1.KrknTargetRequest{}
		if err := h.client.Get(ctx, types.NamespacedName{
			Name:      scenarioRun.Spec.TargetRequestID,
			Namespace: h.namespace,
		}, targetRequest); err != nil {
			logger.Error(err, "Failed to fetch target request", "targetRequestId", scenarioRun.Spec.TargetRequestID)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to validate permissions",
			})
			return
		}

		// Validate user has 'run' permission on all target clusters
		if err := groupauth.ValidateScenarioRunAccess(
			ctx,
			h.client,
			claims.UserID,
			h.namespace,
			scenarioRun.Spec.TargetClusters,
			targetRequest,
		); err != nil {
			logger.Info("User lacks permission to replay scenario",
				"userID", claims.UserID,
				"error", err.Error(),
			)
			writeJSONError(w, http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: err.Error(),
			})
			return
		}
	}

	logger.V(1).Info("RBAC validation passed", "userID", claims.UserID)

	// Step 5: Reconstruct payload for POST /scenarios/run
	payload, err := h.reconstructScenarioRunPayload(ctx, scenarioRun)
	if err != nil {
		logger.Error(err, "Failed to reconstruct scenario payload", "scenarioRunName", scenarioRunName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to reconstruct scenario payload",
		})
		return
	}

	logger.Info("Scenario replay payload reconstructed successfully",
		"jobId", jobID,
		"scenarioRunName", scenarioRunName,
		"userID", claims.UserID)

	// Step 6: Return payload (identical to wizard output)
	writeJSON(w, http.StatusOK, payload)
}

// findPodByJobID finds a pod with the label krkn-job-id={jobId}
func (h *Handler) findPodByJobID(ctx context.Context, jobID string) (*corev1.Pod, error) {
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(h.namespace),
		client.MatchingLabels{"krkn-job-id": jobID},
	}

	if err := h.client.List(ctx, podList, listOpts...); err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("no pod found with krkn-job-id=%s", jobID)
	}

	if len(podList.Items) > 1 {
		return nil, fmt.Errorf("multiple pods found with krkn-job-id=%s (found %d)", jobID, len(podList.Items))
	}

	return &podList.Items[0], nil
}

// extractScenarioRunNameFromPod extracts the KrknScenarioRun name from pod's ownerReferences
func (h *Handler) extractScenarioRunNameFromPod(pod *corev1.Pod) (string, error) {
	for _, ownerRef := range pod.OwnerReferences {
		if ownerRef.Kind == "KrknScenarioRun" {
			return ownerRef.Name, nil
		}
	}
	return "", fmt.Errorf("no KrknScenarioRun owner reference found in pod")
}

// getScenarioRun retrieves a KrknScenarioRun CR by name
func (h *Handler) getScenarioRun(ctx context.Context, name string) (*krknv1alpha1.KrknScenarioRun, error) {
	scenarioRun := &krknv1alpha1.KrknScenarioRun{}
	err := h.client.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: h.namespace,
	}, scenarioRun)
	return scenarioRun, err
}

// reconstructScenarioRunPayload reconstructs a ScenarioRunRequest from a KrknScenarioRun CR
// The payload is 100% identical to what the wizard would send to POST /scenarios/run
func (h *Handler) reconstructScenarioRunPayload(ctx context.Context, scenarioRun *krknv1alpha1.KrknScenarioRun) (*ScenarioRunRequest, error) {
	logger := log.FromContext(ctx)

	payload := &ScenarioRunRequest{
		// Mandatory fields
		TargetRequestID: scenarioRun.Spec.TargetRequestID,
		TargetClusters:  scenarioRun.Spec.TargetClusters,
		ScenarioName:    scenarioRun.Spec.ScenarioName,
		ScenarioImage:   scenarioRun.Spec.ScenarioImage,

		// Optional fields
		Environment:    scenarioRun.Spec.Environment,
		KubeconfigPath: scenarioRun.Spec.KubeconfigPath,

		// Note: MaxRetries, RetryBackoff, RetryDelay are NOT in ScenarioRunRequest
		// They are controller-level settings, not wizard-level inputs
	}

	// Reconstruct FileReferences from saved files
	// Use fileReferences (with UUIDs) when available, fallback to inline files otherwise
	if len(scenarioRun.Spec.Files) > 0 {
		fileReferences := make([]files.FileReference, 0)
		inlineFiles := make([]FileMount, 0)

		for _, f := range scenarioRun.Spec.Files {
			if f.FileID != "" {
				// This file came from a fileReference - reconstruct it
				fileReferences = append(fileReferences, files.FileReference{
					FileID:    f.FileID,
					MountPath: f.MountPath,
				})
				logger.V(1).Info("Reconstructed file reference",
					"fileId", f.FileID,
					"mountPath", f.MountPath)
			} else {
				// This file was inline - keep it as inline
				inlineFiles = append(inlineFiles, FileMount{
					Name:      f.Name,
					Content:   f.Content,
					MountPath: f.MountPath,
				})
				logger.V(1).Info("Preserved inline file",
					"name", f.Name,
					"mountPath", f.MountPath)
			}
		}

		payload.FileReferences = fileReferences
		payload.Files = inlineFiles
	}

	// Reconstruct registry configuration
	// ScenarioRunRequest only has RegistryName (embedded from ScenariosRequest)
	// Explicit credentials (RegistryURL, Token, etc.) are NOT in the wizard payload
	// They are loaded by the controller from the RegistryName Secret
	if scenarioRun.Spec.RegistryName != "" {
		payload.RegistryName = &scenarioRun.Spec.RegistryName
	}
	// Note: Even if the original run used explicit registry credentials,
	// we cannot reconstruct them in the payload because:
	// 1. ScenarioRunRequest doesn't have those fields
	// 2. They come from controller-level configuration, not wizard input

	return payload, nil
}
