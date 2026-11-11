package api

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceAndRollbackOpenAPIContract(t *testing.T) {
	content, err := os.ReadFile("docs/swagger.json")
	require.NoError(t, err)
	var document struct {
		Definitions map[string]struct {
			Enum       []string `json:"enum"`
			Properties map[string]struct {
				Ref      string `json:"$ref"`
				MaxItems *int   `json:"maxItems"`
				AllOf    []struct {
					Ref string `json:"$ref"`
				} `json:"allOf"`
			} `json:"properties"`
		} `json:"definitions"`
	}
	require.NoError(t, json.Unmarshal(content, &document))
	states := document.Definitions["internal_api.RollbackState"].Enum
	require.Equal(t, []string{
		string(RollbackStateChecking), string(RollbackStateAvailable), string(RollbackStateAbsent),
		string(RollbackStateUnavailable), string(RollbackStateRunning), string(RollbackStateSucceeded),
		string(RollbackStateFailed), string(RollbackStateError),
	}, states)
	require.Equal(t, "#/definitions/internal_api.RollbackState", document.Definitions["internal_api.RollbackAvailability"].Properties["state"].Ref)
	for _, name := range []string{"internal_api.workspacePublishRequest", "internal_api.workspaceInfo"} {
		groups := document.Definitions[name].Properties["groups"]
		require.NotNil(t, groups.MaxItems, name)
		require.Equal(t, 1, *groups.MaxItems, name)
	}
	workspace := document.Definitions["internal_api.ScenarioRunRequest"].Properties["workspace"]
	require.Len(t, workspace.AllOf, 1)
	require.Equal(t, "#/definitions/internal_api.WorkspaceSelection", workspace.AllOf[0].Ref)
}
