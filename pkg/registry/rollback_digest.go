package registry

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/krkn-chaos/krknctl/pkg/provider/models"
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
	manifestURI, err := registry.GetV2ScenarioDetailAPIURI(tag)
	if err != nil {
		return "", fmt.Errorf("build private image manifest URI: %w", err)
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{
		InsecureSkipVerify: registry.SkipTLS, // #nosec G402 -- administrator-controlled registry setting.
	}}
	client := &http.Client{Transport: transport, Timeout: 30 * time.Second}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURI, nil)
	if err != nil {
		return "", fmt.Errorf("create private image manifest request: %w", err)
	}
	request.Header.Set("Accept", strings.Join([]string{
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.oci.image.index.v1+json",
	}, ", "))
	if registry.Token != nil && *registry.Token != "" {
		request.Header.Set("Authorization", "Bearer "+*registry.Token)
	} else if registry.Username != nil {
		password := ""
		if registry.Password != nil {
			password = *registry.Password
		}
		request.SetBasicAuth(*registry.Username, password)
	}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("load private image manifest: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("private image tag %q: %w", tag, ErrTagNotFound)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("private image manifest request returned HTTP %d", response.StatusCode)
	}
	digest := response.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return "", errors.New("private image manifest response has no Docker-Content-Digest")
	}
	return digest, nil
}
