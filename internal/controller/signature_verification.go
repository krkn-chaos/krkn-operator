package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
type imageResolver func(context.Context, *krknctlconfig.Config, krknv1alpha1.ScenarioReference, *krknctlmodels.RegistryV2) (resolvedImage, error)

// resolvedImage is a scenario image pinned by digest, with whether the scenario
// declares that it needs a privileged container (the krknctl.privileged image
// label, which krknctl reads too).
type resolvedImage struct {
	image      string
	privileged bool
}

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
func resolveImmutableImage(ctx context.Context, config *krknctlconfig.Config, reference krknv1alpha1.ScenarioReference, privateRegistry *krknctlmodels.RegistryV2) (resolvedImage, error) {
	if err := reference.Validate(); err != nil {
		return resolvedImage{}, fmt.Errorf("invalid scenario reference: %w", err)
	}
	var mode krknctlprovider.Mode = krknctlprovider.Quay
	var base string
	if reference.Private != nil && *reference.Private {
		if privateRegistry == nil {
			return resolvedImage{}, fmt.Errorf("private registry configuration is required")
		}
		base = privateRegistry.GetPrivateRegistryURI()
		// Read the label from the image config over the same connection as the
		// digest: krknctl's private-registry provider does not handle every
		// registry's authentication.
		digest, labels, err := registryutil.ResolvePrivateTagImage(ctx, reference.Name, privateRegistry)
		if err != nil {
			return resolvedImage{}, err
		}
		privileged, err := privilegedLabel(config, labels)
		if err != nil {
			return resolvedImage{}, fmt.Errorf("scenario %q: %w", reference.Name, err)
		}
		return resolvedImage{
			image:      krknctlprovider.ImageReference(base, krknctlmodels.ScenarioTag{Name: reference.Name, Digest: &digest}),
			privileged: privileged,
		}, nil
	} else {
		var err error
		base, err = config.GetQuayImageURI()
		if err != nil {
			return resolvedImage{}, fmt.Errorf("failed to build public scenario image URI: %w", err)
		}
		if reference.Name == "workload-rollback" {
			repositoryAPIURI, err := config.GetQuayScenarioRepositoryAPIURI()
			if err != nil {
				return resolvedImage{}, fmt.Errorf("failed to build public rollback repository URI: %w", err)
			}
			digest, err := registryutil.ResolvePublicTagDigest(ctx, repositoryAPIURI, reference.Name)
			if err != nil {
				return resolvedImage{}, fmt.Errorf("failed to resolve rollback image digest: %w", err)
			}
			// The rollback image is labeled krknctl.privileged=false.
			return resolvedImage{image: krknctlprovider.ImageReference(base, krknctlmodels.ScenarioTag{Name: reference.Name, Digest: &digest})}, nil
		}
	}
	provider := krknctlfactory.NewProviderFactory(config).NewInstance(mode)
	if provider == nil {
		return resolvedImage{}, fmt.Errorf("failed to create image provider")
	}
	detail, err := provider.GetScenarioDetail(reference.Name, privateRegistry)
	if err != nil {
		return resolvedImage{}, fmt.Errorf("failed to resolve image digest for %q: %w", reference.Name, err)
	}
	if detail == nil || detail.Digest == nil || *detail.Digest == "" {
		return resolvedImage{}, fmt.Errorf("scenario %q did not resolve to an image digest", reference.Name)
	}
	return resolvedImage{
		image:      krknctlprovider.ImageReference(base, detail.ScenarioTag),
		privileged: detail.Privileged,
	}, nil
}

// privilegedLabel reads the krknctl.privileged label, named as in krknctl's
// configuration. A missing label means the scenario does not need privileges.
func privilegedLabel(config *krknctlconfig.Config, labels map[string]string) (bool, error) {
	key := strings.TrimSuffix(config.LabelPrivileged, "=")
	value, ok := labels[key]
	if !ok {
		return false, nil
	}
	privileged, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, fmt.Errorf("invalid %s label %q", key, value)
	}
	return privileged, nil
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
