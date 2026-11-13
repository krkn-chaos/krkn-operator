package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/krkn-chaos/krknctl/pkg/provider/models"
	"github.com/krkn-chaos/krknctl/pkg/verify"
	corev1 "k8s.io/api/core/v1"
)

const (
	maxAIImageTags             = 512
	maxImageProbes             = 8
	imageProbeTimeout          = 15 * time.Second
	defaultAIOrchestratorTag   = "krkn-ai-operator"
	defaultAIResultUploaderTag = "krkn-ai-service"
)

var workloadRoles = []string{"hog", "tools", "synFlood", "networkChaos"}

// ValidateAITagMetadata validates admin-selected tags without accepting image URLs.
func ValidateAITagMetadata(metadata AITagMetadata) error {
	if err := validateOptionalTag(metadata.AIOrchestratorTag); err != nil {
		return fmt.Errorf("aiOrchestratorTag is invalid: %w", err)
	}
	if err := validateOptionalTag(metadata.AIResultUploaderTag); err != nil {
		return fmt.Errorf("aiResultUploaderTag is invalid: %w", err)
	}
	for role, tag := range metadata.WorkloadImageTags {
		if !isWorkloadRole(role) {
			return fmt.Errorf("unsupported workloadImageTags role %q", role)
		}
		if err := validateOptionalTag(tag); err != nil {
			return fmt.Errorf("workloadImageTags.%s must be a valid tag", role)
		}
	}
	return nil
}

// InspectAIImages checks the finite set of scenario, workload, and runtime tags
// selected for Krkn-AI. Registry credentials are used only for these probes.
func InspectAIImages(ctx context.Context, secret *corev1.Secret, scenarioTags []string, defaultWorkloadTags map[string]string) (*AIImageInventory, error) {
	if secret == nil {
		return nil, fmt.Errorf("%w: registry secret is missing", ErrAIRegistryConfiguration)
	}
	metadata, err := AIRegistryTagMetadata(secret)
	if err != nil {
		return nil, err
	}
	if err := ValidateAITagMetadata(metadata); err != nil {
		return nil, fmt.Errorf("%w: invalid image tag metadata", ErrAIRegistryConfiguration)
	}
	if metadata.AIOrchestratorTag == "" {
		metadata.AIOrchestratorTag = defaultAIOrchestratorTag
	}
	if metadata.AIResultUploaderTag == "" {
		metadata.AIResultUploaderTag = defaultAIResultUploaderTag
	}
	if len(scenarioTags)+len(defaultWorkloadTags)+2 > maxAIImageTags {
		return nil, fmt.Errorf("%w: image inventory exceeds the supported probe limit", ErrAIRegistryConfiguration)
	}
	registryURL := secret.Annotations[RegistryURLAnnotation]
	repositoryPath := secret.Annotations[ScenarioRepositoryAnnotation]
	if registryURL == "" || strings.Contains(registryURL, "://") || repositoryPath == "" {
		return nil, fmt.Errorf("%w: registry URL or repository is invalid", ErrAIRegistryConfiguration)
	}
	registryConfig, err := ExtractRegistryV2FromSecret(secret)
	if err != nil || registryConfig.Username == nil || *registryConfig.Username == "" || registryConfig.Password == nil || *registryConfig.Password == "" {
		return nil, fmt.Errorf("%w: registry credentials are invalid", ErrAIRegistryConfiguration)
	}

	if err := validateDefaultWorkloadTags(defaultWorkloadTags); err != nil {
		return nil, err
	}

	selectedWorkloadTags := make(map[string]string, len(workloadRoles))
	for _, role := range workloadRoles {
		tag := defaultWorkloadTags[role]
		if override := metadata.WorkloadImageTags[role]; override != "" {
			tag = override
		}
		if err := validateOptionalTag(tag); err != nil || tag == "" {
			return nil, fmt.Errorf("%w: workload image tag for role %s is invalid", ErrAIRegistryConfiguration, role)
		}
		selectedWorkloadTags[role] = tag
	}

	baseRepository := registryURL + "/" + repositoryPath
	parseOptions := []name.Option{name.StrictValidation}
	if registryConfig.Insecure {
		parseOptions = append(parseOptions, name.Insecure)
	}
	repository, err := name.NewRepository(baseRepository, parseOptions...)
	if err != nil {
		return nil, fmt.Errorf("%w: registry repository is invalid", ErrAIRegistryConfiguration)
	}

	probeTags := make([]string, 0, len(scenarioTags)+len(selectedWorkloadTags)+2)
	probeTags = append(probeTags, scenarioTags...)
	for _, role := range workloadRoles {
		probeTags = append(probeTags, selectedWorkloadTags[role])
	}
	probeTags = append(probeTags, metadata.AIOrchestratorTag, metadata.AIResultUploaderTag)
	uniqueTags := make([]string, 0, len(probeTags))
	seen := make(map[string]struct{}, len(probeTags))
	for _, tag := range probeTags {
		if err := validateOptionalTag(tag); err != nil || tag == "" {
			return nil, fmt.Errorf("%w: image catalog contains an invalid tag", ErrAIRegistryConfiguration)
		}
		if _, exists := seen[tag]; !exists {
			seen[tag] = struct{}{}
			uniqueTags = append(uniqueTags, tag)
		}
	}
	if len(uniqueTags) > maxAIImageTags {
		return nil, fmt.Errorf("%w: image inventory exceeds the supported probe limit", ErrAIRegistryConfiguration)
	}

	runtimeTags := map[string]struct{}{metadata.AIOrchestratorTag: {}, metadata.AIResultUploaderTag: {}}
	results, err := probeAITags(ctx, repository, parseOptions, uniqueTags, runtimeTags, registryConfig)
	if err != nil {
		return nil, err
	}
	for _, runtimeTag := range []string{metadata.AIOrchestratorTag, metadata.AIResultUploaderTag} {
		if !results[runtimeTag].found {
			return nil, fmt.Errorf("%w: required private Krkn-AI runtime image is missing", ErrAIRegistryConfiguration)
		}
	}

	available := make([]string, 0, len(scenarioTags)+len(selectedWorkloadTags))
	availableSet := make(map[string]struct{}, cap(available))
	hasScenario := false
	for _, tag := range scenarioTags {
		if results[tag].found {
			if tag != "dummy-scenario" {
				hasScenario = true
			}
			if _, ok := availableSet[tag]; !ok {
				available = append(available, tag)
				availableSet[tag] = struct{}{}
			}
		}
	}
	if !hasScenario {
		return nil, fmt.Errorf("%w: selected private registry has no runnable scenario images", ErrAIRegistryConfiguration)
	}
	for _, role := range workloadRoles {
		tag := selectedWorkloadTags[role]
		if results[tag].found {
			if _, ok := availableSet[tag]; !ok {
				available = append(available, tag)
				availableSet[tag] = struct{}{}
			}
		}
	}
	return &AIImageInventory{
		RegistryName:        secret.Name,
		Repository:          repository.Name(),
		AvailableTags:       available,
		WorkloadImageTags:   selectedWorkloadTags,
		OrchestratorImage:   repository.Name() + "@" + results[metadata.AIOrchestratorTag].digest,
		ResultUploaderImage: repository.Name() + "@" + results[metadata.AIResultUploaderTag].digest,
	}, nil
}

// AIRegistryTagMetadata reads saved registry image tags without exposing Secret data.
func AIRegistryTagMetadata(secret *corev1.Secret) (AITagMetadata, error) {
	if secret == nil {
		return AITagMetadata{}, fmt.Errorf("%w: registry secret is missing", ErrAIRegistryConfiguration)
	}
	metadata := AITagMetadata{
		AIOrchestratorTag:   secret.Annotations[AIOrchestratorTagAnnotation],
		AIResultUploaderTag: secret.Annotations[AIResultUploaderTagAnnotation],
	}
	if encoded := secret.Annotations[WorkloadImageTagsAnnotation]; encoded != "" {
		if err := json.Unmarshal([]byte(encoded), &metadata.WorkloadImageTags); err != nil || metadata.WorkloadImageTags == nil {
			return AITagMetadata{}, fmt.Errorf("%w: workload image tag metadata is invalid", ErrAIRegistryConfiguration)
		}
	}
	return metadata, nil
}

type imageProbeResult struct {
	found  bool
	digest string
}

type imageProbeError struct{ cause error }

func (e *imageProbeError) Error() string { return "private registry image probe failed" }
func (e *imageProbeError) Unwrap() error { return e.cause }

func classifyImageProbeError(err error) error {
	var registryErr *transport.Error
	if errors.As(err, &registryErr) && (registryErr.StatusCode == http.StatusUnauthorized || registryErr.StatusCode == http.StatusForbidden) {
		return fmt.Errorf("%w: registry denied image access", ErrAIRegistryConfiguration)
	}
	return &imageProbeError{cause: err}
}

func probeAITags(ctx context.Context, repository name.Repository, parseOptions []name.Option, tags []string, runtimeTags map[string]struct{}, registryConfig *models.RegistryV2) (map[string]imageProbeResult, error) {
	results := make(map[string]imageProbeResult, len(tags))
	options := verify.OptionsForRegistry(registryConfig)
	keychain := options.Keychain
	if keychain == nil {
		return nil, fmt.Errorf("%w: registry credentials are unavailable", ErrAIRegistryConfiguration)
	}
	remoteOptions := []remote.Option{remote.WithAuthFromKeychain(keychain)}
	remoteOptions = append(remoteOptions, options.RemoteOptions...)
	type probeResult struct {
		tag    string
		result imageProbeResult
		err    error
	}
	probeContext, cancelProbes := context.WithCancel(ctx)
	defer cancelProbes()
	jobs := make(chan string)
	probed := make(chan probeResult, len(tags))
	workers := maxImageProbes
	if len(tags) < workers {
		workers = len(tags)
	}
	for range workers {
		go func() {
			requestOptions := make([]remote.Option, len(remoteOptions)+1)
			copy(requestOptions, remoteOptions)
			for tag := range jobs {
				probeCtx, cancel := context.WithTimeout(probeContext, imageProbeTimeout)
				ref, err := name.NewTag(repository.Name()+":"+tag, parseOptions...)
				if err != nil {
					probed <- probeResult{tag: tag, err: fmt.Errorf("%w: image tag is invalid", ErrAIRegistryConfiguration)}
					cancel()
					continue
				}
				requestOptions[len(remoteOptions)] = remote.WithContext(probeCtx)
				descriptor, err := remote.Get(ref, requestOptions...)
				if err != nil {
					cancel()
					var registryErr *transport.Error
					if errors.As(err, &registryErr) {
						switch registryErr.StatusCode {
						case http.StatusNotFound:
							probed <- probeResult{tag: tag}
							continue
						case http.StatusUnauthorized, http.StatusForbidden:
							probed <- probeResult{tag: tag, err: fmt.Errorf("%w: registry denied image access", ErrAIRegistryConfiguration)}
							continue
						}
					}
					probed <- probeResult{tag: tag, err: &imageProbeError{cause: err}}
					continue
				}
				image, imageErr := descriptor.Image()
				if imageErr != nil {
					if _, indexErr := descriptor.ImageIndex(); indexErr != nil {
						cancel()
						probed <- probeResult{tag: tag, err: &imageProbeError{cause: errors.New("registry returned an invalid OCI manifest")}}
						continue
					}
				}
				if _, runtimeTag := runtimeTags[tag]; runtimeTag {
					if image == nil {
						cancel()
						probed <- probeResult{tag: tag, err: fmt.Errorf("%w: runtime image manifest has no compatible platform", ErrAIRegistryConfiguration)}
						continue
					}
					config, configErr := image.ConfigFile()
					if configErr != nil {
						cancel()
						probed <- probeResult{tag: tag, err: classifyImageProbeError(configErr)}
						continue
					}
					if config.Config.Labels["io.krkn.operator-contract.major"] != "1" {
						cancel()
						probed <- probeResult{tag: tag, err: fmt.Errorf("%w: private runtime image must advertise io.krkn.operator-contract.major=1", ErrAIRegistryConfiguration)}
						continue
					}
				}
				cancel()
				probed <- probeResult{tag: tag, result: imageProbeResult{found: true, digest: descriptor.Digest.String()}}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, tag := range tags {
			select {
			case <-probeContext.Done():
				return
			case jobs <- tag:
			}
		}
	}()
	for range tags {
		select {
		case <-probeContext.Done():
			return nil, &imageProbeError{cause: ctx.Err()}
		case result := <-probed:
			if result.err != nil {
				cancelProbes()
				return nil, result.err
			}
			results[result.tag] = result.result
		}
	}
	return results, nil
}

func validateDefaultWorkloadTags(tags map[string]string) error {
	for _, role := range workloadRoles {
		if tag, ok := tags[role]; !ok || tag == "" || validateOptionalTag(tag) != nil {
			return fmt.Errorf("%w: workload image requirements are missing role %s", ErrAIRegistryConfiguration, role)
		}
	}
	for role := range tags {
		if !isWorkloadRole(role) {
			return fmt.Errorf("%w: workload image requirements contain an unsupported role", ErrAIRegistryConfiguration)
		}
	}
	return nil
}

func isWorkloadRole(role string) bool {
	for _, candidate := range workloadRoles {
		if role == candidate {
			return true
		}
	}
	return false
}

func validateOptionalTag(tag string) error {
	if tag == "" {
		return nil
	}
	_, err := name.NewTag("registry.invalid/krkn/image:"+tag, name.StrictValidation)
	return err
}
