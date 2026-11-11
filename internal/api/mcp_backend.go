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
	"sort"

	"sigs.k8s.io/controller-runtime/pkg/client"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/mcpserver"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

// mcpBackend implements mcpserver.Backend on top of the REST Handler.
type mcpBackend struct {
	h *Handler
}

// WhoAmI returns the robot's identity, groups and per-cluster permissions.
func (b *mcpBackend) WhoAmI(ctx context.Context, p mcpserver.Principal) (mcpserver.WhoAmIOutput, error) {
	principalID := "robot:" + p.RobotName

	groups, err := groupauth.GetUserGroups(ctx, b.h.client, principalID, b.h.namespace)
	if err != nil {
		return mcpserver.WhoAmIOutput{}, &mcpserver.BackendError{Code: mcpserver.CodeInternal, Message: "failed to resolve robot groups"}
	}

	robot := &krknv1alpha1.KrknRobot{}
	if err := b.h.client.Get(ctx, client.ObjectKey{Name: p.RobotName, Namespace: b.h.namespace}, robot); err != nil {
		return mcpserver.WhoAmIOutput{}, &mcpserver.BackendError{Code: mcpserver.CodeInternal, Message: "failed to get robot"}
	}

	groupNames := make([]string, 0, len(groups))
	permissions := groupauth.AggregateClusterPermissions(groups)
	permOutputs := make([]mcpserver.ClusterPermissionOutput, 0, len(permissions))
	for url, actions := range permissions {
		sorted := make([]string, 0, len(actions))
		for _, a := range actions {
			sorted = append(sorted, string(a))
		}
		sort.Strings(sorted)
		permOutputs = append(permOutputs, mcpserver.ClusterPermissionOutput{ClusterAPIURL: url, Actions: sorted})
	}
	sort.Slice(permOutputs, func(i, j int) bool { return permOutputs[i].ClusterAPIURL < permOutputs[j].ClusterAPIURL })

	for _, g := range groups {
		groupNames = append(groupNames, g.Name)
	}
	sort.Strings(groupNames)

	var expiresAt = robot.CreationTimestamp.Time
	if robot.Status.TokenExpiresAt != nil {
		expiresAt = robot.Status.TokenExpiresAt.Time
	}

	return mcpserver.WhoAmIOutput{
		RobotName:      p.RobotName,
		Groups:         groupNames,
		TokenExpiresAt: expiresAt,
		Permissions:    permOutputs,
	}, nil
}
