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
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/kubeconfig"
)

// getKubeconfigFromOperatorTarget retrieves kubeconfig from KrknOperatorTarget
// Returns base64-encoded kubeconfig string
func (h *Handler) getKubeconfigFromOperatorTarget(ctx context.Context, targetUUID string) (string, error) {
	// Fetch KrknOperatorTarget
	var target krknv1alpha1.KrknOperatorTarget
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      targetUUID,
		Namespace: h.namespace,
	}, &target); err != nil {
		return "", fmt.Errorf("failed to fetch KrknOperatorTarget: %w", err)
	}

	// Fetch Secret
	var secret corev1.Secret
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      target.Spec.SecretUUID,
		Namespace: h.namespace,
	}, &secret); err != nil {
		return "", fmt.Errorf("failed to fetch secret: %w", err)
	}

	// Extract kubeconfig from secret data
	kubeconfigData, exists := secret.Data["kubeconfig"]
	if !exists {
		return "", fmt.Errorf("kubeconfig not found in secret")
	}

	// Unmarshal JSON to get base64-encoded kubeconfig
	kubeconfigBase64, err := kubeconfig.UnmarshalSecretData(kubeconfigData)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal kubeconfig from secret: %w", err)
	}

	return kubeconfigBase64, nil
}

// getKubeconfigFromTargetRequest retrieves kubeconfig from KrknTargetRequest (legacy)
// This is for backward compatibility with the old krkn-operator-acm flow
// Returns base64-encoded kubeconfig string
func (h *Handler) getKubeconfigFromTargetRequest(ctx context.Context, targetId string, clusterName string) (string, error) {
	// Fetch the secret with the same name as the KrknTargetRequest ID
	var secret corev1.Secret
	err := h.client.Get(ctx, types.NamespacedName{
		Name:      targetId,
		Namespace: h.namespace,
	}, &secret)

	if err != nil {
		return "", fmt.Errorf("failed to fetch secret: %w", err)
	}

	// Retrieve the managed-clusters JSON from the secret data
	managedClustersBytes, exists := secret.Data["managed-clusters"]
	if !exists {
		return "", fmt.Errorf("managed-clusters not found in secret")
	}

	// Parse the JSON to extract cluster configurations
	// Structure: { "krkn-operator-acm": { "cluster-name": { "kubeconfig": "base64..." } } }
	var managedClusters map[string]map[string]struct {
		Kubeconfig string `json:"kubeconfig"`
	}
	if err := json.Unmarshal(managedClustersBytes, &managedClusters); err != nil {
		return "", fmt.Errorf("failed to parse managed-clusters JSON: %w", err)
	}

	// Get the krkn-operator-acm object
	acmClusters, exists := managedClusters["krkn-operator-acm"]
	if !exists {
		return "", fmt.Errorf("krkn-operator-acm not found in managed-clusters")
	}

	// Check if the requested cluster exists
	clusterConfig, exists := acmClusters[clusterName]
	if !exists {
		return "", fmt.Errorf("cluster '%s' not found in krkn-operator-acm", clusterName)
	}

	// Return the base64-encoded kubeconfig
	return clusterConfig.Kubeconfig, nil
}

// getKubeconfig is a unified function that tries to get kubeconfig from either:
// 1. KrknOperatorTarget (new system) - if only targetUUID is provided
// 2. KrknTargetRequest (legacy) - if targetId and clusterName are provided
//
// Returns base64-encoded kubeconfig string
func (h *Handler) getKubeconfig(ctx context.Context, targetUUID string, targetId string, clusterName string) (string, error) {
	// Try new system first (KrknOperatorTarget)
	if targetUUID != "" {
		kubeconfigBase64, err := h.getKubeconfigFromOperatorTarget(ctx, targetUUID)
		if err == nil {
			return kubeconfigBase64, nil
		}
		// If KrknOperatorTarget not found but we have legacy params, try legacy
		if targetId == "" || clusterName == "" {
			return "", err
		}
	}

	// Fall back to legacy system (KrknTargetRequest)
	if targetId != "" && clusterName != "" {
		return h.getKubeconfigFromTargetRequest(ctx, targetId, clusterName)
	}

	return "", fmt.Errorf("insufficient parameters: provide either targetUUID (new) or targetId+clusterName (legacy)")
}
