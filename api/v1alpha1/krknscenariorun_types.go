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

// FileMount represents a file to be mounted in the scenario pod
type FileMount struct {
	// Name is the name of the file
	Name string `json:"name"`
	// Content is the base64-encoded content of the file
	Content string `json:"content"`
	// MountPath is the absolute path where the file should be mounted
	MountPath string `json:"mountPath"`
}

// ClusterJobStatus represents the status of a scenario job for a specific cluster
type ClusterJobStatus struct {
	// ClusterName is the name of the target cluster
	ClusterName string `json:"clusterName"`
	// JobId is the unique identifier for this job
	JobId string `json:"jobId"`
	// PodName is the name of the pod running the scenario
	PodName string `json:"podName,omitempty"`
	// Phase is the current phase of the job (Pending, Running, Succeeded, Failed)
	// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed
	Phase string `json:"phase"`
	// StartTime is when the job started
	StartTime *metav1.Time `json:"startTime,omitempty"`
	// CompletionTime is when the job completed
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
	// Message contains additional information about the job status
	Message string `json:"message,omitempty"`
}

// KrknScenarioRunSpec defines the desired state of KrknScenarioRun
type KrknScenarioRunSpec struct {
	// TargetRequestId is the reference to the KrknTargetRequest CR
	TargetRequestId string `json:"targetRequestId"`

	// ClusterNames is the list of target clusters to run the scenario on
	// +kubebuilder:validation:MinItems=1
	ClusterNames []string `json:"clusterNames"`

	// ScenarioName is the name of the scenario to run
	ScenarioName string `json:"scenarioName"`

	// ScenarioImage is the container image for the scenario
	ScenarioImage string `json:"scenarioImage"`

	// KubeconfigPath is the path where kubeconfig will be mounted in the pod
	// +optional
	// +kubebuilder:default="/home/krkn/.kube/config"
	KubeconfigPath string `json:"kubeconfigPath,omitempty"`

	// Files is a list of files to mount in the scenario pod
	// +optional
	Files []FileMount `json:"files,omitempty"`

	// Environment is a map of environment variables to set in the scenario pod
	// +optional
	Environment map[string]string `json:"environment,omitempty"`

	// RegistryURL is the URL of the container registry
	// +optional
	RegistryURL string `json:"registryURL,omitempty"`

	// ScenarioRepository is the repository path in the registry
	// +optional
	ScenarioRepository string `json:"scenarioRepository,omitempty"`

	// Token is the authentication token for the registry
	// +optional
	Token string `json:"token,omitempty"`

	// Username is the username for registry authentication
	// +optional
	Username string `json:"username,omitempty"`

	// Password is the password for registry authentication
	// +optional
	Password string `json:"password,omitempty"`
}

// KrknScenarioRunStatus defines the observed state of KrknScenarioRun
type KrknScenarioRunStatus struct {
	// Phase is the overall phase of the scenario run
	// +kubebuilder:validation:Enum=Pending;Running;Succeeded;PartiallyFailed;Failed
	Phase string `json:"phase,omitempty"`

	// TotalTargets is the total number of target clusters
	TotalTargets int `json:"totalTargets,omitempty"`

	// SuccessfulJobs is the number of successfully completed jobs
	SuccessfulJobs int `json:"successfulJobs,omitempty"`

	// FailedJobs is the number of failed jobs
	FailedJobs int `json:"failedJobs,omitempty"`

	// RunningJobs is the number of currently running jobs
	RunningJobs int `json:"runningJobs,omitempty"`

	// ClusterJobs contains the status of each cluster job
	// +optional
	ClusterJobs []ClusterJobStatus `json:"clusterJobs,omitempty"`

	// Conditions represent the latest available observations of the scenario run's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Targets",type=integer,JSONPath=`.status.totalTargets`
// +kubebuilder:printcolumn:name="Succeeded",type=integer,JSONPath=`.status.successfulJobs`
// +kubebuilder:printcolumn:name="Failed",type=integer,JSONPath=`.status.failedJobs`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=ksr

// KrknScenarioRun is the Schema for the krknscenrarioruns API
type KrknScenarioRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KrknScenarioRunSpec   `json:"spec,omitempty"`
	Status KrknScenarioRunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KrknScenarioRunList contains a list of KrknScenarioRun
type KrknScenarioRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KrknScenarioRun `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KrknScenarioRun{}, &KrknScenarioRunList{})
}
