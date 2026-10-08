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
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/kubeconfig"
)

var errAmbiguousClusterProvider = errors.New("operator_name is required to disambiguate a cluster found in multiple providers")

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
// This supports all providers represented in the target request.
// Returns base64-encoded kubeconfig string
func (h *Handler) getKubeconfigFromTargetRequest(ctx context.Context, targetID string, operatorName string, clusterName string) (string, error) {
	// Fetch the secret with the same name as the KrknTargetRequest ID
	var secret corev1.Secret
	err := h.client.Get(ctx, types.NamespacedName{
		Name:      targetID,
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

	var managedClusters map[string]map[string]struct {
		Kubeconfig string `json:"kubeconfig"`
	}
	if err := json.Unmarshal(managedClustersBytes, &managedClusters); err != nil {
		return "", fmt.Errorf("failed to parse managed-clusters JSON: %w", err)
	}

	if operatorName != "" {
		providerClusters, exists := managedClusters[operatorName]
		if !exists {
			return "", fmt.Errorf("provider '%s' not found in managed-clusters", operatorName)
		}
		clusterConfig, exists := providerClusters[clusterName]
		if !exists {
			return "", fmt.Errorf("cluster '%s' not found in provider '%s'", clusterName, operatorName)
		}
		return clusterConfig.Kubeconfig, nil
	}

	var matchedProvider string
	var kubeconfigBase64 string
	for provider, clusters := range managedClusters {
		if clusterConfig, exists := clusters[clusterName]; exists {
			if matchedProvider != "" {
				return "", fmt.Errorf("%w: cluster '%s' exists in providers '%s' and '%s'", errAmbiguousClusterProvider, clusterName, matchedProvider, provider)
			}
			matchedProvider = provider
			kubeconfigBase64 = clusterConfig.Kubeconfig
		}
	}
	if matchedProvider == "" {
		return "", fmt.Errorf("cluster '%s' not found in managed-clusters", clusterName)
	}

	return kubeconfigBase64, nil
}

// getClusterAPIURL retrieves the cluster API URL from either:
// 1. KrknOperatorTarget (new system) - if targetUUID is provided
// 2. KrknTargetRequest (legacy) - if targetID and clusterName are provided
//
// Returns cluster API URL string for permission checks
func (h *Handler) getClusterAPIURL(ctx context.Context, targetUUID string, targetID string, operatorName string, clusterName string) (string, error) {
	// Try new system first (KrknOperatorTarget)
	if targetUUID != "" {
		var target krknv1alpha1.KrknOperatorTarget
		if err := h.client.Get(ctx, types.NamespacedName{
			Name:      targetUUID,
			Namespace: h.namespace,
		}, &target); err != nil {
			// If KrknOperatorTarget not found but we have legacy params, try legacy
			if targetID == "" || clusterName == "" {
				return "", fmt.Errorf("failed to fetch KrknOperatorTarget: %w", err)
			}
		} else {
			return target.Spec.ClusterAPIURL, nil
		}
	}

	// Fall back to legacy system (KrknTargetRequest)
	if targetID != "" && clusterName != "" {
		var targetRequest krknv1alpha1.KrknTargetRequest
		if err := h.client.Get(ctx, types.NamespacedName{
			Name:      targetID,
			Namespace: h.namespace,
		}, &targetRequest); err != nil {
			return "", fmt.Errorf("failed to fetch KrknTargetRequest: %w", err)
		}

		var matchedProvider string
		var matchedCluster *krknv1alpha1.ClusterTarget
		for provider, targets := range targetRequest.Status.TargetData {
			if operatorName != "" && provider != operatorName {
				continue
			}
			for _, cluster := range targets {
				if cluster.ClusterName != clusterName {
					continue
				}
				if matchedCluster != nil {
					if operatorName == "" {
						return "", fmt.Errorf("%w: cluster '%s' exists in providers '%s' and '%s'", errAmbiguousClusterProvider, clusterName, matchedProvider, provider)
					}
					return "", fmt.Errorf("cluster '%s' appears multiple times in provider '%s'", clusterName, operatorName)
				}
				clusterCopy := cluster
				matchedCluster = &clusterCopy
				matchedProvider = provider
			}
		}
		if matchedCluster == nil {
			if operatorName != "" {
				return "", fmt.Errorf("cluster '%s' not found in provider '%s' in target request", clusterName, operatorName)
			}
			return "", fmt.Errorf("cluster '%s' not found in target request", clusterName)
		}

		return matchedCluster.ClusterAPIURL, nil
	}

	return "", fmt.Errorf("insufficient parameters: provide either targetUUID (new) or targetID+clusterName (legacy)")
}

// getKubeconfig is a unified function that tries to get kubeconfig from either:
// 1. KrknOperatorTarget (new system) - if only targetUUID is provided
// 2. KrknTargetRequest (legacy) - if targetID and clusterName are provided
//
// Returns base64-encoded kubeconfig string
func (h *Handler) getKubeconfig(ctx context.Context, targetUUID string, targetID string, operatorName string, clusterName string) (string, error) {
	// Try new system first (KrknOperatorTarget)
	if targetUUID != "" {
		kubeconfigBase64, err := h.getKubeconfigFromOperatorTarget(ctx, targetUUID)
		if err == nil {
			return kubeconfigBase64, nil
		}
		// If KrknOperatorTarget not found but we have legacy params, try legacy
		if targetID == "" || clusterName == "" {
			return "", err
		}
	}

	// Fall back to legacy system (KrknTargetRequest)
	if targetID != "" && clusterName != "" {
		return h.getKubeconfigFromTargetRequest(ctx, targetID, operatorName, clusterName)
	}

	return "", fmt.Errorf("insufficient parameters: provide either targetUUID (new) or targetID+clusterName (legacy)")
}
