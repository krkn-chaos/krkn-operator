package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// KrknWorkspaceSpec publishes an existing PVC under a platform name.
type KrknWorkspaceSpec struct {
	// ClaimName is the PVC in the operator namespace.
	// +kubebuilder:validation:MinLength=1
	ClaimName string `json:"claimName"`
	// ClaimUID pins the publication to this exact PVC instance.
	// +kubebuilder:validation:MinLength=1
	ClaimUID string `json:"claimUid"`
	// DisplayName is an optional human-readable name.
	// +optional
	DisplayName string `json:"displayName,omitempty"`
	// AvailableToAll permits every authenticated platform user to select it.
	// +optional
	AvailableToAll bool `json:"availableToAll,omitempty"`
	// Group permits members of one platform group to select it.
	// +optional
	Group string `json:"group,omitempty"`
}

// KrknWorkspaceStatus reports the observed PVC metadata. It does not prove
// that a scenario process can write to the backing filesystem.
type KrknWorkspaceStatus struct {
	// ObservedGeneration is the workspace spec generation last reconciled.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions includes the Available condition.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="PVC",type=string,JSONPath=`.spec.claimName`
// +kubebuilder:printcolumn:name="Available",type=string,JSONPath=`.status.conditions[?(@.type=="Available")].status`
// +kubebuilder:resource:shortName=kws
// +kubebuilder:metadata:labels=app.kubernetes.io/name=krkn-operator
// KrknWorkspace publishes an externally provisioned PVC for scenario runs.
type KrknWorkspace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KrknWorkspaceSpec   `json:"spec"`
	Status            KrknWorkspaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// KrknWorkspaceList contains KrknWorkspace objects.
type KrknWorkspaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrknWorkspace `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, &KrknWorkspace{}, &KrknWorkspaceList{})
		return nil
	})
}
