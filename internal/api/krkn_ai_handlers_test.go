package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
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
func TestKrknAIStatusDoesNotRequireTheOptionalCRD(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		handler := &Handler{krknAIEnabled: enabled}
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, KrknAIPath+"/status", nil)

		handler.KrknAIRouter(response, request)

		var status KrknAIStatusResponse
		if response.Code != http.StatusOK {
			t.Fatalf("status endpoint returned %d: %s", response.Code, response.Body.String())
		}
		if err := json.Unmarshal(response.Body.Bytes(), &status); err != nil {
			t.Fatalf("decode status response: %v", err)
		}
		if status.Enabled != enabled {
			t.Errorf("enabled = %t, want %t", status.Enabled, enabled)
		}
	}
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
	deleteResponse := httptest.NewRecorder()
	handler.KrknAIRouter(deleteResponse, adminKrknAIRequest(http.MethodDelete, KrknAIPath+"/runs/first-run", ""))
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("run deletion failed: %d %s", deleteResponse.Code, deleteResponse.Body.String())
	}
}

func TestKrknAIRunReadAPIsAuthorizeFromPersistedCRDs(t *testing.T) {
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
				apiURL: {Actions: []string{string(groupauth.ActionView), string(groupauth.ActionCancel)}},
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
	groupRun := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "persisted-run", Namespace: "default", UID: types.UID("persisted-run-uid")},
		Spec: krknv1alpha1.KrknAIRunSpec{
			TargetRequestID: "expired-target",
			TargetClusters:  targetClusters,
		},
	}
	privateRun := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "private-run", Namespace: "default", UID: types.UID("private-run-uid")},
		Spec: krknv1alpha1.KrknAIRunSpec{
			TargetRequestID: "expired-target",
			TargetClusters:  targetClusters,
		},
	}
	childFor := func(parent *krknv1alpha1.KrknAIRun, name, targetAPIURL string) *krknv1alpha1.KrknScenarioRun {
		return &krknv1alpha1.KrknScenarioRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
				Labels:    map[string]string{"krkn.dev/ai-run": parent.Name},
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: krknv1alpha1.GroupVersion.String(),
					Kind:       "KrknAIRun",
					Name:       parent.Name,
					UID:        parent.UID,
				}},
			},
			Spec: krknv1alpha1.KrknScenarioRunSpec{
				TargetRequestID: parent.Spec.TargetRequestID,
				TargetClusters:  targetClusters,
			},
			Status: krknv1alpha1.KrknScenarioRunStatus{ClusterJobs: []krknv1alpha1.ClusterJobStatus{{
				ClusterName:   "cluster",
				ClusterAPIURL: targetAPIURL,
				JobID:         name + "-job",
			}}},
		}
	}
	for _, run := range []*krknv1alpha1.KrknAIRun{groupRun, privateRun} {
		if err := handler.client.Create(context.Background(), run); err != nil {
			t.Fatal(err)
		}
	}
	for _, child := range []*krknv1alpha1.KrknScenarioRun{
		childFor(groupRun, "persisted-run-child", apiURL),
		childFor(privateRun, "private-run-child", "https://private.example"),
	} {
		if err := handler.client.Create(context.Background(), child); err != nil {
			t.Fatal(err)
		}
	}

	list := httptest.NewRecorder()
	handler.KrknAIRouter(list, userKrknAIRequest(http.MethodGet, KrknAIPath+"/runs", "", viewerID))
	var visible []krknv1alpha1.KrknAIRun
	if list.Code != http.StatusOK || json.Unmarshal(list.Body.Bytes(), &visible) != nil ||
		len(visible) != 1 || visible[0].Name != groupRun.Name {
		t.Fatalf("run list did not authorize from persisted child metadata: %d %s", list.Code, list.Body.String())
	}

	for _, run := range []*krknv1alpha1.KrknAIRun{groupRun} {
		response := httptest.NewRecorder()
		handler.KrknAIRouter(response, userKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/"+run.Name, "", viewerID))
		if response.Code != http.StatusOK {
			t.Fatalf("run get %s status = %d: %s", run.Name, response.Code, response.Body.String())
		}

		summary := httptest.NewRecorder()
		handler.KrknAIRouter(summary, userKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/"+run.Name+"/results/summary", "", viewerID))
		var summaryBody KrknAIRunSummaryResponse
		if summary.Code != http.StatusOK || json.Unmarshal(summary.Body.Bytes(), &summaryBody) != nil ||
			summaryBody.Name != run.Name || summaryBody.ArtifactStatus != "not_available" {
			t.Fatalf("run summary %s was not returned without a target request: %d %s", run.Name, summary.Code, summary.Body.String())
		}
	}
	privateResponse := httptest.NewRecorder()
	handler.KrknAIRouter(privateResponse, userKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/"+privateRun.Name, "", viewerID))
	if privateResponse.Code != http.StatusForbidden {
		t.Fatalf("unauthorized run status = %d, want %d: %s", privateResponse.Code, http.StatusForbidden, privateResponse.Body.String())
	}

	activeRun := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "active-run", Namespace: "default"},
		Spec: krknv1alpha1.KrknAIRunSpec{
			TargetRequestID: "target",
			TargetClusters:  targetClusters,
		},
	}
	if err := handler.client.Create(context.Background(), activeRun); err != nil {
		t.Fatal(err)
	}
	cancel := httptest.NewRecorder()
	handler.KrknAIRouter(cancel, userKrknAIRequest(http.MethodDelete, KrknAIPath+"/runs/"+activeRun.Name, "", viewerID))
	if cancel.Code != http.StatusNoContent {
		t.Fatalf("active run cancel status = %d: %s", cancel.Code, cancel.Body.String())
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
			_, _ = w.Write([]byte(`{"artifactStatus":"in_progress","completedGenerations":1,"currentGeneration":1,"bestFitness":75,"fitnessProgression":[]}`))
		case "/v1/runs/uid-typed/scenarios":
			_, _ = w.Write([]byte(`{"scenarios":[]}`))
		case "/v1/runs/uid-typed/scenarios/1/scenario id":
			if !strings.HasSuffix(r.URL.EscapedPath(), "/scenario%20id") {
				t.Errorf("scenario ID was not escaped in service route: %s", r.URL.EscapedPath())
			}
			_, _ = w.Write([]byte(`{"generation":1,"scenarioId":"scenario id","scenarioType":"pod-delete","parameters":[{"name":"namespace","value":"shop"}],"command":"krkn --scenario pod","origin":"initial","parentIds":[],"durationSeconds":1.5,"returnCode":0,"fitnessResult":{"fitnessScore":75,"scores":[{"id":1,"rawScore":42,"normalizedScore":0.75,"query":"sum(rate(http_requests_total[5m]))","queryType":"range"}],"healthCheckFailureScore":0.1,"healthCheckResponseTimeScore":0.2,"krknFailureScore":0.3},"healthChecks":[{"application":"shop","timestamp":"2025-01-01T00:00:02+00:00","elapsedSeconds":2,"responseTimeSeconds":0.12,"statusCode":200,"success":true}],"logPath":"logs/scenario.log","fitnessState":"final"}`))
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
		result.BestFitness == nil || *result.BestFitness != 75 {
		t.Fatalf("summary did not merge run metadata and artifact data: %+v", result)
	}

	index := httptest.NewRecorder()
	handler.KrknAIRouter(index, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/typed-run/results/scenarios", ""))
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
		_, _ = w.Write([]byte(`{"scenarios":[{"generation":0,"scenarioId":"baseline","scenarioType":"baseline","outcome":"succeeded","durationSeconds":60,"fitnessScore":12,"fitnessState":"final"},{"generation":1,"scenarioId":"9","scenarioType":"pod-delete","fitnessScore":0.5,"fitnessState":"final"}]}`))
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

func TestKrknAIRunScenarioIndexPaginatesCompleteUnionAndValidatesQueries(t *testing.T) {
	fixture := newKrknAIScenarioIndexFixture(t)
	for _, invalidQuery := range []string{
		"?page=0", "?page=abc", "?page=", "?limit=0", "?limit=501", "?limit=",
		"?generation=-1", "?generation=abc", "?generation=",
		"?sort=unsupported", "?sort=", "?direction=sideways", "?direction=",
	} {
		before := fixture.serviceCalls
		response := fixture.request(t, invalidQuery)
		if response.Code != http.StatusBadRequest || fixture.serviceCalls != before {
			t.Errorf("invalid query %q status/service calls = %d/%d; want 400/no downstream request: %s",
				invalidQuery, response.Code, fixture.serviceCalls-before, response.Body.String())
		}
	}
	defaults := fixture.readIndex(t, "")
	if defaults.Pagination.Page != 1 || defaults.Pagination.Limit != 100 || defaults.Pagination.Total != 5 {
		t.Fatalf("default pagination = %+v, want page=1 limit=100 total=5", defaults.Pagination)
	}
	bounded := fixture.readIndex(t, "?limit=500&ignored=x")
	if bounded.Pagination.Limit != 500 || bounded.Pagination.Total != 5 {
		t.Fatalf("maximum page size or unknown-key handling failed: %+v", bounded.Pagination)
	}
	for page, wantID := range []string{"baseline", "2", "3", "7", "10"} {
		index := fixture.readIndex(t, "?page="+strconv.Itoa(page+1)+"&limit=1&sort=generation&direction=asc")
		if index.Pagination.Page != page+1 || index.Pagination.Limit != 1 ||
			index.Pagination.Total != 5 || index.Pagination.TotalPages != 5 ||
			len(index.Scenarios) != 1 || index.Scenarios[0].ScenarioID != wantID {
			t.Fatalf("global page %d = %+v, want only %q and total 5", page+1, index, wantID)
		}
	}
	generationFiltered := fixture.readIndex(t, "?generation=1")
	if generationFiltered.Pagination.Total != 4 || len(generationFiltered.Scenarios) != 4 ||
		generationFiltered.Scenarios[0].ScenarioID != "2" || generationFiltered.Scenarios[0].Phase != "Running" {
		t.Fatalf("generation filter returned the wrong rows or lost matching child metadata: %+v", generationFiltered)
	}
	for _, equivalentGeneration := range []string{"?generation=01", "?generation=%2B1"} {
		index := fixture.readIndex(t, equivalentGeneration)
		if index.Pagination.Total != 4 {
			t.Errorf("equivalent generation query %q matched %d rows, want 4", equivalentGeneration, index.Pagination.Total)
		}
	}
}

func TestKrknAIRunScenarioIndexFiltersAndSortsCompleteUnion(t *testing.T) {
	fixture := newKrknAIScenarioIndexFixture(t)
	filteredPage := fixture.readIndex(t, "?scenarioType=PoD&page=2&limit=1&sort=fitnessScore&direction=asc")
	if filteredPage.Pagination.Total != 3 || filteredPage.Pagination.TotalPages != 3 ||
		len(filteredPage.Scenarios) != 1 || filteredPage.Scenarios[0].ScenarioID != "10" {
		t.Fatalf("filtered score page omitted rows from the global union: %+v", filteredPage)
	}
	filteredMissing := fixture.readIndex(t, "?scenarioType=pod&page=3&limit=1&sort=fitnessScore&direction=asc")
	if filteredMissing.Pagination.Total != 3 || len(filteredMissing.Scenarios) != 1 ||
		filteredMissing.Scenarios[0].ScenarioID != "3" || filteredMissing.Scenarios[0].FitnessScore != nil ||
		filteredMissing.Scenarios[0].Phase != "Pending" {
		t.Fatalf("missing-score child row was not retained last: %+v", filteredMissing)
	}
	if matching := fixture.readIndex(t, "?search=2&scenarioType=pod"); matching.Pagination.Total != 1 ||
		len(matching.Scenarios) != 1 || matching.Scenarios[0].ChildRunName != "matching-id-2" ||
		matching.Scenarios[0].Phase != "Running" {
		t.Fatalf("search/type substring filters did not preserve matching child enrichment: %+v", matching)
	}
	for _, test := range []struct {
		key, direction string
		want           []string
	}{
		{key: "scenarioId", direction: "asc", want: []string{"baseline", "2", "3", "7", "10"}},
		{key: "scenarioId", direction: "desc", want: []string{"10", "7", "3", "2", "baseline"}},
		{key: "scenarioType", direction: "asc", want: []string{"baseline", "7", "2", "3", "10"}},
		{key: "scenarioType", direction: "desc", want: []string{"2", "3", "10", "7", "baseline"}},
		{key: "fitnessScore", direction: "asc", want: []string{"baseline", "2", "10", "3", "7"}},
		{key: "fitnessScore", direction: "desc", want: []string{"10", "2", "baseline", "3", "7"}},
		{key: "durationSeconds", direction: "asc", want: []string{"7", "10", "baseline", "2", "3"}},
		{key: "durationSeconds", direction: "desc", want: []string{"baseline", "10", "7", "2", "3"}},
		{key: "outcome", direction: "asc", want: []string{"10", "3", "2", "baseline", "7"}},
		{key: "outcome", direction: "desc", want: []string{"baseline", "7", "2", "3", "10"}},
	} {
		index := fixture.readIndex(t, "?sort="+test.key+"&direction="+test.direction)
		got := make([]string, len(index.Scenarios))
		for i := range index.Scenarios {
			got[i] = index.Scenarios[i].ScenarioID
		}
		if !reflect.DeepEqual(got, test.want) || index.Pagination.Total != 5 {
			t.Errorf("sort %s %s = %v total=%d, want %v total=5", test.key, test.direction, got, index.Pagination.Total, test.want)
		}
	}
}

type krknAIScenarioIndexFixture struct {
	handler      *Handler
	serviceCalls int
}

func newKrknAIScenarioIndexFixture(t *testing.T) *krknAIScenarioIndexFixture {
	t.Helper()
	fixture := &krknAIScenarioIndexFixture{}
	artifactIndex := `{"scenarios":[
		{"generation":0,"scenarioId":"baseline","scenarioType":"baseline","outcome":"succeeded","durationSeconds":60,"fitnessScore":5},
		{"generation":1,"scenarioId":"10","scenarioType":"pod-scenarios","outcome":"failed","durationSeconds":20,"fitnessScore":20},
		{"generation":1,"scenarioId":"2","scenarioType":"pod-scenarios","outcome":"succeeded","fitnessScore":10},
		{"generation":1,"scenarioId":"3","scenarioType":"pod-scenarios","outcome":"failed"},
		{"generation":1,"scenarioId":"7","scenarioType":"network-latency","outcome":"succeeded","durationSeconds":7}
	]}`
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fixture.serviceCalls++
		if r.URL.Path != "/v1/runs/uid-union/scenarios" {
			http.NotFound(w, r)
			return
		}
		if r.URL.RawQuery != "" {
			http.Error(w, "query parameters are not supported", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(artifactIndex))
	}))
	t.Cleanup(service.Close)
	fixture.handler = newKrknAITestHandler(t, service.URL)
	runUID := types.UID("uid-union")
	run := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "union-run", Namespace: "default", UID: runUID},
		Spec:       krknv1alpha1.KrknAIRunSpec{TargetRequestID: "target", TargetClusters: map[string][]string{"provider": {"cluster"}}},
	}
	if err := fixture.handler.client.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	for _, childRun := range []*krknv1alpha1.KrknScenarioRun{
		indexChildRun("matching-id-2", "union-run", run.Name, runUID, "1", "2", "pod-scenarios", "Running"),
		indexChildRun("matching-id-3", "union-run", run.Name, runUID, "1", "3", "pod-scenarios", "Pending"),
	} {
		if err := fixture.handler.client.Create(context.Background(), childRun); err != nil {
			t.Fatal(err)
		}
	}
	return fixture
}

func indexChildRun(name, aiRunLabel, ownerName string, ownerUID types.UID, generation, scenarioID, scenarioType, phase string) *krknv1alpha1.KrknScenarioRun {
	return &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", OwnerReferences: []metav1.OwnerReference{
			{APIVersion: krknv1alpha1.GroupVersion.String(), Kind: "KrknAIRun", Name: ownerName, UID: ownerUID},
		}, Labels: map[string]string{
			"krkn.dev/ai-run": aiRunLabel, "krkn.dev/generation-id": generation,
			"krkn.dev/scenario-id": scenarioID, "krkn.dev/scenario-name": scenarioType,
		}},
		Status: krknv1alpha1.KrknScenarioRunStatus{Phase: phase},
	}
}

func (f *krknAIScenarioIndexFixture) request(t *testing.T, query string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	f.handler.KrknAIRouter(response, adminKrknAIRequest(
		http.MethodGet, KrknAIPath+"/runs/union-run/results/scenarios"+query, "",
	))
	return response
}

func (f *krknAIScenarioIndexFixture) readIndex(t *testing.T, query string) KrknAIScenarioIndexResponse {
	t.Helper()
	response := f.request(t, query)
	if response.Code != http.StatusOK {
		t.Fatalf("index status = %d: %s", response.Code, response.Body.String())
	}
	var result KrknAIScenarioIndexResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestKrknAIRunScenarioIndexReauthorizesAndRejectsRunReplacement(t *testing.T) {
	for _, test := range []struct {
		name           string
		wantStatus     int
		replaceUID     bool
		lateDeny       bool
		wantChildLists int
		wantGroupReads int
	}{
		{name: "late target authorization denial", wantStatus: http.StatusForbidden, lateDeny: true, wantChildLists: 2, wantGroupReads: 2},
		{name: "run UID changed", wantStatus: http.StatusConflict, replaceUID: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			serviceCalls := 0
			service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				serviceCalls++
				_, _ = w.Write([]byte(`{"scenarios":[{"generation":1,"scenarioId":"artifact"}]}`))
			}))
			defer service.Close()
			handler := newKrknAITestHandler(t, service.URL)
			runUID := types.UID("initial-uid")
			run := &krknv1alpha1.KrknAIRun{
				ObjectMeta: metav1.ObjectMeta{Name: "changing-run", Namespace: "default", UID: runUID},
				Spec:       krknv1alpha1.KrknAIRunSpec{TargetRequestID: "target", TargetClusters: map[string][]string{"provider": {"cluster"}}},
			}
			if err := handler.client.Create(context.Background(), run); err != nil {
				t.Fatal(err)
			}
			if test.lateDeny {
				owner := []metav1.OwnerReference{{APIVersion: krknv1alpha1.GroupVersion.String(), Kind: "KrknAIRun", Name: run.Name, UID: runUID}}
				child := &krknv1alpha1.KrknScenarioRun{
					ObjectMeta: metav1.ObjectMeta{Name: "authorization-child", Namespace: "default", OwnerReferences: owner, Labels: map[string]string{
						"krkn.dev/ai-run": "changing-run",
					}},
					Spec: krknv1alpha1.KrknScenarioRunSpec{
						TargetRequestID: "target", TargetClusters: map[string][]string{"provider": {"cluster"}},
					},
					Status: krknv1alpha1.KrknScenarioRunStatus{ClusterJobs: []krknv1alpha1.ClusterJobStatus{
						{ClusterName: "cluster", ClusterAPIURL: "https://cluster.example"},
					}},
				}
				viewer := &krknv1alpha1.KrknUser{ObjectMeta: metav1.ObjectMeta{
					Name: "krknuser-viewer-example-com", Namespace: "default",
					Labels: map[string]string{groupauth.GroupLabelKey("team"): "true"},
				}}
				group := &krknv1alpha1.KrknUserGroup{
					ObjectMeta: metav1.ObjectMeta{Name: "team", Namespace: "default"},
					Spec: krknv1alpha1.KrknUserGroupSpec{ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
						"https://cluster.example": {Actions: []string{"view"}},
					}},
				}
				for _, object := range []client.Object{child, viewer, group} {
					if err := handler.client.Create(context.Background(), object); err != nil {
						t.Fatal(err)
					}
				}
			}
			trackingClient := &krknAIRunReadHookClient{Client: handler.client, replaceUID: test.replaceUID, lateDeny: test.lateDeny}
			handler.client = trackingClient
			response := httptest.NewRecorder()
			request := adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/changing-run/results/scenarios", "")
			if test.lateDeny {
				request = userKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/changing-run/results/scenarios", "", "viewer@example.com")
			}
			handler.KrknAIRouter(response, request)
			if response.Code != test.wantStatus || serviceCalls != 1 || trackingClient.runReads != 2 ||
				trackingClient.childLists != test.wantChildLists || trackingClient.groupReads != test.wantGroupReads {
				t.Fatalf("status/service/run reads/child lists/group reads = %d/%d/%d/%d/%d, want %d/1/2/%d/%d: %s",
					response.Code, serviceCalls, trackingClient.runReads, trackingClient.childLists, trackingClient.groupReads,
					test.wantStatus, test.wantChildLists, test.wantGroupReads, response.Body.String())
			}
		})
	}
}

type krknAIRunReadHookClient struct {
	client.Client
	runReads   int
	childLists int
	groupReads int
	replaceUID bool
	lateDeny   bool
}

func (c *krknAIRunReadHookClient) Get(ctx context.Context, key client.ObjectKey, object client.Object, options ...client.GetOption) error {
	if err := c.Client.Get(ctx, key, object, options...); err != nil {
		return err
	}
	if run, ok := object.(*krknv1alpha1.KrknAIRun); ok {
		c.runReads++
		if c.runReads == 2 && c.replaceUID {
			run.UID = types.UID("replacement-uid")
		}
	}
	if group, ok := object.(*krknv1alpha1.KrknUserGroup); ok {
		c.groupReads++
		if c.lateDeny && c.groupReads == 2 {
			group.Spec.ClusterPermissions = map[string]krknv1alpha1.ClusterPermissionSet{}
		}
	}
	return nil
}

func (c *krknAIRunReadHookClient) List(ctx context.Context, list client.ObjectList, options ...client.ListOption) error {
	if _, ok := list.(*krknv1alpha1.KrknScenarioRunList); ok {
		c.childLists++
	}
	return c.Client.List(ctx, list, options...)
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
