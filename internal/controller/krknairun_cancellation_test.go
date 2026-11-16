package controller

import (
	"context"
	"strings"
	"testing"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestKrknAIRunCancellationBeforeProvisioningRetainsRun(t *testing.T) {
	reconciler, kubeClient, run := newAIRunFixture(t)
	run.Spec.CancelRequested = true
	if err := kubeClient.Update(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	request := ctrl.Request{NamespacedName: types.NamespacedName{Name: run.Name, Namespace: run.Namespace}}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("install finalizer: %v", err)
	}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile pre-start cancellation: %v", err)
	}

	var actual krknv1alpha1.KrknAIRun
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &actual); err != nil {
		t.Fatal(err)
	}
	if actual.Status.Phase != aiRunPhaseCancelled || actual.Status.CompletionTime == nil {
		t.Fatalf("pre-start cancellation was not retained as terminal Cancelled: %+v", actual.Status)
	}
	if !clientContainsFinalizer(&actual, krknAIRunFinalizer) {
		t.Fatal("cancellation removed the run cleanup finalizer")
	}
	finalized := meta.FindStatusCondition(actual.Status.Conditions, "ArtifactsFinalized")
	if finalized == nil || finalized.Status != metav1.ConditionTrue {
		t.Fatalf("pre-start cancellation should finalize without upload: %+v", actual.Status.Conditions)
	}
	committed := meta.FindStatusCondition(actual.Status.Conditions, "ArtifactsCommitted")
	if committed == nil || committed.Status != metav1.ConditionUnknown {
		t.Fatalf("pre-start cancellation must not claim committed artifacts: %+v", committed)
	}
	var pods corev1.PodList
	if err := kubeClient.List(context.Background(), &pods, client.InNamespace(run.Namespace)); err != nil {
		t.Fatal(err)
	}
	if len(pods.Items) != 0 {
		t.Fatalf("cancellation provisioned workloads: %+v", pods.Items)
	}
}

func TestKrknAIRunCancellationWaitsForUploaderAfterExecutionStops(t *testing.T) {
	reconciler, kubeClient, run := newAIRunFixture(t)
	run.Spec.CancelRequested = true
	run.Status.Phase = aiRunPhaseRunning
	run.Status.OrchestratorPodName = "orchestrator-running-uploader"
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: run.Status.OrchestratorPodName, Namespace: run.Namespace},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "orchestrator", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 143, Reason: "Signal"}}},
				{Name: "result-uploader", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
			},
		},
	}
	setAIRunPodOwner(pod, run)
	if err := kubeClient.Create(context.Background(), pod); err != nil {
		t.Fatal(err)
	}
	result, err := reconciler.observeRun(context.Background(), run)
	if err != nil {
		t.Fatalf("observe cancelled run: %v", err)
	}
	if run.Status.Phase != aiRunPhaseCancelled || run.Status.FailureReason != "" || result.RequeueAfter == 0 {
		t.Fatalf("execution should be Cancelled while uploader finalization continues: result=%+v status=%+v", result, run.Status)
	}
	finalized := meta.FindStatusCondition(run.Status.Conditions, "ArtifactsFinalized")
	if finalized == nil || finalized.Status != metav1.ConditionFalse {
		t.Fatalf("uploader still running should remain independently unfinalized: %+v", run.Status.Conditions)
	}
}

func TestKrknAIRunCancellationWithMissingOrchestratorRemainsUnknown(t *testing.T) {
	reconciler, _, run := newAIRunFixture(t)
	run.Spec.CancelRequested = true
	run.Status.Phase = aiRunPhaseRunning
	run.Status.OrchestratorPodName = "missing-orchestrator"
	result, err := reconciler.observeRun(context.Background(), run)
	if err != nil {
		t.Fatalf("observe missing cancelled orchestrator: %v", err)
	}
	if result.RequeueAfter == 0 || run.Status.Phase != aiRunPhaseCancelling || run.Status.FailureReason != "" {
		t.Fatalf("missing orchestrator must remain honestly Cancelling: result=%+v status=%+v", result, run.Status)
	}
	execution := meta.FindStatusCondition(run.Status.Conditions, "ExecutionStopped")
	if execution == nil || execution.Status != metav1.ConditionUnknown {
		t.Fatalf("missing Pod must not prove orchestrator stop: %+v", run.Status.Conditions)
	}
	finalized := meta.FindStatusCondition(run.Status.Conditions, "ArtifactsFinalized")
	if finalized == nil || finalized.Status != metav1.ConditionTrue {
		t.Fatalf("missing uploader must settle finalization without claiming commitment: %+v", run.Status.Conditions)
	}
	committed := meta.FindStatusCondition(run.Status.Conditions, "ArtifactsCommitted")
	if committed == nil || committed.Status != metav1.ConditionUnknown {
		t.Fatalf("missing Pod must leave artifact commitment unknown: %+v", run.Status.Conditions)
	}
}

func TestKrknAIRunCancellationDeletesNeverStartedPendingPod(t *testing.T) {
	reconciler, kubeClient, run := newAIRunFixture(t)
	run.Spec.CancelRequested = true
	run.Status.Phase = aiRunPhaseProvisioning
	run.Status.OrchestratorPodName = "pending-orchestrator"
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: run.Status.OrchestratorPodName, Namespace: run.Namespace, UID: "pending-orchestrator-uid",
		},
		Status: corev1.PodStatus{Phase: corev1.PodPending},
	}
	setAIRunPodOwner(pod, run)
	if err := kubeClient.Create(context.Background(), pod); err != nil {
		t.Fatal(err)
	}
	result, err := reconciler.observeRun(context.Background(), run)
	if err != nil {
		t.Fatalf("observe pending cancellation: %v", err)
	}
	if result.RequeueAfter != 0 || run.Status.Phase != aiRunPhaseCancelled {
		t.Fatalf("never-started Pod should cancel without waiting for a process: result=%+v status=%+v", result, run.Status)
	}
	finalized := meta.FindStatusCondition(run.Status.Conditions, "ArtifactsFinalized")
	if finalized == nil || finalized.Status != metav1.ConditionTrue || finalized.Reason != "NotRequired" {
		t.Fatalf("never-started orchestrator must not require upload: %+v", run.Status.Conditions)
	}
	var deleted corev1.Pod
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(pod), &deleted); err == nil {
		t.Fatal("never-started orchestrator Pod was not deleted")
	}
}

func TestKrknAIRunCancellationDoesNotTrustForeignSameNamePod(t *testing.T) {
	reconciler, kubeClient, run := newAIRunFixture(t)
	run.Spec.CancelRequested = true
	run.Status.Phase = aiRunPhaseRunning
	run.Status.OrchestratorPodName = "foreign-orchestrator"
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: run.Status.OrchestratorPodName, Namespace: run.Namespace,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: krknv1alpha1.GroupVersion.String(), Kind: "KrknAIRun", Name: run.Name, UID: "stale-run-uid",
			}},
		},
		Status: corev1.PodStatus{Phase: corev1.PodSucceeded},
	}
	if err := kubeClient.Create(context.Background(), pod); err != nil {
		t.Fatal(err)
	}
	result, err := reconciler.observeRun(context.Background(), run)
	if err != nil {
		t.Fatalf("observe same-name foreign Pod: %v", err)
	}
	if result.RequeueAfter == 0 || run.Status.Phase != aiRunPhaseCancelling {
		t.Fatalf("foreign Pod termination must not complete the run: result=%+v status=%+v", result, run.Status)
	}
	execution := meta.FindStatusCondition(run.Status.Conditions, "ExecutionStopped")
	if execution == nil || execution.Status != metav1.ConditionUnknown {
		t.Fatalf("foreign Pod must not prove execution stopped: %+v", run.Status.Conditions)
	}
	var retained corev1.Pod
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(pod), &retained); err != nil {
		t.Fatalf("foreign same-name Pod was changed or deleted: %v", err)
	}
}

func TestKrknAIRunCancellationSettlesLostUploaderAttempt(t *testing.T) {
	reconciler, kubeClient, run := newAIRunFixture(t)
	run.Spec.CancelRequested = true
	run.Status.Phase = aiRunPhaseRunning
	run.Status.OrchestratorPodName = "terminal-without-uploader-status"
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: run.Status.OrchestratorPodName, Namespace: run.Namespace},
		Status:     corev1.PodStatus{Phase: corev1.PodFailed},
	}
	setAIRunPodOwner(pod, run)
	if err := kubeClient.Create(context.Background(), pod); err != nil {
		t.Fatal(err)
	}
	result, err := reconciler.observeRun(context.Background(), run)
	if err != nil {
		t.Fatalf("observe terminal Pod without uploader status: %v", err)
	}
	if result.RequeueAfter != 0 || run.Status.Phase != aiRunPhaseCancelled {
		t.Fatalf("terminal Pod without uploader must settle instead of spinning: result=%+v status=%+v", result, run.Status)
	}
	finalized := meta.FindStatusCondition(run.Status.Conditions, "ArtifactsFinalized")
	if finalized == nil || finalized.Status != metav1.ConditionTrue || finalized.Reason != "UploaderStatusUnavailable" {
		t.Fatalf("lost uploader must settle finalization explicitly: %+v", run.Status.Conditions)
	}
	committed := meta.FindStatusCondition(run.Status.Conditions, "ArtifactsCommitted")
	if committed == nil || committed.Status != metav1.ConditionUnknown {
		t.Fatalf("lost uploader must not fabricate an artifact result: %+v", run.Status.Conditions)
	}
}

func TestKrknAIRunCancellationPropagationUsesOwnerUID(t *testing.T) {
	reconciler, kubeClient, run := newAIRunFixture(t)
	owner := metav1.OwnerReference{APIVersion: krknv1alpha1.GroupVersion.String(), Kind: "KrknAIRun", Name: run.Name, UID: run.UID}
	owned := cancellationScenario("owned-child", false)
	owned.Namespace = run.Namespace
	owned.OwnerReferences = []metav1.OwnerReference{owner}
	owned.Status.Phase = "Running"
	stale := cancellationScenario("stale-child", false)
	stale.Namespace = run.Namespace
	stale.OwnerReferences = []metav1.OwnerReference{{APIVersion: owner.APIVersion, Kind: owner.Kind, Name: owner.Name, UID: "old-run-uid"}}
	stale.Status.Phase = "Running"
	if err := kubeClient.Create(context.Background(), owned); err != nil {
		t.Fatal(err)
	}
	if err := kubeClient.Create(context.Background(), stale); err != nil {
		t.Fatal(err)
	}
	run.Spec.CancelRequested = true
	result, err := reconciler.reconcileCancellation(context.Background(), run)
	if err != nil {
		t.Fatalf("reconcile parent cancellation: %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Fatal("parent should continue reconciling while owned child is still active")
	}
	var actualOwned, actualStale krknv1alpha1.KrknScenarioRun
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(owned), &actualOwned); err != nil {
		t.Fatal(err)
	}
	if err := kubeClient.Get(context.Background(), client.ObjectKeyFromObject(stale), &actualStale); err != nil {
		t.Fatal(err)
	}
	if !actualOwned.Spec.CancelRequested || actualStale.Spec.CancelRequested {
		t.Fatalf("cancellation was not restricted to current owner UID: owned=%v stale=%v", actualOwned.Spec.CancelRequested, actualStale.Spec.CancelRequested)
	}
	if len(run.Status.ScenarioRunRefs) != 1 || run.Status.ScenarioRunRefs[0] != owned.Name {
		t.Fatalf("parent references must contain only UID-owned children: %v", run.Status.ScenarioRunRefs)
	}
}

func TestScenarioRunCancellationSignalsPodAndDoesNotRetry(t *testing.T) {
	ctx := context.Background()
	child := cancellationScenario("cancel-running", true)
	child.Status.Phase = "Running"
	child.Status.ReportStatus = &krknv1alpha1.ReportStatus{Generated: true}
	child.Status.ClusterJobs = []krknv1alpha1.ClusterJobStatus{{
		ProviderName: "provider", ClusterName: "cluster", JobID: "job-one", PodName: "scenario-pod",
		Phase: "Pending", MaxRetries: 0, MaxRetriesConfigured: true,
	}}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "scenario-pod", Namespace: "default", Annotations: map[string]string{scenarioCancellationAnnotation: "false"}},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	setScenarioPodOwner(pod, child)
	reconciler, kubeClient := newCancellationScenarioReconciler(t, child, pod)
	result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(child)})
	if err != nil {
		t.Fatalf("reconcile cancellation: %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Fatal("cancelling scenario Pod should be requeued until it stops")
	}
	var actual krknv1alpha1.KrknScenarioRun
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(child), &actual); err != nil {
		t.Fatal(err)
	}
	if actual.Status.Phase != "Cancelling" || actual.Status.FailedJobs != 0 || actual.Status.CancelledJobs != 0 {
		t.Fatalf("active cancellation must be truthful and not failed: %+v", actual.Status)
	}
	var actualPod corev1.Pod
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(pod), &actualPod); err != nil {
		t.Fatal(err)
	}
	if actualPod.Annotations[scenarioCancellationAnnotation] != "true" {
		t.Fatalf("controller did not signal the running Pod: %v", actualPod.Annotations)
	}
	if condition := meta.FindStatusCondition(actual.Status.Conditions, "ExecutionStopped"); condition == nil || condition.Status != metav1.ConditionFalse {
		t.Fatalf("execution must remain active until the Pod stops: %+v", actual.Status.Conditions)
	}
}

func TestScenarioRunCancellationDeletesNeverStartedPendingPod(t *testing.T) {
	ctx := context.Background()
	child := cancellationScenario("cancel-pending", true)
	child.Status.Phase = "Running"
	child.Status.ReportStatus = &krknv1alpha1.ReportStatus{Generated: true}
	child.Status.ClusterJobs = []krknv1alpha1.ClusterJobStatus{{
		ProviderName: "provider", ClusterName: "cluster", JobID: "pending-job", PodName: "pending-pod",
		Phase: "Pending", MaxRetries: 0, MaxRetriesConfigured: true,
	}}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pending-pod", Namespace: "default", UID: "pending-pod-uid"},
		Status:     corev1.PodStatus{Phase: corev1.PodPending},
	}
	setScenarioPodOwner(pod, child)
	reconciler, kubeClient := newCancellationScenarioReconciler(t, child, pod)
	if _, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(child)}); err != nil {
		t.Fatalf("reconcile pending Pod cancellation: %v", err)
	}
	var actual krknv1alpha1.KrknScenarioRun
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(child), &actual); err != nil {
		t.Fatal(err)
	}
	if actual.Status.Phase != "Cancelled" || actual.Status.CancelledJobs != 1 || actual.Status.FailedJobs != 0 ||
		actual.Status.ClusterJobs[0].FailureReason != "CancellationCleanupNotRequired" {
		t.Fatalf("never-started Pod cancellation was not finalized truthfully: %+v", actual.Status)
	}
	var deleted corev1.Pod
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(pod), &deleted); err == nil {
		t.Fatal("never-started scenario Pod was not deleted")
	}
}

func TestScenarioCancellationDoesNotTrustForeignSameNamePod(t *testing.T) {
	ctx := context.Background()
	child := cancellationScenario("foreign-scenario-pod", true)
	child.Status.Phase = "Running"
	child.Status.ReportStatus = &krknv1alpha1.ReportStatus{Generated: true}
	child.Status.ClusterJobs = []krknv1alpha1.ClusterJobStatus{{
		ProviderName: "provider", ClusterName: "cluster", JobID: "foreign-job", PodName: "foreign-scenario-pod",
		Phase: "Running", MaxRetries: 0, MaxRetriesConfigured: true,
	}}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foreign-scenario-pod", Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: krknv1alpha1.GroupVersion.String(), Kind: "KrknScenarioRun", Name: child.Name, UID: "stale-child-uid",
			}},
		},
		Status: corev1.PodStatus{Phase: corev1.PodSucceeded},
	}
	reconciler, kubeClient := newCancellationScenarioReconciler(t, child, pod)
	result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(child)})
	if err != nil {
		t.Fatalf("reconcile foreign scenario Pod: %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Fatal("foreign same-name Pod must remain unresolved")
	}
	var actual krknv1alpha1.KrknScenarioRun
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(child), &actual); err != nil {
		t.Fatal(err)
	}
	if actual.Status.Phase != "Cancelling" || actual.Status.CancelledJobs != 0 || actual.Status.FailedJobs != 0 ||
		actual.Status.ClusterJobs[0].Phase != "Cancelling" {
		t.Fatalf("foreign Pod termination was incorrectly trusted: %+v", actual.Status)
	}
	var retained corev1.Pod
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(pod), &retained); err != nil {
		t.Fatalf("foreign same-name Pod was changed or deleted: %v", err)
	}
}

func TestScenarioRunCancellationWithMissingPodAndZeroRetriesIsCancelledUnknown(t *testing.T) {
	ctx := context.Background()
	child := cancellationScenario("cancel-missing", true)
	child.Status.Phase = "Running"
	child.Status.ReportStatus = &krknv1alpha1.ReportStatus{Generated: true}
	child.Status.ClusterJobs = []krknv1alpha1.ClusterJobStatus{{
		ProviderName: "provider", ClusterName: "cluster", JobID: "job-one", PodName: "missing-pod",
		Phase: "Failed", RetryCount: 0, MaxRetries: 0, MaxRetriesConfigured: true,
	}}
	reconciler, kubeClient := newCancellationScenarioReconciler(t, child)
	result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(child)})
	if err != nil {
		t.Fatalf("reconcile missing Pod cancellation: %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Fatal("unknown cleanup outcome must continue reconciliation")
	}
	var actual krknv1alpha1.KrknScenarioRun
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(child), &actual); err != nil {
		t.Fatal(err)
	}
	if actual.Status.Phase != "Cancelled" || actual.Status.CancelledJobs != 1 || actual.Status.FailedJobs != 0 {
		t.Fatalf("cancellation must take precedence over zero retries and missing Pod: %+v", actual.Status)
	}
	job := actual.Status.ClusterJobs[0]
	if job.Phase != "Cancelled" || job.FailureReason != "CancellationCleanupUnknown" {
		t.Fatalf("missing Pod must not fabricate cleanup success: %+v", job)
	}
	cleanup := meta.FindStatusCondition(actual.Status.Conditions, "CleanupComplete")
	if cleanup == nil || cleanup.Status != metav1.ConditionUnknown {
		t.Fatalf("missing Pod cleanup should be unknown: %+v", actual.Status.Conditions)
	}
}

func TestScenarioPodProjectsCancellationAnnotationAndForwardsSignal(t *testing.T) {
	ctx := context.Background()
	child := cancellationScenario("wrapper", false)
	reconciler, kubeClient := newCancellationScenarioReconciler(t, child)
	resources := &preparedJobResources{jobID: "job-wrapper", clusterName: "cluster", containerImage: "registry.example/scenario:test"}
	podName, err := reconciler.submitScenarioPod(ctx, child, resources)
	if err != nil {
		t.Fatalf("submit scenario Pod: %v", err)
	}
	var pod corev1.Pod
	if err := kubeClient.Get(ctx, types.NamespacedName{Name: podName, Namespace: "default"}, &pod); err != nil {
		t.Fatal(err)
	}
	if pod.Annotations[scenarioCancellationAnnotation] != "false" {
		t.Fatalf("scenario Pod has no initial cancellation annotation: %v", pod.Annotations)
	}
	var projected bool
	for _, volume := range pod.Spec.Volumes {
		if volume.Name == "cancellation" && volume.DownwardAPI != nil && len(volume.DownwardAPI.Items) == 1 {
			item := volume.DownwardAPI.Items[0]
			projected = item.Path == "requested" && item.FieldRef != nil && item.FieldRef.FieldPath == "metadata.annotations['"+scenarioCancellationAnnotation+"']"
		}
	}
	if !projected {
		t.Fatalf("scenario Pod does not project the cancellation annotation: %+v", pod.Spec.Volumes)
	}
	container := pod.Spec.Containers[0]
	if !strings.Contains(strings.Join(container.Args, " "), scenarioCancellationPath) ||
		!strings.Contains(strings.Join(container.Args, " "), "kill -TERM --") ||
		!strings.Contains(strings.Join(container.Args, " "), "CANCELLATION_FORCE_STOPPED") {
		t.Fatalf("scenario wrapper does not forward and bound cancellation: %v", container.Args)
	}
}

func TestCancellationCleanupReceiptRecognizesForcedStop(t *testing.T) {
	outcome, detail, found := cancellationCleanupReceipt([]byte("CANCELLATION_CLEANUP outcome=complete\nCANCELLATION_FORCE_STOPPED grace_seconds=120\n"))
	if !found || outcome != "unknown" || !strings.Contains(detail, "CANCELLATION_FORCE_STOPPED") {
		t.Fatalf("forced process-group stop must override any success receipt: outcome=%q detail=%q found=%v", outcome, detail, found)
	}
}

func TestCancellationCleanupReceiptParsesCoreDecision(t *testing.T) {
	outcome, detail, found := cancellationCleanupReceipt([]byte(
		"CANCELLATION_CLEANUP outcome=complete detail=owned resources stopped\n",
	))
	if !found || outcome != "complete" || detail != "owned resources stopped" {
		t.Fatalf("unexpected cleanup receipt: outcome=%q detail=%q found=%v", outcome, detail, found)
	}
}

func cancellationScenario(name string, cancelled bool) *krknv1alpha1.KrknScenarioRun {
	public := false
	return &krknv1alpha1.KrknScenarioRun{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", UID: types.UID(name + "-uid")},
		Spec: krknv1alpha1.KrknScenarioRunSpec{
			CancelRequested: cancelled,
			TargetRequestID: "target-request",
			TargetClusters:  map[string][]string{"provider": {"cluster"}},
			Scenario:        krknv1alpha1.ScenarioReference{Name: "scenario", Private: &public},
			MaxRetries:      0,
		},
	}
}

func setScenarioPodOwner(pod *corev1.Pod, scenarioRun *krknv1alpha1.KrknScenarioRun) {
	pod.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: krknv1alpha1.GroupVersion.String(),
		Kind:       "KrknScenarioRun",
		Name:       scenarioRun.Name,
		UID:        scenarioRun.UID,
	}}
}

func setAIRunPodOwner(pod *corev1.Pod, aiRun *krknv1alpha1.KrknAIRun) {
	pod.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: krknv1alpha1.GroupVersion.String(),
		Kind:       "KrknAIRun",
		Name:       aiRun.Name,
		UID:        aiRun.UID,
	}}
}

func newCancellationScenarioReconciler(t *testing.T, scenarioRun *krknv1alpha1.KrknScenarioRun, objects ...client.Object) (*KrknScenarioRunReconciler, client.Client) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := krknv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	builder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(scenarioRun).WithStatusSubresource(scenarioRun)
	if len(objects) > 0 {
		builder = builder.WithObjects(objects...)
	}
	kubeClient := builder.Build()
	return &KrknScenarioRunReconciler{Client: kubeClient, Scheme: scheme, Namespace: "default"}, kubeClient
}

func clientContainsFinalizer(object client.Object, finalizer string) bool {
	for _, value := range object.GetFinalizers() {
		if value == finalizer {
			return true
		}
	}
	return false
}
func TestScenarioRunSingleJobCancelFlagSignalsPod(t *testing.T) {
	ctx := context.Background()
	child := cancellationScenario("single-job-cancel", false)
	child.Status.Phase = "Running"
	child.Status.ReportStatus = &krknv1alpha1.ReportStatus{Generated: true}
	child.Status.ClusterJobs = []krknv1alpha1.ClusterJobStatus{{
		ProviderName: "provider", ClusterName: "cluster", JobID: "job-one", PodName: "scenario-pod",
		Phase: "Running", CancelRequested: true, MaxRetries: 0, MaxRetriesConfigured: true,
	}}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "scenario-pod", Namespace: "default"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	setScenarioPodOwner(pod, child)
	reconciler, kubeClient := newCancellationScenarioReconciler(t, child, pod)
	if _, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(child)}); err != nil {
		t.Fatalf("reconcile single-job cancellation: %v", err)
	}
	var actual krknv1alpha1.KrknScenarioRun
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(child), &actual); err != nil {
		t.Fatal(err)
	}
	if len(actual.Status.ClusterJobs) != 1 || actual.Status.ClusterJobs[0].Phase != "Cancelling" {
		t.Fatalf("persisted job cancellation was not drained: %+v", actual.Status.ClusterJobs)
	}
	var actualPod corev1.Pod
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(pod), &actualPod); err != nil {
		t.Fatal(err)
	}
	if actualPod.Annotations[scenarioCancellationAnnotation] != "true" {
		t.Fatalf("single-job cancellation did not signal the Pod: %v", actualPod.Annotations)
	}
}

func TestScenarioRunOwnerCancellationBlocksLatePodCreation(t *testing.T) {
	child := cancellationScenario("late-child", false)
	parent := &krknv1alpha1.KrknAIRun{
		ObjectMeta: metav1.ObjectMeta{Name: "parent-run", Namespace: "default", UID: "parent-run-uid"},
		Spec:       krknv1alpha1.KrknAIRunSpec{CancelRequested: true},
	}
	child.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: krknv1alpha1.GroupVersion.String(), Kind: "KrknAIRun",
		Name: parent.Name, UID: parent.UID,
	}}
	reconciler, kubeClient := newCancellationScenarioReconciler(t, child, parent)
	_, err := reconciler.submitScenarioPod(context.Background(), child, &preparedJobResources{
		jobID: "late-job", clusterName: "cluster", containerImage: "registry.example/scenario:test",
	})
	if err != errScenarioRunCancelled {
		t.Fatalf("late child workload should be blocked by its owner cancellation, error=%v", err)
	}
	var pods corev1.PodList
	if err := kubeClient.List(context.Background(), &pods, client.InNamespace("default")); err != nil {
		t.Fatal(err)
	}
	if len(pods.Items) != 0 {
		t.Fatalf("cancelled parent allowed late child Pod creation: %+v", pods.Items)
	}
}

func TestScenarioRunCancellationPreservesCompletedJobs(t *testing.T) {
	ctx := context.Background()
	child := cancellationScenario("preserve-completed", true)
	child.Status.Phase = "Running"
	child.Status.ReportStatus = &krknv1alpha1.ReportStatus{Generated: true}
	child.Status.ClusterJobs = []krknv1alpha1.ClusterJobStatus{
		{ProviderName: "provider", ClusterName: "done", JobID: "done-job", Phase: "Succeeded"},
		{ProviderName: "provider", ClusterName: "not-started", JobID: "reserved", Phase: "Creating"},
	}
	reconciler, kubeClient := newCancellationScenarioReconciler(t, child)
	if _, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(child)}); err != nil {
		t.Fatalf("reconcile cancellation: %v", err)
	}
	var actual krknv1alpha1.KrknScenarioRun
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(child), &actual); err != nil {
		t.Fatal(err)
	}
	if actual.Status.Phase != "Cancelled" || actual.Status.SuccessfulJobs != 1 || actual.Status.CancelledJobs != 1 ||
		actual.Status.FailedJobs != 0 || actual.Status.ClusterJobs[0].Phase != "Succeeded" {
		t.Fatalf("cancellation did not preserve completed work and score it separately: %+v", actual.Status)
	}
	if actual.Status.ClusterJobs[1].FailureReason != "CancellationCleanupUnknown" {
		t.Fatalf("reserved job with no Pod must not claim cleanup was unnecessary: %+v", actual.Status.ClusterJobs[1])
	}
}

func TestScenarioRunCancelledPhaseCannotResumeScheduling(t *testing.T) {
	ctx := context.Background()
	child := cancellationScenario("phase-cancelled", false)
	child.Status.Phase = "Cancelled"
	reconciler, kubeClient := newCancellationScenarioReconciler(t, child)
	result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(child)})
	if err != nil {
		t.Fatalf("reconcile cancelled phase: %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Fatal("persisting phase-derived cancellation should requeue")
	}
	var actual krknv1alpha1.KrknScenarioRun
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(child), &actual); err != nil {
		t.Fatal(err)
	}
	if !actual.Spec.CancelRequested {
		t.Fatal("cancelled phase did not restore durable cancellation intent")
	}
	var pods corev1.PodList
	if err := kubeClient.List(ctx, &pods, client.InNamespace(child.Namespace)); err != nil {
		t.Fatal(err)
	}
	if len(pods.Items) != 0 {
		t.Fatalf("cancelled scenario resumed scheduling: %+v", pods.Items)
	}
}
