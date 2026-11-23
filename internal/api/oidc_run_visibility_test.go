package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestOIDCMultiClusterRunShowsOnlyAuthorizedJobs(t *testing.T) {
	const namespace = "krkn-operator-system"
	const visibleURL = "https://visible.example"
	const hiddenURL = "https://hidden.example"
	scheme := runtime.NewScheme()
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))
	run := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "multi-run", Namespace: namespace},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetClusters: map[string][]string{"provider": {"visible", "hidden"}},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{
			TotalTargets: 2, SuccessfulJobs: 1, FailedJobs: 1,
			ClusterJobs: []krknv1alpha1.ClusterJobStatus{
				{ProviderName: "provider", ClusterName: "visible", ClusterAPIURL: visibleURL, JobID: "job-visible", Phase: "Succeeded"},
				{ProviderName: "provider", ClusterName: "hidden", ClusterAPIURL: hiddenURL, JobID: "job-hidden", Phase: "Failed"},
			},
			ResiliencyScores: []krknv1alpha1.ClusterResiliencyScore{
				{ProviderName: "provider", ClusterName: "visible", Score: 90},
				{ProviderName: "provider", ClusterName: "hidden", Score: 10},
			},
		},
	}
	group := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "keycloak-users", Namespace: namespace},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			ExternalGroups:     []krknv1alpha1.ExternalGroupMapping{{Provider: "local-keycloak", Group: "/krkn-ops"}},
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{visibleURL: {Actions: []string{"view"}}},
		},
	}
	h := &Handler{client: fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(run, group).Build(), namespace: namespace}
	claims := &auth.Claims{UserID: "oidc-user", Role: "user", Provider: "local-keycloak", Groups: []string{"/krkn-ops"}}
	ctx := context.WithValue(context.Background(), auth.UserClaimsKey, claims)

	filtered := h.FilterScenarioRunsByGroupPermission([]krknv1alpha1.KrknScenarioRun{*run}, ctx)
	require.Len(t, filtered, 1)
	require.Len(t, filtered[0].Status.ClusterJobs, 1)
	require.Equal(t, "job-visible", filtered[0].Status.ClusterJobs[0].JobID)
	require.Equal(t, 1, filtered[0].Status.TotalTargets)
	require.Equal(t, 1, filtered[0].Status.SuccessfulJobs)
	require.Zero(t, filtered[0].Status.FailedJobs)
	require.Len(t, filtered[0].Status.ResiliencyScores, 1)
	require.Equal(t, "visible", filtered[0].Status.ResiliencyScores[0].ClusterName)
	require.Len(t, run.Status.ClusterJobs, 2, "projection must not mutate the stored run")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/run/multi-run", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.GetScenarioRunStatus(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var response ScenarioRunStatusResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Len(t, response.ClusterJobs, 1)
	require.Equal(t, "job-visible", response.ClusterJobs[0].JobID)
	require.Equal(t, 1, response.TotalTargets)
	require.Zero(t, response.FailedJobs)
	require.Len(t, response.ResiliencyScores, 1)
}
