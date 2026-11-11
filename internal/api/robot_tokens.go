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
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/pkg/auth"
)

const (
	// robotTokenPrefix is the prefix of every robot bearer token.
	robotTokenPrefix = "krkn_robot_"
	// robotTokenTTL is how long an issued robot token remains valid.
	robotTokenTTL = 90 * 24 * time.Hour
	// robotLastUsedRefresh throttles LastUsedAt status updates.
	robotLastUsedRefresh = 5 * time.Minute
	// maxTokenStatusAttempts bounds retries when concurrent status writers
	// (LastUsedAt refreshes from in-flight verifications, or a competing
	// rotation) conflict with a token status write.
	maxTokenStatusAttempts = 3
)

var (
	errRobotTokenInvalid = errors.New("invalid robot token")
	errRobotDisabled     = errors.New("robot is disabled")
	// errRobotTokenSuperseded signals that a concurrent rotation replaced the
	// token this attempt issued; the caller retries with a fresh token.
	errRobotTokenSuperseded = errors.New("robot token superseded by concurrent rotation")
)

// RobotIdentity identifies an authenticated robot principal.
type RobotIdentity struct {
	Name      string
	ExpiresAt time.Time
}

// IssuedRobotToken is the one-time result of issuing a robot token.
type IssuedRobotToken struct {
	Token     string
	TokenID   string
	ExpiresAt time.Time
}

// robotTokenService issues, verifies and revokes robot bearer tokens.
// Only the SHA-256 hash of a token is persisted; the raw value is returned
// exactly once, at issuance.
type robotTokenService struct {
	client    client.Client
	namespace string
	now       func() time.Time
	random    io.Reader
}

func newRobotTokenService(c client.Client, ns string) *robotTokenService {
	return &robotTokenService{
		client:    c,
		namespace: ns,
		now:       time.Now,
		random:    rand.Reader,
	}
}

// robotTokenSecretName returns the name of a robot's token Secret.
func robotTokenSecretName(robotName string) string {
	return "krknrobot-" + robotName + "-token"
}

// Issue creates a new token for the robot, replacing any previous one.
// Rotation invalidates the old token immediately (the Secret is replaced
// first), so Issue must not fail after that point merely because a concurrent
// status writer (e.g. an in-flight Verify refreshing LastUsedAt, or a
// competing rotation) bumped the robot's resourceVersion: the admin would get
// an error while the old token is already dead. Status writes therefore
// re-read and retry on conflict, and Issue verifies the final state is
// consistent (status.tokenId matches the Secret's tokenId) before returning
// success, retrying the whole rotation if it was superseded.
func (s *robotTokenService) Issue(ctx context.Context, robotName string) (IssuedRobotToken, error) {
	for attempt := 0; attempt < maxTokenStatusAttempts; attempt++ {
		issued, err := s.issueAttempt(ctx, robotName)
		if err == nil {
			return issued, nil
		}
		if !errors.Is(err, errRobotTokenSuperseded) {
			return IssuedRobotToken{}, err
		}
	}
	return IssuedRobotToken{}, fmt.Errorf("robot %q: token rotation kept being superseded by concurrent rotations: %w", robotName, errRobotTokenSuperseded)
}

// issueAttempt performs one rotation attempt: generate a fresh token, replace
// the Secret, record it on the robot status, and verify the result is still
// the active token. It returns an error wrapping errRobotTokenSuperseded when
// a concurrent rotation won the race.
func (s *robotTokenService) issueAttempt(ctx context.Context, robotName string) (IssuedRobotToken, error) {
	robot := &krknv1alpha1.KrknRobot{}
	if err := s.client.Get(ctx, client.ObjectKey{Name: robotName, Namespace: s.namespace}, robot); err != nil {
		if apierrors.IsNotFound(err) {
			return IssuedRobotToken{}, fmt.Errorf("robot %q: %w", robotName, err)
		}
		return IssuedRobotToken{}, fmt.Errorf("failed to get robot %q: %w", robotName, err)
	}
	if robot.Spec.Disabled {
		return IssuedRobotToken{}, errRobotDisabled
	}

	secret32 := make([]byte, 32)
	if _, err := io.ReadFull(s.random, secret32); err != nil {
		return IssuedRobotToken{}, fmt.Errorf("failed to generate token secret: %w", err)
	}
	tokenIDBytes := make([]byte, 4)
	if _, err := io.ReadFull(s.random, tokenIDBytes); err != nil {
		return IssuedRobotToken{}, fmt.Errorf("failed to generate token id: %w", err)
	}

	now := s.now()
	token := robotTokenPrefix + robotName + "." + base64.RawURLEncoding.EncodeToString(secret32)
	tokenID := hex.EncodeToString(tokenIDBytes)
	expiresAt := now.Add(robotTokenTTL)

	if err := s.upsertTokenSecret(ctx, robot, token, tokenID, expiresAt); err != nil {
		return IssuedRobotToken{}, err
	}
	if err := s.updateRobotTokenStatus(ctx, robotName, tokenID, now, expiresAt); err != nil {
		return IssuedRobotToken{}, err
	}

	active, err := s.tokenIsActive(ctx, robotName, tokenID)
	if err != nil {
		return IssuedRobotToken{}, err
	}
	if !active {
		return IssuedRobotToken{}, fmt.Errorf("robot %q: %w", robotName, errRobotTokenSuperseded)
	}

	return IssuedRobotToken{Token: token, TokenID: tokenID, ExpiresAt: expiresAt}, nil
}

// Revoke deletes the robot's token Secret and clears its token status.
// It is idempotent, and robust to concurrent status writers: the status clear
// re-reads and retries on conflict.
func (s *robotTokenService) Revoke(ctx context.Context, robotName string) error {
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: robotTokenSecretName(robotName), Namespace: s.namespace}}
	if err := s.client.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete token secret: %w", err)
	}

	if err := s.updateRobotStatusWithRetry(ctx, robotName, func(status *krknv1alpha1.KrknRobotStatus) {
		status.TokenID = ""
		status.TokenIssuedAt = nil
		status.TokenExpiresAt = nil
	}); err != nil {
		// The robot may have been deleted concurrently; revoke stays
		// idempotent.
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

// tokenIsActive reports whether tokenID is still the robot's active token:
// both the robot status and the token Secret must point at it.
func (s *robotTokenService) tokenIsActive(ctx context.Context, robotName, tokenID string) (bool, error) {
	robot := &krknv1alpha1.KrknRobot{}
	if err := s.client.Get(ctx, client.ObjectKey{Name: robotName, Namespace: s.namespace}, robot); err != nil {
		return false, fmt.Errorf("failed to get robot %q: %w", robotName, err)
	}
	if robot.Status.TokenID != tokenID {
		return false, nil
	}
	secretObj := &corev1.Secret{}
	if err := s.client.Get(ctx, client.ObjectKey{Name: robotTokenSecretName(robotName), Namespace: s.namespace}, secretObj); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get token secret: %w", err)
	}
	return string(secretObj.Data["tokenId"]) == tokenID, nil
}

// Verify validates a bearer token and returns the robot identity. Every
// authentication failure returns errRobotTokenInvalid; infrastructure errors
// return a wrapped non-sentinel error.
func (s *robotTokenService) Verify(ctx context.Context, token string) (RobotIdentity, error) {
	robotName, err := parseRobotToken(token)
	if err != nil {
		return RobotIdentity{}, err
	}

	robot := &krknv1alpha1.KrknRobot{}
	if err := s.client.Get(ctx, client.ObjectKey{Name: robotName, Namespace: s.namespace}, robot); err != nil {
		if apierrors.IsNotFound(err) {
			return RobotIdentity{}, errRobotTokenInvalid
		}
		return RobotIdentity{}, fmt.Errorf("failed to get robot %q: %w", robotName, err)
	}
	if robot.Spec.Disabled {
		return RobotIdentity{}, errRobotTokenInvalid
	}

	identity, err := s.verifyTokenSecret(ctx, robot, token)
	if err != nil {
		return RobotIdentity{}, err
	}

	s.maybeUpdateLastUsed(ctx, robot)
	return identity, nil
}

// verifyTokenSecret loads the token Secret and compares the hash. The token
// is bound to the robot's immutable UID as well as its name: if a robot is
// deleted and recreated with the same name, tokens issued to the old
// incarnation keep failing verification even before Kubernetes garbage-collects
// the old Secret.
func (s *robotTokenService) verifyTokenSecret(ctx context.Context, robot *krknv1alpha1.KrknRobot, token string) (RobotIdentity, error) {
	robotName := robot.Name
	secretObj := &corev1.Secret{}
	if err := s.client.Get(ctx, client.ObjectKey{Name: robotTokenSecretName(robotName), Namespace: s.namespace}, secretObj); err != nil {
		if apierrors.IsNotFound(err) {
			return RobotIdentity{}, errRobotTokenInvalid
		}
		return RobotIdentity{}, fmt.Errorf("failed to get token secret: %w", err)
	}
	if secretObj.Labels[krknv1alpha1.RobotNameLabel] != robotName {
		return RobotIdentity{}, errRobotTokenInvalid
	}
	if subtle.ConstantTimeCompare(secretObj.Data["robotUID"], []byte(robot.UID)) != 1 {
		return RobotIdentity{}, errRobotTokenInvalid
	}

	expiresAt, err := time.Parse(time.RFC3339, string(secretObj.Data["expiresAt"]))
	if err != nil {
		return RobotIdentity{}, errRobotTokenInvalid
	}
	if !s.now().Before(expiresAt) {
		return RobotIdentity{}, errRobotTokenInvalid
	}

	sum := sha256.Sum256([]byte(token))
	expectedHash := hex.EncodeToString(sum[:])
	if subtle.ConstantTimeCompare([]byte(expectedHash), secretObj.Data["tokenHash"]) != 1 {
		return RobotIdentity{}, errRobotTokenInvalid
	}

	return RobotIdentity{Name: robotName, ExpiresAt: expiresAt}, nil
}

// maybeUpdateLastUsed refreshes Status.LastUsedAt at most once per window.
// A failure here is logged and does not fail verification.
func (s *robotTokenService) maybeUpdateLastUsed(ctx context.Context, robot *krknv1alpha1.KrknRobot) {
	now := s.now()
	if robot.Status.LastUsedAt != nil && now.Sub(robot.Status.LastUsedAt.Time) < robotLastUsedRefresh {
		return
	}
	updated := robot.DeepCopy()
	updated.Status.LastUsedAt = &metav1.Time{Time: now.UTC()}
	if err := s.client.Status().Patch(ctx, updated, client.MergeFrom(robot.DeepCopy())); err != nil {
		log.FromContext(ctx).V(1).Info("Failed to update robot LastUsedAt", "robot", robot.Name, "error", err.Error())
	}
}

// upsertTokenSecret creates or replaces the robot's token Secret.
func (s *robotTokenService) upsertTokenSecret(ctx context.Context, robot *krknv1alpha1.KrknRobot, token, tokenID string, expiresAt time.Time) error {
	sum := sha256.Sum256([]byte(token))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      robotTokenSecretName(robot.Name),
			Namespace: s.namespace,
			Labels: map[string]string{
				krknv1alpha1.RobotTokenLabel: "true",
				krknv1alpha1.RobotNameLabel:  robot.Name,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"tokenHash": []byte(hex.EncodeToString(sum[:])),
			"tokenId":   []byte(tokenID),
			"expiresAt": []byte(expiresAt.UTC().Format(time.RFC3339)),
			// robotUID binds the token to this specific robot incarnation, so
			// a recreated robot with the same name never inherits credentials.
			"robotUID": []byte(robot.UID),
		},
	}
	if err := controllerutil.SetOwnerReference(robot, secret, s.client.Scheme()); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	existing := &corev1.Secret{}
	err := s.client.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, existing)
	switch {
	case apierrors.IsNotFound(err):
		if err := s.client.Create(ctx, secret); err != nil {
			return fmt.Errorf("failed to create token secret: %w", err)
		}
	case err != nil:
		return fmt.Errorf("failed to get token secret: %w", err)
	default:
		updated := existing.DeepCopy()
		updated.Labels = secret.Labels
		updated.Data = secret.Data
		updated.OwnerReferences = secret.OwnerReferences
		if err := s.client.Update(ctx, updated); err != nil {
			return fmt.Errorf("failed to update token secret: %w", err)
		}
	}
	return nil
}

// updateRobotTokenStatus records the active token metadata on the robot's
// status via the conflict-retrying status writer.
func (s *robotTokenService) updateRobotTokenStatus(ctx context.Context, robotName, tokenID string, issuedAt, expiresAt time.Time) error {
	return s.updateRobotStatusWithRetry(ctx, robotName, func(status *krknv1alpha1.KrknRobotStatus) {
		status.TokenID = tokenID
		status.TokenIssuedAt = &metav1.Time{Time: issuedAt.UTC()}
		status.TokenExpiresAt = &metav1.Time{Time: expiresAt.UTC()}
	})
}

// updateRobotStatusWithRetry applies mutate to a fresh copy of the robot's
// status and persists it. Concurrent status writers (Verify's LastUsedAt
// refresh, other rotations) can conflict with the write; on conflict the robot
// is re-read and the write retried, up to maxTokenStatusAttempts times.
func (s *robotTokenService) updateRobotStatusWithRetry(ctx context.Context, robotName string, mutate func(status *krknv1alpha1.KrknRobotStatus)) error {
	var lastErr error
	for attempt := 0; attempt < maxTokenStatusAttempts; attempt++ {
		robot := &krknv1alpha1.KrknRobot{}
		if err := s.client.Get(ctx, client.ObjectKey{Name: robotName, Namespace: s.namespace}, robot); err != nil {
			return fmt.Errorf("failed to get robot %q: %w", robotName, err)
		}
		updated := robot.DeepCopy()
		mutate(&updated.Status)
		if err := s.client.Status().Update(ctx, updated); err != nil {
			if !apierrors.IsConflict(err) {
				return fmt.Errorf("failed to update robot status: %w", err)
			}
			lastErr = fmt.Errorf("failed to update robot status (conflict): %w", err)
			continue
		}
		return nil
	}
	return lastErr
}

// parseRobotToken splits "krkn_robot_<name>.<secret>" and validates the name
// and secret presence.
func parseRobotToken(token string) (name string, err error) {
	if !strings.HasPrefix(token, robotTokenPrefix) {
		return "", errRobotTokenInvalid
	}
	rest := token[len(robotTokenPrefix):]
	idx := strings.Index(rest, ".")
	if idx < 0 {
		return "", errRobotTokenInvalid
	}
	name, secret := rest[:idx], rest[idx+1:]
	if !auth.IsValidRobotName(name) || secret == "" {
		return "", errRobotTokenInvalid
	}
	return name, nil
}
