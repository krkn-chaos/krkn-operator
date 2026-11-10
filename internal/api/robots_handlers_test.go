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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

// newRobotsTestHandler builds a Handler with a fake client for robot admin API tests.
func newRobotsTestHandler(t *testing.T, objs ...runtime.Object) (*Handler, client.Client) {
	t.Helper()
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	c := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(objs...).
		WithStatusSubresource(&krknv1alpha1.KrknRobot{}).
		Build()

	h := NewTestHandler(c, fake.NewSimpleClientset(), "default", "localhost:50051")
	return h, c
}

func createRobotUserContext(role string) context.Context {
	claims := &auth.Claims{UserID: "user1@test.local", Role: role, Name: "User"}
	return context.WithValue(context.Background(), auth.UserClaimsKey, claims)
}

func doRobotsRequest(t *testing.T, h *Handler, method, path string, body any, ctx context.Context) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.RobotsRouter(w, req)
	return w
}

func sreGroupFixture() *krknv1alpha1.KrknUserGroup {
	return &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "sre", Namespace: "default"},
		Spec:       krknv1alpha1.KrknUserGroupSpec{Name: "sre"},
	}
}

func TestRobotsRouter_NonAdminForbidden(t *testing.T) {
	h, _ := newRobotsTestHandler(t, sreGroupFixture())
	admin := createAdminContext()
	user := createRobotUserContext("user")

	cases := []struct{ method, path string }{
		{http.MethodGet, RobotsPath},
		{http.MethodPost, RobotsPath},
		{http.MethodGet, RobotsPath + "/ci-bot"},
		{http.MethodPatch, RobotsPath + "/ci-bot"},
		{http.MethodDelete, RobotsPath + "/ci-bot"},
		{http.MethodPost, RobotsPath + "/ci-bot/token"},
		{http.MethodDelete, RobotsPath + "/ci-bot/token"},
	}
	for _, tc := range cases {
		w := doRobotsRequest(t, h, tc.method, tc.path, nil, user)
		if w.Code != http.StatusForbidden {
			t.Errorf("%s %s as user: status %d, want 403", tc.method, tc.path, w.Code)
		}
		if !strings.Contains(w.Body.String(), "forbidden") {
			t.Errorf("%s %s as user: body %s lacks forbidden", tc.method, tc.path, w.Body.String())
		}
		// Admin passes the gate (may 404 for missing robot, but never 403).
		w = doRobotsRequest(t, h, tc.method, tc.path, nil, admin)
		if w.Code == http.StatusForbidden {
			t.Errorf("%s %s as admin: unexpectedly 403", tc.method, tc.path)
		}
	}
}

func TestCreateRobot_Valid(t *testing.T) {
	h, c := newRobotsTestHandler(t, sreGroupFixture())
	body := CreateRobotRequest{Name: "ci-bot", Description: "CI", Groups: []string{"sre"}}
	w := doRobotsRequest(t, h, http.MethodPost, RobotsPath, body, createAdminContext())
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d, want 201. Body: %s", w.Code, w.Body.String())
	}
	robot := &krknv1alpha1.KrknRobot{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "ci-bot", Namespace: "default"}, robot); err != nil {
		t.Fatalf("robot CR not created: %v", err)
	}
	if robot.Labels[krknv1alpha1.RobotAccountLabel] != "true" {
		t.Error("robot-account label missing")
	}
	if robot.Labels["group.krkn.krkn-chaos.dev/sre"] != "true" {
		t.Error("group label missing")
	}
}

func TestCreateRobot_InvalidName(t *testing.T) {
	h, _ := newRobotsTestHandler(t, sreGroupFixture())
	for _, name := range []string{"", "Bad", "a_b", strings.Repeat("a", 49)} {
		body := CreateRobotRequest{Name: name, Groups: []string{"sre"}}
		w := doRobotsRequest(t, h, http.MethodPost, RobotsPath, body, createAdminContext())
		if w.Code != http.StatusBadRequest {
			t.Errorf("name %q: status %d, want 400", name, w.Code)
		}
	}
}

func TestCreateRobot_UnknownGroup(t *testing.T) {
	h, _ := newRobotsTestHandler(t, sreGroupFixture())
	body := CreateRobotRequest{Name: "ci-bot", Groups: []string{"nope"}}
	w := doRobotsRequest(t, h, http.MethodPost, RobotsPath, body, createAdminContext())
	if w.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreateRobot_Duplicate(t *testing.T) {
	h, _ := newRobotsTestHandler(t, sreGroupFixture(), robotFixture("ci-bot", false))
	body := CreateRobotRequest{Name: "ci-bot"}
	w := doRobotsRequest(t, h, http.MethodPost, RobotsPath, body, createAdminContext())
	if w.Code != http.StatusConflict {
		t.Errorf("status %d, want 409", w.Code)
	}
}

func TestUpdateRobot_ReplacesGroups(t *testing.T) {
	ops := &krknv1alpha1.KrknUserGroup{ObjectMeta: metav1.ObjectMeta{Name: "ops", Namespace: "default"}, Spec: krknv1alpha1.KrknUserGroupSpec{Name: "ops"}}
	robot := robotFixture("ci-bot", false)
	robot.Labels = map[string]string{"group.krkn.krkn-chaos.dev/sre": "true"}
	h, c := newRobotsTestHandler(t, sreGroupFixture(), ops, robot)

	groups := []string{"ops"}
	body := UpdateRobotRequest{Groups: &groups}
	w := doRobotsRequest(t, h, http.MethodPatch, RobotsPath+"/ci-bot", body, createAdminContext())
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, want 200. Body: %s", w.Code, w.Body.String())
	}
	updated := &krknv1alpha1.KrknRobot{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "ci-bot", Namespace: "default"}, updated); err != nil {
		t.Fatal(err)
	}
	if _, ok := updated.Labels["group.krkn.krkn-chaos.dev/sre"]; ok {
		t.Error("sre label still present")
	}
	if updated.Labels["group.krkn.krkn-chaos.dev/ops"] != "true" {
		t.Error("ops label missing")
	}
}

func TestUpdateRobot_Disable(t *testing.T) {
	h, c := newRobotsTestHandler(t, robotFixture("ci-bot", false))
	_ = c
	disabled := true
	body := UpdateRobotRequest{Disabled: &disabled}
	w := doRobotsRequest(t, h, http.MethodPatch, RobotsPath+"/ci-bot", body, createAdminContext())
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, want 200. Body: %s", w.Code, w.Body.String())
	}
	updated := &krknv1alpha1.KrknRobot{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "ci-bot", Namespace: "default"}, updated); err != nil {
		t.Fatal(err)
	}
	if !updated.Spec.Disabled {
		t.Error("disabled not persisted")
	}
}

func TestIssueRobotToken_ReturnsTokenOnce(t *testing.T) {
	h, _ := newRobotsTestHandler(t, robotFixture("ci-bot", false))
	w := doRobotsRequest(t, h, http.MethodPost, RobotsPath+"/ci-bot/token", nil, createAdminContext())
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d, want 201. Body: %s", w.Code, w.Body.String())
	}
	var resp IssueRobotTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("token empty in response")
	}

	w = doRobotsRequest(t, h, http.MethodGet, RobotsPath+"/ci-bot", nil, createAdminContext())
	if w.Code != http.StatusOK {
		t.Fatalf("GET robot status %d", w.Code)
	}
	if strings.Contains(w.Body.String(), resp.Token) {
		t.Error("GET robot response leaks the token")
	}
}

func TestIssueRobotToken_Disabled(t *testing.T) {
	h, _ := newRobotsTestHandler(t, robotFixture("ci-bot", true))
	w := doRobotsRequest(t, h, http.MethodPost, RobotsPath+"/ci-bot/token", nil, createAdminContext())
	if w.Code != http.StatusConflict {
		t.Errorf("status %d, want 409", w.Code)
	}
}

func TestRevokeRobotToken(t *testing.T) {
	h, c := newRobotsTestHandler(t, robotFixture("ci-bot", false))
	if _, err := h.robotTokens.Issue(context.Background(), "ci-bot"); err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	w := doRobotsRequest(t, h, http.MethodDelete, RobotsPath+"/ci-bot/token", nil, createAdminContext())
	if w.Code != http.StatusNoContent {
		t.Fatalf("status %d, want 204", w.Code)
	}
	secret := &corev1.Secret{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "krknrobot-ci-bot-token", Namespace: "default"}, secret); err == nil {
		t.Error("token Secret still exists after revoke")
	}
	w = doRobotsRequest(t, h, http.MethodGet, RobotsPath+"/ci-bot", nil, createAdminContext())
	var resp RobotResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.TokenID != "" {
		t.Errorf("tokenId = %q, want empty", resp.TokenID)
	}
}

func TestDeleteRobot_RemovesTokenSecret(t *testing.T) {
	h, c := newRobotsTestHandler(t, robotFixture("ci-bot", false))
	if _, err := h.robotTokens.Issue(context.Background(), "ci-bot"); err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	w := doRobotsRequest(t, h, http.MethodDelete, RobotsPath+"/ci-bot", nil, createAdminContext())
	if w.Code != http.StatusNoContent {
		t.Fatalf("status %d, want 204", w.Code)
	}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "ci-bot", Namespace: "default"}, &krknv1alpha1.KrknRobot{}); err == nil {
		t.Error("robot CR still exists")
	}
	secret := &corev1.Secret{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "krknrobot-ci-bot-token", Namespace: "default"}, secret); err == nil {
		t.Error("token Secret still exists")
	}
}

func TestRobotsRoute_RequiresJWT(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	c := fakeclient.NewClientBuilder().WithScheme(scheme).Build()

	secretManager, err := auth.NewTestSecretManager(c, "default")
	if err != nil {
		t.Fatalf("test secret manager: %v", err)
	}
	srv := NewServer(0, c, fake.NewSimpleClientset(), "default", "localhost:50051", secretManager)

	req := httptest.NewRequest(http.MethodGet, RobotsPath, nil)
	w := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status %d, want 401 (route must require JWT)", w.Code)
	}
}
