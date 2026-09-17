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

package elasticsearch

import (
	"crypto/x509"
	"fmt"
	"strings"

	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

func validateAccess(groups []string, availableToAll bool) error {
	if len(groups) > 0 && availableToAll {
		return fmt.Errorf("an Elasticsearch config cannot be both public and assigned to a group")
	}
	for _, group := range groups {
		sanitized := groupauth.SanitizeGroupName(strings.TrimSpace(group))
		if sanitized == "" {
			return fmt.Errorf("group name %q is invalid", group)
		}
		if len(sanitized) > 63 {
			return fmt.Errorf("group name %q exceeds the 63-character label limit", group)
		}
	}
	return nil
}

// ValidateCreateRequest validates a CreateElasticsearchConfigRequest.
func ValidateCreateRequest(req *CreateElasticsearchConfigRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.Host == "" {
		return fmt.Errorf("host is required")
	}
	if req.Port < 0 || req.Port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535")
	}
	return validateAccess(req.Groups, req.AvailableToAll)
}

// ValidateUpdateRequest validates an UpdateElasticsearchConfigRequest.
func ValidateUpdateRequest(req *UpdateElasticsearchConfigRequest) error {
	if req.Host == "" {
		return fmt.Errorf("host is required")
	}
	if req.Port < 0 || req.Port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535")
	}
	availableToAll := false
	if req.AvailableToAll != nil {
		availableToAll = *req.AvailableToAll
	}
	return validateAccess(req.Groups, availableToAll)
}
