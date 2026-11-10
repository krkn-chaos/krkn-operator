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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

// newMCPServerTest builds a Server and fake client wired for MCP tests.
func newMCPServerTest(t *testing.T, objs ...runtime.Object) *Server {
	t.Helper()
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	c := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(objs...).
		WithStatusSubresource(&krknv1alpha1.KrknRobot{}).
		Build()

	secretManager, err := auth.NewTestSecretManager(c, "default")
	if err != nil {
		t.Fatalf("test secret manager: %v", err)
	}
	return NewServer(0, c, fake.NewSimpleClientset(), "default", "localhost:50051", secretManager)
}

func TestServer_MCPDisabledByDefault(t *testing.T) {
	srv := newMCPServerTest(t)

	req := httptest.NewRequest(http.MethodPost, MCPPath, strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (MCP disabled by default)", w.Code)
	}
}

func TestServer_MCPEnabled(t *testing.T) {
	t.Setenv(MCPEnabledEnv, "true")
	srv := newMCPServerTest(t)

	req := httptest.NewRequest(http.MethodPost, MCPPath, strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (no auth)", w.Code)
	}
}

func TestServer_RobotTokenRejectedOnREST(t *testing.T) {
	srv := newMCPServerTest(t, robotFixture(robotNameCIBot, false))
	handler := srv.handler

	issued, err := handler.robotTokens.Issue(context.Background(), "ci-bot")
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, HealthPath, nil)
	req.Header.Set("Authorization", "Bearer "+issued.Token)
	w := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (opaque robot token must not pass REST auth)", w.Code)
	}
}

const robotNameCIBot = "ci-bot"

func TestMCPBackend_WhoAmI(t *testing.T) {
	group := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "sre", Namespace: "default"},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: "sre",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://api.a:6443": {Actions: []string{"view", "run"}},
			},
		},
	}
	robot := robotFixture(robotNameCIBot, false)
	robot.Labels = map[string]string{"group.krkn.krkn-chaos.dev/sre": "true"}

	srv := newMCPServerTest(t, group, robot)
	backend := &mcpBackend{h: srv.handler}

	out, err := backend.WhoAmI(context.Background(), mcpPrincipal(robotNameCIBot))
	if err != nil {
		t.Fatalf("WhoAmI failed: %v", err)
	}
	if out.RobotName != robotNameCIBot {
		t.Errorf("RobotName = %q", out.RobotName)
	}
	if len(out.Groups) != 1 || out.Groups[0] != "sre" {
		t.Errorf("Groups = %v, want [sre]", out.Groups)
	}
	if len(out.Permissions) != 1 {
		t.Fatalf("Permissions = %+v, want 1 entry", out.Permissions)
	}
	perm := out.Permissions[0]
	if perm.ClusterAPIURL != "https://api.a:6443" {
		t.Errorf("ClusterAPIURL = %q", perm.ClusterAPIURL)
	}
	if len(perm.Actions) != 2 || perm.Actions[0] != "run" || perm.Actions[1] != "view" {
		t.Errorf("Actions = %v, want [run view] sorted", perm.Actions)
	}
}
