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
	"net/http"
	"runtime/debug"

	"github.com/krkn-chaos/krkn-operator/internal/mcpserver"
)

// newMCPHTTPHandler builds the MCP endpoint handler for this operator
// instance, wiring the robot token verifier to the backend.
func (h *Handler) newMCPHTTPHandler() http.Handler {
	backend := &mcpBackend{h: h}
	verify := func(ctx context.Context, token string) (mcpserver.RobotIdentity, error) {
		identity, err := h.robotTokens.Verify(ctx, token)
		if err != nil {
			if err == errRobotTokenInvalid {
				return mcpserver.RobotIdentity{}, mcpserver.ErrInvalidCredentials
			}
			return mcpserver.RobotIdentity{}, err
		}
		return mcpserver.RobotIdentity{Name: identity.Name, ExpiresAt: identity.ExpiresAt}, nil
	}
	return mcpserver.NewHandler(backend, verify, h.mcpServerVersion())
}

// mcpServerVersion reports a version string for the MCP server handshake. It
// prefers the module version stamped into the build (set for tagged builds);
// local `go run`/`go build` falls back to "dev".
func (h *Handler) mcpServerVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}

// mcpPrincipal wraps a robot name in an mcpserver Principal.
func mcpPrincipal(robotName string) mcpserver.Principal {
	return mcpserver.Principal{RobotName: robotName}
}
