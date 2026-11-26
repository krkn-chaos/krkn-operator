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
	"sort"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/kubeconfig"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var errAmbiguousClusterProvider = errors.New("operator_name is required to disambiguate a cluster found in multiple providers")

// getKubeconfigFromOperatorTarget retrieves kubeconfig from KrknOperatorTarget
// Returns base64-encoded kubeconfig string
func (h *Handler) getKubeconfigFromOperatorTarget(ctx context.Context, targetUUID string) (string, error) {
	// Fetch KrknOperatorTarget
	var target krknv1alpha1.KrknOperatorTarget
	err := h.client.Get(ctx, types.NamespacedName{
		Name:      targetUUID,
		Namespace: h.namespace,
	}, &target)
	if apierrors.IsNotFound(err) {
		// VisualizeInstallCard submits the displayed clusterName, while other
		// API callers submit the target UUID. Accept either identifier so both
		// the selected cluster and older visualize records keep resolving.
		var targets krknv1alpha1.KrknOperatorTargetList
		if listErr := h.client.List(ctx, &targets, client.InNamespace(h.namespace)); listErr != nil {
			return "", fmt.Errorf("failed to list KrknOperatorTargets: %w", listErr)
		}
		matches := make([]*krknv1alpha1.KrknOperatorTarget, 0, 1)
		for i := range targets.Items {
			candidate := &targets.Items[i]
			if candidate.Spec.ClusterName == targetUUID || candidate.Spec.UUID == targetUUID {
				matches = append(matches, candidate)
			}
		}
		switch len(matches) {
		case 1:
			target = *matches[0]
		case 0:
			return "", fmt.Errorf("KrknOperatorTarget with UUID or cluster name %q not found", targetUUID)
		default:
			return "", fmt.Errorf("multiple KrknOperatorTargets match UUID or cluster name %q", targetUUID)
		}
	} else if err != nil {
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

// resolveTargetRequestCluster selects matching legacy status entries used for
// authorization and kubeconfig lookup. When operatorName is empty, repeated
// entries are accepted only when they identify the same API URL.
func (h *Handler) resolveTargetRequestCluster(ctx context.Context, targetID, operatorName, clusterName string) ([]string, krknv1alpha1.ClusterTarget, error) {
	var targetRequest krknv1alpha1.KrknTargetRequest
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      targetID,
		Namespace: h.namespace,
	}, &targetRequest); err != nil {
		return nil, krknv1alpha1.ClusterTarget{}, fmt.Errorf("failed to fetch KrknTargetRequest: %w", err)
	}

	var matchedCluster *krknv1alpha1.ClusterTarget
	providers := make(map[string]struct{})
	for provider, targets := range targetRequest.Status.TargetData {
		if operatorName != "" && provider != operatorName {
			continue
		}
		for _, cluster := range targets {
			if cluster.ClusterName != clusterName {
				continue
			}
			if matchedCluster != nil {
				if cluster.ClusterAPIURL != matchedCluster.ClusterAPIURL {
					if operatorName == "" {
						return nil, krknv1alpha1.ClusterTarget{}, fmt.Errorf("%w: cluster '%s' has different API URLs in target status", errAmbiguousClusterProvider, clusterName)
					}
					return nil, krknv1alpha1.ClusterTarget{}, fmt.Errorf("cluster '%s' appears multiple times in provider '%s' with different API URLs", clusterName, operatorName)
				}
			} else {
				clusterCopy := cluster
				matchedCluster = &clusterCopy
			}
			providers[provider] = struct{}{}
		}
	}
	if matchedCluster == nil {
		if operatorName != "" {
			return nil, krknv1alpha1.ClusterTarget{}, fmt.Errorf("cluster '%s' not found in provider '%s' in target request", clusterName, operatorName)
		}
		return nil, krknv1alpha1.ClusterTarget{}, fmt.Errorf("cluster '%s' not found in target request", clusterName)
	}

	matchedProviders := make([]string, 0, len(providers))
	for provider := range providers {
		matchedProviders = append(matchedProviders, provider)
	}
	sort.Strings(matchedProviders)
	return matchedProviders, *matchedCluster, nil
}

// getKubeconfigFromTargetRequest retrieves a kubeconfig only when every matching
// provider entry and API URL in the managed-clusters Secret agree with target
// status. Duplicate entries must carry identical kubeconfigs.
func (h *Handler) getKubeconfigFromTargetRequest(ctx context.Context, targetID string, operatorName string, clusterName string) (string, error) {
	providers, targetCluster, err := h.resolveTargetRequestCluster(ctx, targetID, operatorName, clusterName)
	if err != nil {
		return "", err
	}

	var secret corev1.Secret
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      targetID,
		Namespace: h.namespace,
	}, &secret); err != nil {
		return "", fmt.Errorf("failed to fetch secret: %w", err)
	}

	managedClustersBytes, exists := secret.Data["managed-clusters"]
	if !exists {
		return "", fmt.Errorf("managed-clusters not found in secret")
	}

	var managedClusters map[string]map[string]struct {
		ClusterAPIURL *string `json:"cluster-api"`
		Kubeconfig    string  `json:"kubeconfig"`
	}
	if err := json.Unmarshal(managedClustersBytes, &managedClusters); err != nil {
		return "", fmt.Errorf("failed to parse managed-clusters JSON: %w", err)
	}

	var kubeconfigBase64 string
	// A cluster name must resolve to one credential across operator keys.
	// Reject duplicates with different API URLs or kubeconfigs so this lookup
	// cannot select the wrong credential; callers must resolve those conflicts.
	for i, provider := range providers {
		providerClusters, exists := managedClusters[provider]
		if !exists {
			return "", fmt.Errorf("provider '%s' not found in managed-clusters", provider)
		}
		clusterConfig, exists := providerClusters[clusterName]
		if !exists {
			return "", fmt.Errorf("cluster '%s' not found in provider '%s'", clusterName, provider)
		}
		if clusterConfig.ClusterAPIURL == nil || *clusterConfig.ClusterAPIURL != targetCluster.ClusterAPIURL {
			return "", fmt.Errorf("cluster '%s' API URL mismatch in provider '%s' managed-clusters", clusterName, provider)
		}
		if i == 0 {
			kubeconfigBase64 = clusterConfig.Kubeconfig
			continue
		}
		if clusterConfig.Kubeconfig != kubeconfigBase64 {
			return "", fmt.Errorf("cluster '%s' has conflicting kubeconfigs across providers", clusterName)
		}
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
		_, cluster, err := h.resolveTargetRequestCluster(ctx, targetID, operatorName, clusterName)
		if err != nil {
			return "", err
		}
		return cluster.ClusterAPIURL, nil
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
