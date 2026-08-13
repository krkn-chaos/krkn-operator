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
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ipRateLimiter provides per-IP rate limiting for HTTP endpoints.
// It maintains a map of rate limiters keyed by client IP address,
// with automatic cleanup of stale entries to prevent memory leaks.
type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rateLimiterEntry
	rate     rate.Limit
	burst    int
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
// Returns a new ipRateLimiter instance.
func newIPRateLimiter(r rate.Limit, b int) *ipRateLimiter {
	rl := &ipRateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		rate:     r,
		burst:    b,
	}
	return rl
}

// getLimiter returns the rate limiter for the given IP address,
// creating one if it does not already exist. Updates the last-seen
// timestamp on each access.
func (i *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	entry, exists := i.limiters[ip]
	if !exists {
		entry = &rateLimiterEntry{
			limiter:  rate.NewLimiter(i.rate, i.burst),
			lastSeen: time.Now(),
		}
		i.limiters[ip] = entry
		return entry.limiter
	}

	entry.lastSeen = time.Now()
	return entry.limiter
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
// rate limiter entries. The goroutine stops when the provided stop channel is closed.
//
// Parameters:
//   - interval: how often to run cleanup (e.g., 10 minutes)
//   - maxAge: entries not seen within this duration are removed (e.g., 10 minutes)
//   - stop: channel that signals the goroutine to exit
func (i *ipRateLimiter) startCleanup(interval, maxAge time.Duration, stop <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				i.cleanup(maxAge)
				logger := log.Log.WithName("rate-limiter")
				i.mu.Lock()
				count := len(i.limiters)
				i.mu.Unlock()
				logger.V(1).Info("Rate limiter cleanup completed", "activeEntries", count)
			case <-stop:
				return
			}
		}
	}()
}

// extractClientIP extracts the client IP address from an HTTP request.
// It checks X-Forwarded-For and X-Real-IP headers first (for reverse proxy setups),
// then falls back to RemoteAddr.
func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (may contain multiple IPs: client, proxy1, proxy2)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (the original client)
		parts := strings.SplitN(xff, ",", 2)
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr (includes port)
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr might not have a port
		return r.RemoteAddr
	}
	return host
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
			ip := extractClientIP(r)
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
