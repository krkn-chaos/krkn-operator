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
		Action:   "subscribe",
		Resource: "jobs",
		Page:     &page,
		Limit:    &limit,
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
	broadcaster.BroadcastJobsPageUpdate()

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
	broadcaster.BroadcastJobsPageUpdate()

	select {
	case <-client.send:
		// expected
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for first broadcast")
	}

	// Second broadcast with same data should be deduplicated
	broadcaster.BroadcastJobsPageUpdate()

	select {
	case <-client.send:
		t.Error("expected second broadcast to be deduplicated (no message)")
	case <-time.After(100 * time.Millisecond):
		// expected: no message due to dedup
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
