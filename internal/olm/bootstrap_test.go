package olm

import (
	"context"
	"strings"
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
	configMap, err := clientset.CoreV1().ConfigMaps("krkn-operator-system").Get(context.Background(), consoleConfigMap, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("console ConfigMap was not created: %v", err)
	}
	if !strings.Contains(configMap.Data["nginx.conf"], "krkn-operator-system.svc.cluster.local") {
		t.Fatalf("console ConfigMap proxy does not target the operator namespace: %s", configMap.Data["nginx.conf"])
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

	configMap.Data["nginx.conf"] = nginxConfig("krkn-operator-system-old")
	if _, err := clientset.CoreV1().ConfigMaps("krkn-operator-system").Update(context.Background(), configMap, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("seed stale console ConfigMap: %v", err)
	}
	if err := EnsureResources(context.Background(), clientset, "krkn-operator-system"); err != nil {
		t.Fatalf("EnsureResources() stale ConfigMap reconciliation error = %v", err)
	}
	configMap, err = clientset.CoreV1().ConfigMaps("krkn-operator-system").Get(context.Background(), consoleConfigMap, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get reconciled console ConfigMap: %v", err)
	}
	if strings.Contains(configMap.Data["nginx.conf"], "krkn-operator-system-old.svc.cluster.local") {
		t.Fatalf("stale console ConfigMap proxy was not reconciled: %s", configMap.Data["nginx.conf"])
	}

	binding, err := clientset.RbacV1().ClusterRoleBindings().Get(context.Background(), scenarioRunnerRole, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get scenario runner binding: %v", err)
	}
	binding.Subjects[0].Namespace = "krkn-operator-system-old"
	if _, err := clientset.RbacV1().ClusterRoleBindings().Update(context.Background(), binding, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("seed stale scenario runner binding: %v", err)
	}
	if err := EnsureResources(context.Background(), clientset, "krkn-operator-system"); err != nil {
		t.Fatalf("EnsureResources() stale RBAC reconciliation error = %v", err)
	}
	binding, err = clientset.RbacV1().ClusterRoleBindings().Get(context.Background(), scenarioRunnerRole, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get reconciled scenario runner binding: %v", err)
	}
	if binding.Subjects[0].Namespace != "krkn-operator-system" {
		t.Fatalf("scenario runner binding namespace = %q, want krkn-operator-system", binding.Subjects[0].Namespace)
	}
}
