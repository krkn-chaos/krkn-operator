package olm

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilvalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	AIServiceName                    = "krkn-operator-ai-service"
	AIServiceTokenSecretName         = "krkn-operator-ai-service-token" // #nosec G101 -- Kubernetes Secret name, not a credential; newAITokenSecret generates the token with crypto/rand.
	AIOrchestratorServiceAccountName = "krkn-operator-krkn-ai-orchestrator"
	AIServiceURL                     = "http://krkn-operator-ai-service:8080"
	aiServiceAccountName             = AIServiceName
	aiOrchestratorRoleName           = AIOrchestratorServiceAccountName
	aiOrchestratorRoleBindingName    = AIOrchestratorServiceAccountName
	aiTokenKey                       = "token"
	aiTokenEnv                       = "KRKNAI_SERVICE_TOKEN" // #nosec G101 -- Environment variable name, not a credential; its value comes from a SecretKeyRef.
	aiServiceContainerName           = "ai-service"
	aiStorageVolumeName              = "artifacts"
	aiArtifactsMountPath             = "/var/lib/krkn-ai"
	aiReconcileInterval              = 30 * time.Second
	aiStorageDefault                 = "1Gi"
	aiLabelName                      = "krkn-operator"
	aiLabelInstance                  = "krkn-operator"
	aiManagedBy                      = "olm"
	aiComponentService               = "ai-service"
	aiComponentOrchestrator          = "ai-orchestrator"
	aiComponentOperator              = "operator"
	aiOperatorControlPlaneLabel      = "control-plane"
	aiOperatorControlPlaneValue      = "controller-manager"
	aiRunLabel                       = "krkn.dev/ai-run"
	aiSCCRole                        = "system:openshift:scc:anyuid"
)

func aiOrchestratorSCCBindingName(namespace string) string {
	return "krkn-operator-ai-orchestrator-anyuid-" + namespaceHash(namespace)
}

// AIConfig holds the OLM configuration required to provision Krkn-AI.
type AIConfig struct {
	Enabled                     bool
	Namespace                   string
	OpenShift                   bool
	OrchestratorImage           string
	OrchestratorImagePullPolicy corev1.PullPolicy
	ServiceImage                string
	ServiceImagePullPolicy      corev1.PullPolicy
	ImagePullSecrets            []corev1.LocalObjectReference
	ExistingClaim               string
	StorageClassName            string
	StorageSize                 resource.Quantity
	ServiceResources            corev1.ResourceRequirements
	NetworkPolicyEnabled        bool
}

// AIConfigFromEnv reads and validates the fixed Krkn-AI OLM environment contract.
// Image values are required only for enabled installations; Ensure also validates
// them when a token Secret proves that a disabled installation must be retained.
func AIConfigFromEnv(namespace string, getenv func(string) string) (AIConfig, error) {
	if getenv == nil {
		return AIConfig{}, errors.New("read Krkn-AI configuration: getenv function is nil")
	}
	if problems := utilvalidation.IsDNS1123Label(namespace); len(problems) != 0 {
		return AIConfig{}, fmt.Errorf("invalid Krkn-AI namespace %q: %s", namespace, strings.Join(problems, ", "))
	}

	cfg := AIConfig{
		Namespace:                   namespace,
		StorageSize:                 resource.MustParse(aiStorageDefault),
		NetworkPolicyEnabled:        true,
		ServiceResources:            defaultAIServiceResources(),
		OrchestratorImagePullPolicy: corev1.PullAlways,
		ServiceImagePullPolicy:      corev1.PullAlways,
	}
	var err error
	if cfg.Enabled, err = readEnvBool("KRKNAI_ENABLED", false, getenv); err != nil {
		return AIConfig{}, err
	}
	if cfg.OpenShift, err = readEnvBool("KRKN_OLM_OPENSHIFT", false, getenv); err != nil {
		return AIConfig{}, err
	}
	if cfg.NetworkPolicyEnabled, err = readEnvBool("KRKNAI_NETWORK_POLICY_ENABLED", true, getenv); err != nil {
		return AIConfig{}, err
	}
	cfg.OrchestratorImage = strings.TrimSpace(getenv("KRKNAI_ORCHESTRATOR_IMAGE"))
	cfg.ServiceImage = strings.TrimSpace(getenv("KRKNAI_SERVICE_IMAGE"))
	if value := getenv("KRKNAI_ORCHESTRATOR_IMAGE_PULL_POLICY"); value != "" {
		cfg.OrchestratorImagePullPolicy, err = parsePullPolicy("KRKNAI_ORCHESTRATOR_IMAGE_PULL_POLICY", value)
		if err != nil {
			return AIConfig{}, err
		}
	}
	if value := getenv("KRKNAI_SERVICE_IMAGE_PULL_POLICY"); value != "" {
		cfg.ServiceImagePullPolicy, err = parsePullPolicy("KRKNAI_SERVICE_IMAGE_PULL_POLICY", value)
		if err != nil {
			return AIConfig{}, err
		}
	}
	if cfg.ImagePullSecrets, err = parseImagePullSecrets(getenv("KRKNAI_IMAGE_PULL_SECRETS")); err != nil {
		return AIConfig{}, err
	}
	cfg.ExistingClaim = strings.TrimSpace(getenv("KRKNAI_STORAGE_EXISTING_CLAIM"))
	if cfg.ExistingClaim != "" {
		if problems := utilvalidation.IsDNS1123Subdomain(cfg.ExistingClaim); len(problems) != 0 {
			return AIConfig{}, fmt.Errorf("invalid KRKNAI_STORAGE_EXISTING_CLAIM %q: %s", cfg.ExistingClaim, strings.Join(problems, ", "))
		}
	}
	cfg.StorageClassName = strings.TrimSpace(getenv("KRKNAI_STORAGE_CLASS_NAME"))
	if value := getenv("KRKNAI_STORAGE_SIZE"); value != "" {
		cfg.StorageSize, err = resource.ParseQuantity(value)
		if err != nil {
			return AIConfig{}, fmt.Errorf("parse KRKNAI_STORAGE_SIZE %q: %w", value, err)
		}
	}
	if cfg.StorageSize.Sign() <= 0 {
		return AIConfig{}, fmt.Errorf("KRKNAI_STORAGE_SIZE must be greater than zero, got %s", cfg.StorageSize.String())
	}
	if value := getenv("KRKNAI_SERVICE_RESOURCES"); value != "" {
		if cfg.ServiceResources, err = parseServiceResources(value); err != nil {
			return AIConfig{}, err
		}
	}
	if cfg.Enabled {
		if err := validateImages(cfg); err != nil {
			return AIConfig{}, err
		}
	}
	return cfg, nil
}

func readEnvBool(name string, fallback bool, getenv func(string) string) (bool, error) {
	value := getenv(name)
	if value == "" {
		return fallback, nil
	}
	switch value {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("parse %s %q as boolean: expected exactly true or false", name, value)
	}
}

func parsePullPolicy(name, value string) (corev1.PullPolicy, error) {
	policy := corev1.PullPolicy(value)
	switch policy {
	case corev1.PullAlways, corev1.PullIfNotPresent, corev1.PullNever:
		return policy, nil
	default:
		return "", fmt.Errorf("invalid %s %q: expected Always, IfNotPresent, or Never", name, value)
	}
}

func parseImagePullSecrets(value string) ([]corev1.LocalObjectReference, error) {
	if value == "" {
		return nil, nil
	}
	var refs []corev1.LocalObjectReference
	if err := decodeStrictJSON("KRKNAI_IMAGE_PULL_SECRETS", value, &refs); err != nil {
		return nil, err
	}
	if refs == nil {
		return nil, errors.New("parse KRKNAI_IMAGE_PULL_SECRETS: expected a JSON list, not null")
	}
	seen := make(map[string]struct{}, len(refs))
	for index, ref := range refs {
		if problems := utilvalidation.IsDNS1123Subdomain(ref.Name); len(problems) != 0 {
			return nil, fmt.Errorf("KRKNAI_IMAGE_PULL_SECRETS[%d] has invalid name %q: %s", index, ref.Name, strings.Join(problems, ", "))
		}
		if _, ok := seen[ref.Name]; ok {
			return nil, fmt.Errorf("KRKNAI_IMAGE_PULL_SECRETS contains duplicate secret %q", ref.Name)
		}
		seen[ref.Name] = struct{}{}
	}
	return refs, nil
}

func parseServiceResources(value string) (corev1.ResourceRequirements, error) {
	var resources corev1.ResourceRequirements
	if err := decodeStrictJSON("KRKNAI_SERVICE_RESOURCES", value, &resources); err != nil {
		return corev1.ResourceRequirements{}, err
	}
	if strings.TrimSpace(value) == "" || strings.TrimSpace(value)[0] != '{' {
		return corev1.ResourceRequirements{}, errors.New("parse KRKNAI_SERVICE_RESOURCES: expected a JSON object")
	}
	for name, quantity := range resources.Requests {
		if quantity.Sign() < 0 {
			return corev1.ResourceRequirements{}, fmt.Errorf("KRKNAI_SERVICE_RESOURCES requests.%s must not be negative", name)
		}
	}
	for name, quantity := range resources.Limits {
		if quantity.Sign() < 0 {
			return corev1.ResourceRequirements{}, fmt.Errorf("KRKNAI_SERVICE_RESOURCES limits.%s must not be negative", name)
		}
	}
	return resources, nil
}

func decodeStrictJSON(name, value string, target any) error {
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("parse %s JSON: %w", name, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("parse %s JSON: trailing value after JSON document", name)
		}
		return fmt.Errorf("parse %s JSON: trailing data: %w", name, err)
	}
	return nil
}

func defaultAIServiceResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
	}
}

func validateImages(cfg AIConfig) error {
	if err := validateImage("KRKNAI_ORCHESTRATOR_IMAGE", cfg.OrchestratorImage); err != nil {
		return err
	}
	return validateImage("KRKNAI_SERVICE_IMAGE", cfg.ServiceImage)
}

func validateImage(name, image string) error {
	if image == "" {
		return fmt.Errorf("%s is required when Krkn-AI is enabled or an existing installation is retained", name)
	}
	if strings.ContainsAny(image, " \t\r\n") {
		return fmt.Errorf("%s must be a valid non-empty image reference without whitespace", name)
	}
	return nil
}

// AIResources owns the namespaced Krkn-AI installation resources and repairs
// mutable resources while the manager holds leader election.
type AIResources struct {
	Clientset kubernetes.Interface
	Config    AIConfig
}

// Ensure detects an existing installation by its managed token Secret. Disabled
// fresh installs are a no-op; disabled existing installs are retained and kept
// usable for previously-created runs and artifacts.
func (r *AIResources) Ensure(ctx context.Context) (bool, error) {
	if r == nil || r.Clientset == nil {
		return false, errors.New("ensure Krkn-AI resources: Kubernetes clientset is nil")
	}
	if problems := utilvalidation.IsDNS1123Label(r.Config.Namespace); len(problems) != 0 {
		return false, fmt.Errorf("invalid Krkn-AI namespace %q: %s", r.Config.Namespace, strings.Join(problems, ", "))
	}
	secretClient := r.Clientset.CoreV1().Secrets(r.Config.Namespace)
	secret, err := secretClient.Get(ctx, AIServiceTokenSecretName, metav1.GetOptions{})
	installed := err == nil
	if err != nil && !apierrors.IsNotFound(err) {
		return false, fmt.Errorf("get Krkn-AI token Secret: %w", err)
	}
	if installed {
		if err := requireOwned(secret.ObjectMeta, "Secret", AIServiceTokenSecretName); err != nil {
			if !r.Config.Enabled {
				return false, nil
			}
			return false, err
		}
		if len(secret.Data[aiTokenKey]) == 0 {
			return false, fmt.Errorf("owned Krkn-AI token Secret %q has no non-empty %q key", AIServiceTokenSecretName, aiTokenKey)
		}
	}
	if !r.Config.Enabled && !installed {
		return false, nil
	}
	if err := validateImages(r.Config); err != nil {
		return installed, err
	}
	if !installed {
		if err := r.checkExistingResourceOwnership(ctx); err != nil {
			return false, err
		}
	}
	if !installed {
		secret, err = newAITokenSecret(r.Config.Namespace)
		if err != nil {
			return false, err
		}
		secret, err = secretClient.Create(ctx, secret, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			secret, err = secretClient.Get(ctx, AIServiceTokenSecretName, metav1.GetOptions{})
		}
		if err != nil {
			return false, fmt.Errorf("create Krkn-AI token Secret: %w", err)
		}
		if err := requireOwned(secret.ObjectMeta, "Secret", AIServiceTokenSecretName); err != nil {
			return false, err
		}
		if len(secret.Data[aiTokenKey]) == 0 {
			return false, fmt.Errorf("owned Krkn-AI token Secret %q has no non-empty %q key", AIServiceTokenSecretName, aiTokenKey)
		}
		installed = true
	}
	if !hasManagedLabels(secret.Labels, aiComponentService) {
		secret.Labels = mergeLabels(secret.Labels, aiLabels(aiComponentService))
		if _, err := secretClient.Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
			return installed, fmt.Errorf("update Krkn-AI token Secret labels: %w", err)
		}
	}
	if err := r.ensureResources(ctx); err != nil {
		return installed, err
	}
	return installed, nil
}

// Start immediately reconciles and then retries at a fixed interval. Errors are
// logged and retried; cancellation ends the runnable cleanly.
func (r *AIResources) Start(ctx context.Context) error {
	logger := ctrl.Log.WithName("olm-ai-resources")
	if _, err := r.Ensure(ctx); err != nil {
		logger.Error(err, "initial AI resource reconciliation failed")
	}
	ticker := time.NewTicker(aiReconcileInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := r.Ensure(ctx); err != nil {
				logger.Error(err, "AI resource reconciliation failed")
			}
		}
	}
}

// NeedLeaderElection ensures only the elected manager mutates the shared objects.
func (*AIResources) NeedLeaderElection() bool { return true }

func newAITokenSecret(namespace string) (*corev1.Secret, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("generate Krkn-AI service token: %w", err)
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: AIServiceTokenSecretName, Namespace: namespace, Labels: aiLabels(aiComponentService)},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{aiTokenKey: []byte(hex.EncodeToString(bytes))},
	}, nil
}

func (r *AIResources) ensureResources(ctx context.Context) error {
	if err := ensureAIServiceAccount(ctx, r.Clientset, r.Config.Namespace, aiServiceAccountName, aiComponentService, true); err != nil {
		return err
	}
	if err := ensureAIServiceAccount(ctx, r.Clientset, r.Config.Namespace, AIOrchestratorServiceAccountName, aiComponentOrchestrator, false); err != nil {
		return err
	}
	claimName, err := r.ensureClaim(ctx)
	if err != nil {
		return err
	}
	if err := ensureAIService(ctx, r.Clientset, r.Config); err != nil {
		return err
	}
	if err := ensureAIDeployment(ctx, r.Clientset, r.Config, claimName); err != nil {
		return err
	}
	if err := ensureAIOrchestratorRole(ctx, r.Clientset, r.Config.Namespace); err != nil {
		return err
	}
	if err := ensureAIOrchestratorRoleBinding(ctx, r.Clientset, r.Config.Namespace); err != nil {
		return err
	}
	if r.Config.NetworkPolicyEnabled {
		if err := ensureAINetworkPolicy(ctx, r.Clientset, r.Config.Namespace); err != nil {
			return err
		}
	} else if r.Config.Enabled {
		if err := removeAINetworkPolicy(ctx, r.Clientset, r.Config.Namespace); err != nil {
			return err
		}
	}
	if r.Config.OpenShift {
		if err := ensureAIOrchestratorSCCBinding(ctx, r.Clientset, r.Config.Namespace); err != nil {
			return err
		}
	}
	return nil
}
func (r *AIResources) checkExistingResourceOwnership(ctx context.Context) error {
	ns := r.Config.Namespace
	check := func(kind, name string, get func() (metav1.Object, error)) error {
		object, err := get()
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("check existing AI %s %q: %w", kind, name, err)
		}
		return requireOwnedLabels(object.GetLabels(), kind, name)
	}
	if err := check("ServiceAccount", aiServiceAccountName, func() (metav1.Object, error) {
		return r.Clientset.CoreV1().ServiceAccounts(ns).Get(ctx, aiServiceAccountName, metav1.GetOptions{})
	}); err != nil {
		return err
	}
	if err := check("ServiceAccount", AIOrchestratorServiceAccountName, func() (metav1.Object, error) {
		return r.Clientset.CoreV1().ServiceAccounts(ns).Get(ctx, AIOrchestratorServiceAccountName, metav1.GetOptions{})
	}); err != nil {
		return err
	}
	claimName := r.Config.ExistingClaim
	managedClaim := claimName == ""
	if managedClaim {
		claimName = AIServiceName
	}
	claim, err := r.Clientset.CoreV1().PersistentVolumeClaims(ns).Get(ctx, claimName, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("check existing AI artifact PVC %q: %w", claimName, err)
	}
	if err == nil {
		if managedClaim {
			if err := requireOwned(claim.ObjectMeta, "PersistentVolumeClaim", claimName); err != nil {
				return err
			}
		}
		if err := validateAIClaim(claim, claimName, r.Config, managedClaim); err != nil {
			return err
		}
	} else if !managedClaim {
		return fmt.Errorf("get configured user-owned AI artifact PVC %q in namespace %q: %w", claimName, ns, err)
	}
	resources := []struct {
		kind string
		name string
		get  func() (metav1.Object, error)
	}{
		{"Service", AIServiceName, func() (metav1.Object, error) {
			return r.Clientset.CoreV1().Services(ns).Get(ctx, AIServiceName, metav1.GetOptions{})
		}},
		{"Deployment", AIServiceName, func() (metav1.Object, error) {
			return r.Clientset.AppsV1().Deployments(ns).Get(ctx, AIServiceName, metav1.GetOptions{})
		}},
		{"Role", aiOrchestratorRoleName, func() (metav1.Object, error) {
			return r.Clientset.RbacV1().Roles(ns).Get(ctx, aiOrchestratorRoleName, metav1.GetOptions{})
		}},
		{"RoleBinding", aiOrchestratorRoleBindingName, func() (metav1.Object, error) {
			return r.Clientset.RbacV1().RoleBindings(ns).Get(ctx, aiOrchestratorRoleBindingName, metav1.GetOptions{})
		}},
		{"NetworkPolicy", AIServiceName, func() (metav1.Object, error) {
			return r.Clientset.NetworkingV1().NetworkPolicies(ns).Get(ctx, AIServiceName, metav1.GetOptions{})
		}},
	}
	if r.Config.OpenShift {
		name := aiOrchestratorSCCBindingName(ns)
		resources = append(resources, struct {
			kind string
			name string
			get  func() (metav1.Object, error)
		}{"ClusterRoleBinding", name, func() (metav1.Object, error) {
			return r.Clientset.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
		}})
	}
	for _, entry := range resources {
		if err := check(entry.kind, entry.name, entry.get); err != nil {
			return err
		}
	}
	return nil
}

func ensureAIServiceAccount(ctx context.Context, clientset kubernetes.Interface, namespace, name, component string, disableToken bool) error {
	client := clientset.CoreV1().ServiceAccounts(namespace)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := client.Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			automount := !disableToken
			desired := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: aiLabels(component)}, AutomountServiceAccountToken: &automount}
			_, err = client.Create(ctx, desired, metav1.CreateOptions{})
			if apierrors.IsAlreadyExists(err) {
				return aiCreateRace("serviceaccounts", name, err)
			}
			if err != nil {
				return fmt.Errorf("create AI service account %q: %w", name, err)
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("get AI service account %q: %w", name, err)
		}
		if err := requireOwned(current.ObjectMeta, "ServiceAccount", name); err != nil {
			return err
		}
		automount := !disableToken
		if current.AutomountServiceAccountToken == nil || *current.AutomountServiceAccountToken != automount || !hasManagedLabels(current.Labels, component) {
			current.AutomountServiceAccountToken = &automount
			applyLabels(current.Labels, component)
			if _, err := client.Update(ctx, current, metav1.UpdateOptions{}); err != nil {
				return fmt.Errorf("update AI service account %q: %w", name, err)
			}
		}
		return nil
	})
}

func (r *AIResources) ensureClaim(ctx context.Context) (string, error) {
	name := r.Config.ExistingClaim
	managed := name == ""
	if managed {
		name = AIServiceName
	}
	client := r.Clientset.CoreV1().PersistentVolumeClaims(r.Config.Namespace)
	claim, err := client.Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) && !managed {
		return "", fmt.Errorf("get configured user-owned AI artifact PVC %q in namespace %q: %w", name, r.Config.Namespace, err)
	}
	if err != nil && !apierrors.IsNotFound(err) {
		return "", fmt.Errorf("get AI artifact PVC %q: %w", name, err)
	}
	if managed && apierrors.IsNotFound(err) {
		desired := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: r.Config.Namespace, Labels: aiLabels(aiComponentService)},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources:   corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: r.Config.StorageSize.DeepCopy()}},
			},
		}
		if r.Config.StorageClassName != "" {
			desired.Spec.StorageClassName = ptr.To(r.Config.StorageClassName)
		}
		claim, err = client.Create(ctx, desired, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			claim, err = client.Get(ctx, name, metav1.GetOptions{})
		}
		if err != nil {
			return "", fmt.Errorf("create managed AI artifact PVC %q: %w", name, err)
		}
	}
	if err := validateAIClaim(claim, name, r.Config, managed); err != nil {
		return "", err
	}
	return name, nil
}

func validateAIClaim(claim *corev1.PersistentVolumeClaim, name string, cfg AIConfig, managed bool) error {
	if claim == nil {
		return fmt.Errorf("AI artifact PVC %q lookup returned no object", name)
	}
	if managed {
		if err := requireOwned(claim.ObjectMeta, "PersistentVolumeClaim", name); err != nil {
			return err
		}
		requested := claim.Spec.Resources.Requests[corev1.ResourceStorage]
		if requested.Sign() <= 0 || requested.Cmp(cfg.StorageSize) < 0 {
			return fmt.Errorf("managed AI artifact PVC %q requests %s, below configured %s; PVCs are retained and are never shrunk or resized by reconciliation", name, requested.String(), cfg.StorageSize.String())
		}
		if cfg.StorageClassName != "" && (claim.Spec.StorageClassName == nil || *claim.Spec.StorageClassName != cfg.StorageClassName) {
			actual := ""
			if claim.Spec.StorageClassName != nil {
				actual = *claim.Spec.StorageClassName
			}
			return fmt.Errorf("managed AI artifact PVC %q uses storageClassName %q but configuration requests %q; immutable storage class is not changed", name, actual, cfg.StorageClassName)
		}
		if !containsAccessMode(claim.Spec.AccessModes, corev1.ReadWriteOnce) {
			return fmt.Errorf("managed AI artifact PVC %q must support ReadWriteOnce access", name)
		}
	}
	if claim.DeletionTimestamp != nil {
		return fmt.Errorf("AI artifact PVC %q is being deleted", name)
	}
	if len(claim.Spec.AccessModes) == 0 {
		return fmt.Errorf("AI artifact PVC %q has no access modes", name)
	}
	return nil
}

func containsAccessMode(modes []corev1.PersistentVolumeAccessMode, wanted corev1.PersistentVolumeAccessMode) bool {
	for _, mode := range modes {
		if mode == wanted {
			return true
		}
	}
	return false
}

func ensureAIService(ctx context.Context, clientset kubernetes.Interface, cfg AIConfig) error {
	client := clientset.CoreV1().Services(cfg.Namespace)
	componentLabels := aiPodLabels(aiComponentService)
	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: AIServiceName, Namespace: cfg.Namespace, Labels: aiLabels(aiComponentService)},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: componentLabels,
			Ports:    []corev1.ServicePort{{Name: "http", Port: 8080, TargetPort: intstr.FromString("http"), Protocol: corev1.ProtocolTCP}},
		},
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := client.Get(ctx, AIServiceName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = client.Create(ctx, desired, metav1.CreateOptions{})
			if apierrors.IsAlreadyExists(err) {
				return aiCreateRace("services", AIServiceName, err)
			}
			if err != nil {
				return fmt.Errorf("create AI service: %w", err)
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("get AI service: %w", err)
		}
		if err := requireOwned(current.ObjectMeta, "Service", AIServiceName); err != nil {
			return err
		}
		if current.Spec.Type == desired.Spec.Type && equality.Semantic.DeepEqual(current.Spec.Selector, desired.Spec.Selector) && equality.Semantic.DeepEqual(current.Spec.Ports, desired.Spec.Ports) && hasManagedLabels(current.Labels, aiComponentService) {
			return nil
		}
		current.Labels = mergeLabels(current.Labels, desired.Labels)
		current.Spec.Type = desired.Spec.Type
		current.Spec.Selector = desired.Spec.Selector
		current.Spec.Ports = desired.Spec.Ports
		if _, err := client.Update(ctx, current, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update AI service: %w", err)
		}
		return nil
	})
}

func ensureAIDeployment(ctx context.Context, clientset kubernetes.Interface, cfg AIConfig, claimName string) error {
	client := clientset.AppsV1().Deployments(cfg.Namespace)
	desired := aiDeployment(cfg, claimName)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := client.Get(ctx, AIServiceName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = client.Create(ctx, desired, metav1.CreateOptions{})
			if apierrors.IsAlreadyExists(err) {
				return aiCreateRace("deployments", AIServiceName, err)
			}
			if err != nil {
				return fmt.Errorf("create AI service deployment: %w", err)
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("get AI service deployment: %w", err)
		}
		if err := requireOwned(current.ObjectMeta, "Deployment", AIServiceName); err != nil {
			return err
		}
		changed, err := reconcileAIDeployment(current, desired)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}
		if _, err := client.Update(ctx, current, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update AI service deployment: %w", err)
		}
		return nil
	})
}
func reconcileAIDeployment(current, desired *appsv1.Deployment) (bool, error) {
	if !equality.Semantic.DeepEqual(current.Spec.Selector, desired.Spec.Selector) {
		return false, fmt.Errorf("owned AI service Deployment %q has immutable selector %v; expected %v", AIServiceName, current.Spec.Selector, desired.Spec.Selector)
	}
	if aiDeploymentFieldsMatch(current, desired) {
		return false, nil
	}
	current.Labels = mergeLabels(current.Labels, desired.Labels)
	current.Spec.Replicas = desired.Spec.Replicas
	current.Spec.Strategy.Type = desired.Spec.Strategy.Type
	current.Spec.Strategy.RollingUpdate = nil
	current.Spec.Template.Labels = mergeLabels(current.Spec.Template.Labels, desired.Spec.Template.Labels)

	pod := &current.Spec.Template.Spec
	desiredPod := &desired.Spec.Template.Spec
	pod.ServiceAccountName = desiredPod.ServiceAccountName
	pod.AutomountServiceAccountToken = desiredPod.AutomountServiceAccountToken
	pod.ImagePullSecrets = desiredPod.ImagePullSecrets
	if pod.SecurityContext == nil {
		pod.SecurityContext = &corev1.PodSecurityContext{}
	}
	pod.SecurityContext.RunAsNonRoot = desiredPod.SecurityContext.RunAsNonRoot
	pod.SecurityContext.RunAsUser = nil
	pod.SecurityContext.RunAsGroup = nil
	pod.SecurityContext.FSGroup = desiredPod.SecurityContext.FSGroup
	pod.SecurityContext.FSGroupChangePolicy = nil
	pod.SecurityContext.SupplementalGroups = nil
	pod.SecurityContext.SeccompProfile = desiredPod.SecurityContext.SeccompProfile
	pod.Volumes = upsertAIVolume(pod.Volumes, desiredPod.Volumes[0])
	pod.Containers = upsertAIContainer(pod.Containers, &desiredPod.Containers[0])
	return true, nil
}

func aiDeploymentFieldsMatch(current, desired *appsv1.Deployment) bool {
	if current.Spec.Replicas == nil || *current.Spec.Replicas != 1 ||
		current.Spec.Strategy.Type != appsv1.RecreateDeploymentStrategyType ||
		current.Spec.Strategy.RollingUpdate != nil ||
		!hasManagedLabels(current.Labels, aiComponentService) ||
		!hasManagedLabels(current.Spec.Template.Labels, aiComponentService) {
		return false
	}
	pod := &current.Spec.Template.Spec
	desiredPod := &desired.Spec.Template.Spec
	if pod.ServiceAccountName != desiredPod.ServiceAccountName ||
		pod.AutomountServiceAccountToken == nil || *pod.AutomountServiceAccountToken ||
		!localObjectReferencesEqual(pod.ImagePullSecrets, desiredPod.ImagePullSecrets) ||
		!podSecurityContextMatches(pod.SecurityContext, desiredPod.SecurityContext) ||
		!hasExactlyOneAIVolume(pod.Volumes, desiredPod.Volumes[0]) {
		return false
	}
	return hasExactlyOneAIContainer(pod.Containers, &desiredPod.Containers[0])
}

func podSecurityContextMatches(current, desired *corev1.PodSecurityContext) bool {
	if current == nil || current.RunAsNonRoot == nil || !*current.RunAsNonRoot ||
		current.RunAsUser != nil || current.RunAsGroup != nil ||
		current.FSGroupChangePolicy != nil || len(current.SupplementalGroups) != 0 ||
		!equality.Semantic.DeepEqual(current.FSGroup, desired.FSGroup) ||
		!equality.Semantic.DeepEqual(current.SeccompProfile, desired.SeccompProfile) {
		return false
	}
	return true
}

func localObjectReferencesEqual(left, right []corev1.LocalObjectReference) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Name != right[index].Name {
			return false
		}
	}
	return true
}

func hasExactlyOneAIVolume(volumes []corev1.Volume, desired corev1.Volume) bool {
	count := 0
	for _, volume := range volumes {
		if volume.Name == aiStorageVolumeName {
			count++
			if !equality.Semantic.DeepEqual(volume, desired) {
				return false
			}
		}
	}
	return count == 1
}

func upsertAIVolume(volumes []corev1.Volume, desired corev1.Volume) []corev1.Volume {
	result := volumes[:0]
	inserted := false
	for _, volume := range volumes {
		if volume.Name == aiStorageVolumeName {
			if !inserted {
				result = append(result, desired)
				inserted = true
			}
			continue
		}
		result = append(result, volume)
	}
	if !inserted {
		result = append(result, desired)
	}
	return result
}

func hasExactlyOneAIContainer(containers []corev1.Container, desired *corev1.Container) bool {
	count := 0
	for i := range containers {
		container := &containers[i]
		if container.Name == aiServiceContainerName {
			count++
			if !aiContainerMatches(container, desired) {
				return false
			}
		}
	}
	return count == 1
}

func aiContainerMatches(current, desired *corev1.Container) bool {
	return current.Image == desired.Image &&
		current.ImagePullPolicy == desired.ImagePullPolicy &&
		hasExactlyOneNamedValue(current.Ports, desired.Ports[0], func(port corev1.ContainerPort) bool { return port.Name == "http" }) &&
		hasExactlyOneNamedValue(current.Env, desired.Env[0], func(env corev1.EnvVar) bool { return env.Name == aiTokenEnv }) &&
		hasExactlyOneNamedValue(current.VolumeMounts, desired.VolumeMounts[0], func(mount corev1.VolumeMount) bool {
			return mount.Name == aiStorageVolumeName || mount.MountPath == aiArtifactsMountPath
		}) &&
		resourceRequirementsEqual(current.Resources, desired.Resources) &&
		current.SecurityContext != nil &&
		current.SecurityContext.AllowPrivilegeEscalation != nil &&
		!*current.SecurityContext.AllowPrivilegeEscalation &&
		equality.Semantic.DeepEqual(current.SecurityContext.Capabilities, desired.SecurityContext.Capabilities)
}

func hasExactlyOneNamedValue[T any](values []T, desired T, matches func(T) bool) bool {
	count := 0
	for _, value := range values {
		if matches(value) {
			count++
			if !equality.Semantic.DeepEqual(value, desired) {
				return false
			}
		}
	}
	return count == 1
}

func resourceRequirementsEqual(left, right corev1.ResourceRequirements) bool {
	return resourceListsEqual(left.Requests, right.Requests) &&
		resourceListsEqual(left.Limits, right.Limits) &&
		semanticSlicesEqual(left.Claims, right.Claims)
}

func resourceListsEqual(left, right corev1.ResourceList) bool {
	if len(left) != len(right) {
		return false
	}
	for name, quantity := range left {
		other, ok := right[name]
		if !ok || quantity.Cmp(other) != 0 {
			return false
		}
	}
	return true
}

func semanticSlicesEqual[T any](left, right []T) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !equality.Semantic.DeepEqual(left[index], right[index]) {
			return false
		}
	}
	return true
}

func upsertAIContainer(containers []corev1.Container, desired *corev1.Container) []corev1.Container {
	result := containers[:0]
	inserted := false
	for i := range containers {
		container := &containers[i]
		if container.Name == aiServiceContainerName {
			if !inserted {
				reconcileAIContainer(container, desired)
				result = append(result, *container)
				inserted = true
			}
			continue
		}
		result = append(result, *container)
	}
	if !inserted {
		result = append(result, *desired)
	}
	return result
}

func reconcileAIContainer(current, desired *corev1.Container) {
	current.Image = desired.Image
	current.ImagePullPolicy = desired.ImagePullPolicy
	current.Ports = upsertAIValue(current.Ports, desired.Ports[0], func(port corev1.ContainerPort) bool { return port.Name == "http" })
	current.Env = upsertAIValue(current.Env, desired.Env[0], func(env corev1.EnvVar) bool { return env.Name == aiTokenEnv })
	current.VolumeMounts = upsertAIValue(current.VolumeMounts, desired.VolumeMounts[0], func(mount corev1.VolumeMount) bool {
		return mount.Name == aiStorageVolumeName || mount.MountPath == aiArtifactsMountPath
	})
	current.Resources = desired.Resources
	if current.SecurityContext == nil {
		current.SecurityContext = &corev1.SecurityContext{}
	}
	current.SecurityContext.AllowPrivilegeEscalation = desired.SecurityContext.AllowPrivilegeEscalation
	current.SecurityContext.Capabilities = desired.SecurityContext.Capabilities
}

func upsertAIValue[T any](values []T, desired T, matches func(T) bool) []T {
	result := values[:0]
	inserted := false
	for _, value := range values {
		if matches(value) {
			if !inserted {
				result = append(result, desired)
				inserted = true
			}
			continue
		}
		result = append(result, value)
	}
	if !inserted {
		result = append(result, desired)
	}
	return result
}

func aiDeployment(cfg AIConfig, claimName string) *appsv1.Deployment {
	podLabels := aiPodLabels(aiComponentService)
	podSecurityContext := &corev1.PodSecurityContext{
		RunAsNonRoot:   ptr.To(true),
		SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
	}
	if !cfg.OpenShift {
		podSecurityContext.FSGroup = ptr.To[int64](1001)
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: AIServiceName, Namespace: cfg.Namespace, Labels: aiLabels(aiComponentService)},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To[int32](1),
			Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: podLabels},
				Spec: corev1.PodSpec{
					ServiceAccountName:           aiServiceAccountName,
					AutomountServiceAccountToken: ptr.To(false),
					ImagePullSecrets:             cfg.ImagePullSecrets,
					SecurityContext:              podSecurityContext,
					Containers: []corev1.Container{{
						Name:            aiServiceContainerName,
						Image:           cfg.ServiceImage,
						ImagePullPolicy: cfg.ServiceImagePullPolicy,
						Ports:           []corev1.ContainerPort{{Name: "http", ContainerPort: 8080, Protocol: corev1.ProtocolTCP}},
						Env:             []corev1.EnvVar{{Name: aiTokenEnv, ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: AIServiceTokenSecretName}, Key: aiTokenKey}}}},
						VolumeMounts:    []corev1.VolumeMount{{Name: aiStorageVolumeName, MountPath: aiArtifactsMountPath}},
						Resources:       cfg.ServiceResources,
						SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: ptr.To(false), Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}}},
					}},
					Volumes: []corev1.Volume{{Name: aiStorageVolumeName, VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claimName}}}},
				},
			},
		},
	}
}

func ensureAIOrchestratorRole(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	client := clientset.RbacV1().Roles(namespace)
	desired := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: aiOrchestratorRoleName, Namespace: namespace, Labels: aiLabels(aiComponentOrchestrator)},
		Rules: []rbacv1.PolicyRule{
			{APIGroups: []string{"krkn.krkn-chaos.dev"}, Resources: []string{"krknscenarioruns"}, Verbs: []string{"create", "get", "list", "watch", "patch", "delete"}},
			{APIGroups: []string{"krkn.krkn-chaos.dev"}, Resources: []string{"krknscenarioruns/status"}, Verbs: []string{"get"}},
			{APIGroups: []string{"krkn.krkn-chaos.dev"}, Resources: []string{"krknairuns"}, Verbs: []string{"get", "watch"}},
			{APIGroups: []string{"krkn.krkn-chaos.dev"}, Resources: []string{"krknairuns/status"}, Verbs: []string{"get"}},
			{APIGroups: []string{"krkn.krkn-chaos.dev"}, Resources: []string{"krknairuns/finalizers"}, Verbs: []string{"update"}},
			{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get", "list"}},
			{APIGroups: []string{""}, Resources: []string{"pods/log"}, Verbs: []string{"get"}},
		},
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := client.Get(ctx, aiOrchestratorRoleName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = client.Create(ctx, desired, metav1.CreateOptions{})
			if apierrors.IsAlreadyExists(err) {
				return aiCreateRace("roles", aiOrchestratorRoleName, err)
			}
			if err != nil {
				return fmt.Errorf("create AI orchestrator Role: %w", err)
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("get AI orchestrator Role: %w", err)
		}
		if err := requireOwned(current.ObjectMeta, "Role", aiOrchestratorRoleName); err != nil {
			return err
		}
		if equality.Semantic.DeepEqual(current.Rules, desired.Rules) && hasManagedLabels(current.Labels, aiComponentOrchestrator) {
			return nil
		}
		current.Labels = mergeLabels(current.Labels, desired.Labels)
		current.Rules = desired.Rules
		if _, err := client.Update(ctx, current, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update AI orchestrator Role: %w", err)
		}
		return nil
	})
}

func ensureAIOrchestratorRoleBinding(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	client := clientset.RbacV1().RoleBindings(namespace)
	desired := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: aiOrchestratorRoleBindingName, Namespace: namespace, Labels: aiLabels(aiComponentOrchestrator)},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "Role", Name: aiOrchestratorRoleName},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: AIOrchestratorServiceAccountName, Namespace: namespace}},
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := client.Get(ctx, aiOrchestratorRoleBindingName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = client.Create(ctx, desired, metav1.CreateOptions{})
			if apierrors.IsAlreadyExists(err) {
				return aiCreateRace("rolebindings", aiOrchestratorRoleBindingName, err)
			}
			if err != nil {
				return fmt.Errorf("create AI orchestrator RoleBinding: %w", err)
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("get AI orchestrator RoleBinding: %w", err)
		}
		if err := requireOwned(current.ObjectMeta, "RoleBinding", aiOrchestratorRoleBindingName); err != nil {
			return err
		}
		if current.RoleRef != desired.RoleRef {
			return fmt.Errorf("owned AI orchestrator RoleBinding %q references immutable roleRef %s/%s; expected Role/%s", aiOrchestratorRoleBindingName, current.RoleRef.Kind, current.RoleRef.Name, aiOrchestratorRoleName)
		}
		if equality.Semantic.DeepEqual(current.Subjects, desired.Subjects) && hasManagedLabels(current.Labels, aiComponentOrchestrator) {
			return nil
		}
		current.Labels = mergeLabels(current.Labels, desired.Labels)
		current.Subjects = desired.Subjects
		if _, err := client.Update(ctx, current, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update AI orchestrator RoleBinding: %w", err)
		}
		return nil
	})
}

func ensureAIOrchestratorSCCBinding(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	client := clientset.RbacV1().ClusterRoleBindings()
	desired := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: aiOrchestratorSCCBindingName(namespace), Labels: aiLabels(aiComponentOrchestrator)},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: aiSCCRole},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: AIOrchestratorServiceAccountName, Namespace: namespace}},
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := client.Get(ctx, aiOrchestratorSCCBindingName(namespace), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = client.Create(ctx, desired, metav1.CreateOptions{})
			if apierrors.IsAlreadyExists(err) {
				return aiCreateRace("clusterrolebindings", aiOrchestratorSCCBindingName(namespace), err)
			}
			if err != nil {
				return fmt.Errorf("create AI orchestrator SCC ClusterRoleBinding: %w", err)
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("get AI orchestrator SCC ClusterRoleBinding: %w", err)
		}
		if err := requireOwned(current.ObjectMeta, "ClusterRoleBinding", aiOrchestratorSCCBindingName(namespace)); err != nil {
			return err
		}
		if current.RoleRef != desired.RoleRef {
			return fmt.Errorf("owned AI orchestrator SCC ClusterRoleBinding %q references immutable roleRef %s/%s; expected ClusterRole/%s", aiOrchestratorSCCBindingName(namespace), current.RoleRef.Kind, current.RoleRef.Name, aiSCCRole)
		}
		if equality.Semantic.DeepEqual(current.Subjects, desired.Subjects) && hasManagedLabels(current.Labels, aiComponentOrchestrator) {
			return nil
		}
		current.Labels = mergeLabels(current.Labels, desired.Labels)
		current.Subjects = desired.Subjects
		if _, err := client.Update(ctx, current, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update AI orchestrator SCC ClusterRoleBinding: %w", err)
		}
		return nil
	})
}

func ensureAINetworkPolicy(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	client := clientset.NetworkingV1().NetworkPolicies(namespace)
	desired := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: AIServiceName, Namespace: namespace, Labels: aiLabels(aiComponentService)},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: aiPodLabels(aiComponentService)},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{
					{PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{
						"app.kubernetes.io/name": aiLabelName, "app.kubernetes.io/instance": aiLabelInstance,
						"app.kubernetes.io/component": aiComponentOperator, aiOperatorControlPlaneLabel: aiOperatorControlPlaneValue,
					}}},
					{PodSelector: &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{{Key: aiRunLabel, Operator: metav1.LabelSelectorOpExists}}}},
				},
				Ports: []networkingv1.NetworkPolicyPort{{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(8080))}},
			}},
		},
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := client.Get(ctx, AIServiceName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = client.Create(ctx, desired, metav1.CreateOptions{})
			if apierrors.IsAlreadyExists(err) {
				return aiCreateRace("networkpolicies", AIServiceName, err)
			}
			if err != nil {
				return fmt.Errorf("create AI service NetworkPolicy: %w", err)
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("get AI service NetworkPolicy: %w", err)
		}
		if err := requireOwned(current.ObjectMeta, "NetworkPolicy", AIServiceName); err != nil {
			return err
		}
		if equality.Semantic.DeepEqual(current.Spec, desired.Spec) && hasManagedLabels(current.Labels, aiComponentService) {
			return nil
		}
		current.Labels = mergeLabels(current.Labels, desired.Labels)
		current.Spec = desired.Spec
		if _, err := client.Update(ctx, current, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("update AI service NetworkPolicy: %w", err)
		}
		return nil
	})
}
func removeAINetworkPolicy(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	client := clientset.NetworkingV1().NetworkPolicies(namespace)
	current, err := client.Get(ctx, AIServiceName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get AI service NetworkPolicy before removal: %w", err)
	}
	if err := requireOwned(current.ObjectMeta, "NetworkPolicy", AIServiceName); err != nil {
		return err
	}
	if err := client.Delete(ctx, AIServiceName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete AI service NetworkPolicy: %w", err)
	}
	return nil
}

func aiLabels(component string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       aiLabelName,
		"app.kubernetes.io/instance":   aiLabelInstance,
		"app.kubernetes.io/managed-by": aiManagedBy,
		"app.kubernetes.io/component":  component,
	}
}

func aiPodLabels(component string) map[string]string {
	return aiLabels(component)
}

func requireOwned(meta metav1.ObjectMeta, kind, name string) error {
	return requireOwnedLabels(meta.Labels, kind, name)
}

func requireOwnedLabels(actual map[string]string, kind, name string) error {
	if actual["app.kubernetes.io/managed-by"] != aiManagedBy ||
		actual["app.kubernetes.io/name"] != aiLabelName ||
		actual["app.kubernetes.io/instance"] != aiLabelInstance {
		return fmt.Errorf("refusing to manage foreign %s %q: expected ownership labels app.kubernetes.io/managed-by=%s, app.kubernetes.io/name=%s, app.kubernetes.io/instance=%s", kind, name, aiManagedBy, aiLabelName, aiLabelInstance)
	}
	return nil
}

func hasManagedLabels(actual map[string]string, component string) bool {
	return actual["app.kubernetes.io/name"] == aiLabelName &&
		actual["app.kubernetes.io/instance"] == aiLabelInstance &&
		actual["app.kubernetes.io/managed-by"] == aiManagedBy &&
		actual["app.kubernetes.io/component"] == component
}

func applyLabels(actual map[string]string, component string) {
	for key, value := range aiLabels(component) {
		actual[key] = value
	}
}

func mergeLabels(current, desired map[string]string) map[string]string {
	merged := make(map[string]string, len(current)+len(desired))
	for key, value := range current {
		merged[key] = value
	}
	for key, value := range desired {
		merged[key] = value
	}
	return merged
}

// Keep this assertion close to the implementation: controller-runtime's Runnable
// requires Start and is consumed by the parent manager wiring.
var _ interface {
	Start(context.Context) error
	NeedLeaderElection() bool
} = (*AIResources)(nil)

func aiCreateRace(resourceName, name string, cause error) error {
	return apierrors.NewConflict(schema.GroupResource{Resource: resourceName}, name, cause)
}
