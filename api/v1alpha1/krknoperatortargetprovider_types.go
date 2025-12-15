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

// KrknOperatorTargetProviderSpec defines the desired state of KrknOperatorTargetProvider.
type KrknOperatorTargetProviderSpec struct {
	// OperatorName is the unique identifier for this operator instance
	OperatorName string `json:"operator-name"`
	// Active indicates whether this provider is actively contributing to target requests
	// +kubebuilder:default=true
	Active bool `json:"active"`
}

// KrknOperatorTargetProviderStatus defines the observed state of KrknOperatorTargetProvider.
type KrknOperatorTargetProviderStatus struct {
	// Timestamp represents the last heartbeat/update time from the operator
	Timestamp metav1.Time `json:"timestamp,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Operator",type=string,JSONPath=`.spec.operator-name`
// +kubebuilder:printcolumn:name="Active",type=boolean,JSONPath=`.spec.active`
// +kubebuilder:printcolumn:name="Last Heartbeat",type=date,JSONPath=`.status.timestamp`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=kotp

// KrknOperatorTargetProvider is the Schema for the krknoperatortargetproviders API.
type KrknOperatorTargetProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KrknOperatorTargetProviderSpec   `json:"spec,omitempty"`
	Status KrknOperatorTargetProviderStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KrknOperatorTargetProviderList contains a list of KrknOperatorTargetProvider.
type KrknOperatorTargetProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrknOperatorTargetProvider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KrknOperatorTargetProvider{}, &KrknOperatorTargetProviderList{})
}
