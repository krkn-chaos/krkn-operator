/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package elasticsearch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// queryTimeout bounds a single Elasticsearch search request so a slow or
// unreachable cluster cannot block an API handler indefinitely.
const queryTimeout = 15 * time.Second

// maxResponseBytes caps how much of an Elasticsearch response we will buffer in
// memory. The requested hit count does not bound the size of individual _source
// documents (or of an error body), so a misbehaving or malicious cluster could
// otherwise stream an unbounded response and exhaust memory. Applied to both
// success and error bodies before they are retained or decoded.
const maxResponseBytes = 50 << 20 // 50 MiB

// maxErrorBodySnippet bounds how much of a non-2xx response body is retained for
// server-side diagnostics. Upstream error bodies can contain arbitrary and
// potentially sensitive content, so only a short, log-only snippet is kept.
const maxErrorBodySnippet = 512

// StatusError describes a non-2xx response from the Elasticsearch cluster. It
// carries the HTTP status and a bounded snippet of the response body for
// server-side diagnostics only. Callers should log it to investigate upstream
// failures but MUST NOT forward its contents to API clients, since the body may
// contain sensitive or unstable upstream detail.
type StatusError struct {
	// StatusCode is the HTTP status returned by the cluster.
	StatusCode int
	// Body is a bounded, log-only snippet of the upstream response body.
	Body string
}

// Error implements error. The message is intended for server-side logs, not for
// return to API clients.
func (e *StatusError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("elasticsearch returned status %d", e.StatusCode)
	}
	return fmt.Sprintf("elasticsearch returned status %d: %s", e.StatusCode, e.Body)
}

// Client executes telemetry queries against Elasticsearch/OpenSearch clusters.
// It is safe for concurrent use and should be long-lived (package- or
// handler-scoped): it caches one *http.Transport per distinct TLS configuration
// so that TCP/TLS connections are pooled and reused across requests instead of
// being torn down and re-established for every query.
//
// Transports are cached lazily, keyed by the TLS configuration they carry, so
// clusters that share the same TLS posture (default verification, a given CA
// bundle, or the insecure opt-in) share a connection pool while differing
// configurations remain isolated.
type Client struct {
	mu         sync.Mutex
	transports map[string]*http.Transport
	// doer, when non-nil, executes every request in place of the pooled
	// per-TLS-configuration http.Client. It exists so tests can inject a stub
	// transport; production callers leave it nil to get the secure, connection
	// pooling default.
	doer Doer
}

// Doer executes HTTP requests. *http.Client satisfies it. It is the single
// external collaborator of Client and is exposed so callers (chiefly tests) can
// substitute a stub without reaching real network endpoints.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Option configures a Client at construction time.
type Option func(*Client)

// WithHTTPClient injects a custom Doer used for every request, bypassing the
// pooled per-TLS-configuration transport. Intended for tests; production callers
// should omit it to retain secure TLS defaults and connection pooling.
func WithHTTPClient(d Doer) Option {
	return func(c *Client) { c.doer = d }
}

// NewClient returns a ready-to-use Client. A single instance should be created
// once and shared for the lifetime of the process to benefit from connection
// pooling. By default it uses a pooled http.Client per TLS configuration with
// secure production defaults; pass WithHTTPClient to inject a custom Doer.
func NewClient(opts ...Option) *Client {
	c := &Client{transports: make(map[string]*http.Transport)}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// transportKey derives a stable cache key from the TLS-affecting fields of
// conn. Connections only differ in behavior by their TLS configuration, so
// host/port/credentials are deliberately excluded: a single pooled transport
// can safely serve many hosts that share the same TLS posture.
func transportKey(conn ConnectionParams) string {
	if conn.InsecureSkipVerify {
		return "insecure"
	}
	if conn.CACert != "" {
		sum := sha256.Sum256([]byte(conn.CACert))
		return "ca:" + hex.EncodeToString(sum[:])
	}
	return "default"
}

// transport returns a cached transport for conn's TLS configuration, creating
// and caching one on first use. It is safe for concurrent callers.
func (c *Client) transport(conn ConnectionParams) (*http.Transport, error) {
	key := transportKey(conn)

	c.mu.Lock()
	defer c.mu.Unlock()
	if t, ok := c.transports[key]; ok {
		return t, nil
	}

	tlsConfig, err := conn.tlsConfig()
	if err != nil {
		return nil, err
	}
	// Clone the stdlib default transport so we inherit its connection-pool and
	// timeout defaults, then attach the per-configuration TLS settings. The type
	// assertion can fail if http.DefaultTransport has been replaced with a
	// non-*http.Transport (e.g. by a test or dependency); fall back to a fresh
	// *http.Transport rather than dereferencing a nil pointer in Clone.
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		base = &http.Transport{}
	}
	transport := base.Clone()
	transport.TLSClientConfig = tlsConfig
	c.transports[key] = transport
	return transport, nil
}

// ConnectionParams holds the resolved connection details needed to query an
// Elasticsearch/OpenSearch cluster. It is assembled server-side from a stored
// config Secret; credentials never cross the API boundary to the client.
type ConnectionParams struct {
	// Host is the cluster host. It may be a bare hostname or a full URL
	// including scheme (e.g. "https://es.example.com"); see baseURL for how the
	// scheme and Port are applied.
	Host string
	// Port is the cluster port, appended to Host when Host does not already
	// specify one.
	Port int
	// Username is the basic-auth username. When empty, no credentials are sent.
	Username string
	// Password is the basic-auth password used together with Username.
	Password string
	// Index is the name of the index to search (e.g. the telemetry index).
	Index string
	// CACert is an optional PEM-encoded certificate (or bundle) to trust in
	// addition to the system roots. It is the preferred way to connect to a
	// self-signed cluster: verification stays on, but the custom CA is honored.
	CACert string
	// InsecureSkipVerify disables TLS certificate verification entirely when
	// true. It is a last-resort, explicit opt-in for self-signed telemetry
	// clusters where no CA material is available; the default (false) verifies
	// the server certificate. Prefer CACert over this.
	InsecureSkipVerify bool
	// RestrictDestination, when true, subjects this connection to the inline
	// destination policy (see ssrf.go): the target scheme/port is enforced, the
	// host is DNS-resolved and every resolved address is rejected if it is
	// loopback, private, link-local, metadata, multicast, or unspecified, and
	// redirects are re-validated per hop. It is set for user-supplied inline
	// connections, which any authenticated user can request, to prevent
	// server-side request forgery. Admin-created saved configs leave it false
	// because they legitimately point at trusted in-cluster (private) clusters.
	RestrictDestination bool
}

// esSearchResponse mirrors the subset of the Elasticsearch _search response we
// consume. Each hit's _source is kept raw so it can be decoded into the raw
// telemetry shape and then flattened into a TelemetryDocument.
type esSearchResponse struct {
	Hits struct {
		Hits []struct {
			Source json.RawMessage `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
	// Aggregations carries the by_job_status terms aggregation used to summarize
	// pass/fail across the whole matched window (independent of the hits size cap).
	Aggregations struct {
		ByJobStatus struct {
			Buckets []struct {
				// KeyAsString is "true"/"false" for the boolean job_status field.
				KeyAsString string `json:"key_as_string"`
				DocCount    int    `json:"doc_count"`
			} `json:"buckets"`
		} `json:"by_job_status"`
	} `json:"aggregations"`
}

// rawTelemetrySource mirrors the subset of a krkn telemetry document _source we
// need to populate the table columns. Scenario-level fields (type, start/end,
// namespace) live inside the scenarios array; we surface the run's first
// scenario for the flattened row.
type rawTelemetrySource struct {
	RunUUID   string `json:"run_uuid"`
	JobStatus bool   `json:"job_status"`
	// Run-level cluster/infrastructure metadata. Pointer fields (bool, int)
	// distinguish absent (nil) from explicit false/zero.
	KubernetesObjectsCount map[string]int    `json:"kubernetes_objects_count"`
	NetworkPlugins         []string          `json:"network_plugins"`
	TotalNodeCount         *int              `json:"total_node_count"`
	CloudInfrastructure    string            `json:"cloud_infrastructure"`
	CloudType              string            `json:"cloud_type"`
	ClusterVersion         string            `json:"cluster_version"`
	MajorVersion           string            `json:"major_version"`
	BuildURL               string            `json:"build_url"`
	FIPSEnabled            *bool             `json:"fips_enabled"`
	Tag                    string            `json:"tag"`
	EtcdEncryptionEnabled  *bool             `json:"etcd_encryption_enabled"`
	IPSecEnabled           *bool             `json:"ipsec_enabled"`
	NodeSummaryInfos       []NodeSummaryInfo `json:"node_summary_infos"`
	Scenarios              []struct {
		ScenarioType   string  `json:"scenario_type"`
		StartTimestamp float64 `json:"start_timestamp"`
		EndTimestamp   float64 `json:"end_timestamp"`
		ExitStatus     int     `json:"exit_status"`
		// Parameters shape varies by scenario type (object keyed by scenario
		// name, whose value may be an object or an array), so it is kept raw and
		// searched for a namespace rather than decoded into a fixed struct.
		Parameters   json.RawMessage `json:"parameters"`
		AffectedPods *AffectedPods   `json:"affected_pods"`
	} `json:"scenarios"`
}

// flatten converts a raw telemetry source into the fixed TelemetryDocument
// surfaced to the UI, deriving scenario-level columns from the first scenario.
func (s rawTelemetrySource) flatten() TelemetryDocument {
	doc := TelemetryDocument{
		RunUUID:  s.RunUUID,
		Status:   s.JobStatus,
		Metadata: s.metadata(),
	}
	if len(s.Scenarios) > 0 {
		sc := s.Scenarios[0]
		doc.ScenarioType = sc.ScenarioType
		doc.StartTimestamp = sc.StartTimestamp
		doc.EndTimestamp = sc.EndTimestamp
		// A non-zero exit status marks a failed scenario even if the overall
		// job reported success.
		if sc.ExitStatus != 0 {
			doc.Status = false
		}
		doc.Namespace = namespaceFromParameters(sc.Parameters)

		// Surface every scenario with its raw parameters for the expanded row.
		// Redact sensitive keys (passwords, tokens, etc.) before exposing to clients.
		doc.Scenarios = make([]ScenarioDetail, 0, len(s.Scenarios))
		for _, scn := range s.Scenarios {
			doc.Scenarios = append(doc.Scenarios, ScenarioDetail{
				ScenarioType:   scn.ScenarioType,
				StartTimestamp: scn.StartTimestamp,
				EndTimestamp:   scn.EndTimestamp,
				ExitStatus:     scn.ExitStatus,
				Parameters:     redactSensitiveParameters(scn.Parameters),
				AffectedPods:   scn.AffectedPods,
			})
		}
	}
	return doc
}

// metadata builds the run-level ClusterMetadata from the raw source, returning
// nil when the source carried none of the metadata fields so the JSON response
// omits an empty object. Pointer fields (bool, int) preserve the distinction
// between absent (nil) and explicit false/zero.
func (s rawTelemetrySource) metadata() *ClusterMetadata {
	empty := len(s.KubernetesObjectsCount) == 0 &&
		len(s.NetworkPlugins) == 0 &&
		s.TotalNodeCount == nil &&
		s.CloudInfrastructure == "" &&
		s.CloudType == "" &&
		s.ClusterVersion == "" &&
		s.MajorVersion == "" &&
		s.BuildURL == "" &&
		s.FIPSEnabled == nil &&
		s.Tag == "" &&
		s.EtcdEncryptionEnabled == nil &&
		s.IPSecEnabled == nil &&
		len(s.NodeSummaryInfos) == 0
	if empty {
		return nil
	}
	return &ClusterMetadata{
		KubernetesObjectsCount: s.KubernetesObjectsCount,
		NetworkPlugins:         s.NetworkPlugins,
		TotalNodeCount:         s.TotalNodeCount,
		CloudInfrastructure:    s.CloudInfrastructure,
		CloudType:              s.CloudType,
		ClusterVersion:         s.ClusterVersion,
		MajorVersion:           s.MajorVersion,
		BuildURL:               s.BuildURL,
		FIPSEnabled:            s.FIPSEnabled,
		Tag:                    s.Tag,
		EtcdEncryptionEnabled:  s.EtcdEncryptionEnabled,
		IPSecEnabled:           s.IPSecEnabled,
		NodeSummaryInfos:       s.NodeSummaryInfos,
	}
}

// sensitiveKeys lists parameter keys that may contain secrets and must be
// redacted before exposing scenario parameters to API clients. Keys are matched
// case-insensitively to catch variations in casing conventions.
var sensitiveKeys = map[string]bool{
	"password":            true,
	"passwd":              true,
	"token":               true,
	"api_key":             true,
	"apikey":              true,
	"secret":              true,
	"secret_key":          true,
	"access_key":          true,
	"private_key":         true,
	"credential":          true,
	"credentials":         true,
	"aws_secret_access_key": true,
	"aws_access_key_id":     true,
	"os_password":           true,
	"azure_client_secret":   true,
	"azure_client_id":       true,
	"gcp_service_account":   true,
	"auth_token":            true,
	"bearer_token":          true,
	"session_token":         true,
	"cookie":                true,
}

// redactSensitiveParameters walks a scenario parameters JSON blob and replaces
// values for sensitive keys with "***REDACTED***". Returns the redacted JSON or
// the original raw message if redaction fails (so a malformed parameters blob
// doesn't crash the query). Preserves the original JSON if no redaction occurred.
func redactSensitiveParameters(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		// Return original if unmarshal fails (malformed JSON)
		return raw
	}
	redacted, changed := redactValueWithTracking(v)
	if !changed {
		// Return original JSON unchanged to preserve key ordering
		return raw
	}
	result, err := json.Marshal(redacted)
	if err != nil {
		// Return original if re-marshal fails
		return raw
	}
	return result
}

// redactValueWithTracking recursively walks a decoded JSON value and replaces
// values for sensitive keys with "***REDACTED***". Returns the redacted value
// and a boolean indicating whether any redaction occurred.
func redactValueWithTracking(v any) (any, bool) {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		changed := false
		for k, nested := range val {
			// Check if this key is sensitive (case-insensitive)
			if isSensitiveKey(k) {
				result[k] = "***REDACTED***"
				changed = true
			} else {
				redacted, nestedChanged := redactValueWithTracking(nested)
				result[k] = redacted
				if nestedChanged {
					changed = true
				}
			}
		}
		return result, changed
	case []any:
		result := make([]any, len(val))
		changed := false
		for i, elem := range val {
			redacted, elemChanged := redactValueWithTracking(elem)
			result[i] = redacted
			if elemChanged {
				changed = true
			}
		}
		return result, changed
	default:
		// Scalar values (string, number, bool, null) are returned as-is
		return val, false
	}
}

// isSensitiveKey checks if a parameter key name matches a sensitive pattern.
// Matching is case-insensitive to catch variations like "Password", "PASSWORD".
func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	return sensitiveKeys[lower]
}

// namespaceFromParameters extracts the target namespace from a scenario's raw
// parameters JSON. The parameters shape varies by scenario type, so the value
// tree is walked recursively for the first "namespace"/"namespace_pattern" key.
func namespaceFromParameters(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return ""
	}
	return findNamespace(v)
}

// findNamespace recursively searches a decoded JSON value for the first non-empty
// "namespace" (or "namespace_pattern") string, checking those keys before
// descending into nested objects and arrays.
func findNamespace(v any) string {
	switch t := v.(type) {
	case map[string]any:
		for _, key := range []string{"namespace", "namespace_pattern"} {
			if s, ok := t[key].(string); ok && s != "" {
				return s
			}
		}
		for _, val := range t {
			if s := findNamespace(val); s != "" {
				return s
			}
		}
	case []any:
		for _, val := range t {
			if s := findNamespace(val); s != "" {
				return s
			}
		}
	}
	return ""
}

// baseURL builds the cluster base URL. If the host already carries a scheme it
// is parsed as a URL and the separately configured port is appended only when
// the host does not already specify one; otherwise https is assumed and the
// port is appended.
func (c ConnectionParams) baseURL() string {
	host := strings.TrimSuffix(c.Host, "/")
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		// If the host already includes a port, or it cannot be parsed, or no
		// port is configured, use it as-is. Otherwise append the configured port.
		if u, err := url.Parse(host); err == nil && u.Port() == "" && c.Port != 0 {
			u.Host = fmt.Sprintf("%s:%d", u.Host, c.Port)
			return strings.TrimSuffix(u.String(), "/")
		}
		return host
	}
	return fmt.Sprintf("https://%s:%d", host, c.Port)
}

// tlsConfig builds the TLS configuration for the cluster connection. By default
// the server certificate is verified against the system roots. A PEM-encoded
// CACert, when provided, is trusted in addition to the system roots so
// self-signed clusters can be reached without disabling verification.
// InsecureSkipVerify is honored only as an explicit last resort and takes
// precedence, in which case no CA material is needed.
func (c ConnectionParams) tlsConfig() (*tls.Config, error) {
	if c.InsecureSkipVerify {
		// Verification is on by default; this path is only reached when an
		// operator explicitly opts into InsecureSkipVerify for a self-signed
		// telemetry cluster with no CA material available.
		return &tls.Config{InsecureSkipVerify: true}, nil // #nosec G402 -- explicit, restricted opt-in
	}

	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if c.CACert != "" {
		// Start from the system roots so the custom CA is additive: a cluster
		// presenting a publicly trusted chain still verifies even when it is not
		// part of the supplied bundle. SystemCertPool can fail (e.g. on a minimal
		// image without a trust store); fall back to an empty pool holding only
		// the supplied CA rather than failing the request.
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM([]byte(c.CACert)) {
			return nil, fmt.Errorf("failed to parse CA certificate: no valid PEM certificates found")
		}
		cfg.RootCAs = pool
	}
	return cfg, nil
}

// QueryTelemetry connects to the Elasticsearch/OpenSearch cluster described by
// conn and returns telemetry documents from conn.Index. size is clamped to the
// supported bounds by the caller. startDate and endDate ("yyyy-MM-dd") bound the
// search by document timestamp; empty values default to a trailing 30-day
// window. Results are sorted newest-first by timestamp (with a deterministic
// _doc tie-breaker) before the size limit is applied, so the most recent
// documents are the ones returned.
//
// The call is bounded by both ctx and an internal queryTimeout (whichever fires
// first); on timeout or cancellation it returns a non-nil error wrapping the
// context cause and no documents.
//
// It returns a non-nil error, and no documents, when:
//   - conn.Index is empty (the config has no telemetry index);
//   - conn carries credentials but resolves to a plaintext http:// URL
//     (credentials are refused rather than sent in the clear);
//   - the configured CACert cannot be parsed into a usable TLS config;
//   - the cluster is unreachable or the request otherwise fails in transit
//     (including ctx timeout/cancellation);
//   - the response body exceeds maxResponseBytes;
//   - the cluster responds with a non-2xx status: the error is a *StatusError
//     carrying the status and a bounded, log-only body snippet (use errors.As);
//   - the top-level response body is not valid JSON in the expected shape.
//
// Partial results are tolerated on success: individual hits whose _source does
// not unmarshal into the expected telemetry shape are skipped rather than
// failing the whole query, so the returned slice may contain fewer documents
// than the cluster reported hits. On success with no matching hits it returns a
// non-nil, empty (len 0) slice and a nil error.
// It also returns a TelemetryStats summary of run-level pass/fail counts derived
// from a terms aggregation on job_status. The aggregation runs over every document
// matching the time-range filter, so the summary spans the whole matched window
// regardless of size (Stats.Pass + Stats.Fail can exceed len(docs)). On any error
// the returned stats are the zero value.
func (c *Client) QueryTelemetry(ctx context.Context, conn ConnectionParams, size int, startDate, endDate string) (docs []TelemetryDocument, stats TelemetryStats, err error) {
	if conn.Index == "" {
		return nil, TelemetryStats{}, fmt.Errorf("telemetry index is not configured for this Elasticsearch config")
	}

	base := conn.baseURL()
	// Never send credentials over plaintext HTTP where they could be observed on
	// the wire. Require TLS whenever a username/password is configured.
	if conn.Username != "" && strings.HasPrefix(base, "http://") {
		return nil, TelemetryStats{}, fmt.Errorf("refusing to send credentials over plaintext HTTP; use https")
	}

	// Inline (user-supplied) connections are subject to the destination policy.
	// Validate before any outbound request so a request that targets a
	// disallowed address is rejected without probing it. The dial-time guard in
	// resolveDoer re-checks the resolved address to close the DNS-rebinding gap.
	if conn.RestrictDestination {
		if err := validateInlineDestination(ctx, base); err != nil {
			return nil, TelemetryStats{}, err
		}
	}

	doer, err := c.resolveDoer(conn)
	if err != nil {
		return nil, TelemetryStats{}, err
	}

	payload, err := buildSearchBody(size, startDate, endDate)
	if err != nil {
		return nil, TelemetryStats{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	req, err := newSearchRequest(ctx, base, conn, payload)
	if err != nil {
		return nil, TelemetryStats{}, err
	}

	respBody, err := executeSearch(doer, req)
	if err != nil {
		return nil, TelemetryStats{}, err
	}

	return decodeTelemetry(respBody)
}

// resolveDoer selects the request executor for conn: an injected Doer (tests)
// takes precedence; otherwise it wraps the pooled per-TLS-configuration
// transport in a timeout-bounded http.Client.
func (c *Client) resolveDoer(conn ConnectionParams) (Doer, error) {
	if c.doer != nil {
		return c.doer, nil
	}
	transport, err := c.transport(conn)
	if err != nil {
		return nil, err
	}
	// Restricted (inline) connections get a dedicated, non-pooled transport with
	// a validating dialer and a redirect policy, so the destination guard applies
	// to the actual connection and to every redirect hop. It is not shared via
	// the TLS-keyed pool because that pool is intentionally host-agnostic. Inline
	// queries are ad-hoc, so forgoing connection reuse here is acceptable.
	if conn.RestrictDestination {
		guarded := transport.Clone()
		guarded.DialContext = guardedDialContext()
		return &http.Client{
			Timeout:       queryTimeout,
			Transport:     guarded,
			CheckRedirect: guardedCheckRedirect,
		}, nil
	}
	// The http.Client is cheap; the pooled transport it wraps is what carries
	// (and reuses) the underlying connections across queries.
	return &http.Client{
		Timeout:   queryTimeout,
		Transport: transport,
	}, nil
}

// buildSearchBody marshals the _search request body for the given size and date
// bounds. Results are sorted newest-first so the size limit keeps the most
// recent documents. startDate/endDate are "yyyy-MM-dd"; empty values default to
// a trailing 30-day window.
func buildSearchBody(size int, startDate, endDate string) ([]byte, error) {
	// Lower bound: default to the start of the day 30 days ago; an explicit
	// startDate is parsed via the "yyyy-MM-dd" format below.
	gte := "now-30d/d"
	if startDate != "" {
		gte = startDate
	}

	// Upper bound: the default window must include telemetry through the current
	// instant, so use an inclusive "now" rather than "now/d" (which is midnight
	// at the start of today and would drop everything logged so far today). An
	// explicit endDate must include only that calendar day, so use an exclusive
	// "lt" at the following midnight ("+1d/d"); an inclusive "lte" there would
	// also match documents timestamped exactly at the next day's boundary.
	timestampRange := map[string]any{
		"format": "yyyy-MM-dd",
		"gte":    gte,
	}
	if endDate != "" {
		timestampRange["lt"] = endDate + "||+1d/d"
	} else {
		timestampRange["lte"] = "now"
	}

	body := map[string]any{
		"size": size,
		"from": 0,
		// Sort newest-first by the same timestamp field the range filter uses so
		// the size limit keeps the most recent documents. unmapped_type keeps the
		// request from failing on indices where timestamp is not mapped, and the
		// _doc tie-breaker makes ordering deterministic across equal timestamps.
		"sort": []any{
			map[string]any{
				"timestamp": map[string]any{
					"order":         "desc",
					"unmapped_type": "date",
				},
			},
			map[string]any{"_doc": map[string]any{"order": "asc"}},
		},
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []any{
					map[string]any{
						"range": map[string]any{
							"timestamp": timestampRange,
						},
					},
				},
			},
		},
		// Aggregate run-level job_status across the whole matched window so the
		// pass/fail summary is independent of the hits size limit. A boolean field
		// yields at most two buckets ("true"/"false").
		"aggs": map[string]any{
			"by_job_status": map[string]any{
				"terms": map[string]any{
					"field": "job_status",
					"size":  10,
				},
			},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode search query: %w", err)
	}
	return payload, nil
}

// newSearchRequest builds the POST _search request against conn.Index, setting
// the JSON content type and basic-auth credentials when configured. The request
// is bound to ctx.
func newSearchRequest(ctx context.Context, base string, conn ConnectionParams, payload []byte) (*http.Request, error) {
	url := fmt.Sprintf("%s/%s/_search", base, conn.Index)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to build search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if conn.Username != "" {
		req.SetBasicAuth(conn.Username, conn.Password)
	}
	return req, nil
}

// executeSearch runs req, enforces the response-size cap, and validates the HTTP
// status. It returns the raw success body (2xx); a non-2xx status yields a
// *StatusError carrying a bounded, log-only body snippet.
func executeSearch(doer Doer, req *http.Request) (body []byte, err error) {
	resp, err := doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach Elasticsearch: %w", err)
	}
	// Propagate a body-close failure, but never let it mask an error from the
	// query itself: only surface the close error when the function is otherwise
	// returning successfully.
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close Elasticsearch response body: %w", closeErr)
		}
	}()

	// Read at most maxResponseBytes+1 so an oversized body is detected without
	// buffering the whole thing: the extra byte tips len() over the limit.
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read Elasticsearch response: %w", err)
	}
	if int64(len(respBody)) > maxResponseBytes {
		return nil, fmt.Errorf("elasticsearch response exceeds the maximum supported size of %d bytes", maxResponseBytes)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Keep only a bounded snippet of the upstream body for diagnostics; the
		// full body is neither logged nor returned to the caller.
		snippet := strings.TrimSpace(string(respBody))
		if len(snippet) > maxErrorBodySnippet {
			snippet = snippet[:maxErrorBodySnippet] + "…(truncated)"
		}
		return nil, &StatusError{StatusCode: resp.StatusCode, Body: snippet}
	}

	return respBody, nil
}

// decodeTelemetry parses a successful _search body and flattens each hit into a
// TelemetryDocument. Hits whose _source does not match the expected telemetry
// shape are skipped rather than failing the whole query, so the returned slice
// may be shorter than the reported hit count; on no hits it is non-nil and empty.
func decodeTelemetry(respBody []byte) (docs []TelemetryDocument, stats TelemetryStats, err error) {
	var parsed esSearchResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, TelemetryStats{}, fmt.Errorf("failed to decode Elasticsearch response: %w", err)
	}

	docs = make([]TelemetryDocument, 0, len(parsed.Hits.Hits))
	for _, hit := range parsed.Hits.Hits {
		var src rawTelemetrySource
		if err := json.Unmarshal(hit.Source, &src); err != nil {
			// Skip documents that don't match the expected telemetry shape
			// rather than failing the whole query.
			continue
		}
		docs = append(docs, src.flatten())
	}

	stats = statsFromBuckets(parsed)

	return docs, stats, nil
}

// statsFromBuckets derives the run-level pass/fail summary from the by_job_status
// terms aggregation. A boolean field yields "true"/"false" buckets; any other key
// is ignored. PassPercent is the percentage of passing runs (0-100, rounded to two
// decimals) and is 0 when no runs matched, avoiding a divide-by-zero.
func statsFromBuckets(parsed esSearchResponse) TelemetryStats {
	var stats TelemetryStats
	for _, b := range parsed.Aggregations.ByJobStatus.Buckets {
		switch b.KeyAsString {
		case "true":
			stats.Pass = b.DocCount
		case "false":
			stats.Fail = b.DocCount
		}
	}
	if total := stats.Pass + stats.Fail; total > 0 {
		stats.PassPercent = math.Round(float64(stats.Pass)/float64(total)*10000) / 100
	}
	return stats
}
