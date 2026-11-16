package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/krkn-chaos/krknctl/pkg/provider/models"
	"github.com/krkn-chaos/krknctl/pkg/verify"
)

// ErrTagNotFound means the requested image tag is absent from a registry.
var ErrTagNotFound = errors.New("image tag not found")

// ResolvePublicTagDigest reads the active digest of a Quay tag without treating
// it as a selectable scenario. Workload images have is_a_scenario=false.
func ResolvePublicTagDigest(ctx context.Context, repositoryAPIURI, tag string) (string, error) {
	u, err := url.Parse(strings.TrimRight(repositoryAPIURI, "/") + "/tag/")
	if err != nil {
		return "", fmt.Errorf("build Quay tag URL: %w", err)
	}
	query := u.Query()
	query.Set("specificTag", tag)
	query.Set("onlyActiveTags", "true")
	u.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create Quay tag request: %w", err)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("load Quay tag %q: %w", tag, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("load Quay tag %q: HTTP %d", tag, response.StatusCode)
	}
	var result struct {
		Tags []struct {
			Name           string `json:"name"`
			ManifestDigest string `json:"manifest_digest"`
			EndTS          *int64 `json:"end_ts"`
		} `json:"tags"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return "", fmt.Errorf("decode Quay tag %q: %w", tag, err)
	}
	for _, entry := range result.Tags {
		if entry.Name == tag && entry.EndTS == nil {
			if entry.ManifestDigest == "" {
				return "", fmt.Errorf("quay tag %q has no digest", tag)
			}
			return entry.ManifestDigest, nil
		}
	}
	return "", fmt.Errorf("quay tag %q: %w", tag, ErrTagNotFound)
}

// ResolvePrivateTagDigest reads a Docker Registry v2 manifest digest without
// requiring the image to be labeled as a selectable scenario.
func ResolvePrivateTagDigest(ctx context.Context, tag string, registry *models.RegistryV2) (string, error) {
	if registry == nil {
		return "", errors.New("private registry configuration is required")
	}

	options := verify.OptionsForRegistry(registry)
	parseOptions := []name.Option{name.StrictValidation}
	if options.Insecure {
		parseOptions = append(parseOptions, name.Insecure)
	}
	ref, err := name.NewTag(registry.GetPrivateRegistryURI()+":"+tag, parseOptions...)
	if err != nil {
		return "", errors.New("private image reference is invalid")
	}

	remoteOptions := make([]remote.Option, 0, len(options.RemoteOptions)+2)
	if options.Keychain != nil {
		remoteOptions = append(remoteOptions, remote.WithAuthFromKeychain(options.Keychain))
	}
	remoteOptions = append(remoteOptions, options.RemoteOptions...)

	resolveContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	remoteOptions = append(remoteOptions, remote.WithContext(resolveContext))

	descriptor, err := remote.Get(ref, remoteOptions...)
	if err != nil {
		if resolveContext.Err() != nil {
			return "", fmt.Errorf("resolve private image manifest: %w", resolveContext.Err())
		}
		var registryError *transport.Error
		if errors.As(err, &registryError) {
			switch registryError.StatusCode {
			case http.StatusNotFound:
				manifestPath := "/v2/" + ref.Context().RepositoryStr() + "/manifests/" + ref.Identifier()
				if registryError.Request != nil && registryError.Request.URL.Path == manifestPath {
					return "", fmt.Errorf("private image tag: %w", ErrTagNotFound)
				}
			case http.StatusUnauthorized, http.StatusForbidden:
				return "", errors.New("private registry denied image access")
			}
		}
		return "", errors.New("resolve private image manifest")
	}
	return descriptor.Digest.String(), nil
}
