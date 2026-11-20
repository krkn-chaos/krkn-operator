package groupauth

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

func TestExternalGroupsMapManyClaimsToOneLocalGroup(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	mapped := &krknv1alpha1.KrknUserGroup{ObjectMeta: metav1.ObjectMeta{Name: "team", Namespace: "default"},
		Spec: krknv1alpha1.KrknUserGroupSpec{ExternalGroups: []krknv1alpha1.ExternalGroupMapping{
			{Provider: "keycloak", Group: "/eng"}, {Provider: "keycloak", Group: "/ops"}}}}
	unmapped := &krknv1alpha1.KrknUserGroup{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "default"},
		Spec: krknv1alpha1.KrknUserGroupSpec{ExternalGroups: []krknv1alpha1.ExternalGroupMapping{{Provider: "dex", Group: "/eng"}}}}
	k8s := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(mapped, unmapped).Build()
	claims := &auth.Claims{UserID: "oidc-user", Provider: "keycloak", Groups: []string{"/ops", "/unknown"}}
	ctx := context.WithValue(context.Background(), auth.UserClaimsKey, claims)
	groups, err := GetUserGroups(ctx, k8s, claims.UserID, "default")
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].Name != "team" {
		t.Fatalf("mapped groups=%v", groups)
	}
	claims.Groups = []string{"/unknown"}
	groups, err = GetUserGroups(ctx, k8s, claims.UserID, "default")
	if err != nil || len(groups) != 0 {
		t.Fatalf("unmapped groups=%v err=%v", groups, err)
	}
	if _, err := GetUserGroups(ctx, k8s, "different-user", "default"); err == nil {
		t.Fatal("identity mismatch accepted")
	}
}

func TestExternalGroupPathIsScopedByProviderInstance(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	mapped := &krknv1alpha1.KrknUserGroup{ObjectMeta: metav1.ObjectMeta{Name: "team", Namespace: "default"},
		Spec: krknv1alpha1.KrknUserGroupSpec{ExternalGroups: []krknv1alpha1.ExternalGroupMapping{
			{Provider: "corp", Group: "/Ops", GroupID: "remote-1"}}}}
	k8s := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(mapped).Build()
	for _, test := range []struct {
		provider string
		want     int
	}{{"corp", 1}, {"partner", 0}} {
		claims := &auth.Claims{UserID: "oidc-user", Provider: test.provider, Groups: []string{"/Ops"}}
		ctx := context.WithValue(context.Background(), auth.UserClaimsKey, claims)
		groups, err := GetUserGroups(ctx, k8s, claims.UserID, "default")
		if err != nil || len(groups) != test.want {
			t.Fatalf("provider=%s groups=%v err=%v", test.provider, groups, err)
		}
	}
}
