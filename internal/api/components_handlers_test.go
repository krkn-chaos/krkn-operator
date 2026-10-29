package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekube "k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestListComponents(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	manifest := krknv1alpha1.ComponentManifest{Version: "v1.1.0-rc4", Components: []krknv1alpha1.ComponentDescription{{Component: "api", Description: "API"}}}
	makeResource := func(namespace, name string) *krknv1alpha1.KrknComponent {
		return &krknv1alpha1.KrknComponent{
			ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
			Status:     krknv1alpha1.KrknComponentStatus{Components: map[string]krknv1alpha1.ComponentManifest{name: manifest}},
		}
	}
	client := fakeclient.NewClientBuilder().WithScheme(scheme).
		WithObjects(makeResource("operator-system", "krkn-operator"), makeResource("targets", "wrong-namespace")).Build()
	handler := &Handler{client: client, namespace: "targets", componentNamespace: "operator-system"}
	w := httptest.NewRecorder()
	handler.ListComponents(w, httptest.NewRequest(http.MethodGet, ComponentsPath, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var response struct {
		Components map[string]krknv1alpha1.ComponentManifest `json:"components"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Components) != 1 || response.Components["krkn-operator"].Version != manifest.Version {
		t.Fatalf("components = %#v", response.Components)
	}
	w = httptest.NewRecorder()
	handler.ListComponents(w, httptest.NewRequest(http.MethodPost, ComponentsPath, nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want 405", w.Code)
	}
}

func TestListComponentsRouteAndAuth(t *testing.T) {
	const namespace = "operator-system"
	const userID = "user@example.com"

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	makeResource := func(name, version string) *krknv1alpha1.KrknComponent {
		return &krknv1alpha1.KrknComponent{
			ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
			Status: krknv1alpha1.KrknComponentStatus{Components: map[string]krknv1alpha1.ComponentManifest{
				name: {Version: version, Components: []krknv1alpha1.ComponentDescription{{Component: "provider", Description: "Target provider"}}},
			}},
		}
	}
	activeUser := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{Name: sanitizeUsername(userID), Namespace: "targets"},
		Spec:       krknv1alpha1.KrknUserSpec{UserID: userID, Role: "user"},
		Status:     krknv1alpha1.KrknUserStatus{Active: true},
	}
	k8sClient := fakeclient.NewClientBuilder().WithScheme(scheme).
		WithObjects(makeResource("krkn-operator", "v1.1.0-rc4"), makeResource("krkn-operator-acm", "v1.1.0"), activeUser).Build()
	secretManager := auth.NewSecretManager(k8sClient, namespace, TokenDuration, "krkn-operator")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startResult := make(chan error, 1)
	go func() { startResult <- secretManager.Start(ctx) }()
	deadline := time.Now().Add(5 * time.Second)
	for !secretManager.IsReady() {
		select {
		case err := <-startResult:
			t.Fatalf("secret manager exited before becoming ready: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("secret manager did not become ready in time")
		}
		time.Sleep(10 * time.Millisecond)
	}

	for _, enabled := range []bool{false, true} {
		server := NewServerWithComponentNamespace(0, k8sClient, fakekube.NewSimpleClientset(), "targets", namespace, "localhost:50051", secretManager, enabled)
		defer func() { _ = server.Shutdown() }()
		mux := server.HTTPHandler()

		request := httptest.NewRequest(http.MethodGet, ComponentsPath, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, request)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("anonymous status = %d, want 401", w.Code)
		}

		tokenGen, err := secretManager.GetTokenGenerator()
		if err != nil {
			t.Fatal(err)
		}
		token, err := tokenGen.GenerateToken(userID, "user", "Regular", "User", "Org")
		if err != nil {
			t.Fatal(err)
		}
		request = httptest.NewRequest(http.MethodGet, ComponentsPath, nil)
		request.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, request)
		if w.Code != http.StatusOK {
			t.Fatalf("authenticated status = %d, body = %s", w.Code, w.Body.String())
		}
		var response ListComponentsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if len(response.Components) != 2 || response.Components["krkn-operator"].Version != "v1.1.0-rc4" || response.Components["krkn-operator-acm"].Version != "v1.1.0" {
			t.Fatalf("components = %#v", response.Components)
		}

		request = httptest.NewRequest(http.MethodGet, KrknAIPath+"/status", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, request)
		if w.Code != http.StatusOK {
			t.Fatalf("Krkn-AI status = %d, body = %s", w.Code, w.Body.String())
		}
		var status KrknAIStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
			t.Fatal(err)
		}
		if status.Enabled != enabled {
			t.Fatalf("Krkn-AI enabled = %t, want %t", status.Enabled, enabled)
		}
	}
}
