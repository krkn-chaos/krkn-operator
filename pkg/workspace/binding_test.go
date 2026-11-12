package workspace

import (
	"errors"
	"strings"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const fixtureWorkspaceUID = "e7b95e74-ffec-4574-8c7a-e102444decae"
const fixtureClaimUID = "9e70715d-9154-4b9e-aa76-1ae6c6b2594e"

func bindingFixture(t *testing.T) (client.Client, *krknv1alpha1.KrknWorkspace, *corev1.PersistentVolumeClaim) {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))
	entry := &krknv1alpha1.KrknWorkspace{
		ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: "operator-system", UID: fixtureWorkspaceUID, Generation: 1},
		Spec:       krknv1alpha1.KrknWorkspaceSpec{ClaimName: "rwx-claim", ClaimUID: fixtureClaimUID},
		Status: krknv1alpha1.KrknWorkspaceStatus{ObservedGeneration: 1, Conditions: []metav1.Condition{{
			Type: "Available", Status: metav1.ConditionTrue, Reason: "PVCReady", ObservedGeneration: 1, LastTransitionTime: metav1.Now(),
		}}},
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "rwx-claim", Namespace: "operator-system", UID: fixtureClaimUID},
		Spec:       corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}},
		Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(entry).WithObjects(entry, pvc).Build(), entry, pvc
}

func TestResolveAndValidateBinding(t *testing.T) {
	reader, entry, pvc := bindingFixture(t)
	binding, err := Resolve(t.Context(), reader, "operator-system", "shared")
	require.NoError(t, err)
	require.Equal(t, "operator-system", binding.Namespace)
	require.Equal(t, "shared", binding.Reference.Name)
	require.Equal(t, fixtureWorkspaceUID, binding.Reference.WorkspaceUID)
	require.Equal(t, "rwx-claim", binding.Reference.ClaimName)
	require.Equal(t, fixtureClaimUID, binding.Reference.ClaimUID)
	require.NoError(t, Validate(t.Context(), reader, binding))

	_, err = Resolve(t.Context(), reader, "other-namespace", "shared")
	require.ErrorIs(t, err, ErrUnavailable)
	require.NoError(t, reader.Delete(t.Context(), pvc))
	replacement := pvc.DeepCopy()
	replacement.ResourceVersion = ""
	replacement.UID = types.UID("dc7f90ee-56d0-4521-b014-d588f2af9341")
	require.NoError(t, reader.Create(t.Context(), replacement))
	require.ErrorIs(t, Validate(t.Context(), reader, binding), ErrReplaced)
	_, err = Resolve(t.Context(), reader, "operator-system", "shared")
	require.ErrorIs(t, err, ErrReplaced)
	require.NoError(t, reader.Delete(t.Context(), entry))
	require.ErrorIs(t, Validate(t.Context(), reader, binding), ErrUnavailable)
}

func TestAvailabilityIsAnObservationButValidationChecksLivePVC(t *testing.T) {
	reader, entry, _ := bindingFixture(t)
	binding, err := Resolve(t.Context(), reader, "operator-system", "shared")
	require.NoError(t, err)
	entry.Status.ObservedGeneration = 0
	require.NoError(t, reader.Status().Update(t.Context(), entry))
	require.False(t, Available(entry))
	_, err = Resolve(t.Context(), reader, "operator-system", "shared")
	require.ErrorIs(t, err, ErrUnavailable)
	require.NoError(t, Validate(t.Context(), reader, binding), "a pinned binding may still use a live PVC while status catches up")
}

func TestSubPathForIsStableAndSeparatesConsumers(t *testing.T) {
	key := WorkloadKey{Consumer: "krkn-ai", WorkloadUID: types.UID(fixtureWorkspaceUID), Target: "cluster-a", Attempt: "first"}
	base, err := SubPathFor(key)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(base, "ws-"))
	require.NotContains(t, base, "/")
	repeat, err := SubPathFor(key)
	require.NoError(t, err)
	require.Equal(t, base, repeat)
	for _, changed := range []WorkloadKey{
		{Consumer: "krkn", WorkloadUID: key.WorkloadUID, Target: key.Target, Attempt: key.Attempt},
		{Consumer: key.Consumer, WorkloadUID: "another-uid", Target: key.Target, Attempt: key.Attempt},
		{Consumer: key.Consumer, WorkloadUID: key.WorkloadUID, Target: "cluster-b", Attempt: key.Attempt},
		{Consumer: key.Consumer, WorkloadUID: key.WorkloadUID, Target: key.Target, Attempt: "second"},
	} {
		other, err := SubPathFor(changed)
		require.NoError(t, err)
		require.NotEqual(t, base, other)
	}
	key.Attempt = ""
	_, err = SubPathFor(key)
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrUnavailable))
	first, err := SubPathFor(WorkloadKey{Consumer: "a\x00b", WorkloadUID: "c", Target: "d", Attempt: "e"})
	require.NoError(t, err)
	second, err := SubPathFor(WorkloadKey{Consumer: "a", WorkloadUID: "b\x00c", Target: "d", Attempt: "e"})
	require.NoError(t, err)
	require.NotEqual(t, first, second)
}

func TestMountBuildsMatchingVolumeAndMount(t *testing.T) {
	binding := Binding{Namespace: "operator-system", Reference: krknv1alpha1.WorkspaceReference{
		Name: "shared", WorkspaceUID: fixtureWorkspaceUID, ClaimName: "rwx-claim", ClaimUID: fixtureClaimUID,
	}}
	path, err := SubPathFor(WorkloadKey{Consumer: "krkn-ai", WorkloadUID: types.UID(fixtureWorkspaceUID), Target: "cluster-a", Attempt: "first"})
	require.NoError(t, err)
	volume, mount, err := Mount(binding, MountOptions{VolumeName: "workspace", MountPath: "/data", SubPath: path, ReadOnly: true})
	require.NoError(t, err)
	require.Equal(t, "rwx-claim", volume.PersistentVolumeClaim.ClaimName)
	require.True(t, volume.PersistentVolumeClaim.ReadOnly)
	require.Equal(t, "workspace", mount.Name)
	require.Equal(t, "/data", mount.MountPath)
	require.Equal(t, path, mount.SubPath)
	require.True(t, mount.ReadOnly)
	_, _, err = Mount(binding, MountOptions{VolumeName: "workspace", MountPath: "/data", SubPath: "../escape"})
	require.Error(t, err)
	_, _, err = Mount(binding, MountOptions{VolumeName: "workspace", MountPath: "relative", SubPath: path})
	require.Error(t, err)
	binding.Reference.WorkspaceUID = ""
	_, _, err = Mount(binding, MountOptions{VolumeName: "workspace", MountPath: "/data", SubPath: path})
	require.ErrorIs(t, err, ErrUnavailable)
}
