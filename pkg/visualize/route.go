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

package visualize

import (
	"context"
	"fmt"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	routeclient "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// GetRouteURL fetches the external URL for a Grafana Route
// Uses the OpenShift route client to query the API server directly (no cache)
func GetRouteURL(ctx context.Context, c client.Client, namespace string) (string, error) {
	logger := log.FromContext(ctx)

	// IMPORTANT: We use the OpenShift route client here instead of the cached controller-runtime
	// client because the operator's cache only watches specific namespaces (e.g., krkn-operator/default),
	// not the target namespace where Grafana is deployed. Using c.Get() causes cache errors:
	// "unknown namespace for the cache"
	//
	// The route client queries the API server directly, bypassing the cache entirely.

	// Get REST config to create route client
	cfg, err := config.GetConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get rest config: %w", err)
	}

	// Create OpenShift route client for direct API access
	routeClient, err := routeclient.NewForConfig(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create route client: %w", err)
	}

	var routeURL string
	var lastError error

	// Poll for up to 30 seconds - Routes usually get hosts assigned immediately
	err = wait.PollImmediate(2*time.Second, 30*time.Second, func() (bool, error) {
		// Direct API call to get Route (bypasses controller-runtime cache)
		route, err := routeClient.Routes(namespace).Get(ctx, "grafana", metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				lastError = err
				logger.V(1).Info("Route not found yet, will retry", "namespace", namespace)
				return false, nil // Keep polling
			}
			// Other errors
			lastError = err
			logger.V(1).Info("Error getting Route, will retry", "error", err.Error())
			return false, nil // Keep polling
		}

		// Check if host is assigned
		if route.Spec.Host == "" {
			logger.V(1).Info("Route exists but host not assigned yet, waiting...")
			return false, nil
		}

		routeURL = fmt.Sprintf("https://%s", route.Spec.Host)
		logger.Info("Route host assigned successfully", "host", route.Spec.Host, "url", routeURL)
		return true, nil
	})

	if err != nil {
		if lastError != nil {
			return "", fmt.Errorf("failed to get Route URL after 30s: %w", lastError)
		}
		return "", fmt.Errorf("failed to get Route URL after 30s: %w", err)
	}

	if routeURL == "" {
		return "", fmt.Errorf("route host not assigned within 30 second timeout")
	}

	return routeURL, nil
}

// CreateGrafanaRoute creates an OpenShift Route for Grafana
func CreateGrafanaRoute(ctx context.Context, c client.Client, namespace, instanceName string) error {
	weight := int32(100)

	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana",
			Namespace: namespace,
			Labels: map[string]string{
				LabelManagedBy: ManagedByValue,
				LabelComponent: ComponentValue,
				LabelName:      instanceName,
				"app":          "grafana",
			},
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   "grafana",
				Weight: &weight,
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("http"),
			},
			TLS: &routev1.TLSConfig{
				Termination:                   routev1.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
			},
		},
	}

	err := c.Create(ctx, route)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}
