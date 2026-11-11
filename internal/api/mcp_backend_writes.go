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
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
//
// For scenario runs the label is a secondary lookup aid: the primary duplicate
// suppression is the deterministic run name derived by idempotentRunName, which
// the REST create enforces atomically via Kubernetes name uniqueness. For graph
// runs the REST create contract (GraphRunCreateRequest) accepts neither a run
// name nor labels, so the label is the only available match and a small
// create-to-label window remains; see findIdempotencyGraphRun.
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

// idempotentRunName derives a deterministic, DNS-1123-safe run name from the
// robot and idempotency key. PostScenarioRun uses the sanitized customRunName
// as the KrknScenarioRun resource name, so a retry or concurrent duplicate
// submission with the same key hits Kubernetes name uniqueness and the create
// fails with AlreadyExists — making the key atomic with creation, with no
// create-then-label window. The robot name is hashed in so keys are scoped per
// robot, and the "mcp-idem-" prefix namespaces the derived name away from
// client-supplied custom run names.
func idempotentRunName(robotName, key string) string {
	sum := sha256.Sum256([]byte("robot:" + robotName + "\x00" + key))
	return "mcp-idem-" + hex.EncodeToString(sum[:8])
}

// scenarioRunStartedOutputFromRun builds the run-started output returned to a
// duplicate submission from an existing run's status.
func scenarioRunStartedOutputFromRun(run *krknv1alpha1.KrknScenarioRun) mcpserver.RunStartedOutput {
	out := mcpserver.RunStartedOutput{Kind: "scenario_run", Name: run.Name}
	for _, job := range run.Status.ClusterJobs {
		out.Jobs = append(out.Jobs, mcpserver.StartedJobOutput{
			ClusterName: job.ClusterName,
			JobID:       job.JobID,
			Status:      job.Phase,
			Success:     true,
		})
	}
	return out
}

// idempotencyCandidate pairs a run name with its creation time so candidates
// can be ordered oldest-first.
type idempotencyCandidate struct {
	name    string
	created metav1.Time
}

// sortIdempotencyCandidates orders candidates oldest-first by creation
// timestamp (name as tie-breaker), so a duplicate key deterministically
// resolves to the first-created run.
func sortIdempotencyCandidates(cands []idempotencyCandidate) {
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].created.Time.Equal(cands[j].created.Time) {
			return cands[i].name < cands[j].name
		}
		return cands[i].created.Time.Before(cands[j].created.Time)
	})
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

	cands := make([]idempotencyCandidate, 0, len(runs.Items))
	for i := range runs.Items {
		if runs.Items[i].Spec.OwnerUserID == "robot:"+p.RobotName {
			cands = append(cands, idempotencyCandidate{
				name:    runs.Items[i].Name,
				created: runs.Items[i].CreationTimestamp,
			})
		}
	}
	if len(cands) == 0 {
		return mcpserver.RunStartedOutput{}, false, nil
	}
	sortIdempotencyCandidates(cands)

	// Multiple matches can only result from a label patch failure on a
	// pre-deterministic-name run; deterministically return the oldest
	// (first created) run.
	run, err := b.getVisibleScenarioRun(ctx, p, cands[0].name)
	if err != nil {
		return mcpserver.RunStartedOutput{}, false, err
	}
	return scenarioRunStartedOutputFromRun(run), true, nil
}

// resolveIdempotencyScenarioConflict maps a 409 conflict from the create
// dispatch to the existing run when that run belongs to this robot and was
// created with the same idempotency key — either because a concurrent
// duplicate submission won the deterministic-name race, or because an earlier
// attempt succeeded but its label patch failed. handled is false when the
// conflicting run belongs to another principal or does not carry this key, in
// which case the conflict error must be propagated unchanged.
func (b *mcpBackend) resolveIdempotencyScenarioConflict(ctx context.Context, p mcpserver.Principal, runName, key, derivedName string) (mcpserver.RunStartedOutput, bool) {
	run := &krknv1alpha1.KrknScenarioRun{}
	if err := b.h.client.Get(ctx, client.ObjectKey{Name: runName, Namespace: b.h.namespace}, run); err != nil {
		return mcpserver.RunStartedOutput{}, false
	}
	if run.Spec.OwnerUserID != "robot:"+p.RobotName {
		return mcpserver.RunStartedOutput{}, false
	}
	if run.Labels[mcpIdempotencyKeyLabel] != key && runName != derivedName {
		return mcpserver.RunStartedOutput{}, false
	}
	visible, err := b.getVisibleScenarioRun(ctx, p, runName)
	if err != nil {
		return mcpserver.RunStartedOutput{}, false
	}
	return scenarioRunStartedOutputFromRun(visible), true
}

// labelScenarioRunIdempotent records the idempotency key on a newly created
// run so the label-based lookup in findIdempotencyScenarioRun can find it.
// Duplicate suppression does not depend on this patch: the run was created
// under a deterministic idempotentRunName, so a retry that misses the label
// still conflicts on the name and is resolved by
// resolveIdempotencyScenarioConflict. A patch failure is therefore logged but
// not fatal.
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
//
// Unlike scenario runs, graph runs cannot use deterministic naming:
// GraphRunCreateRequest (parsed with DisallowUnknownFields) accepts neither a
// run name nor labels, and CreateGraphRun always generates a
// "graphrun-<uuid8>" name server-side. The label-based check-create-label
// sequence is therefore the best the REST contract allows; a submission racing
// the post-create label patch can still create a second run, though the owner
// scoping below means it can only ever duplicate this robot's own run.
func (b *mcpBackend) findIdempotencyGraphRun(ctx context.Context, p mcpserver.Principal, key string) (mcpserver.RunStartedOutput, bool, error) {
	runs := &krknv1alpha1.KrknGraphRunList{}
	if err := b.h.client.List(ctx, runs,
		client.InNamespace(b.h.namespace),
		client.MatchingLabels{mcpIdempotencyKeyLabel: key},
	); err != nil {
		return mcpserver.RunStartedOutput{}, false, err
	}

	cands := make([]idempotencyCandidate, 0, len(runs.Items))
	for i := range runs.Items {
		if runs.Items[i].Spec.OwnerUserID == "robot:"+p.RobotName {
			cands = append(cands, idempotencyCandidate{
				name:    runs.Items[i].Name,
				created: runs.Items[i].CreationTimestamp,
			})
		}
	}
	if len(cands) == 0 {
		return mcpserver.RunStartedOutput{}, false, nil
	}
	sortIdempotencyCandidates(cands)

	// Deterministically return the oldest (first created) run.
	gr := &krknv1alpha1.KrknGraphRun{}
	if err := b.h.client.Get(ctx, client.ObjectKey{Name: cands[0].name, Namespace: b.h.namespace}, gr); err != nil {
		return mcpserver.RunStartedOutput{}, false, err
	}
	if _, ok := b.graphRunVisible(ctx, p, gr); !ok {
		return mcpserver.RunStartedOutput{}, false, nil
	}
	return mcpserver.RunStartedOutput{Kind: "graph_run", Name: gr.Name}, true, nil
}

// labelGraphRunIdempotent records the idempotency key on a newly created graph
// run. A failure is logged but not fatal: the run did start, and the loss only
// affects duplicate suppression for a later retry of the same key (see
// findIdempotencyGraphRun for why graph runs have no stronger mechanism).
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

	// PostScenarioRun uses the sanitized customRunName as the resource name,
	// so deriving the name from (robot, key) makes the idempotency key atomic
	// with creation: a retry or concurrent duplicate gets AlreadyExists from
	// Kubernetes instead of starting a second experiment. When the client
	// supplies its own customRunName, that name already provides uniqueness
	// and is left untouched.
	customRunName := in.CustomRunName
	derivedRunName := ""
	if in.IdempotencyKey != "" && customRunName == "" {
		derivedRunName = idempotentRunName(p.RobotName, in.IdempotencyKey)
		customRunName = derivedRunName
	}

	private := in.PrivateRegistry
	req := ScenarioRunRequest{
		TargetRequestID:         in.TargetRequestID,
		TargetClusters:          in.TargetClusters,
		Categories:              in.Categories,
		Scenario:                krknv1alpha1.ScenarioReference{Name: in.ScenarioName, Private: &private, RegistryName: in.RegistryName},
		Environment:             in.Environment,
		FileReferences:          fileReferencesToAPI(in.FileReferences),
		CustomRunName:           customRunName,
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
		// A 409 means a run with the effective name already exists. When it is
		// this robot's run for the same key, return it instead of the error.
		if in.IdempotencyKey != "" {
			var be *mcpserver.BackendError
			if errors.As(err, &be) && be.Code == mcpserver.CodeConflict {
				effectiveName := derivedRunName
				if effectiveName == "" && in.CustomRunName != "" {
					effectiveName = sanitizeResourceName(in.CustomRunName)
				}
				if effectiveName != "" {
					if existing, handled := b.resolveIdempotencyScenarioConflict(ctx, p, effectiveName, in.IdempotencyKey, derivedRunName); handled {
						return existing, nil
					}
				}
			}
		}
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

	// Visibility is enforced for every principal. The only exception is the
	// owning robot when the run's original target request is unresolvable:
	// graphRunVisible cannot distinguish "request gone" from "hidden", and the
	// owner must be able to replay with a fresh targetRequestId. A non-owner
	// in that situation gets not_found (fail closed) — supplying a fresh
	// targetRequestId must not expose a graph the robot cannot view.
	if _, urlErr := b.h.graphTargetURLs(ctx, gr); urlErr != nil && gr.Spec.OwnerUserID == "robot:"+p.RobotName {
		logMCP(ctx, "graph run target request unresolvable; owner may replay with a fresh targetRequestId", "graphRun", gr.Name)
	} else if _, ok := b.graphRunVisible(ctx, p, gr); !ok {
		return mcpserver.RunStartedOutput{}, errMCPNotFound("graph run %q not found", in.GraphRunName)
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
// permission on every target cluster. Visibility is checked before ownership
// so deletion is not an existence oracle: a hidden run is not_found whether or
// not the robot created it, while a visible run created by someone else is
// forbidden. The dispatched handler re-checks owner.
func (b *mcpBackend) DeleteGraphRun(ctx context.Context, p mcpserver.Principal, in mcpserver.GraphRunRefInput) (mcpserver.ActionOutput, error) {
	gr := &krknv1alpha1.KrknGraphRun{}
	if err := b.h.client.Get(ctx, client.ObjectKey{Name: in.GraphRunName, Namespace: b.h.namespace}, gr); err != nil {
		return mcpserver.ActionOutput{}, errMCPNotFound("graph run %q not found", in.GraphRunName)
	}
	if _, ok := b.graphRunVisible(ctx, p, gr); !ok {
		return mcpserver.ActionOutput{}, errMCPNotFound("graph run %q not found", in.GraphRunName)
	}
	if err := b.h.robotCanDeleteGraphRun(ctx, p.RobotName, gr); err != nil {
		return mcpserver.ActionOutput{}, err
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
