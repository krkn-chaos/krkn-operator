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

package controller

import (
	"context"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// checkProviderActive verifies if the specified provider is registered and active.
// Returns true if the provider exists and is active, false otherwise.
// Logs appropriate messages for not found and inactive cases.
func checkProviderActive(ctx context.Context, c client.Client, operatorName string) (bool, *krknv1alpha1.KrknOperatorTargetProviderList, error) {
	logger := log.FromContext(ctx)

	// List all KrknOperatorTargetProviders
	providerList := &krknv1alpha1.KrknOperatorTargetProviderList{}
	if err := c.List(ctx, providerList); err != nil {
		logger.Error(err, "Failed to list KrknOperatorTargetProviders")
		return false, nil, err
	}

	// Find this operator's provider
	var thisProvider *krknv1alpha1.KrknOperatorTargetProvider
	for i := range providerList.Items {
		if providerList.Items[i].Spec.OperatorName == operatorName {
			thisProvider = &providerList.Items[i]
			break
		}
	}

	if thisProvider == nil {
		logger.Info("Provider not found, skipping reconcile", "provider-name", operatorName)
		return false, providerList, nil
	}

	if !thisProvider.Spec.Active {
		logger.Info("Provider is not active, skipping reconcile", "provider-name", operatorName)
		return false, providerList, nil
	}

	logger.Info("Provider is active, proceeding with reconcile", "provider-name", operatorName)
	return true, providerList, nil
}

// countActiveProviders counts the number of active providers in the given list.
// Returns the count and a slice of active provider names.
func countActiveProviders(providerList *krknv1alpha1.KrknOperatorTargetProviderList) (int, []string) {
	activeCount := 0
	activeNames := []string{}

	for _, provider := range providerList.Items {
		if provider.Spec.Active {
			activeCount++
			activeNames = append(activeNames, provider.Spec.OperatorName)
		}
	}

	return activeCount, activeNames
}
