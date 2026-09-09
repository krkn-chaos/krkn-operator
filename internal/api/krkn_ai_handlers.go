package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	"github.com/krkn-chaos/krkn-operator/pkg/krknaiserver"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *Handler) KrknAIRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, KrknAIPath)
	switch {
	case path == "/discoveries" && r.Method == http.MethodPost:
		h.createKrknAIDiscovery(w, r)
	case path == "/configs" && r.Method == http.MethodPost:
		h.createKrknAIConfig(w, r)
	case path == "/runs" && r.Method == http.MethodGet:
		h.listKrknAIRuns(w, r)
	case path == "/runs" && r.Method == http.MethodPost:
		h.createKrknAIRun(w, r)
	case strings.HasPrefix(path, "/runs/"):
		h.krknAIRunRouter(w, r, strings.TrimPrefix(path, "/runs/"))
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) krknAIRunRouter(w http.ResponseWriter, r *http.Request, remainder string) {
	parts := strings.Split(remainder, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	name := parts[0]
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			h.getKrknAIRun(w, r, name)
		case http.MethodDelete:
			h.deleteKrknAIRun(w, r, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	switch parts[1] {
	case "results":
		if len(parts) == 2 {
			h.proxyKrknAIArtifact(w, r, name, "results")
			return
		}
	case "files":
		if len(parts) > 2 {
			h.proxyKrknAIArtifact(w, r, name, "files/"+strings.Join(parts[2:], "/"))
			return
		}
	}
	http.NotFound(w, r)
}

func decodeKrknAIRequest(r *http.Request, destination any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(destination)
}

func writeKrknAIJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}

func (h *Handler) resolveKrknAITarget(ctx context.Context, requestID string, targets map[string][]string, action groupauth.Action) (*krknv1alpha1.KrknTargetRequest, string, string, error) {
	if requestID == "" || len(targets) != 1 {
		return nil, "", "", fmt.Errorf("exactly one target provider and cluster is required")
	}
	var provider, cluster string
	for selectedProvider, clusters := range targets {
		if selectedProvider == "" || len(clusters) != 1 || clusters[0] == "" {
			return nil, "", "", fmt.Errorf("exactly one target provider and cluster is required")
		}
		provider = selectedProvider
		cluster = clusters[0]
	}
	var target krknv1alpha1.KrknTargetRequest
	if err := h.client.Get(ctx, client.ObjectKey{Name: requestID, Namespace: h.namespace}, &target); err != nil {
		return nil, "", "", fmt.Errorf("target request not found: %w", err)
	}
	if !strings.EqualFold(target.Status.Status, "completed") {
		return nil, "", "", fmt.Errorf("target request is not completed")
	}
	var apiURL string
	for _, candidate := range target.Status.TargetData[provider] {
		if candidate.ClusterName == cluster {
			apiURL = candidate.ClusterAPIURL
			break
		}
	}
	if apiURL == "" {
		return nil, "", "", fmt.Errorf("target cluster not found")
	}
	if !auth.IsAdmin(ctx) {
		claims := auth.GetClaimsFromContext(ctx)
		if claims == nil {
			return nil, "", "", fmt.Errorf("authentication required")
		}
		allowed, err := groupauth.HasClusterPermission(ctx, h.client, claims.UserID, h.namespace, apiURL, action)
		if err != nil {
			return nil, "", "", err
		}
		if !allowed {
			return nil, "", "", fmt.Errorf("target access denied")
		}
	}
	return &target, provider, cluster, nil
}

func (h *Handler) managedClusterKubeconfig(ctx context.Context, requestID, provider, cluster string) (string, error) {
	var secret corev1.Secret
	if err := h.client.Get(ctx, client.ObjectKey{Name: requestID, Namespace: h.namespace}, &secret); err != nil {
		return "", err
	}
	var data map[string]map[string]struct {
		Kubeconfig string `json:"kubeconfig"`
	}
	if err := json.Unmarshal(secret.Data["managed-clusters"], &data); err != nil {
		return "", fmt.Errorf("invalid managed-clusters data: %w", err)
	}
	kubeconfig := data[provider][cluster].Kubeconfig
	if kubeconfig == "" {
		return "", fmt.Errorf("target kubeconfig not found")
	}
	return kubeconfig, nil
}

func (h *Handler) createKrknAIDiscovery(w http.ResponseWriter, r *http.Request) {
	var request KrknAIDiscoveryRequest
	if err := decodeKrknAIRequest(r, &request); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "invalid discovery request"})
		return
	}
	_, provider, cluster, err := h.resolveKrknAITarget(r.Context(), request.TargetRequestID, request.TargetClusters, groupauth.ActionRun)
	if err != nil {
		h.writeKrknAITargetError(w, err)
		return
	}
	kubeconfig, err := h.managedClusterKubeconfig(r.Context(), request.TargetRequestID, provider, cluster)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "target kubeconfig is unavailable"})
		return
	}
	payload, _ := json.Marshal(map[string]string{
		"kubeconfig": kubeconfig, "namespacePattern": request.NamespacePattern,
		"podLabelPattern": request.PodLabelPattern, "nodeLabelPattern": request.NodeLabelPattern, "skipPodName": request.SkipPodName,
	})
	h.proxyKrknAIRequest(w, r, http.MethodPost, "/v1/discoveries", payload)
}

func (h *Handler) createKrknAIConfig(w http.ResponseWriter, r *http.Request) {
	var request KrknAIConfigRequest
	if err := decodeKrknAIRequest(r, &request); err != nil || request.Name == "" || strings.TrimSpace(request.ConfigYAML) == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "name and configYaml are required"})
		return
	}
	var document any
	if err := yaml.Unmarshal([]byte(request.ConfigYAML), &document); err != nil || document == nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "configYaml must be valid YAML"})
		return
	}
	_, provider, cluster, err := h.resolveKrknAITarget(r.Context(), request.TargetRequestID, request.TargetClusters, groupauth.ActionRun)
	if err != nil {
		h.writeKrknAITargetError(w, err)
		return
	}
	created, err := h.createFileInternal(r.Context(), files.CreateFileRequest{
		FileName: request.Name, Content: request.ConfigYAML, Description: request.Description,
		Groups: request.Groups, AvailableToAll: request.AvailableToAll, FilePurpose: files.FilePurposeKrknAIConfig,
	})
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: err.Error()})
		return
	}
	configMap, err := h.loadFileConfigMapByID(r.Context(), created.FileID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "created config could not be loaded"})
		return
	}
	configMap.Data = map[string]string{"krkn-ai.yaml": request.ConfigYAML}
	configMap.Labels[files.KrknAIConfigTargetBindingLabel] = files.HashLogicalName(request.TargetRequestID + "\x00" + provider + "\x00" + cluster)
	configMap.Annotations[files.KrknAIConfigTargetRequestAnnotation] = request.TargetRequestID
	configMap.Annotations[files.KrknAIConfigTargetProviderAnnotation] = provider
	configMap.Annotations[files.KrknAIConfigTargetClusterAnnotation] = cluster
	if err := h.client.Update(r.Context(), configMap); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "failed to bind config target"})
		return
	}
	writeKrknAIJSON(w, http.StatusCreated, map[string]string{"configId": created.FileID})
}

func (h *Handler) createKrknAIRun(w http.ResponseWriter, r *http.Request) {
	var request KrknAIRunRequest
	if err := decodeKrknAIRequest(r, &request); err != nil || request.Name == "" || request.ConfigID == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "name and configId are required"})
		return
	}
	_, provider, cluster, err := h.resolveKrknAITarget(r.Context(), request.TargetRequestID, request.TargetClusters, groupauth.ActionRun)
	if err != nil {
		h.writeKrknAITargetError(w, err)
		return
	}
	configMap, err := h.loadFileConfigMapByID(r.Context(), request.ConfigID)
	if err != nil || (!auth.IsAdmin(r.Context()) && !h.fileIsAccessible(r.Context(), configMap)) {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{Error: "forbidden", Message: "config access denied"})
		return
	}
	binding := files.HashLogicalName(request.TargetRequestID + "\x00" + provider + "\x00" + cluster)
	if configMap.Labels[files.FilePurposeLabel] != files.FilePurposeKrknAIConfig || configMap.Labels[files.KrknAIConfigTargetBindingLabel] != binding {
		writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "conflict", Message: "config is not bound to this target"})
		return
	}
	claims := auth.GetClaimsFromContext(r.Context())
	run := &krknv1alpha1.KrknAIRun{ObjectMeta: metav1.ObjectMeta{Name: request.Name, Namespace: h.namespace}, Spec: krknv1alpha1.KrknAIRunSpec{
		ConfigMapName: "file-" + request.ConfigID, ConfigMapKey: "krkn-ai.yaml", TargetRequestID: request.TargetRequestID,
		TargetClusters: request.TargetClusters, PrometheusURL: request.PrometheusURL, PrometheusTokenSecretRef: request.PrometheusTokenSecretRef,
		ActiveDeadlineSeconds: request.ActiveDeadlineSeconds,
	}}
	if claims != nil {
		run.Spec.OwnerUserID = claims.UserID
	}
	if err := h.client.Create(r.Context(), run); err != nil {
		if apierrors.IsAlreadyExists(err) {
			writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "conflict", Message: "run already exists"})
			return
		}
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "failed to create run"})
		return
	}
	writeKrknAIJSON(w, http.StatusCreated, run)
}

func (h *Handler) fileIsAccessible(ctx context.Context, configMap *corev1.ConfigMap) bool {
	allowed, err := h.canAccessFile(ctx, configMap)
	return err == nil && allowed
}

func (h *Handler) listKrknAIRuns(w http.ResponseWriter, r *http.Request) {
	var runs krknv1alpha1.KrknAIRunList
	if err := h.client.List(r.Context(), &runs, client.InNamespace(h.namespace)); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "failed to list runs"})
		return
	}
	visible := make([]krknv1alpha1.KrknAIRun, 0, len(runs.Items))
	for i := range runs.Items {
		run := &runs.Items[i]
		if _, _, _, err := h.resolveKrknAITarget(r.Context(), run.Spec.TargetRequestID, run.Spec.TargetClusters, groupauth.ActionView); err == nil {
			visible = append(visible, *run)
		}
	}
	writeKrknAIJSON(w, http.StatusOK, visible)
}

func (h *Handler) getKrknAIRun(w http.ResponseWriter, r *http.Request, name string) {
	run, ok := h.authorizeKrknAIRun(w, r, name, groupauth.ActionView)
	if ok {
		writeKrknAIJSON(w, http.StatusOK, run)
	}
}

func (h *Handler) deleteKrknAIRun(w http.ResponseWriter, r *http.Request, name string) {
	run, ok := h.authorizeKrknAIRun(w, r, name, groupauth.ActionCancel)
	if !ok {
		return
	}
	if err := h.client.Delete(r.Context(), run); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "failed to delete run"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) authorizeKrknAIRun(w http.ResponseWriter, r *http.Request, name string, action groupauth.Action) (*krknv1alpha1.KrknAIRun, bool) {
	var run krknv1alpha1.KrknAIRun
	if err := h.client.Get(r.Context(), client.ObjectKey{Name: name, Namespace: h.namespace}, &run); err != nil {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{Error: "not_found", Message: "run not found"})
		return nil, false
	}
	if _, _, _, err := h.resolveKrknAITarget(r.Context(), run.Spec.TargetRequestID, run.Spec.TargetClusters, action); err != nil {
		h.writeKrknAITargetError(w, err)
		return nil, false
	}
	return &run, true
}

func (h *Handler) proxyKrknAIArtifact(w http.ResponseWriter, r *http.Request, name, artifactPath string) {
	run, ok := h.authorizeKrknAIRun(w, r, name, groupauth.ActionView)
	if !ok {
		return
	}
	path := "/v1/runs/" + krknaiserver.EscapePath(string(run.UID)) + "/" + krknaiserver.EscapePath(artifactPath)
	h.proxyKrknAIRequest(w, r, http.MethodGet, path, nil)
}

func (h *Handler) proxyKrknAIRequest(w http.ResponseWriter, r *http.Request, method, path string, payload []byte) {
	response, err := h.artifactClient.Do(r.Context(), method, path, payload)
	if err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{Error: "service_unavailable", Message: "Krkn-AI artifact service is unavailable"})
		return
	}
	body, err := krknaiserver.ReadBody(response)
	if err != nil || response.StatusCode >= 500 {
		writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{Error: "service_unavailable", Message: "Krkn-AI artifact service is unavailable"})
		return
	}
	if contentType := response.Header.Get("Content-Type"); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.WriteHeader(response.StatusCode)
	_, _ = w.Write(body)
}

func (h *Handler) writeKrknAITargetError(w http.ResponseWriter, err error) {
	statusCode := http.StatusForbidden
	if strings.Contains(err.Error(), "not found") {
		statusCode = http.StatusNotFound
	} else if strings.Contains(err.Error(), "exactly one") || strings.Contains(err.Error(), "not completed") {
		statusCode = http.StatusBadRequest
	}
	writeJSONError(w, statusCode, ErrorResponse{Error: "target_access", Message: err.Error()})
}
