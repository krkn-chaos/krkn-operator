package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krkn-chaos/krknctl/pkg/provider/models"
	"github.com/stretchr/testify/require"
)

const testOCIManifest = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},"layers":[]}`

func testManifestDigest() string {
	digest := sha256.Sum256([]byte(testOCIManifest))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func writeTestManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
	w.Header().Set("Docker-Content-Digest", testManifestDigest())
	w.Header().Set("Content-Length", strconv.Itoa(len(testOCIManifest)))
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write([]byte(testOCIManifest))
	}
}

func testPrivateRegistry(serverURL string, insecure bool, username, password string) *models.RegistryV2 {
	registryURL := strings.TrimPrefix(strings.TrimPrefix(serverURL, "http://"), "https://")
	registry := &models.RegistryV2{
		RegistryURL:        registryURL,
		ScenarioRepository: "scenarios",
		Insecure:           insecure,
	}
	if username != "" {
		registry.Username = &username
		registry.Password = &password
	}
	return registry
}

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

func TestResolvePrivateTagDigestWithBasicChallenge(t *testing.T) {
	challenged := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/" && r.URL.Path != "/v2/scenarios/manifests/workload-rollback" {
			http.NotFound(w, r)
			return
		}
		user, password, ok := r.BasicAuth()
		if !ok {
			select {
			case challenged <- struct{}{}:
			default:
			}
			w.Header().Set("WWW-Authenticate", `Basic realm="registry"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if user != "test-user" || password != "test-password" {
			t.Errorf("unexpected basic credentials")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		writeTestManifest(w, r)
	}))
	defer server.Close()
	config := testPrivateRegistry(server.URL, true, "test-user", "test-password")
	digest, err := ResolvePrivateTagDigest(t.Context(), "workload-rollback", config)
	require.NoError(t, err)
	require.Equal(t, testManifestDigest(), digest)
	require.Len(t, challenged, 1)
}

func TestResolvePrivateTagDigestWithBearerChallenge(t *testing.T) {
	challenged := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			user, password, ok := r.BasicAuth()
			if !ok || user != "test-user" || password != "test-password" {
				t.Errorf("token exchange did not use registry credentials")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if r.URL.Query().Get("scope") != "repository:scenarios:pull" {
				t.Errorf("unexpected token scope: %q", r.URL.Query().Get("scope"))
			}
			_, _ = w.Write([]byte(`{"token":"registry-token"}`))
			return
		}
		if r.URL.Path != "/v2/" && r.URL.Path != "/v2/scenarios/manifests/workload-rollback" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer registry-token" {
			select {
			case challenged <- struct{}{}:
			default:
			}
			w.Header().Set("WWW-Authenticate", `Bearer realm="`+serverURL(r)+`/token",service="registry",scope="repository:scenarios:pull"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		writeTestManifest(w, r)
	}))
	defer server.Close()
	config := testPrivateRegistry(server.URL, true, "test-user", "test-password")
	digest, err := ResolvePrivateTagDigest(t.Context(), "workload-rollback", config)
	require.NoError(t, err)
	require.Equal(t, testManifestDigest(), digest)
	require.Len(t, challenged, 1)
}

func serverURL(r *http.Request) string {
	return "http://" + r.Host
}

func TestResolvePrivateTagDigestMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"code":"MANIFEST_UNKNOWN","message":"not found"}]}`))
	}))
	defer server.Close()
	config := testPrivateRegistry(server.URL, true, "", "")
	_, err := ResolvePrivateTagDigest(t.Context(), "workload-rollback", config)
	require.ErrorIs(t, err, ErrTagNotFound)
}

func TestResolvePrivateTagDigestCancellation(t *testing.T) {
	requestStarted := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path != "/v2/scenarios/manifests/workload-rollback" {
			http.NotFound(w, r)
			return
		}
		close(requestStarted)
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	config := testPrivateRegistry(server.URL, true, "", "")
	done := make(chan error, 1)
	go func() {
		_, err := ResolvePrivateTagDigest(ctx, "workload-rollback", config)
		done <- err
	}()

	<-requestStarted
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("resolver did not stop after cancellation")
	}
}

func TestResolvePrivateTagDigestUnauthorizedErrorIsSafe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path != "/v2/scenarios/manifests/workload-rollback" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="registry"`)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("remote secret-body user=test-user password=test-password"))
	}))
	defer server.Close()
	config := testPrivateRegistry(server.URL, true, "test-user", "test-password")
	_, err := ResolvePrivateTagDigest(t.Context(), "workload-rollback", config)
	require.Error(t, err)
	require.EqualError(t, err, "private registry denied image access")
	require.NotContains(t, err.Error(), "secret-body")
	require.NotContains(t, err.Error(), "test-user")
	require.NotContains(t, err.Error(), "test-password")
}

func TestResolvePrivateTagDigestHonorsSkipTLS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path != "/v2/scenarios/manifests/workload-rollback" {
			http.NotFound(w, r)
			return
		}
		writeTestManifest(w, r)
	}))
	defer server.Close()
	config := testPrivateRegistry(server.URL, false, "", "")
	config.SkipTLS = true
	digest, err := ResolvePrivateTagDigest(t.Context(), "workload-rollback", config)
	require.NoError(t, err)
	require.Equal(t, testManifestDigest(), digest)
}

func TestResolvePrivateTagDigestReturnsSafeErrorForInvalidTag(t *testing.T) {
	config := &models.RegistryV2{RegistryURL: "registry.example", ScenarioRepository: "scenarios"}
	_, err := ResolvePrivateTagDigest(t.Context(), "contains space", config)
	require.Error(t, err)
	require.EqualError(t, err, "private image reference is invalid")
	require.False(t, errors.Is(err, ErrTagNotFound))
}
