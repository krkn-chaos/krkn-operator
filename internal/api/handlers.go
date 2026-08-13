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
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
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
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/elasticsearch"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	"github.com/krkn-chaos/krkn-operator/pkg/registry"
	pb "github.com/krkn-chaos/krkn-operator/proto/dataprovider"
)

// sanitizeResourceName converts an email or identifier into a valid Kubernetes resource name.
// Kubernetes resource names must follow RFC 1123 subdomain rules:
// - Contain only lowercase alphanumeric characters, '-' or '.'
// - Start and end with an alphanumeric character
// - Maximum length of 253 characters
//
// This function:
// 1. Converts to lowercase
// 2. Replaces @ and . with -
// 3. Replaces any other invalid characters with -
// 4. Ensures it starts and ends with alphanumeric
//
// Example: "[email protected]" -> "admin-example-com"
func sanitizeResourceName(name string) string {
	// Convert to lowercase
	sanitized := strings.ToLower(name)

	// Replace @ and . with -
	sanitized = strings.ReplaceAll(sanitized, "@", "-")
	sanitized = strings.ReplaceAll(sanitized, ".", "-")

	// Replace any other invalid characters with -
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	sanitized = reg.ReplaceAllString(sanitized, "-")

	// Remove leading/trailing dashes
	sanitized = strings.Trim(sanitized, "-")

	// Ensure maximum length (Kubernetes limit is 253, but we'll be conservative)
	if len(sanitized) > 63 {
		sanitized = sanitized[:63]
		sanitized = strings.TrimRight(sanitized, "-")
	}

	return sanitized
}

// sanitizeRunNameLabel converts a customRunName into a valid Kubernetes label value
// (max 63 chars, alphanumeric + dash/dot/underscore, must start and end with alphanumeric).
func sanitizeRunNameLabel(name string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	sanitized := reg.ReplaceAllString(name, "-")
	sanitized = strings.Trim(sanitized, "-_.")
	if len(sanitized) > 63 {
		sanitized = sanitized[:63]
		sanitized = strings.Trim(sanitized, "-_.")
	}
	if sanitized == "" {
		return "custom-run"
	}
	return sanitized
}

// Handler contains the dependencies for API handlers
type Handler struct {
	client         client.Client
	clientset      kubernetes.Interface
	namespace      string
	grpcServerAddr string
	secretManager  *auth.SecretManager
}

// NewHandler creates a new Handler
func NewHandler(client client.Client, clientset kubernetes.Interface, namespace string, grpcServerAddr string, secretManager *auth.SecretManager) *Handler {
	return &Handler{
		client:         client,
		clientset:      clientset,
		namespace:      namespace,
		grpcServerAddr: grpcServerAddr,
		secretManager:  secretManager,
	}
}

// getTokenGenerator creates a TokenGenerator for JWT validation (used for WebSocket auth)
// It uses the same JWT secret as the HTTP middleware via SecretManager
func (h *Handler) getTokenGenerator(ctx context.Context) (*auth.TokenGenerator, error) {
	tokenGen, err := h.secretManager.GetTokenGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to get token generator from SecretManager: %w", err)
	}
	return tokenGen, nil
}

// GetClusters handles GET /api/v1/clusters endpoint
// It fetches the KrknTargetRequest CR by the provided ID and returns the target data
//
// @Summary List available clusters
// @Description Get the list of target clusters from a KrknTargetRequest by ID
// @Tags clusters
// @Produce json
// @Param id query string true "KrknTargetRequest ID"
// @Success 200 {object} ClustersResponse "List of clusters by provider"
// @Failure 400 {object} ErrorResponse "Missing or invalid ID parameter"
// @Failure 404 {object} ErrorResponse "Target request not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /clusters [get]
func (h *Handler) GetClusters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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
	err := h.client.Get(ctx, types.NamespacedName{
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
			log.FromContext(ctx).Error(err, "Failed to fetch KrknTargetRequest", "id", id)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to fetch KrknTargetRequest",
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
	// Admins see all clusters, regular users see only clusters they have 'run' permission for
	// (this endpoint is used to select clusters for running scenarios)
	targetData := targetRequest.Status.TargetData

	claims := auth.GetClaimsFromContext(ctx)
	if claims != nil && !auth.IsAdmin(ctx) {
		// Regular user: filter by group permissions (requires 'run' permission)
		filteredData, err := groupauth.FilterClustersByPermission(
			ctx,
			h.client,
			claims.UserID,
			h.namespace,
			targetData,
			groupauth.ActionRun,
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
// GetNodes handles GET /api/v1/nodes
// Gets cluster nodes from a target
//
// @Summary Get cluster nodes
// @Description Get list of nodes from a cluster target (supports both KrknOperatorTarget UUID and legacy KrknTargetRequest ID)
// @Tags clusters
// @Produce json
// @Param targetUUID query string false "KrknOperatorTarget UUID (new API)"
// @Param id query string false "KrknTargetRequest ID (legacy)"
// @Param cluster-name query string false "Cluster name (legacy, required with id)"
// @Success 200 {object} object "List of nodes"
// @Failure 400 {object} ErrorResponse "Missing or invalid parameters"
// @Failure 404 {object} ErrorResponse "Target not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /nodes [get]
func (h *Handler) GetNodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

	// Check user permissions (group-based access control)
	// Admins bypass validation, regular users must have 'view' permission on the cluster
	claims := auth.GetClaimsFromContext(ctx)
	if claims != nil && !auth.IsAdmin(ctx) {
		// Get cluster API URL for permission check
		clusterAPIURL, err := h.getClusterAPIURL(ctx, targetUUID, id, clusterName)
		if err != nil {
			log.FromContext(ctx).Error(err, "Failed to get cluster API URL for permission check")
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to validate cluster permissions",
			})
			return
		}

		// Check if user has 'view' permission on this cluster
		hasPermission, err := groupauth.HasClusterPermission(
			ctx,
			h.client,
			claims.UserID,
			h.namespace,
			clusterAPIURL,
			groupauth.ActionView,
		)
		if err != nil {
			log.FromContext(ctx).Error(err, "Failed to check cluster permissions", "userID", claims.UserID)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to validate access permissions",
			})
			return
		}

		if !hasPermission {
			log.FromContext(ctx).Info("User lacks permission to view nodes on cluster",
				"userID", claims.UserID,
				"clusterAPIURL", clusterAPIURL,
			)
			writeJSONError(w, http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "You do not have permission to view nodes on this cluster",
			})
			return
		}
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
		log.FromContext(ctx).Error(err, "Failed to get nodes from gRPC service")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get nodes from gRPC service",
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
//
// @Summary Health check
// @Description Check if the operator API is healthy and responding
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string "API is healthy"
// @Security BearerAuth
// @Router /health [get]
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// GetTargetByUUID handles GET /api/v1/targets/{uuid} endpoint (legacy - checks KrknTargetRequest status)
// This endpoint checks the status of a KrknTargetRequest CR created by krkn-operator-acm
//
// @Summary Get target by UUID
// @Description Get target request status and cluster information by UUID (legacy KrknTargetRequest API)
// @Tags targets
// @Produce json
// @Param uuid path string true "Target UUID"
// @Success 200 {object} object "Target status and clusters"
// @Failure 400 {object} ErrorResponse "Invalid UUID"
// @Failure 404 {object} ErrorResponse "Target not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /targets/{uuid} [get]
func (h *Handler) GetTargetByUUID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uuid, err := extractPathSuffix(r.URL.Path, TargetsPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "UUID " + err.Error(),
		})
		return
	}

	var targetRequest krknv1alpha1.KrknTargetRequest
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      uuid,
		Namespace: h.namespace,
	}, &targetRequest); err != nil {
		if client.IgnoreNotFound(err) == nil {
			w.WriteHeader(http.StatusNotFound)
		} else {
			log.FromContext(ctx).Error(err, "Failed to fetch KrknTargetRequest", "uuid", uuid)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to fetch KrknTargetRequest",
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
//
// @Summary Create target request
// @Description Create a KrknTargetRequest to trigger cluster discovery by krkn-operator-acm (legacy API)
// @Tags targets
// @Accept json
// @Produce json
// @Success 201 {object} object "Target request created with UUID"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /targets [post]
func (h *Handler) PostTarget(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Generate a new UUID
	newUUID := uuid.New().String()

	// Extract user claims for ownership tracking
	claims := auth.GetClaimsFromContext(ctx)

	// Build labels with owner tracking
	labels := make(map[string]string)
	if claims != nil {
		labels["krkn.krkn-chaos.dev/owner-user"] = sanitizeUserID(claims.UserID)
	}

	// Create a new KrknTargetRequest CR
	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      newUUID,
			Namespace: h.namespace,
			Labels:    labels,
		},
		Spec: krknv1alpha1.KrknTargetRequestSpec{
			UUID: newUUID,
		},
	}

	// Create the CR in the cluster
	err := h.client.Create(ctx, targetRequest)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to create KrknTargetRequest", "uuid", newUUID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create KrknTargetRequest",
		})
		return
	}

	// Return 102 Processing with the UUID
	response := map[string]string{
		"uuid": newUUID,
	}
	writeJSON(w, http.StatusAccepted, response)
}

// DeleteTargetByUUID handles DELETE /api/v1/targets/{uuid} endpoint
// It deletes a KrknTargetRequest resource by UUID
// Authorization: Admin can delete any resource, owner can delete their own
//
// @Summary Delete target by UUID
// @Description Delete a KrknTargetRequest resource by UUID. Admins can delete any, users can delete their own.
// @Tags targets
// @Produce json
// @Param uuid path string true "Target UUID"
// @Success 200 {object} object "Target deleted successfully"
// @Failure 400 {object} ErrorResponse "Invalid UUID"
// @Failure 403 {object} ErrorResponse "Insufficient permissions"
// @Failure 404 {object} ErrorResponse "Target not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /targets/{uuid} [delete]
func (h *Handler) DeleteTargetByUUID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx).WithName("delete-target")

	// Extract UUID from path
	uuid, err := extractPathSuffix(r.URL.Path, TargetsPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "UUID " + err.Error(),
		})
		return
	}

	logger.Info("Deleting KrknTargetRequest", "uuid", uuid)

	// Fetch the KrknTargetRequest to verify it exists
	var targetRequest krknv1alpha1.KrknTargetRequest
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      uuid,
		Namespace: h.namespace,
	}, &targetRequest); err != nil {
		if client.IgnoreNotFound(err) == nil {
			logger.Info("KrknTargetRequest not found", "uuid", uuid)
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "KrknTargetRequest with UUID '" + uuid + "' not found",
			})
		} else {
			logger.Error(err, "Failed to get KrknTargetRequest", "uuid", uuid)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to get KrknTargetRequest",
			})
		}
		return
	}

	// Admin bypass - can delete any resource
	if auth.IsAdmin(ctx) {
		if err := h.client.Delete(ctx, &targetRequest); err != nil {
			logger.Error(err, "Failed to delete KrknTargetRequest", "uuid", uuid)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to delete KrknTargetRequest",
			})
			return
		}
		logger.Info("Successfully deleted KrknTargetRequest (admin)", "uuid", uuid)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Non-admin: check ownership
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "No authentication claims found",
		})
		return
	}

	// Extract owner from label
	ownerLabel := targetRequest.Labels["krkn.krkn-chaos.dev/owner-user"]
	currentUserSanitized := sanitizeUserID(claims.UserID)

	if ownerLabel != currentUserSanitized {
		logger.Info("Denying delete - user is not the owner",
			"uuid", uuid,
			"userID", claims.UserID,
			"owner", ownerLabel)
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "You can only delete resources you created",
		})
		return
	}

	// User is the owner - proceed with deletion
	if err := h.client.Delete(ctx, &targetRequest); err != nil {
		logger.Error(err, "Failed to delete KrknTargetRequest", "uuid", uuid)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete KrknTargetRequest",
		})
		return
	}

	logger.Info("Successfully deleted KrknTargetRequest (owner)", "uuid", uuid)
	w.WriteHeader(http.StatusNoContent)
}

// TargetsHandler handles GET, POST, and DELETE for /api/v1/targets endpoints
// It routes to the appropriate handler based on the HTTP method
func (h *Handler) TargetsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.GetTargetByUUID(w, r)
	} else if r.Method == http.MethodPost {
		h.PostTarget(w, r)
	} else if r.Method == http.MethodDelete {
		h.DeleteTargetByUUID(w, r)
	} else {
		writeJSONError(w, http.StatusMethodNotAllowed, ErrorResponse{
			Error:   "method_not_allowed",
			Message: "Only GET, POST, and DELETE methods are allowed",
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
			Requires:          field.Requires,
			MutuallyExcludes:  field.MutuallyExcludes,
			Secret:            field.Secret,
			Group:             field.Group,
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
func (h *Handler) parseRegistryRequest(r *http.Request) (*models.RegistryV2, provider.Mode, error) {
	ctx := r.Context()

	// Empty request body → use default quay.io
	if r.ContentLength == 0 {
		return nil, provider.Quay, nil
	}

	var req ScenariosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, provider.Quay, fmt.Errorf("invalid request body: %w", err)
	}

	// Check for named registry
	if req.RegistryName != nil && *req.RegistryName != "" {
		// Load registry secret
		secret, err := h.loadRegistrySecret(ctx, *req.RegistryName)
		if err != nil {
			return nil, provider.Quay, fmt.Errorf("registry '%s' not found", *req.RegistryName)
		}

		// Check user access
		if !h.canAccessRegistry(ctx, secret) {
			return nil, provider.Quay, fmt.Errorf("access denied to registry '%s'", *req.RegistryName)
		}

		// Extract RegistryV2 from secret
		registryV2, err := registry.ExtractRegistryV2FromSecret(secret)
		if err != nil {
			return nil, provider.Quay, fmt.Errorf("invalid registry secret: %w", err)
		}

		return registryV2, provider.Private, nil
	}

	// No registry specified → use default quay.io
	return nil, provider.Quay, nil
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
//
// @Summary List available scenarios
// @Description Get list of available chaos scenarios from container registry (Quay.io or private registry)
// @Tags scenarios
// @Accept json
// @Produce json
// @Param registry body object false "Registry configuration (optional for private registries)"
// @Success 200 {object} object "List of available scenarios"
// @Failure 400 {object} ErrorResponse "Invalid registry configuration"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /scenarios [post]
func (h *Handler) PostScenarios(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	registry, mode, err := h.parseRegistryRequest(r)
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

	// For API calls: if Token is set (token auth), clear Username/Password to force Bearer auth
	// Username/Password are only for image pull, not for API calls
	apiRegistry := registry
	if registry != nil && registry.Token != nil {
		apiRegistry = &models.RegistryV2{
			RegistryURL:        registry.RegistryURL,
			ScenarioRepository: registry.ScenarioRepository,
			Token:              registry.Token,
			SkipTLS:            registry.SkipTLS,
			Insecure:           registry.Insecure,
			// Username and Password intentionally nil for token-based API auth
		}
	}

	// Get registry images (scenario list)
	scenarioTags, err := scenarioProvider.GetRegistryImages(apiRegistry)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to get scenarios from registry", "registry", apiRegistry)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get scenarios from registry",
		})
		return
	}

	scenarios := filterScenariosByIsAScenario(ctx, scenarioProvider, scenarioTags, apiRegistry)

	// Return response
	response := ScenariosResponse{
		Scenarios: scenarios,
	}

	writeJSON(w, http.StatusOK, response)
}

// maxConcurrentDetailFetches limits how many GetScenarioDetail calls run in parallel
// to avoid overwhelming the upstream registry.
const maxConcurrentDetailFetches = 10

// filterScenariosByIsAScenario concurrently fetches detail for each tag and returns
// only those with IsAScenario == true, preserving input order.
// Uses a fixed worker pool so goroutine count is bounded and context cancellation
// stops queued work immediately (ongoing GetScenarioDetail calls finish since the
// upstream interface does not accept a context).
func filterScenariosByIsAScenario(ctx context.Context, scenarioProvider provider.ScenarioDataProvider, scenarioTags *[]models.ScenarioTag, registry *models.RegistryV2) []ScenarioTag {
	if scenarioTags == nil || len(*scenarioTags) == 0 {
		return []ScenarioTag{}
	}

	tags := *scenarioTags
	results := make([]*ScenarioTag, len(tags))

	work := make(chan int, len(tags))
	for i := range tags {
		work <- i
	}
	close(work)

	workerCount := maxConcurrentDetailFetches
	if len(tags) < workerCount {
		workerCount = len(tags)
	}

	g, gCtx := errgroup.WithContext(ctx)
	for w := 0; w < workerCount; w++ {
		g.Go(func() error {
			for i := range work {
				if gCtx.Err() != nil {
					return nil
				}

				t := tags[i]
				detail, err := scenarioProvider.GetScenarioDetail(t.Name, registry)
				if err != nil {
					log.FromContext(ctx).V(1).Info("Skipping scenario: failed to get detail", "scenario", t.Name, "error", err)
					continue
				}
				if detail == nil || !detail.IsAScenario {
					continue
				}

				results[i] = &ScenarioTag{
					Name:         t.Name,
					Digest:       t.Digest,
					Size:         t.Size,
					LastModified: t.LastModified,
				}
			}
			return nil
		})
	}
	_ = g.Wait()

	scenarios := make([]ScenarioTag, 0, len(tags))
	for _, r := range results {
		if r != nil {
			scenarios = append(scenarios, *r)
		}
	}
	return scenarios
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
//
// @Summary Get scenario details
// @Description Get detailed information about a specific chaos scenario including configuration fields
// @Tags scenarios
// @Accept json
// @Produce json
// @Param scenario_name path string true "Scenario name"
// @Param registry body object false "Registry configuration (optional for private registries)"
// @Success 200 {object} object "Scenario details with input fields"
// @Failure 400 {object} ErrorResponse "Invalid scenario name or registry"
// @Failure 404 {object} ErrorResponse "Scenario not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /scenarios/detail/{scenario_name} [post]
func (h *Handler) PostScenarioDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scenarioName, err := extractPathSuffix(r.URL.Path, ScenariosDetailPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "scenario_name " + err.Error(),
		})
		return
	}

	registry, mode, err := h.parseRegistryRequest(r)
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

	// For API calls: if Token is set (token auth), clear Username/Password to force Bearer auth
	// Username/Password are only for image pull, not for API calls
	apiRegistry := registry
	if registry != nil && registry.Token != nil {
		apiRegistry = &models.RegistryV2{
			RegistryURL:        registry.RegistryURL,
			ScenarioRepository: registry.ScenarioRepository,
			Token:              registry.Token,
			SkipTLS:            registry.SkipTLS,
			Insecure:           registry.Insecure,
			// Username and Password intentionally nil for token-based API auth
		}
	}

	// Get scenario detail
	scenarioDetail, err := scenarioProvider.GetScenarioDetail(scenarioName, apiRegistry)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to get scenario detail", "scenarioName", scenarioName, "registry", registry)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get scenario detail",
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
//
// @Summary Get scenario globals
// @Description Get global environment configuration fields for a specific scenario
// @Tags scenarios
// @Accept json
// @Produce json
// @Param scenario_name path string true "Scenario name"
// @Param registry body object false "Registry configuration (optional for private registries)"
// @Success 200 {object} object "Global fields for scenario"
// @Failure 400 {object} ErrorResponse "Invalid scenario name or registry"
// @Failure 404 {object} ErrorResponse "Scenario not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /scenarios/globals/{scenario_name} [post]
func (h *Handler) PostScenarioGlobals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scenarioName, err := extractPathSuffix(r.URL.Path, ScenariosGlobalsPath+"/")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "scenario_name " + err.Error(),
		})
		return
	}

	registry, mode, err := h.parseRegistryRequest(r)
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

	// For API calls: if Token is set (token auth), clear Username/Password to force Bearer auth
	// Username/Password are only for image pull, not for API calls
	apiRegistry := registry
	if registry != nil && registry.Token != nil {
		apiRegistry = &models.RegistryV2{
			RegistryURL:        registry.RegistryURL,
			ScenarioRepository: registry.ScenarioRepository,
			Token:              registry.Token,
			SkipTLS:            registry.SkipTLS,
			Insecure:           registry.Insecure,
			// Username and Password intentionally nil for token-based API auth
		}
	}

	// Get global environment
	globalDetail, err := scenarioProvider.GetGlobalEnvironment(apiRegistry, scenarioName)
	if err != nil {
		if errors.Is(err, provider.ErrLabelNotFound) {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Global environment for scenario '" + scenarioName + "' not found",
			})
			return
		}
		log.FromContext(ctx).Error(err, "Failed to get global environment", "registry", registry, "scenarioName", scenarioName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get global environment",
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
// Executes a chaos scenario on target clusters
//
// @Summary Run chaos scenario
// @Description Execute a chaos scenario on target clusters with specified configuration
// @Tags scenarios
// @Accept json
// @Produce json
// @Param request body ScenarioRunRequest true "Scenario run configuration"
// @Success 202 {object} object "Scenario run created"
// @Failure 400 {object} ErrorResponse "Invalid request body or validation error"
// @Failure 403 {object} ErrorResponse "Insufficient permissions"
// @Failure 404 {object} ErrorResponse "Target or scenario not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /scenarios/run [post]
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

	// Fetch KrknTargetRequest to build cluster API URL mapping and validate permissions
	targetRequest := &krknv1alpha1.KrknTargetRequest{}
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      req.TargetRequestID,
		Namespace: h.namespace,
	}, targetRequest); err != nil {
		logger.Error(err, "Failed to fetch target request", "targetRequestId", req.TargetRequestID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch target request",
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

	// Validate user permissions (group-based access control)
	// Admins bypass validation, regular users must have 'run' permission on all target clusters
	userClaims := auth.GetClaimsFromContext(ctx)
	if userClaims != nil && !auth.IsAdmin(ctx) {
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

	// Process file references: validate access and translate to FileMount
	translatedFiles := make([]krknv1alpha1.FileMount, 0, len(req.FileReferences))

	for _, fileRef := range req.FileReferences {
		// Validate mount path is absolute
		if !filepath.IsAbs(fileRef.MountPath) {
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: fmt.Sprintf("mount path must be absolute: %s", fileRef.MountPath),
			})
			return
		}

		// Load file ConfigMap by UUID
		fileConfigMap, err := h.loadFileConfigMapByID(ctx, fileRef.FileID)
		if err != nil {
			if apierrors.IsNotFound(err) {
				writeJSONError(w, http.StatusBadRequest, ErrorResponse{
					Error:   "bad_request",
					Message: fmt.Sprintf("file with ID '%s' not found", fileRef.FileID),
				})
			} else {
				logger.Error(err, "Failed to load file ConfigMap", "fileID", fileRef.FileID)
				writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
					Error:   "internal_error",
					Message: "Failed to load file",
				})
			}
			return
		}

		// Validate user has access to this file
		hasAccess, err := h.canAccessFile(ctx, fileConfigMap)
		if err != nil {
			logger.Error(err, "Failed to check file access", "fileID", fileRef.FileID)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to validate file access permissions",
			})

			return
		}
		if !hasAccess {
			writeJSONError(w, http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: fmt.Sprintf("You do not have access to file '%s'", fileRef.FileID),
			})
			return
		}

		// Extract file content from ConfigMap (first data key)
		var fileName, content string
		for k, v := range fileConfigMap.Data {
			fileName = k
			content = v
			break
		}

		// Base64 encode content for FileMount
		encodedContent := base64.StdEncoding.EncodeToString([]byte(content))

		// Translate to FileMount (preserve FileID for replay functionality)
		translatedFiles = append(translatedFiles, krknv1alpha1.FileMount{
			Name:      fileName,
			Content:   encodedContent,
			MountPath: fileRef.MountPath,
			FileID:    fileRef.FileID,
		})

		logger.V(1).Info("Translated file reference",
			"fileID", fileRef.FileID,
			"fileName", fileName,
			"mountPath", fileRef.MountPath,
		)
	}

	// Convert inline Files from API type to CRD type
	inlineFiles := make([]krknv1alpha1.FileMount, 0, len(req.Files))
	for _, f := range req.Files {
		inlineFiles = append(inlineFiles, krknv1alpha1.FileMount{
			Name:      f.Name,
			Content:   f.Content,
			MountPath: f.MountPath,
		})
	}

	// Merge inline Files with translated FileReferences
	allFiles := append(inlineFiles, translatedFiles...)

	// Load registry configuration if specified
	var registryConfig *models.RegistryV2
	if req.RegistryName != nil && *req.RegistryName != "" {
		secret, err := h.loadRegistrySecret(ctx, *req.RegistryName)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Registry '%s' not found", *req.RegistryName),
			})
			return
		}

		// Check user access
		if !h.canAccessRegistry(ctx, secret) {
			writeJSONError(w, http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: fmt.Sprintf("Access denied to registry '%s'", *req.RegistryName),
			})
			return
		}

		// Extract registry configuration
		registryConfig, err = registry.ExtractRegistryV2FromSecret(secret)
		if err != nil {
			logger.Error(err, "Failed to extract registry config from secret", "registryName", *req.RegistryName)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to load registry configuration",
			})
			return
		}

		logger.V(1).Info("Loaded registry configuration", "registryName", *req.RegistryName)
	}

	// Generate scenario run name.
	// When a customRunName is provided, use its sanitized form as the CR name so that
	// Kubernetes enforces uniqueness atomically at create time — no TOCTOU window, and
	// resources created via kubectl are subject to the same constraint.
	var scenarioRunName string
	if req.CustomRunName != "" {
		scenarioRunName = sanitizeResourceName(req.CustomRunName)
	} else {
		scenarioRunName = fmt.Sprintf("%s-%s", req.ScenarioName, uuid.New().String()[:8])
	}
	// Inject Elasticsearch credentials server-side so the password is never sent by the client.
	if req.ElasticsearchConfigName != "" {
		esSecret, err := h.loadElasticsearchConfigSecret(ctx, req.ElasticsearchConfigName)
		if err != nil {
			logger.Error(err, "Failed to load elasticsearch config for run", "name", req.ElasticsearchConfigName)
			writeJSONError(w, http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: fmt.Sprintf("Elasticsearch config '%s' not found or inaccessible", req.ElasticsearchConfigName),
			})
			return
		}
		if req.Environment == nil {
			req.Environment = make(map[string]string)
		}
		// Always inject password; only inject other vars if not already provided by the client.
		if p, ok := esSecret.Data[elasticsearch.SecretKeyPassword]; ok {
			req.Environment["ES_PASSWORD"] = string(p)
		}
		if u, ok := esSecret.Data[elasticsearch.SecretKeyUsername]; ok {
			if _, set := req.Environment["ES_USERNAME"]; !set {
				req.Environment["ES_USERNAME"] = string(u)
			}
		}
	}

	// Create KrknScenarioRun CR
	// Extract user claims for ownership tracking (defensive check for tests)
	claims := auth.GetClaimsFromContext(ctx)

	labels := make(map[string]string)
	ownerUserID := ""
	if claims != nil {
		labels["krkn.krkn-chaos.dev/owner-user"] = sanitizeUserID(claims.UserID)
		ownerUserID = claims.UserID
	}
	if req.CustomRunName != "" {
		labels["krkn.krkn-chaos.dev/custom-run-name"] = sanitizeRunNameLabel(req.CustomRunName)
	}

	scenarioRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scenarioRunName,
			Namespace: h.namespace,
			Labels:    labels,
		},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			TargetRequestID: req.TargetRequestID,
			OwnerUserID:     ownerUserID,
			TargetClusters:  req.TargetClusters,
			ScenarioName:    req.ScenarioName,
			ScenarioImage:   req.ScenarioImage,
			KubeconfigPath:  req.KubeconfigPath,
			Environment:     req.Environment,
			CustomRunName:   req.CustomRunName,
		},
	}

	// Set registry configuration if loaded
	if registryConfig != nil {
		scenarioRun.Spec.RegistryName = *req.RegistryName
		scenarioRun.Spec.RegistryURL = registryConfig.RegistryURL
		scenarioRun.Spec.ScenarioRepository = registryConfig.ScenarioRepository
		if registryConfig.Token != nil {
			scenarioRun.Spec.Token = *registryConfig.Token
		}
		if registryConfig.Username != nil {
			scenarioRun.Spec.Username = *registryConfig.Username
		}
		if registryConfig.Password != nil {
			scenarioRun.Spec.Password = *registryConfig.Password
		}
	}

	// Convert FileMount from API type to CRD type (merged from inline Files and translated FileReferences)
	if len(allFiles) > 0 {
		scenarioRun.Spec.Files = make([]krknv1alpha1.FileMount, len(allFiles))
		for i, f := range allFiles {
			scenarioRun.Spec.Files[i] = krknv1alpha1.FileMount{
				Name:      f.Name,
				Content:   f.Content,
				MountPath: f.MountPath,
				FileID:    f.FileID, // Preserve FileID for replay functionality
			}
		}
	}

	// Create the CR — if customRunName was provided, a name collision produces AlreadyExists.
	if err := h.client.Create(ctx, scenarioRun); err != nil {
		if apierrors.IsAlreadyExists(err) {
			writeJSONError(w, http.StatusConflict, ErrorResponse{
				Error:   "conflict",
				Message: fmt.Sprintf("A scenario run with the name '%s' already exists", req.CustomRunName),
			})
			return
		}
		logger.Error(err, "Failed to create scenario run", "scenarioRunName", scenarioRunName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create scenario run",
		})
		return
	}

	// Set owner reference: ScenarioRun owns KrknTargetRequest
	// This ensures KrknTargetRequest (and its Secret) are cleaned up when ScenarioRun is deleted
	// and remain available for job retries while ScenarioRun exists
	// targetRequest already fetched above for permission validation and cluster API URL mapping
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
		CustomRunName:   req.CustomRunName,
	}

	writeJSON(w, http.StatusCreated, response)
}

// GetScenarioRunStatus handles GET /api/v1/scenarios/run/{scenarioRunName} endpoint
// It returns the current status of a scenario run
//
// @Summary Get scenario run status
// @Description Get current execution status and metrics for a running or completed scenario
// @Tags scenarios
// @Produce json
// @Param scenarioRunName path string true "Scenario run name"
// @Success 200 {object} object "Scenario run status"
// @Failure 400 {object} ErrorResponse "Invalid scenario run name"
// @Failure 403 {object} ErrorResponse "Insufficient permissions"
// @Failure 404 {object} ErrorResponse "Scenario run not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /scenarios/run/{scenarioRunName} [get]
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

	claims := auth.GetClaimsFromContext(ctx)

	// Filter jobs based on permissions (admins see all, users see only authorized jobs)
	filteredJobs := scenarioRun.Status.ClusterJobs
	if claims != nil && !auth.IsAdmin(ctx) {
		// Fetch user groups
		userGroups, err := groupauth.GetUserGroups(ctx, h.client, claims.UserID, h.namespace)
		if err != nil {
			log.FromContext(ctx).Error(err, "Failed to fetch user groups", "userID", claims.UserID)
			writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to fetch user groups",
			})
			return
		}

		// Count jobs with ClusterAPIURL populated (to detect newly created runs)
		jobsWithClusterURL := 0
		for _, job := range scenarioRun.Status.ClusterJobs {
			if job.ClusterAPIURL != "" {
				jobsWithClusterURL++
			}
		}

		// Filter jobs to only those user has view permission for
		filteredJobs = h.filterJobsByPermission(
			scenarioRun.Status.ClusterJobs,
			ctx,
			userGroups,
			groupauth.ActionView,
		)

		// Explicit authorization checks based on job state:
		if len(filteredJobs) == 0 {
			if jobsWithClusterURL == 0 {
				// Case 1: No jobs have ClusterAPIURL (run just created, controller hasn't processed yet)
				// Allow access and return 201 Created with empty jobs array
				response := ScenarioRunStatusResponse{
					ScenarioRunName:  scenarioRunName,
					Phase:            scenarioRun.Status.Phase,
					TotalTargets:     scenarioRun.Status.TotalTargets,
					SuccessfulJobs:   scenarioRun.Status.SuccessfulJobs,
					FailedJobs:       scenarioRun.Status.FailedJobs,
					RunningJobs:      scenarioRun.Status.RunningJobs,
					ClusterJobs:      []ClusterJobStatusResponse{},
					OwnerUserID:      scenarioRun.Spec.OwnerUserID,
					RegistryName:     scenarioRun.Spec.RegistryName,
					GraphRunName:     scenarioRun.Labels["krkn.dev/graph-run"],
					GraphNodeID:      scenarioRun.Labels["krkn.dev/graph-node"],
					ResiliencyScores: convertClusterResiliencyScores(scenarioRun.Status.ResiliencyScores),
				}
				writeJSON(w, http.StatusCreated, response)
				return
			} else {
				// Case 2: Jobs have ClusterAPIURL but user has no permission on any
				// Deny access with 403 Forbidden
				writeJSONError(w, http.StatusForbidden, ErrorResponse{
					Error:   "forbidden",
					Message: "Access denied. You do not have permission to view jobs in this scenario run",
				})
				return
			}
		}
	}

	// Convert filtered ClusterJobStatus to response type
	clusterJobs := make([]ClusterJobStatusResponse, len(filteredJobs))
	for i, job := range filteredJobs {
		clusterJobs[i] = ClusterJobStatusResponse{
			ProviderName:    job.ProviderName,
			ClusterName:     job.ClusterName,
			JobID:           job.JobID,
			PodName:         job.PodName,
			ContainerImage:  job.ContainerImage,
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
		ScenarioRunName:  scenarioRunName,
		Phase:            scenarioRun.Status.Phase,
		TotalTargets:     scenarioRun.Status.TotalTargets,
		SuccessfulJobs:   scenarioRun.Status.SuccessfulJobs,
		FailedJobs:       scenarioRun.Status.FailedJobs,
		RunningJobs:      scenarioRun.Status.RunningJobs,
		ClusterJobs:      clusterJobs,
		OwnerUserID:      scenarioRun.Spec.OwnerUserID,
		RegistryName:     scenarioRun.Spec.RegistryName,
		GraphRunName:     scenarioRun.Labels["krkn.dev/graph-run"],
		GraphNodeID:      scenarioRun.Labels["krkn.dev/graph-node"],
		CustomRunName:    scenarioRun.Spec.CustomRunName,
		ResiliencyScore:  averageResiliencyScore(scenarioRun.Status.ResiliencyScores),
		ResiliencyScores: convertClusterResiliencyScores(scenarioRun.Status.ResiliencyScores),
	}

	writeJSON(w, http.StatusOK, response)
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     checkWebSocketOrigin,
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
	// Path format v1: /api/v1/scenarios/run/{scenarioRunName}/jobs/{jobID}/logs
	// Path format v2: /api/v2/ws/scenarios/run/{scenarioRunName}/jobs/{jobID}/logs
	path := r.URL.Path

	// Try v2 path first, then v1
	var remainder string
	v2Prefix := "/api/v2/ws/scenarios/run/"
	v1Prefix := ScenariosRunPath + "/"

	if strings.HasPrefix(path, v2Prefix) {
		remainder = path[len(v2Prefix):]
	} else if strings.HasPrefix(path, v1Prefix) {
		remainder = path[len(v1Prefix):]
	} else {
		logger.Error(nil, "Invalid logs endpoint path", "path", path, "expected_v1", v1Prefix, "expected_v2", v2Prefix)
		_ = conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Invalid logs endpoint path")) // Best-effort error reporting
		return
	}

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

	// Create context with claims for permission checks
	ctx := context.WithValue(context.Background(), auth.UserClaimsKey, claims)

	// Fetch the scenario run to check permissions
	var scenarioRun krknv1alpha1.KrknScenarioRun
	if err := h.client.Get(ctx, client.ObjectKey{
		Name:      scenarioRunName,
		Namespace: h.namespace,
	}, &scenarioRun); err != nil {
		logger.Error(err, "Failed to fetch scenario run", "scenarioRunName", scenarioRunName)
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: Scenario run '%s' not found", scenarioRunName)))
		return
	}

	// Find the specific job for this jobID
	var targetJob *krknv1alpha1.ClusterJobStatus
	for i := range scenarioRun.Status.ClusterJobs {
		if scenarioRun.Status.ClusterJobs[i].JobID == jobID {
			targetJob = &scenarioRun.Status.ClusterJobs[i]
			break
		}
	}

	if targetJob == nil {
		logger.Error(nil, "Job not found in scenario run",
			"scenarioRunName", scenarioRunName,
			"jobID", jobID)
		_ = conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Job not found in scenario run"))
		return
	}

	// Check if user has 'view' permission on this specific job's cluster
	if !auth.IsAdmin(ctx) {
		if targetJob.ClusterAPIURL == "" {
			logger.Error(nil, "Access denied: job has no cluster API URL",
				"scenarioRunName", scenarioRunName,
				"jobID", jobID,
				"userID", claims.UserID)
			_ = conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Access denied. Job has no cluster API URL"))
			return
		}

		hasAccess, err := groupauth.HasClusterPermission(
			ctx,
			h.client,
			claims.UserID,
			h.namespace,
			targetJob.ClusterAPIURL,
			groupauth.ActionView,
		)

		if err != nil {
			logger.Error(err, "Failed to check job access for logs",
				"scenarioRunName", scenarioRunName,
				"jobID", jobID,
				"userID", claims.UserID)
			_ = conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Failed to validate access permissions"))
			return
		}

		if !hasAccess {
			logger.Info("Access denied: user does not have view permission on job",
				"scenarioRunName", scenarioRunName,
				"jobID", jobID,
				"userID", claims.UserID,
				"clusterAPIURL", targetJob.ClusterAPIURL)
			_ = conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Access denied. You do not have permission to view logs for this job"))
			return
		}
	}

	logger.Info("Permission check passed for log access",
		"scenarioRunName", scenarioRunName,
		"jobID", jobID,
		"userID", claims.UserID,
		"clusterAPIURL", targetJob.ClusterAPIURL,
		"isAdmin", auth.IsAdmin(ctx))

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

	// Use PodName directly from CR status (already fetched above for permissions)
	if targetJob.PodName == "" {
		logger.Error(nil, "Job has no associated pod", "jobID", jobID)
		_ = conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Job has no associated pod — it may have failed before a pod was created")) // Best-effort error reporting
		return
	}

	var pod corev1.Pod
	if err := h.client.Get(ctx, client.ObjectKey{Name: targetJob.PodName, Namespace: h.namespace}, &pod); err != nil {
		if client.IgnoreNotFound(err) == nil {
			logger.Error(nil, "Pod no longer exists", "jobID", jobID, "podName", targetJob.PodName)
			_ = conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Pod no longer exists — logs may have been cleaned up")) // Best-effort error reporting
		} else {
			logger.Error(err, "Failed to get pod", "jobID", jobID, "podName", targetJob.PodName)
			_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: Failed to get pod: %s", err.Error()))) // Best-effort error reporting
		}
		return
	}

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
//
// @Summary List scenario runs
// @Description Get list of all scenario runs with optional filtering by phase or scenario name
// @Tags scenarios
// @Produce json
// @Param phase query string false "Filter by phase (Running, Succeeded, Failed)"
// @Param scenarioName query string false "Filter by scenario name"
// @Param page query int false "Page number (1-based). Omit for all results."
// @Param limit query int false "Items per page (defaults to jobs.defaultPageSize config, fallback 20; max 500). Only used when page is set."
// @Success 200 {object} ScenarioRunListResponse "List of scenario runs with pagination"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /scenarios/run [get]
func (h *Handler) ListScenarioRuns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters for filtering
	phaseFilter := r.URL.Query().Get("phase") // e.g., Running, Succeeded, Failed
	scenarioNameFilter := r.URL.Query().Get("scenarioName")

	// List all KrknScenarioRun CRs in the namespace
	var scenarioRunList krknv1alpha1.KrknScenarioRunList
	if err := h.client.List(ctx, &scenarioRunList, client.InNamespace(h.namespace)); err != nil {
		log.FromContext(ctx).Error(err, "Failed to list scenario runs")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list scenario runs",
		})
		return
	}

	// Filter by group permissions (admins see all, users see runs with group view permission)
	scenarioRunList.Items = h.FilterScenarioRunsByGroupPermission(scenarioRunList.Items, ctx)

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
			ScenarioRunName:  sr.Name,
			ScenarioName:     sr.Spec.ScenarioName,
			Phase:            sr.Status.Phase,
			TotalTargets:     sr.Status.TotalTargets,
			SuccessfulJobs:   sr.Status.SuccessfulJobs,
			FailedJobs:       sr.Status.FailedJobs,
			RunningJobs:      sr.Status.RunningJobs,
			CreatedAt:        sr.CreationTimestamp.Time,
			OwnerUserID:      sr.Spec.OwnerUserID,
			GraphRunName:     sr.Labels["krkn.dev/graph-run"],
			GraphNodeID:      sr.Labels["krkn.dev/graph-node"],
			CustomRunName:    sr.Spec.CustomRunName,
			ResiliencyScore:  averageResiliencyScore(sr.Status.ResiliencyScores),
			ResiliencyScores: convertClusterResiliencyScores(sr.Status.ResiliencyScores),
		}

		runs = append(runs, run)
	}

	sort.Slice(runs, func(i, j int) bool {
		if runs[i].CreatedAt.Equal(runs[j].CreatedAt) {
			return runs[i].ScenarioRunName < runs[j].ScenarioRunName
		}
		return runs[i].CreatedAt.After(runs[j].CreatedAt)
	})

	page, limit := ParsePaginationParams(r, getDefaultPageSize())

	var response ScenarioRunListResponse
	if page == 0 {
		response = ScenarioRunListResponse{
			ScenarioRuns: runs,
			Pagination:   PaginationMeta{Total: len(runs)},
		}
	} else {
		paginated, meta := PaginateSlice(runs, page, limit)
		response = ScenarioRunListResponse{
			ScenarioRuns: paginated,
			Pagination:   meta,
		}
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
		log.FromContext(ctx).Error(err, "Failed to list scenario runs")
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list scenario runs",
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
//
// @Summary Delete scenario run
// @Description Stop and delete a running or completed scenario run
// @Tags scenarios
// @Produce json
// @Param jobID path string true "Scenario run job ID"
// @Success 200 {object} object "Scenario run deleted"
// @Failure 400 {object} ErrorResponse "Invalid job ID"
// @Failure 403 {object} ErrorResponse "Insufficient permissions"
// @Failure 404 {object} ErrorResponse "Scenario run not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /scenarios/run/{jobID} [delete]
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
		log.FromContext(ctx).Error(err, "Failed to list pods", "jobID", jobID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list pods",
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
			if !h.checkScenarioRunAccess(w, r, &scenarioRun) {
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
		log.FromContext(ctx).Error(err, "Failed to delete pod", "podName", pod.Name, "jobID", jobID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete pod",
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

	// Check if user can cancel the entire scenario run
	// Admin can cancel anything, regular users must have 'cancel' permission on ALL jobs
	claims := auth.GetClaimsFromContext(ctx)
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "No authentication claims found",
		})
		return
	}

	hasAccess, err := h.checkScenarioRunCancelAccess(
		ctx,
		claims.UserID,
		&scenarioRun,
	)

	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to validate cancel permissions", "scenarioRunName", scenarioRunName, "userID", claims.UserID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to validate cancel permissions",
		})
		return
	}

	if !hasAccess {
		writeJSONError(w, http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "Access denied. You must have cancel permission on all jobs in this run to delete it",
		})
		return
	}

	log.Log.Info("deleting entire scenario run",
		"scenarioRunName", scenarioRunName,
		"totalJobs", len(scenarioRun.Status.ClusterJobs),
		"phase", scenarioRun.Status.Phase)

	// Delete the CR - owner references will cascade delete all pods/configmaps/secrets
	if err := h.client.Delete(ctx, &scenarioRun); err != nil {
		log.FromContext(ctx).Error(err, "Failed to delete scenario run", "scenarioRunName", scenarioRunName)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete scenario run",
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

	// Check if user has 'cancel' permission on this specific job
	if !h.checkJobAccess(w, r, job, groupauth.ActionCancel, "cancel") {
		return
	}

	log.Log.Info("cancelling single job",
		"scenarioRunName", foundScenarioRun.Name,
		"jobID", jobID,
		"clusterName", job.ClusterName,
		"currentPhase", job.Phase)

	// Set CancelRequested flag
	job.CancelRequested = true

	// Update CR status
	if err := h.client.Status().Update(ctx, foundScenarioRun); err != nil {
		log.FromContext(ctx).Error(err, "Failed to update scenario run status", "scenarioRunName", foundScenarioRun.Name, "jobID", jobID)
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update scenario run status",
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

	for i := range scenarioRunList.Items {
		sr := &scenarioRunList.Items[i]
		for j := range sr.Status.ClusterJobs {
			if sr.Status.ClusterJobs[j].JobID == jobID {
				foundJob = &sr.Status.ClusterJobs[j]
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

	// Check if user has permission to view this specific job
	if !h.checkJobAccess(w, r, foundJob, groupauth.ActionView, "view") {
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

	// Normalize v2 paths to v1 for backward-compatible routing
	// v2 REST endpoints reuse v1 handler logic (same behavior, different path prefix)
	path = strings.Replace(path, "/api/v2/", "/api/v1/", 1)

	// Root endpoint: /api/v1/scenarios/run (or /api/v2/scenarios/run normalized)
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

		// Check for /replay/{jobID} pattern (GET only - scenario replay)
		if strings.HasPrefix(path, ScenariosRunPath+"/replay/") {
			if r.Method == http.MethodGet {
				h.GetScenarioReplay(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

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

// averageResiliencyScore returns the average score from per-cluster resiliency scores,
// or nil if the slice is empty.
func averageResiliencyScore(scores []krknv1alpha1.ClusterResiliencyScore) *float64 {
	if len(scores) == 0 {
		return nil
	}
	var sum float64
	for _, s := range scores {
		sum += s.Score
	}
	avg := sum / float64(len(scores))
	return &avg
}

// NOTE: deleteTargetRequest was removed - KrknTargetRequest is now owned by ScenarioRun
// and will be automatically deleted via Kubernetes garbage collection when ScenarioRun is deleted.
// This ensures the Secret (which is owned by KrknTargetRequest) remains available for job retries.
