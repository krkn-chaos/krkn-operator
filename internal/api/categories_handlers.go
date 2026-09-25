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

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/api/v2"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

const maxCategoryNameLength = 63

const (
	categoryAvailableToAllLabel = "categories.krkn.krkn-chaos.dev/available-to-all"
	categoryCreatedByAnnotation = "categories.krkn.krkn-chaos.dev/created-by"
)

var categoryColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// CategoryResponse represents the public fields of a KrknCategory.
type CategoryResponse struct {
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
	// Groups contains the single group allowed to view the category, when group-scoped.
	Groups []string `json:"groups,omitempty"`
	// AvailableToAll is true for public categories. Categories created without a group default to public.
	AvailableToAll bool `json:"availableToAll"`
	// CreatedBy identifies the user allowed to update or delete the category, unless the caller is an admin.
	CreatedBy string `json:"createdBy,omitempty"`
}

// CategoryListResponse represents a list of categories.
type CategoryListResponse struct {
	Categories []CategoryResponse `json:"categories"`
	Total      int                `json:"total"`
}

// CreateCategoryRequest represents a request to create a category.
type CreateCategoryRequest struct {
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
	// Groups may contain one group. Omit it to create a public category.
	Groups []string `json:"groups,omitempty"`
	// AvailableToAll makes the category public. It cannot be combined with Groups.
	AvailableToAll bool `json:"availableToAll"`
}

// UpdateCategoryRequest represents a request to update a category's mutable fields.
type UpdateCategoryRequest struct {
	Color string `json:"color,omitempty"`
	// Groups and AvailableToAll must be sent together to change visibility. When omitted, the current visibility is preserved.
	Groups         *[]string `json:"groups,omitempty"`
	AvailableToAll *bool     `json:"availableToAll,omitempty"`
}

// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krkncategories,verbs=get;list;watch;create;update;patch;delete

// CategoriesRouter routes requests to /api/v2/categories endpoints.
func (h *Handler) CategoriesRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	if path == v2.CategoriesPath {
		switch r.Method {
		case http.MethodGet:
			h.ListCategories(w, r)
		case http.MethodPost:
			h.CreateCategory(w, r)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
				Error:   "method_not_allowed",
				Message: "Only GET and POST are allowed on " + v2.CategoriesPath,
			})
		}
		return
	}

	if !strings.HasPrefix(path, v2.CategoriesPath+"/") {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Endpoint not found",
		})
		return
	}

	name := strings.TrimPrefix(path, v2.CategoriesPath+"/")
	if name == "" || strings.Contains(name, "/") {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "A single category name is required",
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.GetCategory(w, r, name)
	case http.MethodPut:
		h.UpdateCategory(w, r, name)
	case http.MethodDelete:
		h.DeleteCategory(w, r, name)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only GET, PUT, and DELETE are allowed for a category",
		})
	}
}

// ListCategories handles GET /api/v2/categories.
//
// @Summary List categories
// @Description List public categories, categories in the caller's groups, and categories created by the caller. Admins can list all categories.
// @Tags categories
// @Produce json
// @Success 200 {object} CategoryListResponse "Categories"
// @Failure 401 {object} ErrorResponse "Authentication required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v2/categories [get]
func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("list-categories")
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
		return
	}

	userGroups := map[string]struct{}{}
	if !auth.IsAdmin(ctx) {
		var err error
		userGroups, err = h.getCategoryUserGroups(ctx, claims.UserID)
		if err != nil {
			logger.Error(err, "Failed to get user groups while listing categories")
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to determine category visibility",
			})
			return
		}
	}

	var categoryList krknv1alpha1.KrknCategoryList
	if err := h.client.List(ctx, &categoryList, client.InNamespace(h.namespace)); err != nil {
		logger.Error(err, "Failed to list categories")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list categories",
		})
		return
	}

	categories := make([]CategoryResponse, 0, len(categoryList.Items))
	for i := range categoryList.Items {
		category := &categoryList.Items[i]
		if auth.IsAdmin(ctx) || categoryVisibleToUser(category, claims.UserID, userGroups) {
			categories = append(categories, buildCategoryResponse(category))
		}
	}
	sort.Slice(categories, func(i, j int) bool { return categories[i].Name < categories[j].Name })

	writeJSON(w, http.StatusOK, CategoryListResponse{
		Categories: categories,
		Total:      len(categories),
	})
}

// CreateCategory handles POST /api/v2/categories.
// Authenticated users can create public categories or categories for their own groups.
//
// @Summary Create a category
// @Description Create a public category or a category visible to one of the caller's groups. If no group is specified, the category is public. Admins may select any existing group.
// @Tags categories
// @Accept json
// @Produce json
// @Param request body CreateCategoryRequest true "Category definition"
// @Success 201 {object} CategoryResponse "Created category"
// @Failure 400 {object} ErrorResponse "Invalid category"
// @Failure 401 {object} ErrorResponse "Authentication required"
// @Failure 403 {object} ErrorResponse "The caller does not belong to the selected group"
// @Failure 409 {object} ErrorResponse "Category already exists"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v2/categories [post]
func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("create-category")
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
		return
	}

	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}
	if err := validateCategoryRequest(req.Name, req.Color); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}
	if len(req.Groups) == 0 && !req.AvailableToAll {
		// Preserve the original API behavior: categories without visibility
		// metadata were readable by every authenticated user.
		req.AvailableToAll = true
	}
	isAdmin := auth.IsAdmin(ctx)
	if status, err := h.validateCategoryVisibility(ctx, claims.UserID, isAdmin, req.Groups, req.AvailableToAll); err != nil {
		if status == http.StatusInternalServerError {
			logger.Error(err, "Failed to validate category visibility", "name", req.Name)
		}
		writeJSONError(w, status, ErrorResponse{
			Error:   categoryErrorCode(status),
			Message: err.Error(),
		})
		return
	}

	category := &krknv1alpha1.KrknCategory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: h.namespace,
			Labels:    buildCategoryVisibilityLabels(req.Groups, req.AvailableToAll),
			Annotations: map[string]string{
				categoryCreatedByAnnotation: claims.UserID,
			},
		},
		Spec: krknv1alpha1.KrknCategorySpec{Color: req.Color},
	}
	if err := h.client.Create(ctx, category); err != nil {
		if apierrors.IsAlreadyExists(err) {
			writeJSONError(w, http.StatusConflict, ErrorResponse{
				Error:   "conflict",
				Message: fmt.Sprintf("Category '%s' already exists", req.Name),
			})
			return
		}
		logger.Error(err, "Failed to create category", "name", req.Name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create category",
		})
		return
	}

	writeJSON(w, http.StatusCreated, buildCategoryResponse(category))
}

// GetCategory handles GET /api/v2/categories/{name}.
//
// @Summary Get a category
// @Description Get a public category, a category in the caller's groups, or a category created by the caller. Admins can get all categories.
// @Tags categories
// @Produce json
// @Param name path string true "Category name"
// @Success 200 {object} CategoryResponse "Category"
// @Failure 400 {object} ErrorResponse "Invalid category name"
// @Failure 401 {object} ErrorResponse "Authentication required"
// @Failure 403 {object} ErrorResponse "The caller cannot access this category"
// @Failure 404 {object} ErrorResponse "Category not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v2/categories/{name} [get]
func (h *Handler) GetCategory(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("get-category")
	if err := validateCategoryName(name); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}
	category := &krknv1alpha1.KrknCategory{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: name, Namespace: h.namespace}, category); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Category '%s' not found", name),
			})
		} else {
			logger.Error(err, "Failed to get category", "name", name)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get category",
			})
		}
		return
	}
	if !auth.IsAdmin(ctx) {
		claims := auth.GetClaimsFromContext(ctx)
		if claims == nil {
			writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
				Error:   "unauthorized",
				Message: "Authentication required",
			})
			return
		}
		canView, err := h.canViewCategory(ctx, category, claims.UserID)
		if err != nil {
			logger.Error(err, "Failed to determine category visibility", "name", name)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to determine category visibility",
			})
			return
		}
		if !canView {
			writeJSONError(w, http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have access to this category",
			})
			return
		}
	}
	writeJSON(w, http.StatusOK, buildCategoryResponse(category))
}

// UpdateCategory handles PUT /api/v2/categories/{name}.
//
// @Summary Update a category
// @Description Update a category's color and optionally its visibility. Category creators and admins may update it.
// @Tags categories
// @Accept json
// @Produce json
// @Param name path string true "Category name"
// @Param request body UpdateCategoryRequest true "Mutable category fields"
// @Success 200 {object} CategoryResponse "Updated category"
// @Failure 400 {object} ErrorResponse "Invalid category"
// @Failure 401 {object} ErrorResponse "Authentication required"
// @Failure 403 {object} ErrorResponse "Only the category creator or an admin may update it"
// @Failure 404 {object} ErrorResponse "Category not found"
// @Failure 409 {object} ErrorResponse "Category changed concurrently"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v2/categories/{name} [put]
func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("update-category")
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
		return
	}
	if err := validateCategoryName(name); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}
	if err := validateCategoryColor(req.Color); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	category := &krknv1alpha1.KrknCategory{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: name, Namespace: h.namespace}, category); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Category '%s' not found", name),
			})
		} else {
			logger.Error(err, "Failed to get category for update", "name", name)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get category",
			})
		}
		return
	}

	isAdmin := auth.IsAdmin(ctx)
	if !isAdmin && category.Annotations[categoryCreatedByAnnotation] != claims.UserID {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Only the category creator or an admin may update it",
		})
		return
	}

	groups := categoryGroups(category)
	availableToAll := categoryIsPublic(category)
	visibilityWasUpdated := req.Groups != nil || req.AvailableToAll != nil
	if visibilityWasUpdated {
		if req.Groups == nil || req.AvailableToAll == nil {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "groups and availableToAll must be provided together when changing visibility",
			})
			return
		}
		groups = *req.Groups
		availableToAll = *req.AvailableToAll
		if len(groups) == 0 && !availableToAll {
			availableToAll = true
		}
		if status, err := h.validateCategoryVisibility(ctx, claims.UserID, isAdmin, groups, availableToAll); err != nil {
			if status == http.StatusInternalServerError {
				logger.Error(err, "Failed to validate category visibility", "name", name)
			}
			writeJSONError(w, status, ErrorResponse{
				Error:   categoryErrorCode(status),
				Message: err.Error(),
			})
			return
		}
	}

	category.Spec.Color = req.Color
	if visibilityWasUpdated {
		category.Labels = replaceCategoryVisibilityLabels(category.Labels, groups, availableToAll)
	}
	if err := h.client.Update(ctx, category); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Category '%s' not found", name),
			})
			return
		}
		if apierrors.IsConflict(err) {
			writeJSONError(w, http.StatusConflict, ErrorResponse{
				Error:   "conflict",
				Message: "Category changed during update; retry the request",
			})
			return
		}
		logger.Error(err, "Failed to update category", "name", name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update category",
		})
		return
	}

	writeJSON(w, http.StatusOK, buildCategoryResponse(category))
}

// DeleteCategory handles DELETE /api/v2/categories/{name}.
//
// @Summary Delete a category
// @Description Delete a category created by the caller. Admins may delete any category.
// @Tags categories
// @Produce json
// @Param name path string true "Category name"
// @Success 200 {object} map[string]string "Deletion result"
// @Failure 400 {object} ErrorResponse "Invalid category name"
// @Failure 401 {object} ErrorResponse "Authentication required"
// @Failure 403 {object} ErrorResponse "Only the category creator or an admin may delete it"
// @Failure 404 {object} ErrorResponse "Category not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v2/categories/{name} [delete]
func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("delete-category")
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
		return
	}
	if err := validateCategoryName(name); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	category := &krknv1alpha1.KrknCategory{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: name, Namespace: h.namespace}, category); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Category '%s' not found", name),
			})
		} else {
			logger.Error(err, "Failed to get category for deletion", "name", name)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get category",
			})
		}
		return
	}
	if !auth.IsAdmin(ctx) && category.Annotations[categoryCreatedByAnnotation] != claims.UserID {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Only the category creator or an admin may delete it",
		})
		return
	}

	if err := h.client.Delete(ctx, category); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Category '%s' not found", name),
			})
			return
		}
		logger.Error(err, "Failed to delete category", "name", name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete category",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Category deleted successfully"})
}

func validateCategoryRequest(name, color string) error {
	if err := validateCategoryName(name); err != nil {
		return err
	}
	return validateCategoryColor(color)
}

func (h *Handler) validateCategoryVisibility(ctx context.Context, userID string, isAdmin bool, groups []string, availableToAll bool) (int, error) {
	if len(groups) > 1 {
		return http.StatusBadRequest, fmt.Errorf("a category can be assigned to at most one group")
	}
	if len(groups) > 0 && availableToAll {
		return http.StatusBadRequest, fmt.Errorf("a category cannot be both public and assigned to a group")
	}
	if len(groups) == 0 && !availableToAll {
		return http.StatusBadRequest, fmt.Errorf("a category must be public or assigned to one group")
	}

	for _, groupName := range groups {
		if len(groupName) > maxCategoryNameLength || len(validation.IsDNS1123Subdomain(groupName)) > 0 {
			return http.StatusBadRequest, fmt.Errorf("invalid group name %q", groupName)
		}
		group := &krknv1alpha1.KrknUserGroup{}
		err := h.client.Get(ctx, client.ObjectKey{Name: groupName, Namespace: h.namespace}, group)
		if apierrors.IsNotFound(err) {
			return http.StatusBadRequest, fmt.Errorf("group %q does not exist", groupName)
		}
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to validate group %q: %w", groupName, err)
		}
	}

	if isAdmin || len(groups) == 0 {
		return 0, nil
	}
	userGroups, err := h.getCategoryUserGroups(ctx, userID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to get caller's groups: %w", err)
	}
	if _, ok := userGroups[groups[0]]; !ok {
		return http.StatusForbidden, fmt.Errorf("you can only assign a category to a group you belong to")
	}
	return 0, nil
}

func (h *Handler) getCategoryUserGroups(ctx context.Context, userID string) (map[string]struct{}, error) {
	groups, err := groupauth.GetUserGroups(ctx, h.client, userID, h.namespace)
	if err != nil {
		return nil, fmt.Errorf("get user groups: %w", err)
	}

	groupNames := make(map[string]struct{}, len(groups))
	for i := range groups {
		groupNames[groups[i].Name] = struct{}{}
	}
	return groupNames, nil
}

func (h *Handler) canViewCategory(ctx context.Context, category *krknv1alpha1.KrknCategory, userID string) (bool, error) {
	userGroups, err := h.getCategoryUserGroups(ctx, userID)
	if err != nil {
		return false, err
	}
	return categoryVisibleToUser(category, userID, userGroups), nil
}

func categoryVisibleToUser(category *krknv1alpha1.KrknCategory, userID string, userGroups map[string]struct{}) bool {
	if category.Annotations[categoryCreatedByAnnotation] == userID || categoryIsPublic(category) {
		return true
	}
	for _, groupName := range categoryGroups(category) {
		if _, ok := userGroups[groupName]; ok {
			return true
		}
	}
	return false
}

func categoryGroups(category *krknv1alpha1.KrknCategory) []string {
	groups := groupauth.ExtractGroupNamesFromLabels(category.Labels)
	sort.Strings(groups)
	return groups
}

func categoryIsPublic(category *krknv1alpha1.KrknCategory) bool {
	if category.Labels[categoryAvailableToAllLabel] == "true" {
		return true
	}
	// Categories created before visibility was introduced were public to all
	// authenticated users, so keep that behavior for legacy objects.
	return len(categoryGroups(category)) == 0
}

func buildCategoryVisibilityLabels(groups []string, availableToAll bool) map[string]string {
	return replaceCategoryVisibilityLabels(nil, groups, availableToAll)
}

func replaceCategoryVisibilityLabels(existing map[string]string, groups []string, availableToAll bool) map[string]string {
	labels := make(map[string]string, len(existing)+len(groups)+1)
	for key, value := range existing {
		if key == categoryAvailableToAllLabel || strings.HasPrefix(key, groupauth.GroupLabelPrefix) {
			continue
		}
		labels[key] = value
	}
	if availableToAll {
		labels[categoryAvailableToAllLabel] = "true"
	}
	for _, groupName := range groups {
		labels[groupauth.GroupLabelKey(groupName)] = "true"
	}
	return labels
}

func categoryErrorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusForbidden:
		return "forbidden"
	default:
		return "internal_error"
	}
}

func validateCategoryName(name string) error {
	if len(name) > maxCategoryNameLength {
		return fmt.Errorf("category name must not exceed %d characters", maxCategoryNameLength)
	}
	if problems := validation.IsDNS1123Label(name); len(problems) > 0 {
		return fmt.Errorf("invalid category name %q: %s", name, strings.Join(problems, "; "))
	}
	return nil
}

func validateCategoryColor(color string) error {
	if color != "" && !categoryColorPattern.MatchString(color) {
		return fmt.Errorf("invalid color format %q (must be #RRGGBB)", color)
	}
	return nil
}

func buildCategoryResponse(category *krknv1alpha1.KrknCategory) CategoryResponse {
	return CategoryResponse{
		Name:           category.Name,
		Color:          category.Spec.Color,
		Groups:         categoryGroups(category),
		AvailableToAll: categoryIsPublic(category),
		CreatedBy:      category.Annotations[categoryCreatedByAnnotation],
	}
}
