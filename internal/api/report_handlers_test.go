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
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

const reportTestNamespace = "krkn-operator-system"

func newReportTestHandler(t *testing.T, reportStatus *krknv1alpha1.ReportStatus, configMap *corev1.ConfigMap) *Handler {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	run := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "report-test-run",
			Namespace: reportTestNamespace,
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			ReportStatus: reportStatus,
		},
	}

	objects := []interface{}{run}
	if configMap != nil {
		objects = append(objects, configMap)
	}
	clientBuilder := fakeclient.NewClientBuilder().WithScheme(scheme)
	for _, object := range objects {
		switch typed := object.(type) {
		case *krknv1alpha1.KrknScenarioRun:
			clientBuilder = clientBuilder.WithObjects(typed)
		case *corev1.ConfigMap:
			clientBuilder = clientBuilder.WithObjects(typed)
		}
	}

	return &Handler{
		client:    clientBuilder.Build(),
		namespace: reportTestNamespace,
	}
}

func reportRequest(path string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	claims := &auth.Claims{UserID: "admin@test.com", Role: "admin"}
	return req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, claims))
}

func TestDownloadReportPDF(t *testing.T) {
	pdf := []byte("%PDF-test")
	handler := newReportTestHandler(t, &krknv1alpha1.ReportStatus{
		Generated:    true,
		PDFAvailable: true,
		Location:     "krkn-report-report-test-run",
		FileSize:     int64(len(pdf)),
	}, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "krkn-report-report-test-run", Namespace: reportTestNamespace},
		BinaryData: map[string][]byte{"summary.pdf": pdf},
	})

	w := httptest.NewRecorder()
	handler.DownloadReport(w, reportRequest("/api/v1/scenarios/run/report-test-run/reports/summary.pdf"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("expected PDF content type, got %q", got)
	}
	if got := w.Header().Get("Content-Disposition"); got != "attachment; filename=report-test-run-summary.pdf" {
		t.Fatalf("unexpected content disposition: %q", got)
	}
	if got := w.Body.Bytes(); string(got) != string(pdf) {
		t.Fatalf("unexpected PDF body: %q", got)
	}
}

func TestDownloadReportHTMLThroughV2Router(t *testing.T) {
	handler := newReportTestHandler(t, &krknv1alpha1.ReportStatus{
		Generated:     true,
		HTMLAvailable: true,
		Location:      "krkn-report-report-test-run",
	}, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "krkn-report-report-test-run", Namespace: reportTestNamespace},
		Data:       map[string]string{"summary.html": "<html>report</html>"},
	})

	w := httptest.NewRecorder()
	handler.ScenariosRunRouter(w, reportRequest("/api/v2/scenarios/run/report-test-run/reports/summary.html"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("expected HTML content type, got %q", got)
	}
	if got := w.Body.String(); got != "<html>report</html>" {
		t.Fatalf("unexpected HTML body: %q", got)
	}
}

func TestDownloadReportDeniesUserWithoutReportClusterAccess(t *testing.T) {
	const (
		userID       = "user@example.com"
		allowedURL   = "https://allowed.example.com"
		reportURL    = "https://restricted.example.com"
		reportJobID  = "report-job"
		allowedGroup = "allowed-clusters"
	)

	pdf := []byte("%PDF-restricted")
	handler := newReportTestHandler(t, &krknv1alpha1.ReportStatus{
		Generated:    true,
		PDFAvailable: true,
		Location:     "krkn-report-report-test-run",
	}, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krkn-report-report-test-run",
			Namespace: reportTestNamespace,
			Annotations: map[string]string{
				"krkn.krkn-chaos.dev/report-job-id": reportJobID,
			},
		},
		BinaryData: map[string][]byte{"summary.pdf": pdf},
	})

	var run krknv1alpha1.KrknScenarioRun
	if err := handler.client.Get(context.Background(), client.ObjectKey{Name: "report-test-run", Namespace: reportTestNamespace}, &run); err != nil {
		t.Fatal(err)
	}
	run.Status.ClusterJobs = []krknv1alpha1.ClusterJobStatus{
		{JobID: "allowed-job", ClusterAPIURL: allowedURL},
		{JobID: reportJobID, ClusterAPIURL: reportURL},
	}
	if err := handler.client.Update(context.Background(), &run); err != nil {
		t.Fatal(err)
	}

	user := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-user-example-com",
			Namespace: reportTestNamespace,
			Labels:    map[string]string{groupauth.GroupLabelKey(allowedGroup): "true"},
		},
		Spec: krknv1alpha1.KrknUserSpec{UserID: userID},
	}
	group := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{Name: allowedGroup, Namespace: reportTestNamespace},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: allowedGroup,
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				allowedURL: {Actions: []string{"view"}},
			},
		},
	}
	if err := handler.client.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	if err := handler.client.Create(context.Background(), group); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/run/report-test-run/reports/summary.pdf", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{UserID: userID, Role: "user"}))
	w := httptest.NewRecorder()
	handler.DownloadReport(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDownloadReportUnavailable(t *testing.T) {
	handler := newReportTestHandler(t, &krknv1alpha1.ReportStatus{
		Generated:     true,
		HTMLAvailable: true,
		Location:      "krkn-report-report-test-run",
	}, nil)

	w := httptest.NewRecorder()
	handler.DownloadReport(w, reportRequest("/api/v1/scenarios/run/report-test-run/reports/summary.pdf"))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestDownloadReportRejectsMalformedPath(t *testing.T) {
	handler := newReportTestHandler(t, nil, nil)

	w := httptest.NewRecorder()
	handler.DownloadReport(w, reportRequest("/api/v1/scenarios/run/report-test-run/extra/reports/summary.pdf"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestGetReportStatusPending(t *testing.T) {
	handler := newReportTestHandler(t, nil, nil)

	w := httptest.NewRecorder()
	handler.GetReportStatus(w, reportRequest("/api/v1/scenarios/run/report-test-run/reports/status"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetReportStatusReportsFailure(t *testing.T) {
	handler := newReportTestHandler(t, &krknv1alpha1.ReportStatus{
		Message: "Report generation failed",
	}, nil)

	w := httptest.NewRecorder()
	handler.GetReportStatus(w, reportRequest("/api/v1/scenarios/run/report-test-run/reports/status"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetReportStatusRejectsExtraPathSegments(t *testing.T) {
	handler := newReportTestHandler(t, nil, nil)

	w := httptest.NewRecorder()
	handler.GetReportStatus(w, reportRequest("/api/v1/scenarios/run/report-test-run/extra/reports/status"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestIsReportMarkerStart(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"===KRKN_REPORT_HTML_START===", true},
		{"===KRKN_REPORT_PDF_START===", true},
		{"2026-10-01T16:44:58Z ===KRKN_REPORT_HTML_START===", true},
		{"2026-10-01T16:44:58Z ===KRKN_REPORT_PDF_START===", true},
		{"===KRKN_REPORT_HTML_END===", false},
		{"===KRKN_REPORT_PDF_END===", false},
		{"2025-01-01 scenario started", false},
		{"", false},
		{"===KRKN_REPORT_", false},
		{"SGVsbG8gV29ybGQ=", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := isReportMarkerStart(tt.line); got != tt.want {
				t.Errorf("isReportMarkerStart(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestIsReportMarkerEnd(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"===KRKN_REPORT_HTML_END===", true},
		{"===KRKN_REPORT_PDF_END===", true},
		{"2026-10-01T16:44:58Z ===KRKN_REPORT_HTML_END===", true},
		{"2026-10-01T16:44:58Z ===KRKN_REPORT_PDF_END===", true},
		{"===KRKN_REPORT_HTML_START===", false},
		{"scenario completed", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := isReportMarkerEnd(tt.line); got != tt.want {
				t.Errorf("isReportMarkerEnd(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestFilterReportLines(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name: "filters HTML and PDF blocks",
			input: []string{
				"scenario starting",
				"running chaos",
				"scenario completed",
				"===KRKN_REPORT_HTML_START===",
				"PCFET0NUWVBFIGh0bWw+",
				"PGh0bWw+PGJvZHk+",
				"===KRKN_REPORT_HTML_END===",
				"===KRKN_REPORT_PDF_START===",
				"JVBERi0xLjQK",
				"===KRKN_REPORT_PDF_END===",
				"exit code: 0",
			},
			want: []string{"scenario starting", "running chaos", "scenario completed", "exit code: 0"},
		},
		{
			name: "filters timestamped markers",
			input: []string{
				"2026-10-01T16:44:58Z scenario starting",
				"2026-10-01T16:44:59Z ===KRKN_REPORT_HTML_START===",
				"2026-10-01T16:44:59Z PCFET0NUWVBFIGh0bWw+",
				"2026-10-01T16:45:00Z ===KRKN_REPORT_HTML_END===",
				"2026-10-01T16:45:01Z exit code: 0",
			},
			want: []string{
				"2026-10-01T16:44:58Z scenario starting",
				"2026-10-01T16:45:01Z exit code: 0",
			},
		},
		{
			name:  "no markers is a no-op",
			input: []string{"scenario starting", "running chaos", "scenario completed"},
			want:  []string{"scenario starting", "running chaos", "scenario completed"},
		},
		{
			name:  "nil input returns nil",
			input: nil,
			want:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterReportLines(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d lines, want %d\ngot:  %v\nwant: %v", len(got), len(tt.want), got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("line %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGetScenarioRunLogsFiltersReportBlocks(t *testing.T) {
	const (
		namespace = reportTestNamespace
		runName   = "report-log-test-run"
		jobID     = "report-log-test-job"
		podName   = "report-log-test-pod"
	)

	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	run := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: runName, Namespace: namespace},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{{
				JobID:   jobID,
				PodName: podName,
				Phase:   "Succeeded",
			}},
		},
	}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: namespace}}
	controllerClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(run, pod).
		Build()

	logContent := strings.Join([]string{
		"2026-10-01T16:44:58Z scenario starting",
		"2026-10-01T16:44:59Z ===KRKN_REPORT_HTML_START===",
		"2026-10-01T16:44:59Z PCFET0NUWVBFIGh0bWw+",
		"2026-10-01T16:45:00Z ===KRKN_REPORT_HTML_END===",
		"2026-10-01T16:45:01Z scenario completed",
		"2026-10-01T16:45:02Z ===KRKN_REPORT_PDF_START===",
		"2026-10-01T16:45:02Z JVBERi0xLjQK",
		"2026-10-01T16:45:03Z ===KRKN_REPORT_PDF_END===",
		"2026-10-01T16:45:04Z exit code: 0",
	}, "\n") + "\n"

	var kubernetesAppliedTail bool
	clientset := k8sfake.NewSimpleClientset()
	clientset.PrependReactor("get", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		response := logContent
		if genericAction, ok := action.(k8stesting.GenericAction); ok {
			if options, ok := genericAction.GetValue().(*corev1.PodLogOptions); ok && options.TailLines != nil {
				kubernetesAppliedTail = true
				lines := strings.Split(strings.TrimSuffix(logContent, "\n"), "\n")
				tail := int(*options.TailLines)
				if tail < len(lines) {
					lines = lines[len(lines)-tail:]
				}
				response = strings.Join(lines, "\n") + "\n"
			}
		}
		return true, &runtime.Unknown{Raw: []byte(response)}, nil
	})
	handler := NewTestHandler(controllerClient, clientset, namespace, "localhost:50051")

	tokenGenerator, err := handler.getTokenGenerator()
	if err != nil {
		t.Fatalf("getTokenGenerator() error = %v", err)
	}
	token, err := tokenGenerator.GenerateToken("admin@test.com", "admin", "Admin", "Test", "Org")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(handler.GetScenarioRunLogs))
	defer server.Close()
	defer handler.Shutdown()

	websocketURL := "ws" + strings.TrimPrefix(server.URL, "http") +
		"/api/v1/scenarios/run/" + runName + "/jobs/" + jobID + "/logs?follow=true&tailLines=3&timestamps=true"
	header := http.Header{}
	header.Set("Sec-WebSocket-Protocol", "access_token."+token)
	connection, response, err := websocket.DefaultDialer.Dial(websocketURL, header)
	if err != nil {
		if response == nil {
			t.Fatalf("Dial() error = %v", err)
		}
		t.Fatalf("Dial() error = %v (status %s)", err, response.Status)
	}
	defer connection.Close()
	if err := connection.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v", err)
	}

	var messages []string
	for {
		_, message, err := connection.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				break
			}
			t.Fatalf("ReadMessage() error = %v", err)
		}
		messages = append(messages, string(message))
	}
	if kubernetesAppliedTail {
		t.Fatal("handler passed tailLines to Kubernetes for a followed log stream")
	}

	want := []string{
		"2026-10-01T16:44:58Z scenario starting",
		"2026-10-01T16:45:01Z scenario completed",
		"2026-10-01T16:45:04Z exit code: 0",
	}
	if len(messages) != len(want) {
		t.Fatalf("received %d messages, want %d: %v", len(messages), len(want), messages)
	}
	for i := range want {
		if messages[i] != want[i] {
			t.Errorf("message %d = %q, want %q", i, messages[i], want[i])
		}
	}
}
