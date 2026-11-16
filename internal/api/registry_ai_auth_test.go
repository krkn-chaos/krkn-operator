package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	"github.com/krkn-chaos/krkn-operator/pkg/registry"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPrivateKrknAIRequestsAuthorizeRegistryBeforeCallingService(t *testing.T) {
	const userID = "viewer@example.com"
	var serviceCalls int
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		serviceCalls++
		_, _ = w.Write([]byte(`{"scenarioTags":[],"workloadImageTags":{}}`))
	}))
	defer service.Close()
	handler := newKrknAITestHandler(t, service.URL)
	addKrknAITestClusterPermission(t, handler, userID, "run-viewers", "https://cluster.example", string(groupauth.ActionRun))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "private", Namespace: "default", Labels: map[string]string{registry.AppComponentLabel: registry.ComponentRegistry}},
	}
	if err := handler.client.Create(t.Context(), secret); err != nil {
		t.Fatal(err)
	}

	for _, request := range []struct {
		path string
		body string
	}{
		{path: KrknAIPath + "/discoveries", body: `{"targetRequestId":"target","targetClusters":{"provider":["cluster"]},"registryName":"private"}`},
		{path: KrknAIPath + "/configs/validate", body: `{"configYaml":"generations: 1\n","registryName":"private"}`},
	} {
		response := httptest.NewRecorder()
		handler.KrknAIRouter(response, userKrknAIRequest(http.MethodPost, request.path, request.body, userID))
		if response.Code != http.StatusForbidden {
			t.Fatalf("%s returned %d, want 403: %s", request.path, response.Code, response.Body.String())
		}
	}
	if serviceCalls != 0 {
		t.Fatalf("unauthorized private requests called artifact service %d times", serviceCalls)
	}
}

func TestValidateKrknAIRegistrySelection(t *testing.T) {
	for _, test := range []struct {
		name         string
		registryName string
		wantError    bool
	}{
		{name: "public"},
		{name: "private", registryName: "private"},
		{name: "invalid registry", registryName: "private/name", wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateKrknAIRegistrySelection(test.registryName)
			if (err != nil) != test.wantError {
				t.Fatalf("validateKrknAIRegistrySelection() error = %v, wantError %t", err, test.wantError)
			}
		})
	}
}
