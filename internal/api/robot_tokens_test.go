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
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// newRobotTokenTestService builds a robotTokenService over a fake client with
// deterministic now/random for testing.
func newRobotTokenTestService(t *testing.T, robots ...runtime.Object) (*robotTokenService, client.Client, *time.Time) {
	t.Helper()
	scheme := runtime.NewScheme()
	_ = krknv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	c := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(robots...).
		WithStatusSubresource(&krknv1alpha1.KrknRobot{}).
		Build()

	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	svc := newRobotTokenService(c, "default")
	svc.now = func() time.Time { return now }
	svc.random = rand.Reader
	return svc, c, &now
}

func robotFixture(name string, disabled bool) *krknv1alpha1.KrknRobot {
	return robotFixtureWithUID(name, "uid-"+name, disabled)
}

// robotFixtureWithUID builds a robot with an explicit UID, for tests that
// exercise UID binding across robot recreation.
func robotFixtureWithUID(name, uid string, disabled bool) *krknv1alpha1.KrknRobot {
	return &krknv1alpha1.KrknRobot{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", UID: types.UID(uid)},
		Spec:       krknv1alpha1.KrknRobotSpec{Disabled: disabled},
	}
}

func TestIssue_ReturnsTokenOnceAndStoresOnlyHash(t *testing.T) {
	svc, c, now := newRobotTokenTestService(t, robotFixture("ci-bot", false))

	issued, err := svc.Issue(context.Background(), "ci-bot")
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	if !strings.HasPrefix(issued.Token, "krkn_robot_ci-bot.") {
		t.Errorf("token %q lacks expected prefix", issued.Token)
	}
	if len(issued.TokenID) != 8 {
		t.Errorf("tokenId %q is not 4 bytes hex", issued.TokenID)
	}
	if !issued.ExpiresAt.Equal(now.Add(90 * 24 * time.Hour)) {
		t.Errorf("ExpiresAt = %v, want %v", issued.ExpiresAt, now.Add(90*24*time.Hour))
	}

	secret := &corev1.Secret{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "krknrobot-ci-bot-token", Namespace: "default"}, secret); err != nil {
		t.Fatalf("token Secret not created: %v", err)
	}
	for key := range secret.Data {
		if strings.Contains(string(secret.Data[key]), issued.Token) {
			t.Errorf("Secret key %q contains the raw token", key)
		}
	}
	sum := sha256.Sum256([]byte(issued.Token))
	if string(secret.Data["tokenHash"]) != hex.EncodeToString(sum[:]) {
		t.Errorf("tokenHash != hex(sha256(token))")
	}
}

func TestIssue_DisabledRobot(t *testing.T) {
	svc, _, _ := newRobotTokenTestService(t, robotFixture("ci-bot", true))
	_, err := svc.Issue(context.Background(), "ci-bot")
	if !errors.Is(err, errRobotDisabled) {
		t.Errorf("err = %v, want errRobotDisabled", err)
	}
}

func TestIssue_MissingRobot(t *testing.T) {
	svc, _, _ := newRobotTokenTestService(t)
	_, err := svc.Issue(context.Background(), "ci-bot")
	if err == nil || !apierrors.IsNotFound(errors.Unwrap(err)) {
		t.Errorf("err = %v, want wrapped NotFound", err)
	}
}

func TestVerify_Valid(t *testing.T) {
	svc, _, _ := newRobotTokenTestService(t, robotFixture("ci-bot", false))
	issued, err := svc.Issue(context.Background(), "ci-bot")
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	id, err := svc.Verify(context.Background(), issued.Token)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if id.Name != "ci-bot" {
		t.Errorf("name = %q, want ci-bot", id.Name)
	}
	if !id.ExpiresAt.Equal(issued.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", id.ExpiresAt, issued.ExpiresAt)
	}
}

func TestVerify_RotatedTokenInvalidatesOld(t *testing.T) {
	svc, _, _ := newRobotTokenTestService(t, robotFixture("ci-bot", false))
	first, err := svc.Issue(context.Background(), "ci-bot")
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	second, err := svc.Issue(context.Background(), "ci-bot")
	if err != nil {
		t.Fatalf("Issue (rotate) failed: %v", err)
	}
	if _, err := svc.Verify(context.Background(), first.Token); !errors.Is(err, errRobotTokenInvalid) {
		t.Errorf("old token err = %v, want errRobotTokenInvalid", err)
	}
	if _, err := svc.Verify(context.Background(), second.Token); err != nil {
		t.Errorf("new token err = %v, want nil", err)
	}
}

func TestVerify_Revoked(t *testing.T) {
	svc, _, _ := newRobotTokenTestService(t, robotFixture("ci-bot", false))
	issued, err := svc.Issue(context.Background(), "ci-bot")
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	if err := svc.Revoke(context.Background(), "ci-bot"); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}
	if _, err := svc.Verify(context.Background(), issued.Token); !errors.Is(err, errRobotTokenInvalid) {
		t.Errorf("err = %v, want errRobotTokenInvalid", err)
	}
}

func TestVerify_Expired(t *testing.T) {
	svc, _, now := newRobotTokenTestService(t, robotFixture("ci-bot", false))
	issued, err := svc.Issue(context.Background(), "ci-bot")
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	*now = now.Add(91 * 24 * time.Hour)
	if _, err := svc.Verify(context.Background(), issued.Token); !errors.Is(err, errRobotTokenInvalid) {
		t.Errorf("err = %v, want errRobotTokenInvalid", err)
	}
}

func TestVerify_Disabled(t *testing.T) {
	svc, c, _ := newRobotTokenTestService(t, robotFixture("ci-bot", false))
	issued, err := svc.Issue(context.Background(), "ci-bot")
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	robot := &krknv1alpha1.KrknRobot{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "ci-bot", Namespace: "default"}, robot); err != nil {
		t.Fatal(err)
	}
	robot.Spec.Disabled = true
	if err := c.Update(context.Background(), robot); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Verify(context.Background(), issued.Token); !errors.Is(err, errRobotTokenInvalid) {
		t.Errorf("err = %v, want errRobotTokenInvalid", err)
	}
}

func TestVerify_RobotDeleted(t *testing.T) {
	svc, c, _ := newRobotTokenTestService(t, robotFixture("ci-bot", false))
	issued, err := svc.Issue(context.Background(), "ci-bot")
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	if err := c.Delete(context.Background(), &krknv1alpha1.KrknRobot{ObjectMeta: metav1.ObjectMeta{Name: "ci-bot", Namespace: "default"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Verify(context.Background(), issued.Token); !errors.Is(err, errRobotTokenInvalid) {
		t.Errorf("err = %v, want errRobotTokenInvalid", err)
	}
}

// TestVerify_RobotRecreatedSameName covers the UID-binding guarantee: a robot
// deleted and recreated with the same name gets a new UID, and tokens issued
// to the old incarnation must fail verification even if the old token Secret
// still exists (e.g. before garbage collection).
func TestVerify_RobotRecreatedSameName(t *testing.T) {
	svc, c, _ := newRobotTokenTestService(t, robotFixtureWithUID("ci-bot", "uid-old", false))
	issued, err := svc.Issue(context.Background(), "ci-bot")
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	// Delete the robot but keep the stale token Secret (as if GC has not run
	// yet or the Secret was recreated with the old hash).
	if err := c.Delete(context.Background(), &krknv1alpha1.KrknRobot{ObjectMeta: metav1.ObjectMeta{Name: "ci-bot", Namespace: "default"}}); err != nil {
		t.Fatal(err)
	}
	recreated := robotFixtureWithUID("ci-bot", "uid-new", false)
	recreated.ResourceVersion = ""
	if err := c.Create(context.Background(), recreated); err != nil {
		t.Fatalf("recreating robot failed: %v", err)
	}

	// The old token must not authenticate the new incarnation.
	if _, err := svc.Verify(context.Background(), issued.Token); !errors.Is(err, errRobotTokenInvalid) {
		t.Errorf("old token err = %v, want errRobotTokenInvalid", err)
	}

	// A freshly issued token for the new incarnation must work.
	fresh, err := svc.Issue(context.Background(), "ci-bot")
	if err != nil {
		t.Fatalf("re-issue failed: %v", err)
	}
	if _, err := svc.Verify(context.Background(), fresh.Token); err != nil {
		t.Errorf("fresh token Verify err = %v, want nil", err)
	}
}

func TestVerify_Malformed(t *testing.T) {
	svc, _, _ := newRobotTokenTestService(t, robotFixture("ci-bot", false))
	for _, token := range []string{
		"",
		"krkn_robot_",
		"krkn_robot_ci-bot",
		"krkn_robot_BAD.x",
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ4In0.sig",
	} {
		if _, err := svc.Verify(context.Background(), token); !errors.Is(err, errRobotTokenInvalid) {
			t.Errorf("Verify(%q) err = %v, want errRobotTokenInvalid", token, err)
		}
	}
}

func TestVerify_WrongRobotInToken(t *testing.T) {
	svc, _, _ := newRobotTokenTestService(t, robotFixture("a-bot", false), robotFixture("b-bot", false))
	issued, err := svc.Issue(context.Background(), "a-bot")
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	// Swap the name segment so the secret no longer matches robot b-bot.
	forged := "krkn_robot_b-bot." + strings.SplitN(issued.Token, ".", 2)[1]
	if _, err := svc.Verify(context.Background(), forged); !errors.Is(err, errRobotTokenInvalid) {
		t.Errorf("err = %v, want errRobotTokenInvalid", err)
	}
}

func TestVerify_UpdatesLastUsedThrottled(t *testing.T) {
	svc, c, now := newRobotTokenTestService(t, robotFixture("ci-bot", false))
	issued, err := svc.Issue(context.Background(), "ci-bot")
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	if _, err := svc.Verify(context.Background(), issued.Token); err != nil {
		t.Fatalf("first Verify failed: %v", err)
	}
	robot := &krknv1alpha1.KrknRobot{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "ci-bot", Namespace: "default"}, robot); err != nil {
		t.Fatal(err)
	}
	first := robot.Status.LastUsedAt
	if first == nil || !first.Time.Equal(*now) {
		t.Fatalf("LastUsedAt = %v, want %v", first, *now)
	}
	*now = now.Add(time.Minute)
	if _, err := svc.Verify(context.Background(), issued.Token); err != nil {
		t.Fatalf("second Verify failed: %v", err)
	}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "ci-bot", Namespace: "default"}, robot); err != nil {
		t.Fatal(err)
	}
	if !robot.Status.LastUsedAt.Time.Equal(first.Time) {
		t.Errorf("LastUsedAt changed within throttle window: %v vs %v", robot.Status.LastUsedAt, first)
	}
}

func TestRevoke_Idempotent(t *testing.T) {
	svc, _, _ := newRobotTokenTestService(t, robotFixture("ci-bot", false))
	if err := svc.Revoke(context.Background(), "ci-bot"); err != nil {
		t.Fatalf("first Revoke failed: %v", err)
	}
	if err := svc.Revoke(context.Background(), "ci-bot"); err != nil {
		t.Fatalf("second Revoke failed: %v", err)
	}
}
