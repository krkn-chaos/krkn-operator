package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/krkn-chaos/krkn-operator/pkg/elasticsearch"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const removedGrafanaURLAnnotation = "elasticsearch.krkn.krkn-chaos.dev/grafana-url"

// RemoveElasticsearchGrafanaURLAnnotations removes the retired Grafana URL
// annotation from existing Elasticsearch config Secrets. It is intended for a
// startup migration, not for request handlers, so list/read operations remain
// read-only.
func RemoveElasticsearchGrafanaURLAnnotations(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	secrets, err := clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=%s", elasticsearch.AppNameLabel, elasticsearch.AppName, elasticsearch.AppComponentLabel, elasticsearch.ComponentElasticsearchConfig),
	})
	if err != nil {
		return fmt.Errorf("list Elasticsearch config secrets: %w", err)
	}

	var cleanupErr error
	for i := range secrets.Items {
		secret := &secrets.Items[i]
		if secret.Annotations == nil {
			continue
		}
		if _, exists := secret.Annotations[removedGrafanaURLAnnotation]; !exists {
			continue
		}
		delete(secret.Annotations, removedGrafanaURLAnnotation)
		if _, err := clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove Grafana URL annotation from Secret %q: %w", secret.Name, err))
		}
	}
	return cleanupErr
}
