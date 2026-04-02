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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterPermissionSet defines the actions allowed on a cluster.
type ClusterPermissionSet struct {
	// Actions is the list of allowed actions on this cluster
	// Allowed values: "view" (can view cluster and scenario runs),
	//                 "run" (can launch scenarios),
	//                 "cancel" (can cancel running scenarios)
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:items:Enum=view;run;cancel
	Actions []string `json:"actions"`
}

// KrknUserGroupSpec defines the desired state of KrknUserGroup.
// KrknUserGroup defines cluster access permissions for a group of users.
// User membership is managed via labels on KrknUser CRs: group.krkn.krkn-chaos.dev/<group-name>=true
type KrknUserGroupSpec struct {
	// Name is the group name (duplicates metadata.name for API convenience)
	Name string `json:"name"`

	// Description is a human-readable description of the group's purpose
	// +optional
	Description string `json:"description,omitempty"`

	// ClusterPermissions defines access permissions per cluster
	// Key: cluster API URL (must match ClusterTarget.ClusterAPIURL from KrknTargetRequest)
	// Value: set of allowed actions (view, run, cancel)
	// +kubebuilder:validation:MinProperties=1
	ClusterPermissions map[string]ClusterPermissionSet `json:"clusterPermissions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Description",type=string,JSONPath=`.spec.description`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=kug

// KrknUserGroup is the Schema for the krknusergroups API.
// It defines cluster-level access control for groups of users.
//
// Users are associated with groups via labels on their KrknUser CR:
//   - group.krkn.krkn-chaos.dev/<group-name>: "true"
//
// Permissions are aggregated across all groups a user belongs to (union).
// Admin users bypass all group-based access controls.
type KrknUserGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KrknUserGroupSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// KrknUserGroupList contains a list of KrknUserGroup.
type KrknUserGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrknUserGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KrknUserGroup{}, &KrknUserGroupList{})
}
