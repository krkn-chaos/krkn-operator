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
	v1 "github.com/google/go-containerregistry/pkg/v1"
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
	descriptor, cancel, err := getPrivateTagDescriptor(ctx, tag, registry)
	if err != nil {
		return "", err
	}
	defer cancel()
	return descriptor.Digest.String(), nil
}

// ResolvePrivateTagImage reads a private image's manifest digest and the
// labels of its image config, such as krknctl's scenario labels, over the same
// authenticated connection as ResolvePrivateTagDigest.
func ResolvePrivateTagImage(ctx context.Context, tag string, registry *models.RegistryV2) (string, map[string]string, error) {
	descriptor, cancel, err := getPrivateTagDescriptor(ctx, tag, registry)
	if err != nil {
		return "", nil, err
	}
	defer cancel()
	labels, err := imageConfigLabels(descriptor, registry)
	if err != nil {
		return "", nil, err
	}
	return descriptor.Digest.String(), labels, nil
}

// getPrivateTagDescriptor fetches the manifest descriptor of a private tag. The
// returned cancel function bounds every later request made through the
// descriptor, so callers must call it when done.
func getPrivateTagDescriptor(ctx context.Context, tag string, registry *models.RegistryV2) (*remote.Descriptor, context.CancelFunc, error) {
	if registry == nil {
		return nil, nil, errors.New("private registry configuration is required")
	}

	options := verify.OptionsForRegistry(registry)
	parseOptions := []name.Option{name.StrictValidation}
	if options.Insecure {
		parseOptions = append(parseOptions, name.Insecure)
	}
	ref, err := name.NewTag(registry.GetPrivateRegistryURI()+":"+tag, parseOptions...)
	if err != nil {
		return nil, nil, errors.New("private image reference is invalid")
	}

	remoteOptions := make([]remote.Option, 0, len(options.RemoteOptions)+2)
	if options.Keychain != nil {
		remoteOptions = append(remoteOptions, remote.WithAuthFromKeychain(options.Keychain))
	}
	remoteOptions = append(remoteOptions, options.RemoteOptions...)

	resolveContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	remoteOptions = append(remoteOptions, remote.WithContext(resolveContext))

	descriptor, err := remote.Get(ref, remoteOptions...)
	if err != nil {
		defer cancel()
		if resolveContext.Err() != nil {
			return nil, nil, fmt.Errorf("resolve private image manifest: %w", resolveContext.Err())
		}
		var registryError *transport.Error
		if errors.As(err, &registryError) {
			switch registryError.StatusCode {
			case http.StatusNotFound:
				manifestPath := "/v2/" + ref.Context().RepositoryStr() + "/manifests/" + ref.Identifier()
				if registryError.Request != nil && registryError.Request.URL.Path == manifestPath {
					return nil, nil, fmt.Errorf("private image tag: %w", ErrTagNotFound)
				}
			case http.StatusUnauthorized, http.StatusForbidden:
				return nil, nil, errors.New("private registry denied image access")
			}
		}
		return nil, nil, errors.New("resolve private image manifest")
	}
	return descriptor, cancel, nil
}

// imageConfigLabels returns the labels of the image a descriptor points to.
// For an index it reads the image for the registry's target platform, as
// krknctl does when it lists the scenario, so the labels are those of the
// image that platform pulls.
func imageConfigLabels(descriptor *remote.Descriptor, registry *models.RegistryV2) (map[string]string, error) {
	var image v1.Image
	if descriptor.MediaType.IsIndex() {
		index, err := descriptor.ImageIndex()
		if err != nil {
			return nil, errors.New("read private image index")
		}
		manifest, err := index.IndexManifest()
		if err != nil {
			return nil, errors.New("read private image index")
		}
		for _, child := range manifest.Manifests {
			// Build attestations sit in the index as images for platform
			// unknown/unknown, so they never match a Linux target.
			if !child.MediaType.IsImage() || child.Platform == nil ||
				!registry.MatchesPlatform(child.Platform.OS, child.Platform.Architecture, child.Platform.Variant) {
				continue
			}
			if image, err = index.Image(child.Digest); err != nil {
				return nil, errors.New("read private image manifest")
			}
			break
		}
		if image == nil {
			return nil, fmt.Errorf("private image index has no image for platform %s", registry.GetPlatform())
		}
	} else {
		var err error
		if image, err = descriptor.Image(); err != nil {
			return nil, errors.New("read private image manifest")
		}
	}
	config, err := image.ConfigFile()
	if err != nil {
		return nil, errors.New("read private image config")
	}
	return config.Config.Labels, nil
}
