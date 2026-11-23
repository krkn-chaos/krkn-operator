//go:build integration

package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	"github.com/krkn-chaos/krkn-operator/pkg/identity"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const keycloakIntegrationImage = "quay.io/keycloak/keycloak:26.8.2"

// TestKeycloakOIDCIntegration exercises a real authorization-code/PKCE login,
// operator session issuance, external group mapping, and Admin API discovery.
// Run with: go test -tags=integration ./internal/api -run '^TestKeycloakOIDCIntegration$' -count=1
func TestKeycloakOIDCIntegration(t *testing.T) {
	// Initialize controller-runtime's deferred logger before the SecretManager
	// uses it, so a passing integration test does not print a warning stack.
	log.SetLogger(logr.Discard())

	containerRuntime := os.Getenv("KRKN_KEYCLOAK_TEST_RUNTIME")
	if containerRuntime == "" {
		containerRuntime = "podman"
	}
	if containerRuntime != "podman" && containerRuntime != "docker" {
		t.Fatalf("unsupported Keycloak test container runtime %q", containerRuntime)
	}
	if _, err := exec.LookPath(containerRuntime); err != nil {
		t.Fatalf("%s is required for the Keycloak integration test: %v", containerRuntime, err)
	}
	baseURL, adminPassword := startIntegrationKeycloak(t, containerRuntime)
	admin := newIntegrationKeycloakAdmin(t, baseURL, adminPassword)
	const realm = "krkn-integration"
	const providerID = "integration-keycloak"
	const username = "integration-user"
	const password = "integration-password-123"
	const loginID = "krkn-login-integration"
	const loginSecret = "integration-login-secret"
	const groupsID = "krkn-groups-integration"
	const groupsSecret = "integration-groups-secret"
	issuer := baseURL + "/realms/" + realm
	loginPath := OIDCBasePath + "/" + providerID + "/login"
	callbackPath := OIDCBasePath + "/" + providerID + "/callback"

	var handler *Handler
	apiServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.OIDCRouter(w, r)
	}))
	defer apiServer.Close()

	admin.post("/admin/realms", map[string]any{"realm": realm, "enabled": true})
	loginClient := admin.createClient(realm, map[string]any{
		"clientId": loginID, "enabled": true, "protocol": "openid-connect",
		"publicClient": false, "secret": loginSecret, "standardFlowEnabled": true,
		"directAccessGrantsEnabled": false, "serviceAccountsEnabled": false,
		"redirectUris": []string{apiServer.URL + callbackPath},
	})
	groupsClient := admin.createClient(realm, map[string]any{
		"clientId": groupsID, "enabled": true, "protocol": "openid-connect",
		"publicClient": false, "secret": groupsSecret, "standardFlowEnabled": false,
		"directAccessGrantsEnabled": false, "serviceAccountsEnabled": true,
	})
	admin.post("/admin/realms/"+realm+"/clients/"+loginClient+"/protocol-mappers/models", map[string]any{
		"name": "groups", "protocol": "openid-connect", "protocolMapper": "oidc-group-membership-mapper",
		"config": map[string]string{
			"claim.name": "groups", "full.path": "true",
			"id.token.claim": "true", "access.token.claim": "true", "userinfo.token.claim": "true",
		},
	})
	admin.post("/admin/realms/"+realm+"/groups", map[string]any{"name": "krkn-ops"})
	group := admin.findByName("/admin/realms/"+realm+"/groups?search=krkn-ops&exact=true", "krkn-ops")
	admin.post("/admin/realms/"+realm+"/users", map[string]any{
		"username": username, "enabled": true, "email": "integration@example.invalid",
		"firstName": "Integration", "lastName": "User", "emailVerified": true,
		"credentials": []map[string]any{{"type": "password", "value": password, "temporary": false}},
	})
	user := admin.findByName("/admin/realms/"+realm+"/users?username="+username+"&exact=true", username)
	admin.put("/admin/realms/"+realm+"/users/"+user+"/groups/"+group, nil)

	// Keycloak 26.8 requires kc-admin-api on custom service clients.
	scope := admin.findByName("/admin/realms/"+realm+"/client-scopes", "kc-admin-api")
	admin.put("/admin/realms/"+realm+"/clients/"+groupsClient+"/default-client-scopes/"+scope, nil)
	serviceUser := admin.object("/admin/realms/" + realm + "/clients/" + groupsClient + "/service-account-user")
	realmManagement := admin.findByName("/admin/realms/"+realm+"/clients?clientId=realm-management", "realm-management")
	role := admin.object("/admin/realms/" + realm + "/clients/" + realmManagement + "/roles/view-users")
	admin.post("/admin/realms/"+realm+"/users/"+serviceUser["id"].(string)+"/role-mappings/clients/"+realmManagement, []map[string]any{role})

	directory, err := identity.NewKeycloakDirectory(providerID, issuer, groupsID, groupsSecret, nil)
	if err != nil {
		t.Fatal(err)
	}
	remoteGroups, err := directory.ListGroups(context.Background(), "krkn-ops", 0, 50)
	if err != nil || len(remoteGroups) != 1 || remoteGroups[0].ClaimValue != "/krkn-ops" {
		t.Fatalf("Keycloak group discovery: groups=%+v err=%v", remoteGroups, err)
	}

	provider := identityProviderConfig{
		ID: providerID, Type: "keycloak", DisplayName: "Integration Keycloak",
		Issuer: issuer, ClientID: loginID, ClientSecret: loginSecret,
		RedirectURL: apiServer.URL + callbackPath, Enabled: true, SessionEpoch: "integration-epoch",
	}
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	mappedGroup := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "mapped-ops", Namespace: "default"},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			ExternalGroups: []krknv1alpha1.ExternalGroupMapping{{Provider: providerID, Group: "/krkn-ops"}},
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://cluster.example": {Actions: []string{"view", "run"}},
			},
		},
	}
	k8s := fakeclient.NewClientBuilder().WithScheme(scheme).
		WithObjects(TestJWTSecret("default"), provider.secret("default"), mappedGroup).
		WithStatusSubresource(&krknv1alpha1.KrknUser{}).Build()
	manager, err := auth.NewTestSecretManager(k8s, "default")
	if err != nil {
		t.Fatal(err)
	}
	handler = NewHandler(assigningUIDClient{k8s}, nil, "default", "", manager, KrknAIServiceConfig{})
	defer handler.Shutdown()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	browser := apiServer.Client()
	browser.Timeout = 30 * time.Second
	browser.Jar = jar
	browser.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
		if req.URL.String() == apiServer.URL+"/" {
			return http.ErrUseLastResponse
		}
		return nil
	}
	loginPage, err := browser.Get(apiServer.URL + loginPath)
	if err != nil {
		t.Fatal(err)
	}
	if loginPage.StatusCode != http.StatusOK {
		t.Fatalf("Keycloak login page returned HTTP %d", loginPage.StatusCode)
	}
	action := keycloakLoginAction(t, loginPage.Body)
	_ = loginPage.Body.Close()
	form := url.Values{"username": {username}, "password": {password}}
	callback, err := browser.PostForm(action, form)
	if err != nil {
		t.Fatal(err)
	}
	defer callback.Body.Close()
	if callback.Request.URL.Path != callbackPath || callback.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(io.LimitReader(callback.Body, 512))
		t.Fatalf("OIDC callback: path=%s status=%d body=%q", callback.Request.URL.Path, callback.StatusCode, body)
	}
	var session *http.Cookie
	for _, cookie := range callback.Cookies() {
		if cookie.Name == auth.SessionCookieName {
			session = cookie
		}
	}
	if session == nil || !session.Secure || !session.HttpOnly {
		t.Fatal("operator did not issue a secure browser session")
	}
	generator, err := manager.GetTokenGenerator()
	if err != nil {
		t.Fatal(err)
	}
	claims, err := generator.ValidateToken(session.Value)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Provider != providerID || len(claims.Groups) != 1 || claims.Groups[0] != "/krkn-ops" {
		t.Fatalf("unexpected OIDC principal provider=%q groups=%v", claims.Provider, claims.Groups)
	}
	ctx := context.WithValue(context.Background(), auth.UserClaimsKey, claims)
	localGroups, err := groupauth.GetUserGroups(ctx, k8s, claims.UserID, "default")
	if err != nil || len(localGroups) != 1 || localGroups[0].Name != "mapped-ops" ||
		!groupauth.CanPerformAction(localGroups, "https://cluster.example", groupauth.ActionRun) {
		t.Fatalf("external group mapping failed: groups=%v err=%v", len(localGroups), err)
	}
	var provisioned krknv1alpha1.KrknUser
	if err := k8s.Get(ctx, client.ObjectKey{Name: sanitizeUsername(claims.UserID), Namespace: "default"}, &provisioned); err != nil {
		t.Fatal(err)
	}
	if provisioned.Spec.ExternalUsername != username || !provisioned.Status.Active {
		t.Fatal("OIDC user was not provisioned as an active external account")
	}
}

func startIntegrationKeycloak(t *testing.T, containerRuntime string) (string, string) {
	t.Helper()
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		t.Fatal(err)
	}
	name := "krkn-keycloak-it-" + base64.RawURLEncoding.EncodeToString(random)
	password := base64.RawURLEncoding.EncodeToString(random)
	image := os.Getenv("KRKN_KEYCLOAK_TEST_IMAGE")
	if image == "" {
		image = keycloakIntegrationImage
	}
	cmd := exec.Command(containerRuntime, "run", "--rm", "-d", "--name", name, "-p", "127.0.0.1::8080",
		"-e", "KC_BOOTSTRAP_ADMIN_USERNAME=admin", "-e", "KC_BOOTSTRAP_ADMIN_PASSWORD="+password,
		image, "start-dev")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("start isolated Keycloak: %v: %s", err, out)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if out, err := exec.CommandContext(ctx, containerRuntime, "stop", "-t", "1", name).CombinedOutput(); err != nil {
			t.Errorf("stop isolated Keycloak: %v: %s", err, out)
		}
	})
	out, err := exec.Command(containerRuntime, "port", name, "8080/tcp").Output()
	if err != nil {
		t.Fatal(err)
	}
	baseURL := "http://" + strings.TrimSpace(string(out))
	deadline := time.Now().Add(90 * time.Second)
	httpClient := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := httpClient.Get(baseURL + "/realms/master/.well-known/openid-configuration")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return baseURL, password
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("isolated Keycloak did not become ready within 90 seconds")
	return "", ""
}

type integrationKeycloakAdmin struct {
	t       *testing.T
	baseURL string
	token   string
	client  *http.Client
}

func newIntegrationKeycloakAdmin(t *testing.T, baseURL, password string) *integrationKeycloakAdmin {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	form := url.Values{"client_id": {"admin-cli"}, "grant_type": {"password"},
		"username": {"admin"}, "password": {password}}
	resp, err := client.PostForm(baseURL+"/realms/master/protocol/openid-connect/token", form)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Keycloak admin token returned HTTP %d", resp.StatusCode)
	}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		t.Fatal(err)
	}
	return &integrationKeycloakAdmin{t: t, baseURL: baseURL, token: token.AccessToken, client: client}
}

func (a *integrationKeycloakAdmin) request(method, path string, payload any) []byte {
	a.t.Helper()
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			a.t.Fatal(err)
		}
		body = strings.NewReader(string(encoded))
	}
	req, err := http.NewRequest(method, a.baseURL+path, body)
	if err != nil {
		a.t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		a.t.Fatal(err)
	}
	defer resp.Body.Close()
	result, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		a.t.Fatalf("Keycloak Admin API %s %s returned HTTP %d: %s", method, path, resp.StatusCode, result)
	}
	return result
}

func (a *integrationKeycloakAdmin) post(path string, payload any) {
	a.t.Helper()
	a.request(http.MethodPost, path, payload)
}

func (a *integrationKeycloakAdmin) put(path string, payload any) {
	a.t.Helper()
	a.request(http.MethodPut, path, payload)
}

func (a *integrationKeycloakAdmin) object(path string) map[string]any {
	a.t.Helper()
	var object map[string]any
	if err := json.Unmarshal(a.request(http.MethodGet, path, nil), &object); err != nil {
		a.t.Fatal(err)
	}
	return object
}

func (a *integrationKeycloakAdmin) findByName(path, name string) string {
	a.t.Helper()
	var items []map[string]any
	if err := json.Unmarshal(a.request(http.MethodGet, path, nil), &items); err != nil {
		a.t.Fatal(err)
	}
	for _, item := range items {
		if item["name"] == name || item["clientId"] == name || item["username"] == name {
			if id, ok := item["id"].(string); ok {
				return id
			}
		}
	}
	a.t.Fatalf("Keycloak object %q not found at %s", name, path)
	return ""
}

func (a *integrationKeycloakAdmin) createClient(realm string, representation map[string]any) string {
	a.t.Helper()
	a.post("/admin/realms/"+realm+"/clients", representation)
	id := representation["clientId"].(string)
	return a.findByName("/admin/realms/"+realm+"/clients?clientId="+url.QueryEscape(id), id)
}

var keycloakFormPattern = regexp.MustCompile(`(?is)<form\b[^>]*\bid="kc-form-login"[^>]*>`)
var keycloakActionPattern = regexp.MustCompile(`\baction="([^"]+)"`)

func keycloakLoginAction(t *testing.T, body io.Reader) string {
	t.Helper()
	page, err := io.ReadAll(io.LimitReader(body, 1<<20))
	if err != nil {
		t.Fatal(err)
	}
	form := keycloakFormPattern.Find(page)
	match := keycloakActionPattern.FindSubmatch(form)
	if len(match) != 2 {
		t.Fatal("Keycloak login form action not found")
	}
	return html.UnescapeString(string(match[1]))
}
