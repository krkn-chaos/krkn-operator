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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterTarget represents the target cluster information
type ClusterTarget struct {
	// ClusterName is the name of the managed cluster
	ClusterName string `json:"cluster-name"`
	// ClusterAPIURL is the API server URL of the managed cluster
	ClusterAPIURL string `json:"cluster-api-url"`
}

// KrknTargetRequestSpec defines the desired state of KrknTargetRequest.
type KrknTargetRequestSpec struct {
	// UUID is a unique identifier for this request.
	// The operator will automatically add a label 'krkn.krkn-chaos.dev/uuid' with this value
	// for easy selection: kubectl get krkntargetrequests -l krkn.krkn-chaos.dev/uuid=<uuid>
	UUID string `json:"uuid"`
}

// KrknTargetRequestStatus defines the observed state of KrknTargetRequest.
type KrknTargetRequestStatus struct {
	// Status represents the current state of the request (pending, completed)
	Status string `json:"status,omitempty"`
	// TargetData contains a map of operator-name to list of cluster targets
	// This allows multiple operators to contribute their targets to the same request
	TargetData map[string][]ClusterTarget `json:"targetData,omitempty"`
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
// +kubebuilder:resource:shortName=ktr

// KrknTargetRequest is the Schema for the krkntargetrequests API.
type KrknTargetRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KrknTargetRequestSpec   `json:"spec,omitempty"`
	Status KrknTargetRequestStatus `json:"status,omitempty"`
}

// Default sets default values for KrknTargetRequest
func (r *KrknTargetRequest) Default() {
	if r.Status.Status == "" {
		r.Status.Status = "pending"
	}
}

// +kubebuilder:object:root=true

// KrknTargetRequestList contains a list of KrknTargetRequest.
type KrknTargetRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrknTargetRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KrknTargetRequest{}, &KrknTargetRequestList{})
}
