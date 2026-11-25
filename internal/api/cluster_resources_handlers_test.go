/*
Copyright 2026.

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
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	pb "github.com/krkn-chaos/krkn-operator/proto/dataprovider"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type resourceProviderStub struct {
	pb.UnimplementedDataProviderServiceServer
	request  *pb.ExecuteKubectlRequest
	response *pb.ExecuteKubectlResponse
	err      error
}

func (s *resourceProviderStub) ExecuteKubectl(_ context.Context, request *pb.ExecuteKubectlRequest) (*pb.ExecuteKubectlResponse, error) {
	s.request = request
	return s.response, s.err
}

func newClusterResourceTestHandler(t *testing.T, stub *resourceProviderStub) *Handler {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	pb.RegisterDataProviderServiceServer(server, stub)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	handler := newKubeconfigHelperHandler(t, "target-1",
		map[string]map[string]map[string]string{
			"operator-a": {"shared": {"cluster-api": "https://a", "kubeconfig": "kubeconfig-a"}},
			"operator-b": {"shared": {"cluster-api": "https://b", "kubeconfig": "kubeconfig-b"}},
		},
		map[string][]krknv1alpha1.ClusterTarget{
			"operator-a": {{ClusterName: "shared", ClusterAPIURL: "https://a"}},
			"operator-b": {{ClusterName: "shared", ClusterAPIURL: "https://b"}},
		})
	handler.grpcServerAddr = listener.Addr().String()
	return handler
}

func TestGetClusterResourcesReturnsLimitedPodDataFromSelectedProvider(t *testing.T) {
	output := `{"items":[{"metadata":{"name":"web-0","namespace":"team-a","labels":{"app":"web"},"annotations":{"secret":"hidden"}},"spec":{"containers":[{"name":"main","image":"private-image"}],"nodeName":"worker-0"}}]}`
	stub := &resourceProviderStub{response: &pb.ExecuteKubectlResponse{StdoutBase64: base64.StdEncoding.EncodeToString([]byte(output))}}
	handler := newClusterResourceTestHandler(t, stub)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/cluster-resources?id=target-1&cluster-name=shared&operator-name=operator-b&resource=pods&namespace=team-a", nil)
	response := httptest.NewRecorder()

	handler.GetClusterResources(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", response.Header().Get("Cache-Control"))
	}
	if stub.request == nil || stub.request.KubeconfigBase64 != "kubeconfig-b" || stub.request.Command != "kubectl" || stub.request.Subcommand != "get" || len(stub.request.Args) != 1 || stub.request.Args[0] != "pods" || stub.request.Flags["n"] != "team-a" || stub.request.Flags["o"] != "json" {
		t.Fatalf("unexpected provider request: %+v", stub.request)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	items, ok := body["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %#v", body["items"])
	}
	if strings.Contains(response.Body.String(), "hidden") || strings.Contains(response.Body.String(), "private-image") || strings.Contains(response.Body.String(), "worker-0") {
		t.Fatalf("response exposed unrelated pod data: %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"name":"main"`) || !strings.Contains(response.Body.String(), `"app":"web"`) {
		t.Fatalf("response lost option data: %s", response.Body.String())
	}
}

func TestGetClusterResourcesRejectsInvalidInputBeforeProviderCall(t *testing.T) {
	stub := &resourceProviderStub{response: &pb.ExecuteKubectlResponse{}}
	handler := newClusterResourceTestHandler(t, stub)
	cases := []string{
		"id=target-1&cluster-name=shared&operator-name=operator-a&resource=secrets",
		"id=target-1&cluster-name=shared&operator-name=operator-a&resource=pods&namespace=team-a&all-namespaces=true",
		"id=target-1&cluster-name=shared&operator-name=operator-a&resource=pods&namespace=team-a&all-namespaces=garbage",
		"id=target-1&cluster-name=shared&operator-name=operator-a&resource=pods&namespace=--watch",
		"id=target-1&cluster-name=shared&operator-name=operator-a&resource=nodes&namespace=team-a",
		"id=target-1&cluster-name=shared&operator-name=operator-a&resource=pods",
	}
	for _, query := range cases {
		t.Run(query, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/cluster-resources?"+query, nil)
			response := httptest.NewRecorder()
			handler.GetClusterResources(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
		})
	}
	if stub.request != nil {
		t.Fatalf("invalid input reached provider: %+v", stub.request)
	}
}

func TestGetClusterResourcesMapsProviderFailure(t *testing.T) {
	stub := &resourceProviderStub{response: &pb.ExecuteKubectlResponse{ExitCode: 1, StderrBase64: base64.StdEncoding.EncodeToString([]byte("private cluster detail"))}}
	handler := newClusterResourceTestHandler(t, stub)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/cluster-resources?id=target-1&cluster-name=shared&operator-name=operator-a&resource=nodes", nil)
	response := httptest.NewRecorder()

	handler.GetClusterResources(response, request)

	if response.Code != http.StatusBadGateway || strings.Contains(response.Body.String(), "private cluster detail") {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestGetClusterResourcesUsesAllNamespacesFlag(t *testing.T) {
	stub := &resourceProviderStub{response: &pb.ExecuteKubectlResponse{StdoutBase64: base64.StdEncoding.EncodeToString([]byte(`{"items":[]}`))}}
	handler := newClusterResourceTestHandler(t, stub)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/cluster-resources?id=target-1&cluster-name=shared&operator-name=operator-a&resource=vmi&all-namespaces=true", nil)
	response := httptest.NewRecorder()

	handler.GetClusterResources(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "{\"items\":[]}\n" {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if stub.request == nil || len(stub.request.BooleanFlags) != 1 || stub.request.BooleanFlags[0] != "A" || stub.request.Flags["n"] != "" {
		t.Fatalf("unexpected provider scope: %+v", stub.request)
	}
}

func TestGetClusterResourcesRequiresRunPermission(t *testing.T) {
	stub := &resourceProviderStub{response: &pb.ExecuteKubectlResponse{StdoutBase64: base64.StdEncoding.EncodeToString([]byte(`{"items":[]}`))}}
	handler := newClusterResourceTestHandler(t, stub)
	userName, err := groupauth.SanitizeUserIDForResourceName("viewer@example.com")
	if err != nil {
		t.Fatal(err)
	}
	user := &krknv1alpha1.KrknUser{ObjectMeta: metav1.ObjectMeta{
		Name: userName, Namespace: handler.namespace,
		Labels: map[string]string{groupauth.GroupLabelKey("viewers"): "true"},
	}}
	group := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "viewers", Namespace: handler.namespace},
		Spec: krknv1alpha1.KrknUserGroupSpec{ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
			"https://a": {Actions: []string{"view"}},
		}},
	}
	if err := handler.client.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	if err := handler.client.Create(context.Background(), group); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/cluster-resources?id=target-1&cluster-name=shared&operator-name=operator-a&resource=nodes", nil)
	request = request.WithContext(withTestClaims(request.Context(), "viewer@example.com", "user"))
	response := httptest.NewRecorder()

	handler.GetClusterResources(response, request)

	if response.Code != http.StatusForbidden || stub.request != nil {
		t.Fatalf("status = %d, body = %s, provider request = %+v", response.Code, response.Body.String(), stub.request)
	}
}

func TestGetClusterResourcesRejectsMalformedProviderResponse(t *testing.T) {
	for _, output := range []string{`not-json`, `{}`, `{"items":null}`} {
		t.Run(output, func(t *testing.T) {
			stub := &resourceProviderStub{response: &pb.ExecuteKubectlResponse{StdoutBase64: base64.StdEncoding.EncodeToString([]byte(output))}}
			handler := newClusterResourceTestHandler(t, stub)
			request := httptest.NewRequest(http.MethodGet, "/api/v1/cluster-resources?id=target-1&cluster-name=shared&operator-name=operator-a&resource=nodes", nil)
			response := httptest.NewRecorder()

			handler.GetClusterResources(response, request)

			if response.Code != http.StatusBadGateway || strings.Contains(response.Body.String(), output) {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestGetClusterResourcesMapsRPCFailures(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		code int
	}{
		{name: "deadline", err: status.Error(codes.DeadlineExceeded, "private deadline detail"), code: http.StatusGatewayTimeout},
		{name: "unavailable", err: status.Error(codes.Unavailable, "private provider detail"), code: http.StatusBadGateway},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &resourceProviderStub{err: tc.err}
			handler := newClusterResourceTestHandler(t, stub)
			request := httptest.NewRequest(http.MethodGet, "/api/v1/cluster-resources?id=target-1&cluster-name=shared&operator-name=operator-a&resource=nodes", nil)
			response := httptest.NewRecorder()
			handler.GetClusterResources(response, request)
			if response.Code != tc.code || strings.Contains(response.Body.String(), "private") {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestGetClusterResourcesRejectsUnknownProvider(t *testing.T) {
	stub := &resourceProviderStub{response: &pb.ExecuteKubectlResponse{}}
	handler := newClusterResourceTestHandler(t, stub)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/cluster-resources?id=target-1&cluster-name=shared&operator-name=missing&resource=nodes", nil)
	response := httptest.NewRecorder()
	handler.GetClusterResources(response, request)
	if response.Code != http.StatusNotFound || stub.request != nil {
		t.Fatalf("status = %d, body = %s, provider request = %+v", response.Code, response.Body.String(), stub.request)
	}
}
