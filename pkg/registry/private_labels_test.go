package registry

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	ggcrregistry "github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/require"
)

// labelRegistry serves an in-memory registry behind Basic authentication, so
// the tests also check that the image config is read with the same credentials
// as the manifest.
func labelRegistry(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	inner := ggcrregistry.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, password, ok := r.BasicAuth(); !ok || user != "test-user" || password != "test-password" {
			w.Header().Set("WWW-Authenticate", `Basic realm="registry"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		inner.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	return server, strings.TrimPrefix(server.URL, "http://")
}

func labeledImage(t *testing.T, labels map[string]string) v1.Image {
	t.Helper()
	image, err := random.Image(64, 1)
	require.NoError(t, err)
	config, err := image.ConfigFile()
	require.NoError(t, err)
	config.Config.Labels = labels
	image, err = mutate.ConfigFile(image, config)
	require.NoError(t, err)
	return image
}

func pushTag(t *testing.T, host, tag string) name.Tag {
	t.Helper()
	ref, err := name.NewTag(host+"/scenarios:"+tag, name.Insecure)
	require.NoError(t, err)
	return ref
}

var testBasicAuth = remote.WithAuth(&authn.Basic{Username: "test-user", Password: "test-password"})

func TestResolvePrivateTagImageReadsConfigLabels(t *testing.T) {
	server, host := labelRegistry(t)
	image := labeledImage(t, map[string]string{"krknctl.privileged": "true"})
	require.NoError(t, remote.Write(pushTag(t, host, "node-io-hog"), image, testBasicAuth))

	digest, labels, err := ResolvePrivateTagImage(t.Context(), "node-io-hog", testPrivateRegistry(server.URL, true, "test-user", "test-password"))

	require.NoError(t, err)
	want, err := image.Digest()
	require.NoError(t, err)
	require.Equal(t, want.String(), digest)
	require.Equal(t, "true", labels["krknctl.privileged"])
}

func TestResolvePrivateTagImageSkipsAttestationsInAnIndex(t *testing.T) {
	server, host := labelRegistry(t)
	attestation := labeledImage(t, nil)
	platformImage := labeledImage(t, map[string]string{"krknctl.privileged": "true"})
	index := mutate.AppendManifests(empty.Index,
		mutate.IndexAddendum{Add: attestation, Descriptor: v1.Descriptor{Platform: &v1.Platform{OS: "unknown", Architecture: "unknown"}}},
		mutate.IndexAddendum{Add: platformImage, Descriptor: v1.Descriptor{Platform: &v1.Platform{OS: "linux", Architecture: "arm64"}}},
	)
	require.NoError(t, remote.WriteIndex(pushTag(t, host, "node-io-hog"), index, testBasicAuth))
	registry := testPrivateRegistry(server.URL, true, "test-user", "test-password")
	registry.Platform = "linux/arm64"

	digest, labels, err := ResolvePrivateTagImage(t.Context(), "node-io-hog", registry)

	require.NoError(t, err)
	want, err := index.Digest()
	require.NoError(t, err)
	require.Equal(t, want.String(), digest)
	require.Equal(t, "true", labels["krknctl.privileged"])
}

func TestResolvePrivateTagImageReadsTheTargetPlatformsLabels(t *testing.T) {
	server, host := labelRegistry(t)
	amd64 := labeledImage(t, map[string]string{"krknctl.privileged": "false"})
	arm64 := labeledImage(t, map[string]string{"krknctl.privileged": "true"})
	index := mutate.AppendManifests(empty.Index,
		mutate.IndexAddendum{Add: amd64, Descriptor: v1.Descriptor{Platform: &v1.Platform{OS: "linux", Architecture: "amd64"}}},
		mutate.IndexAddendum{Add: arm64, Descriptor: v1.Descriptor{Platform: &v1.Platform{OS: "linux", Architecture: "arm64", Variant: "v8"}}},
	)
	require.NoError(t, remote.WriteIndex(pushTag(t, host, "node-io-hog"), index, testBasicAuth))

	for platform, want := range map[string]string{"linux/amd64": "false", "linux/arm64": "true", "linux/arm64/v8": "true"} {
		t.Run(platform, func(t *testing.T) {
			registry := testPrivateRegistry(server.URL, true, "test-user", "test-password")
			registry.Platform = platform

			_, labels, err := ResolvePrivateTagImage(t.Context(), "node-io-hog", registry)

			require.NoError(t, err)
			require.Equal(t, want, labels["krknctl.privileged"])
		})
	}

	t.Run("no image for the platform", func(t *testing.T) {
		registry := testPrivateRegistry(server.URL, true, "test-user", "test-password")
		registry.Platform = "linux/s390x"

		_, _, err := ResolvePrivateTagImage(t.Context(), "node-io-hog", registry)

		require.ErrorContains(t, err, "private image index has no image for platform linux/s390x")
	})
}

func TestResolvePrivateTagImageMissingTag(t *testing.T) {
	server, _ := labelRegistry(t)

	_, _, err := ResolvePrivateTagImage(t.Context(), "missing", testPrivateRegistry(server.URL, true, "test-user", "test-password"))

	require.True(t, errors.Is(err, ErrTagNotFound), "error: %v", err)
}
