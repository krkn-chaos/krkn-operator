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

package visualize

import (
	"fmt"
	"regexp"
)

var (
	// Valid Kubernetes resource name pattern (lowercase alphanumeric and hyphens)
	namePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
)

// ValidateCreateRequest validates a CreateVisualizeRequest
func ValidateCreateRequest(req *CreateVisualizeRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}

	if !namePattern.MatchString(req.Name) {
		return fmt.Errorf("name must be lowercase alphanumeric with hyphens (e.g., 'my-visualize')")
	}

	if len(req.TargetClusters) == 0 {
		return fmt.Errorf("at least one target cluster is required")
	}

	if req.GrafanaPassword == "" {
		return fmt.Errorf("grafanaPassword is required")
	}

	if len(req.GrafanaPassword) < 4 {
		return fmt.Errorf("grafanaPassword must be at least 4 characters")
	}

	// Validate namespace if provided
	if req.Namespace != "" && !namePattern.MatchString(req.Namespace) {
		return fmt.Errorf("namespace must be a valid Kubernetes namespace name")
	}

	// Validate Prometheus config if not auto-detecting
	if !req.AutoDetectPrometheus && req.PrometheusURL == "" {
		return fmt.Errorf("prometheusUrl is required when autoDetectPrometheus is false")
	}

	return nil
}
