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

package provider

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestCheckClusterLiveness(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		handler    http.Handler
		kubeconfig string
		timeout    time.Duration
		cancel     bool
		wantErr    bool
	}{
		{
			name:       "reachable API server",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unhealthy API server",
			statusCode: http.StatusServiceUnavailable,
			wantErr:    true,
		},
		{
			name: "request timeout",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				time.Sleep(100 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			}),
			timeout: 10 * time.Millisecond,
			wantErr: true,
		},
		{
			name: "canceled context",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				time.Sleep(100 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			}),
			cancel:  true,
			wantErr: true,
		},
		{
			name:       "invalid kubeconfig",
			kubeconfig: "not-base64",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.kubeconfig != "" {
				if err := CheckClusterLiveness(context.Background(), tt.kubeconfig, tt.timeout); err == nil {
					t.Fatal("CheckClusterLiveness() error = nil, want error")
				}
				return
			}

			server := httptest.NewServer(handlerOrStatus(tt.statusCode, tt.handler))
			defer server.Close()

			kubeconfig := testKubeconfig(t, server.URL)
			ctx := context.Background()
			if tt.cancel {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err := CheckClusterLiveness(ctx, kubeconfig, tt.timeout)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckClusterLiveness() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func handlerOrStatus(statusCode int, handler http.Handler) http.Handler {
	if handler != nil {
		return handler
	}
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusCode)
	})
}

func testKubeconfig(t *testing.T, serverURL string) string {
	t.Helper()
	config := clientcmdapi.NewConfig()
	config.Clusters["test"] = &clientcmdapi.Cluster{Server: serverURL}
	config.AuthInfos["test-user"] = &clientcmdapi.AuthInfo{}
	config.Contexts["test-context"] = &clientcmdapi.Context{Cluster: "test", AuthInfo: "test-user"}
	config.CurrentContext = "test-context"

	data, err := clientcmd.Write(*config)
	if err != nil {
		t.Fatalf("failed to write test kubeconfig: %v", err)
	}
	return base64.StdEncoding.EncodeToString(data)
}
