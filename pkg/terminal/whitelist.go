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

// Package terminal provides command validation and whitelisting for kubectl/oc commands
package terminal

import (
	"fmt"
	"strings"
)

// AllowedSubcommands defines read-only kubectl/oc subcommands
var AllowedSubcommands = map[string]bool{
	"get":      true,
	"describe": true,
	"logs":     true,
	"top":      true,
	"explain":  true,
	"version":  true,
	"api-resources": true,
	"api-versions":  true,
	"cluster-info":  true,
}

// BlockedFlags defines flags that enable streaming or watch modes
// These are blocked in v1 because they require WebSocket/SSE support
var BlockedFlags = []string{
	"watch",
	"w",
	"follow",
	"f",
	"watch-only",
}

// ValidateCommand validates that a command is safe to execute (read-only)
// Returns an error if the command is not permitted
func ValidateCommand(cmd *ParsedCommand) error {
	// Validate command is kubectl or oc
	if cmd.Command != "kubectl" && cmd.Command != "oc" {
		return fmt.Errorf("command must be 'kubectl' or 'oc', got '%s'", cmd.Command)
	}

	// Validate subcommand is in whitelist
	if !AllowedSubcommands[cmd.Subcommand] {
		return fmt.Errorf("subcommand '%s' is not permitted (read-only commands only)", cmd.Subcommand)
	}

	// Check for blocked streaming flags
	for _, blockedFlag := range BlockedFlags {
		// Check boolean flags
		for _, flag := range cmd.BooleanFlags {
			if flag == blockedFlag {
				return fmt.Errorf("flag '--%s' is not supported (streaming commands blocked in v1)", blockedFlag)
			}
		}

		// Check valued flags (in case user passes --watch=true)
		if _, exists := cmd.Flags[blockedFlag]; exists {
			return fmt.Errorf("flag '--%s' is not supported (streaming commands blocked in v1)", blockedFlag)
		}
	}

	return nil
}

// IsStreamingFlag checks if a flag name indicates streaming behavior
func IsStreamingFlag(flagName string) bool {
	// Remove all leading dashes and convert to lowercase
	normalized := strings.ToLower(strings.TrimLeft(flagName, "-"))
	for _, blocked := range BlockedFlags {
		if normalized == blocked {
			return true
		}
	}
	return false
}
