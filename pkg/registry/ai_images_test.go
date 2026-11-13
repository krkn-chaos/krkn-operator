package registry

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateAITagMetadata(t *testing.T) {
	if err := ValidateAITagMetadata(AITagMetadata{
		AIOrchestratorTag: "combined", AIResultUploaderTag: "combined",
		WorkloadImageTags: map[string]string{"hog": "hog-v1", "networkChaos": "chaos-v2"},
	}); err != nil {
		t.Fatalf("valid combined runtime tag configuration rejected: %v", err)
	}
	for _, metadata := range []AITagMetadata{
		{AIOrchestratorTag: "registry.example/image:latest"},
		{WorkloadImageTags: map[string]string{"unknown": "tag"}},
		{WorkloadImageTags: map[string]string{"tools": "bad/tag"}},
	} {
		if err := ValidateAITagMetadata(metadata); err == nil {
			t.Errorf("ValidateAITagMetadata(%+v) unexpectedly succeeded", metadata)
		}
	}
}

func TestInspectAIImagesUsesPrivateAuthAndReturnsDigestPinnedRuntimeImages(t *testing.T) {
	configBytes := []byte(`{"architecture":"amd64","os":"linux","config":{"Labels":{"io.krkn.operator-contract.major":"1","krknctl.is_a_scenario":"false"}}}`)
	configHash := sha256.Sum256(configBytes)
	configDigest := "sha256:" + hex.EncodeToString(configHash[:])
	manifestBytes, err := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.oci.image.manifest.v1+json",
		"config": map[string]any{
			"mediaType": "application/vnd.oci.image.config.v1+json",
			"digest":    configDigest,
			"size":      len(configBytes),
		},
		"layers": []any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifestHash := sha256.Sum256(manifestBytes)
	manifestDigest := "sha256:" + hex.EncodeToString(manifestHash[:])
	available := map[string]bool{
		"scenario-v1": true, "custom-hog-v2": true, "workload-krkn-tools": true,
		"workload-krkn-syn-flood": true,
		"orch-v1":                 true, "upload-v1": true,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
			w.WriteHeader(http.StatusOK)
			return
		}
		if strings.Contains(r.URL.Path, "/manifests/") || strings.Contains(r.URL.Path, "/blobs/") {
			if r.Header.Get("Authorization") != "Basic "+base64.StdEncoding.EncodeToString([]byte("robot:private-password")) {
				w.Header().Set("WWW-Authenticate", `Basic realm="test-registry"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}
		if strings.HasSuffix(r.URL.Path, "/blobs/"+configDigest) {
			_, _ = w.Write(configBytes)
			return
		}
		if strings.Contains(r.URL.Path, "/manifests/") {
			tag := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
			if !available[tag] {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			w.Header().Set("Docker-Content-Digest", manifestDigest)
			_, _ = w.Write(manifestBytes)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	registryHost := strings.TrimPrefix(server.URL, "http://")
	configJSON, err := BuildDockerConfigJSON(registryHost, "robot", "private-password")
	if err != nil {
		t.Fatal(err)
	}
	metadata := AITagMetadata{
		AIOrchestratorTag: "orch-v1", AIResultUploaderTag: "upload-v1",
		WorkloadImageTags: map[string]string{
			"hog": "custom-hog-v2", "tools": "missing-tools-v2",
			"networkChaos": "workload-krkn-tools",
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "private", Labels: BuildRegistryLabels(AuthTypePassword, nil, false),
			Annotations: BuildRegistryAnnotations(registryHost, "nested/org/repo", "", false, true, "", metadata),
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{corev1.DockerConfigJsonKey: configJSON},
	}
	defaults := map[string]string{
		"hog": "workload-krkn-hog", "tools": "workload-krkn-tools",
		"synFlood": "workload-krkn-syn-flood", "networkChaos": "workload-krkn-network-chaos",
	}
	inventory, err := InspectAIImages(context.Background(), secret, []string{"scenario-v1", "missing-scenario"}, defaults)
	if err != nil {
		t.Fatalf("InspectAIImages() error = %v", err)
	}
	if inventory.RegistryName != "private" || inventory.Repository != registryHost+"/nested/org/repo" {
		t.Fatalf("inventory registry location = %+v", inventory)
	}
	if len(inventory.AvailableTags) != 4 || inventory.WorkloadImageTags["tools"] != "missing-tools-v2" || inventory.WorkloadImageTags["hog"] != "custom-hog-v2" {
		t.Fatalf("missing or overridden tags were not handled: %+v", inventory)
	}
	if inventory.OrchestratorImage != inventory.Repository+"@"+manifestDigest || inventory.ResultUploaderImage != inventory.Repository+"@"+manifestDigest {
		t.Fatalf("runtime images are not digest-pinned: %+v", inventory)
	}

	delete(available, "orch-v1")
	delete(available, "upload-v1")
	available["krkn-ai-operator"] = true
	available["krkn-ai-service"] = true
	available["workload-krkn-hog"] = true
	for _, blank := range []bool{false, true} {
		defaultSecret := secret.DeepCopy()
		delete(defaultSecret.Annotations, AIOrchestratorTagAnnotation)
		delete(defaultSecret.Annotations, AIResultUploaderTagAnnotation)
		delete(defaultSecret.Annotations, WorkloadImageTagsAnnotation)
		if blank {
			defaultSecret.Annotations[AIOrchestratorTagAnnotation] = ""
			defaultSecret.Annotations[AIResultUploaderTagAnnotation] = ""
			defaultSecret.Annotations[WorkloadImageTagsAnnotation] = `{"hog":"","tools":"","synFlood":"","networkChaos":""}`
		}
		defaultInventory, err := InspectAIImages(context.Background(), defaultSecret, []string{"scenario-v1"}, defaults)
		if err != nil {
			t.Fatalf("default private tags (blank=%t): %v", blank, err)
		}
		if defaultInventory.OrchestratorImage != inventory.OrchestratorImage || defaultInventory.ResultUploaderImage != inventory.ResultUploaderImage || defaultInventory.WorkloadImageTags["hog"] != "workload-krkn-hog" {
			t.Fatalf("private default images were not resolved: %+v", defaultInventory)
		}
	}
	defaultSecret := secret.DeepCopy()
	delete(defaultSecret.Annotations, AIOrchestratorTagAnnotation)
	delete(defaultSecret.Annotations, AIResultUploaderTagAnnotation)
	delete(available, "krkn-ai-service")
	if _, err := InspectAIImages(context.Background(), defaultSecret, []string{"scenario-v1"}, defaults); !errors.Is(err, ErrAIRegistryConfiguration) {
		t.Fatalf("missing default service image error = %v", err)
	}
	available["orch-v1"] = true
	available["upload-v1"] = true

	available["dummy-scenario"] = true
	delete(available, "scenario-v1")
	_, err = InspectAIImages(context.Background(), secret, []string{"dummy-scenario", "scenario-v1"}, defaults)
	if !errors.Is(err, ErrAIRegistryConfiguration) || !strings.Contains(err.Error(), "no runnable scenario images") {
		t.Fatalf("baseline-only registry error = %v", err)
	}
	available["scenario-v1"] = true

	delete(available, "upload-v1")
	_, err = InspectAIImages(context.Background(), secret, []string{"scenario-v1"}, defaults)
	if !errors.Is(err, ErrAIRegistryConfiguration) {
		t.Fatalf("missing private runtime image error = %v, want ErrAIRegistryConfiguration", err)
	}
}

func TestInspectAIImagesRejectsIncompleteRuntimeConfiguration(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "private", Annotations: map[string]string{
			AIOrchestratorTagAnnotation: "orch", AIResultUploaderTagAnnotation: "upload",
		}},
	}
	_, err := InspectAIImages(context.Background(), secret, nil, map[string]string{
		"hog": "h", "tools": "t", "synFlood": "s", "networkChaos": "n",
	})
	if !errors.Is(err, ErrAIRegistryConfiguration) {
		t.Fatalf("InspectAIImages() error = %v, want ErrAIRegistryConfiguration", err)
	}
}
