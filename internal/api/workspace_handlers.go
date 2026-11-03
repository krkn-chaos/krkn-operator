package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	"github.com/krkn-chaos/krkn-operator/pkg/workspace"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type workspacePublishRequest struct {
	ClaimName      string   `json:"claimName"`
	DisplayName    string   `json:"displayName,omitempty"`
	Groups         []string `json:"groups,omitempty"`
	AvailableToAll bool     `json:"availableToAll,omitempty"`
}

type workspaceInfo struct {
	Name           string   `json:"name"`
	DisplayName    string   `json:"displayName,omitempty"`
	ClaimName      string   `json:"claimName,omitempty"`
	Groups         []string `json:"groups,omitempty"`
	AvailableToAll bool     `json:"availableToAll"`
	Available      bool     `json:"available"`
	Reason         string   `json:"reason,omitempty"`
}

type availableWorkspacePVC struct {
	Name string `json:"name"`
}

func workspaceAvailable(entry *krknv1alpha1.KrknWorkspace) bool {
	condition := meta.FindStatusCondition(entry.Status.Conditions, "Available")
	return entry.DeletionTimestamp == nil && entry.Status.ObservedGeneration == entry.Generation &&
		condition != nil && condition.Status == metav1.ConditionTrue && condition.ObservedGeneration == entry.Generation
}

func workspaceInfoFromCR(entry *krknv1alpha1.KrknWorkspace, admin bool) workspaceInfo {
	info := workspaceInfo{Name: entry.Name, DisplayName: entry.Spec.DisplayName,
		AvailableToAll: entry.Spec.AvailableToAll, Available: workspaceAvailable(entry)}
	if entry.Spec.Group != "" {
		info.Groups = []string{entry.Spec.Group}
	}
	if admin {
		info.ClaimName = entry.Spec.ClaimName
	}
	if entry.Status.ObservedGeneration != entry.Generation {
		info.Reason = "Pending"
	} else if condition := meta.FindStatusCondition(entry.Status.Conditions, "Available"); condition != nil {
		info.Reason = condition.Reason
	} else {
		info.Reason = "Pending"
	}
	return info
}

func publicGraphWorkspaces(nodes map[string]krknv1alpha1.GraphScenarioNode) map[string]krknv1alpha1.GraphScenarioNode {
	redacted := make(map[string]krknv1alpha1.GraphScenarioNode, len(nodes))
	for id, node := range nodes {
		if node.Workspace != nil {
			ref := *node.Workspace
			ref.ClaimName = ""
			node.Workspace = &ref
		}
		redacted[id] = node
	}
	return redacted
}

func (h *Handler) canAccessWorkspace(ctx context.Context, entry *krknv1alpha1.KrknWorkspace) (bool, error) {
	if auth.IsAdmin(ctx) {
		return workspace.CanAccess(entry, true, nil), nil
	}
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		return false, nil
	}
	groups, err := groupauth.GetUserGroups(ctx, h.client, claims.UserID, h.namespace)
	if err != nil {
		return false, fmt.Errorf("get workspace user groups: %w", err)
	}
	names := make([]string, len(groups))
	for i := range groups {
		names[i] = groups[i].Name
	}
	return workspace.CanAccess(entry, false, names), nil
}

// resolveWorkspaceSelection pins the platform workspace and its backing PVC.
func (h *Handler) resolveWorkspaceSelection(w http.ResponseWriter, r *http.Request, requested *krknv1alpha1.WorkspaceReference) (*krknv1alpha1.WorkspaceReference, bool) {
	if requested == nil {
		return nil, true
	}
	if requested.Name == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "workspace.name is required"})
		return nil, false
	}
	entry := &krknv1alpha1.KrknWorkspace{}
	if err := h.client.Get(r.Context(), client.ObjectKey{Namespace: h.namespace, Name: requested.Name}, entry); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "workspace_unavailable", Message: "Workspace is missing"})
		} else {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to load workspace"})
		}
		return nil, false
	}
	if requested.WorkspaceUID != "" && requested.WorkspaceUID != string(entry.UID) ||
		requested.ClaimName != "" && requested.ClaimName != entry.Spec.ClaimName ||
		requested.ClaimUID != "" && requested.ClaimUID != entry.Spec.ClaimUID {
		writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "workspace_unavailable", Message: "Workspace binding changed"})
		return nil, false
	}
	allowed, err := h.canAccessWorkspace(r.Context(), entry)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to validate workspace access"})
		return nil, false
	}
	if !allowed {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{Error: "forbidden", Message: "Workspace is not available to this user"})
		return nil, false
	}
	if !workspaceAvailable(entry) {
		writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "workspace_unavailable", Message: "Workspace PVC is not available"})
		return nil, false
	}
	pvc := &corev1.PersistentVolumeClaim{}
	if err := h.client.Get(r.Context(), client.ObjectKey{Namespace: h.namespace, Name: entry.Spec.ClaimName}, pvc); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "workspace_unavailable", Message: "Workspace PVC is missing"})
		} else {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to load workspace PVC"})
		}
		return nil, false
	}
	if err := workspace.ValidatePVC(pvc, types.UID(entry.Spec.ClaimUID)); err != nil {
		writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "workspace_unavailable", Message: err.Error()})
		return nil, false
	}
	return &krknv1alpha1.WorkspaceReference{Name: entry.Name, WorkspaceUID: string(entry.UID),
		ClaimName: pvc.Name, ClaimUID: string(pvc.UID)}, true
}

// WorkspaceRouter manages platform workspaces backed by existing PVCs.
func (h *Handler) WorkspaceRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(strings.Replace(r.URL.Path, "/api/v2/", "/api/v1/", 1), "/")
	if path == WorkspacePVCsPath {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method_not_allowed", Message: "Method not allowed"})
			return
		}
		if !auth.IsAdmin(r.Context()) {
			writeJSONError(w, http.StatusForbidden, ErrorResponse{Error: "forbidden", Message: "Admin privileges required"})
			return
		}
		h.listAvailableWorkspacePVCs(w, r)
		return
	}
	if path == WorkspacesPath || path == WorkspacesPath+"/available" {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method_not_allowed", Message: "Method not allowed"})
			return
		}
		if path == WorkspacesPath && !auth.IsAdmin(r.Context()) {
			writeJSONError(w, http.StatusForbidden, ErrorResponse{Error: "forbidden", Message: "Admin privileges required"})
			return
		}
		h.listWorkspaces(w, r)
		return
	}
	if !strings.HasPrefix(path, WorkspacesPath+"/") || !auth.IsAdmin(r.Context()) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{Error: "forbidden", Message: "Admin privileges required"})
		return
	}
	name := strings.TrimPrefix(path, WorkspacesPath+"/")
	if name == "" || strings.Contains(name, "/") {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "Invalid workspace name"})
		return
	}
	if problems := validation.IsDNS1123Subdomain(name); len(problems) != 0 {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "Invalid workspace name"})
		return
	}
	switch r.Method {
	case http.MethodPut:
		h.publishWorkspace(w, r, name)
	case http.MethodDelete:
		h.unpublishWorkspace(w, r, name)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method_not_allowed", Message: "Method not allowed"})
	}
}

// @Summary List PVCs eligible for workspace publication
// @Description Returns Bound Filesystem ReadWriteMany PVCs in the operator namespace. Admin only.
// @Tags workspaces
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /workspaces/pvcs/available [get]
// @Security BearerAuth
func (h *Handler) listAvailableWorkspacePVCs(w http.ResponseWriter, r *http.Request) {
	var pvcList corev1.PersistentVolumeClaimList
	if err := h.client.List(r.Context(), &pvcList, client.InNamespace(h.namespace)); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to list workspace PVCs"})
		return
	}
	items := make([]availableWorkspacePVC, 0)
	for i := range pvcList.Items {
		pvc := &pvcList.Items[i]
		if workspace.ValidatePVC(pvc, "") == nil {
			items = append(items, availableWorkspacePVC{Name: pvc.Name})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	writeJSON(w, http.StatusOK, map[string]any{"pvcs": items})
}

// @Summary List platform workspaces visible to the caller
// @Tags workspaces
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /workspaces [get]
// @Router /workspaces/available [get]
// @Security BearerAuth
func (h *Handler) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	var entries krknv1alpha1.KrknWorkspaceList
	if err := h.client.List(r.Context(), &entries, client.InNamespace(h.namespace)); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to list workspaces"})
		return
	}
	admin := auth.IsAdmin(r.Context())
	items := make([]workspaceInfo, 0, len(entries.Items))
	for i := range entries.Items {
		entry := &entries.Items[i]
		if (strings.HasSuffix(r.URL.Path, "/available") || !admin) && !workspaceAvailable(entry) {
			continue
		}
		allowed, err := h.canAccessWorkspace(r.Context(), entry)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to validate workspace access"})
			return
		}
		if allowed {
			items = append(items, workspaceInfoFromCR(entry, admin))
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspaces": items})
}

// @Summary Publish an existing RWX PVC as a named workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param name path string true "Workspace name"
// @Param request body workspacePublishRequest true "Workspace definition"
// @Success 200 {object} workspaceInfo
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /workspaces/{name} [put]
// @Security BearerAuth
func (h *Handler) publishWorkspace(w http.ResponseWriter, r *http.Request, name string) {
	var request workspacePublishRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || request.ClaimName == "" || len(request.Groups) > 1 ||
		(request.AvailableToAll && len(request.Groups) != 0) {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "claimName and one visibility choice are required"})
		return
	}
	if len(request.Groups) == 1 {
		group := &krknv1alpha1.KrknUserGroup{}
		if err := h.client.Get(r.Context(), client.ObjectKey{Namespace: h.namespace, Name: request.Groups[0]}, group); err != nil {
			if apierrors.IsNotFound(err) {
				writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "Workspace group does not exist"})
			} else {
				writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to load workspace group"})
			}
			return
		}
	}
	pvc := &corev1.PersistentVolumeClaim{}
	if err := h.client.Get(r.Context(), client.ObjectKey{Namespace: h.namespace, Name: request.ClaimName}, pvc); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{Error: "not_found", Message: "PVC not found"})
		} else {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to load PVC"})
		}
		return
	}
	if err := workspace.ValidatePVC(pvc, ""); err != nil {
		writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "workspace_unavailable", Message: err.Error()})
		return
	}
	groupName := ""
	if len(request.Groups) != 0 {
		groupName = request.Groups[0]
	}
	entry := &krknv1alpha1.KrknWorkspace{}
	errBindingChanged := errors.New("workspace PVC binding cannot be changed")
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := h.client.Get(r.Context(), client.ObjectKey{Namespace: h.namespace, Name: name}, entry)
		if apierrors.IsNotFound(err) {
			entry = &krknv1alpha1.KrknWorkspace{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: h.namespace},
				Spec: krknv1alpha1.KrknWorkspaceSpec{ClaimName: pvc.Name, ClaimUID: string(pvc.UID),
					DisplayName: request.DisplayName, AvailableToAll: request.AvailableToAll, Group: groupName}}
			return h.client.Create(r.Context(), entry)
		}
		if err != nil {
			return err
		}
		if entry.Spec.ClaimName != pvc.Name || entry.Spec.ClaimUID != string(pvc.UID) {
			return errBindingChanged
		}
		entry.Spec.DisplayName, entry.Spec.AvailableToAll, entry.Spec.Group = request.DisplayName, request.AvailableToAll, groupName
		return h.client.Update(r.Context(), entry)
	})
	if errors.Is(err, errBindingChanged) || apierrors.IsAlreadyExists(err) {
		writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "workspace_conflict", Message: "Workspace is bound to another PVC"})
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to publish workspace"})
		return
	}
	writeJSON(w, http.StatusOK, workspaceInfoFromCR(entry, true))
}

// @Summary Unpublish a workspace without deleting its PVC
// @Tags workspaces
// @Param name path string true "Workspace name"
// @Success 204
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /workspaces/{name} [delete]
// @Security BearerAuth
func (h *Handler) unpublishWorkspace(w http.ResponseWriter, r *http.Request, name string) {
	entry := &krknv1alpha1.KrknWorkspace{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: h.namespace}}
	if err := h.client.Delete(r.Context(), entry); err != nil {
		if apierrors.IsNotFound(err) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{Error: "not_found", Message: "Workspace not found"})
		} else {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "Failed to unpublish workspace"})
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
