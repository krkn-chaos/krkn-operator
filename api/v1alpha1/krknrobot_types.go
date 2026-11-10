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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Labels applied to KrknRobot resources and their token Secrets.
const (
	// RobotAccountLabel marks a KrknRobot resource.
	RobotAccountLabel = "krkn.krkn-chaos.dev/robot-account"
	// RobotNameLabel identifies the robot a token Secret belongs to.
	RobotNameLabel = "krkn.krkn-chaos.dev/robot"
	// RobotTokenLabel marks a Secret holding a robot token hash.
	RobotTokenLabel = "krkn.krkn-chaos.dev/robot-token"
)

// KrknRobotSpec defines the desired state of KrknRobot.
// A KrknRobot is a machine identity used by AI agents to access the MCP
// endpoint. Group membership is expressed via group labels on the resource.
type KrknRobotSpec struct {
	// Description is a human-readable description of the robot's purpose
	// +kubebuilder:validation:MaxLength=256
	// +optional
	Description string `json:"description,omitempty"`

	// Disabled disables the robot: token verification fails immediately
	// +optional
	Disabled bool `json:"disabled,omitempty"`
}

// KrknRobotStatus defines the observed state of KrknRobot.
type KrknRobotStatus struct {
	// TokenID identifies the currently active token
	// +optional
	TokenID string `json:"tokenId,omitempty"`

	// TokenIssuedAt is the timestamp when the active token was issued
	// +optional
	TokenIssuedAt *metav1.Time `json:"tokenIssuedAt,omitempty"`

	// TokenExpiresAt is the timestamp when the active token expires
	// +optional
	TokenExpiresAt *metav1.Time `json:"tokenExpiresAt,omitempty"`

	// LastUsedAt is the timestamp of the robot's last authenticated request
	// +optional
	LastUsedAt *metav1.Time `json:"lastUsedAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Disabled",type=boolean,JSONPath=`.spec.disabled`
// +kubebuilder:printcolumn:name="TokenExpires",type="date",JSONPath=`.status.tokenExpiresAt`
// +kubebuilder:printcolumn:name="LastUsed",type="date",JSONPath=`.status.lastUsedAt`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=kr
// +kubebuilder:metadata:labels=app.kubernetes.io/name=krkn-operator

// KrknRobot is the Schema for the krknrobots API.
// It represents a machine identity for the krkn-operator MCP endpoint.
//
// The KrknRobot CRD must be labeled with:
//   - krkn.krkn-chaos.dev/robot-account: "true"
//   - krkn.krkn-chaos.dev/<group>: "true" for each group the robot belongs to
type KrknRobot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KrknRobotSpec   `json:"spec,omitempty"`
	Status KrknRobotStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KrknRobotList contains a list of KrknRobot.
type KrknRobotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrknRobot `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, &KrknRobot{}, &KrknRobotList{})
		return nil
	})
}
