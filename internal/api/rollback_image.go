package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/registry"
	"github.com/krkn-chaos/krkn-operator/pkg/signatureverification"
	krknctlconfig "github.com/krkn-chaos/krknctl/pkg/config"
	"github.com/krkn-chaos/krknctl/pkg/provider"
	"github.com/krkn-chaos/krknctl/pkg/provider/models"
	"github.com/krkn-chaos/krknctl/pkg/verify"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const rollbackImageName = "workload-rollback"
const rollbackImageCacheTTL = 30 * time.Second

type rollbackImageCheck struct {
	status    int
	code      string
	message   string
	checkedAt time.Time
}

func (r rollbackImageCheck) available() bool { return r.status == 0 }

type rollbackImageCache struct {
	mu      sync.Mutex
	entries map[string]rollbackImageCheck
}

func (c *rollbackImageCache) get(key string) (rollbackImageCheck, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok {
		return rollbackImageCheck{}, false
	}
	if time.Since(entry.checkedAt) >= rollbackImageCacheTTL {
		delete(c.entries, key)
		return rollbackImageCheck{}, false
	}
	return entry, true
}

func (c *rollbackImageCache) put(key string, entry rollbackImageCheck) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[string]rollbackImageCheck)
	}
	for existingKey, existing := range c.entries {
		if time.Since(existing.checkedAt) >= rollbackImageCacheTTL {
			delete(c.entries, existingKey)
		}
	}
	entry.checkedAt = time.Now()
	c.entries[key] = entry
}

func resolveRollbackDigest(ctx context.Context, private bool, registryConfig *models.RegistryV2) (string, error) {
	if private {
		return registry.ResolvePrivateTagDigest(ctx, rollbackImageName, registryConfig)
	}
	config, err := krknctlconfig.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("load public image configuration: %w", err)
	}
	repositoryAPIURI, err := config.GetQuayScenarioRepositoryAPIURI()
	if err != nil {
		return "", fmt.Errorf("build public rollback repository URI: %w", err)
	}
	return registry.ResolvePublicTagDigest(ctx, repositoryAPIURI, rollbackImageName)
}

// checkRollbackImage validates the image that the controller will resolve for
// this rollback. GET may reuse a short-lived result; POST always checks again.
func (h *Handler) checkRollbackImage(ctx context.Context, run *krknv1alpha1.KrknScenarioRun, useCache bool) rollbackImageCheck {
	private := run.Spec.Scenario.Private != nil && *run.Spec.Scenario.Private
	var mode provider.Mode = provider.Quay
	var registryConfig *models.RegistryV2
	registryIdentity := "public"
	if private {
		mode = provider.Private
		secret, err := h.loadRegistrySecret(ctx, run.Spec.Scenario.RegistryName)
		if err != nil {
			log.FromContext(ctx).Error(err, "Failed to load original scenario registry for rollback", "registryName", run.Spec.Scenario.RegistryName)
			return rollbackImageCheck{status: http.StatusConflict, code: "rollback_registry_unavailable", message: "The original scenario registry is unavailable"}
		}
		if !h.canAccessRegistry(ctx, secret) {
			return rollbackImageCheck{status: http.StatusForbidden, code: "forbidden", message: "Access to the original scenario registry is required"}
		}
		registryConfig, err = registry.ExtractRegistryV2FromSecret(secret)
		if err != nil {
			log.FromContext(ctx).Error(err, "Invalid rollback registry configuration", "registryName", secret.Name)
			return rollbackImageCheck{status: http.StatusConflict, code: "rollback_registry_unavailable", message: "The original scenario registry configuration is invalid"}
		}
		registryIdentity = fmt.Sprintf("%s/%s/%s", secret.Name, secret.UID, secret.ResourceVersion)
	}
	verifySignatures, err := signatureverification.GetEnabled(ctx, h.client, h.namespace)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to read rollback image signature setting")
		return rollbackImageCheck{status: http.StatusServiceUnavailable, code: "rollback_image_check_failed", message: "Could not check rollback image signature policy"}
	}
	cacheKey := fmt.Sprintf("%s/%t/%t", registryIdentity, private, verifySignatures)
	if useCache {
		if cached, ok := h.rollbackImages.get(cacheKey); ok {
			return cached
		}
	}
	resolver := h.rollbackDigestResolver
	if resolver == nil {
		resolver = resolveRollbackDigest
	}
	digest, err := resolver(ctx, private, registryConfig)
	if ctxErr := ctx.Err(); ctxErr != nil {
		err = ctxErr
	}
	if errors.Is(err, registry.ErrTagNotFound) {
		result := rollbackImageCheck{status: http.StatusConflict, code: "rollback_image_unavailable", message: "The workload-rollback image is not available in the selected registry"}
		if useCache {
			h.rollbackImages.put(cacheKey, result)
		}
		return result
	}
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to inspect rollback image", "registry", registryIdentity)
		return rollbackImageCheck{status: http.StatusServiceUnavailable, code: "rollback_image_check_failed", message: "Could not check the rollback image registry"}
	}
	if digest == "" {
		return rollbackImageCheck{status: http.StatusServiceUnavailable, code: "rollback_image_check_failed", message: "The rollback image registry did not return a digest"}
	}
	if verifySignatures {
		factory := h.scenarioProviderFactory
		if factory == nil {
			factory = createScenarioProvider
		}
		scenarioProvider, err := factory(mode)
		if err != nil {
			log.FromContext(ctx).Error(err, "Failed to create rollback image provider")
			return rollbackImageCheck{status: http.StatusServiceUnavailable, code: "rollback_image_check_failed", message: "Could not verify the rollback image signature"}
		}
		status, err := scenarioProvider.GetImageSignatureStatus(ctx, registryConfig, models.ScenarioTag{Name: rollbackImageName, Digest: &digest})
		if ctxErr := ctx.Err(); ctxErr != nil {
			err = ctxErr
		} else if err == nil && status == verify.SignatureUnknown {
			err = errors.New("rollback image signature status is unknown")
		}
		if err != nil {
			log.FromContext(ctx).Error(err, "Failed to verify rollback image signature", "registry", registryIdentity)
			return rollbackImageCheck{status: http.StatusServiceUnavailable, code: "rollback_image_check_failed", message: "Could not verify the rollback image signature"}
		}
		if status != verify.SignatureSigned {
			result := rollbackImageCheck{status: http.StatusConflict, code: "rollback_image_untrusted", message: "The workload-rollback image does not have a trusted signature"}
			if useCache {
				h.rollbackImages.put(cacheKey, result)
			}
			return result
		}
	}
	result := rollbackImageCheck{}
	if useCache {
		h.rollbackImages.put(cacheKey, result)
	}
	return result
}
