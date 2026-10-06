package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/krkn-chaos/krkn-operator/internal/crdmigration"
)

const migrationGuardRequeue = 5 * time.Second

func migrationPending(ctx context.Context, reader client.Reader, namespace string, annotations map[string]string) (bool, error) {
	if annotations[crdmigration.AnnotationKey] != "" {
		return true, nil
	}
	guard := &corev1.ConfigMap{}
	key := types.NamespacedName{Name: crdmigration.GuardConfigMapName, Namespace: namespace}
	if err := reader.Get(ctx, key, guard); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("check custom resource migration guard: %w", err)
	}
	return true, nil
}
