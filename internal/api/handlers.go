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

// GetClusters handles GET /clusters endpoint
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
			// CR not found
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "KrknTargetRequest with id '" + id + "' not found",
			})
			return
		}
		// Other error
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch KrknTargetRequest: " + err.Error(),
		})
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

// GetNodes handles GET /nodes endpoint
// It retrieves the kubeconfig for a specific cluster from a secret
func (h *Handler) GetNodes(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	clusterName := r.URL.Query().Get("cluster-name")

	// Validate required parameters
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "id parameter is required",
		})
		return
	}

	if clusterName == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "cluster-name parameter is required",
		})
		return
	}

	// Get kubeconfig from target using common helper function
	kubeconfigBase64, err := h.getKubeconfigFromTarget(context.Background(), id, clusterName)
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
	// The kubeconfig is already in base64 format
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

// HealthCheck handles GET /health endpoint
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
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

// getKubeconfigFromTarget retrieves the base64-encoded kubeconfig for a specific cluster
// from the KrknTargetRequest secret
func (h *Handler) getKubeconfigFromTarget(ctx context.Context, targetId string, clusterName string) (string, error) {
	// Fetch the secret with the same name as the KrknTargetRequest ID
	var secret corev1.Secret
	err := h.client.Get(ctx, types.NamespacedName{
		Name:      targetId,
		Namespace: h.namespace,
	}, &secret)

	if err != nil {
		return "", fmt.Errorf("failed to fetch secret: %w", err)
	}

	// Retrieve the managed-clusters JSON from the secret data
	managedClustersBytes, exists := secret.Data["managed-clusters"]
	if !exists {
		return "", fmt.Errorf("managed-clusters not found in secret")
	}

	// Parse the JSON to extract cluster configurations
	// Structure: { "krkn-operator-acm": { "cluster-name": { "kubeconfig": "base64..." } } }
	var managedClusters map[string]map[string]struct {
		Kubeconfig string `json:"kubeconfig"`
	}
	if err := json.Unmarshal(managedClustersBytes, &managedClusters); err != nil {
		return "", fmt.Errorf("failed to parse managed-clusters JSON: %w", err)
	}

	// Get the krkn-operator-acm object
	acmClusters, exists := managedClusters["krkn-operator-acm"]
	if !exists {
		return "", fmt.Errorf("krkn-operator-acm not found in managed-clusters")
	}

	// Check if the requested cluster exists
	clusterConfig, exists := acmClusters[clusterName]
	if !exists {
		return "", fmt.Errorf("cluster '%s' not found in krkn-operator-acm", clusterName)
	}

	// Return the base64-encoded kubeconfig
	return clusterConfig.Kubeconfig, nil
}

// GetTargetByUUID handles GET /targets/{UUID} endpoint
// It checks the status of a KrknTargetRequest CR by UUID
func (h *Handler) GetTargetByUUID(w http.ResponseWriter, r *http.Request) {
	// Extract UUID from path: /targets/{UUID}
	// r.URL.Path looks like "/targets/some-uuid-here"
	path := r.URL.Path
	prefix := "/targets/"

	if len(path) <= len(prefix) {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "UUID parameter is required in path",
		})
		return
	}

	uuid := path[len(prefix):]
	if uuid == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "UUID parameter is required",
		})
		return
	}

	// Fetch the KrknTargetRequest CR by UUID
	var targetRequest krknv1alpha1.KrknTargetRequest
	err := h.client.Get(context.Background(), types.NamespacedName{
		Name:      uuid,
		Namespace: h.namespace,
	}, &targetRequest)

	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// CR not found - return 404
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Other error
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch KrknTargetRequest: " + err.Error(),
		})
		return
	}

	// Check status
	if targetRequest.Status.Status != "Completed" {
		// Not completed - return 100 Continue
		w.WriteHeader(http.StatusContinue)
		return
	}

	// Completed - return 200 OK
	w.WriteHeader(http.StatusOK)
}

// TargetsHandler handles both GET /targets/{UUID} and POST /targets endpoints
// It routes to the appropriate handler based on the HTTP method
func (h *Handler) TargetsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetTargetByUUID(w, r)
	case http.MethodPost:
		h.PostTarget(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only GET and POST methods are allowed",
		})
	}
}

// PostTarget handles POST /targets endpoint
// It creates a new KrknTargetRequest CR with a generated UUID
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

// PostScenarios handles POST /scenarios endpoint
// It returns the list of available krkn scenarios from quay.io or a private registry
func (h *Handler) PostScenarios(w http.ResponseWriter, r *http.Request) {
	// Parse optional request body
	var req ScenariosRequest
	var registry *models.RegistryV2
	var mode provider.Mode

	// Check if body is provided
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Invalid request body: " + err.Error(),
			})
			return
		}

		// If registry info is provided, validate and use private registry mode
		if req.RegistryURL != "" && req.ScenarioRepository != "" {
			registry = &models.RegistryV2{
				Username:           req.Username,
				Password:           req.Password,
				Token:              req.Token,
				RegistryURL:        req.RegistryURL,
				ScenarioRepository: req.ScenarioRepository,
				SkipTLS:            req.SkipTLS,
				Insecure:           req.Insecure,
			}
			mode = provider.Private
		} else if req.RegistryURL != "" || req.ScenarioRepository != "" {
			// Partial registry info provided - error
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Both registryUrl and scenarioRepository are required for private registry",
			})
			return
		} else {
			// Body provided but no registry info - use quay.io
			mode = provider.Quay
		}
	} else {
		// No body provided - default to quay.io
		mode = provider.Quay
	}

	// Load krknctl config
	cfg, err := config.LoadConfig()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to load krknctl config: " + err.Error(),
		})
		return
	}

	// Create provider factory
	providerFactory := factory.NewProviderFactory(&cfg)

	// Get provider instance based on mode
	scenarioProvider := providerFactory.NewInstance(mode)
	if scenarioProvider == nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create scenario provider",
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

	// Convert krknctl models to API response types
	var scenarios []ScenarioTag
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

// PostScenarioDetail handles POST /scenarios/detail/{scenario_name} endpoint
// It returns detailed information about a specific scenario including input fields
func (h *Handler) PostScenarioDetail(w http.ResponseWriter, r *http.Request) {
	// Extract scenario_name from path: /scenarios/detail/{scenario_name}
	path := r.URL.Path
	prefix := "/scenarios/detail/"

	if len(path) <= len(prefix) {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "scenario_name parameter is required in path",
		})
		return
	}

	scenarioName := path[len(prefix):]
	if scenarioName == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "scenario_name parameter cannot be empty",
		})
		return
	}

	// Parse optional request body (same as /scenarios for registry config)
	var req ScenariosRequest
	var registry *models.RegistryV2
	var mode provider.Mode

	// Check if body is provided
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Invalid request body: " + err.Error(),
			})
			return
		}

		// If registry info is provided, validate and use private registry mode
		if req.RegistryURL != "" && req.ScenarioRepository != "" {
			registry = &models.RegistryV2{
				Username:           req.Username,
				Password:           req.Password,
				Token:              req.Token,
				RegistryURL:        req.RegistryURL,
				ScenarioRepository: req.ScenarioRepository,
				SkipTLS:            req.SkipTLS,
				Insecure:           req.Insecure,
			}
			mode = provider.Private
		} else if req.RegistryURL != "" || req.ScenarioRepository != "" {
			// Partial registry info provided - error
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Both registryUrl and scenarioRepository are required for private registry",
			})
			return
		} else {
			// Body provided but no registry info - use quay.io
			mode = provider.Quay
		}
	} else {
		// No body provided - default to quay.io
		mode = provider.Quay
	}

	// Load krknctl config
	cfg, err := config.LoadConfig()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to load krknctl config: " + err.Error(),
		})
		return
	}

	// Create provider factory
	providerFactory := factory.NewProviderFactory(&cfg)

	// Get provider instance based on mode
	scenarioProvider := providerFactory.NewInstance(mode)
	if scenarioProvider == nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create scenario provider",
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

	// Check if scenario was found
	if scenarioDetail == nil {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Scenario '" + scenarioName + "' not found",
		})
		return
	}

	// Convert krknctl models.ScenarioDetail to ScenarioDetailResponse
	// This ensures Type fields are serialized as strings instead of int64
	var fields []InputFieldResponse
	for _, field := range scenarioDetail.Fields {
		fields = append(fields, InputFieldResponse{
			Name:              field.Name,
			ShortDescription:  field.ShortDescription,
			Description:       field.Description,
			Variable:          field.Variable,
			Type:              field.Type.String(), // Convert enum to string
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

	response := ScenarioDetailResponse{
		Name:         scenarioDetail.Name,
		Digest:       scenarioDetail.Digest,
		Size:         scenarioDetail.Size,
		LastModified: scenarioDetail.LastModified,
		Title:        scenarioDetail.Title,
		Description:  scenarioDetail.Description,
		Fields:       fields,
	}

	writeJSON(w, http.StatusOK, response)
}

// PostScenarioGlobals handles POST /scenarios/globals/{scenario_name} endpoint
// It returns global environment fields for a specific scenario
func (h *Handler) PostScenarioGlobals(w http.ResponseWriter, r *http.Request) {
	// Extract scenario_name from path: /scenarios/globals/{scenario_name}
	path := r.URL.Path
	prefix := "/scenarios/globals/"

	if len(path) <= len(prefix) {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "scenario_name parameter is required in path",
		})
		return
	}

	scenarioName := path[len(prefix):]
	if scenarioName == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "scenario_name parameter cannot be empty",
		})
		return
	}

	// Parse optional request body (same as /scenarios for registry config)
	var req ScenariosRequest
	var registry *models.RegistryV2
	var mode provider.Mode

	// Check if body is provided
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Invalid request body: " + err.Error(),
			})
			return
		}

		// If registry info is provided, validate and use private registry mode
		if req.RegistryURL != "" && req.ScenarioRepository != "" {
			registry = &models.RegistryV2{
				Username:           req.Username,
				Password:           req.Password,
				Token:              req.Token,
				RegistryURL:        req.RegistryURL,
				ScenarioRepository: req.ScenarioRepository,
				SkipTLS:            req.SkipTLS,
				Insecure:           req.Insecure,
			}
			mode = provider.Private
		} else if req.RegistryURL != "" || req.ScenarioRepository != "" {
			// Partial registry info provided - error
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Both registryUrl and scenarioRepository are required for private registry",
			})
			return
		} else {
			// Body provided but no registry info - use quay.io
			mode = provider.Quay
		}
	} else {
		// No body provided - default to quay.io
		mode = provider.Quay
	}

	// Load krknctl config
	cfg, err := config.LoadConfig()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to load krknctl config: " + err.Error(),
		})
		return
	}

	// Create provider factory
	providerFactory := factory.NewProviderFactory(&cfg)

	// Get provider instance based on mode
	scenarioProvider := providerFactory.NewInstance(mode)
	if scenarioProvider == nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create scenario provider",
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

	// Check if global environment was found
	if globalDetail == nil {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Global environment for scenario '" + scenarioName + "' not found",
		})
		return
	}

	// Convert krknctl models.ScenarioDetail to ScenarioDetailResponse
	// This ensures Type fields are serialized as strings instead of int64
	var fields []InputFieldResponse
	for _, field := range globalDetail.Fields {
		fields = append(fields, InputFieldResponse{
			Name:              field.Name,
			ShortDescription:  field.ShortDescription,
			Description:       field.Description,
			Variable:          field.Variable,
			Type:              field.Type.String(), // Convert enum to string
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

	response := ScenarioDetailResponse{
		Name:         globalDetail.Name,
		Digest:       globalDetail.Digest,
		Size:         globalDetail.Size,
		LastModified: globalDetail.LastModified,
		Title:        globalDetail.Title,
		Description:  globalDetail.Description,
		Fields:       fields,
	}

	writeJSON(w, http.StatusOK, response)
}

// PostScenarioRun handles POST /scenarios/run endpoint
// It creates and starts a new scenario job as a Kubernetes pod
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
	if req.TargetId == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "targetId is required",
		})
		return
	}
	if req.ClusterName == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "clusterName is required",
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

	// Set default kubeconfig path if not provided
	kubeconfigPath := req.KubeconfigPath
	if kubeconfigPath == "" {
		kubeconfigPath = "/home/krkn/.kube/config"
	}

	// Generate unique job ID
	jobId := uuid.New().String()

	ctx := context.Background()

	// Get kubeconfig from target using common helper function
	kubeconfigBase64, err := h.getKubeconfigFromTarget(ctx, req.TargetId, req.ClusterName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
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

	// Kubeconfig is base64 encoded, decode it for ConfigMap
	kubeconfigDecoded, err := base64.StdEncoding.DecodeString(kubeconfigBase64)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to decode kubeconfig: " + err.Error(),
		})
		return
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
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create kubeconfig ConfigMap: " + err.Error(),
		})
		return
	}

	// Create ConfigMaps for user-provided files
	var fileConfigMaps []string
	for _, file := range req.Files {
		// Sanitize filename for ConfigMap name
		sanitizedName := strings.ReplaceAll(file.Name, "/", "-")
		sanitizedName = strings.ReplaceAll(sanitizedName, ".", "-")
		configMapName := fmt.Sprintf("krkn-job-%s-file-%s", jobId, sanitizedName)

		// Decode base64 content
		fileContent, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			// Cleanup created ConfigMaps on error
			h.client.Delete(ctx, kubeconfigConfigMap)
			for _, cm := range fileConfigMaps {
				h.client.Delete(ctx, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cm,
						Namespace: h.namespace,
					},
				})
			}
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Failed to decode file content for '" + file.Name + "': " + err.Error(),
			})
			return
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
			// Cleanup on error
			h.client.Delete(ctx, kubeconfigConfigMap)
			for _, cm := range fileConfigMaps {
				h.client.Delete(ctx, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cm,
						Namespace: h.namespace,
					},
				})
			}
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to create file ConfigMap: " + err.Error(),
			})
			return
		}

		fileConfigMaps = append(fileConfigMaps, configMapName)
	}

	// Prepare pod specification
	podName := fmt.Sprintf("krkn-job-%s", jobId)

	// Build environment variables
	envVars := []corev1.EnvVar{}
	for key, value := range req.Environment {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
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

	// Handle private registry authentication
	var imagePullSecrets []corev1.LocalObjectReference
	if req.RegistryURL != "" && req.ScenarioRepository != "" {
		// Create ImagePullSecret
		imagePullSecretName := fmt.Sprintf("krkn-job-%s-registry", jobId)

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
			// Cleanup on error
			h.client.Delete(ctx, kubeconfigConfigMap)
			for _, cm := range fileConfigMaps {
				h.client.Delete(ctx, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cm,
						Namespace: h.namespace,
					},
				})
			}
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to create ImagePullSecret: " + err.Error(),
			})
			return
		}

		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{
			Name: imagePullSecretName,
		})
	}

	// Add writable tmp volume for scenarios that need to write temporary files
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

	// SecurityContext for running as krkn user (UID 1001)
	// Requires ServiceAccount with anyuid SCC permissions
	var runAsUser int64 = 1001
	var runAsGroup int64 = 1001
	var fsGroup int64 = 1001

	// Create the pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: h.namespace,
			Labels: map[string]string{
				"app":                "krkn-scenario",
				"krkn-job-id":        jobId,
				"krkn-scenario-name": req.ScenarioName,
				"krkn-target-id":     req.TargetId,
				"krkn-cluster-name":  req.ClusterName,
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
		// Cleanup on error
		h.client.Delete(ctx, kubeconfigConfigMap)
		for _, cm := range fileConfigMaps {
			h.client.Delete(ctx, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cm,
					Namespace: h.namespace,
				},
			})
		}
		if len(imagePullSecrets) > 0 {
			h.client.Delete(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      imagePullSecrets[0].Name,
					Namespace: h.namespace,
				},
			})
		}
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create pod: " + err.Error(),
		})
		return
	}

	// Return response
	response := ScenarioRunResponse{
		JobId:   jobId,
		Status:  "Pending",
		PodName: podName,
	}

	writeJSON(w, http.StatusCreated, response)
}

// GetScenarioRunStatus handles GET /scenarios/run/{jobId} endpoint
// It returns the current status of a specific job
func (h *Handler) GetScenarioRunStatus(w http.ResponseWriter, r *http.Request) {
	// Extract jobId from path
	path := r.URL.Path
	prefix := "/scenarios/run/"

	if len(path) <= len(prefix) {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "jobId parameter is required in path",
		})
		return
	}

	// Check if this is the logs endpoint
	if strings.HasSuffix(path, "/logs") {
		// This should be handled by GetScenarioRunLogs
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid endpoint",
		})
		return
	}

	jobId := path[len(prefix):]
	if jobId == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "jobId parameter cannot be empty",
		})
		return
	}

	ctx := context.Background()

	// Find pod by label
	var podList corev1.PodList
	err := h.client.List(ctx, &podList, client.InNamespace(h.namespace), client.MatchingLabels{
		"krkn-job-id": jobId,
	})

	if err != nil {
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

	// Extract info from labels
	targetId := pod.Labels["krkn-target-id"]
	clusterName := pod.Labels["krkn-cluster-name"]
	scenarioName := pod.Labels["krkn-scenario-name"]

	// Map pod phase to job status
	status := string(pod.Status.Phase)
	if status == "Succeeded" || status == "Failed" {
		// Keep as is
	} else if status == "Running" {
		status = "Running"
	} else if status == "Pending" {
		status = "Pending"
	} else {
		status = "Unknown"
	}

	// Build response
	response := JobStatusResponse{
		JobId:        jobId,
		TargetId:     targetId,
		ClusterName:  clusterName,
		ScenarioName: scenarioName,
		Status:       status,
		PodName:      pod.Name,
	}

	// Add timestamps if available
	if pod.Status.StartTime != nil {
		startTime := pod.Status.StartTime.Time
		response.StartTime = &startTime
	}

	// Check for completion time
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionFalse {
			if pod.Status.Phase == "Succeeded" || pod.Status.Phase == "Failed" {
				completionTime := condition.LastTransitionTime.Time
				response.CompletionTime = &completionTime
			}
		}
	}

	// Add message if pod failed
	if pod.Status.Phase == "Failed" {
		response.Message = pod.Status.Message
		if response.Message == "" && len(pod.Status.ContainerStatuses) > 0 {
			if pod.Status.ContainerStatuses[0].State.Terminated != nil {
				response.Message = pod.Status.ContainerStatuses[0].State.Terminated.Reason
			}
		}
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

// GetScenarioRunLogs handles GET /scenarios/run/{jobId}/logs endpoint
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

	// Extract jobId from path
	path := r.URL.Path
	prefix := "/scenarios/run/"
	suffix := "/logs"

	if !strings.HasSuffix(path, suffix) {
		logger.Error(nil, "Invalid logs endpoint path",
			"path", path,
			"expected_suffix", suffix)
		conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Invalid logs endpoint"))
		return
	}

	jobId := path[len(prefix) : len(path)-len(suffix)]
	if jobId == "" {
		logger.Error(nil, "Empty jobId in request path", "path", path)
		conn.WriteMessage(websocket.TextMessage, []byte("ERROR: jobId parameter cannot be empty"))
		return
	}

	logger.Info("WebSocket connection established", "jobId", jobId, "client_ip", r.RemoteAddr)

	ctx := context.Background()

	// Find pod by label
	var podList corev1.PodList
	err = h.client.List(ctx, &podList, client.InNamespace(h.namespace), client.MatchingLabels{
		"krkn-job-id": jobId,
	})

	if err != nil {
		logger.Error(err, "Failed to list pods",
			"jobId", jobId,
			"namespace", h.namespace)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: Failed to list pods: %s", err.Error())))
		return
	}

	if len(podList.Items) == 0 {
		logger.Error(nil, "Job not found",
			"jobId", jobId,
			"namespace", h.namespace)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: Job with ID '%s' not found", jobId)))
		return
	}

	pod := podList.Items[0]
	logger.Info("Found pod for job", "jobId", jobId, "podName", pod.Name, "podPhase", pod.Status.Phase)

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
		"jobId", jobId,
		"podName", pod.Name,
		"follow", follow,
		"timestamps", timestamps)

	// Get log stream from Kubernetes API
	req := h.clientset.CoreV1().Pods(h.namespace).GetLogs(pod.Name, logOptions)
	stream, err := req.Stream(ctx)
	if err != nil {
		logger.Error(err, "Failed to open log stream",
			"jobId", jobId,
			"podName", pod.Name,
			"namespace", h.namespace)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: Failed to open log stream: %s", err.Error())))
		return
	}
	defer stream.Close()

	logger.Info("Streaming logs started", "jobId", jobId, "podName", pod.Name)

	// Read logs line by line and send via WebSocket
	scanner := bufio.NewScanner(stream)
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		err := conn.WriteMessage(websocket.TextMessage, []byte(line))
		if err != nil {
			logger.Error(err, "Failed to write log line to WebSocket, client likely disconnected",
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
			"jobId", jobId,
			"podName", pod.Name,
			"linesStreamed", lineCount)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: Log stream error: %s", err.Error())))
		return
	}

	logger.Info("Log streaming completed",
		"jobId", jobId,
		"podName", pod.Name,
		"totalLines", lineCount)

	// Send close message
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

// ListScenarioRuns handles GET /scenarios/run endpoint
// It returns a list of all scenario jobs
func (h *Handler) ListScenarioRuns(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Parse query parameters for filtering
	statusFilter := r.URL.Query().Get("status")
	scenarioNameFilter := r.URL.Query().Get("scenarioName")
	targetIdFilter := r.URL.Query().Get("targetId")
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
		targetId := pod.Labels["krkn-target-id"]
		clusterName := pod.Labels["krkn-cluster-name"]
		scenarioName := pod.Labels["krkn-scenario-name"]

		// Apply filters
		if statusFilter != "" && string(pod.Status.Phase) != statusFilter {
			continue
		}
		if scenarioNameFilter != "" && scenarioName != scenarioNameFilter {
			continue
		}
		if targetIdFilter != "" && targetId != targetIdFilter {
			continue
		}
		if clusterNameFilter != "" && clusterName != clusterNameFilter {
			continue
		}

		// Map pod phase to job status
		status := string(pod.Status.Phase)

		jobStatus := JobStatusResponse{
			JobId:        jobId,
			TargetId:     targetId,
			ClusterName:  clusterName,
			ScenarioName: scenarioName,
			Status:       status,
			PodName:      pod.Name,
		}

		// Add timestamps
		if pod.Status.StartTime != nil {
			startTime := pod.Status.StartTime.Time
			jobStatus.StartTime = &startTime
		}

		// Check for completion time
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionFalse {
				if pod.Status.Phase == "Succeeded" || pod.Status.Phase == "Failed" {
					completionTime := condition.LastTransitionTime.Time
					jobStatus.CompletionTime = &completionTime
				}
			}
		}

		jobs = append(jobs, jobStatus)
	}

	response := JobsListResponse{
		Jobs: jobs,
	}

	writeJSON(w, http.StatusOK, response)
}

// DeleteScenarioRun handles DELETE /scenarios/run/{jobId} endpoint
// It stops and deletes a running job
func (h *Handler) DeleteScenarioRun(w http.ResponseWriter, r *http.Request) {
	// Extract jobId from path
	path := r.URL.Path
	prefix := "/scenarios/run/"

	if len(path) <= len(prefix) {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "jobId parameter is required in path",
		})
		return
	}

	jobId := path[len(prefix):]
	if jobId == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "jobId parameter cannot be empty",
		})
		return
	}

	ctx := context.Background()

	// Find pod by label
	var podList corev1.PodList
	err := h.client.List(ctx, &podList, client.InNamespace(h.namespace), client.MatchingLabels{
		"krkn-job-id": jobId,
	})

	if err != nil {
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

	// Delete pod with 5 second grace period
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

	// Delete associated ConfigMaps
	var configMapList corev1.ConfigMapList
	err = h.client.List(ctx, &configMapList, client.InNamespace(h.namespace), client.MatchingLabels{
		"krkn-job-id": jobId,
	})

	if err == nil {
		for _, cm := range configMapList.Items {
			h.client.Delete(ctx, &cm)
		}
	}

	// Delete associated Secrets (ImagePullSecrets)
	var secretList corev1.SecretList
	err = h.client.List(ctx, &secretList, client.InNamespace(h.namespace), client.MatchingLabels{
		"krkn-job-id": jobId,
	})

	if err == nil {
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

// ScenariosRunRouter routes requests to /scenarios/run endpoints
func (h *Handler) ScenariosRunRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// POST /scenarios/run - create new job
	if path == "/scenarios/run" && r.Method == http.MethodPost {
		h.PostScenarioRun(w, r)
		return
	}

	// GET /scenarios/run - list all jobs
	if path == "/scenarios/run" && r.Method == http.MethodGet {
		h.ListScenarioRuns(w, r)
		return
	}

	// Path with jobId: /scenarios/run/{jobId}...
	if strings.HasPrefix(path, "/scenarios/run/") {
		// GET /scenarios/run/{jobId}/logs - stream logs
		if strings.HasSuffix(path, "/logs") && r.Method == http.MethodGet {
			h.GetScenarioRunLogs(w, r)
			return
		}

		// GET /scenarios/run/{jobId} - get job status
		if r.Method == http.MethodGet {
			h.GetScenarioRunStatus(w, r)
			return
		}

		// DELETE /scenarios/run/{jobId} - delete job
		if r.Method == http.MethodDelete {
			h.DeleteScenarioRun(w, r)
			return
		}
	}

	// Method not allowed
	writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
		Error:   "method_not_allowed",
		Message: "Method " + r.Method + " not allowed for path " + path,
	})
}
