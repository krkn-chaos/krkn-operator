package identity

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"sync"
)

const maxCachedProviderHTTPClients = 128

var providerHTTPClientCache = struct {
	sync.Mutex
	clients map[providerHTTPClientKey]*http.Client
}{clients: make(map[providerHTTPClientKey]*http.Client)}

type providerHTTPClientKey struct {
	caHash [sha256.Size]byte
	skip   bool
}

// NewHTTPClient returns an HTTP client with the requested provider TLS trust.
// caPEM is appended to the host's system roots. When skipVerification is true,
// certificate verification is disabled; use this only for isolated tests.
// With no custom settings, the shared default client is returned unchanged.
func NewHTTPClient(caPEM string, skipVerification bool) (*http.Client, error) {
	if caPEM == "" && !skipVerification {
		return http.DefaultClient, nil
	}
	if caPEM != "" {
		if err := ValidateCACert(caPEM); err != nil {
			return nil, err
		}
	}
	key := providerHTTPClientKey{caHash: sha256.Sum256([]byte(caPEM)), skip: skipVerification}
	providerHTTPClientCache.Lock()
	defer providerHTTPClientCache.Unlock()
	if cached := providerHTTPClientCache.clients[key]; cached != nil {
		return cached, nil
	}

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if caPEM != "" {
		roots, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("load system certificate roots: %w", err)
		}
		if roots == nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM([]byte(caPEM)) {
			return nil, fmt.Errorf("CA certificate contains no valid PEM certificates")
		}
		tlsConfig.RootCAs = roots
	}
	// #nosec G402 -- insecure mode is an explicit per-provider opt-in for isolated tests.
	tlsConfig.InsecureSkipVerify = skipVerification

	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("default HTTP transport is not an *http.Transport")
	}
	transport := baseTransport.Clone()
	transport.TLSClientConfig = tlsConfig
	client := &http.Client{Transport: transport}
	if len(providerHTTPClientCache.clients) >= maxCachedProviderHTTPClients {
		for evictedKey, evictedClient := range providerHTTPClientCache.clients {
			evictedClient.CloseIdleConnections()
			delete(providerHTTPClientCache.clients, evictedKey)
			break
		}
	}
	providerHTTPClientCache.clients[key] = client
	return client, nil
}

// ValidateCACert checks that caPEM contains at least one PEM certificate.
func ValidateCACert(caPEM string) error {
	if caPEM == "" {
		return nil
	}
	certificates := x509.NewCertPool()
	if !certificates.AppendCertsFromPEM([]byte(caPEM)) {
		return fmt.Errorf("CA certificate contains no valid PEM certificates")
	}
	return nil
}
