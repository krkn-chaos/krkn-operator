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

package mcpserver

import (
	"context"
	"errors"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// withAudit wraps a tool handler with one audit log line per call. describe
// must return only identifiers (run names, job IDs, cluster names), never
// environment values or file contents.
func withAudit[In, Out any](tool string, describe func(In) []any, h mcp.ToolHandlerFor[In, Out]) mcp.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		start := time.Now()
		principal, principalErr := principalFromRequest(req)

		result, out, err := h(ctx, req, in)

		outcome := "ok"
		if err != nil {
			outcome = "error"
			var be *BackendError
			if errors.As(err, &be) && (be.Code == CodeForbidden || be.Code == CodeNotFound) {
				outcome = "denied"
			}
		}

		fields := make([]any, 0, 8)
		if principalErr == nil {
			fields = append(fields, "robot", principal.RobotName)
		}
		fields = append(fields, "tool", tool, "outcome", outcome, "durationMs", time.Since(start).Milliseconds())
		if describe != nil {
			fields = append(fields, describe(in)...)
		}
		log.Log.WithName("mcp-audit").Info("mcp tool call", fields...)
		return result, out, err
	}
}
