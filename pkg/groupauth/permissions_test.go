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

func TestGetUserGroups(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	tests := []struct {
		name      string
		userID    string
		user      *krknv1alpha1.KrknUser
		groups    []krknv1alpha1.KrknUserGroup
		wantCount int
		wantErr   bool
	}{
		{
			name:   "user with single group",
			userID: "user@example.com",
			user: &krknv1alpha1.KrknUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "krknuser-user-example-com",
					Namespace: "default",
					Labels: map[string]string{
						"group.krkn.krkn-chaos.dev/dev-team": "true",
					},
				},
			},
			groups: []krknv1alpha1.KrknUserGroup{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dev-team",
						Namespace: "default",
					},
					Spec: krknv1alpha1.KrknUserGroupSpec{
						Name: "dev-team",
					},
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:   "user with multiple groups",
			userID: "user@example.com",
			user: &krknv1alpha1.KrknUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "krknuser-user-example-com",
					Namespace: "default",
					Labels: map[string]string{
						"group.krkn.krkn-chaos.dev/dev-team":  "true",
						"group.krkn.krkn-chaos.dev/ops-team":  "true",
						"group.krkn.krkn-chaos.dev/prod-team": "true",
					},
				},
			},
			groups: []krknv1alpha1.KrknUserGroup{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dev-team",
						Namespace: "default",
					},
					Spec: krknv1alpha1.KrknUserGroupSpec{
						Name: "dev-team",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ops-team",
						Namespace: "default",
					},
					Spec: krknv1alpha1.KrknUserGroupSpec{
						Name: "ops-team",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "prod-team",
						Namespace: "default",
					},
					Spec: krknv1alpha1.KrknUserGroupSpec{
						Name: "prod-team",
					},
				},
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:   "user with no groups",
			userID: "user@example.com",
			user: &krknv1alpha1.KrknUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "krknuser-user-example-com",
					Namespace: "default",
					Labels:    map[string]string{},
				},
			},
			groups:    []krknv1alpha1.KrknUserGroup{},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "user not found",
			userID:    "nonexistent@example.com",
			user:      nil,
			groups:    []krknv1alpha1.KrknUserGroup{},
			wantCount: 0,
			wantErr:   true,
		},
		{
			name:   "user with deleted group",
			userID: "user@example.com",
			user: &krknv1alpha1.KrknUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "krknuser-user-example-com",
					Namespace: "default",
					Labels: map[string]string{
						"group.krkn.krkn-chaos.dev/dev-team":     "true",
						"group.krkn.krkn-chaos.dev/deleted-team": "true",
					},
				},
			},
			groups: []krknv1alpha1.KrknUserGroup{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dev-team",
						Namespace: "default",
					},
					Spec: krknv1alpha1.KrknUserGroupSpec{
						Name: "dev-team",
					},
				},
				// deleted-team is not in the groups list
			},
			wantCount: 1, // Only dev-team should be returned
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := []runtime.Object{}
			if tt.user != nil {
				objects = append(objects, tt.user)
			}
			for i := range tt.groups {
				objects = append(objects, &tt.groups[i])
			}

			fakeClient := fakeclient.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objects...).
				Build()

			got, err := GetUserGroups(context.Background(), fakeClient, tt.userID, "default")

			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserGroups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantCount {
				t.Errorf("GetUserGroups() returned %d groups, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestAggregateClusterPermissions(t *testing.T) {
	tests := []struct {
		name       string
		userGroups []krknv1alpha1.KrknUserGroup
		want       map[string][]Action
	}{
		{
			name: "single group with single cluster",
			userGroups: []krknv1alpha1.KrknUserGroup{
				{
					Spec: krknv1alpha1.KrknUserGroupSpec{
						ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
							"https://api.cluster1.com": {
								Actions: []string{"view", "run"},
							},
						},
					},
				},
			},
			want: map[string][]Action{
				"https://api.cluster1.com": {ActionView, ActionRun},
			},
		},
		{
			name: "multiple groups with same cluster (union)",
			userGroups: []krknv1alpha1.KrknUserGroup{
				{
					Spec: krknv1alpha1.KrknUserGroupSpec{
						ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
							"https://api.cluster1.com": {
								Actions: []string{"view"},
							},
						},
					},
				},
				{
					Spec: krknv1alpha1.KrknUserGroupSpec{
						ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
							"https://api.cluster1.com": {
								Actions: []string{"run", "cancel"},
							},
						},
					},
				},
			},
			want: map[string][]Action{
				"https://api.cluster1.com": {ActionView, ActionRun, ActionCancel},
			},
		},
		{
			name: "multiple groups with different clusters",
			userGroups: []krknv1alpha1.KrknUserGroup{
				{
					Spec: krknv1alpha1.KrknUserGroupSpec{
						ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
							"https://api.cluster1.com": {
								Actions: []string{"view", "run"},
							},
						},
					},
				},
				{
					Spec: krknv1alpha1.KrknUserGroupSpec{
						ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
							"https://api.cluster2.com": {
								Actions: []string{"view"},
							},
						},
					},
				},
			},
			want: map[string][]Action{
				"https://api.cluster1.com": {ActionView, ActionRun},
				"https://api.cluster2.com": {ActionView},
			},
		},
		{
			name:       "no groups",
			userGroups: []krknv1alpha1.KrknUserGroup{},
			want:       map[string][]Action{},
		},
		{
			name: "group with no permissions",
			userGroups: []krknv1alpha1.KrknUserGroup{
				{
					Spec: krknv1alpha1.KrknUserGroupSpec{
						ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{},
					},
				},
			},
			want: map[string][]Action{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AggregateClusterPermissions(tt.userGroups)

			// Check if all expected clusters are present
			if len(got) != len(tt.want) {
				t.Errorf("AggregateClusterPermissions() returned %d clusters, want %d", len(got), len(tt.want))
			}

			for clusterURL, wantActions := range tt.want {
				gotActions, ok := got[clusterURL]
				if !ok {
					t.Errorf("AggregateClusterPermissions() missing cluster %s", clusterURL)
					continue
				}

				// Convert to maps for comparison (order doesn't matter)
				gotMap := make(map[Action]bool)
				for _, a := range gotActions {
					gotMap[a] = true
				}
				wantMap := make(map[Action]bool)
				for _, a := range wantActions {
					wantMap[a] = true
				}

				if len(gotMap) != len(wantMap) {
					t.Errorf("AggregateClusterPermissions() cluster %s has %d actions, want %d", clusterURL, len(gotActions), len(wantActions))
					continue
				}

				for action := range wantMap {
					if !gotMap[action] {
						t.Errorf("AggregateClusterPermissions() cluster %s missing action %s", clusterURL, action)
					}
				}
			}
		})
	}
}

func TestCanPerformAction(t *testing.T) {
	userGroups := []krknv1alpha1.KrknUserGroup{
		{
			Spec: krknv1alpha1.KrknUserGroupSpec{
				ClusterPermissions: map[string]krknv1alpha1.ClusterPermissionSet{
					"https://api.cluster1.com": {
						Actions: []string{"view", "run"},
					},
					"https://api.cluster2.com": {
						Actions: []string{"view"},
					},
				},
			},
		},
	}

	tests := []struct {
		name          string
		clusterAPIURL string
		action        Action
		want          bool
	}{
		{
			name:          "allowed view on cluster1",
			clusterAPIURL: "https://api.cluster1.com",
			action:        ActionView,
			want:          true,
		},
		{
			name:          "allowed run on cluster1",
			clusterAPIURL: "https://api.cluster1.com",
			action:        ActionRun,
			want:          true,
		},
		{
			name:          "denied cancel on cluster1",
			clusterAPIURL: "https://api.cluster1.com",
			action:        ActionCancel,
			want:          false,
		},
		{
			name:          "allowed view on cluster2",
			clusterAPIURL: "https://api.cluster2.com",
			action:        ActionView,
			want:          true,
		},
		{
			name:          "denied run on cluster2",
			clusterAPIURL: "https://api.cluster2.com",
			action:        ActionRun,
			want:          false,
		},
		{
			name:          "denied on unknown cluster",
			clusterAPIURL: "https://api.cluster3.com",
			action:        ActionView,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanPerformAction(userGroups, tt.clusterAPIURL, tt.action)
			if got != tt.want {
				t.Errorf("CanPerformAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCountGroupMembers(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)

	tests := []struct {
		name      string
		groupName string
		users     []krknv1alpha1.KrknUser
		want      int
		wantErr   bool
	}{
		{
			name:      "group with 2 members",
			groupName: "dev-team",
			users: []krknv1alpha1.KrknUser{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "krknuser-user1-example-com",
						Namespace: "default",
						Labels: map[string]string{
							"group.krkn.krkn-chaos.dev/dev-team": "true",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "krknuser-user2-example-com",
						Namespace: "default",
						Labels: map[string]string{
							"group.krkn.krkn-chaos.dev/dev-team": "true",
						},
					},
				},
			},
			want:    2,
			wantErr: false,
		},
		{
			name:      "group with no members",
			groupName: "empty-team",
			users:     []krknv1alpha1.KrknUser{},
			want:      0,
			wantErr:   false,
		},
		{
			name:      "group with members in different groups",
			groupName: "dev-team",
			users: []krknv1alpha1.KrknUser{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "krknuser-user1-example-com",
						Namespace: "default",
						Labels: map[string]string{
							"group.krkn.krkn-chaos.dev/dev-team": "true",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "krknuser-user2-example-com",
						Namespace: "default",
						Labels: map[string]string{
							"group.krkn.krkn-chaos.dev/ops-team": "true",
						},
					},
				},
			},
			want:    1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := make([]runtime.Object, len(tt.users))
			for i := range tt.users {
				objects[i] = &tt.users[i]
			}

			fakeClient := fakeclient.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objects...).
				Build()

			got, err := CountGroupMembers(context.Background(), fakeClient, tt.groupName, "default")

			if (err != nil) != tt.wantErr {
				t.Errorf("CountGroupMembers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("CountGroupMembers() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSanitizeUserID(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "standard email",
			email: "user@example.com",
			want:  "krknuser-user-example-com",
		},
		{
			name:  "email with subdomain",
			email: "user@mail.example.com",
			want:  "krknuser-user-mail-example-com",
		},
		{
			name:  "uppercase email",
			email: "User@Example.Com",
			want:  "krknuser-user-example-com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeUserID(tt.email)
			if got != tt.want {
				t.Errorf("sanitizeUserID() = %q, want %q", got, tt.want)
			}
		})
	}
}
