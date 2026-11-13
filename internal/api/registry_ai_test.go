package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/registry"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRegistryAITagMetadataRoundTripsAndCanBeCleared(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	kubeClient := controllerfake.NewClientBuilder().WithScheme(scheme).Build()
	handler := NewTestHandler(kubeClient, fake.NewSimpleClientset(), "default", "unused")
	adminRequest := func(method, path, body string) *http.Request {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		return req.WithContext(context.WithValue(req.Context(), auth.UserClaimsKey, &auth.Claims{UserID: "admin@example.com", Role: "admin"}))
	}

	create := httptest.NewRecorder()
	handler.RegistriesRouter(create, adminRequest(http.MethodPost, RegistriesPath, `{"name":"private","registryUrl":"registry.example:5000","scenarioRepository":"org/repo","authType":"token","username":"runner","password":"never-return-this","aiOrchestratorTag":"orch-v1","aiResultUploaderTag":"upload-v1","workloadImageTags":{"hog":"hog-v1"}}`))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d: %s", create.Code, create.Body.String())
	}

	get := httptest.NewRecorder()
	handler.RegistriesRouter(get, adminRequest(http.MethodGet, RegistriesPath+"/private", ""))
	var response registry.RegistryResponse
	if get.Code != http.StatusOK || json.Unmarshal(get.Body.Bytes(), &response) != nil {
		t.Fatalf("get response = %d %s", get.Code, get.Body.String())
	}
	if response.AIOrchestratorTag != "orch-v1" || response.AIResultUploaderTag != "upload-v1" || response.WorkloadImageTags["hog"] != "hog-v1" {
		t.Fatalf("AI image tags did not round-trip: %+v", response.AITagMetadata)
	}
	if strings.Contains(get.Body.String(), "never-return-this") {
		t.Fatal("registry response exposed credentials")
	}
	available := httptest.NewRecorder()
	handler.ListAvailableRegistries(available, adminRequest(http.MethodGet, RegistriesPath+"/available", ""))
	var summaries registry.AvailableRegistriesResponse
	if available.Code != http.StatusOK || json.Unmarshal(available.Body.Bytes(), &summaries) != nil || len(summaries.Registries) != 1 {
		t.Fatalf("available registries response = %d %s", available.Code, available.Body.String())
	}
	if summaries.Registries[0].AIOrchestratorTag != "orch-v1" || summaries.Registries[0].WorkloadImageTags["hog"] != "hog-v1" {
		t.Fatal("available registry summary omitted image tag metadata")
	}
	if strings.Contains(available.Body.String(), "never-return-this") {
		t.Fatal("available registry summary exposed credentials")
	}

	update := httptest.NewRecorder()
	handler.RegistriesRouter(update, adminRequest(http.MethodPut, RegistriesPath+"/private", `{"registryUrl":"registry.example:5000","scenarioRepository":"org/repo","authType":"token"}`))
	if update.Code != http.StatusOK {
		t.Fatalf("update status = %d: %s", update.Code, update.Body.String())
	}
	var stored corev1.Secret
	if err := kubeClient.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "private"}, &stored); err != nil {
		t.Fatal(err)
	}
	savedTags, err := registry.AIRegistryTagMetadata(&stored)
	if err != nil || savedTags.AIOrchestratorTag != "orch-v1" || savedTags.AIResultUploaderTag != "upload-v1" || savedTags.WorkloadImageTags["hog"] != "hog-v1" {
		t.Fatalf("omitted update fields did not preserve AI metadata: %+v, error %v", savedTags, err)
	}
	clear := httptest.NewRecorder()
	handler.RegistriesRouter(clear, adminRequest(http.MethodPut, RegistriesPath+"/private", `{"registryUrl":"registry.example:5000","scenarioRepository":"org/repo","authType":"token","aiOrchestratorTag":"","aiResultUploaderTag":"","workloadImageTags":{}}`))
	if clear.Code != http.StatusOK {
		t.Fatalf("clear update status = %d: %s", clear.Code, clear.Body.String())
	}
	if err := kubeClient.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "private"}, &stored); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{registry.AIOrchestratorTagAnnotation, registry.AIResultUploaderTagAnnotation, registry.WorkloadImageTagsAnnotation} {
		if _, found := stored.Annotations[key]; found {
			t.Errorf("explicitly cleared update retained AI annotation %q", key)
		}
	}
}
