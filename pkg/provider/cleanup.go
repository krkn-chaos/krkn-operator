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

package provider

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CreatedTimeExtractor is a function that extracts the Created timestamp from a Kubernetes object
type CreatedTimeExtractor func(obj client.Object) *metav1.Time

// CleanupOldResources deletes all instances of a specific CRD type in a namespace
// whose Created field is older than the specified number of seconds.
//
// This function is idempotent and safe for concurrent execution by multiple operators.
// It handles conflicts gracefully (logs warnings but doesn't fail) and never panics.
//
// Parameters:
//   - ctx: Context for the operation
//   - c: Kubernetes client
//   - emptyList: An empty instance of the list type (e.g., &krknv1alpha1.KrknOperatorTargetProviderConfigList{})
//   - namespace: Namespace to search in
//   - olderThanSeconds: Age threshold in seconds - resources older than this will be deleted
//   - getCreatedTime: Function to extract the Created timestamp from an object
//
// Returns:
//   - deletedCount: Number of resources successfully deleted
//   - error: Non-nil only for critical errors (listing failures); deletion conflicts are logged but don't cause errors
//
// Example usage:
//
//	deletedCount, err := provider.CleanupOldResources(
//	    ctx,
//	    client,
//	    &krknv1alpha1.KrknOperatorTargetProviderConfigList{},
//	    "krkn-operator-system",
//	    3600, // Delete resources older than 1 hour
//	    func(obj client.Object) *metav1.Time {
//	        config := obj.(*krknv1alpha1.KrknOperatorTargetProviderConfig)
//	        return config.Status.Created
//	    },
//	)
func CleanupOldResources(
	ctx context.Context,
	c client.Client,
	emptyList client.ObjectList,
	namespace string,
	olderThanSeconds int64,
	getCreatedTime CreatedTimeExtractor,
) (int, error) {
	logger := log.FromContext(ctx).WithName("cleanup")

	// Validate inputs
	if emptyList == nil {
		return 0, fmt.Errorf("emptyList cannot be nil")
	}
	if namespace == "" {
		return 0, fmt.Errorf("namespace cannot be empty")
	}
	if olderThanSeconds <= 0 {
		return 0, fmt.Errorf("olderThanSeconds must be positive")
	}
	if getCreatedTime == nil {
		return 0, fmt.Errorf("getCreatedTime function cannot be nil")
	}

	// List all resources of this type in the namespace
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
	}

	if err := c.List(ctx, emptyList, listOpts...); err != nil {
		logger.Error(err, "Failed to list resources for cleanup")
		return 0, fmt.Errorf("failed to list resources: %w", err)
	}

	// Extract items from the list using reflection-safe approach
	items, err := extractItemsFromList(emptyList)
	if err != nil {
		logger.Error(err, "Failed to extract items from list")
		return 0, fmt.Errorf("failed to extract items: %w", err)
	}

	// Calculate the cutoff time
	cutoffTime := time.Now().Add(-time.Duration(olderThanSeconds) * time.Second)

	deletedCount := 0

	// Iterate through items and delete old ones
	for _, item := range items {
		obj, ok := item.(client.Object)
		if !ok {
			continue
		}

		// Extract Created timestamp using the provided function
		var createdTime *metav1.Time
		func() {
			// Use a defer-recover to catch any panics from the extractor function
			defer func() {
				if r := recover(); r != nil {
					createdTime = nil
				}
			}()
			createdTime = getCreatedTime(obj)
		}()

		// Skip if no Created time or if it's too recent
		if createdTime == nil || createdTime.Time.After(cutoffTime) {
			continue
		}

		// Resource is old enough - attempt to delete
		if err := deleteResourceSafely(ctx, c, obj, logger); err != nil {
			// Check error type
			if apierrors.IsNotFound(err) {
				// Resource already deleted by another operator, silently continue
				continue
			} else if apierrors.IsConflict(err) {
				logger.Info("Conflict during deletion (concurrent operation)",
					"resource", obj.GetName())
				continue
			} else {
				// For other errors, log as warning but continue
				logger.Info("Failed to delete resource",
					"resource", obj.GetName(),
					"error", err.Error())
				continue
			}
		}

		// Log successful deletion
		logger.Info("Deleted old resource",
			"resource", obj.GetName(),
			"age", time.Since(createdTime.Time))
		deletedCount++
	}

	return deletedCount, nil
}

// extractItemsFromList extracts the Items field from a client.ObjectList using reflection
func extractItemsFromList(list client.ObjectList) ([]interface{}, error) {
	// Use reflection to extract the Items field from the list
	listValue := reflect.ValueOf(list)

	// Dereference pointer if necessary
	if listValue.Kind() == reflect.Ptr {
		listValue = listValue.Elem()
	}

	// Check if the value is a struct
	if listValue.Kind() != reflect.Struct {
		return nil, fmt.Errorf("list is not a struct: %T", list)
	}

	// Get the Items field
	itemsField := listValue.FieldByName("Items")
	if !itemsField.IsValid() {
		return nil, fmt.Errorf("list does not have an Items field: %T", list)
	}

	// Check if Items is a slice
	if itemsField.Kind() != reflect.Slice {
		return nil, fmt.Errorf("Items field is not a slice: %T", list)
	}

	// Extract items into []interface{}
	itemCount := itemsField.Len()
	items := make([]interface{}, itemCount)
	for i := 0; i < itemCount; i++ {
		item := itemsField.Index(i)
		// If the item is addressable, take its address (to get a pointer)
		// This is needed because Kubernetes objects are often passed by pointer
		if item.CanAddr() {
			items[i] = item.Addr().Interface()
		} else {
			items[i] = item.Interface()
		}
	}

	return items, nil
}

// deleteResourceSafely attempts to delete a resource with error recovery
func deleteResourceSafely(ctx context.Context, c client.Client, obj client.Object, logger logr.Logger) (err error) {
	// Use defer-recover to ensure we never panic
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during deletion: %v", r)
			logger.Error(err, "Recovered from panic during resource deletion",
				"resource", obj.GetName())
		}
	}()

	// Attempt deletion
	return c.Delete(ctx, obj)
}
