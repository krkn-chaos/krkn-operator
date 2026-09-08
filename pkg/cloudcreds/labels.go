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

package cloudcreds

import (
	"strings"
	"time"

	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

// Label and annotation keys for cloud credential Secrets
const (
	AppNameLabel      = "app.kubernetes.io/name"
	AppComponentLabel = "app.kubernetes.io/component"

	ProviderTypeLabel   = "cloudcreds.krkn.krkn-chaos.dev/provider-type"
	AvailableToAllLabel = "cloudcreds.krkn.krkn-chaos.dev/available-to-all"

	DescriptionAnnotation = "cloudcreds.krkn.krkn-chaos.dev/description"
	CreatedByAnnotation   = "cloudcreds.krkn.krkn-chaos.dev/created-by"
	CreatedAtAnnotation   = "cloudcreds.krkn.krkn-chaos.dev/created-at"
	UpdatedByAnnotation   = "cloudcreds.krkn.krkn-chaos.dev/updated-by"
	UpdatedAtAnnotation   = "cloudcreds.krkn.krkn-chaos.dev/updated-at"

	AppName                  = "krkn-operator"
	ComponentCloudCredential = "cloud-credential"
)

// BuildLabels creates the labels map for a cloud credential Secret
func BuildLabels(provider string, groups []string, availableToAll bool) map[string]string {
	labels := map[string]string{
		AppNameLabel:      AppName,
		AppComponentLabel: ComponentCloudCredential,
		ProviderTypeLabel: provider,
	}

	if availableToAll {
		labels[AvailableToAllLabel] = "true"
	}

	for _, groupName := range groups {
		groupLabel := groupauth.GroupLabelKey(groupName)
		labels[groupLabel] = "true"
	}

	return labels
}

// BuildAnnotations creates the annotations map for a cloud credential Secret
func BuildAnnotations(description, createdBy string) map[string]string {
	annotations := map[string]string{
		CreatedByAnnotation: createdBy,
		CreatedAtAnnotation: time.Now().UTC().Format(time.RFC3339),
	}

	if description != "" {
		annotations[DescriptionAnnotation] = description
	}

	return annotations
}

// UpdateAnnotations updates the annotations for a cloud credential Secret
func UpdateAnnotations(existing map[string]string, description, updatedBy string) map[string]string {
	updated := make(map[string]string)
	for k, v := range existing {
		updated[k] = v
	}

	updated[UpdatedByAnnotation] = updatedBy
	updated[UpdatedAtAnnotation] = time.Now().UTC().Format(time.RFC3339)

	if description != "" {
		updated[DescriptionAnnotation] = description
	} else {
		delete(updated, DescriptionAnnotation)
	}

	return updated
}

// ExtractGroupsFromLabels extracts group names from cloud credential Secret labels
func ExtractGroupsFromLabels(labels map[string]string) []string {
	groups := []string{}

	for key, value := range labels {
		if strings.HasPrefix(key, groupauth.GroupLabelPrefix) && value == "true" {
			groupName := strings.TrimPrefix(key, groupauth.GroupLabelPrefix)
			groups = append(groups, groupName)
		}
	}

	return groups
}
