package files

import (
	"strings"
	"testing"
)

func TestValidateFileType(t *testing.T) {
	tests := []struct {
		name     string
		fileType string
		wantErr  bool
	}{
		{name: "empty means untyped", fileType: ""},
		{name: "simple name", fileType: "alerts"},
		{name: "dashes and dots", fileType: "kube-config.v2"},
		{name: "mixed case and underscores", fileType: "Alerts_Team"},
		{name: "too long", fileType: strings.Repeat("a", 64), wantErr: true},
		{name: "space", fileType: "my alerts", wantErr: true},
		{name: "leading dash", fileType: "-alerts", wantErr: true},
		{name: "slash", fileType: "alerts/prod", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileType(tt.fileType)
			if tt.wantErr && err == nil {
				t.Fatalf("expected an error for %q", tt.fileType)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected %q to be valid, got %v", tt.fileType, err)
			}
		})
	}
}

func TestApplyFileType(t *testing.T) {
	t.Run("sets a type", func(t *testing.T) {
		labels := map[string]string{AppNameLabel: AppName}
		ApplyFileType(labels, "alerts")

		if got := ExtractFileTypeFromLabels(labels); got != "alerts" {
			t.Fatalf("expected %q, got %q", "alerts", got)
		}
		if labels[FileTypeLabelPrefix+"alerts"] != "true" {
			t.Fatalf("expected the type label, got %v", labels)
		}
	})

	t.Run("replaces an existing type", func(t *testing.T) {
		labels := map[string]string{
			FileTypeLabelPrefix + "alerts":  "true",
			FileTypeLabelPrefix + "metrics": "true",
		}
		ApplyFileType(labels, "metrics")

		if got := ExtractFileTypeFromLabels(labels); got != "metrics" {
			t.Fatalf("expected %q, got %q", "metrics", got)
		}
		if _, stale := labels[FileTypeLabelPrefix+"alerts"]; stale {
			t.Fatalf("expected the previous type label to be removed, got %v", labels)
		}
	})

	t.Run("empty clears the type and keeps other labels", func(t *testing.T) {
		labels := map[string]string{
			FileTypeLabelPrefix + "alerts": "true",
			AppNameLabel:                   AppName,
		}
		ApplyFileType(labels, "")

		if got := ExtractFileTypeFromLabels(labels); got != "" {
			t.Fatalf("expected the type to be cleared, got %q", got)
		}
		if labels[AppNameLabel] != AppName {
			t.Fatalf("expected unrelated labels to be preserved, got %v", labels)
		}
	})
}

func TestExtractFileTypeFromLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   string
	}{
		{name: "no labels", labels: nil, want: ""},
		{name: "no type", labels: map[string]string{AppNameLabel: AppName}, want: ""},
		{
			name:   "ignores non-true values",
			labels: map[string]string{FileTypeLabelPrefix + "alerts": "false"},
			want:   "",
		},
		{
			name:   "reads the type",
			labels: map[string]string{FileTypeLabelPrefix + "alerts": "true"},
			want:   "alerts",
		},
		{
			name:   "reads the historical label key",
			labels: map[string]string{"file-type.krkn.krkn-chaos.dev/kubeconfig": "true"},
			want:   "kubeconfig",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractFileTypeFromLabels(tt.labels); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
