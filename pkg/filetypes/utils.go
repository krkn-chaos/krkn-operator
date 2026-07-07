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

package filetypes

import (
	"fmt"
	"regexp"
)

var (
	// hexColorPattern matches valid hex color codes (#RRGGBB)
	hexColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
)

// ValidateFileTypeName validates that a file type name is valid.
// Names must be non-empty and contain only valid characters.
func ValidateFileTypeName(name string) error {
	if name == "" {
		return fmt.Errorf("file type name cannot be empty")
	}

	// Additional validation could be added here
	// For now, we rely on Kubernetes resource name validation
	return nil
}

// ValidateColor validates that a color string is a valid hex color code.
// Empty strings are considered valid (uses UI default).
func ValidateColor(color string) error {
	if color == "" {
		return nil
	}

	if !hexColorPattern.MatchString(color) {
		return fmt.Errorf("invalid color format: %s (must be #RRGGBB hex format, e.g., #FF5733)", color)
	}

	return nil
}

// ValidateCreateRequest validates a CreateFileTypeRequest.
func ValidateCreateRequest(req *CreateFileTypeRequest) error {
	if err := ValidateFileTypeName(req.Name); err != nil {
		return err
	}

	if err := ValidateColor(req.Color); err != nil {
		return err
	}

	return nil
}

// ValidateUpdateRequest validates an UpdateFileTypeRequest.
func ValidateUpdateRequest(req *UpdateFileTypeRequest) error {
	if err := ValidateFileTypeName(req.Name); err != nil {
		return err
	}

	if err := ValidateColor(req.Color); err != nil {
		return err
	}

	return nil
}
