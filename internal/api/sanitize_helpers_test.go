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
	"testing"

	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

// mustSanitizeUserIDForResourceName is a test helper that sanitizes a user ID
// into a Kubernetes resource name and fails the test if the (known-good) input
// unexpectedly produces an error. It keeps table-driven tests concise now that
// groupauth.SanitizeUserIDForResourceName returns an error.
func mustSanitizeUserIDForResourceName(t *testing.T, email string) string {
	t.Helper()
	name, err := groupauth.SanitizeUserIDForResourceName(email)
	if err != nil {
		t.Fatalf("SanitizeUserIDForResourceName(%q) unexpected error: %v", email, err)
	}
	return name
}
