package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestKrknAICancellationRetainsRunAndRespectsTerminalState(t *testing.T) {
	for _, test := range []struct {
		phase      string
		intent     bool
		wantCode   int
		wantIntent bool
	}{
		{"", false, http.StatusAccepted, true},
		{"Pending", false, http.StatusAccepted, true},
		{"Provisioning", false, http.StatusAccepted, true},
		{"Running", false, http.StatusAccepted, true},
		{"Cancelling", true, http.StatusAccepted, true},
		{"Cancelled", true, http.StatusOK, true},
		{"Succeeded", false, http.StatusConflict, false},
		{"Failed", false, http.StatusConflict, false},
	} {
		t.Run(test.phase, func(t *testing.T) {
			handler := newKrknAITestHandler(t, "http://unused")
			run := &krknv1alpha1.KrknAIRun{
				ObjectMeta: metav1.ObjectMeta{Name: "retained", Namespace: "default", UID: "retained-uid"},
				Spec:       krknv1alpha1.KrknAIRunSpec{CancelRequested: test.intent},
				Status:     krknv1alpha1.KrknAIRunStatus{Phase: test.phase},
			}
			if err := handler.client.Create(context.Background(), run); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				response := httptest.NewRecorder()
				handler.KrknAIRouter(response, adminKrknAIRequest(http.MethodPost, KrknAIPath+"/runs/retained/cancel", ""))
				if response.Code != test.wantCode {
					t.Fatalf("cancel status=%d body=%s", response.Code, response.Body.String())
				}
			}
			var retained krknv1alpha1.KrknAIRun
			if err := handler.client.Get(context.Background(), client.ObjectKeyFromObject(run), &retained); err != nil {
				t.Fatal(err)
			}
			if retained.UID != run.UID || !retained.DeletionTimestamp.IsZero() || retained.Spec.CancelRequested != test.wantIntent || retained.Status.Phase != test.phase {
				t.Fatalf("cancellation deleted or misclassified the run: %+v", retained)
			}
			response := httptest.NewRecorder()
			handler.KrknAIRouter(response, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/retained", ""))
			if response.Code != http.StatusOK {
				t.Fatalf("retained run is not observable: %d", response.Code)
			}
		})
	}
}

func TestKrknAICancellationRequiresCancelPermission(t *testing.T) {
	handler := newKrknAITestHandler(t, "http://unused")
	run := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "protected", Namespace: "default", UID: "protected-uid"},
		Spec:       krknv1alpha1.KrknAIRunSpec{TargetRequestID: "target", TargetClusters: map[string][]string{"provider": {"cluster"}}},
		Status:     krknv1alpha1.KrknAIRunStatus{Phase: "Running"},
	}
	if err := handler.client.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.KrknAIRouter(response, userKrknAIRequest(http.MethodPost, KrknAIPath+"/runs/protected/cancel", "", "admin@example.com"))
	if response.Code != http.StatusForbidden {
		t.Fatalf("unauthorized cancel status=%d body=%s", response.Code, response.Body.String())
	}
	if err := handler.client.Get(context.Background(), client.ObjectKeyFromObject(run), run); err != nil {
		t.Fatal(err)
	}
	if run.Spec.CancelRequested {
		t.Fatal("denied cancellation changed execution intent")
	}
}

type completeAIRunOnCancelClient struct{ client.Client }

func (c completeAIRunOnCancelClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if requested, ok := obj.(*krknv1alpha1.KrknAIRun); ok && requested.Spec.CancelRequested {
		var completed krknv1alpha1.KrknAIRun
		if err := c.Client.Get(ctx, client.ObjectKeyFromObject(requested), &completed); err != nil {
			return err
		}
		completed.Status.Phase = "Succeeded"
		if err := c.Client.Update(ctx, &completed); err != nil {
			return err
		}
		return apierrors.NewConflict(schema.GroupResource{Group: "krkn.krkn-chaos.dev", Resource: "krknairuns"}, requested.Name, nil)
	}
	return c.Client.Update(ctx, obj, opts...)
}

func TestKrknAICancellationDoesNotOverwriteConcurrentCompletion(t *testing.T) {
	handler := newKrknAITestHandler(t, "http://unused")
	run := &krknv1alpha1.KrknAIRun{ObjectMeta: metav1.ObjectMeta{Name: "race", Namespace: "default"}, Status: krknv1alpha1.KrknAIRunStatus{Phase: "Running"}}
	if err := handler.client.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	handler.client = completeAIRunOnCancelClient{handler.client}
	response := httptest.NewRecorder()
	handler.KrknAIRouter(response, adminKrknAIRequest(http.MethodPost, KrknAIPath+"/runs/race/cancel", ""))
	if response.Code != http.StatusConflict {
		t.Fatalf("completion race status=%d body=%s", response.Code, response.Body.String())
	}
	if err := handler.client.Get(context.Background(), client.ObjectKeyFromObject(run), run); err != nil {
		t.Fatal(err)
	}
	if run.Status.Phase != "Succeeded" || run.Spec.CancelRequested {
		t.Fatalf("completion was overwritten: %+v", run)
	}
}

func TestKrknAICancelledGenerationRetainsRawButNotNormalizedScores(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := "provisional"
		if r.URL.Path == "/v1/runs/scores-uid/scenarios/0/1" {
			state = "final"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"generation": 1, "scenarioId": "1", "fitnessState": state,
			"fitnessResult": map[string]any{"fitnessScore": 0.8, "scores": []map[string]any{{"id": 0, "rawScore": 0, "normalizedScore": 0.8}}},
		})
	}))
	defer service.Close()
	handler := newKrknAITestHandler(t, service.URL)
	run := &krknv1alpha1.KrknAIRun{ObjectMeta: metav1.ObjectMeta{Name: "scores", Namespace: "default", UID: "scores-uid"}, Status: krknv1alpha1.KrknAIRunStatus{Phase: "Cancelled"}}
	if err := handler.client.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	for _, generation := range []string{"0", "1"} {
		response := httptest.NewRecorder()
		handler.KrknAIRouter(response, adminKrknAIRequest(http.MethodGet, KrknAIPath+"/runs/scores/results/scenarios/"+generation+"/1", ""))
		var detail KrknAIScenarioDetailResponse
		if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &detail) != nil {
			t.Fatalf("detail status=%d body=%s", response.Code, response.Body.String())
		}
		if len(detail.FitnessResult.Scores) != 1 || detail.FitnessResult.Scores[0].RawScore == nil || *detail.FitnessResult.Scores[0].RawScore != 0 {
			t.Fatalf("raw zero lost: %+v", detail)
		}
		if generation == "1" {
			if detail.FitnessState != "cancelled" || detail.FitnessResult.FitnessScore != nil || detail.FitnessResult.Scores[0].NormalizedScore != nil {
				t.Fatalf("cancelled generation exposed finalized values: %+v", detail)
			}
		} else if detail.FitnessState != "final" || detail.FitnessResult.FitnessScore == nil || detail.FitnessResult.Scores[0].NormalizedScore == nil {
			t.Fatalf("completed generation lost final values: %+v", detail)
		}
	}
}
