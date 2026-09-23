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

package cloudcreds

import "strings"

// CloudEnvVarPrefixes are environment-variable prefixes that carry cloud
// provider credentials. When a cloudCredentialRef is set these must not appear
// as plaintext in the CRD — the controller injects them via SecretKeyRef.
var CloudEnvVarPrefixes = []string{
	"AWS_",
	"AZURE_",
	"OS_",
	"GOOGLE_",
	"BMC_",
	"VSPHERE_",
	"IBMC_",
}

// IsCloudEnvVar reports whether key is a cloud credential / CLOUD_TYPE env var
// that must be stripped from run requests when a cloudCredentialRef is set.
func IsCloudEnvVar(key string) bool {
	if key == "CLOUD_TYPE" || key == "DISKS" {
		return true
	}
	for _, prefix := range CloudEnvVarPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// StripCloudEnvVars returns a copy of env with cloud credential keys removed.
// Nil input yields nil. Used server-side so the secrecy invariant does not
// rely solely on the console stripping those fields.
func StripCloudEnvVars(env map[string]string) map[string]string {
	if env == nil {
		return nil
	}
	out := make(map[string]string, len(env))
	for k, v := range env {
		if IsCloudEnvVar(k) {
			continue
		}
		out[k] = v
	}
	return out
}
