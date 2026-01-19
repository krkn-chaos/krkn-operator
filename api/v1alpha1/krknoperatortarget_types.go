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

// KrknOperatorTargetSpec defines the desired state of KrknOperatorTarget.
type KrknOperatorTargetSpec struct {
	// UUID is the unique identifier for this target
	UUID string `json:"uuid"`

	// ClusterName is the name of the target cluster
	ClusterName string `json:"clusterName"`

	// ClusterAPIURL is the Kubernetes API server URL
	// +optional
	ClusterAPIURL string `json:"clusterAPIURL,omitempty"`

	// SecretType specifies the authentication method
	// +kubebuilder:validation:Enum=kubeconfig;token;credentials
	SecretType string `json:"secretType"`

	// SecretUUID is the UUID of the Secret containing the kubeconfig
	SecretUUID string `json:"secretUUID"`

	// CABundle is the base64-encoded CA certificate bundle for TLS verification
	// Optional - if not provided and SecretType is not kubeconfig, TLS verification will be skipped
	// +optional
	CABundle string `json:"caBundle,omitempty"`

	// InsecureSkipTLSVerify skips TLS certificate verification
	// Only used when CABundle is not provided
	// +kubebuilder:default=false
	// +optional
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty"`
}

// KrknOperatorTargetStatus defines the observed state of KrknOperatorTarget.
type KrknOperatorTargetStatus struct {
	// Ready indicates whether the target is ready to be used
	// +kubebuilder:default=true
	Ready bool `json:"ready,omitempty"`

	// LastUpdated is the timestamp of the last update
	// +optional
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.clusterName`
// +kubebuilder:printcolumn:name="API URL",type=string,JSONPath=`.spec.clusterAPIURL`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.secretType`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=kot

// KrknOperatorTarget is the Schema for the krknoperatortargets API.
type KrknOperatorTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KrknOperatorTargetSpec   `json:"spec,omitempty"`
	Status KrknOperatorTargetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KrknOperatorTargetList contains a list of KrknOperatorTarget.
type KrknOperatorTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrknOperatorTarget `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KrknOperatorTarget{}, &KrknOperatorTargetList{})
}
