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

// KrknFileTypeSpec defines the desired state of KrknFileType.
// KrknFileType represents a file category (e.g., "config", "script", "template")
// for organizing ConfigMap-based files in the krkn-operator file management system.
//
// File types are auto-created when first referenced in a file and can be reused
// across multiple files. Each file has exactly one type, associated via labels:
//
//	file-type.krkn.krkn-chaos.dev/<type-name>: "true"
//
// UI metadata (color) helps users visually distinguish file categories.
type KrknFileTypeSpec struct {
	// Name is the file type identifier (e.g., "config", "script", "template")
	// This duplicates metadata.name for API convenience
	Name string `json:"name"`

	// Color is an optional hex color for UI display (e.g., "#FF5733")
	// If empty, the UI will use a default color
	// +optional
	// +kubebuilder:validation:Pattern=`^#[0-9A-Fa-f]{6}$|^$`
	Color string `json:"color,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Color",type=string,JSONPath=`.spec.color`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=kft

// KrknFileType is the Schema for the krknfiletypes API.
// It defines metadata for file categories used in the ConfigMap-based file management system.
//
// File types are automatically created when first used in a file, allowing for
// GitHub-style label management where types can be created on-the-fly or reused.
//
// Each file can have only one type, enforced at the API level.
// Types cannot be deleted if files are still using them (enforced by the API handler).
type KrknFileType struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KrknFileTypeSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// KrknFileTypeList contains a list of KrknFileType.
type KrknFileTypeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrknFileType `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KrknFileType{}, &KrknFileTypeList{})
}
