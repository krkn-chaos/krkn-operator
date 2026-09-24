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

package provider

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestBackfillLegacyProviderConfigLabel(t *testing.T) {
	const namespace = "krkn-operator-system"

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}

	legacy := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      LegacyProviderConfigMapName,
			Namespace: namespace,
			Labels:    map[string]string{"existing": "label"},
		},
		Data: map[string]string{"API_PORT": "8080"},
	}
	unrelated := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: namespace},
		Data:       map[string]string{"setting": "value"},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(legacy, unrelated).Build()

	changed, err := BackfillLegacyProviderConfigLabel(context.Background(), k8sClient, namespace)
	if err != nil {
		t.Fatalf("BackfillLegacyProviderConfigLabel() error = %v", err)
	}
	if !changed {
		t.Fatal("BackfillLegacyProviderConfigLabel() changed = false, want true")
	}

	var updated corev1.ConfigMap
	if err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name: LegacyProviderConfigMapName, Namespace: namespace,
	}, &updated); err != nil {
		t.Fatalf("get migrated ConfigMap: %v", err)
	}
	if got := updated.Labels[ProviderConfigLabel]; got != ProviderConfigLabelValue {
		t.Fatalf("provider label = %q, want %q", got, ProviderConfigLabelValue)
	}
	if got := updated.Labels["existing"]; got != "label" {
		t.Fatalf("existing label = %q, want %q", got, "label")
	}
	if got := updated.Data["API_PORT"]; got != "8080" {
		t.Fatalf("ConfigMap data = %q, want %q", got, "8080")
	}

	var untouched corev1.ConfigMap
	if err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name: "unrelated", Namespace: namespace,
	}, &untouched); err != nil {
		t.Fatalf("get unrelated ConfigMap: %v", err)
	}
	if _, ok := untouched.Labels[ProviderConfigLabel]; ok {
		t.Fatal("unrelated ConfigMap received provider label")
	}

	changed, err = BackfillLegacyProviderConfigLabel(context.Background(), k8sClient, namespace)
	if err != nil {
		t.Fatalf("second BackfillLegacyProviderConfigLabel() error = %v", err)
	}
	if changed {
		t.Fatal("second BackfillLegacyProviderConfigLabel() changed = true, want false")
	}
}

func TestBackfillLegacyProviderConfigLabelMissingConfigMap(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}

	changed, err := BackfillLegacyProviderConfigLabel(
		context.Background(),
		fake.NewClientBuilder().WithScheme(scheme).Build(),
		"krkn-operator-system",
	)
	if err != nil {
		t.Fatalf("BackfillLegacyProviderConfigLabel() error = %v", err)
	}
	if changed {
		t.Fatal("BackfillLegacyProviderConfigLabel() changed = true, want false")
	}
}
