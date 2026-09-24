package controller

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func TestExtractDelimitedContent(t *testing.T) {
	tests := []struct {
		name  string
		logs  string
		found bool
		want  string
	}{
		{name: "extracts content", logs: "before STARTpayloadEND after", found: true, want: "payload"},
		{name: "uses the last marker pair", logs: "STARToldEND STARTnewEND", found: true, want: "new"},
		{name: "missing start marker", logs: "payloadEND", found: false},
		{name: "missing end marker", logs: "STARTpayload", found: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := extractDelimitedContent(tt.logs, "START", "END")
			if found != tt.found || got != tt.want {
				t.Fatalf("extractDelimitedContent() = (%q, %t), want (%q, %t)", got, found, tt.want, tt.found)
			}
		})
	}
}

func TestDecodeReportPayloadsPreservesValidReport(t *testing.T) {
	html := base64.StdEncoding.EncodeToString([]byte("<html>report</html>"))
	data, binaryData, messages := decodeReportPayloads(html, "not-base64")

	if got := data["summary.html"]; got != "<html>report</html>" {
		t.Fatalf("unexpected decoded HTML: %q", got)
	}
	if len(binaryData) != 0 {
		t.Fatalf("expected no decoded PDF, got %d entries", len(binaryData))
	}
	if len(messages) != 1 || !strings.Contains(messages[0], "failed to decode PDF report") {
		t.Fatalf("unexpected decode messages: %v", messages)
	}
}

func TestDecodeReportPayloadsReportsInvalidHTML(t *testing.T) {
	data, binaryData, messages := decodeReportPayloads("not-base64", "")

	if len(data) != 0 || len(binaryData) != 0 {
		t.Fatalf("expected no decoded reports, got data=%v binaryData=%v", data, binaryData)
	}
	if len(messages) != 1 || !strings.Contains(messages[0], "failed to decode HTML report") {
		t.Fatalf("unexpected decode messages: %v", messages)
	}
}

func TestTargetClusterCount(t *testing.T) {
	tests := []struct {
		name           string
		targetClusters map[string][]string
		want           int
	}{
		{name: "single cluster", targetClusters: map[string][]string{"provider": {"cluster-a"}}, want: 1},
		{name: "multiple clusters from one provider", targetClusters: map[string][]string{"provider": {"cluster-a", "cluster-b"}}, want: 2},
		{name: "multiple providers", targetClusters: map[string][]string{"provider-a": {"cluster-a"}, "provider-b": {"cluster-b"}}, want: 2},
		{name: "empty targets", targetClusters: map[string][]string{}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := targetClusterCount(tt.targetClusters); got != tt.want {
				t.Fatalf("targetClusterCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestExtractAndStoreReportsRejectsMultiClusterRun(t *testing.T) {
	scenarioRun := &krknv1alpha1.KrknScenarioRun{}
	scenarioRun.Name = "multi-cluster-run"
	scenarioRun.Spec.TargetClusters = map[string][]string{
		"provider": {"cluster-a", "cluster-b"},
	}

	reconciler := &KrknScenarioRunReconciler{}
	err := reconciler.extractAndStoreReports(context.Background(), scenarioRun, &krknv1alpha1.ClusterJobStatus{})
	if err != nil {
		t.Fatalf("extractAndStoreReports() returned an error: %v", err)
	}
	if scenarioRun.Status.ReportStatus == nil {
		t.Fatal("expected report status for unsupported multi-cluster run")
	}
	if !strings.Contains(scenarioRun.Status.ReportStatus.Message, "not supported for multi-cluster") {
		t.Fatalf("unexpected report status message: %q", scenarioRun.Status.ReportStatus.Message)
	}
}

func TestValidateReportPayloadSize(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]string
		binary  map[string][]byte
		wantErr bool
	}{
		{
			name:    "at supported limit",
			data:    map[string]string{"summary.html": strings.Repeat("h", int(maxReportConfigMapDataBytes))},
			binary:  map[string][]byte{},
			wantErr: false,
		},
		{
			name:    "over supported limit",
			data:    map[string]string{"summary.html": strings.Repeat("h", int(maxReportConfigMapDataBytes))},
			binary:  map[string][]byte{"summary.pdf": {0}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateReportPayloadSize(tt.data, tt.binary)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateReportPayloadSize() error = %v, wantErr %t", err, tt.wantErr)
			}
		})
	}
}
