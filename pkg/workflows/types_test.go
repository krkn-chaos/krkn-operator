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

package workflows

import (
	"encoding/json"
	"testing"
)

func TestUpdateWorkflowRequestCategoriesPointerSemantics(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantNil    bool
		wantValues []string
	}{
		{
			name:    "omitted categories preserve existing assignments",
			body:    `{"workflowName":"workflow","graph":{}}`,
			wantNil: true,
		},
		{
			name:       "provided categories are decoded",
			body:       `{"workflowName":"workflow","graph":{},"categories":["graph-runs","other"]}`,
			wantValues: []string{"graph-runs", "other"},
		},
		{
			name:       "explicit empty categories remain non-nil",
			body:       `{"workflowName":"workflow","graph":{},"categories":[]}`,
			wantValues: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var request UpdateWorkflowRequest
			if err := json.Unmarshal([]byte(tt.body), &request); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if request.Categories == nil {
				if !tt.wantNil {
					t.Fatal("Categories is nil, want a non-nil list")
				}
				return
			}
			if tt.wantNil {
				t.Fatal("Categories is non-nil, want omitted field to remain nil")
			}
			if len(*request.Categories) != len(tt.wantValues) {
				t.Fatalf("Categories = %v, want %v", *request.Categories, tt.wantValues)
			}
			for i := range tt.wantValues {
				if (*request.Categories)[i] != tt.wantValues[i] {
					t.Errorf("Categories = %v, want %v", *request.Categories, tt.wantValues)
				}
			}
		})
	}
}
