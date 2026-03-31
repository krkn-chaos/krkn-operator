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

// Package groupauth provides group-based authorization and permission management for cluster access control.
// It supports label-based group membership and permission aggregation across multiple user groups.
package groupauth

// Action represents a permission action that can be performed on a cluster
type Action string

const (
	// ActionView allows viewing cluster details and scenario runs
	ActionView Action = "view"

	// ActionRun allows launching chaos scenarios on the cluster
	ActionRun Action = "run"

	// ActionCancel allows canceling running chaos scenarios
	ActionCancel Action = "cancel"
)

// GroupLabelPrefix is the label prefix for group membership on KrknUser CRs
// Format: group.krkn.krkn-chaos.dev/<group-name>=true
const GroupLabelPrefix = "group.krkn.krkn-chaos.dev/"

// IsValidAction checks if the given action is valid
func IsValidAction(action string) bool {
	switch Action(action) {
	case ActionView, ActionRun, ActionCancel:
		return true
	default:
		return false
	}
}
