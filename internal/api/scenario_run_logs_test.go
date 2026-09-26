package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekubernetes "k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

func TestPreStartFailureReason(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		want   string
	}{
		{name: "invalid image", reason: "InvalidImageName", want: "InvalidImageName"},
		{name: "transient container creation", reason: "ContainerCreating", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &corev1.Pod{Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{{
					State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: tt.reason}},
				}},
			}}
			if got := preStartFailureReason(pod); got != tt.want {
				t.Errorf("preStartFailureReason() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetScenarioRunLogs_MaxRetriesExceededStreamsLatestPodLogs(t *testing.T) {
	const namespace = "default"
	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "scenario-run", Namespace: namespace},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{{
				JobID:       "job-1",
				ClusterName: "cluster-1",
				PodName:     "latest-retry-pod",
				Phase:       "MaxRetriesExceeded",
				RetryCount:  3,
				MaxRetries:  3,
			}},
		},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "latest-retry-pod", Namespace: namespace},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{{
				State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled"}},
			}},
		},
	}
	client := fakeclient.NewClientBuilder().WithScheme(scheme).
		WithObjects(TestJWTSecret(namespace), scenarioRun, pod).Build()
	clientset := fakekubernetes.NewSimpleClientset()
	handler := NewTestHandler(client, clientset, namespace, "")
	tokenGenerator := auth.NewTokenGenerator([]byte("test-jwt-secret-key-32-bytes!!!!"), TokenDuration, "test-issuer")
	token, err := tokenGenerator.GenerateToken("admin@example.com", "admin", "Admin", "User", "Org")
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(handler.GetScenarioRunLogs))
	defer server.Close()
	conn, response, err := websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(server.URL, "http")+"/api/v2/ws/scenarios/run/scenario-run/jobs/job-1/logs",
		http.Header{"Sec-WebSocket-Protocol": []string{"access_token." + token}},
	)
	if err != nil {
		if response != nil {
			t.Fatalf("dial WebSocket: %v (status %s)", err, response.Status)
		}
		t.Fatal(err)
	}
	defer conn.Close()

	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read streamed logs: %v", err)
	}
	if string(message) != "fake logs" {
		t.Fatalf("expected latest pod logs, got %q", message)
	}

	for _, action := range clientset.Actions() {
		if action.GetVerb() == "get" && action.GetResource().Resource == "pods" && action.GetSubresource() == "log" {
			return
		}
	}
	t.Fatal("expected handler to request logs for the latest retry pod")
}

func TestGetScenarioRunLogs_PreStartFailureSkipsPodLogs(t *testing.T) {
	const namespace = "default"
	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "scenario-run", Namespace: namespace},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{{
				JobID:         "job-1",
				ClusterName:   "cluster-1",
				PodName:       "failed-pod",
				Phase:         "Failed",
				FailureReason: "ContainerError",
			}},
		},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "failed-pod", Namespace: namespace},
		Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
			State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}},
		}}},
	}
	client := fakeclient.NewClientBuilder().WithScheme(scheme).
		WithObjects(TestJWTSecret(namespace), scenarioRun, pod).Build()
	clientset := fakekubernetes.NewSimpleClientset()
	handler := NewTestHandler(client, clientset, namespace, "")
	tokenGenerator := auth.NewTokenGenerator([]byte("test-jwt-secret-key-32-bytes!!!!"), TokenDuration, "test-issuer")
	token, err := tokenGenerator.GenerateToken("admin@example.com", "admin", "Admin", "User", "Org")
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(handler.GetScenarioRunLogs))
	defer server.Close()
	conn, response, err := websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(server.URL, "http")+"/api/v2/ws/scenarios/run/scenario-run/jobs/job-1/logs",
		http.Header{"Sec-WebSocket-Protocol": []string{"access_token." + token}},
	)
	if err != nil {
		if response != nil {
			t.Fatalf("dial WebSocket: %v (status %s)", err, response.Status)
		}
		t.Fatal(err)
	}
	defer conn.Close()

	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed-job response: %v", err)
	}
	expected := "ERROR: Logs unavailable because the pod failed before startup (ImagePullBackOff)"
	if string(message) != expected {
		t.Fatalf("expected %q, got %q", expected, message)
	}
	for _, action := range clientset.Actions() {
		if action.GetVerb() == "get" && action.GetResource().Resource == "pods" && action.GetSubresource() == "log" {
			t.Fatal("failed job should not request pod logs")
		}
	}
}
