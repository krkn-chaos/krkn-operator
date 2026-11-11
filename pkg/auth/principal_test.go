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

package auth

import (
	"strings"
	"testing"
	"time"
)

func TestRobotPrincipalID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid", "ci-bot", "robot:ci-bot", false},
		{"empty", "", "", true},
		{"uppercase", "CI", "", true},
		{"underscore", "a_b", "", true},
		{"too long", strings.Repeat("a", 49), "", true},
		{"leading dash", "-a", "", true},
		{"single char", "a", "robot:a", false},
		{"max length", strings.Repeat("a", 48), "robot:" + strings.Repeat("a", 48), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RobotPrincipalID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("RobotPrincipalID(%q) = %q, want error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("RobotPrincipalID(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("RobotPrincipalID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseRobotPrincipalID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		wantName string
		wantOK   bool
	}{
		{"valid", "robot:ci-bot", "ci-bot", true},
		{"empty name", "robot:", "", false},
		{"invalid name", "robot:Bad", "", false},
		{"email", "user@example.com", "", false},
		{"no colon", "robotci", "", false},
		{"empty", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, ok := ParseRobotPrincipalID(tt.id)
			if ok != tt.wantOK {
				t.Fatalf("ParseRobotPrincipalID(%q) ok = %v, want %v", tt.id, ok, tt.wantOK)
			}
			if ok && name != tt.wantName {
				t.Errorf("ParseRobotPrincipalID(%q) name = %q, want %q", tt.id, name, tt.wantName)
			}
		})
	}
}

func TestIsValidRobotName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid", "ci-bot", true},
		{"single", "a", true},
		{"numeric", "bot123", true},
		{"empty", "", false},
		{"uppercase", "Bot", false},
		{"underscore", "a_b", false},
		{"leading dash", "-a", false},
		{"trailing dash", "a-", false},
		{"dots", "a.b", false},
		{"colons", "a:b", false},
		{"49 chars", strings.Repeat("a", 49), false},
		{"48 chars", strings.Repeat("a", 48), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidRobotName(tt.input); got != tt.want {
				t.Errorf("IsValidRobotName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateToken_RejectsRobotRole(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)
	token, err := tg.GenerateToken("x@y.io", string(RoleRobot), "X", "Y", "")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	_, err = tg.ValidateToken(token)
	if err == nil {
		t.Fatal("ValidateToken accepted a robot-role token, want error")
	}
	if !strings.Contains(err.Error(), "robot") {
		t.Errorf("error %q does not mention robot", err.Error())
	}
}

func TestValidateToken_RejectsRobotPrincipal(t *testing.T) {
	tg := NewTokenGenerator(
		[]byte("test-secret-key-at-least-32-bytes-long"),
		24*time.Hour,
		"krkn-operator",
	)
	token, err := tg.GenerateToken("robot:ci-bot", "user", "X", "Y", "")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	_, err = tg.ValidateToken(token)
	if err == nil {
		t.Fatal("ValidateToken accepted a robot-principal token, want error")
	}
}
