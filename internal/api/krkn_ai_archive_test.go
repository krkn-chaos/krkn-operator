package api

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type krknAIArchiveTestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

func krknAIArchiveManifestJSON(t *testing.T, files ...krknAIArchiveTestFile) []byte {
	t.Helper()
	payload, err := json.Marshal(struct {
		Files []krknAIArchiveTestFile `json:"files"`
	}{Files: files})
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func krknAIArchiveTestEntry(path string, content []byte) krknAIArchiveTestFile {
	digest := sha256.Sum256(content)
	return krknAIArchiveTestFile{Path: path, SHA256: hex.EncodeToString(digest[:]), Size: int64(len(content))}
}

func newKrknAIArchiveTestHandler(t *testing.T, service http.Handler) *Handler {
	t.Helper()
	server := httptest.NewServer(service)
	t.Cleanup(server.Close)
	handler := newKrknAITestHandler(t, server.URL)
	run := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "archive-run", Namespace: "default", UID: "archive-uid"},
		Spec:       krknv1alpha1.KrknAIRunSpec{TargetRequestID: "target", TargetClusters: map[string][]string{"provider": {"cluster"}}},
	}
	if err := handler.client.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestKrknAIRunArchiveDownloadContainsCommittedTextAndBinaryFiles(t *testing.T) {
	text := []byte("fitness summary\nall paths preserved\n")
	binary := []byte{0x00, 0xff, 0x10, 0x80, 0x42}
	manifest := krknAIArchiveManifestJSON(t,
		krknAIArchiveTestEntry("reports/fitness.txt", text),
		krknAIArchiveTestEntry("raw/generation-1/data.bin", binary),
	)
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/runs/archive-uid/results" {
			_, _ = w.Write(manifest)
			return
		}
		switch r.URL.Path {
		case "/v1/runs/archive-uid/files/reports/fitness.txt":
			_, _ = w.Write(text)
		case "/v1/runs/archive-uid/files/raw/generation-1/data.bin":
			_, _ = w.Write(binary)
		default:
			http.NotFound(w, r)
		}
	}))
	defer service.Close()
	handler := newKrknAITestHandler(t, service.URL)
	run := &krknv1alpha1.KrknAIRun{ObjectMeta: metav1.ObjectMeta{Name: "archive-run", Namespace: "default", UID: "archive-uid"}, Spec: krknv1alpha1.KrknAIRunSpec{TargetRequestID: "target", TargetClusters: map[string][]string{"provider": {"cluster"}}}}
	if err := handler.client.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	handler.KrknAIRouter(response, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/archive-run/results/download", ""))
	if response.Code != http.StatusOK {
		t.Fatalf("archive status = %d: %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/zip" ||
		response.Header().Get("Content-Disposition") != `attachment; filename=archive-run-results.zip` {
		t.Fatalf("unexpected archive headers: %v", response.Header())
	}
	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatalf("returned body is not a complete ZIP: %v", err)
	}
	found := make(map[string][]byte)
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		contents, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("reading %q: read=%v close=%v", file.Name, readErr, closeErr)
		}
		found[file.Name] = contents
	}
	if !bytes.Equal(found["reports/fitness.txt"], text) || !bytes.Equal(found["raw/generation-1/data.bin"], binary) || len(found) != 2 {
		t.Fatalf("archive entries did not preserve paths and bytes: %#v", found)
	}
}

func TestKrknAIRunArchiveDownloadRequiresAuthorizationBeforeServiceRead(t *testing.T) {
	var serviceCalls int
	handler := newKrknAIArchiveTestHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serviceCalls++
		_, _ = w.Write([]byte(`{"files":[]}`))
	}))
	request := userKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/archive-run/results/download", "", "unrelated@example.com")
	response := httptest.NewRecorder()
	handler.KrknAIRouter(response, request)
	if response.Code != http.StatusForbidden || serviceCalls != 0 {
		t.Fatalf("unauthorized archive status/calls = %d/%d, want 403/0: %s", response.Code, serviceCalls, response.Body.String())
	}
}

func TestKrknAIRunArchiveDownloadRejectsSnapshotMismatchWithoutZip(t *testing.T) {
	committed := []byte("committed version")
	changedSameSize := append([]byte(nil), committed...)
	changedSameSize[0] = 'C'
	for _, test := range []struct {
		name     string
		manifest krknAIArchiveTestFile
		body     []byte
	}{
		{name: "hash differs at same size", manifest: krknAIArchiveTestEntry("result.txt", committed), body: changedSameSize},
		{name: "size differs", manifest: krknAIArchiveTestFile{Path: "result.txt", SHA256: krknAIArchiveTestEntry("result.txt", committed).SHA256, Size: int64(len(committed) + 1)}, body: committed},
	} {
		t.Run(test.name, func(t *testing.T) {
			manifest := krknAIArchiveManifestJSON(t, test.manifest)
			handler := newKrknAIArchiveTestHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v1/runs/archive-uid/results":
					_, _ = w.Write(manifest)
				case "/v1/runs/archive-uid/files/result.txt":
					_, _ = w.Write(test.body)
				default:
					http.NotFound(w, r)
				}
			}))
			response := httptest.NewRecorder()
			handler.KrknAIRouter(response, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/archive-run/results/download", ""))
			if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), `"artifact_updating"`) {
				t.Fatalf("snapshot mismatch status/body = %d/%s", response.Code, response.Body.String())
			}
			if response.Header().Get("Content-Type") == "application/zip" || bytes.HasPrefix(response.Body.Bytes(), []byte("PK\x03\x04")) {
				t.Fatalf("snapshot mismatch returned ZIP data: headers=%v body=%q", response.Header(), response.Body.Bytes())
			}
		})
	}
}

func TestKrknAIRunArchiveDownloadRejectsUnsafeOrDuplicateMembers(t *testing.T) {
	validFile := krknAIArchiveTestEntry("safe/file.txt", []byte("ok"))
	for _, test := range []struct {
		name  string
		files []krknAIArchiveTestFile
	}{
		{name: "traversal", files: []krknAIArchiveTestFile{{Path: "../escape.txt", SHA256: validFile.SHA256, Size: validFile.Size}}},
		{name: "absolute", files: []krknAIArchiveTestFile{{Path: "/absolute.txt", SHA256: validFile.SHA256, Size: validFile.Size}}},
		{name: "backslash", files: []krknAIArchiveTestFile{{Path: `nested\\escape.txt`, SHA256: validFile.SHA256, Size: validFile.Size}}},
		{name: "duplicate", files: []krknAIArchiveTestFile{validFile, validFile}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var fileReads int
			manifest := krknAIArchiveManifestJSON(t, test.files...)
			handler := newKrknAIArchiveTestHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/v1/runs/archive-uid/results" {
					_, _ = w.Write(manifest)
					return
				}
				fileReads++
				_, _ = w.Write([]byte("ok"))
			}))
			response := httptest.NewRecorder()
			handler.KrknAIRouter(response, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/archive-run/results/download", ""))
			if response.Code != http.StatusBadGateway || fileReads != 0 {
				t.Fatalf("invalid manifest status/file reads = %d/%d: %s", response.Code, fileReads, response.Body.String())
			}
		})
	}
}

func TestKrknAIRunArchiveDownloadPropagatesMissingManifest(t *testing.T) {
	handler := newKrknAIArchiveTestHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"error":"not_found","message":"manifest not found"}`)
	}))
	response := httptest.NewRecorder()
	handler.KrknAIRouter(response, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/archive-run/results/download", ""))
	if response.Code != http.StatusNotFound || response.Header().Get("Content-Type") != "application/json" || strings.Contains(response.Header().Get("Content-Type"), "zip") {
		t.Fatalf("missing manifest was not propagated as an error: %d %v %s", response.Code, response.Header(), response.Body.String())
	}
}
