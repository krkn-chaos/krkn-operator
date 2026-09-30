package api

import (
	"context"
	"fmt"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

// addRequestedRunCategoryLabels validates a run's complete category selection
// before adding any labels. This keeps category assignment in the same Create
// operation as the run itself and avoids partially categorized CRs.
func (h *Handler) addRequestedRunCategoryLabels(w http.ResponseWriter, r *http.Request, categories []string, labels map[string]string) bool {
	if len(categories) == 0 {
		return true
	}

	ctx := r.Context()
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication required to assign categories",
		})
		return false
	}

	selectedLabels := make(map[string]string, len(categories))
	seen := make(map[string]struct{}, len(categories))
	for _, categoryName := range categories {
		if err := validateCategoryName(categoryName); err != nil {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: err.Error(),
			})
			return false
		}
		if _, exists := seen[categoryName]; exists {
			continue
		}
		seen[categoryName] = struct{}{}

		category := &krknv1alpha1.KrknCategory{}
		if err := h.client.Get(ctx, client.ObjectKey{Name: categoryName, Namespace: h.namespace}, category); err != nil {
			if apierrors.IsNotFound(err) {
				writeJSONError(w, http.StatusNotFound, ErrorResponse{
					Error:   "not_found",
					Message: fmt.Sprintf("Category %q not found", categoryName),
				})
				return false
			}
			log.FromContext(ctx).Error(err, "Failed to load selected run category", "category", categoryName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to validate selected categories",
			})
			return false
		}

		if !auth.IsAdmin(ctx) {
			canView, err := h.canViewCategory(ctx, category, claims.UserID)
			if err != nil {
				log.FromContext(ctx).Error(err, "Failed to determine selected run category visibility", "category", categoryName)
				writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
					Error:   "internal_error",
					Message: "Failed to validate selected categories",
				})
				return false
			}
			if !canView {
				writeJSONError(w, http.StatusForbidden, ErrorResponse{
					Error:   "forbidden",
					Message: "You do not have access to this category",
				})
				return false
			}
		}

		selectedLabels[krknv1alpha1.CategoryEntityLabelPrefix+categoryName] = "true"
	}

	if labels == nil {
		labels = make(map[string]string, len(selectedLabels))
	}
	for key, value := range selectedLabels {
		labels[key] = value
	}
	return true
}

func (h *Handler) visibleRunCategoryNames(ctx context.Context, labels map[string]string) ([]string, error) {
	names := runCategoryNames(labels)
	if len(names) == 0 {
		return nil, nil
	}

	visible, err := h.VisibleCategoryNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("load visible categories for run replay: %w", err)
	}

	filtered := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := visible[name]; ok {
			filtered = append(filtered, name)
		}
	}
	return filtered, nil
}
