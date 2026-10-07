package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *Handler) KrknAIRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.EscapedPath(), KrknAIPath)
	switch {
	case path == "/status" && r.Method == http.MethodGet:
		h.getKrknAIStatus(w, r)
	case path == "/discoveries" && r.Method == http.MethodPost:
		h.createKrknAIDiscovery(w, r)
	case path == "/configs/validate" && r.Method == http.MethodPost:
		h.validateKrknAIConfigRoute(w, r)
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
	name, err := url.PathUnescape(parts[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}
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
		switch {
		case len(parts) == 3 && parts[2] == "download":
			h.downloadKrknAIRunArchive(w, r, name)
			return
		case len(parts) == 2:
			h.proxyKrknAIArtifact(w, r, name, "results")
			return
		case len(parts) == 3 && parts[2] == "summary":
			h.getKrknAIRunResultsSummary(w, r, name)
			return
		case len(parts) == 3 && parts[2] == "scenarios":
			h.getKrknAIRunScenarioIndex(w, r, name)
			return
		case len(parts) == 5 && parts[2] == "scenarios":
			generation, generationErr := url.PathUnescape(parts[3])
			scenarioID, scenarioErr := url.PathUnescape(parts[4])
			if generationErr != nil || scenarioErr != nil {
				http.NotFound(w, r)
				return
			}
			h.getKrknAIRunScenarioDetail(w, r, name, generation, scenarioID)
			return
		}
	case "files":
		if len(parts) > 2 {
			artifactPath, err := url.PathUnescape(strings.Join(parts[2:], "/"))
			if err != nil {
				http.NotFound(w, r)
				return
			}
			h.proxyKrknAIArtifact(w, r, name, "files/"+artifactPath)
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

func (h *Handler) resolveKrknAITarget(ctx context.Context, requestID string, targets map[string][]string, action groupauth.Action) (string, string, error) {
	if requestID == "" || len(targets) != 1 {
		return "", "", fmt.Errorf("exactly one target provider and cluster is required")
	}
	var provider, cluster string
	for selectedProvider, clusters := range targets {
		if selectedProvider == "" || len(clusters) != 1 || clusters[0] == "" {
			return "", "", fmt.Errorf("exactly one target provider and cluster is required")
		}
		provider = selectedProvider
		cluster = clusters[0]
	}
	var target krknv1alpha1.KrknTargetRequest
	if err := h.client.Get(ctx, client.ObjectKey{Name: requestID, Namespace: h.namespace}, &target); err != nil {
		return "", "", fmt.Errorf("target request not found: %w", err)
	}
	if !strings.EqualFold(target.Status.Status, "completed") {
		return "", "", fmt.Errorf("target request is not completed")
	}
	var apiURL string
	for _, candidate := range target.Status.TargetData[provider] {
		if candidate.ClusterName == cluster {
			apiURL = candidate.ClusterAPIURL
			break
		}
	}
	if apiURL == "" {
		return "", "", fmt.Errorf("target cluster not found")
	}
	if err := h.authorizeKrknAITargetAPIURL(ctx, apiURL, action); err != nil {
		return "", "", err
	}
	return provider, cluster, nil
}

func (h *Handler) authorizeKrknAITargetAPIURL(ctx context.Context, apiURL string, action groupauth.Action) error {
	if apiURL == "" {
		return fmt.Errorf("target cluster API URL is required")
	}
	if auth.IsAdmin(ctx) {
		return nil
	}
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		return fmt.Errorf("authentication required")
	}
	allowed, err := groupauth.HasClusterPermission(ctx, h.client, claims.UserID, h.namespace, apiURL, action)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("target access denied")
	}
	return nil
}

func (h *Handler) authorizeKrknAIRunTarget(ctx context.Context, run *krknv1alpha1.KrknAIRun, action groupauth.Action) error {
	if auth.IsAdmin(ctx) {
		return nil
	}
	// Cancellation can use the live request while active; reads never resolve it.
	if action == groupauth.ActionCancel {
		if _, _, err := h.resolveKrknAITarget(ctx, run.Spec.TargetRequestID, run.Spec.TargetClusters, action); err == nil {
			return nil
		} else if !apierrors.IsNotFound(err) {
			return err
		}
	}
	apiURL, err := h.krknAIRunTargetAPIURLFromChildren(ctx, run)
	if err != nil {
		return err
	}
	return h.authorizeKrknAITargetAPIURL(ctx, apiURL, action)
}

func (h *Handler) krknAIRunTargetAPIURLFromChildren(ctx context.Context, run *krknv1alpha1.KrknAIRun) (string, error) {
	if run.Spec.TargetRequestID == "" || len(run.Spec.TargetClusters) != 1 {
		return "", fmt.Errorf("Krkn-AI run has no resolvable target")
	}
	var providerName, clusterName string
	for provider, clusters := range run.Spec.TargetClusters {
		if provider == "" || len(clusters) != 1 || clusters[0] == "" {
			return "", fmt.Errorf("Krkn-AI run has no resolvable target")
		}
		providerName, clusterName = provider, clusters[0]
	}
	var children krknv1alpha1.KrknScenarioRunList
	if err := h.client.List(ctx, &children, client.InNamespace(h.namespace), client.MatchingLabels{
		"krkn.dev/ai-run": krknAIRunLabelValue(run.Name),
	}); err != nil {
		return "", fmt.Errorf("failed to list Krkn-AI child runs: %w", err)
	}
	apiURL := ""
	for i := range children.Items {
		child := &children.Items[i]
		if !hasKrknAIRunOwnerUID(child.OwnerReferences, run.UID) ||
			child.Spec.TargetRequestID != run.Spec.TargetRequestID ||
			len(child.Spec.TargetClusters) != 1 ||
			len(child.Spec.TargetClusters[providerName]) != 1 ||
			child.Spec.TargetClusters[providerName][0] != clusterName {
			continue
		}
		for _, job := range child.Status.ClusterJobs {
			if job.ClusterName != clusterName || job.ClusterAPIURL == "" {
				continue
			}
			if apiURL != "" && apiURL != job.ClusterAPIURL {
				return "", fmt.Errorf("Krkn-AI child runs contain conflicting target API URLs")
			}
			apiURL = job.ClusterAPIURL
		}
	}
	if apiURL == "" {
		return "", fmt.Errorf("target API URL is unavailable from Krkn-AI child runs")
	}
	return apiURL, nil
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

func (h *Handler) writeKrknAIDisabled(w http.ResponseWriter) bool {
	if h.krknAIEnabled {
		return false
	}
	writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{
		Error: "service_unavailable", Message: "Krkn-AI is disabled",
	})
	return true
}

// getKrknAIStatus reports the Helm feature flag without requiring the optional KrknAIRun CRD.
//
// @Summary Get Krkn-AI feature availability
// @Description Report whether Krkn-AI is enabled in this operator deployment.
// @Tags krkn-ai
// @Produce json
// @Success 200 {object} KrknAIStatusResponse "Feature availability"
// @Security BearerAuth
// @Router /krkn-ai/status [get]
func (h *Handler) getKrknAIStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, KrknAIStatusResponse{Enabled: h.krknAIEnabled})
}

// @Summary Validate a Krkn-AI configuration
// @Description Validate YAML against the Krkn-AI configuration schema.
// @Tags krkn-ai
// @Accept json
// @Produce json
// @Param request body KrknAIConfigValidationRequest true "Configuration YAML"
// @Success 200 {object} KrknAIConfigValidationResponse "Configuration is valid"
// @Failure 422 {object} KrknAIConfigValidationResponse "Configuration validation errors"
// @Failure 503 {object} ErrorResponse "Krkn-AI service unavailable or disabled"
// @Security BearerAuth
// @Router /krkn-ai/configs/validate [post]
func (h *Handler) validateKrknAIConfigRoute(w http.ResponseWriter, r *http.Request) {
	if h.writeKrknAIDisabled(w) {
		return
	}
	var request KrknAIConfigValidationRequest
	if err := decodeKrknAIRequest(r, &request); err != nil || strings.TrimSpace(request.ConfigYAML) == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "configYaml is required"})
		return
	}
	if h.validateKrknAIConfig(w, r, request.ConfigYAML) {
		writeJSON(w, http.StatusOK, KrknAIConfigValidationResponse{Valid: true})
	}
}

// validateKrknAIConfig returns true only when the service accepts the YAML.
// On rejection or service failure it writes the appropriate response.
func (h *Handler) validateKrknAIConfig(w http.ResponseWriter, r *http.Request, configYAML string) bool {
	payload, _ := json.Marshal(KrknAIConfigValidationRequest{ConfigYAML: configYAML})
	response, err := h.artifactClient.Do(r.Context(), http.MethodPost, "/v1/configs/validate", payload)
	if err != nil {
		writeKrknAIServiceUnavailable(w)
		return false
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnprocessableEntity {
		var validation KrknAIConfigValidationResponse
		if err := json.NewDecoder(response.Body).Decode(&validation); err != nil {
			writeKrknAIServiceUnavailable(w)
			return false
		}
		writeJSON(w, http.StatusUnprocessableEntity, validation)
		return false
	}
	if response.StatusCode != http.StatusOK {
		writeKrknAIServiceUnavailable(w)
		return false
	}
	var validation KrknAIConfigValidationResponse
	if err := json.NewDecoder(response.Body).Decode(&validation); err != nil || !validation.Valid {
		writeKrknAIServiceUnavailable(w)
		return false
	}
	return true
}

func writeKrknAIServiceUnavailable(w http.ResponseWriter) {
	writeJSONError(w, http.StatusServiceUnavailable, ErrorResponse{
		Error: "service_unavailable", Message: "Krkn-AI artifact service is unavailable",
	})
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
	if h.writeKrknAIDisabled(w) {
		return
	}
	var request KrknAIDiscoveryRequest
	if err := decodeKrknAIRequest(r, &request); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: "invalid discovery request"})
		return
	}
	provider, cluster, err := h.resolveKrknAITarget(r.Context(), request.TargetRequestID, request.TargetClusters, groupauth.ActionRun)
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
// @Failure 422 {object} KrknAIConfigValidationResponse "Configuration validation errors"
// @Failure 503 {object} ErrorResponse "Krkn-AI service unavailable or disabled"
// @Security BearerAuth
// @Router /krkn-ai/configs [post]
func (h *Handler) createKrknAIConfig(w http.ResponseWriter, r *http.Request) {
	if h.writeKrknAIDisabled(w) {
		return
	}
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
	provider, cluster, err := h.resolveKrknAITarget(r.Context(), request.TargetRequestID, request.TargetClusters, groupauth.ActionRun)
	if err != nil {
		h.writeKrknAITargetError(w, err)
		return
	}
	if !h.validateKrknAIConfig(w, r, request.ConfigYAML) {
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
			Labels:      files.BuildFileLabels(fileID, request.Groups, request.AvailableToAll, files.FilePurposeKrknAIConfig, request.Name),
			Annotations: files.BuildFileAnnotations(request.Description, claims.UserID, request.Name),
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
// @Failure 503 {object} ErrorResponse "Krkn-AI is disabled"
// @Security BearerAuth
// @Router /krkn-ai/runs [post]
func (h *Handler) createKrknAIRun(w http.ResponseWriter, r *http.Request) {
	if h.writeKrknAIDisabled(w) {
		return
	}
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
	provider, cluster, err := h.resolveKrknAITarget(r.Context(), request.TargetRequestID, request.TargetClusters, groupauth.ActionRun)
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
		TargetClusters:        request.TargetClusters,
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
		if err := h.authorizeKrknAIRunTarget(r.Context(), run, groupauth.ActionView); err == nil {
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
	if err := h.authorizeKrknAIRunTarget(r.Context(), &run, action); err != nil {
		h.writeKrknAITargetError(w, err)
		return nil, false
	}
	return &run, true
}

// @Summary Get Krkn-AI run results summary
// @Description Combine authorized run metadata with the committed artifact summary.
// @Tags krkn-ai
// @Produce json
// @Param name path string true "KrknAIRun name"
// @Success 200 {object} KrknAIRunSummaryResponse "Run and artifact summary"
// @Failure 403 {object} ErrorResponse "Target access denied"
// @Failure 404 {object} ErrorResponse "Run not found"
// @Failure 502 {object} ErrorResponse "Committed artifact data is corrupt"
// @Failure 503 {object} ErrorResponse "Artifact upload is updating or service unavailable"
// @Security BearerAuth
// @Router /krkn-ai/runs/{name}/results/summary [get]
func (h *Handler) getKrknAIRunResultsSummary(w http.ResponseWriter, r *http.Request, name string) {
	run, ok := h.authorizeKrknAIRun(w, r, name, groupauth.ActionView)
	if !ok {
		return
	}
	path := "/" + krknaiserver.EscapePath("v1/runs/"+string(run.UID)+"/summary")
	response, err := h.artifactClient.Do(r.Context(), http.MethodGet, path, nil)
	if err != nil {
		writeKrknAIServiceUnavailable(w)
		return
	}
	defer response.Body.Close()

	artifact := KrknAIArtifactSummary{
		ArtifactStatus: "not_available", FitnessProgression: []KrknAIFitnessProgression{},
	}
	switch response.StatusCode {
	case http.StatusOK:
		if err := json.NewDecoder(response.Body).Decode(&artifact); err != nil {
			writeJSONError(w, http.StatusBadGateway, ErrorResponse{Error: "artifact_error", Message: "Krkn-AI returned invalid committed summary data"})
			return
		}
	case http.StatusNotFound:
		// A run without a committed manifest is still a valid, visible run.
	default:
		copyKrknAIResponse(w, response, "application/json")
		return
	}
	cluster := ""
	providers := make([]string, 0, len(run.Spec.TargetClusters))
	for provider := range run.Spec.TargetClusters {
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	if len(providers) > 0 && len(run.Spec.TargetClusters[providers[0]]) > 0 {
		clusters := append([]string(nil), run.Spec.TargetClusters[providers[0]]...)
		sort.Strings(clusters)
		cluster = clusters[0]
	}
	if artifact.FitnessProgression == nil {
		artifact.FitnessProgression = []KrknAIFitnessProgression{}
	}
	writeJSON(w, http.StatusOK, KrknAIRunSummaryResponse{
		Name: run.Name, Phase: run.Status.Phase, CreatedAt: run.CreationTimestamp.Time,
		Cluster: cluster, OrchestratorPodName: run.Status.OrchestratorPodName, FailureReason: run.Status.FailureReason,
		ArtifactStatus: artifact.ArtifactStatus, CompletedGenerations: artifact.CompletedGenerations,
		CurrentGeneration:  artifact.CurrentGeneration,
		CompletedScenarios: artifact.CompletedScenarios, ConfiguredGenerations: artifact.ConfiguredGenerations,
		PopulationSize: artifact.PopulationSize, BestFitness: artifact.BestFitness,
		AverageFitness: artifact.AverageFitness, BaselineFitness: artifact.BaselineFitness,
		FitnessProgression: artifact.FitnessProgression,
	})
}

// @Summary List Krkn-AI run scenarios
// @Description Return a globally filtered, sorted, and paginated authorized scenario index merged with matching child job metadata.
// @Tags krkn-ai
// @Produce json
// @Param name path string true "KrknAIRun name"
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Page size (default 100, maximum 500)"
// @Param generation query int false "Generation"
// @Param scenarioType query string false "Scenario type substring"
// @Param search query string false "Scenario ID or type substring"
// @Param sort query string false "generation, scenarioId, scenarioType, fitnessScore, outcome, durationSeconds"
// @Param direction query string false "asc or desc"
// @Success 200 {object} KrknAIScenarioIndexResponse "Scenario index"
// @Failure 403 {object} ErrorResponse "Target access denied"
// @Failure 404 {object} ErrorResponse "Run not found"
// @Failure 502 {object} ErrorResponse "Committed artifact data is corrupt"
// @Failure 503 {object} ErrorResponse "Artifact upload is updating or service unavailable"
// @Security BearerAuth
// @Router /krkn-ai/runs/{name}/results/scenarios [get]
func (h *Handler) getKrknAIRunScenarioIndex(w http.ResponseWriter, r *http.Request, name string) {
	run, ok := h.authorizeKrknAIRun(w, r, name, groupauth.ActionView)
	if !ok {
		return
	}
	query, err := parseKrknAIScenarioQuery(r.URL.Query())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Message: err.Error()})
		return
	}
	initialUID := run.UID
	const endpoint = "v1/runs/"
	path := "/" + krknaiserver.EscapePath(endpoint+string(initialUID)+"/scenarios")
	response, err := h.artifactClient.Do(r.Context(), http.MethodGet, path, nil)
	if err != nil {
		writeKrknAIServiceUnavailable(w)
		return
	}
	defer response.Body.Close()
	index := KrknAIScenarioIndexResponse{Scenarios: []KrknAIScenarioIndexItem{}}
	switch response.StatusCode {
	case http.StatusOK:
		if err := json.NewDecoder(response.Body).Decode(&index); err != nil {
			writeJSONError(w, http.StatusBadGateway, ErrorResponse{Error: "artifact_error", Message: "Krkn-AI returned invalid committed scenario data"})
			return
		}
	case http.StatusNotFound:
		// No manifest means no artifact rows; child runs may still be visible.
	default:
		copyKrknAIResponse(w, response, "application/json")
		return
	}
	if index.Scenarios == nil {
		index.Scenarios = []KrknAIScenarioIndexItem{}
	}
	// Re-authorize immediately before reading child resources; artifact reads do
	// not grant permission to enumerate associated scenario runs.
	run, ok = h.authorizeKrknAIRun(w, r, name, groupauth.ActionView)
	if !ok {
		return
	}
	if run.UID != initialUID {
		writeJSONError(w, http.StatusConflict, ErrorResponse{Error: "run_changed", Message: "Krkn-AI run changed while reading results"})
		return
	}
	if err := h.mergeKrknAIChildRuns(r, run, &index); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Message: "failed to list Krkn-AI scenario runs"})
		return
	}
	filtered := index.Scenarios[:0]
	for _, row := range index.Scenarios {
		if matchesKrknAIScenarioFilter(query, row.Generation, row.ScenarioID, row.ScenarioType) {
			filtered = append(filtered, row)
		}
	}
	sortKrknAIScenarios(filtered, query.sort, query.direction)
	page, limit := query.page, query.limit
	total := len(filtered)
	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}
	if page > totalPages {
		filtered = []KrknAIScenarioIndexItem{}
	} else {
		offset := (page - 1) * limit
		end := offset + limit
		if end > total {
			end = total
		}
		filtered = filtered[offset:end]
	}
	index.Scenarios = filtered
	index.Pagination = KrknAIScenarioPagination{Page: page, Limit: limit, Total: total, TotalPages: totalPages}
	writeJSON(w, http.StatusOK, index)
}

// @Summary Get a Krkn-AI scenario result
// @Description Return one committed scenario result after target authorization.
// @Tags krkn-ai
// @Produce json
// @Param name path string true "KrknAIRun name"
// @Param generation path string true "Generation identifier"
// @Param scenarioId path string true "Scenario identifier"
// @Success 200 {object} KrknAIScenarioDetailResponse "Scenario details"
// @Failure 403 {object} ErrorResponse "Target access denied"
// @Failure 404 {object} ErrorResponse "Run or scenario not found"
// @Failure 502 {object} ErrorResponse "Committed artifact data is corrupt"
// @Failure 503 {object} ErrorResponse "Artifact upload is updating or service unavailable"
// @Security BearerAuth
// @Router /krkn-ai/runs/{name}/results/scenarios/{generation}/{scenarioId} [get]
func (h *Handler) getKrknAIRunScenarioDetail(w http.ResponseWriter, r *http.Request, name, generation, scenarioID string) {
	run, ok := h.authorizeKrknAIRun(w, r, name, groupauth.ActionView)
	if !ok {
		return
	}
	path := "/v1/runs/" + krknaiserver.EscapePath(string(run.UID)+"/scenarios/"+generation) + "/" + url.PathEscape(scenarioID)
	response, err := h.artifactClient.Do(r.Context(), http.MethodGet, path, nil)
	if err != nil {
		writeKrknAIServiceUnavailable(w)
		return
	}
	copyKrknAIResponse(w, response, "application/json")
}

func (h *Handler) mergeKrknAIChildRuns(r *http.Request, run *krknv1alpha1.KrknAIRun, index *KrknAIScenarioIndexResponse) error {
	var children krknv1alpha1.KrknScenarioRunList
	if err := h.client.List(r.Context(), &children, client.InNamespace(h.namespace), client.MatchingLabels{
		"krkn.dev/ai-run": krknAIRunLabelValue(run.Name),
	}); err != nil {
		return err
	}
	rows := make(map[krknAIScenarioKey]int, len(index.Scenarios))
	unique := index.Scenarios[:0]
	for _, scenario := range index.Scenarios {
		key := krknAIScenarioKey{generation: scenario.Generation, scenarioID: scenario.ScenarioID}
		if _, found := rows[key]; found {
			continue
		}
		rows[key] = len(unique)
		unique = append(unique, scenario)
	}
	index.Scenarios = unique
	for i := range children.Items {
		child := &children.Items[i]
		if !hasKrknAIRunOwnerUID(child.OwnerReferences, run.UID) {
			continue
		}
		generationLabel := child.Labels["krkn.dev/generation-id"]
		scenarioID := child.Labels["krkn.dev/scenario-id"]
		generation, err := strconv.Atoi(generationLabel)
		if err != nil || generation < 0 || scenarioID == "" {
			continue
		}
		scenarioType := child.Labels["krkn.dev/scenario-name"]
		key := krknAIScenarioKey{generation: generation, scenarioID: scenarioID}
		rowIndex, found := rows[key]
		if !found {
			index.Scenarios = append(index.Scenarios, KrknAIScenarioIndexItem{Generation: generation, ScenarioID: scenarioID})
			rowIndex = len(index.Scenarios) - 1
			rows[key] = rowIndex
		}
		row := &index.Scenarios[rowIndex]
		row.ChildRunName = child.Name
		row.Phase = child.Status.Phase
		if row.ScenarioType == "" {
			row.ScenarioType = scenarioType
		}
		for _, job := range child.Status.ClusterJobs {
			if row.JobID == "" && job.JobID != "" {
				row.JobID = job.JobID
			}
			if row.PodName == "" && job.PodName != "" {
				row.PodName = job.PodName
			}
			if row.JobID != "" && row.PodName != "" {
				break
			}
		}
	}
	return nil
}

func krknAIRunLabelValue(name string) string {
	if len(name) <= 63 {
		return name
	}
	return files.HashLogicalName(name)
}

func hasKrknAIRunOwnerUID(references []metav1.OwnerReference, uid types.UID) bool {
	for _, reference := range references {
		if reference.UID == uid {
			return true
		}
	}
	return false
}

type krknAIScenarioKey struct {
	generation int
	scenarioID string
}

const (
	krknAIScenarioBaselineID          = "baseline"
	krknAIScenarioSortDurationSeconds = "durationSeconds"
	krknAIScenarioSortFitnessScore    = "fitnessScore"
	krknAIScenarioSortGeneration      = "generation"
	krknAIScenarioSortID              = "scenarioId"
	krknAIScenarioSortType            = "scenarioType"
	krknAIScenarioSortOutcome         = "outcome"
	krknAIScenarioOutcomeFailed       = "failed"
)

type krknAIScenarioQuery struct {
	page          int
	limit         int
	generation    int
	hasGeneration bool
	scenarioType  string
	search        string
	sort          string
	direction     string
}

func parseKrknAIScenarioQuery(values url.Values) (krknAIScenarioQuery, error) {
	query := krknAIScenarioQuery{page: 1, limit: 100, sort: krknAIScenarioSortGeneration, direction: "asc"}
	if values.Has("page") {
		page, err := strconv.Atoi(values.Get("page"))
		if err != nil || page <= 0 {
			return query, fmt.Errorf("page must be a positive integer")
		}
		query.page = page
	}
	if values.Has("limit") {
		limit, err := strconv.Atoi(values.Get("limit"))
		if err != nil || limit <= 0 || limit > 500 {
			return query, fmt.Errorf("limit must be an integer from 1 to 500")
		}
		query.limit = limit
	}
	if values.Has(krknAIScenarioSortGeneration) {
		generation, err := strconv.Atoi(values.Get(krknAIScenarioSortGeneration))
		if err != nil || generation < 0 {
			return query, fmt.Errorf("generation must be a non-negative integer")
		}
		query.generation = generation
		query.hasGeneration = true
	}
	if values.Has(krknAIScenarioSortType) {
		query.scenarioType = strings.ToLower(values.Get(krknAIScenarioSortType))
	}
	if values.Has("search") {
		query.search = strings.ToLower(values.Get("search"))
	}
	if values.Has("sort") {
		query.sort = values.Get("sort")
		switch query.sort {
		case krknAIScenarioSortGeneration, krknAIScenarioSortID, krknAIScenarioSortType,
			krknAIScenarioSortFitnessScore, krknAIScenarioSortOutcome, krknAIScenarioSortDurationSeconds:
		default:
			return query, fmt.Errorf("sort must be one of generation, scenarioId, scenarioType, fitnessScore, outcome, durationSeconds")
		}
	}
	if values.Has("direction") {
		query.direction = values.Get("direction")
		if query.direction != "asc" && query.direction != "desc" {
			return query, fmt.Errorf("direction must be asc or desc")
		}
	}
	return query, nil
}

func matchesKrknAIScenarioFilter(query krknAIScenarioQuery, generation int, scenarioID, scenarioType string) bool {
	if query.hasGeneration && generation != query.generation {
		return false
	}
	var scenarioTypeLower string
	if query.scenarioType != "" || query.search != "" {
		scenarioTypeLower = strings.ToLower(scenarioType)
	}
	if query.scenarioType != "" && !strings.Contains(scenarioTypeLower, query.scenarioType) {
		return false
	}
	if query.search != "" &&
		!strings.Contains(strings.ToLower(scenarioID), query.search) &&
		!strings.Contains(scenarioTypeLower, query.search) {
		return false
	}
	return true
}

func sortKrknAIScenarios(scenarios []KrknAIScenarioIndexItem, sortKey, direction string) {
	descending := direction == "desc"
	sort.SliceStable(scenarios, func(i, j int) bool {
		left, right := scenarios[i], scenarios[j]
		comparison := 0
		switch sortKey {
		case krknAIScenarioSortGeneration:
			comparison = compareInt(left.Generation, right.Generation)
		case krknAIScenarioSortID:
			comparison = compareKrknAIScenarioIDs(left.ScenarioID, right.ScenarioID)
		case krknAIScenarioSortType:
			comparison = compareKrknAIScenarioStrings(left.ScenarioType, right.ScenarioType)
		case krknAIScenarioSortFitnessScore:
			comparison = compareKrknAIScenarioMeasurements(left.FitnessScore, right.FitnessScore)
		case krknAIScenarioSortDurationSeconds:
			comparison = compareKrknAIScenarioMeasurements(left.DurationSeconds, right.DurationSeconds)
		case krknAIScenarioSortOutcome:
			comparison = compareKrknAIScenarioStrings(krknAIScenarioStatus(left), krknAIScenarioStatus(right))
		}
		if comparison != 0 {
			if sortKey == krknAIScenarioSortFitnessScore && (left.FitnessScore == nil || right.FitnessScore == nil) ||
				sortKey == krknAIScenarioSortDurationSeconds && (left.DurationSeconds == nil || right.DurationSeconds == nil) {
				return comparison < 0
			}
			if descending {
				return comparison > 0
			}
			return comparison < 0
		}
		return compareKrknAIScenarioIdentity(left, right) < 0
	})
}

func compareInt(left, right int) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func compareKrknAIScenarioIDs(left, right string) int {
	if left == krknAIScenarioBaselineID && right != krknAIScenarioBaselineID {
		return -1
	}
	if right == krknAIScenarioBaselineID && left != krknAIScenarioBaselineID {
		return 1
	}
	leftNumeric, rightNumeric := isKrknAIScenarioNumericID(left), isKrknAIScenarioNumericID(right)
	if leftNumeric && rightNumeric {
		leftSignificant, rightSignificant := left, right
		for len(leftSignificant) > 1 && leftSignificant[0] == '0' {
			leftSignificant = leftSignificant[1:]
		}
		for len(rightSignificant) > 1 && rightSignificant[0] == '0' {
			rightSignificant = rightSignificant[1:]
		}
		if len(leftSignificant) < len(rightSignificant) {
			return -1
		}
		if len(leftSignificant) > len(rightSignificant) {
			return 1
		}
		if comparison := strings.Compare(leftSignificant, rightSignificant); comparison != 0 {
			return comparison
		}
	}
	return compareKrknAIScenarioStrings(left, right)
}

func isKrknAIScenarioNumericID(value string) bool {
	if value == "" {
		return false
	}
	for i := range value {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

func compareKrknAIScenarioStrings(left, right string) int {
	leftIndex, rightIndex := 0, 0
	for leftIndex < len(left) && rightIndex < len(right) {
		leftRune, leftSize := utf8.DecodeRuneInString(left[leftIndex:])
		rightRune, rightSize := utf8.DecodeRuneInString(right[rightIndex:])
		leftRune, rightRune = unicode.ToLower(leftRune), unicode.ToLower(rightRune)
		if leftRune < rightRune {
			return -1
		}
		if leftRune > rightRune {
			return 1
		}
		leftIndex += leftSize
		rightIndex += rightSize
	}
	if leftIndex < len(left) {
		return 1
	}
	if rightIndex < len(right) {
		return -1
	}
	return strings.Compare(left, right)
}

func compareKrknAIScenarioMeasurements(left, right *float64) int {
	if left == nil && right != nil {
		return 1
	}
	if left != nil && right == nil {
		return -1
	}
	if left == nil {
		return 0
	}
	if *left < *right {
		return -1
	}
	if *left > *right {
		return 1
	}
	return 0
}

func compareKrknAIScenarioIdentity(left, right KrknAIScenarioIndexItem) int {
	if comparison := compareInt(left.Generation, right.Generation); comparison != 0 {
		return comparison
	}
	return compareKrknAIScenarioIDs(left.ScenarioID, right.ScenarioID)
}

func krknAIScenarioStatus(row KrknAIScenarioIndexItem) string {
	if row.Phase != "" {
		return row.Phase
	}
	switch row.Outcome {
	case "succeeded":
		return "Succeeded"
	case krknAIScenarioOutcomeFailed:
		return "Failed"
	default:
		return "Result pending"
	}
}

func copyKrknAIResponse(w http.ResponseWriter, response *http.Response, contentType string) {
	defer response.Body.Close()
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, response.Body)
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
	message := err.Error()
	statusCode := http.StatusForbidden
	switch {
	case strings.HasPrefix(message, "target request not found:"),
		message == "target cluster not found":
		statusCode = http.StatusNotFound
	case strings.Contains(message, "exactly one"),
		strings.Contains(message, "not completed"):
		statusCode = http.StatusBadRequest
	}
	writeJSONError(w, statusCode, ErrorResponse{Error: "target_access", Message: message})
}
