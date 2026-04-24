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
	"errors"
	"strings"
)

// Custom error types for better error handling
var (
	// ErrCommandNotFound is returned when command is not kubectl or oc
	ErrCommandNotFound = errors.New("command_not_found")

	// ErrCommandNotPermitted is returned when subcommand or flag is not allowed
	ErrCommandNotPermitted = errors.New("command_not_permitted")
)

// BlockedFlags defines flags that enable streaming or watch modes
// These are blocked in v1 because they require WebSocket/SSE support
// NOTE: This is kept for backward compatibility, use BlockedFlagsSpec for the canonical list
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
	// Validate command is in allowed list (kubectl or oc)
	if !IsCommandAllowed(cmd.Command) {
		return ErrCommandNotFound
	}

	// Empty subcommand is allowed (shows help output)
	if cmd.Subcommand != "" {
		// Validate subcommand is in whitelist for this command
		if !IsSubcommandAllowed(cmd.Command, cmd.Subcommand) {
			return ErrCommandNotPermitted
		}
	}

	// Check for blocked streaming flags
	for _, blockedFlag := range BlockedFlags {
		// Check boolean flags
		for _, flag := range cmd.BooleanFlags {
			if flag == blockedFlag {
				return ErrCommandNotPermitted
			}
		}

		// Check valued flags (in case user passes --watch=true)
		if _, exists := cmd.Flags[blockedFlag]; exists {
			return ErrCommandNotPermitted
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
