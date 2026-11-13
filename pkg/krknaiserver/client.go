// Package krknaiserver provides an authenticated client for the Krkn-AI artifact service.
package krknaiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/krkn-chaos/krkn-operator/pkg/registry"
)

// Client sends authenticated requests to the Krkn-AI artifact service.
type Client struct {
	baseURL         string
	token           string
	httpClient      *http.Client
	discoveryClient *http.Client
}

// DefaultRequestTimeout bounds ordinary artifact-service requests.
const DefaultRequestTimeout = 30 * time.Second

// DefaultDiscoveryTimeout bounds synchronous discovery at its usage sites.
const DefaultDiscoveryTimeout = 5 * time.Minute

// New returns an artifact-service client with bounded artifact and discovery
// request timeouts.
func New(baseURL, token string) *Client {
	return &Client{
		baseURL:         strings.TrimRight(baseURL, "/"),
		token:           token,
		httpClient:      &http.Client{Timeout: DefaultRequestTimeout},
		discoveryClient: &http.Client{Timeout: DefaultDiscoveryTimeout},
	}
}

// Do sends one authenticated artifact-service request to path. Callers own and
// must close the response body.
func (c *Client) Do(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	return c.do(c.httpClient, ctx, method, path, body)
}

// DoDiscovery sends one authenticated discovery request with the fixed
// discovery timeout.
func (c *Client) DoDiscovery(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	return c.do(c.discoveryClient, ctx, method, path, body)
}

func (c *Client) do(httpClient *http.Client, ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	if c.baseURL == "" || c.token == "" {
		return nil, fmt.Errorf("Krkn-AI artifact service is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	return httpClient.Do(req)
}

// ImageRequirements describes the canonical Krkn-AI image catalog.
type ImageRequirements struct {
	ScenarioTags      []string          `json:"scenarioTags"`
	WorkloadImageTags map[string]string `json:"workloadImageTags"`
}

// ImageRequirements retrieves the canonical private-registry image catalog.
func (c *Client) ImageRequirements(ctx context.Context) (ImageRequirements, error) {
	response, err := c.Do(ctx, http.MethodGet, "/v1/images/requirements", nil)
	if err != nil {
		return ImageRequirements{}, fmt.Errorf("request image requirements: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ImageRequirements{}, fmt.Errorf("image requirements service returned HTTP %d", response.StatusCode)
	}
	var requirements ImageRequirements
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&requirements); err != nil {
		return ImageRequirements{}, errors.New("decode image requirements response")
	}
	return requirements, nil
}

// ErrInvalidConfig reports that the Krkn-AI service rejected a YAML config.
var ErrInvalidConfig = errors.New("Krkn-AI configuration is invalid")

// ConfigValidationIssue is one safe validation detail from the service.
type ConfigValidationIssue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// ConfigValidationError carries structured validation details for 422 responses.
type ConfigValidationError struct {
	Errors []ConfigValidationIssue `json:"errors,omitempty"`
}

func (e *ConfigValidationError) Error() string { return ErrInvalidConfig.Error() }

func (e *ConfigValidationError) Unwrap() error { return ErrInvalidConfig }

// ValidateConfigImages asks the Krkn-AI service to validate YAML against a
// private registry inventory. The inventory contains no registry credentials.
func (c *Client) ValidateConfigImages(ctx context.Context, configYAML string, inventory *registry.AIImageInventory) error {
	if err := ValidateScenarioSelection(configYAML); err != nil {
		return err
	}
	registryName := ""
	if inventory != nil {
		registryName = inventory.RegistryName
	}
	payload, err := json.Marshal(struct {
		ConfigYAML     string                     `json:"configYaml"`
		RegistryName   string                     `json:"registryName,omitempty"`
		ImageInventory *registry.AIImageInventory `json:"imageInventory"`
	}{ConfigYAML: configYAML, RegistryName: registryName, ImageInventory: inventory})
	if err != nil {
		return errors.New("encode image validation request")
	}
	response, err := c.Do(ctx, http.MethodPost, "/v1/configs/validate", payload)
	if err != nil {
		return fmt.Errorf("request image validation: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnprocessableEntity {
		var validation ConfigValidationError
		if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&validation); err != nil {
			return &ConfigValidationError{}
		}
		return &validation
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("image validation service returned HTTP %d", response.StatusCode)
	}
	var validation struct {
		Valid bool `json:"valid"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&validation); err != nil {
		return errors.New("decode image validation response")
	}
	if !validation.Valid {
		return ErrInvalidConfig
	}
	return nil
}

// EscapePath URL-escapes each segment while preserving path separators.
func EscapePath(path string) string {
	parts := strings.Split(path, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

// ReadBody reads and closes an artifact-service response body.
func ReadBody(response *http.Response) ([]byte, error) {
	defer response.Body.Close()
	return io.ReadAll(response.Body)
}
