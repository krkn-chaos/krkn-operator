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

package visualize

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// grafanaIniFixture mirrors the [security] section of the grafana.ini shipped
// in the upstream krkn-visualize template.
const grafanaIniFixture = `#################################### Security ####################################
[security]
# default admin user, created on startup
admin_user = admin

# default admin password, can be changed before first start of grafana,  or in profile settings
admin_password = secret

# disable protection against brute force login attempts
;disable_brute_force_login_protection = false

#################################### Snapshots ###########################
[snapshots]
;external_enabled = true
`

func TestEnsureIniSetting(t *testing.T) {
	tests := []struct {
		name          string
		ini           string
		section       string
		key           string
		value         string
		wantChanged   bool
		wantSubstring string
		wantNotIn     string
	}{
		{
			name:          "inserts key into existing section",
			ini:           grafanaIniFixture,
			section:       "security",
			key:           "allow_embedding",
			value:         "true",
			wantChanged:   true,
			wantSubstring: "allow_embedding = true",
			wantNotIn:     "[snapshots]\nallow_embedding",
		},
		{
			name:          "appends missing section",
			ini:           "[server]\nhttp_port = 3000\n",
			section:       "security",
			key:           "allow_embedding",
			value:         "true",
			wantChanged:   true,
			wantSubstring: "[security]\nallow_embedding = true",
		},
		{
			name:          "replaces existing value in target section",
			ini:           "[security]\nallow_embedding = false\n\n[server]\nhttp_port = 3000\n",
			section:       "security",
			key:           "allow_embedding",
			value:         "true",
			wantChanged:   true,
			wantSubstring: "allow_embedding = true",
			wantNotIn:     "allow_embedding = false",
		},
		{
			name:        "no change when already set",
			ini:         "[security]\nallow_embedding = true\n",
			section:     "security",
			key:         "allow_embedding",
			value:       "true",
			wantChanged: false,
		},
		{
			name:        "ignores commented keys",
			ini:         "[security]\n;allow_embedding = true\n",
			section:     "security",
			key:         "allow_embedding",
			value:       "true",
			wantChanged: true,
		},
		{
			name:          "inserts into last section of file",
			ini:           "[server]\nhttp_port = 3000\n",
			section:       "server",
			key:           "domain",
			value:         "example.com",
			wantChanged:   true,
			wantSubstring: "http_port = 3000\ndomain = example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, changed := ensureIniSetting(tt.ini, tt.section, tt.key, tt.value)
			assert.Equal(t, tt.wantChanged, changed)
			if tt.wantSubstring != "" {
				assert.Contains(t, updated, tt.wantSubstring)
			}
			if tt.wantNotIn != "" {
				assert.NotContains(t, updated, tt.wantNotIn)
			}
			if !tt.wantChanged {
				assert.Equal(t, tt.ini, updated)
			}
		})
	}
}

func TestEnableGrafanaEmbedding(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))

	newConfigMap := func(ini string) *corev1.ConfigMap {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      UpstreamGrafanaConfigMap,
				Namespace: "krkn-visualize",
			},
			Data: map[string]string{GrafanaIniKey: ini},
		}
	}
	newDeployment := func() *appsv1.Deployment {
		return &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      UpstreamGrafanaDeployment,
				Namespace: "krkn-visualize",
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "krkn-visualize"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "krkn-visualize"}},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "grafana"}}},
				},
			},
		}
	}

	t.Run("patches config and restarts deployment", func(t *testing.T) {
		k8sClient := fake.NewClientBuilder().WithScheme(scheme).
			WithObjects(newConfigMap(grafanaIniFixture), newDeployment()).Build()

		require.NoError(t, EnableGrafanaEmbedding(context.Background(), k8sClient, "krkn-visualize"))

		configMap := &corev1.ConfigMap{}
		require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKeyFromObject(newConfigMap("")), configMap))
		assert.Contains(t, configMap.Data[GrafanaIniKey], "allow_embedding = true")
		// The setting must land in [security], not the following [snapshots] section.
		securitySection := configMap.Data[GrafanaIniKey][:strings.Index(configMap.Data[GrafanaIniKey], "[snapshots]")]
		assert.Contains(t, securitySection, "allow_embedding = true")

		deployment := &appsv1.Deployment{}
		require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKeyFromObject(newDeployment()), deployment))
		assert.NotEmpty(t, deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"])
	})

	t.Run("idempotent when already enabled", func(t *testing.T) {
		alreadyEnabled := strings.Replace(grafanaIniFixture, "admin_password = secret", "admin_password = secret\nallow_embedding = true", 1)
		k8sClient := fake.NewClientBuilder().WithScheme(scheme).
			WithObjects(newConfigMap(alreadyEnabled), newDeployment()).Build()

		require.NoError(t, EnableGrafanaEmbedding(context.Background(), k8sClient, "krkn-visualize"))

		configMap := &corev1.ConfigMap{}
		require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKeyFromObject(newConfigMap("")), configMap))
		assert.Equal(t, alreadyEnabled, configMap.Data[GrafanaIniKey])

		deployment := &appsv1.Deployment{}
		require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKeyFromObject(newDeployment()), deployment))
		assert.Empty(t, deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"])
	})

	t.Run("fails when config map is missing", func(t *testing.T) {
		k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		err := EnableGrafanaEmbedding(context.Background(), k8sClient, "krkn-visualize")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get Grafana config ConfigMap")
	})
}
