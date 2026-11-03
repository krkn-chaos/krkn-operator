// Package workspace defines the published PVC contract used by scenario pods.
package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/google/uuid"
	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const MountPath = "/var/lib/krkn/workspace"

// ConflictsWithMount reports whether a requested file mount would hide the
// workspace mount, or be hidden by it.
func ConflictsWithMount(candidate string) bool {
	clean := path.Clean(candidate)
	return clean == MountPath || strings.HasPrefix(clean, MountPath+"/") || strings.HasPrefix(MountPath, clean+"/")
}

var (
	ErrUnavailable = errors.New("workspace PVC is unavailable")
	ErrReplaced    = errors.New("workspace PVC was replaced")
)

// CanAccess reports whether a user may select the workspace.
func CanAccess(entry *krknv1alpha1.KrknWorkspace, isAdmin bool, userGroups []string) bool {
	if entry == nil || entry.DeletionTimestamp != nil {
		return false
	}
	if isAdmin {
		return true
	}
	if entry.Spec.AvailableToAll {
		return true
	}
	for _, member := range userGroups {
		if entry.Spec.Group != "" && entry.Spec.Group == member {
			return true
		}
	}
	return false
}

// ValidatePVC rejects claims that cannot provide the filesystem workspace.
// expectedUID pins a run to the PVC selected when it was created.
func ValidatePVC(pvc *corev1.PersistentVolumeClaim, expectedUID types.UID) error {
	if pvc == nil || pvc.DeletionTimestamp != nil {
		return fmt.Errorf("%w: claim is missing or deleting", ErrUnavailable)
	}
	if expectedUID != "" && pvc.UID != expectedUID {
		return fmt.Errorf("%w: expected UID %s, got %s", ErrReplaced, expectedUID, pvc.UID)
	}
	if pvc.UID == "" || pvc.Status.Phase != corev1.ClaimBound {
		return fmt.Errorf("%w: claim %s is not bound", ErrUnavailable, pvc.Name)
	}
	if pvc.Spec.VolumeMode != nil && *pvc.Spec.VolumeMode != corev1.PersistentVolumeFilesystem {
		return fmt.Errorf("%w: claim %s is not a filesystem", ErrUnavailable, pvc.Name)
	}
	for _, mode := range pvc.Spec.AccessModes {
		if mode == corev1.ReadWriteMany {
			return nil
		}
	}
	return fmt.Errorf("%w: claim %s does not request ReadWriteMany", ErrUnavailable, pvc.Name)
}

// SubPath assigns a unique directory to a run, target, and execution attempt.
// Target names are hashed so user-provided names cannot escape the volume root.
func SubPath(runUID types.UID, provider, cluster, attemptID string) (string, error) {
	if runUID == "" || provider == "" || cluster == "" || attemptID == "" {
		return "", errors.New("workspace identity is incomplete")
	}
	if _, err := uuid.Parse(string(runUID)); err != nil {
		return "", fmt.Errorf("invalid run UID: %w", err)
	}
	if _, err := uuid.Parse(attemptID); err != nil {
		return "", fmt.Errorf("invalid attempt ID: %w", err)
	}
	checksum := sha256.Sum256([]byte(provider + "\x00" + cluster))
	return path.Join("runs", string(runUID), hex.EncodeToString(checksum[:16]), attemptID), nil
}
