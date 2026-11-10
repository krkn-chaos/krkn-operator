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
	"net/url"
	"testing"

	"github.com/krkn-chaos/krkn-operator/internal/mcpserver"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

func TestDispatchAsRobot_SetsRobotClaims(t *testing.T) {
	h, _ := newRobotsTestHandler(t)

	var gotUserID, gotRole string
	var gotAdmin bool
	fn := func(w http.ResponseWriter, r *http.Request) {
		claims := auth.GetClaimsFromContext(r.Context())
		if claims != nil {
			gotUserID = claims.UserID
			gotRole = claims.Role
			gotAdmin = auth.IsAdmin(r.Context())
		}
		w.WriteHeader(http.StatusOK)
	}

	res, err := h.dispatchAsRobot(t.Context(), "ci-bot", http.MethodGet, "/api/v1/x", nil, nil, nil, fn)
	if err != nil {
		t.Fatalf("dispatchAsRobot failed: %v", err)
	}
	if res.Status != http.StatusOK {
		t.Errorf("status = %d, want 200", res.Status)
	}
	if gotUserID != "robot:ci-bot" {
		t.Errorf("claims UserID = %q, want robot:ci-bot", gotUserID)
	}
	if gotRole != string(auth.RoleRobot) {
		t.Errorf("claims Role = %q, want robot", gotRole)
	}
	if gotAdmin {
		t.Error("robot claims must not be admin")
	}
}

func TestDispatchAsRobot_PathAndBody(t *testing.T) {
	h, _ := newRobotsTestHandler(t)

	var gotPath string
	var gotQuery url.Values
	var gotBody map[string]string
	fn := func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}

	query := url.Values{"a": []string{"b"}}
	res, err := h.dispatchAsRobot(t.Context(), "ci-bot", http.MethodPost, ScenariosRunPath, query, map[string]string{"k": "v"}, map[string]string{"X-Test": "1"}, fn)
	if err != nil {
		t.Fatalf("dispatchAsRobot failed: %v", err)
	}
	if res.Status != http.StatusCreated {
		t.Errorf("status = %d, want 201", res.Status)
	}
	if gotPath != ScenariosRunPath {
		t.Errorf("path = %q, want %q", gotPath, ScenariosRunPath)
	}
	if gotQuery.Get("a") != "b" {
		t.Errorf("query = %v, want a=b", gotQuery)
	}
	if gotBody["k"] != "v" {
		t.Errorf("body = %v, want k=v", gotBody)
	}
	var parsed map[string]any
	if err := json.Unmarshal(res.Body, &parsed); err != nil {
		t.Fatalf("decode result body: %v", err)
	}
	if parsed["ok"] != true {
		t.Errorf("result body = %s, want ok:true", res.Body)
	}
}

func TestBackendErrorFromDispatch(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantCode mcpserver.ErrorCode
		wantNil  bool
		wantMsg  string
	}{
		{"200", 200, `{}`, "", true, ""},
		{"400", 400, `{"error":"bad_request","message":"bad input"}`, mcpserver.CodeInvalidArg, false, "bad input"},
		{"403", 403, `{"error":"forbidden","message":"no way"}`, mcpserver.CodeForbidden, false, "no way"},
		{"404", 404, `{"error":"not_found","message":"missing"}`, mcpserver.CodeNotFound, false, "missing"},
		{"409", 409, `{"error":"conflict","message":"exists"}`, mcpserver.CodeConflict, false, "exists"},
		{"503", 503, `{"error":"unavailable","message":"down"}`, mcpserver.CodeUnavailable, false, "down"},
		{"500 secret", 500, `{"error":"internal_error","message":"secret detail"}`, mcpserver.CodeInternal, false, "internal error"},
		{"garbage body", 400, `not json`, mcpserver.CodeInvalidArg, false, "Bad Request"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := backendErrorFromDispatch(dispatchResult{Status: tt.status, Body: []byte(tt.body)})
			if tt.wantNil {
				if err != nil {
					t.Errorf("err = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("err = nil, want error")
			}
			var be *mcpserver.BackendError
			if !asBackendErr(err, &be) {
				t.Fatalf("err %v is not BackendError", err)
			}
			if be.Code != tt.wantCode {
				t.Errorf("code = %s, want %s", be.Code, tt.wantCode)
			}
			if be.Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", be.Message, tt.wantMsg)
			}
		})
	}
}

// asBackendErr is a test-local errors.As.
func asBackendErr(err error, target **mcpserver.BackendError) bool {
	for err != nil {
		if be, ok := err.(*mcpserver.BackendError); ok {
			*target = be
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
