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

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func TestListUserGroups_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	group1 := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dev-team",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name:        "dev-team",
			Description: "Development team",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://api.cluster1.com": {
					Actions: []string{"view", "run"},
				},
			},
		},
	}

	group2 := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ops-team",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name:        "ops-team",
			Description: "Operations team",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://api.cluster2.com": {
					Actions: []string{"view", "run", "cancel"},
				},
			},
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(group1, group2).
		Build()

	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("GET", GroupsPath, nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.ListUserGroups(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response ListUserGroupsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Total != 2 {
		t.Errorf("Expected 2 groups, got %d", response.Total)
	}

	if len(response.Groups) != 2 {
		t.Errorf("Expected 2 groups in array, got %d", len(response.Groups))
	}
}

func TestListUserGroups_Forbidden(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("GET", GroupsPath, nil)
	req = req.WithContext(createUserContext("user@example.com")) // Non-admin user
	w := httptest.NewRecorder()

	handler.ListUserGroups(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestGetUserGroup_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	group := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dev-team",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name:        "dev-team",
			Description: "Development team",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://api.cluster1.com": {
					Actions: []string{"view", "run"},
				},
			},
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(group).
		Build()

	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("GET", GroupsPath+"/dev-team", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.GetUserGroup(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response UserGroupResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Name != "dev-team" {
		t.Errorf("Expected group name 'dev-team', got '%s'", response.Name)
	}
}

func TestGetUserGroup_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("GET", GroupsPath+"/nonexistent", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.GetUserGroup(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestCreateUserGroup_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	createReq := CreateUserGroupRequest{
		Name:        "dev-team",
		Description: "Development team",
		ClusterPermissions: map[string]ClusterPermissionSet{
			"https://api.cluster1.com": {
				Actions: []string{"view", "run"},
			},
		},
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", GroupsPath, bytes.NewReader(body))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.CreateUserGroup(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response UserGroupResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Name != "dev-team" {
		t.Errorf("Expected group name 'dev-team', got '%s'", response.Name)
	}
}

func TestCreateUserGroup_ValidationError(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	tests := []struct {
		name    string
		request CreateUserGroupRequest
	}{
		{
			name: "empty name",
			request: CreateUserGroupRequest{
				Name: "",
				ClusterPermissions: map[string]ClusterPermissionSet{
					"https://api.cluster1.com": {
						Actions: []string{"view"},
					},
				},
			},
		},
		{
			name: "empty permissions",
			request: CreateUserGroupRequest{
				Name:               "test-group",
				ClusterPermissions: map[string]ClusterPermissionSet{},
			},
		},
		{
			name: "invalid action",
			request: CreateUserGroupRequest{
				Name: "test-group",
				ClusterPermissions: map[string]ClusterPermissionSet{
					"https://api.cluster1.com": {
						Actions: []string{"invalid"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", GroupsPath, bytes.NewReader(body))
			req = req.WithContext(createAdminContext())
			w := httptest.NewRecorder()

			handler.CreateUserGroup(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d. Body: %s", http.StatusBadRequest, w.Code, w.Body.String())
			}
		})
	}
}

func TestUpdateUserGroup_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	existingGroup := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dev-team",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name:        "dev-team",
			Description: "Old description",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://api.cluster1.com": {
					Actions: []string{"view"},
				},
			},
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(existingGroup).
		Build()

	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	updateReq := UpdateUserGroupRequest{
		Description: strPtr("New description"),
		ClusterPermissions: map[string]ClusterPermissionSet{
			"https://api.cluster1.com": {
				Actions: []string{"view", "run"},
			},
		},
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("PATCH", GroupsPath+"/dev-team", bytes.NewReader(body))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.UpdateUserGroup(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response UpdateUserGroupResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify the update was successful
	if response.Group.Name != "dev-team" {
		t.Errorf("Expected group name 'dev-team', got '%s'", response.Group.Name)
	}

	if response.Group.Description != "New description" {
		t.Errorf("Expected description 'New description', got '%s'", response.Group.Description)
	}

	// Verify cluster permissions were updated
	if len(response.Group.ClusterPermissions) == 0 {
		t.Error("Expected cluster permissions to be updated")
	}

	perms, ok := response.Group.ClusterPermissions["https://api.cluster1.com"]
	if !ok {
		t.Error("Expected cluster permissions for cluster1")
	} else if len(perms.Actions) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(perms.Actions))
	}
}

func TestDeleteUserGroup_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	group := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dev-team",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: "dev-team",
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(group).
		Build()

	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("DELETE", GroupsPath+"/dev-team", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.DeleteUserGroup(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestListGroupMembers_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	group := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dev-team",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: "dev-team",
		},
	}

	user1 := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-user1-example-com",
			Namespace: "default",
			Labels: map[string]string{
				"group.krkn.krkn-chaos.dev/dev-team": "true",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID: "user1@example.com",
			Name:   "User",
			Surname: "One",
			Role:   "user",
		},
	}

	user2 := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-user2-example-com",
			Namespace: "default",
			Labels: map[string]string{
				"group.krkn.krkn-chaos.dev/dev-team": "true",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID: "user2@example.com",
			Name:   "User",
			Surname: "Two",
			Role:   "user",
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(group, user1, user2).
		Build()

	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("GET", GroupsPath+"/dev-team/members", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.ListGroupMembers(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response ListGroupMembersResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Total != 2 {
		t.Errorf("Expected 2 members, got %d", response.Total)
	}
}

func TestAddGroupMember_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	group := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dev-team",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: "dev-team",
		},
	}

	user := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-user-example-com",
			Namespace: "default",
			Labels:    map[string]string{}, // No group labels initially
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID: "user@example.com",
			Name:   "Test",
			Surname: "User",
			Role:   "user",
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(group, user).
		Build()

	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	addReq := AddGroupMemberRequest{
		UserID: "user@example.com",
	}

	body, _ := json.Marshal(addReq)
	req := httptest.NewRequest("POST", GroupsPath+"/dev-team/members", bytes.NewReader(body))
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.AddGroupMember(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestRemoveGroupMember_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	group := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dev-team",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: "dev-team",
		},
	}

	user := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-user-example-com",
			Namespace: "default",
			Labels: map[string]string{
				"group.krkn.krkn-chaos.dev/dev-team": "true",
			},
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID: "user@example.com",
			Name:   "Test",
			Surname: "User",
			Role:   "user",
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(group, user).
		Build()

	fakeClientset := fake.NewSimpleClientset()
	handler := NewHandler(fakeClient, fakeClientset, "default", "localhost:50051")

	req := httptest.NewRequest("DELETE", GroupsPath+"/dev-team/members/user@example.com", nil)
	req = req.WithContext(createAdminContext())
	w := httptest.NewRecorder()

	handler.RemoveGroupMember(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

// Helper function
func strPtr(s string) *string {
	return &s
}
