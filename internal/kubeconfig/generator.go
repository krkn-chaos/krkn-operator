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
	"encoding/json"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// GenerateFromToken creates a kubeconfig with token authentication
// Returns base64-encoded kubeconfig string
func GenerateFromToken(clusterName, apiURL, caBundle, token string, insecureSkipTLS bool) (string, error) {
	config := clientcmdapi.NewConfig()

	// Add cluster
	cluster := clientcmdapi.NewCluster()
	cluster.Server = apiURL
	cluster.InsecureSkipTLSVerify = insecureSkipTLS

	if caBundle != "" && !insecureSkipTLS {
		cluster.CertificateAuthorityData = []byte(caBundle)
	}

	config.Clusters[clusterName] = cluster

	// Add user with token
	authInfo := clientcmdapi.NewAuthInfo()
	authInfo.Token = token
	config.AuthInfos[clusterName+"-user"] = authInfo

	// Add context
	context := clientcmdapi.NewContext()
	context.Cluster = clusterName
	context.AuthInfo = clusterName + "-user"
	config.Contexts[clusterName+"-context"] = context

	// Set current context
	config.CurrentContext = clusterName + "-context"

	// Convert to YAML bytes
	kubeconfigBytes, err := clientcmd.Write(*config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal kubeconfig: %w", err)
	}

	// Return base64-encoded string
	return base64.StdEncoding.EncodeToString(kubeconfigBytes), nil
}

// GenerateFromCredentials creates a kubeconfig with basic auth (username/password)
// Returns base64-encoded kubeconfig string
func GenerateFromCredentials(clusterName, apiURL, caBundle, username, password string, insecureSkipTLS bool) (string, error) {
	config := clientcmdapi.NewConfig()

	// Add cluster
	cluster := clientcmdapi.NewCluster()
	cluster.Server = apiURL
	cluster.InsecureSkipTLSVerify = insecureSkipTLS

	if caBundle != "" && !insecureSkipTLS {
		cluster.CertificateAuthorityData = []byte(caBundle)
	}

	config.Clusters[clusterName] = cluster

	// Add user with credentials
	authInfo := clientcmdapi.NewAuthInfo()
	authInfo.Username = username
	authInfo.Password = password
	config.AuthInfos[clusterName+"-user"] = authInfo

	// Add context
	context := clientcmdapi.NewContext()
	context.Cluster = clusterName
	context.AuthInfo = clusterName + "-user"
	config.Contexts[clusterName+"-context"] = context

	// Set current context
	config.CurrentContext = clusterName + "-context"

	// Convert to YAML bytes
	kubeconfigBytes, err := clientcmd.Write(*config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal kubeconfig: %w", err)
	}

	// Return base64-encoded string
	return base64.StdEncoding.EncodeToString(kubeconfigBytes), nil
}

// ExtractAPIURL extracts the server URL from a base64-encoded kubeconfig
func ExtractAPIURL(kubeconfigBase64 string) (string, error) {
	// Decode base64
	kubeconfigBytes, err := base64.StdEncoding.DecodeString(kubeconfigBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode kubeconfig: %w", err)
	}

	// Load kubeconfig
	config, err := clientcmd.Load(kubeconfigBytes)
	if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Get current context
	if config.CurrentContext == "" {
		return "", fmt.Errorf("no current context set in kubeconfig")
	}

	context, exists := config.Contexts[config.CurrentContext]
	if !exists {
		return "", fmt.Errorf("current context '%s' not found in kubeconfig", config.CurrentContext)
	}

	// Get cluster from context
	cluster, exists := config.Clusters[context.Cluster]
	if !exists {
		return "", fmt.Errorf("cluster '%s' not found in kubeconfig", context.Cluster)
	}

	if cluster.Server == "" {
		return "", fmt.Errorf("cluster server URL is empty")
	}

	return cluster.Server, nil
}

// Validate checks if a base64-encoded kubeconfig is valid
func Validate(kubeconfigBase64 string) error {
	// Decode base64
	kubeconfigBytes, err := base64.StdEncoding.DecodeString(kubeconfigBase64)
	if err != nil {
		return fmt.Errorf("invalid base64 encoding: %w", err)
	}

	// Try to load kubeconfig
	config, err := clientcmd.Load(kubeconfigBytes)
	if err != nil {
		return fmt.Errorf("invalid kubeconfig format: %w", err)
	}

	// Validate required fields
	if len(config.Clusters) == 0 {
		return fmt.Errorf("kubeconfig must contain at least one cluster")
	}

	if len(config.AuthInfos) == 0 {
		return fmt.Errorf("kubeconfig must contain at least one user")
	}

	if len(config.Contexts) == 0 {
		return fmt.Errorf("kubeconfig must contain at least one context")
	}

	if config.CurrentContext == "" {
		return fmt.Errorf("kubeconfig must have a current context set")
	}

	// Validate current context exists
	if _, exists := config.Contexts[config.CurrentContext]; !exists {
		return fmt.Errorf("current context '%s' does not exist", config.CurrentContext)
	}

	return nil
}

// SecretData represents the JSON structure stored in the Secret
type SecretData struct {
	Kubeconfig string `json:"kubeconfig"`
}

// MarshalSecretData creates the JSON data to be stored in the Secret
func MarshalSecretData(kubeconfigBase64 string) ([]byte, error) {
	data := SecretData{
		Kubeconfig: kubeconfigBase64,
	}
	return json.Marshal(data)
}

// UnmarshalSecretData extracts the kubeconfig from Secret JSON data
func UnmarshalSecretData(secretBytes []byte) (string, error) {
	var data SecretData
	if err := json.Unmarshal(secretBytes, &data); err != nil {
		return "", fmt.Errorf("failed to unmarshal secret data: %w", err)
	}

	if data.Kubeconfig == "" {
		return "", fmt.Errorf("kubeconfig field is empty in secret data")
	}

	return data.Kubeconfig, nil
}
