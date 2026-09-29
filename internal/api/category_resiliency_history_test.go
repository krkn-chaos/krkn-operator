package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	v2 "github.com/krkn-chaos/krkn-operator/internal/api/v2"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildCategoryResiliencyHistoryGroupsTypedConfigurationsAndSortsPerCluster(t *testing.T) {
	baseTime := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	scenarioBase := scenarioRunConfigurationFixture()
	scenarioEquivalent := scenarioBase.DeepCopy()
	scenarioEquivalent.Spec.Scenario.RegistryName = "registry-b"
	scenarioEquivalent.Spec.CloudCredentialRef = "credential-b"
	scenarioEquivalent.Spec.Files[0].FileID = "different-file-id"
	scenarioDifferent := scenarioBase.DeepCopy()
	scenarioDifferent.Spec.Environment["POD_COUNT"] = "3"

	graphBase := graphRunConfigurationFixture()
	graphNode := graphBase.Spec.Graph["node-a"]
	graphNode.Volumes["node-a-metrics-file"] = "/etc/metrics.yaml"
	graphBase.Spec.Graph["node-a"] = graphNode
	graphEquivalent := graphBase.DeepCopy()
	graphNode = graphEquivalent.Spec.Graph["node-a"]
	graphNode.Comment = "display-only comment"
	graphNode.Scenario.RegistryName = "registry-b"
	graphEquivalent.Spec.Graph["node-a"] = graphNode
	graphDifferent := graphBase.DeepCopy()
	graphNode = graphDifferent.Spec.Graph["node-b"]
	graphNode.Env["LATENCY_MS"] = "200"
	graphDifferent.Spec.Graph["node-b"] = graphNode
	graphReplay := graphBase.DeepCopy()
	graphReplayNodeA := graphReplay.Spec.Graph["node-a"]
	graphReplayNodeB := graphReplay.Spec.Graph["node-b"]
	graphReplayNodeB.DependsOn = stringPointer("replay-node-a")
	// Score collection mounts are omitted from the replay but must not split
	// the scenario configuration group.
	graphReplayNodeA.Volumes = map[string]string{"scenario-file": "/etc/scenario.yaml"}
	graphReplayNodeB.Volumes = nil
	graphReplay.Spec.Graph = map[string]krknv1alpha1.GraphScenarioNode{
		"replay-node-a": graphReplayNodeA,
		"replay-node-b": graphReplayNodeB,
	}

	runs := []categoryHistoryRun{
		scenarioHistoryRun("scenario-old", baseTime.Add(3*time.Hour), scenarioBase, []krknv1alpha1.ClusterResiliencyScore{
			{ClusterName: "cluster-a", Score: 70, Status: "calculated"},
			{ClusterName: "cluster-a", Score: 100, Status: "error"},
			{ClusterName: "cluster-b", Score: 0, Status: "calculated"},
			{Score: 50, Status: "calculated"}, // A score without a cluster cannot form a datapoint.
		}),
		scenarioHistoryRun("scenario-equivalent", baseTime.Add(time.Hour), scenarioEquivalent, []krknv1alpha1.ClusterResiliencyScore{
			{ClusterName: "cluster-a", Score: 80, Status: "calculated"},
		}),
		scenarioHistoryRun("scenario-different", baseTime.Add(2*time.Hour), scenarioDifferent, []krknv1alpha1.ClusterResiliencyScore{
			{ClusterName: "cluster-a", Score: 90, Status: "calculated"},
		}),
		graphHistoryRun("graph-first", baseTime, graphBase, []krknv1alpha1.GraphClusterScore{
			{ClusterName: "cluster-a", ProviderName: "provider-a", Calculated: 60, Status: "fail", Baseline: floatPointer(65)},
		}),
		graphHistoryRun("graph-equivalent", baseTime.Add(4*time.Hour), graphEquivalent, []krknv1alpha1.GraphClusterScore{
			{ClusterName: "cluster-b", ProviderName: "provider-b", Calculated: 65, Status: "pass"},
		}),
		graphHistoryRun("graph-different", baseTime.Add(5*time.Hour), graphDifferent, []krknv1alpha1.GraphClusterScore{
			{ClusterName: "cluster-a", ProviderName: "provider-a", Calculated: 0, Status: "no-baseline"},
		}),
		graphHistoryRun("graph-replay", baseTime.Add(6*time.Hour), graphReplay, []krknv1alpha1.GraphClusterScore{
			{ClusterName: "cluster-a", ProviderName: "provider-a", Calculated: 75, Status: "pass", Baseline: floatPointer(70)},
		}),
		scenarioHistoryRun("scenario-without-score", baseTime.Add(7*time.Hour), scenarioBase, []krknv1alpha1.ClusterResiliencyScore{
			{ClusterName: "cluster-a", Score: 100, Status: "error"},
		}),
		graphHistoryRun("graph-without-score", baseTime.Add(8*time.Hour), graphBase, []krknv1alpha1.GraphClusterScore{
			{ClusterName: "cluster-a", Calculated: -1, Status: "calculating"},
			{ClusterName: "cluster-b", Calculated: 80, Status: "error"},
			{ClusterName: "cluster-c", Calculated: 80, Status: "unknown-status"},
		}),
	}

	response := buildCategoryResiliencyHistory(runs)
	if len(response.ConfigurationGroups) != 4 {
		t.Fatalf("configuration group count = %d, want 4: %+v", len(response.ConfigurationGroups), response.ConfigurationGroups)
	}

	clusterA := response.Clusters["cluster-a"]
	wantRuns := []string{"graph-first", "scenario-equivalent", "scenario-different", "scenario-old", "graph-different", "graph-replay"}
	if got := datapointRunIDs(clusterA); !slices.Equal(got, wantRuns) {
		t.Fatalf("cluster-a run order = %v, want %v", got, wantRuns)
	}
	clusterB := response.Clusters["cluster-b"]
	if got, want := datapointRunIDs(clusterB), []string{"scenario-old", "graph-equivalent"}; !slices.Equal(got, want) {
		t.Fatalf("cluster-b run order = %v, want %v", got, want)
	}
	if len(response.Clusters) != 2 {
		t.Fatalf("cluster set = %v, want only scored clusters cluster-a and cluster-b", response.Clusters)
	}

	groupFor := func(cluster string, runID string) string {
		t.Helper()
		for _, point := range response.Clusters[cluster] {
			if point.RunID == runID {
				return point.ConfigurationGroupID
			}
		}
		t.Fatalf("no datapoint for %s/%s", cluster, runID)
		return ""
	}
	if got, want := groupFor("cluster-a", "scenario-old"), groupFor("cluster-b", "scenario-old"); got != want {
		t.Fatalf("one run received different group IDs by cluster: %q and %q", got, want)
	}
	if got, want := groupFor("cluster-a", "scenario-old"), "scenario-runs/scenario-equivalent"; got != want {
		t.Fatalf("equivalent scenario group ID = %q, want %q", got, want)
	}
	if got, want := groupFor("cluster-a", "graph-first"), groupFor("cluster-b", "graph-equivalent"); got != want {
		t.Fatalf("equivalent graph configurations have different group IDs: %q and %q", got, want)
	}
	if got, want := groupFor("cluster-a", "graph-first"), groupFor("cluster-a", "graph-replay"); got != want {
		t.Fatalf("graph replay with regenerated node IDs and omitted score mounts has configuration group %q, want shared group %q", want, got)
	}
	graphPoint := datapointForRunID(t, clusterA, "graph-first")
	if graphPoint.Baseline == nil || *graphPoint.Baseline != 65 {
		t.Fatalf("graph datapoint baseline = %v, want 65", graphPoint.Baseline)
	}
	if scenarioPoint := datapointForRunID(t, clusterA, "scenario-old"); scenarioPoint.Baseline != nil {
		t.Fatalf("scenario datapoint baseline = %v, want omitted", *scenarioPoint.Baseline)
	}
	if groupFor("cluster-a", "scenario-different") == groupFor("cluster-a", "scenario-old") {
		t.Fatal("different scenario parameters unexpectedly share a configuration group")
	}
	if groupFor("cluster-a", "graph-different") == groupFor("cluster-a", "graph-first") {
		t.Fatal("different graph node parameters unexpectedly share a configuration group")
	}
	for _, group := range response.ConfigurationGroups {
		if group.RepresentativeRunID == "scenario-without-score" || group.RepresentativeRunID == "graph-without-score" {
			t.Fatalf("a run without a score created a configuration group: %+v", group)
		}
	}
}

func TestScenarioHistoryScoresOnlyKeepsCalculatedClusterScores(t *testing.T) {
	run := scenarioRunConfigurationFixture()
	run.Status.ClusterJobs = []krknv1alpha1.ClusterJobStatus{
		{ClusterName: "cluster-a", ProviderName: "provider-a"},
	}
	run.Status.ResiliencyScores = []krknv1alpha1.ClusterResiliencyScore{
		{ClusterName: "cluster-a", Score: 82.5, Status: "calculated"},
		{ClusterName: "cluster-a", Score: 10, Status: "error"},
		{Score: 50, Status: "calculated"},
	}

	got := scenarioHistoryScores(run)
	if len(got) != 1 || got[0].clusterName != "cluster-a" || got[0].providerName != "provider-a" || got[0].score != 82.5 || got[0].baseline != nil {
		t.Fatalf("scenario score datapoints = %+v, want one calculated score enriched with provider", got)
	}
}

func TestGraphHistoryScoresOnlyKeepsFinalScoresIncludingZero(t *testing.T) {
	run := graphRunConfigurationFixture()
	run.Spec.ResiliencyScoreBaseline = floatPointer(85)
	run.Status.ResiliencyScores = []krknv1alpha1.GraphClusterScore{
		{ClusterName: "cluster-a", ProviderName: "provider-a", Calculated: 0, Status: "pass", Baseline: floatPointer(0)},
		{ClusterName: "cluster-e", Calculated: 80, Status: "fail"},
		{ClusterName: "cluster-b", Calculated: -1, Status: "calculating"},
		{ClusterName: "cluster-c", Calculated: 80, Status: "error"},
		{ClusterName: "cluster-d", Calculated: 80, Status: "unknown"},
		{Calculated: 90, Status: "pass"},
	}

	got := graphHistoryScores(run)
	if len(got) != 2 || got[0].clusterName != "cluster-a" || got[0].providerName != "provider-a" || got[0].score != 0 || got[0].baseline == nil || *got[0].baseline != 0 {
		t.Fatalf("graph score datapoints = %+v, want zero score and zero per-cluster baseline retained", got)
	}
	if got[1].clusterName != "cluster-e" || got[1].baseline == nil || *got[1].baseline != 85 {
		t.Fatalf("graph score without status baseline = %+v, want spec baseline fallback 85", got[1])
	}
}

func TestCategoryResiliencyHistoryHandlerReturnsScoredCategoryRuns(t *testing.T) {
	const categoryName = "score-history"
	categoryLabel := krknv1alpha1.CategoryEntityLabelPrefix + categoryName
	category := &krknv1alpha1.KrknCategory{ObjectMeta: metav1.ObjectMeta{
		Name:      categoryName,
		Namespace: "default",
		Labels:    map[string]string{categoryAvailableToAllLabel: "true"},
	}}
	scoredRun := scenarioRunConfigurationFixture()
	scoredRun.Name = "scored-run"
	scoredRun.Namespace = "default"
	scoredRun.Labels = map[string]string{categoryLabel: "true"}
	scoredRun.CreationTimestamp = metav1.NewTime(time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC))
	scoredRun.Status.ClusterJobs = []krknv1alpha1.ClusterJobStatus{{ClusterName: "cluster-a", ProviderName: "provider-a"}}
	scoredRun.Status.ResiliencyScores = []krknv1alpha1.ClusterResiliencyScore{{ClusterName: "cluster-a", Score: 91.25, Status: "calculated"}}
	noScoreRun := scenarioRunConfigurationFixture()
	noScoreRun.Name = "run-without-score"
	noScoreRun.Namespace = "default"
	noScoreRun.Labels = map[string]string{categoryLabel: "true"}
	noScoreRun.Status.ResiliencyScores = []krknv1alpha1.ClusterResiliencyScore{{ClusterName: "cluster-a", Score: 50, Status: "error"}}

	handler, _ := newCategoryTestHandler(t, category, scoredRun, noScoreRun)
	response := httptest.NewRecorder()
	handler.CategoriesRouter(response, newCategoryRequest(http.MethodGet,
		v2.CategoriesPath+"/"+categoryName+"/resiliency-history", "", "admin"))
	if response.Code != http.StatusOK {
		t.Fatalf("history status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}

	var history CategoryResiliencyHistoryResponse
	if err := json.Unmarshal(response.Body.Bytes(), &history); err != nil {
		t.Fatalf("decode history response: %v", err)
	}
	points := history.Clusters["cluster-a"]
	if len(points) != 1 || points[0].RunID != "scored-run" || points[0].Score != 91.25 || points[0].ProviderName != "provider-a" {
		t.Fatalf("cluster history points = %+v, want only the scored run", points)
	}
	if len(history.ConfigurationGroups) != 1 {
		t.Fatalf("configuration groups = %+v, want only the group for scored run", history.ConfigurationGroups)
	}
}

func TestCategoryResiliencyHistoryRespectsCategoryAndPerClusterVisibility(t *testing.T) {
	const categoryName = "team-history"
	const categoryTeam = "category-team"
	const clusterViewTeam = "cluster-a-viewers"
	categoryLabel := krknv1alpha1.CategoryEntityLabelPrefix + categoryName
	clusterAURL := "https://cluster-a.example.com:6443"
	clusterBURL := "https://cluster-b.example.com:6443"

	category := &krknv1alpha1.KrknCategory{ObjectMeta: metav1.ObjectMeta{
		Name:      categoryName,
		Namespace: "default",
		Labels:    map[string]string{groupauth.GroupLabelKey(categoryTeam): "true"},
	}}
	clusterAGroup := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{Name: clusterViewTeam, Namespace: "default"},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: clusterViewTeam,
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				clusterAURL: {Actions: []string{"view"}},
			},
		},
	}
	categoryGroup := newCategoryTestGroup(categoryTeam)
	user := newCategoryTestUser(t, "category-user@example.com", categoryTeam, clusterViewTeam)

	scenarioRun := scenarioRunConfigurationFixture()
	scenarioRun.Name = "multi-cluster-scenario"
	scenarioRun.Namespace = "default"
	scenarioRun.Labels = map[string]string{categoryLabel: "true"}
	scenarioRun.Status.ClusterJobs = []krknv1alpha1.ClusterJobStatus{
		{ProviderName: "provider-a", ClusterName: "cluster-a", ClusterAPIURL: clusterAURL},
		{ProviderName: "provider-b", ClusterName: "cluster-b", ClusterAPIURL: clusterBURL},
	}
	scenarioRun.Status.ResiliencyScores = []krknv1alpha1.ClusterResiliencyScore{
		{ClusterName: "cluster-a", Score: 91, Status: "calculated"},
		{ClusterName: "cluster-b", Score: 17, Status: "calculated"},
	}
	hiddenScenarioRun := scenarioRunConfigurationFixture()
	hiddenScenarioRun.Name = "hidden-cluster-scenario"
	hiddenScenarioRun.Namespace = "default"
	hiddenScenarioRun.Labels = map[string]string{categoryLabel: "true"}
	hiddenScenarioRun.Spec.Scenario.Name = "hidden-scenario"
	hiddenScenarioRun.Status.ClusterJobs = []krknv1alpha1.ClusterJobStatus{{
		ProviderName: "provider-b", ClusterName: "cluster-b", ClusterAPIURL: clusterBURL,
	}}
	hiddenScenarioRun.Status.ResiliencyScores = []krknv1alpha1.ClusterResiliencyScore{{
		ClusterName: "cluster-b", Score: 1, Status: "calculated",
	}}

	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "history-targets", Namespace: "default"},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			TargetData: map[string][]krknv1alpha1.ClusterTarget{
				"provider-a": {{ClusterName: "cluster-a", ClusterAPIURL: clusterAURL}},
				"provider-b": {{ClusterName: "cluster-b", ClusterAPIURL: clusterBURL}},
			},
		},
	}
	graphRun := graphRunConfigurationFixture()
	graphRun.Name = "multi-cluster-graph"
	graphRun.Namespace = "default"
	graphRun.Labels = map[string]string{categoryLabel: "true"}
	graphRun.Spec.TargetRequestID = targetRequest.Name
	graphRun.Spec.TargetClusters = map[string][]string{
		"provider-a": {"cluster-a"},
		"provider-b": {"cluster-b"},
	}
	graphRun.Status.ResiliencyScores = []krknv1alpha1.GraphClusterScore{
		{ProviderName: "provider-a", ClusterName: "cluster-a", Calculated: 88, Status: "pass"},
		{ProviderName: "provider-b", ClusterName: "cluster-b", Calculated: 12, Status: "fail"},
	}
	hiddenGraphRun := graphRunConfigurationFixture()
	hiddenGraphRun.Name = "hidden-cluster-graph"
	hiddenGraphRun.Namespace = "default"
	hiddenGraphRun.Labels = map[string]string{categoryLabel: "true"}
	hiddenGraphRun.Spec.Graph["node-a"] = krknv1alpha1.GraphScenarioNode{
		Scenario: scenarioReference("hidden-scenario", false, ""),
	}
	hiddenGraphRun.Spec.TargetRequestID = targetRequest.Name
	hiddenGraphRun.Spec.TargetClusters = map[string][]string{"provider-b": {"cluster-b"}}
	hiddenGraphRun.Status.ResiliencyScores = []krknv1alpha1.GraphClusterScore{{
		ProviderName: "provider-b", ClusterName: "cluster-b", Calculated: 1, Status: "fail",
	}}

	handler, _ := newCategoryTestHandler(t, category, categoryGroup, clusterAGroup, user,
		scenarioRun, graphRun, hiddenScenarioRun, hiddenGraphRun, targetRequest)
	response := httptest.NewRecorder()
	handler.CategoriesRouter(response, newCategoryRequest(http.MethodGet,
		v2.CategoriesPath+"/"+categoryName+"/resiliency-history", "", "user"))
	if response.Code != http.StatusOK {
		t.Fatalf("history status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}

	var history CategoryResiliencyHistoryResponse
	if err := json.Unmarshal(response.Body.Bytes(), &history); err != nil {
		t.Fatalf("decode history response: %v", err)
	}
	clusterAPoints := history.Clusters["cluster-a"]
	if len(clusterAPoints) != 2 {
		t.Fatalf("visible cluster-a datapoints = %+v, want scenario and graph scores", clusterAPoints)
	}
	if _, exists := history.Clusters["cluster-b"]; exists {
		t.Fatalf("history leaked scores from inaccessible cluster-b: %+v", history.Clusters["cluster-b"])
	}
	if len(history.ConfigurationGroups) != 2 {
		t.Fatalf("configuration groups = %+v, want groups only for runs with visible scores", history.ConfigurationGroups)
	}
}

func TestCategoryResiliencyHistoryHandlerValidatesAuthenticationMethodAndCategory(t *testing.T) {
	path := v2.CategoriesPath + "/missing-category/resiliency-history"
	privateCategory := &krknv1alpha1.KrknCategory{ObjectMeta: metav1.ObjectMeta{
		Name:      "private-category",
		Namespace: "default",
		Labels:    map[string]string{groupauth.GroupLabelKey("private-team"): "true"},
	}}
	clusterGroup := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-a-viewers", Namespace: "default"},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: "cluster-a-viewers",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://cluster-a.example.com:6443": {Actions: []string{"view"}},
			},
		},
	}
	user := newCategoryTestUser(t, "category-user@example.com", "cluster-a-viewers")
	handler, _ := newCategoryTestHandler(t, privateCategory, newCategoryTestGroup("private-team"), clusterGroup, user)
	tests := []struct {
		name       string
		request    *http.Request
		wantStatus int
	}{
		{name: "authentication required", request: httptest.NewRequest(http.MethodGet, path, nil), wantStatus: http.StatusUnauthorized},
		{name: "method not allowed", request: newCategoryRequest(http.MethodPost, path, "", "admin"), wantStatus: http.StatusMethodNotAllowed},
		{name: "category not found", request: newCategoryRequest(http.MethodGet, path, "", "admin"), wantStatus: http.StatusNotFound},
		{name: "invalid category name", request: newCategoryRequest(http.MethodGet, v2.CategoriesPath+"/bad%20name/resiliency-history", "", "admin"), wantStatus: http.StatusBadRequest},
		{name: "private category denied", request: newCategoryRequest(http.MethodGet, v2.CategoriesPath+"/private-category/resiliency-history", "", "user"), wantStatus: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.CategoriesRouter(response, test.request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func scenarioHistoryRun(
	runID string,
	createdAt time.Time,
	configuration *krknv1alpha1.KrknScenarioRun,
	scores []krknv1alpha1.ClusterResiliencyScore,
) categoryHistoryRun {
	run := configuration.DeepCopy()
	run.Name = runID
	run.CreationTimestamp = metav1.NewTime(createdAt)
	run.Status.ResiliencyScores = scores
	return categoryHistoryRun{
		runType:   categoryHistoryScenarioRunType,
		runID:     runID,
		createdAt: createdAt,
		scenario:  run,
		scores:    scenarioHistoryScores(run),
	}
}

func graphHistoryRun(
	runID string,
	createdAt time.Time,
	configuration *krknv1alpha1.KrknGraphRun,
	scores []krknv1alpha1.GraphClusterScore,
) categoryHistoryRun {
	run := configuration.DeepCopy()
	run.Name = runID
	run.CreationTimestamp = metav1.NewTime(createdAt)
	run.Status.ResiliencyScores = scores
	return categoryHistoryRun{
		runType:   categoryHistoryGraphRunType,
		runID:     runID,
		createdAt: createdAt,
		graph:     run,
		scores:    graphHistoryScores(run),
	}
}

func datapointRunIDs(points []CategoryResiliencyDataPoint) []string {
	runIDs := make([]string, 0, len(points))
	for _, point := range points {
		runIDs = append(runIDs, point.RunID)
	}
	return runIDs
}

func datapointForRunID(t *testing.T, points []CategoryResiliencyDataPoint, runID string) CategoryResiliencyDataPoint {
	t.Helper()
	for _, point := range points {
		if point.RunID == runID {
			return point
		}
	}
	t.Fatalf("no datapoint for run %q", runID)
	return CategoryResiliencyDataPoint{}
}
