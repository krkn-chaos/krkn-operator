package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	registryutil "github.com/krkn-chaos/krkn-operator/pkg/registry"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func addAIRegistrySecret(t *testing.T, c client.Client) *corev1.Secret {
	t.Helper()
	auth, err := registryutil.BuildDockerConfigJSON("registry.example:5000", "synthetic-user", "synthetic-password")
	if err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "corporate", Namespace: "operator",
			Labels:      registryutil.BuildRegistryLabels(registryutil.AuthTypePassword, nil, true),
			Annotations: registryutil.BuildRegistryAnnotations("registry.example:5000", "team/shared", "", false, false, "test"),
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{corev1.DockerConfigJsonKey: auth},
	}
	if err := c.Create(context.Background(), secret); err != nil {
		t.Fatal(err)
	}
	return secret
}

func privateAIRunFixture(t *testing.T, validationStatus int) (*KrknAIRunReconciler, client.WithWatch, *krknv1alpha1.KrknAIRun) {
	t.Helper()
	r, c, run := newAIRunFixture(t)
	addAIRegistrySecret(t, c)
	run.Spec.RegistryName = "corporate"
	if err := c.Update(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	if err := c.Create(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: r.ServiceTokenSecretName, Namespace: run.Namespace},
		Data:       map[string][]byte{"token": []byte("synthetic-service-token")},
	}); err != nil {
		t.Fatal(err)
	}
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer synthetic-service-token" {
			t.Error("service request lacked its installation credential")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/v1/images/requirements":
			_, _ = w.Write([]byte(`{"scenarioTags":["node-cpu-hog"],"workloadImageTags":{"hog":"workload-krkn-hog"}}`))
		case "/v1/configs/validate":
			var body struct {
				Inventory *registryutil.AIImageInventory `json:"imageInventory"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil || body.Inventory == nil || body.Inventory.RegistryName != "corporate" {
				t.Error("private image validation lacked the resolved registry inventory")
			}
			w.WriteHeader(validationStatus)
			if validationStatus == http.StatusOK {
				_, _ = w.Write([]byte(`{"valid":true}`))
			} else {
				_, _ = w.Write([]byte(`{"errors":[{"path":"scenario.node-cpu-hog","message":"required helper image unavailable"}]}`))
			}
		default:
			http.NotFound(w, request)
		}
	}))
	t.Cleanup(service.Close)
	r.ServiceURL = service.URL
	r.registryInspector = func(_ context.Context, secret *corev1.Secret, tags []string, helpers map[string]string) (*registryutil.AIImageInventory, error) {
		if secret.Name != "corporate" || len(tags) != 1 || tags[0] != "node-cpu-hog" || helpers["hog"] != "workload-krkn-hog" {
			t.Fatal("image inspection did not use the service's canonical requirements")
		}
		return &registryutil.AIImageInventory{
			RegistryName: "corporate", Repository: "registry.example:5000/team/shared",
			AvailableTags:       []string{"node-cpu-hog", "workload-krkn-hog"},
			WorkloadImageTags:   map[string]string{"hog": "workload-krkn-hog"},
			OrchestratorImage:   "registry.example:5000/team/shared@sha256:" + strings.Repeat("a", 64),
			ResultUploaderImage: "registry.example:5000/team/shared@sha256:" + strings.Repeat("b", 64),
		}, nil
	}
	return r, c, run
}

func TestPrivateAIRunUsesSelectedRuntimeImagesAndContext(t *testing.T) {
	r, c, run := privateAIRunFixture(t, http.StatusOK)
	installationImage := r.OrchestratorImage
	if err := r.ensureProvisioned(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	var pod corev1.Pod
	if err := c.Get(context.Background(), client.ObjectKey{Name: run.Status.OrchestratorPodName, Namespace: run.Namespace}, &pod); err != nil {
		t.Fatal(err)
	}
	if pod.Spec.Containers[0].Image != "registry.example:5000/team/shared@sha256:"+strings.Repeat("a", 64) ||
		pod.Spec.Containers[1].Image != "registry.example:5000/team/shared@sha256:"+strings.Repeat("b", 64) {
		t.Fatalf("runtime images did not come from the selected shared repository: %s, %s", pod.Spec.Containers[0].Image, pod.Spec.Containers[1].Image)
	}
	if r.OrchestratorImage != installationImage {
		t.Fatal("a private run changed the installation-wide image")
	}
	if len(pod.Spec.ImagePullSecrets) != 2 || pod.Spec.ImagePullSecrets[1].Name != "corporate" {
		t.Fatalf("selected pull Secret was not added alongside installation credentials: %+v", pod.Spec.ImagePullSecrets)
	}
	env := map[string]string{}
	for _, value := range pod.Spec.Containers[0].Env {
		env[value.Name] = value.Value
		if strings.Contains(value.Value, "synthetic-password") {
			t.Fatal("registry credentials leaked into Pod environment")
		}
	}
	if env["KRKNAI_OPERATOR_REGISTRY_NAME"] != "corporate" {
		t.Fatal("registry selection was lost before the CLI")
	}
	var inventory registryutil.AIImageInventory
	if err := json.Unmarshal([]byte(env["KRKNAI_IMAGE_INVENTORY"]), &inventory); err != nil || inventory.Repository != "registry.example:5000/team/shared" {
		t.Fatal("orchestrator received invalid image inventory")
	}
}

func TestPrivateAIRunValidatesImagesBeforeCreatingWorkloads(t *testing.T) {
	for _, status := range []int{http.StatusUnprocessableEntity, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			r, c, run := privateAIRunFixture(t, status)
			err := r.ensureProvisioned(context.Background(), run)
			if err == nil {
				t.Fatal("invalid or unavailable image validation was ignored")
			}
			var permanent *configurationError
			if errors.As(err, &permanent) != (status == http.StatusUnprocessableEntity) {
				t.Fatalf("incorrect configuration versus operational failure classification: %v", err)
			}
			var pods corev1.PodList
			var maps corev1.ConfigMapList
			if err := c.List(context.Background(), &pods); err != nil {
				t.Fatal(err)
			}
			if err := c.List(context.Background(), &maps); err != nil {
				t.Fatal(err)
			}
			if len(pods.Items) != 0 || len(maps.Items) != 1 {
				t.Fatal("workloads or credential ConfigMaps were created before private preflight")
			}
		})
	}
}

func TestAIChildCannotChangeParentRegistry(t *testing.T) {
	r, c, parent := privateAIRunFixture(t, http.StatusOK)
	controller := &KrknScenarioRunReconciler{Client: c, Scheme: r.Scheme, Namespace: r.Namespace}
	for _, tc := range []struct {
		name, registry string
		private        bool
		uid            types.UID
		allowed        bool
	}{
		{"matching", "corporate", true, parent.UID, true},
		{"public downgrade", "", false, parent.UID, false},
		{"different private registry", "another", true, parent.UID, false},
		{"replacement parent", "corporate", true, types.UID("replacement"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			child := &krknv1alpha1.KrknScenarioRun{
				ObjectMeta: metav1.ObjectMeta{Namespace: parent.Namespace, OwnerReferences: []metav1.OwnerReference{{APIVersion: krknv1alpha1.GroupVersion.String(), Kind: "KrknAIRun", Name: parent.Name, UID: tc.uid}}},
				Spec:       krknv1alpha1.KrknScenarioRunSpec{Scenario: krknv1alpha1.ScenarioReference{Name: "node-cpu-hog", Private: &tc.private, RegistryName: tc.registry}},
			}
			if err := controller.validateAIChildRegistry(context.Background(), child); (err == nil) != tc.allowed {
				t.Fatalf("parent/child registry check = %v, allowed=%v", err, tc.allowed)
			}
		})
	}
}
