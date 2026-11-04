/*
Copyright 2026.

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

	"github.com/krkn-chaos/krknctl/pkg/backup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestRestoreLegacyProviderConfigs_AppliesCompatibleSettingsToDestinationConfig(t *testing.T) {
	const (
		destinationNamespace = "destination-provider"
		destinationConfigMap = "destination-config"
	)

	schemaJSON := `[
        {"name":"setting","variable":"SETTING","type":"string","required":"false"},
        {"name":"port","variable":"PORT","type":"number","required":"false"}
    ]`
	client := newLegacyRestoreDynamicClient(
		legacyProvider("krkn-operator", true),
		legacyConfigMap(destinationNamespace, destinationConfigMap, map[string]string{
			"KEEP": "existing",
		}),
	)
	client.PrependReactor("get", legacyProviderConfigRequestResource.Resource, func(action ktesting.Action) (bool, runtime.Object, error) {
		getAction := action.(ktesting.GetAction)
		return true, legacyProviderConfigRequest(getAction.GetNamespace(), getAction.GetName(), "krkn-operator", destinationNamespace, destinationConfigMap, schemaJSON), nil
	})

	err := RestoreLegacyProviderConfigs(context.Background(), client, "restore-ns", []backup.LegacyProviderConfig{{
		Provider: "krkn-operator",
		Values: map[string]string{
			"SETTING": "restored",
			"PORT":    "not-a-number",
		},
	}})
	if err != nil {
		t.Fatalf("RestoreLegacyProviderConfigs() error = %v", err)
	}

	configMap, err := client.Resource(legacyConfigMapResource).Namespace(destinationNamespace).Get(
		context.Background(), destinationConfigMap, metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("get destination ConfigMap: %v", err)
	}
	data, _, err := unstructured.NestedStringMap(configMap.Object, "data")
	if err != nil {
		t.Fatalf("read destination ConfigMap data: %v", err)
	}
	if data["SETTING"] != "restored" {
		t.Fatalf("SETTING = %q, want %q", data["SETTING"], "restored")
	}
	if data["KEEP"] != "existing" {
		t.Fatalf("KEEP = %q, want %q", data["KEEP"], "existing")
	}
	if _, found := data["PORT"]; found {
		t.Fatal("incompatible PORT setting was applied")
	}
}

func TestRestoreLegacyProviderConfigs_SkipsMissingProvider(t *testing.T) {
	for _, test := range []struct {
		name        string
		destination string
		source      string
	}{
		{name: "ACM backup to normal destination", destination: "krkn-operator", source: "krkn-operator-acm"},
		{name: "normal backup to ACM destination", destination: "krkn-operator-acm", source: "krkn-operator"},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := newLegacyRestoreDynamicClient(legacyProvider(test.destination, true))
			err := RestoreLegacyProviderConfigs(context.Background(), client, "restore-ns", []backup.LegacyProviderConfig{{
				Provider: test.source,
				Values:   map[string]string{"SETTING": "value"},
			}})
			if err != nil {
				t.Fatalf("RestoreLegacyProviderConfigs() error = %v", err)
			}

			configMaps, err := client.Resource(legacyConfigMapResource).Namespace("restore-ns").List(
				context.Background(), metav1.ListOptions{},
			)
			if err != nil {
				t.Fatalf("list destination ConfigMaps: %v", err)
			}
			if len(configMaps.Items) != 0 {
				t.Fatalf("created %d ConfigMaps for a missing provider", len(configMaps.Items))
			}
		})
	}
}

func newLegacyRestoreDynamicClient(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
	listKinds := map[schema.GroupVersionResource]string{
		legacyProviderRegistrationResource:  "KrknOperatorTargetProviderList",
		legacyProviderConfigRequestResource: "KrknOperatorTargetProviderConfigList",
		legacyConfigMapResource:             "ConfigMapList",
	}
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds, objects...)
}

func legacyProvider(name string, active bool) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "krkn.krkn-chaos.dev/v1alpha1",
		"kind":       "KrknOperatorTargetProvider",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": "provider-system",
		},
		"spec": map[string]interface{}{
			"operator-name": name,
			"active":        active,
		},
	}}
}

func legacyConfigMap(namespace, name string, data map[string]string) *unstructured.Unstructured {
	values := make(map[string]interface{}, len(data))
	for key, value := range data {
		values[key] = value
	}
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"labels": map[string]interface{}{
				ProviderConfigLabel: ProviderConfigLabelValue,
			},
		},
		"data": values,
	}}
}

func legacyProviderConfigRequest(namespace, name, provider, configNamespace, configMap, schemaJSON string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "krkn.krkn-chaos.dev/v1alpha1",
		"kind":       "KrknOperatorTargetProviderConfig",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
		"status": map[string]interface{}{
			"status": "Completed",
			"configData": map[string]interface{}{
				provider: map[string]interface{}{
					"config-map":    configMap,
					"namespace":     configNamespace,
					"config-schema": schemaJSON,
				},
			},
		},
	}}
}
