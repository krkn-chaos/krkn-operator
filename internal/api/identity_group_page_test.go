package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krkn-chaos/krkn-operator/pkg/identity"
)

type pagedTestDirectory struct{}

func (pagedTestDirectory) Provider() string { return "corp" }
func (pagedTestDirectory) ListGroups(context.Context, string, int, int) ([]identity.Group, error) {
	return nil, nil
}
func (pagedTestDirectory) ListGroupsPage(_ context.Context, _ string, first, _ int) (identity.GroupPage, error) {
	return identity.GroupPage{
		Groups:    []identity.Group{{Provider: "corp", ID: "nested", ClaimValue: "/Root/Nested"}},
		NextFirst: first + 1, HasMore: true,
	}, nil
}

func TestListExternalGroupsReturnsProviderPageCursor(t *testing.T) {
	h := &Handler{groupDirectories: map[string]identity.GroupDirectory{"corp": pagedTestDirectory{}}}
	request := httptest.NewRequest(http.MethodGet, IdentityProvidersPath+"/corp/groups?first=5&max=1", nil)
	request = request.WithContext(createAdminContext())
	response := httptest.NewRecorder()
	h.ListExternalGroups(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("group discovery status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Groups    []identity.Group `json:"groups"`
		NextFirst int              `json:"nextFirst"`
		HasMore   bool             `json:"hasMore"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Groups) != 1 || body.NextFirst != 6 || !body.HasMore {
		t.Fatalf("unexpected group page: %+v", body)
	}
}
