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
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
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
	if serviceURL == "http://unused" {
		service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/configs/validate" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"valid":true}`))
				return
			}
			http.NotFound(w, r)
		}))
		t.Cleanup(service.Close)
		serviceURL = service.URL
	}
	return &Handler{
		client:    fake.NewClientBuilder().WithScheme(scheme).WithObjects(target, secret, admin).Build(),
		namespace: "default", artifactClient: krknaiserver.New(serviceURL, "token"), krknAIEnabled: true,
	}
}

func adminKrknAIRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	return req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{UserID: "admin@example.com", Role: "admin"}))
}

func userKrknAIRequest(method, path, body, userID string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	return req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{
		UserID: userID,
		Role:   "user",
	}))
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
	if config.Annotations[files.WorkflowNameAnnotation] != "first-config" {
		t.Fatalf("saved config logical name must match its ConfigMap name: %+v", config.Annotations)
	}
	if fileInfo := buildFileInfo(&config); fileInfo.FileName != "first-config" {
		t.Fatalf("saved config file listing name = %q, want ConfigMap name", fileInfo.FileName)
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
	if run.Spec.TargetClusterAPIURL != "https://cluster.example" {
		t.Fatalf("run target API URL = %q, want stable target snapshot", run.Spec.TargetClusterAPIURL)
	}
	deleteResponse := httptest.NewRecorder()
	handler.KrknAIRouter(deleteResponse, adminKrknAIRequest(http.MethodDelete, KrknAIPath+"/runs/first-run", ""))
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("run deletion failed: %d %s", deleteResponse.Code, deleteResponse.Body.String())
	}
}

func TestKrknAIRunReadAPIsUseDurableTargetAPIURL(t *testing.T) {
	handler := newKrknAITestHandler(t, "http://unused")
	const (
		viewerID  = "viewer@example.com"
		groupName = "run-viewers"
		apiURL    = "https://cluster.example"
	)
	viewer := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-viewer-example-com",
			Namespace: "default",
			Labels:    map[string]string{groupauth.GroupLabelKey(groupName): "true"},
		},
		Spec: krknv1alpha1.KrknUserSpec{UserID: viewerID, Role: "user"},
	}
	group := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{Name: groupName, Namespace: "default"},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: groupName,
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				apiURL: {Actions: []string{string(groupauth.ActionView)}},
			},
		},
	}
	if err := handler.client.Create(context.Background(), viewer); err != nil {
		t.Fatal(err)
	}
	if err := handler.client.Create(context.Background(), group); err != nil {
		t.Fatal(err)
	}

	targetClusters := map[string][]string{"provider": {"cluster"}}
	storedURLRun := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "stored-url-run", Namespace: "default", UID: types.UID("stored-url-uid")},
		Spec: krknv1alpha1.KrknAIRunSpec{
			TargetRequestID:     "expired-target",
			TargetClusters:      targetClusters,
			TargetClusterAPIURL: apiURL,
		},
	}
	legacyRun := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "legacy-run", Namespace: "default", UID: types.UID("legacy-run-uid")},
		Spec: krknv1alpha1.KrknAIRunSpec{
			TargetRequestID: "expired-target",
			TargetClusters:  targetClusters,
		},
	}
	privateRun := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "private-run", Namespace: "default", UID: types.UID("private-run-uid")},
		Spec: krknv1alpha1.KrknAIRunSpec{
			TargetRequestID:     "expired-target",
			TargetClusters:      targetClusters,
			TargetClusterAPIURL: "https://private.example",
		},
	}
	child := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "legacy-run-child",
			Namespace: "default",
			Labels:    map[string]string{"krkn.dev/ai-run": legacyRun.Name},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: krknv1alpha1.GroupVersion.String(),
				Kind:       "KrknAIRun",
				Name:       legacyRun.Name,
				UID:        legacyRun.UID,
			}},
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: "expired-target",
			TargetClusters:  targetClusters,
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{ClusterJobs: []krknv1alpha1.ClusterJobStatus{{
			ClusterName:   "cluster",
			ClusterAPIURL: apiURL,
			JobID:         "legacy-job",
		}}},
	}
	for _, run := range []*krknv1alpha1.KrknAIRun{storedURLRun, legacyRun, privateRun} {
		if err := handler.client.Create(context.Background(), run); err != nil {
			t.Fatal(err)
		}
	}
	if err := handler.client.Create(context.Background(), child); err != nil {
		t.Fatal(err)
	}

	list := httptest.NewRecorder()
	handler.KrknAIRouter(list, userKrknAIRequest(http.MethodGet, KrknAIPath+"/runs", "", viewerID))
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d: %s", list.Code, list.Body.String())
	}
	var visible []krknv1alpha1.KrknAIRun
	if err := json.Unmarshal(list.Body.Bytes(), &visible); err != nil {
		t.Fatal(err)
	}
	visibleNames := map[string]bool{}
	for _, run := range visible {
		visibleNames[run.Name] = true
	}
	if len(visibleNames) != 2 || !visibleNames[storedURLRun.Name] || !visibleNames[legacyRun.Name] {
		t.Fatalf("authorized runs with no target request = %+v", visibleNames)
	}

	for _, name := range []string{storedURLRun.Name, legacyRun.Name} {
		response := httptest.NewRecorder()
		handler.KrknAIRouter(response, userKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/"+name, "", viewerID))
		if response.Code != http.StatusOK {
			t.Fatalf("get %s status = %d: %s", name, response.Code, response.Body.String())
		}
	}
	privateResponse := httptest.NewRecorder()
	handler.KrknAIRouter(privateResponse, userKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/private-run", "", viewerID))
	if privateResponse.Code != http.StatusForbidden {
		t.Fatalf("unauthorized run status = %d, want %d: %s", privateResponse.Code, http.StatusForbidden, privateResponse.Body.String())
	}

	summary := httptest.NewRecorder()
	handler.KrknAIRouter(summary, userKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/legacy-run/results/summary", "", viewerID))
	var summaryBody KrknAIRunSummaryResponse
	if summary.Code != http.StatusOK || json.Unmarshal(summary.Body.Bytes(), &summaryBody) != nil ||
		summaryBody.Name != legacyRun.Name || summaryBody.ArtifactStatus != "not_available" {
		t.Fatalf("legacy summary was not returned through child target metadata: %d %s", summary.Code, summary.Body.String())
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
func TestKrknAIRunTypedResultsAuthorizeAndProxyByUID(t *testing.T) {
	var serviceCalls int
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serviceCalls++
		switch r.URL.Path {
		case "/v1/runs/uid-typed/summary":
			_, _ = w.Write([]byte(`{"artifactStatus":"in_progress","completedGenerations":1,"bestFitness":0.75,"fitnessProgression":[]}`))
		case "/v1/runs/uid-typed/scenarios":
			if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("search") != "cpu" || r.URL.Query().Get("ignored") != "" {
				t.Errorf("unexpected forwarded query: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"scenarios":[],"pagination":{"page":2,"limit":10,"total":0,"totalPages":0}}`))
		case "/v1/runs/uid-typed/scenarios/1/scenario id":
			if !strings.HasSuffix(r.URL.EscapedPath(), "/scenario%20id") {
				t.Errorf("scenario ID was not escaped in service route: %s", r.URL.EscapedPath())
			}
			_, _ = w.Write([]byte(`{"generation":1,"scenarioId":"scenario id","scenarioType":"pod-delete","parameters":[{"name":"namespace","value":"shop"}],"command":"krkn --scenario pod","origin":"initial","parentIds":[],"durationSeconds":1.5,"returnCode":0,"fitnessResult":{"fitnessScore":0.75,"scores":[{"id":1,"fitnessScore":0.5,"weightedScore":0.25,"normalizedScore":0.75}],"healthCheckFailureScore":0.1,"healthCheckResponseTimeScore":0.2,"krknFailureScore":0.3},"healthChecks":[{"application":"shop","timestamp":"2025-01-01T00:00:02+00:00","elapsedSeconds":2,"responseTimeSeconds":0.12,"statusCode":200,"success":true}],"logPath":"logs/scenario.log","fitnessState":"final"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer service.Close()
	handler := newKrknAITestHandler(t, service.URL)
	run := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "typed-run", Namespace: "default", UID: "uid-typed"},
		Spec:       krknv1alpha1.KrknAIRunSpec{TargetRequestID: "target", TargetClusters: map[string][]string{"provider": {"cluster"}}},
		Status:     krknv1alpha1.KrknAIRunStatus{Phase: "Running", OrchestratorPodName: "orchestrator-1"},
	}
	if err := handler.client.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}

	deniedRequest := httptest.NewRequest(http.MethodGet, KrknAIPath+"/runs/typed-run/results/summary", nil)
	deniedRequest = deniedRequest.WithContext(context.WithValue(deniedRequest.Context(), auth.UserClaimsKey, &auth.Claims{UserID: "user@example.com", Role: "user"}))
	denied := httptest.NewRecorder()
	handler.KrknAIRouter(denied, deniedRequest)
	if denied.Code != http.StatusForbidden || serviceCalls != 0 {
		t.Fatalf("unauthorized result read status/calls = %d/%d, want 403/0", denied.Code, serviceCalls)
	}

	summary := httptest.NewRecorder()
	handler.KrknAIRouter(summary, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/typed-run/results/summary", ""))
	if summary.Code != http.StatusOK {
		t.Fatalf("summary status = %d: %s", summary.Code, summary.Body.String())
	}
	var result KrknAIRunSummaryResponse
	if err := json.Unmarshal(summary.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Name != "typed-run" || result.Phase != "Running" || result.Cluster != "cluster" ||
		result.OrchestratorPodName != "orchestrator-1" || result.ArtifactStatus != "in_progress" ||
		result.BestFitness == nil || *result.BestFitness != 0.75 {
		t.Fatalf("summary did not merge run metadata and artifact data: %+v", result)
	}

	index := httptest.NewRecorder()
	handler.KrknAIRouter(index, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/typed-run/results/scenarios?page=2&search=cpu&ignored=x", ""))
	if index.Code != http.StatusOK || serviceCalls != 2 {
		t.Fatalf("index status/calls = %d/%d: %s", index.Code, serviceCalls, index.Body.String())
	}
	detail := httptest.NewRecorder()
	handler.KrknAIRouter(detail, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/typed-run/results/scenarios/1/scenario%20id", ""))
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), `"scenarioId":"scenario id"`) || serviceCalls != 3 {
		t.Fatalf("detail proxy status/calls = %d/%d: %s", detail.Code, serviceCalls, detail.Body.String())
	}

	var detailResult KrknAIScenarioDetailResponse
	if err := json.Unmarshal(detail.Body.Bytes(), &detailResult); err != nil {
		t.Fatal(err)
	}
	if detailResult.FitnessResult.Scores[0].ID != 1 ||
		detailResult.HealthChecks[0].ElapsedSeconds == nil ||
		*detailResult.HealthChecks[0].ElapsedSeconds != 2 {
		t.Fatalf("typed scenario detail did not match committed service schema: %+v", detailResult)
	}
}

func TestKrknAIRunMissingManifestReturnsEmptyTypedResults(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer service.Close()
	handler := newKrknAITestHandler(t, service.URL)
	run := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "pending-run", Namespace: "default", UID: "uid-pending"},
		Spec:       krknv1alpha1.KrknAIRunSpec{TargetRequestID: "target", TargetClusters: map[string][]string{"provider": {"cluster"}}},
	}
	if err := handler.client.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	summary := httptest.NewRecorder()
	handler.KrknAIRouter(summary, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/pending-run/results/summary", ""))
	var summaryPayload KrknAIRunSummaryResponse
	if summary.Code != http.StatusOK || json.Unmarshal(summary.Body.Bytes(), &summaryPayload) != nil ||
		summaryPayload.ArtifactStatus != "not_available" || summaryPayload.BestFitness != nil {
		t.Fatalf("missing summary manifest response = %d %s", summary.Code, summary.Body.String())
	}
	index := httptest.NewRecorder()
	handler.KrknAIRouter(index, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/pending-run/results/scenarios", ""))
	var indexPayload KrknAIScenarioIndexResponse
	if index.Code != http.StatusOK || json.Unmarshal(index.Body.Bytes(), &indexPayload) != nil ||
		len(indexPayload.Scenarios) != 0 || indexPayload.Pagination.Total != 0 {
		t.Fatalf("missing scenario manifest response = %d %s", index.Code, index.Body.String())
	}
}

func TestKrknAITypedResultsPreserveRetryableAndCorruptStatuses(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/summary"):
			w.WriteHeader(http.StatusServiceUnavailable)
		case strings.HasSuffix(r.URL.Path, "/scenarios"):
			w.WriteHeader(http.StatusBadGateway)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer service.Close()
	handler := newKrknAITestHandler(t, service.URL)
	run := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "status-run", Namespace: "default", UID: "uid-status"},
		Spec:       krknv1alpha1.KrknAIRunSpec{TargetRequestID: "target", TargetClusters: map[string][]string{"provider": {"cluster"}}},
	}
	if err := handler.client.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		path string
		want int
	}{
		{path: "/runs/status-run/results/summary", want: http.StatusServiceUnavailable},
		{path: "/runs/status-run/results/scenarios", want: http.StatusBadGateway},
		{path: "/runs/status-run/results/scenarios/1/missing", want: http.StatusNotFound},
	} {
		response := httptest.NewRecorder()
		handler.KrknAIRouter(response, adminKrknAIRequest(http.MethodGet, KrknAIPath+test.path, ""))
		if response.Code != test.want {
			t.Errorf("%s status = %d, want %d: %s", test.path, response.Code, test.want, response.Body.String())
		}
	}
}

func TestKrknAIRunScenarioIndexJoinsChildRunsByGenerationAndOwnerUID(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"scenarios":[{"generation":0,"scenarioId":"baseline","scenarioType":"baseline","outcome":"succeeded","durationSeconds":60,"fitnessScore":12,"fitnessState":"final"},{"generation":1,"scenarioId":"9","scenarioType":"pod-delete","fitnessScore":0.5,"fitnessState":"final"}],"pagination":{"page":1,"limit":100,"total":2,"totalPages":1}}`))
	}))
	defer service.Close()
	handler := newKrknAITestHandler(t, service.URL)
	runUID := types.UID("uid-children")
	run := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "children-run", Namespace: "default", UID: runUID},
		Spec:       krknv1alpha1.KrknAIRunSpec{TargetRequestID: "target", TargetClusters: map[string][]string{"provider": {"cluster"}}},
	}
	owner := []metav1.OwnerReference{{APIVersion: krknv1alpha1.GroupVersion.String(), Kind: "KrknAIRun", Name: run.Name, UID: runUID}}
	children := []*krknv1alpha1.KrknScenarioRun{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "child-gen-1", Namespace: "default", OwnerReferences: owner, Labels: map[string]string{
				"krkn.dev/ai-run": "children-run", "krkn.dev/generation-id": "1", "krkn.dev/scenario-id": "9", "krkn.dev/scenario-name": "pod-delete",
			}},
			Status: krknv1alpha1.KrknScenarioRunStatus{Phase: "Running", ClusterJobs: []krknv1alpha1.ClusterJobStatus{{JobID: "job-1", PodName: "pod-1"}}},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "child-gen-2", Namespace: "default", OwnerReferences: owner, Labels: map[string]string{
				"krkn.dev/ai-run": "children-run", "krkn.dev/generation-id": "2", "krkn.dev/scenario-id": "9", "krkn.dev/scenario-name": "pod-delete",
			}},
			Status: krknv1alpha1.KrknScenarioRunStatus{Phase: "Pending"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "child-baseline", Namespace: "default", OwnerReferences: owner, Labels: map[string]string{
				"krkn.dev/ai-run": "children-run", "krkn.dev/generation-id": "0", "krkn.dev/scenario-id": "baseline", "krkn.dev/scenario-name": "baseline",
			}},
			Status: krknv1alpha1.KrknScenarioRunStatus{Phase: "Running", ClusterJobs: []krknv1alpha1.ClusterJobStatus{{JobID: "baseline-job", PodName: "baseline-pod"}}},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "stale-child", Namespace: "default", OwnerReferences: []metav1.OwnerReference{{UID: "old-uid"}}, Labels: map[string]string{
				"krkn.dev/ai-run": "children-run", "krkn.dev/generation-id": "3", "krkn.dev/scenario-id": "9",
			}},
		},
	}
	if err := handler.client.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	for _, child := range children {
		if err := handler.client.Create(context.Background(), child); err != nil {
			t.Fatal(err)
		}
	}

	response := httptest.NewRecorder()
	handler.KrknAIRouter(response, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/children-run/results/scenarios", ""))
	if response.Code != http.StatusOK {
		t.Fatalf("index status = %d: %s", response.Code, response.Body.String())
	}
	var index KrknAIScenarioIndexResponse
	if err := json.Unmarshal(response.Body.Bytes(), &index); err != nil {
		t.Fatal(err)
	}
	if len(index.Scenarios) != 3 {
		t.Fatalf("baseline result or generation-specific child rows missing: %+v", index.Scenarios)
	}
	if index.Pagination.Total != 3 {
		t.Fatalf("total = %d, want 3", index.Pagination.Total)
	}
	baseline, first, second := index.Scenarios[0], index.Scenarios[1], index.Scenarios[2]
	if baseline.Generation != 0 || baseline.ScenarioID != "baseline" || baseline.ScenarioType != "baseline" ||
		baseline.ChildRunName != "child-baseline" || baseline.Phase != "Running" ||
		baseline.JobID != "baseline-job" || baseline.PodName != "baseline-pod" {
		t.Fatalf("baseline artifact row child metadata was not joined: %+v", baseline)
	}
	if first.Generation != 1 || first.ScenarioID != "9" || first.ChildRunName != "child-gen-1" ||
		first.Phase != "Running" || first.JobID != "job-1" || first.PodName != "pod-1" {
		t.Fatalf("artifact row child metadata was not joined: %+v", first)
	}
	if second.Generation != 2 || second.ScenarioID != "9" || second.ChildRunName != "child-gen-2" || second.Phase != "Pending" {
		t.Fatalf("child-only row did not retain generation identity: %+v", second)
	}
}

func TestKrknAIConfigValidationPassesThrough422AndBlocksPersistence(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/configs/validate" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":[{"path":"genetic.generations","message":"Invalid value"}],"credential":"must-not-pass"}`))
	}))
	defer service.Close()
	handler := newKrknAITestHandler(t, service.URL)
	body := `{"configYaml":"generations: 0\n"}`
	response := httptest.NewRecorder()
	handler.KrknAIRouter(response, adminKrknAIRequest(http.MethodPost, KrknAIPath+"/configs/validate", body))
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), `"message":"Invalid value"`) ||
		strings.Contains(response.Body.String(), "must-not-pass") {
		t.Fatalf("validation response was not safely passed through: %d %s", response.Code, response.Body.String())
	}
	create := httptest.NewRecorder()
	handler.KrknAIRouter(create, adminKrknAIRequest(
		http.MethodPost, KrknAIPath+"/configs",
		`{"name":"invalid-config","configYaml":"generations: 0\n","targetRequestId":"target","targetClusters":{"provider":["cluster"]}}`,
	))
	if create.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid config creation status = %d, want 422: %s", create.Code, create.Body.String())
	}
	var configs corev1.ConfigMapList
	if err := handler.client.List(context.Background(), &configs); err != nil {
		t.Fatal(err)
	}
	if len(configs.Items) != 0 {
		t.Fatalf("invalid config was persisted: %+v", configs.Items)
	}
}

func TestKrknAIConfigOwnerCanListAndReadSavedConfig(t *testing.T) {
	handler := newKrknAITestHandler(t, "http://unused")
	ownedConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ai-robot-shop-example-c241ebb4",
			Namespace: "default",
			Labels: files.BuildFileLabels(
				"config-file-id", nil, false, files.FilePurposeKrknAIConfig, "ai-robot-shop-example-c241ebb4",
			),
			Annotations: files.BuildFileAnnotations("", "owner@example.com", "ai-robot-shop-example-c241ebb4"),
		},
		Data: map[string]string{files.KrknAIConfigFileName: "generations: 1\n"},
	}
	otherConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "another-private-config",
			Namespace: "default",
			Labels: files.BuildFileLabels(
				"other-config-file-id", nil, false, files.FilePurposeKrknAIConfig, "another-private-config",
			),
			Annotations: files.BuildFileAnnotations("", "other@example.com", "another-private-config"),
		},
		Data: map[string]string{files.KrknAIConfigFileName: "generations: 2\n"},
	}
	otherUser := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{Name: "krknuser-viewer-example-com", Namespace: "default"},
	}
	if err := handler.client.Create(context.Background(), ownedConfig); err != nil {
		t.Fatal(err)
	}
	if err := handler.client.Create(context.Background(), otherConfig); err != nil {
		t.Fatal(err)
	}
	if err := handler.client.Create(context.Background(), otherUser); err != nil {
		t.Fatal(err)
	}

	userRequest := func(method, path, userID string) *http.Request {
		request := httptest.NewRequest(method, path, nil)
		return request.WithContext(context.WithValue(request.Context(), auth.UserClaimsKey, &auth.Claims{
			UserID: userID, Role: "user",
		}))
	}
	list := httptest.NewRecorder()
	handler.ListAvailableFiles(list, userRequest(http.MethodGet, FilesPath+"/available?filePurpose=krkn-ai-config", "owner@example.com"))
	var available files.AvailableFilesResponse
	if list.Code != http.StatusOK || json.Unmarshal(list.Body.Bytes(), &available) != nil {
		t.Fatalf("owner config listing status/body = %d %s", list.Code, list.Body.String())
	}
	if len(available.Files) != 1 || available.Files[0].FileID != "config-file-id" {
		t.Fatalf("owner should see only their own saved config, got %+v", available.Files)
	}

	read := httptest.NewRecorder()
	handler.GetFile(read, userRequest(http.MethodGet, FilesPath+"/config-file-id", "owner@example.com"))
	var response files.FileResponse
	if read.Code != http.StatusOK || json.Unmarshal(read.Body.Bytes(), &response) != nil {
		t.Fatalf("owner config read status/body = %d %s", read.Code, read.Body.String())
	}
	if response.FilePurpose != files.FilePurposeKrknAIConfig || response.Content != "generations: 1\n" {
		t.Fatalf("owner config read returned wrong file: %+v", response)
	}

	deniedList := httptest.NewRecorder()
	handler.ListAvailableFiles(deniedList, userRequest(http.MethodGet, FilesPath+"/available?filePurpose=krkn-ai-config", "viewer@example.com"))
	available = files.AvailableFilesResponse{}
	if deniedList.Code != http.StatusOK || json.Unmarshal(deniedList.Body.Bytes(), &available) != nil || len(available.Files) != 0 {
		t.Fatalf("unrelated user config listing status/body = %d %s", deniedList.Code, deniedList.Body.String())
	}
	deniedRead := httptest.NewRecorder()
	handler.GetFile(deniedRead, userRequest(http.MethodGet, FilesPath+"/config-file-id", "viewer@example.com"))
	if deniedRead.Code != http.StatusForbidden {
		t.Fatalf("unrelated user config read status = %d, want %d: %s", deniedRead.Code, http.StatusForbidden, deniedRead.Body.String())
	}
}
func TestKrknAIDisabledRejectsCreateEndpoints(t *testing.T) {
	handler := newKrknAITestHandler(t, "http://unused")
	handler.krknAIEnabled = false
	for _, endpoint := range []string{
		"/discoveries", "/configs/validate", "/configs", "/runs",
	} {
		response := httptest.NewRecorder()
		handler.KrknAIRouter(response, adminKrknAIRequest(http.MethodPost, KrknAIPath+endpoint, `{}`))
		if response.Code != http.StatusServiceUnavailable {
			t.Errorf("%s status = %d, want %d", endpoint, response.Code, http.StatusServiceUnavailable)
		}
	}
}
