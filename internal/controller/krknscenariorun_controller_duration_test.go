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

package controller

import "testing"

func TestDurationToActiveDeadline(t *testing.T) {
	tests := []struct {
		name        string
		duration    string
		wantSeconds *int64
		wantErr     bool
	}{
		{name: "empty means no deadline", duration: "", wantSeconds: nil, wantErr: false},
		{name: "seconds", duration: "30s", wantSeconds: ptrInt64(30), wantErr: false},
		{name: "minutes", duration: "5m", wantSeconds: ptrInt64(300), wantErr: false},
		{name: "compound", duration: "1h30m", wantSeconds: ptrInt64(5400), wantErr: false},
		// Sub-second and fractional durations must round UP, never truncate to 0
		// (a 0 activeDeadlineSeconds is rejected by the Kubernetes API).
		{name: "sub-second rounds up to 1", duration: "500ms", wantSeconds: ptrInt64(1), wantErr: false},
		{name: "tiny value rounds up to 1", duration: "1ns", wantSeconds: ptrInt64(1), wantErr: false},
		{name: "fractional seconds round up", duration: "1.9s", wantSeconds: ptrInt64(2), wantErr: false},
		{name: "invalid", duration: "notaduration", wantSeconds: nil, wantErr: true},
		{name: "zero", duration: "0s", wantSeconds: nil, wantErr: true},
		{name: "negative", duration: "-5m", wantSeconds: nil, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := durationToActiveDeadline(tt.duration)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error for duration %q, got nil", tt.duration)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for duration %q: %v", tt.duration, err)
			}
			switch {
			case tt.wantSeconds == nil && got != nil:
				t.Fatalf("expected nil, got %d", *got)
			case tt.wantSeconds != nil && got == nil:
				t.Fatalf("expected %d, got nil", *tt.wantSeconds)
			case tt.wantSeconds != nil && got != nil && *got != *tt.wantSeconds:
				t.Fatalf("expected %d, got %d", *tt.wantSeconds, *got)
			}
		})
	}
}

func ptrInt64(v int64) *int64 {
	return &v
}
