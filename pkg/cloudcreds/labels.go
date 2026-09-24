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
	ComponentCloudCredential = "cloud-credential" // #nosec G101 -- Kubernetes label component name, not a credential value
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

// ResolveAccessControlForUpdate returns the groups and availableToAll values that
// should be written on update. Omitted pointer fields keep the existing Secret
// access labels so partial updates (e.g. rotating a region) cannot silently
// revoke group or public access.
//
// When either ACL field is present in the request, omitted sibling fields use
// safe defaults (empty groups / false) so an explicit public or group change
// fully replaces the prior ACL rather than merging stale labels.
func ResolveAccessControlForUpdate(
	existingLabels map[string]string,
	groups *[]string,
	availableToAll *bool,
) (resolvedGroups []string, resolvedAvailableToAll bool) {
	aclPresent := groups != nil || availableToAll != nil
	if !aclPresent {
		return ExtractGroupsFromLabels(existingLabels), existingLabels[AvailableToAllLabel] == "true"
	}

	if groups != nil {
		resolvedGroups = append([]string(nil), (*groups)...)
	}
	if availableToAll != nil {
		resolvedAvailableToAll = *availableToAll
	}
	return resolvedGroups, resolvedAvailableToAll
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
