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
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

const (
	gcpMountPath  = "/home/krkn/.gcp/service-account.json"
	gcpVolumeName = "cloud-cred-gcp"
)

// InjectCredentials returns the env vars, volumes, and volume mounts needed to
// inject cloud provider credentials into a scenario pod. All env vars use
// SecretKeyRef to avoid exposing credentials as plaintext in the pod spec.
func InjectCredentials(secret *corev1.Secret) ([]corev1.EnvVar, []corev1.Volume, []corev1.VolumeMount, error) {
	provider := secret.Labels[ProviderTypeLabel]
	if provider == "" {
		return nil, nil, nil, fmt.Errorf("secret %s missing provider-type label", secret.Name)
	}

	switch provider {
	case ProviderAWS:
		return injectAWS(secret.Name), nil, nil, nil
	case ProviderGCP:
		envVars, volumes, mounts := injectGCP(secret.Name)
		return envVars, volumes, mounts, nil
	case ProviderAzure:
		return injectAzure(secret.Name), nil, nil, nil
	case ProviderOpenStack:
		return injectOpenStack(secret.Name), nil, nil, nil
	case ProviderBaremetal:
		return injectBaremetal(secret.Name), nil, nil, nil
	case ProviderVMware:
		return injectVMware(secret.Name), nil, nil, nil
	case ProviderIBMCloud:
		return injectIBMCloud(secret.Name), nil, nil, nil
	default:
		return nil, nil, nil, fmt.Errorf("unsupported cloud provider: %s", provider)
	}
}

func secretKeyRef(secretName, key string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
			Key:                  key,
		},
	}
}

func injectAWS(secretName string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "CLOUD_TYPE", Value: "aws"},
		{Name: "AWS_ACCESS_KEY_ID", ValueFrom: secretKeyRef(secretName, SecretKeyAWSAccessKeyID)},
		{Name: "AWS_SECRET_ACCESS_KEY", ValueFrom: secretKeyRef(secretName, SecretKeyAWSSecretAccessKey)},
		{Name: "AWS_DEFAULT_REGION", ValueFrom: secretKeyRef(secretName, SecretKeyAWSDefaultRegion)},
	}
}

func injectGCP(secretName string) ([]corev1.EnvVar, []corev1.Volume, []corev1.VolumeMount) {
	envVars := []corev1.EnvVar{
		{Name: "CLOUD_TYPE", Value: "gcp"},
		{Name: "GOOGLE_APPLICATION_CREDENTIALS", Value: gcpMountPath},
	}

	volumes := []corev1.Volume{
		{
			Name: gcpVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
					Items: []corev1.KeyToPath{
						{Key: SecretKeyGCPServiceAccountJSON, Path: "service-account.json"},
					},
				},
			},
		},
	}

	mounts := []corev1.VolumeMount{
		{
			Name:      gcpVolumeName,
			MountPath: "/home/krkn/.gcp",
			ReadOnly:  true,
		},
	}

	return envVars, volumes, mounts
}

func injectAzure(secretName string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "CLOUD_TYPE", Value: "azure"},
		{Name: "AZURE_TENANT_ID", ValueFrom: secretKeyRef(secretName, SecretKeyAzureTenantID)},
		{Name: "AZURE_CLIENT_ID", ValueFrom: secretKeyRef(secretName, SecretKeyAzureClientID)},
		{Name: "AZURE_CLIENT_SECRET", ValueFrom: secretKeyRef(secretName, SecretKeyAzureClientSecret)},
		{Name: "AZURE_SUBSCRIPTION_ID", ValueFrom: secretKeyRef(secretName, SecretKeyAzureSubscriptionID)},
	}
}

func injectOpenStack(secretName string) []corev1.EnvVar {
	optional := true
	return []corev1.EnvVar{
		{Name: "CLOUD_TYPE", Value: "openstack"},
		{Name: "OS_AUTH_URL", ValueFrom: secretKeyRef(secretName, SecretKeyOSAuthURL)},
		{Name: "OS_USERNAME", ValueFrom: secretKeyRef(secretName, SecretKeyOSUsername)},
		{Name: "OS_PASSWORD", ValueFrom: secretKeyRef(secretName, SecretKeyOSPassword)},
		{Name: "OS_PROJECT_NAME", ValueFrom: secretKeyRef(secretName, SecretKeyOSProjectName)},
		{Name: "OS_DOMAIN_NAME", ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
				Key:                  SecretKeyOSDomainName,
				Optional:             &optional,
			},
		}},
	}
}

func injectBaremetal(secretName string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "CLOUD_TYPE", Value: "bm"},
		{Name: "BMC_USER", ValueFrom: secretKeyRef(secretName, SecretKeyBMCUser)},
		{Name: "BMC_PASSWORD", ValueFrom: secretKeyRef(secretName, SecretKeyBMCPassword)},
		{Name: "BMC_ADDR", ValueFrom: secretKeyRef(secretName, SecretKeyBMCAddr)},
	}
}

func injectVMware(secretName string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "CLOUD_TYPE", Value: "vmware"},
		{Name: "VSPHERE_IP", ValueFrom: secretKeyRef(secretName, SecretKeyVSphereIP)},
		{Name: "VSPHERE_USERNAME", ValueFrom: secretKeyRef(secretName, SecretKeyVSphereUsername)},
		{Name: "VSPHERE_PASSWORD", ValueFrom: secretKeyRef(secretName, SecretKeyVSpherePassword)},
	}
}

func injectIBMCloud(secretName string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "CLOUD_TYPE", Value: "ibmcloud"},
		{Name: "IBMC_URL", ValueFrom: secretKeyRef(secretName, SecretKeyIBMCURL)},
		{Name: "IBMC_APIKEY", ValueFrom: secretKeyRef(secretName, SecretKeyIBMCAPIKey)},
	}
}
