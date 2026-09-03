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

package elasticsearch

import (
	"crypto/x509"
	"fmt"
	"strings"
)

// ValidateCreateRequest validates a CreateElasticsearchConfigRequest.
func ValidateCreateRequest(req *CreateElasticsearchConfigRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.Host == "" {
		return fmt.Errorf("host is required")
	}
	if req.Port < 0 || req.Port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535")
	}
	return validateTLSSettings(req.Host, req.Username, req.CACert)
}

// ValidateUpdateRequest validates an UpdateElasticsearchConfigRequest.
func ValidateUpdateRequest(req *UpdateElasticsearchConfigRequest) error {
	if req.Host == "" {
		return fmt.Errorf("host is required")
	}
	if req.Port < 0 || req.Port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535")
	}
	return validateTLSSettings(req.Host, req.Username, req.CACert)
}

// validateTLSSettings enforces that credentials are never sent over plaintext
// HTTP and that any supplied CA certificate is valid PEM. It is shared by the
// create and update paths so both reject insecure configurations up front.
func validateTLSSettings(host, username, caCert string) error {
	if username != "" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(host)), "http://") {
		return fmt.Errorf("credentials require a TLS connection; use an https host")
	}
	if caCert != "" {
		if !x509.NewCertPool().AppendCertsFromPEM([]byte(caCert)) {
			return fmt.Errorf("caCert must be a valid PEM-encoded certificate")
		}
	}
	return nil
}
