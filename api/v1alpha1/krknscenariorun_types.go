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
	"fmt"
	"regexp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var scenarioNamePattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)

// ClusterResiliencyScore represents the resiliency score for a specific cluster
type ClusterResiliencyScore struct {
	// ClusterName is the name of the cluster this score applies to
	ClusterName string `json:"clusterName"`
	// Score is the calculated resiliency score for this cluster (0-100)
	Score float64 `json:"score"`
}

// FileMount represents a file to be mounted in the scenario pod
type FileMount struct {
	// Name is the name of the file
	Name string `json:"name"`
	// Content is the base64-encoded content of the file
	Content string `json:"content"`
	// MountPath is the absolute path where the file should be mounted
	MountPath string `json:"mountPath"`
	// FileID is the UUID of the source ConfigMap (optional, preserved for replay functionality)
	// +optional
	FileID string `json:"fileId,omitempty"`
}

// ClusterJobStatus represents the status of a scenario job for a specific cluster
type ClusterJobStatus struct {
	// ProviderName is the name of the provider that owns this cluster
	ProviderName string `json:"providerName"`
	// ClusterName is the name of the target cluster
	ClusterName string `json:"clusterName"`
	// ClusterAPIURL is the API URL of the cluster for permission checks
	// +optional
	ClusterAPIURL string `json:"clusterApiUrl,omitempty"`
	// JobID is the unique identifier for this job
	JobID string `json:"jobId"`
	// PodName is the name of the pod running the scenario
	PodName string `json:"podName,omitempty"`
	// ContainerImage is the full container image path (registry/repository:tag) being run
	// +optional
	ContainerImage string `json:"containerImage,omitempty"`
	// Phase is the current phase of the job (Pending, Running, Succeeded, Failed, Retrying, Cancelled, MaxRetriesExceeded)
	// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed;Retrying;Cancelled;MaxRetriesExceeded
	Phase string `json:"phase"`
	// StartTime is when the job started
	StartTime *metav1.Time `json:"startTime,omitempty"`
	// CompletionTime is when the job completed
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
	// Message contains additional information about the job status
	Message string `json:"message,omitempty"`

	// RetryCount is the number of times this job has been retried
	// +optional
	RetryCount int `json:"retryCount,omitempty"`
	// MaxRetries is the maximum number of retries allowed for this job
	// +optional
	MaxRetries int `json:"maxRetries,omitempty"`
	// CancelRequested indicates if the user has requested cancellation
	// +optional
	CancelRequested bool `json:"cancelRequested,omitempty"`
	// LastRetryTime is when the last retry was initiated
	// +optional
	LastRetryTime *metav1.Time `json:"lastRetryTime,omitempty"`
	// FailureReason contains a categorized failure reason (OOMKilled, ContainerError, etc.)
	// +optional
	FailureReason string `json:"failureReason,omitempty"`
}

// ScenarioReference identifies a scenario without allowing the caller to
// provide an executable image reference. The operator resolves the image
// through krknctl using this identity and the selected registry Secret.
type ScenarioReference struct {
	// Name is the scenario tag/name to resolve.
	Name string `json:"name"`
	// Private selects a saved private registry when true, or krknctl's public
	// Quay provider when false. A pointer makes the field mandatory on input.
	Private *bool `json:"private"`
	// RegistryName identifies the saved private registry when Private is true.
	RegistryName string `json:"registryName,omitempty"`
}

// Validate checks that a scenario reference is complete and unambiguous.
// It rejects empty or malformed tag names, missing Private values, private
// scenarios without RegistryName, and public scenarios with RegistryName.
// Names must be valid container tags: they may contain letters, digits,
// underscores, dots, and hyphens, must start with a letter, digit, or
// underscore, and must be at most 128 characters. Registry paths, digests,
// whitespace, and additional tag separators are not accepted.
func (r ScenarioReference) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("scenario.name is required")
	}
	if !scenarioNamePattern.MatchString(r.Name) {
		return fmt.Errorf("scenario.name must be a valid container tag")
	}
	if r.Private == nil {
		return fmt.Errorf("scenario.private is required")
	}
	if *r.Private {
		if r.RegistryName == "" {
			return fmt.Errorf("scenario.registryName is required for private scenarios")
		}
		return nil
	}
	if r.RegistryName != "" {
		return fmt.Errorf("scenario.registryName must be omitted for public scenarios")
	}
	return nil
}

// KrknScenarioRunSpec defines the desired state of KrknScenarioRun
type KrknScenarioRunSpec struct {
	// TargetRequestID is the reference to the KrknTargetRequest CR
	TargetRequestID string `json:"targetRequestId"`

	// OwnerUserID is the email address of the user who created this scenario run
	// +optional
	OwnerUserID string `json:"ownerUserId,omitempty"`

	// CustomRunName is a user-provided label for the run, displayed in the console
	// +optional
	CustomRunName string `json:"customRunName,omitempty"`

	// TargetClusters is a map of provider-name to list of cluster names
	// Example: {"krkn-operator": ["cluster1", "cluster2"], "krkn-operator-acm": ["cluster3"]}
	// +kubebuilder:validation:MinProperties=1
	TargetClusters map[string][]string `json:"targetClusters"`

	// Scenario identifies the scenario and registry to resolve. It is the only
	// source of image identity; complete image references are not accepted.
	Scenario ScenarioReference `json:"scenario"`

	// Legacy fields remain only as Go compatibility shims for code that builds
	// old in-memory test objects. They are not serialized into the CRD and are
	// never used to select a pod image.
	ScenarioName       string `json:"-"`
	ScenarioImage      string `json:"-"`
	RegistryURL        string `json:"-"`
	ScenarioRepository string `json:"-"`
	Token              string `json:"-"`
	Username           string `json:"-"`
	Password           string `json:"-"`
	RegistryName       string `json:"-"`

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

	// MaxRetries is the maximum number of times to retry failed jobs
	// +optional
	// +kubebuilder:default=3
	MaxRetries int `json:"maxRetries,omitempty"`

	// RetryBackoff determines the backoff strategy for retries (exponential or fixed)
	// +optional
	// +kubebuilder:validation:Enum=exponential;fixed
	// +kubebuilder:default="exponential"
	RetryBackoff string `json:"retryBackoff,omitempty"`

	// RetryDelay is the initial delay before retrying (e.g., "10s")
	// +optional
	// +kubebuilder:default="10s"
	RetryDelay string `json:"retryDelay,omitempty"`
}

// KrknScenarioRunStatus defines the observed state of KrknScenarioRun
type KrknScenarioRunStatus struct {
	// Phase is the overall phase of the scenario run
	// +kubebuilder:validation:Enum=Pending;Running;Succeeded;PartiallyFailed;Failed
	Phase string `json:"phase,omitempty"`

	// Message contains human-readable context for the current phase.
	// +optional
	Message string `json:"message,omitempty"`

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

	// ResiliencyScores contains per-cluster resiliency scores for this scenario run.
	// Each entry represents the score calculated from the pod logs of a specific cluster.
	// When a scenario runs on multiple clusters, this array will contain one entry per cluster.
	// Populated only when the parent KrknGraphRun has Spec.ResiliencyScoreEnabled set to true.
	// +optional
	ResiliencyScores []ClusterResiliencyScore `json:"resiliencyScores,omitempty"`

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
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, &KrknScenarioRun{}, &KrknScenarioRunList{})
		return nil
	})
}
