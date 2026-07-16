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
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "valid - simple",
			email:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "valid - subdomain and plus tag",
			email:   "user.name+tag@mail.example.co.uk",
			wantErr: false,
		},
		{
			name:    "invalid - empty",
			email:   "",
			wantErr: true,
		},
		{
			name:    "invalid - no @",
			email:   "notanemail",
			wantErr: true,
		},
		{
			name:    "invalid - double @",
			email:   "a@@b.com",
			wantErr: true,
		},
		{
			name:    "invalid - missing domain",
			email:   "user@",
			wantErr: true,
		},
		{
			name:    "invalid - missing local part",
			email:   "@example.com",
			wantErr: true,
		},
		{
			name:    "invalid - space in address",
			email:   "user name@example.com",
			wantErr: true,
		},
		{
			// Local part is 244 chars + "@a.co" (5) = 249, valid length but keep it
			// under the limit; then a variant that exceeds 254.
			name:    "invalid - exceeds max length",
			email:   strings.Repeat("a", 250) + "@example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}
