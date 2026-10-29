package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestKrknAIRunDeletionRemainsHiddenWhileFinalizing(t *testing.T) {
	serviceCalls := 0
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serviceCalls++
		http.Error(w, "deleting runs must not read artifacts", http.StatusInternalServerError)
	}))
	defer service.Close()
	handler := newKrknAITestHandler(t, service.URL)
	run := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "deleting-run", Namespace: "default", UID: "deleting-uid",
			Finalizers: []string{"test.krkn.dev/hold-cleanup"},
		},
		Status: krknv1alpha1.KrknAIRunStatus{Phase: "Succeeded"},
	}
	other := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "other-run", Namespace: "default", UID: "other-uid"},
		Status:     krknv1alpha1.KrknAIRunStatus{Phase: "Running"},
	}
	for _, resource := range []client.Object{run, other} {
		if err := handler.client.Create(context.Background(), resource); err != nil {
			t.Fatal(err)
		}
	}
	for attempt := range 2 {
		response := httptest.NewRecorder()
		handler.KrknAIRouter(response, adminKrknAIRequest(http.MethodDelete, KrknAIPath+"/runs/"+run.Name, ""))
		if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
			t.Fatalf("delete attempt %d: status=%d body=%s", attempt, response.Code, response.Body.String())
		}
	}
	var terminating krknv1alpha1.KrknAIRun
	if err := handler.client.Get(context.Background(), client.ObjectKeyFromObject(run), &terminating); err != nil {
		t.Fatal(err)
	}
	if terminating.DeletionTimestamp.IsZero() || len(terminating.Finalizers) != 1 {
		t.Fatalf("cleanup must remain pending in Kubernetes: %+v", terminating.ObjectMeta)
	}

	list := httptest.NewRecorder()
	handler.KrknAIRouter(list, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs", ""))
	var visible []krknv1alpha1.KrknAIRun
	if list.Code != http.StatusOK || json.Unmarshal(list.Body.Bytes(), &visible) != nil {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}
	if len(visible) != 1 || visible[0].UID != other.UID {
		t.Fatalf("refresh resurrected a deleting run or hid another run: %+v", visible)
	}
	for _, suffix := range []string{
		"", "/results", "/results/summary", "/results/scenarios",
		"/results/scenarios/0/1", "/results/download", "/files/scenario.log",
	} {
		t.Run("hidden"+suffix, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.KrknAIRouter(response, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/"+run.Name+suffix, ""))
			if response.Code != http.StatusNotFound {
				t.Fatalf("deleting run read returned %d: %s", response.Code, response.Body.String())
			}
		})
	}
	if serviceCalls != 0 {
		t.Fatalf("deleting run accessed artifact service %d times", serviceCalls)
	}
	otherResponse := httptest.NewRecorder()
	handler.KrknAIRouter(otherResponse, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/"+other.Name, ""))
	if otherResponse.Code != http.StatusOK {
		t.Fatalf("unrelated run is inaccessible: %d %s", otherResponse.Code, otherResponse.Body.String())
	}
}

func TestKrknAIRunDeletedArtifactsKeepConsoleNotFoundContract(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
		_, _ = w.Write([]byte(`{"detail":"run has been deleted"}`))
	}))
	defer service.Close()
	handler := newKrknAITestHandler(t, service.URL)
	run := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "stale-cache-run", Namespace: "default", UID: "stale-uid"},
	}
	if err := handler.client.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{
		"/results", "/results/summary", "/results/scenarios",
		"/results/scenarios/0/1", "/results/download", "/files/scenario.log",
	} {
		t.Run(suffix, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.KrknAIRouter(response, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/"+run.Name+suffix, ""))
			var failure ErrorResponse
			if response.Code != http.StatusNotFound || json.Unmarshal(response.Body.Bytes(), &failure) != nil {
				t.Fatalf("deleted artifact status=%d body=%s", response.Code, response.Body.String())
			}
			if failure.Error != "not_found" {
				t.Fatalf("deleted artifact response=%+v", failure)
			}
			if response.Header().Get("Content-Disposition") != "" {
				t.Fatal("deleted artifact response must not trigger a file download")
			}
		})
	}
}
