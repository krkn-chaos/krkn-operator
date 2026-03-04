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

package configmap

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/krkn-chaos/krkn-operator/pkg/configstore"
)

// SyncConfigMapToStore syncs a native Kubernetes ConfigMap (key-value format) to kvstore.
// Each key-value pair in configMap.Data is directly copied to the store.
// This function is designed for use in reconcile loops across the operator ecosystem.
//
// Returns error if configMap or store is nil.
func SyncConfigMapToStore(configMap *corev1.ConfigMap, store *kvstore.Store) error {
	if configMap == nil {
		return fmt.Errorf("configMap is nil")
	}

	if store == nil {
		return fmt.Errorf("store is nil")
	}

	// Direct copy from ConfigMap.Data to store
	for key, value := range configMap.Data {
		store.SetValue(key, value)
	}

	return nil
}

// WriteConfigMapData writes a map of key-value pairs to a ConfigMap's Data field.
// This function is the counterpart to SyncConfigMapToStore, ensuring compatibility
// across the operator ecosystem.
//
// Returns error if configMap or data is nil.
func WriteConfigMapData(configMap *corev1.ConfigMap, data map[string]string) error {
	if configMap == nil {
		return fmt.Errorf("configMap is nil")
	}

	if data == nil {
		return fmt.Errorf("data is nil")
	}

	// Initialize Data map if nil
	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	// Copy all key-value pairs to ConfigMap.Data
	for key, value := range data {
		configMap.Data[key] = value
	}

	return nil
}
