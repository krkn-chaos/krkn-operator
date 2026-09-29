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
	// TargetRequestID identifies the target-discovery request containing the selected cluster credentials.
	TargetRequestID string `json:"targetRequestId"`

	// TargetClusters selects exactly one provider and cluster from TargetRequestID.
	// +kubebuilder:validation:MinProperties=1
	TargetClusters map[string][]string `json:"targetClusters"`

	// ConfigMapName identifies the ConfigMap containing the krkn-ai YAML.
	ConfigMapName string `json:"configMapName"`

	// ConfigMapKey is the data key containing the krkn-ai YAML.
	// +optional
	// +kubebuilder:default=krkn-ai.yaml
	ConfigMapKey string `json:"configMapKey,omitempty"`

	// OrchestratorImage overrides the installation-level Krkn-AI orchestrator image for this run.
	// +optional
	OrchestratorImage string `json:"orchestratorImage,omitempty"`

	// OwnerUserID identifies the API user that created the run.
	// +optional
	OwnerUserID string `json:"ownerUserId,omitempty"`

	// PrometheusURL overrides the Prometheus endpoint used by the orchestrator.
	// +optional
	PrometheusURL string `json:"prometheusUrl,omitempty"`

	// PrometheusTokenSecretRef names a Secret with a "token" key in the operator namespace.
	// +optional
	PrometheusTokenSecretRef string `json:"prometheusTokenSecretRef,omitempty"`

	// ActiveDeadlineSeconds limits the lifetime of the orchestrator Pod.
	// +optional
	// +kubebuilder:default=21600
	// +kubebuilder:validation:Minimum=1
	ActiveDeadlineSeconds int64 `json:"activeDeadlineSeconds,omitempty"`
}

// KrknAIRunStatus defines the observed state of KrknAIRun.
type KrknAIRunStatus struct {
	// Phase is the current run lifecycle phase.
	// +kubebuilder:validation:Enum=Pending;Provisioning;Running;Succeeded;Failed;Cancelled
	Phase string `json:"phase,omitempty"`

	// OrchestratorPodName identifies the Pod executing this run.
	OrchestratorPodName string `json:"orchestratorPodName,omitempty"`
	// StartTime records when reconciliation initialized the run.
	StartTime *metav1.Time `json:"startTime,omitempty"`
	// CompletionTime records when the run reached a terminal phase.
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
	// ScenarioRunRefs lists child KrknScenarioRun resource names.
	ScenarioRunRefs []string `json:"scenarioRunRefs,omitempty"`
	// FailureReason contains the terminal orchestrator or uploader failure.
	FailureReason string `json:"failureReason,omitempty"`
	// Conditions reports independently observable outcomes such as artifact commitment.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=kair
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 63",message="metadata.name must be at most 63 characters because it identifies generated workloads"

// KrknAIRun executes one Krkn-AI optimization run against a selected target cluster.
type KrknAIRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KrknAIRunSpec   `json:"spec,omitempty"`
	Status KrknAIRunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KrknAIRunList contains a list of KrknAIRun resources.
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
