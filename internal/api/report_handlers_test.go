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
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
