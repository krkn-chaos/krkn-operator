package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	"github.com/krkn-chaos/krkn-operator/pkg/krknaiserver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newKrknAITestHandler(t *testing.T, serviceURL string) *Handler {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	kubeconfig := base64.StdEncoding.EncodeToString([]byte("apiVersion: v1\n"))
	managed := `{"provider":{"cluster":{"kubeconfig":"` + kubeconfig + `"}}}`
	target := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: "default"},
		Status: krknv1alpha1.KrknTargetRequestStatus{Status: "Completed", TargetData: map[string][]krknv1alpha1.ClusterTarget{
			"provider": {{ClusterName: "cluster", ClusterAPIURL: "https://cluster.example"}},
		}},
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: "default"}, Data: map[string][]byte{"managed-clusters": []byte(managed)}}
	admin := &krknv1alpha1.KrknUser{ObjectMeta: metav1.ObjectMeta{Name: "krknuser-admin-example-com", Namespace: "default"}}
	return &Handler{client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(target, secret, admin).Build(), namespace: "default", artifactClient: krknaiserver.New(serviceURL, "token")}
}

func adminKrknAIRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	return req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{UserID: "admin@example.com", Role: "admin"}))
}

func TestKrknAIConfigBindingAndRunCRUD(t *testing.T) {
	handler := newKrknAITestHandler(t, "http://unused")
	configBody := `{"name":"first-config","configYaml":"generations: 1\n","targetRequestId":"target","targetClusters":{"provider":["cluster"]}}`
	configResponse := httptest.NewRecorder()
	handler.KrknAIRouter(configResponse, adminKrknAIRequest(http.MethodPost, KrknAIPath+"/configs", configBody))
	if configResponse.Code != http.StatusCreated {
		t.Fatalf("config creation failed: %d %s", configResponse.Code, configResponse.Body.String())
	}
	var created map[string]string
	if err := json.Unmarshal(configResponse.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	var config corev1.ConfigMap
	if err := handler.client.Get(context.Background(), types.NamespacedName{Name: "first-config", Namespace: "default"}, &config); err != nil {
		t.Fatalf("named config ConfigMap was not created: %v", err)
	}
	if len(config.Data) != 1 || config.Data[files.KrknAIConfigFileName] != "generations: 1\n" {
		t.Fatalf("config must contain only the fixed Krkn-AI key: %+v", config.Data)
	}
	if config.Labels[files.FileIDLabel] != created["configId"] {
		t.Fatalf("config ID must identify the named ConfigMap: %+v", config.Labels)
	}
	var configMaps corev1.ConfigMapList
	if err := handler.client.List(context.Background(), &configMaps); err != nil {
		t.Fatal(err)
	}
	if len(configMaps.Items) != 1 {
		t.Fatalf("config creation must create exactly one ConfigMap, got %d", len(configMaps.Items))
	}
	for _, configMap := range configMaps.Items {
		if configMap.Labels[files.AppComponentLabel] == files.ComponentFileReservation {
			t.Fatalf("config creation must not create a reservation ConfigMap: %s", configMap.Name)
		}
	}
	runBody := `{"name":"first-run","configId":"` + created["configId"] + `","targetRequestId":"target","targetClusters":{"provider":["cluster"]}}`
	runResponse := httptest.NewRecorder()
	handler.KrknAIRouter(runResponse, adminKrknAIRequest(http.MethodPost, KrknAIPath+"/runs", runBody))
	if runResponse.Code != http.StatusCreated {
		t.Fatalf("run creation failed: %d %s", runResponse.Code, runResponse.Body.String())
	}
	var run krknv1alpha1.KrknAIRun
	if err := handler.client.Get(context.Background(), types.NamespacedName{Name: "first-run", Namespace: "default"}, &run); err != nil {
		t.Fatal(err)
	}
	if run.Spec.ConfigMapName != "first-config" || run.Spec.ConfigMapKey != files.KrknAIConfigFileName {
		t.Fatalf("run does not reference its source config directly: %+v", run.Spec)
	}

	listResponse := httptest.NewRecorder()
	handler.KrknAIRouter(listResponse, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs", ""))
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), "first-run") {
		t.Fatalf("run list did not include created run: %d %s", listResponse.Code, listResponse.Body.String())
	}
	deleteResponse := httptest.NewRecorder()
	handler.KrknAIRouter(deleteResponse, adminKrknAIRequest(http.MethodDelete, KrknAIPath+"/runs/first-run", ""))
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("run deletion failed: %d %s", deleteResponse.Code, deleteResponse.Body.String())
	}
}

func TestKrknAICreateRequestsRejectInvalidKubernetesValues(t *testing.T) {
	handler := newKrknAITestHandler(t, "http://unused")

	for _, test := range []struct {
		name string
		body string
	}{
		{
			name: "invalid config name",
			body: `{"name":"Invalid_Name","configYaml":"generations: 1\n","targetRequestId":"target","targetClusters":{"provider":["cluster"]}}`,
		},
		{
			name: "non-object config",
			body: `{"name":"list-config","configYaml":"- generations\n- 1\n","targetRequestId":"target","targetClusters":{"provider":["cluster"]}}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.KrknAIRouter(response, adminKrknAIRequest(http.MethodPost, KrknAIPath+"/configs", test.body))
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
			}
		})
	}

	configResponse := httptest.NewRecorder()
	handler.KrknAIRouter(configResponse, adminKrknAIRequest(
		http.MethodPost,
		KrknAIPath+"/configs",
		`{"name":"deadline-config","configYaml":"generations: 1\n","targetRequestId":"target","targetClusters":{"provider":["cluster"]}}`,
	))
	if configResponse.Code != http.StatusCreated {
		t.Fatalf("config creation failed: %d %s", configResponse.Code, configResponse.Body.String())
	}
	var created map[string]string
	if err := json.Unmarshal(configResponse.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	overlongRunBody, err := json.Marshal(KrknAIRunRequest{
		Name:            strings.Repeat("a", 64),
		ConfigID:        created["configId"],
		TargetRequestID: "target",
		TargetClusters:  map[string][]string{"provider": {"cluster"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	overlongResponse := httptest.NewRecorder()
	handler.KrknAIRouter(overlongResponse, adminKrknAIRequest(http.MethodPost, KrknAIPath+"/runs", string(overlongRunBody)))
	if overlongResponse.Code != http.StatusBadRequest {
		t.Fatalf("overlong run name status = %d, want %d: %s", overlongResponse.Code, http.StatusBadRequest, overlongResponse.Body.String())
	}

	response := httptest.NewRecorder()
	handler.KrknAIRouter(response, adminKrknAIRequest(
		http.MethodPost,
		KrknAIPath+"/runs",
		`{"name":"deadline-run","configId":"`+created["configId"]+`","targetRequestId":"target","targetClusters":{"provider":["cluster"]},"activeDeadlineSeconds":0}`,
	))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("explicit zero deadline status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}

func TestKrknAIConfigRejectsMutationWhileRunIsActive(t *testing.T) {
	handler := newKrknAITestHandler(t, "http://unused")
	configBody := `{"name":"locked-config","configYaml":"generations: 1\n","targetRequestId":"target","targetClusters":{"provider":["cluster"]}}`
	configResponse := httptest.NewRecorder()
	handler.KrknAIRouter(configResponse, adminKrknAIRequest(http.MethodPost, KrknAIPath+"/configs", configBody))
	if configResponse.Code != http.StatusCreated {
		t.Fatalf("config creation failed: %d %s", configResponse.Code, configResponse.Body.String())
	}
	var created map[string]string
	if err := json.Unmarshal(configResponse.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	runBody := `{"name":"locked-run","configId":"` + created["configId"] + `","targetRequestId":"target","targetClusters":{"provider":["cluster"]}}`
	runResponse := httptest.NewRecorder()
	handler.KrknAIRouter(runResponse, adminKrknAIRequest(http.MethodPost, KrknAIPath+"/runs", runBody))
	if runResponse.Code != http.StatusCreated {
		t.Fatalf("run creation failed: %d %s", runResponse.Code, runResponse.Body.String())
	}

	updateResponse := httptest.NewRecorder()
	handler.UpdateFile(updateResponse, adminKrknAIRequest(
		http.MethodPut, FilesPath+"/"+created["configId"],
		`{"fileName":"locked-config","content":"generations: 2\n","availableToAll":true}`,
	))
	if updateResponse.Code != http.StatusConflict {
		t.Fatalf("active config update returned %d: %s", updateResponse.Code, updateResponse.Body.String())
	}

	deleteResponse := httptest.NewRecorder()
	handler.DeleteFile(deleteResponse, adminKrknAIRequest(http.MethodDelete, FilesPath+"/"+created["configId"], ""))
	if deleteResponse.Code != http.StatusConflict {
		t.Fatalf("active config delete returned %d: %s", deleteResponse.Code, deleteResponse.Body.String())
	}

}

func TestKrknAIDiscoveryAndArtifactProxyRequireTargetAccess(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Fatal("service token was not forwarded")
		}
		if r.URL.Path == "/v1/discoveries" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"configYaml":"kubeconfig_file_path: /input/kubeconfig\n","warnings":[]}`))
			return
		}
		if r.URL.Path == "/v1/runs/run-uid/results" {
			_, _ = w.Write([]byte(`{"files":[]}`))
			return
		}
		if r.URL.Path == "/v1/runs/run-uid/files/report.html" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<script>alert("stored artifact")</script>`))
			return
		}
		http.NotFound(w, r)
	}))
	defer service.Close()
	handler := newKrknAITestHandler(t, service.URL)
	discoveryResponse := httptest.NewRecorder()
	discoveryBody := `{"targetRequestId":"target","targetClusters":{"provider":["cluster"]}}`
	handler.KrknAIRouter(discoveryResponse, adminKrknAIRequest(http.MethodPost, KrknAIPath+"/discoveries", discoveryBody))
	if discoveryResponse.Code != http.StatusOK || !strings.Contains(discoveryResponse.Body.String(), "/input/kubeconfig") {
		t.Fatalf("discovery proxy failed: %d %s", discoveryResponse.Code, discoveryResponse.Body.String())
	}
	run := &krknv1alpha1.KrknAIRun{ObjectMeta: metav1.ObjectMeta{Name: "finished", Namespace: "default", UID: "run-uid"}, Spec: krknv1alpha1.KrknAIRunSpec{TargetRequestID: "target", TargetClusters: map[string][]string{"provider": {"cluster"}}}}
	if err := handler.client.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	artifactResponse := httptest.NewRecorder()
	handler.KrknAIRouter(artifactResponse, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/finished/results", ""))
	if artifactResponse.Code != http.StatusOK || artifactResponse.Body.String() != `{"files":[]}` {
		t.Fatalf("artifact proxy failed: %d %s", artifactResponse.Code, artifactResponse.Body.String())
	}
	fileResponse := httptest.NewRecorder()
	handler.KrknAIRouter(fileResponse, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/finished/files/report.html", ""))
	if fileResponse.Code != http.StatusOK || fileResponse.Body.String() != `<script>alert("stored artifact")</script>` {
		t.Fatalf("artifact file proxy failed: %d %s", fileResponse.Code, fileResponse.Body.String())
	}
	if fileResponse.Header().Get("Content-Type") != "application/octet-stream" ||
		fileResponse.Header().Get("Content-Disposition") != "attachment" ||
		fileResponse.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("artifact response headers must force a safe download: %+v", fileResponse.Header())
	}
	unauthorized := httptest.NewRecorder()
	handler.KrknAIRouter(unauthorized, httptest.NewRequest(http.MethodPost, KrknAIPath+"/discoveries", strings.NewReader(discoveryBody)))
	if unauthorized.Code != http.StatusForbidden {
		t.Fatalf("discovery without target authorization returned %d", unauthorized.Code)
	}
}
