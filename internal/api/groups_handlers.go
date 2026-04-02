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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

// ListUserGroups handles GET /api/v1/groups
// Lists all user groups (admin only)
func (h *Handler) ListUserGroups(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("list-user-groups")

	// Check admin privileges
	if !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// List all KrknUserGroup CRDs
	var groups krknv1alpha1.KrknUserGroupList
	if err := h.client.List(ctx, &groups, client.InNamespace(h.namespace)); err != nil {
		logger.Error(err, "Failed to list user groups")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list user groups: " + err.Error(),
		})
		return
	}

	// Convert to response format
	groupResponses := make([]UserGroupResponse, len(groups.Items))
	for i, group := range groups.Items {
		groupResponses[i] = buildUserGroupResponse(ctx, h.client, &group, h.namespace)
	}

	logger.Info("Listed user groups", "total", len(groupResponses))

	writeJSON(w, http.StatusOK, ListUserGroupsResponse{
		Groups: groupResponses,
		Total:  len(groupResponses),
	})
}

// GetUserGroup handles GET /api/v1/groups/:groupName
// Returns a single user group (admin only)
func (h *Handler) GetUserGroup(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("get-user-group")

	// Check admin privileges
	if !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Extract groupName from path
	groupName, err := extractPathSuffix(r.URL.Path, GroupsPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid groupName in path",
		})
		return
	}

	// Remove /members suffix if present
	groupName = strings.TrimSuffix(groupName, "/members")

	// Fetch group
	group := &krknv1alpha1.KrknUserGroup{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: groupName, Namespace: h.namespace}, group); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("User group '%s' not found", groupName),
			})
		} else {
			logger.Error(err, "Failed to get user group", "groupName", groupName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get user group",
			})
		}
		return
	}

	logger.Info("Retrieved user group", "groupName", groupName)
	writeJSON(w, http.StatusOK, buildUserGroupResponse(ctx, h.client, group, h.namespace))
}

// CreateUserGroup handles POST /api/v1/groups
// Creates a new user group (admin only)
func (h *Handler) CreateUserGroup(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("create-user-group")

	// Check admin privileges
	if !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Parse request body
	var req CreateUserGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate request
	if req.Name == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Group name is required",
		})
		return
	}

	// Validate group name length to prevent label truncation and collisions
	// Group names become part of labels: "group.krkn.krkn-chaos.dev/<name>"
	// Kubernetes label names (after the prefix) are limited to 63 characters
	sanitizedName := groupauth.SanitizeGroupName(req.Name)

	if sanitizedName == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Group name contains only invalid characters",
		})
		return
	}

	if len(sanitizedName) > 63 {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: fmt.Sprintf("Group name is too long. After sanitization, it must be 63 characters or less (current: %d). Use a shorter name.", len(sanitizedName)),
		})
		return
	}

	// Validate that the sanitized name matches the original (no truncation occurred)
	// This prevents two different group names from colliding after sanitization
	if sanitizedName != strings.ToLower(req.Name) {
		// Name was modified during sanitization - warn user
		logger.V(1).Info("Group name was sanitized",
			"original", req.Name,
			"sanitized", sanitizedName)
	}

	if len(req.ClusterPermissions) == 0 {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "At least one cluster permission is required",
		})
		return
	}

	// Validate actions
	if err := validateClusterPermissions(req.ClusterPermissions); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	// Convert to CRD ClusterPermissions format
	clusterPerms := make(map[string]krknv1alpha1.ClusterPermissionSet)
	for apiURL, permSet := range req.ClusterPermissions {
		clusterPerms[apiURL] = krknv1alpha1.ClusterPermissionSet{
			Actions: permSet.Actions,
		}
	}

	// Create KrknUserGroup CRD using sanitized name for consistency with labels
	// This ensures CR name matches the label suffix extracted by GetUserGroups
	group := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sanitizedName,
			Namespace: h.namespace,
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name:               req.Name, // Keep original name in spec for display purposes
			Description:        req.Description,
			ClusterPermissions: clusterPerms,
		},
	}

	if err := h.client.Create(ctx, group); err != nil {
		if apierrors.IsAlreadyExists(err) {
			writeJSONError(w, http.StatusConflict, ErrorResponse{
				Error:   "conflict",
				Message: fmt.Sprintf("User group '%s' already exists", req.Name),
			})
		} else {
			logger.Error(err, "Failed to create user group", "groupName", req.Name)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to create user group",
			})
		}
		return
	}

	logger.Info("Created user group", "groupName", req.Name, "clusterCount", len(req.ClusterPermissions))

	writeJSON(w, http.StatusCreated, CreateUserGroupResponse{
		Message: "User group created successfully",
		Name:    req.Name,
	})
}

// UpdateUserGroup handles PATCH /api/v1/groups/:groupName
// Updates a user group (admin only)
func (h *Handler) UpdateUserGroup(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("update-user-group")

	// Check admin privileges
	if !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Extract groupName from path
	groupName, err := extractPathSuffix(r.URL.Path, GroupsPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid groupName in path",
		})
		return
	}

	// Parse request body
	var req UpdateUserGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate actions if provided
	if req.ClusterPermissions != nil {
		if err := validateClusterPermissions(req.ClusterPermissions); err != nil {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: err.Error(),
			})
			return
		}
	}

	// Fetch existing group
	group := &krknv1alpha1.KrknUserGroup{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: groupName, Namespace: h.namespace}, group); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("User group '%s' not found", groupName),
			})
		} else {
			logger.Error(err, "Failed to get user group", "groupName", groupName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get user group",
			})
		}
		return
	}

	// Apply updates
	updated := false

	if req.Description != nil {
		group.Spec.Description = *req.Description
		updated = true
	}

	if req.ClusterPermissions != nil {
		clusterPerms := make(map[string]krknv1alpha1.ClusterPermissionSet)
		for apiURL, permSet := range req.ClusterPermissions {
			clusterPerms[apiURL] = krknv1alpha1.ClusterPermissionSet{
				Actions: permSet.Actions,
			}
		}
		group.Spec.ClusterPermissions = clusterPerms
		updated = true
	}

	if !updated {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "No fields to update",
		})
		return
	}

	// Update group
	if err := h.client.Update(ctx, group); err != nil {
		logger.Error(err, "Failed to update user group", "groupName", groupName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update user group",
		})
		return
	}

	logger.Info("Updated user group", "groupName", groupName)

	writeJSON(w, http.StatusOK, UpdateUserGroupResponse{
		Message: "User group updated successfully",
		Group:   buildUserGroupResponse(ctx, h.client, group, h.namespace),
	})
}

// DeleteUserGroup handles DELETE /api/v1/groups/:groupName
// Deletes a user group (admin only)
func (h *Handler) DeleteUserGroup(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("delete-user-group")

	// Check admin privileges
	if !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Extract groupName from path
	groupName, err := extractPathSuffix(r.URL.Path, GroupsPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid groupName in path",
		})
		return
	}

	// Fetch group
	group := &krknv1alpha1.KrknUserGroup{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: groupName, Namespace: h.namespace}, group); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("User group '%s' not found", groupName),
			})
		} else {
			logger.Error(err, "Failed to get user group", "groupName", groupName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get user group",
			})
		}
		return
	}

	// Remove group labels from all members
	if err := removeGroupFromAllMembers(ctx, h.client, groupName, h.namespace); err != nil {
		logger.Error(err, "Failed to remove group labels from users", "groupName", groupName)
		// Continue with deletion even if label removal fails
	}

	// Delete group
	if err := h.client.Delete(ctx, group); err != nil {
		logger.Error(err, "Failed to delete user group", "groupName", groupName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete user group",
		})
		return
	}

	logger.Info("Deleted user group", "groupName", groupName)

	writeJSON(w, http.StatusOK, DeleteUserGroupResponse{
		Message: "User group deleted successfully",
	})
}

// ListGroupMembers handles GET /api/v1/groups/:groupName/members
// Lists all members of a group (admin only)
func (h *Handler) ListGroupMembers(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("list-group-members")

	// Check admin privileges
	if !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Extract groupName from path
	groupName, err := extractGroupNameFromMembersPath(r.URL.Path)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	// Verify group exists
	group := &krknv1alpha1.KrknUserGroup{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: groupName, Namespace: h.namespace}, group); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("User group '%s' not found", groupName),
			})
		} else {
			logger.Error(err, "Failed to get user group", "groupName", groupName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get user group",
			})
		}
		return
	}

	// List users with group label
	labelKey := groupauth.GroupLabelKey(groupName)
	userList := &krknv1alpha1.KrknUserList{}
	err = h.client.List(ctx, userList,
		client.InNamespace(h.namespace),
		client.MatchingLabels{labelKey: "true"},
	)

	if err != nil {
		logger.Error(err, "Failed to list group members", "groupName", groupName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list group members",
		})
		return
	}

	// Convert to response format
	members := make([]UserResponse, len(userList.Items))
	for i, user := range userList.Items {
		members[i] = buildUserResponse(&user)
	}

	logger.Info("Listed group members", "groupName", groupName, "memberCount", len(members))

	writeJSON(w, http.StatusOK, ListGroupMembersResponse{
		Members:   members,
		Total:     len(members),
		GroupName: groupName,
	})
}

// AddGroupMember handles POST /api/v1/groups/:groupName/members
// Adds a user to a group by adding label (admin only)
func (h *Handler) AddGroupMember(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("add-group-member")

	// Check admin privileges
	if !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Extract groupName from path
	groupName, err := extractGroupNameFromMembersPath(r.URL.Path)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	// Parse request body
	var req AddGroupMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	if req.UserID == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "UserID is required",
		})
		return
	}

	// Verify group exists
	group := &krknv1alpha1.KrknUserGroup{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: groupName, Namespace: h.namespace}, group); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("User group '%s' not found", groupName),
			})
		} else {
			logger.Error(err, "Failed to get user group", "groupName", groupName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get user group",
			})
		}
		return
	}

	// Fetch user
	userName := sanitizeUsername(req.UserID)
	user := &krknv1alpha1.KrknUser{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: userName, Namespace: h.namespace}, user); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("User '%s' not found", req.UserID),
			})
		} else {
			logger.Error(err, "Failed to get user", "userID", req.UserID)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get user",
			})
		}
		return
	}

	// Add group label
	labelKey := groupauth.GroupLabelKey(groupName)
	if user.Labels == nil {
		user.Labels = make(map[string]string)
	}

	if user.Labels[labelKey] == "true" {
		writeJSONError(w, http.StatusConflict, ErrorResponse{
			Error:   "conflict",
			Message: fmt.Sprintf("User '%s' is already a member of group '%s'", req.UserID, groupName),
		})
		return
	}

	user.Labels[labelKey] = "true"

	if err := h.client.Update(ctx, user); err != nil {
		logger.Error(err, "Failed to add group label to user", "userID", req.UserID, "groupName", groupName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to add user to group",
		})
		return
	}

	logger.Info("Added user to group", "userID", req.UserID, "groupName", groupName)

	writeJSON(w, http.StatusOK, AddGroupMemberResponse{
		Message:   "User added to group successfully",
		UserID:    req.UserID,
		GroupName: groupName,
	})
}

// RemoveGroupMember handles DELETE /api/v1/groups/:groupName/members/:userId
// Removes a user from a group by removing label (admin only)
func (h *Handler) RemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	logger := log.FromContext(ctx).WithName("remove-group-member")

	// Check admin privileges
	if !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	// Extract groupName and userID from path
	groupName, userID, err := extractGroupNameAndUserIDFromPath(r.URL.Path)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	// Verify group exists
	group := &krknv1alpha1.KrknUserGroup{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: groupName, Namespace: h.namespace}, group); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("User group '%s' not found", groupName),
			})
		} else {
			logger.Error(err, "Failed to get user group", "groupName", groupName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get user group",
			})
		}
		return
	}

	// Fetch user
	userName := sanitizeUsername(userID)
	user := &krknv1alpha1.KrknUser{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: userName, Namespace: h.namespace}, user); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("User '%s' not found", userID),
			})
		} else {
			logger.Error(err, "Failed to get user", "userID", userID)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get user",
			})
		}
		return
	}

	// Remove group label
	labelKey := groupauth.GroupLabelKey(groupName)
	if user.Labels == nil || user.Labels[labelKey] != "true" {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: fmt.Sprintf("User '%s' is not a member of group '%s'", userID, groupName),
		})
		return
	}

	delete(user.Labels, labelKey)

	if err := h.client.Update(ctx, user); err != nil {
		logger.Error(err, "Failed to remove group label from user", "userID", userID, "groupName", groupName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to remove user from group",
		})
		return
	}

	logger.Info("Removed user from group", "userID", userID, "groupName", groupName)

	writeJSON(w, http.StatusOK, RemoveGroupMemberResponse{
		Message: "User removed from group successfully",
	})
}

// GroupsRouter routes requests to /api/v1/groups endpoints
func (h *Handler) GroupsRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Normalize path by removing trailing slash for root endpoint
	normalizedPath := strings.TrimSuffix(path, "/")

	// Root endpoint: /api/v1/groups or /api/v1/groups/
	if normalizedPath == GroupsPath {
		if r.Method == http.MethodGet {
			h.ListUserGroups(w, r)
			return
		}

		if r.Method == http.MethodPost {
			h.CreateUserGroup(w, r)
			return
		}

		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only GET and POST are allowed on " + GroupsPath,
		})
		return
	}

	// Group-specific endpoints: /api/v1/groups/:groupName
	if strings.HasPrefix(path, GroupsPath+"/") {
		// Members endpoints: /api/v1/groups/:groupName/members
		// Use HasSuffix or segment matching to avoid matching group names containing "members"
		pathAfterGroups := strings.TrimPrefix(path, GroupsPath+"/")
		segments := strings.Split(pathAfterGroups, "/")

		// Check if this is a members endpoint (second segment is "members")
		if len(segments) >= 2 && segments[1] == "members" {
			// DELETE /api/v1/groups/:groupName/members/:userId
			if r.Method == http.MethodDelete && strings.Count(path, "/") == 6 {
				h.RemoveGroupMember(w, r)
				return
			}

			// GET /api/v1/groups/:groupName/members
			if r.Method == http.MethodGet && strings.HasSuffix(path, "/members") {
				h.ListGroupMembers(w, r)
				return
			}

			// POST /api/v1/groups/:groupName/members
			if r.Method == http.MethodPost && strings.HasSuffix(path, "/members") {
				h.AddGroupMember(w, r)
				return
			}

			writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
				Error:   "method_not_allowed",
				Message: "Invalid method for members endpoint",
			})
			return
		}

		// GET /api/v1/groups/:groupName
		if r.Method == http.MethodGet {
			h.GetUserGroup(w, r)
			return
		}

		// PATCH /api/v1/groups/:groupName
		if r.Method == http.MethodPatch {
			h.UpdateUserGroup(w, r)
			return
		}

		// DELETE /api/v1/groups/:groupName
		if r.Method == http.MethodDelete {
			h.DeleteUserGroup(w, r)
			return
		}

		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Invalid method for group endpoint",
		})
		return
	}

	writeJSONError(w, http.StatusNotFound, ErrorResponse{
		Error:   "not_found",
		Message: "Endpoint not found",
	})
}

// Helper functions

// buildUserGroupResponse converts a KrknUserGroup CR to UserGroupResponse
func buildUserGroupResponse(ctx context.Context, k8sClient client.Client, group *krknv1alpha1.KrknUserGroup, namespace string) UserGroupResponse {
	var createdAt *time.Time
	if !group.CreationTimestamp.IsZero() {
		t := group.CreationTimestamp.Time
		createdAt = &t
	}

	// Convert ClusterPermissions to API format
	clusterPerms := make(map[string]ClusterPermissionSet)
	for apiURL, permSet := range group.Spec.ClusterPermissions {
		clusterPerms[apiURL] = ClusterPermissionSet{
			Actions: permSet.Actions,
		}
	}

	// Count members
	memberCount, _ := groupauth.CountGroupMembers(ctx, k8sClient, group.Name, namespace)

	return UserGroupResponse{
		Name:               group.Spec.Name,
		Description:        group.Spec.Description,
		ClusterPermissions: clusterPerms,
		MemberCount:        memberCount,
		CreatedAt:          createdAt,
	}
}

// validateClusterPermissions validates that all actions are valid
func validateClusterPermissions(permissions map[string]ClusterPermissionSet) error {
	for apiURL, permSet := range permissions {
		if len(permSet.Actions) == 0 {
			return fmt.Errorf("cluster %s must have at least one action", apiURL)
		}

		for _, action := range permSet.Actions {
			if !groupauth.IsValidAction(action) {
				return fmt.Errorf("invalid action '%s' for cluster %s. Valid actions: view, run, cancel", action, apiURL)
			}
		}
	}
	return nil
}

// extractGroupNameFromMembersPath extracts group name from /api/v1/groups/:groupName/members
func extractGroupNameFromMembersPath(path string) (string, error) {
	// Remove prefix and suffix
	trimmed := strings.TrimPrefix(path, GroupsPath+"/")
	trimmed = strings.TrimSuffix(trimmed, "/members")

	if trimmed == "" || strings.Contains(trimmed, "/") {
		return "", fmt.Errorf("invalid path format")
	}

	return trimmed, nil
}

// extractGroupNameAndUserIDFromPath extracts group name and userID from /api/v1/groups/:groupName/members/:userId
func extractGroupNameAndUserIDFromPath(path string) (string, string, error) {
	// Expected format: /api/v1/groups/:groupName/members/:userId
	parts := strings.Split(strings.TrimPrefix(path, GroupsPath+"/"), "/")

	if len(parts) != 3 || parts[1] != "members" {
		return "", "", fmt.Errorf("invalid path format. Expected: " + GroupsPath + "/:groupName/members/:userId")
	}

	groupName := parts[0]
	userID := parts[2]

	if groupName == "" || userID == "" {
		return "", "", fmt.Errorf("groupName and userId cannot be empty")
	}

	return groupName, userID, nil
}

// removeGroupFromAllMembers removes the group label from all users
func removeGroupFromAllMembers(ctx context.Context, k8sClient client.Client, groupName, namespace string) error {
	labelKey := groupauth.GroupLabelKey(groupName)

	// List all users with this group label
	userList := &krknv1alpha1.KrknUserList{}
	err := k8sClient.List(ctx, userList,
		client.InNamespace(namespace),
		client.MatchingLabels{labelKey: "true"},
	)

	if err != nil {
		return fmt.Errorf("failed to list users with group label: %w", err)
	}

	// Remove label from each user
	for i := range userList.Items {
		user := &userList.Items[i]
		delete(user.Labels, labelKey)

		if err := k8sClient.Update(ctx, user); err != nil {
			// Log but continue
			log.FromContext(ctx).Error(err, "Failed to remove group label from user",
				"userID", user.Spec.UserID,
				"groupName", groupName,
			)
		}
	}

	return nil
}
