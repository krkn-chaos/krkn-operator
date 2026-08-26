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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// UpstreamRouteName is the fixed Route name created by the visualize deploy script.
const UpstreamRouteName = "krkn-visualize"

// GetRouteURL fetches the external URL for a Grafana Route. The caller should
// pass a direct client for the cluster that owns the Route.
func GetRouteURL(ctx context.Context, c client.Client, namespace, routeName string) (string, error) {
	logger := log.FromContext(ctx)

	var routeURL string
	var lastError error

	// Poll for up to 30 seconds - Routes usually get hosts assigned immediately
	err := wait.PollImmediate(2*time.Second, 30*time.Second, func() (bool, error) {
		// The target-cluster client is uncached, so this queries the selected
		// cluster directly rather than the operator's host-cluster cache.
		route := &routev1.Route{}
		err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: routeName}, route)
		if err != nil {
			if apierrors.IsNotFound(err) {
				lastError = err
				logger.V(1).Info("Route not found yet, will retry", "namespace", namespace, "route", routeName)
				return false, nil // Keep polling
			}
			// Other errors
			lastError = err
			logger.V(1).Info("Error getting Route, will retry", "error", err.Error())
			return false, nil // Keep polling
		}

		url, err := RouteURL(route)
		if err != nil {
			logger.V(1).Info("Route exists but host not assigned yet, waiting...")
			return false, nil
		}

		routeURL = url
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

// RouteURL returns the external URL for a Grafana Route. The scheme follows
// the Route's TLS configuration: routes without TLS termination (like the one
// created by the upstream krkn-visualize deploy script) are only served over
// plain HTTP, while TLS-terminated routes use HTTPS.
func RouteURL(route *routev1.Route) (string, error) {
	if route == nil || route.Spec.Host == "" {
		return "", fmt.Errorf("route host is not assigned")
	}
	scheme := "http"
	if route.Spec.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, route.Spec.Host), nil
}

// CreateGrafanaRoute creates an OpenShift Route for Grafana
func CreateGrafanaRoute(ctx context.Context, c client.Client, namespace, instanceName string) error {
	weight := int32(100)

	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ResourceName(instanceName, "grafana"),
			Namespace: namespace,
			Labels: map[string]string{
				LabelManagedBy: ManagedByValue,
				LabelComponent: ComponentValue,
				LabelName:      instanceName,
				"app":          ResourceName(instanceName, "grafana"),
			},
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   ResourceName(instanceName, "grafana"),
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
