package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ComponentDescription describes a feature exposed by a Krkn operator binary.
type ComponentDescription struct {
	// Component is the unique feature name within this manifest.
	Component string `json:"component"`
	// Description explains the feature exposed by the component.
	Description string `json:"description"`
}

// ComponentManifest is the version manifest bundled in an operator binary.
type ComponentManifest struct {
	// Version is the exact Git tag embedded in the component build.
	Version string `json:"version"`
	// Components lists the named features exposed by this build.
	Components []ComponentDescription `json:"components"`
}

// KrknComponentStatus contains manifests published by component operators.
type KrknComponentStatus struct {
	Components map[string]ComponentManifest `json:"components,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=kc

// KrknComponent publishes the observed version of an operator binary.
type KrknComponent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            KrknComponentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KrknComponentList contains a list of KrknComponent resources.
type KrknComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrknComponent `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, &KrknComponent{}, &KrknComponentList{})
		return nil
	})
}
