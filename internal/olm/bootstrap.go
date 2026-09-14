// Package olm bootstraps resources that are not installed by an OLM CSV.
package olm

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	rbacv1typed "k8s.io/client-go/kubernetes/typed/rbac/v1"
)

const (
	operatorName       = "krkn-operator-operator"
	consoleName        = "krkn-operator-console"
	metricsServiceName = "krkn-operator-operator-metrics"
	consoleConfigMap   = "krkn-operator-console-nginx"
	jwtSecretName      = "krkn-operator-jwt" // #nosec G101 -- This is a Secret name, not a credential; the value is generated at runtime.
	scenarioRunnerSA   = "krkn-operator-krkn-scenario-runner"
	scenarioRunnerRole = "krkn-operator-scenario-runner"
	scenarioRunnerSCC  = "system:openshift:scc:anyuid"
)

// EnsureResources creates the non-deployment resources that OLM cannot install
// from a CSV install strategy. It is called by the OLM bootstrap init container
// before the operator and console deployments are started. The namespace must
// be the namespace containing the operator deployment. When openshift is true,
// it also grants the scenario-runner service account the anyuid SCC required by
// the OpenShift chart profile. The operation is idempotent and safe to retry;
// resources created before a later error remain in the cluster.
func EnsureResources(ctx context.Context, clientset kubernetes.Interface, namespace string, openshift bool) error {
	labels := map[string]string{
		"app.kubernetes.io/name":       "krkn-operator",
		"app.kubernetes.io/instance":   "krkn-operator",
		"app.kubernetes.io/managed-by": "olm",
	}

	if err := ensureServiceAccount(ctx, clientset, namespace, scenarioRunnerSA, labels); err != nil {
		return err
	}
	if err := ensureJWTSecret(ctx, clientset, namespace, labels); err != nil {
		return err
	}
	if err := ensureConfigMap(ctx, clientset, namespace, labels); err != nil {
		return err
	}
	if err := ensureServices(ctx, clientset, namespace, labels); err != nil {
		return err
	}
	if err := ensureScenarioRunnerRBAC(ctx, clientset, namespace, labels, openshift); err != nil {
		return err
	}
	return nil
}

func ensureServiceAccount(ctx context.Context, clientset kubernetes.Interface, namespace, name string, labels map[string]string) error {
	_, err := clientset.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil || apierrors.IsAlreadyExists(err) {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get service account %s: %w", name, err)
	}
	_, err = clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
	}, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create service account %s: %w", name, err)
	}
	return nil
}

func ensureJWTSecret(ctx context.Context, clientset kubernetes.Interface, namespace string, labels map[string]string) error {
	_, err := clientset.CoreV1().Secrets(namespace).Get(ctx, jwtSecretName, metav1.GetOptions{})
	if err == nil || apierrors.IsAlreadyExists(err) {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get JWT secret: %w", err)
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return fmt.Errorf("generate JWT secret: %w", err)
	}
	_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: jwtSecretName, Namespace: namespace, Labels: labels},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"jwt-secret": secret},
	}, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create JWT secret: %w", err)
	}
	return nil
}

func ensureConfigMap(ctx context.Context, clientset kubernetes.Interface, namespace string, labels map[string]string) error {
	configMaps := clientset.CoreV1().ConfigMaps(namespace)
	desiredData := map[string]string{"nginx.conf": nginxConfig(namespace)}
	current, err := configMaps.Get(ctx, consoleConfigMap, metav1.GetOptions{})
	if err == nil {
		if current.Data["nginx.conf"] == desiredData["nginx.conf"] {
			return nil
		}
		current.Data = desiredData
		current.Labels = labels
		if _, err := configMaps.Update(ctx, current, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update console configmap: %w", err)
		}
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get console configmap: %w", err)
	}
	_, err = configMaps.Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: consoleConfigMap, Namespace: namespace, Labels: labels},
		Data:       desiredData,
	}, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create console configmap: %w", err)
	}
	return nil
}

func ensureServices(ctx context.Context, clientset kubernetes.Interface, namespace string, labels map[string]string) error {
	resources := []*corev1.Service{
		{
			ObjectMeta: metav1.ObjectMeta{Name: operatorName, Namespace: namespace, Labels: labels},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{"app.kubernetes.io/component": "operator", "app.kubernetes.io/instance": "krkn-operator"},
				Ports:    []corev1.ServicePort{{Name: "http", Port: 8080, TargetPort: intstrFromInt(8080)}, {Name: "grpc", Port: 50051, TargetPort: intstrFromInt(50051)}},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: metricsServiceName, Namespace: namespace, Labels: labels},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{"app.kubernetes.io/component": "operator", "app.kubernetes.io/instance": "krkn-operator"},
				Ports:    []corev1.ServicePort{{Name: "https", Port: 8443, TargetPort: intstrFromInt(8443)}},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: consoleName, Namespace: namespace, Labels: labels},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{"app.kubernetes.io/component": "console", "app.kubernetes.io/instance": "krkn-operator"},
				Ports:    []corev1.ServicePort{{Name: "http", Port: 8080, TargetPort: intstrFromInt(8080)}},
			},
		},
	}
	for _, service := range resources {
		_, err := clientset.CoreV1().Services(namespace).Get(ctx, service.Name, metav1.GetOptions{})
		if err == nil || apierrors.IsAlreadyExists(err) {
			continue
		}
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("get service %s: %w", service.Name, err)
		}
		if _, err := clientset.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("create service %s: %w", service.Name, err)
		}
	}
	return nil
}

func ensureScenarioRunnerRBAC(ctx context.Context, clientset kubernetes.Interface, namespace string, labels map[string]string, openshift bool) error {
	_, err := clientset.RbacV1().ClusterRoles().Get(ctx, scenarioRunnerRole, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = clientset.RbacV1().ClusterRoles().Create(ctx, &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: scenarioRunnerRole, Labels: labels},
			Rules: []rbacv1.PolicyRule{
				{APIGroups: []string{""}, Resources: []string{"nodes", "pods", "services", "namespaces"}, Verbs: []string{"get", "list", "watch"}},
				{APIGroups: []string{""}, Resources: []string{"pods", "pods/log", "pods/exec"}, Verbs: []string{"create", "delete", "get", "list", "patch", "update", "watch"}},
				{APIGroups: []string{""}, Resources: []string{"nodes"}, Verbs: []string{"get", "list", "patch", "update"}},
			},
		}, metav1.CreateOptions{})
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("create scenario runner cluster role: %w", err)
		}
	} else if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("get scenario runner cluster role: %w", err)
	}

	bindings := clientset.RbacV1().ClusterRoleBindings()
	bindingName := scenarioRunnerBindingName(namespace)
	_, err = bindings.Get(ctx, bindingName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = bindings.Create(ctx, &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: bindingName, Labels: labels},
			RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: scenarioRunnerRole},
			Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: scenarioRunnerSA, Namespace: namespace}},
		}, metav1.CreateOptions{})
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("create scenario runner cluster role binding: %w", err)
		}
	} else if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("get scenario runner cluster role binding: %w", err)
	}
	if openshift {
		if err := ensureSCCBinding(ctx, bindings, namespace, labels); err != nil {
			return err
		}
	}
	return nil
}

func ensureSCCBinding(ctx context.Context, bindings rbacv1typed.ClusterRoleBindingInterface, namespace string, labels map[string]string) error {
	name := scenarioRunnerSCCBindingName(namespace)
	_, err := bindings.Get(ctx, name, metav1.GetOptions{})
	if err == nil || apierrors.IsAlreadyExists(err) {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get scenario runner SCC binding: %w", err)
	}
	_, err = bindings.Create(ctx, &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: scenarioRunnerSCC},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: scenarioRunnerSA, Namespace: namespace}},
	}, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create scenario runner SCC binding: %w", err)
	}
	return nil
}

func scenarioRunnerBindingName(namespace string) string {
	return "krkn-operator-scenario-runner-" + namespaceHash(namespace)
}

func scenarioRunnerSCCBindingName(namespace string) string {
	return "krkn-operator-scenario-runner-scc-" + namespaceHash(namespace)
}

func namespaceHash(namespace string) string {
	digest := sha256.Sum256([]byte(namespace))
	return hex.EncodeToString(digest[:])[:10]
}

func intstrFromInt(value int32) intstr.IntOrString {
	return intstr.FromInt(int(value))
}

func nginxConfig(namespace string) string {
	return fmt.Sprintf(`worker_processes auto;
error_log /tmp/error.log notice;
pid /tmp/nginx.pid;
events { worker_connections 1024; }
http {
  include /etc/nginx/mime.types;
  default_type application/octet-stream;
  map $http_upgrade $connection_upgrade { default upgrade; '' close; }
  client_body_temp_path /tmp/client_temp;
  proxy_temp_path /tmp/proxy_temp;
  server {
    listen 8080;
    server_name localhost;
    location / { root /usr/share/nginx/html; index index.html index.htm; try_files $uri $uri/ /index.html; }
    location /api/ {
      proxy_pass http://%s.%s.svc.cluster.local:8080;
      proxy_set_header Host $host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      proxy_set_header X-Forwarded-Proto $scheme;
      proxy_http_version 1.1;
      proxy_set_header Upgrade $http_upgrade;
      proxy_set_header Connection $connection_upgrade;
      proxy_connect_timeout 120s;
      proxy_send_timeout 3600s;
      proxy_read_timeout 3600s;
    }
  }
}`, operatorName, namespace)
}
