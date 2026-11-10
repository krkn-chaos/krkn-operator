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

package v1alpha1

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// TestKrknRobotDeepCopy verifies DeepCopy returns an equal but distinct object.
func TestKrknRobotDeepCopy(t *testing.T) {
	now := metav1.NewTime(time.Now().UTC().Truncate(time.Second))
	robot := &KrknRobot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ci-bot",
			Namespace: "krkn-operator",
		},
		Spec: KrknRobotSpec{
			Description: "CI automation",
		},
		Status: KrknRobotStatus{
			TokenID:        "abcd1234",
			TokenIssuedAt:  &now,
			TokenExpiresAt: &now,
		},
	}

	copied := robot.DeepCopy()
	if copied == robot {
		t.Fatal("DeepCopy returned the same pointer")
	}
	if !robot.Status.TokenExpiresAt.Equal(copied.Status.TokenExpiresAt) {
		t.Errorf("TokenExpiresAt mismatch: %v vs %v", robot.Status.TokenExpiresAt, copied.Status.TokenExpiresAt)
	}
	if copied.Status.TokenExpiresAt == robot.Status.TokenExpiresAt {
		t.Error("TokenExpiresAt pointers are shared, want distinct")
	}
	if copied.Spec.Description != robot.Spec.Description {
		t.Errorf("Description mismatch: %q vs %q", copied.Spec.Description, robot.Spec.Description)
	}
}

// TestKrknRobotRegistered verifies the type is registered in the scheme.
func TestKrknRobotRegistered(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme failed: %v", err)
	}
	gvk := GroupVersion.WithKind("KrknRobot")
	obj, err := scheme.New(gvk)
	if err != nil {
		t.Fatalf("scheme.New(%v) failed: %v", gvk, err)
	}
	if _, ok := obj.(*KrknRobot); !ok {
		t.Errorf("scheme.New(%v) returned %T, want *KrknRobot", gvk, obj)
	}
	listGVK := GroupVersion.WithKind("KrknRobotList")
	listObj, err := scheme.New(listGVK)
	if err != nil {
		t.Fatalf("scheme.New(%v) failed: %v", listGVK, err)
	}
	if _, ok := listObj.(*KrknRobotList); !ok {
		t.Errorf("scheme.New(%v) returned %T, want *KrknRobotList", listGVK, listObj)
	}
}
