package controller

import (
	"context"
	"fmt"

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

func resolveImageSignature(ctx context.Context, config *krknctlconfig.Config, reference krknv1alpha1.ScenarioReference, privateRegistry *krknctlmodels.RegistryV2, image string) (verify.SignatureStatus, error) {
	var mode krknctlprovider.Mode = krknctlprovider.Quay
	if reference.Private != nil && *reference.Private {
		mode = krknctlprovider.Private
	}
	dataProvider := krknctlfactory.NewProviderFactory(config).NewInstance(mode)
	if dataProvider == nil {
		return verify.SignatureUnknown, fmt.Errorf("failed to create image signature provider")
	}
	status, err := dataProvider.GetImageSignatureStatus(ctx, privateRegistry, krknctlmodels.ScenarioTag{Name: reference.Name})
	if err != nil {
		return verify.SignatureUnknown, fmt.Errorf("failed to verify signature for image %q: %w", image, err)
	}
	return status, nil
}

func verifyResolvedImageSignature(ctx context.Context, config *krknctlconfig.Config, reference krknv1alpha1.ScenarioReference, privateRegistry *krknctlmodels.RegistryV2, image string) error {
	status, err := resolveImageSignature(ctx, config, reference, privateRegistry, image)
	if err != nil {
		return err
	}
	if status != verify.SignatureSigned {
		return &InvalidImageSignatureError{Image: image, Status: status}
	}
	return nil
}
