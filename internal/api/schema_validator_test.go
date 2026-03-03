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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertBoolFieldsToStrings(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:        "boolean true to string",
			input:       `{"name":"test","variable":"TEST","type":"string","required":true}`,
			expected:    `{"name":"test","required":"true","type":"string","variable":"TEST"}`,
			expectError: false,
		},
		{
			name:        "boolean false to string",
			input:       `{"name":"test","variable":"TEST","type":"string","required":false}`,
			expected:    `{"name":"test","required":"false","type":"string","variable":"TEST"}`,
			expectError: false,
		},
		{
			name:        "secret boolean true to string",
			input:       `{"name":"test","variable":"TEST","type":"string","secret":true}`,
			expected:    `{"name":"test","secret":"true","type":"string","variable":"TEST"}`,
			expectError: false,
		},
		{
			name:        "both required and secret as booleans",
			input:       `{"name":"test","variable":"TEST","type":"string","required":true,"secret":false}`,
			expected:    `{"name":"test","required":"true","secret":"false","type":"string","variable":"TEST"}`,
			expectError: false,
		},
		{
			name:        "already string values unchanged",
			input:       `{"name":"test","variable":"TEST","type":"string","required":"true"}`,
			expected:    `{"name":"test","required":"true","type":"string","variable":"TEST"}`,
			expectError: false,
		},
		{
			name:        "no boolean fields",
			input:       `{"name":"test","variable":"TEST","type":"string"}`,
			expected:    `{"name":"test","type":"string","variable":"TEST"}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertBoolFieldsToStrings([]byte(tt.input))

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.JSONEq(t, tt.expected, string(result))
			}
		})
	}
}

func TestValidateValueAgainstSchema_WithBooleanRequired(t *testing.T) {
	// Schema with boolean required field (the problematic case)
	schema := `[
		{
			"name": "Cluster Name",
			"short_description": "The cluster name",
			"description": "Name of the cluster to target",
			"variable": "CLUSTER_NAME",
			"type": "string",
			"required": true,
			"validator": "^[a-zA-Z0-9-]+$"
		},
		{
			"name": "Optional Field",
			"short_description": "An optional field",
			"description": "This field is optional",
			"variable": "OPTIONAL_FIELD",
			"type": "string",
			"required": false
		}
	]`

	tests := []struct {
		name        string
		key         string
		value       interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid value for required field",
			key:         "CLUSTER_NAME",
			value:       "my-cluster-123",
			expectError: false,
		},
		{
			name:        "invalid value for required field",
			key:         "CLUSTER_NAME",
			value:       "invalid cluster!",
			expectError: true,
			errorMsg:    "failed to validate",
		},
		{
			name:        "valid value for optional field",
			key:         "OPTIONAL_FIELD",
			value:       "some-value",
			expectError: false,
		},
		{
			name:        "field not in schema",
			key:         "UNKNOWN_FIELD",
			value:       "value",
			expectError: true,
			errorMsg:    "not found in schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateValueAgainstSchema(tt.key, tt.value, schema)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
