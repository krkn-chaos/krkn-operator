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

package api

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// defaultMaxRateLimiterEntries bounds the number of per-IP limiters retained in
// memory. When the cap is reached, the least-recently-seen entry is evicted
// before a new one is created. This prevents a burst of unique client keys from
// growing the map faster than the periodic cleanup can reclaim it.
const defaultMaxRateLimiterEntries = 50000

// ipRateLimiter provides per-IP rate limiting for HTTP endpoints.
// It maintains a map of rate limiters keyed by client IP address,
// with automatic cleanup of stale entries to prevent memory leaks.
type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rateLimiterEntry
	rate     rate.Limit
	burst    int
	// trustedProxies lists CIDR ranges whose members are allowed to set
	// forwarding headers (X-Forwarded-For / X-Real-IP). Requests arriving
	// directly from any other peer have those headers ignored. Nil/empty
	// means no proxy is trusted and RemoteAddr is always used.
	trustedProxies []*net.IPNet
	// maxEntries caps the size of the limiters map (0 disables the cap).
	maxEntries int
}

// rateLimiterEntry holds a rate limiter and its last-seen timestamp
// for cleanup of stale entries.
type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// newIPRateLimiter creates a new per-IP rate limiter.
//
// Parameters:
//   - r: the rate limit (events per second). Use rate.Every(d) for time-based rates.
//   - b: the burst size (maximum number of events allowed in a single burst).
//
// Returns a new ipRateLimiter instance. By default no proxies are trusted;
// configure trusted proxies with setTrustedProxies to honor forwarding headers.
func newIPRateLimiter(r rate.Limit, b int) *ipRateLimiter {
	rl := &ipRateLimiter{
		limiters:   make(map[string]*rateLimiterEntry),
		rate:       r,
		burst:      b,
		maxEntries: defaultMaxRateLimiterEntries,
	}
	return rl
}

// setTrustedProxies configures the CIDR ranges whose members are trusted to set
// forwarding headers. Passing nil or an empty slice disables header trust.
func (i *ipRateLimiter) setTrustedProxies(cidrs []*net.IPNet) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.trustedProxies = cidrs
}

// getLimiter returns the rate limiter for the given IP address,
// creating one if it does not already exist. Updates the last-seen
// timestamp on each access. When the entry cap is reached, the
// least-recently-seen entry is evicted to bound memory usage.
func (i *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	if entry, exists := i.limiters[ip]; exists {
		entry.lastSeen = time.Now()
		return entry.limiter
	}

	if i.maxEntries > 0 && len(i.limiters) >= i.maxEntries {
		i.evictOldestLocked()
	}

	entry := &rateLimiterEntry{
		limiter:  rate.NewLimiter(i.rate, i.burst),
		lastSeen: time.Now(),
	}
	i.limiters[ip] = entry
	return entry.limiter
}

// evictOldestLocked removes the entry with the oldest lastSeen timestamp.
// The caller must hold i.mu.
func (i *ipRateLimiter) evictOldestLocked() {
	var oldestKey string
	var oldestSeen time.Time
	first := true
	for k, e := range i.limiters {
		if first || e.lastSeen.Before(oldestSeen) {
			oldestKey, oldestSeen, first = k, e.lastSeen, false
		}
	}
	if !first {
		delete(i.limiters, oldestKey)
	}
}

// cleanup removes entries that have not been seen for longer than maxAge.
// This prevents unbounded memory growth from unique IPs.
func (i *ipRateLimiter) cleanup(maxAge time.Duration) {
	i.mu.Lock()
	defer i.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for ip, entry := range i.limiters {
		if entry.lastSeen.Before(cutoff) {
			delete(i.limiters, ip)
		}
	}
}

// startCleanup starts a background goroutine that periodically removes stale
// rate limiter entries. The goroutine stops when the provided context is
// cancelled, keeping cancellation consistent with Go conventions and easy to
// compose with the rest of the server lifecycle.
//
// Parameters:
//   - ctx: cancelled to signal the cleanup goroutine to exit
//   - interval: how often to run cleanup (e.g., 10 minutes)
//   - maxAge: entries not seen within this duration are removed (e.g., 10 minutes)
func (i *ipRateLimiter) startCleanup(ctx context.Context, interval, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		logger := log.Log.WithName("rate-limiter")
		for {
			select {
			case <-ticker.C:
				i.cleanup(maxAge)
				i.mu.Lock()
				count := len(i.limiters)
				i.mu.Unlock()
				logger.V(1).Info("Rate limiter cleanup completed", "activeEntries", count)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// parseTrustedProxyCIDRs parses a comma-separated list of CIDR ranges (e.g.
// "10.0.0.0/8,192.168.0.0/16") into IPNets. Individual bare IPs are also
// accepted and treated as /32 (IPv4) or /128 (IPv6). Invalid entries are
// skipped and reported to the caller so they can be logged.
func parseTrustedProxyCIDRs(raw string) ([]*net.IPNet, []string) {
	var nets []*net.IPNet
	var invalid []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ipNet, err := net.ParseCIDR(part); err == nil {
			nets = append(nets, ipNet)
			continue
		}
		// Accept a bare IP by promoting it to a host CIDR.
		if ip := net.ParseIP(part); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			nets = append(nets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
			continue
		}
		invalid = append(invalid, part)
	}
	return nets, invalid
}

// hostFromRemoteAddr strips the port from an HTTP RemoteAddr, returning the
// host portion (which may be an IP or, rarely, an unresolved value).
func hostFromRemoteAddr(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}

// isTrustedProxy reports whether ip falls within any of the trusted CIDR ranges.
func isTrustedProxy(ip net.IP, trusted []*net.IPNet) bool {
	for _, n := range trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// clientIPFromHeaders extracts and validates a client IP from forwarding
// headers, returning nil if neither header carries a parseable IP. The
// leftmost X-Forwarded-For entry (the original client) is preferred.
func clientIPFromHeaders(r *http.Request) net.IP {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first := strings.TrimSpace(strings.SplitN(xff, ",", 2)[0])
		if ip := net.ParseIP(first); ip != nil {
			return ip
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		if ip := net.ParseIP(xri); ip != nil {
			return ip
		}
	}
	return nil
}

// extractClientIP derives the client IP used as the rate-limiter key.
//
// Security: forwarding headers (X-Forwarded-For / X-Real-IP) are attacker
// controllable, so they are only honored when the direct peer (RemoteAddr) is a
// configured trusted proxy. This prevents clients from spoofing arbitrary
// limiter keys to bypass throttling or from flooding the limiter map with
// unique, non-IP keys. The returned value is always a validated, canonicalized
// IP string, falling back to the raw RemoteAddr only when it cannot be parsed.
func extractClientIP(r *http.Request, trustedProxies []*net.IPNet) string {
	remoteIP := net.ParseIP(hostFromRemoteAddr(r.RemoteAddr))

	if remoteIP != nil && isTrustedProxy(remoteIP, trustedProxies) {
		if forwarded := clientIPFromHeaders(r); forwarded != nil {
			return forwarded.String()
		}
	}

	if remoteIP != nil {
		return remoteIP.String()
	}
	// RemoteAddr was not a valid IP; key on the raw value so all such traffic
	// does not collapse into a single shared bucket.
	return r.RemoteAddr
}

// rateLimitMiddleware creates an HTTP middleware that enforces per-IP rate limiting.
// When a client exceeds the rate limit, it receives a 429 Too Many Requests response.
//
// Parameters:
//   - limiter: the per-IP rate limiter to use
//
// Returns a middleware function that wraps an http.Handler.
func rateLimitMiddleware(limiter *ipRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limiter.mu.Lock()
			trusted := limiter.trustedProxies
			limiter.mu.Unlock()

			ip := extractClientIP(r, trusted)
			if !limiter.getLimiter(ip).Allow() {
				logger := log.Log.WithName("rate-limiter")
				logger.Info("Rate limit exceeded",
					"client_ip", ip,
					"path", r.URL.Path,
					"method", r.Method,
				)
				writeJSONError(w, http.StatusTooManyRequests, ErrorResponse{
					Error:   "rate_limited",
					Message: "Too many requests, please try again later",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
