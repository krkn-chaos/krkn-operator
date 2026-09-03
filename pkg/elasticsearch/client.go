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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// queryTimeout bounds a single Elasticsearch search request so a slow or
// unreachable cluster cannot block an API handler indefinitely.
const queryTimeout = 15 * time.Second

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
}

// rawTelemetrySource mirrors the subset of a krkn telemetry document _source we
// need to populate the table columns. Scenario-level fields (type, start/end,
// namespace) live inside the scenarios array; we surface the run's first
// scenario for the flattened row.
type rawTelemetrySource struct {
	RunUUID   string `json:"run_uuid"`
	JobStatus bool   `json:"job_status"`
	Scenarios []struct {
		ScenarioType   string `json:"scenario_type"`
		StartTimestamp int64  `json:"start_timestamp"`
		EndTimestamp   int64  `json:"end_timestamp"`
		ExitStatus     int    `json:"exit_status"`
		// Parameters shape varies by scenario type (object keyed by scenario
		// name, whose value may be an object or an array), so it is kept raw and
		// searched for a namespace rather than decoded into a fixed struct.
		Parameters json.RawMessage `json:"parameters"`
	} `json:"scenarios"`
}

// flatten converts a raw telemetry source into the fixed TelemetryDocument
// surfaced to the UI, deriving scenario-level columns from the first scenario.
func (s rawTelemetrySource) flatten() TelemetryDocument {
	doc := TelemetryDocument{
		RunUUID: s.RunUUID,
		Status:  s.JobStatus,
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
	}
	return doc
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
		pool := x509.NewCertPool()
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
func QueryTelemetry(ctx context.Context, conn ConnectionParams, size int, startDate, endDate string) (docs []TelemetryDocument, err error) {
	if conn.Index == "" {
		return nil, fmt.Errorf("telemetry index is not configured for this Elasticsearch config")
	}

	base := conn.baseURL()
	// Never send credentials over plaintext HTTP where they could be observed on
	// the wire. Require TLS whenever a username/password is configured.
	if conn.Username != "" && strings.HasPrefix(base, "http://") {
		return nil, fmt.Errorf("refusing to send credentials over plaintext HTTP; use https")
	}

	tlsConfig, err := conn.tlsConfig()
	if err != nil {
		return nil, err
	}

	// Default to a trailing 30-day window when a bound is not supplied. The
	// endDate is made inclusive of the whole selected day via date-math rounding.
	gte := "now-30d/d"
	if startDate != "" {
		gte = startDate
	}
	lte := "now/d"
	if endDate != "" {
		lte = endDate + "||+1d/d"
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
							"timestamp": map[string]any{
								"format": "yyyy-MM-dd",
								"gte":    gte,
								"lte":    lte,
							},
						},
					},
				},
			},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode search query: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_search", base, conn.Index)

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to build search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if conn.Username != "" {
		req.SetBasicAuth(conn.Username, conn.Password)
	}

	client := &http.Client{
		Timeout: queryTimeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	resp, err := client.Do(req)
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Elasticsearch response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Elasticsearch returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed esSearchResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("failed to decode Elasticsearch response: %w", err)
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

	return docs, nil
}
