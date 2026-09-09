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

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/elasticsearch"
)

// newEsTestSecret creates a pre-built Elasticsearch config Secret for use in tests.
func newEsTestSecret(name, namespace, host string, port int) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    elasticsearch.BuildLabels(),
			Annotations: elasticsearch.BuildAnnotations(
				host, port, "telemetry-idx", "metrics-idx", "alerts-idx", "", "admin@test.local",
			),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			elasticsearch.SecretKeyUsername: []byte("elastic"),
			elasticsearch.SecretKeyPassword: []byte("secret-pass"),
		},
	}
}

func newEsScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	return scheme
}

// ── CreateElasticsearchConfig ────────────────────────────────────────────────

func TestCreateElasticsearchConfig_Success(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := elasticsearch.CreateElasticsearchConfigRequest{
		Name:           "prod-es",
		Host:           "https://es.example.com",
		Port:           9200,
		Username:       "elastic",
		Password:       "s3cr3t",
		TelemetryIndex: "krkn-telemetry",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, ElasticsearchConfigsPath, bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.CreateElasticsearchConfig(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp elasticsearch.CreateElasticsearchConfigResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Name != "prod-es" {
		t.Errorf("expected name 'prod-es', got %q", resp.Name)
	}
}

func TestCreateElasticsearchConfig_Forbidden(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := elasticsearch.CreateElasticsearchConfigRequest{Name: "es", Host: "https://es.example.com"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, ElasticsearchConfigsPath, bytes.NewReader(b))
	req = req.WithContext(createUserContext("user@example.com"))
	w := httptest.NewRecorder()

	handler.CreateElasticsearchConfig(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestCreateElasticsearchConfig_MissingName(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := elasticsearch.CreateElasticsearchConfigRequest{Host: "https://es.example.com"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, ElasticsearchConfigsPath, bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.CreateElasticsearchConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateElasticsearchConfig_MissingHost(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := elasticsearch.CreateElasticsearchConfigRequest{Name: "es"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, ElasticsearchConfigsPath, bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.CreateElasticsearchConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateElasticsearchConfig_InvalidPort(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := elasticsearch.CreateElasticsearchConfigRequest{Name: "es", Host: "https://es.example.com", Port: 99999}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, ElasticsearchConfigsPath, bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.CreateElasticsearchConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateElasticsearchConfig_DuplicateName(t *testing.T) {
	existing := newEsTestSecret("prod-es", "default", "https://es.example.com", 9200)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).WithObjects(existing).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := elasticsearch.CreateElasticsearchConfigRequest{Name: "prod-es", Host: "https://es.example.com"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, ElasticsearchConfigsPath, bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.CreateElasticsearchConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for duplicate, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateElasticsearchConfig_InvalidBody(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodPost, ElasticsearchConfigsPath, bytes.NewReader([]byte("not-json")))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.CreateElasticsearchConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ── ListElasticsearchConfigs ─────────────────────────────────────────────────

func TestListElasticsearchConfigs_Success(t *testing.T) {
	s1 := newEsTestSecret("es-prod", "default", "https://prod.es.example.com", 9200)
	s2 := newEsTestSecret("es-staging", "default", "https://staging.es.example.com", 9300)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).WithObjects(s1, s2).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, ElasticsearchConfigsPath, nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.ListElasticsearchConfigs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp elasticsearch.ListElasticsearchConfigsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected Total=2, got %d", resp.Total)
	}
	if len(resp.Configs) != 2 {
		t.Errorf("expected 2 configs, got %d", len(resp.Configs))
	}
}

func TestListElasticsearchConfigs_Empty(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, ElasticsearchConfigsPath, nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.ListElasticsearchConfigs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp elasticsearch.ListElasticsearchConfigsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("expected Total=0, got %d", resp.Total)
	}
}

func TestListElasticsearchConfigs_NonAdminAllowed(t *testing.T) {
	s := newEsTestSecret("es-prod", "default", "https://es.example.com", 9200)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).WithObjects(s).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, ElasticsearchConfigsPath, nil)
	req = req.WithContext(createUserContext("user@example.com"))
	w := httptest.NewRecorder()

	handler.ListElasticsearchConfigs(w, req)

	// List is open to any authenticated user, not admin-only
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for non-admin list, got %d", w.Code)
	}
}

func TestListElasticsearchConfigs_ExcludesNonEsSecrets(t *testing.T) {
	esSecret := newEsTestSecret("es-prod", "default", "https://es.example.com", 9200)
	// A secret without ES labels — must not appear in the list
	otherSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unrelated-secret",
			Namespace: "default",
			Labels:    map[string]string{"app": "something-else"},
		},
		Type: corev1.SecretTypeOpaque,
	}
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).WithObjects(esSecret, otherSecret).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, ElasticsearchConfigsPath, nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.ListElasticsearchConfigs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp elasticsearch.ListElasticsearchConfigsResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Errorf("expected 1 ES config (not the unrelated secret), got %d", resp.Total)
	}
}

// ── GetElasticsearchConfig ───────────────────────────────────────────────────

func TestGetElasticsearchConfig_Success(t *testing.T) {
	secret := newEsTestSecret("prod-es", "default", "https://es.example.com", 9200)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).WithObjects(secret).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, ElasticsearchConfigsPath+"/prod-es", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.GetElasticsearchConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp elasticsearch.ElasticsearchConfigResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Name != "prod-es" {
		t.Errorf("expected name 'prod-es', got %q", resp.Name)
	}
	if resp.Host != "https://es.example.com" {
		t.Errorf("expected host 'https://es.example.com', got %q", resp.Host)
	}
	if resp.Port != 9200 {
		t.Errorf("expected port 9200, got %d", resp.Port)
	}
	if resp.Username != "elastic" {
		t.Errorf("expected username 'elastic', got %q", resp.Username)
	}
}

func TestGetElasticsearchConfig_NotFound(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, ElasticsearchConfigsPath+"/nonexistent", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.GetElasticsearchConfig(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetElasticsearchConfig_Forbidden(t *testing.T) {
	secret := newEsTestSecret("prod-es", "default", "https://es.example.com", 9200)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).WithObjects(secret).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, ElasticsearchConfigsPath+"/prod-es", nil)
	req = req.WithContext(createUserContext("user@example.com"))
	w := httptest.NewRecorder()

	handler.GetElasticsearchConfig(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// ── UpdateElasticsearchConfig ────────────────────────────────────────────────

func TestUpdateElasticsearchConfig_Success(t *testing.T) {
	secret := newEsTestSecret("prod-es", "default", "https://es.example.com", 9200)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).WithObjects(secret).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := elasticsearch.UpdateElasticsearchConfigRequest{
		Host:           "https://new-es.example.com",
		Port:           9300,
		Username:       "newuser",
		Password:       "newpassword",
		TelemetryIndex: "new-telemetry",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, ElasticsearchConfigsPath+"/prod-es", bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.UpdateElasticsearchConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp elasticsearch.UpdateElasticsearchConfigResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Name != "prod-es" {
		t.Errorf("expected name 'prod-es', got %q", resp.Name)
	}
}

func TestUpdateElasticsearchConfig_PreservesPasswordWhenEmpty(t *testing.T) {
	secret := newEsTestSecret("prod-es", "default", "https://es.example.com", 9200)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).WithObjects(secret).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	// Update with empty password — existing password must be preserved
	body := elasticsearch.UpdateElasticsearchConfigRequest{
		Host:     "https://new-es.example.com",
		Username: "elastic",
		Password: "",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, ElasticsearchConfigsPath+"/prod-es", bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.UpdateElasticsearchConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Read the updated secret and verify the password was preserved
	var updatedSecret corev1.Secret
	key := types.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}
	if err := fakeClient.Get(context.Background(), key, &updatedSecret); err == nil {
		pass := string(updatedSecret.Data[elasticsearch.SecretKeyPassword])
		if pass != "secret-pass" {
			t.Errorf("expected password preserved as 'secret-pass', got %q", pass)
		}
	}
}

func TestUpdateElasticsearchConfig_PreservesUsernameWhenEmpty(t *testing.T) {
	secret := newEsTestSecret("prod-es", "default", "https://es.example.com", 9200)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).WithObjects(secret).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	// Update with empty username — existing username must be preserved
	body := elasticsearch.UpdateElasticsearchConfigRequest{
		Host:     "https://new-es.example.com",
		Username: "",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, ElasticsearchConfigsPath+"/prod-es", bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.UpdateElasticsearchConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updatedSecret corev1.Secret
	key := types.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}
	if err := fakeClient.Get(context.Background(), key, &updatedSecret); err != nil {
		t.Fatalf("failed to get updated secret: %v", err)
	}
	if user := string(updatedSecret.Data[elasticsearch.SecretKeyUsername]); user != "elastic" {
		t.Errorf("expected username preserved as 'elastic', got %q", user)
	}
}

func TestUpdateElasticsearchConfig_NilDataMapDoesNotPanic(t *testing.T) {
	// Construct a secret that has no Data map (nil) to verify the handler
	// initializes it before writing rather than panicking.
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-data-es",
			Namespace: "default",
			Labels:    elasticsearch.BuildLabels(),
			Annotations: elasticsearch.BuildAnnotations(
				"https://es.example.com", 9200, "", "", "", "", "admin@test.local",
			),
		},
		Type: corev1.SecretTypeOpaque,
		Data: nil,
	}
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).WithObjects(secret).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := elasticsearch.UpdateElasticsearchConfigRequest{
		Host:     "https://new-es.example.com",
		Username: "elastic",
		Password: "s3cr3t",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, ElasticsearchConfigsPath+"/no-data-es", bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.UpdateElasticsearchConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (no panic), got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateElasticsearchConfig_NotFound(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := elasticsearch.UpdateElasticsearchConfigRequest{Host: "https://es.example.com"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, ElasticsearchConfigsPath+"/nonexistent", bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.UpdateElasticsearchConfig(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestUpdateElasticsearchConfig_MissingHost(t *testing.T) {
	secret := newEsTestSecret("prod-es", "default", "https://es.example.com", 9200)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).WithObjects(secret).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := elasticsearch.UpdateElasticsearchConfigRequest{Port: 9200} // missing Host
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, ElasticsearchConfigsPath+"/prod-es", bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.UpdateElasticsearchConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdateElasticsearchConfig_Forbidden(t *testing.T) {
	secret := newEsTestSecret("prod-es", "default", "https://es.example.com", 9200)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).WithObjects(secret).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := elasticsearch.UpdateElasticsearchConfigRequest{Host: "https://es.example.com"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, ElasticsearchConfigsPath+"/prod-es", bytes.NewReader(b))
	req = req.WithContext(createUserContext("user@example.com"))
	w := httptest.NewRecorder()

	handler.UpdateElasticsearchConfig(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// ── DeleteElasticsearchConfig ────────────────────────────────────────────────

func TestDeleteElasticsearchConfig_Success(t *testing.T) {
	secret := newEsTestSecret("prod-es", "default", "https://es.example.com", 9200)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).WithObjects(secret).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodDelete, ElasticsearchConfigsPath+"/prod-es", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.DeleteElasticsearchConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp elasticsearch.DeleteElasticsearchConfigResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Message == "" {
		t.Error("expected non-empty message in delete response")
	}
}

func TestDeleteElasticsearchConfig_NotFound(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodDelete, ElasticsearchConfigsPath+"/nonexistent", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.DeleteElasticsearchConfig(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeleteElasticsearchConfig_Forbidden(t *testing.T) {
	secret := newEsTestSecret("prod-es", "default", "https://es.example.com", 9200)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).WithObjects(secret).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodDelete, ElasticsearchConfigsPath+"/prod-es", nil)
	req = req.WithContext(createUserContext("user@example.com"))
	w := httptest.NewRecorder()

	handler.DeleteElasticsearchConfig(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// ── ElasticsearchConfigsRouter ───────────────────────────────────────────────

func TestElasticsearchConfigsRouter_RoutesCorrectly(t *testing.T) {
	s1 := newEsTestSecret("es-a", "default", "https://a.example.com", 9200)
	s2 := newEsTestSecret("es-b", "default", "https://b.example.com", 9200)

	tests := []struct {
		method     string
		path       string
		body       interface{}
		wantStatus int
	}{
		{http.MethodGet, ElasticsearchConfigsPath, nil, http.StatusOK},
		{http.MethodPost, ElasticsearchConfigsPath, elasticsearch.CreateElasticsearchConfigRequest{Name: "new-es", Host: "https://new.example.com"}, http.StatusCreated},
		{http.MethodGet, ElasticsearchConfigsPath + "/es-a", nil, http.StatusOK},
		{http.MethodPut, ElasticsearchConfigsPath + "/es-a", elasticsearch.UpdateElasticsearchConfigRequest{Host: "https://updated.example.com"}, http.StatusOK},
		{http.MethodDelete, ElasticsearchConfigsPath + "/es-b", nil, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s %s", tt.method, tt.path), func(t *testing.T) {
			fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).WithObjects(s1.DeepCopy(), s2.DeepCopy()).Build()
			handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

			var bodyReader *bytes.Reader
			if tt.body != nil {
				b, _ := json.Marshal(tt.body)
				bodyReader = bytes.NewReader(b)
			} else {
				bodyReader = bytes.NewReader(nil)
			}

			req := httptest.NewRequest(tt.method, tt.path, bodyReader)
			req = req.WithContext(createAdminContext())
			w := httptest.NewRecorder()

			handler.ElasticsearchConfigsRouter(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestElasticsearchConfigsRouter_MethodNotAllowed(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodPatch, ElasticsearchConfigsPath, nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.ElasticsearchConfigsRouter(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestElasticsearchConfigsRouter_NotFound(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, ElasticsearchConfigsPath+"/a/b/extra", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.ElasticsearchConfigsRouter(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ── QueryElasticsearchTelemetry ──────────────────────────────────────────────

// newEsTestSecretWithHost builds a config Secret whose host annotation points at
// the given URL (including scheme) and whose telemetry index is set, so the
// query handler can be driven against an httptest server.
func newEsTestSecretWithHost(name, namespace, host, telemetryIndex string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    elasticsearch.BuildLabels(),
			Annotations: elasticsearch.BuildAnnotations(
				host, 9200, telemetryIndex, "", "", "", "admin@test.local",
			),
		},
		Type: corev1.SecretTypeOpaque,
		// No credentials: the httptest server speaks plaintext HTTP, and the
		// client refuses to send credentials over an unencrypted connection.
		Data: map[string][]byte{},
	}
}

func TestQueryElasticsearchTelemetry_Success(t *testing.T) {
	esServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"hits":{"hits":[{"_source":{"run_uuid":"abc","job_status":true,"scenarios":[{"scenario_type":"pod","start_timestamp":100,"end_timestamp":200,"exit_status":0,"parameters":[{"config":{"namespace":"ns1"}}]}]}}]},"aggregations":{"by_job_status":{"buckets":[{"key":1,"key_as_string":"true","doc_count":1}]}}}`))
	}))
	defer esServer.Close()

	secret := newEsTestSecretWithHost("prod-es", "default", esServer.URL, "krkn-telemetry")
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).WithObjects(secret).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body, _ := json.Marshal(elasticsearch.QueryTelemetryRequest{ConfigName: "prod-es"})
	req := httptest.NewRequest(http.MethodPost, ElasticsearchQueryPath, bytes.NewReader(body))
	req = req.WithContext(createUserContext("user@example.com"))
	w := httptest.NewRecorder()

	handler.QueryElasticsearchTelemetry(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp elasticsearch.QueryTelemetryResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Total != 1 || len(resp.Documents) != 1 {
		t.Fatalf("expected 1 document, got %d", resp.Total)
	}
	if resp.Documents[0].RunUUID != "abc" || resp.Documents[0].ScenarioType != "pod" || resp.Documents[0].Namespace != "ns1" {
		t.Errorf("unexpected document: %+v", resp.Documents[0])
	}
	wantStats := elasticsearch.TelemetryStats{Pass: 1, Fail: 0, PassPercent: 100}
	if resp.Stats != wantStats {
		t.Errorf("got stats %+v, want %+v", resp.Stats, wantStats)
	}
}

func TestQueryElasticsearchTelemetry_ConfigNotFound(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body, _ := json.Marshal(elasticsearch.QueryTelemetryRequest{ConfigName: "missing"})
	req := httptest.NewRequest(http.MethodPost, ElasticsearchQueryPath, bytes.NewReader(body))
	req = req.WithContext(createUserContext("user@example.com"))
	w := httptest.NewRecorder()

	handler.QueryElasticsearchTelemetry(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestQueryElasticsearchTelemetry_MissingConfigName(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body, _ := json.Marshal(elasticsearch.QueryTelemetryRequest{})
	req := httptest.NewRequest(http.MethodPost, ElasticsearchQueryPath, bytes.NewReader(body))
	req = req.WithContext(createUserContext("user@example.com"))
	w := httptest.NewRecorder()

	handler.QueryElasticsearchTelemetry(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestQueryElasticsearchTelemetry_MethodNotAllowed(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, ElasticsearchQueryPath, nil)
	req = req.WithContext(createUserContext("user@example.com"))
	w := httptest.NewRecorder()

	handler.QueryElasticsearchTelemetry(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestQueryElasticsearchTelemetry_UpstreamError(t *testing.T) {
	// The upstream body contains a secret-looking marker; the sanitized response
	// must not leak it back to the caller.
	const upstreamSecret = "SENSITIVE-UPSTREAM-DETAIL"
	esServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized","reason":"` + upstreamSecret + `"}`))
	}))
	defer esServer.Close()

	secret := newEsTestSecretWithHost("prod-es", "default", esServer.URL, "krkn-telemetry")
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newEsScheme()).WithObjects(secret).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body, _ := json.Marshal(elasticsearch.QueryTelemetryRequest{ConfigName: "prod-es"})
	req := httptest.NewRequest(http.MethodPost, ElasticsearchQueryPath, bytes.NewReader(body))
	req = req.WithContext(createUserContext("user@example.com"))
	w := httptest.NewRecorder()

	handler.QueryElasticsearchTelemetry(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}

	respBody := w.Body.String()
	// The caller must receive a stable, sanitized message with no upstream body
	// content or status details leaked.
	if strings.Contains(respBody, upstreamSecret) {
		t.Errorf("response leaked upstream body content: %s", respBody)
	}
	if strings.Contains(respBody, "401") || strings.Contains(respBody, "unauthorized") {
		t.Errorf("response leaked upstream status detail: %s", respBody)
	}
	if !strings.Contains(respBody, "Failed to query Elasticsearch") {
		t.Errorf("expected stable sanitized message, got: %s", respBody)
	}
}

// TestQueryElasticsearchTelemetry_RouteAndAuth exercises the telemetry query
// endpoint through the real server mux and authentication middleware, rather
// than calling the handler directly. This verifies the route is registered at
// ElasticsearchQueryPath and that RequireAuth gates it: unauthenticated
// requests are rejected, and an authenticated request reaches the handler.
func TestQueryElasticsearchTelemetry_RouteAndAuth(t *testing.T) {
	const namespace = "krkn-operator-system"
	const userID = "user@example.com"

	esServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"hits":{"hits":[{"_source":{"run_uuid":"abc","job_status":true,"scenarios":[{"scenario_type":"pod","start_timestamp":100,"end_timestamp":200,"exit_status":0,"parameters":[{"config":{"namespace":"ns1"}}]}]}}]},"aggregations":{"by_job_status":{"buckets":[{"key":1,"key_as_string":"true","doc_count":1}]}}}`))
	}))
	defer esServer.Close()

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = krknv1alpha1.AddToScheme(scheme)

	esConfig := newEsTestSecretWithHost("prod-es", namespace, esServer.URL, "krkn-telemetry")
	activeUser := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sanitizeUsername(userID),
			Namespace: namespace,
		},
		Spec:   krknv1alpha1.KrknUserSpec{UserID: userID, Role: "user"},
		Status: krknv1alpha1.KrknUserStatus{Active: true},
	}

	k8sClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(esConfig, activeUser).
		Build()

	// Start a real SecretManager so the middleware can validate tokens against
	// the same JWT secret the server uses.
	secretManager := auth.NewSecretManager(k8sClient, namespace, TokenDuration, "krkn-operator")
	smCtx, smCancel := context.WithCancel(context.Background())
	defer smCancel()
	go func() { _ = secretManager.Start(smCtx) }()

	deadline := time.Now().Add(5 * time.Second)
	for !secretManager.IsReady() {
		if time.Now().After(deadline) {
			t.Fatal("secret manager did not become ready in time")
		}
		time.Sleep(10 * time.Millisecond)
	}

	server := NewServer(0, k8sClient, fake.NewSimpleClientset(), namespace, "localhost:50051", secretManager)
	defer func() { _ = server.Shutdown() }()
	mux := server.HTTPHandler()

	newRequest := func() *http.Request {
		body, _ := json.Marshal(elasticsearch.QueryTelemetryRequest{ConfigName: "prod-es"})
		req := httptest.NewRequest(http.MethodPost, ElasticsearchQueryPath, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		return req
	}

	t.Run("unauthenticated request is rejected", func(t *testing.T) {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, newRequest())
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 without a token, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("authenticated request reaches the handler", func(t *testing.T) {
		tokenGen, err := secretManager.GetTokenGenerator()
		if err != nil {
			t.Fatalf("failed to get token generator: %v", err)
		}
		token, err := tokenGen.GenerateToken(userID, "user", "Regular", "User", "Org")
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		req := newRequest()
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 with a valid token, got %d: %s", w.Code, w.Body.String())
		}
		var resp elasticsearch.QueryTelemetryResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Total != 1 || len(resp.Documents) != 1 || resp.Documents[0].RunUUID != "abc" {
			t.Fatalf("unexpected telemetry response: %+v", resp)
		}
	})
}
