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

package registry

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExtractRegistryV2FromSecret_Token(t *testing.T) {
	// Create a valid dockerconfigjson with token auth (username:token format)
	authStr := base64.StdEncoding.EncodeToString([]byte("$oauthtoken:my-secret-token"))
	dockerConfig := DockerConfig{
		Auths: map[string]AuthEntry{
			"registry.example.com": {
				Auth: authStr,
			},
		},
	}
	dockerConfigJSON, _ := json.Marshal(dockerConfig)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-registry",
			Namespace: "default",
			Labels: map[string]string{
				AppNameLabel:      AppName,
				AppComponentLabel: ComponentRegistry,
				AuthTypeLabel:     AuthTypeToken,
			},
			Annotations: map[string]string{
				RegistryURLAnnotation:        "registry.example.com",
				ScenarioRepositoryAnnotation: "krkn-chaos/krkn-hub",
				SkipTLSAnnotation:            "true",
				InsecureAnnotation:           "false",
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfigJSON,
		},
	}

	registryV2, err := ExtractRegistryV2FromSecret(secret)
	if err != nil {
		t.Fatalf("ExtractRegistryV2FromSecret() error = %v", err)
	}

	if registryV2.RegistryURL != "registry.example.com" {
		t.Errorf("RegistryURL = %v, want registry.example.com", registryV2.RegistryURL)
	}
	if registryV2.ScenarioRepository != "krkn-chaos/krkn-hub" {
		t.Errorf("ScenarioRepository = %v, want krkn-chaos/krkn-hub", registryV2.ScenarioRepository)
	}
	if registryV2.SkipTLS != true {
		t.Errorf("SkipTLS = %v, want true", registryV2.SkipTLS)
	}
	if registryV2.Insecure != false {
		t.Errorf("Insecure = %v, want false", registryV2.Insecure)
	}
	if registryV2.Username == nil || *registryV2.Username != "$oauthtoken" {
		t.Errorf("Username = %v, want $oauthtoken", registryV2.Username)
	}
	if registryV2.Password == nil || *registryV2.Password != "my-secret-token" {
		t.Errorf("Password = %v, want my-secret-token", registryV2.Password)
	}
	// For token auth, Token field should be populated with password value
	if registryV2.Token == nil || *registryV2.Token != "my-secret-token" {
		t.Errorf("Token = %v, want my-secret-token", registryV2.Token)
	}
}

func TestExtractRegistryV2FromSecret_Password(t *testing.T) {
	// Create a valid dockerconfigjson for password auth
	authStr := base64.StdEncoding.EncodeToString([]byte("myuser:mypassword"))
	dockerConfig := DockerConfig{
		Auths: map[string]AuthEntry{
			"registry.io": {
				Auth: authStr,
			},
		},
	}
	dockerConfigJSON, _ := json.Marshal(dockerConfig)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-registry",
			Namespace: "default",
			Labels: map[string]string{
				AppNameLabel:      AppName,
				AppComponentLabel: ComponentRegistry,
				AuthTypeLabel:     AuthTypePassword,
			},
			Annotations: map[string]string{
				RegistryURLAnnotation:        "registry.io",
				ScenarioRepositoryAnnotation: "org/repo",
				SkipTLSAnnotation:            "false",
				InsecureAnnotation:           "true",
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfigJSON,
		},
	}

	registryV2, err := ExtractRegistryV2FromSecret(secret)
	if err != nil {
		t.Fatalf("ExtractRegistryV2FromSecret() error = %v", err)
	}

	if registryV2.RegistryURL != "registry.io" {
		t.Errorf("RegistryURL = %v, want registry.io", registryV2.RegistryURL)
	}
	if registryV2.Username == nil || *registryV2.Username != "myuser" {
		t.Errorf("Username = %v, want myuser", registryV2.Username)
	}
	if registryV2.Password == nil || *registryV2.Password != "mypassword" {
		t.Errorf("Password = %v, want mypassword", registryV2.Password)
	}
	// For password auth, Token field should NOT be populated
	if registryV2.Token != nil {
		t.Errorf("Token should be nil for password auth, got %v", *registryV2.Token)
	}
}

func TestExtractRegistryV2FromSecret_InvalidType(t *testing.T) {
	secret := &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
	}

	_, err := ExtractRegistryV2FromSecret(secret)
	if err == nil {
		t.Error("Expected error for invalid secret type")
	}
	if !strings.Contains(err.Error(), "invalid secret type") {
		t.Errorf("Error message should mention invalid secret type, got: %v", err)
	}
}

func TestExtractRegistryV2FromSecret_MissingComponent(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				AppNameLabel: AppName,
				// Missing AppComponentLabel
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}

	_, err := ExtractRegistryV2FromSecret(secret)
	if err == nil {
		t.Error("Expected error for missing component label")
	}
	if !strings.Contains(err.Error(), "not a registry secret") {
		t.Errorf("Error should mention not a registry secret, got: %v", err)
	}
}

func TestExtractRegistryV2FromSecret_MissingAnnotations(t *testing.T) {
	dockerConfig := DockerConfig{
		Auths: map[string]AuthEntry{
			"registry.io": {Auth: "dGVzdA=="},
		},
	}
	dockerConfigJSON, _ := json.Marshal(dockerConfig)

	tests := []struct {
		name        string
		annotations map[string]string
		wantErr     string
	}{
		{
			name: "missing registry URL",
			annotations: map[string]string{
				ScenarioRepositoryAnnotation: "org/repo",
			},
			wantErr: "missing required annotation",
		},
		{
			name: "missing scenario repository",
			annotations: map[string]string{
				RegistryURLAnnotation: "registry.io",
			},
			wantErr: "missing required annotation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						AppNameLabel:      AppName,
						AppComponentLabel: ComponentRegistry,
						AuthTypeLabel:     AuthTypeToken,
					},
					Annotations: tt.annotations,
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: dockerConfigJSON,
				},
			}

			_, err := ExtractRegistryV2FromSecret(secret)
			if err == nil {
				t.Error("Expected error for missing annotation")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Error should contain %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestBuildDockerConfigJSON_OAuth(t *testing.T) {
	dockerConfigJSON, err := BuildDockerConfigJSON(
		"registry.example.com",
		"$oauthtoken",
		"my-secret-token",
	)
	if err != nil {
		t.Fatalf("BuildDockerConfigJSON() error = %v", err)
	}

	var dockerConfig DockerConfig
	if err := json.Unmarshal(dockerConfigJSON, &dockerConfig); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	authEntry, exists := dockerConfig.Auths["registry.example.com"]
	if !exists {
		t.Error("Auth entry not found for registry.example.com")
	}

	decodedAuth, err := base64.StdEncoding.DecodeString(authEntry.Auth)
	if err != nil {
		t.Fatalf("Failed to decode auth: %v", err)
	}

	if string(decodedAuth) != "$oauthtoken:my-secret-token" {
		t.Errorf("Decoded auth = %v, want $oauthtoken:my-secret-token", string(decodedAuth))
	}
}

func TestBuildDockerConfigJSON_Password(t *testing.T) {
	dockerConfigJSON, err := BuildDockerConfigJSON(
		"registry.io",
		"myuser",
		"mypassword",
	)
	if err != nil {
		t.Fatalf("BuildDockerConfigJSON() error = %v", err)
	}

	var dockerConfig DockerConfig
	if err := json.Unmarshal(dockerConfigJSON, &dockerConfig); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	authEntry, exists := dockerConfig.Auths["registry.io"]
	if !exists {
		t.Error("Auth entry not found for registry.io")
	}

	decodedAuth, err := base64.StdEncoding.DecodeString(authEntry.Auth)
	if err != nil {
		t.Fatalf("Failed to decode auth: %v", err)
	}

	if string(decodedAuth) != "myuser:mypassword" {
		t.Errorf("Decoded auth = %v, want myuser:mypassword", string(decodedAuth))
	}
}

func TestBuildDockerConfigJSON_MissingCredentials(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		wantErr  string
	}{
		{
			name:     "missing username",
			username: "",
			password: "pass",
			wantErr:  "username and password are required",
		},
		{
			name:     "missing password",
			username: "user",
			password: "",
			wantErr:  "username and password are required",
		},
		{
			name:     "missing both",
			username: "",
			password: "",
			wantErr:  "username and password are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildDockerConfigJSON(
				"registry.io",
				tt.username,
				tt.password,
			)
			if err == nil {
				t.Error("Expected error for missing credentials")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Error should contain %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidateRegistrySecret(t *testing.T) {
	tests := []struct {
		name    string
		secret  *corev1.Secret
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid token secret",
			secret: func() *corev1.Secret {
				authStr := base64.StdEncoding.EncodeToString([]byte("$oauthtoken:token"))
				dockerConfig := DockerConfig{
					Auths: map[string]AuthEntry{
						"registry.io": {Auth: authStr},
					},
				}
				dockerConfigJSON, _ := json.Marshal(dockerConfig)

				return &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							AppNameLabel:      AppName,
							AppComponentLabel: ComponentRegistry,
							AuthTypeLabel:     AuthTypeToken,
						},
						Annotations: map[string]string{
							RegistryURLAnnotation:        "registry.io",
							ScenarioRepositoryAnnotation: "org/repo",
							SkipTLSAnnotation:            "false",
							InsecureAnnotation:           "false",
							CreatedByAnnotation:          "admin",
							CreatedAtAnnotation:          "2025-01-01T00:00:00Z",
						},
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						corev1.DockerConfigJsonKey: dockerConfigJSON,
					},
				}
			}(),
			wantErr: false,
		},
		{
			name: "invalid secret type",
			secret: &corev1.Secret{
				Type: corev1.SecretTypeOpaque,
			},
			wantErr: true,
			errMsg:  "invalid secret type",
		},
		{
			name: "missing required label",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						AppNameLabel: AppName,
						// Missing AppComponentLabel
					},
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
			wantErr: true,
			errMsg:  "missing or invalid label",
		},
		{
			name: "auth without colon",
			secret: func() *corev1.Secret {
				authStr := base64.StdEncoding.EncodeToString([]byte("invalidformat"))
				dockerConfig := DockerConfig{
					Auths: map[string]AuthEntry{
						"registry.io": {Auth: authStr},
					},
				}
				dockerConfigJSON, _ := json.Marshal(dockerConfig)

				return &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							AppNameLabel:      AppName,
							AppComponentLabel: ComponentRegistry,
						},
						Annotations: map[string]string{
							RegistryURLAnnotation:        "registry.io",
							ScenarioRepositoryAnnotation: "org/repo",
							SkipTLSAnnotation:            "false",
							InsecureAnnotation:           "false",
							CreatedByAnnotation:          "admin",
							CreatedAtAnnotation:          "2025-01-01T00:00:00Z",
						},
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						corev1.DockerConfigJsonKey: dockerConfigJSON,
					},
				}
			}(),
			wantErr: true,
			errMsg:  "auth must be in format base64(username:password)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegistrySecret(tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRegistrySecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Error should contain %q, got: %v", tt.errMsg, err)
			}
		})
	}
}
