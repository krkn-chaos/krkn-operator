package kubeconfig

import (
	"context"
	"errors"
	"fmt"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AttemptConfigMapName returns the name used to retain an attempt's kubeconfig.
func AttemptConfigMapName(jobID string) string {
	return "krkn-job-" + jobID + "-kubeconfig"
}

// LoadOriginalAttempt loads a retained kubeconfig only when its owner and
// attempt identity still match the original scenario run and cluster.
func LoadOriginalAttempt(ctx context.Context, k8sClient client.Client, namespace string, source *krknv1alpha1.KrknScenarioRun, jobID, clusterName string) (string, error) {
	configMap := &corev1.ConfigMap{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: AttemptConfigMapName(jobID)}, configMap); err != nil {
		return "", fmt.Errorf("load original attempt kubeconfig: %w", err)
	}
	owned := false
	for _, owner := range configMap.OwnerReferences {
		if owner.UID == source.UID && owner.Name == source.Name && owner.Kind == "KrknScenarioRun" {
			owned = true
			break
		}
	}
	if !owned || configMap.Labels["krkn-job-id"] != jobID || configMap.Labels["krkn-cluster-name"] != clusterName || configMap.Data["config"] == "" {
		return "", errors.New("original attempt kubeconfig identity is invalid")
	}
	return configMap.Data["config"], nil
}
