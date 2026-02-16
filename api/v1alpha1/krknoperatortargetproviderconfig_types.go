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

// ProviderConfigData contains configuration information from a provider
type ProviderConfigData struct {
	// ConfigMap is the name of the ConfigMap containing the provider's configuration
	ConfigMap string `json:"config-map"`
	// ConfigSchema is the JSON schema for the provider's configuration (as a JSON string)
	ConfigSchema string `json:"config-schema,omitempty"`
}

// KrknOperatorTargetProviderConfigSpec defines the desired state of KrknOperatorTargetProviderConfig.
type KrknOperatorTargetProviderConfigSpec struct {
	// UUID is a unique identifier for this config request.
	// The operator will automatically add a label 'krkn.krkn-chaos.dev/uuid' with this value
	// for easy selection: kubectl get krknoperatortargetproviderconfigs -l krkn.krkn-chaos.dev/uuid=<uuid>
	UUID string `json:"uuid"`
}

// KrknOperatorTargetProviderConfigStatus defines the observed state of KrknOperatorTargetProviderConfig.
type KrknOperatorTargetProviderConfigStatus struct {
	// Status represents the current state of the request (pending, completed)
	Status string `json:"status,omitempty"`
	// ConfigData contains a map of operator-name to provider configuration data
	// This allows multiple operators to contribute their configuration schemas to the same request
	ConfigData map[string]ProviderConfigData `json:"configData,omitempty"`
	// Created is the timestamp when the CR was created and set to pending
	Created *metav1.Time `json:"created,omitempty"`
	// Completed is the timestamp when the CR was marked as completed
	Completed *metav1.Time `json:"completed,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="UUID",type=string,JSONPath=`.spec.uuid`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=kotpc

// KrknOperatorTargetProviderConfig is the Schema for the krknoperatortargetproviderconfigs API.
type KrknOperatorTargetProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KrknOperatorTargetProviderConfigSpec   `json:"spec,omitempty"`
	Status KrknOperatorTargetProviderConfigStatus `json:"status,omitempty"`
}

// Default sets default values for KrknOperatorTargetProviderConfig
func (r *KrknOperatorTargetProviderConfig) Default() {
	if r.Status.Status == "" {
		r.Status.Status = "pending"
	}
}

// +kubebuilder:object:root=true

// KrknOperatorTargetProviderConfigList contains a list of KrknOperatorTargetProviderConfig.
type KrknOperatorTargetProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrknOperatorTargetProviderConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KrknOperatorTargetProviderConfig{}, &KrknOperatorTargetProviderConfigList{})
}
