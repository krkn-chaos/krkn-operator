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

package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

// ValidateValueAgainstSchema validates a single value against a JSON schema property
func ValidateValueAgainstSchema(key string, value interface{}, schemaJSON string) error {
	// Parse the full schema
	var fullSchema map[string]interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &fullSchema); err != nil {
		return fmt.Errorf("invalid schema JSON: %w", err)
	}

	// Extract properties section
	properties, ok := fullSchema["properties"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("schema has no properties section")
	}

	// Support nested keys (e.g., "api.port" means {"api": {"port": ...}})
	propertySchema, err := extractNestedProperty(properties, key)
	if err != nil {
		return err
	}

	// Build a mini-schema for validation
	miniSchema := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{key: propertySchema},
	}

	// Build document to validate
	document := map[string]interface{}{key: value}

	// Validate using gojsonschema
	schemaLoader := gojsonschema.NewGoLoader(miniSchema)
	documentLoader := gojsonschema.NewGoLoader(document)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid() {
		// Collect all validation errors
		var errMsgs []string
		for _, desc := range result.Errors() {
			errMsgs = append(errMsgs, desc.String())
		}
		return fmt.Errorf("failed to validate %s: %v - %s", key, value, strings.Join(errMsgs, "; "))
	}

	return nil
}

// extractNestedProperty extracts a nested property from a schema using dot notation
// For example, "api.port" will navigate to properties["api"]["properties"]["port"]
func extractNestedProperty(properties map[string]interface{}, key string) (interface{}, error) {
	// Split key by dots
	parts := strings.Split(key, ".")

	current := properties
	var propertySchema interface{}

	for i, part := range parts {
		// Get the property at this level
		prop, ok := current[part]
		if !ok {
			return nil, fmt.Errorf("field %s not found in schema", key)
		}

		// If this is the last part, we found the schema
		if i == len(parts)-1 {
			propertySchema = prop
			break
		}

		// Otherwise, navigate deeper
		propMap, ok := prop.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("field %s is not an object in schema", strings.Join(parts[:i+1], "."))
		}

		// Get nested properties
		nestedProps, ok := propMap["properties"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("field %s has no nested properties in schema", strings.Join(parts[:i+1], "."))
		}

		current = nestedProps
	}

	return propertySchema, nil
}
