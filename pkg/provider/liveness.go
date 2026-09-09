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
	"fmt"
	"net/http"
	"strings"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const defaultLivenessTimeout = 5 * time.Second

// CheckClusterLiveness verifies that a Kubernetes API server can be reached
// with the supplied base64-encoded kubeconfig. The kubeconfig's TLS and
// authentication settings are preserved. A non-positive timeout uses the
// default timeout.
func CheckClusterLiveness(ctx context.Context, kubeconfigBase64 string, timeout time.Duration) error {
	if ctx == nil {
		return fmt.Errorf("context must not be nil")
	}

	kubeconfigBytes, err := base64.StdEncoding.DecodeString(kubeconfigBase64)
	if err != nil {
		return fmt.Errorf("failed to decode kubeconfig: %w", err)
	}

	config, err := clientcmd.Load(kubeconfigBytes)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	restConfig, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return fmt.Errorf("failed to build Kubernetes client config: %w", err)
	}

	transport, err := rest.TransportFor(restConfig)
	if err != nil {
		return fmt.Errorf("failed to build Kubernetes transport: %w", err)
	}

	if timeout <= 0 {
		timeout = defaultLivenessTimeout
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodGet,
		strings.TrimRight(restConfig.Host, "/")+"/version",
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes liveness request: %w", err)
	}

	response, err := (&http.Client{Transport: transport}).Do(request)
	if err != nil {
		return fmt.Errorf("api server liveness check failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("api server liveness check returned HTTP status %s", response.Status)
	}

	return nil
}
