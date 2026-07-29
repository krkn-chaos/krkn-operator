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

// Package files provides types and utilities for file management in the krkn-operator.
// It includes request/response types for file CRUD operations, file references for scenario runs,
// and validation utilities for file content and metadata.
package files

// CreateFileRequest represents a request to create a new file ConfigMap
type CreateFileRequest struct {
	// FileName is the key in the ConfigMap data (the actual file name)
	FileName string `json:"fileName"`
	// Content is the file content
	Content string `json:"content"`
	// StudioLayout is optional frontend visual layout data (stored as separate ConfigMap entry)
	StudioLayout string `json:"studioLayout,omitempty"`
	// Description is an optional description of the file
	Description string `json:"description,omitempty"`
	// FileType is an optional file type category (e.g., "config", "script") - for user categorization
	FileType string `json:"fileType,omitempty"`
	// Groups is a list of group names that can access this file
	Groups []string `json:"groups,omitempty"`
	// AvailableToAll makes the file accessible to all users
	AvailableToAll bool `json:"availableToAll,omitempty"`
	// FilePurpose is an optional system-level classification (e.g., "workflow-template")
	FilePurpose string `json:"filePurpose,omitempty"`
}

// CreateFileResponse is the response for create file requests
type CreateFileResponse struct {
	Message string `json:"message"`
	// FileID is the generated UUID for this file (used for retrieval)
	FileID string `json:"fileId"`
}

// UpdateFileRequest represents a request to update a file ConfigMap
type UpdateFileRequest struct {
	// FileName is the key in the ConfigMap data (the actual file name)
	FileName string `json:"fileName"`
	// Content is the file content
	Content string `json:"content"`
	// StudioLayout is optional frontend visual layout data (stored as separate ConfigMap entry)
	StudioLayout string `json:"studioLayout,omitempty"`
	// Description is an optional description of the file
	Description string `json:"description,omitempty"`
	// FileType is an optional file type category (e.g., "config", "script") - for user categorization
	FileType string `json:"fileType,omitempty"`
	// Groups is a list of group names that can access this file
	Groups []string `json:"groups,omitempty"`
	// AvailableToAll makes the file accessible to all users
	AvailableToAll bool `json:"availableToAll,omitempty"`
	// FilePurpose is an optional system-level classification (e.g., "workflow-template")
	FilePurpose string `json:"filePurpose,omitempty"`
}

// UpdateFileResponse is the response for update file requests
type UpdateFileResponse struct {
	Message string `json:"message"`
	// FileID is the UUID for this file
	FileID string `json:"fileId"`
}

// DeleteFileResponse is the response for delete file requests
type DeleteFileResponse struct {
	Message string `json:"message"`
}

// FileResponse represents a file ConfigMap in API responses
type FileResponse struct {
	// FileID is the UUID identifier for this file
	FileID string `json:"fileId"`
	// FileName is the key in the ConfigMap data (the actual file name)
	FileName string `json:"fileName"`
	// Content is the file content
	Content string `json:"content"`
	// StudioLayout is optional frontend visual layout data
	StudioLayout string `json:"studioLayout,omitempty"`
	// Description is an optional description of the file
	Description string `json:"description,omitempty"`
	// FileType is an optional file type category (e.g., "config", "script") - for user categorization
	FileType string `json:"fileType,omitempty"`
	// FilePurpose is the system-level classification (e.g., "workflow-template")
	FilePurpose string `json:"filePurpose,omitempty"`
	// Groups is a list of group names that can access this file
	Groups []string `json:"groups,omitempty"`
	// AvailableToAll makes the file accessible to all users
	AvailableToAll bool `json:"availableToAll"`
	// CreatedAt is the timestamp when the file was created
	CreatedAt string `json:"createdAt,omitempty"`
	// CreatedBy is the email of the user who created the file
	CreatedBy string `json:"createdBy,omitempty"`
	// UpdatedAt is the timestamp when the file was last updated
	UpdatedAt string `json:"updatedAt,omitempty"`
	// UpdatedBy is the email of the user who last updated the file
	UpdatedBy string `json:"updatedBy,omitempty"`
}

// FileInfo represents minimal file information for user-facing lists
type FileInfo struct {
	// FileID is the UUID identifier for this file
	FileID string `json:"fileId"`
	// FileName is the key in the ConfigMap data (the actual file name)
	FileName string `json:"fileName"`
	// Description is an optional description of the file
	Description string `json:"description,omitempty"`
	// FileType is an optional file type category (e.g., "config", "script") - for user categorization
	FileType string `json:"fileType,omitempty"`
	// FilePurpose is the system-level classification (e.g., "workflow-template")
	FilePurpose string `json:"filePurpose,omitempty"`
	// CreatedAt is the timestamp when the file was created (ISO 8601)
	CreatedAt string `json:"createdAt,omitempty"`
	// UpdatedAt is the timestamp when the file was last updated (ISO 8601)
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// ListFilesResponse is the response for list files requests
type ListFilesResponse struct {
	// Files is the list of file ConfigMaps
	Files []FileResponse `json:"files"`
	// Total is the total number of files returned
	Total int `json:"total"`
}

// AvailableFilesResponse is the response for available files requests
type AvailableFilesResponse struct {
	// Files is the list of files available to the current user
	Files []FileInfo `json:"files"`
}

// FileReference represents a reference to a managed file to mount in scenario pod
type FileReference struct {
	// FileID is the UUID of the file to mount
	FileID string `json:"fileId"`
	// MountPath is the absolute path where the file should be mounted
	MountPath string `json:"mountPath"`
}
