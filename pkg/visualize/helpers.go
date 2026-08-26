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
	"time"
)

// BuildLabels creates the labels map for a krkn-visualize ConfigMap
func BuildLabels() map[string]string {
	return map[string]string{
		LabelManagedBy: ManagedByValue,
		LabelComponent: ComponentValue,
		LabelName:      "krkn-visualize",
	}
}

// BuildAnnotations creates the annotations map for a krkn-visualize ConfigMap
func BuildAnnotations(
	targetCluster string,
	namespace string,
	esConfigName string,
	status string,
	grafanaURL string,
	errorMessage string,
	createdBy string,
	autoDetectPrometheus bool,
	prometheusURL string,
) map[string]string {
	now := time.Now().UTC().Format(time.RFC3339)

	annotations := map[string]string{
		AnnotationTargetCluster: targetCluster,
		AnnotationNamespace:     namespace,
		AnnotationStatus:        status,
		AnnotationCreatedBy:     createdBy,
		AnnotationCreatedAt:     now,
		AnnotationUpdatedAt:     now,
	}

	if esConfigName != "" {
		annotations[AnnotationElasticsearchConfig] = esConfigName
	}

	if grafanaURL != "" {
		annotations[AnnotationGrafanaURL] = grafanaURL
	}

	if errorMessage != "" {
		annotations[AnnotationErrorMessage] = errorMessage
	}

	if autoDetectPrometheus {
		annotations[AnnotationAutoDetectPrometheus] = "true"
	} else {
		annotations[AnnotationAutoDetectPrometheus] = "false"
		if prometheusURL != "" {
			annotations[AnnotationPrometheusURL] = prometheusURL
		}
	}

	return annotations
}

// UpdateAnnotations updates the annotations map with new status/error/grafanaURL
func UpdateAnnotations(annotations map[string]string, status, grafanaURL, errorMessage string) map[string]string {
	if status != "" {
		annotations[AnnotationStatus] = status
	}
	if grafanaURL != "" {
		annotations[AnnotationGrafanaURL] = grafanaURL
	}
	if errorMessage != "" {
		annotations[AnnotationErrorMessage] = errorMessage
	} else {
		delete(annotations, AnnotationErrorMessage)
	}
	annotations[AnnotationUpdatedAt] = time.Now().UTC().Format(time.RFC3339)
	return annotations
}

// ParseInstanceResponse converts a ConfigMap to VisualizeInstanceResponse
func ParseInstanceResponse(name string, annotations map[string]string) VisualizeInstanceResponse {
	return VisualizeInstanceResponse{
		Name:                    name,
		TargetCluster:           annotations[AnnotationTargetCluster],
		Namespace:               annotations[AnnotationNamespace],
		ElasticsearchConfigName: annotations[AnnotationElasticsearchConfig],
		Status:                  annotations[AnnotationStatus],
		GrafanaURL:              annotations[AnnotationGrafanaURL],
		ErrorMessage:            annotations[AnnotationErrorMessage],
		CreatedAt:               annotations[AnnotationCreatedAt],
		CreatedBy:               annotations[AnnotationCreatedBy],
		UpdatedAt:               annotations[AnnotationUpdatedAt],
	}
}
