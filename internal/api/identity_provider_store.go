package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const identityProviderLabel = "krkn.krkn-chaos.dev/identity-provider"

var providerIDPattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

var errIdentityProviderNotFound = errors.New("identity provider not found")

// identityProviderConfig is the operator's private, Secret-backed OIDC config.
// ID is an immutable instance identifier, not a provider product name. Issuer
// is immutable after creation so existing group grants cannot move to a realm
// with an identical group path.
type identityProviderConfig struct {
	ID                string
	Type              string
	DisplayName       string
	Issuer            string
	ClientID          string
	ClientSecret      string
	RedirectURL       string
	AdminClientID     string
	AdminClientSecret string
	Enabled           bool
	SessionEpoch      string
	Source            string
}

type identityProviderStore struct {
	client    client.Client
	reader    client.Reader
	namespace string
}

func newIdentityProviderStore(k8s client.Client, namespace string) *identityProviderStore {
	return &identityProviderStore{client: k8s, reader: k8s, namespace: namespace}
}

func identityProviderSecretName(id string) string { return "krkn-identity-provider-" + id }

func validProviderID(id string) bool { return len(id) <= 63 && providerIDPattern.MatchString(id) }

func secretProvider(secret *corev1.Secret) (identityProviderConfig, error) {
	if secret.Labels[identityProviderLabel] != "true" {
		return identityProviderConfig{}, fmt.Errorf("identity provider Secret has invalid label")
	}
	id := strings.TrimPrefix(secret.Name, "krkn-identity-provider-")
	if !validProviderID(id) || secret.Name != identityProviderSecretName(id) {
		return identityProviderConfig{}, fmt.Errorf("identity provider Secret has invalid name")
	}
	data := secret.Data
	config := identityProviderConfig{ID: id, Type: string(data["type"]), DisplayName: string(data["displayName"]),
		Issuer: string(data["issuer"]), ClientID: string(data["clientId"]), ClientSecret: string(data["clientSecret"]),
		RedirectURL: string(data["redirectUrl"]), AdminClientID: string(data["adminClientId"]),
		AdminClientSecret: string(data["adminClientSecret"]), Enabled: string(data["enabled"]) == "true",
		SessionEpoch: string(data["sessionEpoch"]), Source: "secret"}
	if err := validateProviderConfig(config); err != nil {
		return identityProviderConfig{}, fmt.Errorf("invalid identity provider %q: %w", id, err)
	}
	return config, nil
}

func (c identityProviderConfig) secret(namespace string) *corev1.Secret {
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: identityProviderSecretName(c.ID), Namespace: namespace,
		Labels: map[string]string{identityProviderLabel: "true", "app.kubernetes.io/name": "krkn-operator"}},
		Type: corev1.SecretTypeOpaque, Data: map[string][]byte{
			"type": []byte(c.Type), "displayName": []byte(c.DisplayName), "issuer": []byte(c.Issuer),
			"clientId": []byte(c.ClientID), "clientSecret": []byte(c.ClientSecret), "redirectUrl": []byte(c.RedirectURL),
			"adminClientId": []byte(c.AdminClientID), "adminClientSecret": []byte(c.AdminClientSecret),
			"enabled":      []byte(fmt.Sprint(c.Enabled)),
			"sessionEpoch": []byte(c.SessionEpoch),
		}}
}

func legacyKeycloakConfig() (identityProviderConfig, bool) {
	c := identityProviderConfig{ID: "keycloak", Type: "keycloak", DisplayName: "Keycloak",
		Issuer: os.Getenv("KEYCLOAK_ISSUER_URL"), ClientID: os.Getenv("KEYCLOAK_OIDC_CLIENT_ID"),
		ClientSecret: os.Getenv("KEYCLOAK_OIDC_CLIENT_SECRET"), RedirectURL: os.Getenv("KEYCLOAK_OIDC_REDIRECT_URL"),
		AdminClientID: os.Getenv("KEYCLOAK_ADMIN_CLIENT_ID"), AdminClientSecret: os.Getenv("KEYCLOAK_ADMIN_CLIENT_SECRET"),
		Enabled: true, Source: "environment"}
	return c, c.Issuer != "" && c.ClientID != "" && c.ClientSecret != "" && c.RedirectURL != ""
}

func (s *identityProviderStore) get(ctx context.Context, id string) (identityProviderConfig, error) {
	if !validProviderID(id) {
		return identityProviderConfig{}, fmt.Errorf("invalid identity provider ID")
	}
	var secret corev1.Secret
	err := s.reader.Get(ctx, client.ObjectKey{Name: identityProviderSecretName(id), Namespace: s.namespace}, &secret)
	if err == nil {
		return secretProvider(&secret)
	}
	if !apierrors.IsNotFound(err) {
		return identityProviderConfig{}, fmt.Errorf("read identity provider: %w", err)
	}
	if id == "keycloak" {
		if config, ok := legacyKeycloakConfig(); ok {
			return config, nil
		}
	}
	return identityProviderConfig{}, fmt.Errorf("%w: %s", errIdentityProviderNotFound, id)
}

func (s *identityProviderStore) list(ctx context.Context) ([]identityProviderConfig, error) {
	var secrets corev1.SecretList
	if err := s.reader.List(ctx, &secrets, client.InNamespace(s.namespace), client.MatchingLabels{identityProviderLabel: "true"}); err != nil {
		return nil, fmt.Errorf("list identity providers: %w", err)
	}
	configs := make([]identityProviderConfig, 0, len(secrets.Items)+1)
	legacyFound := false
	for i := range secrets.Items {
		c, err := secretProvider(&secrets.Items[i])
		if err != nil {
			return nil, err
		}
		if c.ID == "keycloak" {
			legacyFound = true
		}
		configs = append(configs, c)
	}
	if !legacyFound {
		if c, ok := legacyKeycloakConfig(); ok {
			configs = append(configs, c)
		}
	}
	return configs, nil
}

func validIdentityURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	return u.Scheme == "http" && (u.Hostname() == "localhost" || net.ParseIP(u.Hostname()) != nil && net.ParseIP(u.Hostname()).IsLoopback())
}

func validateProviderConfig(c identityProviderConfig) error {
	if !validProviderID(c.ID) || c.Type != "keycloak" || strings.TrimSpace(c.DisplayName) == "" {
		return fmt.Errorf("valid ID, type keycloak and displayName are required")
	}
	if !validIdentityURL(c.Issuer) || !validIdentityURL(c.RedirectURL) {
		return fmt.Errorf("issuer and redirectUrl must be HTTPS URLs (HTTP loopback is allowed for tests)")
	}
	if !strings.HasSuffix(c.RedirectURL, OIDCBasePath+"/"+c.ID+"/callback") {
		return fmt.Errorf("redirectUrl must end in the provider callback path")
	}
	if c.ClientID == "" || c.ClientSecret == "" {
		return fmt.Errorf("OIDC clientId and clientSecret are required")
	}
	if c.Source == "secret" && c.SessionEpoch == "" {
		return fmt.Errorf("session epoch is required")
	}
	if (c.AdminClientID == "") != (c.AdminClientSecret == "") {
		return fmt.Errorf("both adminClientId and adminClientSecret are required for group discovery")
	}
	return nil
}
