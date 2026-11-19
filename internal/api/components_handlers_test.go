package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/krknaiserver"
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

func TestListComponentsKrknAIVersion(t *testing.T) {
	for _, tt := range []struct {
		name    string
		enabled bool
		status  int
		version string
	}{
		{"enabled", true, http.StatusOK, "1.2.3"},
		{"disabled", false, http.StatusOK, ""},
		{"unavailable", true, http.StatusServiceUnavailable, ""},
		{"older service", true, http.StatusNotFound, ""},
		{"unresponsive", true, 0, ""},
		{"no client", true, 0, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if tt.status == 0 {
					<-r.Context().Done()
					return
				}
				w.WriteHeader(tt.status)
				_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.2.3"})
			}))
			defer service.Close()
			scheme := runtime.NewScheme()
			if err := krknv1alpha1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			k8sClient := fakeclient.NewClientBuilder().WithScheme(scheme).
				WithObjects(&krknv1alpha1.KrknComponent{
					ObjectMeta: metav1.ObjectMeta{Namespace: "operator-system", Name: "krkn-operator"},
					Status: krknv1alpha1.KrknComponentStatus{Components: map[string]krknv1alpha1.ComponentManifest{
						"krkn-operator": {Version: "v1.0.0", Components: []krknv1alpha1.ComponentDescription{{Component: "api", Description: "API"}}},
					}},
				}).Build()
			handler := &Handler{
				client: k8sClient, namespace: "operator-system", krknAIEnabled: tt.enabled,
			}
			if tt.name != "no client" {
				handler.artifactClient = krknaiserver.New(service.URL, "token")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			w := httptest.NewRecorder()
			handler.ListComponents(w, httptest.NewRequest(http.MethodGet, ComponentsPath, nil).WithContext(ctx))
			if w.Code != http.StatusOK || ctx.Err() != nil {
				t.Fatalf("components response = %d, context error = %v", w.Code, ctx.Err())
			}
			var response ListComponentsResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			operator := response.Components["krkn-operator"]
			if operator.Version != "v1.0.0" {
				t.Fatal("AI availability affected the operator version")
			}
			wantOperatorComponents := 1
			if tt.enabled {
				wantOperatorComponents++
			}
			if len(operator.Components) != wantOperatorComponents || operator.Components[0].Component != "api" ||
				(tt.enabled && operator.Components[1].Component != "krkn-ai-operator") {
				t.Fatalf("operator components = %+v", operator.Components)
			}
			ai, present := response.Components["krkn-ai-service"]
			if present != (tt.version != "") || ai.Version != tt.version {
				t.Fatalf("AI manifest = %+v, present = %t", ai, present)
			}
			if present && (len(ai.Components) != 1 || ai.Components[0].Component != "rest-api") {
				t.Fatalf("AI components = %+v", ai.Components)
			}
			if _, present := response.Components["krkn-ai"]; present {
				t.Fatal("unexpected standalone krkn-ai entry")
			}
			if !tt.enabled && calls.Load() != 0 {
				t.Fatal("disabled AI was contacted")
			}
		})
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
		server := NewServerWithComponentNamespace(0, k8sClient, fakekube.NewSimpleClientset(), "targets", namespace, "localhost:50051", secretManager, KrknAIServiceConfig{Enabled: enabled})
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
