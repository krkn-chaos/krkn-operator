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
	"net/http"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	pb "github.com/krkn-chaos/krkn-operator/proto/dataprovider"
	corev1 "k8s.io/api/core/v1"
)

// Handler contains the dependencies for API handlers
type Handler struct {
	client         client.Client
	namespace      string
	grpcServerAddr string
}

// NewHandler creates a new Handler
func NewHandler(client client.Client, namespace string, grpcServerAddr string) *Handler {
	return &Handler{
		client:         client,
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

	// Fetch the secret with the same name as the KrknTargetRequest ID
	var secret corev1.Secret
	err := h.client.Get(context.Background(), types.NamespacedName{
		Name:      id,
		Namespace: h.namespace,
	}, &secret)

	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Secret not found
			writeJSONError(w, http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Secret with id '" + id + "' not found",
			})
			return
		}
		// Other error
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to fetch Secret: " + err.Error(),
		})
		return
	}

	// Retrieve the managed-clusters JSON from the secret data
	managedClustersBytes, exists := secret.Data["managed-clusters"]
	if !exists {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "managed-clusters not found in secret",
		})
		return
	}

	// Parse the JSON to extract cluster configurations
	// Structure: { "krkn-operator-acm": { "cluster-name": { "kubeconfig": "base64..." } } }
	var managedClusters map[string]map[string]struct {
		Kubeconfig string `json:"kubeconfig"`
	}
	if err := json.Unmarshal(managedClustersBytes, &managedClusters); err != nil {
		writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to parse managed-clusters JSON: " + err.Error(),
		})
		return
	}

	// Get the krkn-operator-acm object
	acmClusters, exists := managedClusters["krkn-operator-acm"]
	if !exists {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "krkn-operator-acm not found in managed-clusters",
		})
		return
	}

	// Check if the requested cluster exists
	clusterConfig, exists := acmClusters[clusterName]
	if !exists {
		writeJSONError(w, http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Cluster '" + clusterName + "' not found in krkn-operator-acm",
		})
		return
	}

	// Call gRPC service to get nodes
	// The kubeconfig is already in base64 format in the secret
	nodes, err := h.callGetNodesGRPC(clusterConfig.Kubeconfig)
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
			// Spec fields can be empty or set based on request body if needed
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
