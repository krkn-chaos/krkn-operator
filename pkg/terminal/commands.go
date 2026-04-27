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

package terminal

// SubcommandSpec defines a permitted subcommand
type SubcommandSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CommandSpec defines a permitted command and its allowed subcommands
type CommandSpec struct {
	Name               string           `json:"name"`
	Description        string           `json:"description"`
	AllowedSubcommands []SubcommandSpec `json:"subcommands"`
}

// BlockedFlagSpec defines a blocked flag
type BlockedFlagSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AvailableCommands defines all permitted commands and their subcommands
// This is the single source of truth for terminal command validation
var AvailableCommands = map[string]CommandSpec{
	"kubectl": {
		Name:        "kubectl",
		Description: "Kubernetes command-line tool",
		AllowedSubcommands: []SubcommandSpec{
			{Name: "get", Description: "Display one or many resources"},
			{Name: "describe", Description: "Show details of a specific resource"},
			{Name: "logs", Description: "Print container logs"},
			{Name: "top", Description: "Display resource usage (CPU/Memory)"},
			{Name: "explain", Description: "Get documentation for a resource"},
			{Name: "version", Description: "Print client and server version information"},
			{Name: "api-resources", Description: "Print supported API resources"},
			{Name: "api-versions", Description: "Print supported API versions"},
			{Name: "cluster-info", Description: "Display cluster information"},
		},
	},
	"oc": {
		Name:        "oc",
		Description: "OpenShift command-line tool",
		AllowedSubcommands: []SubcommandSpec{
			{Name: "get", Description: "Display one or many resources"},
			{Name: "describe", Description: "Show details of a specific resource"},
			{Name: "logs", Description: "Print container logs"},
			{Name: "top", Description: "Display resource usage (CPU/Memory)"},
			{Name: "explain", Description: "Get documentation for a resource"},
			{Name: "version", Description: "Print client and server version information"},
			{Name: "api-resources", Description: "Print supported API resources"},
			{Name: "api-versions", Description: "Print supported API versions"},
			{Name: "cluster-info", Description: "Display cluster information"},
		},
	},
}

// BlockedFlagsSpec defines all blocked flags and why they're blocked
var BlockedFlagsSpec = []BlockedFlagSpec{
	{Name: "watch", Description: "Streaming flag blocked in v1 - requires WebSocket/SSE"},
	{Name: "w", Description: "Short form of --watch"},
	{Name: "follow", Description: "Streaming flag blocked in v1 - requires WebSocket/SSE"},
	{Name: "f", Description: "Short form of --follow"},
	{Name: "watch-only", Description: "Streaming flag blocked in v1 - requires WebSocket/SSE"},
}

// IsCommandAllowed checks if a command name is in the allowed list
func IsCommandAllowed(command string) bool {
	_, exists := AvailableCommands[command]
	return exists
}

// IsSubcommandAllowed checks if a subcommand is allowed for a given command
func IsSubcommandAllowed(command, subcommand string) bool {
	cmdSpec, exists := AvailableCommands[command]
	if !exists {
		return false
	}

	for _, sc := range cmdSpec.AllowedSubcommands {
		if sc.Name == subcommand {
			return true
		}
	}
	return false
}

// GetAllowedCommands returns the list of all allowed commands
func GetAllowedCommands() []CommandSpec {
	commands := make([]CommandSpec, 0, len(AvailableCommands))
	for _, cmd := range AvailableCommands {
		commands = append(commands, cmd)
	}
	return commands
}

// GetBlockedFlags returns the list of all blocked flags
func GetBlockedFlags() []BlockedFlagSpec {
	return BlockedFlagsSpec
}
