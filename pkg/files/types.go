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

package files

// CreateFileRequest represents a request to create a new file ConfigMap
type CreateFileRequest struct {
	// Name is the ConfigMap name (must be unique, RFC 1123 compliant)
	Name string `json:"name"`
	// FileName is the key in the ConfigMap data (the actual file name)
	FileName string `json:"fileName"`
	// Content is the file content
	Content string `json:"content"`
	// Description is an optional description of the file
	Description string `json:"description,omitempty"`
	// FileType is an optional file type category (e.g., "config", "script")
	FileType string `json:"fileType,omitempty"`
	// Groups is a list of group names that can access this file
	Groups []string `json:"groups,omitempty"`
	// AvailableToAll makes the file accessible to all users
	AvailableToAll bool `json:"availableToAll,omitempty"`
}

// CreateFileResponse is the response for create file requests
type CreateFileResponse struct {
	Message string `json:"message"`
	Name    string `json:"name"`
}

// UpdateFileRequest represents a request to update a file ConfigMap
type UpdateFileRequest struct {
	// FileName is the key in the ConfigMap data (the actual file name)
	FileName string `json:"fileName"`
	// Content is the file content
	Content string `json:"content"`
	// Description is an optional description of the file
	Description string `json:"description,omitempty"`
	// FileType is an optional file type category (e.g., "config", "script")
	FileType string `json:"fileType,omitempty"`
	// Groups is a list of group names that can access this file
	Groups []string `json:"groups,omitempty"`
	// AvailableToAll makes the file accessible to all users
	AvailableToAll bool `json:"availableToAll,omitempty"`
}

// UpdateFileResponse is the response for update file requests
type UpdateFileResponse struct {
	Message string `json:"message"`
	Name    string `json:"name"`
}

// DeleteFileResponse is the response for delete file requests
type DeleteFileResponse struct {
	Message string `json:"message"`
}

// FileResponse represents a file ConfigMap in API responses
type FileResponse struct {
	Name           string   `json:"name"`
	FileName       string   `json:"fileName"`
	Content        string   `json:"content"`
	Description    string   `json:"description,omitempty"`
	FileType       string   `json:"fileType,omitempty"`
	Groups         []string `json:"groups,omitempty"`
	AvailableToAll bool     `json:"availableToAll"`
	CreatedAt      string   `json:"createdAt,omitempty"`
	CreatedBy      string   `json:"createdBy,omitempty"`
	UpdatedAt      string   `json:"updatedAt,omitempty"`
	UpdatedBy      string   `json:"updatedBy,omitempty"`
}

// FileInfo represents minimal file information for user-facing lists
type FileInfo struct {
	Name        string `json:"name"`
	FileName    string `json:"fileName"`
	Description string `json:"description,omitempty"`
	FileType    string `json:"fileType,omitempty"`
}

// ListFilesResponse is the response for list files requests
type ListFilesResponse struct {
	Files []FileResponse `json:"files"`
	Total int            `json:"total"`
}

// AvailableFilesResponse is the response for available files requests
type AvailableFilesResponse struct {
	Files []FileInfo `json:"files"`
}
