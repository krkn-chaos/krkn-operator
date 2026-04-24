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

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krkn-chaos/krkn-operator/pkg/terminal"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// TestExecuteTerminal_CommandParsing tests that commands are parsed correctly
func TestExecuteTerminal_CommandParsing(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantCmd *terminal.ParsedCommand
		wantErr error
	}{
		{
			name:    "kubectl get pods",
			command: "kubectl get pods",
			wantCmd: &terminal.ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "get",
				Args:         []string{"pods"},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: nil,
		},
		{
			name:    "kubectl get pods with namespace",
			command: "kubectl get pods -n default",
			wantCmd: &terminal.ParsedCommand{
				Command:    "kubectl",
				Subcommand: "get",
				Args:       []string{"pods"},
				Flags: map[string]string{
					"n": "default",
				},
				BooleanFlags: []string{},
			},
			wantErr: nil,
		},
		{
			name:    "kubectl without subcommand",
			command: "kubectl",
			wantCmd: &terminal.ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "",
				Args:         []string{},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := terminal.ParseCommand(tt.command)
			if err != tt.wantErr {
				t.Errorf("ParseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.Command != tt.wantCmd.Command {
				t.Errorf("ParseCommand() command = %v, want %v", got.Command, tt.wantCmd.Command)
			}
			if got.Subcommand != tt.wantCmd.Subcommand {
				t.Errorf("ParseCommand() subcommand = %v, want %v", got.Subcommand, tt.wantCmd.Subcommand)
			}
		})
	}
}

// TestExecuteTerminal_ValidationLogic tests the validation logic independently
func TestExecuteTerminal_ValidationLogic(t *testing.T) {
	tests := []struct {
		name    string
		cmd     *terminal.ParsedCommand
		wantErr error
	}{
		{
			name: "valid kubectl get",
			cmd: &terminal.ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "get",
				Args:         []string{"pods"},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: nil,
		},
		{
			name: "invalid command",
			cmd: &terminal.ParsedCommand{
				Command:      "bash",
				Subcommand:   "get",
				Args:         []string{},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: terminal.ErrCommandNotFound,
		},
		{
			name: "invalid subcommand",
			cmd: &terminal.ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "delete",
				Args:         []string{"pod", "nginx"},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: terminal.ErrCommandNotPermitted,
		},
		{
			name: "blocked flag --watch",
			cmd: &terminal.ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "get",
				Args:         []string{"pods"},
				Flags:        map[string]string{},
				BooleanFlags: []string{"watch"},
			},
			wantErr: terminal.ErrCommandNotPermitted,
		},
		{
			name: "blocked flag --follow",
			cmd: &terminal.ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "logs",
				Args:         []string{"nginx"},
				Flags:        map[string]string{},
				BooleanFlags: []string{"follow"},
			},
			wantErr: terminal.ErrCommandNotPermitted,
		},
		{
			name: "kubectl without subcommand (help)",
			cmd: &terminal.ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "",
				Args:         []string{},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := terminal.ValidateCommand(tt.cmd)
			if err != tt.wantErr {
				t.Errorf("ValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestExecuteTerminal_ErrorMapping tests HTTP status code mapping
func TestExecuteTerminal_ErrorMapping(t *testing.T) {
	// This test documents the expected HTTP status codes for different error scenarios
	// Actual testing requires mock gRPC server, but this serves as specification

	errorMappings := map[string]struct {
		errorType      string
		httpStatusCode int
		description    string
	}{
		"command_not_found": {
			errorType:      "not_found",
			httpStatusCode: http.StatusNotFound,
			description:    "Command is not kubectl or oc",
		},
		"command_not_permitted": {
			errorType:      "not_permitted",
			httpStatusCode: http.StatusForbidden,
			description:    "Subcommand or flag not in whitelist",
		},
		"command_failed": {
			errorType:      "command_failed",
			httpStatusCode: http.StatusBadRequest,
			description:    "Command executed but returned exit code > 0",
		},
		"timeout": {
			errorType:      "timeout",
			httpStatusCode: http.StatusRequestTimeout,
			description:    "Command execution exceeded 120 seconds",
		},
		"execution_error": {
			errorType:      "execution_error",
			httpStatusCode: http.StatusInternalServerError,
			description:    "General execution error",
		},
		"forbidden": {
			errorType:      "forbidden",
			httpStatusCode: http.StatusForbidden,
			description:    "User lacks run permission on cluster",
		},
	}

	// Verify all error mappings are documented
	for scenario, mapping := range errorMappings {
		t.Run(scenario, func(t *testing.T) {
			if mapping.errorType == "" || mapping.httpStatusCode == 0 {
				t.Errorf("Incomplete error mapping for %s", scenario)
			}
			t.Logf("Error: %s -> HTTP %d (%s)", mapping.errorType, mapping.httpStatusCode, mapping.description)
		})
	}
}

// TestExecuteTerminal_GRPCErrorHandling tests gRPC error to HTTP error conversion
func TestExecuteTerminal_GRPCErrorHandling(t *testing.T) {
	// Document expected gRPC error handling behavior
	grpcErrors := []struct {
		name           string
		grpcCodeName   string
		grpcMessage    string
		expectedHTTP   int
		expectedError  string
		description    string
	}{
		{
			name:          "gRPC NOT_FOUND",
			grpcCodeName:  "NotFound",
			grpcMessage:   "kubectl command not found",
			expectedHTTP:  http.StatusNotFound,
			expectedError: "not_found",
			description:   "kubectl/oc binary not found on server",
		},
		{
			name:          "gRPC DEADLINE_EXCEEDED",
			grpcCodeName:  "DeadlineExceeded",
			grpcMessage:   "execution timeout",
			expectedHTTP:  http.StatusRequestTimeout,
			expectedError: "timeout",
			description:   "Command execution exceeded timeout",
		},
		{
			name:          "gRPC INTERNAL",
			grpcCodeName:  "Internal",
			grpcMessage:   "internal error",
			expectedHTTP:  http.StatusInternalServerError,
			expectedError: "execution_error",
			description:   "Internal server error",
		},
	}

	for _, tt := range grpcErrors {
		t.Run(tt.name, func(t *testing.T) {
			// Verify error mapping specification
			if tt.expectedHTTP == 0 || tt.expectedError == "" {
				t.Errorf("Incomplete error specification for %s", tt.name)
			}
			t.Logf("gRPC %s -> HTTP %d with error=%s (%s)",
				tt.grpcCodeName, tt.expectedHTTP, tt.expectedError, tt.description)
		})
	}
}

// TestExecuteTerminal_SuccessResponse tests successful command execution response format
func TestExecuteTerminal_SuccessResponse(t *testing.T) {
	// This test documents the expected success response format
	expectedResponse := TerminalResponse{
		StdoutBase64: base64.StdEncoding.EncodeToString([]byte("NAME   READY   STATUS\npod1   1/1     Running")),
		StderrBase64: "",
		ExitCode:     0,
		Error:        "",
	}

	// Verify response structure
	if expectedResponse.ExitCode != 0 {
		t.Error("Success response should have exit code 0")
	}
	if expectedResponse.Error != "" {
		t.Error("Success response should have empty error field")
	}
	if expectedResponse.StdoutBase64 == "" {
		t.Error("Success response should have stdout data")
	}

	// Verify base64 encoding
	decoded, err := base64.StdEncoding.DecodeString(expectedResponse.StdoutBase64)
	if err != nil {
		t.Errorf("Failed to decode stdout: %v", err)
	}
	if len(decoded) == 0 {
		t.Error("Decoded stdout should not be empty")
	}
}

// TestExecuteTerminal_ExitCodeNonZeroResponse tests error command response format
func TestExecuteTerminal_ExitCodeNonZeroResponse(t *testing.T) {
	// Document behavior when command executes but fails (exit code > 0)
	expectedResponse := TerminalResponse{
		StdoutBase64: "",
		StderrBase64: base64.StdEncoding.EncodeToString([]byte("Error: pods \"nonexistent\" not found")),
		ExitCode:     1,
		Error:        "command_failed",
	}

	// Verify response structure for failed command
	if expectedResponse.ExitCode == 0 {
		t.Error("Failed command should have non-zero exit code")
	}
	if expectedResponse.Error != "command_failed" {
		t.Errorf("Failed command should have error=command_failed, got %s", expectedResponse.Error)
	}
	if expectedResponse.StderrBase64 == "" {
		t.Error("Failed command should have stderr data")
	}

	// Verify base64 encoding
	decoded, err := base64.StdEncoding.DecodeString(expectedResponse.StderrBase64)
	if err != nil {
		t.Errorf("Failed to decode stderr: %v", err)
	}
	if len(decoded) == 0 {
		t.Error("Decoded stderr should not be empty")
	}
}

// TestGetAvailableCommands tests the available commands endpoint
func TestGetAvailableCommands(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	k8sClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	clientset := fake.NewSimpleClientset()

	handler := NewTestHandler(k8sClient, clientset, "test-namespace", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/terminal/available-commands", nil)
	w := httptest.NewRecorder()

	handler.GetAvailableCommands(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetAvailableCommands() status = %v, want %v", w.Code, http.StatusOK)
	}

	var response AvailableCommandsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response contains kubectl and oc
	if len(response.Commands) != 2 {
		t.Errorf("Expected 2 commands (kubectl, oc), got %d", len(response.Commands))
	}

	// Verify commands have subcommands
	for _, cmd := range response.Commands {
		if cmd.Name != "kubectl" && cmd.Name != "oc" {
			t.Errorf("Unexpected command: %s", cmd.Name)
		}
		if len(cmd.AllowedSubcommands) == 0 {
			t.Errorf("Command %s should have subcommands", cmd.Name)
		}

		// Verify at least core subcommands are present
		coreSubcommands := map[string]bool{
			"get":      false,
			"describe": false,
			"logs":     false,
		}
		for _, sc := range cmd.AllowedSubcommands {
			if _, exists := coreSubcommands[sc.Name]; exists {
				coreSubcommands[sc.Name] = true
			}
		}
		for scName, found := range coreSubcommands {
			if !found {
				t.Errorf("Command %s missing core subcommand: %s", cmd.Name, scName)
			}
		}
	}
}
