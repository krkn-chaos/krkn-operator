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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func TestCleanupOldResources(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	ctx := context.Background()
	namespace := "test-namespace"

	t.Run("deletes old resources", func(t *testing.T) {
		// Create test data
		now := metav1.Now()
		oneHourAgo := metav1.NewTime(now.Add(-1 * time.Hour))
		twoHoursAgo := metav1.NewTime(now.Add(-2 * time.Hour))

		oldConfig1 := &krknv1alpha1.KrknOperatorTargetProviderConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "old-config-1",
				Namespace: namespace,
			},
			Spec: krknv1alpha1.KrknOperatorTargetProviderConfigSpec{
				UUID: "uuid-1",
			},
			Status: krknv1alpha1.KrknOperatorTargetProviderConfigStatus{
				Created: &twoHoursAgo,
			},
		}

		oldConfig2 := &krknv1alpha1.KrknOperatorTargetProviderConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "old-config-2",
				Namespace: namespace,
			},
			Spec: krknv1alpha1.KrknOperatorTargetProviderConfigSpec{
				UUID: "uuid-2",
			},
			Status: krknv1alpha1.KrknOperatorTargetProviderConfigStatus{
				Created: &oneHourAgo,
			},
		}

		recentConfig := &krknv1alpha1.KrknOperatorTargetProviderConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "recent-config",
				Namespace: namespace,
			},
			Spec: krknv1alpha1.KrknOperatorTargetProviderConfigSpec{
				UUID: "uuid-3",
			},
			Status: krknv1alpha1.KrknOperatorTargetProviderConfigStatus{
				Created: &now,
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(oldConfig1, oldConfig2, recentConfig).
			Build()

		// Clean up resources older than 30 minutes (1800 seconds)
		deletedCount, err := CleanupOldResources(
			ctx,
			fakeClient,
			&krknv1alpha1.KrknOperatorTargetProviderConfigList{},
			namespace,
			1800, // 30 minutes
			func(obj client.Object) *metav1.Time {
				config := obj.(*krknv1alpha1.KrknOperatorTargetProviderConfig)
				return config.Status.Created
			},
		)

		assert.NoError(t, err)
		assert.Equal(t, 2, deletedCount, "Should delete 2 old resources")

		// Verify recent config still exists
		var remainingList krknv1alpha1.KrknOperatorTargetProviderConfigList
		err = fakeClient.List(ctx, &remainingList, client.InNamespace(namespace))
		assert.NoError(t, err)
		assert.Equal(t, 1, len(remainingList.Items), "Should have 1 remaining resource")
		assert.Equal(t, "recent-config", remainingList.Items[0].Name)
	})

	t.Run("handles missing Created field gracefully", func(t *testing.T) {
		configNoCreated := &krknv1alpha1.KrknOperatorTargetProviderConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "no-created",
				Namespace: namespace,
			},
			Spec: krknv1alpha1.KrknOperatorTargetProviderConfigSpec{
				UUID: "uuid-no-created",
			},
			Status: krknv1alpha1.KrknOperatorTargetProviderConfigStatus{
				Created: nil, // No Created timestamp
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(configNoCreated).
			Build()

		deletedCount, err := CleanupOldResources(
			ctx,
			fakeClient,
			&krknv1alpha1.KrknOperatorTargetProviderConfigList{},
			namespace,
			3600,
			func(obj client.Object) *metav1.Time {
				config := obj.(*krknv1alpha1.KrknOperatorTargetProviderConfig)
				return config.Status.Created
			},
		)

		assert.NoError(t, err)
		assert.Equal(t, 0, deletedCount, "Should not delete resources without Created timestamp")

		// Verify resource still exists
		var remainingList krknv1alpha1.KrknOperatorTargetProviderConfigList
		err = fakeClient.List(ctx, &remainingList, client.InNamespace(namespace))
		assert.NoError(t, err)
		assert.Equal(t, 1, len(remainingList.Items))
	})

	t.Run("validates input parameters", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		// Nil list
		_, err := CleanupOldResources(ctx, fakeClient, nil, namespace, 3600, func(obj client.Object) *metav1.Time { return nil })
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "emptyList cannot be nil")

		// Empty namespace
		_, err = CleanupOldResources(ctx, fakeClient, &krknv1alpha1.KrknOperatorTargetProviderConfigList{}, "", 3600, func(obj client.Object) *metav1.Time { return nil })
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "namespace cannot be empty")

		// Negative seconds
		_, err = CleanupOldResources(ctx, fakeClient, &krknv1alpha1.KrknOperatorTargetProviderConfigList{}, namespace, -1, func(obj client.Object) *metav1.Time { return nil })
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "olderThanSeconds must be positive")

		// Nil extractor function
		_, err = CleanupOldResources(ctx, fakeClient, &krknv1alpha1.KrknOperatorTargetProviderConfigList{}, namespace, 3600, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "getCreatedTime function cannot be nil")
	})

	t.Run("handles empty list", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		deletedCount, err := CleanupOldResources(
			ctx,
			fakeClient,
			&krknv1alpha1.KrknOperatorTargetProviderConfigList{},
			namespace,
			3600,
			func(obj client.Object) *metav1.Time {
				config := obj.(*krknv1alpha1.KrknOperatorTargetProviderConfig)
				return config.Status.Created
			},
		)

		assert.NoError(t, err)
		assert.Equal(t, 0, deletedCount, "Should delete 0 resources from empty list")
	})

	t.Run("handles panic in extractor function", func(t *testing.T) {
		now := metav1.Now()
		config := &krknv1alpha1.KrknOperatorTargetProviderConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-config",
				Namespace: namespace,
			},
			Spec: krknv1alpha1.KrknOperatorTargetProviderConfigSpec{
				UUID: "uuid-panic",
			},
			Status: krknv1alpha1.KrknOperatorTargetProviderConfigStatus{
				Created: &now,
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(config).
			Build()

		// Extractor that panics
		deletedCount, err := CleanupOldResources(
			ctx,
			fakeClient,
			&krknv1alpha1.KrknOperatorTargetProviderConfigList{},
			namespace,
			3600,
			func(obj client.Object) *metav1.Time {
				panic("test panic")
			},
		)

		// Should not panic, should handle gracefully
		assert.NoError(t, err)
		assert.Equal(t, 0, deletedCount, "Should skip resources when extractor panics")
	})
}

func TestExtractItemsFromList(t *testing.T) {
	t.Run("extracts items from list", func(t *testing.T) {
		list := &krknv1alpha1.KrknOperatorTargetProviderConfigList{
			Items: []krknv1alpha1.KrknOperatorTargetProviderConfig{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "config-1"},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "config-2"},
				},
			},
		}

		items, err := extractItemsFromList(list)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(items))
	})

	t.Run("handles empty list", func(t *testing.T) {
		list := &krknv1alpha1.KrknOperatorTargetProviderConfigList{
			Items: []krknv1alpha1.KrknOperatorTargetProviderConfig{},
		}

		items, err := extractItemsFromList(list)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(items))
	})
}
