// Package signatureverification stores the operator-wide image signature
// verification setting shared by the REST API and reconcilers.
package signatureverification

import (
	"context"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConfigMapName = "krkn-operator-signature-verification"
	EnabledKey    = "enabled"
)

// GetEnabled returns the configured verification state. Verification is
// enabled by default so a missing ConfigMap cannot silently disable the check.
func GetEnabled(ctx context.Context, k8sClient client.Client, namespace string) (bool, error) {
	var settings corev1.ConfigMap
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: ConfigMapName, Namespace: namespace}, &settings); err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, fmt.Errorf("failed to read signature verification setting: %w", err)
	}

	raw, ok := settings.Data[EnabledKey]
	if !ok || raw == "" {
		return true, nil
	}
	enabled, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("invalid %s value %q in ConfigMap %s: %w", EnabledKey, raw, ConfigMapName, err)
	}
	return enabled, nil
}

// SetEnabled persists the verification state in the operator namespace.
func SetEnabled(ctx context.Context, k8sClient client.Client, namespace string, enabled bool) error {
	var settings corev1.ConfigMap
	key := client.ObjectKey{Name: ConfigMapName, Namespace: namespace}
	if err := k8sClient.Get(ctx, key, &settings); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to read signature verification setting: %w", err)
		}
		settings = corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: ConfigMapName, Namespace: namespace},
			Data:       map[string]string{EnabledKey: strconv.FormatBool(enabled)},
		}
		if err := k8sClient.Create(ctx, &settings); err != nil {
			return fmt.Errorf("failed to create signature verification setting: %w", err)
		}
		return nil
	}

	if settings.Data == nil {
		settings.Data = make(map[string]string)
	}
	settings.Data[EnabledKey] = strconv.FormatBool(enabled)
	if err := k8sClient.Update(ctx, &settings); err != nil {
		return fmt.Errorf("failed to update signature verification setting: %w", err)
	}
	return nil
}
