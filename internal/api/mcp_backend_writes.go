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
	"regexp"
	"sort"
	"strconv"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/mcpserver"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
)

// mcpIdempotencyKeyLabel marks a run created through the MCP endpoint with the
// client-supplied idempotency key, so retries do not start duplicate
// experiments. The key is scoped to the submitting robot via the owner check.
const mcpIdempotencyKeyLabel = "krkn.krkn-chaos.dev/mcp-idempotency-key"

// idempotencyKeyRegexp constrains client-supplied keys to a valid Kubernetes
// label value of at most 64 characters.
var idempotencyKeyRegexp = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._-]{0,62}[a-zA-Z0-9])?$`)

// validateIdempotencyKey rejects keys that cannot be stored as a label value.
func validateIdempotencyKey(key string) error {
	if !idempotencyKeyRegexp.MatchString(key) {
		return errMCPInvalidArg("idempotencyKey must be 1-64 characters of letters, digits, '.', '_' or '-', starting and ending with an alphanumeric")
	}
	return nil
}

// findIdempotencyScenarioRun returns the robot-owned run previously created
// with the given idempotency key, if any. Only runs owned by this robot and
// visible to it are considered, so a key never leaks another principal's run.
func (b *mcpBackend) findIdempotencyScenarioRun(ctx context.Context, p mcpserver.Principal, key string) (mcpserver.RunStartedOutput, bool, error) {
	runs := &krknv1alpha1.KrknScenarioRunList{}
	if err := b.h.client.List(ctx, runs,
		client.InNamespace(b.h.namespace),
		client.MatchingLabels{mcpIdempotencyKeyLabel: key},
	); err != nil {
		return mcpserver.RunStartedOutput{}, false, err
	}

	names := make([]string, 0, len(runs.Items))
	for i := range runs.Items {
		if runs.Items[i].Spec.OwnerUserID == "robot:"+p.RobotName {
			names = append(names, runs.Items[i].Name)
		}
	}
	if len(names) == 0 {
		return mcpserver.RunStartedOutput{}, false, nil
	}
	sort.Strings(names)

	// Multiple matches can only result from a race or a label patch failure;
	// deterministically return the oldest (first created) run.
	run, err := b.getVisibleScenarioRun(ctx, p, names[0])
	if err != nil {
		return mcpserver.RunStartedOutput{}, false, err
	}
	out := mcpserver.RunStartedOutput{Kind: "scenario_run", Name: run.Name}
	for _, job := range run.Status.ClusterJobs {
		out.Jobs = append(out.Jobs, mcpserver.StartedJobOutput{
			ClusterName: job.ClusterName,
			JobID:       job.JobID,
			Status:      job.Phase,
			Success:     true,
		})
	}
	return out, true, nil
}

// labelScenarioRunIdempotent records the idempotency key on a newly created
// run. A failure is logged but not fatal: the run did start, and the loss only
// affects duplicate suppression for a later retry of the same key.
func (b *mcpBackend) labelScenarioRunIdempotent(ctx context.Context, runName, key string) {
	run := &krknv1alpha1.KrknScenarioRun{ObjectMeta: metav1.ObjectMeta{Name: runName, Namespace: b.h.namespace}}
	patch := client.MergeFrom(run.DeepCopy())
	if run.Labels == nil {
		run.Labels = map[string]string{}
	}
	run.Labels[mcpIdempotencyKeyLabel] = key
	if err := b.h.client.Patch(ctx, run, patch); err != nil {
		log.FromContext(ctx).Error(err, "Failed to label scenario run with idempotency key", "scenarioRun", runName)
	}
}

// findIdempotencyGraphRun returns the robot-owned graph run previously created
// with the given idempotency key, if any.
func (b *mcpBackend) findIdempotencyGraphRun(ctx context.Context, p mcpserver.Principal, key string) (mcpserver.RunStartedOutput, bool, error) {
	runs := &krknv1alpha1.KrknGraphRunList{}
	if err := b.h.client.List(ctx, runs,
		client.InNamespace(b.h.namespace),
		client.MatchingLabels{mcpIdempotencyKeyLabel: key},
	); err != nil {
		return mcpserver.RunStartedOutput{}, false, err
	}

	names := make([]string, 0, len(runs.Items))
	for i := range runs.Items {
		if runs.Items[i].Spec.OwnerUserID == "robot:"+p.RobotName {
			names = append(names, runs.Items[i].Name)
		}
	}
	if len(names) == 0 {
		return mcpserver.RunStartedOutput{}, false, nil
	}
	sort.Strings(names)

	gr := &krknv1alpha1.KrknGraphRun{}
	if err := b.h.client.Get(ctx, client.ObjectKey{Name: names[0], Namespace: b.h.namespace}, gr); err != nil {
		return mcpserver.RunStartedOutput{}, false, err
	}
	if _, ok := b.graphRunVisible(ctx, p, gr); !ok {
		return mcpserver.RunStartedOutput{}, false, nil
	}
	return mcpserver.RunStartedOutput{Kind: "graph_run", Name: gr.Name}, true, nil
}

// labelGraphRunIdempotent records the idempotency key on a newly created graph
// run. A failure is logged but not fatal; see labelScenarioRunIdempotent.
func (b *mcpBackend) labelGraphRunIdempotent(ctx context.Context, runName, key string) {
	gr := &krknv1alpha1.KrknGraphRun{ObjectMeta: metav1.ObjectMeta{Name: runName, Namespace: b.h.namespace}}
	patch := client.MergeFrom(gr.DeepCopy())
	if gr.Labels == nil {
		gr.Labels = map[string]string{}
	}
	gr.Labels[mcpIdempotencyKeyLabel] = key
	if err := b.h.client.Patch(ctx, gr, patch); err != nil {
		log.FromContext(ctx).Error(err, "Failed to label graph run with idempotency key", "graphRun", runName)
	}
}

// RunScenario dispatches POST /api/v1/scenarios/run as the robot, so every
// existing validation (run permission on all targets, file, registry, cloud
// credential and Elasticsearch access) runs unchanged.
func (b *mcpBackend) RunScenario(ctx context.Context, p mcpserver.Principal, in mcpserver.RunScenarioInput) (mcpserver.RunStartedOutput, error) {
	if in.IdempotencyKey != "" {
		if err := validateIdempotencyKey(in.IdempotencyKey); err != nil {
			return mcpserver.RunStartedOutput{}, err
		}
		if existing, found, err := b.findIdempotencyScenarioRun(ctx, p, in.IdempotencyKey); err != nil {
			return mcpserver.RunStartedOutput{}, errMCPInternal()
		} else if found {
			return existing, nil
		}
	}

	private := in.PrivateRegistry
	req := ScenarioRunRequest{
		TargetRequestID:         in.TargetRequestID,
		TargetClusters:          in.TargetClusters,
		Categories:              in.Categories,
		Scenario:                krknv1alpha1.ScenarioReference{Name: in.ScenarioName, Private: &private, RegistryName: in.RegistryName},
		Environment:             in.Environment,
		FileReferences:          fileReferencesToAPI(in.FileReferences),
		CustomRunName:           in.CustomRunName,
		MaxRetries:              in.MaxRetries,
		CloudCredentialRef:      in.CloudCredentialRef,
		ElasticsearchConfigName: in.ElasticsearchConfigName,
		ResiliencyScoreEnabled:  in.ResiliencyScoreEnabled,
	}

	res, err := b.h.dispatchAsRobot(ctx, p.RobotName, "POST", ScenariosRunPath, nil, req, nil, b.h.PostScenarioRun)
	if err != nil {
		return mcpserver.RunStartedOutput{}, errMCPInternal()
	}
	if err := backendErrorFromWriteDispatch(res); err != nil {
		return mcpserver.RunStartedOutput{}, err
	}

	var created ScenarioRunCreateResponse
	if err := decodeJSON(res.Body, &created); err != nil {
		return mcpserver.RunStartedOutput{}, errMCPInternal()
	}
	if in.IdempotencyKey != "" {
		b.labelScenarioRunIdempotent(ctx, created.ScenarioRunName, in.IdempotencyKey)
	}
	return mcpserver.RunStartedOutput{
		Kind: "scenario_run",
		Name: created.ScenarioRunName,
		Jobs: jobsFromCreateResponse(created),
	}, nil
}

// fileReferencesToAPI maps MCP file references to the REST type.
func fileReferencesToAPI(in []mcpserver.FileReferenceInput) []files.FileReference {
	if len(in) == 0 {
		return nil
	}
	out := make([]files.FileReference, 0, len(in))
	for _, ref := range in {
		out = append(out, files.FileReference{FileID: ref.FileID, MountPath: ref.MountPath})
	}
	return out
}

// jobsFromCreateResponse maps the REST create response to per-target jobs.
func jobsFromCreateResponse(created ScenarioRunCreateResponse) []mcpserver.StartedJobOutput {
	if len(created.TargetClusters) == 0 {
		return nil
	}
	jobs := make([]mcpserver.StartedJobOutput, 0, len(created.TargetClusters))
	for _, clusters := range created.TargetClusters {
		for _, name := range clusters {
			jobs = append(jobs, mcpserver.StartedJobOutput{
				ClusterName: name,
				Status:      "Pending",
				Success:     true,
			})
		}
	}
	return jobs
}

// ReplayScenarioRun re-submits an existing run's configuration. The payload
// (which contains env values and inline file contents) is never returned or
// logged.
func (b *mcpBackend) ReplayScenarioRun(ctx context.Context, p mcpserver.Principal, in mcpserver.ReplayScenarioRunInput) (mcpserver.RunStartedOutput, error) {
	run, err := b.getVisibleScenarioRun(ctx, p, in.ScenarioRunName)
	if err != nil {
		return mcpserver.RunStartedOutput{}, err
	}

	targetRequestID := run.Spec.TargetRequestID
	if in.TargetRequestID != "" {
		targetRequestID = in.TargetRequestID
	}
	// Fail deterministically when the target request is gone, before dispatch.
	tr := &krknv1alpha1.KrknTargetRequest{}
	if err := b.h.client.Get(ctx, client.ObjectKey{Name: targetRequestID, Namespace: b.h.namespace}, tr); err != nil {
		return mcpserver.RunStartedOutput{}, errMCPInvalidArg(
			"original target request is unavailable; call krkn_list_clusters and pass targetRequestId")
	}

	payload, err := b.h.reconstructScenarioRunPayload(withRobotClaims(ctx, p.RobotName), run)
	if err != nil {
		return mcpserver.RunStartedOutput{}, errMCPInternal()
	}
	payload.TargetRequestID = targetRequestID
	if in.CustomRunName != "" {
		payload.CustomRunName = in.CustomRunName
	}
	payload.ScenarioImage = ""

	res, err := b.h.dispatchAsRobot(ctx, p.RobotName, "POST", ScenariosRunPath, nil, payload, nil, b.h.PostScenarioRun)
	if err != nil {
		return mcpserver.RunStartedOutput{}, errMCPInternal()
	}
	if err := backendErrorFromWriteDispatch(res); err != nil {
		return mcpserver.RunStartedOutput{}, err
	}

	var created ScenarioRunCreateResponse
	if err := decodeJSON(res.Body, &created); err != nil {
		return mcpserver.RunStartedOutput{}, errMCPInternal()
	}
	return mcpserver.RunStartedOutput{
		Kind: "scenario_run",
		Name: created.ScenarioRunName,
		Jobs: jobsFromCreateResponse(created),
	}, nil
}

// CancelJob dispatches DELETE /api/v1/scenarios/run/jobs/{jobID} as the
// robot. DeleteSingleJob parses that path and itself checks cancel
// permission on the job's cluster.
func (b *mcpBackend) CancelJob(ctx context.Context, p mcpserver.Principal, in mcpserver.CancelJobInput) (mcpserver.ActionOutput, error) {
	// The job must belong to a run visible to the robot.
	run, err := b.getVisibleScenarioRun(ctx, p, in.ScenarioRunName)
	if err != nil {
		return mcpserver.ActionOutput{}, err
	}
	found := false
	for _, job := range run.Status.ClusterJobs {
		if job.JobID == in.JobID {
			found = true
			break
		}
	}
	if !found {
		return mcpserver.ActionOutput{}, errMCPNotFound("job %q not found", in.JobID)
	}

	res, err := b.h.dispatchAsRobot(ctx, p.RobotName, "DELETE", ScenariosRunJobsPath+"/"+in.JobID, nil, nil, nil, b.h.DeleteSingleJob)
	if err != nil {
		return mcpserver.ActionOutput{}, errMCPInternal()
	}
	if err := backendErrorFromWriteDispatch(res); err != nil {
		return mcpserver.ActionOutput{}, err
	}
	return mcpserver.ActionOutput{Message: "cancellation requested for job " + in.JobID}, nil
}

// DeleteCompleteScenarioRun dispatches DELETE /api/v1/scenarios/run/{name} with
// DeleteScenarioRunComplete, which requires cancel on all jobs. The legacy
// DeleteScenarioRun (view-only check) is never used from MCP.
func (b *mcpBackend) DeleteCompleteScenarioRun(ctx context.Context, p mcpserver.Principal, in mcpserver.ScenarioRunRefInput) (mcpserver.ActionOutput, error) {
	if _, err := b.getVisibleScenarioRun(ctx, p, in.ScenarioRunName); err != nil {
		return mcpserver.ActionOutput{}, err
	}

	res, err := b.h.dispatchAsRobot(ctx, p.RobotName, "DELETE", ScenariosRunPath+"/"+in.ScenarioRunName, nil, nil, nil, b.h.DeleteScenarioRunComplete)
	if err != nil {
		return mcpserver.ActionOutput{}, errMCPInternal()
	}
	if err := backendErrorFromWriteDispatch(res); err != nil {
		return mcpserver.ActionOutput{}, err
	}
	return mcpserver.ActionOutput{Message: "scenario run " + in.ScenarioRunName + " deleted"}, nil
}

// RunWorkflow starts a graph run from a saved workflow. The workflow must be
// accessible to the robot; CreateGraphRun re-validates run permission on all
// targets, file volumes, registries and cloud credentials.
func (b *mcpBackend) RunWorkflow(ctx context.Context, p mcpserver.Principal, in mcpserver.RunWorkflowInput) (mcpserver.RunStartedOutput, error) {
	if in.IdempotencyKey != "" {
		if err := validateIdempotencyKey(in.IdempotencyKey); err != nil {
			return mcpserver.RunStartedOutput{}, err
		}
		if existing, found, err := b.findIdempotencyGraphRun(ctx, p, in.IdempotencyKey); err != nil {
			return mcpserver.RunStartedOutput{}, errMCPInternal()
		} else if found {
			return existing, nil
		}
	}

	fileResp, err := b.h.getFileByIDInternal(withRobotClaims(ctx, p.RobotName), in.WorkflowID)
	if err != nil {
		return mcpserver.RunStartedOutput{}, errMCPNotFound("workflow %q not found", in.WorkflowID)
	}
	if fileResp.FilePurpose != files.FilePurposeWorkflow {
		return mcpserver.RunStartedOutput{}, errMCPInvalidArg("file %q is not a workflow template", in.WorkflowID)
	}
	wf, err := convertFileResponseToWorkflow(*fileResp)
	if err != nil {
		return mcpserver.RunStartedOutput{}, errMCPInternal()
	}

	headers := map[string]string{}
	if in.ResiliencyScoreEnabled {
		if in.ResiliencyBaseline == nil {
			return mcpserver.RunStartedOutput{}, errMCPInvalidArg("resiliencyBaseline is required when resiliencyScoreEnabled is true")
		}
		headers["X-Resiliency-Score"] = "true"
		headers["X-Resiliency-Baseline"] = strconvFormatFloat(*in.ResiliencyBaseline)
		if in.ResiliencyMountPath != "" {
			headers["X-Resiliency-Mount-Path"] = in.ResiliencyMountPath
		}
	}

	req := GraphRunCreateRequest{
		Graph:              wf.Graph,
		TargetRequestID:    in.TargetRequestID,
		TargetClusters:     in.TargetClusters,
		Categories:         in.Categories,
		CloudCredentialRef: in.CloudCredentialRef,
		MaxRetries:         in.MaxRetries,
	}

	res, err := b.h.dispatchAsRobot(ctx, p.RobotName, "POST", GraphRunsPath, nil, req, headers, b.h.CreateGraphRun)
	if err != nil {
		return mcpserver.RunStartedOutput{}, errMCPInternal()
	}
	if err := backendErrorFromWriteDispatch(res); err != nil {
		return mcpserver.RunStartedOutput{}, err
	}
	started, err := graphRunStarted(res)
	if err != nil {
		return mcpserver.RunStartedOutput{}, err
	}
	if in.IdempotencyKey != "" {
		b.labelGraphRunIdempotent(ctx, started.Name, in.IdempotencyKey)
	}
	return started, nil
}

// strconvFormatFloat formats a baseline for the X-Resiliency-Baseline header.
func strconvFormatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// graphRunStarted parses a CreateGraphRun 201 response.
func graphRunStarted(res dispatchResult) (mcpserver.RunStartedOutput, error) {
	var created GraphRunDetailResponse
	if err := decodeJSON(res.Body, &created); err != nil {
		return mcpserver.RunStartedOutput{}, errMCPInternal()
	}
	out := mcpserver.RunStartedOutput{Kind: "graph_run", Name: created.Name}
	for _, clusters := range created.Spec.TargetClusters {
		for _, name := range clusters {
			out.Jobs = append(out.Jobs, mcpserver.StartedJobOutput{
				ClusterName: name,
				Status:      "Pending",
				Success:     true,
			})
		}
	}
	return out, nil
}

// ReplayGraphRun re-submits an existing visible graph run. The graph is never
// returned or logged.
func (b *mcpBackend) ReplayGraphRun(ctx context.Context, p mcpserver.Principal, in mcpserver.ReplayGraphRunInput) (mcpserver.RunStartedOutput, error) {
	gr := &krknv1alpha1.KrknGraphRun{}
	if err := b.h.client.Get(ctx, client.ObjectKey{Name: in.GraphRunName, Namespace: b.h.namespace}, gr); err != nil {
		return mcpserver.RunStartedOutput{}, errMCPNotFound("graph run %q not found", in.GraphRunName)
	}

	// A run hidden by permissions (with a resolvable target request) is
	// not_found, so replay is not an existence oracle. Only when the target
	// request itself is unresolvable do we fall through to the stale-request
	// error below.
	if _, err := b.h.graphTargetURLs(ctx, gr); err == nil {
		if _, ok := b.graphRunVisible(ctx, p, gr); !ok {
			return mcpserver.RunStartedOutput{}, errMCPNotFound("graph run %q not found", in.GraphRunName)
		}
	}

	// A stale target request fails with a clear invalid_argument asking for a
	// fresh one (Review Focus 5).
	targetRequestID := gr.Spec.TargetRequestID
	if in.TargetRequestID != "" {
		targetRequestID = in.TargetRequestID
	}
	tr := &krknv1alpha1.KrknTargetRequest{}
	if err := b.h.client.Get(ctx, client.ObjectKey{Name: targetRequestID, Namespace: b.h.namespace}, tr); err != nil {
		return mcpserver.RunStartedOutput{}, errMCPInvalidArg(
			"original target request is unavailable; call krkn_list_clusters and pass targetRequestId")
	}

	categories, err := b.h.visibleRunCategoryNames(withRobotClaims(ctx, p.RobotName), gr.Labels)
	if err != nil {
		return mcpserver.RunStartedOutput{}, errMCPInternal()
	}

	headers := map[string]string{}
	if gr.Spec.ResiliencyScoreEnabled {
		if gr.Spec.ResiliencyScoreBaseline == nil {
			return mcpserver.RunStartedOutput{}, errMCPInvalidArg("resiliencyBaseline is required when resiliencyScoreEnabled is true")
		}
		headers["X-Resiliency-Score"] = "true"
		headers["X-Resiliency-Baseline"] = strconvFormatFloat(*gr.Spec.ResiliencyScoreBaseline)
		if gr.Spec.ResiliencyMountPath != "" {
			headers["X-Resiliency-Mount-Path"] = gr.Spec.ResiliencyMountPath
		}
	}

	req := GraphRunCreateRequest{
		Graph:              gr.Spec.Graph,
		TargetRequestID:    targetRequestID,
		TargetClusters:     gr.Spec.TargetClusters,
		Categories:         categories,
		CloudCredentialRef: gr.Spec.CloudCredentialRef,
		MaxRetries:         &gr.Spec.MaxRetries,
	}

	res, err := b.h.dispatchAsRobot(ctx, p.RobotName, "POST", GraphRunsPath, nil, req, headers, b.h.CreateGraphRun)
	if err != nil {
		return mcpserver.RunStartedOutput{}, errMCPInternal()
	}
	if err := backendErrorFromWriteDispatch(res); err != nil {
		return mcpserver.RunStartedOutput{}, err
	}
	return graphRunStarted(res)
}

// DeleteGraphRun deletes a graph run the robot created, requiring cancel
// permission on every target cluster. The dispatched handler re-checks owner.
func (b *mcpBackend) DeleteGraphRun(ctx context.Context, p mcpserver.Principal, in mcpserver.GraphRunRefInput) (mcpserver.ActionOutput, error) {
	gr := &krknv1alpha1.KrknGraphRun{}
	if err := b.h.client.Get(ctx, client.ObjectKey{Name: in.GraphRunName, Namespace: b.h.namespace}, gr); err != nil {
		return mcpserver.ActionOutput{}, errMCPNotFound("graph run %q not found", in.GraphRunName)
	}
	if err := b.h.robotCanDeleteGraphRun(ctx, p.RobotName, gr); err != nil {
		return mcpserver.ActionOutput{}, err
	}
	if _, ok := b.graphRunVisible(ctx, p, gr); !ok {
		return mcpserver.ActionOutput{}, errMCPNotFound("graph run %q not found", in.GraphRunName)
	}

	res, err := b.h.dispatchAsRobot(ctx, p.RobotName, "DELETE", GraphRunsPath+"/"+in.GraphRunName, nil, nil, nil, b.h.DeleteGraphRun)
	if err != nil {
		return mcpserver.ActionOutput{}, errMCPInternal()
	}
	if err := backendErrorFromWriteDispatch(res); err != nil {
		return mcpserver.ActionOutput{}, err
	}
	return mcpserver.ActionOutput{Message: "graph run " + in.GraphRunName + " deleted"}, nil
}

// robotCanDeleteGraphRun enforces the robot-specific deletion rule: the robot
// must own the run AND hold cancel on every target cluster. An unresolvable
// target denies.
func (h *Handler) robotCanDeleteGraphRun(ctx context.Context, robotName string, gr *krknv1alpha1.KrknGraphRun) error {
	if gr.Spec.OwnerUserID != "robot:"+robotName {
		return &mcpserver.BackendError{Code: mcpserver.CodeForbidden, Message: "robots can only delete graph runs they created"}
	}
	urls, err := h.graphTargetURLs(ctx, gr)
	if err != nil {
		return &mcpserver.BackendError{Code: mcpserver.CodeForbidden, Message: "cannot verify cancel permission on all target clusters"}
	}
	perms, err := h.robotClusterPerms(ctx, robotName)
	if err != nil {
		return errMCPInternal()
	}
	for _, clusters := range gr.Spec.TargetClusters {
		for _, name := range clusters {
			url, ok := urls[name]
			if !ok || !perms.can(url, groupauth.ActionCancel) {
				return &mcpserver.BackendError{Code: mcpserver.CodeForbidden, Message: "cancel permission is required on every target cluster"}
			}
		}
	}
	return nil
}
