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
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"unicode/utf8"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

// robotTokenPathSegment is the sub-resource segment of the robot token route.
const robotTokenPathSegment = "token"

// maxRobotDescriptionLength mirrors the KrknRobotSpec.Description CRD
// validation (MaxLength=256). The CRD limit counts characters, not bytes, so
// the handlers must measure with utf8.RuneCountInString to stay consistent.
const maxRobotDescriptionLength = 256

// errGroupLookup marks a group lookup failure that is not simply a missing
// group (API failure, RBAC denial); it maps to a 500 rather than a 400.
var errGroupLookup = errors.New("failed to look up group")

// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknrobots,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=krkn.krkn-chaos.dev,resources=krknrobots/status,verbs=get;update;patch

// RobotsRouter routes /api/v1/robots admin endpoints. Every method requires
// admin privileges; non-admin callers receive 403 before any decode.
func (h *Handler) RobotsRouter(w http.ResponseWriter, r *http.Request) {
	if !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "This operation requires admin privileges",
		})
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	switch {
	case path == RobotsPath:
		h.robotsRoot(w, r)
	case strings.HasPrefix(path, RobotsPath+"/"):
		h.robotsItem(w, r, strings.TrimPrefix(path, RobotsPath+"/"))
	default:
		writeJSONError(w, http.StatusNotFound, ErrorResponse{Error: "not_found", Message: "Not found"})
	}
}

// robotsRoot handles GET/POST on /api/v1/robots.
func (h *Handler) robotsRoot(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListRobots(w, r)
	case http.MethodPost:
		h.CreateRobot(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only GET and POST are allowed on " + RobotsPath,
		})
	}
}

// robotsItem routes /api/v1/robots/{name}[/token].
func (h *Handler) robotsItem(w http.ResponseWriter, r *http.Request, rest string) {
	segments := strings.Split(rest, "/")
	name := segments[0]

	if len(segments) == 2 && segments[1] == robotTokenPathSegment {
		switch r.Method {
		case http.MethodPost:
			h.IssueRobotToken(w, r, name)
		case http.MethodDelete:
			h.RevokeRobotToken(w, r, name)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
				Error:   "method_not_allowed",
				Message: "Only POST and DELETE are allowed on robot token endpoints",
			})
		}
		return
	}

	if len(segments) != 1 {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{Error: "not_found", Message: "Not found"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.GetRobot(w, r, name)
	case http.MethodPatch:
		h.UpdateRobot(w, r, name)
	case http.MethodDelete:
		h.DeleteRobot(w, r, name)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only GET, PATCH and DELETE are allowed on robot resources",
		})
	}
}

// ListRobots handles GET /api/v1/robots
//
// @Summary List robot accounts
// @Description List all robot accounts (admin only)
// @Tags robots
// @Produce json
// @Success 200 {object} RobotListResponse "List of robots"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /robots [get]
func (h *Handler) ListRobots(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("list-robots")

	list := &krknv1alpha1.KrknRobotList{}
	if err := h.client.List(ctx, list, client.InNamespace(h.namespace)); err != nil {
		logger.Error(err, "Failed to list robots")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to list robots"})
		return
	}

	sort.Slice(list.Items, func(i, j int) bool { return list.Items[i].Name < list.Items[j].Name })
	robots := make([]RobotResponse, 0, len(list.Items))
	for i := range list.Items {
		robots = append(robots, robotToResponse(&list.Items[i]))
	}
	writeJSON(w, http.StatusOK, RobotListResponse{Robots: robots, Total: len(robots)})
}

// CreateRobot handles POST /api/v1/robots
//
// @Summary Create a robot account
// @Description Create a robot account with optional group memberships (admin only)
// @Tags robots
// @Accept json
// @Produce json
// @Success 201 {object} RobotResponse "Created robot"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 409 {object} ErrorResponse "Robot already exists"
// @Security BearerAuth
// @Router /robots [post]
func (h *Handler) CreateRobot(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateRobotRequest
	if err := decodeRobotsBody(w, r, &req); err != nil {
		return
	}
	if !auth.IsValidRobotName(req.Name) {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "Invalid robot name"})
		return
	}
	if robotDescriptionTooLong(req.Description) {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: fmt.Sprintf("Description exceeds %d characters", maxRobotDescriptionLength)})
		return
	}

	labels, err := h.resolveGroupLabels(ctx, req.Groups)
	if err != nil {
		writeGroupLabelsError(w, err)
		return
	}
	labels[krknv1alpha1.RobotAccountLabel] = "true"

	robot := &krknv1alpha1.KrknRobot{
		ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: h.namespace, Labels: labels},
		Spec:       krknv1alpha1.KrknRobotSpec{Description: req.Description},
	}
	if err := h.client.Create(ctx, robot); err != nil {
		if apierrors.IsAlreadyExists(err) {
			writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "conflict", Message: "Robot already exists"})
			return
		}
		log.FromContext(ctx).Error(err, "Failed to create robot", "robot", req.Name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to create robot"})
		return
	}

	h.auditRobotAction(ctx, "create", req.Name)
	writeJSON(w, http.StatusCreated, robotToResponse(robot))
}

// GetRobot handles GET /api/v1/robots/{name}
//
// @Summary Get a robot account
// @Description Get a robot account by name (admin only)
// @Tags robots
// @Produce json
// @Success 200 {object} RobotResponse "Robot details"
// @Failure 404 {object} ErrorResponse "Robot not found"
// @Security BearerAuth
// @Router /robots/{name} [get]
func (h *Handler) GetRobot(w http.ResponseWriter, r *http.Request, name string) {
	robot, ok := h.getRobotOr404(w, r, name)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, robotToResponse(robot))
}

// UpdateRobot handles PATCH /api/v1/robots/{name}
//
// @Summary Update a robot account
// @Description Update description, disabled flag or group memberships (admin only). Groups replaces all group labels.
// @Tags robots
// @Accept json
// @Produce json
// @Success 200 {object} RobotResponse "Updated robot"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 404 {object} ErrorResponse "Robot not found"
// @Failure 409 {object} ErrorResponse "Robot was modified concurrently"
// @Security BearerAuth
// @Router /robots/{name} [patch]
func (h *Handler) UpdateRobot(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()

	var req UpdateRobotRequest
	if err := decodeRobotsBody(w, r, &req); err != nil {
		return
	}

	robot, ok := h.getRobotOr404(w, r, name)
	if !ok {
		return
	}

	updated := robot.DeepCopy()
	labels := make(map[string]string, len(updated.Labels))
	for k, v := range updated.Labels {
		labels[k] = v
	}
	if req.Description != nil {
		if robotDescriptionTooLong(*req.Description) {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: fmt.Sprintf("Description exceeds %d characters", maxRobotDescriptionLength)})
			return
		}
		updated.Spec.Description = *req.Description
	}
	if req.Disabled != nil {
		updated.Spec.Disabled = *req.Disabled
	}
	if req.Groups != nil {
		groupLabels, err := h.resolveGroupLabels(ctx, *req.Groups)
		if err != nil {
			writeGroupLabelsError(w, err)
			return
		}
		for key := range labels {
			if strings.HasPrefix(key, groupauth.GroupLabelPrefix) {
				delete(labels, key)
			}
		}
		for key, value := range groupLabels {
			labels[key] = value
		}
	}
	updated.Labels = labels

	if err := h.client.Update(ctx, updated); err != nil {
		if apierrors.IsConflict(err) {
			writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "conflict", Message: "Robot was modified concurrently; fetch it again and retry"})
			return
		}
		log.FromContext(ctx).Error(err, "Failed to update robot", "robot", name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to update robot"})
		return
	}

	h.auditRobotAction(ctx, "update", name)
	writeJSON(w, http.StatusOK, robotToResponse(updated))
}

// DeleteRobot handles DELETE /api/v1/robots/{name}
//
// @Summary Delete a robot account
// @Description Delete a robot account and its token secret (admin only)
// @Tags robots
// @Success 204 "No content"
// @Failure 404 {object} ErrorResponse "Robot not found"
// @Security BearerAuth
// @Router /robots/{name} [delete]
func (h *Handler) DeleteRobot(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()

	robot, ok := h.getRobotOr404(w, r, name)
	if !ok {
		return
	}
	if err := h.client.Delete(ctx, robot); err != nil {
		log.FromContext(ctx).Error(err, "Failed to delete robot", "robot", name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to delete robot"})
		return
	}
	// Best-effort token secret cleanup; the owner reference also garbage
	// collects it in a live cluster.
	if err := h.robotTokens.Revoke(ctx, name); err != nil {
		log.FromContext(ctx).Error(err, "Failed to delete token secret for robot", "robot", name)
	}

	h.auditRobotAction(ctx, "delete", name)
	w.WriteHeader(http.StatusNoContent)
}

// IssueRobotToken handles POST /api/v1/robots/{name}/token
//
// @Summary Issue a robot token
// @Description Issue a new bearer token for a robot, replacing any previous one (admin only). The token is shown exactly once.
// @Tags robots
// @Produce json
// @Success 201 {object} IssueRobotTokenResponse "Issued token"
// @Failure 404 {object} ErrorResponse "Robot not found"
// @Failure 409 {object} ErrorResponse "Robot is disabled"
// @Security BearerAuth
// @Router /robots/{name}/token [post]
func (h *Handler) IssueRobotToken(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()

	if _, ok := h.getRobotOr404(w, r, name); !ok {
		return
	}
	issued, err := h.robotTokens.Issue(ctx, name)
	if err != nil {
		writeRobotsServiceError(ctx, w, name, err)
		return
	}

	h.auditRobotAction(ctx, "issue-token", name)
	writeJSON(w, http.StatusCreated, IssueRobotTokenResponse{
		Token:     issued.Token,
		TokenID:   issued.TokenID,
		ExpiresAt: issued.ExpiresAt,
		Message:   "Store this token now; it will not be shown again.",
	})
}

// RevokeRobotToken handles DELETE /api/v1/robots/{name}/token
//
// @Summary Revoke a robot token
// @Description Revoke the active token of a robot (admin only)
// @Tags robots
// @Success 204 "No content"
// @Failure 404 {object} ErrorResponse "Robot not found"
// @Security BearerAuth
// @Router /robots/{name}/token [delete]
func (h *Handler) RevokeRobotToken(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()

	if _, ok := h.getRobotOr404(w, r, name); !ok {
		return
	}
	if err := h.robotTokens.Revoke(ctx, name); err != nil {
		log.FromContext(ctx).Error(err, "Failed to revoke robot token", "robot", name)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to revoke token"})
		return
	}

	h.auditRobotAction(ctx, "revoke-token", name)
	w.WriteHeader(http.StatusNoContent)
}

// getRobotOr404 loads a robot or writes a 404 response.
func (h *Handler) getRobotOr404(w http.ResponseWriter, r *http.Request, name string) (*krknv1alpha1.KrknRobot, bool) {
	robot := &krknv1alpha1.KrknRobot{}
	if err := h.client.Get(r.Context(), client.ObjectKey{Name: name, Namespace: h.namespace}, robot); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{Error: "not_found", Message: fmt.Sprintf("Robot '%s' not found", name)})
		} else {
			log.FromContext(r.Context()).Error(err, "Failed to get robot", "robot", name)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to get robot"})
		}
		return nil, false
	}
	return robot, true
}

// resolveGroupLabels validates that each group exists and returns its labels.
// A missing group yields a client error; any other lookup failure (API
// failure, RBAC denial) is logged and wrapped with errGroupLookup so callers
// can map it to a 500 instead of blaming the request.
func (h *Handler) resolveGroupLabels(ctx context.Context, groups []string) (map[string]string, error) {
	labels := make(map[string]string, len(groups))
	for _, g := range groups {
		if strings.TrimSpace(g) == "" {
			return nil, fmt.Errorf("group name must not be empty")
		}
		name := groupauth.SanitizeGroupName(g)
		group := &krknv1alpha1.KrknUserGroup{}
		if err := h.client.Get(ctx, client.ObjectKey{Name: name, Namespace: h.namespace}, group); err != nil {
			if apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("group %q not found", g)
			}
			log.FromContext(ctx).Error(err, "Failed to get group", "group", name)
			return nil, fmt.Errorf("failed to look up group %q: %w", g, errGroupLookup)
		}
		labels[groupauth.GroupLabelKey(name)] = "true"
	}
	return labels, nil
}

// writeGroupLabelsError maps resolveGroupLabels errors: unknown or empty
// group names are client errors (400); lookup failures are already logged in
// resolveGroupLabels and map to a generic 500 that leaks no internals.
func writeGroupLabelsError(w http.ResponseWriter, err error) {
	if errors.Is(err, errGroupLookup) {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to resolve groups"})
		return
	}
	writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: err.Error()})
}

// robotDescriptionTooLong reports whether a robot description exceeds the
// CRD's MaxLength limit. The limit counts characters (not bytes), matching
// Kubernetes string validation, so multibyte descriptions are measured with
// utf8.RuneCountInString.
func robotDescriptionTooLong(description string) bool {
	return utf8.RuneCountInString(description) > maxRobotDescriptionLength
}

// decodeRobotsBody decodes a strict JSON body, writing the error response.
func decodeRobotsBody(w http.ResponseWriter, r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "Invalid request body"})
		return err
	}
	return nil
}

// writeRobotsServiceError maps token service errors to HTTP responses. The
// cause of unexpected failures is logged (structured, without token
// material) so operators can diagnose them.
func writeRobotsServiceError(ctx context.Context, w http.ResponseWriter, robotName string, err error) {
	switch {
	case isErrRobotDisabled(err):
		writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "conflict", Message: "Robot is disabled"})
	case apierrors.IsNotFound(err):
		writeJSONError(w, http.StatusNotFound, ErrorResponse{Error: "not_found", Message: "Robot not found"})
	default:
		log.FromContext(ctx).Error(err, "Failed to issue robot token", "robot", robotName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to issue token"})
	}
}

// isErrRobotDisabled reports whether err wraps errRobotDisabled.
func isErrRobotDisabled(err error) bool {
	return errors.Is(err, errRobotDisabled)
}

// auditRobotAction logs an admin mutation on a robot. Never logs tokens.
func (h *Handler) auditRobotAction(ctx context.Context, action, robot string) {
	claims := auth.GetClaimsFromContext(ctx)
	admin := ""
	if claims != nil {
		admin = claims.UserID
	}
	log.FromContext(ctx).WithName("robot-audit").Info("robot admin action", "admin", admin, "action", action, "robot", robot)
}

// robotToResponse maps a KrknRobot CR to its admin API response. It copies
// data out of the CR so the caller can never mutate the cached object.
func robotToResponse(robot *krknv1alpha1.KrknRobot) RobotResponse {
	groups := make([]string, 0)
	for key, value := range robot.Labels {
		if value == "true" && strings.HasPrefix(key, groupauth.GroupLabelPrefix) {
			groups = append(groups, strings.TrimPrefix(key, groupauth.GroupLabelPrefix))
		}
	}
	sort.Strings(groups)

	resp := RobotResponse{
		Name:        robot.Name,
		Description: robot.Spec.Description,
		Disabled:    robot.Spec.Disabled,
		Groups:      groups,
		TokenID:     robot.Status.TokenID,
		CreatedAt:   robot.CreationTimestamp.Time,
	}
	if robot.Status.TokenIssuedAt != nil {
		t := robot.Status.TokenIssuedAt.Time
		resp.TokenIssuedAt = &t
	}
	if robot.Status.TokenExpiresAt != nil {
		t := robot.Status.TokenExpiresAt.Time
		resp.TokenExpiresAt = &t
	}
	if robot.Status.LastUsedAt != nil {
		t := robot.Status.LastUsedAt.Time
		resp.LastUsedAt = &t
	}
	return resp
}
