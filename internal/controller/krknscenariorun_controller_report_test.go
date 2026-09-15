package controller

import (
	"encoding/base64"
	"strings"
	"testing"
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
