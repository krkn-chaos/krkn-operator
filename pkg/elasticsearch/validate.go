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
	"encoding/pem"
	"fmt"
	"net/url"
	"strings"

	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

// validateTLSSettings enforces that credentials are never sent over plaintext
// HTTP and that any supplied CA certificate is valid PEM. It is shared by the
// create and update paths so both reject insecure configurations up front.
func validateTLSSettings(host, username, caCert string) error {
	parsed, err := url.Parse(host)
	if err != nil {
		return fmt.Errorf("host must be a valid URL")
	}
	if username != "" && strings.EqualFold(parsed.Scheme, "http") {
		return fmt.Errorf("credentials require a TLS connection; use an https host")
	}
	if caCert == "" {
		return nil
	}

	rest := []byte(caCert)
	parsedCertificate := false
	for len(rest) > 0 {
		block, remaining := pem.Decode(rest)
		if block == nil {
			break
		}
		rest = remaining
		if block.Type != "CERTIFICATE" {
			continue
		}
		if _, err := x509.ParseCertificate(block.Bytes); err != nil {
			return fmt.Errorf("caCert must be a valid PEM-encoded certificate")
		}
		parsedCertificate = true
	}
	if !parsedCertificate {
		return fmt.Errorf("caCert must be a valid PEM-encoded certificate")
	}
	return nil
}

func validateAccess(groups []string, availableToAll bool) error {
	if len(groups) > 0 && availableToAll {
		return fmt.Errorf("an Elasticsearch config cannot be both public and assigned to a group")
	}
	for _, group := range groups {
		sanitized := groupauth.SanitizeGroupName(strings.TrimSpace(group))
		if sanitized == "" {
			return fmt.Errorf("group name %q is invalid", group)
		}
		if len(sanitized) > 63 {
			return fmt.Errorf("group name %q exceeds the 63-character label limit", group)
		}
	}
	return nil
}

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
	if err := validateTLSSettings(req.Host, req.Username, req.CACert); err != nil {
		return err
	}
	availableToAll := req.AvailableToAll != nil && *req.AvailableToAll
	return validateAccess(req.Groups, availableToAll)
}

// ValidateUpdateRequest validates an UpdateElasticsearchConfigRequest. A nil
// CACert means "leave the stored CA unchanged" and is not validated here; a
// non-nil value (including an explicit empty string that clears the stored CA)
// is validated as PEM.
func ValidateUpdateRequest(req *UpdateElasticsearchConfigRequest) error {
	if req.Host == "" {
		return fmt.Errorf("host is required")
	}
	if req.Port < 0 || req.Port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535")
	}
	caCert := ""
	if req.CACert != nil {
		caCert = *req.CACert
	}
	if err := validateTLSSettings(req.Host, req.Username, caCert); err != nil {
		return err
	}
	availableToAll := false
	if req.AvailableToAll != nil {
		availableToAll = *req.AvailableToAll
	}
	return validateAccess(req.Groups, availableToAll)
}

// ValidateMergedConnection validates a fully merged connection before it is
// persisted, using the effective username and CA that result after empty update
// fields fall back to the stored Secret values. An update can omit credentials to
// keep the existing ones, so the merged (not the request-only) view is what the
// query client will later enforce; validating it here prevents an update from
// returning success for a configuration the client would refuse (e.g. a retained
// username over a newly set http:// host).
func ValidateMergedConnection(host, username, caCert string) error {
	return validateTLSSettings(host, username, caCert)
}
