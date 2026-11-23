package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestInvalidProviderDoesNotHideValidLoginChoice(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	valid := identityProviderConfig{ID: "working", Type: "keycloak", DisplayName: "Working",
		Issuer: "https://keycloak.example/realms/working", ClientID: "login", ClientSecret: "secret",
		RedirectURL: "https://operator.example" + OIDCBasePath + "/working/callback",
		Enabled:     true, SessionEpoch: "epoch"}
	broken := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: identityProviderSecretName("broken"), Namespace: "default",
		Labels: map[string]string{identityProviderLabel: "true"}}, Data: map[string][]byte{"type": []byte("keycloak")}}
	k8s := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(valid.secret("default"), broken).Build()
	h := &Handler{providerStore: newIdentityProviderStore(k8s, "default")}

	public := httptest.NewRecorder()
	h.PublicIdentityProviders(public, httptest.NewRequest(http.MethodGet, AuthBasePath+"/providers", nil))
	if public.Code != http.StatusOK {
		t.Fatalf("public provider listing returned %d: %s", public.Code, public.Body.String())
	}
	var publicBody struct {
		Providers []struct {
			ID string `json:"id"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(public.Body.Bytes(), &publicBody); err != nil {
		t.Fatal(err)
	}
	if len(publicBody.Providers) != 1 || publicBody.Providers[0].ID != "working" {
		t.Fatalf("valid login choice hidden by malformed Secret: %+v", publicBody.Providers)
	}

	admin := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, IdentityProvidersPath, nil)
	request = request.WithContext(context.WithValue(request.Context(), auth.UserClaimsKey, &auth.Claims{Role: string(auth.RoleAdmin)}))
	h.IdentityProviders(admin, request)
	if admin.Code != http.StatusOK {
		t.Fatalf("admin provider listing returned %d: %s", admin.Code, admin.Body.String())
	}
	var adminBody struct {
		InvalidProviders []invalidIdentityProvider `json:"invalidProviders"`
	}
	if err := json.Unmarshal(admin.Body.Bytes(), &adminBody); err != nil {
		t.Fatal(err)
	}
	if len(adminBody.InvalidProviders) != 1 || adminBody.InvalidProviders[0].SecretName != broken.Name {
		t.Fatalf("invalid provider diagnostic missing: %+v", adminBody.InvalidProviders)
	}
}

func TestInvalidKeycloakSecretDoesNotExposeLegacyProvider(t *testing.T) {
	t.Setenv("KEYCLOAK_ISSUER_URL", "https://legacy.example/realms/main")
	t.Setenv("KEYCLOAK_OIDC_CLIENT_ID", "legacy-login")
	t.Setenv("KEYCLOAK_OIDC_CLIENT_SECRET", "legacy-secret")
	t.Setenv("KEYCLOAK_OIDC_REDIRECT_URL", "https://operator.example"+OIDCBasePath+"/keycloak/callback")
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	broken := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: identityProviderSecretName("keycloak"), Namespace: "default",
		Labels: map[string]string{identityProviderLabel: "true"}}}
	k8s := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(broken).Build()
	configs, invalid, err := newIdentityProviderStore(k8s, "default").list(t.Context())
	if err != nil || len(configs) != 0 || len(invalid) != 1 {
		t.Fatalf("malformed Secret failed to shadow legacy provider: configs=%+v invalid=%+v err=%v", configs, invalid, err)
	}
}
