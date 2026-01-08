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
	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// ClustersResponse represents the response for GET /clusters endpoint
type ClustersResponse struct {
	// TargetData contains a map of operator-name to list of cluster targets
	TargetData map[string][]krknv1alpha1.ClusterTarget `json:"targetData"`
	// Status represents the current state of the request (pending, completed)
	Status string `json:"status"`
}

// NodesResponse represents the response for GET /nodes endpoint
type NodesResponse struct {
	// Nodes contains the list of node names in the cluster
	Nodes []string `json:"nodes"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
