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

Assisted-by: Claude Sonnet 4.5 (claude-sonnet-4-5@20250929)
*/

package controller

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

const (
	testOperatorName      = "krkn-operator"
	testOperatorNamespace = "krkn-operator-system"
	testRequestName       = "test-request"
	testUUID              = "test-uuid-123"
)

func setupTestReconciler(objs ...client.Object) *KrknTargetRequestReconciler {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(&krknv1alpha1.KrknTargetRequest{}, &krknv1alpha1.KrknOperatorTargetProvider{}).
		Build()

	return &KrknTargetRequestReconciler{
		Client:            fakeClient,
		Scheme:            scheme,
		OperatorName:      testOperatorName,
		OperatorNamespace: testOperatorNamespace,
	}
}

func TestReconcile_SetsUUIDLabel(t *testing.T) {
	request := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRequestName,
			Namespace: testOperatorNamespace,
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{
			UUID: testUUID,
		},
	}

	reconciler := setupTestReconciler(request)
	ctx := context.Background()

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testRequestName,
			Namespace: testOperatorNamespace,
		},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify UUID label was set
	var updated krknv1alpha1.KrknTargetRequest
	if err := reconciler.Get(ctx, types.NamespacedName{
		Name:      testRequestName,
		Namespace: testOperatorNamespace,
	}, &updated); err != nil {
		t.Fatalf("Failed to get request: %v", err)
	}

	if label, exists := updated.Labels["krkn.krkn-chaos.dev/uuid"]; !exists || label != testUUID {
		t.Errorf("Expected UUID label to be %s, got %s (exists: %v)", testUUID, label, exists)
	}
}

func TestReconcile_InitializesStatus(t *testing.T) {
	request := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRequestName,
			Namespace: testOperatorNamespace,
			Labels: map[string]string{
				"krkn.krkn-chaos.dev/uuid": testUUID,
			},
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{
			UUID: testUUID,
		},
	}

	reconciler := setupTestReconciler(request)
	ctx := context.Background()

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testRequestName,
			Namespace: testOperatorNamespace,
		},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify status was initialized
	var updated krknv1alpha1.KrknTargetRequest
	if err := reconciler.Get(ctx, types.NamespacedName{
		Name:      testRequestName,
		Namespace: testOperatorNamespace,
	}, &updated); err != nil {
		t.Fatalf("Failed to get request: %v", err)
	}

	if updated.Status.Status != "pending" {
		t.Errorf("Expected status to be 'pending', got %s", updated.Status.Status)
	}

	if updated.Status.Created == nil {
		t.Error("Expected Created timestamp to be set")
	}
}

func TestReconcile_PopulatesTargetData(t *testing.T) {
	request := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRequestName,
			Namespace: testOperatorNamespace,
			Labels: map[string]string{
				"krkn.krkn-chaos.dev/uuid": testUUID,
			},
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{
			UUID: testUUID,
		},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status:  "pending",
			Created: &metav1.Time{Time: time.Now()},
		},
	}

	target1 := &krknv1alpha1.KrknOperatorTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "target-1",
			Namespace: testOperatorNamespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetSpec{
			UUID:          "uuid-1",
			ClusterName:   "cluster-1",
			ClusterAPIURL: "https://api.cluster1.com:6443",
		},
		Status: krknv1alpha1.KrknOperatorTargetStatus{
			Ready: true,
		},
	}

	target2 := &krknv1alpha1.KrknOperatorTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "target-2",
			Namespace: testOperatorNamespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetSpec{
			UUID:          "uuid-2",
			ClusterName:   "cluster-2",
			ClusterAPIURL: "https://api.cluster2.com:6443",
		},
		Status: krknv1alpha1.KrknOperatorTargetStatus{
			Ready: true,
		},
	}

	reconciler := setupTestReconciler(request, target1, target2)
	ctx := context.Background()

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testRequestName,
			Namespace: testOperatorNamespace,
		},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify target data was populated
	var updated krknv1alpha1.KrknTargetRequest
	if err := reconciler.Get(ctx, types.NamespacedName{
		Name:      testRequestName,
		Namespace: testOperatorNamespace,
	}, &updated); err != nil {
		t.Fatalf("Failed to get request: %v", err)
	}

	targets, exists := updated.Status.TargetData[testOperatorName]
	if !exists {
		t.Fatal("Expected target data to exist for krkn-operator")
	}

	if len(targets) != 2 {
		t.Errorf("Expected 2 targets, got %d", len(targets))
	}

	// Verify target details
	expectedTargets := map[string]string{
		"cluster-1": "https://api.cluster1.com:6443",
		"cluster-2": "https://api.cluster2.com:6443",
	}

	for _, target := range targets {
		expectedURL, exists := expectedTargets[target.ClusterName]
		if !exists {
			t.Errorf("Unexpected cluster name: %s", target.ClusterName)
			continue
		}
		if target.ClusterAPIURL != expectedURL {
			t.Errorf("Expected URL %s for cluster %s, got %s", expectedURL, target.ClusterName, target.ClusterAPIURL)
		}
	}
}

func TestReconcile_MarksCompleted(t *testing.T) {
	request := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRequestName,
			Namespace: testOperatorNamespace,
			Labels: map[string]string{
				"krkn.krkn-chaos.dev/uuid": testUUID,
			},
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{
			UUID: testUUID,
		},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status:  "pending",
			Created: &metav1.Time{Time: time.Now()},
		},
	}

	provider := &krknv1alpha1.KrknOperatorTargetProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testOperatorName,
			Namespace: testOperatorNamespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderSpec{
			OperatorName: testOperatorName,
			Active:       true,
		},
	}

	reconciler := setupTestReconciler(request, provider)
	ctx := context.Background()

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testRequestName,
			Namespace: testOperatorNamespace,
		},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify request was marked as completed
	var updated krknv1alpha1.KrknTargetRequest
	if err := reconciler.Get(ctx, types.NamespacedName{
		Name:      testRequestName,
		Namespace: testOperatorNamespace,
	}, &updated); err != nil {
		t.Fatalf("Failed to get request: %v", err)
	}

	if updated.Status.Status != "Completed" {
		t.Errorf("Expected status to be 'Completed', got %s", updated.Status.Status)
	}

	if updated.Status.Completed == nil {
		t.Error("Expected Completed timestamp to be set")
	}
}

func TestReconcile_SkipsCompletedRequests(t *testing.T) {
	now := metav1.Now()
	request := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRequestName,
			Namespace: testOperatorNamespace,
			Labels: map[string]string{
				"krkn.krkn-chaos.dev/uuid": testUUID,
			},
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{
			UUID: testUUID,
		},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status:    "Completed",
			Created:   &now,
			Completed: &now,
		},
	}

	reconciler := setupTestReconciler(request)
	ctx := context.Background()

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testRequestName,
			Namespace: testOperatorNamespace,
		},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify request was not modified
	var updated krknv1alpha1.KrknTargetRequest
	if err := reconciler.Get(ctx, types.NamespacedName{
		Name:      testRequestName,
		Namespace: testOperatorNamespace,
	}, &updated); err != nil {
		t.Fatalf("Failed to get request: %v", err)
	}

	if updated.Status.Status != "Completed" {
		t.Errorf("Expected status to remain 'Completed', got %s", updated.Status.Status)
	}
}

func TestReconcile_HandlesEmptyTargetList(t *testing.T) {
	request := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRequestName,
			Namespace: testOperatorNamespace,
			Labels: map[string]string{
				"krkn.krkn-chaos.dev/uuid": testUUID,
			},
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{
			UUID: testUUID,
		},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status:  "pending",
			Created: &metav1.Time{Time: time.Now()},
		},
	}

	provider := &krknv1alpha1.KrknOperatorTargetProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testOperatorName,
			Namespace: testOperatorNamespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetProviderSpec{
			OperatorName: testOperatorName,
			Active:       true,
		},
	}

	reconciler := setupTestReconciler(request, provider)
	ctx := context.Background()

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testRequestName,
			Namespace: testOperatorNamespace,
		},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify target data is empty but request is still completed
	var updated krknv1alpha1.KrknTargetRequest
	if err := reconciler.Get(ctx, types.NamespacedName{
		Name:      testRequestName,
		Namespace: testOperatorNamespace,
	}, &updated); err != nil {
		t.Fatalf("Failed to get request: %v", err)
	}

	targets, exists := updated.Status.TargetData[testOperatorName]
	if !exists {
		t.Fatal("Expected target data to exist for krkn-operator")
	}

	if len(targets) != 0 {
		t.Errorf("Expected 0 targets, got %d", len(targets))
	}

	if updated.Status.Status != "Completed" {
		t.Errorf("Expected status to be 'Completed', got %s", updated.Status.Status)
	}
}

func TestReconcile_OnlyIncludesReadyTargets(t *testing.T) {
	request := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRequestName,
			Namespace: testOperatorNamespace,
			Labels: map[string]string{
				"krkn.krkn-chaos.dev/uuid": testUUID,
			},
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{
			UUID: testUUID,
		},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			Status:  "pending",
			Created: &metav1.Time{Time: time.Now()},
		},
	}

	readyTarget := &krknv1alpha1.KrknOperatorTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ready-target",
			Namespace: testOperatorNamespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetSpec{
			UUID:          "uuid-ready",
			ClusterName:   "ready-cluster",
			ClusterAPIURL: "https://api.ready.com:6443",
		},
		Status: krknv1alpha1.KrknOperatorTargetStatus{
			Ready: true,
		},
	}

	notReadyTarget := &krknv1alpha1.KrknOperatorTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "not-ready-target",
			Namespace: testOperatorNamespace,
		},
		Spec: krknv1alpha1.KrknOperatorTargetSpec{
			UUID:          "uuid-not-ready",
			ClusterName:   "not-ready-cluster",
			ClusterAPIURL: "https://api.notready.com:6443",
		},
		Status: krknv1alpha1.KrknOperatorTargetStatus{
			Ready: false,
		},
	}

	reconciler := setupTestReconciler(request, readyTarget, notReadyTarget)
	ctx := context.Background()

	_, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      testRequestName,
			Namespace: testOperatorNamespace,
		},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify only ready target was included
	var updated krknv1alpha1.KrknTargetRequest
	if err := reconciler.Get(ctx, types.NamespacedName{
		Name:      testRequestName,
		Namespace: testOperatorNamespace,
	}, &updated); err != nil {
		t.Fatalf("Failed to get request: %v", err)
	}

	targets, exists := updated.Status.TargetData[testOperatorName]
	if !exists {
		t.Fatal("Expected target data to exist for krkn-operator")
	}

	if len(targets) != 1 {
		t.Errorf("Expected 1 target, got %d", len(targets))
	}

	if targets[0].ClusterName != "ready-cluster" {
		t.Errorf("Expected cluster name 'ready-cluster', got %s", targets[0].ClusterName)
	}
}
