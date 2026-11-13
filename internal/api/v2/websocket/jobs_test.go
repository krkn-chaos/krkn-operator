/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package websocket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHandleClientMessage_JobsSubscribe(t *testing.T) {
	hub := NewHub()
	handler := NewHandler(hub, nil, "test-namespace", &mockAuthzChecker{}, nil)

	client := &Client{
		userID:          "test-user",
		isAdmin:         false,
		send:            make(chan []byte, 256),
		subscriptions:   make(map[string]map[string]bool),
		paginationState: make(map[string]*PaginationClientState),
	}

	page := 2
	limit := 5
	msg := &ClientMessage{
		Action:         "subscribe",
		Resource:       "jobs",
		Page:           &page,
		Limit:          &limit,
		Categories:     []string{"resilience", "network"},
		GroupRollbacks: true,
		SubscriptionID: "jobs-sub-2",
	}

	handler.handleClientMessage(client, msg)

	// Verify subscription was created
	client.mu.RLock()
	_, subscribed := client.subscriptions["jobs"]
	ps := client.paginationState["jobs"]
	client.mu.RUnlock()

	if !subscribed {
		t.Error("expected client to be subscribed to 'jobs'")
	}
	if ps == nil {
		t.Fatal("expected pagination state to be set")
	}
	if ps.Page != 2 {
		t.Errorf("expected page 2, got %d", ps.Page)
	}
	if ps.Limit != 5 {
		t.Errorf("expected limit 5, got %d", ps.Limit)
	}
	if ps.SubscriptionID != "jobs-sub-2" {
		t.Errorf("expected subscription ID jobs-sub-2, got %q", ps.SubscriptionID)
	}
	if !ps.GroupRollbacks {
		t.Error("expected grouped rollback mode to be stored")
	}
	if len(ps.Categories) != 2 || ps.Categories[0] != "resilience" || ps.Categories[1] != "network" {
		t.Errorf("expected category filters to be stored, got %v", ps.Categories)
	}
}

func TestHandleClientMessage_JobsSubscribeDefaults(t *testing.T) {
	hub := NewHub()
	handler := NewHandler(hub, nil, "test-namespace", &mockAuthzChecker{}, nil)

	client := &Client{
		userID:          "test-user",
		isAdmin:         false,
		send:            make(chan []byte, 256),
		subscriptions:   make(map[string]map[string]bool),
		paginationState: make(map[string]*PaginationClientState),
	}

	msg := &ClientMessage{
		Action:   "subscribe",
		Resource: "jobs",
	}

	handler.handleClientMessage(client, msg)

	client.mu.RLock()
	ps := client.paginationState["jobs"]
	client.mu.RUnlock()

	if ps == nil {
		t.Fatal("expected pagination state to be set")
	}
	if ps.Page != 1 {
		t.Errorf("expected default page 1, got %d", ps.Page)
	}
	if ps.Limit <= 0 {
		t.Errorf("expected positive default limit, got %d", ps.Limit)
	}
}

func TestSendJobsSnapshot(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	now := time.Now()
	sr1 := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "sr-1",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
			Labels:            map[string]string{},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{Phase: "Succeeded"},
	}
	sr2 := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "sr-2",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(now.Add(-1 * time.Hour)),
			Labels:            map[string]string{},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{Phase: "Running"},
	}
	gr1 := &krknv1alpha1.KrknGraphRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "gr-1",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(now),
		},
		Status: krknv1alpha1.KrknGraphRunStatus{Phase: "Running"},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(sr1, sr2, gr1).
		Build()

	hub := NewHub()
	handler := NewHandler(hub, fakeClient, "default", &mockAuthzChecker{}, nil)

	client := &Client{
		userID:          "test-user",
		isAdmin:         true,
		send:            make(chan []byte, 256),
		subscriptions:   make(map[string]map[string]bool),
		paginationState: make(map[string]*PaginationClientState),
	}

	// Subscribe with page 1, limit 2
	page := 1
	limit := 2
	msg := &ClientMessage{
		Action:   "subscribe",
		Resource: "jobs",
		Page:     &page,
		Limit:    &limit,
	}
	handler.handleClientMessage(client, msg)

	// Read the snapshot message
	select {
	case data := <-client.send:
		var serverMsg ServerMessage
		if err := json.Unmarshal(data, &serverMsg); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if serverMsg.Resource != "jobs" {
			t.Errorf("expected resource 'jobs', got '%s'", serverMsg.Resource)
		}
		if serverMsg.Event != "snapshot" {
			t.Errorf("expected event 'snapshot', got '%s'", serverMsg.Event)
		}
		if serverMsg.Pagination == nil {
			t.Fatal("expected pagination metadata")
		}
		if serverMsg.Pagination.Total != 3 {
			t.Errorf("expected total 3, got %d", serverMsg.Pagination.Total)
		}
		if serverMsg.Pagination.TotalPages != 2 {
			t.Errorf("expected 2 pages, got %d", serverMsg.Pagination.TotalPages)
		}
		if serverMsg.Pagination.Page != 1 {
			t.Errorf("expected page 1, got %d", serverMsg.Pagination.Page)
		}

		// Parse the data to verify items
		dataBytes, _ := json.Marshal(serverMsg.Data)
		var snapshot WSUnifiedJobsSnapshot
		if err := json.Unmarshal(dataBytes, &snapshot); err != nil {
			t.Fatalf("failed to unmarshal snapshot data: %v", err)
		}
		if len(snapshot.Jobs) != 2 {
			t.Errorf("expected 2 items on page 1, got %d", len(snapshot.Jobs))
		}

	case <-time.After(time.Second):
		t.Error("timeout waiting for snapshot message")
	}
}

func TestSendJobsSnapshotFiltersBeforeSortingAndPagination(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	matchTime := time.Date(2026, time.October, 2, 12, 0, 0, 0, time.UTC)
	newestTime := time.Date(2026, time.October, 3, 12, 0, 0, 0, time.UTC)
	matchingRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "scenario-match", Namespace: "default", CreationTimestamp: metav1.NewTime(matchTime)},
		Spec:       krknv1alpha1.KrknScenarioRunSpec{OwnerUserID: "alice@example.com", CustomRunName: "Needle custom label"},
		Status:     krknv1alpha1.KrknScenarioRunStatus{Phase: "Succeeded"},
	}
	newestRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "newest-nonmatch", Namespace: "default", CreationTimestamp: metav1.NewTime(newestTime)},
		Spec:       krknv1alpha1.KrknScenarioRunSpec{OwnerUserID: "bob@example.com", CustomRunName: "another run"},
		Status:     krknv1alpha1.KrknScenarioRunStatus{Phase: "Succeeded"},
	}
	wrongOwnerRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "wrong-owner", Namespace: "default", CreationTimestamp: metav1.NewTime(matchTime)},
		Spec:       krknv1alpha1.KrknScenarioRunSpec{OwnerUserID: "bob@example.com", CustomRunName: "Needle custom label"},
		Status:     krknv1alpha1.KrknScenarioRunStatus{Phase: "Succeeded"},
	}
	wrongNameRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "wrong-name", Namespace: "default", CreationTimestamp: metav1.NewTime(matchTime)},
		Spec:       krknv1alpha1.KrknScenarioRunSpec{OwnerUserID: "alice@example.com", CustomRunName: "another label"},
		Status:     krknv1alpha1.KrknScenarioRunStatus{Phase: "Succeeded"},
	}
	wrongDateRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "wrong-date", Namespace: "default", CreationTimestamp: metav1.NewTime(matchTime.Add(-24 * time.Hour))},
		Spec:       krknv1alpha1.KrknScenarioRunSpec{OwnerUserID: "alice@example.com", CustomRunName: "Needle custom label"},
		Status:     krknv1alpha1.KrknScenarioRunStatus{Phase: "Succeeded"},
	}
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(matchingRun, newestRun, wrongOwnerRun, wrongNameRun, wrongDateRun).Build()
	handler := NewHandler(NewHub(), fakeClient, "default", &mockAuthzChecker{}, nil)
	client := &Client{
		userID: "test-user", isAdmin: true, send: make(chan []byte, 16),
		subscriptions:   make(map[string]map[string]bool),
		paginationState: make(map[string]*PaginationClientState),
	}

	var request ClientMessage
	err := json.Unmarshal([]byte(`{"action":"subscribe","resource":"jobs","page":1,"limit":1,"sortBy":"createdAt","sortOrder":"desc","subscriptionId":"filtered-sub","filters":{"owner":"alice@example.com","createdAtFrom":"2026-10-02T12:00:00Z","createdAtTo":"2026-10-02T12:00:00Z","runName":"needle"}}`), &request)
	if err != nil {
		t.Fatalf("decode jobs subscription: %v", err)
	}
	handler.handleClientMessage(client, &request)

	select {
	case data := <-client.send:
		var message ServerMessage
		if err := json.Unmarshal(data, &message); err != nil {
			t.Fatalf("decode jobs snapshot: %v", err)
		}
		if message.Pagination == nil {
			t.Fatal("expected pagination metadata")
		}
		if message.Pagination.Total != 1 || message.Pagination.TotalPages != 1 {
			t.Fatalf("expected one filtered result and page, got %#v", message.Pagination)
		}
		var snapshot WSUnifiedJobsSnapshot
		payload, err := json.Marshal(message.Data)
		if err != nil {
			t.Fatalf("encode snapshot payload: %v", err)
		}
		if err := json.Unmarshal(payload, &snapshot); err != nil {
			t.Fatalf("decode snapshot payload: %v", err)
		}
		if len(snapshot.Jobs) != 1 || snapshot.Jobs[0].Name != "scenario-match" {
			t.Fatalf("expected matching run from outside unfiltered top page, got %#v", snapshot.Jobs)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for filtered jobs snapshot")
	}
}

func TestBroadcastJobsPageUpdate(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	now := time.Now()
	sr1 := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "sr-1",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
			Labels:            map[string]string{},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{Phase: "Running"},
	}
	sr2 := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "sr-2",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(now.Add(-1 * time.Hour)),
			Labels:            map[string]string{},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{Phase: "Running"},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(sr1, sr2).
		Build()

	hub := NewHub()
	go hub.Run()

	broadcaster := NewBroadcaster(hub, &mockAuthzChecker{}, fakeClient, "default")

	// Client on page 1 with limit 1
	client := &Client{
		userID:  "test-user",
		isAdmin: true,
		send:    make(chan []byte, 256),
		subscriptions: map[string]map[string]bool{
			"jobs": {"*": true},
		},
		paginationState: map[string]*PaginationClientState{
			"jobs": {Page: 1, Limit: 1},
		},
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Trigger page update
	broadcaster.BroadcastJobsPageUpdate(context.Background())

	select {
	case data := <-client.send:
		var serverMsg ServerMessage
		if err := json.Unmarshal(data, &serverMsg); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if serverMsg.Resource != "jobs" {
			t.Errorf("expected resource 'jobs', got '%s'", serverMsg.Resource)
		}
		if serverMsg.Pagination == nil {
			t.Fatal("expected pagination metadata")
		}
		if serverMsg.Pagination.Total != 2 {
			t.Errorf("expected total 2, got %d", serverMsg.Pagination.Total)
		}
		if serverMsg.Pagination.Page != 1 {
			t.Errorf("expected page 1, got %d", serverMsg.Pagination.Page)
		}

		dataBytes, _ := json.Marshal(serverMsg.Data)
		var snapshot WSUnifiedJobsSnapshot
		if err := json.Unmarshal(dataBytes, &snapshot); err != nil {
			t.Fatalf("failed to unmarshal snapshot: %v", err)
		}
		if len(snapshot.Jobs) != 1 {
			t.Errorf("expected 1 item on page 1 (limit 1), got %d", len(snapshot.Jobs))
		}

	case <-time.After(time.Second):
		t.Error("timeout waiting for broadcast")
	}
}

func TestBroadcastJobsPageUpdate_Dedup(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	sr := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "sr-1",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(time.Now()),
			Labels:            map[string]string{},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{Phase: "Running"},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(sr).
		Build()

	hub := NewHub()
	go hub.Run()

	broadcaster := NewBroadcaster(hub, &mockAuthzChecker{}, fakeClient, "default")

	client := &Client{
		userID:  "test-user",
		isAdmin: true,
		send:    make(chan []byte, 256),
		subscriptions: map[string]map[string]bool{
			"jobs": {"*": true},
		},
		paginationState: map[string]*PaginationClientState{
			"jobs": {Page: 1, Limit: 10},
		},
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// First broadcast should send
	broadcaster.BroadcastJobsPageUpdate(context.Background())

	select {
	case <-client.send:
		// expected
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for first broadcast")
	}

	// Second broadcast with same data should be deduplicated
	broadcaster.BroadcastJobsPageUpdate(context.Background())

	select {
	case <-client.send:
		t.Error("expected second broadcast to be deduplicated (no message)")
	case <-time.After(100 * time.Millisecond):
		// expected: no message due to dedup
	}
}

func TestUnsubscribe_CleansUpJobsState(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	sr := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "sr-1",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(time.Now()),
			Labels:            map[string]string{},
		},
		Status: krknv1alpha1.KrknScenarioRunStatus{Phase: "Running"},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(sr).
		Build()

	hub := NewHub()
	go hub.Run()

	broadcaster := NewBroadcaster(hub, &mockAuthzChecker{}, fakeClient, "default")

	client := &Client{
		userID:  "test-user",
		isAdmin: true,
		send:    make(chan []byte, 256),
		subscriptions: map[string]map[string]bool{
			"jobs": {"*": true},
		},
		paginationState: map[string]*PaginationClientState{
			"jobs": {Page: 1, Limit: 10},
		},
	}

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Unsubscribe from jobs
	client.Unsubscribe("jobs", []string{"*"})

	// Verify subscription and pagination state cleaned up
	client.mu.RLock()
	_, hasSub := client.subscriptions["jobs"]
	_, hasPS := client.paginationState["jobs"]
	client.mu.RUnlock()

	if hasSub {
		t.Error("expected jobs subscription to be removed after unsubscribe")
	}
	if hasPS {
		t.Error("expected jobs pagination state to be removed after unsubscribe")
	}

	// Broadcast should NOT send to unsubscribed client
	broadcaster.BroadcastJobsPageUpdate(context.Background())

	select {
	case <-client.send:
		t.Error("expected no message after unsubscribe")
	case <-time.After(100 * time.Millisecond):
		// expected: no message
	}
}

func TestBuildUnifiedJobList_SkipsGraphRunChildren(t *testing.T) {
	now := time.Now()

	scenarioRuns := []krknv1alpha1.KrknScenarioRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "standalone",
				CreationTimestamp: metav1.NewTime(now),
				Labels:            map[string]string{},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "graph-child",
				CreationTimestamp: metav1.NewTime(now.Add(-time.Hour)),
				Labels:            map[string]string{"krkn.dev/graph-run": "parent"},
			},
		},
	}

	graphRuns := []krknv1alpha1.KrknGraphRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "parent",
				CreationTimestamp: metav1.NewTime(now.Add(-30 * time.Minute)),
			},
		},
	}

	jobs := buildUnifiedJobList(scenarioRuns, graphRuns)

	if len(jobs) != 2 {
		t.Errorf("expected 2 items (standalone + parent graph), got %d", len(jobs))
	}

	for _, job := range jobs {
		if job.Name == "graph-child" {
			t.Error("graph-run child should not appear in unified list")
		}
	}
}

func TestPaginateJobItems(t *testing.T) {
	items := make([]WSUnifiedJobItem, 10)
	for i := range items {
		items[i] = WSUnifiedJobItem{Name: "item"}
	}

	// Page 1 of 10 items with limit 3
	page, meta := paginateJobItems(items, 1, 3)
	if len(page) != 3 {
		t.Errorf("expected 3 items, got %d", len(page))
	}
	if meta.Total != 10 {
		t.Errorf("expected total 10, got %d", meta.Total)
	}
	if meta.TotalPages != 4 {
		t.Errorf("expected 4 pages, got %d", meta.TotalPages)
	}

	// Beyond total
	page, meta = paginateJobItems(items, 100, 3)
	if len(page) != 0 {
		t.Errorf("expected 0 items beyond total, got %d", len(page))
	}
	if meta.Total != 10 {
		t.Errorf("expected total 10 even beyond range, got %d", meta.Total)
	}
}

func TestFilterJobsByCategoriesMatchesAnySelectedCategory(t *testing.T) {
	jobs := []WSUnifiedJobItem{
		{Name: "resilience-run", Categories: []string{"resilience"}},
		{Name: "network-run", Categories: []string{"network"}},
		{Name: "uncategorized-run"},
	}

	filtered := filterJobsByCategories(jobs, []string{"resilience", "network"})
	if len(filtered) != 2 || filtered[0].Name != "resilience-run" || filtered[1].Name != "network-run" {
		t.Fatalf("expected runs matching either category, got %#v", filtered)
	}

	if got := filterJobsByCategories(jobs, nil); len(got) != len(jobs) {
		t.Fatalf("empty category filter should keep all jobs, got %d of %d", len(got), len(jobs))
	}
}

func TestFilterJobsBySubscriptionFiltersMatchesRunNamesCaseInsensitively(t *testing.T) {
	jobs := []WSUnifiedJobItem{
		{
			Type: "scenarioRun", Name: "scenario-id",
			ScenarioRun: &ScenarioRunStatusResponse{ScenarioRunName: "scenario-id", CustomRunName: "Custom Chaos Label"},
		},
		{Type: "graphRun", Name: "Graph-Experiment-9"},
	}
	for _, testCase := range []struct {
		needle string
		want   string
	}{
		{needle: "SCENARIO-ID", want: "scenario-id"},
		{needle: "custom chaos", want: "scenario-id"},
		{needle: "graph-experiment", want: "Graph-Experiment-9"},
	} {
		filtered := filterJobsBySubscriptionFilters(append([]WSUnifiedJobItem(nil), jobs...), JobsSubscriptionFilters{RunName: testCase.needle})
		if len(filtered) != 1 || filtered[0].Name != testCase.want {
			t.Errorf("run-name filter %q returned %#v, want %q", testCase.needle, filtered, testCase.want)
		}
	}
}

type categoryVisibilityMock struct {
	mockAuthzChecker
	visible map[string]struct{}
}

func (m *categoryVisibilityMock) VisibleCategoryNames(context.Context) (map[string]struct{}, error) {
	return m.visible, nil
}

func TestBroadcastJobsPageUpdateEchoesSubscriptionAndAppliesCategoriesAndFilters(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	scenarioRuns := []krknv1alpha1.KrknScenarioRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "resilience-run", Namespace: "default",
				CreationTimestamp: metav1.NewTime(time.Now()),
				Labels:            map[string]string{krknv1alpha1.CategoryEntityLabelPrefix + "resilience": "true"},
			},
			Status: krknv1alpha1.KrknScenarioRunStatus{Phase: "Succeeded"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "network-run", Namespace: "default",
				CreationTimestamp: metav1.NewTime(time.Now().Add(-time.Minute)),
				Labels:            map[string]string{krknv1alpha1.CategoryEntityLabelPrefix + "network": "true"},
			},
			Status: krknv1alpha1.KrknScenarioRunStatus{Phase: "Succeeded"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "resilience-other", Namespace: "default",
				CreationTimestamp: metav1.NewTime(time.Now().Add(time.Minute)),
				Labels:            map[string]string{krknv1alpha1.CategoryEntityLabelPrefix + "resilience": "true"},
			},
			Status: krknv1alpha1.KrknScenarioRunStatus{Phase: "Succeeded"},
		},
	}
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(&scenarioRuns[0], &scenarioRuns[1]).Build()
	hub := NewHub()
	go hub.Run()
	client := &Client{
		userID: "test-user", isAdmin: true, send: make(chan []byte, 16),
		subscriptions: map[string]map[string]bool{"jobs": {"*": true}},
		paginationState: map[string]*PaginationClientState{
			"jobs": {Page: 1, Limit: 1, Categories: []string{"resilience"}, Filters: JobsSubscriptionFilters{RunName: "resilience-run"}, SubscriptionID: "jobs-sub-current"},
		},
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	broadcaster := NewBroadcaster(hub, &categoryVisibilityMock{visible: map[string]struct{}{"resilience": {}, "network": {}}}, fakeClient, "default")
	broadcaster.BroadcastJobsPageUpdate(context.Background())

	select {
	case data := <-client.send:
		var message struct {
			SubscriptionID string `json:"subscriptionId"`
			Data           struct {
				Jobs []WSUnifiedJobItem `json:"jobs"`
			} `json:"data"`
		}
		if err := json.Unmarshal(data, &message); err != nil {
			t.Fatalf("decode jobs snapshot: %v", err)
		}
		if message.SubscriptionID != "jobs-sub-current" {
			t.Errorf("expected current subscription ID, got %q", message.SubscriptionID)
		}
		if len(message.Data.Jobs) != 1 || message.Data.Jobs[0].Name != "resilience-run" {
			t.Errorf("expected only category-matching jobs, got %#v", message.Data.Jobs)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for jobs snapshot")
	}
}
