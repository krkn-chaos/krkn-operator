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
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/krkn-chaos/krknctl/pkg/config"
	"github.com/krkn-chaos/krknctl/pkg/provider"
	"github.com/krkn-chaos/krknctl/pkg/provider/factory"
	"github.com/krkn-chaos/krknctl/pkg/provider/models"
	"github.com/krkn-chaos/krknctl/pkg/typing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	pb "github.com/krkn-chaos/krkn-operator/proto/dataprovider"
)

// Handler contains the dependencies for API handlers
type Handler struct {
	client         client.Client
	clientset      kubernetes.Interface
	namespace      string
	grpcServerAddr string
}

// NewHandler creates a new Handler
func NewHandler(client client.Client, clientset kubernetes.Interface, namespace string, grpcServerAddr string) *Handler {
	return &Handler{
		client:         client,
		clientset:      clientset,
		namespace:      namespace,
		grpcServerAddr: grpcServerAddr,
	}
}

// GetClusters handles GET /api/v1/clusters endpoint
// It fetches the KrknTargetRequest CR by the provided ID and returns the target data
func (h *Handler) GetClusters(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "id parameter is required",
		})
		return
	}

	// Fetch the KrknTargetRequest CR
	var targetRequest krknv1alpha1.KrknTargetRequest
	err := h.client.Get(context.Background(), types.NamespacedName{
		Name:      id,
		Namespace: h.namespace,
	}, &targetRequest)

	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "KrknTargetRequest with id '" + id + "' not found",
			})
		} else {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to fetch KrknTargetRequest: " + err.Error(),
			})
		}
		return
	}

	// Check if the request is completed
	if targetRequest.Status.Status != "Completed" {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "KrknTargetRequest with id '" + id + "' is not completed",
		})
		return
	}

	// Return the target data
	response := ClustersResponse{
		TargetData: targetRequest.Status.TargetData,
		Status:     targetRequest.Status.Status,
	}

	writeJSON(w, http.StatusOK, response)
}

// GetNodes handles GET /api/v1/nodes endpoint
// Supports both new and legacy parameter formats:
// - New: ?targetUUID=<uuid>
// - Legacy: ?id=<targetRequestId>&cluster-name=<clusterName>
func (h *Handler) GetNodes(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// New parameter (KrknOperatorTarget)
	targetUUID := r.URL.Query().Get("targetUUID")

	// Legacy parameters (KrknTargetRequest)
	id := r.URL.Query().Get("id")
	clusterName := r.URL.Query().Get("cluster-name")

	// Validate that at least one set of parameters is provided
	if targetUUID == "" && (id == "" || clusterName == "") {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Either targetUUID (new) or id+cluster-name (legacy) parameters are required",
		})
		return
	}

	// Get kubeconfig using unified helper function
	kubeconfigBase64, err := h.getKubeconfig(ctx, targetUUID, id, clusterName)
	if err != nil {
		if client.IgnoreNotFound(err) == nil || strings.Contains(err.Error(), "not found") {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
			return
		}
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
		return
	}

	// Call gRPC service to get nodes
	nodes, err := h.callGetNodesGRPC(kubeconfigBase64)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get nodes from gRPC service: " + err.Error(),
		})
		return
	}

	// Return the list of nodes
	response := NodesResponse{
		Nodes: nodes,
	}

	writeJSON(w, http.StatusOK, response)
}

// HealthCheck handles GET /api/v1/health endpoint
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// GetTargetByUUID handles GET /api/v1/targets/{uuid} endpoint (legacy - checks KrknTargetRequest status)
// This endpoint checks the status of a KrknTargetRequest CR created by krkn-operator-acm
func (h *Handler) GetTargetByUUID(w http.ResponseWriter, r *http.Request) {
	uuid, err := extractPathSuffix(r.URL.Path, "/api/v1/targets/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "UUID " + err.Error(),
		})
		return
	}

	var targetRequest krknv1alpha1.KrknTargetRequest
	if err := h.client.Get(context.Background(), types.NamespacedName{
		Name:      uuid,
		Namespace: h.namespace,
	}, &targetRequest); err != nil {
		if client.IgnoreNotFound(err) == nil {
			w.WriteHeader(http.StatusNotFound)
		} else {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to fetch KrknTargetRequest: " + err.Error(),
			})
		}
		return
	}

	if targetRequest.Status.Status != "Completed" {
		w.WriteHeader(http.StatusContinue)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// PostTarget handles POST /api/v1/targets endpoint (legacy - creates KrknTargetRequest)
// This endpoint triggers the krkn-operator-acm to discover and return target clusters
func (h *Handler) PostTarget(w http.ResponseWriter, r *http.Request) {
	// Generate a new UUID
	newUUID := uuid.New().String()

	// Create a new KrknTargetRequest CR
	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      newUUID,
			Namespace: h.namespace,
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{
			UUID: newUUID,
		},
	}

	// Create the CR in the cluster
	err := h.client.Create(context.Background(), targetRequest)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create KrknTargetRequest: " + err.Error(),
		})
		return
	}

	// Return 102 Processing with the UUID
	response := map[string]string{
		"uuid": newUUID,
	}
	writeJSON(w, http.StatusProcessing, response)
}

// TargetsHandler handles both GET /api/v1/targets/{UUID} and POST /api/v1/targets endpoints
// It routes to the appropriate handler based on the HTTP method
func (h *Handler) TargetsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.GetTargetByUUID(w, r)
	} else if r.Method == http.MethodPost {
		h.PostTarget(w, r)
	} else {
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only GET and POST methods are allowed",
		})
	}
}

// convertInputFields converts krknctl InputField models to API InputFieldResponse format.
// This ensures Type fields are serialized as strings instead of int64 enums.
func convertInputFields(fields []typing.InputField) []InputFieldResponse {
	result := make([]InputFieldResponse, 0, len(fields))
	for _, field := range fields {
		result = append(result, InputFieldResponse{
			Name:              field.Name,
			ShortDescription:  field.ShortDescription,
			Description:       field.Description,
			Variable:          field.Variable,
			Type:              field.Type.String(),
			Default:           field.Default,
			Validator:         field.Validator,
			ValidationMessage: field.ValidationMessage,
			Separator:         field.Separator,
			AllowedValues:     field.AllowedValues,
			Required:          field.Required,
			MountPath:         field.MountPath,
			Requires:          field.Requires,
			MutuallyExcludes:  field.MutuallyExcludes,
			Secret:            field.Secret,
		})
	}
	return result
}

// writeJSON writes a JSON response with the given status code
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeJSONError writes a JSON error response with the given status code
func writeJSONError(w http.ResponseWriter, status int, err ErrorResponse) {
	writeJSON(w, status, err)
}

// callGetNodesGRPC calls the data provider gRPC service to get nodes
func (h *Handler) callGetNodesGRPC(kubeconfigBase64 string) ([]string, error) {
	// Create gRPC connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		h.grpcServerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Create client
	grpcClient := pb.NewDataProviderServiceClient(conn)

	// Call GetNodes RPC
	req := &pb.GetNodesRequest{
		KubeconfigBase64: kubeconfigBase64,
	}

	resp, err := grpcClient.GetNodes(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Nodes, nil
}

// parseRegistryRequest parses and validates the registry request from the HTTP body.
// Returns the registry configuration, provider mode, and any error.
func parseRegistryRequest(r *http.Request) (*models.RegistryV2, provider.Mode, error) {
	if r.ContentLength == 0 {
		return nil, provider.Quay, nil
	}

	var req ScenariosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, provider.Quay, fmt.Errorf("invalid request body: %w", err)
	}

	if req.RegistryURL == "" && req.ScenarioRepository == "" {
		return nil, provider.Quay, nil
	}

	if req.RegistryURL == "" || req.ScenarioRepository == "" {
		return nil, provider.Quay, fmt.Errorf("both registryUrl and scenarioRepository are required for private registry")
	}

	registry := &models.RegistryV2{
		Username:           req.Username,
		Password:           req.Password,
		Token:              req.Token,
		RegistryURL:        req.RegistryURL,
		ScenarioRepository: req.ScenarioRepository,
		SkipTLS:            req.SkipTLS,
		Insecure:           req.Insecure,
	}

	return registry, provider.Private, nil
}

// createScenarioProvider creates and returns a scenario provider instance.
// Returns an error if config loading or provider creation fails.
func createScenarioProvider(mode provider.Mode) (provider.ScenarioDataProvider, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load krknctl config: %w", err)
	}

	providerFactory := factory.NewProviderFactory(&cfg)
	scenarioProvider := providerFactory.NewInstance(mode)
	if scenarioProvider == nil {
		return nil, fmt.Errorf("failed to create scenario provider")
	}

	return scenarioProvider, nil
}

// PostScenarios handles POST /api/v1/scenarios endpoint
// It returns the list of available krkn scenarios from quay.io or a private registry
func (h *Handler) PostScenarios(w http.ResponseWriter, r *http.Request) {
	registry, mode, err := parseRegistryRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	scenarioProvider, err := createScenarioProvider(mode)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
		return
	}

	// Get registry images (scenario list)
	scenarioTags, err := scenarioProvider.GetRegistryImages(registry)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get scenarios from registry: " + err.Error(),
		})
		return
	}

	scenarios := make([]ScenarioTag, 0)
	if scenarioTags != nil {
		for _, tag := range *scenarioTags {
			scenarios = append(scenarios, ScenarioTag{
				Name:         tag.Name,
				Digest:       tag.Digest,
				Size:         tag.Size,
				LastModified: tag.LastModified,
			})
		}
	}

	// Return response
	response := ScenariosResponse{
		Scenarios: scenarios,
	}

	writeJSON(w, http.StatusOK, response)
}

// extractPathSuffix extracts a suffix from a URL path given a prefix.
// Returns the suffix and an error if the path is invalid.
func extractPathSuffix(path string, prefix string) (string, error) {
	if len(path) <= len(prefix) {
		return "", fmt.Errorf("path parameter is required")
	}

	suffix := path[len(prefix):]
	if suffix == "" {
		return "", fmt.Errorf("path parameter cannot be empty")
	}

	return suffix, nil
}

// PostScenarioDetail handles POST /api/v1/scenarios/detail/{scenario_name} endpoint
// It returns detailed information about a specific scenario including input fields
func (h *Handler) PostScenarioDetail(w http.ResponseWriter, r *http.Request) {
	scenarioName, err := extractPathSuffix(r.URL.Path, "/api/v1/scenarios/detail/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "scenario_name " + err.Error(),
		})
		return
	}

	registry, mode, err := parseRegistryRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	scenarioProvider, err := createScenarioProvider(mode)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
		return
	}

	// Get scenario detail
	scenarioDetail, err := scenarioProvider.GetScenarioDetail(scenarioName, registry)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get scenario detail: " + err.Error(),
		})
		return
	}

	if scenarioDetail == nil {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Scenario '" + scenarioName + "' not found",
		})
		return
	}

	response := ScenarioDetailResponse{
		Name:         scenarioDetail.Name,
		Digest:       scenarioDetail.Digest,
		Size:         scenarioDetail.Size,
		LastModified: scenarioDetail.LastModified,
		Title:        scenarioDetail.Title,
		Description:  scenarioDetail.Description,
		Fields:       convertInputFields(scenarioDetail.Fields),
	}

	writeJSON(w, http.StatusOK, response)
}

// PostScenarioGlobals handles POST /api/v1/scenarios/globals/{scenario_name} endpoint
// It returns global environment fields for a specific scenario
func (h *Handler) PostScenarioGlobals(w http.ResponseWriter, r *http.Request) {
	scenarioName, err := extractPathSuffix(r.URL.Path, "/api/v1/scenarios/globals/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "scenario_name " + err.Error(),
		})
		return
	}

	registry, mode, err := parseRegistryRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: err.Error(),
		})
		return
	}

	scenarioProvider, err := createScenarioProvider(mode)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
		return
	}

	// Get global environment
	globalDetail, err := scenarioProvider.GetGlobalEnvironment(registry, scenarioName)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get global environment: " + err.Error(),
		})
		return
	}

	if globalDetail == nil {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Global environment for scenario '" + scenarioName + "' not found",
		})
		return
	}

	response := ScenarioDetailResponse{
		Name:         globalDetail.Name,
		Digest:       globalDetail.Digest,
		Size:         globalDetail.Size,
		LastModified: globalDetail.LastModified,
		Title:        globalDetail.Title,
		Description:  globalDetail.Description,
		Fields:       convertInputFields(globalDetail.Fields),
	}

	writeJSON(w, http.StatusOK, response)
}

// PostScenarioRun handles POST /api/v1/scenarios/run endpoint
// It creates and starts a new scenario job as a Kubernetes pod
// createScenarioJob creates a krkn scenario job for a single cluster
// Returns jobId, podName, and error
func (h *Handler) createScenarioJob(
	ctx context.Context,
	req ScenarioRunRequest,
	clusterName string,
) (jobId string, podName string, err error) {
	// Generate unique job ID
	jobId = uuid.New().String()

	// Set default kubeconfig path if not provided
	kubeconfigPath := req.KubeconfigPath
	if kubeconfigPath == "" {
		kubeconfigPath = "/home/krkn/.kube/config"
	}

	// Get kubeconfig using targetRequestId and clusterName
	kubeconfigBase64, err := h.getKubeconfig(ctx, "", req.TargetRequestId, clusterName)
	if err != nil {
		return "", "", fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Decode kubeconfig for ConfigMap
	kubeconfigDecoded, err := base64.StdEncoding.DecodeString(kubeconfigBase64)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode kubeconfig: %w", err)
	}

	// Create ConfigMap for kubeconfig
	kubeconfigConfigMapName := fmt.Sprintf("krkn-job-%s-kubeconfig", jobId)
	kubeconfigConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubeconfigConfigMapName,
			Namespace: h.namespace,
			Labels: map[string]string{
				"krkn-job-id": jobId,
			},
		},
		Data: map[string]string{
			"config": string(kubeconfigDecoded),
		},
	}

	if err := h.client.Create(ctx, kubeconfigConfigMap); err != nil {
		return "", "", fmt.Errorf("failed to create kubeconfig ConfigMap: %w", err)
	}

	// Track created resources for cleanup on error
	var fileConfigMaps []string
	var imagePullSecretName string

	// Cleanup helper
	cleanup := func() {
		h.client.Delete(ctx, kubeconfigConfigMap)
		for _, cm := range fileConfigMaps {
			h.client.Delete(ctx, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cm,
					Namespace: h.namespace,
				},
			})
		}
		if imagePullSecretName != "" {
			h.client.Delete(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      imagePullSecretName,
					Namespace: h.namespace,
				},
			})
		}
	}

	// Create ConfigMaps for user-provided files
	for _, file := range req.Files {
		// Sanitize filename for ConfigMap name
		sanitizedName := strings.ReplaceAll(file.Name, "/", "-")
		sanitizedName = strings.ReplaceAll(sanitizedName, ".", "-")
		configMapName := fmt.Sprintf("krkn-job-%s-file-%s", jobId, sanitizedName)

		// Decode base64 content
		fileContent, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			cleanup()
			return "", "", fmt.Errorf("failed to decode file content for '%s': %w", file.Name, err)
		}

		fileConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: h.namespace,
				Labels: map[string]string{
					"krkn-job-id": jobId,
				},
			},
			Data: map[string]string{
				file.Name: string(fileContent),
			},
		}

		if err := h.client.Create(ctx, fileConfigMap); err != nil {
			cleanup()
			return "", "", fmt.Errorf("failed to create file ConfigMap: %w", err)
		}

		fileConfigMaps = append(fileConfigMaps, configMapName)
	}

	// Handle private registry authentication
	var imagePullSecrets []corev1.LocalObjectReference
	if req.RegistryURL != "" && req.ScenarioRepository != "" {
		imagePullSecretName = fmt.Sprintf("krkn-job-%s-registry", jobId)

		// Build docker config JSON
		authStr := ""
		if req.Token != nil && *req.Token != "" {
			authStr = base64.StdEncoding.EncodeToString([]byte(*req.Token))
		} else if req.Username != nil && req.Password != nil {
			authStr = base64.StdEncoding.EncodeToString([]byte(*req.Username + ":" + *req.Password))
		}

		dockerConfig := map[string]interface{}{
			"auths": map[string]interface{}{
				req.RegistryURL: map[string]string{
					"auth": authStr,
				},
			},
		}

		dockerConfigJSON, _ := json.Marshal(dockerConfig)

		imagePullSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      imagePullSecretName,
				Namespace: h.namespace,
				Labels: map[string]string{
					"krkn-job-id": jobId,
				},
			},
			Type: corev1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				".dockerconfigjson": dockerConfigJSON,
			},
		}

		if err := h.client.Create(ctx, imagePullSecret); err != nil {
			cleanup()
			return "", "", fmt.Errorf("failed to create ImagePullSecret: %w", err)
		}

		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{
			Name: imagePullSecretName,
		})
	}

	// Build volumes and volume mounts
	volumes := []corev1.Volume{
		{
			Name: "kubeconfig",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: kubeconfigConfigMapName,
					},
				},
			},
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "kubeconfig",
			MountPath: kubeconfigPath,
			SubPath:   "config",
		},
	}

	// Add file mounts
	for i, file := range req.Files {
		sanitizedName := strings.ReplaceAll(file.Name, "/", "-")
		sanitizedName = strings.ReplaceAll(sanitizedName, ".", "-")
		volumeName := fmt.Sprintf("file-%d", i)

		volumes = append(volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fileConfigMaps[i],
					},
				},
			},
		})

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: file.MountPath,
			SubPath:   file.Name,
		})
	}

	// Add writable tmp volume
	volumes = append(volumes, corev1.Volume{
		Name: "tmp",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "tmp",
		MountPath: "/tmp",
	})

	envVars := make([]corev1.EnvVar, 0, len(req.Environment))
	for key, value := range req.Environment {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	// SecurityContext for running as krkn user (UID 1001)
	var runAsUser int64 = 1001
	var runAsGroup int64 = 1001
	var fsGroup int64 = 1001

	// Create the pod
	podName = fmt.Sprintf("krkn-job-%s", jobId)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: h.namespace,
			Labels: map[string]string{
				"app":                 "krkn-scenario",
				"krkn-job-id":         jobId,
				"krkn-scenario-name":  req.ScenarioName,
				"krkn-cluster-name":   clusterName,
				"krkn-target-request": req.TargetRequestId,
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "krkn-operator-krkn-scenario-runner",
			RestartPolicy:      corev1.RestartPolicyNever,
			ImagePullSecrets:   imagePullSecrets,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser:  &runAsUser,
				RunAsGroup: &runAsGroup,
				FSGroup:    &fsGroup,
			},
			Containers: []corev1.Container{
				{
					Name:            "scenario",
					Image:           req.ScenarioImage,
					Env:             envVars,
					VolumeMounts:    volumeMounts,
					ImagePullPolicy: corev1.PullAlways,
				},
			},
			Volumes: volumes,
		},
	}

	if err := h.client.Create(ctx, pod); err != nil {
		cleanup()
		return "", "", fmt.Errorf("failed to create pod: %w", err)
	}

	return jobId, podName, nil
}

func (h *Handler) PostScenarioRun(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req ScenarioRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.TargetRequestId == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "targetRequestId is required",
		})
		return
	}

	if len(req.ClusterNames) == 0 {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "clusterNames is required and must contain at least one cluster name",
		})
		return
	}

	if req.ScenarioImage == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "scenarioImage is required",
		})
		return
	}

	if req.ScenarioName == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "scenarioName is required",
		})
		return
	}

	// Validate cluster names (no duplicates or empty strings)
	seen := make(map[string]bool, len(req.ClusterNames))
	for _, clusterName := range req.ClusterNames {
		if clusterName == "" {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "clusterNames cannot contain empty strings",
			})
			return
		}
		if seen[clusterName] {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "clusterNames contains duplicates: " + clusterName,
			})
			return
		}
		seen[clusterName] = true
	}

	ctx := context.Background()

	// Generate scenario run name
	scenarioRunName := fmt.Sprintf("%s-%s", req.ScenarioName, uuid.New().String()[:8])

	// Create KrknScenarioRun CR
	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scenarioRunName,
			Namespace: h.namespace,
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestId: req.TargetRequestId,
			ClusterNames:    req.ClusterNames,
			ScenarioName:    req.ScenarioName,
			ScenarioImage:   req.ScenarioImage,
			KubeconfigPath:  req.KubeconfigPath,
			Environment:     req.Environment,
			RegistryURL:     req.RegistryURL,
			ScenarioRepository: req.ScenarioRepository,
		},
	}

	// Convert FileMount from API type to CRD type
	if len(req.Files) > 0 {
		scenarioRun.Spec.Files = make([]krknv1alpha1.FileMount, len(req.Files))
		for i, f := range req.Files {
			scenarioRun.Spec.Files[i] = krknv1alpha1.FileMount{
				Name:      f.Name,
				Content:   f.Content,
				MountPath: f.MountPath,
			}
		}
	}

	// Set optional registry auth fields
	if req.Token != nil {
		scenarioRun.Spec.Token = *req.Token
	}
	if req.Username != nil {
		scenarioRun.Spec.Username = *req.Username
	}
	if req.Password != nil {
		scenarioRun.Spec.Password = *req.Password
	}

	// Create the CR
	if err := h.client.Create(ctx, scenarioRun); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create scenario run: " + err.Error(),
		})
		return
	}

	response := ScenarioRunCreateResponse{
		ScenarioRunName: scenarioRunName,
		ClusterNames:    req.ClusterNames,
		TotalTargets:    len(req.ClusterNames),
	}

	writeJSON(w, http.StatusCreated, response)
}

// GetScenarioRunStatus handles GET /api/v1/scenarios/run/{scenarioRunName} endpoint
// It returns the current status of a scenario run
func (h *Handler) GetScenarioRunStatus(w http.ResponseWriter, r *http.Request) {
	scenarioRunName, err := extractPathSuffix(r.URL.Path, "/api/v1/scenarios/run/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "scenarioRunName " + err.Error(),
		})
		return
	}

	ctx := context.Background()

	// Fetch the KrknScenarioRun CR
	var scenarioRun krknv1alpha1.KrknScenarioRun
	err = h.client.Get(ctx, client.ObjectKey{
		Name:      scenarioRunName,
		Namespace: h.namespace,
	}, &scenarioRun)

	if err != nil {
		status := http.StatusInternalServerError
		errMsg := "Failed to fetch scenario run: " + err.Error()
		errCode := "internal_error"

		if client.IgnoreNotFound(err) == nil {
			status = http.StatusNotFound
			errMsg = "Scenario run '" + scenarioRunName + "' not found"
			errCode = "not_found"
		}

		writeJSONError(w, status, ErrorResponse{Error: errCode, Message: errMsg})
		return
	}

	// Convert ClusterJobStatus to response type
	clusterJobs := make([]ClusterJobStatusResponse, len(scenarioRun.Status.ClusterJobs))
	for i, job := range scenarioRun.Status.ClusterJobs {
		clusterJobs[i] = ClusterJobStatusResponse{
			ClusterName:    job.ClusterName,
			JobId:          job.JobId,
			PodName:        job.PodName,
			Phase:          job.Phase,
			Message:        job.Message,
			StartTime:      convertMetaTime(job.StartTime),
			CompletionTime: convertMetaTime(job.CompletionTime),
		}
	}

	response := ScenarioRunStatusResponse{
		ScenarioRunName: scenarioRunName,
		Phase:           scenarioRun.Status.Phase,
		TotalTargets:    scenarioRun.Status.TotalTargets,
		SuccessfulJobs:  scenarioRun.Status.SuccessfulJobs,
		FailedJobs:      scenarioRun.Status.FailedJobs,
		RunningJobs:     scenarioRun.Status.RunningJobs,
		ClusterJobs:     clusterJobs,
	}

	writeJSON(w, http.StatusOK, response)
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now - in production you should validate the origin
		return true
	},
}

// GetScenarioRunLogs handles GET /api/v1/scenarios/run/{scenarioRunName}/jobs/{jobId}/logs endpoint
// It streams the stdout/stderr logs of a running or completed job via WebSocket
func (h *Handler) GetScenarioRunLogs(w http.ResponseWriter, r *http.Request) {
	logger := log.Log.WithName("websocket-logs")

	// Upgrade to WebSocket IMMEDIATELY to avoid protocol mismatch
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(err, "WebSocket upgrade failed",
			"url", r.URL.String(),
			"headers", r.Header,
			"client_ip", r.RemoteAddr)
		return
	}
	defer conn.Close()

	// Extract scenarioRunName and jobId from path
	// Path format: /api/v1/scenarios/run/{scenarioRunName}/jobs/{jobId}/logs
	path := r.URL.Path
	prefix := "/api/v1/scenarios/run/"

	if !strings.HasPrefix(path, prefix) {
		logger.Error(nil, "Invalid logs endpoint path", "path", path)
		conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Invalid logs endpoint"))
		return
	}

	// Remove prefix
	remainder := path[len(prefix):]

	// Split by "/jobs/" and "/logs"
	parts := strings.Split(remainder, "/jobs/")
	if len(parts) != 2 {
		logger.Error(nil, "Invalid logs endpoint path format", "path", path)
		conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Invalid path format. Expected: /api/v1/scenarios/run/{scenarioRunName}/jobs/{jobId}/logs"))
		return
	}

	scenarioRunName := parts[0]
	jobIdAndLogs := parts[1]

	// Extract jobId (remove "/logs" suffix)
	if !strings.HasSuffix(jobIdAndLogs, "/logs") {
		logger.Error(nil, "Invalid logs endpoint path format", "path", path)
		conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Invalid path format. Expected: /api/v1/scenarios/run/{scenarioRunName}/jobs/{jobId}/logs"))
		return
	}

	jobId := strings.TrimSuffix(jobIdAndLogs, "/logs")

	if scenarioRunName == "" || jobId == "" {
		logger.Error(nil, "Empty scenarioRunName or jobId in request path", "path", path)
		conn.WriteMessage(websocket.TextMessage, []byte("ERROR: scenarioRunName and jobId cannot be empty"))
		return
	}

	logger.Info("WebSocket connection established", "scenarioRunName", scenarioRunName, "jobId", jobId, "client_ip", r.RemoteAddr)

	ctx := context.Background()

	// Find pod by jobId label (no need to fetch the CR)
	var podList corev1.PodList
	if err := h.client.List(ctx, &podList, client.InNamespace(h.namespace), client.MatchingLabels{
		"krkn-job-id": jobId,
	}); err != nil {
		logger.Error(err, "Failed to list pods", "jobId", jobId)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: Failed to list pods: %s", err.Error())))
		return
	}

	if len(podList.Items) == 0 {
		logger.Error(nil, "Job not found", "jobId", jobId)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: Job with ID '%s' not found", jobId)))
		return
	}

	pod := podList.Items[0]
	logger.Info("Found pod for job", "scenarioRunName", scenarioRunName, "jobId", jobId, "podName", pod.Name, "podPhase", pod.Status.Phase)

	// Parse query parameters
	follow := r.URL.Query().Get("follow") == "true"
	timestamps := r.URL.Query().Get("timestamps") == "true"
	tailLinesStr := r.URL.Query().Get("tailLines")

	// Build pod logs options
	logOptions := &corev1.PodLogOptions{
		Container:  "scenario",
		Follow:     follow,
		Timestamps: timestamps,
	}

	// Parse tailLines if provided
	if tailLinesStr != "" {
		tailLines, err := strconv.ParseInt(tailLinesStr, 10, 64)
		if err == nil && tailLines > 0 {
			logOptions.TailLines = &tailLines
		}
	}

	logger.Info("Opening log stream",
		"scenarioRunName", scenarioRunName,
		"jobId", jobId,
		"podName", pod.Name,
		"follow", follow,
		"timestamps", timestamps)

	// Get log stream from Kubernetes API
	req := h.clientset.CoreV1().Pods(h.namespace).GetLogs(pod.Name, logOptions)
	stream, err := req.Stream(ctx)
	if err != nil {
		logger.Error(err, "Failed to open log stream",
			"scenarioRunName", scenarioRunName,
			"jobId", jobId,
			"podName", pod.Name,
			"namespace", h.namespace)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: Failed to open log stream: %s", err.Error())))
		return
	}
	defer stream.Close()

	logger.Info("Streaming logs started", "scenarioRunName", scenarioRunName, "jobId", jobId, "podName", pod.Name)

	// Read logs line by line and send via WebSocket
	scanner := bufio.NewScanner(stream)
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		err := conn.WriteMessage(websocket.TextMessage, []byte(line))
		if err != nil {
			logger.Error(err, "Failed to write log line to WebSocket, client likely disconnected",
				"scenarioRunName", scenarioRunName,
				"jobId", jobId,
				"podName", pod.Name,
				"linesStreamed", lineCount)
			return
		}
		lineCount++
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		logger.Error(err, "Log stream scanner error",
			"scenarioRunName", scenarioRunName,
			"jobId", jobId,
			"podName", pod.Name,
			"linesStreamed", lineCount)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: Log stream error: %s", err.Error())))
		return
	}

	logger.Info("Log streaming completed",
		"scenarioRunName", scenarioRunName,
		"jobId", jobId,
		"podName", pod.Name,
		"totalLines", lineCount)

	// Send close message
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

// ListScenarioRuns handles GET /api/v1/scenarios/run endpoint
// It returns a list of all scenario jobs
func (h *Handler) ListScenarioRuns(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Parse query parameters for filtering
	statusFilter := r.URL.Query().Get("status")
	scenarioNameFilter := r.URL.Query().Get("scenarioName")
	clusterNameFilter := r.URL.Query().Get("clusterName")

	// List all pods with krkn-scenario label
	var podList corev1.PodList
	err := h.client.List(ctx, &podList, client.InNamespace(h.namespace), client.MatchingLabels{
		"app": "krkn-scenario",
	})

	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list pods: " + err.Error(),
		})
		return
	}

	var jobs []JobStatusResponse

	for _, pod := range podList.Items {
		jobId := pod.Labels["krkn-job-id"]
		clusterName := pod.Labels["krkn-cluster-name"]
		scenarioName := pod.Labels["krkn-scenario-name"]

		if (statusFilter != "" && string(pod.Status.Phase) != statusFilter) ||
			(scenarioNameFilter != "" && scenarioName != scenarioNameFilter) ||
			(clusterNameFilter != "" && clusterName != clusterNameFilter) {
			continue
		}

		jobStatus := JobStatusResponse{
			JobId:        jobId,
			ClusterName:  clusterName,
			ScenarioName: scenarioName,
			Status:       string(pod.Status.Phase),
			PodName:      pod.Name,
		}

		if pod.Status.StartTime != nil {
			startTime := pod.Status.StartTime.Time
			jobStatus.StartTime = &startTime
		}

		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionFalse &&
				(pod.Status.Phase == "Succeeded" || pod.Status.Phase == "Failed") {
				completionTime := condition.LastTransitionTime.Time
				jobStatus.CompletionTime = &completionTime
			}
		}

		jobs = append(jobs, jobStatus)
	}

	response := JobsListResponse{
		Jobs: jobs,
	}

	writeJSON(w, http.StatusOK, response)
}

// DeleteScenarioRun handles DELETE /api/v1/scenarios/run/{jobId} endpoint
// It stops and deletes a running job
func (h *Handler) DeleteScenarioRun(w http.ResponseWriter, r *http.Request) {
	jobId, err := extractPathSuffix(r.URL.Path, "/api/v1/scenarios/run/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "jobId " + err.Error(),
		})
		return
	}

	ctx := context.Background()

	var podList corev1.PodList
	if err := h.client.List(ctx, &podList, client.InNamespace(h.namespace), client.MatchingLabels{
		"krkn-job-id": jobId,
	}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list pods: " + err.Error(),
		})
		return
	}

	if len(podList.Items) == 0 {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Job with ID '" + jobId + "' not found",
		})
		return
	}

	pod := podList.Items[0]

	gracePeriod := int64(5)
	deleteOptions := client.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	if err := h.client.Delete(ctx, &pod, &deleteOptions); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete pod: " + err.Error(),
		})
		return
	}

	var configMapList corev1.ConfigMapList
	if err := h.client.List(ctx, &configMapList, client.InNamespace(h.namespace), client.MatchingLabels{
		"krkn-job-id": jobId,
	}); err == nil {
		for _, cm := range configMapList.Items {
			h.client.Delete(ctx, &cm)
		}
	}

	var secretList corev1.SecretList
	if err := h.client.List(ctx, &secretList, client.InNamespace(h.namespace), client.MatchingLabels{
		"krkn-job-id": jobId,
	}); err == nil {
		for _, secret := range secretList.Items {
			h.client.Delete(ctx, &secret)
		}
	}

	response := JobStatusResponse{
		JobId:   jobId,
		Status:  "Stopped",
		Message: "Job stopped and deleted successfully",
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) ScenariosRunRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if path == "/api/v1/scenarios/run" {
		if r.Method == http.MethodPost {
			h.PostScenarioRun(w, r)
		} else if r.Method == http.MethodGet {
			h.ListScenarioRuns(w, r)
		} else {
			writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
				Error:   "method_not_allowed",
				Message: "Method " + r.Method + " not allowed for path " + path,
			})
		}
		return
	}

	if strings.HasPrefix(path, "/api/v1/scenarios/run/") {
		// Check for logs endpoint: /api/v1/scenarios/run/{scenarioRunName}/jobs/{jobId}/logs
		if strings.Contains(path, "/jobs/") && strings.HasSuffix(path, "/logs") && r.Method == http.MethodGet {
			h.GetScenarioRunLogs(w, r)
		} else if r.Method == http.MethodGet {
			h.GetScenarioRunStatus(w, r)
		} else if r.Method == http.MethodDelete {
			h.DeleteScenarioRun(w, r)
		} else {
			writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
				Error:   "method_not_allowed",
				Message: "Method " + r.Method + " not allowed for path " + path,
			})
		}
		return
	}

	writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
		Error:   "method_not_allowed",
		Message: "Method " + r.Method + " not allowed for path " + path,
	})
}

// convertMetaTime converts metav1.Time to *time.Time
func convertMetaTime(mt *metav1.Time) *time.Time {
	if mt == nil {
		return nil
	}
	t := mt.Time
	return &t
}
