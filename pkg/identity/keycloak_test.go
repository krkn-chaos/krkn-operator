package identity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKeycloakDirectoryListsGroupsWithClaimPaths(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/main/protocol/openid-connect/token":
			if r.Method != http.MethodPost {
				t.Errorf("token method=%s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"service-token","token_type":"Bearer","expires_in":3600}`))
		case "/admin/realms/main/groups":
			if r.Header.Get("Authorization") != "Bearer service-token" {
				t.Errorf("group authorization=%q", r.Header.Get("Authorization"))
			}
			q := r.URL.Query()
			if q.Get("search") != "ops" || q.Get("first") != "2" || q.Get("max") != "10" {
				t.Errorf("query=%v", q)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"parent","name":"Team","path":"/Team","subGroups":[{"id":"child","name":"Ops","path":"/Team/Ops"}]}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	directory, err := NewKeycloakDirectory("keycloak", server.URL+"/realms/main", "id", "secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	groups, err := directory.ListGroups(context.Background(), "ops", 2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 || groups[1].ID != "child" || groups[1].ClaimValue != "/Team/Ops" {
		t.Fatalf("groups=%v", groups)
	}
}

func TestExternalIdentityKeyUsesIssuerAndSubject(t *testing.T) {
	key := ExternalIdentity{Provider: "keycloak", Issuer: "https://id.example/realms/main", Subject: "42"}
	id, err := key.UserID()
	if err != nil {
		t.Fatal(err)
	}
	block, err := key.BlockSecretName()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(id, "oidc-") || !strings.HasPrefix(block, "krkn-oidc-block-") {
		t.Fatalf("id=%s block=%s", id, block)
	}
	other := key
	other.Subject = "43"
	otherID, _ := other.UserID()
	if otherID == id {
		t.Fatal("different subject produced same ID")
	}
	other = key
	other.Issuer = "https://id.example/realms/other"
	otherID, _ = other.UserID()
	if otherID == id {
		t.Fatal("different issuer produced same ID")
	}
}
