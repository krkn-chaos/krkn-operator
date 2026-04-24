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

// TerminalRequest represents a request to execute a kubectl/oc command
type TerminalRequest struct {
	// ClusterID is the identifier of the target cluster
	ClusterID string `json:"cluster_id" binding:"required"`

	// UUID is the KrknTargetRequest UUID containing cluster kubeconfig
	UUID string `json:"uuid" binding:"required"`

	// Command is the full command string to execute
	// Example: "kubectl get pods -n default --output=yaml"
	Command string `json:"command" binding:"required"`
}

// TerminalResponse represents the response from command execution
type TerminalResponse struct {
	// StdoutBase64 is the command stdout output encoded in base64
	StdoutBase64 string `json:"stdout_base64"`

	// StderrBase64 is the command stderr output encoded in base64
	StderrBase64 string `json:"stderr_base64"`

	// ExitCode is the command exit code (0 = success)
	ExitCode int `json:"exit_code"`

	// Error contains the error type if execution failed
	// Possible values: "not_found", "not_permitted", "command_failed", "execution_error", "timeout"
	Error string `json:"error,omitempty"`
}

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
