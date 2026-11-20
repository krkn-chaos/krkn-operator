package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/identity"
)

func oidcTestAccount(t *testing.T, key identity.ExternalIdentity, uid string) (*krknv1alpha1.KrknUser, *auth.Claims) {
	t.Helper()
	userID, err := key.UserID()
	if err != nil {
		t.Fatal(err)
	}
	user := &krknv1alpha1.KrknUser{ObjectMeta: metav1.ObjectMeta{
		Name: sanitizeUsername(userID), Namespace: "default", UID: types.UID(uid),
		Labels: map[string]string{UserAccountLabel: "true", AdminRoleLabel: "user"}},
		Spec: krknv1alpha1.KrknUserSpec{UserID: userID, Role: "user",
			AuthProvider: key.Provider, ExternalIssuer: key.Issuer, ExternalSubject: key.Subject,
			ExternalUsername: "tullio", ExternalEmail: "tullio@example.com"},
		Status: krknv1alpha1.KrknUserStatus{Active: true}}
	claims := &auth.Claims{UserID: userID, Provider: key.Provider,
		ExternalIssuer: key.Issuer, ExternalSubject: key.Subject, AccountUID: uid}
	return user, claims
}

func TestOIDCDeleteAndBlacklist(t *testing.T) {
	ctx := context.Background()
	key := identity.ExternalIdentity{Provider: "keycloak", Issuer: "https://id.example/realms/main", Subject: "subject-42"}
	user, claims := oidcTestAccount(t, key, "first-uid")
	h := setupUserTestHandler(user)
	defer h.Shutdown()
	checker := newK8sUserStatusChecker(h.client, "default")
	if active, err := checker.IsExternalIdentityActive(ctx, claims); err != nil || !active {
		t.Fatalf("initial account active=%v err=%v", active, err)
	}

	req := httptest.NewRequest(http.MethodDelete, UsersPath+"/"+claims.UserID+"?blacklist=true", nil).WithContext(createAdminContext())
	w := httptest.NewRecorder()
	h.DeleteUser(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", w.Code, w.Body.String())
	}
	name, _ := key.BlockSecretName()
	var block corev1.Secret
	if err := h.client.Get(ctx, client.ObjectKey{Name: name, Namespace: "default"}, &block); err != nil {
		t.Fatal(err)
	}
	if got := string(block.Data["username"]); got != "tullio" {
		t.Fatalf("username=%q", got)
	}
	if got := string(block.Data["subject"]); got != key.Subject {
		t.Fatalf("subject=%q", got)
	}
	if active, err := checker.IsExternalIdentityActive(ctx, claims); err != nil || active {
		t.Fatalf("blocked account active=%v err=%v", active, err)
	}
	if _, err := h.ensureExternalUser(ctx, key, &identity.OIDCPrincipal{Username: "tullio"}); err == nil {
		t.Fatal("blocked identity was provisioned")
	}

	list := httptest.NewRecorder()
	h.IdentityBlocks(list, httptest.NewRequest(http.MethodGet, IdentityBlocksPath, nil).WithContext(createAdminContext()))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), "tullio") {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}
	filtered := httptest.NewRecorder()
	h.IdentityBlocks(filtered, httptest.NewRequest(http.MethodGet, IdentityBlocksPath+"?authProvider=other", nil).WithContext(createAdminContext()))
	if filtered.Code != http.StatusOK || strings.Contains(filtered.Body.String(), "tullio") {
		t.Fatalf("filtered blocks status=%d body=%s", filtered.Code, filtered.Body.String())
	}
	remove := httptest.NewRecorder()
	h.IdentityBlocks(remove, httptest.NewRequest(http.MethodDelete, IdentityBlocksPath+"/"+name, nil).WithContext(createAdminContext()))
	if remove.Code != http.StatusNoContent {
		t.Fatalf("unblock status=%d", remove.Code)
	}
	if _, err := h.getOIDCBlock(ctx, key); err != nil {
		t.Fatal(err)
	}
}

func TestOIDCDeleteInvalidatesOldSessionAfterRecreate(t *testing.T) {
	ctx := context.Background()
	key := identity.ExternalIdentity{Provider: "keycloak", Issuer: "https://id.example/realms/main", Subject: "subject-42"}
	user, claims := oidcTestAccount(t, key, "first-uid")
	h := setupUserTestHandler(user)
	defer h.Shutdown()
	checker := newK8sUserStatusChecker(h.client, "default")
	w := httptest.NewRecorder()
	h.DeleteUser(w, httptest.NewRequest(http.MethodDelete, UsersPath+"/"+claims.UserID, nil).WithContext(createAdminContext()))
	if w.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", w.Code, w.Body.String())
	}
	if block, err := h.getOIDCBlock(ctx, key); err != nil || block != nil {
		t.Fatalf("ordinary delete created block: %v %v", block, err)
	}
	user.UID = types.UID("second-uid")
	user.ResourceVersion = ""
	if err := h.client.Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	if active, err := checker.IsExternalIdentityActive(ctx, claims); err != nil || active {
		t.Fatalf("old session revived: active=%v err=%v", active, err)
	}
	claims.AccountUID = "second-uid"
	if active, err := checker.IsExternalIdentityActive(ctx, claims); err != nil || !active {
		t.Fatalf("new session active=%v err=%v", active, err)
	}
}

func TestOIDCBlacklistCollisionKeepsAccount(t *testing.T) {
	ctx := context.Background()
	key := identity.ExternalIdentity{Provider: "keycloak", Issuer: "https://id.example/realms/main", Subject: "subject-42"}
	user, claims := oidcTestAccount(t, key, "first-uid")
	name, _ := key.BlockSecretName()
	collision := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"}, Data: map[string][]byte{"subject": []byte("different")}}
	h := setupUserTestHandler(user, collision)
	defer h.Shutdown()
	w := httptest.NewRecorder()
	h.DeleteUser(w, httptest.NewRequest(http.MethodDelete, UsersPath+"/"+claims.UserID+"?blacklist=true", nil).WithContext(createAdminContext()))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var remaining krknv1alpha1.KrknUser
	if err := h.client.Get(ctx, client.ObjectKey{Name: user.Name, Namespace: "default"}, &remaining); err != nil {
		t.Fatal("account deleted despite block failure:", err)
	}
}

func TestOIDCDisableRejectsExistingSession(t *testing.T) {
	ctx := context.Background()
	key := identity.ExternalIdentity{Provider: "keycloak", Issuer: "https://id.example/realms/main", Subject: "subject-42"}
	user, claims := oidcTestAccount(t, key, "first-uid")
	h := setupUserTestHandler(user)
	defer h.Shutdown()
	checker := newK8sUserStatusChecker(h.client, "default")
	req := httptest.NewRequest(http.MethodPatch, UsersPath+"/"+claims.UserID, strings.NewReader(`{"active":false}`))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()
	h.UpdateUser(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("disable status=%d body=%s", w.Code, w.Body.String())
	}
	if active, err := checker.IsExternalIdentityActive(ctx, claims); err != nil || active {
		t.Fatalf("disabled session active=%v err=%v", active, err)
	}
}

func TestOIDCCheckerUsesUncachedReaderForBlock(t *testing.T) {
	ctx := context.Background()
	key := identity.ExternalIdentity{Provider: "keycloak", Issuer: "https://id.example/realms/main", Subject: "subject-42"}
	user, claims := oidcTestAccount(t, key, "first-uid")
	blockName, _ := key.BlockSecretName()
	block := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: blockName, Namespace: "default",
		Labels: map[string]string{oidcBlockLabel: "true"}},
		Data: map[string][]byte{"provider": []byte(key.Provider), "issuer": []byte(key.Issuer), "subject": []byte(key.Subject)}}
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	staleCache := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(user).Build()
	direct := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(user, block).Build()
	checker := newK8sUserStatusChecker(staleCache, "default")
	checker.identityReader = direct
	active, err := checker.IsExternalIdentityActive(ctx, claims)
	if err != nil || active {
		t.Fatalf("cached read bypassed block: active=%v err=%v", active, err)
	}
}
