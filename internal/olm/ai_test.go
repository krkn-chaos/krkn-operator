package olm

import (
	"context"
	"errors"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestAIConfigFromEnvDefaultsOffWithoutImages(t *testing.T) {
	config, err := AIConfigFromEnv("test-namespace", func(string) string { return "" })
	if err != nil {
		t.Fatalf("AIConfigFromEnv() error = %v", err)
	}
	if config.Enabled {
		t.Fatal("Krkn-AI must default to disabled")
	}

	clientset := fake.NewSimpleClientset()
	installed, err := (&AIResources{Clientset: clientset, Config: config}).Ensure(context.Background())
	if err != nil {
		t.Fatalf("Ensure() for disabled fresh install error = %v", err)
	}
	if installed {
		t.Fatal("disabled fresh install reported installed")
	}
	secrets, err := clientset.CoreV1().Secrets(config.Namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("list Secrets: %v", err)
	}
	if len(secrets.Items) != 0 {
		t.Fatalf("disabled fresh install created %d Secret(s)", len(secrets.Items))
	}
	deployments, err := clientset.AppsV1().Deployments(config.Namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("list Deployments: %v", err)
	}
	if len(deployments.Items) != 0 {
		t.Fatalf("disabled fresh install created %d Deployment(s)", len(deployments.Items))
	}
}

func TestAIResourcesStableTokenAndReconcilesDeploymentDrift(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset()
	config := testAIConfig("ai-system")
	resources := &AIResources{Clientset: clientset, Config: config}
	installed, err := resources.Ensure(ctx)
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if !installed {
		t.Fatal("enabled install reported not installed")
	}
	secret, err := clientset.CoreV1().Secrets(config.Namespace).Get(ctx, AIServiceTokenSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get token Secret: %v", err)
	}
	token := string(secret.Data[aiTokenKey])

	deployment, err := clientset.AppsV1().Deployments(config.Namespace).Get(ctx, AIServiceName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get artifact Deployment: %v", err)
	}
	updates := 0
	clientset.PrependReactor("update", "*", func(ktesting.Action) (bool, runtime.Object, error) {
		updates++
		return false, nil, nil
	})
	if _, err := resources.Ensure(ctx); err != nil {
		t.Fatalf("idempotent Ensure() error = %v", err)
	}
	if updates != 0 {
		t.Fatalf("idempotent Ensure() performed %d updates", updates)
	}

	// A subsequent reconcile must preserve the credential and repair image drift.
	deployment.Spec.Template.Spec.Containers[0].Image = "old/image:tag"
	if _, err := clientset.AppsV1().Deployments(config.Namespace).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("inject Deployment drift: %v", err)
	}
	config.ServiceImage = "registry.example/krkn-ai-service:v2"
	config.OrchestratorImage = "registry.example/krkn-ai-operator:v2"
	resources.Config = config
	if installed, err := resources.Ensure(ctx); err != nil || !installed {
		t.Fatalf("Ensure() after image change = (%v, %v)", installed, err)
	}
	deployment, err = clientset.AppsV1().Deployments(config.Namespace).Get(ctx, AIServiceName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get upgraded Deployment: %v", err)
	}
	if got := deployment.Spec.Template.Spec.Containers[0].Image; got != config.ServiceImage {
		t.Fatalf("reconciled service image = %q, want %q", got, config.ServiceImage)
	}
	secret, err = clientset.CoreV1().Secrets(config.Namespace).Get(ctx, AIServiceTokenSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get stable token Secret: %v", err)
	}
	if got := string(secret.Data[aiTokenKey]); got != token {
		t.Fatalf("service token changed during upgrade: got %q, want original token", got)
	}

	// Turning the opt-in off after installation retains the service and token.
	config.Enabled = false
	resources.Config = config
	if installed, err := resources.Ensure(ctx); err != nil || !installed {
		t.Fatalf("Ensure() after opt-out = (%v, %v)", installed, err)
	}
	if _, err := clientset.AppsV1().Deployments(config.Namespace).Get(ctx, AIServiceName, metav1.GetOptions{}); err != nil {
		t.Fatalf("disabled installed configuration removed artifact Deployment: %v", err)
	}
	if _, err := clientset.CoreV1().Secrets(config.Namespace).Get(ctx, AIServiceTokenSecretName, metav1.GetOptions{}); err != nil {
		t.Fatalf("disabled installed configuration removed token Secret: %v", err)
	}
	if _, err := clientset.CoreV1().PersistentVolumeClaims(config.Namespace).Get(ctx, AIServiceName, metav1.GetOptions{}); err != nil {
		t.Fatalf("disabled installed configuration removed managed artifact PVC: %v", err)
	}
}

func TestAIResourcesPreservesDefaultedDeploymentFields(t *testing.T) {
	ctx := context.Background()
	config := testAIConfig("defaulted-deployment")
	clientset := fake.NewSimpleClientset()
	resources := &AIResources{Clientset: clientset, Config: config}
	if _, err := resources.Ensure(ctx); err != nil {
		t.Fatalf("initial Ensure() error = %v", err)
	}
	deployments := clientset.AppsV1().Deployments(config.Namespace)
	deployment, err := deployments.Get(ctx, AIServiceName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get artifact Deployment: %v", err)
	}
	deployment.Annotations = map[string]string{"user.example/owner": "operator-team"}
	deployment.Spec.RevisionHistoryLimit = new(int32(10))
	deployment.Spec.ProgressDeadlineSeconds = new(int32(600))
	deployment.Spec.Template.Annotations = map[string]string{"user.example/template": "retained"}
	deployment.Spec.Template.Spec.DNSPolicy = corev1.DNSClusterFirst
	deployment.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyAlways
	deployment.Spec.Template.Spec.SchedulerName = "default-scheduler"
	deployment.Spec.Template.Spec.TerminationGracePeriodSeconds = new(int64(30))
	deployment.Spec.Template.Spec.EnableServiceLinks = new(true)
	deployment.Spec.Template.Spec.NodeSelector = map[string]string{"pool": "general"}
	container := &deployment.Spec.Template.Spec.Containers[0]
	container.TerminationMessagePath = corev1.TerminationMessagePathDefault
	container.TerminationMessagePolicy = corev1.TerminationMessageReadFile
	if _, err := deployments.Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("seed API-defaulted Deployment fields: %v", err)
	}
	expected := deployment.DeepCopy()

	updates := 0
	clientset.PrependReactor("update", "*", func(ktesting.Action) (bool, runtime.Object, error) {
		updates++
		return false, nil, nil
	})
	if _, err := resources.Ensure(ctx); err != nil {
		t.Fatalf("unchanged Ensure() error = %v", err)
	}
	if updates != 0 {
		t.Fatalf("unchanged API-defaulted resources caused %d updates", updates)
	}

	config.ServiceImage = "registry.example/krkn-ai-service:v2"
	resources.Config = config
	if _, err := resources.Ensure(ctx); err != nil {
		t.Fatalf("Ensure() after image upgrade error = %v", err)
	}
	if updates != 1 {
		t.Fatalf("image upgrade performed %d updates, want only the Deployment update", updates)
	}
	deployment, err = deployments.Get(ctx, AIServiceName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get upgraded artifact Deployment: %v", err)
	}
	expected.Spec.Template.Spec.Containers[0].Image = config.ServiceImage
	if !equality.Semantic.DeepEqual(deployment.Spec, expected.Spec) ||
		!equality.Semantic.DeepEqual(deployment.Annotations, expected.Annotations) {
		t.Fatal("image upgrade changed fields other than the configured service image")
	}

	// Empty configuration explicitly clears owned pull secrets and resources.
	config.ImagePullSecrets = nil
	config.ServiceResources = corev1.ResourceRequirements{}
	resources.Config = config
	if _, err := resources.Ensure(ctx); err != nil {
		t.Fatalf("Ensure() clearing pull secrets/resources error = %v", err)
	}
	deployment, err = deployments.Get(ctx, AIServiceName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get Deployment after clearing owned fields: %v", err)
	}
	container = &deployment.Spec.Template.Spec.Containers[0]
	if len(deployment.Spec.Template.Spec.ImagePullSecrets) != 0 ||
		len(container.Resources.Requests) != 0 || len(container.Resources.Limits) != 0 {
		t.Fatalf("empty configuration did not clear owned pull secrets/resources: pod=%#v container=%#v", deployment.Spec.Template.Spec, container)
	}
}

func TestAIResourcesUsesNondefaultNamespaceAndNamespaceScopedSCCBindings(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset()
	for _, namespace := range []string{"team-one", "team-two"} {
		config := testAIConfig(namespace)
		config.OpenShift = true
		resources := &AIResources{Clientset: clientset, Config: config}
		if _, err := resources.Ensure(ctx); err != nil {
			t.Fatalf("Ensure(%s) error = %v", namespace, err)
		}
		bindingName := aiOrchestratorSCCBindingName(namespace)
		binding, err := clientset.RbacV1().ClusterRoleBindings().Get(ctx, bindingName, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get SCC binding %q: %v", bindingName, err)
		}
		if binding.RoleRef.Kind != "ClusterRole" || binding.RoleRef.Name != aiSCCRole {
			t.Fatalf("SCC binding roleRef = %#v", binding.RoleRef)
		}
		if len(binding.Subjects) != 1 || binding.Subjects[0].Kind != "ServiceAccount" || binding.Subjects[0].Name != AIOrchestratorServiceAccountName || binding.Subjects[0].Namespace != namespace {
			t.Fatalf("SCC binding subjects = %#v", binding.Subjects)
		}
		if binding.Labels["app.kubernetes.io/managed-by"] != "olm" {
			t.Fatalf("SCC binding ownership label = %q", binding.Labels["app.kubernetes.io/managed-by"])
		}
		if _, err := clientset.AppsV1().Deployments(namespace).Get(ctx, AIServiceName, metav1.GetOptions{}); err != nil {
			t.Fatalf("artifact Deployment not installed in %q: %v", namespace, err)
		}
		if _, err := clientset.AppsV1().Deployments("default").Get(ctx, AIServiceName, metav1.GetOptions{}); err == nil && namespace == "team-one" {
			t.Fatal("AI Deployment was installed into default namespace")
		}
		if podSpec := func() corev1.PodSpec {
			deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, AIServiceName, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("get OpenShift artifact Deployment: %v", err)
			}
			return deployment.Spec.Template.Spec
		}(); podSpec.SecurityContext == nil || podSpec.SecurityContext.FSGroup != nil || podSpec.SecurityContext.RunAsUser != nil {
			t.Fatalf("OpenShift artifact Pod must leave UID/fsGroup assignment to SCC admission: %#v", podSpec.SecurityContext)
		}
	}
	if aiOrchestratorSCCBindingName("team-one") == aiOrchestratorSCCBindingName("team-two") {
		t.Fatal("SCC binding names must differ between namespaces")
	}
	if _, err := clientset.CoreV1().ServiceAccounts("team-one").Get(ctx, AIOrchestratorServiceAccountName, metav1.GetOptions{}); err != nil {
		t.Fatalf("orchestrator ServiceAccount not created: %v", err)
	}
	roleBinding, err := clientset.RbacV1().RoleBindings("team-one").Get(ctx, aiOrchestratorRoleBindingName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get namespaced RoleBinding: %v", err)
	}
	if len(roleBinding.Subjects) != 1 || roleBinding.Subjects[0].Namespace != "team-one" {
		t.Fatalf("namespaced RoleBinding subjects = %#v", roleBinding.Subjects)
	}
}

func TestAIResourcesLeavesUserPVCUnchanged(t *testing.T) {
	ctx := context.Background()
	config := testAIConfig("custom-namespace")
	config.ExistingClaim = "user-artifacts"
	original := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: config.ExistingClaim, Namespace: config.Namespace, Labels: map[string]string{"owner": "user"}},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources:        corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("2Gi")}},
			StorageClassName: new("user-storage"),
		},
	}
	clientset := fake.NewSimpleClientset(original)
	if _, err := (&AIResources{Clientset: clientset, Config: config}).Ensure(ctx); err != nil {
		t.Fatalf("Ensure() with user-owned PVC error = %v", err)
	}
	stored, err := clientset.CoreV1().PersistentVolumeClaims(config.Namespace).Get(ctx, config.ExistingClaim, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get user-owned PVC: %v", err)
	}
	storedSize := stored.Spec.Resources.Requests[corev1.ResourceStorage]
	if stored.Spec.StorageClassName == nil || *stored.Spec.StorageClassName != "user-storage" || storedSize.Cmp(resource.MustParse("2Gi")) != 0 || stored.Labels["owner"] != "user" {
		t.Fatalf("user-owned PVC was changed: %#v", stored)
	}
	deployment, err := clientset.AppsV1().Deployments(config.Namespace).Get(ctx, AIServiceName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get artifact Deployment: %v", err)
	}
	if got := deployment.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName; got != config.ExistingClaim {
		t.Fatalf("artifact volume claim = %q, want %q", got, config.ExistingClaim)
	}
	if _, err := clientset.CoreV1().PersistentVolumeClaims(config.Namespace).Get(ctx, AIServiceName, metav1.GetOptions{}); err == nil {
		t.Fatal("configured user claim should suppress creation of the managed PVC")
	}
}

func TestAIResourcesRejectsManagedClaimConflictsWithoutMutation(t *testing.T) {
	ctx := context.Background()
	config := testAIConfig("claim-conflict")
	claim := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: AIServiceName, Namespace: config.Namespace, Labels: aiLabels(aiComponentService)},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources:   corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("512Mi")}},
		},
	}
	clientset := fake.NewSimpleClientset(claim)
	if _, err := (&AIResources{Clientset: clientset, Config: config}).Ensure(ctx); err == nil {
		t.Fatalf("Ensure() error = %v, want explicit managed PVC size conflict", err)
	}
	stored, err := clientset.CoreV1().PersistentVolumeClaims(config.Namespace).Get(ctx, AIServiceName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get unchanged PVC: %v", err)
	}
	storedSize := stored.Spec.Resources.Requests[corev1.ResourceStorage]
	if storedSize.Cmp(resource.MustParse("512Mi")) != 0 {
		t.Fatalf("managed PVC request was changed to %s", storedSize.String())
	}
	if _, err := clientset.CoreV1().Secrets(config.Namespace).Get(ctx, AIServiceTokenSecretName, metav1.GetOptions{}); err == nil {
		t.Fatal("invalid configuration must not create an installation marker Secret")
	}
}

func TestAIConfigFromEnvRejectsInvalidInputs(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
	}{
		{name: "enabled boolean", env: map[string]string{"KRKNAI_ENABLED": "sometimes"}},
		{name: "openshift boolean", env: map[string]string{"KRKN_OLM_OPENSHIFT": "yes"}},
		{name: "network policy boolean", env: map[string]string{"KRKNAI_NETWORK_POLICY_ENABLED": "1"}},
		{name: "missing enabled images", env: map[string]string{"KRKNAI_ENABLED": "true"}},
		{name: "invalid pull policy", env: map[string]string{"KRKNAI_SERVICE_IMAGE_PULL_POLICY": "OnChange"}},
		{name: "invalid pull secrets", env: map[string]string{"KRKNAI_IMAGE_PULL_SECRETS": "not-json"}},
		{name: "unknown pull secret field", env: map[string]string{"KRKNAI_IMAGE_PULL_SECRETS": `[{"name":"pull","unexpected":true}]`}},
		{name: "trailing pull secrets value", env: map[string]string{"KRKNAI_IMAGE_PULL_SECRETS": `[{"name":"pull"}] {}`}},
		{name: "null pull secrets", env: map[string]string{"KRKNAI_IMAGE_PULL_SECRETS": "null"}},
		{name: "invalid quantity", env: map[string]string{"KRKNAI_STORAGE_SIZE": "not-a-quantity"}},
		{name: "nonpositive quantity", env: map[string]string{"KRKNAI_STORAGE_SIZE": "0"}},
		{name: "invalid service resource quantity", env: map[string]string{"KRKNAI_SERVICE_RESOURCES": `{"requests":{"cpu":"lots"}}`}},
		{name: "unknown service resource field", env: map[string]string{"KRKNAI_SERVICE_RESOURCES": `{"requests":{"cpu":"1"},"unexpected":{}}`}},
		{name: "trailing service resources value", env: map[string]string{"KRKNAI_SERVICE_RESOURCES": `{} {}`}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := AIConfigFromEnv("valid-namespace", func(name string) string { return test.env[name] })
			if err == nil {
				t.Fatal("AIConfigFromEnv() unexpectedly accepted invalid input")
			}
		})
	}
}

func TestAIResourcesRefusesForeignObjectsAndPropagatesAPIErrors(t *testing.T) {
	ctx := context.Background()
	config := testAIConfig("foreign-test")
	foreign := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name: AIServiceName, Namespace: config.Namespace,
		Labels: map[string]string{"app.kubernetes.io/managed-by": "Helm"},
	}}
	clientset := fake.NewSimpleClientset(foreign)
	if _, err := (&AIResources{Clientset: clientset, Config: config}).Ensure(ctx); err == nil || !strings.Contains(err.Error(), "refusing to manage foreign Deployment") {
		t.Fatalf("Ensure() with foreign Deployment error = %v", err)
	}
	if _, err := clientset.CoreV1().Secrets(config.Namespace).Get(ctx, AIServiceTokenSecretName, metav1.GetOptions{}); err == nil {
		t.Fatal("foreign resource check must precede creation of the token marker")
	}

	apiFailure := errors.New("service API unavailable")
	clientset = fake.NewSimpleClientset()
	clientset.PrependReactor("get", "services", func(ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, apiFailure
	})
	_, err := (&AIResources{Clientset: clientset, Config: testAIConfig("error-test")}).Ensure(ctx)
	if !errors.Is(err, apiFailure) {
		t.Fatalf("Ensure() error = %v, want wrapped API error %v", err, apiFailure)
	}
}

func TestAIResourcesReconcilesNetworkPolicyDrift(t *testing.T) {
	ctx := context.Background()
	config := testAIConfig("policy-test")
	clientset := fake.NewSimpleClientset()
	resources := &AIResources{Clientset: clientset, Config: config}
	if _, err := resources.Ensure(ctx); err != nil {
		t.Fatalf("initial Ensure() error = %v", err)
	}
	policy, err := clientset.NetworkingV1().NetworkPolicies(config.Namespace).Get(ctx, AIServiceName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get NetworkPolicy: %v", err)
	}
	policy.Spec.Ingress = nil
	if _, err := clientset.NetworkingV1().NetworkPolicies(config.Namespace).Update(ctx, policy, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("inject NetworkPolicy drift: %v", err)
	}
	if _, err := resources.Ensure(ctx); err != nil {
		t.Fatalf("Ensure() after NetworkPolicy drift error = %v", err)
	}
	policy, err = clientset.NetworkingV1().NetworkPolicies(config.Namespace).Get(ctx, AIServiceName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get reconciled NetworkPolicy: %v", err)
	}
	if len(policy.Spec.Ingress) != 1 || len(policy.Spec.Ingress[0].From) != 2 {
		t.Fatalf("NetworkPolicy drift was not repaired: %#v", policy.Spec.Ingress)
	}
}

func testAIConfig(namespace string) AIConfig {
	return AIConfig{
		Enabled:                     true,
		Namespace:                   namespace,
		OrchestratorImage:           "registry.example/krkn-ai-orchestrator:v1",
		OrchestratorImagePullPolicy: corev1.PullIfNotPresent,
		ServiceImage:                "registry.example/krkn-ai-service:v1",
		ServiceImagePullPolicy:      corev1.PullIfNotPresent,
		ImagePullSecrets:            []corev1.LocalObjectReference{{Name: "private-registry"}},
		StorageSize:                 resource.MustParse("1Gi"),
		ServiceResources:            defaultAIServiceResources(),
		NetworkPolicyEnabled:        true,
	}
}
