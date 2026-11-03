package main

import (
	"errors"
	"testing"

	"k8s.io/client-go/kubernetes"
)

func TestBootstrapOLMResourcesPropagatesClientError(t *testing.T) {
	originalClientset := newOLMClientset
	t.Cleanup(func() { newOLMClientset = originalClientset })

	wantErr := errors.New("client unavailable")
	newOLMClientset = func() (kubernetes.Interface, error) {
		return nil, wantErr
	}

	if err := bootstrapOLMResources("krkn-operator-test", false); !errors.Is(err, wantErr) {
		t.Fatalf("bootstrapOLMResources() error = %v, want wrapped %v", err, wantErr)
	}
}
