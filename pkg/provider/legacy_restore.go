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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/krkn-chaos/krknctl/pkg/backup"
	"github.com/krkn-chaos/krknctl/pkg/typing"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

const (
	legacyProviderDiscoveryTimeout = 30 * time.Second
	legacyProviderPollInterval     = 100 * time.Millisecond
)

var (
	legacyProviderRegistrationResource = schema.GroupVersionResource{
		Group: "krkn.krkn-chaos.dev", Version: "v1alpha1", Resource: "krknoperatortargetproviders",
	}
	legacyProviderConfigRequestResource = schema.GroupVersionResource{
		Group: "krkn.krkn-chaos.dev", Version: "v1alpha1", Resource: "krknoperatortargetproviderconfigs",
	}
	legacyConfigMapResource = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
)

// RestoreLegacyProviderConfigs migrates settings from the legacy raw-provider
// backup format. It discovers the destination provider and destination
// ConfigMap; it never copies source ConfigMap metadata or provider CRs.
func RestoreLegacyProviderConfigs(ctx context.Context, dynClient dynamic.Interface, namespace string, configs []backup.LegacyProviderConfig) error {
	if len(configs) == 0 {
		return nil
	}
	logger := log.FromContext(ctx)
	active, err := legacyActiveProviderNames(ctx, dynClient)
	if err != nil {
		return err
	}

	requested := make(map[string]struct{}, len(configs))
	for _, config := range configs {
		requested[config.Provider] = struct{}{}
	}
	needsDiscovery := false
	for providerName := range requested {
		if active[providerName] {
			needsDiscovery = true
			break
		}
		logger.Info("Skipping legacy provider settings", "provider", providerName, "reason", "provider is not active on the destination")
	}
	if !needsDiscovery {
		return nil
	}

	destinations, err := discoverLegacyProviderConfigData(ctx, dynClient, namespace)
	if err != nil {
		return err
	}

	var failures []string
	for _, source := range configs {
		destination, ok := destinations[source.Provider]
		if !ok || destination.ConfigMap == "" || destination.Namespace == "" {
			logger.Info("Skipping legacy provider settings", "provider", source.Provider, "reason", "provider did not publish destination configuration")
			continue
		}
		if destination.ConfigSchema == "" {
			logger.Info("Skipping legacy provider settings", "provider", source.Provider, "reason", "destination provider did not publish a schema")
			continue
		}

		compatible := make(map[string]string)
		for key, value := range source.Values {
			if err := validateLegacyProviderValue(key, value, destination.ConfigSchema); err != nil {
				logger.Info("Skipping legacy provider setting", "provider", source.Provider, "setting", key, "reason", "destination schema rejected setting")
				continue
			}
			compatible[key] = value
		}
		if len(compatible) == 0 {
			continue
		}
		if err := applyLegacyDestinationConfigMap(ctx, dynClient, destination, compatible); err != nil {
			failures = append(failures, fmt.Sprintf("provider %q: %v", source.Provider, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("legacy provider settings restore failed: %s", strings.Join(failures, "; "))
	}
	return nil
}

func legacyActiveProviderNames(ctx context.Context, dynClient dynamic.Interface) (map[string]bool, error) {
	list, err := dynClient.Resource(legacyProviderRegistrationResource).List(ctx, metav1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return map[string]bool{}, nil
		}
		return nil, fmt.Errorf("failed to list registered providers: %w", err)
	}
	active := make(map[string]bool, len(list.Items))
	for i := range list.Items {
		name, _, err := unstructured.NestedString(list.Items[i].Object, "spec", "operator-name")
		if err != nil || name == "" {
			continue
		}
		isActive, found, err := unstructured.NestedBool(list.Items[i].Object, "spec", "active")
		if err != nil {
			return nil, err
		}
		if !found {
			isActive = true
		}
		// A stale inactive registration must not override an active one with
		// the same provider identity in another namespace.
		active[name] = active[name] || isActive
	}
	return active, nil
}

func discoverLegacyProviderConfigData(ctx context.Context, dynClient dynamic.Interface, namespace string) (map[string]krknv1alpha1.ProviderConfigData, error) {
	requestUUID := uuid.New().String()
	request := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "krkn.krkn-chaos.dev/v1alpha1",
		"kind":       "KrknOperatorTargetProviderConfig",
		"metadata": map[string]interface{}{
			"name":      requestUUID,
			"namespace": namespace,
		},
		"spec": map[string]interface{}{"uuid": requestUUID},
	}}
	resource := dynClient.Resource(legacyProviderConfigRequestResource).Namespace(namespace)
	if _, err := resource.Create(ctx, request, metav1.CreateOptions{}); err != nil {
		return nil, fmt.Errorf("failed to request destination provider configuration: %w", err)
	}
	defer func() { _ = resource.Delete(context.Background(), requestUUID, metav1.DeleteOptions{}) }()

	deadline := time.NewTimer(legacyProviderDiscoveryTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(legacyProviderPollInterval)
	defer ticker.Stop()
	for {
		current, err := resource.Get(ctx, requestUUID, metav1.GetOptions{})
		if err == nil {
			status, _, statusErr := unstructured.NestedString(current.Object, "status", "status")
			if statusErr != nil {
				return nil, fmt.Errorf("failed to read destination provider configuration status: %w", statusErr)
			}
			if strings.EqualFold(status, "completed") {
				return legacyProviderConfigDataFromObject(current)
			}
		} else if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to read destination provider configuration: %w", err)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, fmt.Errorf("timed out waiting for destination provider configuration")
		case <-ticker.C:
		}
	}
}

func legacyProviderConfigDataFromObject(obj *unstructured.Unstructured) (map[string]krknv1alpha1.ProviderConfigData, error) {
	raw, found, err := unstructured.NestedMap(obj.Object, "status", "configData")
	if err != nil {
		return nil, fmt.Errorf("failed to read destination provider configuration data: %w", err)
	}
	if !found {
		return map[string]krknv1alpha1.ProviderConfigData{}, nil
	}
	result := make(map[string]krknv1alpha1.ProviderConfigData, len(raw))
	for name, value := range raw {
		fields, ok := value.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("provider %q returned invalid configuration data", name)
		}
		configMap, _, err := unstructured.NestedString(fields, "config-map")
		if err != nil {
			return nil, err
		}
		providerNamespace, _, err := unstructured.NestedString(fields, "namespace")
		if err != nil {
			return nil, err
		}
		schemaJSON, _, err := unstructured.NestedString(fields, "config-schema")
		if err != nil {
			return nil, err
		}
		result[name] = krknv1alpha1.ProviderConfigData{ConfigMap: configMap, Namespace: providerNamespace, ConfigSchema: schemaJSON}
	}
	return result, nil
}

func validateLegacyProviderValue(key, value, schemaJSON string) error {
	var rawFields []json.RawMessage
	if err := json.Unmarshal([]byte(schemaJSON), &rawFields); err != nil {
		return fmt.Errorf("invalid provider schema: %w", err)
	}
	fields := make([]typing.InputField, len(rawFields))
	for i, raw := range rawFields {
		if err := fields[i].UnmarshalJSON(raw); err != nil {
			return err
		}
	}
	for i := range fields {
		if fields[i].Variable != nil && *fields[i].Variable == key {
			valueCopy := value
			_, err := fields[i].Validate(&valueCopy)
			return err
		}
	}
	return fmt.Errorf("setting is not present in destination schema")
}

func applyLegacyDestinationConfigMap(ctx context.Context, dynClient dynamic.Interface, data krknv1alpha1.ProviderConfigData, values map[string]string) error {
	resource := dynClient.Resource(legacyConfigMapResource).Namespace(data.Namespace)
	configMap, err := resource.Get(ctx, data.ConfigMap, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		configMap = &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      data.ConfigMap,
				"namespace": data.Namespace,
				"labels": map[string]interface{}{
					ProviderConfigLabel: ProviderConfigLabelValue,
				},
			},
			"data": stringMapToInterface(values),
		}}
		_, err = resource.Create(ctx, configMap, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	current, _, err := unstructured.NestedStringMap(configMap.Object, "data")
	if err != nil {
		return err
	}
	if current == nil {
		current = map[string]string{}
	}
	for key, value := range values {
		current[key] = value
	}
	if err := unstructured.SetNestedStringMap(configMap.Object, current, "data"); err != nil {
		return err
	}
	_, err = resource.Update(ctx, configMap, metav1.UpdateOptions{})
	return err
}

func stringMapToInterface(values map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
