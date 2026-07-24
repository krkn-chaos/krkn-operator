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

package v2

// API version constants
const (
	// APIVersion is the v2 API version
	APIVersion = "v2"

	// APIBasePath is the base path for all v2 API endpoints
	APIBasePath = "/api/" + APIVersion
)

// REST endpoints (reuse v1 handlers for backward compatibility)
const (
	// Scenarios endpoints (same as v1)
	ScenariosPath        = APIBasePath + "/scenarios"
	ScenariosRunPath     = ScenariosPath + "/run"
	ScenariosRunJobsPath = ScenariosRunPath + "/jobs"

	// Graph Run endpoints (same as v1)
	GraphRunsPath = APIBasePath + "/graphruns"

	// Dashboard endpoints (same as v1)
	DashboardPath           = APIBasePath + "/dashboard"
	DashboardActiveRunsPath = DashboardPath + "/active-runs"
)

// WebSocket endpoints (NEW - real-time multiplexed APIs)
const (
	WebSocketBasePath = APIBasePath + "/ws"

	// Real-time scenario run updates (multiplexed - subscribe to multiple run IDs)
	WebSocketRunsPath = WebSocketBasePath + "/runs"

	// Real-time graph run updates (multiplexed - subscribe to multiple graph run IDs)
	WebSocketGraphRunsPath = WebSocketBasePath + "/graphruns"

	// Real-time dashboard active runs
	WebSocketDashboardActiveRunsPath = WebSocketBasePath + "/dashboard/active-runs"
)
