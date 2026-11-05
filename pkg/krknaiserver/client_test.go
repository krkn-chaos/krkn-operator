package krknaiserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDiscoveryDeadlineDoesNotShortenOrdinaryRequests(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(75 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
		}
	}))
	defer service.Close()
	client := New(service.URL, "token")
	client.discoveryClient.Timeout = 25 * time.Millisecond

	_, err := client.DoDiscovery(context.Background(), http.MethodPost, "/v1/discoveries", nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("discovery error = %v, want deadline exceeded", err)
	}

	response, err := client.Do(context.Background(), http.MethodPost, "/v1/configs/validate", nil)
	if err != nil {
		t.Fatalf("ordinary request inherited the short discovery deadline: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("ordinary request status = %d, want 200", response.StatusCode)
	}
}
