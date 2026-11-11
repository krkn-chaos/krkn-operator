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

package api

import "time"

// CreateRobotRequest is the body of POST /api/v1/robots.
type CreateRobotRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Groups      []string `json:"groups,omitempty"`
}

// UpdateRobotRequest is the body of PATCH /api/v1/robots/{name}.
// Only provided fields are updated; Groups replaces all group labels.
type UpdateRobotRequest struct {
	Description *string   `json:"description,omitempty"`
	Disabled    *bool     `json:"disabled,omitempty"`
	Groups      *[]string `json:"groups,omitempty"`
}

// RobotResponse is the admin view of a robot account. It never contains
// token values or hashes, only token metadata.
type RobotResponse struct {
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	Disabled       bool       `json:"disabled"`
	Groups         []string   `json:"groups"`
	TokenID        string     `json:"tokenId"`
	TokenIssuedAt  *time.Time `json:"tokenIssuedAt,omitempty"`
	TokenExpiresAt *time.Time `json:"tokenExpiresAt,omitempty"`
	LastUsedAt     *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

// RobotListResponse is the body of GET /api/v1/robots. MCPEnabled reports
// whether the operator's MCP endpoint is currently mounted, so the console
// can tell admins whether issued robot tokens are usable.
type RobotListResponse struct {
	Robots     []RobotResponse `json:"robots"`
	Total      int             `json:"total"`
	MCPEnabled bool            `json:"mcpEnabled"`
}

// IssueRobotTokenResponse is the one-time response of POST /api/v1/robots/{name}/token.
type IssueRobotTokenResponse struct {
	Token     string    `json:"token"`
	TokenID   string    `json:"tokenId"`
	ExpiresAt time.Time `json:"expiresAt"`
	Message   string    `json:"message"`
}
