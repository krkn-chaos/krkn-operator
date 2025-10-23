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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KrknTargetSpec defines the desired state of KrknTarget.
type KrknTargetAction string

const ActionCreate KrknTargetAction = "create"
const ActionDelete KrknTargetAction = "delete"
const ActionUpdate KrknTargetAction = "update"

type KrknTargetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Name        string `json:"name"`
	APIEndpoint string `json:"apiEndpoint"`
	Token       string `json:"token"`
	// +kubebuilder:validation:Enum=create;delete;update
	Action KrknTargetAction `json:"action"`
}

// KrknTargetStatus defines the observed state of KrknTarget.
type KrknTargetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KrknTarget is the Schema for the krkntargets API.
type KrknTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KrknTargetSpec   `json:"spec,omitempty"`
	Status KrknTargetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KrknTargetList contains a list of KrknTarget.
type KrknTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrknTarget `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KrknTarget{}, &KrknTargetList{})
}
