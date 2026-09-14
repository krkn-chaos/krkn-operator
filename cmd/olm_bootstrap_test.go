package main

import (
	"context"
	"errors"
	"testing"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestBootstrapOLMResourcesWiring(t *testing.T) {
	originalClientset := newOLMClientset
	originalEnsure := ensureOLMResources
	t.Cleanup(func() {
		newOLMClientset = originalClientset
		ensureOLMResources = originalEnsure
	})

	clientset := fake.NewSimpleClientset()
	newOLMClientset = func() (kubernetes.Interface, error) {
		return clientset, nil
	}
	var gotNamespace string
	var gotOpenShift bool
	ensureOLMResources = func(_ context.Context, _ kubernetes.Interface, namespace string, openshift bool) error {
		gotNamespace = namespace
		gotOpenShift = openshift
		return nil
	}

	if err := bootstrapOLMResources("krkn-operator-test", true); err != nil {
		t.Fatalf("bootstrapOLMResources() error = %v", err)
	}
	if gotNamespace != "krkn-operator-test" || !gotOpenShift {
		t.Fatalf("bootstrapOLMResources() forwarded namespace=%q openshift=%t", gotNamespace, gotOpenShift)
	}
}

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
