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

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestValidateCreateRequest(t *testing.T) {
	validGCPJSON := base64.StdEncoding.EncodeToString([]byte(`{"type":"service_account","project_id":"my-project"}`))

	tests := []struct {
		name    string
		req     CreateCloudCredentialRequest
		wantErr string
	}{
		{
			name:    "missing name",
			req:     CreateCloudCredentialRequest{Provider: ProviderAWS},
			wantErr: "name is required",
		},
		{
			name:    "reserved name 'available'",
			req:     CreateCloudCredentialRequest{Name: "available", Provider: ProviderAWS},
			wantErr: "reserved",
		},
		{
			name:    "missing provider",
			req:     CreateCloudCredentialRequest{Name: "test"},
			wantErr: "provider is required",
		},
		{
			name:    "invalid provider",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: "oracle"},
			wantErr: "provider must be one of",
		},
		// AWS
		{
			name:    "aws missing access key id",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderAWS, AWSSecretAccessKey: "s", AWSDefaultRegion: "r"},
			wantErr: "awsAccessKeyId is required",
		},
		{
			name:    "aws missing secret access key",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderAWS, AWSAccessKeyID: "k", AWSDefaultRegion: "r"},
			wantErr: "awsSecretAccessKey is required",
		},
		{
			name:    "aws missing region",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderAWS, AWSAccessKeyID: "k", AWSSecretAccessKey: "s"},
			wantErr: "awsDefaultRegion is required",
		},
		{
			name: "aws valid",
			req:  CreateCloudCredentialRequest{Name: "test", Provider: ProviderAWS, AWSAccessKeyID: "AKIAIOSFODNN7EXAMPLE", AWSSecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", AWSDefaultRegion: "us-east-1"},
		},
		// GCP
		{
			name:    "gcp missing service account json",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderGCP},
			wantErr: "gcpServiceAccountJson is required",
		},
		{
			name:    "gcp invalid base64",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderGCP, GCPServiceAccountJSON: "not-base64!!!"},
			wantErr: "valid base64",
		},
		{
			name:    "gcp invalid json",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderGCP, GCPServiceAccountJSON: base64.StdEncoding.EncodeToString([]byte("not json"))},
			wantErr: "valid JSON",
		},
		{
			name:    "gcp missing type field",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderGCP, GCPServiceAccountJSON: base64.StdEncoding.EncodeToString([]byte(`{"project_id":"p"}`))},
			wantErr: "missing required field 'type'",
		},
		{
			name:    "gcp missing project_id field",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderGCP, GCPServiceAccountJSON: base64.StdEncoding.EncodeToString([]byte(`{"type":"service_account"}`))},
			wantErr: "missing required field 'project_id'",
		},
		{
			name: "gcp valid",
			req:  CreateCloudCredentialRequest{Name: "test", Provider: ProviderGCP, GCPServiceAccountJSON: validGCPJSON},
		},
		// Azure
		{
			name:    "azure missing tenant id",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderAzure, AzureClientID: "c", AzureClientSecret: "s", AzureSubscriptionID: "sub"},
			wantErr: "azureTenantId is required",
		},
		{
			name:    "azure missing client id",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderAzure, AzureTenantID: "t", AzureClientSecret: "s", AzureSubscriptionID: "sub"},
			wantErr: "azureClientId is required",
		},
		{
			name:    "azure missing client secret",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderAzure, AzureTenantID: "t", AzureClientID: "c", AzureSubscriptionID: "sub"},
			wantErr: "azureClientSecret is required",
		},
		{
			name:    "azure missing subscription id",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderAzure, AzureTenantID: "t", AzureClientID: "c", AzureClientSecret: "s"},
			wantErr: "azureSubscriptionId is required",
		},
		{
			name: "azure valid",
			req:  CreateCloudCredentialRequest{Name: "test", Provider: ProviderAzure, AzureTenantID: "t", AzureClientID: "c", AzureClientSecret: "s", AzureSubscriptionID: "sub"},
		},
		// OpenStack
		{
			name:    "openstack missing auth url",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderOpenStack, OSUsername: "u", OSPassword: "p", OSProjectName: "proj"},
			wantErr: "osAuthUrl is required",
		},
		{
			name:    "openstack missing username",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderOpenStack, OSAuthURL: "http://auth", OSPassword: "p", OSProjectName: "proj"},
			wantErr: "osUsername is required",
		},
		{
			name:    "openstack missing password",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderOpenStack, OSAuthURL: "http://auth", OSUsername: "u", OSProjectName: "proj"},
			wantErr: "osPassword is required",
		},
		{
			name:    "openstack missing project name",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderOpenStack, OSAuthURL: "http://auth", OSUsername: "u", OSPassword: "p"},
			wantErr: "osProjectName is required",
		},
		{
			name: "openstack valid",
			req:  CreateCloudCredentialRequest{Name: "test", Provider: ProviderOpenStack, OSAuthURL: "http://auth:5000/v3", OSUsername: "admin", OSPassword: "secret", OSProjectName: "myproject"},
		},
		// Baremetal
		{
			name:    "baremetal missing bmc user",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderBaremetal, BMCPassword: "p", BMCAddr: "192.168.1.100"},
			wantErr: "bmcUser is required",
		},
		{
			name:    "baremetal missing bmc password",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderBaremetal, BMCUser: "admin", BMCAddr: "192.168.1.100"},
			wantErr: "bmcPassword is required",
		},
		{
			name:    "baremetal missing bmc addr",
			req:     CreateCloudCredentialRequest{Name: "test", Provider: ProviderBaremetal, BMCUser: "admin", BMCPassword: "p"},
			wantErr: "bmcAddr is required",
		},
		{
			name: "baremetal valid",
			req:  CreateCloudCredentialRequest{Name: "test", Provider: ProviderBaremetal, BMCUser: "admin", BMCPassword: "secret", BMCAddr: "192.168.1.100"},
		},
		// Reserved names
		{
			name:    "reserved name 'baremetal'",
			req:     CreateCloudCredentialRequest{Name: "baremetal", Provider: ProviderBaremetal, BMCUser: "u", BMCPassword: "p", BMCAddr: "a"},
			wantErr: "reserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateRequest(&tt.req)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.wantErr)
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidateUpdateRequest(t *testing.T) {
	tests := []struct {
		name             string
		req              UpdateCloudCredentialRequest
		existingProvider string
		wantErr          string
	}{
		{
			name:             "gcp invalid json on update",
			req:              UpdateCloudCredentialRequest{GCPServiceAccountJSON: base64.StdEncoding.EncodeToString([]byte("not json"))},
			existingProvider: ProviderGCP,
			wantErr:          "valid JSON",
		},
		{
			name:             "gcp empty json on update is ok (keep existing)",
			req:              UpdateCloudCredentialRequest{},
			existingProvider: ProviderGCP,
		},
		{
			name:             "aws update with partial fields is ok",
			req:              UpdateCloudCredentialRequest{AWSDefaultRegion: "eu-west-1"},
			existingProvider: ProviderAWS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateRequest(&tt.req, tt.existingProvider)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.wantErr)
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
