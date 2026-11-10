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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/mcpserver"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

// maxDispatchBody caps the buffered dispatch response at 8 MiB.
const maxDispatchBody = 8 << 20

// withRobotClaims returns a context carrying robot claims, so dispatched
// requests pass through the existing authorization middleware unchanged.
func withRobotClaims(ctx context.Context, robotName string) context.Context {
	return context.WithValue(ctx, auth.UserClaimsKey, &auth.Claims{
		UserID: "robot:" + robotName,
		Role:   string(auth.RoleRobot),
		Name:   robotName,
	})
}

// dispatchResult captures the outcome of an in-process handler dispatch.
type dispatchResult struct {
	Status int
	Body   []byte
}

// bufferedResponseWriter captures a handler's response in memory. It is
// intentionally in-package: net/http/httptest must not be imported by
// production code.
type bufferedResponseWriter struct {
	header http.Header
	status int
	buf    bytes.Buffer
}

func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{header: make(http.Header)}
}

func (w *bufferedResponseWriter) Header() http.Header { return w.header }

func (w *bufferedResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *bufferedResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.buf.Len()+len(p) > maxDispatchBody {
		return 0, fmt.Errorf("dispatched response exceeds %d bytes", maxDispatchBody)
	}
	return w.buf.Write(p)
}

// dispatchAsRobot invokes an existing REST handler in-process with robot
// claims in the context, so every existing validation runs unchanged. path
// must be the real REST path because handlers parse r.URL.Path.
func (h *Handler) dispatchAsRobot(ctx context.Context, robotName, method, path string, query url.Values, body any, headers map[string]string, fn http.HandlerFunc) (dispatchResult, error) {
	var jsonBody []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return dispatchResult{}, fmt.Errorf("failed to encode dispatch body: %w", err)
		}
		jsonBody = encoded
	}

	target := path
	if len(query) > 0 {
		target = path + "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(withRobotClaims(ctx, robotName), method, target, bytes.NewReader(jsonBody))
	if err != nil {
		return dispatchResult{}, fmt.Errorf("failed to build dispatch request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	w := newBufferedResponseWriter()
	fn(w, req)

	if w.status == 0 {
		w.status = http.StatusOK
	}
	return dispatchResult{Status: w.status, Body: w.buf.Bytes()}, nil
}

// backendErrorFromWriteDispatch maps a write dispatch result, replacing any
// 403 body with a generic message: run-creation denials name target clusters
// and their API URLs, which must not leak to robots that cannot view them.
func backendErrorFromWriteDispatch(res dispatchResult) error {
	err := backendErrorFromDispatch(res)
	var be *mcpserver.BackendError
	if errors.As(err, &be) && be.Code == mcpserver.CodeForbidden {
		return &mcpserver.BackendError{Code: mcpserver.CodeForbidden, Message: "permission denied"}
	}
	return err
}

// backendErrorFromDispatch maps a dispatch HTTP status to a BackendError.
// Handler messages in 4xx bodies are already client-safe; 5xx never leaks.
func backendErrorFromDispatch(res dispatchResult) error {
	if res.Status >= 200 && res.Status < 300 {
		return nil
	}

	var code mcpserver.ErrorCode
	switch res.Status {
	case http.StatusBadRequest:
		code = mcpserver.CodeInvalidArg
	case http.StatusForbidden:
		code = mcpserver.CodeForbidden
	case http.StatusNotFound:
		code = mcpserver.CodeNotFound
	case http.StatusConflict:
		code = mcpserver.CodeConflict
	case http.StatusServiceUnavailable:
		code = mcpserver.CodeUnavailable
	default:
		return &mcpserver.BackendError{Code: mcpserver.CodeInternal, Message: "internal error"}
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(res.Body, &errResp); err == nil && errResp.Message != "" {
		return &mcpserver.BackendError{Code: code, Message: errResp.Message}
	}
	return &mcpserver.BackendError{Code: code, Message: http.StatusText(res.Status)}
}

// clusterPerms maps a cluster API URL to the robot's allowed actions.
type clusterPerms map[string]map[groupauth.Action]bool

// robotClusterPerms resolves the robot's per-cluster permissions.
func (h *Handler) robotClusterPerms(ctx context.Context, robotName string) (clusterPerms, error) {
	groups, err := groupauth.GetUserGroups(ctx, h.client, "robot:"+robotName, h.namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve robot groups: %w", err)
	}
	perms := make(clusterPerms)
	for _, group := range groups {
		for clusterURL, permSet := range group.Spec.ClusterPermissions {
			if perms[clusterURL] == nil {
				perms[clusterURL] = make(map[groupauth.Action]bool)
			}
			for _, action := range permSet.Actions {
				perms[clusterURL][groupauth.Action(action)] = true
			}
		}
	}
	return perms, nil
}

// can reports whether the robot may perform action a on the cluster at
// clusterURL.
func (p clusterPerms) can(clusterURL string, a groupauth.Action) bool {
	actions, ok := p[clusterURL]
	return ok && actions[a]
}

// graphTargetURLs resolves the target clusters of a graph run to their API
// URLs via the run's KrknTargetRequest. A missing request is an error so
// callers can fail closed.
func (h *Handler) graphTargetURLs(ctx context.Context, gr *krknv1alpha1.KrknGraphRun) (map[string]string, error) {
	logger := log.FromContext(ctx)

	tr := &krknv1alpha1.KrknTargetRequest{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: gr.Spec.TargetRequestID, Namespace: h.namespace}, tr); err != nil {
		logger.V(1).Info("Failed to get target request for graph run", "graphRun", gr.Name, "error", err.Error())
		return nil, fmt.Errorf("target request %q not found", gr.Spec.TargetRequestID)
	}

	urls := make(map[string]string)
	for _, clusters := range tr.Status.TargetData {
		for _, cluster := range clusters {
			urls[cluster.ClusterName] = cluster.ClusterAPIURL
		}
	}
	return urls, nil
}
