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
	"regexp"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

const (
	// resourceNamePrefix is prepended to sanitized user IDs when building
	// KrknUser resource names, keeping them in a dedicated namespace and
	// guaranteeing the name starts with an alphanumeric character.
	resourceNamePrefix = "krknuser-"

	// maxLabelValueLength is the maximum length of a Kubernetes label value
	// (RFC 1123). See k8s.io/apimachinery/pkg/util/validation.
	maxLabelValueLength = 63

	// maxResourceNameLength is the maximum length of a Kubernetes resource name
	// (RFC 1123 subdomain). See k8s.io/apimachinery/pkg/util/validation.
	maxResourceNameLength = 253
)

var (
	// labelValueRegexp matches a valid Kubernetes label value: an alphanumeric
	// character optionally followed by alphanumerics, '-', '_' or '.', and
	// ending with an alphanumeric.
	labelValueRegexp = regexp.MustCompile(`^[a-z0-9]([-a-z0-9_.]*[a-z0-9])?$`)

	// resourceNameRegexp matches a valid Kubernetes resource name
	// (RFC 1123 subdomain): lowercase alphanumerics separated by '-' or '.'.
	resourceNameRegexp = regexp.MustCompile(`^[a-z0-9]([-a-z0-9.]*[a-z0-9])?$`)
)

// GetUserGroups fetches all KrknUserGroup CRs that the user belongs to.
// Membership is determined by labels on the KrknUser CR.
//
// Parameters:
//   - ctx: Context for the request
//   - k8sClient: Kubernetes client for API calls
//   - userID: Email address of the user
//   - namespace: Namespace where CRs are located
//
// Returns the list of groups the user belongs to, or an error.
func GetUserGroups(ctx context.Context, k8sClient client.Client, userID, namespace string) ([]krknv1alpha1.KrknUserGroup, error) {
	logger := log.FromContext(ctx).WithName("groupauth.GetUserGroups")

	// Fetch KrknUser by email
	userName, err := SanitizeUserIDForResourceName(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID %q: %w", userID, err)
	}
	user := &krknv1alpha1.KrknUser{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: userName, Namespace: namespace}, user); err != nil {
		return nil, fmt.Errorf("failed to get user %s: %w", userID, err)
	}

	// Extract group names from labels
	groupNames := ExtractGroupNamesFromLabels(user.Labels)
	if len(groupNames) == 0 {
		logger.V(1).Info("User has no group memberships", "userID", userID)
		return []krknv1alpha1.KrknUserGroup{}, nil
	}

	// Fetch KrknUserGroup CRs by name
	groups := make([]krknv1alpha1.KrknUserGroup, 0, len(groupNames))
	for _, groupName := range groupNames {
		group := &krknv1alpha1.KrknUserGroup{}
		if err := k8sClient.Get(ctx, client.ObjectKey{Name: groupName, Namespace: namespace}, group); err != nil {
			// Log warning but continue - group may have been deleted
			logger.V(1).Info("Failed to fetch group, skipping", "groupName", groupName, "error", err.Error())
			continue
		}
		groups = append(groups, *group)
	}

	logger.V(1).Info("Fetched user groups", "userID", userID, "groupCount", len(groups))
	return groups, nil
}

// AggregateClusterPermissions aggregates permissions from all user groups.
// Permissions are combined using union logic - if any group grants an action, it's allowed.
//
// Parameters:
//   - userGroups: List of groups the user belongs to
//
// Returns a map of clusterAPIURL -> list of allowed actions.
func AggregateClusterPermissions(userGroups []krknv1alpha1.KrknUserGroup) map[string][]Action {
	// Use map[Action]bool for deduplication
	permissionsMap := make(map[string]map[Action]bool)

	for _, group := range userGroups {
		for clusterAPIURL, permSet := range group.Spec.ClusterPermissions {
			if permissionsMap[clusterAPIURL] == nil {
				permissionsMap[clusterAPIURL] = make(map[Action]bool)
			}

			// Union: add all actions from this group
			for _, action := range permSet.Actions {
				permissionsMap[clusterAPIURL][Action(action)] = true
			}
		}
	}

	// Convert to []Action
	result := make(map[string][]Action)
	for clusterAPIURL, actionSet := range permissionsMap {
		actions := make([]Action, 0, len(actionSet))
		for action := range actionSet {
			actions = append(actions, action)
		}
		result[clusterAPIURL] = actions
	}

	return result
}

// CanPerformAction checks if the user can perform the given action on the cluster.
//
// Parameters:
//   - userGroups: List of groups the user belongs to
//   - clusterAPIURL: The cluster API URL to check
//   - action: The action to validate (view, run, cancel)
//
// Returns true if the user has permission, false otherwise.
func CanPerformAction(userGroups []krknv1alpha1.KrknUserGroup, clusterAPIURL string, action Action) bool {
	permissions := AggregateClusterPermissions(userGroups)

	allowedActions, hasAccess := permissions[clusterAPIURL]
	if !hasAccess {
		return false
	}

	for _, allowed := range allowedActions {
		if allowed == action {
			return true
		}
	}

	return false
}

// HasClusterPermission checks if a user has a specific permission on a cluster.
// This is a convenience wrapper that combines GetUserGroups and CanPerformAction.
//
// Parameters:
//   - ctx: Context for the request
//   - k8sClient: Kubernetes client
//   - userID: Email address of the user
//   - namespace: Namespace where user and group CRs are located
//   - clusterAPIURL: The cluster API URL to check permission for
//   - action: The action to check (e.g., ActionView, ActionRun, ActionCancel)
//
// Returns true if the user has the permission, false otherwise.
func HasClusterPermission(
	ctx context.Context,
	k8sClient client.Client,
	userID string,
	namespace string,
	clusterAPIURL string,
	action Action,
) (bool, error) {
	// Fetch user groups
	userGroups, err := GetUserGroups(ctx, k8sClient, userID, namespace)
	if err != nil {
		return false, fmt.Errorf("failed to get user groups: %w", err)
	}

	// No groups = no permissions
	if len(userGroups) == 0 {
		return false, nil
	}

	// Check if user can perform the action
	return CanPerformAction(userGroups, clusterAPIURL, action), nil
}

// CountGroupMembers counts the number of KrknUsers that belong to a group.
// Used for populating group metadata/stats.
//
// Parameters:
//   - ctx: Context for the request
//   - k8sClient: Kubernetes client
//   - groupName: Name of the group
//   - namespace: Namespace where users are located
//
// Returns the count of members, or an error.
func CountGroupMembers(ctx context.Context, k8sClient client.Client, groupName, namespace string) (int, error) {
	labelKey := GroupLabelKey(groupName)

	userList := &krknv1alpha1.KrknUserList{}
	err := k8sClient.List(ctx, userList,
		client.InNamespace(namespace),
		client.MatchingLabels{labelKey: "true"},
	)

	if err != nil {
		return 0, fmt.Errorf("failed to list users for group %s: %w", groupName, err)
	}

	return len(userList.Items), nil
}

// SanitizeUserIDForResourceName converts an email address to a valid Kubernetes resource name.
// It replaces @ and . with -, converts to lowercase, and adds the "krknuser-" prefix.
// Use this when constructing a KrknUser resource name for Kubernetes API lookups.
//
// It returns an error if the input is empty or would produce an identifier that
// is not a valid Kubernetes resource name (RFC 1123 subdomain). Failing fast here
// prevents identity confusion in authorization lookups caused by silently
// generating invalid or ambiguous resource names.
//
// Example: "user@example.com" -> "krknuser-user-example-com"
func SanitizeUserIDForResourceName(email string) (string, error) {
	label, err := SanitizeUserIDForLabel(email)
	if err != nil {
		return "", err
	}
	name := resourceNamePrefix + label
	if len(name) > maxResourceNameLength || !resourceNameRegexp.MatchString(name) {
		return "", fmt.Errorf("user ID %q does not produce a valid Kubernetes resource name", email)
	}
	return name, nil
}

// SanitizeUserIDForLabel converts an email address to a valid Kubernetes label value.
// It replaces @ and . with -, then converts to lowercase to comply with
// Kubernetes label value requirements (RFC 1123). No prefix is added.
// Use this when setting or comparing owner-user labels on Kubernetes resources.
//
// It returns an error if the input is empty or would produce a value that is not
// a valid Kubernetes label value. Failing fast prevents invalid owner labels that
// would otherwise be rejected by the API server or, worse, silently break
// ownership comparisons that authorization relies on.
//
// Example: "user@example.com" -> "user-example-com"
func SanitizeUserIDForLabel(email string) (string, error) {
	if strings.TrimSpace(email) == "" {
		return "", fmt.Errorf("cannot sanitize an empty user ID")
	}
	sanitized := strings.ReplaceAll(email, "@", "-")
	sanitized = strings.ReplaceAll(sanitized, ".", "-")
	sanitized = strings.ToLower(sanitized)
	if len(sanitized) > maxLabelValueLength || !labelValueRegexp.MatchString(sanitized) {
		return "", fmt.Errorf("user ID %q does not produce a valid Kubernetes label value", email)
	}
	return sanitized, nil
}
