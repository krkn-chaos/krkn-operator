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

package provider_test

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/provider"
)

// ExampleCleanupOldResources demonstrates how to clean up old KrknOperatorTargetProviderConfig resources
func ExampleCleanupOldResources_providerConfig() {
	// Assume we have a Kubernetes client
	var c client.Client
	ctx := context.Background()
	namespace := "krkn-operator-system"

	// Clean up config requests older than 1 hour (3600 seconds)
	deletedCount, err := provider.CleanupOldResources(
		ctx,
		c,
		&krknv1alpha1.KrknOperatorTargetProviderConfigList{},
		namespace,
		3600, // Delete resources older than 1 hour
		func(obj client.Object) *metav1.Time {
			config := obj.(*krknv1alpha1.KrknOperatorTargetProviderConfig)
			return config.Status.Created
		},
	)

	if err != nil {
		fmt.Printf("Error during cleanup: %v\n", err)
		return
	}

	fmt.Printf("Cleaned up %d old config requests\n", deletedCount)
}

// ExampleCleanupOldResources_targetRequest demonstrates cleanup for KrknTargetRequest resources
func ExampleCleanupOldResources_targetRequest() {
	var c client.Client
	ctx := context.Background()
	namespace := "krkn-operator-system"

	// Clean up target requests older than 24 hours (86400 seconds)
	deletedCount, err := provider.CleanupOldResources(
		ctx,
		c,
		&krknv1alpha1.KrknTargetRequestList{},
		namespace,
		86400, // 24 hours
		func(obj client.Object) *metav1.Time {
			request := obj.(*krknv1alpha1.KrknTargetRequest)
			return request.Status.Created
		},
	)

	if err != nil {
		fmt.Printf("Error during cleanup: %v\n", err)
		return
	}

	fmt.Printf("Cleaned up %d old target requests\n", deletedCount)
}

// ExampleCleanupOldResources_cronJob demonstrates how to use this in a CronJob controller
func ExampleCleanupOldResources_cronJob() {
	var c client.Client
	ctx := context.Background()

	// This could be run periodically (e.g., every hour) by a CronJob or controller
	cleanupConfigs := func() error {
		deletedCount, err := provider.CleanupOldResources(
			ctx,
			c,
			&krknv1alpha1.KrknOperatorTargetProviderConfigList{},
			"krkn-operator-system",
			7200, // 2 hours
			func(obj client.Object) *metav1.Time {
				config := obj.(*krknv1alpha1.KrknOperatorTargetProviderConfig)
				return config.Status.Created
			},
		)
		if err != nil {
			return fmt.Errorf("failed to cleanup configs: %w", err)
		}
		fmt.Printf("Cleaned up %d old configs\n", deletedCount)
		return nil
	}

	// Run cleanup
	if err := cleanupConfigs(); err != nil {
		fmt.Printf("Cleanup failed: %v\n", err)
	}
}
