package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	v2 "github.com/krkn-chaos/krkn-operator/internal/api/v2"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestQueryResiliencyHistoryAggregatesSelectedCategoriesAndClusters(t *testing.T) {
	categoryOne := publicHistoryCategory("category-one")
	categoryTwo := publicHistoryCategory("category-two")
	labelOne := krknv1alpha1.CategoryEntityLabelPrefix + categoryOne.Name
	labelTwo := krknv1alpha1.CategoryEntityLabelPrefix + categoryTwo.Name

	sharedRun := scenarioRunConfigurationFixture()
	sharedRun.Name = "shared-run"
	sharedRun.Namespace = "default"
	sharedRun.Labels = map[string]string{labelOne: "true", labelTwo: "true"}
	sharedRun.CreationTimestamp = metav1.NewTime(time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC))
	sharedRun.Status.ClusterJobs = []krknv1alpha1.ClusterJobStatus{
		{ClusterName: "cluster-a", ProviderName: "provider-a"},
		{ClusterName: "cluster-b", ProviderName: "provider-b"},
	}
	sharedRun.Status.ResiliencyScores = []krknv1alpha1.ClusterResiliencyScore{
		{ClusterName: "cluster-a", Score: 75, Status: "calculated"},
		{ClusterName: "cluster-b", Score: 25, Status: "calculated"},
	}

	olderRun := scenarioRunConfigurationFixture()
	olderRun.Name = "older-run"
	olderRun.Namespace = "default"
	olderRun.Labels = map[string]string{labelTwo: "true"}
	olderRun.CreationTimestamp = metav1.NewTime(time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC))
	olderRun.Status.ClusterJobs = []krknv1alpha1.ClusterJobStatus{{ClusterName: "cluster-a", ProviderName: "provider-a"}}
	olderRun.Status.ResiliencyScores = []krknv1alpha1.ClusterResiliencyScore{{ClusterName: "cluster-a", Score: 0, Status: "calculated"}}

	graphRun := graphRunConfigurationFixture()
	graphRun.Name = "graph-run"
	graphRun.Namespace = "default"
	graphRun.Labels = map[string]string{labelOne: "true"}
	graphRun.CreationTimestamp = metav1.NewTime(time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC))
	graphRun.Status.ResiliencyScores = []krknv1alpha1.GraphClusterScore{{
		ClusterName: "cluster-a", ProviderName: "provider-a", Calculated: 88, Status: "pass",
	}}

	noScoreRun := scenarioRunConfigurationFixture()
	noScoreRun.Name = "no-score-run"
	noScoreRun.Namespace = "default"
	noScoreRun.Labels = map[string]string{labelOne: "true"}
	noScoreRun.Status.ResiliencyScores = []krknv1alpha1.ClusterResiliencyScore{{ClusterName: "cluster-a", Score: 99, Status: "error"}}
	otherClusterRun := scenarioRunConfigurationFixture()
	otherClusterRun.Name = "unselected-cluster-run"
	otherClusterRun.Namespace = "default"
	otherClusterRun.Labels = map[string]string{labelOne: "true"}
	otherClusterRun.Spec.Environment["POD_COUNT"] = "99"
	otherClusterRun.Status.ResiliencyScores = []krknv1alpha1.ClusterResiliencyScore{{ClusterName: "cluster-b", Score: 44, Status: "calculated"}}

	handler, _ := newCategoryTestHandler(t, categoryOne, categoryTwo, sharedRun, olderRun, graphRun, noScoreRun, otherClusterRun)
	countingClient := &historyQueryCountingClient{Client: handler.client}
	handler.client = countingClient
	response := httptest.NewRecorder()
	handler.QueryResiliencyHistory(response, newCategoryRequest(http.MethodPost, v2.ResiliencyHistoryPath,
		`{"categories":["category-two","category-one","category-one"],"clusters":["cluster-a","cluster-a"]}`, "admin"))
	if response.Code != http.StatusOK {
		t.Fatalf("history status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}

	var history ResiliencyHistoryQueryResponse
	if err := json.Unmarshal(response.Body.Bytes(), &history); err != nil {
		t.Fatalf("decode history response: %v", err)
	}
	if len(history.Clusters) != 1 {
		t.Fatalf("clusters = %+v, want only selected cluster-a", history.Clusters)
	}
	clusterHistory := history.Clusters["cluster-a"]
	if len(clusterHistory) != 2 {
		t.Fatalf("cluster categories = %+v, want both selected categories", clusterHistory)
	}
	categoryOnePoints := clusterHistory["category-one"]
	if len(categoryOnePoints) != 2 || categoryOnePoints[0].RunID != "shared-run" || categoryOnePoints[1].RunID != "graph-run" {
		t.Fatalf("category-one datapoints = %+v, want scored scenario and graph runs sorted by date", categoryOnePoints)
	}
	categoryTwoPoints := clusterHistory["category-two"]
	if len(categoryTwoPoints) != 2 || categoryTwoPoints[0].RunID != "older-run" || categoryTwoPoints[1].RunID != "shared-run" {
		t.Fatalf("category-two datapoints = %+v, want its two scored runs sorted by date", categoryTwoPoints)
	}
	if categoryTwoPoints[0].Score != 0 {
		t.Fatalf("zero score was lost: %+v", categoryTwoPoints[0])
	}
	if len(history.ConfigurationGroups["category-one"]) != 2 || len(history.ConfigurationGroups["category-two"]) != 1 {
		t.Fatalf("configuration groups are not category-scoped: %+v", history.ConfigurationGroups)
	}
	if countingClient.listCalls != 2 {
		t.Fatalf("Kubernetes list calls = %d, want one per run type for a multi-category query", countingClient.listCalls)
	}
	for _, categoryGroups := range history.ConfigurationGroups {
		for groupID, group := range categoryGroups {
			found := false
			for _, categories := range history.Clusters {
				for _, points := range categories {
					for _, point := range points {
						if point.ConfigurationGroupID == groupID && point.RunType == group.RunType {
							found = true
						}
					}
				}
			}
			if !found {
				t.Errorf("configuration group %q has no selected datapoint", groupID)
			}
		}
	}
}

type historyQueryCountingClient struct {
	client.Client
	listCalls int
}

func (c *historyQueryCountingClient) List(ctx context.Context, list client.ObjectList, options ...client.ListOption) error {
	c.listCalls++
	return c.Client.List(ctx, list, options...)
}

func TestQueryResiliencyHistoryEnforcesCategoryAndClusterVisibility(t *testing.T) {
	const categoryName = "team-history"
	const categoryTeam = "category-team"
	const clusterViewTeam = "cluster-a-viewers"
	categoryLabel := krknv1alpha1.CategoryEntityLabelPrefix + categoryName
	clusterAURL := "https://cluster-a.example.com:6443"
	clusterBURL := "https://cluster-b.example.com:6443"
	category := &krknv1alpha1.KrknCategory{ObjectMeta: metav1.ObjectMeta{
		Name: categoryName, Namespace: "default", Labels: map[string]string{groupauth.GroupLabelKey(categoryTeam): "true"},
	}}
	clusterGroup := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{Name: clusterViewTeam, Namespace: "default"},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: clusterViewTeam,
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				clusterAURL: {Actions: []string{"view"}},
			},
		},
	}
	user := newCategoryTestUser(t, "category-user@example.com", categoryTeam, clusterViewTeam)
	run := scenarioRunConfigurationFixture()
	run.Name = "multi-cluster-run"
	run.Namespace = "default"
	run.Labels = map[string]string{categoryLabel: "true"}
	run.Status.ClusterJobs = []krknv1alpha1.ClusterJobStatus{
		{ClusterName: "cluster-a", ClusterAPIURL: clusterAURL},
		{ClusterName: "cluster-b", ClusterAPIURL: clusterBURL},
	}
	run.Status.ResiliencyScores = []krknv1alpha1.ClusterResiliencyScore{
		{ClusterName: "cluster-a", Score: 91, Status: "calculated"},
		{ClusterName: "cluster-b", Score: 12, Status: "calculated"},
	}
	privateCategory := &krknv1alpha1.KrknCategory{ObjectMeta: metav1.ObjectMeta{
		Name: "private-category", Namespace: "default",
		Labels: map[string]string{groupauth.GroupLabelKey("private-team"): "true"},
	}}
	handler, _ := newCategoryTestHandler(t, category, privateCategory, newCategoryTestGroup(categoryTeam),
		newCategoryTestGroup("private-team"), clusterGroup, user, run)

	response := httptest.NewRecorder()
	handler.QueryResiliencyHistory(response, newCategoryRequest(http.MethodPost, v2.ResiliencyHistoryPath,
		`{"categories":["team-history"],"clusters":["cluster-a","cluster-b"]}`, "user"))
	if response.Code != http.StatusOK {
		t.Fatalf("visible history status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	var history ResiliencyHistoryQueryResponse
	if err := json.Unmarshal(response.Body.Bytes(), &history); err != nil {
		t.Fatalf("decode history response: %v", err)
	}
	if len(history.Clusters) != 1 || len(history.Clusters["cluster-a"][categoryName]) != 1 {
		t.Fatalf("history should include only visible cluster-a score: %+v", history.Clusters)
	}
	if _, leaked := history.Clusters["cluster-b"]; leaked {
		t.Fatalf("history leaked cluster-b score: %+v", history.Clusters["cluster-b"])
	}

	privateResponse := httptest.NewRecorder()
	handler.QueryResiliencyHistory(privateResponse, newCategoryRequest(http.MethodPost, v2.ResiliencyHistoryPath,
		`{"categories":["private-category"],"clusters":["cluster-a"]}`, "user"))
	if privateResponse.Code != http.StatusForbidden {
		t.Fatalf("private category status = %d, want forbidden: %s", privateResponse.Code, privateResponse.Body.String())
	}
}

func TestQueryResiliencyHistoryValidatesMethodAuthenticationAndRequest(t *testing.T) {
	privateCategory := &krknv1alpha1.KrknCategory{ObjectMeta: metav1.ObjectMeta{
		Name: "private-category", Namespace: "default",
		Labels: map[string]string{groupauth.GroupLabelKey("private-team"): "true"},
	}}
	handler, _ := newCategoryTestHandler(t, privateCategory, newCategoryTestGroup("private-team"))
	tests := []struct {
		name       string
		request    *http.Request
		wantStatus int
	}{
		{name: "authentication required", request: httptest.NewRequest(http.MethodPost, v2.ResiliencyHistoryPath, nil), wantStatus: http.StatusUnauthorized},
		{name: "method not allowed", request: newCategoryRequest(http.MethodGet, v2.ResiliencyHistoryPath, "", "admin"), wantStatus: http.StatusMethodNotAllowed},
		{name: "malformed JSON", request: newCategoryRequest(http.MethodPost, v2.ResiliencyHistoryPath, `{`, "admin"), wantStatus: http.StatusBadRequest},
		{name: "trailing JSON value", request: newCategoryRequest(http.MethodPost, v2.ResiliencyHistoryPath, `{"categories":["private-category"],"clusters":["cluster-a"]}{}`, "admin"), wantStatus: http.StatusBadRequest},
		{name: "unknown field", request: newCategoryRequest(http.MethodPost, v2.ResiliencyHistoryPath, `{"categories":["private-category"],"clusters":["cluster-a"],"extra":true}`, "admin"), wantStatus: http.StatusBadRequest},
		{name: "empty categories", request: newCategoryRequest(http.MethodPost, v2.ResiliencyHistoryPath, `{"categories":[],"clusters":["cluster-a"]}`, "admin"), wantStatus: http.StatusBadRequest},
		{name: "empty clusters", request: newCategoryRequest(http.MethodPost, v2.ResiliencyHistoryPath, `{"categories":["private-category"],"clusters":[]}`, "admin"), wantStatus: http.StatusBadRequest},
		{name: "blank cluster", request: newCategoryRequest(http.MethodPost, v2.ResiliencyHistoryPath, `{"categories":["private-category"],"clusters":[" "]}`, "admin"), wantStatus: http.StatusBadRequest},
		{name: "invalid category", request: newCategoryRequest(http.MethodPost, v2.ResiliencyHistoryPath, `{"categories":["bad category"],"clusters":["cluster-a"]}`, "admin"), wantStatus: http.StatusBadRequest},
		{name: "missing category", request: newCategoryRequest(http.MethodPost, v2.ResiliencyHistoryPath, `{"categories":["missing-category"],"clusters":["cluster-a"]}`, "admin"), wantStatus: http.StatusNotFound},
		{name: "private category denied", request: newCategoryRequest(http.MethodPost, v2.ResiliencyHistoryPath, `{"categories":["private-category"],"clusters":["cluster-a"]}`, "user"), wantStatus: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.QueryResiliencyHistory(response, test.request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, test.wantStatus, response.Body.String())
			}
			if test.name == "method not allowed" && response.Header().Get("Allow") != http.MethodPost {
				t.Fatalf("Allow header = %q, want POST", response.Header().Get("Allow"))
			}
		})
	}
}

func publicHistoryCategory(name string) *krknv1alpha1.KrknCategory {
	return &krknv1alpha1.KrknCategory{ObjectMeta: metav1.ObjectMeta{
		Name: name, Namespace: "default", Labels: map[string]string{categoryAvailableToAllLabel: "true"},
	}}
}
