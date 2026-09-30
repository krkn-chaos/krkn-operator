package controller

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	registryutil "github.com/krkn-chaos/krkn-operator/pkg/registry"
	"github.com/krkn-chaos/krkn-operator/pkg/signatureverification"
	krknctlconfig "github.com/krkn-chaos/krknctl/pkg/config"
	krknctlprovider "github.com/krkn-chaos/krknctl/pkg/provider"
	krknctlfactory "github.com/krkn-chaos/krknctl/pkg/provider/factory"
	krknctlmodels "github.com/krkn-chaos/krknctl/pkg/provider/models"
	"github.com/krkn-chaos/krknctl/pkg/verify"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InvalidImageSignatureError identifies a terminal image trust failure. It is
// deliberately distinct from infrastructure errors so reconcilers can avoid
// retrying an image that will never become valid without a spec or setting
// change.
type InvalidImageSignatureError struct {
	Image  string
	Status verify.SignatureStatus
}

type imageSignatureVerifier func(context.Context, *krknctlconfig.Config, krknv1alpha1.ScenarioReference, *krknctlmodels.RegistryV2, string) (verify.SignatureStatus, error)
type imageResolver func(context.Context, *krknctlconfig.Config, krknv1alpha1.ScenarioReference, *krknctlmodels.RegistryV2) (string, error)

func (e *InvalidImageSignatureError) Error() string {
	return fmt.Sprintf("image %q does not have a valid signature (status: %s)", e.Image, e.Status)
}

func imageSignatureVerificationEnabled(ctx context.Context, k8sClient client.Client, namespace string) (bool, error) {
	return signatureverification.GetEnabled(ctx, k8sClient, namespace)
}

func resolvePrivateRegistry(ctx context.Context, k8sClient client.Client, namespace string, reference krknv1alpha1.ScenarioReference) (*krknctlmodels.RegistryV2, error) {
	if reference.Private == nil || !*reference.Private {
		return nil, nil
	}
	var registrySecret corev1.Secret
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: reference.RegistryName, Namespace: namespace}, &registrySecret); err != nil {
		return nil, fmt.Errorf("failed to load private registry %q: %w", reference.RegistryName, err)
	}
	privateRegistry, err := registryutil.ExtractRegistryV2FromSecret(&registrySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to load private registry configuration: %w", err)
	}
	return privateRegistry, nil
}

// resolveImmutableImage resolves the scenario metadata once and returns an
// image reference pinned to the digest observed during that resolution.
func resolveImmutableImage(ctx context.Context, config *krknctlconfig.Config, reference krknv1alpha1.ScenarioReference, privateRegistry *krknctlmodels.RegistryV2) (string, error) {
	if err := reference.Validate(); err != nil {
		return "", fmt.Errorf("invalid scenario reference: %w", err)
	}
	var mode krknctlprovider.Mode = krknctlprovider.Quay
	var base string
	if reference.Private != nil && *reference.Private {
		if privateRegistry == nil {
			return "", fmt.Errorf("private registry configuration is required")
		}
		base = privateRegistry.GetPrivateRegistryURI()
		digest, err := resolvePrivateImageDigest(ctx, reference.Name, privateRegistry)
		if err != nil {
			return "", err
		}
		return krknctlprovider.ImageReference(base, krknctlmodels.ScenarioTag{Name: reference.Name, Digest: &digest}), nil
	} else {
		var err error
		base, err = config.GetQuayImageURI()
		if err != nil {
			return "", fmt.Errorf("failed to build public scenario image URI: %w", err)
		}
	}
	provider := krknctlfactory.NewProviderFactory(config).NewInstance(mode)
	if provider == nil {
		return "", fmt.Errorf("failed to create image provider")
	}
	detail, err := provider.GetScenarioDetail(reference.Name, privateRegistry)
	if err != nil {
		return "", fmt.Errorf("failed to resolve image digest for %q: %w", reference.Name, err)
	}
	if detail == nil || detail.Digest == nil || *detail.Digest == "" {
		return "", fmt.Errorf("scenario %q did not resolve to an image digest", reference.Name)
	}
	return krknctlprovider.ImageReference(base, detail.ScenarioTag), nil
}

func resolvePrivateImageDigest(ctx context.Context, scenario string, registry *krknctlmodels.RegistryV2) (string, error) {
	manifestURI, err := registry.GetV2ScenarioDetailAPIURI(scenario)
	if err != nil {
		return "", fmt.Errorf("failed to build private image manifest URI: %w", err)
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{
		InsecureSkipVerify: registry.SkipTLS, // #nosec G402 -- skipTLS is an explicit administrator-controlled registry setting.
	}}
	client := &http.Client{Transport: transport, Timeout: 30 * time.Second}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURI, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create private image manifest request: %w", err)
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
		return "", fmt.Errorf("failed to resolve private image digest: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(response.Body)
		return "", fmt.Errorf("private image manifest request returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	digest := response.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return "", fmt.Errorf("private image manifest response did not include Docker-Content-Digest")
	}
	return digest, nil
}

func scenarioTagForImage(reference krknv1alpha1.ScenarioReference, image string) krknctlmodels.ScenarioTag {
	tag := krknctlmodels.ScenarioTag{Name: reference.Name}
	if _, digest, ok := strings.Cut(image, "@"); ok && digest != "" {
		tag.Digest = &digest
	}
	return tag
}

func resolveImageSignature(ctx context.Context, config *krknctlconfig.Config, reference krknv1alpha1.ScenarioReference, privateRegistry *krknctlmodels.RegistryV2, image string) (verify.SignatureStatus, error) {
	var mode krknctlprovider.Mode = krknctlprovider.Quay
	if reference.Private != nil && *reference.Private {
		mode = krknctlprovider.Private
	}
	dataProvider := krknctlfactory.NewProviderFactory(config).NewInstance(mode)
	if dataProvider == nil {
		return verify.SignatureUnknown, fmt.Errorf("failed to create image signature provider")
	}
	status, err := dataProvider.GetImageSignatureStatus(ctx, privateRegistry, scenarioTagForImage(reference, image))
	if err != nil {
		return verify.SignatureUnknown, fmt.Errorf("failed to verify signature for image %q: %w", image, err)
	}
	return status, nil
}
