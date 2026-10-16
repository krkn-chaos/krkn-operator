// Package krknaiserver provides an authenticated client for the Krkn-AI artifact service.
package krknaiserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client sends authenticated requests to the Krkn-AI artifact service.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// New returns an artifact-service client with a bounded request timeout.
func New(baseURL, token string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Do sends one authenticated request to path. Callers own and must close the
// response body.
func (c *Client) Do(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
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
	return c.httpClient.Do(req)
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
