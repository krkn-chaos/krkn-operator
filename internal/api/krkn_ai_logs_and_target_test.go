package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestKrknAIEnabledConstructorPropagation(t *testing.T) {
	handler := NewHandler(nil, nil, "default", "", &auth.SecretManager{}, false)
	if handler.krknAIEnabled {
		t.Fatal("NewHandler did not retain disabled Krkn-AI runtime flag")
	}
	handler.Shutdown()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	client := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	secretManager := auth.NewSecretManager(client, "default", time.Hour, "test-issuer")
	for _, enabled := range []bool{false, true} {
		server := NewServer(0, client, nil, "default", "", secretManager, enabled)
		if server.handler.krknAIEnabled != enabled {
			t.Errorf("NewServer flag propagation = %t, want %t", server.handler.krknAIEnabled, enabled)
		}
		if err := server.Shutdown(); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	}
}

func TestDeleteTargetByUUIDRejectsKrknAIReferences(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	for _, reference := range []string{"run", "config"} {
		t.Run(reference, func(t *testing.T) {
			target := &krknv1alpha1.KrknTargetRequest{
				ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: "default"},
			}
			objects := []runtime.Object{target}
			if reference == "run" {
				objects = append(objects, &krknv1alpha1.KrknAIRun{
					ObjectMeta: metav1.ObjectMeta{Name: "ai-run", Namespace: "default"},
					Spec:       krknv1alpha1.KrknAIRunSpec{TargetRequestID: "target"},
				})
			} else {
				objects = append(objects, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ai-config",
						Namespace: "default",
						Labels:    map[string]string{files.FilePurposeLabel: files.FilePurposeKrknAIConfig},
						Annotations: map[string]string{
							files.KrknAIConfigTargetRequestAnnotation: "target",
						},
					},
				})
			}
			k8sClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()
			handler := NewTestHandler(k8sClient, nil, "default", "")
			t.Cleanup(handler.Shutdown)

			req := httptest.NewRequest(http.MethodDelete, TargetsPath+"/target", nil)
			req = req.WithContext(createAdminContext())
			response := httptest.NewRecorder()
			handler.DeleteTargetByUUID(response, req)
			if response.Code != http.StatusConflict {
				t.Fatalf("delete status = %d, want %d: %s", response.Code, http.StatusConflict, response.Body.String())
			}
			var stillPresent krknv1alpha1.KrknTargetRequest
			if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "target", Namespace: "default"}, &stillPresent); err != nil {
				t.Fatalf("referenced target was deleted: %v", err)
			}
		})
	}
}

func TestKrknAIOrchestratorLogsWebSocketAuthorizationAndContainers(t *testing.T) {
	const namespace = "default"
	var mu sync.Mutex
	var requestedContainers []string
	var requestedQueries []url.Values
	logAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		mu.Lock()
		requestedContainers = append(requestedContainers, query.Get("container"))
		requestedQueries = append(requestedQueries, query)
		mu.Unlock()
		w.Header().Set("Content-Type", "text/plain")
		if r.URL.Query().Get("container") == "orchestrator" {
			_, _ = io.WriteString(w, "orchestrator output\n")
			return
		}
		_, _ = io.WriteString(w, "scenario output\n")
	}))
	defer logAPI.Close()
	clientset, err := kubernetes.NewForConfig(&rest.Config{Host: logAPI.URL})
	if err != nil {
		t.Fatal(err)
	}

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	target := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: namespace},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status: "Completed",
			TargetData: map[string][]krknv1alpha1.ClusterTarget{
				"provider": {{ClusterName: "cluster", ClusterAPIURL: "https://cluster.example"}},
			},
		},
	}
	aiRun := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "ai-run", Namespace: namespace, UID: types.UID("run-uid")},
		Spec: krknv1alpha1.KrknAIRunSpec{
			TargetRequestID: "target",
			TargetClusters:  map[string][]string{"provider": {"cluster"}},
		},
		Status: krknv1alpha1.KrknAIRunStatus{OrchestratorPodName: "orchestrator-pod"},
	}
	mismatchedRun := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "mismatched-run", Namespace: namespace, UID: types.UID("expected-run-uid")},
		Spec:       aiRun.Spec,
		Status:     krknv1alpha1.KrknAIRunStatus{OrchestratorPodName: "mismatched-pod"},
	}
	waitingRun := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "waiting-run", Namespace: namespace, UID: types.UID("waiting-run-uid")},
		Spec:       aiRun.Spec,
	}
	orchestratorPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "orchestrator-pod",
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{{APIVersion: krknv1alpha1.GroupVersion.String(), Kind: "KrknAIRun", Name: aiRun.Name, UID: aiRun.UID}},
		},
	}
	mismatchedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "mismatched-pod",
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{{APIVersion: krknv1alpha1.GroupVersion.String(), Kind: "KrknAIRun", Name: mismatchedRun.Name, UID: types.UID("different-run-uid")}},
		},
	}
	childRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "child-run", Namespace: namespace},
		Status: krknv1alpha1.KrknScenarioRunStatus{ClusterJobs: []krknv1alpha1.ClusterJobStatus{{
			JobID: "job-1", PodName: "child-pod", ClusterAPIURL: "https://cluster.example",
		}}},
	}
	childPod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "child-pod", Namespace: namespace}}
	k8sClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
		target, aiRun, mismatchedRun, waitingRun, orchestratorPod, mismatchedPod, childRun, childPod, TestJWTSecret(namespace),
	).Build()
	secretManager, err := auth.NewTestSecretManager(k8sClient, namespace)
	if err != nil {
		t.Fatal(err)
	}
	tokenGen, err := secretManager.GetTokenGenerator()
	if err != nil {
		t.Fatal(err)
	}
	adminToken, err := tokenGen.GenerateToken("admin@example.com", "admin", "Admin", "User", "Org")
	if err != nil {
		t.Fatal(err)
	}
	userToken, err := tokenGen.GenerateToken("user@example.com", "user", "User", "Test", "Org")
	if err != nil {
		t.Fatal(err)
	}

	apiServer := NewServer(0, k8sClient, clientset, namespace, "", secretManager, false)
	defer func() { _ = apiServer.Shutdown() }()
	server := httptest.NewServer(apiServer.HTTPHandler())
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	dial := func(token, path string) (*websocket.Conn, *http.Response, error) {
		return websocket.DefaultDialer.Dial(wsURL+path, http.Header{"Sec-WebSocket-Protocol": []string{"access_token." + token}})
	}
	connect := func(token, path string) string {
		t.Helper()
		conn, response, err := dial(token, path)
		if err != nil {
			if response != nil {
				_ = response.Body.Close()
			}
			t.Fatalf("WebSocket handshake failed: %v", err)
		}
		defer conn.Close()
		_, message, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read WebSocket text: %v", err)
		}
		return string(message)
	}

	message := connect(adminToken, "/api/v2/ws/krkn-ai/runs/ai-run/logs?follow=true&tailLines=9&timestamps=true")
	if message != "orchestrator output" {
		t.Fatalf("orchestrator log = %q", message)
	}
	message = connect(adminToken, "/api/v2/ws/krkn-ai/runs/waiting-run/logs")
	if !strings.HasPrefix(message, "WAITING: ") {
		t.Fatalf("waiting response = %q", message)
	}
	if _, response, err := dial("not-a-valid-jwt", "/api/v2/ws/krkn-ai/runs/ai-run/logs"); err == nil {
		t.Fatal("invalid access_token unexpectedly upgraded WebSocket")
	} else if response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("invalid token handshake response = %v, want HTTP %d", response, http.StatusUnauthorized)
	} else {
		_ = response.Body.Close()
	}

	if _, response, err := dial(userToken, "/api/v2/ws/krkn-ai/runs/ai-run/logs"); err == nil {
		t.Fatal("unauthorized target viewer unexpectedly upgraded WebSocket")
	} else if response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("unauthorized handshake response = %v, want HTTP %d", response, http.StatusForbidden)
	} else {
		_ = response.Body.Close()
	}
	if _, response, err := dial(adminToken, "/api/v2/ws/krkn-ai/runs/mismatched-run/logs"); err == nil {
		t.Fatal("mismatched Pod owner unexpectedly upgraded WebSocket")
	} else if response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("mismatched Pod handshake response = %v, want HTTP %d", response, http.StatusForbidden)
	} else {
		_ = response.Body.Close()
	}

	message = connect(adminToken, "/api/v2/ws/scenarios/run/child-run/jobs/job-1/logs?follow=false")
	if message != "scenario output" {
		t.Fatalf("child scenario log = %q", message)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(requestedContainers) != 2 || requestedContainers[0] != "orchestrator" || requestedContainers[1] != "scenario" {
		t.Fatalf("Pod log containers = %v, want [orchestrator scenario]", requestedContainers)
	}
	orchestratorQuery := requestedQueries[0]
	if orchestratorQuery.Get("follow") != "true" || orchestratorQuery.Get("timestamps") != "true" || orchestratorQuery.Get("tailLines") != "9" {
		t.Fatalf("orchestrator log query = %v, want follow=true timestamps=true tailLines=9", orchestratorQuery)
	}
}
