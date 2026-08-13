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
*/

package api

import (
	"context"
	"fmt"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// k8sUserStatusChecker implements auth.UserStatusChecker by looking up KrknUser CRs
// in the Kubernetes API server. It checks the Status.Active field to determine
// whether a user account is still active.
type k8sUserStatusChecker struct {
	k8sClient client.Client
	namespace string
}

// newK8sUserStatusChecker creates a new checker that looks up KrknUser CRs.
//
// Parameters:
//   - k8sClient: the Kubernetes client used to read KrknUser CRs
//   - namespace: the namespace where KrknUser CRs are stored
//
// Returns a k8sUserStatusChecker instance.
func newK8sUserStatusChecker(k8sClient client.Client, namespace string) *k8sUserStatusChecker {
	return &k8sUserStatusChecker{
		k8sClient: k8sClient,
		namespace: namespace,
	}
}

// IsUserActive looks up the KrknUser CR for the given userID and returns
// whether the account is active. The user is looked up by the sanitized
// resource name (same as how user CRs are created during registration).
//
// Returns false if the user is not found (account may have been deleted).
func (c *k8sUserStatusChecker) IsUserActive(ctx context.Context, userID string) (bool, error) {
	// KrknUser CRs are named by sanitizing the email address
	resourceName := sanitizeResourceName(userID)

	var user krknv1alpha1.KrknUser
	if err := c.k8sClient.Get(ctx, client.ObjectKey{
		Name:      resourceName,
		Namespace: c.namespace,
	}, &user); err != nil {
		if client.IgnoreNotFound(err) == nil {
			// User CR not found -- treat as inactive (account deleted)
			return false, nil
		}
		return false, fmt.Errorf("failed to look up user %q: %w", userID, err)
	}

	return user.Status.Active, nil
}
