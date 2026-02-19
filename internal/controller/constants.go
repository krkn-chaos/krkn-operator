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

package controller

const (
	// CleanupThresholdSeconds is the age threshold in seconds for cleaning up old completed resources
	// Used by KrknTargetRequest and KrknOperatorTargetProviderConfig controllers
	CleanupThresholdSeconds = 3600 // 1 hour
)
