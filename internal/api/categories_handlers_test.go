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
	"net/http/httptest"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/api/v2"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

func newCategoryTestHandler(t *testing.T, objects ...runtime.Object) (*Handler, *runtime.Scheme) {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("register API scheme: %v", err)
	}
	defaultUserName, err := groupauth.SanitizeUserIDForResourceName("category-user@example.com")
	if err != nil {
		t.Fatalf("sanitize default test user ID: %v", err)
	}
	hasDefaultUser := false
	for _, object := range objects {
		if user, ok := object.(*krknv1alpha1.KrknUser); ok && user.Name == defaultUserName && user.Namespace == "default" {
			hasDefaultUser = true
			break
		}
	}
	if !hasDefaultUser {
		objects = append(objects, &krknv1alpha1.KrknUser{
			ObjectMeta: metav1.ObjectMeta{Name: defaultUserName, Namespace: "default"},
			Spec:       krknv1alpha1.KrknUserSpec{UserID: "category-user@example.com", Role: "user"},
		})
	}

	k8sClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(objects...).
		Build()

	return &Handler{client: k8sClient, namespace: "default"}, scheme
}

func newCategoryRequest(method, path, body, role string) *http.Request {
	return newCategoryRequestAs(method, path, body, role, "category-user@example.com")
}

func newCategoryRequestAs(method, path, body, role, userID string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	claims := &auth.Claims{UserID: userID, Role: role}
	ctx := context.WithValue(req.Context(), auth.UserClaimsKey, claims)
	return req.WithContext(ctx)
}

func newCategoryTestUser(t *testing.T, userID string, groupNames ...string) *krknv1alpha1.KrknUser {
	t.Helper()
	name, err := groupauth.SanitizeUserIDForResourceName(userID)
	if err != nil {
		t.Fatalf("sanitize test user ID %q: %v", userID, err)
	}
	labels := make(map[string]string, len(groupNames))
	for _, groupName := range groupNames {
		labels[groupauth.GroupLabelKey(groupName)] = "true"
	}
	return &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", Labels: labels},
		Spec:       krknv1alpha1.KrknUserSpec{UserID: userID, Role: "user"},
	}
}

func newCategoryTestGroup(groupName string) *krknv1alpha1.KrknUserGroup {
	return &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{Name: groupName, Namespace: "default"},
		Spec:       krknv1alpha1.KrknUserGroupSpec{Name: groupName},
	}
}

func TestCategoriesRouterCRUD(t *testing.T) {
	handler, scheme := newCategoryTestHandler(t)
	adminRole := "admin"

	createResponse := httptest.NewRecorder()
	handler.CategoriesRouter(createResponse, newCategoryRequest(
		http.MethodPost,
		v2.CategoriesPath,
		`{"name":"network-chaos","color":"#FF5733","availableToAll":true}`,
		adminRole,
	))
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", createResponse.Code, http.StatusCreated, createResponse.Body.String())
	}
	var created CategoryResponse
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Name != "network-chaos" || created.Color != "#FF5733" || !created.AvailableToAll {
		t.Fatalf("create response = %+v, want category name and color", created)
	}

	duplicateResponse := httptest.NewRecorder()
	handler.CategoriesRouter(duplicateResponse, newCategoryRequest(
		http.MethodPost,
		v2.CategoriesPath,
		`{"name":"network-chaos","availableToAll":true}`,
		adminRole,
	))
	if duplicateResponse.Code != http.StatusConflict {
		t.Fatalf("duplicate create status = %d, want %d: %s", duplicateResponse.Code, http.StatusConflict, duplicateResponse.Body.String())
	}

	listResponse := httptest.NewRecorder()
	handler.CategoriesRouter(listResponse, newCategoryRequest(http.MethodGet, v2.CategoriesPath, "", "user"))
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d: %s", listResponse.Code, http.StatusOK, listResponse.Body.String())
	}
	var listed CategoryListResponse
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if listed.Total != 1 || len(listed.Categories) != 1 || listed.Categories[0].Name != "network-chaos" {
		t.Fatalf("list response = %+v, want one category", listed)
	}

	getResponse := httptest.NewRecorder()
	handler.CategoriesRouter(getResponse, newCategoryRequest(http.MethodGet, v2.CategoriesPath+"/network-chaos", "", "user"))
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d: %s", getResponse.Code, http.StatusOK, getResponse.Body.String())
	}

	updateResponse := httptest.NewRecorder()
	handler.CategoriesRouter(updateResponse, newCategoryRequest(
		http.MethodPut,
		v2.CategoriesPath+"/network-chaos",
		`{"color":"#3366FF"}`,
		adminRole,
	))
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d: %s", updateResponse.Code, http.StatusOK, updateResponse.Body.String())
	}
	var updated CategoryResponse
	if err := json.Unmarshal(updateResponse.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated.Color != "#3366FF" {
		t.Fatalf("updated color = %q, want %q", updated.Color, "#3366FF")
	}

	deleteResponse := httptest.NewRecorder()
	handler.CategoriesRouter(deleteResponse, newCategoryRequest(http.MethodDelete, v2.CategoriesPath+"/network-chaos", "", adminRole))
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d: %s", deleteResponse.Code, http.StatusOK, deleteResponse.Body.String())
	}

	missingResponse := httptest.NewRecorder()
	handler.CategoriesRouter(missingResponse, newCategoryRequest(http.MethodGet, v2.CategoriesPath+"/network-chaos", "", "user"))
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("get deleted category status = %d, want %d: %s", missingResponse.Code, http.StatusNotFound, missingResponse.Body.String())
	}

	category := &krknv1alpha1.KrknCategory{}
	gkinds, _, err := scheme.ObjectKinds(category)
	if err != nil {
		t.Fatalf("get registered category kinds: %v", err)
	}
	found := false
	for _, gvk := range gkinds {
		if gvk == krknv1alpha1.GroupVersion.WithKind("KrknCategory") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("registered kinds = %v, want %s", gkinds, krknv1alpha1.GroupVersion.WithKind("KrknCategory"))
	}
}

func TestCategoriesRouterUserCanManageOwnCategory(t *testing.T) {
	handler, _ := newCategoryTestHandler(t, newCategoryTestUser(t, "other@example.com"))

	createResponse := httptest.NewRecorder()
	handler.CategoriesRouter(createResponse, newCategoryRequest(
		http.MethodPost,
		v2.CategoriesPath,
		`{"name":"owned-category"}`,
		"user",
	))
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("user create status = %d, want %d: %s", createResponse.Code, http.StatusCreated, createResponse.Body.String())
	}
	var created CategoryResponse
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.CreatedBy != "category-user@example.com" || !created.AvailableToAll {
		t.Fatalf("created category = %+v, want creator and public visibility", created)
	}

	updateResponse := httptest.NewRecorder()
	handler.CategoriesRouter(updateResponse, newCategoryRequest(
		http.MethodPut,
		v2.CategoriesPath+"/owned-category",
		`{"color":"#AABBCC"}`,
		"user",
	))
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("owner update status = %d, want %d: %s", updateResponse.Code, http.StatusOK, updateResponse.Body.String())
	}

	otherUserUpdate := httptest.NewRecorder()
	handler.CategoriesRouter(otherUserUpdate, newCategoryRequestAs(
		http.MethodPut,
		v2.CategoriesPath+"/owned-category",
		`{"color":"#123456"}`,
		"user",
		"other@example.com",
	))
	if otherUserUpdate.Code != http.StatusForbidden {
		t.Fatalf("non-owner update status = %d, want %d: %s", otherUserUpdate.Code, http.StatusForbidden, otherUserUpdate.Body.String())
	}

	otherUserDelete := httptest.NewRecorder()
	handler.CategoriesRouter(otherUserDelete, newCategoryRequestAs(
		http.MethodDelete,
		v2.CategoriesPath+"/owned-category",
		"",
		"user",
		"other@example.com",
	))
	if otherUserDelete.Code != http.StatusForbidden {
		t.Fatalf("non-owner delete status = %d, want %d: %s", otherUserDelete.Code, http.StatusForbidden, otherUserDelete.Body.String())
	}

	deleteResponse := httptest.NewRecorder()
	handler.CategoriesRouter(deleteResponse, newCategoryRequest(http.MethodDelete, v2.CategoriesPath+"/owned-category", "", "user"))
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("owner delete status = %d, want %d: %s", deleteResponse.Code, http.StatusOK, deleteResponse.Body.String())
	}
}

func TestCategoriesRouterGroupVisibility(t *testing.T) {
	groupA := newCategoryTestGroup("team-a")
	groupB := newCategoryTestGroup("team-b")
	userA := newCategoryTestUser(t, "member-a@example.com", "team-a")
	userB := newCategoryTestUser(t, "member-b@example.com", "team-b")
	publicCategory := &krknv1alpha1.KrknCategory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "public-category",
			Namespace: "default",
			Labels:    map[string]string{categoryAvailableToAllLabel: "true"},
		},
	}
	legacyCategory := &krknv1alpha1.KrknCategory{
		ObjectMeta: metav1.ObjectMeta{Name: "legacy-category", Namespace: "default"},
	}
	categoryA := &krknv1alpha1.KrknCategory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "team-a-category",
			Namespace: "default",
			Labels:    map[string]string{groupauth.GroupLabelKey("team-a"): "true"},
		},
	}
	categoryB := &krknv1alpha1.KrknCategory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "team-b-category",
			Namespace: "default",
			Labels:    map[string]string{groupauth.GroupLabelKey("team-b"): "true"},
		},
	}
	handler, _ := newCategoryTestHandler(t, groupA, groupB, userA, userB, publicCategory, legacyCategory, categoryA, categoryB)

	listResponse := httptest.NewRecorder()
	handler.CategoriesRouter(listResponse, newCategoryRequestAs(http.MethodGet, v2.CategoriesPath, "", "user", "member-a@example.com"))
	if listResponse.Code != http.StatusOK {
		t.Fatalf("group list status = %d, want %d: %s", listResponse.Code, http.StatusOK, listResponse.Body.String())
	}
	var listed CategoryListResponse
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode group list response: %v", err)
	}
	if listed.Total != 3 || listed.Categories[0].Name != "legacy-category" || listed.Categories[1].Name != "public-category" || listed.Categories[2].Name != "team-a-category" {
		t.Fatalf("group list = %+v, want legacy public, public, and team-a categories only", listed)
	}

	groupGet := httptest.NewRecorder()
	handler.CategoriesRouter(groupGet, newCategoryRequestAs(http.MethodGet, v2.CategoriesPath+"/team-a-category", "", "user", "member-a@example.com"))
	if groupGet.Code != http.StatusOK {
		t.Fatalf("group category get status = %d, want %d: %s", groupGet.Code, http.StatusOK, groupGet.Body.String())
	}

	otherGroupGet := httptest.NewRecorder()
	handler.CategoriesRouter(otherGroupGet, newCategoryRequestAs(http.MethodGet, v2.CategoriesPath+"/team-b-category", "", "user", "member-a@example.com"))
	if otherGroupGet.Code != http.StatusForbidden {
		t.Fatalf("other group get status = %d, want %d: %s", otherGroupGet.Code, http.StatusForbidden, otherGroupGet.Body.String())
	}

	adminList := httptest.NewRecorder()
	handler.CategoriesRouter(adminList, newCategoryRequest(http.MethodGet, v2.CategoriesPath, "", "admin"))
	var allCategories CategoryListResponse
	if adminList.Code != http.StatusOK {
		t.Fatalf("admin list status = %d, want %d: %s", adminList.Code, http.StatusOK, adminList.Body.String())
	}
	if err := json.Unmarshal(adminList.Body.Bytes(), &allCategories); err != nil {
		t.Fatalf("decode admin list response: %v", err)
	}
	if allCategories.Total != 4 {
		t.Fatalf("admin list total = %d, want 4", allCategories.Total)
	}
}

func TestCategoriesRouterGroupCreationRequiresMembership(t *testing.T) {
	groupA := newCategoryTestGroup("team-a")
	groupB := newCategoryTestGroup("team-b")
	member := newCategoryTestUser(t, "member@example.com", "team-a")
	handler, _ := newCategoryTestHandler(t, groupA, groupB, member)

	createOwnGroup := httptest.NewRecorder()
	handler.CategoriesRouter(createOwnGroup, newCategoryRequestAs(
		http.MethodPost,
		v2.CategoriesPath,
		`{"name":"team-category","groups":["team-a"]}`,
		"user",
		"member@example.com",
	))
	if createOwnGroup.Code != http.StatusCreated {
		t.Fatalf("own group create status = %d, want %d: %s", createOwnGroup.Code, http.StatusCreated, createOwnGroup.Body.String())
	}
	var created CategoryResponse
	if err := json.Unmarshal(createOwnGroup.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode group create response: %v", err)
	}
	if created.AvailableToAll || len(created.Groups) != 1 || created.Groups[0] != "team-a" {
		t.Fatalf("created group category = %+v, want team-a-only visibility", created)
	}

	changeToPublic := httptest.NewRecorder()
	handler.CategoriesRouter(changeToPublic, newCategoryRequestAs(
		http.MethodPut,
		v2.CategoriesPath+"/team-category",
		`{"groups":[],"availableToAll":true}`,
		"user",
		"member@example.com",
	))
	if changeToPublic.Code != http.StatusOK {
		t.Fatalf("owner visibility update status = %d, want %d: %s", changeToPublic.Code, http.StatusOK, changeToPublic.Body.String())
	}

	createOtherGroup := httptest.NewRecorder()
	handler.CategoriesRouter(createOtherGroup, newCategoryRequestAs(
		http.MethodPost,
		v2.CategoriesPath,
		`{"name":"other-team-category","groups":["team-b"]}`,
		"user",
		"member@example.com",
	))
	if createOtherGroup.Code != http.StatusForbidden {
		t.Fatalf("other group create status = %d, want %d: %s", createOtherGroup.Code, http.StatusForbidden, createOtherGroup.Body.String())
	}

	createContradictoryVisibility := httptest.NewRecorder()
	handler.CategoriesRouter(createContradictoryVisibility, newCategoryRequestAs(
		http.MethodPost,
		v2.CategoriesPath,
		`{"name":"contradictory-visibility","groups":["team-a"],"availableToAll":true}`,
		"user",
		"member@example.com",
	))
	if createContradictoryVisibility.Code != http.StatusBadRequest {
		t.Fatalf("contradictory visibility create status = %d, want %d: %s", createContradictoryVisibility.Code, http.StatusBadRequest, createContradictoryVisibility.Body.String())
	}

	createMissingGroup := httptest.NewRecorder()
	handler.CategoriesRouter(createMissingGroup, newCategoryRequestAs(
		http.MethodPost,
		v2.CategoriesPath,
		`{"name":"missing-group-category","groups":["team-missing"]}`,
		"admin",
		"admin@example.com",
	))
	if createMissingGroup.Code != http.StatusBadRequest {
		t.Fatalf("missing group create status = %d, want %d: %s", createMissingGroup.Code, http.StatusBadRequest, createMissingGroup.Body.String())
	}
}

func TestCategoriesRouterValidationAndMethods(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "missing name", body: `{"color":"#123456"}`},
		{name: "invalid name", body: `{"name":"Invalid_Name"}`},
		{name: "invalid color", body: `{"name":"valid-name","color":"red"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _ := newCategoryTestHandler(t)
			response := httptest.NewRecorder()
			handler.CategoriesRouter(response, newCategoryRequest(http.MethodPost, v2.CategoriesPath, tt.body, "admin"))
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
			}
		})
	}

	handler, _ := newCategoryTestHandler(t)
	rootMethodResponse := httptest.NewRecorder()
	handler.CategoriesRouter(rootMethodResponse, newCategoryRequest(http.MethodPut, v2.CategoriesPath, `{}`, "admin"))
	if rootMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("root method status = %d, want %d", rootMethodResponse.Code, http.StatusMethodNotAllowed)
	}

	itemMethodResponse := httptest.NewRecorder()
	handler.CategoriesRouter(itemMethodResponse, newCategoryRequest(http.MethodPost, v2.CategoriesPath+"/name", `{}`, "admin"))
	if itemMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("item method status = %d, want %d", itemMethodResponse.Code, http.StatusMethodNotAllowed)
	}

	invalidPathResponse := httptest.NewRecorder()
	handler.CategoriesRouter(invalidPathResponse, newCategoryRequest(http.MethodGet, v2.CategoriesPath+"/nested/name", "", "user"))
	if invalidPathResponse.Code != http.StatusBadRequest {
		t.Fatalf("nested item path status = %d, want %d", invalidPathResponse.Code, http.StatusBadRequest)
	}
}

func TestCategoriesRouterNameMaximumLength(t *testing.T) {
	handler, _ := newCategoryTestHandler(t)
	maxLengthName := strings.Repeat("a", maxCategoryNameLength)
	createAtLimitResponse := httptest.NewRecorder()
	handler.CategoriesRouter(createAtLimitResponse, newCategoryRequest(
		http.MethodPost,
		v2.CategoriesPath,
		`{"name":"`+maxLengthName+`","availableToAll":true}`,
		"admin",
	))
	if createAtLimitResponse.Code != http.StatusCreated {
		t.Fatalf("create at maximum name length status = %d, want %d: %s", createAtLimitResponse.Code, http.StatusCreated, createAtLimitResponse.Body.String())
	}

	tooLongName := strings.Repeat("a", maxCategoryNameLength+1)
	tooLongBody, err := json.Marshal(CreateCategoryRequest{Name: tooLongName})
	if err != nil {
		t.Fatalf("marshal overlong category name: %v", err)
	}
	requests := []struct {
		name   string
		method string
		path   string
		body   string
		role   string
	}{
		{name: "create", method: http.MethodPost, path: v2.CategoriesPath, body: string(tooLongBody), role: "admin"},
		{name: "get", method: http.MethodGet, path: v2.CategoriesPath + "/" + tooLongName, role: "user"},
		{name: "update", method: http.MethodPut, path: v2.CategoriesPath + "/" + tooLongName, body: `{}`, role: "admin"},
		{name: "delete", method: http.MethodDelete, path: v2.CategoriesPath + "/" + tooLongName, role: "admin"},
	}
	for _, tt := range requests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.CategoriesRouter(response, newCategoryRequest(tt.method, tt.path, tt.body, tt.role))
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
			}
		})
	}
}

func TestKrknCategoryDeepCopy(t *testing.T) {
	category := &krknv1alpha1.KrknCategory{
		ObjectMeta: metav1.ObjectMeta{Name: "network-chaos", Namespace: "default"},
		Spec:       krknv1alpha1.KrknCategorySpec{Color: "#FF5733"},
	}
	copy := category.DeepCopy()
	if copy == category || copy.Name != category.Name || copy.Spec.Color != category.Spec.Color {
		t.Fatalf("DeepCopy() = %#v, want a distinct category with the same data", copy)
	}
	copy.Spec.Color = "#3366FF"
	if category.Spec.Color != "#FF5733" {
		t.Fatalf("mutating the copy changed the original color to %q", category.Spec.Color)
	}

	var object runtime.Object = category
	if _, ok := object.DeepCopyObject().(*krknv1alpha1.KrknCategory); !ok {
		t.Fatalf("DeepCopyObject() returned %T, want *KrknCategory", object.DeepCopyObject())
	}
}
