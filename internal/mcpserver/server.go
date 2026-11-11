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
	"fmt"
	"net/http"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// NewHandler builds the stateless Streamable HTTP MCP endpoint behind robot
// bearer authentication. The handler is safe to serve from any replica: no
// session state is kept in process memory.
func NewHandler(backend Backend, verify TokenVerifier, version string) http.Handler {
	srv := mcp.NewServer(&mcp.Implementation{Name: "krkn-operator", Version: version}, nil)
	registerReadTools(srv, backend)
	registerWriteTools(srv, backend)

	streamable := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, &mcp.StreamableHTTPOptions{
		Stateless:           true,
		JSONResponse:        true,
		MaxRequestBodyBytes: 1 << 20,
	})
	return mcpauth.RequireBearerToken(verifyAdapter(verify), nil)(streamable)
}

// verifyAdapter adapts a TokenVerifier to the SDK verifier contract.
//
// The SDK writes the verifier's error message into the 500 response body, so
// only ErrInvalidToken is forwarded verbatim; everything else becomes a
// generic message with the real error logged server-side.
func verifyAdapter(verify TokenVerifier) mcpauth.TokenVerifier {
	return func(ctx context.Context, token string, req *http.Request) (*mcpauth.TokenInfo, error) {
		identity, err := verify(ctx, token)
		if err != nil {
			if errors.Is(err, ErrInvalidCredentials) {
				return nil, fmt.Errorf("%w", mcpauth.ErrInvalidToken)
			}
			log.FromContext(ctx).Error(err, "MCP token verification failed")
			return nil, errors.New("authentication temporarily unavailable")
		}
		return &mcpauth.TokenInfo{
			UserID:     "robot:" + identity.Name,
			Expiration: identity.ExpiresAt,
			Extra:      map[string]any{"robot": identity.Name},
		}, nil
	}
}

// principalFromRequest extracts the robot identity the SDK stored on the
// request. Missing or malformed identity fails closed.
func principalFromRequest(req *mcp.CallToolRequest) (Principal, error) {
	if req.Extra == nil || req.Extra.TokenInfo == nil {
		return Principal{}, errors.New("unauthenticated request")
	}
	raw, ok := req.Extra.TokenInfo.Extra["robot"]
	if !ok {
		return Principal{}, errors.New("missing robot identity")
	}
	name, ok := raw.(string)
	if !ok || name == "" {
		return Principal{}, errors.New("invalid robot identity")
	}
	return Principal{RobotName: name}, nil
}

// toolError converts an error into an MCP tool error result. Non-BackendError
// errors never leak their message to the client.
func toolError(err error) *mcp.CallToolResult {
	var be *BackendError
	if errors.As(err, &be) {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("%s: %s", be.Code, be.Message)}},
		}
	}
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: "internal: internal error"}},
	}
}

// handlerToolError converts a tool error into (result, zero output, err).
//
// The error is returned alongside the client-safe IsError result so the audit
// wrapper can classify the outcome; the wrapper then strips it before the
// result reaches the SDK, preserving the IsError wire behavior.
func handlerToolError[Out any](err error) (*mcp.CallToolResult, Out, error) {
	var zero Out
	return toolError(err), zero, err
}
