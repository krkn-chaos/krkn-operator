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
	namespace := "krkn-operator-system"
	if err := EnsureResources(context.Background(), clientset, namespace, false); err != nil {
		t.Fatalf("EnsureResources() error = %v", err)
	}

	if _, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), jwtSecretName, metav1.GetOptions{}); err != nil {
		t.Fatalf("JWT secret was not created: %v", err)
	}
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.Background(), consoleConfigMap, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("console ConfigMap was not created: %v", err)
	}
	if !strings.Contains(configMap.Data["nginx.conf"], namespace+".svc.cluster.local") {
		t.Fatalf("console ConfigMap proxy does not target the operator namespace: %s", configMap.Data["nginx.conf"])
	}
	for _, name := range []string{operatorName, metricsServiceName, consoleName} {
		if _, err := clientset.CoreV1().Services(namespace).Get(context.Background(), name, metav1.GetOptions{}); err != nil {
			t.Fatalf("service %q was not created: %v", name, err)
		}
	}
	if _, err := clientset.RbacV1().ClusterRoleBindings().Get(context.Background(), scenarioRunnerBindingName(namespace), metav1.GetOptions{}); err != nil {
		t.Fatalf("scenario runner binding was not created: %v", err)
	}

	if err := EnsureResources(context.Background(), clientset, namespace, false); err != nil {
		t.Fatalf("EnsureResources() second call error = %v", err)
	}

	configMap.Data["nginx.conf"] = nginxConfig(namespace + "-old")
	if _, err := clientset.CoreV1().ConfigMaps(namespace).Update(context.Background(), configMap, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("seed stale console ConfigMap: %v", err)
	}
	if err := EnsureResources(context.Background(), clientset, namespace, false); err != nil {
		t.Fatalf("EnsureResources() stale ConfigMap reconciliation error = %v", err)
	}
	configMap, err = clientset.CoreV1().ConfigMaps(namespace).Get(context.Background(), consoleConfigMap, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get reconciled console ConfigMap: %v", err)
	}
	if strings.Contains(configMap.Data["nginx.conf"], namespace+"-old.svc.cluster.local") {
		t.Fatalf("stale console ConfigMap proxy was not reconciled: %s", configMap.Data["nginx.conf"])
	}

	secondNamespace := namespace + "-old"
	if err := EnsureResources(context.Background(), clientset, secondNamespace, false); err != nil {
		t.Fatalf("EnsureResources() second namespace error = %v", err)
	}
	for _, currentNamespace := range []string{namespace, secondNamespace} {
		binding, err := clientset.RbacV1().ClusterRoleBindings().Get(context.Background(), scenarioRunnerBindingName(currentNamespace), metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get scenario runner binding for %s: %v", currentNamespace, err)
		}
		if binding.Subjects[0].Namespace != currentNamespace {
			t.Fatalf("scenario runner binding namespace = %q, want %s", binding.Subjects[0].Namespace, currentNamespace)
		}
	}
	if scenarioRunnerBindingName(namespace) == scenarioRunnerBindingName(secondNamespace) {
		t.Fatal("scenario runner binding names must be unique per namespace")
	}

	openShiftNamespace := "krkn-operator-ocp"
	if err := EnsureResources(context.Background(), clientset, openShiftNamespace, true); err != nil {
		t.Fatalf("EnsureResources() OpenShift error = %v", err)
	}
	sccBinding, err := clientset.RbacV1().ClusterRoleBindings().Get(context.Background(), scenarioRunnerSCCBindingName(openShiftNamespace), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("OpenShift SCC binding was not created: %v", err)
	}
	if sccBinding.RoleRef.Name != scenarioRunnerSCC || sccBinding.Subjects[0].Namespace != openShiftNamespace {
		t.Fatalf("unexpected OpenShift SCC binding: %#v", sccBinding)
	}
}
