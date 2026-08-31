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

// KrknAIRunSpec defines the desired state of KrknAIRun.
type KrknAIRunSpec struct {
	TargetRequestID string `json:"targetRequestId"`

	// +kubebuilder:validation:MinProperties=1
	TargetClusters map[string][]string `json:"targetClusters"`

	// ConfigYAMLBase64 is the base64-encoded full krkn-ai.yaml.
	ConfigYAMLBase64 string `json:"configYamlBase64"`

	// +optional
	OrchestratorImage string `json:"orchestratorImage,omitempty"`

	// +optional
	OwnerUserID string `json:"ownerUserId,omitempty"`

	// +optional
	// +kubebuilder:default=0
	ScenarioMaxRetries int `json:"scenarioMaxRetries,omitempty"`

	// +optional
	PrometheusURL string `json:"prometheusUrl,omitempty"`

	// PrometheusTokenSecretRef names a Secret with a "token" key in the operator namespace.
	// +optional
	PrometheusTokenSecretRef string `json:"prometheusTokenSecretRef,omitempty"`

	// +optional
	Storage KrknAIRunStorageSpec `json:"storage,omitempty"`

	// +optional
	// +kubebuilder:default=21600
	ActiveDeadlineSeconds int64 `json:"activeDeadlineSeconds,omitempty"`
}

// KrknAIRunStorageSpec configures the results PVC.
type KrknAIRunStorageSpec struct {
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`

	// +optional
	// +kubebuilder:default="5Gi"
	Size string `json:"size,omitempty"`
}

// KrknAIRunStatus defines the observed state of KrknAIRun.
type KrknAIRunStatus struct {
	// +kubebuilder:validation:Enum=Pending;Provisioning;Running;Succeeded;Failed;Cancelled
	Phase string `json:"phase,omitempty"`

	OrchestratorPodName string             `json:"orchestratorPodName,omitempty"`
	PVCName             string             `json:"pvcName,omitempty"`
	StartTime           *metav1.Time       `json:"startTime,omitempty"`
	CompletionTime      *metav1.Time       `json:"completionTime,omitempty"`
	ScenarioRunRefs     []string           `json:"scenarioRunRefs,omitempty"`
	FailureReason       string             `json:"failureReason,omitempty"`
	Conditions          []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=kair

type KrknAIRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KrknAIRunSpec   `json:"spec,omitempty"`
	Status KrknAIRunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type KrknAIRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrknAIRun `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, &KrknAIRun{}, &KrknAIRunList{})
		return nil
	})
}
