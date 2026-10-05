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
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/krkn-chaos/krkn-operator/internal/kubeconfig"
	"github.com/krkn-chaos/krkn-operator/pkg/provider"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	krknOperatorTargetResource = schema.GroupVersionResource{
		Group: "krkn.krkn-chaos.dev", Version: "v1alpha1", Resource: "krknoperatortargets",
	}
	secretResource = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
)

// refreshRestoredTargetStatuses is intentionally kept outside the shared
// archive package. Target readiness is operator-specific and depends on the
// provider liveness implementation; the shared package only restores admin
// configuration through the Kubernetes API.
//
// The direct dynamic client is intentional here. Restore writes resources
// through the direct Kubernetes API, while a controller-runtime client may
// still have a stale cache immediately afterwards.
func refreshRestoredTargetStatuses(ctx context.Context, dynamicClient dynamic.Interface, namespace string) error {
	if dynamicClient == nil {
		return fmt.Errorf("dynamic Kubernetes client is not configured")
	}

	logger := log.FromContext(ctx)
	targets := dynamicClient.Resource(krknOperatorTargetResource).Namespace(namespace)
	secrets := dynamicClient.Resource(secretResource).Namespace(namespace)
	targetList, err := targets.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list restored targets: %w", err)
	}

	for i := range targetList.Items {
		target := &targetList.Items[i]
		clusterName, _, err := unstructured.NestedString(target.Object, "spec", "clusterName")
		if err != nil {
			return fmt.Errorf("failed to read cluster name for restored target %q: %w", target.GetName(), err)
		}
		secretName, _, err := unstructured.NestedString(target.Object, "spec", "secretUUID")
		if err != nil {
			return fmt.Errorf("failed to read Secret name for restored target %q: %w", target.GetName(), err)
		}

		ready, err := restoredTargetIsReady(ctx, secrets, secretName, clusterName, logger)
		if err != nil {
			return fmt.Errorf("failed to check restored target %q: %w", clusterName, err)
		}
		if err := updateRestoredTargetStatus(ctx, targets, target.GetName(), ready); err != nil {
			return fmt.Errorf("failed to update restored target %q status: %w", clusterName, err)
		}
	}

	return nil
}

func restoredTargetIsReady(ctx context.Context, secrets dynamic.ResourceInterface, secretName, clusterName string, logger logr.Logger) (bool, error) {
	if secretName == "" {
		logger.Info("Restored target Secret name is empty", "clusterName", clusterName)
		return false, nil
	}

	secret, err := secrets.Get(ctx, secretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		logger.Info("Restored target Secret was not found", "clusterName", clusterName)
		return false, nil
	}
	if err != nil {
		return false, err
	}

	encodedSecretData, found, err := unstructured.NestedString(secret.Object, "data", "kubeconfig")
	if err != nil {
		return false, fmt.Errorf("failed to read kubeconfig from Secret %q: %w", secretName, err)
	}
	if !found {
		logger.Info("Restored target Secret has no kubeconfig", "clusterName", clusterName)
		return false, nil
	}

	secretData, err := base64.StdEncoding.DecodeString(encodedSecretData)
	if err != nil {
		logger.Info("Restored target kubeconfig Secret data could not be decoded", "clusterName", clusterName, "error", err)
		return false, nil
	}
	kubeconfigBase64, err := kubeconfig.UnmarshalSecretData(secretData)
	if err != nil {
		logger.Info("Restored target kubeconfig could not be decoded", "clusterName", clusterName, "error", err)
		return false, nil
	}

	if err := provider.CheckClusterLiveness(ctx, kubeconfigBase64, 0); err != nil {
		logger.Info("Restored target liveness check failed", "clusterName", clusterName, "error", err)
		return false, nil
	}
	return true, nil
}

func updateRestoredTargetStatus(ctx context.Context, targets dynamic.ResourceInterface, name string, ready bool) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		target, err := targets.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		status, found, err := unstructured.NestedMap(target.Object, "status")
		if err != nil {
			return err
		}
		if !found {
			status = make(map[string]interface{})
		}
		status["ready"] = ready
		status["lastUpdated"] = time.Now().UTC().Format(time.RFC3339Nano)
		if err := unstructured.SetNestedMap(target.Object, status, "status"); err != nil {
			return err
		}

		_, err = targets.UpdateStatus(ctx, target, metav1.UpdateOptions{})
		return err
	})
}
