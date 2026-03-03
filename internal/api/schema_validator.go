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

	"github.com/krkn-chaos/krknctl/pkg/typing"
)

// ValidateValueAgainstSchema validates a single value against typing.InputField schema
// The schema is a JSON array of typing.InputField objects
func ValidateValueAgainstSchema(key string, value interface{}, schemaJSON string) error {
	// Parse as raw JSON array first
	var rawFields []json.RawMessage
	if err := json.Unmarshal([]byte(schemaJSON), &rawFields); err != nil {
		return fmt.Errorf("invalid schema JSON: %w", err)
	}

	// Pre-process each field to convert boolean values to strings
	// This is a workaround for krknctl's typing.InputField.UnmarshalJSON expecting all values as strings
	processedFields := make([]json.RawMessage, len(rawFields))
	for i, raw := range rawFields {
		processed, err := convertBoolFieldsToStrings(raw)
		if err != nil {
			return fmt.Errorf("failed to process field %d: %w", i, err)
		}
		processedFields[i] = processed
	}

	// Unmarshal each field using InputField's custom UnmarshalJSON
	fields := make([]typing.InputField, len(processedFields))
	for i, raw := range processedFields {
		if err := fields[i].UnmarshalJSON(raw); err != nil {
			return fmt.Errorf("failed to unmarshal field %d: %w", i, err)
		}
	}

	// Find the field definition matching the key (check Variable field)
	var matchingField *typing.InputField
	for i := range fields {
		field := &fields[i]
		// The key should match the Variable field
		if field.Variable != nil && *field.Variable == key {
			matchingField = field
			break
		}
	}

	if matchingField == nil {
		return fmt.Errorf("field %s not found in schema", key)
	}

	// Convert value to string pointer (all values come as strings from the API)
	valueStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	// Use the typing.InputField.Validate() method
	_, err := matchingField.Validate(&valueStr)
	if err != nil {
		return fmt.Errorf("failed to validate %s: %w", key, err)
	}

	return nil
}

// convertBoolFieldsToStrings converts boolean values to strings in JSON
// This is a workaround for krknctl's typing.InputField.UnmarshalJSON which expects
// all field values as strings (it unmarshals to map[string]string internally)
func convertBoolFieldsToStrings(raw json.RawMessage) (json.RawMessage, error) {
	// Unmarshal to generic map to inspect types
	var fieldMap map[string]interface{}
	if err := json.Unmarshal(raw, &fieldMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal field: %w", err)
	}

	// Convert boolean values to strings for fields that krknctl expects as strings
	boolFields := []string{"required", "secret"}
	for _, fieldName := range boolFields {
		if val, exists := fieldMap[fieldName]; exists {
			// Check if it's a boolean
			if boolVal, ok := val.(bool); ok {
				// Convert to string representation
				if boolVal {
					fieldMap[fieldName] = "true"
				} else {
					fieldMap[fieldName] = "false"
				}
			}
		}
	}

	// Marshal back to JSON
	processed, err := json.Marshal(fieldMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal processed field: %w", err)
	}

	return processed, nil
}
