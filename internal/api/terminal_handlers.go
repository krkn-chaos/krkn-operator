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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/krkn-chaos/krkn-operator/pkg/terminal"
	pb "github.com/krkn-chaos/krkn-operator/proto/dataprovider"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ExecuteTerminal handles POST /api/v1/terminal
// Executes kubectl/oc commands with read-only validation
func (h *Handler) ExecuteTerminal(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req TerminalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid JSON request body",
		})
		return
	}

	// Validate required fields
	if req.ClusterID == "" || req.UUID == "" || req.Command == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "cluster_id, uuid, and command are required",
		})
		return
	}

	ctx := r.Context()

	// Quick check: command must start with "kubectl" or "oc" before parsing
	// This ensures we return 404 for invalid commands like "ls", "bash", etc.
	cmdLower := strings.ToLower(strings.TrimSpace(req.Command))
	if !strings.HasPrefix(cmdLower, "kubectl ") &&
	   !strings.HasPrefix(cmdLower, "oc ") &&
	   cmdLower != "kubectl" &&
	   cmdLower != "oc" {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Command must be kubectl or oc",
		})
		return
	}

	// Get kubeconfig using legacy helper (UUID maps to target ID, ClusterID maps to cluster name)
	kubeconfigBase64, err := h.getKubeconfig(ctx, "", req.UUID, req.ClusterID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: fmt.Sprintf("Failed to get kubeconfig: %v", err),
		})
		return
	}

	// Special case: "kubectl" or "oc" without subcommand should show help
	// Skip parsing and validation, send directly to gRPC
	if cmdLower == "kubectl" || cmdLower == "oc" {
		parsedCmd := &terminal.ParsedCommand{
			Command:      req.Command,
			Subcommand:   "",
			Args:         []string{},
			Flags:        map[string]string{},
			BooleanFlags: []string{},
		}
		response, err := h.executeKubectlViaGRPC(ctx, kubeconfigBase64, parsedCmd)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "execution_error",
				Message: "Command execution failed",
			})
			return
		}
		// Return response (will contain help output)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Parse command
	parsedCmd, err := terminal.ParseCommand(req.Command)
	if err != nil {
		// If parsing failed and it's not kubectl/oc, return 404
		// (though this should have been caught above)
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_command",
			Message: fmt.Sprintf("Failed to parse command: %v", err),
		})
		return
	}

	// Validate command (read-only, no streaming)
	if err := terminal.ValidateCommand(parsedCmd); err != nil {
		// Check if command not found (not kubectl/oc) → 404
		if errors.Is(err, terminal.ErrCommandNotFound) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Command must be kubectl or oc",
			})
			return
		}
		// Command not permitted (subcommand/flag blocked) → 403
		if errors.Is(err, terminal.ErrCommandNotPermitted) {
			writeJSONError(w, http.StatusForbidden, ErrorResponse{
				Error:   "not_permitted",
				Message: "Command not permitted",
			})
			return
		}
		// Fallback for unknown validation errors → 400
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_command",
			Message: "Invalid command",
		})
		return
	}

	// Execute via gRPC data provider
	response, err := h.executeKubectlViaGRPC(ctx, kubeconfigBase64, parsedCmd)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "execution_error",
			Message: "Command execution failed",
		})
		return
	}

	// Check if gRPC returned an error
	if response.Error != "" {
		switch response.Error {
		case "not_found":
			// kubectl/oc command not found on server → 404
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Command not found on server",
			})
		case "timeout":
			// Command execution timed out → 408
			writeJSONError(w, http.StatusRequestTimeout, ErrorResponse{
				Error:   "timeout",
				Message: "Command execution timed out",
			})
		default:
			// Other execution errors → 500
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "execution_error",
				Message: "Command execution failed",
			})
		}
		return
	}

	// Check exit code - if > 0, command failed → 400 BUT still return stdout/stderr
	if response.ExitCode > 0 {
		response.Error = "command_failed"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Success - return response with stdout/stderr
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// executeKubectlViaGRPC executes a kubectl command via the gRPC data provider
func (h *Handler) executeKubectlViaGRPC(ctx context.Context, kubeconfigBase64 string, cmd *terminal.ParsedCommand) (*TerminalResponse, error) {
	// Connect to gRPC server
	conn, err := grpc.Dial(h.grpcServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}
	defer conn.Close()

	client := pb.NewDataProviderServiceClient(conn)

	// Build gRPC request
	grpcReq := &pb.ExecuteKubectlRequest{
		KubeconfigBase64: kubeconfigBase64,
		Command:          cmd.Command,
		Subcommand:       cmd.Subcommand,
		Args:             cmd.Args,
		Flags:            cmd.Flags,
		BooleanFlags:     cmd.BooleanFlags,
		TimeoutSeconds:   120, // Fixed timeout as per requirements
	}

	// Execute with context timeout
	ctx, cancel := context.WithTimeout(ctx, 130*time.Second) // 130s to allow gRPC 120s timeout
	defer cancel()

	grpcResp, err := client.ExecuteKubectl(ctx, grpcReq)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}

	// Map gRPC response to API response
	response := &TerminalResponse{
		StdoutBase64: grpcResp.StdoutBase64,
		StderrBase64: grpcResp.StderrBase64,
		ExitCode:     int(grpcResp.ExitCode),
		Error:        grpcResp.Error,
	}

	return response, nil
}
