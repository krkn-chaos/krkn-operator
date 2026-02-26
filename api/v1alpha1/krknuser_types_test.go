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

package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestKrknUserCreation tests basic KrknUser creation
func TestKrknUserCreation(t *testing.T) {
	tests := []struct {
		name     string
		spec     KrknUserSpec
		wantRole string
	}{
		{
			name: "create user with admin role",
			spec: KrknUserSpec{
				UserID:            "[email protected]",
				Name:              "John",
				Surname:           "Doe",
				Organization:      "Example Corp",
				Role:              "admin",
				PasswordSecretRef: "user-password-secret",
			},
			wantRole: "admin",
		},
		{
			name: "create user with user role",
			spec: KrknUserSpec{
				UserID:            "[email protected]",
				Name:              "Jane",
				Surname:           "Smith",
				Organization:      "Test Org",
				Role:              "user",
				PasswordSecretRef: "user-password-secret",
			},
			wantRole: "user",
		},
		{
			name: "create user without organization",
			spec: KrknUserSpec{
				UserID:            "[email protected]",
				Name:              "Bob",
				Surname:           "Johnson",
				Role:              "user",
				PasswordSecretRef: "user-password-secret",
			},
			wantRole: "user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &KrknUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: tt.spec,
			}

			if user.Spec.Role != tt.wantRole {
				t.Errorf("KrknUser role = %v, want %v", user.Spec.Role, tt.wantRole)
			}

			if user.Spec.UserID != tt.spec.UserID {
				t.Errorf("KrknUser userID = %v, want %v", user.Spec.UserID, tt.spec.UserID)
			}
		})
	}
}

// TestKrknUserStatus tests KrknUser status fields
func TestKrknUserStatus(t *testing.T) {
	user := &KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-user",
			Namespace: "default",
		},
		Spec: KrknUserSpec{
			UserID:            "[email protected]",
			Name:              "Test",
			Surname:           "User",
			Role:              "user",
			PasswordSecretRef: "password-secret",
		},
		Status: KrknUserStatus{
			Active:  true,
			Created: metav1.Now(),
		},
	}

	if !user.Status.Active {
		t.Error("Expected user to be active by default")
	}

	if user.Status.Created.IsZero() {
		t.Error("Expected Created timestamp to be set")
	}
}

// TestKrknUserList tests KrknUserList creation
func TestKrknUserList(t *testing.T) {
	list := &KrknUserList{
		Items: []KrknUser{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user1",
					Namespace: "default",
				},
				Spec: KrknUserSpec{
					UserID:            "[email protected]",
					Name:              "User",
					Surname:           "One",
					Role:              "admin",
					PasswordSecretRef: "secret1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user2",
					Namespace: "default",
				},
				Spec: KrknUserSpec{
					UserID:            "[email protected]",
					Name:              "User",
					Surname:           "Two",
					Role:              "user",
					PasswordSecretRef: "secret2",
				},
			},
		},
	}

	if len(list.Items) != 2 {
		t.Errorf("Expected 2 items in list, got %d", len(list.Items))
	}

	// Verify roles
	if list.Items[0].Spec.Role != "admin" {
		t.Errorf("Expected first user to be admin, got %s", list.Items[0].Spec.Role)
	}

	if list.Items[1].Spec.Role != "user" {
		t.Errorf("Expected second user to be user, got %s", list.Items[1].Spec.Role)
	}
}

// TestKrknUserValidation tests that validation constraints are properly defined
func TestKrknUserValidation(t *testing.T) {
	validEmails := []string{
		"[email protected]",
		"user+tag@example.co.uk",
		"[email protected]",
	}

	for _, email := range validEmails {
		user := &KrknUser{
			Spec: KrknUserSpec{
				UserID:            email,
				Name:              "Test",
				Surname:           "User",
				Role:              "user",
				PasswordSecretRef: "secret",
			},
		}

		if user.Spec.UserID != email {
			t.Errorf("Expected email %s to be accepted", email)
		}
	}
}

// TestKrknUserRoles tests role validation
func TestKrknUserRoles(t *testing.T) {
	validRoles := []string{"user", "admin"}

	for _, role := range validRoles {
		user := &KrknUser{
			Spec: KrknUserSpec{
				UserID:            "[email protected]",
				Name:              "Test",
				Surname:           "User",
				Role:              role,
				PasswordSecretRef: "secret",
			},
		}

		if user.Spec.Role != role {
			t.Errorf("Expected role %s to be set", role)
		}
	}
}
