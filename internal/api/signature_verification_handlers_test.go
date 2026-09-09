package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krkn-chaos/krkn-operator/pkg/signatureverification"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSignatureVerificationSettingsHandler_GetIsAvailableToUsers(t *testing.T) {
	handler := setupFilesTestHandler()
	req := addUserContext(httptest.NewRequest(http.MethodGet, SignatureVerificationSettingsPath, nil), "user@test.example")
	w := httptest.NewRecorder()

	handler.SignatureVerificationSettingsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	var response SignatureVerificationSettingsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Enabled {
		t.Fatal("expected verification to be enabled by default")
	}
}

func TestSignatureVerificationSettingsHandler_PatchRequiresAdmin(t *testing.T) {
	handler := setupFilesTestHandler()
	body := bytes.NewBufferString(`{"enabled":false}`)
	req := addUserContext(httptest.NewRequest(http.MethodPatch, SignatureVerificationSettingsPath, body), "user@test.example")
	w := httptest.NewRecorder()

	handler.SignatureVerificationSettingsHandler(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", w.Code)
	}
}

func TestSignatureVerificationSettingsHandler_AdminCanUpdateAndRead(t *testing.T) {
	handler := setupFilesTestHandler()
	body := bytes.NewBufferString(`{"enabled":false}`)
	patchRequest := addAdminContext(httptest.NewRequest(http.MethodPatch, SignatureVerificationSettingsPath, body))
	patchResponse := httptest.NewRecorder()

	handler.SignatureVerificationSettingsHandler(patchResponse, patchRequest)

	if patchResponse.Code != http.StatusOK {
		t.Fatalf("expected patch status 200, got %d: %s", patchResponse.Code, patchResponse.Body.String())
	}

	getRequest := addUserContext(httptest.NewRequest(http.MethodGet, SignatureVerificationSettingsPath, nil), "user@test.example")
	getResponse := httptest.NewRecorder()
	handler.SignatureVerificationSettingsHandler(getResponse, getRequest)

	var response SignatureVerificationSettingsResponse
	if err := json.Unmarshal(getResponse.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Enabled {
		t.Fatal("expected user GET to observe the admin's disabled setting")
	}

	var settings corev1.ConfigMap
	if err := handler.client.Get(context.Background(), client.ObjectKey{Name: signatureverification.ConfigMapName, Namespace: handler.namespace}, &settings); err != nil {
		t.Fatalf("get persisted settings: %v", err)
	}
	if settings.Data[signatureverification.EnabledKey] != "false" {
		t.Fatalf("expected persisted false setting, got %q", settings.Data[signatureverification.EnabledKey])
	}
}
