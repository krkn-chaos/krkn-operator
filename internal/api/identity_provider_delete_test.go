package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDeleteIdentityProviderRemovesOnlyItsMappingsAndAllowsRetryAndReuse(t *testing.T) {
	clearLegacyKeycloakEnvironment(t)
	h := setupUserTestHandler()
	defer h.Shutdown()
	createTestIdentityProvider(t, h, "corp")
	createTestGroup(t, h, "team", []krknv1alpha1.ExternalGroupMapping{
		{Provider: "corp", Group: "/Ops"},
		{Provider: "partner", Group: "/Ops"},
	})

	response := deleteIdentityProviderRequest(h, "corp")
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", response.Code, response.Body.String())
	}
	group := getTestGroup(t, h, "team")
	if len(group.Spec.ExternalGroups) != 1 || group.Spec.ExternalGroups[0].Provider != "partner" {
		t.Fatalf("delete removed the wrong group mappings: %+v", group.Spec.ExternalGroups)
	}
	var tombstone corev1.Secret
	if err := h.providerStore.reader.Get(context.Background(), client.ObjectKey{Name: identityProviderSecretName("corp"), Namespace: h.namespace}, &tombstone); err != nil {
		t.Fatalf("expected deletion tombstone: %v", err)
	}
	if tombstone.Labels[identityProviderTombstoneLabel] != "true" || len(tombstone.Data) != 1 || len(tombstone.Data["clientSecret"]) != 0 {
		t.Fatalf("provider credentials remain in tombstone: labels=%v dataKeys=%v", tombstone.Labels, tombstone.Data)
	}
	if _, err := h.providerStore.get(context.Background(), "corp"); !errors.Is(err, errIdentityProviderNotFound) {
		t.Fatalf("deleted provider is still available: %v", err)
	}

	if retryResponse := deleteIdentityProviderRequest(h, "corp"); retryResponse.Code != http.StatusNoContent {
		t.Fatalf("retry status=%d body=%s", retryResponse.Code, retryResponse.Body.String())
	}
	if createResponse := postTestIdentityProviderRequest(h, "corp"); createResponse.Code != http.StatusCreated {
		t.Fatalf("reuse ID status=%d body=%s", createResponse.Code, createResponse.Body.String())
	}
	if _, err := h.providerStore.get(context.Background(), "corp"); err != nil {
		t.Fatalf("recreated provider is unavailable: %v", err)
	}
}

func TestDeleteIdentityProviderOverrideDoesNotReactivateEnvironmentProvider(t *testing.T) {
	t.Setenv("KEYCLOAK_ISSUER_URL", "https://id.example/realms/environment")
	t.Setenv("KEYCLOAK_OIDC_CLIENT_ID", "environment-login")
	t.Setenv("KEYCLOAK_OIDC_CLIENT_SECRET", "environment-secret")
	t.Setenv("KEYCLOAK_OIDC_REDIRECT_URL", "https://operator.example/callback")
	h := setupUserTestHandler()
	defer h.Shutdown()
	createTestIdentityProvider(t, h, "keycloak")
	createTestGroup(t, h, "team", []krknv1alpha1.ExternalGroupMapping{{Provider: "keycloak", Group: "/Ops"}})

	response := deleteIdentityProviderRequest(h, "keycloak")
	if response.Code != http.StatusConflict {
		t.Fatalf("delete override status=%d body=%s", response.Code, response.Body.String())
	}
	var secret corev1.Secret
	if err := h.providerStore.reader.Get(context.Background(), client.ObjectKey{Name: identityProviderSecretName("keycloak"), Namespace: h.namespace}, &secret); err != nil {
		t.Fatalf("provider override was removed: %v", err)
	}
	group := getTestGroup(t, h, "team")
	if len(group.Spec.ExternalGroups) != 1 || group.Spec.ExternalGroups[0].Provider != "keycloak" {
		t.Fatalf("override deletion changed its mappings: %+v", group.Spec.ExternalGroups)
	}
}

func TestDeleteIdentityProviderIsRecoverableAfterGroupCleanupFailure(t *testing.T) {
	clearLegacyKeycloakEnvironment(t)
	h := setupUserTestHandler()
	defer h.Shutdown()
	createTestIdentityProvider(t, h, "corp")
	createTestGroup(t, h, "team", []krknv1alpha1.ExternalGroupMapping{{Provider: "corp", Group: "/Ops"}})
	failingClient := &failOnceGroupUpdateClient{Client: h.providerStore.client}
	h.providerStore.client = failingClient

	response := deleteIdentityProviderRequest(h, "corp")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("failed cleanup status=%d body=%s", response.Code, response.Body.String())
	}
	config, err := h.providerStore.get(context.Background(), "corp")
	if err != nil || config.Enabled || !config.Deleting {
		t.Fatalf("provider was not left safely disabled for retry: config=%+v err=%v", config, err)
	}
	if retryResponse := deleteIdentityProviderRequest(h, "corp"); retryResponse.Code != http.StatusNoContent {
		t.Fatalf("retry failed: status=%d body=%s", retryResponse.Code, retryResponse.Body.String())
	}
	if len(getTestGroup(t, h, "team").Spec.ExternalGroups) != 0 {
		t.Fatal("retry left provider mappings behind")
	}
}

func TestDeleteIdentityProviderIsRecoverableAfterTombstoneUpdateFailure(t *testing.T) {
	clearLegacyKeycloakEnvironment(t)
	h := setupUserTestHandler()
	defer h.Shutdown()
	createTestIdentityProvider(t, h, "corp")
	createTestGroup(t, h, "team", []krknv1alpha1.ExternalGroupMapping{{Provider: "corp", Group: "/Ops"}})
	h.providerStore.client = &failOnceTombstoneUpdateClient{Client: h.providerStore.client}

	response := deleteIdentityProviderRequest(h, "corp")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("failed tombstone update status=%d body=%s", response.Code, response.Body.String())
	}
	config, err := h.providerStore.get(context.Background(), "corp")
	if err != nil || config.Enabled || !config.Deleting {
		t.Fatalf("provider was not left safely disabled for retry: config=%+v err=%v", config, err)
	}
	if len(getTestGroup(t, h, "team").Spec.ExternalGroups) != 0 {
		t.Fatal("provider mappings were not removed before retry")
	}
	if retryResponse := deleteIdentityProviderRequest(h, "corp"); retryResponse.Code != http.StatusNoContent {
		t.Fatalf("retry failed: status=%d body=%s", retryResponse.Code, retryResponse.Body.String())
	}
}

func TestDeleteWaitsForInFlightGroupMappingUpdate(t *testing.T) {
	clearLegacyKeycloakEnvironment(t)
	h := setupUserTestHandler()
	defer h.Shutdown()
	createTestIdentityProvider(t, h, "corp")
	createTestGroup(t, h, "team", nil)
	blockingClient := &blockingGroupUpdateClient{Client: h.client, started: make(chan struct{}), release: make(chan struct{})}
	h.client = blockingClient
	tombstoneClient := &signalTombstoneUpdateClient{Client: h.providerStore.client, tombstoned: make(chan struct{})}
	h.providerStore.client = tombstoneClient

	updateDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		req := httptest.NewRequest(http.MethodPatch, GroupsPath+"/team", strings.NewReader(`{"externalGroups":[{"provider":"corp","group":"/Ops"}]}`)).WithContext(createAdminContext())
		response := httptest.NewRecorder()
		h.UpdateUserGroup(response, req)
		updateDone <- response
	}()
	select {
	case <-blockingClient.started:
	case <-time.After(3 * time.Second):
		t.Fatal("group mapping update did not reach persistence")
	}

	deleteDone := make(chan *httptest.ResponseRecorder, 1)
	deleteStarted := make(chan struct{})
	go func() {
		close(deleteStarted)
		deleteDone <- deleteIdentityProviderRequest(h, "corp")
	}()
	<-deleteStarted
	select {
	case <-tombstoneClient.tombstoned:
		t.Fatal("provider deletion passed an in-flight mapping update")
	case <-time.After(50 * time.Millisecond):
	}
	close(blockingClient.release)

	if updateResponse := <-updateDone; updateResponse.Code != http.StatusOK {
		t.Fatalf("mapping update status=%d body=%s", updateResponse.Code, updateResponse.Body.String())
	}
	if deleteResponse := <-deleteDone; deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("provider delete status=%d body=%s", deleteResponse.Code, deleteResponse.Body.String())
	}
	if len(getTestGroup(t, h, "team").Spec.ExternalGroups) != 0 {
		t.Fatal("provider deletion preserved an in-flight grant")
	}

	update := httptest.NewRequest(http.MethodPatch, GroupsPath+"/team", strings.NewReader(`{"externalGroups":[{"provider":"corp","group":"/Ops"}]}`)).WithContext(createAdminContext())
	response := httptest.NewRecorder()
	h.UpdateUserGroup(response, update)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("mapping to deleted provider status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestDeleteIdentityProviderUnknownIDReturnsNotFound(t *testing.T) {
	clearLegacyKeycloakEnvironment(t)
	h := setupUserTestHandler()
	defer h.Shutdown()
	response := deleteIdentityProviderRequest(h, "missing")
	if response.Code != http.StatusNotFound {
		t.Fatalf("unknown provider status=%d body=%s", response.Code, response.Body.String())
	}
}

func clearLegacyKeycloakEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{"KEYCLOAK_ISSUER_URL", "KEYCLOAK_OIDC_CLIENT_ID", "KEYCLOAK_OIDC_CLIENT_SECRET", "KEYCLOAK_OIDC_REDIRECT_URL", "KEYCLOAK_ADMIN_CLIENT_ID", "KEYCLOAK_ADMIN_CLIENT_SECRET"} {
		t.Setenv(key, "")
	}
}

func createTestIdentityProvider(t *testing.T, h *Handler, id string) {
	t.Helper()
	c := identityProviderConfig{ID: id, Type: "keycloak", DisplayName: "Test provider",
		Issuer: "https://id.example/realms/test", ClientID: "login", ClientSecret: "client-secret",
		RedirectURL: "https://operator.example" + OIDCBasePath + "/" + id + "/callback", Enabled: true, SessionEpoch: "epoch"}
	if err := h.providerStore.client.Create(context.Background(), c.secret(h.namespace)); err != nil {
		t.Fatal(err)
	}
}

func createTestGroup(t *testing.T, h *Handler, name string, mappings []krknv1alpha1.ExternalGroupMapping) {
	t.Helper()
	group := &krknv1alpha1.KrknUserGroup{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: h.namespace}, Spec: krknv1alpha1.KrknUserGroupSpec{
		Name: name, ExternalGroups: mappings,
	}}
	if err := h.client.Create(context.Background(), group); err != nil {
		t.Fatal(err)
	}
}

func getTestGroup(t *testing.T, h *Handler, name string) *krknv1alpha1.KrknUserGroup {
	t.Helper()
	var group krknv1alpha1.KrknUserGroup
	if err := h.providerStore.reader.Get(context.Background(), client.ObjectKey{Name: name, Namespace: h.namespace}, &group); err != nil {
		t.Fatal(err)
	}
	return &group
}

func deleteIdentityProviderRequest(h *Handler, id string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodDelete, IdentityProvidersPath+"/"+id, nil).WithContext(createAdminContext())
	response := httptest.NewRecorder()
	h.IdentityProviders(response, request)
	return response
}

func postTestIdentityProviderRequest(h *Handler, id string) *httptest.ResponseRecorder {
	body := `{"id":"` + id + `","type":"keycloak","displayName":"Test provider",` +
		`"issuer":"https://id.example/realms/test","clientId":"login","clientSecret":"client-secret",` +
		`"redirectUrl":"https://operator.example` + OIDCBasePath + `/` + id + `/callback","enabled":true}`
	request := httptest.NewRequest(http.MethodPost, IdentityProvidersPath, strings.NewReader(body)).WithContext(createAdminContext())
	response := httptest.NewRecorder()
	h.IdentityProviders(response, request)
	return response
}

type failOnceGroupUpdateClient struct {
	client.Client
	mu     sync.Mutex
	failed bool
}

func (c *failOnceGroupUpdateClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if _, ok := obj.(*krknv1alpha1.KrknUserGroup); ok {
		c.mu.Lock()
		defer c.mu.Unlock()
		if !c.failed {
			c.failed = true
			return errors.New("simulated group update failure")
		}
	}
	return c.Client.Update(ctx, obj, opts...)
}

type blockingGroupUpdateClient struct {
	client.Client
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (c *blockingGroupUpdateClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if _, ok := obj.(*krknv1alpha1.KrknUserGroup); ok {
		c.once.Do(func() {
			close(c.started)
			<-c.release
		})
	}
	return c.Client.Update(ctx, obj, opts...)
}

type signalTombstoneUpdateClient struct {
	client.Client
	tombstoned chan struct{}
	once       sync.Once
}

func (c *signalTombstoneUpdateClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if secret, ok := obj.(*corev1.Secret); ok && secret.Labels[identityProviderTombstoneLabel] == "true" {
		c.once.Do(func() { close(c.tombstoned) })
	}
	return c.Client.Update(ctx, obj, opts...)
}

type failOnceTombstoneUpdateClient struct {
	client.Client
	mu     sync.Mutex
	failed bool
}

func (c *failOnceTombstoneUpdateClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if secret, ok := obj.(*corev1.Secret); ok && secret.Labels[identityProviderTombstoneLabel] == "true" {
		c.mu.Lock()
		defer c.mu.Unlock()
		if !c.failed {
			c.failed = true
			return errors.New("simulated tombstone update failure")
		}
	}
	return c.Client.Update(ctx, obj, opts...)
}
