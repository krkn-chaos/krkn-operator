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

// Secret data key constants for each provider
const (
	// AWS
	SecretKeyAWSAccessKeyID     = "aws-access-key-id"
	SecretKeyAWSSecretAccessKey = "aws-secret-access-key"
	SecretKeyAWSDefaultRegion   = "aws-default-region"

	// GCP
	SecretKeyGCPServiceAccountJSON = "gcp-service-account-json"

	// Azure
	SecretKeyAzureTenantID       = "azure-tenant-id"
	SecretKeyAzureClientID       = "azure-client-id"
	SecretKeyAzureClientSecret   = "azure-client-secret"
	SecretKeyAzureSubscriptionID = "azure-subscription-id"

	// OpenStack
	SecretKeyOSAuthURL     = "os-auth-url"
	SecretKeyOSUsername    = "os-username"
	SecretKeyOSPassword    = "os-password"
	SecretKeyOSProjectName = "os-project-name"
	SecretKeyOSDomainName  = "os-domain-name"

	// Baremetal (IPMI/BMC)
	SecretKeyBMCUser     = "bmc-user"
	SecretKeyBMCPassword = "bmc-password"
	SecretKeyBMCAddr     = "bmc-addr"

	// VMware vSphere
	SecretKeyVSphereIP       = "vsphere-ip"
	SecretKeyVSphereUsername = "vsphere-username"
	SecretKeyVSpherePassword = "vsphere-password"

	// IBM Cloud
	SecretKeyIBMCURL    = "ibmc-url"
	SecretKeyIBMCAPIKey = "ibmc-apikey"
)

// CreateCloudCredentialRequest represents the request to create a cloud credential config
type CreateCloudCredentialRequest struct {
	Name           string   `json:"name"`
	Provider       string   `json:"provider"`
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
