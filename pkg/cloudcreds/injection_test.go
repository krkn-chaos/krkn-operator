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
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeSecret(name, provider string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				ProviderTypeLabel: provider,
			},
		},
		Data: data,
	}
}

func TestInjectCredentialsAWS(t *testing.T) {
	secret := makeSecret("aws-prod", ProviderAWS, map[string][]byte{
		SecretKeyAWSAccessKeyID:     []byte("AKIATEST"),
		SecretKeyAWSSecretAccessKey: []byte("secret"),
		SecretKeyAWSDefaultRegion:   []byte("us-east-1"),
	})

	envVars, volumes, mounts, err := InjectCredentials(secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(volumes) != 0 {
		t.Errorf("expected 0 volumes, got %d", len(volumes))
	}
	if len(mounts) != 0 {
		t.Errorf("expected 0 mounts, got %d", len(mounts))
	}
	if len(envVars) != 3 {
		t.Fatalf("expected 3 env vars, got %d", len(envVars))
	}

	expectedEnvs := map[string]string{
		"AWS_ACCESS_KEY_ID":     SecretKeyAWSAccessKeyID,
		"AWS_SECRET_ACCESS_KEY": SecretKeyAWSSecretAccessKey,
		"AWS_DEFAULT_REGION":    SecretKeyAWSDefaultRegion,
	}

	for _, env := range envVars {
		expectedKey, ok := expectedEnvs[env.Name]
		if !ok {
			t.Errorf("unexpected env var: %s", env.Name)
			continue
		}
		if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
			t.Errorf("env var %s should use SecretKeyRef", env.Name)
			continue
		}
		if env.ValueFrom.SecretKeyRef.Name != "aws-prod" {
			t.Errorf("env var %s secret name = %q, want %q", env.Name, env.ValueFrom.SecretKeyRef.Name, "aws-prod")
		}
		if env.ValueFrom.SecretKeyRef.Key != expectedKey {
			t.Errorf("env var %s secret key = %q, want %q", env.Name, env.ValueFrom.SecretKeyRef.Key, expectedKey)
		}
	}
}

func TestInjectCredentialsGCP(t *testing.T) {
	secret := makeSecret("gcp-prod", ProviderGCP, map[string][]byte{
		SecretKeyGCPServiceAccountJSON: []byte(`{"type":"service_account"}`),
	})

	envVars, volumes, mounts, err := InjectCredentials(secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(envVars) != 1 {
		t.Fatalf("expected 1 env var, got %d", len(envVars))
	}
	if envVars[0].Name != "GOOGLE_APPLICATION_CREDENTIALS" {
		t.Errorf("env var name = %q, want GOOGLE_APPLICATION_CREDENTIALS", envVars[0].Name)
	}
	if envVars[0].Value != gcpMountPath {
		t.Errorf("env var value = %q, want %q", envVars[0].Value, gcpMountPath)
	}
	if envVars[0].ValueFrom != nil {
		t.Error("GOOGLE_APPLICATION_CREDENTIALS should use Value, not ValueFrom")
	}

	if len(volumes) != 1 {
		t.Fatalf("expected 1 volume, got %d", len(volumes))
	}
	if volumes[0].Name != gcpVolumeName {
		t.Errorf("volume name = %q, want %q", volumes[0].Name, gcpVolumeName)
	}
	if volumes[0].VolumeSource.Secret == nil {
		t.Fatal("expected Secret volume source")
	}
	if volumes[0].VolumeSource.Secret.SecretName != "gcp-prod" {
		t.Errorf("volume secret name = %q, want %q", volumes[0].VolumeSource.Secret.SecretName, "gcp-prod")
	}

	if len(mounts) != 1 {
		t.Fatalf("expected 1 mount, got %d", len(mounts))
	}
	if mounts[0].Name != gcpVolumeName {
		t.Errorf("mount name = %q, want %q", mounts[0].Name, gcpVolumeName)
	}
	if mounts[0].MountPath != "/home/krkn/.gcp" {
		t.Errorf("mount path = %q, want %q", mounts[0].MountPath, "/home/krkn/.gcp")
	}
	if !mounts[0].ReadOnly {
		t.Error("expected mount to be read-only")
	}
}

func TestInjectCredentialsAzure(t *testing.T) {
	secret := makeSecret("azure-prod", ProviderAzure, map[string][]byte{
		SecretKeyAzureTenantID:       []byte("tenant"),
		SecretKeyAzureClientID:       []byte("client"),
		SecretKeyAzureClientSecret:   []byte("secret"),
		SecretKeyAzureSubscriptionID: []byte("sub"),
	})

	envVars, volumes, mounts, err := InjectCredentials(secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(volumes) != 0 || len(mounts) != 0 {
		t.Error("expected no volumes or mounts for Azure")
	}
	if len(envVars) != 4 {
		t.Fatalf("expected 4 env vars, got %d", len(envVars))
	}

	for _, env := range envVars {
		if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
			t.Errorf("env var %s should use SecretKeyRef", env.Name)
		}
		if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name != "azure-prod" {
			t.Errorf("env var %s secret name = %q, want %q", env.Name, env.ValueFrom.SecretKeyRef.Name, "azure-prod")
		}
	}
}

func TestInjectCredentialsOpenStack(t *testing.T) {
	secret := makeSecret("os-prod", ProviderOpenStack, map[string][]byte{
		SecretKeyOSAuthURL:     []byte("http://auth:5000"),
		SecretKeyOSUsername:    []byte("admin"),
		SecretKeyOSPassword:    []byte("pass"),
		SecretKeyOSProjectName: []byte("project"),
		SecretKeyOSDomainName:  []byte("Default"),
	})

	envVars, volumes, mounts, err := InjectCredentials(secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(volumes) != 0 || len(mounts) != 0 {
		t.Error("expected no volumes or mounts for OpenStack")
	}
	if len(envVars) != 5 {
		t.Fatalf("expected 5 env vars, got %d", len(envVars))
	}

	for _, env := range envVars {
		if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
			t.Errorf("env var %s should use SecretKeyRef", env.Name)
		}
	}
}

func TestInjectCredentialsBaremetal(t *testing.T) {
	secret := makeSecret("bm-prod", ProviderBaremetal, map[string][]byte{
		SecretKeyBMCUser:     []byte("admin"),
		SecretKeyBMCPassword: []byte("secret"),
		SecretKeyBMCAddr:     []byte("192.168.1.100"),
	})

	envVars, volumes, mounts, err := InjectCredentials(secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(volumes) != 0 || len(mounts) != 0 {
		t.Error("expected no volumes or mounts for Baremetal")
	}
	if len(envVars) != 4 {
		t.Fatalf("expected 4 env vars, got %d", len(envVars))
	}

	// CLOUD_TYPE should be a plain value, not SecretKeyRef
	cloudTypeFound := false
	for _, env := range envVars {
		if env.Name == "CLOUD_TYPE" {
			cloudTypeFound = true
			if env.Value != "bm" {
				t.Errorf("CLOUD_TYPE value = %q, want %q", env.Value, "bm")
			}
			if env.ValueFrom != nil {
				t.Error("CLOUD_TYPE should use Value, not ValueFrom")
			}
		}
	}
	if !cloudTypeFound {
		t.Error("expected CLOUD_TYPE env var")
	}

	// BMC_USER, BMC_PASSWORD, BMC_ADDR should use SecretKeyRef
	for _, env := range envVars {
		if env.Name == "CLOUD_TYPE" {
			continue
		}
		if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
			t.Errorf("env var %s should use SecretKeyRef", env.Name)
		}
		if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name != "bm-prod" {
			t.Errorf("env var %s secret name = %q, want %q", env.Name, env.ValueFrom.SecretKeyRef.Name, "bm-prod")
		}
	}
}

func TestInjectCredentialsVMware(t *testing.T) {
	secret := makeSecret("vmware-prod", ProviderVMware, map[string][]byte{
		SecretKeyVSphereIP:       []byte("10.0.0.1"),
		SecretKeyVSphereUsername: []byte("admin@vsphere.local"),
		SecretKeyVSpherePassword: []byte("secret"),
	})

	envVars, volumes, mounts, err := InjectCredentials(secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(volumes) != 0 || len(mounts) != 0 {
		t.Error("expected no volumes or mounts for VMware")
	}
	if len(envVars) != 4 {
		t.Fatalf("expected 4 env vars, got %d", len(envVars))
	}

	cloudTypeFound := false
	for _, env := range envVars {
		if env.Name == "CLOUD_TYPE" {
			cloudTypeFound = true
			if env.Value != "vmware" {
				t.Errorf("CLOUD_TYPE value = %q, want %q", env.Value, "vmware")
			}
		}
	}
	if !cloudTypeFound {
		t.Error("expected CLOUD_TYPE env var")
	}

	for _, env := range envVars {
		if env.Name == "CLOUD_TYPE" {
			continue
		}
		if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
			t.Errorf("env var %s should use SecretKeyRef", env.Name)
		}
	}
}

func TestInjectCredentialsIBMCloud(t *testing.T) {
	secret := makeSecret("ibm-prod", ProviderIBMCloud, map[string][]byte{
		SecretKeyIBMCURL:    []byte("https://us-south.iaas.cloud.ibm.com/v1"),
		SecretKeyIBMCAPIKey: []byte("my-api-key"),
	})

	envVars, volumes, mounts, err := InjectCredentials(secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(volumes) != 0 || len(mounts) != 0 {
		t.Error("expected no volumes or mounts for IBM Cloud")
	}
	if len(envVars) != 3 {
		t.Fatalf("expected 3 env vars, got %d", len(envVars))
	}

	cloudTypeFound := false
	for _, env := range envVars {
		if env.Name == "CLOUD_TYPE" {
			cloudTypeFound = true
			if env.Value != "ibmcloud" {
				t.Errorf("CLOUD_TYPE value = %q, want %q", env.Value, "ibmcloud")
			}
		}
	}
	if !cloudTypeFound {
		t.Error("expected CLOUD_TYPE env var")
	}

	for _, env := range envVars {
		if env.Name == "CLOUD_TYPE" {
			continue
		}
		if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
			t.Errorf("env var %s should use SecretKeyRef", env.Name)
		}
	}
}

func TestInjectCredentialsMissingProviderLabel(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "bad",
			Labels: map[string]string{},
		},
	}

	_, _, _, err := InjectCredentials(secret)
	if err == nil {
		t.Fatal("expected error for missing provider label")
	}
	if !strings.Contains(err.Error(), "missing provider-type label") {
		t.Errorf("error = %q, want it to mention missing provider label", err.Error())
	}
}

func TestInjectCredentialsUnsupportedProvider(t *testing.T) {
	secret := makeSecret("bad", "oracle", nil)

	_, _, _, err := InjectCredentials(secret)
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
	if !strings.Contains(err.Error(), "unsupported cloud provider") {
		t.Errorf("error = %q, want it to mention unsupported provider", err.Error())
	}
}
