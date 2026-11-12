package registry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/krkn-chaos/krknctl/pkg/provider/models"
	"github.com/stretchr/testify/require"
)

func TestResolvePublicTagDigestSelectsActiveExactTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/repository/test/tag/", r.URL.Path)
		require.Equal(t, "workload-rollback", r.URL.Query().Get("specificTag"))
		require.Equal(t, "true", r.URL.Query().Get("onlyActiveTags"))
		_, _ = w.Write([]byte(`{"tags":[{"name":"other","manifest_digest":"sha256:wrong"},{"name":"workload-rollback","manifest_digest":"sha256:old","end_ts":1},{"name":"workload-rollback","manifest_digest":"sha256:current","end_ts":null}]}`))
	}))
	defer server.Close()
	digest, err := ResolvePublicTagDigest(t.Context(), server.URL+"/api/v1/repository/test", "workload-rollback")
	require.NoError(t, err)
	require.Equal(t, "sha256:current", digest)
}

func TestResolvePublicTagDigestMissingAndCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tags":[]}`))
	}))
	defer server.Close()
	_, err := ResolvePublicTagDigest(t.Context(), server.URL+"/api/v1/repository/test", "workload-rollback")
	require.ErrorIs(t, err, ErrTagNotFound)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = ResolvePublicTagDigest(ctx, server.URL+"/api/v1/repository/test", "workload-rollback")
	require.Error(t, err)
}

func TestResolvePrivateTagDigestWithAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v2/scenarios/manifests/workload-rollback", r.URL.Path)
		require.Contains(t, r.Header.Get("Accept"), "application/vnd.oci.image.index.v1+json")
		user, password, ok := r.BasicAuth()
		require.True(t, ok)
		require.Equal(t, "test-user", user)
		require.Equal(t, "test-password", password)
		w.Header().Set("Docker-Content-Digest", "sha256:private")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	user, password := "test-user", "test-password"
	config := &models.RegistryV2{RegistryURL: strings.TrimPrefix(server.URL, "http://"), ScenarioRepository: "scenarios", Insecure: true, Username: &user, Password: &password}
	digest, err := ResolvePrivateTagDigest(t.Context(), "workload-rollback", config)
	require.NoError(t, err)
	require.Equal(t, "sha256:private", digest)
}

func TestResolvePrivateTagDigestMissing(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	config := &models.RegistryV2{RegistryURL: strings.TrimPrefix(server.URL, "http://"), ScenarioRepository: "scenarios", Insecure: true}
	_, err := ResolvePrivateTagDigest(t.Context(), "workload-rollback", config)
	require.True(t, errors.Is(err, ErrTagNotFound), err)
}
