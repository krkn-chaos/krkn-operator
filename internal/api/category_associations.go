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
	"fmt"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

const (
	categoryEntityTypeScenarioRun = "scenario-runs"
	categoryEntityTypeGraphRun    = "graph-runs"
)

// removeCategoryAssociations removes the category label from all associated runs.
// It is safe to retry after a partial failure because already-removed labels are
// no longer selected by the list queries.
func (h *Handler) removeCategoryAssociations(ctx context.Context, categoryName string) error {
	labelKey := krknv1alpha1.CategoryEntityLabelPrefix + categoryName
	listOptions := []client.ListOption{
		client.InNamespace(h.namespace),
		client.MatchingLabels{labelKey: "true"},
	}

	scenarioRuns := &krknv1alpha1.KrknScenarioRunList{}
	if err := h.client.List(ctx, scenarioRuns, listOptions...); err != nil {
		return fmt.Errorf("list KrknScenarioRun associations: %w", err)
	}
	for i := range scenarioRuns.Items {
		run := &scenarioRuns.Items[i]
		original := run.DeepCopy()
		removeCategoryLabel(run, labelKey)
		if err := h.client.Patch(ctx, run, client.MergeFrom(original)); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("remove category label from KrknScenarioRun %q: %w", run.Name, err)
		}
	}

	graphRuns := &krknv1alpha1.KrknGraphRunList{}
	if err := h.client.List(ctx, graphRuns, listOptions...); err != nil {
		return fmt.Errorf("list KrknGraphRun associations: %w", err)
	}
	for i := range graphRuns.Items {
		run := &graphRuns.Items[i]
		original := run.DeepCopy()
		removeCategoryLabel(run, labelKey)
		if err := h.client.Patch(ctx, run, client.MergeFrom(original)); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("remove category label from KrknGraphRun %q: %w", run.Name, err)
		}
	}

	return nil
}

func removeCategoryLabel(object client.Object, labelKey string) {
	labels := object.GetLabels()
	delete(labels, labelKey)
	if len(labels) == 0 {
		labels = nil
	}
	object.SetLabels(labels)
}

// CategoryEntityAssociationResponse describes a category association change.
type CategoryEntityAssociationResponse struct {
	Category   string `json:"category"`
	EntityType string `json:"entityType"`
	EntityName string `json:"entityName"`
	Associated bool   `json:"associated"`
}

// AssociateCategoryEntity handles PUT /api/v2/categories/{category}/entities/{entityType}/{entityName}.
//
// @Summary Associate a run with a category
// @Description Add one category label to a scenario or graph run. Existing category and metadata labels are preserved.
// @Tags categories
// @Produce json
// @Param category path string true "Category name"
// @Param entityType path string true "Run type" Enums(scenario-runs, graph-runs)
// @Param entityName path string true "Run name"
// @Success 200 {object} CategoryEntityAssociationResponse "Association created or already present"
// @Failure 400 {object} ErrorResponse "Invalid category, run name, or run type"
// @Failure 401 {object} ErrorResponse "Authentication required"
// @Failure 403 {object} ErrorResponse "The caller cannot access the category or run"
// @Failure 404 {object} ErrorResponse "Category or run not found"
// @Failure 409 {object} ErrorResponse "Run changed concurrently"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Failure 503 {object} ErrorResponse "Run operations are temporarily unavailable during CRD migration"
// @Security BearerAuth
// @Router /v2/categories/{category}/entities/{entityType}/{entityName} [put]
func (h *Handler) AssociateCategoryEntity(w http.ResponseWriter, r *http.Request, categoryName, entityType, entityName string) {
	h.changeCategoryEntityAssociation(w, r, categoryName, entityType, entityName, true)
}

// UnassociateCategoryEntity handles DELETE /api/v2/categories/{category}/entities/{entityType}/{entityName}.
//
// @Summary Remove a run from a category
// @Description Remove one category label from a scenario or graph run. Other category and metadata labels are preserved.
// @Tags categories
// @Produce json
// @Param category path string true "Category name"
// @Param entityType path string true "Run type" Enums(scenario-runs, graph-runs)
// @Param entityName path string true "Run name"
// @Success 200 {object} CategoryEntityAssociationResponse "Association removed or already absent"
// @Failure 400 {object} ErrorResponse "Invalid category, run name, or run type"
// @Failure 401 {object} ErrorResponse "Authentication required"
// @Failure 403 {object} ErrorResponse "The caller cannot access the category or run"
// @Failure 404 {object} ErrorResponse "Category or run not found"
// @Failure 409 {object} ErrorResponse "Run changed concurrently"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Failure 503 {object} ErrorResponse "Run operations are temporarily unavailable during CRD migration"
// @Security BearerAuth
// @Router /v2/categories/{category}/entities/{entityType}/{entityName} [delete]
func (h *Handler) UnassociateCategoryEntity(w http.ResponseWriter, r *http.Request, categoryName, entityType, entityName string) {
	h.changeCategoryEntityAssociation(w, r, categoryName, entityType, entityName, false)
}

func (h *Handler) changeCategoryEntityAssociation(
	w http.ResponseWriter,
	r *http.Request,
	categoryName, entityType, entityName string,
	associate bool,
) {
	if h.rejectRunWritesDuringCRDMigration(w, r) {
		return
	}
	ctx := r.Context()
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required",
		})
		return
	}
	if err := validateCategoryName(categoryName); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}
	if entityType != categoryEntityTypeScenarioRun && entityType != categoryEntityTypeGraphRun {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "entityType must be scenario-runs or graph-runs",
		})
		return
	}
	if problems := validation.IsDNS1123Subdomain(entityName); len(problems) > 0 {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: fmt.Sprintf("invalid run name %q: %s", entityName, strings.Join(problems, "; ")),
		})
		return
	}

	logger := log.FromContext(ctx).WithName("category-association").WithValues(
		"category", categoryName,
		"entityType", entityType,
		"entityName", entityName,
	)
	category := &krknv1alpha1.KrknCategory{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: categoryName, Namespace: h.namespace}, category); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Category %q not found", categoryName),
			})
			return
		}
		logger.Error(err, "Failed to load category")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to load category",
		})
		return
	}

	if !auth.IsAdmin(ctx) {
		canView, err := h.canViewCategory(ctx, category, claims.UserID)
		if err != nil {
			logger.Error(err, "Failed to determine category visibility")
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

	var object client.Object
	entityKind := ""
	switch entityType {
	case categoryEntityTypeScenarioRun:
		run := &krknv1alpha1.KrknScenarioRun{}
		if err := h.client.Get(ctx, client.ObjectKey{Name: entityName, Namespace: h.namespace}, run); err != nil {
			writeCategoryRunNotFoundOrInternalError(w, logger, err, entityType, entityName)
			return
		}
		if !h.checkScenarioRunAccess(w, r, run) {
			return
		}
		object = run
		entityKind = "KrknScenarioRun"
	case categoryEntityTypeGraphRun:
		run := &krknv1alpha1.KrknGraphRun{}
		if err := h.client.Get(ctx, client.ObjectKey{Name: entityName, Namespace: h.namespace}, run); err != nil {
			writeCategoryRunNotFoundOrInternalError(w, logger, err, entityType, entityName)
			return
		}
		if !auth.IsAdmin(ctx) {
			hasAccess, err := h.checkGraphRunGroupAccess(ctx, claims.UserID, run, groupauth.ActionView)
			if err != nil {
				logger.Error(err, "Failed to check graph run access")
				writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
					Error:   "internal_error",
					Message: "Failed to validate graph run access",
				})
				return
			}
			if !hasAccess {
				writeJSONError(w, http.StatusForbidden, ErrorResponse{
					Error:   "forbidden",
					Message: "You do not have permission to view this graph run",
				})
				return
			}
		}
		object = run
		entityKind = "KrknGraphRun"
	}

	labelKey := krknv1alpha1.CategoryEntityLabelPrefix + categoryName
	labels := object.GetLabels()
	labelValue, labelExists := labels[labelKey]
	if (associate && labelValue == "true") || (!associate && !labelExists) {
		writeJSON(w, http.StatusOK, CategoryEntityAssociationResponse{
			Category: categoryName, EntityType: entityKind, EntityName: entityName, Associated: associate,
		})
		return
	}

	original := object.DeepCopyObject().(client.Object)
	if labels == nil {
		labels = map[string]string{}
	}
	if associate {
		labels[labelKey] = "true"
	} else {
		delete(labels, labelKey)
		if len(labels) == 0 {
			labels = nil
		}
	}
	object.SetLabels(labels)
	if err := h.client.Patch(ctx, object, client.MergeFrom(original)); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Run %q not found", entityName),
			})
			return
		}
		if apierrors.IsConflict(err) {
			writeJSONError(w, http.StatusConflict, ErrorResponse{
				Error:   "conflict",
				Message: "Run changed during category association; retry the request",
			})
			return
		}
		logger.Error(err, "Failed to update category association")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update category association",
		})
		return
	}

	writeJSON(w, http.StatusOK, CategoryEntityAssociationResponse{
		Category: categoryName, EntityType: entityKind, EntityName: entityName, Associated: associate,
	})
}

func writeCategoryRunNotFoundOrInternalError(
	w http.ResponseWriter,
	logger logr.Logger,
	err error,
	entityType, entityName string,
) {
	if apierrors.IsNotFound(err) {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: fmt.Sprintf("Run %q not found", entityName),
		})
		return
	}
	logger.Error(err, "Failed to load run", "entityType", entityType, "entityName", entityName)
	writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
		Error:   "internal_error",
		Message: "Failed to load run",
	})
}
