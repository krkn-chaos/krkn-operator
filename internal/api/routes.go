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

// API version constants
// When bumping API version, only change APIVersion constant
const (
	// APIVersion is the current API version
	APIVersion = "v1"

	// APIBasePath is the base path for all API endpoints
	APIBasePath = "/api/" + APIVersion
)

// Authentication endpoints
const (
	AuthBasePath     = APIBasePath + "/auth"
	AuthIsRegistered = AuthBasePath + "/is-registered"
	AuthRegister     = AuthBasePath + "/register"
	AuthLogin        = AuthBasePath + "/login"
	AuthRefresh      = AuthBasePath + "/refresh"
	AuthLogout       = AuthBasePath + "/logout"
)

// Core resource endpoints
const (
	HealthPath   = APIBasePath + "/health"
	ClustersPath = APIBasePath + "/clusters"
	NodesPath    = APIBasePath + "/nodes"
)

// Legacy targets endpoints (deprecated, use OperatorTargetsPath)
const (
	TargetsPath = APIBasePath + "/targets"
)

// Scenarios endpoints
const (
	ScenariosPath        = APIBasePath + "/scenarios"
	ScenariosDetailPath  = ScenariosPath + "/detail"
	ScenariosGlobalsPath = ScenariosPath + "/globals"
	ScenariosRunPath     = ScenariosPath + "/run"
	ScenariosRunJobsPath = ScenariosRunPath + "/jobs"
)

// Dashboard endpoints
const (
	DashboardPath           = APIBasePath + "/dashboard"
	DashboardActiveRunsPath = DashboardPath + "/active-runs"
)

// User management endpoints
const (
	UsersPath  = APIBasePath + "/users"
	GroupsPath = APIBasePath + "/groups"
)

// Provider endpoints
const (
	ProvidersPath      = APIBasePath + "/providers"
	ProviderConfigPath = APIBasePath + "/provider-config"
)

// Operator configuration endpoints
const (
	OperatorPath        = APIBasePath + "/operator"
	OperatorTargetsPath = OperatorPath + "/targets"
)
