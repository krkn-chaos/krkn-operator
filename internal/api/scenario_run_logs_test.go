package api

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func TestIsFailedJob(t *testing.T) {
	tests := []struct {
		name          string
		phase         string
		failureReason string
		want          bool
	}{
		{name: "image pull backoff", phase: "Failed", failureReason: "ImagePullBackOff", want: true},
		{name: "container config error", phase: "Failed", failureReason: "CreateContainerConfigError", want: true},
		{name: "scenario container error", phase: "Failed", failureReason: "ContainerError", want: true},
		{name: "max retries exceeded", phase: "MaxRetriesExceeded", want: false},
		{name: "cancelled", phase: "Cancelled", want: true},
		{name: "running image pull backoff", phase: "Running", failureReason: "ImagePullBackOff", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &krknv1alpha1.ClusterJobStatus{Phase: tt.phase, FailureReason: tt.failureReason}
			if got := isFailedJob(job); got != tt.want {
				t.Errorf("isFailedJob() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPreStartFailureReason(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		want   string
	}{
		{name: "invalid image", reason: "InvalidImageName", want: "InvalidImageName"},
		{name: "transient container creation", reason: "ContainerCreating", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &corev1.Pod{Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{{
					State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: tt.reason}},
				}},
			}}
			if got := preStartFailureReason(pod); got != tt.want {
				t.Errorf("preStartFailureReason() = %q, want %q", got, tt.want)
			}
		})
	}
}
