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

package terminal

import (
	"strings"
	"testing"
)

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name    string
		cmd     *ParsedCommand
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid kubectl get",
			cmd: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "get",
				Args:         []string{"pods"},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name: "valid oc get",
			cmd: &ParsedCommand{
				Command:      "oc",
				Subcommand:   "get",
				Args:         []string{"routes"},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name: "valid kubectl describe",
			cmd: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "describe",
				Args:         []string{"pod", "nginx"},
				Flags:        map[string]string{"namespace": "default"},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name: "valid kubectl logs",
			cmd: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "logs",
				Args:         []string{"nginx"},
				Flags:        map[string]string{"tail": "100"},
				BooleanFlags: []string{"timestamps"},
			},
			wantErr: false,
		},
		{
			name: "valid kubectl top",
			cmd: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "top",
				Args:         []string{"nodes"},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name: "valid kubectl explain",
			cmd: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "explain",
				Args:         []string{"pods"},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name: "valid kubectl version",
			cmd: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "version",
				Args:         []string{},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name: "invalid command - not kubectl or oc",
			cmd: &ParsedCommand{
				Command:      "bash",
				Subcommand:   "get",
				Args:         []string{},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: true,
			errMsg:  "command must be 'kubectl' or 'oc'",
		},
		{
			name: "invalid subcommand - write operation",
			cmd: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "delete",
				Args:         []string{"pod", "nginx"},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: true,
			errMsg:  "is not permitted",
		},
		{
			name: "invalid subcommand - apply",
			cmd: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "apply",
				Args:         []string{},
				Flags:        map[string]string{"filename": "deploy.yaml"},
				BooleanFlags: []string{},
			},
			wantErr: true,
			errMsg:  "is not permitted",
		},
		{
			name: "invalid subcommand - create",
			cmd: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "create",
				Args:         []string{},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: true,
			errMsg:  "is not permitted",
		},
		{
			name: "blocked flag - watch",
			cmd: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "get",
				Args:         []string{"pods"},
				Flags:        map[string]string{},
				BooleanFlags: []string{"watch"},
			},
			wantErr: true,
			errMsg:  "streaming commands blocked",
		},
		{
			name: "blocked flag - w (short)",
			cmd: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "get",
				Args:         []string{"pods"},
				Flags:        map[string]string{},
				BooleanFlags: []string{"w"},
			},
			wantErr: true,
			errMsg:  "streaming commands blocked",
		},
		{
			name: "blocked flag - follow",
			cmd: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "logs",
				Args:         []string{"nginx"},
				Flags:        map[string]string{},
				BooleanFlags: []string{"follow"},
			},
			wantErr: true,
			errMsg:  "streaming commands blocked",
		},
		{
			name: "blocked flag - f (short)",
			cmd: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "logs",
				Args:         []string{"nginx"},
				Flags:        map[string]string{},
				BooleanFlags: []string{"f"},
			},
			wantErr: true,
			errMsg:  "streaming commands blocked",
		},
		{
			name: "blocked flag - watch with value",
			cmd: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "get",
				Args:         []string{"pods"},
				Flags:        map[string]string{"watch": "true"},
				BooleanFlags: []string{},
			},
			wantErr: true,
			errMsg:  "streaming commands blocked",
		},
		{
			name: "blocked flag - watch-only",
			cmd: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "get",
				Args:         []string{"pods"},
				Flags:        map[string]string{},
				BooleanFlags: []string{"watch-only"},
			},
			wantErr: true,
			errMsg:  "streaming commands blocked",
		},
		{
			name: "valid command with allowed flags",
			cmd: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "get",
				Args:         []string{"pods"},
				Flags:        map[string]string{"namespace": "default", "output": "yaml"},
				BooleanFlags: []string{"show-labels", "all-namespaces"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCommand(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateCommand() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestIsStreamingFlag(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		want     bool
	}{
		{"watch flag", "watch", true},
		{"w flag", "w", true},
		{"follow flag", "follow", true},
		{"f flag", "f", true},
		{"watch-only flag", "watch-only", true},
		{"namespace flag", "namespace", false},
		{"output flag", "output", false},
		{"all-namespaces flag", "all-namespaces", false},
		{"show-labels flag", "show-labels", false},
		{"with dashes", "--watch", true},
		{"with single dash", "-w", true},
		{"uppercase", "WATCH", true},
		{"mixed case", "Follow", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStreamingFlag(tt.flagName); got != tt.want {
				t.Errorf("IsStreamingFlag(%q) = %v, want %v", tt.flagName, got, tt.want)
			}
		})
	}
}
