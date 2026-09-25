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

// KrknCategorySpec defines the desired state of a KrknCategory.
type KrknCategorySpec struct {
	// Color is an optional hex color for UI display (e.g., "#FF5733").
	// If empty, consumers may use a default color.
	// +optional
	// +kubebuilder:validation:Pattern=`^#[0-9A-Fa-f]{6}$|^$`
	Color string `json:"color,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Color",type=string,JSONPath=`.spec.color`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=kcat

// KrknCategory is the Schema for the krkncategories API.
// The Kubernetes resource name identifies the category. Public and group
// visibility is stored in this resource's metadata labels; labels associating
// categories with runs are managed separately from the category definition.
type KrknCategory struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KrknCategorySpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// KrknCategoryList contains a list of KrknCategory.
type KrknCategoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrknCategory `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, &KrknCategory{}, &KrknCategoryList{})
		return nil
	})
}
