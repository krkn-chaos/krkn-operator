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

package controller

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

func TestGetOwnerLabel_Robot(t *testing.T) {
	robotRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "r1"},
		Spec:       krknv1alpha1.KrknScenarioRunSpec{OwnerUserID: "robot:ci-bot"},
	}
	if got := getOwnerLabel(robotRun); got != "robot.ci-bot" {
		t.Errorf("getOwnerLabel(robot:ci-bot) = %q, want robot.ci-bot", got)
	}

	humanRun := &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: "r2"},
		Spec:       krknv1alpha1.KrknScenarioRunSpec{OwnerUserID: "a.b@c.io"},
	}
	if got := getOwnerLabel(humanRun); got != "a-b-c-io" {
		t.Errorf("getOwnerLabel(a.b@c.io) = %q, want a-b-c-io (legacy behavior)", got)
	}
}
