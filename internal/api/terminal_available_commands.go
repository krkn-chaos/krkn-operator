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
	"encoding/json"
	"net/http"

	"github.com/krkn-chaos/krkn-operator/pkg/terminal"
)

// AvailableCommandsResponse represents the response for GET /api/v1/terminal/available-commands
type AvailableCommandsResponse struct {
	Commands     []terminal.CommandSpec     `json:"commands"`
	BlockedFlags []terminal.BlockedFlagSpec `json:"blocked_flags"`
}

// GetAvailableCommands handles GET /api/v1/terminal/available-commands
// Returns the list of all permitted commands, subcommands, and blocked flags
func (h *Handler) GetAvailableCommands(w http.ResponseWriter, r *http.Request) {
	response := AvailableCommandsResponse{
		Commands:     terminal.GetAllowedCommands(),
		BlockedFlags: terminal.GetBlockedFlags(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response) // If encoding fails, client gets partial response
}
