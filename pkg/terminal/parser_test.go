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
	"reflect"
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *ParsedCommand
		wantErr bool
	}{
		{
			name:  "basic kubectl get pods",
			input: "kubectl get pods",
			want: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "get",
				Args:         []string{"pods"},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name:  "kubectl with namespace flag",
			input: "kubectl get pods -n default",
			want: &ParsedCommand{
				Command:    "kubectl",
				Subcommand: "get",
				Args:       []string{"pods"},
				Flags: map[string]string{
					"n": "default",
				},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name:  "kubectl with long flag using equals",
			input: "kubectl get pods --namespace=kube-system",
			want: &ParsedCommand{
				Command:    "kubectl",
				Subcommand: "get",
				Args:       []string{"pods"},
				Flags: map[string]string{
					"namespace": "kube-system",
				},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name:  "kubectl with long flag using space",
			input: "kubectl get pods --namespace kube-system",
			want: &ParsedCommand{
				Command:    "kubectl",
				Subcommand: "get",
				Args:       []string{"pods"},
				Flags: map[string]string{
					"namespace": "kube-system",
				},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name:  "kubectl with boolean flag",
			input: "kubectl get pods --all-namespaces",
			want: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "get",
				Args:         []string{"pods"},
				Flags:        map[string]string{},
				BooleanFlags: []string{"all-namespaces"},
			},
			wantErr: false,
		},
		{
			name:  "kubectl with multiple flags",
			input: "kubectl get pods -n default --output=yaml --show-labels",
			want: &ParsedCommand{
				Command:    "kubectl",
				Subcommand: "get",
				Args:       []string{"pods"},
				Flags: map[string]string{
					"n":      "default",
					"output": "yaml",
				},
				BooleanFlags: []string{"show-labels"},
			},
			wantErr: false,
		},
		{
			name:  "kubectl with resource and name",
			input: "kubectl get pod nginx",
			want: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "get",
				Args:         []string{"pod", "nginx"},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name:  "kubectl describe with multiple args",
			input: "kubectl describe pod nginx -n default",
			want: &ParsedCommand{
				Command:    "kubectl",
				Subcommand: "describe",
				Args:       []string{"pod", "nginx"},
				Flags: map[string]string{
					"n": "default",
				},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name:  "oc command",
			input: "oc get routes",
			want: &ParsedCommand{
				Command:      "oc",
				Subcommand:   "get",
				Args:         []string{"routes"},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name:  "kubectl logs with flags",
			input: "kubectl logs nginx --tail=100 --timestamps",
			want: &ParsedCommand{
				Command:    "kubectl",
				Subcommand: "logs",
				Args:       []string{"nginx"},
				Flags: map[string]string{
					"tail": "100",
				},
				BooleanFlags: []string{"timestamps"},
			},
			wantErr: false,
		},
		{
			name:  "quoted pod name with spaces",
			input: `kubectl get pod "my pod with spaces"`,
			want: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "get",
				Args:         []string{"pod", "my pod with spaces"},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name:  "quoted namespace value",
			input: `kubectl get pods -n "my namespace"`,
			want: &ParsedCommand{
				Command:    "kubectl",
				Subcommand: "get",
				Args:       []string{"pods"},
				Flags: map[string]string{
					"n": "my namespace",
				},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name:  "single quotes",
			input: `kubectl get pod 'my-pod'`,
			want: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "get",
				Args:         []string{"pod", "my-pod"},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name:  "mixed quotes",
			input: `kubectl get pod "nginx" -n 'default'`,
			want: &ParsedCommand{
				Command:    "kubectl",
				Subcommand: "get",
				Args:       []string{"pod", "nginx"},
				Flags: map[string]string{
					"n": "default",
				},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name:  "flag value with equals sign in quotes",
			input: `kubectl get pods -l "app=nginx"`,
			want: &ParsedCommand{
				Command:    "kubectl",
				Subcommand: "get",
				Args:       []string{"pods"},
				Flags: map[string]string{
					"l": "app=nginx",
				},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name:  "complex real-world command",
			input: `kubectl get pods -n kube-system -l "app=nginx" --output=json --show-labels`,
			want: &ParsedCommand{
				Command:    "kubectl",
				Subcommand: "get",
				Args:       []string{"pods"},
				Flags: map[string]string{
					"n":      "kube-system",
					"l":      "app=nginx",
					"output": "json",
				},
				BooleanFlags: []string{"show-labels"},
			},
			wantErr: false,
		},
		{
			name:    "empty command",
			input:   "",
			want:    nil,
			wantErr: true,
		},
		{
			name:  "only command, no subcommand",
			input: "kubectl",
			want: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "",
				Args:         []string{},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name:    "unclosed quote",
			input:   `kubectl get pods "unclosed`,
			want:    nil,
			wantErr: true,
		},
		{
			name:  "multiple spaces between tokens",
			input: "kubectl    get     pods    -n    default",
			want: &ParsedCommand{
				Command:    "kubectl",
				Subcommand: "get",
				Args:       []string{"pods"},
				Flags: map[string]string{
					"n": "default",
				},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name:  "tabs between tokens",
			input: "kubectl\tget\tpods\t-n\tdefault",
			want: &ParsedCommand{
				Command:    "kubectl",
				Subcommand: "get",
				Args:       []string{"pods"},
				Flags: map[string]string{
					"n": "default",
				},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name:  "trailing spaces",
			input: "kubectl get pods   ",
			want: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "get",
				Args:         []string{"pods"},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name:  "leading spaces",
			input: "   kubectl get pods",
			want: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "get",
				Args:         []string{"pods"},
				Flags:        map[string]string{},
				BooleanFlags: []string{},
			},
			wantErr: false,
		},
		{
			name:  "boolean flag at end of command",
			input: "kubectl get pods -n default -A",
			want: &ParsedCommand{
				Command:    "kubectl",
				Subcommand: "get",
				Args:       []string{"pods"},
				Flags: map[string]string{
					"n": "default",
				},
				BooleanFlags: []string{"A"},
			},
			wantErr: false,
		},
		{
			name:  "multiple boolean flags",
			input: "kubectl get pods --all-namespaces --show-labels -w",
			want: &ParsedCommand{
				Command:      "kubectl",
				Subcommand:   "get",
				Args:         []string{"pods"},
				Flags:        map[string]string{},
				BooleanFlags: []string{"all-namespaces", "show-labels", "w"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseCommand() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:    "simple tokens",
			input:   "kubectl get pods",
			want:    []string{"kubectl", "get", "pods"},
			wantErr: false,
		},
		{
			name:    "double quoted string",
			input:   `kubectl get "my pod"`,
			want:    []string{"kubectl", "get", "my pod"},
			wantErr: false,
		},
		{
			name:    "single quoted string",
			input:   `kubectl get 'my pod'`,
			want:    []string{"kubectl", "get", "my pod"},
			wantErr: false,
		},
		{
			name:    "mixed quotes",
			input:   `kubectl "get" 'pods'`,
			want:    []string{"kubectl", "get", "pods"},
			wantErr: false,
		},
		{
			name:    "quote inside different quote type",
			input:   `kubectl get "pod's name"`,
			want:    []string{"kubectl", "get", "pod's name"},
			wantErr: false,
		},
		{
			name:    "unclosed double quote",
			input:   `kubectl get "unclosed`,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unclosed single quote",
			input:   `kubectl get 'unclosed`,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unclosed quote with multi-byte last char (emoji)",
			input:   `kubectl get "test 😀`,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unclosed quote with multi-byte last char (unicode)",
			input:   `kubectl get "test 中文`,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "only spaces",
			input:   "   ",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "multiple spaces between tokens",
			input:   "kubectl    get     pods",
			want:    []string{"kubectl", "get", "pods"},
			wantErr: false,
		},
		{
			name:    "tabs between tokens",
			input:   "kubectl\tget\tpods",
			want:    []string{"kubectl", "get", "pods"},
			wantErr: false,
		},
		{
			name:    "empty quoted string",
			input:   `kubectl get ""`,
			want:    []string{"kubectl", "get"},
			wantErr: false,
		},
		{
			name:    "spaces in quoted string",
			input:   `kubectl get "pod  with  spaces"`,
			want:    []string{"kubectl", "get", "pod  with  spaces"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tokenize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("tokenize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("tokenize() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseLongFlag(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantFlagName  string
		wantFlagValue string
		wantHasValue  bool
	}{
		{
			name:          "flag with embedded value",
			input:         "--output=yaml",
			wantFlagName:  "output",
			wantFlagValue: "yaml",
			wantHasValue:  true,
		},
		{
			name:          "flag without value",
			input:         "--all-namespaces",
			wantFlagName:  "all-namespaces",
			wantFlagValue: "",
			wantHasValue:  false,
		},
		{
			name:          "flag with empty value",
			input:         "--output=",
			wantFlagName:  "output",
			wantFlagValue: "",
			wantHasValue:  true,
		},
		{
			name:          "flag with equals in value",
			input:         "--selector=app=nginx",
			wantFlagName:  "selector",
			wantFlagValue: "app=nginx",
			wantHasValue:  true,
		},
		{
			name:          "flag with multiple equals",
			input:         "--label=key=value=extra",
			wantFlagName:  "label",
			wantFlagValue: "key=value=extra",
			wantHasValue:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotValue, gotHasValue := parseLongFlag(tt.input)
			if gotName != tt.wantFlagName {
				t.Errorf("parseLongFlag() flagName = %v, want %v", gotName, tt.wantFlagName)
			}
			if gotValue != tt.wantFlagValue {
				t.Errorf("parseLongFlag() flagValue = %v, want %v", gotValue, tt.wantFlagValue)
			}
			if gotHasValue != tt.wantHasValue {
				t.Errorf("parseLongFlag() hasValue = %v, want %v", gotHasValue, tt.wantHasValue)
			}
		})
	}
}
