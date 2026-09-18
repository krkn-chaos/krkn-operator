package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	"github.com/krkn-chaos/krkn-operator/pkg/krknaiserver"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"
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

func validateKubernetesResourceName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if problems := validation.IsDNS1123Subdomain(name); len(problems) > 0 {
		return fmt.Errorf("name must be a valid Kubernetes resource name: %s", strings.Join(problems, "; "))
	}
	return nil
}

func validateKrknAIRunName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if problems := validation.IsDNS1123Label(name); len(problems) > 0 {
		return fmt.Errorf("name must be a valid Kubernetes label name: %s", strings.Join(problems, "; "))
	}
	return nil
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

// @Summary Discover a target cluster for Krkn-AI
// @Description Generate a Krkn-AI configuration from one authorized target cluster without persisting it.
// @Tags krkn-ai
// @Accept json
// @Produce json
// @Param request body KrknAIDiscoveryRequest true "Discovery parameters"
// @Success 200 {object} object "Generated Krkn-AI configuration"
// @Failure 400 {object} ErrorResponse "Invalid discovery request"
// @Failure 403 {object} ErrorResponse "Target access denied"
// @Failure 404 {object} ErrorResponse "Target request not found"
// @Failure 503 {object} ErrorResponse "Krkn-AI service unavailable"
// @Security BearerAuth
// @Router /krkn-ai/discoveries [post]
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

// @Summary Create a Krkn-AI configuration
// @Description Validate and persist a Krkn-AI YAML configuration bound to one authorized target cluster.
// @Tags krkn-ai
// @Accept json
// @Produce json
// @Param request body KrknAIConfigRequest true "Configuration and target binding"
// @Success 201 {object} map[string]string "Created configuration ID"
// @Failure 400 {object} ErrorResponse "Invalid name, YAML, or target selection"
// @Failure 401 {object} ErrorResponse "Authentication required"
// @Failure 403 {object} ErrorResponse "Target access denied"
// @Failure 409 {object} ErrorResponse "Configuration name already exists"
// @Failure 500 {object} ErrorResponse "Configuration creation failed"
// @Security BearerAuth
// @Router /krkn-ai/configs [post]
func (h *Handler) createKrknAIConfig(w http.ResponseWriter, r *http.Request) {
	var request KrknAIConfigRequest
	if err := decodeKrknAIRequest(r, &request); err != nil || strings.TrimSpace(request.ConfigYAML) == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "name and configYaml are required"})
		return
	}
	if err := validateKubernetesResourceName(request.Name); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: err.Error()})
		return
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(request.ConfigYAML), &document); err != nil || len(document) == 0 {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "configYaml must be a non-empty YAML object"})
		return
	}
	_, provider, cluster, err := h.resolveKrknAITarget(r.Context(), request.TargetRequestID, request.TargetClusters, groupauth.ActionRun)
	if err != nil {
		h.writeKrknAITargetError(w, err)
		return
	}
	claims := auth.GetClaimsFromContext(r.Context())
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized", Message: "authentication required"})
		return
	}

	fileID := uuid.New().String()
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        request.Name,
			Namespace:   h.namespace,
			Labels:      files.BuildFileLabels(fileID, "", request.Groups, request.AvailableToAll, files.FilePurposeKrknAIConfig, request.Name),
			Annotations: files.BuildFileAnnotations(request.Description, claims.UserID, ""),
		},
		Data: map[string]string{files.KrknAIConfigFileName: request.ConfigYAML},
	}
	configMap.Labels[files.KrknAIConfigTargetBindingLabel] = files.HashLogicalName(request.TargetRequestID + "\x00" + provider + "\x00" + cluster)
	configMap.Annotations[files.KrknAIConfigTargetRequestAnnotation] = request.TargetRequestID
	configMap.Annotations[files.KrknAIConfigTargetProviderAnnotation] = provider
	configMap.Annotations[files.KrknAIConfigTargetClusterAnnotation] = cluster
	if err := h.client.Create(r.Context(), configMap); err != nil {
		if apierrors.IsAlreadyExists(err) {
			writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "conflict", Message: "config name already exists"})
			return
		}
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "failed to create config"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"configId": fileID})
}

// loadKrknAIConfigMapByID reads through the typed client in production, bypassing
// the controller cache after a config has just been created.
func (h *Handler) loadKrknAIConfigMapByID(ctx context.Context, fileID string) (*corev1.ConfigMap, error) {
	if h.clientset == nil {
		return h.loadFileConfigMapByID(ctx, fileID)
	}
	selector := labels.Set{
		files.AppNameLabel:      files.AppName,
		files.AppComponentLabel: files.ComponentFile,
		files.FileIDLabel:       fileID,
	}.AsSelector().String()
	configMaps, err := h.clientset.CoreV1().ConfigMaps(h.namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	if len(configMaps.Items) == 0 {
		return nil, apierrors.NewNotFound(corev1.Resource("configmap"), fileID)
	}
	if len(configMaps.Items) > 1 {
		return nil, fmt.Errorf("multiple ConfigMaps found with file ID %q", fileID)
	}
	return &configMaps.Items[0], nil
}

// @Summary Create a Krkn-AI run
// @Description Start a Krkn-AI run from a persisted configuration bound to one authorized target cluster.
// @Tags krkn-ai
// @Accept json
// @Produce json
// @Param request body KrknAIRunRequest true "Run parameters"
// @Success 201 {object} krknv1alpha1.KrknAIRun "Created run"
// @Failure 400 {object} ErrorResponse "Invalid name, deadline, or target selection"
// @Failure 403 {object} ErrorResponse "Configuration or target access denied"
// @Failure 409 {object} ErrorResponse "Run name exists or configuration target mismatch"
// @Failure 500 {object} ErrorResponse "Run creation failed"
// @Security BearerAuth
// @Router /krkn-ai/runs [post]
func (h *Handler) createKrknAIRun(w http.ResponseWriter, r *http.Request) {
	var request KrknAIRunRequest
	if err := decodeKrknAIRequest(r, &request); err != nil || request.ConfigID == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "name and configId are required"})
		return
	}
	if err := validateKrknAIRunName(request.Name); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: err.Error()})
		return
	}
	activeDeadlineSeconds := int64(21600)
	if request.ActiveDeadlineSeconds != nil {
		if *request.ActiveDeadlineSeconds <= 0 {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "activeDeadlineSeconds must be positive"})
			return
		}
		activeDeadlineSeconds = *request.ActiveDeadlineSeconds
	}
	_, provider, cluster, err := h.resolveKrknAITarget(r.Context(), request.TargetRequestID, request.TargetClusters, groupauth.ActionRun)
	if err != nil {
		h.writeKrknAITargetError(w, err)
		return
	}
	configMap, err := h.loadKrknAIConfigMapByID(r.Context(), request.ConfigID)
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
		ConfigMapName: configMap.Name, ConfigMapKey: files.KrknAIConfigFileName, TargetRequestID: request.TargetRequestID,
		TargetClusters: request.TargetClusters, PrometheusURL: request.PrometheusURL, PrometheusTokenSecretRef: request.PrometheusTokenSecretRef,
		ActiveDeadlineSeconds: activeDeadlineSeconds,
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
	writeJSON(w, http.StatusCreated, run)
}

func (h *Handler) fileIsAccessible(ctx context.Context, configMap *corev1.ConfigMap) bool {
	allowed, err := h.canAccessFile(ctx, configMap)
	return err == nil && allowed
}

// @Summary List Krkn-AI runs
// @Description List runs whose target clusters are visible to the authenticated user.
// @Tags krkn-ai
// @Produce json
// @Success 200 {array} krknv1alpha1.KrknAIRun "Visible runs"
// @Failure 500 {object} ErrorResponse "Run listing failed"
// @Security BearerAuth
// @Router /krkn-ai/runs [get]
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
	writeJSON(w, http.StatusOK, visible)
}

// @Summary Get a Krkn-AI run
// @Description Return one run after authorizing access to its target cluster.
// @Tags krkn-ai
// @Produce json
// @Param name path string true "KrknAIRun name"
// @Success 200 {object} krknv1alpha1.KrknAIRun "Run details"
// @Failure 403 {object} ErrorResponse "Target access denied"
// @Failure 404 {object} ErrorResponse "Run not found"
// @Security BearerAuth
// @Router /krkn-ai/runs/{name} [get]
func (h *Handler) getKrknAIRun(w http.ResponseWriter, r *http.Request, name string) {
	run, ok := h.authorizeKrknAIRun(w, r, name, groupauth.ActionView)
	if ok {
		writeJSON(w, http.StatusOK, run)
	}
}

// @Summary Delete a Krkn-AI run
// @Description Cancel a run by deleting its KrknAIRun resource after target authorization.
// @Tags krkn-ai
// @Param name path string true "KrknAIRun name"
// @Success 204 "Run deleted"
// @Failure 403 {object} ErrorResponse "Target access denied"
// @Failure 404 {object} ErrorResponse "Run not found"
// @Failure 500 {object} ErrorResponse "Run deletion failed"
// @Security BearerAuth
// @Router /krkn-ai/runs/{name} [delete]
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

// @Summary Download Krkn-AI run artifacts
// @Description Return the committed result manifest or stream one artifact file after target authorization.
// @Tags krkn-ai
// @Produce json
// @Param name path string true "KrknAIRun name"
// @Param path path string false "Artifact path"
// @Success 200 {object} object "Result manifest"
// @Failure 403 {object} ErrorResponse "Target access denied"
// @Failure 404 {object} ErrorResponse "Run or artifact not found"
// @Failure 503 {object} ErrorResponse "Krkn-AI artifact service unavailable"
// @Security BearerAuth
// @Router /krkn-ai/runs/{name}/results [get]
// @Router /krkn-ai/runs/{name}/files/{path} [get]
func (h *Handler) proxyKrknAIArtifact(w http.ResponseWriter, r *http.Request, name, artifactPath string) {
	run, ok := h.authorizeKrknAIRun(w, r, name, groupauth.ActionView)
	if !ok {
		return
	}
	path := "/v1/runs/" + krknaiserver.EscapePath(string(run.UID)) + "/" + krknaiserver.EscapePath(artifactPath)
	response, err := h.artifactClient.Do(r.Context(), http.MethodGet, path, nil)
	if err != nil || response.StatusCode >= http.StatusInternalServerError {
		writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{Error: "service_unavailable", Message: "Krkn-AI artifact service is unavailable"})
		return
	}
	defer response.Body.Close()

	if artifactPath == "results" {
		w.Header().Set("Content-Type", "application/json")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment")
		w.Header().Set("X-Content-Type-Options", "nosniff")
	}
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, response.Body)
}

func (h *Handler) proxyKrknAIRequest(w http.ResponseWriter, r *http.Request, method, path string, payload []byte) {
	response, err := h.artifactClient.Do(r.Context(), method, path, payload)
	if err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{Error: "service_unavailable", Message: "Krkn-AI artifact service is unavailable"})
		return
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusInternalServerError {
		writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{Error: "service_unavailable", Message: "Krkn-AI artifact service is unavailable"})
		return
	}
	var responsePayload any
	if err := json.NewDecoder(response.Body).Decode(&responsePayload); err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{Error: "service_unavailable", Message: "Krkn-AI artifact service returned an invalid response"})
		return
	}
	writeJSON(w, response.StatusCode, responsePayload)
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
