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

// Package filetypes provides types and utilities for managing file type metadata
// in the krkn-operator file management system.
package filetypes

// CreateFileTypeRequest represents a request to create a new file type.
type CreateFileTypeRequest struct {
	// Name is the file type identifier (e.g., "config", "script")
	Name string `json:"name"`

	// Color is an optional hex color for UI display (e.g., "#FF5733")
	Color string `json:"color,omitempty"`
}

// UpdateFileTypeRequest represents a request to update an existing file type.
type UpdateFileTypeRequest struct {
	// Name is the file type identifier (cannot be changed, but included for consistency)
	Name string `json:"name"`

	// Color is an optional hex color for UI display
	Color string `json:"color,omitempty"`
}

// FileTypeResponse represents a file type in API responses.
type FileTypeResponse struct {
	// Name is the file type identifier
	Name string `json:"name"`

	// Color is the hex color for UI display
	Color string `json:"color,omitempty"`

	// CreatedAt is the creation timestamp in RFC3339 format
	CreatedAt string `json:"createdAt,omitempty"`

	// UsageCount is the number of files currently using this type
	UsageCount int `json:"usageCount"`
}

// ListFileTypesResponse represents a list of file types in API responses.
type ListFileTypesResponse struct {
	// FileTypes is the list of file type metadata
	FileTypes []FileTypeResponse `json:"fileTypes"`

	// Total is the total number of file types
	Total int `json:"total"`
}
