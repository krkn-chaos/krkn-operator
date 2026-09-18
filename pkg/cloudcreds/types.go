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

// Package cloudcreds provides functionality for managing cloud provider
// credentials in the krkn-operator ecosystem. Credentials are stored as
// Kubernetes Secrets with labeled metadata and injected into scenario pods
// via SecretKeyRef env vars and SecretVolumeSource file mounts.
package cloudcreds

// Cloud provider constants
const (
	ProviderAWS       = "aws"
	ProviderGCP       = "gcp"
	ProviderAzure     = "azure"
	ProviderOpenStack = "openstack"
	ProviderBaremetal = "baremetal"
	ProviderVMware    = "vmware"
	ProviderIBMCloud  = "ibmcloud"
)

// Secret data key constants for each provider (Kubernetes Secret .Data keys, not credential values).
const (
	// AWS
	SecretKeyAWSAccessKeyID     = "aws-access-key-id"     // #nosec G101 -- Secret data key name, not a credential value
	SecretKeyAWSSecretAccessKey = "aws-secret-access-key" // #nosec G101 -- Secret data key name, not a credential value
	SecretKeyAWSDefaultRegion   = "aws-default-region"    // #nosec G101 -- Secret data key name, not a credential value

	// GCP
	SecretKeyGCPServiceAccountJSON = "gcp-service-account-json" // #nosec G101 -- Secret data key name, not a credential value

	// Azure
	SecretKeyAzureTenantID       = "azure-tenant-id"       // #nosec G101 -- Secret data key name, not a credential value
	SecretKeyAzureClientID       = "azure-client-id"       // #nosec G101 -- Secret data key name, not a credential value
	SecretKeyAzureClientSecret   = "azure-client-secret"   // #nosec G101 -- Secret data key name, not a credential value
	SecretKeyAzureSubscriptionID = "azure-subscription-id" // #nosec G101 -- Secret data key name, not a credential value

	// OpenStack
	SecretKeyOSAuthURL     = "os-auth-url"     // #nosec G101 -- Secret data key name, not a credential value
	SecretKeyOSUsername    = "os-username"     // #nosec G101 -- Secret data key name, not a credential value
	SecretKeyOSPassword    = "os-password"     // #nosec G101 -- Secret data key name, not a credential value
	SecretKeyOSProjectName = "os-project-name" // #nosec G101 -- Secret data key name, not a credential value
	SecretKeyOSDomainName  = "os-domain-name"  // #nosec G101 -- Secret data key name, not a credential value

	// Baremetal (IPMI/BMC)
	SecretKeyBMCUser     = "bmc-user"     // #nosec G101 -- Secret data key name, not a credential value
	SecretKeyBMCPassword = "bmc-password" // #nosec G101 -- Secret data key name, not a credential value
	SecretKeyBMCAddr     = "bmc-addr"     // #nosec G101 -- Secret data key name, not a credential value

	// VMware vSphere
	SecretKeyVSphereIP       = "vsphere-ip"       // #nosec G101 -- Secret data key name, not a credential value
	SecretKeyVSphereUsername = "vsphere-username" // #nosec G101 -- Secret data key name, not a credential value
	SecretKeyVSpherePassword = "vsphere-password" // #nosec G101 -- Secret data key name, not a credential value

	// IBM Cloud
	SecretKeyIBMCURL    = "ibmc-url"    // #nosec G101 -- Secret data key name, not a credential value
	SecretKeyIBMCAPIKey = "ibmc-apikey" // #nosec G101 -- Secret data key name, not a credential value
)

// CreateCloudCredentialRequest represents the request to create a cloud credential config
type CreateCloudCredentialRequest struct {
	// Name is the unique Secret name for the credential.
	Name string `json:"name"`
	// Provider identifies the cloud provider for this credential.
	Provider string `json:"provider"`
	// Description is optional human-readable context for the credential.
	Description string `json:"description,omitempty"`
	// Groups controls which user groups may access the credential.
	Groups []string `json:"groups,omitempty"`
	// AvailableToAll grants access to all authenticated users.
	AvailableToAll bool `json:"availableToAll,omitempty"`

	// AWS fields
	AWSAccessKeyID     string `json:"awsAccessKeyId,omitempty"`
	AWSSecretAccessKey string `json:"awsSecretAccessKey,omitempty"`
	AWSDefaultRegion   string `json:"awsDefaultRegion,omitempty"`

	// GCP fields
	GCPServiceAccountJSON string `json:"gcpServiceAccountJson,omitempty"`

	// Azure fields
	AzureTenantID       string `json:"azureTenantId,omitempty"`
	AzureClientID       string `json:"azureClientId,omitempty"`
	AzureClientSecret   string `json:"azureClientSecret,omitempty"`
	AzureSubscriptionID string `json:"azureSubscriptionId,omitempty"`

	// OpenStack fields
	OSAuthURL     string `json:"osAuthUrl,omitempty"`
	OSUsername    string `json:"osUsername,omitempty"`
	OSPassword    string `json:"osPassword,omitempty"`
	OSProjectName string `json:"osProjectName,omitempty"`
	OSDomainName  string `json:"osDomainName,omitempty"`

	// Baremetal (IPMI/BMC) fields
	BMCUser     string `json:"bmcUser,omitempty"`
	BMCPassword string `json:"bmcPassword,omitempty"`
	BMCAddr     string `json:"bmcAddr,omitempty"`

	// VMware vSphere fields
	VSphereIP       string `json:"vsphereIp,omitempty"`
	VSphereUsername string `json:"vsphereUsername,omitempty"`
	VSpherePassword string `json:"vspherePassword,omitempty"`

	// IBM Cloud fields
	IBMCURL    string `json:"ibmcUrl,omitempty"`
	IBMCAPIKey string `json:"ibmcApikey,omitempty"`
}

// UpdateCloudCredentialRequest represents the request to update a cloud credential config.
// Provider is immutable and cannot be changed after creation.
type UpdateCloudCredentialRequest struct {
	Description    string   `json:"description,omitempty"`
	Groups         []string `json:"groups,omitempty"`
	AvailableToAll bool     `json:"availableToAll,omitempty"`

	// AWS fields
	AWSAccessKeyID     string `json:"awsAccessKeyId,omitempty"`
	AWSSecretAccessKey string `json:"awsSecretAccessKey,omitempty"`
	AWSDefaultRegion   string `json:"awsDefaultRegion,omitempty"`

	// GCP fields
	GCPServiceAccountJSON string `json:"gcpServiceAccountJson,omitempty"`

	// Azure fields
	AzureTenantID       string `json:"azureTenantId,omitempty"`
	AzureClientID       string `json:"azureClientId,omitempty"`
	AzureClientSecret   string `json:"azureClientSecret,omitempty"`
	AzureSubscriptionID string `json:"azureSubscriptionId,omitempty"`

	// OpenStack fields
	OSAuthURL     string `json:"osAuthUrl,omitempty"`
	OSUsername    string `json:"osUsername,omitempty"`
	OSPassword    string `json:"osPassword,omitempty"`
	OSProjectName string `json:"osProjectName,omitempty"`
	OSDomainName  string `json:"osDomainName,omitempty"`

	// Baremetal (IPMI/BMC) fields
	BMCUser     string `json:"bmcUser,omitempty"`
	BMCPassword string `json:"bmcPassword,omitempty"`
	BMCAddr     string `json:"bmcAddr,omitempty"`

	// VMware vSphere fields
	VSphereIP       string `json:"vsphereIp,omitempty"`
	VSphereUsername string `json:"vsphereUsername,omitempty"`
	VSpherePassword string `json:"vspherePassword,omitempty"`

	// IBM Cloud fields
	IBMCURL    string `json:"ibmcUrl,omitempty"`
	IBMCAPIKey string `json:"ibmcApikey,omitempty"`
}

// CloudCredentialResponse represents a cloud credential in API responses.
// Secret data is intentionally never included.
type CloudCredentialResponse struct {
	Name           string   `json:"name"`
	Provider       string   `json:"provider"`
	Description    string   `json:"description,omitempty"`
	Groups         []string `json:"groups,omitempty"`
	AvailableToAll bool     `json:"availableToAll,omitempty"`
	CreatedAt      string   `json:"createdAt,omitempty"`
	CreatedBy      string   `json:"createdBy,omitempty"`
	UpdatedAt      string   `json:"updatedAt,omitempty"`
	UpdatedBy      string   `json:"updatedBy,omitempty"`
}

// ListCloudCredentialsResponse represents the response for listing cloud credentials
type ListCloudCredentialsResponse struct {
	Credentials []CloudCredentialResponse `json:"credentials"`
	Total       int                       `json:"total"`
}

// CreateCloudCredentialResponse represents the response after creating a cloud credential
type CreateCloudCredentialResponse struct {
	Message string `json:"message"`
	Name    string `json:"name"`
}

// UpdateCloudCredentialResponse represents the response after updating a cloud credential
type UpdateCloudCredentialResponse struct {
	Message string `json:"message"`
	Name    string `json:"name"`
}

// DeleteCloudCredentialResponse represents the response after deleting a cloud credential
type DeleteCloudCredentialResponse struct {
	Message string `json:"message"`
}

// ValidProviders is the set of supported cloud provider types
var ValidProviders = map[string]bool{
	ProviderAWS:       true,
	ProviderGCP:       true,
	ProviderAzure:     true,
	ProviderOpenStack: true,
	ProviderBaremetal: true,
	ProviderVMware:    true,
	ProviderIBMCloud:  true,
}
