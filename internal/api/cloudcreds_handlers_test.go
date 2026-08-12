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
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/krkn-chaos/krkn-operator/pkg/cloudcreds"
)

func newCloudCredScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	return scheme
}

func newAWSTestSecret(name, namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      cloudcreds.BuildLabels(cloudcreds.ProviderAWS, nil, false),
			Annotations: cloudcreds.BuildAnnotations("test aws creds", "admin@test.local"),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			cloudcreds.SecretKeyAWSAccessKeyID:     []byte("AKIATEST"),
			cloudcreds.SecretKeyAWSSecretAccessKey: []byte("secret"),
			cloudcreds.SecretKeyAWSDefaultRegion:   []byte("us-east-1"),
		},
	}
}

func newGCPTestSecret(name, namespace string) *corev1.Secret {
	decoded := []byte(`{"type":"service_account","project_id":"test-project"}`)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      cloudcreds.BuildLabels(cloudcreds.ProviderGCP, nil, true),
			Annotations: cloudcreds.BuildAnnotations("test gcp creds", "admin@test.local"),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			cloudcreds.SecretKeyGCPServiceAccountJSON: decoded,
		},
	}
}

func newGroupedAWSSecret(name, namespace string, groups []string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      cloudcreds.BuildLabels(cloudcreds.ProviderAWS, groups, false),
			Annotations: cloudcreds.BuildAnnotations("grouped aws", "admin@test.local"),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			cloudcreds.SecretKeyAWSAccessKeyID:     []byte("AKIATEST"),
			cloudcreds.SecretKeyAWSSecretAccessKey: []byte("secret"),
			cloudcreds.SecretKeyAWSDefaultRegion:   []byte("us-east-1"),
		},
	}
}

// ── CreateCloudCredential ───────────────────────────────────────────────────

func TestCreateCloudCredential_AWS_Success(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := cloudcreds.CreateCloudCredentialRequest{
		Name:               "aws-prod",
		Provider:           cloudcreds.ProviderAWS,
		Description:        "Production AWS",
		AWSAccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		AWSSecretAccessKey: "wJalrXUtnFEMI/K7MDENG",
		AWSDefaultRegion:   "us-east-1",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, CloudCredentialsPath, bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.CreateCloudCredential(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp cloudcreds.CreateCloudCredentialResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Name != "aws-prod" {
		t.Errorf("expected name 'aws-prod', got %q", resp.Name)
	}
}

func TestCreateCloudCredential_GCP_Success(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	gcpJSON := base64.StdEncoding.EncodeToString([]byte(`{"type":"service_account","project_id":"my-project"}`))
	body := cloudcreds.CreateCloudCredentialRequest{
		Name:                  "gcp-prod",
		Provider:              cloudcreds.ProviderGCP,
		GCPServiceAccountJSON: gcpJSON,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, CloudCredentialsPath, bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.CreateCloudCredential(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateCloudCredential_Forbidden(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := cloudcreds.CreateCloudCredentialRequest{
		Name: "aws-prod", Provider: cloudcreds.ProviderAWS,
		AWSAccessKeyID: "k", AWSSecretAccessKey: "s", AWSDefaultRegion: "r",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, CloudCredentialsPath, bytes.NewReader(b))
	req = req.WithContext(createUserContext("user@example.com"))
	w := httptest.NewRecorder()

	handler.CreateCloudCredential(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestCreateCloudCredential_MissingName(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := cloudcreds.CreateCloudCredentialRequest{Provider: cloudcreds.ProviderAWS}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, CloudCredentialsPath, bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.CreateCloudCredential(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateCloudCredential_InvalidProvider(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := cloudcreds.CreateCloudCredentialRequest{Name: "test", Provider: "oracle"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, CloudCredentialsPath, bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.CreateCloudCredential(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateCloudCredential_ReservedName(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := cloudcreds.CreateCloudCredentialRequest{
		Name: "available", Provider: cloudcreds.ProviderAWS,
		AWSAccessKeyID: "k", AWSSecretAccessKey: "s", AWSDefaultRegion: "r",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, CloudCredentialsPath, bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.CreateCloudCredential(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateCloudCredential_DuplicateName(t *testing.T) {
	existing := newAWSTestSecret("aws-prod", "default")
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).WithObjects(existing).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := cloudcreds.CreateCloudCredentialRequest{
		Name: "aws-prod", Provider: cloudcreds.ProviderAWS,
		AWSAccessKeyID: "k", AWSSecretAccessKey: "s", AWSDefaultRegion: "r",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, CloudCredentialsPath, bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.CreateCloudCredential(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateCloudCredential_InvalidBody(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodPost, CloudCredentialsPath, bytes.NewReader([]byte("not-json")))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.CreateCloudCredential(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ── ListCloudCredentials ────────────────────────────────────────────────────

func TestListCloudCredentials_Success(t *testing.T) {
	s1 := newAWSTestSecret("aws-prod", "default")
	s2 := newGCPTestSecret("gcp-prod", "default")
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).WithObjects(s1, s2).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, CloudCredentialsPath, nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.ListCloudCredentials(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp cloudcreds.ListCloudCredentialsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected Total=2, got %d", resp.Total)
	}
}

func TestListCloudCredentials_Empty(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, CloudCredentialsPath, nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.ListCloudCredentials(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp cloudcreds.ListCloudCredentialsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("expected Total=0, got %d", resp.Total)
	}
}

func TestListCloudCredentials_NonAdminForbidden(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, CloudCredentialsPath, nil)
	req = req.WithContext(createUserContext("user@example.com"))
	w := httptest.NewRecorder()

	handler.ListCloudCredentials(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestListCloudCredentials_ExcludesNonCloudCredSecrets(t *testing.T) {
	cloudSecret := newAWSTestSecret("aws-prod", "default")
	otherSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-other-secret",
			Namespace: "default",
			Labels:    map[string]string{"app.kubernetes.io/component": "not-cloud-credential"},
		},
		Data: map[string][]byte{"key": []byte("value")},
	}
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).WithObjects(cloudSecret, otherSecret).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, CloudCredentialsPath, nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.ListCloudCredentials(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp cloudcreds.ListCloudCredentialsResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Errorf("expected Total=1 (only cloud cred), got %d", resp.Total)
	}
}

// ── GetCloudCredential ──────────────────────────────────────────────────────

func TestGetCloudCredential_Success(t *testing.T) {
	existing := newAWSTestSecret("aws-prod", "default")
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).WithObjects(existing).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, CloudCredentialsPath+"/aws-prod", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.GetCloudCredential(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp cloudcreds.CloudCredentialResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Name != "aws-prod" {
		t.Errorf("expected name 'aws-prod', got %q", resp.Name)
	}
	if resp.Provider != cloudcreds.ProviderAWS {
		t.Errorf("expected provider 'aws', got %q", resp.Provider)
	}
}

func TestGetCloudCredential_NotFound(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, CloudCredentialsPath+"/nonexistent", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.GetCloudCredential(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetCloudCredential_ResponseExcludesSecretData(t *testing.T) {
	existing := newAWSTestSecret("aws-prod", "default")
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).WithObjects(existing).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, CloudCredentialsPath+"/aws-prod", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.GetCloudCredential(w, req)

	body := w.Body.String()
	if bytes.Contains([]byte(body), []byte("AKIATEST")) {
		t.Error("response should not contain Secret data (access key)")
	}
	if bytes.Contains([]byte(body), []byte("secret")) && bytes.Contains([]byte(body), []byte("AccessKey")) {
		t.Error("response should not contain Secret data")
	}
}

// ── UpdateCloudCredential ───────────────────────────────────────────────────

func TestUpdateCloudCredential_Success(t *testing.T) {
	existing := newAWSTestSecret("aws-prod", "default")
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).WithObjects(existing).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := cloudcreds.UpdateCloudCredentialRequest{
		Description:    "Updated description",
		AWSDefaultRegion: "eu-west-1",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, CloudCredentialsPath+"/aws-prod", bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.UpdateCloudCredential(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateCloudCredential_NotFound(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	body := cloudcreds.UpdateCloudCredentialRequest{AWSDefaultRegion: "eu-west-1"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, CloudCredentialsPath+"/nonexistent", bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.UpdateCloudCredential(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ── DeleteCloudCredential ───────────────────────────────────────────────────

func TestDeleteCloudCredential_Success(t *testing.T) {
	existing := newAWSTestSecret("aws-prod", "default")
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).WithObjects(existing).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodDelete, CloudCredentialsPath+"/aws-prod", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.DeleteCloudCredential(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCloudCredential_NotFound(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodDelete, CloudCredentialsPath+"/nonexistent", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.DeleteCloudCredential(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ── ListAvailableCloudCredentials ───────────────────────────────────────────

func TestListAvailableCloudCredentials_AdminSeesAll(t *testing.T) {
	s1 := newAWSTestSecret("aws-prod", "default")
	s2 := newGroupedAWSSecret("aws-team", "default", []string{"team-a"})
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).WithObjects(s1, s2).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, CloudCredentialsAvailablePath, nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.ListAvailableCloudCredentials(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp cloudcreds.ListCloudCredentialsResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Errorf("expected admin to see 2 credentials, got %d", resp.Total)
	}
}

func TestListAvailableCloudCredentials_AvailableToAll(t *testing.T) {
	s := newGCPTestSecret("gcp-public", "default") // availableToAll=true
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).WithObjects(s).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, CloudCredentialsAvailablePath, nil)
	req = req.WithContext(createUserContext("user@example.com"))
	w := httptest.NewRecorder()

	handler.ListAvailableCloudCredentials(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp cloudcreds.ListCloudCredentialsResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Errorf("expected user to see 1 available-to-all credential, got %d", resp.Total)
	}
}

func TestListAvailableCloudCredentials_NonMemberSeesNothing(t *testing.T) {
	s := newGroupedAWSSecret("aws-team", "default", []string{"team-a"})
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).WithObjects(s).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodGet, CloudCredentialsAvailablePath, nil)
	req = req.WithContext(createUserContext("outsider@example.com"))
	w := httptest.NewRecorder()

	handler.ListAvailableCloudCredentials(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp cloudcreds.ListCloudCredentialsResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 0 {
		t.Errorf("expected non-member to see 0 credentials, got %d", resp.Total)
	}
}

// ── Router ──────────────────────────────────────────────────────────────────

func TestCloudCredentialsRouter_MethodNotAllowed(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodPatch, CloudCredentialsPath, nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.CloudCredentialsRouter(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestCloudCredentialsRouter_SubpathMethodNotAllowed(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	req := httptest.NewRequest(http.MethodPatch, CloudCredentialsPath+"/aws-prod", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.CloudCredentialsRouter(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// ── GCP base64 decode verification ──────────────────────────────────────────

func TestCreateCloudCredential_GCP_DecodesBase64(t *testing.T) {
	fakeClient := fakeclient.NewClientBuilder().WithScheme(newCloudCredScheme()).Build()
	handler := NewTestHandler(fakeClient, fake.NewSimpleClientset(), "default", "localhost:50051")

	originalJSON := `{"type":"service_account","project_id":"my-project"}`
	gcpJSON := base64.StdEncoding.EncodeToString([]byte(originalJSON))
	body := cloudcreds.CreateCloudCredentialRequest{
		Name:                  "gcp-test",
		Provider:              cloudcreds.ProviderGCP,
		GCPServiceAccountJSON: gcpJSON,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, CloudCredentialsPath, bytes.NewReader(b))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.CreateCloudCredential(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the stored Secret has decoded JSON, not base64
	var stored corev1.Secret
	if err := fakeClient.Get(req.Context(), k8stypes.NamespacedName{
		Name: "gcp-test", Namespace: "default",
	}, &stored); err != nil {
		t.Fatalf("failed to get stored secret: %v", err)
	}

	storedData := string(stored.Data[cloudcreds.SecretKeyGCPServiceAccountJSON])
	if storedData != originalJSON {
		t.Errorf("expected decoded JSON %q, got %q", originalJSON, storedData)
	}
}
