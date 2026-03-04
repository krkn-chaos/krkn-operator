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
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/krkn-chaos/krkn-operator/pkg/configstore"
)

func TestSyncConfigMapToStore(t *testing.T) {
	tests := []struct {
		name          string
		configMap     *corev1.ConfigMap
		expectedStore map[string]string
		expectError   bool
		errorMsg      string
	}{
		{
			name: "single key-value pair",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Data: map[string]string{
					"KEY1": "value1",
				},
			},
			expectedStore: map[string]string{
				"KEY1": "value1",
			},
		},
		{
			name: "multiple key-value pairs",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Data: map[string]string{
					"KEY1": "value1",
					"KEY2": "value2",
					"KEY3": "value3",
				},
			},
			expectedStore: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
				"KEY3": "value3",
			},
		},
		{
			name: "empty ConfigMap.Data",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Data:       map[string]string{},
			},
			expectedStore: map[string]string{},
		},
		{
			name: "empty value",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Data: map[string]string{
					"EMPTY_KEY": "",
				},
			},
			expectedStore: map[string]string{
				"EMPTY_KEY": "",
			},
		},
		{
			name: "special characters in keys and values",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Data: map[string]string{
					"ACM_SECRET_LOCAL_CLUSTER": "application-manager",
					"DB_HOST":                  "localhost:5432",
					"URL":                      "https://example.com/path?query=value",
				},
			},
			expectedStore: map[string]string{
				"ACM_SECRET_LOCAL_CLUSTER": "application-manager",
				"DB_HOST":                  "localhost:5432",
				"URL":                      "https://example.com/path?query=value",
			},
		},
		{
			name: "multiline values",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Data: map[string]string{
					"MULTILINE": "line1\nline2\nline3",
				},
			},
			expectedStore: map[string]string{
				"MULTILINE": "line1\nline2\nline3",
			},
		},
		{
			name: "overwrite existing store values",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Data: map[string]string{
					"KEY1": "new_value",
				},
			},
			expectedStore: map[string]string{
				"KEY1": "new_value",
			},
		},
		{
			name:        "nil configMap returns error",
			configMap:   nil,
			expectError: true,
			errorMsg:    "configMap is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh store for each test
			store := kvstore.Get()

			// Clear the store before test
			snapshot := store.Snapshot()
			for k := range snapshot {
				store.Delete(k)
			}

			// Pre-populate store if testing overwrite scenario
			if tt.name == "overwrite existing store values" {
				store.SetValue("KEY1", "old_value")
			}

			err := SyncConfigMapToStore(tt.configMap, store)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("expected error message %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify all expected keys are in the store
			for key, expectedValue := range tt.expectedStore {
				value, ok := store.GetValue(key)
				if !ok {
					t.Errorf("key %s not found in store", key)
					continue
				}
				if value != expectedValue {
					t.Errorf("key %s: got %q, want %q", key, value, expectedValue)
				}
			}

			// Verify no extra keys are in the store
			storeSnapshot := store.Snapshot()
			for key := range storeSnapshot {
				if _, expected := tt.expectedStore[key]; !expected {
					t.Errorf("unexpected key %s in store with value %q", key, storeSnapshot[key])
				}
			}

			// Clean up
			for k := range storeSnapshot {
				store.Delete(k)
			}
		})
	}
}

func TestSyncConfigMapToStore_NilStore(t *testing.T) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Data: map[string]string{
			"KEY1": "value1",
		},
	}

	err := SyncConfigMapToStore(configMap, nil)

	if err == nil {
		t.Error("expected error for nil store but got nil")
	}

	expectedErrMsg := "store is nil"
	if err.Error() != expectedErrMsg {
		t.Errorf("expected error message %q, got %q", expectedErrMsg, err.Error())
	}
}

func TestWriteConfigMapData(t *testing.T) {
	tests := []struct {
		name             string
		configMap        *corev1.ConfigMap
		data             map[string]string
		expectedData     map[string]string
		expectError      bool
		errorMsg         string
		initializeData   bool
		initialData      map[string]string
	}{
		{
			name:      "single key-value pair",
			configMap: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			data: map[string]string{
				"KEY1": "value1",
			},
			expectedData: map[string]string{
				"KEY1": "value1",
			},
		},
		{
			name:      "multiple key-value pairs",
			configMap: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			data: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
				"KEY3": "value3",
			},
			expectedData: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
				"KEY3": "value3",
			},
		},
		{
			name:         "empty data map",
			configMap:    &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			data:         map[string]string{},
			expectedData: map[string]string{},
		},
		{
			name:      "empty value",
			configMap: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			data: map[string]string{
				"EMPTY_KEY": "",
			},
			expectedData: map[string]string{
				"EMPTY_KEY": "",
			},
		},
		{
			name:      "special characters in keys and values",
			configMap: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			data: map[string]string{
				"ACM_SECRET_LOCAL_CLUSTER": "application-manager",
				"DB_HOST":                  "localhost:5432",
				"URL":                      "https://example.com/path?query=value",
			},
			expectedData: map[string]string{
				"ACM_SECRET_LOCAL_CLUSTER": "application-manager",
				"DB_HOST":                  "localhost:5432",
				"URL":                      "https://example.com/path?query=value",
			},
		},
		{
			name:      "multiline values",
			configMap: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			data: map[string]string{
				"MULTILINE": "line1\nline2\nline3",
			},
			expectedData: map[string]string{
				"MULTILINE": "line1\nline2\nline3",
			},
		},
		{
			name:      "configMap.Data initially nil",
			configMap: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			data: map[string]string{
				"KEY1": "value1",
			},
			expectedData: map[string]string{
				"KEY1": "value1",
			},
		},
		{
			name:           "overwrite existing ConfigMap.Data",
			configMap:      &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			initializeData: true,
			initialData: map[string]string{
				"KEY1": "old_value",
				"KEY2": "keep_me",
			},
			data: map[string]string{
				"KEY1": "new_value",
			},
			expectedData: map[string]string{
				"KEY1": "new_value",
				"KEY2": "keep_me",
			},
		},
		{
			name:        "nil configMap returns error",
			configMap:   nil,
			data:        map[string]string{"KEY1": "value1"},
			expectError: true,
			errorMsg:    "configMap is nil",
		},
		{
			name:        "nil data returns error",
			configMap:   &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test"}},
			data:        nil,
			expectError: true,
			errorMsg:    "data is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize configMap.Data if needed
			if tt.initializeData {
				tt.configMap.Data = make(map[string]string)
				for k, v := range tt.initialData {
					tt.configMap.Data[k] = v
				}
			}

			err := WriteConfigMapData(tt.configMap, tt.data)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("expected error message %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify all expected keys are in ConfigMap.Data
			for key, expectedValue := range tt.expectedData {
				value, ok := tt.configMap.Data[key]
				if !ok {
					t.Errorf("key %s not found in ConfigMap.Data", key)
					continue
				}
				if value != expectedValue {
					t.Errorf("key %s: got %q, want %q", key, value, expectedValue)
				}
			}

			// Verify no extra keys are in ConfigMap.Data
			for key := range tt.configMap.Data {
				if _, expected := tt.expectedData[key]; !expected {
					t.Errorf("unexpected key %s in ConfigMap.Data with value %q", key, tt.configMap.Data[key])
				}
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that WriteConfigMapData + SyncConfigMapToStore produce identical results
	originalData := map[string]string{
		"ACM_SECRET_LOCAL_CLUSTER": "application-manager",
		"ACM_NAMESPACE":            "open-cluster-management",
		"DB_HOST":                  "localhost",
		"API_PORT":                 "8080",
	}

	// Write to ConfigMap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
	}
	err := WriteConfigMapData(configMap, originalData)
	if err != nil {
		t.Fatalf("WriteConfigMapData failed: %v", err)
	}

	// Sync to store
	store := kvstore.Get()
	snapshot := store.Snapshot()
	for k := range snapshot {
		store.Delete(k)
	}

	err = SyncConfigMapToStore(configMap, store)
	if err != nil {
		t.Fatalf("SyncConfigMapToStore failed: %v", err)
	}

	// Verify store contains all original data
	storeSnapshot := store.Snapshot()
	for key, expectedValue := range originalData {
		value, ok := storeSnapshot[key]
		if !ok {
			t.Errorf("key %s not found in store after round-trip", key)
			continue
		}
		if value != expectedValue {
			t.Errorf("key %s: got %q, want %q", key, value, expectedValue)
		}
	}

	// Verify no extra keys
	for key := range storeSnapshot {
		if _, expected := originalData[key]; !expected {
			t.Errorf("unexpected key %s in store after round-trip", key)
		}
	}

	// Clean up
	for k := range storeSnapshot {
		store.Delete(k)
	}
}
