package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/identity"
)

const oidcBlockLabel = "krkn.krkn-chaos.dev/oidc-block"

// OIDCBlock is the admin-facing representation of a blocked external identity.
// Username and email are snapshots for display; only issuer and subject match.
type OIDCBlock struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Issuer   string `json:"issuer"`
	Subject  string `json:"subject"`
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
}

func (h *Handler) getOIDCBlock(ctx context.Context, key identity.ExternalIdentity) (*corev1.Secret, error) {
	name, err := key.BlockSecretName()
	if err != nil {
		return nil, err
	}
	var secret corev1.Secret
	if err := h.identityReader.Get(ctx, client.ObjectKey{Name: name, Namespace: h.namespace}, &secret); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read OIDC block: %w", err)
	}
	if secret.Labels[oidcBlockLabel] != "true" || string(secret.Data["provider"]) != key.Provider ||
		string(secret.Data["issuer"]) != key.Issuer || string(secret.Data["subject"]) != key.Subject {
		return nil, fmt.Errorf("OIDC block key collision or corrupt Secret")
	}
	return &secret, nil
}

// createOIDCBlock records the deny decision before deleting an account.
// An existing matching block is accepted so retries are safe.
func (h *Handler) createOIDCBlock(ctx context.Context, key identity.ExternalIdentity, username, email string) error {
	if existing, err := h.getOIDCBlock(ctx, key); err != nil {
		return err
	} else if existing != nil {
		return nil
	}
	name, _ := key.BlockSecretName()
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: h.namespace,
		Labels: map[string]string{oidcBlockLabel: "true", "app.kubernetes.io/name": "krkn-operator"}},
		Type: corev1.SecretTypeOpaque, Data: map[string][]byte{
			"provider": []byte(key.Provider), "issuer": []byte(key.Issuer), "subject": []byte(key.Subject),
			"username": []byte(username), "email": []byte(email),
		}}
	if err := h.client.Create(ctx, secret); err != nil {
		if apierrors.IsAlreadyExists(err) {
			if existing, checkErr := h.getOIDCBlock(ctx, key); checkErr == nil && existing != nil {
				return nil
			}
		}
		return fmt.Errorf("create OIDC block: %w", err)
	}
	return nil
}

// IdentityBlocks handles GET /api/v1/identity-blocks and
// DELETE /api/v1/identity-blocks/{block-secret-name}. Only admins can access it.
func (h *Handler) IdentityBlocks(w http.ResponseWriter, r *http.Request) {
	if !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{Error: "forbidden", Message: "Admin privileges required"})
		return
	}
	if r.URL.Path == IdentityBlocksPath && r.Method == http.MethodGet {
		var secrets corev1.SecretList
		if err := h.identityReader.List(r.Context(), &secrets, client.InNamespace(h.namespace),
			client.MatchingLabels{oidcBlockLabel: "true"}); err != nil {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to list blocks"})
			return
		}
		blocks := make([]OIDCBlock, 0, len(secrets.Items))
		for _, secret := range secrets.Items {
			blocks = append(blocks, OIDCBlock{ID: secret.Name,
				Provider: string(secret.Data["provider"]), Issuer: string(secret.Data["issuer"]),
				Subject: string(secret.Data["subject"]), Username: string(secret.Data["username"]),
				Email: string(secret.Data["email"])})
		}
		writeJSON(w, http.StatusOK, struct {
			Blocks []OIDCBlock `json:"blocks"`
		}{Blocks: blocks})
		return
	}
	if strings.HasPrefix(r.URL.Path, IdentityBlocksPath+"/") && r.Method == http.MethodDelete {
		name := strings.TrimPrefix(r.URL.Path, IdentityBlocksPath+"/")
		if !strings.HasPrefix(name, "krkn-oidc-block-") || strings.Contains(name, "/") {
			http.NotFound(w, r)
			return
		}
		var secret corev1.Secret
		if err := h.identityReader.Get(r.Context(), client.ObjectKey{Name: name, Namespace: h.namespace}, &secret); err != nil {
			if apierrors.IsNotFound(err) {
				http.NotFound(w, r)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to read block"})
			return
		}
		if secret.Labels[oidcBlockLabel] != "true" {
			http.NotFound(w, r)
			return
		}
		if err := h.client.Delete(r.Context(), &secret); err != nil {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to remove block"})
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.NotFound(w, r)
}
