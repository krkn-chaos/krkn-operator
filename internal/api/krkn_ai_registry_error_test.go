package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/krkn-chaos/krkn-operator/pkg/registry"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPrivateKrknAIReportsRegistryConfigurationFailure(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/requirements" {
			t.Errorf("unexpected service request: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"scenarioTags":["pod-scenarios"],"workloadImageTags":{"hog":"workload-krkn-hog","tools":"workload-krkn-tools","synFlood":"workload-krkn-syn-flood","networkChaos":"workload-krkn-network-chaos"}}`))
	}))
	defer service.Close()
	handler := newKrknAITestHandler(t, service.URL)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "private", Namespace: handler.namespace, Labels: registry.BuildRegistryLabels(registry.AuthTypePassword, nil, true)},
		Data:       map[string][]byte{corev1.DockerConfigJsonKey: []byte("synthetic-credential-that-must-not-appear")},
	}
	if err := handler.client.Create(t.Context(), secret); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.KrknAIRouter(response, adminKrknAIRequest(http.MethodPost, KrknAIPath+"/configs/validate", `{"configYaml":"generations: 1\n","registryName":"private"}`))
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "registry") {
		t.Fatalf("registry configuration failure was not actionable: %d %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "synthetic-credential") {
		t.Fatal("registry error exposed credential contents")
	}
}
