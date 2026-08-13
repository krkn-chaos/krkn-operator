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
	"encoding/json"
	"fmt"
)

// reservedNames are credential names that would collide with API sub-path routes
var reservedNames = map[string]bool{
	"available": true,
	"aws":       true,
	"gcp":       true,
	"azure":     true,
	"openstack": true,
	"baremetal": true,
	"vmware":    true,
	"ibmcloud":  true,
}

// ValidateCreateRequest validates a CreateCloudCredentialRequest
func ValidateCreateRequest(req *CreateCloudCredentialRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if reservedNames[req.Name] {
		return fmt.Errorf("name '%s' is reserved and cannot be used", req.Name)
	}
	if req.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if !ValidProviders[req.Provider] {
		return fmt.Errorf("provider must be one of: aws, gcp, azure, openstack")
	}

	return validateProviderFields(req.Provider, req.AWSAccessKeyID, req.AWSSecretAccessKey, req.AWSDefaultRegion,
		req.GCPServiceAccountJSON,
		req.AzureTenantID, req.AzureClientID, req.AzureClientSecret, req.AzureSubscriptionID,
		req.OSAuthURL, req.OSUsername, req.OSPassword, req.OSProjectName,
		req.BMCUser, req.BMCPassword, req.BMCAddr,
		req.VSphereIP, req.VSphereUsername, req.VSpherePassword,
		req.IBMCURL, req.IBMCAPIKey)
}

// ValidateUpdateRequest validates an UpdateCloudCredentialRequest against the existing provider
func ValidateUpdateRequest(req *UpdateCloudCredentialRequest, existingProvider string) error {
	return validateProviderFieldsForUpdate(existingProvider,
		req.AWSAccessKeyID, req.AWSSecretAccessKey, req.AWSDefaultRegion,
		req.GCPServiceAccountJSON,
		req.AzureTenantID, req.AzureClientID, req.AzureClientSecret, req.AzureSubscriptionID,
		req.OSAuthURL, req.OSUsername, req.OSPassword, req.OSProjectName,
		req.BMCUser, req.BMCPassword, req.BMCAddr,
		req.VSphereIP, req.VSphereUsername, req.VSpherePassword,
		req.IBMCURL, req.IBMCAPIKey)
}

func validateProviderFields(provider string,
	awsKeyID, awsSecret, awsRegion string,
	gcpJSON string,
	azTenant, azClient, azSecret, azSub string,
	osAuth, osUser, osPass, osProject string,
	bmcUser, bmcPass, bmcAddr string,
	vsphereIP, vsphereUser, vspherePass string,
	ibmcURL, ibmcAPIKey string,
) error {
	switch provider {
	case ProviderAWS:
		if awsKeyID == "" {
			return fmt.Errorf("awsAccessKeyId is required for AWS provider")
		}
		if awsSecret == "" {
			return fmt.Errorf("awsSecretAccessKey is required for AWS provider")
		}
		if awsRegion == "" {
			return fmt.Errorf("awsDefaultRegion is required for AWS provider")
		}
	case ProviderGCP:
		if gcpJSON == "" {
			return fmt.Errorf("gcpServiceAccountJson is required for GCP provider")
		}
		if err := validateGCPServiceAccountJSON(gcpJSON); err != nil {
			return fmt.Errorf("invalid gcpServiceAccountJson: %w", err)
		}
	case ProviderAzure:
		if azTenant == "" {
			return fmt.Errorf("azureTenantId is required for Azure provider")
		}
		if azClient == "" {
			return fmt.Errorf("azureClientId is required for Azure provider")
		}
		if azSecret == "" {
			return fmt.Errorf("azureClientSecret is required for Azure provider")
		}
		if azSub == "" {
			return fmt.Errorf("azureSubscriptionId is required for Azure provider")
		}
	case ProviderOpenStack:
		if osAuth == "" {
			return fmt.Errorf("osAuthUrl is required for OpenStack provider")
		}
		if osUser == "" {
			return fmt.Errorf("osUsername is required for OpenStack provider")
		}
		if osPass == "" {
			return fmt.Errorf("osPassword is required for OpenStack provider")
		}
		if osProject == "" {
			return fmt.Errorf("osProjectName is required for OpenStack provider")
		}
	case ProviderBaremetal:
		if bmcUser == "" {
			return fmt.Errorf("bmcUser is required for Baremetal provider")
		}
		if bmcPass == "" {
			return fmt.Errorf("bmcPassword is required for Baremetal provider")
		}
		if bmcAddr == "" {
			return fmt.Errorf("bmcAddr is required for Baremetal provider")
		}
	case ProviderVMware:
		if vsphereIP == "" {
			return fmt.Errorf("vsphereIp is required for VMware provider")
		}
		if vsphereUser == "" {
			return fmt.Errorf("vsphereUsername is required for VMware provider")
		}
		if vspherePass == "" {
			return fmt.Errorf("vspherePassword is required for VMware provider")
		}
	case ProviderIBMCloud:
		if ibmcURL == "" {
			return fmt.Errorf("ibmcUrl is required for IBM Cloud provider")
		}
		if ibmcAPIKey == "" {
			return fmt.Errorf("ibmcApikey is required for IBM Cloud provider")
		}
	}
	return nil
}

// validateProviderFieldsForUpdate validates fields for an update request.
// On update, fields are optional (empty means "keep existing"), so we only
// validate non-empty values.
func validateProviderFieldsForUpdate(provider string,
	awsKeyID, awsSecret, awsRegion string,
	gcpJSON string,
	azTenant, azClient, azSecret, azSub string,
	osAuth, osUser, osPass, osProject string,
	bmcUser, bmcPass, bmcAddr string,
	vsphereIP, vsphereUser, vspherePass string,
	ibmcURL, ibmcAPIKey string,
) error {
	if provider == ProviderGCP && gcpJSON != "" {
		if err := validateGCPServiceAccountJSON(gcpJSON); err != nil {
			return fmt.Errorf("invalid gcpServiceAccountJson: %w", err)
		}
	}
	return nil
}

func validateGCPServiceAccountJSON(encoded string) error {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("must be valid base64: %w", err)
	}

	var sa map[string]interface{}
	if err := json.Unmarshal(decoded, &sa); err != nil {
		return fmt.Errorf("must be valid JSON: %w", err)
	}

	if _, ok := sa["type"]; !ok {
		return fmt.Errorf("missing required field 'type'")
	}
	if _, ok := sa["project_id"]; !ok {
		return fmt.Errorf("missing required field 'project_id'")
	}

	return nil
}
