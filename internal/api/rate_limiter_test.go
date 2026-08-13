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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestIPRateLimiter_AllowsUnderLimit(t *testing.T) {
	// Create a limiter with rate of 10/s and burst of 5
	limiter := newIPRateLimiter(rate.Limit(10), 5)

	ip := "192.168.1.1"
	rl := limiter.getLimiter(ip)

	// Should allow up to burst size immediately
	for i := 0; i < 5; i++ {
		if !rl.Allow() {
			t.Errorf("Request %d should have been allowed", i+1)
		}
	}
}

func TestIPRateLimiter_BlocksOverLimit(t *testing.T) {
	// Create a limiter with very low rate and burst of 2
	limiter := newIPRateLimiter(rate.Limit(0.01), 2) // 0.01/s = very slow refill

	ip := "10.0.0.1"
	rl := limiter.getLimiter(ip)

	// Exhaust the burst
	for i := 0; i < 2; i++ {
		if !rl.Allow() {
			t.Fatalf("Initial burst request %d should have been allowed", i+1)
		}
	}

	// The next request should be blocked (burst exhausted, rate too slow)
	if rl.Allow() {
		t.Error("Request after burst exhaustion should have been blocked")
	}
}

func TestIPRateLimiter_SeparateLimitersPerIP(t *testing.T) {
	limiter := newIPRateLimiter(rate.Limit(0.01), 1) // burst of 1

	// First IP uses its burst
	rl1 := limiter.getLimiter("10.0.0.1")
	if !rl1.Allow() {
		t.Error("First request from IP1 should be allowed")
	}
	if rl1.Allow() {
		t.Error("Second request from IP1 should be blocked")
	}

	// Second IP should have its own independent limiter
	rl2 := limiter.getLimiter("10.0.0.2")
	if !rl2.Allow() {
		t.Error("First request from IP2 should be allowed (separate limiter)")
	}
}

func TestIPRateLimiter_Cleanup(t *testing.T) {
	limiter := newIPRateLimiter(rate.Limit(1), 1)

	// Access an IP to create an entry
	limiter.getLimiter("10.0.0.1")

	// Verify entry exists
	limiter.mu.Lock()
	if len(limiter.limiters) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(limiter.limiters))
	}

	// Backdating the last-seen time to simulate staleness
	limiter.limiters["10.0.0.1"].lastSeen = time.Now().Add(-20 * time.Minute)
	limiter.mu.Unlock()

	// Run cleanup with 10 minute max age
	limiter.cleanup(10 * time.Minute)

	limiter.mu.Lock()
	remaining := len(limiter.limiters)
	limiter.mu.Unlock()

	if remaining != 0 {
		t.Errorf("Expected 0 entries after cleanup, got %d", remaining)
	}
}

func TestIPRateLimiter_CleanupPreservesFresh(t *testing.T) {
	limiter := newIPRateLimiter(rate.Limit(1), 1)

	// Create two entries
	limiter.getLimiter("10.0.0.1") // fresh
	limiter.getLimiter("10.0.0.2") // will be stale

	// Make one stale
	limiter.mu.Lock()
	limiter.limiters["10.0.0.2"].lastSeen = time.Now().Add(-20 * time.Minute)
	limiter.mu.Unlock()

	limiter.cleanup(10 * time.Minute)

	limiter.mu.Lock()
	remaining := len(limiter.limiters)
	_, freshExists := limiter.limiters["10.0.0.1"]
	_, staleExists := limiter.limiters["10.0.0.2"]
	limiter.mu.Unlock()

	if remaining != 1 {
		t.Errorf("Expected 1 entry after cleanup, got %d", remaining)
	}
	if !freshExists {
		t.Error("Fresh entry should have been preserved")
	}
	if staleExists {
		t.Error("Stale entry should have been removed")
	}
}

func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name:       "from RemoteAddr with port",
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "from RemoteAddr without port",
			remoteAddr: "192.168.1.1",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "from X-Forwarded-For single IP",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1"},
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "10.0.0.1",
		},
		{
			name:       "from X-Forwarded-For multiple IPs",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1, 10.0.0.2, 10.0.0.3"},
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "10.0.0.1",
		},
		{
			name:       "from X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "172.16.0.1"},
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "172.16.0.1",
		},
		{
			name: "X-Forwarded-For takes priority over X-Real-IP",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1",
				"X-Real-IP":       "172.16.0.1",
			},
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := extractClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %q, got %q", tt.expectedIP, ip)
			}
		})
	}
}

func TestRateLimitMiddleware_AllowsNormalTraffic(t *testing.T) {
	limiter := newIPRateLimiter(rate.Limit(10), 5) // generous limit
	middleware := rateLimitMiddleware(limiter)

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(testHandler)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	if !handlerCalled {
		t.Error("Expected handler to be called")
	}
}

func TestRateLimitMiddleware_Blocks429(t *testing.T) {
	// Very restrictive: burst of 1, essentially no refill
	limiter := newIPRateLimiter(rate.Limit(0.001), 1)
	middleware := rateLimitMiddleware(limiter)

	handlerCallCount := 0
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCallCount++
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(testHandler)

	// First request should succeed (uses burst)
	req1 := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First request: expected status %d, got %d", http.StatusOK, w1.Code)
	}

	// Second request should be rate limited
	req2 := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req2.RemoteAddr = "10.0.0.1:12345"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("Second request: expected status %d, got %d", http.StatusTooManyRequests, w2.Code)
	}

	// Verify the response body contains the expected error
	var errResp ErrorResponse
	if err := json.NewDecoder(w2.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error != "rate_limited" {
		t.Errorf("Expected error code 'rate_limited', got %q", errResp.Error)
	}

	// Only the first request should have reached the handler
	if handlerCallCount != 1 {
		t.Errorf("Expected handler to be called 1 time, got %d", handlerCallCount)
	}
}

func TestRateLimitMiddleware_DifferentIPsIndependent(t *testing.T) {
	limiter := newIPRateLimiter(rate.Limit(0.001), 1) // burst of 1
	middleware := rateLimitMiddleware(limiter)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(testHandler)

	// First IP uses its burst
	req1 := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("IP1 first request: expected %d, got %d", http.StatusOK, w1.Code)
	}

	// First IP is now limited
	req2 := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req2.RemoteAddr = "10.0.0.1:12345"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("IP1 second request: expected %d, got %d", http.StatusTooManyRequests, w2.Code)
	}

	// Second IP should still work
	req3 := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req3.RemoteAddr = "10.0.0.2:12345"
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Errorf("IP2 first request: expected %d, got %d", http.StatusOK, w3.Code)
	}
}
