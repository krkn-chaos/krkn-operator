package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/krknaiserver"
	registryutil "github.com/krkn-chaos/krkn-operator/pkg/registry"
	krknctlmodels "github.com/krkn-chaos/krknctl/pkg/provider/models"
	corev1 "k8s.io/api/core/v1"
	utilvalidation "k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type aiRunRegistryImages struct {
	orchestrator  string
	uploader      string
	registry      *krknctlmodels.RegistryV2
	pullSecrets   []corev1.LocalObjectReference
	inventoryJSON string
}

func (r *KrknAIRunReconciler) resolveRunRegistry(ctx context.Context, run *krknv1alpha1.KrknAIRun, configYAML string) (*aiRunRegistryImages, error) {
	if err := krknaiserver.ValidateScenarioSelection(configYAML); err != nil {
		return nil, &configurationError{err: fmt.Errorf("no runnable scenarios are enabled: %w", err)}
	}
	images := &aiRunRegistryImages{
		orchestrator: r.OrchestratorImage,
		uploader:     r.ServiceImage,
		pullSecrets:  r.ImagePullSecrets,
	}
	if run.Spec.RegistryName == "" {
		return images, nil
	}
	if len(utilvalidation.IsDNS1123Subdomain(run.Spec.RegistryName)) != 0 {
		return nil, &configurationError{err: fmt.Errorf("registryName must be a valid Secret name")}
	}

	var secret corev1.Secret
	if err := r.APIReader.Get(ctx, client.ObjectKey{Namespace: run.Namespace, Name: run.Spec.RegistryName}, &secret); err != nil {
		return nil, requiredResourceError(fmt.Errorf("read private registry %q: %w", run.Spec.RegistryName, err))
	}
	privateRegistry, err := registryutil.ExtractRegistryV2FromSecret(&secret)
	if err != nil {
		return nil, &configurationError{err: fmt.Errorf("invalid private registry %q: %w", secret.Name, err)}
	}
	var token corev1.Secret
	if err := r.APIReader.Get(ctx, client.ObjectKey{Namespace: run.Namespace, Name: r.ServiceTokenSecretName}, &token); err != nil {
		return nil, requiredResourceError(fmt.Errorf("read artifact service credential: %w", err))
	}
	if len(token.Data["token"]) == 0 {
		return nil, &configurationError{err: fmt.Errorf("artifact service token is missing")}
	}
	service := krknaiserver.New(r.ServiceURL, string(token.Data["token"]))
	requirements, err := service.ImageRequirements(ctx)
	if err != nil {
		return nil, fmt.Errorf("read Krkn-AI image requirements: %w", err)
	}
	inspect := r.registryInspector
	if inspect == nil {
		inspect = registryutil.InspectAIImages
	}
	inventory, err := inspect(ctx, &secret, requirements.ScenarioTags, requirements.WorkloadImageTags)
	if err != nil {
		if errors.Is(err, registryutil.ErrAIRegistryConfiguration) {
			return nil, &configurationError{err: err}
		}
		return nil, err
	}
	// A direct CR must pass the same image checks as an API-created run.
	if err := service.ValidateConfigImages(ctx, configYAML, inventory); err != nil {
		if errors.Is(err, krknaiserver.ErrInvalidConfig) {
			return nil, &configurationError{err: err}
		}
		return nil, fmt.Errorf("validate private Krkn-AI configuration: %w", err)
	}
	data, err := json.Marshal(inventory)
	if err != nil {
		return nil, err
	}
	images.orchestrator, images.uploader = inventory.OrchestratorImage, inventory.ResultUploaderImage
	images.registry, images.inventoryJSON = privateRegistry, string(data)
	for _, reference := range r.ImagePullSecrets {
		if reference.Name == secret.Name {
			return images, nil
		}
	}
	images.pullSecrets = make([]corev1.LocalObjectReference, 0, len(r.ImagePullSecrets)+1)
	images.pullSecrets = append(images.pullSecrets, r.ImagePullSecrets...)
	images.pullSecrets = append(images.pullSecrets, corev1.LocalObjectReference{Name: secret.Name})
	return images, nil
}

func (r *KrknScenarioRunReconciler) validateAIChildRegistry(ctx context.Context, run *krknv1alpha1.KrknScenarioRun) error {
	for _, owner := range run.OwnerReferences {
		if owner.APIVersion != krknv1alpha1.GroupVersion.String() || owner.Kind != "KrknAIRun" {
			continue
		}
		var parent krknv1alpha1.KrknAIRun
		if err := r.Get(ctx, client.ObjectKey{Namespace: run.Namespace, Name: owner.Name}, &parent); err != nil {
			return fmt.Errorf("read parent registry selection: %w", err)
		}
		if parent.UID != owner.UID {
			return fmt.Errorf("parent KrknAIRun identity changed")
		}
		reference := run.Spec.Scenario
		if reference.Private == nil || *reference.Private != (parent.Spec.RegistryName != "") || reference.RegistryName != parent.Spec.RegistryName {
			return fmt.Errorf("scenario registry does not match parent KrknAIRun")
		}
	}
	return nil
}
