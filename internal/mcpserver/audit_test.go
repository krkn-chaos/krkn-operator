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
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// auditSink captures log lines emitted while the global logger is swapped.
// controller-runtime only honors the first log.SetLogger call, so the sink is
// installed once per test process, not restored between subtests.
type auditSink struct {
	mu    sync.Mutex
	lines []string
}

func (s *auditSink) logger() logr.Logger {
	return funcr.New(func(prefix, args string) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.lines = append(s.lines, args)
	}, funcr.Options{})
}

func (s *auditSink) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lines = nil
}

func (s *auditSink) snapshot() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.lines...)
}

var auditTestSink = &auditSink{}

// TestMain installs the capturing sink as the package's global logger.
// controller-runtime's log.Log only honors the first SetLogger call, so this
// happens once for the whole test binary instead of per test.
func TestMain(m *testing.M) {
	log.SetLogger(auditTestSink.logger())
	os.Exit(m.Run())
}

// TestWithAudit_Outcome drives krkn_whoami through a real client session with
// a stub backend returning different error kinds, and asserts the audit line
// records the correct outcome while the client still receives an IsError
// result (not a transport error) for failures.
func TestWithAudit_Outcome(t *testing.T) {
	tests := []struct {
		name        string
		backend     *stubBackend
		wantOutcome string
		wantIsError bool
	}{
		{
			name:        "ok",
			backend:     &stubBackend{whoami: WhoAmIOutput{RobotName: "ci-bot"}},
			wantOutcome: "ok",
		},
		{
			name:        "forbidden",
			backend:     &stubBackend{err: &BackendError{Code: CodeForbidden, Message: "denied"}},
			wantOutcome: "denied",
			wantIsError: true,
		},
		{
			name:        "not_found",
			backend:     &stubBackend{err: &BackendError{Code: CodeNotFound, Message: "no such run"}},
			wantOutcome: "denied",
			wantIsError: true,
		},
		{
			name:        "internal error",
			backend:     &stubBackend{err: errors.New("etcd timeout")},
			wantOutcome: "error",
			wantIsError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auditTestSink.reset()

			session := newTestSession(t, tt.backend, staticVerifier{identity: validIdentity()}.verify)

			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      "krkn_whoami",
				Arguments: map[string]any{},
			})
			if err != nil {
				t.Fatalf("CallTool returned transport error %v, want IsError result", err)
			}
			if result.IsError != tt.wantIsError {
				t.Errorf("result.IsError = %v, want %v", result.IsError, tt.wantIsError)
			}

			found := false
			for _, line := range auditTestSink.snapshot() {
				if !strings.Contains(line, `"tool"="krkn_whoami"`) {
					continue
				}
				found = true
				if !strings.Contains(line, `"outcome"="`+tt.wantOutcome+`"`) {
					t.Errorf("audit outcome = %q, want %q (line: %s)", line, tt.wantOutcome, line)
				}
			}
			if !found {
				lines := auditTestSink.snapshot()
				t.Errorf("no audit line for krkn_whoami found in %d captured lines: %v", len(lines), lines)
			}
		})
	}
}

// TestAuditOutcome_Classification unit-tests the outcome classifier directly,
// including an IsError result returned without a Go error.
func TestAuditOutcome_Classification(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		result *mcp.CallToolResult
		want   string
	}{
		{
			name: "success",
			want: "ok",
		},
		{
			name:   "iserror result without go error",
			result: &mcp.CallToolResult{IsError: true},
			want:   "error",
		},
		{
			name: "forbidden",
			err:  &BackendError{Code: CodeForbidden, Message: "denied"},
			want: "denied",
		},
		{
			name: "not_found",
			err:  &BackendError{Code: CodeNotFound, Message: "missing"},
			want: "denied",
		},
		{
			name: "internal backend error",
			err:  &BackendError{Code: CodeInternal, Message: "boom"},
			want: "error",
		},
		{
			name: "wrapped backend error",
			err:  wrapErr(&BackendError{Code: CodeForbidden, Message: "denied"}),
			want: "denied",
		},
		{
			name: "plain error",
			err:  errors.New("boom"),
			want: "error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := auditOutcome(tt.err, tt.result); got != tt.want {
				t.Errorf("auditOutcome() = %q, want %q", got, tt.want)
			}
		})
	}
}

// wrapErr adds a layer of wrapping around an error to exercise errors.As.
func wrapErr(err error) error {
	return &wrappedError{err: err}
}

type wrappedError struct {
	err error
}

func (w *wrappedError) Error() string { return "wrapped: " + w.err.Error() }
func (w *wrappedError) Unwrap() error { return w.err }
