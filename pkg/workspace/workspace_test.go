package workspace

import (
	"errors"
	"strings"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func boundPVC() *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "shared", UID: "9e70715d-9154-4b9e-aa76-1ae6c6b2594e",
		},
		Spec:   corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}},
		Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
	}
}

func TestValidatePVC(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*corev1.PersistentVolumeClaim)
		want error
	}{
		{name: "bound RWX claim"},
		{name: "deleting", edit: func(p *corev1.PersistentVolumeClaim) { now := metav1.Now(); p.DeletionTimestamp = &now }, want: ErrUnavailable},
		{name: "pending", edit: func(p *corev1.PersistentVolumeClaim) { p.Status.Phase = corev1.ClaimPending }, want: ErrUnavailable},
		{name: "read write once", edit: func(p *corev1.PersistentVolumeClaim) {
			p.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		}, want: ErrUnavailable},
		{name: "replaced", edit: func(p *corev1.PersistentVolumeClaim) { p.UID = "21c9e93e-12c3-4b1b-9283-2c9f20e91a6b" }, want: ErrReplaced},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pvc := boundPVC()
			if tc.edit != nil {
				tc.edit(pvc)
			}
			err := ValidatePVC(pvc, types.UID("9e70715d-9154-4b9e-aa76-1ae6c6b2594e"))
			if tc.want == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.want)
			}
		})
	}
}

func TestCanAccess(t *testing.T) {
	entry := &krknv1alpha1.KrknWorkspace{Spec: krknv1alpha1.KrknWorkspaceSpec{Group: "team-a"}}
	require.True(t, CanAccess(entry, true, nil))
	require.True(t, CanAccess(entry, false, []string{"team-a"}))
	require.False(t, CanAccess(entry, false, []string{"team-b"}))
	entry.Spec.AvailableToAll = true
	require.True(t, CanAccess(entry, false, nil))
	now := metav1.Now()
	entry.DeletionTimestamp = &now
	require.False(t, CanAccess(entry, true, nil))
}

func TestSubPathSeparatesRunsTargetsAndAttempts(t *testing.T) {
	runA := types.UID("9e70715d-9154-4b9e-aa76-1ae6c6b2594e")
	runB := types.UID("a0f3ef38-2b28-4823-b0d0-aeb55c7eb73a")
	attemptA := "af3caf28-bb79-40a3-a00c-05ecbe209d10"
	attemptB := "dc7f90ee-56d0-4521-b014-d588f2af9341"
	base, err := SubPath(runA, "provider", "cluster", attemptA)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(base, "run-"+string(runA)+"-"))
	require.NotContains(t, base, "/")
	for _, args := range []struct {
		run                        types.UID
		provider, cluster, attempt string
	}{
		{runB, "provider", "cluster", attemptA},
		{runA, "other", "cluster", attemptA},
		{runA, "provider", "other", attemptA},
		{runA, "provider", "cluster", attemptB},
	} {
		other, err := SubPath(args.run, args.provider, args.cluster, args.attempt)
		require.NoError(t, err)
		require.NotEqual(t, base, other)
	}
	repeat, err := SubPath(runA, "provider", "cluster", attemptA)
	require.NoError(t, err)
	require.Equal(t, base, repeat)
	_, err = SubPath(runA, "provider", "../escape", "not-an-id")
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrUnavailable))
}

func TestConflictsWithMount(t *testing.T) {
	for _, candidate := range []string{
		MountPath, MountPath + "/config.yaml", "/var/lib/krkn", "/var/lib/krkn/workspace/../workspace/file",
	} {
		require.True(t, ConflictsWithMount(candidate), candidate)
	}
	require.False(t, ConflictsWithMount("/etc/krkn/config.yaml"))
}
