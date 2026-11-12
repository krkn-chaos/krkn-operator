package workspace

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"path"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Binding records the exact workspace and PVC selected in a caller-supplied
// namespace. Persist Reference with the workload so retries can revalidate it.
type Binding struct {
	Namespace string
	Reference krknv1alpha1.WorkspaceReference
}

// Available reports the reconciler's observation for the current workspace
// generation. It is metadata only: it does not prove filesystem write access.
func Available(entry *krknv1alpha1.KrknWorkspace) bool {
	if entry == nil || entry.DeletionTimestamp != nil {
		return false
	}
	condition := meta.FindStatusCondition(entry.Status.Conditions, "Available")
	return entry.Status.ObservedGeneration == entry.Generation &&
		condition != nil && condition.Status == metav1.ConditionTrue && condition.ObservedGeneration == entry.Generation
}

// Resolve finds a published workspace and its live PVC in namespace. The caller
// supplies its Pod namespace and handles platform user authorization separately.
// A stale status, missing claim, or incompatible claim returns ErrUnavailable;
// a claim replaced since publication returns ErrReplaced.
func Resolve(ctx context.Context, reader client.Reader, namespace, name string) (Binding, error) {
	if namespace == "" || name == "" {
		return Binding{}, fmt.Errorf("%w: namespace and workspace name are required", ErrUnavailable)
	}
	entry := &krknv1alpha1.KrknWorkspace{}
	if err := reader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, entry); err != nil {
		if apierrors.IsNotFound(err) {
			return Binding{}, fmt.Errorf("%w: workspace %q is missing", ErrUnavailable, name)
		}
		return Binding{}, fmt.Errorf("load workspace %q: %w", name, err)
	}
	if !Available(entry) || entry.UID == "" || entry.Spec.ClaimName == "" || entry.Spec.ClaimUID == "" {
		return Binding{}, fmt.Errorf("%w: workspace %q is not available", ErrUnavailable, name)
	}
	pvc := &corev1.PersistentVolumeClaim{}
	if err := reader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: entry.Spec.ClaimName}, pvc); err != nil {
		if apierrors.IsNotFound(err) {
			return Binding{}, fmt.Errorf("%w: workspace PVC is missing", ErrUnavailable)
		}
		return Binding{}, fmt.Errorf("load workspace PVC: %w", err)
	}
	if err := ValidatePVC(pvc, types.UID(entry.Spec.ClaimUID)); err != nil {
		return Binding{}, fmt.Errorf("workspace PVC: %w", err)
	}
	return Binding{Namespace: namespace, Reference: krknv1alpha1.WorkspaceReference{
		Name: entry.Name, WorkspaceUID: string(entry.UID), ClaimName: pvc.Name, ClaimUID: string(pvc.UID),
	}}, nil
}

// Validate checks a persisted binding against the current workspace and PVC.
// Unlike Resolve, it checks live resources without requiring the reconciler's
// status to have caught up; consumers should call it before creating each Pod.
func Validate(ctx context.Context, reader client.Reader, binding Binding) error {
	ref := binding.Reference
	if binding.Namespace == "" || ref.Name == "" || ref.WorkspaceUID == "" || ref.ClaimName == "" || ref.ClaimUID == "" {
		return fmt.Errorf("%w: workspace binding is incomplete", ErrUnavailable)
	}
	entry := &krknv1alpha1.KrknWorkspace{}
	if err := reader.Get(ctx, client.ObjectKey{Namespace: binding.Namespace, Name: ref.Name}, entry); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("%w: workspace %q is missing", ErrUnavailable, ref.Name)
		}
		return fmt.Errorf("load workspace %q: %w", ref.Name, err)
	}
	if entry.DeletionTimestamp != nil {
		return fmt.Errorf("%w: workspace %q is deleting", ErrUnavailable, ref.Name)
	}
	if string(entry.UID) != ref.WorkspaceUID || entry.Spec.ClaimName != ref.ClaimName || entry.Spec.ClaimUID != ref.ClaimUID {
		return fmt.Errorf("%w: workspace %q binding changed", ErrReplaced, ref.Name)
	}
	pvc := &corev1.PersistentVolumeClaim{}
	if err := reader.Get(ctx, client.ObjectKey{Namespace: binding.Namespace, Name: ref.ClaimName}, pvc); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("%w: workspace PVC is missing", ErrUnavailable)
		}
		return fmt.Errorf("load workspace PVC: %w", err)
	}
	if err := ValidatePVC(pvc, types.UID(ref.ClaimUID)); err != nil {
		return fmt.Errorf("workspace PVC: %w", err)
	}
	return nil
}

// WorkloadKey identifies one consumer workload, target, and execution attempt.
// External consumers such as krkn-ai can use it to derive stable, isolated
// directories from the same published workspace without changing Krkn paths.
// Reusing the same key retrieves the same directory; a different component
// produces a different directory. Every field must be nonempty and stable.
type WorkloadKey struct {
	Consumer    string
	WorkloadUID types.UID
	Target      string
	Attempt     string
}

// SubPathFor returns a deterministic single directory under the PVC root.
// Its hash keeps caller-provided values out of the filesystem path. The legacy
// SubPath format remains unchanged for existing Krkn rollback artifacts.
// Other operators sharing this namespace, including krkn-ai, may use this
// function with a distinct Consumer value to avoid path collisions.
func SubPathFor(key WorkloadKey) (string, error) {
	if key.Consumer == "" || key.WorkloadUID == "" || key.Target == "" || key.Attempt == "" {
		return "", fmt.Errorf("workspace workload identity is incomplete")
	}
	identity := []byte("workspace-v1")
	for _, field := range []string{key.Consumer, string(key.WorkloadUID), key.Target, key.Attempt} {
		identity = binary.AppendUvarint(identity, uint64(len(field)))
		identity = append(identity, field...)
	}
	checksum := sha256.Sum256(identity)
	return "ws-" + hex.EncodeToString(checksum[:20]), nil
}

// MountOptions describes one container mount of a resolved binding.
type MountOptions struct {
	VolumeName string
	MountPath  string
	SubPath    string
	ReadOnly   bool
}

// Mount builds a PVC volume and matching container mount. The caller adds them
// to its Pod and sets its own security context; it also owns the supplied Pod
// namespace, which must be the same namespace used for Resolve or Validate.
func Mount(binding Binding, options MountOptions) (corev1.Volume, corev1.VolumeMount, error) {
	if binding.Namespace == "" || binding.Reference.Name == "" || binding.Reference.WorkspaceUID == "" ||
		binding.Reference.ClaimName == "" || binding.Reference.ClaimUID == "" {
		return corev1.Volume{}, corev1.VolumeMount{}, fmt.Errorf("%w: workspace binding is incomplete", ErrUnavailable)
	}
	if options.VolumeName == "" || !path.IsAbs(options.MountPath) || options.SubPath == "" ||
		path.Clean(options.SubPath) != options.SubPath || path.Base(options.SubPath) != options.SubPath ||
		options.SubPath == "." || options.SubPath == ".." {
		return corev1.Volume{}, corev1.VolumeMount{}, fmt.Errorf("invalid workspace mount options")
	}
	volume := corev1.Volume{Name: options.VolumeName, VolumeSource: corev1.VolumeSource{
		PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
			ClaimName: binding.Reference.ClaimName, ReadOnly: options.ReadOnly,
		},
	}}
	mount := corev1.VolumeMount{Name: options.VolumeName, MountPath: options.MountPath,
		SubPath: options.SubPath, ReadOnly: options.ReadOnly}
	return volume, mount, nil
}
