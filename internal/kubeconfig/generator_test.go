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

Assisted-by: Claude Sonnet 4.5 (claude-sonnet-4-5@20250929)
*/

package kubeconfig

import (
	"encoding/base64"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
)

func TestGenerateFromToken(t *testing.T) {
	tests := []struct {
		name            string
		clusterName     string
		apiURL          string
		caBundle        string
		token           string
		insecureSkipTLS bool
		wantErr         bool
	}{
		{
			name:            "valid token kubeconfig",
			clusterName:     "test-cluster",
			apiURL:          "https://api.test.com:6443",
			caBundle:        "",
			token:           "test-token-123",
			insecureSkipTLS: true,
			wantErr:         false,
		},
		{
			name:            "valid token with CA bundle",
			clusterName:     "prod-cluster",
			apiURL:          "https://api.prod.com:6443",
			caBundle:        "LS0tLS1CRUdJTi...",
			token:           "prod-token",
			insecureSkipTLS: false,
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeconfigBase64, err := GenerateFromToken(tt.clusterName, tt.apiURL, tt.caBundle, tt.token, tt.insecureSkipTLS)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateFromToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			// Validate the generated kubeconfig
			if err := Validate(kubeconfigBase64); err != nil {
				t.Errorf("Generated kubeconfig is invalid: %v", err)
			}

			// Decode and verify structure
			kubeconfigBytes, err := base64.StdEncoding.DecodeString(kubeconfigBase64)
			if err != nil {
				t.Errorf("Failed to decode kubeconfig: %v", err)
			}

			config, err := clientcmd.Load(kubeconfigBytes)
			if err != nil {
				t.Errorf("Failed to load kubeconfig: %v", err)
			}

			// Verify cluster
			if config.Clusters[tt.clusterName] == nil {
				t.Errorf("Cluster '%s' not found in kubeconfig", tt.clusterName)
			}

			if config.Clusters[tt.clusterName].Server != tt.apiURL {
				t.Errorf("Expected API URL %s, got %s", tt.apiURL, config.Clusters[tt.clusterName].Server)
			}

			// Verify user
			userName := tt.clusterName + "-user"
			if config.AuthInfos[userName] == nil {
				t.Errorf("User '%s' not found in kubeconfig", userName)
			}

			if config.AuthInfos[userName].Token != tt.token {
				t.Errorf("Expected token %s, got %s", tt.token, config.AuthInfos[userName].Token)
			}
		})
	}
}

func TestGenerateFromCredentials(t *testing.T) {
	tests := []struct {
		name            string
		clusterName     string
		apiURL          string
		caBundle        string
		username        string
		password        string
		insecureSkipTLS bool
		wantErr         bool
	}{
		{
			name:            "valid credentials kubeconfig",
			clusterName:     "test-cluster",
			apiURL:          "https://api.test.com:6443",
			caBundle:        "",
			username:        "admin",
			password:        "secret123",
			insecureSkipTLS: true,
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeconfigBase64, err := GenerateFromCredentials(tt.clusterName, tt.apiURL, tt.caBundle, tt.username, tt.password, tt.insecureSkipTLS)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateFromCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			// Validate the generated kubeconfig
			if err := Validate(kubeconfigBase64); err != nil {
				t.Errorf("Generated kubeconfig is invalid: %v", err)
			}

			// Decode and verify structure
			kubeconfigBytes, err := base64.StdEncoding.DecodeString(kubeconfigBase64)
			if err != nil {
				t.Errorf("Failed to decode kubeconfig: %v", err)
			}

			config, err := clientcmd.Load(kubeconfigBytes)
			if err != nil {
				t.Errorf("Failed to load kubeconfig: %v", err)
			}

			// Verify user credentials
			userName := tt.clusterName + "-user"
			if config.AuthInfos[userName].Username != tt.username {
				t.Errorf("Expected username %s, got %s", tt.username, config.AuthInfos[userName].Username)
			}

			if config.AuthInfos[userName].Password != tt.password {
				t.Errorf("Expected password %s, got %s", tt.password, config.AuthInfos[userName].Password)
			}
		})
	}
}

func TestExtractAPIURL(t *testing.T) {
	// Generate a test kubeconfig
	kubeconfigBase64, err := GenerateFromToken("test-cluster", "https://api.test.com:6443", "", "test-token", true)
	if err != nil {
		t.Fatalf("Failed to generate test kubeconfig: %v", err)
	}

	tests := []struct {
		name             string
		kubeconfigBase64 string
		wantURL          string
		wantErr          bool
	}{
		{
			name:             "valid kubeconfig",
			kubeconfigBase64: kubeconfigBase64,
			wantURL:          "https://api.test.com:6443",
			wantErr:          false,
		},
		{
			name:             "invalid base64",
			kubeconfigBase64: "not-valid-base64!!!",
			wantURL:          "",
			wantErr:          true,
		},
		{
			name:             "invalid yaml",
			kubeconfigBase64: base64.StdEncoding.EncodeToString([]byte("invalid: yaml: content:")),
			wantURL:          "",
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := ExtractAPIURL(tt.kubeconfigBase64)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractAPIURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if url != tt.wantURL {
				t.Errorf("ExtractAPIURL() = %v, want %v", url, tt.wantURL)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	// Generate a valid kubeconfig
	validKubeconfig, err := GenerateFromToken("test-cluster", "https://api.test.com:6443", "", "test-token", true)
	if err != nil {
		t.Fatalf("Failed to generate test kubeconfig: %v", err)
	}

	tests := []struct {
		name             string
		kubeconfigBase64 string
		wantErr          bool
	}{
		{
			name:             "valid kubeconfig",
			kubeconfigBase64: validKubeconfig,
			wantErr:          false,
		},
		{
			name:             "invalid base64",
			kubeconfigBase64: "not-valid-base64!!!",
			wantErr:          true,
		},
		{
			name:             "empty string",
			kubeconfigBase64: "",
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.kubeconfigBase64)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMarshalUnmarshalSecretData(t *testing.T) {
	kubeconfigBase64 := "YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmln"

	// Marshal
	secretBytes, err := MarshalSecretData(kubeconfigBase64)
	if err != nil {
		t.Fatalf("MarshalSecretData() error = %v", err)
	}

	// Unmarshal
	extractedKubeconfig, err := UnmarshalSecretData(secretBytes)
	if err != nil {
		t.Fatalf("UnmarshalSecretData() error = %v", err)
	}

	if extractedKubeconfig != kubeconfigBase64 {
		t.Errorf("Expected kubeconfig %s, got %s", kubeconfigBase64, extractedKubeconfig)
	}
}

func TestUnmarshalSecretData_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		secretBytes []byte
		wantErr     bool
	}{
		{
			name:        "invalid JSON",
			secretBytes: []byte("not json"),
			wantErr:     true,
		},
		{
			name:        "empty kubeconfig field",
			secretBytes: []byte(`{"kubeconfig":""}`),
			wantErr:     true,
		},
		{
			name:        "missing kubeconfig field",
			secretBytes: []byte(`{}`),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UnmarshalSecretData(tt.secretBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalSecretData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
