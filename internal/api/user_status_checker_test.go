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
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func newUserStatusScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add scheme: %v", err)
	}
	return scheme
}

func newKrknUser(userID string, active bool) *krknv1alpha1.KrknUser {
	return &krknv1alpha1.KrknUser{
		ObjectMeta: metav1.ObjectMeta{
			// CRs are created as "krknuser-<sanitized>"; use the same helper as
			// registration so the test reflects the real naming convention.
			Name:      sanitizeUsername(userID),
			Namespace: "krkn-system",
		},
		Spec: krknv1alpha1.KrknUserSpec{
			UserID: userID,
		},
		Status: krknv1alpha1.KrknUserStatus{
			Active: active,
		},
	}
}

func TestK8sUserStatusChecker_IsUserActive(t *testing.T) {
	const namespace = "krkn-system"

	tests := []struct {
		name       string
		userID     string
		objects    []client.Object
		wantActive bool
		wantErr    bool
	}{
		{
			name:       "found and active",
			userID:     "[email protected]",
			objects:    []client.Object{newKrknUser("[email protected]", true)},
			wantActive: true,
		},
		{
			name:       "found and inactive",
			userID:     "[email protected]",
			objects:    []client.Object{newKrknUser("[email protected]", false)},
			wantActive: false,
		},
		{
			name:       "not found treated as inactive",
			userID:     "[email protected]",
			objects:    nil,
			wantActive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(newUserStatusScheme(t)).
				WithObjects(tt.objects...).
				Build()

			checker := newK8sUserStatusChecker(fakeClient, namespace)
			active, err := checker.IsUserActive(context.Background(), tt.userID)

			if tt.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if active != tt.wantActive {
				t.Errorf("IsUserActive() = %v, want %v", active, tt.wantActive)
			}
		})
	}
}

// TestK8sUserStatusChecker_LookupErrorPropagates verifies that a non-NotFound
// error from the Kubernetes API is propagated (so the middleware fails closed)
// rather than being swallowed as "inactive".
func TestK8sUserStatusChecker_LookupErrorPropagates(t *testing.T) {
	apiErr := errors.New("k8s API unavailable")

	fakeClient := fake.NewClientBuilder().
		WithScheme(newUserStatusScheme(t)).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
				return apiErr
			},
		}).
		Build()

	checker := newK8sUserStatusChecker(fakeClient, "krkn-system")
	active, err := checker.IsUserActive(context.Background(), "[email protected]")

	if err == nil {
		t.Fatal("expected error to propagate, got nil")
	}
	if !errors.Is(err, apiErr) {
		t.Errorf("expected wrapped API error, got %v", err)
	}
	if active {
		t.Error("expected active=false when lookup fails")
	}
}
