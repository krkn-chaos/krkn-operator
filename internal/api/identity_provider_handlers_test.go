package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/identity"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func TestIdentityProviderLifecycleAndSessionRevocation(t *testing.T) {
	t.Setenv("KEYCLOAK_ISSUER_URL", "")
	h := setupUserTestHandler()
	defer h.Shutdown()
	create := `{"id":"corp","type":"keycloak","displayName":"Corporate Keycloak",` +
		`"issuer":"https://id.example/realms/corp","clientId":"operator-login",` +
		`"clientSecret":"sensitive-login-secret",` +
		`"redirectUrl":"https://operator.example/api/v1/auth/oidc/corp/callback",` +
		`"adminClientId":"operator-groups","adminClientSecret":"sensitive-admin-secret","enabled":true}`
	request := httptest.NewRequest(http.MethodPost, IdentityProvidersPath, strings.NewReader(create)).WithContext(createAdminContext())
	w := httptest.NewRecorder()
	h.IdentityProviders(w, request)
	if w.Code != http.StatusCreated || strings.Contains(w.Body.String(), "sensitive-") {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}
	var provider IdentityProviderResponse
	if err := json.Unmarshal(w.Body.Bytes(), &provider); err != nil || !provider.HasClientSecret || !provider.HasAdminClientSecret {
		t.Fatalf("provider=%+v err=%v", provider, err)
	}
	key := identity.ExternalIdentity{Provider: "corp", Issuer: "https://id.example/realms/corp", Subject: "subject-42"}
	user, claims := oidcTestAccount(t, key, "account-uid")
	initialConfig, err := h.providerStore.get(context.Background(), "corp")
	if err != nil {
		t.Fatal(err)
	}
	claims.ProviderEpoch = initialConfig.SessionEpoch
	if err := h.client.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	checker := newK8sUserStatusChecker(h.client, "default")
	checker.providerStore = h.providerStore
	if active, err := checker.IsExternalIdentityActive(context.Background(), claims); err != nil || !active {
		t.Fatalf("enabled session active=%v err=%v", active, err)
	}

	public := httptest.NewRecorder()
	h.PublicIdentityProviders(public, httptest.NewRequest(http.MethodGet, AuthBasePath+"/providers", nil))
	if public.Code != http.StatusOK || !strings.Contains(public.Body.String(), `"loginUrl":"/api/v1/auth/oidc/corp/login"`) || strings.Contains(public.Body.String(), "sensitive-") {
		t.Fatalf("public providers status=%d body=%s", public.Code, public.Body.String())
	}

	patch := httptest.NewRequest(http.MethodPatch, IdentityProvidersPath+"/corp", strings.NewReader(`{"enabled":false}`)).WithContext(createAdminContext())
	w = httptest.NewRecorder()
	h.IdentityProviders(w, patch)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"enabled":false`) {
		t.Fatalf("disable status=%d body=%s", w.Code, w.Body.String())
	}
	if active, err := checker.IsExternalIdentityActive(context.Background(), claims); err != nil || active {
		t.Fatalf("disabled session active=%v err=%v", active, err)
	}
	public = httptest.NewRecorder()
	h.PublicIdentityProviders(public, httptest.NewRequest(http.MethodGet, AuthBasePath+"/providers", nil))
	if strings.Contains(public.Body.String(), "corp") {
		t.Fatalf("disabled provider remains public: %s", public.Body.String())
	}
	login := httptest.NewRecorder()
	h.OIDCRouter(login, httptest.NewRequest(http.MethodGet, AuthBasePath+"/oidc/corp/login", nil))
	if login.Code != http.StatusServiceUnavailable {
		t.Fatalf("disabled login status=%d", login.Code)
	}

	patch = httptest.NewRequest(http.MethodPatch, IdentityProvidersPath+"/corp", strings.NewReader(`{"issuer":"https://other.example/realms/corp"}`)).WithContext(createAdminContext())
	w = httptest.NewRecorder()
	h.IdentityProviders(w, patch)
	if w.Code != http.StatusConflict {
		t.Fatalf("issuer change status=%d", w.Code)
	}
	patch = httptest.NewRequest(http.MethodPatch, IdentityProvidersPath+"/corp", strings.NewReader(`{"enabled":true}`)).WithContext(createAdminContext())
	w = httptest.NewRecorder()
	h.IdentityProviders(w, patch)
	if w.Code != http.StatusOK {
		t.Fatalf("reenable status=%d body=%s", w.Code, w.Body.String())
	}
	if active, err := checker.IsExternalIdentityActive(context.Background(), claims); err != nil || active {
		t.Fatalf("old session revived after re-enable: active=%v err=%v", active, err)
	}
	currentConfig, err := h.providerStore.get(context.Background(), "corp")
	if err != nil {
		t.Fatal(err)
	}
	claims.ProviderEpoch = currentConfig.SessionEpoch
	if active, err := checker.IsExternalIdentityActive(context.Background(), claims); err != nil || !active {
		t.Fatalf("new session active=%v err=%v", active, err)
	}
	unauthorized := httptest.NewRecorder()
	h.IdentityProviders(unauthorized, httptest.NewRequest(http.MethodGet, IdentityProvidersPath, nil))
	if unauthorized.Code != http.StatusForbidden {
		t.Fatalf("unauthorized provider list status=%d", unauthorized.Code)
	}
}

func TestLegacyKeycloakEnvironmentProviderRemainsAvailable(t *testing.T) {
	t.Setenv("KEYCLOAK_ISSUER_URL", "https://id.example/realms/main")
	t.Setenv("KEYCLOAK_OIDC_CLIENT_ID", "login")
	t.Setenv("KEYCLOAK_OIDC_CLIENT_SECRET", "secret")
	t.Setenv("KEYCLOAK_OIDC_REDIRECT_URL", "https://operator.example"+KeycloakCallback)
	h := setupUserTestHandler()
	defer h.Shutdown()
	config, err := h.providerStore.get(context.Background(), "keycloak")
	if err != nil || config.Source != "environment" || !config.Enabled {
		t.Fatalf("legacy config=%+v err=%v", config, err)
	}
}

func TestListUsersFiltersOIDCProvider(t *testing.T) {
	corp, _ := oidcTestAccount(t, identity.ExternalIdentity{Provider: "corp", Issuer: "https://id.example/realms/corp", Subject: "user-1"}, "corp-uid")
	partner, _ := oidcTestAccount(t, identity.ExternalIdentity{Provider: "partner", Issuer: "https://id.example/realms/partner", Subject: "user-2"}, "partner-uid")
	h := setupUserTestHandler(corp, partner)
	defer h.Shutdown()
	w := httptest.NewRecorder()
	h.ListUsers(w, httptest.NewRequest(http.MethodGet, UsersPath+"?authProvider=corp", nil).WithContext(createAdminContext()))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var response ListUsersResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || response.Total != 1 || response.Users[0].AuthProvider != "corp" {
		t.Fatalf("response=%+v err=%v", response, err)
	}
}

func TestGroupDiscoveryUsesSelectedKeycloakInstance(t *testing.T) {
	var issuer string
	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/corp/protocol/openid-connect/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"service-token","token_type":"Bearer","expires_in":60}`))
		case "/admin/realms/corp/groups":
			if r.Header.Get("Authorization") != "Bearer service-token" {
				t.Errorf("missing service-account token: %q", r.Header.Get("Authorization"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"group-42","name":"Ops","path":"/Ops"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer idp.Close()
	issuer = idp.URL + "/realms/corp"
	config := identityProviderConfig{ID: "corp", Type: "keycloak", DisplayName: "Corporate Keycloak",
		Issuer: issuer, ClientID: "login", ClientSecret: "login-secret",
		RedirectURL:   "https://operator.example/api/v1/auth/oidc/corp/callback",
		AdminClientID: "groups", AdminClientSecret: "groups-secret", Enabled: true,
		SessionEpoch: "epoch", Source: "secret"}
	h := setupUserTestHandler(config.secret("default"))
	defer h.Shutdown()
	w := httptest.NewRecorder()
	h.IdentityProvidersRouter(w, httptest.NewRequest(http.MethodGet,
		IdentityProvidersPath+"/corp/groups?search=ops", nil).WithContext(createAdminContext()))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"provider":"corp"`) ||
		!strings.Contains(w.Body.String(), `"id":"group-42"`) || !strings.Contains(w.Body.String(), `"claimValue":"/Ops"`) {
		t.Fatalf("group discovery status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestGroupDiscoveryLogsUpstreamStatusWithoutStackTrace(t *testing.T) {
	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/corp/protocol/openid-connect/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"service-token","token_type":"Bearer","expires_in":60}`))
		case "/admin/realms/corp/groups":
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		default:
			http.NotFound(w, r)
		}
	}))
	defer idp.Close()
	config := identityProviderConfig{ID: "corp", Type: "keycloak", DisplayName: "Corporate Keycloak",
		Issuer: idp.URL + "/realms/corp", ClientID: "login", ClientSecret: "login-secret",
		RedirectURL:   "https://operator.example/api/v1/auth/oidc/corp/callback",
		AdminClientID: "groups", AdminClientSecret: "groups-secret", Enabled: true,
		SessionEpoch: "epoch", Source: "secret"}
	h := setupUserTestHandler(config.secret("default"))
	defer h.Shutdown()
	var entries []capturedLogEntry
	logger := logr.New(captureLogSink{entries: &entries})
	request := httptest.NewRequest(http.MethodGet, IdentityProvidersPath+"/corp/groups", nil)
	request = request.WithContext(log.IntoContext(createAdminContext(), logger))
	w := httptest.NewRecorder()
	h.IdentityProvidersRouter(w, request)
	if w.Code != http.StatusBadGateway || strings.Contains(w.Body.String(), "401") {
		t.Fatalf("group discovery status=%d body=%s", w.Code, w.Body.String())
	}
	if len(entries) != 1 || entries[0].isError || entries[0].msg != "Failed to list external groups" {
		t.Fatalf("expected one stack-free diagnostic log, got %+v", entries)
	}
	if provider, ok := kvValue(entries[0].keysAndValues, "provider"); !ok || provider != "corp" {
		t.Fatalf("missing provider in diagnostic log: %+v", entries[0])
	}
	if cause, ok := kvValue(entries[0].keysAndValues, "cause"); !ok || cause != "keycloak group request returned HTTP 401" {
		t.Fatalf("missing upstream status in diagnostic log: %+v", entries[0])
	}
}

func TestExternalMappingRejectsUnknownProviderID(t *testing.T) {
	config := identityProviderConfig{ID: "corp", Type: "keycloak", DisplayName: "Corporate Keycloak",
		Issuer: "https://id.example/realms/corp", ClientID: "login", ClientSecret: "secret",
		RedirectURL:  "https://operator.example/api/v1/auth/oidc/corp/callback",
		SessionEpoch: "epoch", Source: "secret"}
	h := setupUserTestHandler(config.secret("default"))
	defer h.Shutdown()
	if err := h.validateExternalGroupProviders(context.Background(), []krknv1alpha1.ExternalGroupMapping{
		{Provider: "corp", Group: "/Ops", GroupID: "group-42"}}); err != nil {
		t.Fatal(err)
	}
	if err := h.validateExternalGroupProviders(context.Background(), []krknv1alpha1.ExternalGroupMapping{
		{Provider: "partner", Group: "/Ops", GroupID: "group-42"}}); err == nil {
		t.Fatal("unknown provider ID was accepted")
	}
}

func TestSessionExposesMappedGroupsForOIDCEmptyState(t *testing.T) {
	group := &krknv1alpha1.KrknUserGroup{ObjectMeta: metav1.ObjectMeta{Name: "ops", Namespace: "default"},
		Spec: krknv1alpha1.KrknUserGroupSpec{ExternalGroups: []krknv1alpha1.ExternalGroupMapping{
			{Provider: "corp", Group: "/Ops"}}}}
	h := setupUserTestHandler(group)
	defer h.Shutdown()
	claims := &auth.Claims{UserID: "oidc-user", Role: "user", Provider: "corp", Groups: []string{"/Unknown"}}
	read := func() string {
		request := httptest.NewRequest(http.MethodGet, AuthSession, nil)
		request = request.WithContext(context.WithValue(request.Context(), auth.UserClaimsKey, claims))
		w := httptest.NewRecorder()
		h.GetSession(w, request)
		if w.Code != http.StatusOK {
			t.Fatalf("session status=%d body=%s", w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	if !strings.Contains(read(), `"mappedGroups":[]`) {
		t.Fatal("missing unmapped-groups empty state")
	}
	claims.Groups = []string{"/Ops"}
	if !strings.Contains(read(), `"mappedGroups":["ops"]`) {
		t.Fatal("mapped local group not returned")
	}
}
