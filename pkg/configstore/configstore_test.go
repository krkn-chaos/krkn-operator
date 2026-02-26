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

package kvstore

import (
	"sync"
	"testing"
)

// TestGet_Singleton verifies that Get() always returns the same instance
func TestGet_Singleton(t *testing.T) {
	store1 := Get()
	store2 := Get()

	if store1 != store2 {
		t.Error("Get() should return the same instance (singleton pattern)")
	}
}

// TestSetValue_GetValue tests basic set and get operations
func TestSetValue_GetValue(t *testing.T) {
	store := Get()

	// Clean up before test
	store.Delete("test-key")

	// Test setting and getting a value
	store.SetValue("test-key", "test-value")

	value, ok := store.GetValue("test-key")
	if !ok {
		t.Error("Expected key to exist")
	}
	if value != "test-value" {
		t.Errorf("Expected value 'test-value', got '%s'", value)
	}
}

// TestGetValue_NonExistent tests getting a non-existent key
func TestGetValue_NonExistent(t *testing.T) {
	store := Get()

	value, ok := store.GetValue("non-existent-key")
	if ok {
		t.Error("Expected key to not exist")
	}
	if value != "" {
		t.Errorf("Expected empty string for non-existent key, got '%s'", value)
	}
}

// TestSetValue_Update tests updating an existing value
func TestSetValue_Update(t *testing.T) {
	store := Get()

	store.SetValue("update-key", "initial-value")
	store.SetValue("update-key", "updated-value")

	value, ok := store.GetValue("update-key")
	if !ok {
		t.Error("Expected key to exist")
	}
	if value != "updated-value" {
		t.Errorf("Expected value 'updated-value', got '%s'", value)
	}
}

// TestDelete tests deleting a key
func TestDelete(t *testing.T) {
	store := Get()

	// Set a value
	store.SetValue("delete-key", "delete-value")

	// Verify it exists
	if !store.Exists("delete-key") {
		t.Error("Expected key to exist before deletion")
	}

	// Delete it
	store.Delete("delete-key")

	// Verify it doesn't exist
	if store.Exists("delete-key") {
		t.Error("Expected key to not exist after deletion")
	}

	// Verify GetValue returns false
	_, ok := store.GetValue("delete-key")
	if ok {
		t.Error("Expected GetValue to return false for deleted key")
	}
}

// TestDelete_NonExistent tests deleting a non-existent key (should not panic)
func TestDelete_NonExistent(t *testing.T) {
	store := Get()

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Delete on non-existent key should not panic: %v", r)
		}
	}()

	store.Delete("non-existent-delete-key")
}

// TestExists tests the Exists method
func TestExists(t *testing.T) {
	store := Get()

	store.Delete("exists-key")

	// Should not exist
	if store.Exists("exists-key") {
		t.Error("Expected key to not exist")
	}

	// Set value
	store.SetValue("exists-key", "value")

	// Should exist
	if !store.Exists("exists-key") {
		t.Error("Expected key to exist")
	}

	// Delete it
	store.Delete("exists-key")

	// Should not exist again
	if store.Exists("exists-key") {
		t.Error("Expected key to not exist after deletion")
	}
}

// TestSnapshot tests the Snapshot method
func TestSnapshot(t *testing.T) {
	store := Get()

	// Clean slate
	snapshot := store.Snapshot()
	for k := range snapshot {
		store.Delete(k)
	}

	// Add some values
	testData := map[string]string{
		"snap-key1": "value1",
		"snap-key2": "value2",
		"snap-key3": "value3",
	}

	for k, v := range testData {
		store.SetValue(k, v)
	}

	// Get snapshot
	snapshot = store.Snapshot()

	// Verify all keys are in snapshot
	for k, expectedValue := range testData {
		actualValue, ok := snapshot[k]
		if !ok {
			t.Errorf("Expected key '%s' in snapshot", k)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("For key '%s', expected value '%s', got '%s'", k, expectedValue, actualValue)
		}
	}

	// Verify snapshot is a copy (modifying it doesn't affect store)
	snapshot["snap-key1"] = "modified"

	value, _ := store.GetValue("snap-key1")
	if value == "modified" {
		t.Error("Snapshot should be a copy, not a reference to internal map")
	}
	if value != "value1" {
		t.Errorf("Original value should be unchanged, expected 'value1', got '%s'", value)
	}

	// Clean up
	for k := range testData {
		store.Delete(k)
	}
}

// TestConcurrentAccess tests thread-safety with concurrent operations
func TestConcurrentAccess(t *testing.T) {
	store := Get()

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // 3 types of operations

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := "concurrent-key"
				value := "value"
				store.SetValue(key, value)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				store.GetValue("concurrent-key")
			}
		}(i)
	}

	// Concurrent deletes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				store.Delete("concurrent-key")
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// If we got here without deadlock or data race, test passed
}

// TestConcurrentSnapshot tests thread-safety of Snapshot with concurrent modifications
func TestConcurrentSnapshot(t *testing.T) {
	store := Get()

	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Concurrent snapshots
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = store.Snapshot()
			}
		}()
	}

	// Concurrent modifications while snapshots are being taken
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				store.SetValue("snapshot-concurrent-key", "value")
				store.Delete("snapshot-concurrent-key")
			}
		}(i)
	}

	wg.Wait()
}

// TestMultipleKeys tests storing and retrieving multiple keys
func TestMultipleKeys(t *testing.T) {
	store := Get()

	// Set multiple keys
	keys := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
		"key4": "value4",
		"key5": "value5",
	}

	for k, v := range keys {
		store.SetValue(k, v)
	}

	// Verify all keys
	for k, expectedValue := range keys {
		value, ok := store.GetValue(k)
		if !ok {
			t.Errorf("Expected key '%s' to exist", k)
			continue
		}
		if value != expectedValue {
			t.Errorf("For key '%s', expected '%s', got '%s'", k, expectedValue, value)
		}
	}

	// Clean up
	for k := range keys {
		store.Delete(k)
	}
}

// TestEmptyValue tests storing an empty string value
func TestEmptyValue(t *testing.T) {
	store := Get()

	store.SetValue("empty-key", "")

	value, ok := store.GetValue("empty-key")
	if !ok {
		t.Error("Expected key with empty value to exist")
	}
	if value != "" {
		t.Errorf("Expected empty string, got '%s'", value)
	}

	if !store.Exists("empty-key") {
		t.Error("Exists should return true for key with empty value")
	}

	// Clean up
	store.Delete("empty-key")
}
