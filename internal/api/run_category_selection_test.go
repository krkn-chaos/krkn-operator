package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

func runCategoryFixture(name, group string) *krknv1alpha1.KrknCategory {
	labels := map[string]string{}
	if group == "" {
		labels[categoryAvailableToAllLabel] = "true"
	} else {
		labels[groupauth.GroupLabelKey(group)] = "true"
	}
	return &krknv1alpha1.KrknCategory{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", Labels: labels},
		Spec:       krknv1alpha1.KrknCategorySpec{Color: "#123456"},
	}
}

func TestPostScenarioRun_CategoriesArePersistedOnCreatedCR(t *testing.T) {
	const targetRequestID = "category-target"
	handler := setupScenarioRunTestHandler(targetRequestID, map[string]string{
		"test-cluster": "YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmNsdXN0ZXJzOiBbXQpjb250ZXh0czogW10KdXNlcnM6IFtd",
	})
	for _, name := range []string{"network", "reliability"} {
		require.NoError(t, handler.client.Create(t.Context(), runCategoryFixture(name, "")))
	}

	requestBody := `{"targetRequestId":"category-target","targetClusters":{"krkn-operator":["test-cluster"]},"scenario":{"name":"pod-delete","private":false},"categories":["network","reliability"]}`
	request := newCategoryRequestAs(http.MethodPost, ScenariosRunPath, requestBody, "admin", "admin@example.com")
	response := httptest.NewRecorder()
	handler.PostScenarioRun(response, request)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	var created ScenarioRunCreateResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &created))
	run := &krknv1alpha1.KrknScenarioRun{}
	require.NoError(t, handler.client.Get(request.Context(), client.ObjectKey{Name: created.ScenarioRunName, Namespace: "default"}, run))
	require.Equal(t, "true", run.Labels[krknv1alpha1.CategoryEntityLabelPrefix+"network"])
	require.Equal(t, "true", run.Labels[krknv1alpha1.CategoryEntityLabelPrefix+"reliability"])
	require.Equal(t, "pod-delete", run.Spec.Scenario.Name)
}

func TestPostScenarioRun_RejectsInaccessibleCategoryBeforeCreatingCR(t *testing.T) {
	const (
		targetRequestID = "category-target"
		userID          = "runner@example.com"
		runGroup        = "scenario-runners"
		privateGroup    = "private-categories"
	)
	handler := setupScenarioRunTestHandler(targetRequestID, map[string]string{
		"test-cluster": "YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmNsdXN0ZXJzOiBbXQpjb250ZXh0czogW10KdXNlcnM6IFtd",
	})
	group := newCategoryTestGroup(runGroup)
	group.Spec.ClusterPermissions = map[string]krknv1alpha1.ClusterPermissionSet{
		"https://test-cluster.example.com:6443": {Actions: []string{"run"}},
	}
	user := newCategoryTestUser(t, userID, runGroup)
	require.NoError(t, handler.client.Create(t.Context(), group))
	require.NoError(t, handler.client.Create(t.Context(), user))
	require.NoError(t, handler.client.Create(t.Context(), runCategoryFixture("private", privateGroup)))

	requestBody := `{"targetRequestId":"category-target","targetClusters":{"krkn-operator":["test-cluster"]},"scenario":{"name":"pod-delete","private":false},"categories":["private"]}`
	request := newCategoryRequestAs(http.MethodPost, ScenariosRunPath, requestBody, "user", userID)
	response := httptest.NewRecorder()
	handler.PostScenarioRun(response, request)
	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())

	runs := &krknv1alpha1.KrknScenarioRunList{}
	require.NoError(t, handler.client.List(request.Context(), runs, client.InNamespace("default")))
	require.Empty(t, runs.Items, "a category authorization failure must not leave a runnable CR")
}

func TestPostScenarioRun_RejectsInvalidCategoryNameBeforeCreatingCR(t *testing.T) {
	handler := setupScenarioRunTestHandler("category-target", map[string]string{
		"test-cluster": "YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmNsdXN0ZXJzOiBbXQpjb250ZXh0czogW10KdXNlcnM6IFtd",
	})
	requestBody := `{"targetRequestId":"category-target","targetClusters":{"krkn-operator":["test-cluster"]},"scenario":{"name":"pod-delete","private":false},"categories":["invalid category"]}`
	request := newCategoryRequestAs(http.MethodPost, ScenariosRunPath, requestBody, "admin", "admin@example.com")
	response := httptest.NewRecorder()
	handler.PostScenarioRun(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())

	runs := &krknv1alpha1.KrknScenarioRunList{}
	require.NoError(t, handler.client.List(request.Context(), runs, client.InNamespace("default")))
	require.Empty(t, runs.Items, "an invalid category name must not leave a runnable CR")
}

func graphRunCategoryTestHandler(t *testing.T, objects ...client.Object) *Handler {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	target := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "graph-category-target", Namespace: "default"},
		Spec:       krknv1alpha1.KrknTargetRequestSpec{UUID: "graph-category-target"},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status: "Completed",
			TargetData: map[string][]krknv1alpha1.ClusterTarget{
				"krkn-operator": {{ClusterName: "test-cluster", ClusterAPIURL: "https://test-cluster.example.com:6443"}},
			},
		},
	}
	initial := append([]client.Object{target}, objects...)
	return &Handler{
		client:    fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(initial...).Build(),
		namespace: "default",
	}
}

func TestCreateGraphRun_CategoriesArePersistedOnCreatedCR(t *testing.T) {
	handler := graphRunCategoryTestHandler(t, runCategoryFixture("resilience", ""))
	requestBody, err := json.Marshal(GraphRunCreateRequest{
		Graph: map[string]krknv1alpha1.GraphScenarioNode{
			"node-1": {Scenario: publicScenarioReference("pod-delete")},
		},
		TargetRequestID: "graph-category-target",
		TargetClusters:  map[string][]string{"krkn-operator": {"test-cluster"}},
		Categories:      []string{"resilience"},
	})
	require.NoError(t, err)
	request := newCategoryRequestAs(http.MethodPost, GraphRunsPath, string(requestBody), "admin", "admin@example.com")
	response := httptest.NewRecorder()
	handler.CreateGraphRun(response, request)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	var created GraphRunDetailResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &created))
	run := &krknv1alpha1.KrknGraphRun{}
	require.NoError(t, handler.client.Get(request.Context(), client.ObjectKey{Name: created.Name, Namespace: "default"}, run))
	require.Equal(t, "true", run.Labels[krknv1alpha1.CategoryEntityLabelPrefix+"resilience"])
}

func TestCreateGraphRun_RejectsMissingCategoryBeforeCreatingCR(t *testing.T) {
	handler := graphRunCategoryTestHandler(t)
	requestBody, err := json.Marshal(GraphRunCreateRequest{
		Graph: map[string]krknv1alpha1.GraphScenarioNode{
			"node-1": {Scenario: publicScenarioReference("pod-delete")},
		},
		TargetRequestID: "graph-category-target",
		TargetClusters:  map[string][]string{"krkn-operator": {"test-cluster"}},
		Categories:      []string{"does-not-exist"},
	})
	require.NoError(t, err)
	request := newCategoryRequestAs(http.MethodPost, GraphRunsPath, string(requestBody), "admin", "admin@example.com")
	response := httptest.NewRecorder()
	handler.CreateGraphRun(response, request)
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())

	runs := &krknv1alpha1.KrknGraphRunList{}
	require.NoError(t, handler.client.List(request.Context(), runs, client.InNamespace("default")))
	require.Empty(t, runs.Items, "an invalid category must not leave a runnable CR")
}

func TestReconstructScenarioRunPayload_PreservesOnlyVisibleCategories(t *testing.T) {
	const userID = "replay-user@example.com"
	visibleCategory := runCategoryFixture("network", "")
	hiddenCategory := runCategoryFixture("private", "private-team")
	user := newCategoryTestUser(t, userID)
	handler, scheme := newCategoryTestHandler(t, visibleCategory, hiddenCategory, user)
	require.NotNil(t, scheme)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(withTestClaims(request.Context(), userID, "user"))

	run := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
			krknv1alpha1.CategoryEntityLabelPrefix + "network":  "true",
			krknv1alpha1.CategoryEntityLabelPrefix + "private":  "true",
			krknv1alpha1.CategoryEntityLabelPrefix + "deleted":  "true",
			krknv1alpha1.CategoryEntityLabelPrefix + "unmarked": "false",
		}},
		Spec: krknv1alpha1.KrknScenarioRunSpec{Scenario: publicScenarioReference("pod-delete")},
	}
	payload, err := handler.reconstructScenarioRunPayload(request.Context(), run)
	require.NoError(t, err)
	require.Equal(t, []string{"network"}, payload.Categories)
}

func withTestClaims(ctx context.Context, userID, role string) context.Context {
	return context.WithValue(ctx, auth.UserClaimsKey, &auth.Claims{UserID: userID, Role: role})
}
