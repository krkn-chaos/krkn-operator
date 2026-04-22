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

// Package terminal provides command parsing and validation for kubectl/oc terminal commands
package terminal

import (
	"fmt"
	"strings"
)

// ParsedCommand represents a parsed kubectl/oc command
type ParsedCommand struct {
	// Command is either "kubectl" or "oc"
	Command string

	// Subcommand is the kubectl subcommand (e.g., "get", "describe")
	Subcommand string

	// Args are positional arguments (e.g., ["pods", "nginx"])
	Args []string

	// Flags are named flags with values (e.g., {"namespace": "default", "output": "yaml"})
	Flags map[string]string

	// BooleanFlags are flags without values (e.g., ["all-namespaces", "show-labels"])
	BooleanFlags []string
}

// ParseCommand parses a command string into a structured ParsedCommand
// Example: "kubectl get pods -n default --output=yaml --all-namespaces"
func ParseCommand(cmdString string) (*ParsedCommand, error) {
	if cmdString == "" {
		return nil, fmt.Errorf("command string cannot be empty")
	}

	// Tokenize the command string (handles quoted strings)
	tokens, err := tokenize(cmdString)
	if err != nil {
		return nil, fmt.Errorf("failed to tokenize command: %w", err)
	}

	if len(tokens) == 0 {
		return nil, fmt.Errorf("no tokens found in command")
	}

	cmd := &ParsedCommand{
		Flags:        make(map[string]string),
		Args:         []string{},
		BooleanFlags: []string{},
	}

	// First token is the command (kubectl or oc)
	cmd.Command = tokens[0]

	if len(tokens) < 2 {
		return nil, fmt.Errorf("missing subcommand after %s", cmd.Command)
	}

	// Second token is the subcommand
	cmd.Subcommand = tokens[1]

	// Parse remaining tokens as args or flags
	i := 2
	for i < len(tokens) {
		token := tokens[i]

		if strings.HasPrefix(token, "--") {
			// Long flag
			flagName, flagValue, hasValue := parseLongFlag(token)

			if hasValue {
				// Flag with embedded value: --output=yaml
				cmd.Flags[flagName] = flagValue
				i++
			} else if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
				// Flag with next token as value: --output yaml
				cmd.Flags[flagName] = tokens[i+1]
				i += 2
			} else {
				// Boolean flag: --all-namespaces
				cmd.BooleanFlags = append(cmd.BooleanFlags, flagName)
				i++
			}
		} else if strings.HasPrefix(token, "-") && len(token) > 1 {
			// Short flag: -n default
			flagName := strings.TrimPrefix(token, "-")

			if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
				// Short flag with value
				cmd.Flags[flagName] = tokens[i+1]
				i += 2
			} else {
				// Boolean short flag
				cmd.BooleanFlags = append(cmd.BooleanFlags, flagName)
				i++
			}
		} else {
			// Positional argument
			cmd.Args = append(cmd.Args, token)
			i++
		}
	}

	return cmd, nil
}

// tokenize splits a command string into tokens, respecting quoted strings
// Example: `kubectl get "my pod" -n default` -> ["kubectl", "get", "my pod", "-n", "default"]
func tokenize(s string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for i, r := range s {
		switch {
		case r == '"' || r == '\'':
			if !inQuote {
				// Start of quoted string
				inQuote = true
				quoteChar = r
			} else if r == quoteChar {
				// End of quoted string
				inQuote = false
				quoteChar = 0
			} else {
				// Different quote char inside quote
				current.WriteRune(r)
			}

		case r == ' ' || r == '\t':
			if inQuote {
				// Space inside quote is literal
				current.WriteRune(r)
			} else if current.Len() > 0 {
				// End of token
				tokens = append(tokens, current.String())
				current.Reset()
			}

		default:
			current.WriteRune(r)
		}

		// Check for unclosed quote at end
		if i == len(s)-1 && inQuote {
			return nil, fmt.Errorf("unclosed quote in command")
		}
	}

	// Add final token if any
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens, nil
}

// parseLongFlag parses a long flag (--flag or --flag=value)
// Returns: flagName, flagValue, hasValue
func parseLongFlag(flag string) (string, string, bool) {
	// Remove leading --
	flag = strings.TrimPrefix(flag, "--")

	// Check for embedded value: --output=yaml
	if idx := strings.Index(flag, "="); idx != -1 {
		return flag[:idx], flag[idx+1:], true
	}

	// No embedded value: --all-namespaces
	return flag, "", false
}
