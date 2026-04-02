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

Assisted-by: Claude Sonnet 4.5 (claude-sonnet-4-5@20250929)
*/

package groupauth

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func TestValidateScenarioRunAccess(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	// Setup common fixtures
	user := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-user-example-com",
			Namespace: "default",
			Labels: map[string]string{
				"group.krkn.krkn-chaos.dev/dev-team": "true",
			},
		},
	}

	devGroup := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dev-team",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: "dev-team",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://api.cluster1.com": {
					Actions: []string{"view", "run"},
				},
				"https://api.cluster2.com": {
					Actions: []string{"view"}, // No run permission
				},
			},
		},
	}

	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-request",
			Namespace: "default",
		},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			TargetData: map[string][]krknv1alpha1.ClusterTarget{
				"provider1": {
					{
						ClusterName:   "cluster1",
						ClusterAPIURL: "https://api.cluster1.com",
					},
					{
						ClusterName:   "cluster2",
						ClusterAPIURL: "https://api.cluster2.com",
					},
				},
			},
		},
	}

	tests := []struct {
		name           string
		targetClusters map[string][]string
		wantErr        bool
		errContains    string
	}{
		{
			name: "user has run permission on requested cluster",
			targetClusters: map[string][]string{
				"provider1": {"cluster1"},
			},
			wantErr: false,
		},
		{
			name: "user lacks run permission on requested cluster",
			targetClusters: map[string][]string{
				"provider1": {"cluster2"},
			},
			wantErr:     true,
			errContains: "does not have permission to run scenarios",
		},
		{
			name: "user has permission on one cluster but not the other",
			targetClusters: map[string][]string{
				"provider1": {"cluster1", "cluster2"},
			},
			wantErr:     true,
			errContains: "does not have permission to run scenarios",
		},
		{
			name: "cluster not found in target request",
			targetClusters: map[string][]string{
				"provider1": {"nonexistent"},
			},
			wantErr:     true,
			errContains: "cluster nonexistent not found in target request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fakeclient.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(user, devGroup).
				Build()

			err := ValidateScenarioRunAccess(
				context.Background(),
				fakeClient,
				"user@example.com",
				"default",
				tt.targetClusters,
				targetRequest,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateScenarioRunAccess() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateScenarioRunAccess() error = %v, should contain %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestValidateScenarioRunAccess_NoGroups(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	// User with no group memberships
	user := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-user-example-com",
			Namespace: "default",
			Labels:    map[string]string{}, // No group labels
		},
	}

	targetRequest := &krknv1alpha1.KrknTargetRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-request",
			Namespace: "default",
		},
		Status: krknv1alpha1.KrknTargetRequestStatus{
			TargetData: map[string][]krknv1alpha1.ClusterTarget{
				"provider1": {
					{
						ClusterName:   "cluster1",
						ClusterAPIURL: "https://api.cluster1.com",
					},
				},
			},
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(user).
		Build()

	err := ValidateScenarioRunAccess(
		context.Background(),
		fakeClient,
		"user@example.com",
		"default",
		map[string][]string{"provider1": {"cluster1"}},
		targetRequest,
	)

	if err == nil {
		t.Error("ValidateScenarioRunAccess() should return error for user with no groups")
	}

	if !contains(err.Error(), "does not belong to any groups") {
		t.Errorf("ValidateScenarioRunAccess() error should mention no groups, got: %v", err)
	}
}

func TestFilterClustersByPermission(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	user := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-user-example-com",
			Namespace: "default",
			Labels: map[string]string{
				"group.krkn.krkn-chaos.dev/dev-team": "true",
			},
		},
	}

	devGroup := &krknv1alpha1.KrknUserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dev-team",
			Namespace: "default",
		},
		Spec: krknv1alpha1.KrknUserGroupSpec{
			Name: "dev-team",
			ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
				"https://api.cluster1.com": {
					Actions: []string{"view", "run"},
				},
				"https://api.cluster2.com": {
					Actions: []string{"view"},
				},
				// cluster3 not in permissions
			},
		},
	}

	tests := []struct {
		name       string
		targetData map[string][]krknv1alpha1.ClusterTarget
		action     Action
		wantCount  int
	}{
		{
			name: "filter clusters with view permission",
			targetData: map[string][]krknv1alpha1.ClusterTarget{
				"provider1": {
					{
						ClusterName:   "cluster1",
						ClusterAPIURL: "https://api.cluster1.com",
					},
					{
						ClusterName:   "cluster2",
						ClusterAPIURL: "https://api.cluster2.com",
					},
					{
						ClusterName:   "cluster3",
						ClusterAPIURL: "https://api.cluster3.com",
					},
				},
			},
			action:    ActionView,
			wantCount: 2, // cluster1 and cluster2 have view
		},
		{
			name: "filter clusters with run permission",
			targetData: map[string][]krknv1alpha1.ClusterTarget{
				"provider1": {
					{
						ClusterName:   "cluster1",
						ClusterAPIURL: "https://api.cluster1.com",
					},
					{
						ClusterName:   "cluster2",
						ClusterAPIURL: "https://api.cluster2.com",
					},
				},
			},
			action:    ActionRun,
			wantCount: 1, // only cluster1 has run
		},
		{
			name: "filter clusters with cancel permission",
			targetData: map[string][]krknv1alpha1.ClusterTarget{
				"provider1": {
					{
						ClusterName:   "cluster1",
						ClusterAPIURL: "https://api.cluster1.com",
					},
				},
			},
			action:    ActionCancel,
			wantCount: 0, // no cluster has cancel
		},
		{
			name:       "empty target data",
			targetData: map[string][]krknv1alpha1.ClusterTarget{},
			action:     ActionView,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fakeclient.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(user, devGroup).
				Build()

			filtered, err := FilterClustersByPermission(
				context.Background(),
				fakeClient,
				"user@example.com",
				"default",
				tt.targetData,
				tt.action,
			)

			if err != nil {
				t.Errorf("FilterClustersByPermission() error = %v", err)
				return
			}

			actualCount := countClustersFromTargetData(filtered)
			if actualCount != tt.wantCount {
				t.Errorf("FilterClustersByPermission() returned %d clusters, want %d", actualCount, tt.wantCount)
			}
		})
	}
}

func TestFilterClustersByPermission_NoGroups(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	user := &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "krknuser-user-example-com",
			Namespace: "default",
			Labels:    map[string]string{}, // No groups
		},
	}

	targetData := map[string][]krknv1alpha1.ClusterTarget{
		"provider1": {
			{
				ClusterName:   "cluster1",
				ClusterAPIURL: "https://api.cluster1.com",
			},
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(user).
		Build()

	filtered, err := FilterClustersByPermission(
		context.Background(),
		fakeClient,
		"user@example.com",
		"default",
		targetData,
		ActionView,
	)

	if err != nil {
		t.Errorf("FilterClustersByPermission() error = %v", err)
		return
	}

	if len(filtered) != 0 {
		t.Errorf("FilterClustersByPermission() should return empty for user with no groups, got %d clusters", len(filtered))
	}
}

func TestBuildClusterAPIURLMap(t *testing.T) {
	targetRequest := &krknv1alpha1.KrknTargetRequest{
		Status: krknv1alpha1.KrknTargetRequestStatus{
			TargetData: map[string][]krknv1alpha1.ClusterTarget{
				"provider1": {
					{
						ClusterName:   "cluster1",
						ClusterAPIURL: "https://api.cluster1.com",
					},
					{
						ClusterName:   "cluster2",
						ClusterAPIURL: "https://api.cluster2.com",
					},
				},
				"provider2": {
					{
						ClusterName:   "cluster3",
						ClusterAPIURL: "https://api.cluster3.com",
					},
				},
			},
		},
	}

	result := buildClusterAPIURLMap(targetRequest)

	expected := map[string]string{
		"cluster1": "https://api.cluster1.com",
		"cluster2": "https://api.cluster2.com",
		"cluster3": "https://api.cluster3.com",
	}

	if len(result) != len(expected) {
		t.Errorf("buildClusterAPIURLMap() returned %d clusters, want %d", len(result), len(expected))
	}

	for clusterName, wantURL := range expected {
		gotURL, exists := result[clusterName]
		if !exists {
			t.Errorf("buildClusterAPIURLMap() missing cluster %s", clusterName)
			continue
		}
		if gotURL != wantURL {
			t.Errorf("buildClusterAPIURLMap() cluster %s = %s, want %s", clusterName, gotURL, wantURL)
		}
	}
}

func TestCountClusters(t *testing.T) {
	tests := []struct {
		name           string
		targetClusters map[string][]string
		want           int
	}{
		{
			name: "multiple providers",
			targetClusters: map[string][]string{
				"provider1": {"cluster1", "cluster2"},
				"provider2": {"cluster3"},
			},
			want: 3,
		},
		{
			name: "single provider",
			targetClusters: map[string][]string{
				"provider1": {"cluster1"},
			},
			want: 1,
		},
		{
			name:           "empty map",
			targetClusters: map[string][]string{},
			want:           0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countClusters(tt.targetClusters)
			if got != tt.want {
				t.Errorf("countClusters() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCountClustersFromTargetData(t *testing.T) {
	tests := []struct {
		name       string
		targetData map[string][]krknv1alpha1.ClusterTarget
		want       int
	}{
		{
			name: "multiple providers",
			targetData: map[string][]krknv1alpha1.ClusterTarget{
				"provider1": {
					{ClusterName: "cluster1"},
					{ClusterName: "cluster2"},
				},
				"provider2": {
					{ClusterName: "cluster3"},
				},
			},
			want: 3,
		},
		{
			name: "single provider",
			targetData: map[string][]krknv1alpha1.ClusterTarget{
				"provider1": {
					{ClusterName: "cluster1"},
				},
			},
			want: 1,
		},
		{
			name:       "empty map",
			targetData: map[string][]krknv1alpha1.ClusterTarget{},
			want:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countClustersFromTargetData(tt.targetData)
			if got != tt.want {
				t.Errorf("countClustersFromTargetData() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestHasAction(t *testing.T) {
	tests := []struct {
		name           string
		actions        []Action
		requiredAction Action
		want           bool
	}{
		{
			name:           "action exists in list",
			actions:        []Action{ActionView, ActionRun, ActionCancel},
			requiredAction: ActionRun,
			want:           true,
		},
		{
			name:           "action does not exist in list",
			actions:        []Action{ActionView, ActionCancel},
			requiredAction: ActionRun,
			want:           false,
		},
		{
			name:           "empty action list",
			actions:        []Action{},
			requiredAction: ActionView,
			want:           false,
		},
		{
			name:           "nil action list",
			actions:        nil,
			requiredAction: ActionView,
			want:           false,
		},
		{
			name:           "single action matches",
			actions:        []Action{ActionView},
			requiredAction: ActionView,
			want:           true,
		},
		{
			name:           "single action does not match",
			actions:        []Action{ActionView},
			requiredAction: ActionRun,
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasAction(tt.actions, tt.requiredAction)
			if got != tt.want {
				t.Errorf("hasAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
