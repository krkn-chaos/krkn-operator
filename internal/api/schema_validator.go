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

	// Unmarshal each field using InputField's custom UnmarshalJSON
	fields := make([]typing.InputField, len(rawFields))
	for i, raw := range rawFields {
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
