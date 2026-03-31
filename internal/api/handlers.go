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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
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

// getTokenGenerator creates a TokenGenerator for JWT validation (used for WebSocket auth)
// It uses the same JWT secret as the HTTP middleware
func (h *Handler) getTokenGenerator(ctx context.Context) (*auth.TokenGenerator, error) {
	jwtSecret, err := h.getOrCreateJWTSecret(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWT secret: %w", err)
	}
	return auth.NewTokenGenerator(jwtSecret, TokenDuration, "krkn-operator"), nil
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

	// Filter clusters based on user permissions
	// Admins see all clusters, regular users see only clusters they have 'view' permission for
	ctx := r.Context()
	targetData := targetRequest.Status.TargetData

	claims := auth.GetClaimsFromContext(ctx)
	if claims != nil && !auth.IsAdmin(ctx) {
		// Regular user: filter by group permissions
		filteredData, err := groupauth.FilterClustersByPermission(
			ctx,
			h.client,
			claims.UserID,
			h.namespace,
			targetData,
			groupauth.ActionView,
		)
		if err != nil {
			log.FromContext(ctx).Error(err, "Failed to filter clusters by permission", "userID", claims.UserID)
			// Continue with empty result rather than failing
			filteredData = map[string][]krknv1alpha1.ClusterTarget{}
		}
		targetData = filteredData
	}

	// Return the target data (filtered for regular users, unfiltered for admins)
	response := ClustersResponse{
		TargetData: targetData,
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
	uuid, err := extractPathSuffix(r.URL.Path, TargetsPath+"/")
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
		w.WriteHeader(http.StatusAccepted)
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
	writeJSON(w, http.StatusAccepted, response)
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
	_ = json.NewEncoder(w).Encode(data) // If encoding fails, client gets partial response
}

// writeJSONError writes a JSON error response with the given status code
func writeJSONError(w http.ResponseWriter, status int, err ErrorResponse) {
	// Log internal server errors for debugging
	if status >= 500 {
		logger := log.Log.WithName("api")
		logger.Error(fmt.Errorf("%s", err.Message), "Internal server error", "error_code", err.Error, "status", status)
	}
	writeJSON(w, status, err)
}

// callGetNodesGRPC calls the data provider gRPC service to get nodes
func (h *Handler) callGetNodesGRPC(kubeconfigBase64 string) ([]string, error) {
	// Create gRPC connection
	conn, err := grpc.NewClient(
		h.grpcServerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Create context with timeout for RPC call
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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
	scenarioName, err := extractPathSuffix(r.URL.Path, ScenariosDetailPath+"/")
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
	scenarioName, err := extractPathSuffix(r.URL.Path, ScenariosGlobalsPath+"/")
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


func (h *Handler) PostScenarioRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

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
	if req.TargetRequestID == "" {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "targetRequestId is required",
		})
		return
	}

	if len(req.TargetClusters) == 0 {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "targetClusters is required and must contain at least one provider with clusters",
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

	// Validate cluster names across all providers (no duplicates or empty strings)
	seen := make(map[string]string) // map[clusterName]providerName
	for providerName, clusterNames := range req.TargetClusters {
		if providerName == "" {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "provider names cannot be empty",
			})
			return
		}
		if len(clusterNames) == 0 {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "provider '" + providerName + "' must have at least one cluster",
			})
			return
		}
		for _, clusterName := range clusterNames {
			if clusterName == "" {
				writeJSONError(w, http.StatusBadRequest, ErrorResponse{
					Error:   "bad_request",
					Message: "cluster names cannot be empty",
				})
				return
			}
			if existingProvider, exists := seen[clusterName]; exists {
				writeJSONError(w, http.StatusBadRequest, ErrorResponse{
					Error:   "bad_request",
					Message: "cluster '" + clusterName + "' appears in multiple providers: '" + existingProvider + "' and '" + providerName + "'",
				})
				return
			}
			seen[clusterName] = providerName
		}
	}

	// Validate user permissions (group-based access control)
	// Admins bypass validation, regular users must have 'run' permission on all target clusters
	userClaims := auth.GetClaimsFromContext(ctx)
	if userClaims != nil && !auth.IsAdmin(ctx) {
		// Fetch KrknTargetRequest to get cluster API URLs
		targetRequest := &krknv1alpha1.KrknTargetRequest{}
		if err := h.client.Get(ctx, types.NamespacedName{
			Name:      req.TargetRequestID,
			Namespace: h.namespace,
		}, targetRequest); err != nil {
			logger.Error(err, "Failed to fetch target request for permission validation", "targetRequestId", req.TargetRequestID)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to validate cluster permissions: target request not found",
			})
			return
		}

		// Check if target request is completed
		if targetRequest.Status.Status != "Completed" {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Target request is not completed yet",
			})
			return
		}

		// Validate user has 'run' permission on all target clusters
		if err := groupauth.ValidateScenarioRunAccess(
			ctx,
			h.client,
			userClaims.UserID,
			h.namespace,
			req.TargetClusters,
			targetRequest,
		); err != nil {
			logger.Info("User lacks permission to run scenarios on requested clusters",
				"userID", userClaims.UserID,
				"error", err.Error(),
			)
			writeJSONError(w, http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: err.Error(),
			})
			return
		}

		logger.V(1).Info("User permission validated for scenario run",
			"userID", userClaims.UserID,
			"clusterCount", len(seen),
		)
	}

	// Generate scenario run name
	scenarioRunName := fmt.Sprintf("%s-%s", req.ScenarioName, uuid.New().String()[:8])

	// Create KrknScenarioRun CR
	// Extract user claims for ownership tracking (defensive check for tests)
	claims := auth.GetClaimsFromContext(ctx)

	labels := make(map[string]string)
	ownerUserID := ""
	if claims != nil {
		labels["krkn.krkn-chaos.dev/owner-user"] = sanitizeUserID(claims.UserID)
		ownerUserID = claims.UserID
	}

	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scenarioRunName,
			Namespace: h.namespace,
			Labels:    labels,
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID:    req.TargetRequestID,
			OwnerUserID:        ownerUserID,
			TargetClusters:     req.TargetClusters,
			ScenarioName:       req.ScenarioName,
			ScenarioImage:      req.ScenarioImage,
			KubeconfigPath:     req.KubeconfigPath,
			Environment:        req.Environment,
			RegistryURL:        req.RegistryURL,
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

	// Set owner reference: ScenarioRun owns KrknTargetRequest
	// This ensures KrknTargetRequest (and its Secret) are cleaned up when ScenarioRun is deleted
	// and remain available for job retries while ScenarioRun exists
	targetRequest := &krknv1alpha1.KrknTargetRequest{}
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      req.TargetRequestID,
		Namespace: h.namespace,
	}, targetRequest); err == nil {
		// Set ScenarioRun as owner of KrknTargetRequest
		if err := ctrl.SetControllerReference(scenarioRun, targetRequest, h.client.Scheme()); err != nil {
			logger.Error(err, "failed to set owner reference on KrknTargetRequest",
				"scenarioRun", scenarioRun.Name,
				"targetRequestId", req.TargetRequestID)
			// Continue - not critical for scenario run creation
		} else {
			if err := h.client.Update(ctx, targetRequest); err != nil {
				logger.Error(err, "failed to update KrknTargetRequest with owner reference",
					"targetRequestId", req.TargetRequestID)
				// Continue - not critical
			} else {
				logger.Info("set owner reference on KrknTargetRequest",
					"scenarioRun", scenarioRun.Name,
					"targetRequestId", req.TargetRequestID)
			}
		}
	} else {
		logger.Error(err, "failed to get KrknTargetRequest for setting owner reference",
			"targetRequestId", req.TargetRequestID)
		// Continue - KrknTargetRequest might have been deleted manually
	}

	// Calculate total targets from all providers
	totalTargets := 0
	for _, clusters := range req.TargetClusters {
		totalTargets += len(clusters)
	}

	response := ScenarioRunCreateResponse{
		ScenarioRunName: scenarioRunName,
		TargetClusters:  req.TargetClusters,
		TotalTargets:    totalTargets,
		OwnerUserID:     ownerUserID,
	}

	writeJSON(w, http.StatusCreated, response)
}

// GetScenarioRunStatus handles GET /api/v1/scenarios/run/{scenarioRunName} endpoint
// It returns the current status of a scenario run
func (h *Handler) GetScenarioRunStatus(w http.ResponseWriter, r *http.Request) {
	scenarioRunName, err := extractPathSuffix(r.URL.Path, ScenariosRunPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "scenarioRunName " + err.Error(),
		})
		return
	}

	ctx := r.Context()

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

	// Check access permissions
	if !checkScenarioRunAccess(w, r, &scenarioRun) {
		return
	}

	// Convert ClusterJobStatus to response type
	clusterJobs := make([]ClusterJobStatusResponse, len(scenarioRun.Status.ClusterJobs))
	for i, job := range scenarioRun.Status.ClusterJobs {
		clusterJobs[i] = ClusterJobStatusResponse{
			ProviderName:    job.ProviderName,
			ClusterName:     job.ClusterName,
			JobID:           job.JobID,
			PodName:         job.PodName,
			Phase:           job.Phase,
			Message:         job.Message,
			StartTime:       convertMetaTime(job.StartTime),
			CompletionTime:  convertMetaTime(job.CompletionTime),
			RetryCount:      job.RetryCount,
			MaxRetries:      job.MaxRetries,
			CancelRequested: job.CancelRequested,
			FailureReason:   job.FailureReason,
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
		OwnerUserID:     scenarioRun.Spec.OwnerUserID,
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
	// Support "access_token" subprotocol for JWT authentication
	Subprotocols: []string{"access_token"},
}

// isWebSocketDisconnectError checks if an error is a normal WebSocket client disconnection
func isWebSocketDisconnectError(err error) bool {
	if err == nil {
		return false
	}

	// Check for WebSocket close errors
	if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
		return true
	}

	// Check error message for common disconnection patterns
	errMsg := err.Error()
	disconnectPatterns := []string{
		"broken pipe",
		"connection reset by peer",
		"use of closed network connection",
		"i/o timeout",
		"EOF",
		"client disconnected",
	}

	for _, pattern := range disconnectPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// GetScenarioRunLogs handles GET /api/v1/scenarios/run/{scenarioRunName}/jobs/{jobID}/logs endpoint
// It streams the stdout/stderr logs of a running or completed job via WebSocket
func (h *Handler) GetScenarioRunLogs(w http.ResponseWriter, r *http.Request) {
	logger := log.Log.WithName("websocket-logs")

	logger.Info("🔌 WebSocket connection request received",
		"path", r.URL.Path,
		"client_ip", r.RemoteAddr,
		"user_agent", r.Header.Get("User-Agent"))

	// Extract JWT token from WebSocket subprotocol header BEFORE upgrade
	// Frontend sends: new WebSocket(url, `access_token.${jwt_token}`)
	// Format: "access_token.<jwt_token>"
	protocols := r.Header.Get("Sec-WebSocket-Protocol")
	logger.V(1).Info("📋 Received WebSocket headers",
		"Sec-WebSocket-Protocol", protocols,
		"Sec-WebSocket-Version", r.Header.Get("Sec-WebSocket-Version"),
		"Sec-WebSocket-Key", r.Header.Get("Sec-WebSocket-Key"))

	if protocols == "" {
		logger.Info("❌ WebSocket authentication failed: missing Sec-WebSocket-Protocol header",
			"path", r.URL.Path,
			"client_ip", r.RemoteAddr,
			"headers", r.Header)
		http.Error(w, "Unauthorized: Missing Sec-WebSocket-Protocol header", http.StatusUnauthorized)
		return
	}

	// Parse protocol: split on first '.' to separate prefix from token
	// Example: "access_token.eyJhbGc..." → ["access_token", "eyJhbGc..."]
	logger.V(1).Info("🔍 Parsing Sec-WebSocket-Protocol",
		"raw_protocol", protocols,
		"protocol_length", len(protocols))

	protocolParts := strings.SplitN(protocols, ".", 2)
	logger.V(1).Info("🔍 Protocol parts after split",
		"parts_count", len(protocolParts),
		"part_0", func() string {
			if len(protocolParts) > 0 {
				return protocolParts[0]
			}
			return "<none>"
		}(),
		"part_1_length", func() int {
			if len(protocolParts) > 1 {
				return len(protocolParts[1])
			}
			return 0
		}())

	if len(protocolParts) != 2 || protocolParts[0] != "access_token" {
		logger.Info("❌ WebSocket authentication failed: invalid protocol format",
			"path", r.URL.Path,
			"protocol", protocols,
			"parts_count", len(protocolParts),
			"expected_format", "access_token.<jwt>",
			"client_ip", r.RemoteAddr)
		http.Error(w, "Unauthorized: Invalid Sec-WebSocket-Protocol format. Expected: access_token.<jwt>", http.StatusUnauthorized)
		return
	}

	token := protocolParts[1]
	if token == "" {
		logger.Info("❌ WebSocket authentication failed: empty token in subprotocol",
			"path", r.URL.Path,
			"client_ip", r.RemoteAddr)
		http.Error(w, "Unauthorized: Missing authentication token", http.StatusUnauthorized)
		return
	}

	// Mask token for logging (show first/last 10 chars)
	maskedToken := func() string {
		if len(token) <= 20 {
			return "***"
		}
		return token[:10] + "..." + token[len(token)-10:]
	}()

	logger.Info("🔑 JWT token extracted from subprotocol",
		"token_length", len(token),
		"token_preview", maskedToken)

	// Get TokenGenerator and validate token
	logger.V(1).Info("🔐 Getting TokenGenerator for validation")
	tokenGen, err := h.getTokenGenerator(r.Context())
	if err != nil {
		logger.Error(err, "❌ Failed to get TokenGenerator for WebSocket auth")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	logger.Info("🔐 Validating JWT token")
	claims, err := tokenGen.ValidateToken(token)
	if err != nil {
		logger.Info("❌ WebSocket authentication failed: invalid token",
			"path", r.URL.Path,
			"error", err.Error(),
			"token_preview", maskedToken,
			"client_ip", r.RemoteAddr)
		http.Error(w, "Unauthorized: Invalid or expired token", http.StatusUnauthorized)
		return
	}

	logger.Info("✅ WebSocket authentication successful",
		"userId", claims.UserID,
		"role", claims.Role,
		"path", r.URL.Path,
		"client_ip", r.RemoteAddr)

	// Upgrade to WebSocket with the FULL subprotocol in response
	// WebSocket spec requires server to respond with one of the client's requested subprotocols
	// Client sent: "access_token.<jwt_token>"
	// Server must respond with the SAME value (not just "access_token")
	logger.Info("⬆️ Upgrading connection to WebSocket",
		"response_protocol", protocols)

	conn, err := upgrader.Upgrade(w, r, http.Header{
		"Sec-WebSocket-Protocol": []string{protocols}, // Echo back the full protocol
	})
	if err != nil {
		logger.Error(err, "❌ WebSocket upgrade failed",
			"url", r.URL.String(),
			"headers", r.Header,
			"client_ip", r.RemoteAddr)
		return
	}
	defer conn.Close()

	logger.Info("✅ WebSocket connection established",
		"userId", claims.UserID,
		"client_ip", r.RemoteAddr)

	// Extract scenarioRunName and jobID from path
	// Path format: /api/v1/scenarios/run/{scenarioRunName}/jobs/{jobID}/logs
	path := r.URL.Path
	prefix := ScenariosRunPath + "/"

	if !strings.HasPrefix(path, prefix) {
		logger.Error(nil, "Invalid logs endpoint path", "path", path)
		_ = conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Invalid logs endpoint")) // Best-effort error reporting
		return
	}

	// Remove prefix
	remainder := path[len(prefix):]

	// Split by "/jobs/" and "/logs"
	parts := strings.Split(remainder, "/jobs/")
	if len(parts) != 2 {
		logger.Error(nil, "Invalid logs endpoint path format", "path", path)
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: Invalid path format. Expected: %s/{scenarioRunName}/jobs/{jobID}/logs", ScenariosRunPath))) // Best-effort error reporting
		return
	}

	scenarioRunName := parts[0]
	jobIDAndLogs := parts[1]

	// Extract jobID (remove "/logs" suffix)
	if !strings.HasSuffix(jobIDAndLogs, "/logs") {
		logger.Error(nil, "Invalid logs endpoint path format", "path", path)
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: Invalid path format. Expected: %s/{scenarioRunName}/jobs/{jobID}/logs", ScenariosRunPath))) // Best-effort error reporting
		return
	}

	jobID := strings.TrimSuffix(jobIDAndLogs, "/logs")

	if scenarioRunName == "" || jobID == "" {
		logger.Error(nil, "Empty scenarioRunName or jobID in request path", "path", path)
		_ = conn.WriteMessage(websocket.TextMessage, []byte("ERROR: scenarioRunName and jobID cannot be empty")) // Best-effort error reporting
		return
	}

	logger.Info("WebSocket connection established", "scenarioRunName", scenarioRunName, "jobID", jobID, "client_ip", r.RemoteAddr)

	// Set up ping/pong handlers to detect client disconnection
	pongWait := 60 * time.Second
	_ = conn.SetReadDeadline(time.Now().Add(pongWait)) // Best-effort timeout
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait)) // Best-effort timeout
		return nil
	})

	// Start ping ticker
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Channel to signal when to stop pinging
	done := make(chan struct{})
	defer close(done)

	// Goroutine to send periodic pings
	go func() {
		for {
			select {
			case <-pingTicker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					logger.V(1).Info("Failed to send ping, client disconnected",
						"scenarioRunName", scenarioRunName,
						"jobID", jobID)
					return
				}
			case <-done:
				return
			}
		}
	}()

	ctx := context.Background()

	// Find pod by jobID label (no need to fetch the CR)
	var podList corev1.PodList
	if err := h.client.List(ctx, &podList, client.InNamespace(h.namespace), client.MatchingLabels{
		"krkn-job-id": jobID,
	}); err != nil {
		logger.Error(err, "Failed to list pods", "jobID", jobID)
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: Failed to list pods: %s", err.Error()))) // Best-effort error reporting
		return
	}

	if len(podList.Items) == 0 {
		logger.Error(nil, "Job not found", "jobID", jobID)
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: Job with ID '%s' not found", jobID))) // Best-effort error reporting
		return
	}

	pod := podList.Items[0]
	logger.Info("Found pod for job", "scenarioRunName", scenarioRunName, "jobID", jobID, "podName", pod.Name, "podPhase", pod.Status.Phase)

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
		"jobID", jobID,
		"podName", pod.Name,
		"follow", follow,
		"timestamps", timestamps)

	// Get log stream from Kubernetes API
	req := h.clientset.CoreV1().Pods(h.namespace).GetLogs(pod.Name, logOptions)
	stream, err := req.Stream(ctx)
	if err != nil {
		logger.Error(err, "Failed to open log stream",
			"scenarioRunName", scenarioRunName,
			"jobID", jobID,
			"podName", pod.Name,
			"namespace", h.namespace)
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: Failed to open log stream: %s", err.Error()))) // Best-effort error reporting
		return
	}
	defer stream.Close()

	logger.Info("Streaming logs started", "scenarioRunName", scenarioRunName, "jobID", jobID, "podName", pod.Name)

	// Read logs line by line and send via WebSocket
	scanner := bufio.NewScanner(stream)
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		err := conn.WriteMessage(websocket.TextMessage, []byte(line))
		if err != nil {
			// Check if this is a normal client disconnection
			if isWebSocketDisconnectError(err) {
				logger.Info("WebSocket client disconnected",
					"scenarioRunName", scenarioRunName,
					"jobID", jobID,
					"podName", pod.Name,
					"linesStreamed", lineCount)
			} else {
				logger.Error(err, "Unexpected WebSocket write error",
					"scenarioRunName", scenarioRunName,
					"jobID", jobID,
					"podName", pod.Name,
					"linesStreamed", lineCount)
			}
			return
		}
		lineCount++
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		logger.Error(err, "Log stream scanner error",
			"scenarioRunName", scenarioRunName,
			"jobID", jobID,
			"podName", pod.Name,
			"linesStreamed", lineCount)
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: Log stream error: %s", err.Error()))) // Best-effort error reporting
		return
	}

	logger.Info("Log streaming completed",
		"scenarioRunName", scenarioRunName,
		"jobID", jobID,
		"podName", pod.Name,
		"totalLines", lineCount)

	// Send close message (ignore error if client already disconnected)
	if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		if !isWebSocketDisconnectError(err) {
			logger.V(1).Info("Failed to send close message, client may have already disconnected",
				"scenarioRunName", scenarioRunName,
				"jobID", jobID,
				"error", err.Error())
		}
	}
}

// ListScenarioRuns handles GET /api/v1/scenarios/run endpoint
// It returns a list of all scenario runs (KrknScenarioRun CRs)
func (h *Handler) ListScenarioRuns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters for filtering
	phaseFilter := r.URL.Query().Get("phase") // e.g., Running, Succeeded, Failed
	scenarioNameFilter := r.URL.Query().Get("scenarioName")

	// List all KrknScenarioRun CRs in the namespace
	var scenarioRunList krknv1alpha1.KrknScenarioRunList
	if err := h.client.List(ctx, &scenarioRunList, client.InNamespace(h.namespace)); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list scenario runs: " + err.Error(),
		})
		return
	}

	// Filter by ownership (admins see all, users see only their own)
	scenarioRunList.Items = filterScenarioRunsByOwnership(scenarioRunList.Items, ctx)

	// Convert to response format with optional filtering
	runs := make([]ScenarioRunListItem, 0)
	for _, sr := range scenarioRunList.Items {
		// Apply filters
		if phaseFilter != "" && sr.Status.Phase != phaseFilter {
			continue
		}
		if scenarioNameFilter != "" && sr.Spec.ScenarioName != scenarioNameFilter {
			continue
		}

		run := ScenarioRunListItem{
			ScenarioRunName: sr.Name,
			ScenarioName:    sr.Spec.ScenarioName,
			Phase:           sr.Status.Phase,
			TotalTargets:    sr.Status.TotalTargets,
			SuccessfulJobs:  sr.Status.SuccessfulJobs,
			FailedJobs:      sr.Status.FailedJobs,
			RunningJobs:     sr.Status.RunningJobs,
			CreatedAt:       sr.CreationTimestamp.Time,
			OwnerUserID:     sr.Spec.OwnerUserID,
		}

		runs = append(runs, run)
	}

	response := ScenarioRunListResponse{
		ScenarioRuns: runs,
	}

	writeJSON(w, http.StatusOK, response)
}

// GetActiveRunsOverview handles GET /api/v1/dashboard/active-runs endpoint
// It returns an overview of currently running scenario runs
// Accessible to all authenticated users - all users see all active runs (global dashboard)
func (h *Handler) GetActiveRunsOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// List all KrknScenarioRun CRs in the namespace
	var scenarioRunList krknv1alpha1.KrknScenarioRunList
	if err := h.client.List(ctx, &scenarioRunList, client.InNamespace(h.namespace)); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list scenario runs: " + err.Error(),
		})
		return
	}

	// NOTE: No ownership filtering - this is a global dashboard showing all active runs to all users

	// Track cluster to runs mapping and active runs count
	clusterRuns := make(map[string][]string)
	activeRunsCount := 0

	// Process each scenario run
	for _, sr := range scenarioRunList.Items {
		hasRunningJobs := false

		// Check each cluster job in this scenario run
		for _, job := range sr.Status.ClusterJobs {
			// Only count jobs that are currently running
			if job.Phase == "Running" {
				hasRunningJobs = true

				// Add this scenario run to the cluster's list
				clusterRuns[job.ClusterName] = append(clusterRuns[job.ClusterName], sr.Name)
			}
		}

		// Count this scenario run as active if it has any running jobs
		if hasRunningJobs {
			activeRunsCount++
		}
	}

	response := ActiveRunsOverviewResponse{
		TotalActiveRuns: activeRunsCount,
		TotalClusters:   len(clusterRuns),
		ClusterRuns:     clusterRuns,
	}

	writeJSON(w, http.StatusOK, response)
}

// DeleteScenarioRun handles DELETE /api/v1/scenarios/run/{jobID} endpoint
// It stops and deletes a running job
func (h *Handler) DeleteScenarioRun(w http.ResponseWriter, r *http.Request) {
	jobID, err := extractPathSuffix(r.URL.Path, ScenariosRunPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "jobID " + err.Error(),
		})
		return
	}

	ctx := r.Context()

	var podList corev1.PodList
	if err := h.client.List(ctx, &podList, client.InNamespace(h.namespace), client.MatchingLabels{
		"krkn-job-id": jobID,
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
			Message: "Job with ID '" + jobID + "' not found",
		})
		return
	}

	pod := podList.Items[0]

	// Find parent ScenarioRun and check access
	scenarioRunName := pod.Labels["krkn-scenario-run"]
	if scenarioRunName != "" {
		var scenarioRun krknv1alpha1.KrknScenarioRun
		if err := h.client.Get(ctx, client.ObjectKey{
			Name:      scenarioRunName,
			Namespace: h.namespace,
		}, &scenarioRun); err == nil {
			// Check access permissions on parent ScenarioRun
			if !checkScenarioRunAccess(w, r, &scenarioRun) {
				return
			}
		}
		// If ScenarioRun not found, continue anyway (might have been deleted)
	}

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
		"krkn-job-id": jobID,
	}); err == nil {
		for _, cm := range configMapList.Items {
			_ = h.client.Delete(ctx, &cm) // Best-effort cleanup
		}
	}

	var secretList corev1.SecretList
	if err := h.client.List(ctx, &secretList, client.InNamespace(h.namespace), client.MatchingLabels{
		"krkn-job-id": jobID,
	}); err == nil {
		for _, secret := range secretList.Items {
			_ = h.client.Delete(ctx, &secret) // Best-effort cleanup
		}
	}

	response := JobStatusResponse{
		JobID:   jobID,
		Status:  "Stopped",
		Message: "Job stopped and deleted successfully",
	}

	writeJSON(w, http.StatusOK, response)
}

// DeleteScenarioRunComplete handles DELETE /api/v1/scenarios/run/{scenarioRunName}
// It deletes the entire KrknScenarioRun CR (all jobs)
func (h *Handler) DeleteScenarioRunComplete(w http.ResponseWriter, r *http.Request) {
	scenarioRunName, err := extractPathSuffix(r.URL.Path, ScenariosRunPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "scenarioRunName " + err.Error(),
		})
		return
	}

	ctx := r.Context()

	// Fetch the KrknScenarioRun CR
	var scenarioRun krknv1alpha1.KrknScenarioRun
	if err := h.client.Get(ctx, client.ObjectKey{
		Name:      scenarioRunName,
		Namespace: h.namespace,
	}, &scenarioRun); err != nil {
		if client.IgnoreNotFound(err) == nil {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Scenario run '" + scenarioRunName + "' not found",
			})
		} else {
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get scenario run: " + err.Error(),
			})
		}
		return
	}

	// Check access permissions
	if !checkScenarioRunAccess(w, r, &scenarioRun) {
		return
	}

	log.Log.Info("deleting entire scenario run",
		"scenarioRunName", scenarioRunName,
		"totalJobs", len(scenarioRun.Status.ClusterJobs),
		"phase", scenarioRun.Status.Phase)

	// Delete the CR - owner references will cascade delete all pods/configmaps/secrets
	if err := h.client.Delete(ctx, &scenarioRun); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete scenario run: " + err.Error(),
		})
		return
	}

	log.Log.Info("scenario run deleted successfully",
		"scenarioRunName", scenarioRunName)

	w.WriteHeader(http.StatusNoContent)
}

// DeleteSingleJob handles DELETE /api/v1/scenarios/run/jobs/{jobID}
// It cancels a single job by setting CancelRequested flag and deleting the pod
func (h *Handler) DeleteSingleJob(w http.ResponseWriter, r *http.Request) {
	// Parse path: /api/v1/scenarios/run/jobs/{jobID}
	jobID, err := extractPathSuffix(r.URL.Path, ScenariosRunJobsPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "jobID " + err.Error(),
		})
		return
	}

	ctx := context.Background()

	// Find KrknScenarioRun containing this jobID
	var scenarioRunList krknv1alpha1.KrknScenarioRunList
	if err := h.client.List(ctx, &scenarioRunList, client.InNamespace(h.namespace)); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list scenario runs: " + err.Error(),
		})
		return
	}

	// Search for job across all scenario runs
	var foundScenarioRun *krknv1alpha1.KrknScenarioRun
	var foundJobIndex int = -1

	for i := range scenarioRunList.Items {
		sr := &scenarioRunList.Items[i]
		for j, job := range sr.Status.ClusterJobs {
			if job.JobID == jobID {
				foundScenarioRun = sr
				foundJobIndex = j
				break
			}
		}
		if foundScenarioRun != nil {
			break
		}
	}

	if foundScenarioRun == nil {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Job '" + jobID + "' not found",
		})
		return
	}

	job := &foundScenarioRun.Status.ClusterJobs[foundJobIndex]

	log.Log.Info("cancelling single job",
		"scenarioRunName", foundScenarioRun.Name,
		"jobID", jobID,
		"clusterName", job.ClusterName,
		"currentPhase", job.Phase)

	// Set CancelRequested flag
	job.CancelRequested = true

	// Update CR status
	if err := h.client.Status().Update(ctx, foundScenarioRun); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update scenario run status: " + err.Error(),
		})
		return
	}

	log.Log.Info("set CancelRequested flag",
		"scenarioRunName", foundScenarioRun.Name,
		"jobID", jobID)

	// Delete the pod (controller will see CancelRequested and not retry)
	var podList corev1.PodList
	if err := h.client.List(ctx, &podList, client.InNamespace(h.namespace), client.MatchingLabels{
		"krkn-job-id": jobID,
	}); err == nil && len(podList.Items) > 0 {
		pod := podList.Items[0]
		gracePeriod := int64(5)
		deleteOptions := client.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		}

		if err := h.client.Delete(ctx, &pod, &deleteOptions); err != nil {
			log.Log.Error(err, "failed to delete pod during job cancellation",
				"scenarioRunName", foundScenarioRun.Name,
				"jobID", jobID,
				"podName", pod.Name)
		} else {
			log.Log.Info("deleted pod for cancelled job",
				"scenarioRunName", foundScenarioRun.Name,
				"jobID", jobID,
				"podName", pod.Name)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetSingleJob handles GET /api/v1/scenarios/run/jobs/{jobID}
// It returns the status of a single job by jobID (jobID is unique across all scenario runs)
func (h *Handler) GetSingleJob(w http.ResponseWriter, r *http.Request) {
	// Parse path: /api/v1/scenarios/run/jobs/{jobID}
	jobID, err := extractPathSuffix(r.URL.Path, ScenariosRunJobsPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "jobID " + err.Error(),
		})
		return
	}

	ctx := r.Context()

	// Find KrknScenarioRun containing this jobID
	var scenarioRunList krknv1alpha1.KrknScenarioRunList
	if err := h.client.List(ctx, &scenarioRunList, client.InNamespace(h.namespace)); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list scenario runs: " + err.Error(),
		})
		return
	}

	// Search for job across all scenario runs
	var foundJob *krknv1alpha1.ClusterJobStatus
	var foundScenarioRun *krknv1alpha1.KrknScenarioRun

	for i := range scenarioRunList.Items {
		sr := &scenarioRunList.Items[i]
		for j := range sr.Status.ClusterJobs {
			if sr.Status.ClusterJobs[j].JobID == jobID {
				foundJob = &sr.Status.ClusterJobs[j]
				foundScenarioRun = sr
				break
			}
		}
		if foundJob != nil {
			break
		}
	}

	if foundJob == nil {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Job '" + jobID + "' not found",
		})
		return
	}

	// Check access permissions on parent ScenarioRun
	if !checkScenarioRunAccess(w, r, foundScenarioRun) {
		return
	}

	// Convert to response type
	response := ClusterJobStatusResponse{
		ProviderName:    foundJob.ProviderName,
		ClusterName:     foundJob.ClusterName,
		JobID:           foundJob.JobID,
		PodName:         foundJob.PodName,
		Phase:           foundJob.Phase,
		Message:         foundJob.Message,
		StartTime:       convertMetaTime(foundJob.StartTime),
		CompletionTime:  convertMetaTime(foundJob.CompletionTime),
		RetryCount:      foundJob.RetryCount,
		MaxRetries:      foundJob.MaxRetries,
		CancelRequested: foundJob.CancelRequested,
		FailureReason:   foundJob.FailureReason,
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) ScenariosRunRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Root endpoint: /api/v1/scenarios/run
	if path == ScenariosRunPath {
		switch r.Method {
		case http.MethodPost:
			h.PostScenarioRun(w, r)
		case http.MethodGet:
			h.ListScenarioRuns(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Nested endpoints
	if strings.HasPrefix(path, ScenariosRunPath+"/") {
		// Note: WebSocket logs endpoint (/jobs/{jobID}/logs) is handled in server.go
		// before reaching this router, so no need to check for it here

		// Check for /jobs/{jobID} pattern (GET or DELETE single job)
		if strings.HasPrefix(path, ScenariosRunJobsPath+"/") {
			switch r.Method {
			case http.MethodGet:
				h.GetSingleJob(w, r)
			case http.MethodDelete:
				h.DeleteSingleJob(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		// Single scenario run: /api/v1/scenarios/run/{scenarioRunName}
		switch r.Method {
		case http.MethodGet:
			h.GetScenarioRunStatus(w, r)
		case http.MethodDelete:
			h.DeleteScenarioRunComplete(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

// convertMetaTime converts metav1.Time to *time.Time
func convertMetaTime(mt *metav1.Time) *time.Time {
	if mt == nil {
		return nil
	}
	t := mt.Time
	return &t
}

// NOTE: deleteTargetRequest was removed - KrknTargetRequest is now owned by ScenarioRun
// and will be automatically deleted via Kubernetes garbage collection when ScenarioRun is deleted.
// This ensures the Secret (which is owned by KrknTargetRequest) remains available for job retries.
