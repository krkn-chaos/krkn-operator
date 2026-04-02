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

package groupauth

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// ValidateScenarioRunAccess validates that a user has permission to run scenarios
// on all specified target clusters.
//
// This function:
// 1. Fetches the user's groups
// 2. Aggregates permissions from all groups
// 3. Validates the user has 'run' permission on each target cluster
//
// Parameters:
//   - ctx: Context for the request
//   - k8sClient: Kubernetes client for API calls
//   - userID: Email address of the user
//   - namespace: Namespace where CRs are located
//   - targetClusters: Map of provider -> cluster names to validate
//   - targetRequest: The KrknTargetRequest containing cluster API URLs
//
// Returns nil if validation passes, or an error describing the permission violation.
func ValidateScenarioRunAccess(
	ctx context.Context,
	k8sClient client.Client,
	userID string,
	namespace string,
	targetClusters map[string][]string,
	targetRequest *krknv1alpha1.KrknTargetRequest,
) error {
	logger := log.FromContext(ctx).WithName("groupauth.ValidateScenarioRunAccess")

	// 1. Fetch user groups
	userGroups, err := GetUserGroups(ctx, k8sClient, userID, namespace)
	if err != nil {
		return fmt.Errorf("failed to fetch user groups: %w", err)
	}

	if len(userGroups) == 0 {
		return fmt.Errorf("user %s does not belong to any groups and has no cluster access", userID)
	}

	// 2. Aggregate permissions once (optimization: avoid re-aggregating for each cluster)
	aggregatedPermissions := AggregateClusterPermissions(userGroups)

	// 3. Build clusterName -> apiURL mapping from TargetRequest
	clusterAPIURLMap := buildClusterAPIURLMap(targetRequest)

	// 4. Validate each target cluster
	for provider, clusterNames := range targetClusters {
		for _, clusterName := range clusterNames {
			apiURL, exists := clusterAPIURLMap[clusterName]
			if !exists {
				return fmt.Errorf("cluster %s not found in target request (provider: %s)", clusterName, provider)
			}

			// Check if user has 'run' permission on this cluster
			if !hasAction(aggregatedPermissions[apiURL], ActionRun) {
				logger.V(1).Info("Permission denied",
					"userID", userID,
					"clusterName", clusterName,
					"clusterAPIURL", apiURL,
					"requiredAction", ActionRun,
				)
				return fmt.Errorf("user %s does not have permission to run scenarios on cluster %s (API URL: %s)", userID, clusterName, apiURL)
			}
		}
	}

	logger.V(1).Info("Scenario run access validated",
		"userID", userID,
		"clusterCount", countClusters(targetClusters),
	)
	return nil
}

// FilterClustersByPermission filters clusters based on user permissions.
// Only returns clusters the user has the specified action permission for.
//
// Parameters:
//   - ctx: Context for the request
//   - k8sClient: Kubernetes client
//   - userID: Email address of the user
//   - namespace: Namespace where CRs are located
//   - targetData: Map of provider -> cluster targets from KrknTargetRequest
//   - requiredAction: The action required (typically ActionView for GET /clusters)
//
// Returns filtered targetData containing only permitted clusters.
func FilterClustersByPermission(
	ctx context.Context,
	k8sClient client.Client,
	userID string,
	namespace string,
	targetData map[string][]krknv1alpha1.ClusterTarget,
	requiredAction Action,
) (map[string][]krknv1alpha1.ClusterTarget, error) {
	logger := log.FromContext(ctx).WithName("groupauth.FilterClustersByPermission")

	// Fetch user groups
	userGroups, err := GetUserGroups(ctx, k8sClient, userID, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user groups: %w", err)
	}

	if len(userGroups) == 0 {
		// No groups = no cluster access for regular users
		logger.V(1).Info("User has no group memberships, filtering all clusters", "userID", userID)
		return map[string][]krknv1alpha1.ClusterTarget{}, nil
	}

	// Aggregate permissions once (optimization: avoid re-aggregating for each cluster)
	aggregatedPermissions := AggregateClusterPermissions(userGroups)

	// Filter clusters by permission
	filtered := make(map[string][]krknv1alpha1.ClusterTarget)

	for provider, clusters := range targetData {
		allowedClusters := make([]krknv1alpha1.ClusterTarget, 0)

		for _, cluster := range clusters {
			if hasAction(aggregatedPermissions[cluster.ClusterAPIURL], requiredAction) {
				allowedClusters = append(allowedClusters, cluster)
			}
		}

		if len(allowedClusters) > 0 {
			filtered[provider] = allowedClusters
		}
	}

	logger.V(1).Info("Filtered clusters by permission",
		"userID", userID,
		"requiredAction", requiredAction,
		"originalCount", countClustersFromTargetData(targetData),
		"filteredCount", countClustersFromTargetData(filtered),
	)

	return filtered, nil
}

// buildClusterAPIURLMap builds a map from cluster name to API URL
func buildClusterAPIURLMap(targetRequest *krknv1alpha1.KrknTargetRequest) map[string]string {
	result := make(map[string]string)

	for _, targets := range targetRequest.Status.TargetData {
		for _, cluster := range targets {
			result[cluster.ClusterName] = cluster.ClusterAPIURL
		}
	}

	return result
}

// countClusters counts total clusters in targetClusters map
func countClusters(targetClusters map[string][]string) int {
	total := 0
	for _, clusters := range targetClusters {
		total += len(clusters)
	}
	return total
}

// countClustersFromTargetData counts total clusters in targetData map
func countClustersFromTargetData(targetData map[string][]krknv1alpha1.ClusterTarget) int {
	total := 0
	for _, clusters := range targetData {
		total += len(clusters)
	}
	return total
}

// hasAction checks if an action exists in the given slice of actions
func hasAction(actions []Action, requiredAction Action) bool {
	for _, action := range actions {
		if action == requiredAction {
			return true
		}
	}
	return false
}
