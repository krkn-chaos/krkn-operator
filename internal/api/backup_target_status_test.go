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

package api

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krkn-chaos/krkn-operator/internal/kubeconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	clientgotesting "k8s.io/client-go/testing"
)

func TestRefreshRestoredTargetStatusesMarksUnreachableTargetNotReady(t *testing.T) {
	client, targetName := newTargetStatusDynamicClient(t, "http://127.0.0.1:1", true)

	if err := refreshRestoredTargetStatuses(context.Background(), client, "default"); err != nil {
		t.Fatalf("refreshRestoredTargetStatuses() error = %v", err)
	}

	assertTargetStatus(t, client, targetName, false)
}

func TestRefreshRestoredTargetStatusesMarksReachableTargetReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/version" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, targetName := newTargetStatusDynamicClient(t, server.URL, false)

	if err := refreshRestoredTargetStatuses(context.Background(), client, "default"); err != nil {
		t.Fatalf("refreshRestoredTargetStatuses() error = %v", err)
	}

	assertTargetStatus(t, client, targetName, true)
}

func TestRefreshRestoredTargetStatusesTreatsInvalidOrMissingSecretsAsNotReady(t *testing.T) {
	tests := []struct {
		name       string
		secretName string
		secret     *unstructured.Unstructured
	}{
		{
			name:       "missing secret",
			secretName: "missing-secret",
		},
		{
			name:       "invalid secret data",
			secretName: "invalid-secret",
			secret: &unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Secret",
				"metadata": map[string]interface{}{
					"name":      "invalid-secret",
					"namespace": "default",
				},
				"data": map[string]interface{}{"kubeconfig": "not-base64"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, targetName := newTargetStatusDynamicClientWithSecret(t, tt.secretName, tt.secret, true)

			if err := refreshRestoredTargetStatuses(context.Background(), client, "default"); err != nil {
				t.Fatalf("refreshRestoredTargetStatuses() error = %v", err)
			}

			assertTargetStatus(t, client, targetName, false)
		})
	}
}

func TestRefreshRestoredTargetStatusesRequiresDynamicClient(t *testing.T) {
	if err := refreshRestoredTargetStatuses(context.Background(), nil, "default"); err == nil {
		t.Fatal("refreshRestoredTargetStatuses() expected an error for a nil client")
	}
}

func TestRefreshRestoredTargetStatusesPropagatesListError(t *testing.T) {
	client, _ := newTargetStatusDynamicClient(t, "http://127.0.0.1:1", true)
	client.PrependReactor("list", "krknoperatortargets", func(clientgotesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("list failed")
	})

	if err := refreshRestoredTargetStatuses(context.Background(), client, "default"); err == nil {
		t.Fatal("refreshRestoredTargetStatuses() expected the list error to be propagated")
	}
}

func TestRefreshRestoredTargetStatusesPropagatesStatusUpdateError(t *testing.T) {
	client, _ := newTargetStatusDynamicClient(t, "http://127.0.0.1:1", true)
	client.PrependReactor("update", "krknoperatortargets", func(clientgotesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("status update failed")
	})

	if err := refreshRestoredTargetStatuses(context.Background(), client, "default"); err == nil {
		t.Fatal("refreshRestoredTargetStatuses() expected the status update error to be propagated")
	}
}

func newTargetStatusDynamicClient(t *testing.T, serverURL string, ready bool) (*dynamicfake.FakeDynamicClient, string) {
	t.Helper()
	kubeconfigBase64, err := kubeconfig.GenerateFromToken("target", serverURL, "", "token", true)
	if err != nil {
		t.Fatalf("GenerateFromToken() error = %v", err)
	}
	secretData, err := kubeconfig.MarshalSecretData(kubeconfigBase64)
	if err != nil {
		t.Fatalf("MarshalSecretData() error = %v", err)
	}
	secret := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name":      "target-secret",
			"namespace": "default",
		},
		"data": map[string]interface{}{
			"kubeconfig": base64.StdEncoding.EncodeToString(secretData),
		},
	}}
	return newTargetStatusDynamicClientWithSecret(t, "target-secret", secret, ready)
}

func newTargetStatusDynamicClientWithSecret(t *testing.T, secretName string, secret *unstructured.Unstructured, ready bool) (*dynamicfake.FakeDynamicClient, string) {
	t.Helper()
	const targetName = "target-one"
	objects := []runtime.Object{newTargetStatusObject(targetName, secretName, ready)}
	if secret != nil {
		objects = append(objects, secret)
	}

	listKinds := map[schema.GroupVersionResource]string{
		krknOperatorTargetResource: "KrknOperatorTargetList",
		secretResource:             "SecretList",
	}
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds, objects...), targetName
}

func newTargetStatusObject(name, secretName string, ready bool) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "krkn.krkn-chaos.dev/v1alpha1",
		"kind":       "KrknOperatorTarget",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"clusterName": "target",
			"secretUUID":  secretName,
		},
		"status": map[string]interface{}{
			"ready": ready,
		},
	}}
}

func assertTargetStatus(t *testing.T, client *dynamicfake.FakeDynamicClient, targetName string, wantReady bool) {
	t.Helper()
	target, err := client.Resource(krknOperatorTargetResource).Namespace("default").Get(context.Background(), targetName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	ready, found, err := unstructured.NestedBool(target.Object, "status", "ready")
	if err != nil {
		t.Fatalf("NestedBool() error = %v", err)
	}
	if !found {
		t.Fatal("target status.ready was not written")
	}
	if ready != wantReady {
		t.Fatalf("target status.ready = %v, want %v", ready, wantReady)
	}
	lastUpdated, found, err := unstructured.NestedString(target.Object, "status", "lastUpdated")
	if err != nil {
		t.Fatalf("NestedString() error = %v", err)
	}
	if !found || lastUpdated == "" {
		t.Fatal("target status.lastUpdated was not written")
	}
}
