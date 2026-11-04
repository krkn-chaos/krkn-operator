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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

var (
	krknOperatorTargetResource = schema.GroupVersionResource{
		Group: "krkn.krkn-chaos.dev", Version: "v1alpha1", Resource: "krknoperatortargets",
	}
	secretResource = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
)

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
