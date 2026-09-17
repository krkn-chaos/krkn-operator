package api

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/krkn-chaos/krkn-operator/pkg/elasticsearch"
)

func TestRemoveElasticsearchGrafanaURLAnnotations(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "with-grafana",
				Namespace: "default",
				Labels: map[string]string{
					elasticsearch.AppNameLabel:      elasticsearch.AppName,
					elasticsearch.AppComponentLabel: elasticsearch.ComponentElasticsearchConfig,
				},
				Annotations: map[string]string{removedGrafanaURLAnnotation: "https://grafana.example.com"},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "unrelated",
				Namespace: "default",
				Annotations: map[string]string{
					removedGrafanaURLAnnotation: "https://grafana.example.com",
				},
			},
		},
	)

	if err := RemoveElasticsearchGrafanaURLAnnotations(context.Background(), clientset, "default"); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	secret, err := clientset.CoreV1().Secrets("default").Get(context.Background(), "with-grafana", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get cleaned Secret: %v", err)
	}
	if _, exists := secret.Annotations[removedGrafanaURLAnnotation]; exists {
		t.Error("expected Grafana URL annotation to be removed")
	}

	if _, err := clientset.CoreV1().Secrets("default").Get(context.Background(), "unrelated", metav1.GetOptions{}); err != nil {
		t.Fatalf("expected unrelated Secret to remain: %v", err)
	}
}
