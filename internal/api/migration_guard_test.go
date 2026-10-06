package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/krkn-chaos/krkn-operator/internal/crdmigration"
)

func TestRunMutationsAreUnavailableWhileCRDMigrationGuardExists(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	guard := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: crdmigration.GuardConfigMapName, Namespace: "operator"}}
	handler := &Handler{client: fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(guard).Build(), namespace: "operator"}

	for _, endpoint := range []struct {
		name string
		run  func(http.ResponseWriter, *http.Request)
	}{
		{name: "scenario", run: handler.PostScenarioRun},
		{name: "graph", run: handler.CreateGraphRun},
		{name: "scenario delete", run: handler.DeleteScenarioRun},
		{name: "scenario delete complete", run: handler.DeleteScenarioRunComplete},
		{name: "single job cancellation", run: handler.DeleteSingleJob},
		{name: "graph delete", run: handler.DeleteGraphRun},
		{name: "category association", run: func(w http.ResponseWriter, r *http.Request) {
			handler.AssociateCategoryEntity(w, r, "category", "scenario-runs", "run")
		}},
	} {
		t.Run(endpoint.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
			endpoint.run(response, request)
			if response.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want 503: %s", response.Code, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), "CRD migration") {
				t.Fatalf("response does not explain migration pause: %s", response.Body.String())
			}
		})
	}
}

func TestRunMutationGuardFailsClosedWhenKubernetesReadFails(t *testing.T) {
	clientset := k8sfake.NewSimpleClientset()
	clientset.PrependReactor("get", "configmaps", func(ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("temporary API server failure")
	})
	handler := &Handler{clientset: clientset, namespace: "operator"}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	handler.PostScenarioRun(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503: %s", response.Code, response.Body.String())
	}
}
