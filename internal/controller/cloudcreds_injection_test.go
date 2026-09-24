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

package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/krkn-chaos/krkn-operator/pkg/cloudcreds"
)

func TestAppendCloudCredentialInjection_UsesSecretKeyRef(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	const (
		credName  = "aws-pod-contract"
		namespace = "default"
	)

	cred := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      credName,
			Namespace: namespace,
			Labels:    cloudcreds.BuildLabels(cloudcreds.ProviderAWS, nil, true),
		},
		Data: map[string][]byte{
			cloudcreds.SecretKeyAWSAccessKeyID:     []byte("AKIAPOD"),
			cloudcreds.SecretKeyAWSSecretAccessKey: []byte("pod-secret"),
			cloudcreds.SecretKeyAWSDefaultRegion:   []byte("eu-central-1"),
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cred).Build()
	reconciler := &KrknScenarioRunReconciler{
		Client:    fakeClient,
		Namespace: namespace,
	}

	baseEnv := []corev1.EnvVar{
		{Name: "TIMEOUT", Value: "300"},
		{Name: "NODE_NAME", Value: "worker-1"},
	}

	envVars, volumes, mounts, err := reconciler.appendCloudCredentialInjection(
		context.Background(),
		credName,
		baseEnv,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("appendCloudCredentialInjection: %v", err)
	}
	if len(volumes) != 0 || len(mounts) != 0 {
		t.Errorf("AWS injection should not add volumes/mounts, got volumes=%d mounts=%d", len(volumes), len(mounts))
	}

	byName := map[string]corev1.EnvVar{}
	for _, env := range envVars {
		byName[env.Name] = env
	}

	if env := byName["TIMEOUT"]; env.Value != "300" || env.ValueFrom != nil {
		t.Errorf("TIMEOUT should stay plaintext from the CR Environment, got %+v", env)
	}

	for _, name := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_DEFAULT_REGION"} {
		env, ok := byName[name]
		if !ok {
			t.Fatalf("missing %s", name)
		}
		if env.Value != "" {
			t.Errorf("%s must not be plaintext; Value=%q", name, env.Value)
		}
		if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
			t.Fatalf("%s must use SecretKeyRef", name)
		}
		if env.ValueFrom.SecretKeyRef.Name != credName {
			t.Errorf("%s Secret name = %q, want %q", name, env.ValueFrom.SecretKeyRef.Name, credName)
		}
	}

	if env := byName["CLOUD_TYPE"]; env.Value != "aws" {
		t.Errorf("CLOUD_TYPE = %q, want aws", env.Value)
	}
}

func TestAppendCloudCredentialInjection_EmptyRefNoop(t *testing.T) {
	reconciler := &KrknScenarioRunReconciler{Namespace: "default"}
	in := []corev1.EnvVar{{Name: "TIMEOUT", Value: "1"}}
	out, volumes, mounts, err := reconciler.appendCloudCredentialInjection(
		context.Background(), "", in, nil, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 || out[0].Name != "TIMEOUT" {
		t.Errorf("expected unchanged env, got %+v", out)
	}
	if volumes != nil || mounts != nil {
		t.Errorf("expected nil volumes/mounts, got volumes=%v mounts=%v", volumes, mounts)
	}
}
