package api

import (
	"context"
	"errors"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/identity"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestOIDCProvisioningRecoversAfterInitialStatusFailure(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	failFirstActivation := true
	k8s := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&krknv1alpha1.KrknUser{}).
		WithInterceptorFuncs(interceptor.Funcs{SubResourceUpdate: func(ctx context.Context, c client.Client,
			subResource string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
			if subResource == "status" && failFirstActivation {
				if _, ok := obj.(*krknv1alpha1.KrknUser); ok {
					failFirstActivation = false
					return errors.New("temporary status write failure")
				}
			}
			return c.SubResource(subResource).Update(ctx, obj, opts...)
		}}).Build()
	h := &Handler{client: assigningUIDClient{k8s}, identityReader: k8s, namespace: "default"}
	key := identity.ExternalIdentity{Provider: "corp", Issuer: "https://keycloak.example/realms/corp", Subject: "subject-1"}
	principal := &identity.OIDCPrincipal{Identity: key, Username: "alice"}
	if _, err := h.ensureExternalUser(t.Context(), key, principal); err == nil {
		t.Fatal("first activation unexpectedly succeeded")
	}
	user, err := h.ensureExternalUser(t.Context(), key, principal)
	if err != nil || !user.Status.Active || user.Status.Created.IsZero() {
		t.Fatalf("retry did not activate initial OIDC account: user=%+v err=%v", user, err)
	}
	user.Status.Active = false
	if err := k8s.Status().Update(t.Context(), user); err != nil {
		t.Fatal(err)
	}
	if _, err := h.ensureExternalUser(t.Context(), key, principal); err == nil {
		t.Fatal("admin-disabled OIDC account was reactivated")
	}
}
