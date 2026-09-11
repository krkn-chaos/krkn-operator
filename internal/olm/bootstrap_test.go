package olm

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEnsureResources(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	if err := EnsureResources(context.Background(), clientset, "krkn-operator-system"); err != nil {
		t.Fatalf("EnsureResources() error = %v", err)
	}

	if _, err := clientset.CoreV1().Secrets("krkn-operator-system").Get(context.Background(), jwtSecretName, metav1.GetOptions{}); err != nil {
		t.Fatalf("JWT secret was not created: %v", err)
	}
	if _, err := clientset.CoreV1().ConfigMaps("krkn-operator-system").Get(context.Background(), consoleConfigMap, metav1.GetOptions{}); err != nil {
		t.Fatalf("console ConfigMap was not created: %v", err)
	}
	for _, name := range []string{operatorName, metricsServiceName, consoleName} {
		if _, err := clientset.CoreV1().Services("krkn-operator-system").Get(context.Background(), name, metav1.GetOptions{}); err != nil {
			t.Fatalf("service %q was not created: %v", name, err)
		}
	}
	if _, err := clientset.RbacV1().ClusterRoleBindings().Get(context.Background(), scenarioRunnerRole, metav1.GetOptions{}); err != nil {
		t.Fatalf("scenario runner binding was not created: %v", err)
	}

	if err := EnsureResources(context.Background(), clientset, "krkn-operator-system"); err != nil {
		t.Fatalf("EnsureResources() second call error = %v", err)
	}
}
