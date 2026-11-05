package controller

import (
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestWorkspaceReconcilerTracksPVCState(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))
	entry := &krknv1alpha1.KrknWorkspace{ObjectMeta: metav1.ObjectMeta{Name: "friendly", Namespace: "default", Generation: 1},
		Spec: krknv1alpha1.KrknWorkspaceSpec{ClaimName: "shared", ClaimUID: "9e70715d-9154-4b9e-aa76-1ae6c6b2594e"}}
	c := fakeclient.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(entry).WithObjects(entry).Build()
	r := &KrknWorkspaceReconciler{Client: c, Namespace: "default"}
	key := types.NamespacedName{Name: "friendly", Namespace: "default"}
	check := func(status metav1.ConditionStatus, reason string) {
		t.Helper()
		result, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: key})
		require.NoError(t, err)
		require.NotZero(t, result.RequeueAfter)
		got := &krknv1alpha1.KrknWorkspace{}
		require.NoError(t, c.Get(t.Context(), key, got))
		require.Equal(t, got.Generation, got.Status.ObservedGeneration)
		condition := meta.FindStatusCondition(got.Status.Conditions, "Available")
		require.NotNil(t, condition)
		require.Equal(t, status, condition.Status)
		require.Equal(t, reason, condition.Reason)
	}
	check(metav1.ConditionFalse, "PVCMissing")
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: "default", UID: types.UID(entry.Spec.ClaimUID)},
		Spec:   corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}},
		Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending}}
	require.NoError(t, c.Create(t.Context(), pvc))
	check(metav1.ConditionFalse, "PVCUnavailable")
	pvc.Status.Phase = corev1.ClaimBound
	require.NoError(t, c.Status().Update(t.Context(), pvc))
	check(metav1.ConditionTrue, "PVCReady")
	pvc.UID = "dc7f90ee-56d0-4521-b014-d588f2af9341"
	require.NoError(t, c.Delete(t.Context(), &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: "default"}}))
	pvc.ResourceVersion = ""
	require.NoError(t, c.Create(t.Context(), pvc))
	check(metav1.ConditionFalse, "PVCReplaced")
}

func TestWorkspaceReconcilerMapsPVCWithoutOwningIt(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, krknv1alpha1.AddToScheme(scheme))
	entry := &krknv1alpha1.KrknWorkspace{ObjectMeta: metav1.ObjectMeta{Name: "friendly", Namespace: "default"},
		Spec: krknv1alpha1.KrknWorkspaceSpec{ClaimName: "shared", ClaimUID: "uid"}}
	r := &KrknWorkspaceReconciler{Client: fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(entry).Build(), Namespace: "default"}
	requests := r.mapPVC(t.Context(), &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: "default"}})
	require.Equal(t, []ctrl.Request{{NamespacedName: client.ObjectKey{Namespace: "default", Name: "friendly"}}}, requests)
	require.Empty(t, r.mapPVC(t.Context(), &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "default"}}))
}
