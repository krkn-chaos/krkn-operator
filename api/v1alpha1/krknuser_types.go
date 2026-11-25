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
	"k8s.io/apimachinery/pkg/runtime"
)

// KrknUserSpec defines the desired state of KrknUser.
// KrknUser serves as an authentication entity for the REST APIs.
// Local users have a password Secret; external OIDC users have an issuer and subject instead.
type KrknUserSpec struct {
	// UserID is the local email or a stable oidc-<hash> identifier.
	UserID string `json:"userId"`

	// Name is the first name of the user
	Name string `json:"name"`

	// Surname is the last name of the user
	Surname string `json:"surname"`

	// Organization is the user's organization name
	// +optional
	Organization string `json:"organization,omitempty"`

	// Role defines the user's permission level
	// +kubebuilder:validation:Enum=user;admin
	// +kubebuilder:default=user
	Role string `json:"role"`

	// PasswordSecretRef references the local user's bcrypt hash Secret.
	// It must be empty for OIDC users.
	// +optional
	PasswordSecretRef string `json:"passwordSecretRef,omitempty"`

	// AuthProvider is empty for local users or the immutable OIDC provider ID.
	// +optional
	AuthProvider string `json:"authProvider,omitempty"`
	// ExternalIssuer and ExternalSubject bind an OIDC user to the verified ID token.
	// +optional
	ExternalIssuer string `json:"externalIssuer,omitempty"`
	// +optional
	ExternalSubject string `json:"externalSubject,omitempty"`
	// ExternalUsername and ExternalEmail are display-only snapshots.
	// +optional
	ExternalUsername string `json:"externalUsername,omitempty"`
	// +optional
	ExternalEmail string `json:"externalEmail,omitempty"`
}

// KrknUserStatus defines the observed state of KrknUser.
type KrknUserStatus struct {
	// Active indicates whether the user account is active
	// +kubebuilder:default=true
	Active bool `json:"active,omitempty"`

	// Created is the timestamp when the user was created
	// +optional
	Created metav1.Time `json:"created,omitempty"`

	// LastLogin is the timestamp of the user's last successful login
	// +optional
	LastLogin metav1.Time `json:"lastLogin,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="UserID",type=string,JSONPath=`.spec.userId`
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Surname",type=string,JSONPath=`.spec.surname`
// +kubebuilder:printcolumn:name="Role",type=string,JSONPath=`.spec.role`
// +kubebuilder:printcolumn:name="Active",type=boolean,JSONPath=`.status.active`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=ku
// +kubebuilder:metadata:labels=app.kubernetes.io/name=krkn-operator

// KrknUser is the Schema for the krknusers API.
// It represents an authentication entity for the krkn-operator REST APIs.
//
// The KrknUser CRD must be labeled with:
//   - krkn.krkn-chaos.dev/user-account: "true"
//   - krkn.krkn-chaos.dev/role: <user|admin>
type KrknUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KrknUserSpec   `json:"spec,omitempty"`
	Status KrknUserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KrknUserList contains a list of KrknUser.
type KrknUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrknUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, &KrknUser{}, &KrknUserList{})
		return nil
	})
}
